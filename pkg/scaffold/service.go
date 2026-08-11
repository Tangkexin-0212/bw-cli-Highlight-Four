package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

const defaultServicePort = 9100

var serviceNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
var legacyGatewayTargetPattern = regexp.MustCompile(`(?s)\nfunc ` + `gateway` + `GRPC` + `Target` + `\(envName string, fallback string\) string \{\s*value := strings\.TrimSpace\(os\.Get` + `env\(envName\)\)\s*if value == "" \{\s*return fallback\s*\}\s*return value\s*\}\s*`)
var legacyServiceTargetFallbackPattern = regexp.MustCompile(`cfg\.ServiceTarget\("([^"]+)",\s*[0-9]+\)`)
var gatewayClientDialPattern = regexp.MustCompile(`(?m)^\t([A-Za-z]\w*)Conn, err := grpc\.Dial`)
var gatewayClientConnsPattern = regexp.MustCompile(`conns:\s+\[\]\*grpc\.ClientConn\{([^}]*)\}`)

// ServiceOptions controls bw-cli service generation inside an existing project.
type ServiceOptions struct {
	RootDir           string
	Name              string
	Port              int
	RunProto          bool
	RunTidy           bool
	NacosSyncRequired bool
	NacosDataID       string
}

type serviceTemplateData struct {
	Module       string
	InputName    string
	Dir          string
	ProtoFile    string
	ProtoPackage string
	GoPackage    string
	GoIdent      string
	Pascal       string
	ServiceName  string
	Port         int
	TableName    string
}

// AddService creates a complete gRPC service skeleton in an existing bw-cli project.
func AddService(opts ServiceOptions) error {
	root, err := serviceRoot(opts.RootDir)
	if err != nil {
		return err
	}
	module, err := readModule(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	port := opts.Port
	if port == 0 {
		port, err = nextServicePort(root)
		if err != nil {
			return err
		}
	}
	data, err := buildServiceTemplateData(module, opts.Name, port)
	if err != nil {
		return err
	}
	if err := ensureServiceDoesNotExist(root, data); err != nil {
		return err
	}
	if err := writeServiceFiles(root, data); err != nil {
		return err
	}
	if err := writeGatewayServiceFiles(root, data); err != nil {
		return err
	}
	if err := addServiceMakeTarget(root, data.Dir); err != nil {
		return err
	}
	if err := addServiceConfig(root, data); err != nil {
		return err
	}
	if enabled, dataID := nacosEnabled(root); enabled {
		fmt.Printf("nacos is enabled: sync configs/config.yaml service changes to Nacos data_id %s\n", dataID)
	}
	if opts.RunProto {
		if err := runProjectCommand(root, "go", "run", "./tools/protogen"); err != nil {
			return fmt.Errorf("generate proto for %s: %w", data.Dir, err)
		}
	}
	if err := gofmtService(root, data.Dir); err != nil {
		return err
	}
	if opts.RunTidy {
		if err := runProjectCommand(root, "go", "mod", "tidy"); err != nil {
			return fmt.Errorf("go mod tidy: %w", err)
		}
	}
	return nil
}

func serviceRoot(root string) (string, error) {
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("go.mod not found in %s", abs)
		}
		return "", err
	}
	return abs, nil
}

func buildServiceTemplateData(module string, rawName string, port int) (serviceTemplateData, error) {
	parts, err := splitServiceName(rawName)
	if err != nil {
		return serviceTemplateData{}, err
	}
	if port == 0 {
		port = defaultServicePort
	}
	if port < 0 || port > 65535 {
		return serviceTemplateData{}, fmt.Errorf("port must be between 1 and 65535")
	}
	dir := strings.Join(parts, "_")
	pascal := toPascal(parts)
	return serviceTemplateData{
		Module:       module,
		InputName:    strings.TrimSpace(rawName),
		Dir:          dir,
		ProtoFile:    dir + ".proto",
		ProtoPackage: dir + ".v1",
		GoPackage:    strings.ToLower(pascal) + "v1",
		GoIdent:      lowerFirst(pascal),
		Pascal:       pascal,
		ServiceName:  strings.Join(parts, "-") + "-service",
		Port:         port,
		TableName:    dir + "s",
	}, nil
}

