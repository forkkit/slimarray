# polyarray

[![Travis](https://travis-ci.com/openacid/polyarray.svg?branch=main)](https://travis-ci.com/openacid/polyarray)
[![AppVeyor](https://ci.appveyor.com/api/projects/status/m0vvvrru7a1g4mae/branch/main?svg=true)](https://ci.appveyor.com/project/drmingdrmer/polyarray/branch/main)
![test](https://github.com/openacid/polyarray/workflows/test/badge.svg)

[![Report card](https://goreportcard.com/badge/github.com/openacid/polyarray)](https://goreportcard.com/report/github.com/openacid/polyarray)
[![Coverage Status](https://coveralls.io/repos/github/openacid/polyarray/badge.svg?branch=main&service=github)](https://coveralls.io/github/openacid/polyarray?branch=main&service=github)

[![GoDoc](https://godoc.org/github.com/openacid/polyarray?status.svg)](http://godoc.org/github.com/openacid/polyarray)
[![Sourcegraph](https://sourcegraph.com/github.com/openacid/polyarray/-/badge.svg)](https://sourcegraph.com/github.com/openacid/polyarray?badge)

PolyArray: space efficient `uint32` array.
It uses polynomial to compress and store an array.
A `uint32` costs only **5 bits** in a sorted array of a million number in range `[0, 1000*1000]`(17% of original data).

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Why](#why)
- [What It Is And What It Is Not](#what-it-is-and-what-it-is-not)
- [Install](#install)
- [Synopsis](#synopsis)
  - [Build a PolyArray](#build-a-polyarray)
- [How it works](#how-it-works)
    - [The General Idea](#the-general-idea)
    - [What It Is And What It Is Not](#what-it-is-and-what-it-is-not-1)
    - [Data Structure](#data-structure)
    - [Uncompacted Data Structures](#uncompacted-data-structures)
    - [Compact](#compact)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Why

- **Space efficient**: In a sorted array, an elt only takes about **10 bits** to
    store a 32-bit int.

```
== Memory cost stats of sorted random uint array ==

n=1000 rng=[0, 1000]:

           n: 1000
   mem_total: 824
    bits/elt: 6

n=1000000 rng=[0, 1000000]:

           n: 1000000
   mem_total: 702624
    bits/elt: 5

n=1000000 rng=[0, 1000000000]:

           n: 1000000
   mem_total: 2078304
    bits/elt: 16
```

- **Fast access**: A `Get` takes 10 ns. Run and see the benchmark: `go test . -bench=.`.

- **Adaptive**: It does not require the data to be totally sorted to compress
    it. E.g., PolyArray is perfect to store online user histogram data.


# What It Is And What It Is Not

Another space efficient data structure to store uint32 array is trie(Aka prefix
tree or radix tree). It is possible to use bitmap-based btree like structure
to reduce space(very likely in such case it provides higher compression rate).
But it requires the array to be **sorted**.

PolyArray does not have such restriction. It is more adaptive with data
layout. To achieve high compression rate, it only requires the data has a
overall trend, e.g., **roughly sorted**.

Additionally, it also accept duplicated element in the array, which
a bitmap based or tree-like data structure does not allow.

In the [ipv4-list](./example/iplist) example, we feed 450,000 ipv4 to PolyArray.
We see that PolyArray costs as small as gzip-ed data(`2.1 MB vs 2.0 MB`),
while it provides instance access to the data without decompressing it.

# Install

```sh
go get github.com/openacid/polyarray
```

# Synopsis

## Build a PolyArray

```go
package polyarray_test

import (
	"fmt"

	"github.com/openacid/polyarray"
)

func ExamplePolyArray() {

	nums := []uint32{
		0, 16, 32, 48, 64, 79, 95, 111, 126, 142, 158, 174, 190, 206, 222, 236,
		252, 268, 275, 278, 281, 283, 285, 289, 296, 301, 304, 307, 311, 313, 318,
		321, 325, 328, 335, 339, 344, 348, 353, 357, 360, 364, 369, 372, 377, 383,
		387, 393, 399, 404, 407, 410, 415, 418, 420, 422, 426, 430, 434, 439, 444,
		446, 448, 451, 456, 459, 462, 465, 470, 473, 479, 482, 488, 490, 494, 500,
		506, 509, 513, 519, 521, 528, 530, 534, 537, 540, 544, 546, 551, 556, 560,
		566, 568, 572, 574, 576, 580, 585, 588, 592, 594, 600, 603, 606, 608, 610,
		614, 620, 623, 628, 630, 632, 638, 644, 647, 653, 658, 660, 662, 665, 670,
		672, 676, 681, 683, 687, 689, 691, 693, 695, 697, 703, 706, 710, 715, 719,
		722, 726, 731, 735, 737, 741, 748, 750, 753, 757, 763, 766, 768, 775, 777,
		782, 785, 791, 795, 798, 800, 806, 811, 815, 818, 821, 824, 829, 832, 836,
		838, 842, 846, 850, 855, 860, 865, 870, 875, 878, 882, 886, 890, 895, 900,
		906, 910, 913, 916, 921, 925, 929, 932, 937, 940, 942, 944, 946, 952, 954,
		956, 958, 962, 966, 968, 971, 975, 979, 983, 987, 989, 994, 997, 1000,
	}

	a := polyarray.NewPolyArray(nums)

	fmt.Println("last elt is:", a.Get(int32(a.Len()-1)))

	st := a.Stat()
	for _, k := range []string{
		"elt_width",
		"mem_elts",
		"bits/elt"} {
		fmt.Printf("%10s : %d\n", k, st[k])
	}

	// Unordered output:
	// last elt is: 1000
	//  elt_width : 3
	//   mem_elts : 112
	//   bits/elt : 14
}
```

# How it works

package polyarray uses polynomial to compress and store an array of uint32. A
uint32 costs only 5 bits in a sorted array of a million number in range [0,
1000*1000].


### The General Idea

We use a polynomial y = a + bx + cx² to describe the overall trend of the
numbers. And for every number i we add a residual to fit the gap between y(i)
and nums[i]. E.g. If there are 4 numbers: 0, 15, 33, 50 The polynomial and
residuals are:

    y = 16x
    0, -1, 1, 2

In this case the residuals require 3 bits for each of them. To retrieve the
numbers, we evaluate y(i) and add the residual to it:

    get(0) = y(0) + 0 = 16 * 0 + 0 = 0
    get(1) = y(1) - 1 = 16 * 1 - 1 = 15
    get(2) = y(2) + 1 = 16 * 2 + 1 = 33
    get(3) = y(3) + 2 = 16 * 3 + 2 = 50


### What It Is And What It Is Not

Another space efficient data structure to store uint32 array is trie or prefix
tree or radix tree. It is possible to use bitmap-based btree like structure to
reduce space(very likely in such case it provides higher compression rate). But
it requires the array to be sorted.

PolyArray does not have such restriction. It is more adaptive with data layout.
To achieve high compression rate, it only requires the data has a overall trend,
e.g., roughly sorted, as seen in the above 4 integers examples. Additionally, it
also accept duplicated element in the array, which a bitmap based or tree-like
data structure does not allow.


### Data Structure

PolyArray splits the entire array into segments(Seg), each of which has 1024
numbers. And then it splits every segment into several spans. Every span has its
own polynomial. A span has 16*k numbers. A segment has at most 64 spans.

            seg[0]                      seg[1]
            1024 nums                   1024 nums
    |-------+---------------+---|---------------------------|...
     span[0]    span[1]
     16 nums    32 nums      ..


### Uncompacted Data Structures

A PolyArray is a compacted data structure. The original data structures are
defined as follow(assumes original user data is `nums []uint32`):

    Seg strcut {
      SpansBitmap   uint64      // describe span layout
      OnesCount     uint64      // count `1` in preceding Seg.
      Spans       []Span
    }

    Span struct {
      width         int32       // is retrieved from SpansBitmap

      Polynomial [3]double      //
      Config strcut {           //
        Offset        int32     // residual offset
        ResidualWidth int32     // number of bits a residual requires
      }
      Residuals  [width][ResidualWidth]bit // pack into PolyArray.Residuals
    }

A span stores 16*k int32 in it, where k ∈ [1, 64).

`Seg.SpansBitmap` describes the layout of Span-s in a Seg. A "1" at i-th bit and
a "1" at j-th bit means a Span stores `nums[i*16:j*16]`, e.g.:

    100101110000......
    <-- least significant bit

In the above example:

    span[0] has 16*3 nums in it.
    span[1] has 16*2 nums in it.
    span[2] has 16*1 nums in it.

`Seg.OnesCount` caches the total count of "1" in all preceding Seg.SpansBitmap.
This accelerate locating a Span in the packed field PolyArray.Polynomials .

`Span.width` is the count of numbers stored in this span. It does not need to be
stored because it can be calculated by counting the "0" between two "1" in
`Seg.SpansBitmap`.

`Span.Polynomial` stores 3 coefficients of the polynomial describing the overall
trend of this span. I.e. the `[a, b, c]` in `y = a + bx + cx²`

`Span.Config.Offset` adjust the offset to locate a residual. In a span we want
to have that:

    residual position = Config.Offset + (i%1024) * Config.ResidualWidth

But if the preceding span has smaller residual width, the "offset" could be
negative, e.g.: span[0] has residual of width 0 and 16 residuals, span[1] has
residual of width 4. Then the "offset" of span[1] is `-16*4` in order to
satisify: `(-16*4) + i * 4` is the correct residual position, for i in [16, 32).

`Span.Config.ResidualWidth` specifies the number of bits to store every residual
in this span, it must be a power of 2: `2^k`.

`Span.Residuals` is an array of residuals of length `Span.width`. Every elt in
it is a `ResidualWidth`-bits integers.


### Compact

PolyArray compact `Seg` into a dense format:

    PolyArray.Bitmap = [
      Seg[0].SpansBitmap,
      Seg[0].OnesCount,
      Seg[1].SpansBitmap,
      Seg[1].OnesCount,
      ... ]

    PolyArray.Polynomials = [
      Seg[0].Spans[0].Polynomials,
      Seg[0].Spans[1].Polynomials,
      ...
      Seg[1].Spans[0].Polynomials,
      Seg[1].Spans[1].Polynomials,
      ...
    ]

    PolyArray.Configs = [
      Seg[0].Spans[0].Config
      Seg[0].Spans[1].Config
      ...
      Seg[1].Spans[0].Config
      Seg[1].Spans[1].Config
      ...
    ]

`PolyArray.Residuals` simply packs the residuals of every nums[i] together.
