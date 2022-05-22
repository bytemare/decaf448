// SPDX-License-Group: MIT
//
// Copyright (C) 2022 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package decaf448

import (
	"crypto/rand"
	"crypto/subtle"
	"math/big"
)

const (
	// untwisted edwards curve equation: y2 + x2 ≡ 1 - 39081 x2 y2

	// p = 2^448 - 2^224 - 1
	fieldOrder = "726838724295606890549323807888004534353641360687318060281490199180612328166730772686396383698676545930088884461843637361053498018365439"
)

var (
	curveOrder, _ = newElement().SetString(fieldOrder, 10)

	zero     = newElement().SetInt(big.NewInt(0))
	one      = newElement().SetInt(big.NewInt(1))
	minusOne = newElement().Subtract(zero, one)
	two      = newElement().SetInt(big.NewInt(2))
	// (p-3)/4 = 2^446-2^222-1
	pMinus3Div4, _ = newElement().SetString("3fffffffffffffffffffffffffffffffffffffffffffffffffffffffbfffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
)

func (e *Element) expPMinus3mod4() *Element {
	return e.Exp(e, pMinus3Div4)
}

func reverse(b []byte) []byte {
	for i := len(b)/2 - 1; i >= 0; i-- {
		opp := len(b) - 1 - i
		b[i], b[opp] = b[opp], b[i]
	}

	return b
}

type Element struct {
	int big.Int
}

func newElement() *Element {
	var e Element
	return &e
}

func (e *Element) reduce(x, mod *big.Int) *Element {
	e.int.Mod(x, mod)
	return e
}

func (e *Element) Zero() *Element {
	*e = *zero
	return e
}

func (e *Element) One() *Element {
	*e = *one
	return e
}

func (e *Element) Set(u *Element) *Element {
	return e.SetInt(&u.int)
}

func (e *Element) SetInt(u *big.Int) *Element {
	e.int.Set(u)
	return e
}

func (e *Element) SetString(u string, base int) (*Element, error) {
	if _, ok := e.int.SetString(u, base); !ok {
		panic(nil)
	}

	return e, nil
}

func (e *Element) SetBytesBig(u []byte) (*Element, error) {
	e.int.SetBytes(u)
	return e, nil
}

func (e *Element) SetBytesLittle(u []byte) (*Element, error) {
	v := make([]byte, len(u))
	copy(v, u)
	e.int.SetBytes(reverse(v))
	return e, nil
}

func (e *Element) Random(order *Element) *Element {
	r, _ := rand.Int(rand.Reader, &order.int)
	e.int.Set(r)

	return e
}

func (e *Element) Bytes() []byte {
	return e.int.Bytes()
}

func (e *Element) Add(u, v *Element) *Element {
	return e.reduce(e.int.Add(&u.int, &v.int), &curveOrder.int)
}

func (e *Element) Subtract(u, v *Element) *Element {
	return e.reduce(e.int.Sub(&u.int, &v.int), &curveOrder.int)
}

func (e *Element) Multiply(u, v *Element) *Element {
	return e.reduce(e.int.Mul(&u.int, &v.int), &curveOrder.int)
}

func (e *Element) Square(u *Element) *Element {
	return e.reduce(e.int.Mul(&u.int, &u.int), &curveOrder.int)
}

func (e *Element) Negate(u *Element) *Element {
	return e.reduce(e.int.Neg(&u.int), &curveOrder.int)
}

func (e *Element) Invert(u, exp *Element) *Element {
	e.int.Exp(&u.int, &exp.int, &curveOrder.int)
	return e
}

func (e *Element) Exp(u, v *Element) *Element {
	e.int.Exp(&u.int, &v.int, &curveOrder.int)
	return e
}

func (e *Element) Compare(u *Element) int {
	return e.int.Cmp(&u.int)
}

func (e *Element) IsZero() int {
	switch e.int.Sign() {
	case 0:
		return 1
	default:
		return 0
	}
}

func (e *Element) IsNegative() int {
	switch e.int.Sign() {
	case -1:
		return 1
	default:
		return 0
	}
}

func (e *Element) IsEqualCT(u *Element) int {
	var su, sv [56]byte
	e.int.FillBytes(su[:])
	u.int.FillBytes(sv[:])
	return subtle.ConstantTimeCompare(su[:], sv[:])
}

func (e *Element) SelectCT(u, v *Element, cond int) *Element {
	// TODO: constant-time
	switch cond {
	case 1:
		e.Set(u)
	default:
		e.Set(v)
	}

	return e
}

func (e *Element) SwapCT(u *Element, condition bool) {
	// TODO: constant-time
	var v Element
	switch condition {
	case true:
		v.Set(u)
	case false:
		v.Set(e)
	}

	e.Set(&v)
}

func (e *Element) IsSquareCT() bool {
	pMinus1div2 := newElement().One()
	pMinus1div2.Subtract(curveOrder, pMinus1div2)
	pMinus1div2.int.Rsh(&pMinus1div2.int, 1)

	return e.IsEqualCT(newElement().Exp(e, pMinus1div2)) == 1
}

func (e *Element) AbsoluteCT(u *Element) *Element {
	minU := newElement().Negate(u)
	e.SelectCT(minU, u, u.IsNegative())

	return e
}

func (e *Element) SqrtRatio(u, v *Element) (wasSquare int, fe *Element) {
	/*
		SQRT_RATIO_M1(u, v) is defined as follows:

		   r = u * (u * v)^((p - 3) / 4) // Note: (p - 3) / 4 is an integer.

		   check = v * r^2
		   was_square = CT_EQ(check, u)

		   // Choose the non-negative square root.
		   r = CT_ABS(r)

		   return (was_square, r)
	*/
	var r, check Element
	r.Multiply(u, v)
	r.expPMinus3mod4()
	r.Multiply(&r, u)

	check.Square(&r)
	check.Multiply(v, &check)
	wasSquare = check.IsEqualCT(u)

	e.AbsoluteCT(&r)

	return wasSquare, e
}
