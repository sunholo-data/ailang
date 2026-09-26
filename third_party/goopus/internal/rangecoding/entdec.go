/* Copyright (c) 2001-2011 Timothy B. Terriberry
   Copyright (c) 2008-2009 Xiph.Org Foundation */
/*
   Redistribution and use in source and binary forms, with or without
   modification, are permitted provided that the following conditions
   are met:

   - Redistributions of source code must retain the above copyright
   notice, this list of conditions and the following disclaimer.

   - Redistributions in binary form must reproduce the above copyright
   notice, this list of conditions and the following disclaimer in the
   documentation and/or other materials provided with the distribution.

   THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
   ``AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
   LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
   A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER
   OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
   EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
   PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
   PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
   LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
   NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
   SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// This file transliterates celt/entdec.c: a range decoder based upon the FIFO
// arithmetic code of Martin (1979) / Moffat-Neal-Witten (1998). See entdec.c for
// the full reference notes.

package rangecoding

// Decoder is the entropy decoder context (celt/entcode.h struct ec_ctx, decoder
// view). The field layout mirrors Encoder so the same ec_tell()/ec_tell_frac()
// helpers serve both, matching C's single ec_ctx used for encode and decode.
type Decoder struct {
	// Buffered input.
	buf []byte
	// The size of the buffer.
	storage uint32
	// The offset at which the last byte containing raw bits was read.
	end_offs uint32
	// Bits that will be read from at the end.
	end_window ecWindow
	// Number of valid bits in end_window.
	nend_bits int
	// The total number of whole bits read. This does not include partial bits
	// currently in the range coder.
	nbits_total int
	// The offset at which the next range coder byte will be read.
	offs uint32
	// The number of values in the current range.
	rng uint32
	// The difference between the top of the current range and the input value,
	// minus one.
	val uint32
	// The saved normalization factor from ec_decode().
	ext uint32
	// A buffered input symbol, awaiting carry propagation.
	rem int
	// Nonzero if an error occurred.
	error int
}

func (_this *Decoder) read_byte() int {
	if _this.offs < _this.storage {
		b := _this.buf[_this.offs]
		_this.offs++
		return int(b)
	}
	return 0
}

func (_this *Decoder) read_byte_from_end() int {
	if _this.end_offs < _this.storage {
		_this.end_offs++
		return int(_this.buf[_this.storage-_this.end_offs])
	}
	return 0
}

// dec_normalize normalizes the contents of val and rng so that rng lies entirely
// in the high-order symbol.
func (_this *Decoder) dec_normalize() {
	/*If the range is too small, rescale it and input some bits.*/
	for _this.rng <= ecCodeBot {
		var sym int
		_this.nbits_total += ecSymBits
		_this.rng <<= ecSymBits
		/*Use up the remaining bits from our last symbol.*/
		sym = _this.rem
		/*Read the next value from the input.*/
		_this.rem = _this.read_byte()
		/*Take the rest of the bits we need from this new symbol.*/
		sym = (sym<<ecSymBits | _this.rem) >> (ecSymBits - ecCodeExtra)
		/*And subtract them from val, capped to be less than EC_CODE_TOP.*/
		_this.val = ((_this.val << ecSymBits) + (ecSymMax &^ uint32(sym))) & (ecCodeTop - 1)
	}
}

// Init initializes the decoder over _buf (ec_dec_init); the buffer size is
// len(_buf).
func (_this *Decoder) Init(_buf []byte) {
	_this.buf = _buf
	_this.storage = uint32(len(_buf))
	_this.end_offs = 0
	_this.end_window = 0
	_this.nend_bits = 0
	/*This is the offset from which ec_tell() will subtract partial bits.
	  The final value after the ec_dec_normalize() call will be the same as in
	   the encoder, but we have to compensate for the bits that are added there.*/
	_this.nbits_total = ecCodeBits + 1 -
		((ecCodeBits-ecCodeExtra)/ecSymBits)*ecSymBits
	_this.offs = 0
	_this.rng = 1 << ecCodeExtra
	_this.rem = _this.read_byte()
	_this.val = _this.rng - 1 - uint32(_this.rem>>(ecSymBits-ecCodeExtra))
	_this.error = 0
	/*Normalize the interval.*/
	_this.dec_normalize()
}

// Decode calculates the cumulative frequency for the next symbol (ec_decode).
// This must be followed by exactly one DecUpdate before Decode is called again.
func (_this *Decoder) Decode(_ft uint32) uint32 {
	var s uint32
	_this.ext = celtUdiv(_this.rng, _ft)
	s = _this.val / _this.ext
	return _ft - min(s+1, _ft)
}

// DecodeBin is equivalent to Decode with _ft==1<<_bits (ec_decode_bin).
func (_this *Decoder) DecodeBin(_bits uint32) uint32 {
	var s uint32
	_this.ext = _this.rng >> _bits
	s = _this.val / _this.ext
	return (uint32(1) << _bits) - min(s+1, uint32(1)<<_bits)
}

// DecUpdate advances the decoder past the next symbol using the frequency
// information the symbol was encoded with (ec_dec_update). Exactly one Decode
// must precede it.
func (_this *Decoder) DecUpdate(_fl, _fh, _ft uint32) {
	s := _this.ext * (_ft - _fh)
	_this.val -= s
	if _fl > 0 {
		_this.rng = _this.ext * (_fh - _fl)
	} else {
		_this.rng = _this.rng - s
	}
	_this.dec_normalize()
	_this.trace(traceDecode, _fl, _fh, _ft)
}

