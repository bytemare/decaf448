// SPDX-License-Group: MIT
//
// Copyright (C) 2022 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

// Package decaf448 implements the Decaf448 group of prime order
//
//		l = 2^446 - 13818066809895115352007386748515426880336692474882178609894547503885
//
// as specified in https://datatracker.ietf.org/doc/draft-irtf-cfrg-ristretto255-decaf448.
package decaf448

import (
	"errors"
	"math/big"
)

type DecafElement struct {
	p Point
}

func NewGroupElement() *DecafElement {
	var e DecafElement
	return &e
}

var (
	oneMinusD, _     = newElement().SetString("39082", 10)
	oneMinusTwoD, _  = newElement().SetString("78163", 10)
	sqrtMinusD, _    = newElement().SetString("98944233647732219769177004876929019128417576295529901074099889598043702116001257856802131563896515373927712232092845883226922417596214", 10)
	invSqrtMinusD, _ = newElement().SetString("315019913931389607337177038330951043522456072897266928557328499619017160722351061360252776265186336876723201881398623946864393857820716", 10)
	// D = -39081
	D, _ = newElement().SetString("726838724295606890549323807888004534353641360687318060281490199180612328166730772686396383698676545930088884461843637361053498018326358", 10)
)

func (e *DecafElement) Encode() []byte {
	/*
		A group element with internal representation (x0, y0, z0, t0) is
		   encoded as follows:

		   1.  Process the internal representation into a field element s as
		       follows:

		   u1 = (x0 + t0) * (x0 - t0)

		   // Ignore was_square since this is always square.
		   (_, invsqrt) = SQRT_RATIO_M1(1, u1 * ONE_MINUS_D * x0^2)

		   ratio = CT_ABS(invsqrt * u1 * SQRT_MINUS_D)
		   u2 = INVSQRT_MINUS_D * ratio * z0 - t0
		   s = CT_ABS(ONE_MINUS_D * invsqrt * x0 * u2)

		   2.  Return the 56-byte little-endian encoding of s.

		   Note that decoding and then re-encoding a valid group element will
		   yield an identical byte string.
	*/

	var u1, u2, ratio, s Element
	u1.Add(&e.p.X, &e.p.T)
	u2.Subtract(&e.p.X, &e.p.T)
	u1.Multiply(&u1, &u2)

	u2.Square(&e.p.X)
	u2.Multiply(&u2, oneMinusD)
	u2.Multiply(&u2, &u1)
	_, invsqrt := newElement().SqrtRatio(one, &u2)

	ratio.Multiply(invsqrt, &u1)
	ratio.Multiply(&ratio, sqrtMinusD)
	ratio.AbsoluteCT(&ratio)

	u2.Multiply(invSqrtMinusD, &ratio)
	u2.Multiply(&u2, &e.p.Z)
	u2.Subtract(&u2, &e.p.T)

	s.Multiply(oneMinusD, invsqrt)
	s.Multiply(&s, &e.p.X)
	s.Multiply(&s, &u2)
	s.AbsoluteCT(&s)

	return reverse(s.int.Bytes())
}

func (e *DecafElement) Decode(input []byte) *DecafElement {
	/*
		All elements are encoded as a 56-byte string.  Decoding proceeds as
		   follows:

		   1.  First, interpret the string as an integer s in little-endian
		       representation.  If the length of the string is not 56 bytes, or
		       if the resulting value is >= p, decoding fails.

		       *  Note: unlike [RFC7748] field element decoding, non-canonical
		          values are rejected.  The test vectors in Appendix B.2
		          exercise these edge cases.

		   2.  If IS_NEGATIVE(s) returns TRUE, decoding fails.

		   3.  Process s as follows:

		   ss = s^2
		   u1 = 1 + ss
		   u2 = u1^2 - 4 * D * ss
		   (was_square, invsqrt) = SQRT_RATIO_M1(1, u2 * u1^2)
		   u3 = CT_ABS(2 * s * invsqrt * u1 * SQRT_MINUS_D)
		   x = u3 * invsqrt * u2 * INVSQRT_MINUS_D
		   y = (1 - ss) * invsqrt * u1
		   t = x * y

		   4.  If was_square is FALSE then decoding fails.  Otherwise, return
		       the group element represented by the internal representation (x,
		       y, 1, t).
	*/
	if len(input) != 56 {
		panic(errors.New("invalid length"))
	}

	s, _ := newElement().SetBytesLittle(input)

	if curveOrder.Compare(s) != 1 {
		panic(errors.New("out of order"))
	}

	if s.IsNegative() == 1 {
		panic(errors.New("negative"))
	}

	var ss, u1, u2, u22, u3, t, x, y Element
	four := newElement().SetInt(big.NewInt(4))

	// ss = s^2
	// u1 = 1 + ss
	ss.Square(s)
	u1.Add(&ss, one)

	// u2 = u1^2 - 4 * D * ss
	u2.Multiply(&u1, &u1)
	u22.Multiply(four, D)
	u22.Multiply(&u22, &ss)
	u2.Subtract(&u2, &u22)

	// (was_square, invsqrt) = SQRT_RATIO_M1(1, u2 * u1^2)
	u22.Multiply(&u1, &u1)
	wasSquare, invsqrt := newElement().SqrtRatio(one, u22.Multiply(&u2, &u22))

	// u3 = CT_ABS(2 * s * invsqrt * u1 * SQRT_MINUS_D)
	u3.Multiply(two, s)
	u3.Multiply(&u3, invsqrt)
	u3.Multiply(&u3, &u1)
	u3.Multiply(&u3, sqrtMinusD)
	u3.AbsoluteCT(&u3)

	// x = u3 * invsqrt * u2 * INVSQRT_MINUS_D
	x.Multiply(&u3, invsqrt)
	x.Multiply(&x, &u2)
	x.Multiply(&x, invSqrtMinusD)

	// y = (1 - ss) * invsqrt * u1
	y.Subtract(one, &ss)
	y.Multiply(&y, invsqrt)
	y.Multiply(&y, &u1)

	t.Multiply(&x, &y)

	if !(wasSquare == 1) {
		panic(errors.New("not square"))
	}

	e.p.X.Set(&x)
	e.p.Y.Set(&y)
	e.p.T.Set(&t)
	e.p.Z.Set(one)

	return e
}

