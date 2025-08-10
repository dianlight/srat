# GitHub Copilot Rules for SRAT Project

## Go Language and Project Structure Guidelines

### Package Organization

1. **Package naming**: Use descriptive, lowercase package names without underscores
2. **Import organization**: Group imports in this order:
   - Standard library imports
   - External library imports
   - Internal project imports (github.com/dianlight/srat/...)
3. **Project structure**: Follow the established structure:
   - `api/` - HTTP handlers and API endpoints
   - `service/` - Business logic and service layer
   - `repository/` - Data access layer
   - `dto/` - Data Transfer Objects
   - `dbom/` - Database Object Models
   - `converter/` - Object conversion utilities
   - `config/` - Configuration management
   - `internal/` - Internal utilities and app setup

### API Handler Conventions

1. **Handler structure**: Follow this pattern for API handlers:

   ```go
   type SomethingHandler struct {
       apiContext   *dto.ContextState
       dirtyService service.DirtyDataServiceInterface
       someService  service.SomeServiceInterface
   }
   ```

2. **Constructor pattern**: Use this pattern for handler constructors:

   ```go
   func NewSomethingHandler(
       apiContext *dto.ContextState,
       dirtyService service.DirtyDataServiceInterface,
       someService service.SomeServiceInterface,
   ) *SomethingHandler {
       p := new(SomethingHandler)
       p.apiContext = apiContext
       p.dirtyService = dirtyService
       p.someService = someService
       return p
   }
   ```

3. **Registration pattern**: Use this pattern for registering HTTP routes:

   ```go
   func (self *SomethingHandler) RegisterSomethingHandler(api huma.API) {
       huma.Get(api, "/something", self.GetSomething, huma.OperationTags("something"))
       huma.Post(api, "/something", self.CreateSomething, huma.OperationTags("something"))
       huma.Put(api, "/something/{id}", self.UpdateSomething, huma.OperationTags("something"))
       huma.Delete(api, "/something/{id}", self.DeleteSomething, huma.OperationTags("something"))
   }
   ```

4. **Handler method signatures**: Use this pattern for handler methods:

   ```go
   func (self *SomethingHandler) GetSomething(ctx context.Context, input *struct{}) (*struct{ Body dto.Something }, error) {
       // Implementation
       return &struct{ Body dto.Something }{Body: result}, nil
   }
   ```

5. **Path parameters**: Use this pattern for path parameters:

   ```go
   func (self *SomethingHandler) GetSomethingById(ctx context.Context, input *struct {
       ID string `path:"id" maxLength:"30" example:"123" doc:"ID of the something"`
   }) (*struct{ Body dto.Something }, error)
   ```

6. **Request body handling**: Use this pattern for request bodies:

   ```go
   func (self *SomethingHandler) CreateSomething(ctx context.Context, input *struct {
       Body dto.Something `required:"true"`
   }) (*struct{ Status int; Body dto.Something }, error) {
       // Implementation
       return &struct{ Status int; Body dto.Something }{
           Status: http.StatusCreated,
           Body: result,
       }, nil
   }
   ```

### Service Layer Conventions

1. **Service interfaces**: Define interfaces for all services:

   ```go
   type SomethingServiceInterface interface {
       GetSomething(id string) (*dto.Something, error)
       CreateSomething(something dto.Something) (*dto.Something, error)
       UpdateSomething(id string, something dto.Something) (*dto.Something, error)
       DeleteSomething(id string) error
   }
   ```

2. **Service implementation**: Follow this pattern:

   ```go
   type SomethingService struct {
       repo      repository.SomethingRepositoryInterface
       broadcast BroadcasterServiceInterface
   }

   type SomethingServiceParams struct {
       fx.In
       Repo      repository.SomethingRepositoryInterface
       Broadcast BroadcasterServiceInterface
   }

   func NewSomethingService(in SomethingServiceParams) SomethingServiceInterface {
       return &SomethingService{
           repo:      in.Repo,
           broadcast: in.Broadcast,
       }
   }
   ```

### Repository Layer Conventions

