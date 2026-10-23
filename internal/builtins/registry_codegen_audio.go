package builtins

// Generated-Go counterparts of the audio byte boundaries. The canonical header
// validator is shared within these helpers; cross-engine fixtures pin it against
// the evaluator implementation rather than trusting a round trip alone.
const audioCodegenHeader = `header := func(n, rate, ch, bits int64) ([]byte, error) {
 if n < 0 { return nil, fmt.Errorf("dataLen must be nonnegative (got %d)", n) }
 if rate <= 0 || rate > 4294967295 { return nil, fmt.Errorf("rate must be in 1..4294967295 (got %d)", rate) }
 if ch <= 0 || ch > 65535 { return nil, fmt.Errorf("channels must be in 1..65535 (got %d)", ch) }
 if bits != 8 && bits != 16 && bits != 24 && bits != 32 { return nil, fmt.Errorf("bitsPerSample must be 8, 16, 24 or 32 integer PCM (got %d)", bits) }
 if ch > 65535/(bits/8) { return nil, fmt.Errorf("blockAlign exceeds the 16-bit field") }
 align := ch*(bits/8)
 if rate > 4294967295/align { return nil, fmt.Errorf("byteRate exceeds the 32-bit field") }
 if n%align != 0 { return nil, fmt.Errorf("dataLen %d is not a whole %d-byte frame", n, align) }
 if n > 4294967295-36-n%2 { return nil, fmt.Errorf("RIFF size exceeds the 32-bit field") }
 h:=make([]byte,44); copy(h,"RIFF"); copy(h[8:],"WAVEfmt "); h[16]=16; h[20]=1; copy(h[36:],"data")
 put:=func(i int,v int64,w int){for j:=0;j<w;j++ {h[i+j]=byte(v>>uint(8*j))}}
 put(4,36+n+n%2,4); put(22,ch,2); put(24,rate,4); put(28,rate*align,4); put(32,align,2); put(34,bits,2); put(40,n,4)
 return h,nil
}
`

func registerAudioCodegenSpecs() {
	for _, checked := range []bool{false, true} {
		suffix, exportSuffix := "", ""
		if checked {
			suffix, exportSuffix = "_checked", "Checked"
		}
		for _, fromPCM := range []bool{false, true} {
			name, exportName, helperName := "_audio_wav_header", "wavHeader", "AudioWavHeader"
			op := "audio.wavHeader"
			start := `n:=toInt64(arg0)`
			if fromPCM {
				name, exportName, helperName, op = "_audio_wav_from_pcm", "wavFromPcm", "AudioWavFromPcm", "audio.wavFromPcm"
				start = `pcm:=arg0.([]byte); n:=int64(len(pcm))`
			}
			body := audioCodegenHeader + start + `; rate,ch,bits:=toInt64(arg1),toInt64(arg2),toInt64(arg3)
 var out []byte; var err error
 `
			if fromPCM {
				body += `if n>100*1024*1024 { err=fmt.Errorf("pcm length %d exceeds %d-byte limit",n,100*1024*1024) } else { out,err=header(n,rate,ch,bits) }
 if err==nil { pad:=n%2; if n>int64(int(^uint(0)>>1))-44-pad {err=fmt.Errorf("WAV output length is not representable on this host")} else { h:=out;out=make([]byte,44+int(n+pad));copy(out,h);copy(out[44:],pcm) } }
 `
			} else {
				body += `out,err=header(n,rate,ch,bits)
`
			}
			if checked {
				body += `if err!=nil{return NewResultErr("` + op + `: "+err.Error())};return NewResultOk(out)`
			} else {
				body += `if err!=nil{panic("` + op + `: "+err.Error())};return out`
			}
			registerIfMissing(name+suffix, 4, true, &GoCodegenSpec{Helper: &GoHelperSpec{FuncName: helperName + exportSuffix, Signature: "func " + helperName + exportSuffix + "(arg0,arg1,arg2,arg3 interface{}) interface{}", Body: body}, StdlibModule: "std/audio", StdlibName: exportName + exportSuffix})
		}
	}
	registerIfMissing("_audio_pcm16_to_floats", 4, true, &GoCodegenSpec{Helper: &GoHelperSpec{FuncName: "AudioPCM16ToFloats", Signature: "func AudioPCM16ToFloats(input, off, count, channels interface{}) interface{}", Body: audioCodegenDecode}, StdlibModule: "std/audio", StdlibName: "pcm16ToFloats"})
	registerIfMissing("_audio_floats_to_pcm16", 3, true, &GoCodegenSpec{Helper: &GoHelperSpec{FuncName: "AudioFloatsToPCM16", Signature: "func AudioFloatsToPCM16(input, channels, policy interface{}) interface{}", Body: audioCodegenEncode}, StdlibModule: "std/audio", StdlibName: "floatsToPcm16"})
	registerIfMissing("_audio_wav_info_codegen", 1, true, &GoCodegenSpec{Helper: &GoHelperSpec{FuncName: "AudioWavInfo", Signature: "func AudioWavInfo(input interface{}) interface{}", Body: audioCodegenHeader + audioCodegenInfo}, StdlibModule: "std/audio", StdlibName: "wavInfo"})
}

