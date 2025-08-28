# Introduction
This tutorial explains in more details the `Helloworld` CLI created in the first tutorial.

It is very common to create a static binary CLI for a whole Golang project. Actually, both Kubernetes and Docker are built this way, with several subcommands to start different servers and interacting with the servers from a client. We are going to stick to this approach. Having a monolithic binary makes life easier and a complex distributed system easier to use and manage. You can still use different subcommands to deploy several microservices or manage a whole distributed system.

## Project Directory Structure
This project follows the [golang-standards/project-layout](https://github.com/golang-standards/project-layout) conventions for organizing Go applications.

| Directory/File | Purpose | Description |
|----------------|---------|-------------|
| `bin/` | Compiled binaries | Contains executable files built from your application code |
| `cmd/` | Application entry points | Main applications for this project. The directory name should match the name of the executable |
| `internal/` | Private application code | Application code that you don't want others importing in their applications or libraries |
| `pkg/` | Library code | Library code that's ok to use by external applications (e.g., `/pkg/mypubliclib`) |
| `Dockerfile` | Container configuration | Docker build instructions for containerizing the application |
| `go.mod` | Module definition | Go module file that defines the module path and dependency requirements |
| `go.sum` | Dependency checksums | Contains cryptographic checksums of module dependencies for verification |
| `Makefile` | Build automation | Build script with common tasks like testing, building, and cleaning |
| `README.md` | Project documentation | Main project documentation with setup and usage instructions |
| `rename-project.sh` | Setup script | Shell script to rename project references from template to your project |

# Directory Structure Details
**`/cmd`**
Main applications for this project. The directory name for each application should match the name of the executable you want to have. Don't put a lot of code in the application directory. If you think the code can be imported and used in other projects, then it should live in the `/pkg` directory.

**`/internal`**
Private application and library code. This is the code you don't want others importing in their applications or libraries. Note that this layout pattern is enforced by the Go compiler itself.

**`/pkg`**
Library code that's ok to use by external applications. Other projects will import these libraries expecting them to work, so think twice before you put something here. Note that the `internal` directory is a better way to ensure your private packages are not importable because it's enforced by Go.

**`/bin`**
Directory for compiled application binaries. This directory is typically added to `.gitignore` since it contains build artifacts.

The remaining files (`go.mod`, `go.sum`, `Makefile`, `README.md`, `Dockerfile`) are standard project files that provide module management, build automation, documentation, containerization.

# Main Entry Point
The main entry point for our CLI application is located in cmd/main.go:

```go
package main

import (
    "github.com/johankristianss/d7024e-tutorial/internal/cli"
    "github.com/johankristianss/d7024e-tutorial/pkg/build"
)

var (
    BuildVersion string = ""
    BuildTime    string = ""
)

func main() {
    build.BuildVersion = BuildVersion
    build.BuildTime = BuildTime
    cli.Execute()
}
```

As can be seen, it looks quite empty. The `BuildVersion` and `BuildTime` variable is filled in by the compiler. This is very useful for keeping track which version of the CLI is being used. We will use the [Cobra](https://cobra.dev) framework to create the acutal CLI. Cobra is a very popular framework in the Golang open source community is for example used by both Docker and Kubernetes.

The `cli.Execute()` bootstraps the Cobra.

# Understanding Cobra
Let's look at the `Execute` function definition in the `internal/cli/root.go` file. The `Execute` function is called by the `cmd/main.go` mentioned above.

```go
package cli

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

const TimeLayout = "2006-01-02 15:04:05"

var Verbose bool

func init() {
    rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose
output")
}

var rootCmd = &cobra.Command{
    Use:   "helloworld",
    Short: "helloworld",
    Long:  "helloworld",
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

```
The `Execute` function initializes the Cobra command framework and handles the CLI execution flow. Here's how it works:

The `rootCmd` variable creates the foundation of our CLI by defining a `cobra.Command` struct. This establishes `helloworld` as the base command that users will invoke. The `init()` function runs automatically when the package loads, setting up a persistent `--verbose` flag that becomes available across all commands and subcommands in the CLI hierarchy.

The `Verbose` boolean variable is bound to the `-v/--verbose` flag using `PersistentFlags().BoolVarP()`, demonstrating how Cobra manages command-line arguments and makes them accessible throughout the application.


The `internal/cli/talk.go` file below demonstrates how to implement a subcommand:

```go
package cli

import (
    "github.com/johankristianss/d7024e-tutorial/pkg/helloworld"
    "github.com/spf13/cobra"
)

func init() {
    rootCmd.AddCommand(TalkCmd)
}

var TalkCmd = &cobra.Command{
    Use:   "talk",
    Short: "Say something",
    Long:  "Say something",
    Run: func(cmd *cobra.Command, args []string) {
        hellworld := helloworld.NewHelloWorld()
        hellworld.Talk()

}
```

This subcommand architecture demonstrates several key Cobra patterns:

The init() function automatically registers the `TalkCmd` with the root command using `rootCmd.AddCommand()`, creating the command hierarchy that enables `./bin/helloworld talk`. 

TalkCmd defines the subcommand's interface, including usage text and help information that appears when users run --help. The `Run` function serves as a thin wrapper that delegates actual work to the `pkg/helloworld` package, maintaining clean separation between CLI concerns and application logic. This pattern of creating a new instance `helloworld.NewHelloWorld())` and calling its methods promotes testability and modularity in the codebase. Let's now explain how `helloworld` package works. 

# Understanding the Helloworld package
Let's open `pkg/helloworld/helloworld.go`. The code below should be almost self-explanatory. The [Logrus](https://github.com/sirupsen/logrus) logging framework is a powerful loging framerwork that you can use instead of just standard printout using `fmt.Println()`. Instead of plain text output, Logrus allows you to log structured data with key-value pairs, making logs easier to parse and analyze programmatically.

```go
package helloworld

import (
    "errors"
    "fmt"

    log "github.com/sirupsen/logrus"
)

type HelloWorld struct {
    msg string
}

func NewHelloWorld() *HelloWorld {
    err := errors.New("This is an error")

    if err != nil {
        log.WithFields(log.Fields{"Error": err}).Error("Error detected")
    }

    return &HelloWorld{
        msg: "Hello, World!",
    }
}

func (hello *HelloWorld) Talk() {
    log.WithFields(log.Fields{"Msg": hello.msg, "OtherMsg": "Logging is cool!"}).Info("Talking...")
    fmt.Println(hello.msg)
}
```

### Testing
All Golang funciton should be tested. A common pattern is to add a `_test.go` prefix to the source file being tested, for example `helloworld_test.go` will test functions defined in `helloworld.go`.  

```go
package helloworld

import (
    "testing"
)

func TestNewHelloWorld(t *testing.T) {
    hw := NewHelloWorld()

    expectedMessage := "Hello, World!"
    if hw.msg != expectedMessage {
        t.Errorf("expected msg to be %q, but got %q", expectedMessage, hw.msg)
    }
}
```

Above a simple test for `NewHelloWorld` function. To run all tests in the `helloworld` package, type:

```console
go test -v
```

```console
=== RUN   TestNewHelloWorld
ERRO[0000] Error detected                                Error="This is an error"
--- PASS: TestNewHelloWorld (0.00s)
PASS
ok  	github.com/johankristianss/d7024e-tutorial/pkg/helloworld	0.001s
```

To run a single test, type:
```console
go test -v -test.run TestNewHelloWorld
```

You can also test the entire project by typing `make test` in the project root. Remember to update the `Makefile`if you add more packages.

# Advantage of this Golang structure
This project structure offers several significant benefits for developing maintainable and scalable Go applications. 

## Separation of Concerns
**Clear Package Boundaries**: The structure enforces clean separation between CLI handling `internal/cli`, application logic `pkg/helloworld`, and application entry points (`cmd/main.go`). This makes the codebase easier to understand and modify.

**Testability**: By isolating application logic in the `pkg/` directory, you can test core functionality independently of CLI concerns. The `HelloWorld` struct can be tested without involving command-line parsing or user interface elements.

**Reusability**: Code in the `pkg/` directory can be imported and reused by other projects, while `internal/` code remains private to your application.

## Scalability and Maintainability
**Modular Architecture**: Adding new subcommands becomes straightforward. Simply create a new file in `internal/cli/` and register it with the root command. Each command remains self-contained and seperate from other subcommands.

**Version Management**: The build-time injection of `BuildVersion` and `BuildTime` variables provides excellent traceability for deployments and debugging in production environments.

**Standard Layout**: Following the `golang-standards/project-layout` conventions means other Go developers can quickly understand your project structure and contribute effectively.

## Operational Benefits
**Single Binary Deployment**: The monolithic CLI approach means you only need to manage one executable file, simplifying deployment, distribution, and dependency management across different environments.

**Consistent Interface**: All functionality is accessible through a unified command interface with consistent help documentation, flag handling, and error reporting thanks to Cobra.

**Production Ready**: The structure includes essential production features like structured logging with Logrus, proper error handling, and comprehensive testing patterns.

## Development Workflow Advantages
**Clear Testing Strategy**: The `_test.go` naming convention and package structure make it obvious where tests belong and what they're testing.

**Build Automation**: The `Makefile` provides standardized commands for common development tasks, ensuring consistent builds and testing across team members.

**Easy Extension**: Whether you need to add new application logic, CLI commands, or external integrations, the structure provides clear places for each type of code.

This architecture scales well from simple CLI tools to complex distributed systems, as demonstrated by its adoption in major projects like Kubernetes and Docker.