// DecBitLogp decodes a bit that has a 1/(1<<_logp) probability of being a one
// (ec_dec_bit_logp).
func (_this *Decoder) DecBitLogp(_logp uint32) int {
	var r uint32
	var d uint32
	var s uint32
	var ret int
	r = _this.rng
	d = _this.val
	s = r >> _logp
	if d < s {
		ret = 1
	} else {
		ret = 0
	}
	if ret == 0 {
		_this.val = d - s
	}
	if ret != 0 {
		_this.rng = s
	} else {
		_this.rng = r - s
	}
	_this.dec_normalize()
	_this.trace(traceDecBitLogp, uint32(ret), _logp, 0)
	return ret
}

// DecIcdf decodes a symbol given an "inverse" CDF table (ec_dec_icdf). No call to
// DecUpdate is necessary after this call.
func (_this *Decoder) DecIcdf(_icdf []byte, _ftb uint32) int {
	var r uint32
	var d uint32
	var s uint32
	var t uint32
	var ret int
	s = _this.rng
	d = _this.val
	r = s >> _ftb
	ret = -1
	for {
		t = s
		ret++
		s = r * uint32(_icdf[ret])
		if d >= s {
			break
		}
	}
	_this.val = d - s
	_this.rng = t - s
	_this.dec_normalize()
	_this.trace(traceDecIcdf, uint32(ret), _ftb, 0)
	return ret
}

// DecUint extracts a raw unsigned integer with a non-power-of-2 range from the
// stream (ec_dec_uint). No call to DecUpdate is necessary after this call.
// _ft: The number of integers that can be decoded (one more than the max).
func (_this *Decoder) DecUint(_ft uint32) uint32 {
	var ft uint32
	var s uint32
	var ftb int
	/*In order to optimize EC_ILOG(), it is undefined for the value 0.*/
	/* celt_assert(_ft>1) */
	_ft--
	ftb = ecIlog(_ft)
	if ftb > ecUintBits {
		var t uint32
		ftb -= ecUintBits
		ft = (_ft >> ftb) + 1
		s = _this.Decode(ft)
		_this.DecUpdate(s, s+1, ft)
		t = s<<ftb | _this.DecBits(uint32(ftb))
		if t <= _ft {
			return t
		}
		_this.error = 1
		return _ft
	}
	_ft++
	s = _this.Decode(_ft)
	_this.DecUpdate(s, s+1, _ft)
	return s
}

// DecBits extracts a sequence of raw bits from the stream (ec_dec_bits).
// _bits: The number of bits to extract, between 0 and 25.
func (_this *Decoder) DecBits(_bits uint32) uint32 {
	var window ecWindow
	var available int
	var ret uint32
	window = _this.end_window
	available = _this.nend_bits
	if available < int(_bits) {
		for {
			window |= uint32(_this.read_byte_from_end()) << available
			available += ecSymBits
			if available > ecWindowSize-ecSymBits {
				break
			}
		}
	}
	ret = window & ((uint32(1) << _bits) - 1)
	window >>= _bits
	available -= int(_bits)
	_this.end_window = window
	_this.nend_bits = available
	_this.nbits_total += int(_bits)
	_this.trace(traceDecBits, ret, _bits, 0)
	return ret
}

// Tell returns the number of whole bits used so far (ec_tell).
func (_this *Decoder) Tell() int { return ecTell(_this.nbits_total, _this.rng) }

// SetTellForSilence fast-forwards the whole-bit counter so that Tell() reports
// targetBits, mirroring celt_decoder.c:1324 (dec->nbits_total += tell -
// ec_tell(dec)) when the CELT silence flag is set: the decoder pretends it has
// consumed all remaining bits so every subsequent budget check fails and no more
// symbols are read. ADDED for the phase-2 CELT decoder port; it is the only
// write accessor the port needs on the range decoder.
func (_this *Decoder) SetTellForSilence(targetBits int) {
	_this.nbits_total += targetBits - _this.Tell()
}

// TellFrac returns the number of bits used so far scaled by 2**BITRES
// (ec_tell_frac).
func (_this *Decoder) TellFrac() uint32 { return ecTellFrac(_this.nbits_total, _this.rng) }

// Error returns nonzero if an error has occurred (ec_get_error).
func (_this *Decoder) Error() int { return _this.error }

// RangeBytes returns the number of range coder bytes consumed so far
// (ec_range_bytes).
func (_this *Decoder) RangeBytes() uint32 { return _this.offs }

// Storage returns the size of the input buffer in bytes (the ec_ctx storage
// field). CELT reads dec->storage*8 as its total bit budget (e.g.
// unquant_coarse_energy / unquant_fine_energy in celt/quant_bands.c).
func (_this *Decoder) Storage() uint32 { return _this.storage }

// ShrinkStorage reduces the buffer size seen by raw-bit reads from the end,
// implementing the "dec.storage -= redundancy_bytes" step of opus_decode_frame's
// redundancy handling (src/opus_decoder.c:526). The redundant CELT frame occupies
// the last redundancy_bytes of the packet, and the main frame's CELT decode must
// not read raw bits (read_byte_from_end) into that tail.
func (_this *Decoder) ShrinkStorage(n uint32) { _this.storage -= n }

// Rng returns the current range register (the rng field), for tests and
// differential comparison.
func (_this *Decoder) Rng() uint32 { return _this.rng }

// Val returns the current val register, for tests and differential comparison.
func (_this *Decoder) Val() uint32 { return _this.val }
