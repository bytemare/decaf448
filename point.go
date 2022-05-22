// SPDX-License-Group: MIT
//
// Copyright (C) 2022 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package decaf448

import "math/big"

type projP2 struct {
	x, y, z Element
}

func (p *projP2) fromExtended(q *Point) *projP2 {
	p.x.Set(&q.X)
	p.y.Set(&q.Y)
	p.z.Set(&q.Z)

	return p
}

type Point struct {
	/*
		Extended Twisted Edwards Coordinates System

		"Twisted Edwards Curves Revisited" - 2008
		https://link.springer.com/content/pdf/10.1007/978-3-540-89255-7_20.pdf
	*/
	X, Y, T, Z Element
}

func (p *Point) fromP2(q *projP2) *Point {
	p.X.Multiply(&q.x, &q.z)
	p.Y.Multiply(&q.y, &q.y)
	p.T.Multiply(&q.x, &q.y)
	p.Z.Square(&q.z)

	return p
}

func pZero() *Point {
	var p Point
	p.X.SetInt(big.NewInt(0))
	p.Y.SetInt(big.NewInt(1))
	p.T.SetInt(big.NewInt(0))
	p.Z.SetInt(big.NewInt(1))

	return &p
}

func (p *Point) Set(q *Point) *Point {
	p.X.Set(&q.X)
	p.Y.Set(&q.Y)
	p.T.Set(&q.T)
	p.Z.Set(&q.Z)

	return p
}

func (p *Point) Negate(q *Point) *Point {
	p.X.Negate(&q.X)
	p.Y.Set(&q.Y)
	p.T.Negate(&q.T)
	p.Z.Set(&q.Z)

	return p
}

func (p *Point) Subtract(q *Point) *Point {
	var minusq Point
	return p.Add(minusq.Negate(q))
}

func (p *Point) IsEqual(q *Point) int {
	var f0, f1 Element

	f0.Multiply(&p.X, &q.Y)
	f1.Multiply(&p.Y, &q.X)
	res := f0.IsEqualCT(&f1)

	f0.Multiply(&p.Y, &q.Y)
	f1.Multiply(&p.X, &q.X)
	res = res | f0.IsEqualCT(&f1)

	return res
}

func (p *Point) IsInfinity() int {
	return p.IsEqual(pZero())
}

func (p *Point) Copy() *Point {
	var q Point
	q.X.Set(&p.X)
	q.Y.Set(&p.Y)
	q.T.Set(&p.T)
	q.Z.Set(&p.Z)

	return &q
}

// q = l = 2^446 - 13818066809895115352007386748515426880336692474882178609894547503885
// = 181709681073901722637330951972001133588410340171829515070372549795146003961539585716195755291692375963310293709091662304773755859649779
// h = 4
const orderPrime = "181709681073901722637330951972001133588410340171829515070372549795146003961539585716195755291692375963310293709091662304773755859649779"

var groupOrder, _ = newElement().SetString(orderPrime, 10)

func (p *Point) ScalarMult(s *Element, q *Point) *Point {
	if groupOrder.int.Cmp(&s.int) <= 0 {
		panic("scalar out of order")
	}

	r0 := pZero()
	r1 := q.Copy()
	for i := s.int.BitLen() - 1; i >= 0; i-- {
		if s.int.Bit(i) == 0 {
			r1.Add(r0)
			r0.Double()
		} else {
			r0.Add(r1)
			r1.Double()
		}
	}

	p.Set(r0)

	return p
}

func (p *Point) Double() *Point {
	/*
		The point P3 = (X3,Y3,T3,Z3) = P1 + P1 is given by

		$ A = X1^2 $
		$ B = Y1^2 $
		$ C = 2 \times Z1^2 $
		$ D = a \times A $
		$ E = (X1 + Y1)^2 - A -B $
		$ G = D + B $
		$ F = G - C $
		$ H = D - B $
		$ X3 = E \times F $
		$ Y3 = G \times H $
		$ T3 = E \times H $
		$ Z3 = F \times G $
	*/

	var a, b, c, d, e, f, g, h Element
	a.Square(&p.X)
	b.Square(&p.Y)
	c.Square(&p.Z)
	c.Multiply(two, &c)
	d.Set(&a)
	e.Add(&p.X, &p.Y)
	e.Square(&e)
	e.Subtract(&e, &a)
	e.Subtract(&e, &b)
	g.Add(&d, &b)
	f.Subtract(&g, &c)
	h.Subtract(&d, &b)

	p.X.Multiply(&e, &f)
	p.Y.Multiply(&g, &h)
	p.T.Multiply(&e, &h)
	p.Z.Multiply(&f, &g)

	return p
}

func (p *Point) Add(q *Point) *Point {
	var a, b, c, d, e, f, g, h, ee, ff Element
	a.Multiply(&p.X, &q.X) // A = x1*x2
	b.Multiply(&p.Y, &q.Y) // B = y1*y2
	c.Multiply(&q.T, &p.T) // C = d*t1*t2
	c.Multiply(&c, D)      //
	d.Multiply(&p.Z, &q.Z) // D = z1*z2
	ee.Add(&p.X, &p.Y)     // x1+y1
	ff.Add(&q.X, &q.Y)     // x2+y2
	e.Multiply(&ee, &ff)   // E = (x1+y1)*(x2+y2)-A-B
	e.Subtract(&e, &a)     //
	e.Subtract(&e, &b)     //
	f.Subtract(&d, &c)     // F = D-C
	g.Add(&d, &c)          // g = D+C
	h.Subtract(&b, &a)     // H = B-A
	p.X.Multiply(&e, &f)   // X = E * F
	p.Y.Multiply(&g, &h)   // Y = G * H
	p.T.Multiply(&e, &h)   // T = E * H
	p.Z.Multiply(&f, &g)   // Z = F * G

	return p
}

func (p *Point) Add2(q *Point) *Point {

	return p
}