const audioCodegenDecode = `pcm:=input.([]byte); offset,frames,ch:=toInt64(off),toInt64(count),toInt64(channels)
 const op="audio.pcm16ToFloats: "
 if offset<0 || offset%2!=0 || offset>int64(len(pcm)){return NewResultErr(op+"offset must be even and inside the byte buffer")}
 if frames<0{return NewResultErr(op+"frames must be nonnegative")}
 if ch<=0 || ch>65535{return NewResultErr(op+"channels must be in 1..65535")}
 fb:=ch*2
 if frames>(int64(len(pcm))-offset)/fb || frames>(100*1024*1024)/fb{return NewResultErr(op+"requested PCM extent exceeds the buffer or 100 MiB limit")}
 samples:=frames*ch
 if samples>int64(int(^uint(0)>>1))/8{return NewResultErr(op+"decoded array length is not representable on this host")}
 out:=make([]float64,int(samples))
 for i:=range out {p:=int(offset)+2*i;v:=uint16(pcm[p])|uint16(pcm[p+1])<<8;out[i]=float64(int16(v))/32768}
 return NewResultOk(FloatArrayVal(out))`

const audioCodegenEncode = `ch:=toInt64(channels); const op="audio.floatsToPcm16: "
 if ch<=0 || ch>65535{return NewResultErr(op+"channels must be in 1..65535")}
 v:=reflect.ValueOf(input); if v.Kind()!=reflect.Slice {panic(op+"expected Array[float]")}
 if int64(v.Len())%ch!=0{return NewResultErr(op+"sample array must contain whole interleaved frames")}
 if v.Len()>100*1024*1024/2{return NewResultErr(op+"encoded PCM exceeds the 100 MiB limit")}
 clip:=false
 switch p:=policy.(type){case bool:clip=p;case *Pcm16Quantization:clip=p.Kind==Pcm16QuantizationKindClipOutOfRange;default:panic(op+"invalid quantization policy")}
 xs:=ConvertToFloatArray(input)
 for i,x:=range xs {if math.IsNaN(x)||math.IsInf(x,0){return NewResultErr(fmt.Sprintf("%ssample %d is not finite",op,i))};if !clip&&(x< -1||x>1){return NewResultErr(fmt.Sprintf("%ssample %d is outside [-1,1]",op,i))}}
 out:=make([]byte,len(xs)*2)
 for i,x:=range xs {if x< -1{x= -1};if x>1{x=1};q:=math.RoundToEven(x*32768);if q< -32768{q= -32768};if q>32767{q=32767};n:=uint16(int16(q));out[2*i]=byte(n);out[2*i+1]=byte(n>>8)}
 return NewResultOk(out)`

const audioCodegenInfo = `b:=input.([]byte)
 fail:=func(s string) interface{} {return NewResultErr("audio.wavInfo: "+s)}
 if len(b)<12 || string(b[:4])!="RIFF" || string(b[8:12])!="WAVE" {return fail("expected RIFF/WAVE prefix")}
 le:=func(p,w int)int64{var n int64;for i:=0;i<w;i++{n|=int64(b[p+i])<<uint(8*i)};return n}
 end:=le(4,4)+8
 if end<12||end>int64(len(b)){return fail("RIFF extent exceeds the byte buffer")}
 var fmtAt,fmtLen,dataAt,dataLen int64;haveFmt,haveData:=false,false
 chunks:=0
 for p:=int64(12);p<end; {
  if chunks>=65536{return fail("chunk count exceeds 65536 limit")};chunks++
  if end-p<8{return fail("truncated chunk header")}
  n:=le(int(p+4),4);a:=p+8
  if n>end-a || n%2>end-a-n{return fail("chunk payload or padding exceeds RIFF extent")}
  switch string(b[p:p+4]){case "fmt ":if haveFmt{return fail("duplicate fmt chunk")};haveFmt=true;fmtAt,fmtLen=a,n
   case "data":if haveData{return fail("duplicate data chunk")};haveData=true;dataAt,dataLen=a,n}
  p=a+n+n%2
 }
 if !haveFmt||!haveData{return fail("missing fmt or data chunk")}
 if fmtLen<16{return fail("fmt chunk must contain at least 16 bytes")}
 if le(int(fmtAt),2)!=1{return fail("unsupported WAV encoding; expected integer PCM")}
 if fmtLen>16 {if fmtLen<18{return fail("short fmt extension")};if le(int(fmtAt+16),2)>fmtLen-18{return fail("fmt extension exceeds chunk")}}
 ch,rate,br,align,bits:=le(int(fmtAt+2),2),le(int(fmtAt+4),4),le(int(fmtAt+8),4),le(int(fmtAt+12),2),le(int(fmtAt+14),2)
 if _,err:=header(dataLen,rate,ch,bits);err!=nil{return fail(err.Error())}
 if align!=ch*(bits/8)||br!=rate*align{return fail("inconsistent blockAlign or byteRate")}
 return NewResultOk(&WavInfo{Encoding:NewWavEncodingIntegerPcm(),Layout:NewPcmLayoutInterleaved(),Rate:rate,Channels:ch,ContainerBits:bits,ValidBits:bits,BlockAlign:align,ByteRate:br,DataOffset:dataAt,DataBytes:dataLen,Frames:dataLen/align})`