1. **Repository interfaces**: Define interfaces with GORM operations:

   ```go
   type SomethingRepositoryInterface interface {
       All() ([]dbom.Something, error)
       Find(id string) (*dbom.Something, error)
       Save(something *dbom.Something) error
       Delete(id string) error
   }
   ```

2. **Repository implementation**: Use GORM with thread-safe patterns:

   ```go
   type SomethingRepository struct {
       db    *gorm.DB
       mutex sync.RWMutex
   }

   func NewSomethingRepository(db *gorm.DB) SomethingRepositoryInterface {
       return &SomethingRepository{
           db:    db,
           mutex: sync.RWMutex{},
       }
   }
   ```

### Error Handling

1. **Error types**: Define domain-specific errors in `dto/error_code.go`:

   ```go
   var ErrorSomethingNotFound = errors.Base("Something not found")
   var ErrorSomethingAlreadyExists = errors.Base("Something already exists")
   ```

2. **Error wrapping**: Use gitlab.com/tozd/go/errors for error wrapping:

   ```go
   if err != nil {
       return nil, errors.Wrap(err, "failed to get something")
   }
   ```

3. **HTTP error mapping**: Map domain errors to HTTP status codes:

   ```go
   if errors.Is(err, dto.ErrorSomethingNotFound) {
       return nil, huma.Error404NotFound(err.Error())
   }
   if errors.Is(err, dto.ErrorSomethingAlreadyExists) {
       return nil, huma.Error409Conflict(err.Error())
   }
   ```

### Testing Conventions

1. **Test suite structure**: Use testify/suite for all tests:

   ```go
   type SomethingHandlerSuite struct {
       suite.Suite
       app              *fxtest.App
       handler          *api.SomethingHandler
       mockService      service.SomethingServiceInterface
       ctx              context.Context
       cancel           context.CancelFunc
   }
   ```

2. **SetupTest pattern**: Use FX dependency injection in tests:

   ```go
   func (suite *SomethingHandlerSuite) SetupTest() {
       suite.app = fxtest.New(suite.T(),
           fx.Provide(
               func() *matchers.MockController { return mock.NewMockController(suite.T()) },
               func() (context.Context, context.CancelFunc) {
                   return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
               },
               api.NewSomethingHandler,
               mock.Mock[service.SomethingServiceInterface],
               func() *dto.ContextState {
                   return &dto.ContextState{
                       ReadOnlyMode:    false,
                       Heartbeat:       1,
                       DockerInterface: "hassio",
                       DockerNet:       "172.30.32.0/23",
                   }
               },
           ),
           fx.Populate(&suite.handler),
           fx.Populate(&suite.mockService),
           fx.Populate(&suite.ctx),
           fx.Populate(&suite.cancel),
       )
       suite.app.RequireStart()
   }
   ```

3. **TearDownTest pattern**: Always clean up resources:

   ```go
   func (suite *SomethingHandlerSuite) TearDownTest() {
       if suite.cancel != nil {
           suite.cancel()
           suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
       }
       suite.app.RequireStop()
   }
   ```

4. **HTTP test pattern**: Use humatest for HTTP endpoint testing:

   ```go
   func (suite *SomethingHandlerSuite) TestGetSomething() {
       _, api := humatest.New(suite.T())
       suite.handler.RegisterSomethingHandler(api)

       mock.When(suite.mockService.GetSomething(mock.Any[string]())).ThenReturn(expected, nil)

       resp := api.Get("/something/123")
       suite.Equal(http.StatusOK, resp.Code)

       var result dto.Something
       err := json.Unmarshal(resp.Body.Bytes(), &result)
       suite.Require().NoError(err)
       suite.Equal(expected, result)
   }
   ```

5. **Test runner**: Always include the test runner:

   ```go
   func TestSomethingHandlerSuite(t *testing.T) {
       suite.Run(t, new(SomethingHandlerSuite))
   }
   ```

### Dependency Injection with FX

1. **Service registration**: Register services as FX providers:

   ```go
   fx.Provide(
       server.AsHumaRoute(api.NewSomethingHandler),
       service.NewSomethingService,
       repository.NewSomethingRepository,
   )
   ```