func (e *DecafElement) OneWayMap(input []byte) *DecafElement {
	v := make([]byte, len(input))
	copy(v, input)
	v = reverse(v)

	p1 := _map(v[:56])
	p2 := _map(v[56:112])
	e.p.Set(p1.Add(p2))

	return e
}

func _map(input []byte) *Point {
	/*
		The MAP function is defined on a 56-byte string as:

		   1.  Interpret the string as an integer r in little-endian
		       representation.  Reduce r modulo p to obtain a field element t.

		       *  Note: similarly to [RFC7748] field element decoding, and
		          unlike field element decoding in Section 5.3.1, non-canonical
		          values are accepted.

		   2.  Process t as follows:

		   r = -t^2
		   u0 = d * (r-1)
		   u1 = (u0 + 1) * (u0 - r)

		   (was_square, v) = SQRT_RATIO_M1(ONE_MINUS_TWO_D, (r + 1) * u1)
		   v_prime = CT_SELECT(v IF was_square ELSE t * v)
		   sgn     = CT_SELECT(1 IF was_square ELSE -1)
		   s = v_prime * (r + 1)

		   w0 = 2 * CT_ABS(s)
		   w1 = s^2 + 1
		   w2 = s^2 - 1
		   w3 = v_prime * s * (r - 1) * ONE_MINUS_TWO_D + sgn

		   3.  Return the group element represented by the internal
		       representation (w0*w3, w2*w1, w1*w3, w0*w2).
	*/

	r, _ := newElement().SetBytesBig(input)
	t := newElement().reduce(&r.int, &curveOrder.int)

	var u0, u01, u0r, u1, rMinOne, rPlusOne Element

	// r = -t^2
	//	   u0 = d * (r-1)
	//	   u1 = (u0 + 1) * (u0 - r)
	r.Square(t)
	r.Negate(r)
	rMinOne.Subtract(r, one)
	u0.Multiply(D, &rMinOne)
	u01.Add(&u0, one)
	u0r.Subtract(&u0, r)
	u1.Multiply(&u01, &u0r)

	// (was_square, v) = SQRT_RATIO_M1(ONE_MINUS_TWO_D, (r + 1) * u1)
	//	   v_prime = CT_SELECT(v IF was_square ELSE t * v)
	//	   sgn     = CT_SELECT(1 IF was_square ELSE -1)
	//	   s = v_prime * (r + 1)
	var vPrime, sgn, s Element
	rPlusOne.Add(r, one)
	u1.Multiply(&u1, &rPlusOne)
	wasSquare, v := newElement().SqrtRatio(oneMinusTwoD, &u1)
	vPrime.SelectCT(v, newElement().Multiply(t, v), wasSquare)
	sgn.SelectCT(one, minusOne, wasSquare)
	s.Multiply(&vPrime, &rPlusOne)

	// w0 = 2 * CT_ABS(s)
	//	   w1 = s^2 + 1
	//	   w2 = s^2 - 1
	//	   w3 = v_prime * s * (r - 1) * ONE_MINUS_TWO_D + sgn
	var w0, w1, w2, w3 Element
	w0.Multiply(two, newElement().AbsoluteCT(&s))
	w1.Square(&s)
	w1.Add(&s, one)
	w2.Square(&s)
	w2.Subtract(&w2, one)
	w3.Multiply(&vPrime, &s)
	w3.Multiply(&w3, &rMinOne)
	w3.Multiply(&w3, oneMinusTwoD)
	w3.Add(&w3, &sgn)

	var p Point
	p.X.Multiply(&w0, &w3)
	p.Y.Multiply(&w2, &w1)
	p.T.Multiply(&w0, &w2)
	p.Z.Multiply(&w1, &w3)

	return &p
}