func splitServiceName(rawName string) ([]string, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return nil, errors.New("service name is required")
	}
	if !serviceNamePattern.MatchString(name) {
		return nil, fmt.Errorf("service name %q must start with a letter and only contain letters, digits, hyphen or underscore", rawName)
	}
	rawParts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	})
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			return nil, fmt.Errorf("service name %q contains empty segment", rawName)
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func toPascal(parts []string) string {
	var b strings.Builder
	for _, part := range parts {
		runes := []rune(part)
		for i, r := range runes {
			if i == 0 {
				b.WriteRune(unicode.ToUpper(r))
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func ensureServiceDoesNotExist(root string, data serviceTemplateData) error {
	for _, rel := range []string{
		filepath.Join("cmd", data.Dir),
		filepath.Join("internal", data.Dir),
		filepath.Join("api", "proto", data.Dir),
		filepath.Join("api", "gen", data.Dir),
	} {
		if exists(filepath.Join(root, rel)) {
			return fmt.Errorf("service %s already exists: %s", data.Dir, rel)
		}
	}
	return nil
}

func writeServiceFiles(root string, data serviceTemplateData) error {
	files := map[string]string{
		filepath.Join("api", "proto", data.Dir, "v1", data.ProtoFile):      renderServiceTemplate(serviceProtoTemplate, data),
		filepath.Join("cmd", data.Dir, "main.go"):                          renderServiceTemplate(serviceMainTemplate, data),
		filepath.Join("internal", data.Dir, "model", data.Dir+".go"):       renderServiceTemplate(serviceModelTemplate, data),
		filepath.Join("internal", data.Dir, "model", "repository.go"):      renderServiceTemplate(serviceRepositoryTemplate, data),
		filepath.Join("internal", data.Dir, "dto", "command.go"):           renderServiceTemplate(serviceCommandTemplate, data),
		filepath.Join("internal", data.Dir, "dto", data.Dir+".go"):         renderServiceTemplate(serviceDTOTemplate, data),
		filepath.Join("internal", data.Dir, "service", "service.go"):       renderServiceTemplate(serviceUseCaseTemplate, data),
		filepath.Join("internal", data.Dir, "service", "service_test.go"):  renderServiceTemplate(serviceUseCaseTestTemplate, data),
		filepath.Join("internal", data.Dir, "repo", "gorm_repository.go"):  renderServiceTemplate(serviceGormRepoTemplate, data),
		filepath.Join("internal", data.Dir, "repo", "mongo_repository.go"): renderServiceTemplate(serviceMongoRepoTemplate, data),
		filepath.Join("internal", data.Dir, "handler", "server.go"):        renderServiceTemplate(serviceHandlerTemplate, data),
		filepath.Join("docs", "services", data.Dir+".md"):                  renderServiceTemplate(serviceDocTemplate, data),
	}
	for rel, content := range files {
		if err := writeNewFile(filepath.Join(root, rel), []byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func writeGatewayServiceFiles(root string, data serviceTemplateData) error {
	routerDir := filepath.Join(root, "internal", "gateway", "router")
	if !exists(routerDir) {
		return nil
	}
	commonPath := filepath.Join(root, "internal", "gateway", "handler", "common.go")
	if err := ensureGatewayCommonFile(commonPath, data); err != nil {
		return err
	}
	clientsPath := filepath.Join(root, "internal", "gateway", "client", "clients.go")
	if err := ensureGatewayClientsFile(clientsPath, data); err != nil {
		return err
	}
	files := map[string]string{
		filepath.Join("internal", "gateway", "request", data.Dir+"_request.go"): renderServiceTemplate(gatewayRequestTemplate, data),
		filepath.Join("internal", "gateway", "handler", data.Dir+"_handler.go"): renderServiceTemplate(gatewayHandlerTemplate, data),
		filepath.Join("internal", "gateway", "router", data.Dir+"_routes.go"):   renderServiceTemplate(gatewayRoutesTemplate, data),
	}
	for rel, content := range files {
		if err := writeNewFile(filepath.Join(root, rel), []byte(content)); err != nil {
			return err
		}
	}
	if err := patchGatewayRouter(root, data); err != nil {
		return err
	}
	return nil
}

func ensureGatewayCommonFile(path string, data serviceTemplateData) error {
	if !exists(path) {
		return writeNewFile(path, []byte(renderServiceTemplate(gatewayCommonTemplate, data)))
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(content)
	if strings.Contains(text, "func "+"configuredGateway"+"GRPCTarget") {
		return nil
	}
	if legacyGatewayTargetPattern.MatchString(text) {
		text = legacyGatewayTargetPattern.ReplaceAllString(text, "\n")
		text = removeImport(text, "\"os\"")
		return os.WriteFile(path, []byte(text), 0o644)
	}
	return nil
}

func ensureGatewayClientsFile(path string, data serviceTemplateData) error {
	if !exists(path) {
		return writeNewFile(path, []byte(renderServiceTemplate(gatewayClientsTemplate, data)))
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(content)
	text = legacyServiceTargetFallbackPattern.ReplaceAllString(text, `cfg.ServiceTarget("$1")`)
	field := fmt.Sprintf("\t%s  %s.%sServiceClient\n", data.Pascal, data.GoPackage, data.Pascal)
	if strings.Contains(text, field) {
		return nil
	}

	text = ensureImport(text, fmt.Sprintf("%s %q", data.GoPackage, data.Module+"/api/gen/"+data.Dir+"/v1"))
	text = strings.Replace(text, "type Clients struct {\n", "type Clients struct {\n"+field, 1)

	targetLine := fmt.Sprintf("\t%sTarget := cfg.ServiceTarget(%q)\n", data.GoIdent, data.Dir)
	if loc := gatewayClientDialPattern.FindStringIndex(text); loc != nil {
		text = text[:loc[0]] + targetLine + text[loc[0]:]
	}

	existingConnNames := gatewayClientConnNames(text)
	dialBlock := fmt.Sprintf("\t%sConn, err := grpc.Dial(%sTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))\n\tif err != nil {\n%s\t\treturn nil, fmt.Errorf(\"dial %s service: %%w\", err)\n\t}\n", data.GoIdent, data.GoIdent, closeConnLines(existingConnNames), data.Dir)
	if marker := "\n\tlog.Info(\"grpc clients initialized\","; strings.Contains(text, marker) {
		text = strings.Replace(text, marker, "\n"+dialBlock+marker, 1)
	}

	zapLine := fmt.Sprintf("\t\tzap.String(\"%s_target\", %sTarget),", data.Dir, data.GoIdent)
	if marker := "\n\t)\n\treturn &Clients{"; strings.Contains(text, marker) && !strings.Contains(text, zapLine) {
		text = strings.Replace(text, marker, "\n"+zapLine+marker, 1)
	}

	clientLine := fmt.Sprintf("\t\t%s:  %s.New%sServiceClient(%sConn),\n", data.Pascal, data.GoPackage, data.Pascal, data.GoIdent)
	if strings.Contains(text, "\t\tConfig: cfg,\n") {
		text = strings.Replace(text, "\t\tConfig: cfg,\n", clientLine+"\t\tConfig: cfg,\n", 1)
	}
	text = gatewayClientConnsPattern.ReplaceAllStringFunc(text, func(value string) string {
		if strings.Contains(value, data.GoIdent+"Conn") {
			return value
		}
		return strings.TrimRight(value, "}") + ", " + data.GoIdent + "Conn}"
	})
	return os.WriteFile(path, []byte(text), 0o644)
}

func gatewayClientConnNames(text string) []string {
	matches := gatewayClientDialPattern.FindAllStringSubmatch(text, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1]+"Conn")
	}
	return names
}

func closeConnLines(connNames []string) string {
	var b strings.Builder
	for _, name := range connNames {
		fmt.Fprintf(&b, "\t\t%s.Close()\n", name)
	}
	return b.String()
}

func ensureImport(text string, quotedPackage string) string {
	if strings.Contains(text, quotedPackage) {
		return text
	}
	if strings.Contains(text, "import (\n") {
		return strings.Replace(text, "import (\n", "import (\n\t"+quotedPackage+"\n", 1)
	}
	return text
}

func removeImport(text string, quotedPackage string) string {
	lines := strings.Split(text, "\n")
	out := lines[:0]
	for _, line := range lines {
		if strings.TrimSpace(line) == quotedPackage {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func writeNewFile(path string, data []byte) error {
	if exists(path) {
		return fmt.Errorf("file already exists: %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func renderServiceTemplate(body string, data serviceTemplateData) string {
	tpl := template.Must(template.New("service").Parse(body))
	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		panic(err)
	}
	return out.String()
}

func addServiceMakeTarget(root string, serviceDir string) error {
	path := filepath.Join(root, "Makefile")
	if !exists(path) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	target := "run-" + serviceDir
	if strings.Contains(text, "\n"+target+":") || strings.HasPrefix(text, target+":") {
		return nil
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, ".PHONY:") && !strings.Contains(line, " "+target) {
			lines[i] = strings.TrimRight(line, " ") + " " + target
			break
		}
	}
	text = strings.TrimRight(strings.Join(lines, "\n"), "\n")
	text += "\n\n" + target + ":\n\t$(GO) run ./cmd/" + serviceDir + "\n"
	return os.WriteFile(path, []byte(text), 0o644)
}

func nextServicePort(root string) (int, error) {
	path := filepath.Join(root, "configs", "config.yaml")
	if !exists(path) {
		return defaultServicePort, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	maxPort := defaultServicePort - 1
	portPattern := regexp.MustCompile(`(?m)^\s+port:\s*([0-9]+)\s*$`)
	for _, match := range portPattern.FindAllStringSubmatch(string(data), -1) {
		var port int
		if _, err := fmt.Sscanf(match[1], "%d", &port); err == nil && port > maxPort {
			maxPort = port
		}
	}
	if maxPort < defaultServicePort {
		return defaultServicePort, nil
	}
	return maxPort + 1, nil
}

func addServiceConfig(root string, data serviceTemplateData) error {
	path := filepath.Join(root, "configs", "config.yaml")
	if !exists(path) {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(content)
	serviceBlock := fmt.Sprintf(`  %s:
    name: %s
    port: %d
    target: 127.0.0.1:%d
`, data.Dir, data.ServiceName, data.Port, data.Port)
	if regexp.MustCompile(`(?m)^\s{2}` + regexp.QuoteMeta(data.Dir) + `:\s*$`).MatchString(text) {
		return nil
	}
	if !regexp.MustCompile(`(?m)^services:\s*$`).MatchString(text) {
		insert := "\nservices:\n" + serviceBlock
		if idx := strings.Index(text, "\ndatabase:"); idx >= 0 {
			text = text[:idx] + insert + text[idx:]
		} else {
			text = strings.TrimRight(text, "\n") + insert + "\n"
		}
		return os.WriteFile(path, []byte(text), 0o644)
	}
	lines := strings.Split(text, "\n")
	insertAt := len(lines)
	for i, line := range lines {
		if strings.TrimSpace(line) == "services:" {
			insertAt = i + 1
			continue
		}
		if insertAt != len(lines) && line != "" && !strings.HasPrefix(line, " ") && strings.HasSuffix(line, ":") {
			insertAt = i
			break
		}
	}
	blockLines := strings.Split(strings.TrimRight(serviceBlock, "\n"), "\n")
	lines = append(lines[:insertAt], append(blockLines, lines[insertAt:]...)...)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func nacosEnabled(root string) (bool, string) {
	path := filepath.Join(root, "configs", "nacos.yaml")
	if !exists(path) {
		return false, "xiaolanshu.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, "xiaolanshu.yaml"
	}
	text := string(data)
	enabled := regexp.MustCompile(`(?m)^enabled:\s*true\s*$`).MatchString(text)
	dataID := "xiaolanshu.yaml"
	if match := regexp.MustCompile(`(?m)^data_id:\s*"?([^"\n]+)"?\s*$`).FindStringSubmatch(text); len(match) == 2 {
		dataID = strings.TrimSpace(match[1])
	}
	return enabled, dataID
}

func gofmtService(root string, serviceDir string) error {
	args := []string{
		filepath.Join("cmd", serviceDir, "main.go"),
		filepath.Join("internal", serviceDir, "model", serviceDir+".go"),
		filepath.Join("internal", serviceDir, "model", "repository.go"),
		filepath.Join("internal", serviceDir, "dto", "command.go"),
		filepath.Join("internal", serviceDir, "dto", serviceDir+".go"),
		filepath.Join("internal", serviceDir, "service", "service.go"),
		filepath.Join("internal", serviceDir, "service", "service_test.go"),
		filepath.Join("internal", serviceDir, "repo", "gorm_repository.go"),
		filepath.Join("internal", serviceDir, "repo", "mongo_repository.go"),
		filepath.Join("internal", serviceDir, "handler", "server.go"),
	}
	for _, rel := range []string{
		filepath.Join("internal", "gateway", "handler", "common.go"),
		filepath.Join("internal", "gateway", "request", serviceDir+"_request.go"),
		filepath.Join("internal", "gateway", "handler", serviceDir+"_handler.go"),
		filepath.Join("internal", "gateway", "router", serviceDir+"_routes.go"),
		filepath.Join("internal", "gateway", "router", "router.go"),
		filepath.Join("internal", "gateway", "router", "v1.go"),
	} {
		if exists(filepath.Join(root, rel)) {
			args = append(args, rel)
		}
	}
	return runProjectCommand(root, "gofmt", append([]string{"-w"}, args...)...)
}

func patchGatewayRouter(root string, data serviceTemplateData) error {
	routerPath := filepath.Join(root, "internal", "gateway", "router", "router.go")
	if exists(routerPath) {
		routerBytes, err := os.ReadFile(routerPath)
		if err != nil {
			return err
		}
		routerText := string(routerBytes)
		routerText = ensureImport(routerText, fmt.Sprintf("%q", data.Module+"/internal/gateway/client"))
		if strings.Contains(routerText, "func New(log *zap.Logger, middlewareCfg config.MiddlewareConfig) *gin.Engine") {
			routerText = strings.Replace(routerText, "func New(log *zap.Logger, middlewareCfg config.MiddlewareConfig) *gin.Engine", "func New(clients *client.Clients, log *zap.Logger, middlewareCfg config.MiddlewareConfig) *gin.Engine", 1)
		}
		if strings.Contains(routerText, "registerAPIRoutes(r)") {
			routerText = strings.Replace(routerText, "registerAPIRoutes(r)", "registerAPIRoutes(r, clients, log)", 1)
		}
		if err := os.WriteFile(routerPath, []byte(routerText), 0o644); err != nil {
			return err
		}
	}
	if err := patchGatewayMain(root, data); err != nil {
		return err
	}

	v1Path := filepath.Join(root, "internal", "gateway", "router", "v1.go")
	if !exists(v1Path) {
		return nil
	}
	v1Bytes, err := os.ReadFile(v1Path)
	if err != nil {
		return err
	}
	v1Text := string(v1Bytes)
	registration := fmt.Sprintf("register%sRoutes(v1, handler.New%sHandler(clients.%s, log))", data.Pascal, data.Pascal, data.Pascal)
	if strings.Contains(v1Text, registration) {
		return nil
	}
	if strings.Contains(v1Text, "func registerAPIRoutes(r *gin.Engine)") {
		return os.WriteFile(v1Path, []byte(renderServiceTemplate(cleanGatewayV1WithServiceTemplate, data)), 0o644)
	}
	if !strings.Contains(v1Text, "func registerAPIRoutes(r *gin.Engine, clients *client.Clients") {
		return nil
	}
	if strings.Contains(v1Text, "\n\t_ = v1\n") {
		v1Text = strings.Replace(v1Text, "\n\t_ = v1\n", "\n\t"+registration+"\n", 1)
		return os.WriteFile(v1Path, []byte(v1Text), 0o644)
	}
	index := strings.LastIndex(v1Text, "\n}")
	if index == -1 {
		return nil
	}
	v1Text = v1Text[:index] + "\n\t" + registration + v1Text[index:]
	return os.WriteFile(v1Path, []byte(v1Text), 0o644)
}

func patchGatewayMain(root string, data serviceTemplateData) error {
	mainPath := filepath.Join(root, "cmd", "gateway", "main.go")
	if !exists(mainPath) {
		return nil
	}
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return err
	}
	text := string(content)
	text = ensureImport(text, fmt.Sprintf("%q", data.Module+"/internal/gateway/client"))
	if strings.Contains(text, "engine := router.New(log, cfg.Middleware)") {
		initBlock := `gatewayClients, err := client.New(cfg, log)
	if err != nil {
		log.Fatal("initialize grpc clients failed", zap.Error(err))
	}
	defer gatewayClients.Close()

	engine := router.New(gatewayClients, log, cfg.Middleware)`
		text = strings.Replace(text, "engine := router.New(log, cfg.Middleware)", initBlock, 1)
	}
	return os.WriteFile(mainPath, []byte(text), 0o644)
}

func runProjectCommand(root string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
