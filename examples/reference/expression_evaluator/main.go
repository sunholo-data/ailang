package main

import "fmt"

type Expr interface{ eval() int }

type Num struct{ n int }
type Add struct{ l, r Expr }
type Mul struct{ l, r Expr }

func (n Num) eval() int { return n.n }
func (a Add) eval() int { return a.l.eval() + a.r.eval() }
func (m Mul) eval() int { return m.l.eval() * m.r.eval() }

func main() {
	expr := Mul{Add{Num{3}, Num{4}}, Add{Num{2}, Num{5}}}
	fmt.Println(expr.eval())
}