2. **Parameter structs**: Use parameter structs for dependency injection:

   ```go
   type SomethingParams struct {
       fx.In
       SomeService  service.SomeServiceInterface
       SomeRepo     repository.SomeRepositoryInterface
   }
   ```

### Data Transfer and Conversion

1. **DTO definitions**: Define DTOs with JSON tags and validation:

   ```go
   type Something struct {
       ID   string `json:"id"`
       Name string `json:"name" validate:"required,min=1,max=100"`
   }
   ```

2. **Converter pattern**: Use goverter for automatic conversions:

   ```go
   // goverter:converter
   // goverter:output:file ./converter_gen.go
   type SomethingConverter interface {
       DtoToDbom(source dto.Something) dbom.Something
       DbomToDto(source dbom.Something) dto.Something
   }
   ```

### Async Operations and Broadcasting

1. **Dirty state tracking**: Mark data as dirty after modifications:

   ```go
   self.dirtyService.SetDirtySomething()
   ```

2. **Async notifications**: Use goroutines for async operations:

   ```go
   go self.someService.NotifyClient()
   ```

3. **Context with WaitGroup**: Use context with WaitGroup for async lifecycle:

   ```go
   context.WithValue(context.Background(), "wg", &sync.WaitGroup{})
   ```

### Huma API Framework Usage

1. **Operation tags**: Use consistent operation tags for API grouping:

   ```go
   huma.OperationTags("system"), huma.OperationTags("share"), huma.OperationTags("user")
   ```

2. **AutoPatch**: Enable autopatch for PATCH operations:

   ```go
   autopatch.AutoPatch(api)
   ```

3. **Response structures**: Use anonymous structs for responses:

   ```go
   return &struct{ Body dto.Something }{Body: result}, nil
   ```

### Logging and Observability

1. **Structured logging**: Use slog for structured logging:

   ```go
   slog.Error("Failed to create something", "name", input.Body.Name, "error", err)
   slog.Debug("Processing request", "id", input.ID)
   ```

2. **Error context**: Provide context in error messages:

   ```go
   return nil, errors.Wrapf(err, "failed to create something %s", input.Body.Name)
   ```

### Configuration and Context

1. **Context state**: Use dto.ContextState for application configuration:

   ```go
   type ContextState struct {
       ReadOnlyMode    bool
       ProtectedMode   bool
       SecureMode      bool
       DockerInterface string
       DockerNet       string
       Heartbeat       int
   }
   ```

2. **Template handling**: Load templates into context state:

   ```go
   sharedResources.Template, err = os.ReadFile("../templates/smb.gtpl")
   ```

### Mock and Testing Utilities

1. **Mockio usage**: Use mockio/v2 for mocking:

   ```go
   mock.When(suite.mockService.Method(mock.Any[Type]())).ThenReturn(expected, nil)
   mock.Verify(suite.mockService, matchers.Times(1)).Method()
   ```

2. **Test data paths**: Use relative paths for test data:

   ```go
   "../../test/data/config.json"
   "../templates/smb.gtpl"
   ```

## Code Quality Guidelines

1. **Self receivers**: Use `self` as the receiver name for methods
2. **Interface naming**: End interface names with `Interface`
3. **Constructor naming**: Use `NewTypeName` pattern for constructors
4. **Error handling**: Always handle errors explicitly, don't ignore them
5. **Thread safety**: Use mutexes for shared state in repositories
6. **Resource cleanup**: Always implement proper cleanup in teardown methods
7. **Documentation**: Provide comprehensive godoc comments for exported functions
8. **Validation**: Validate input parameters at API boundaries
9. **Consistent formatting**: Use gofmt and follow Go conventions
10. **Import aliases**: Use consistent import aliases (e.g., `errors` for `gitlab.com/tozd/go/errors`)

### Build and Deployment Conventions

1. **Binary output structure**: Follow the established build destination patterns:

   ```t
   backend/
   ├── dist/           # Production builds (CGO_ENABLED=0, optimized)
   │   ├── x86_64/     # AMD64 architecture binaries
   │   ├── armv7/      # ARM v7 architecture binaries
   │   └── aarch64/    # ARM64 architecture binaries
   └── tmp/            # Development/test builds (with debug symbols)
       ├── srat        # Quick development binary
       └── x86_64/     # Test builds for x86_64
           ├── srat-cli      # CLI tool binary
           ├── srat-openapi  # OpenAPI generator binary
           └── srat-server   # Main server binary
   ```

