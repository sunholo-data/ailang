/***********************************************************************
Copyright (c) 2006-2011, Skype Limited. All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:
- Redistributions of source code must retain the above copyright notice,
this list of conditions and the following disclaimer.
- Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.
- Neither the name of Internet Society, IETF or IETF Trust, nor the
names of specific contributors, may be used to endorse or promote
products derived from this software without specific prior written
permission.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
***********************************************************************/

// Transliteration of silk/sort.c (libopus v1.6.1). Only the variant reachable
// from the decode-path NLSF chain is ported here:
// silk_insertion_sort_increasing_all_values_int16, the fall-back sort used by
// silk_NLSF_stabilize when the iterative stabilizer fails to converge. Names
// follow the C for diffability.

package silk

// silkInsertionSortIncreasingAllValuesInt16 is
// silk/sort.c silk_insertion_sort_increasing_all_values_int16: insertion sort of
// a (in place) into increasing order. value is opus_int (the input values are
// int16, so nothing is lost by widening).
func silkInsertionSortIncreasingAllValuesInt16(a []int16, L int) {
	var value int
	var i, j int

	/* Safety checks */
	// celt_assert( L > 0 );

	/* Sort vector elements by value, increasing order */
	for i = 1; i < L; i++ {
		value = int(a[i])
		for j = i - 1; (j >= 0) && (value < int(a[j])); j-- {
			a[j+1] = a[j] /* Shift value */
		}
		a[j+1] = int16(value) /* Write value */
	}
}
