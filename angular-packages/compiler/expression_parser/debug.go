package expression_parser

import "fmt"

func DebugCall(ast *Call) {
    fmt.Printf("Call Receiver: %T %v\n", ast.Receiver, ast.Receiver)
}