2. **Build targets**: Use the established build destinations:
   - **Pxroduction builds**: `./dist/${ARCH}/` for optimized, stripped binaries
   - **Development builds**: `./tmp/` for development and testing
   - **Architecture-specific**: Always organize by architecture (x86_64, armv7, aarch64)

3. **Binary naming convention**: Follow the pattern:
   - `srat-server` - Main HTTP server application
   - `srat-cli` - Command-line interface tool
   - `srat-openapi` - OpenAPI documentation generator
   - `srat` - Quick development binary (tmp builds only)

4. **Makefile integration**: When creating new binaries:

   ```makefile
   # Production build to dist/${ARCH}/
   CGO_ENABLED=0 $(AARGS) go build -C $(SRC_DIRS) -tags=embedallowed
     -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION)"
     -o ../dist/$(ARCH)/ ./...

   # Development build to tmp/
   CGO_ENABLED=0 go build -C $(SRC_DIRS) -tags=embedallowed_no
     -gcflags=all="-N -l" -o ../tmp/ ./cmd/binary-name
   ```

5. **Multi-architecture support**: Always consider these architectures:
   - `GOARCH=amd64` → `dist/x86_64/`
   - `GOARCH=arm GOARM=7` → `dist/armv7/`
   - `GOARCH=arm64` → `dist/aarch64/`

6. **Build flags consistency**:
   - **Production**: `-tags=embedallowed -ldflags="-s -w"` (stripped, optimized)
   - **Development**: `-tags=embedallowed_no -gcflags=all="-N -l"` (debug symbols)
   - **Version injection**: Always include version, commit hash, and build timestamp

7. **Static asset embedding**: Frontend assets are embedded in binaries:
   - Static files built to `./src/web/static/`
   - Embedded using Go embed directives with `embedallowed` tag
   - Development builds use `embedallowed_no` to skip embedding

## Markdown Documentation Rules

### File Updates

- **Always update CHANGELOG.md** when making significant changes to features, bug fixes, or breaking changes
- **Update version-specific documentation** when API changes are made
- **Maintain README.md accuracy** when project structure, installation, or usage changes
- **Update implementation documentation** (IMPLEMENTATION\_\*.md files) when architectural changes are made

### Content Standards

