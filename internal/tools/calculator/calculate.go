// * generate by claude sonnet 4.5
package calculator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
)

func Calc(expression string) (string, error) {
	if expression == "" {
		return "", fmt.Errorf("expression is required")
	}

	parseExpr, err := parser.ParseExpr(expression)
	if err != nil {
		return "", fmt.Errorf("parser.ParseExpr: %w", err)
	}

	result, err := eval(parseExpr)
	if err != nil {
		return "", err
	}

	if result == math.Trunc(result) && !math.IsInf(result, 0) {
		return strconv.FormatInt(int64(result), 10), nil
	}
	return strconv.FormatFloat(result, 'f', -1, 64), nil
}

func eval(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		if n.Kind != token.INT && n.Kind != token.FLOAT {
			return 0, fmt.Errorf("unsupported literal type: %s", n.Kind)
		}
		v, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", n.Value)
		}
		return v, nil

	case *ast.UnaryExpr:
		v, err := eval(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -v, nil
		case token.ADD:
			return v, nil
		}
		return 0, fmt.Errorf("unsupported unary operator: %s", n.Op)

	case *ast.BinaryExpr:
		left, err := eval(n.X)
		if err != nil {
			return 0, err
		}
		right, err := eval(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case token.REM:
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			return math.Mod(left, right), nil
		case token.XOR:
			return math.Pow(left, right), nil
		}
		return 0, fmt.Errorf("unsupported operator: %s", n.Op)

	case *ast.ParenExpr:
		return eval(n.X)

	case *ast.CallExpr:
		return evalFunc(n)
	}

	return 0, fmt.Errorf("unsupported expression type: %T", node)
}

func evalFunc(call *ast.CallExpr) (float64, error) {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return 0, fmt.Errorf("unsupported function call")
	}

	if len(call.Args) == 0 {
		return 0, fmt.Errorf("function %s requires arguments", ident.Name)
	}

	arg, err := eval(call.Args[0])
	if err != nil {
		return 0, err
	}

	switch ident.Name {
	case "sqrt":
		if arg < 0 {
			return 0, fmt.Errorf("sqrt of negative number")
		}
		return math.Sqrt(arg), nil
	case "abs":
		return math.Abs(arg), nil
	case "ceil":
		return math.Ceil(arg), nil
	case "floor":
		return math.Floor(arg), nil
	case "round":
		return math.Round(arg), nil
	case "log":
		if arg <= 0 {
			return 0, fmt.Errorf("log of non-positive number")
		}
		return math.Log(arg), nil
	case "log2":
		if arg <= 0 {
			return 0, fmt.Errorf("log2 of non-positive number")
		}
		return math.Log2(arg), nil
	case "log10":
		if arg <= 0 {
			return 0, fmt.Errorf("log10 of non-positive number")
		}
		return math.Log10(arg), nil
	case "sin":
		return math.Sin(arg), nil
	case "cos":
		return math.Cos(arg), nil
	case "tan":
		return math.Tan(arg), nil
	case "pow":
		if len(call.Args) < 2 {
			return 0, fmt.Errorf("pow requires 2 arguments")
		}
		exp, err := eval(call.Args[1])
		if err != nil {
			return 0, err
		}
		return math.Pow(arg, exp), nil
	}

	return 0, fmt.Errorf("unknown function: %s", ident.Name)
}
