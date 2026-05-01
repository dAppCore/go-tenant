# Development

Run commands from the module directory:

```sh
cd go
GOWORK=off GOPROXY=direct GOSUMDB=off go test -count=1 -short ./...
```

Run the v0.9.0 audit from the repository root:

```sh
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```