- Use proper heading hierarchy (# > ## > ### > ####)
- Include code examples in triple backticks with language specification
- Add table of contents for documents longer than 10 sections
- Use consistent bullet points (- instead of \* or +)
- Include links with descriptive text rather than raw URLs
- End files with a single newline character

### Documentation Quality

- **Keep documentation current**: Update docs when code changes
- **Be specific**: Use concrete examples rather than vague descriptions
- **Include context**: Explain why changes were made, not just what changed
- **Test examples**: Ensure all code examples are functional and accurate
- **Cross-reference**: Link related documentation appropriately

## Code Documentation Rules

### API Documentation

- Update OpenAPI specs when API endpoints change
- Include request/response examples in API documentation
- Document error codes and their meanings
- Update Huma v2 documentation when handlers change

### Go Code Documentation

- Follow Go documentation conventions with proper godoc comments
- Document exported functions, types, and variables
- Include usage examples for complex functions
- Update test documentation when test patterns change

### Frontend Documentation

- Update component documentation when React components change
- Document TypeScript interfaces and types
- Include prop documentation for components
- Update build/deployment documentation when scripts change

## Workflow and CI/CD Rules

### Validation Requirements

- All Markdown files must pass linting (markdownlint)
- Links in documentation must be valid and accessible
- Code examples in documentation must be syntactically correct
- Documentation must follow project style guide

### Automated Updates

- Version numbers in documentation should match release versions
- Badge URLs and status indicators should be current
- Example configurations should reflect current schema
- Installation instructions should match current process

## Project-Specific Rules



### SRAT Architecture

- Update backend documentation when Go services change
- Update frontend documentation when React/TypeScript changes occur
- Maintain Home Assistant integration documentation
- Keep Docker configuration documentation current

### File-Specific Rules

#### README.md

- Update badges when repository status changes
- Update installation instructions for version changes
- Update feature list when capabilities change
- Update sponsor information as needed

#### CHANGELOG.md

- **Update timing**: Always update CHANGELOG.md AFTER validating changes work correctly:
  1. Make code changes and test thoroughly
  2. Validate API endpoints, build processes, or feature functionality
  3. Run tests and ensure CI passes
  4. Then document the validated changes in CHANGELOG.md

- **Change significance criteria**: Update CHANGELOG.md for these types of changes:
  - **API changes**: New endpoints, modified request/response schemas, authentication changes
  - **Breaking changes**: Changes that require user action or break existing functionality
  - **New features**: User-facing functionality additions (new handlers, services, UI components)
  - **Bug fixes**: Corrections that affect user experience or system behavior
  - **Security updates**: Authentication, authorization, or vulnerability fixes
  - **Build/deployment changes**: Makefile updates, Docker changes, architecture modifications
  - **Database migrations**: Schema changes or data model updates

- **Documentation format**: Follow the project's established emoji-based format:

  ```markdown
  ## 2025.08.\* [ 🚧 Unreleased ]

  ### ✨ Features

  - New share management API endpoints [#123](https://github.com/dianlight/srat/issues/123)
  - Support for ARM64 architecture builds
  - Manage recycle bin option for shares

  #### **🚧 Work in progress**

  - [ ] Help screen or overlay help/tour [#82](https://github.com/dianlight/srat/issues/82)
  - [x] Smart Control [#100](https://github.com/dianlight/srat/issues/100)

  ### 🐛 Bug Fixes

  - Fix share enable/disable functionality not working as expected
  - Resolved memory leak in SSE broadcasting [#456](https://github.com/dianlight/srat/issues/456)
  - Fix admin user renaming issues requiring full addon reboot

  ### 🏗 Chore

  - Updated Huma v2 framework integration
  - Modified binary output structure in Makefile
  - Implement watchdog functionality
  ```

- **Required information**: Include for each entry:
  - Clear description of what changed and why
  - Issue/PR references in format `[#123](https://github.com/dianlight/srat/issues/123)`
  - External issue references like `dianlight/hassio-addons#448`
  - Migration steps for breaking changes
  - Version compatibility notes when relevant

- **Project-specific formatting conventions**:
  - Use date-based versioning: `2025.08.*` for unreleased, `2025.06.1-dev.801` for releases
  - Status indicators: `[ 🚧 Unreleased ]`, `[ 🧪 Pre-release ]`
  - Work in progress sections with checkboxes: `- [ ]` (todo), `- [x]` (completed)
  - `[W]` prefix for items marked as "Work in progress"
  - Categorize by: `✨ Features`, `🐛 Bug Fixes`, `🏗 Chore`
  - Group similar items and use "Work in progress" subsections for ongoing tasks

- **Validation before documentation**: Ensure these validations pass before updating CHANGELOG.md:
  - All tests pass (`make test`, `make test_ci`)
  - Builds succeed for all target architectures
  - API endpoints return expected responses
  - Frontend builds and integrates correctly
  - Documentation links and examples work

#### Implementation Docs

- Update IMPLEMENTATION\_\*.md when architectural decisions change
- Include reasoning for technical choices
- Maintain decision records for future reference
- Update when dependencies or integrations change

## Quality Gates

### Before Merge

- All documentation must be spell-checked
- Links must be validated
- Code examples must be tested
- Version references must be current

### Review Requirements

- Documentation changes require review from maintainers
- Breaking changes must include migration documentation
- New features must include user-facing documentation
- API changes must include updated OpenAPI specs

## Automation Guidelines

### GitHub Actions Integration

- Documentation validation runs on all PR branches
- Link checking occurs on schedule and PR events
- Version consistency is verified across all files
- Style guide compliance is enforced automatically

### Tool Configuration

- Use markdownlint for consistent formatting
- Use link-check for URL validation
- Use spell-check for content quality
- Use prettier for consistent formatting where applicable
