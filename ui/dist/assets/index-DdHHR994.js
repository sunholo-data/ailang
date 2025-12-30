var sh=Object.defineProperty;var uh=(e,t,n)=>t in e?sh(e,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):e[t]=n;var Oe=(e,t,n)=>uh(e,typeof t!="symbol"?t+"":t,n);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var el=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function $a(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var ld={exports:{}},El={},od={exports:{}},Z={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var di=Symbol.for("react.element"),ch=Symbol.for("react.portal"),dh=Symbol.for("react.fragment"),ph=Symbol.for("react.strict_mode"),fh=Symbol.for("react.profiler"),hh=Symbol.for("react.provider"),mh=Symbol.for("react.context"),gh=Symbol.for("react.forward_ref"),vh=Symbol.for("react.suspense"),xh=Symbol.for("react.memo"),yh=Symbol.for("react.lazy"),nu=Symbol.iterator;function kh(e){return e===null||typeof e!="object"?null:(e=nu&&e[nu]||e["@@iterator"],typeof e=="function"?e:null)}var ad={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},sd=Object.assign,ud={};function hr(e,t,n){this.props=e,this.context=t,this.refs=ud,this.updater=n||ad}hr.prototype.isReactComponent={};hr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};hr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function cd(){}cd.prototype=hr.prototype;function Ha(e,t,n){this.props=e,this.context=t,this.refs=ud,this.updater=n||ad}var Ua=Ha.prototype=new cd;Ua.constructor=Ha;sd(Ua,hr.prototype);Ua.isPureReactComponent=!0;var ru=Array.isArray,dd=Object.prototype.hasOwnProperty,Va={current:null},pd={key:!0,ref:!0,__self:!0,__source:!0};function fd(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)dd.call(t,r)&&!pd.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var u=Array(a),c=0;c<a;c++)u[c]=arguments[c+2];i.children=u}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:di,type:e,key:l,ref:o,props:i,_owner:Va.current}}function wh(e,t){return{$$typeof:di,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function Wa(e){return typeof e=="object"&&e!==null&&e.$$typeof===di}function Sh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var iu=/\/+/g;function Kl(e,t){return typeof e=="object"&&e!==null&&e.key!=null?Sh(""+e.key):t.toString(36)}function Bi(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case di:case ch:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Kl(o,0):r,ru(i)?(n="",e!=null&&(n=e.replace(iu,"$&/")+"/"),Bi(i,t,n,"",function(c){return c})):i!=null&&(Wa(i)&&(i=wh(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(iu,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",ru(e))for(var a=0;a<e.length;a++){l=e[a];var u=r+Kl(l,a);o+=Bi(l,t,n,u,i)}else if(u=kh(e),typeof u=="function")for(e=u.call(e),a=0;!(l=e.next()).done;)l=l.value,u=r+Kl(l,a++),o+=Bi(l,t,n,u,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function xi(e,t,n){if(e==null)return e;var r=[],i=0;return Bi(e,r,"","",function(l){return t.call(n,l,i++)}),r}function bh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Ue={current:null},$i={transition:null},_h={ReactCurrentDispatcher:Ue,ReactCurrentBatchConfig:$i,ReactCurrentOwner:Va};function hd(){throw Error("act(...) is not supported in production builds of React.")}Z.Children={map:xi,forEach:function(e,t,n){xi(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return xi(e,function(){t++}),t},toArray:function(e){return xi(e,function(t){return t})||[]},only:function(e){if(!Wa(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};Z.Component=hr;Z.Fragment=dh;Z.Profiler=fh;Z.PureComponent=Ha;Z.StrictMode=ph;Z.Suspense=vh;Z.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=_h;Z.act=hd;Z.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=sd({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Va.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(u in t)dd.call(t,u)&&!pd.hasOwnProperty(u)&&(r[u]=t[u]===void 0&&a!==void 0?a[u]:t[u])}var u=arguments.length-2;if(u===1)r.children=n;else if(1<u){a=Array(u);for(var c=0;c<u;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:di,type:e.type,key:i,ref:l,props:r,_owner:o}};Z.createContext=function(e){return e={$$typeof:mh,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:hh,_context:e},e.Consumer=e};Z.createElement=fd;Z.createFactory=function(e){var t=fd.bind(null,e);return t.type=e,t};Z.createRef=function(){return{current:null}};Z.forwardRef=function(e){return{$$typeof:gh,render:e}};Z.isValidElement=Wa;Z.lazy=function(e){return{$$typeof:yh,_payload:{_status:-1,_result:e},_init:bh}};Z.memo=function(e,t){return{$$typeof:xh,type:e,compare:t===void 0?null:t}};Z.startTransition=function(e){var t=$i.transition;$i.transition={};try{e()}finally{$i.transition=t}};Z.unstable_act=hd;Z.useCallback=function(e,t){return Ue.current.useCallback(e,t)};Z.useContext=function(e){return Ue.current.useContext(e)};Z.useDebugValue=function(){};Z.useDeferredValue=function(e){return Ue.current.useDeferredValue(e)};Z.useEffect=function(e,t){return Ue.current.useEffect(e,t)};Z.useId=function(){return Ue.current.useId()};Z.useImperativeHandle=function(e,t,n){return Ue.current.useImperativeHandle(e,t,n)};Z.useInsertionEffect=function(e,t){return Ue.current.useInsertionEffect(e,t)};Z.useLayoutEffect=function(e,t){return Ue.current.useLayoutEffect(e,t)};Z.useMemo=function(e,t){return Ue.current.useMemo(e,t)};Z.useReducer=function(e,t,n){return Ue.current.useReducer(e,t,n)};Z.useRef=function(e){return Ue.current.useRef(e)};Z.useState=function(e){return Ue.current.useState(e)};Z.useSyncExternalStore=function(e,t,n){return Ue.current.useSyncExternalStore(e,t,n)};Z.useTransition=function(){return Ue.current.useTransition()};Z.version="18.3.1";od.exports=Z;var z=od.exports;const Jt=$a(z);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var jh=z,Ch=Symbol.for("react.element"),Nh=Symbol.for("react.fragment"),Eh=Object.prototype.hasOwnProperty,Th=jh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,Lh={key:!0,ref:!0,__self:!0,__source:!0};function md(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)Eh.call(t,r)&&!Lh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:Ch,type:e,key:l,ref:o,props:i,_owner:Th.current}}El.Fragment=Nh;El.jsx=md;El.jsxs=md;ld.exports=El;var s=ld.exports,Mo={},gd={exports:{}},st={},vd={exports:{}},xd={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(_,B){var g=_.length;_.push(B);e:for(;0<g;){var L=g-1>>>1,R=_[L];if(0<i(R,B))_[L]=B,_[g]=R,g=L;else break e}}function n(_){return _.length===0?null:_[0]}function r(_){if(_.length===0)return null;var B=_[0],g=_.pop();if(g!==B){_[0]=g;e:for(var L=0,R=_.length,y=R>>>1;L<y;){var J=2*(L+1)-1,he=_[J],ee=J+1,ye=_[ee];if(0>i(he,g))ee<R&&0>i(ye,he)?(_[L]=ye,_[ee]=g,L=ee):(_[L]=he,_[J]=g,L=J);else if(ee<R&&0>i(ye,g))_[L]=ye,_[ee]=g,L=ee;else break e}}return B}function i(_,B){var g=_.sortIndex-B.sortIndex;return g!==0?g:_.id-B.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var u=[],c=[],d=1,p=null,f=3,h=!1,k=!1,w=!1,I=typeof setTimeout=="function"?setTimeout:null,m=typeof clearTimeout=="function"?clearTimeout:null,v=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function x(_){for(var B=n(c);B!==null;){if(B.callback===null)r(c);else if(B.startTime<=_)r(c),B.sortIndex=B.expirationTime,t(u,B);else break;B=n(c)}}function b(_){if(w=!1,x(_),!k)if(n(u)!==null)k=!0,G(N);else{var B=n(c);B!==null&&oe(b,B.startTime-_)}}function N(_,B){k=!1,w&&(w=!1,m(P),P=-1),h=!0;var g=f;try{for(x(B),p=n(u);p!==null&&(!(p.expirationTime>B)||_&&!j());){var L=p.callback;if(typeof L=="function"){p.callback=null,f=p.priorityLevel;var R=L(p.expirationTime<=B);B=e.unstable_now(),typeof R=="function"?p.callback=R:p===n(u)&&r(u),x(B)}else r(u);p=n(u)}if(p!==null)var y=!0;else{var J=n(c);J!==null&&oe(b,J.startTime-B),y=!1}return y}finally{p=null,f=g,h=!1}}var S=!1,C=null,P=-1,D=5,A=-1;function j(){return!(e.unstable_now()-A<D)}function E(){if(C!==null){var _=e.unstable_now();A=_;var B=!0;try{B=C(!0,_)}finally{B?U():(S=!1,C=null)}}else S=!1}var U;if(typeof v=="function")U=function(){v(E)};else if(typeof MessageChannel<"u"){var V=new MessageChannel,W=V.port2;V.port1.onmessage=E,U=function(){W.postMessage(null)}}else U=function(){I(E,0)};function G(_){C=_,S||(S=!0,U())}function oe(_,B){P=I(function(){_(e.unstable_now())},B)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(_){_.callback=null},e.unstable_continueExecution=function(){k||h||(k=!0,G(N))},e.unstable_forceFrameRate=function(_){0>_||125<_?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):D=0<_?Math.floor(1e3/_):5},e.unstable_getCurrentPriorityLevel=function(){return f},e.unstable_getFirstCallbackNode=function(){return n(u)},e.unstable_next=function(_){switch(f){case 1:case 2:case 3:var B=3;break;default:B=f}var g=f;f=B;try{return _()}finally{f=g}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(_,B){switch(_){case 1:case 2:case 3:case 4:case 5:break;default:_=3}var g=f;f=_;try{return B()}finally{f=g}},e.unstable_scheduleCallback=function(_,B,g){var L=e.unstable_now();switch(typeof g=="object"&&g!==null?(g=g.delay,g=typeof g=="number"&&0<g?L+g:L):g=L,_){case 1:var R=-1;break;case 2:R=250;break;case 5:R=1073741823;break;case 4:R=1e4;break;default:R=5e3}return R=g+R,_={id:d++,callback:B,priorityLevel:_,startTime:g,expirationTime:R,sortIndex:-1},g>L?(_.sortIndex=g,t(c,_),n(u)===null&&_===n(c)&&(w?(m(P),P=-1):w=!0,oe(b,g-L))):(_.sortIndex=R,t(u,_),k||h||(k=!0,G(N))),_},e.unstable_shouldYield=j,e.unstable_wrapCallback=function(_){var B=f;return function(){var g=f;f=B;try{return _.apply(this,arguments)}finally{f=g}}}})(xd);vd.exports=xd;var Ph=vd.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var Ih=z,at=Ph;function M(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var yd=new Set,qr={};function An(e,t){ar(e,t),ar(e+"Capture",t)}function ar(e,t){for(qr[e]=t,e=0;e<t.length;e++)yd.add(t[e])}var Ut=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),Do=Object.prototype.hasOwnProperty,zh=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,lu={},ou={};function Ah(e){return Do.call(ou,e)?!0:Do.call(lu,e)?!1:zh.test(e)?ou[e]=!0:(lu[e]=!0,!1)}function Rh(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function Mh(e,t,n,r){if(t===null||typeof t>"u"||Rh(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Ve(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ie={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ie[e]=new Ve(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ie[t]=new Ve(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ie[e]=new Ve(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ie[e]=new Ve(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ie[e]=new Ve(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ie[e]=new Ve(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ie[e]=new Ve(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ie[e]=new Ve(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ie[e]=new Ve(e,5,!1,e.toLowerCase(),null,!1,!1)});var Qa=/[\-:]([a-z])/g;function qa(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(Qa,qa);Ie[t]=new Ve(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(Qa,qa);Ie[t]=new Ve(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(Qa,qa);Ie[t]=new Ve(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ie[e]=new Ve(e,1,!1,e.toLowerCase(),null,!1,!1)});Ie.xlinkHref=new Ve("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ie[e]=new Ve(e,1,!1,e.toLowerCase(),null,!0,!0)});function Ka(e,t,n,r){var i=Ie.hasOwnProperty(t)?Ie[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(Mh(t,n,i,r)&&(n=null),r||i===null?Ah(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var qt=Ih.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,yi=Symbol.for("react.element"),$n=Symbol.for("react.portal"),Hn=Symbol.for("react.fragment"),Ya=Symbol.for("react.strict_mode"),Fo=Symbol.for("react.profiler"),kd=Symbol.for("react.provider"),wd=Symbol.for("react.context"),Ga=Symbol.for("react.forward_ref"),Oo=Symbol.for("react.suspense"),Bo=Symbol.for("react.suspense_list"),Xa=Symbol.for("react.memo"),Zt=Symbol.for("react.lazy"),Sd=Symbol.for("react.offscreen"),au=Symbol.iterator;function wr(e){return e===null||typeof e!="object"?null:(e=au&&e[au]||e["@@iterator"],typeof e=="function"?e:null)}var ve=Object.assign,Yl;function Pr(e){if(Yl===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Yl=t&&t[1]||""}return`
`+Yl+e}var Gl=!1;function Xl(e,t){if(!e||Gl)return"";Gl=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var u=`
`+i[o].replace(" at new "," at ");return e.displayName&&u.includes("<anonymous>")&&(u=u.replace("<anonymous>",e.displayName)),u}while(1<=o&&0<=a);break}}}finally{Gl=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?Pr(e):""}function Dh(e){switch(e.tag){case 5:return Pr(e.type);case 16:return Pr("Lazy");case 13:return Pr("Suspense");case 19:return Pr("SuspenseList");case 0:case 2:case 15:return e=Xl(e.type,!1),e;case 11:return e=Xl(e.type.render,!1),e;case 1:return e=Xl(e.type,!0),e;default:return""}}function $o(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Hn:return"Fragment";case $n:return"Portal";case Fo:return"Profiler";case Ya:return"StrictMode";case Oo:return"Suspense";case Bo:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case wd:return(e.displayName||"Context")+".Consumer";case kd:return(e._context.displayName||"Context")+".Provider";case Ga:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Xa:return t=e.displayName||null,t!==null?t:$o(e.type)||"Memo";case Zt:t=e._payload,e=e._init;try{return $o(e(t))}catch{}}return null}function Fh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return $o(t);case 8:return t===Ya?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function hn(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function bd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function Oh(e){var t=bd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function ki(e){e._valueTracker||(e._valueTracker=Oh(e))}function _d(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=bd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function tl(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function Ho(e,t){var n=t.checked;return ve({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function su(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=hn(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function jd(e,t){t=t.checked,t!=null&&Ka(e,"checked",t,!1)}function Uo(e,t){jd(e,t);var n=hn(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?Vo(e,t.type,n):t.hasOwnProperty("defaultValue")&&Vo(e,t.type,hn(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function uu(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function Vo(e,t,n){(t!=="number"||tl(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var Ir=Array.isArray;function Zn(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+hn(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function Wo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(M(91));return ve({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function cu(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(M(92));if(Ir(n)){if(1<n.length)throw Error(M(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:hn(n)}}function Cd(e,t){var n=hn(t.value),r=hn(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function du(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function Nd(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Qo(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?Nd(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var wi,Ed=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(wi=wi||document.createElement("div"),wi.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=wi.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Kr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Rr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},Bh=["Webkit","ms","Moz","O"];Object.keys(Rr).forEach(function(e){Bh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Rr[t]=Rr[e]})});function Td(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Rr.hasOwnProperty(e)&&Rr[e]?(""+t).trim():t+"px"}function Ld(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=Td(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var $h=ve({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function qo(e,t){if(t){if($h[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(M(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(M(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(M(61))}if(t.style!=null&&typeof t.style!="object")throw Error(M(62))}}function Ko(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Yo=null;function Ja(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Go=null,er=null,tr=null;function pu(e){if(e=hi(e)){if(typeof Go!="function")throw Error(M(280));var t=e.stateNode;t&&(t=zl(t),Go(e.stateNode,e.type,t))}}function Pd(e){er?tr?tr.push(e):tr=[e]:er=e}function Id(){if(er){var e=er,t=tr;if(tr=er=null,pu(e),t)for(e=0;e<t.length;e++)pu(t[e])}}function zd(e,t){return e(t)}function Ad(){}var Jl=!1;function Rd(e,t,n){if(Jl)return e(t,n);Jl=!0;try{return zd(e,t,n)}finally{Jl=!1,(er!==null||tr!==null)&&(Ad(),Id())}}function Yr(e,t){var n=e.stateNode;if(n===null)return null;var r=zl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(M(231,t,typeof n));return n}var Xo=!1;if(Ut)try{var Sr={};Object.defineProperty(Sr,"passive",{get:function(){Xo=!0}}),window.addEventListener("test",Sr,Sr),window.removeEventListener("test",Sr,Sr)}catch{Xo=!1}function Hh(e,t,n,r,i,l,o,a,u){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Mr=!1,nl=null,rl=!1,Jo=null,Uh={onError:function(e){Mr=!0,nl=e}};function Vh(e,t,n,r,i,l,o,a,u){Mr=!1,nl=null,Hh.apply(Uh,arguments)}function Wh(e,t,n,r,i,l,o,a,u){if(Vh.apply(this,arguments),Mr){if(Mr){var c=nl;Mr=!1,nl=null}else throw Error(M(198));rl||(rl=!0,Jo=c)}}function Rn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function Md(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function fu(e){if(Rn(e)!==e)throw Error(M(188))}function Qh(e){var t=e.alternate;if(!t){if(t=Rn(e),t===null)throw Error(M(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return fu(i),e;if(l===r)return fu(i),t;l=l.sibling}throw Error(M(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(M(189))}}if(n.alternate!==r)throw Error(M(190))}if(n.tag!==3)throw Error(M(188));return n.stateNode.current===n?e:t}function Dd(e){return e=Qh(e),e!==null?Fd(e):null}function Fd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=Fd(e);if(t!==null)return t;e=e.sibling}return null}var Od=at.unstable_scheduleCallback,hu=at.unstable_cancelCallback,qh=at.unstable_shouldYield,Kh=at.unstable_requestPaint,ke=at.unstable_now,Yh=at.unstable_getCurrentPriorityLevel,Za=at.unstable_ImmediatePriority,Bd=at.unstable_UserBlockingPriority,il=at.unstable_NormalPriority,Gh=at.unstable_LowPriority,$d=at.unstable_IdlePriority,Tl=null,Pt=null;function Xh(e){if(Pt&&typeof Pt.onCommitFiberRoot=="function")try{Pt.onCommitFiberRoot(Tl,e,void 0,(e.current.flags&128)===128)}catch{}}var bt=Math.clz32?Math.clz32:em,Jh=Math.log,Zh=Math.LN2;function em(e){return e>>>=0,e===0?32:31-(Jh(e)/Zh|0)|0}var Si=64,bi=4194304;function zr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function ll(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=zr(a):(l&=o,l!==0&&(r=zr(l)))}else o=n&~i,o!==0?r=zr(o):l!==0&&(r=zr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-bt(t),i=1<<n,r|=e[n],t&=~i;return r}function tm(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function nm(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-bt(l),a=1<<o,u=i[o];u===-1?(!(a&n)||a&r)&&(i[o]=tm(a,t)):u<=t&&(e.expiredLanes|=a),l&=~a}}function Zo(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function Hd(){var e=Si;return Si<<=1,!(Si&4194240)&&(Si=64),e}function Zl(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function pi(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-bt(t),e[t]=n}function rm(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-bt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function es(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-bt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var le=0;function Ud(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Vd,ts,Wd,Qd,qd,ea=!1,_i=[],on=null,an=null,sn=null,Gr=new Map,Xr=new Map,tn=[],im="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function mu(e,t){switch(e){case"focusin":case"focusout":on=null;break;case"dragenter":case"dragleave":an=null;break;case"mouseover":case"mouseout":sn=null;break;case"pointerover":case"pointerout":Gr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Xr.delete(t.pointerId)}}function br(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=hi(t),t!==null&&ts(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function lm(e,t,n,r,i){switch(t){case"focusin":return on=br(on,e,t,n,r,i),!0;case"dragenter":return an=br(an,e,t,n,r,i),!0;case"mouseover":return sn=br(sn,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Gr.set(l,br(Gr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Xr.set(l,br(Xr.get(l)||null,e,t,n,r,i)),!0}return!1}function Kd(e){var t=_n(e.target);if(t!==null){var n=Rn(t);if(n!==null){if(t=n.tag,t===13){if(t=Md(n),t!==null){e.blockedOn=t,qd(e.priority,function(){Wd(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Hi(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=ta(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Yo=r,n.target.dispatchEvent(r),Yo=null}else return t=hi(n),t!==null&&ts(t),e.blockedOn=n,!1;t.shift()}return!0}function gu(e,t,n){Hi(e)&&n.delete(t)}function om(){ea=!1,on!==null&&Hi(on)&&(on=null),an!==null&&Hi(an)&&(an=null),sn!==null&&Hi(sn)&&(sn=null),Gr.forEach(gu),Xr.forEach(gu)}function _r(e,t){e.blockedOn===t&&(e.blockedOn=null,ea||(ea=!0,at.unstable_scheduleCallback(at.unstable_NormalPriority,om)))}function Jr(e){function t(i){return _r(i,e)}if(0<_i.length){_r(_i[0],e);for(var n=1;n<_i.length;n++){var r=_i[n];r.blockedOn===e&&(r.blockedOn=null)}}for(on!==null&&_r(on,e),an!==null&&_r(an,e),sn!==null&&_r(sn,e),Gr.forEach(t),Xr.forEach(t),n=0;n<tn.length;n++)r=tn[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<tn.length&&(n=tn[0],n.blockedOn===null);)Kd(n),n.blockedOn===null&&tn.shift()}var nr=qt.ReactCurrentBatchConfig,ol=!0;function am(e,t,n,r){var i=le,l=nr.transition;nr.transition=null;try{le=1,ns(e,t,n,r)}finally{le=i,nr.transition=l}}function sm(e,t,n,r){var i=le,l=nr.transition;nr.transition=null;try{le=4,ns(e,t,n,r)}finally{le=i,nr.transition=l}}function ns(e,t,n,r){if(ol){var i=ta(e,t,n,r);if(i===null)uo(e,t,r,al,n),mu(e,r);else if(lm(i,e,t,n,r))r.stopPropagation();else if(mu(e,r),t&4&&-1<im.indexOf(e)){for(;i!==null;){var l=hi(i);if(l!==null&&Vd(l),l=ta(e,t,n,r),l===null&&uo(e,t,r,al,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else uo(e,t,r,null,n)}}var al=null;function ta(e,t,n,r){if(al=null,e=Ja(r),e=_n(e),e!==null)if(t=Rn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=Md(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return al=e,null}function Yd(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(Yh()){case Za:return 1;case Bd:return 4;case il:case Gh:return 16;case $d:return 536870912;default:return 16}default:return 16}}var rn=null,rs=null,Ui=null;function Gd(){if(Ui)return Ui;var e,t=rs,n=t.length,r,i="value"in rn?rn.value:rn.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Ui=i.slice(e,1<r?1-r:void 0)}function Vi(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function ji(){return!0}function vu(){return!1}function ut(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?ji:vu,this.isPropagationStopped=vu,this}return ve(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=ji)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=ji)},persist:function(){},isPersistent:ji}),t}var mr={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},is=ut(mr),fi=ve({},mr,{view:0,detail:0}),um=ut(fi),eo,to,jr,Ll=ve({},fi,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:ls,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==jr&&(jr&&e.type==="mousemove"?(eo=e.screenX-jr.screenX,to=e.screenY-jr.screenY):to=eo=0,jr=e),eo)},movementY:function(e){return"movementY"in e?e.movementY:to}}),xu=ut(Ll),cm=ve({},Ll,{dataTransfer:0}),dm=ut(cm),pm=ve({},fi,{relatedTarget:0}),no=ut(pm),fm=ve({},mr,{animationName:0,elapsedTime:0,pseudoElement:0}),hm=ut(fm),mm=ve({},mr,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),gm=ut(mm),vm=ve({},mr,{data:0}),yu=ut(vm),xm={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},ym={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},km={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function wm(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=km[e])?!!t[e]:!1}function ls(){return wm}var Sm=ve({},fi,{key:function(e){if(e.key){var t=xm[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Vi(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?ym[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:ls,charCode:function(e){return e.type==="keypress"?Vi(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Vi(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),bm=ut(Sm),_m=ve({},Ll,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),ku=ut(_m),jm=ve({},fi,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:ls}),Cm=ut(jm),Nm=ve({},mr,{propertyName:0,elapsedTime:0,pseudoElement:0}),Em=ut(Nm),Tm=ve({},Ll,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),Lm=ut(Tm),Pm=[9,13,27,32],os=Ut&&"CompositionEvent"in window,Dr=null;Ut&&"documentMode"in document&&(Dr=document.documentMode);var Im=Ut&&"TextEvent"in window&&!Dr,Xd=Ut&&(!os||Dr&&8<Dr&&11>=Dr),wu=" ",Su=!1;function Jd(e,t){switch(e){case"keyup":return Pm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Zd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Un=!1;function zm(e,t){switch(e){case"compositionend":return Zd(t);case"keypress":return t.which!==32?null:(Su=!0,wu);case"textInput":return e=t.data,e===wu&&Su?null:e;default:return null}}function Am(e,t){if(Un)return e==="compositionend"||!os&&Jd(e,t)?(e=Gd(),Ui=rs=rn=null,Un=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Xd&&t.locale!=="ko"?null:t.data;default:return null}}var Rm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function bu(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!Rm[e.type]:t==="textarea"}function ep(e,t,n,r){Pd(r),t=sl(t,"onChange"),0<t.length&&(n=new is("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Fr=null,Zr=null;function Mm(e){dp(e,0)}function Pl(e){var t=Qn(e);if(_d(t))return e}function Dm(e,t){if(e==="change")return t}var tp=!1;if(Ut){var ro;if(Ut){var io="oninput"in document;if(!io){var _u=document.createElement("div");_u.setAttribute("oninput","return;"),io=typeof _u.oninput=="function"}ro=io}else ro=!1;tp=ro&&(!document.documentMode||9<document.documentMode)}function ju(){Fr&&(Fr.detachEvent("onpropertychange",np),Zr=Fr=null)}function np(e){if(e.propertyName==="value"&&Pl(Zr)){var t=[];ep(t,Zr,e,Ja(e)),Rd(Mm,t)}}function Fm(e,t,n){e==="focusin"?(ju(),Fr=t,Zr=n,Fr.attachEvent("onpropertychange",np)):e==="focusout"&&ju()}function Om(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return Pl(Zr)}function Bm(e,t){if(e==="click")return Pl(t)}function $m(e,t){if(e==="input"||e==="change")return Pl(t)}function Hm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var jt=typeof Object.is=="function"?Object.is:Hm;function ei(e,t){if(jt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!Do.call(t,i)||!jt(e[i],t[i]))return!1}return!0}function Cu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function Nu(e,t){var n=Cu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=Cu(n)}}function rp(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?rp(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function ip(){for(var e=window,t=tl();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=tl(e.document)}return t}function as(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function Um(e){var t=ip(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&rp(n.ownerDocument.documentElement,n)){if(r!==null&&as(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=Nu(n,l);var o=Nu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Vm=Ut&&"documentMode"in document&&11>=document.documentMode,Vn=null,na=null,Or=null,ra=!1;function Eu(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;ra||Vn==null||Vn!==tl(r)||(r=Vn,"selectionStart"in r&&as(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Or&&ei(Or,r)||(Or=r,r=sl(na,"onSelect"),0<r.length&&(t=new is("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Vn)))}function Ci(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Wn={animationend:Ci("Animation","AnimationEnd"),animationiteration:Ci("Animation","AnimationIteration"),animationstart:Ci("Animation","AnimationStart"),transitionend:Ci("Transition","TransitionEnd")},lo={},lp={};Ut&&(lp=document.createElement("div").style,"AnimationEvent"in window||(delete Wn.animationend.animation,delete Wn.animationiteration.animation,delete Wn.animationstart.animation),"TransitionEvent"in window||delete Wn.transitionend.transition);function Il(e){if(lo[e])return lo[e];if(!Wn[e])return e;var t=Wn[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in lp)return lo[e]=t[n];return e}var op=Il("animationend"),ap=Il("animationiteration"),sp=Il("animationstart"),up=Il("transitionend"),cp=new Map,Tu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function gn(e,t){cp.set(e,t),An(t,[e])}for(var oo=0;oo<Tu.length;oo++){var ao=Tu[oo],Wm=ao.toLowerCase(),Qm=ao[0].toUpperCase()+ao.slice(1);gn(Wm,"on"+Qm)}gn(op,"onAnimationEnd");gn(ap,"onAnimationIteration");gn(sp,"onAnimationStart");gn("dblclick","onDoubleClick");gn("focusin","onFocus");gn("focusout","onBlur");gn(up,"onTransitionEnd");ar("onMouseEnter",["mouseout","mouseover"]);ar("onMouseLeave",["mouseout","mouseover"]);ar("onPointerEnter",["pointerout","pointerover"]);ar("onPointerLeave",["pointerout","pointerover"]);An("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));An("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));An("onBeforeInput",["compositionend","keypress","textInput","paste"]);An("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));An("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));An("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var Ar="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),qm=new Set("cancel close invalid load scroll toggle".split(" ").concat(Ar));function Lu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,Wh(r,t,void 0,e),e.currentTarget=null}function dp(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],u=a.instance,c=a.currentTarget;if(a=a.listener,u!==l&&i.isPropagationStopped())break e;Lu(i,a,c),l=u}else for(o=0;o<r.length;o++){if(a=r[o],u=a.instance,c=a.currentTarget,a=a.listener,u!==l&&i.isPropagationStopped())break e;Lu(i,a,c),l=u}}}if(rl)throw e=Jo,rl=!1,Jo=null,e}function de(e,t){var n=t[sa];n===void 0&&(n=t[sa]=new Set);var r=e+"__bubble";n.has(r)||(pp(t,e,2,!1),n.add(r))}function so(e,t,n){var r=0;t&&(r|=4),pp(n,e,r,t)}var Ni="_reactListening"+Math.random().toString(36).slice(2);function ti(e){if(!e[Ni]){e[Ni]=!0,yd.forEach(function(n){n!=="selectionchange"&&(qm.has(n)||so(n,!1,e),so(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[Ni]||(t[Ni]=!0,so("selectionchange",!1,t))}}function pp(e,t,n,r){switch(Yd(t)){case 1:var i=am;break;case 4:i=sm;break;default:i=ns}n=i.bind(null,t,n,e),i=void 0,!Xo||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function uo(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var u=o.tag;if((u===3||u===4)&&(u=o.stateNode.containerInfo,u===i||u.nodeType===8&&u.parentNode===i))return;o=o.return}for(;a!==null;){if(o=_n(a),o===null)return;if(u=o.tag,u===5||u===6){r=l=o;continue e}a=a.parentNode}}r=r.return}Rd(function(){var c=l,d=Ja(n),p=[];e:{var f=cp.get(e);if(f!==void 0){var h=is,k=e;switch(e){case"keypress":if(Vi(n)===0)break e;case"keydown":case"keyup":h=bm;break;case"focusin":k="focus",h=no;break;case"focusout":k="blur",h=no;break;case"beforeblur":case"afterblur":h=no;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":h=xu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":h=dm;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":h=Cm;break;case op:case ap:case sp:h=hm;break;case up:h=Em;break;case"scroll":h=um;break;case"wheel":h=Lm;break;case"copy":case"cut":case"paste":h=gm;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":h=ku}var w=(t&4)!==0,I=!w&&e==="scroll",m=w?f!==null?f+"Capture":null:f;w=[];for(var v=c,x;v!==null;){x=v;var b=x.stateNode;if(x.tag===5&&b!==null&&(x=b,m!==null&&(b=Yr(v,m),b!=null&&w.push(ni(v,b,x)))),I)break;v=v.return}0<w.length&&(f=new h(f,k,null,n,d),p.push({event:f,listeners:w}))}}if(!(t&7)){e:{if(f=e==="mouseover"||e==="pointerover",h=e==="mouseout"||e==="pointerout",f&&n!==Yo&&(k=n.relatedTarget||n.fromElement)&&(_n(k)||k[Vt]))break e;if((h||f)&&(f=d.window===d?d:(f=d.ownerDocument)?f.defaultView||f.parentWindow:window,h?(k=n.relatedTarget||n.toElement,h=c,k=k?_n(k):null,k!==null&&(I=Rn(k),k!==I||k.tag!==5&&k.tag!==6)&&(k=null)):(h=null,k=c),h!==k)){if(w=xu,b="onMouseLeave",m="onMouseEnter",v="mouse",(e==="pointerout"||e==="pointerover")&&(w=ku,b="onPointerLeave",m="onPointerEnter",v="pointer"),I=h==null?f:Qn(h),x=k==null?f:Qn(k),f=new w(b,v+"leave",h,n,d),f.target=I,f.relatedTarget=x,b=null,_n(d)===c&&(w=new w(m,v+"enter",k,n,d),w.target=x,w.relatedTarget=I,b=w),I=b,h&&k)t:{for(w=h,m=k,v=0,x=w;x;x=Fn(x))v++;for(x=0,b=m;b;b=Fn(b))x++;for(;0<v-x;)w=Fn(w),v--;for(;0<x-v;)m=Fn(m),x--;for(;v--;){if(w===m||m!==null&&w===m.alternate)break t;w=Fn(w),m=Fn(m)}w=null}else w=null;h!==null&&Pu(p,f,h,w,!1),k!==null&&I!==null&&Pu(p,I,k,w,!0)}}e:{if(f=c?Qn(c):window,h=f.nodeName&&f.nodeName.toLowerCase(),h==="select"||h==="input"&&f.type==="file")var N=Dm;else if(bu(f))if(tp)N=$m;else{N=Om;var S=Fm}else(h=f.nodeName)&&h.toLowerCase()==="input"&&(f.type==="checkbox"||f.type==="radio")&&(N=Bm);if(N&&(N=N(e,c))){ep(p,N,n,d);break e}S&&S(e,f,c),e==="focusout"&&(S=f._wrapperState)&&S.controlled&&f.type==="number"&&Vo(f,"number",f.value)}switch(S=c?Qn(c):window,e){case"focusin":(bu(S)||S.contentEditable==="true")&&(Vn=S,na=c,Or=null);break;case"focusout":Or=na=Vn=null;break;case"mousedown":ra=!0;break;case"contextmenu":case"mouseup":case"dragend":ra=!1,Eu(p,n,d);break;case"selectionchange":if(Vm)break;case"keydown":case"keyup":Eu(p,n,d)}var C;if(os)e:{switch(e){case"compositionstart":var P="onCompositionStart";break e;case"compositionend":P="onCompositionEnd";break e;case"compositionupdate":P="onCompositionUpdate";break e}P=void 0}else Un?Jd(e,n)&&(P="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(P="onCompositionStart");P&&(Xd&&n.locale!=="ko"&&(Un||P!=="onCompositionStart"?P==="onCompositionEnd"&&Un&&(C=Gd()):(rn=d,rs="value"in rn?rn.value:rn.textContent,Un=!0)),S=sl(c,P),0<S.length&&(P=new yu(P,e,null,n,d),p.push({event:P,listeners:S}),C?P.data=C:(C=Zd(n),C!==null&&(P.data=C)))),(C=Im?zm(e,n):Am(e,n))&&(c=sl(c,"onBeforeInput"),0<c.length&&(d=new yu("onBeforeInput","beforeinput",null,n,d),p.push({event:d,listeners:c}),d.data=C))}dp(p,t)})}function ni(e,t,n){return{instance:e,listener:t,currentTarget:n}}function sl(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Yr(e,n),l!=null&&r.unshift(ni(e,l,i)),l=Yr(e,t),l!=null&&r.push(ni(e,l,i))),e=e.return}return r}function Fn(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function Pu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,u=a.alternate,c=a.stateNode;if(u!==null&&u===r)break;a.tag===5&&c!==null&&(a=c,i?(u=Yr(n,l),u!=null&&o.unshift(ni(n,u,a))):i||(u=Yr(n,l),u!=null&&o.push(ni(n,u,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Km=/\r\n?/g,Ym=/\u0000|\uFFFD/g;function Iu(e){return(typeof e=="string"?e:""+e).replace(Km,`
`).replace(Ym,"")}function Ei(e,t,n){if(t=Iu(t),Iu(e)!==t&&n)throw Error(M(425))}function ul(){}var ia=null,la=null;function oa(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var aa=typeof setTimeout=="function"?setTimeout:void 0,Gm=typeof clearTimeout=="function"?clearTimeout:void 0,zu=typeof Promise=="function"?Promise:void 0,Xm=typeof queueMicrotask=="function"?queueMicrotask:typeof zu<"u"?function(e){return zu.resolve(null).then(e).catch(Jm)}:aa;function Jm(e){setTimeout(function(){throw e})}function co(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Jr(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Jr(t)}function un(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function Au(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var gr=Math.random().toString(36).slice(2),Tt="__reactFiber$"+gr,ri="__reactProps$"+gr,Vt="__reactContainer$"+gr,sa="__reactEvents$"+gr,Zm="__reactListeners$"+gr,eg="__reactHandles$"+gr;function _n(e){var t=e[Tt];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Vt]||n[Tt]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=Au(e);e!==null;){if(n=e[Tt])return n;e=Au(e)}return t}e=n,n=e.parentNode}return null}function hi(e){return e=e[Tt]||e[Vt],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function Qn(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(M(33))}function zl(e){return e[ri]||null}var ua=[],qn=-1;function vn(e){return{current:e}}function pe(e){0>qn||(e.current=ua[qn],ua[qn]=null,qn--)}function ue(e,t){qn++,ua[qn]=e.current,e.current=t}var mn={},De=vn(mn),Ye=vn(!1),Tn=mn;function sr(e,t){var n=e.type.contextTypes;if(!n)return mn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Ge(e){return e=e.childContextTypes,e!=null}function cl(){pe(Ye),pe(De)}function Ru(e,t,n){if(De.current!==mn)throw Error(M(168));ue(De,t),ue(Ye,n)}function fp(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(M(108,Fh(e)||"Unknown",i));return ve({},n,r)}function dl(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||mn,Tn=De.current,ue(De,e),ue(Ye,Ye.current),!0}function Mu(e,t,n){var r=e.stateNode;if(!r)throw Error(M(169));n?(e=fp(e,t,Tn),r.__reactInternalMemoizedMergedChildContext=e,pe(Ye),pe(De),ue(De,e)):pe(Ye),ue(Ye,n)}var Ot=null,Al=!1,po=!1;function hp(e){Ot===null?Ot=[e]:Ot.push(e)}function tg(e){Al=!0,hp(e)}function xn(){if(!po&&Ot!==null){po=!0;var e=0,t=le;try{var n=Ot;for(le=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}Ot=null,Al=!1}catch(i){throw Ot!==null&&(Ot=Ot.slice(e+1)),Od(Za,xn),i}finally{le=t,po=!1}}return null}var Kn=[],Yn=0,pl=null,fl=0,dt=[],pt=0,Ln=null,Bt=1,$t="";function wn(e,t){Kn[Yn++]=fl,Kn[Yn++]=pl,pl=e,fl=t}function mp(e,t,n){dt[pt++]=Bt,dt[pt++]=$t,dt[pt++]=Ln,Ln=e;var r=Bt;e=$t;var i=32-bt(r)-1;r&=~(1<<i),n+=1;var l=32-bt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,Bt=1<<32-bt(t)+i|n<<i|r,$t=l+e}else Bt=1<<l|n<<i|r,$t=e}function ss(e){e.return!==null&&(wn(e,1),mp(e,1,0))}function us(e){for(;e===pl;)pl=Kn[--Yn],Kn[Yn]=null,fl=Kn[--Yn],Kn[Yn]=null;for(;e===Ln;)Ln=dt[--pt],dt[pt]=null,$t=dt[--pt],dt[pt]=null,Bt=dt[--pt],dt[pt]=null}var ot=null,it=null,fe=!1,St=null;function gp(e,t){var n=ht(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function Du(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,ot=e,it=un(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,ot=e,it=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=Ln!==null?{id:Bt,overflow:$t}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=ht(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,ot=e,it=null,!0):!1;default:return!1}}function ca(e){return(e.mode&1)!==0&&(e.flags&128)===0}function da(e){if(fe){var t=it;if(t){var n=t;if(!Du(e,t)){if(ca(e))throw Error(M(418));t=un(n.nextSibling);var r=ot;t&&Du(e,t)?gp(r,n):(e.flags=e.flags&-4097|2,fe=!1,ot=e)}}else{if(ca(e))throw Error(M(418));e.flags=e.flags&-4097|2,fe=!1,ot=e}}}function Fu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;ot=e}function Ti(e){if(e!==ot)return!1;if(!fe)return Fu(e),fe=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!oa(e.type,e.memoizedProps)),t&&(t=it)){if(ca(e))throw vp(),Error(M(418));for(;t;)gp(e,t),t=un(t.nextSibling)}if(Fu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(M(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){it=un(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}it=null}}else it=ot?un(e.stateNode.nextSibling):null;return!0}function vp(){for(var e=it;e;)e=un(e.nextSibling)}function ur(){it=ot=null,fe=!1}function cs(e){St===null?St=[e]:St.push(e)}var ng=qt.ReactCurrentBatchConfig;function Cr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(M(309));var r=n.stateNode}if(!r)throw Error(M(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(M(284));if(!n._owner)throw Error(M(290,e))}return e}function Li(e,t){throw e=Object.prototype.toString.call(t),Error(M(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Ou(e){var t=e._init;return t(e._payload)}function xp(e){function t(m,v){if(e){var x=m.deletions;x===null?(m.deletions=[v],m.flags|=16):x.push(v)}}function n(m,v){if(!e)return null;for(;v!==null;)t(m,v),v=v.sibling;return null}function r(m,v){for(m=new Map;v!==null;)v.key!==null?m.set(v.key,v):m.set(v.index,v),v=v.sibling;return m}function i(m,v){return m=fn(m,v),m.index=0,m.sibling=null,m}function l(m,v,x){return m.index=x,e?(x=m.alternate,x!==null?(x=x.index,x<v?(m.flags|=2,v):x):(m.flags|=2,v)):(m.flags|=1048576,v)}function o(m){return e&&m.alternate===null&&(m.flags|=2),m}function a(m,v,x,b){return v===null||v.tag!==6?(v=yo(x,m.mode,b),v.return=m,v):(v=i(v,x),v.return=m,v)}function u(m,v,x,b){var N=x.type;return N===Hn?d(m,v,x.props.children,b,x.key):v!==null&&(v.elementType===N||typeof N=="object"&&N!==null&&N.$$typeof===Zt&&Ou(N)===v.type)?(b=i(v,x.props),b.ref=Cr(m,v,x),b.return=m,b):(b=Xi(x.type,x.key,x.props,null,m.mode,b),b.ref=Cr(m,v,x),b.return=m,b)}function c(m,v,x,b){return v===null||v.tag!==4||v.stateNode.containerInfo!==x.containerInfo||v.stateNode.implementation!==x.implementation?(v=ko(x,m.mode,b),v.return=m,v):(v=i(v,x.children||[]),v.return=m,v)}function d(m,v,x,b,N){return v===null||v.tag!==7?(v=En(x,m.mode,b,N),v.return=m,v):(v=i(v,x),v.return=m,v)}function p(m,v,x){if(typeof v=="string"&&v!==""||typeof v=="number")return v=yo(""+v,m.mode,x),v.return=m,v;if(typeof v=="object"&&v!==null){switch(v.$$typeof){case yi:return x=Xi(v.type,v.key,v.props,null,m.mode,x),x.ref=Cr(m,null,v),x.return=m,x;case $n:return v=ko(v,m.mode,x),v.return=m,v;case Zt:var b=v._init;return p(m,b(v._payload),x)}if(Ir(v)||wr(v))return v=En(v,m.mode,x,null),v.return=m,v;Li(m,v)}return null}function f(m,v,x,b){var N=v!==null?v.key:null;if(typeof x=="string"&&x!==""||typeof x=="number")return N!==null?null:a(m,v,""+x,b);if(typeof x=="object"&&x!==null){switch(x.$$typeof){case yi:return x.key===N?u(m,v,x,b):null;case $n:return x.key===N?c(m,v,x,b):null;case Zt:return N=x._init,f(m,v,N(x._payload),b)}if(Ir(x)||wr(x))return N!==null?null:d(m,v,x,b,null);Li(m,x)}return null}function h(m,v,x,b,N){if(typeof b=="string"&&b!==""||typeof b=="number")return m=m.get(x)||null,a(v,m,""+b,N);if(typeof b=="object"&&b!==null){switch(b.$$typeof){case yi:return m=m.get(b.key===null?x:b.key)||null,u(v,m,b,N);case $n:return m=m.get(b.key===null?x:b.key)||null,c(v,m,b,N);case Zt:var S=b._init;return h(m,v,x,S(b._payload),N)}if(Ir(b)||wr(b))return m=m.get(x)||null,d(v,m,b,N,null);Li(v,b)}return null}function k(m,v,x,b){for(var N=null,S=null,C=v,P=v=0,D=null;C!==null&&P<x.length;P++){C.index>P?(D=C,C=null):D=C.sibling;var A=f(m,C,x[P],b);if(A===null){C===null&&(C=D);break}e&&C&&A.alternate===null&&t(m,C),v=l(A,v,P),S===null?N=A:S.sibling=A,S=A,C=D}if(P===x.length)return n(m,C),fe&&wn(m,P),N;if(C===null){for(;P<x.length;P++)C=p(m,x[P],b),C!==null&&(v=l(C,v,P),S===null?N=C:S.sibling=C,S=C);return fe&&wn(m,P),N}for(C=r(m,C);P<x.length;P++)D=h(C,m,P,x[P],b),D!==null&&(e&&D.alternate!==null&&C.delete(D.key===null?P:D.key),v=l(D,v,P),S===null?N=D:S.sibling=D,S=D);return e&&C.forEach(function(j){return t(m,j)}),fe&&wn(m,P),N}function w(m,v,x,b){var N=wr(x);if(typeof N!="function")throw Error(M(150));if(x=N.call(x),x==null)throw Error(M(151));for(var S=N=null,C=v,P=v=0,D=null,A=x.next();C!==null&&!A.done;P++,A=x.next()){C.index>P?(D=C,C=null):D=C.sibling;var j=f(m,C,A.value,b);if(j===null){C===null&&(C=D);break}e&&C&&j.alternate===null&&t(m,C),v=l(j,v,P),S===null?N=j:S.sibling=j,S=j,C=D}if(A.done)return n(m,C),fe&&wn(m,P),N;if(C===null){for(;!A.done;P++,A=x.next())A=p(m,A.value,b),A!==null&&(v=l(A,v,P),S===null?N=A:S.sibling=A,S=A);return fe&&wn(m,P),N}for(C=r(m,C);!A.done;P++,A=x.next())A=h(C,m,P,A.value,b),A!==null&&(e&&A.alternate!==null&&C.delete(A.key===null?P:A.key),v=l(A,v,P),S===null?N=A:S.sibling=A,S=A);return e&&C.forEach(function(E){return t(m,E)}),fe&&wn(m,P),N}function I(m,v,x,b){if(typeof x=="object"&&x!==null&&x.type===Hn&&x.key===null&&(x=x.props.children),typeof x=="object"&&x!==null){switch(x.$$typeof){case yi:e:{for(var N=x.key,S=v;S!==null;){if(S.key===N){if(N=x.type,N===Hn){if(S.tag===7){n(m,S.sibling),v=i(S,x.props.children),v.return=m,m=v;break e}}else if(S.elementType===N||typeof N=="object"&&N!==null&&N.$$typeof===Zt&&Ou(N)===S.type){n(m,S.sibling),v=i(S,x.props),v.ref=Cr(m,S,x),v.return=m,m=v;break e}n(m,S);break}else t(m,S);S=S.sibling}x.type===Hn?(v=En(x.props.children,m.mode,b,x.key),v.return=m,m=v):(b=Xi(x.type,x.key,x.props,null,m.mode,b),b.ref=Cr(m,v,x),b.return=m,m=b)}return o(m);case $n:e:{for(S=x.key;v!==null;){if(v.key===S)if(v.tag===4&&v.stateNode.containerInfo===x.containerInfo&&v.stateNode.implementation===x.implementation){n(m,v.sibling),v=i(v,x.children||[]),v.return=m,m=v;break e}else{n(m,v);break}else t(m,v);v=v.sibling}v=ko(x,m.mode,b),v.return=m,m=v}return o(m);case Zt:return S=x._init,I(m,v,S(x._payload),b)}if(Ir(x))return k(m,v,x,b);if(wr(x))return w(m,v,x,b);Li(m,x)}return typeof x=="string"&&x!==""||typeof x=="number"?(x=""+x,v!==null&&v.tag===6?(n(m,v.sibling),v=i(v,x),v.return=m,m=v):(n(m,v),v=yo(x,m.mode,b),v.return=m,m=v),o(m)):n(m,v)}return I}var cr=xp(!0),yp=xp(!1),hl=vn(null),ml=null,Gn=null,ds=null;function ps(){ds=Gn=ml=null}function fs(e){var t=hl.current;pe(hl),e._currentValue=t}function pa(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function rr(e,t){ml=e,ds=Gn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(Ke=!0),e.firstContext=null)}function gt(e){var t=e._currentValue;if(ds!==e)if(e={context:e,memoizedValue:t,next:null},Gn===null){if(ml===null)throw Error(M(308));Gn=e,ml.dependencies={lanes:0,firstContext:e}}else Gn=Gn.next=e;return t}var jn=null;function hs(e){jn===null?jn=[e]:jn.push(e)}function kp(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,hs(t)):(n.next=i.next,i.next=n),t.interleaved=n,Wt(e,r)}function Wt(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var en=!1;function ms(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function wp(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Ht(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function cn(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,re&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Wt(e,n)}return i=r.interleaved,i===null?(t.next=t,hs(r)):(t.next=i.next,i.next=t),r.interleaved=t,Wt(e,n)}function Wi(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,es(e,n)}}function Bu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function gl(e,t,n,r){var i=e.updateQueue;en=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var u=a,c=u.next;u.next=null,o===null?l=c:o.next=c,o=u;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=u))}if(l!==null){var p=i.baseState;o=0,d=c=u=null,a=l;do{var f=a.lane,h=a.eventTime;if((r&f)===f){d!==null&&(d=d.next={eventTime:h,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var k=e,w=a;switch(f=t,h=n,w.tag){case 1:if(k=w.payload,typeof k=="function"){p=k.call(h,p,f);break e}p=k;break e;case 3:k.flags=k.flags&-65537|128;case 0:if(k=w.payload,f=typeof k=="function"?k.call(h,p,f):k,f==null)break e;p=ve({},p,f);break e;case 2:en=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,f=i.effects,f===null?i.effects=[a]:f.push(a))}else h={eventTime:h,lane:f,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=h,u=p):d=d.next=h,o|=f;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;f=a,a=f.next,f.next=null,i.lastBaseUpdate=f,i.shared.pending=null}}while(!0);if(d===null&&(u=p),i.baseState=u,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);In|=o,e.lanes=o,e.memoizedState=p}}function $u(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(M(191,i));i.call(r)}}}var mi={},It=vn(mi),ii=vn(mi),li=vn(mi);function Cn(e){if(e===mi)throw Error(M(174));return e}function gs(e,t){switch(ue(li,t),ue(ii,e),ue(It,mi),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Qo(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Qo(t,e)}pe(It),ue(It,t)}function dr(){pe(It),pe(ii),pe(li)}function Sp(e){Cn(li.current);var t=Cn(It.current),n=Qo(t,e.type);t!==n&&(ue(ii,e),ue(It,n))}function vs(e){ii.current===e&&(pe(It),pe(ii))}var me=vn(0);function vl(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var fo=[];function xs(){for(var e=0;e<fo.length;e++)fo[e]._workInProgressVersionPrimary=null;fo.length=0}var Qi=qt.ReactCurrentDispatcher,ho=qt.ReactCurrentBatchConfig,Pn=0,ge=null,be=null,Ce=null,xl=!1,Br=!1,oi=0,rg=0;function ze(){throw Error(M(321))}function ys(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!jt(e[n],t[n]))return!1;return!0}function ks(e,t,n,r,i,l){if(Pn=l,ge=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Qi.current=e===null||e.memoizedState===null?ag:sg,e=n(r,i),Br){l=0;do{if(Br=!1,oi=0,25<=l)throw Error(M(301));l+=1,Ce=be=null,t.updateQueue=null,Qi.current=ug,e=n(r,i)}while(Br)}if(Qi.current=yl,t=be!==null&&be.next!==null,Pn=0,Ce=be=ge=null,xl=!1,t)throw Error(M(300));return e}function ws(){var e=oi!==0;return oi=0,e}function Nt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return Ce===null?ge.memoizedState=Ce=e:Ce=Ce.next=e,Ce}function vt(){if(be===null){var e=ge.alternate;e=e!==null?e.memoizedState:null}else e=be.next;var t=Ce===null?ge.memoizedState:Ce.next;if(t!==null)Ce=t,be=e;else{if(e===null)throw Error(M(310));be=e,e={memoizedState:be.memoizedState,baseState:be.baseState,baseQueue:be.baseQueue,queue:be.queue,next:null},Ce===null?ge.memoizedState=Ce=e:Ce=Ce.next=e}return Ce}function ai(e,t){return typeof t=="function"?t(e):t}function mo(e){var t=vt(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=be,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,u=null,c=l;do{var d=c.lane;if((Pn&d)===d)u!==null&&(u=u.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var p={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};u===null?(a=u=p,o=r):u=u.next=p,ge.lanes|=d,In|=d}c=c.next}while(c!==null&&c!==l);u===null?o=r:u.next=a,jt(r,t.memoizedState)||(Ke=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=u,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,ge.lanes|=l,In|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function go(e){var t=vt(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);jt(l,t.memoizedState)||(Ke=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function bp(){}function _p(e,t){var n=ge,r=vt(),i=t(),l=!jt(r.memoizedState,i);if(l&&(r.memoizedState=i,Ke=!0),r=r.queue,Ss(Np.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||Ce!==null&&Ce.memoizedState.tag&1){if(n.flags|=2048,si(9,Cp.bind(null,n,r,i,t),void 0,null),Ne===null)throw Error(M(349));Pn&30||jp(n,t,i)}return i}function jp(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=ge.updateQueue,t===null?(t={lastEffect:null,stores:null},ge.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function Cp(e,t,n,r){t.value=n,t.getSnapshot=r,Ep(t)&&Tp(e)}function Np(e,t,n){return n(function(){Ep(t)&&Tp(e)})}function Ep(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!jt(e,n)}catch{return!0}}function Tp(e){var t=Wt(e,1);t!==null&&_t(t,e,1,-1)}function Hu(e){var t=Nt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:ai,lastRenderedState:e},t.queue=e,e=e.dispatch=og.bind(null,ge,e),[t.memoizedState,e]}function si(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=ge.updateQueue,t===null?(t={lastEffect:null,stores:null},ge.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function Lp(){return vt().memoizedState}function qi(e,t,n,r){var i=Nt();ge.flags|=e,i.memoizedState=si(1|t,n,void 0,r===void 0?null:r)}function Rl(e,t,n,r){var i=vt();r=r===void 0?null:r;var l=void 0;if(be!==null){var o=be.memoizedState;if(l=o.destroy,r!==null&&ys(r,o.deps)){i.memoizedState=si(t,n,l,r);return}}ge.flags|=e,i.memoizedState=si(1|t,n,l,r)}function Uu(e,t){return qi(8390656,8,e,t)}function Ss(e,t){return Rl(2048,8,e,t)}function Pp(e,t){return Rl(4,2,e,t)}function Ip(e,t){return Rl(4,4,e,t)}function zp(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function Ap(e,t,n){return n=n!=null?n.concat([e]):null,Rl(4,4,zp.bind(null,t,e),n)}function bs(){}function Rp(e,t){var n=vt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ys(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function Mp(e,t){var n=vt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ys(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function Dp(e,t,n){return Pn&21?(jt(n,t)||(n=Hd(),ge.lanes|=n,In|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,Ke=!0),e.memoizedState=n)}function ig(e,t){var n=le;le=n!==0&&4>n?n:4,e(!0);var r=ho.transition;ho.transition={};try{e(!1),t()}finally{le=n,ho.transition=r}}function Fp(){return vt().memoizedState}function lg(e,t,n){var r=pn(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},Op(e))Bp(t,n);else if(n=kp(e,t,n,r),n!==null){var i=He();_t(n,e,r,i),$p(n,t,r)}}function og(e,t,n){var r=pn(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(Op(e))Bp(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,jt(a,o)){var u=t.interleaved;u===null?(i.next=i,hs(t)):(i.next=u.next,u.next=i),t.interleaved=i;return}}catch{}finally{}n=kp(e,t,i,r),n!==null&&(i=He(),_t(n,e,r,i),$p(n,t,r))}}function Op(e){var t=e.alternate;return e===ge||t!==null&&t===ge}function Bp(e,t){Br=xl=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function $p(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,es(e,n)}}var yl={readContext:gt,useCallback:ze,useContext:ze,useEffect:ze,useImperativeHandle:ze,useInsertionEffect:ze,useLayoutEffect:ze,useMemo:ze,useReducer:ze,useRef:ze,useState:ze,useDebugValue:ze,useDeferredValue:ze,useTransition:ze,useMutableSource:ze,useSyncExternalStore:ze,useId:ze,unstable_isNewReconciler:!1},ag={readContext:gt,useCallback:function(e,t){return Nt().memoizedState=[e,t===void 0?null:t],e},useContext:gt,useEffect:Uu,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,qi(4194308,4,zp.bind(null,t,e),n)},useLayoutEffect:function(e,t){return qi(4194308,4,e,t)},useInsertionEffect:function(e,t){return qi(4,2,e,t)},useMemo:function(e,t){var n=Nt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=Nt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=lg.bind(null,ge,e),[r.memoizedState,e]},useRef:function(e){var t=Nt();return e={current:e},t.memoizedState=e},useState:Hu,useDebugValue:bs,useDeferredValue:function(e){return Nt().memoizedState=e},useTransition:function(){var e=Hu(!1),t=e[0];return e=ig.bind(null,e[1]),Nt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=ge,i=Nt();if(fe){if(n===void 0)throw Error(M(407));n=n()}else{if(n=t(),Ne===null)throw Error(M(349));Pn&30||jp(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Uu(Np.bind(null,r,l,e),[e]),r.flags|=2048,si(9,Cp.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=Nt(),t=Ne.identifierPrefix;if(fe){var n=$t,r=Bt;n=(r&~(1<<32-bt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=oi++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=rg++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},sg={readContext:gt,useCallback:Rp,useContext:gt,useEffect:Ss,useImperativeHandle:Ap,useInsertionEffect:Pp,useLayoutEffect:Ip,useMemo:Mp,useReducer:mo,useRef:Lp,useState:function(){return mo(ai)},useDebugValue:bs,useDeferredValue:function(e){var t=vt();return Dp(t,be.memoizedState,e)},useTransition:function(){var e=mo(ai)[0],t=vt().memoizedState;return[e,t]},useMutableSource:bp,useSyncExternalStore:_p,useId:Fp,unstable_isNewReconciler:!1},ug={readContext:gt,useCallback:Rp,useContext:gt,useEffect:Ss,useImperativeHandle:Ap,useInsertionEffect:Pp,useLayoutEffect:Ip,useMemo:Mp,useReducer:go,useRef:Lp,useState:function(){return go(ai)},useDebugValue:bs,useDeferredValue:function(e){var t=vt();return be===null?t.memoizedState=e:Dp(t,be.memoizedState,e)},useTransition:function(){var e=go(ai)[0],t=vt().memoizedState;return[e,t]},useMutableSource:bp,useSyncExternalStore:_p,useId:Fp,unstable_isNewReconciler:!1};function kt(e,t){if(e&&e.defaultProps){t=ve({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function fa(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:ve({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var Ml={isMounted:function(e){return(e=e._reactInternals)?Rn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=He(),i=pn(e),l=Ht(r,i);l.payload=t,n!=null&&(l.callback=n),t=cn(e,l,i),t!==null&&(_t(t,e,i,r),Wi(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=He(),i=pn(e),l=Ht(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=cn(e,l,i),t!==null&&(_t(t,e,i,r),Wi(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=He(),r=pn(e),i=Ht(n,r);i.tag=2,t!=null&&(i.callback=t),t=cn(e,i,r),t!==null&&(_t(t,e,r,n),Wi(t,e,r))}};function Vu(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!ei(n,r)||!ei(i,l):!0}function Hp(e,t,n){var r=!1,i=mn,l=t.contextType;return typeof l=="object"&&l!==null?l=gt(l):(i=Ge(t)?Tn:De.current,r=t.contextTypes,l=(r=r!=null)?sr(e,i):mn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=Ml,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function Wu(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&Ml.enqueueReplaceState(t,t.state,null)}function ha(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},ms(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=gt(l):(l=Ge(t)?Tn:De.current,i.context=sr(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(fa(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&Ml.enqueueReplaceState(i,i.state,null),gl(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function pr(e,t){try{var n="",r=t;do n+=Dh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function vo(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function ma(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var cg=typeof WeakMap=="function"?WeakMap:Map;function Up(e,t,n){n=Ht(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){wl||(wl=!0,ja=r),ma(e,t)},n}function Vp(e,t,n){n=Ht(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){ma(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){ma(e,t),typeof r!="function"&&(dn===null?dn=new Set([this]):dn.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function Qu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new cg;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=_g.bind(null,e,t,n),t.then(e,e))}function qu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Ku(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Ht(-1,1),t.tag=2,cn(n,t,1))),n.lanes|=1),e)}var dg=qt.ReactCurrentOwner,Ke=!1;function $e(e,t,n,r){t.child=e===null?yp(t,null,n,r):cr(t,e.child,n,r)}function Yu(e,t,n,r,i){n=n.render;var l=t.ref;return rr(t,i),r=ks(e,t,n,r,l,i),n=ws(),e!==null&&!Ke?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Qt(e,t,i)):(fe&&n&&ss(t),t.flags|=1,$e(e,t,r,i),t.child)}function Gu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!Ps(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,Wp(e,t,l,r,i)):(e=Xi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:ei,n(o,r)&&e.ref===t.ref)return Qt(e,t,i)}return t.flags|=1,e=fn(l,r),e.ref=t.ref,e.return=t,t.child=e}function Wp(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(ei(l,r)&&e.ref===t.ref)if(Ke=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(Ke=!0);else return t.lanes=e.lanes,Qt(e,t,i)}return ga(e,t,n,r,i)}function Qp(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},ue(Jn,rt),rt|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,ue(Jn,rt),rt|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,ue(Jn,rt),rt|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,ue(Jn,rt),rt|=r;return $e(e,t,i,n),t.child}function qp(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ga(e,t,n,r,i){var l=Ge(n)?Tn:De.current;return l=sr(t,l),rr(t,i),n=ks(e,t,n,r,l,i),r=ws(),e!==null&&!Ke?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Qt(e,t,i)):(fe&&r&&ss(t),t.flags|=1,$e(e,t,n,i),t.child)}function Xu(e,t,n,r,i){if(Ge(n)){var l=!0;dl(t)}else l=!1;if(rr(t,i),t.stateNode===null)Ki(e,t),Hp(t,n,r),ha(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var u=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=gt(c):(c=Ge(n)?Tn:De.current,c=sr(t,c));var d=n.getDerivedStateFromProps,p=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";p||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||u!==c)&&Wu(t,o,r,c),en=!1;var f=t.memoizedState;o.state=f,gl(t,r,o,i),u=t.memoizedState,a!==r||f!==u||Ye.current||en?(typeof d=="function"&&(fa(t,n,d,r),u=t.memoizedState),(a=en||Vu(t,n,a,r,f,u,c))?(p||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=u),o.props=r,o.state=u,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,wp(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:kt(t.type,a),o.props=c,p=t.pendingProps,f=o.context,u=n.contextType,typeof u=="object"&&u!==null?u=gt(u):(u=Ge(n)?Tn:De.current,u=sr(t,u));var h=n.getDerivedStateFromProps;(d=typeof h=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==p||f!==u)&&Wu(t,o,r,u),en=!1,f=t.memoizedState,o.state=f,gl(t,r,o,i);var k=t.memoizedState;a!==p||f!==k||Ye.current||en?(typeof h=="function"&&(fa(t,n,h,r),k=t.memoizedState),(c=en||Vu(t,n,c,r,f,k,u)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,k,u),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,k,u)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&f===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&f===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=k),o.props=r,o.state=k,o.context=u,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&f===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&f===e.memoizedState||(t.flags|=1024),r=!1)}return va(e,t,n,r,l,i)}function va(e,t,n,r,i,l){qp(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&Mu(t,n,!1),Qt(e,t,l);r=t.stateNode,dg.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=cr(t,e.child,null,l),t.child=cr(t,null,a,l)):$e(e,t,a,l),t.memoizedState=r.state,i&&Mu(t,n,!0),t.child}function Kp(e){var t=e.stateNode;t.pendingContext?Ru(e,t.pendingContext,t.pendingContext!==t.context):t.context&&Ru(e,t.context,!1),gs(e,t.containerInfo)}function Ju(e,t,n,r,i){return ur(),cs(i),t.flags|=256,$e(e,t,n,r),t.child}var xa={dehydrated:null,treeContext:null,retryLane:0};function ya(e){return{baseLanes:e,cachePool:null,transitions:null}}function Yp(e,t,n){var r=t.pendingProps,i=me.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),ue(me,i&1),e===null)return da(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Ol(o,r,0,null),e=En(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=ya(n),t.memoizedState=xa,e):_s(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return pg(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var u={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=u,t.deletions=null):(r=fn(i,u),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=fn(a,l):(l=En(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?ya(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=xa,r}return l=e.child,e=l.sibling,r=fn(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function _s(e,t){return t=Ol({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function Pi(e,t,n,r){return r!==null&&cs(r),cr(t,e.child,null,n),e=_s(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function pg(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=vo(Error(M(422))),Pi(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Ol({mode:"visible",children:r.children},i,0,null),l=En(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&cr(t,e.child,null,o),t.child.memoizedState=ya(o),t.memoizedState=xa,l);if(!(t.mode&1))return Pi(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(M(419)),r=vo(l,r,void 0),Pi(e,t,o,r)}if(a=(o&e.childLanes)!==0,Ke||a){if(r=Ne,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Wt(e,i),_t(r,e,i,-1))}return Ls(),r=vo(Error(M(421))),Pi(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=jg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,it=un(i.nextSibling),ot=t,fe=!0,St=null,e!==null&&(dt[pt++]=Bt,dt[pt++]=$t,dt[pt++]=Ln,Bt=e.id,$t=e.overflow,Ln=t),t=_s(t,r.children),t.flags|=4096,t)}function Zu(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),pa(e.return,t,n)}function xo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Gp(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if($e(e,t,r.children,n),r=me.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Zu(e,n,t);else if(e.tag===19)Zu(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(ue(me,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&vl(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),xo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&vl(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}xo(t,!0,n,null,l);break;case"together":xo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Ki(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Qt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),In|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(M(153));if(t.child!==null){for(e=t.child,n=fn(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=fn(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function fg(e,t,n){switch(t.tag){case 3:Kp(t),ur();break;case 5:Sp(t);break;case 1:Ge(t.type)&&dl(t);break;case 4:gs(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;ue(hl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(ue(me,me.current&1),t.flags|=128,null):n&t.child.childLanes?Yp(e,t,n):(ue(me,me.current&1),e=Qt(e,t,n),e!==null?e.sibling:null);ue(me,me.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Gp(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),ue(me,me.current),r)break;return null;case 22:case 23:return t.lanes=0,Qp(e,t,n)}return Qt(e,t,n)}var Xp,ka,Jp,Zp;Xp=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};ka=function(){};Jp=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,Cn(It.current);var l=null;switch(n){case"input":i=Ho(e,i),r=Ho(e,r),l=[];break;case"select":i=ve({},i,{value:void 0}),r=ve({},r,{value:void 0}),l=[];break;case"textarea":i=Wo(e,i),r=Wo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=ul)}qo(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(qr.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var u=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&u!==a&&(u!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||u&&u.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in u)u.hasOwnProperty(o)&&a[o]!==u[o]&&(n||(n={}),n[o]=u[o])}else n||(l||(l=[]),l.push(c,n)),n=u;else c==="dangerouslySetInnerHTML"?(u=u?u.__html:void 0,a=a?a.__html:void 0,u!=null&&a!==u&&(l=l||[]).push(c,u)):c==="children"?typeof u!="string"&&typeof u!="number"||(l=l||[]).push(c,""+u):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(qr.hasOwnProperty(c)?(u!=null&&c==="onScroll"&&de("scroll",e),l||a===u||(l=[])):(l=l||[]).push(c,u))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};Zp=function(e,t,n,r){n!==r&&(t.flags|=4)};function Nr(e,t){if(!fe)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function Ae(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function hg(e,t,n){var r=t.pendingProps;switch(us(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return Ae(t),null;case 1:return Ge(t.type)&&cl(),Ae(t),null;case 3:return r=t.stateNode,dr(),pe(Ye),pe(De),xs(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(Ti(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,St!==null&&(Ea(St),St=null))),ka(e,t),Ae(t),null;case 5:vs(t);var i=Cn(li.current);if(n=t.type,e!==null&&t.stateNode!=null)Jp(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(M(166));return Ae(t),null}if(e=Cn(It.current),Ti(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[Tt]=t,r[ri]=l,e=(t.mode&1)!==0,n){case"dialog":de("cancel",r),de("close",r);break;case"iframe":case"object":case"embed":de("load",r);break;case"video":case"audio":for(i=0;i<Ar.length;i++)de(Ar[i],r);break;case"source":de("error",r);break;case"img":case"image":case"link":de("error",r),de("load",r);break;case"details":de("toggle",r);break;case"input":su(r,l),de("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},de("invalid",r);break;case"textarea":cu(r,l),de("invalid",r)}qo(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&Ei(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&Ei(r.textContent,a,e),i=["children",""+a]):qr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&de("scroll",r)}switch(n){case"input":ki(r),uu(r,l,!0);break;case"textarea":ki(r),du(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=ul)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=Nd(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[Tt]=t,e[ri]=r,Xp(e,t,!1,!1),t.stateNode=e;e:{switch(o=Ko(n,r),n){case"dialog":de("cancel",e),de("close",e),i=r;break;case"iframe":case"object":case"embed":de("load",e),i=r;break;case"video":case"audio":for(i=0;i<Ar.length;i++)de(Ar[i],e);i=r;break;case"source":de("error",e),i=r;break;case"img":case"image":case"link":de("error",e),de("load",e),i=r;break;case"details":de("toggle",e),i=r;break;case"input":su(e,r),i=Ho(e,r),de("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=ve({},r,{value:void 0}),de("invalid",e);break;case"textarea":cu(e,r),i=Wo(e,r),de("invalid",e);break;default:i=r}qo(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var u=a[l];l==="style"?Ld(e,u):l==="dangerouslySetInnerHTML"?(u=u?u.__html:void 0,u!=null&&Ed(e,u)):l==="children"?typeof u=="string"?(n!=="textarea"||u!=="")&&Kr(e,u):typeof u=="number"&&Kr(e,""+u):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(qr.hasOwnProperty(l)?u!=null&&l==="onScroll"&&de("scroll",e):u!=null&&Ka(e,l,u,o))}switch(n){case"input":ki(e),uu(e,r,!1);break;case"textarea":ki(e),du(e);break;case"option":r.value!=null&&e.setAttribute("value",""+hn(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Zn(e,!!r.multiple,l,!1):r.defaultValue!=null&&Zn(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=ul)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return Ae(t),null;case 6:if(e&&t.stateNode!=null)Zp(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(M(166));if(n=Cn(li.current),Cn(It.current),Ti(t)){if(r=t.stateNode,n=t.memoizedProps,r[Tt]=t,(l=r.nodeValue!==n)&&(e=ot,e!==null))switch(e.tag){case 3:Ei(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&Ei(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[Tt]=t,t.stateNode=r}return Ae(t),null;case 13:if(pe(me),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(fe&&it!==null&&t.mode&1&&!(t.flags&128))vp(),ur(),t.flags|=98560,l=!1;else if(l=Ti(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(M(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(M(317));l[Tt]=t}else ur(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;Ae(t),l=!1}else St!==null&&(Ea(St),St=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||me.current&1?_e===0&&(_e=3):Ls())),t.updateQueue!==null&&(t.flags|=4),Ae(t),null);case 4:return dr(),ka(e,t),e===null&&ti(t.stateNode.containerInfo),Ae(t),null;case 10:return fs(t.type._context),Ae(t),null;case 17:return Ge(t.type)&&cl(),Ae(t),null;case 19:if(pe(me),l=t.memoizedState,l===null)return Ae(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)Nr(l,!1);else{if(_e!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=vl(e),o!==null){for(t.flags|=128,Nr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return ue(me,me.current&1|2),t.child}e=e.sibling}l.tail!==null&&ke()>fr&&(t.flags|=128,r=!0,Nr(l,!1),t.lanes=4194304)}else{if(!r)if(e=vl(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),Nr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!fe)return Ae(t),null}else 2*ke()-l.renderingStartTime>fr&&n!==1073741824&&(t.flags|=128,r=!0,Nr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=ke(),t.sibling=null,n=me.current,ue(me,r?n&1|2:n&1),t):(Ae(t),null);case 22:case 23:return Ts(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?rt&1073741824&&(Ae(t),t.subtreeFlags&6&&(t.flags|=8192)):Ae(t),null;case 24:return null;case 25:return null}throw Error(M(156,t.tag))}function mg(e,t){switch(us(t),t.tag){case 1:return Ge(t.type)&&cl(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return dr(),pe(Ye),pe(De),xs(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return vs(t),null;case 13:if(pe(me),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(M(340));ur()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return pe(me),null;case 4:return dr(),null;case 10:return fs(t.type._context),null;case 22:case 23:return Ts(),null;case 24:return null;default:return null}}var Ii=!1,Me=!1,gg=typeof WeakSet=="function"?WeakSet:Set,H=null;function Xn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){xe(e,t,r)}else n.current=null}function wa(e,t,n){try{n()}catch(r){xe(e,t,r)}}var ec=!1;function vg(e,t){if(ia=ol,e=ip(),as(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,u=-1,c=0,d=0,p=e,f=null;t:for(;;){for(var h;p!==n||i!==0&&p.nodeType!==3||(a=o+i),p!==l||r!==0&&p.nodeType!==3||(u=o+r),p.nodeType===3&&(o+=p.nodeValue.length),(h=p.firstChild)!==null;)f=p,p=h;for(;;){if(p===e)break t;if(f===n&&++c===i&&(a=o),f===l&&++d===r&&(u=o),(h=p.nextSibling)!==null)break;p=f,f=p.parentNode}p=h}n=a===-1||u===-1?null:{start:a,end:u}}else n=null}n=n||{start:0,end:0}}else n=null;for(la={focusedElem:e,selectionRange:n},ol=!1,H=t;H!==null;)if(t=H,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,H=e;else for(;H!==null;){t=H;try{var k=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(k!==null){var w=k.memoizedProps,I=k.memoizedState,m=t.stateNode,v=m.getSnapshotBeforeUpdate(t.elementType===t.type?w:kt(t.type,w),I);m.__reactInternalSnapshotBeforeUpdate=v}break;case 3:var x=t.stateNode.containerInfo;x.nodeType===1?x.textContent="":x.nodeType===9&&x.documentElement&&x.removeChild(x.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(M(163))}}catch(b){xe(t,t.return,b)}if(e=t.sibling,e!==null){e.return=t.return,H=e;break}H=t.return}return k=ec,ec=!1,k}function $r(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&wa(t,n,l)}i=i.next}while(i!==r)}}function Dl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function Sa(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function ef(e){var t=e.alternate;t!==null&&(e.alternate=null,ef(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[Tt],delete t[ri],delete t[sa],delete t[Zm],delete t[eg])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function tf(e){return e.tag===5||e.tag===3||e.tag===4}function tc(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||tf(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function ba(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=ul));else if(r!==4&&(e=e.child,e!==null))for(ba(e,t,n),e=e.sibling;e!==null;)ba(e,t,n),e=e.sibling}function _a(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(_a(e,t,n),e=e.sibling;e!==null;)_a(e,t,n),e=e.sibling}var Le=null,wt=!1;function Gt(e,t,n){for(n=n.child;n!==null;)nf(e,t,n),n=n.sibling}function nf(e,t,n){if(Pt&&typeof Pt.onCommitFiberUnmount=="function")try{Pt.onCommitFiberUnmount(Tl,n)}catch{}switch(n.tag){case 5:Me||Xn(n,t);case 6:var r=Le,i=wt;Le=null,Gt(e,t,n),Le=r,wt=i,Le!==null&&(wt?(e=Le,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):Le.removeChild(n.stateNode));break;case 18:Le!==null&&(wt?(e=Le,n=n.stateNode,e.nodeType===8?co(e.parentNode,n):e.nodeType===1&&co(e,n),Jr(e)):co(Le,n.stateNode));break;case 4:r=Le,i=wt,Le=n.stateNode.containerInfo,wt=!0,Gt(e,t,n),Le=r,wt=i;break;case 0:case 11:case 14:case 15:if(!Me&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&wa(n,t,o),i=i.next}while(i!==r)}Gt(e,t,n);break;case 1:if(!Me&&(Xn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){xe(n,t,a)}Gt(e,t,n);break;case 21:Gt(e,t,n);break;case 22:n.mode&1?(Me=(r=Me)||n.memoizedState!==null,Gt(e,t,n),Me=r):Gt(e,t,n);break;default:Gt(e,t,n)}}function nc(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new gg),t.forEach(function(r){var i=Cg.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function yt(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:Le=a.stateNode,wt=!1;break e;case 3:Le=a.stateNode.containerInfo,wt=!0;break e;case 4:Le=a.stateNode.containerInfo,wt=!0;break e}a=a.return}if(Le===null)throw Error(M(160));nf(l,o,i),Le=null,wt=!1;var u=i.alternate;u!==null&&(u.return=null),i.return=null}catch(c){xe(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)rf(t,e),t=t.sibling}function rf(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(yt(t,e),Ct(e),r&4){try{$r(3,e,e.return),Dl(3,e)}catch(w){xe(e,e.return,w)}try{$r(5,e,e.return)}catch(w){xe(e,e.return,w)}}break;case 1:yt(t,e),Ct(e),r&512&&n!==null&&Xn(n,n.return);break;case 5:if(yt(t,e),Ct(e),r&512&&n!==null&&Xn(n,n.return),e.flags&32){var i=e.stateNode;try{Kr(i,"")}catch(w){xe(e,e.return,w)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,u=e.updateQueue;if(e.updateQueue=null,u!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&jd(i,l),Ko(a,o);var c=Ko(a,l);for(o=0;o<u.length;o+=2){var d=u[o],p=u[o+1];d==="style"?Ld(i,p):d==="dangerouslySetInnerHTML"?Ed(i,p):d==="children"?Kr(i,p):Ka(i,d,p,c)}switch(a){case"input":Uo(i,l);break;case"textarea":Cd(i,l);break;case"select":var f=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var h=l.value;h!=null?Zn(i,!!l.multiple,h,!1):f!==!!l.multiple&&(l.defaultValue!=null?Zn(i,!!l.multiple,l.defaultValue,!0):Zn(i,!!l.multiple,l.multiple?[]:"",!1))}i[ri]=l}catch(w){xe(e,e.return,w)}}break;case 6:if(yt(t,e),Ct(e),r&4){if(e.stateNode===null)throw Error(M(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(w){xe(e,e.return,w)}}break;case 3:if(yt(t,e),Ct(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Jr(t.containerInfo)}catch(w){xe(e,e.return,w)}break;case 4:yt(t,e),Ct(e);break;case 13:yt(t,e),Ct(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(Ns=ke())),r&4&&nc(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Me=(c=Me)||d,yt(t,e),Me=c):yt(t,e),Ct(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for(H=e,d=e.child;d!==null;){for(p=H=d;H!==null;){switch(f=H,h=f.child,f.tag){case 0:case 11:case 14:case 15:$r(4,f,f.return);break;case 1:Xn(f,f.return);var k=f.stateNode;if(typeof k.componentWillUnmount=="function"){r=f,n=f.return;try{t=r,k.props=t.memoizedProps,k.state=t.memoizedState,k.componentWillUnmount()}catch(w){xe(r,n,w)}}break;case 5:Xn(f,f.return);break;case 22:if(f.memoizedState!==null){ic(p);continue}}h!==null?(h.return=f,H=h):ic(p)}d=d.sibling}e:for(d=null,p=e;;){if(p.tag===5){if(d===null){d=p;try{i=p.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=p.stateNode,u=p.memoizedProps.style,o=u!=null&&u.hasOwnProperty("display")?u.display:null,a.style.display=Td("display",o))}catch(w){xe(e,e.return,w)}}}else if(p.tag===6){if(d===null)try{p.stateNode.nodeValue=c?"":p.memoizedProps}catch(w){xe(e,e.return,w)}}else if((p.tag!==22&&p.tag!==23||p.memoizedState===null||p===e)&&p.child!==null){p.child.return=p,p=p.child;continue}if(p===e)break e;for(;p.sibling===null;){if(p.return===null||p.return===e)break e;d===p&&(d=null),p=p.return}d===p&&(d=null),p.sibling.return=p.return,p=p.sibling}}break;case 19:yt(t,e),Ct(e),r&4&&nc(e);break;case 21:break;default:yt(t,e),Ct(e)}}function Ct(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(tf(n)){var r=n;break e}n=n.return}throw Error(M(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Kr(i,""),r.flags&=-33);var l=tc(e);_a(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=tc(e);ba(e,a,o);break;default:throw Error(M(161))}}catch(u){xe(e,e.return,u)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function xg(e,t,n){H=e,lf(e)}function lf(e,t,n){for(var r=(e.mode&1)!==0;H!==null;){var i=H,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||Ii;if(!o){var a=i.alternate,u=a!==null&&a.memoizedState!==null||Me;a=Ii;var c=Me;if(Ii=o,(Me=u)&&!c)for(H=i;H!==null;)o=H,u=o.child,o.tag===22&&o.memoizedState!==null?lc(i):u!==null?(u.return=o,H=u):lc(i);for(;l!==null;)H=l,lf(l),l=l.sibling;H=i,Ii=a,Me=c}rc(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,H=l):rc(e)}}function rc(e){for(;H!==null;){var t=H;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Me||Dl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Me)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:kt(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&$u(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}$u(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var u=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":u.autoFocus&&n.focus();break;case"img":u.src&&(n.src=u.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var p=d.dehydrated;p!==null&&Jr(p)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(M(163))}Me||t.flags&512&&Sa(t)}catch(f){xe(t,t.return,f)}}if(t===e){H=null;break}if(n=t.sibling,n!==null){n.return=t.return,H=n;break}H=t.return}}function ic(e){for(;H!==null;){var t=H;if(t===e){H=null;break}var n=t.sibling;if(n!==null){n.return=t.return,H=n;break}H=t.return}}function lc(e){for(;H!==null;){var t=H;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Dl(4,t)}catch(u){xe(t,n,u)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(u){xe(t,i,u)}}var l=t.return;try{Sa(t)}catch(u){xe(t,l,u)}break;case 5:var o=t.return;try{Sa(t)}catch(u){xe(t,o,u)}}}catch(u){xe(t,t.return,u)}if(t===e){H=null;break}var a=t.sibling;if(a!==null){a.return=t.return,H=a;break}H=t.return}}var yg=Math.ceil,kl=qt.ReactCurrentDispatcher,js=qt.ReactCurrentOwner,mt=qt.ReactCurrentBatchConfig,re=0,Ne=null,Se=null,Pe=0,rt=0,Jn=vn(0),_e=0,ui=null,In=0,Fl=0,Cs=0,Hr=null,qe=null,Ns=0,fr=1/0,Ft=null,wl=!1,ja=null,dn=null,zi=!1,ln=null,Sl=0,Ur=0,Ca=null,Yi=-1,Gi=0;function He(){return re&6?ke():Yi!==-1?Yi:Yi=ke()}function pn(e){return e.mode&1?re&2&&Pe!==0?Pe&-Pe:ng.transition!==null?(Gi===0&&(Gi=Hd()),Gi):(e=le,e!==0||(e=window.event,e=e===void 0?16:Yd(e.type)),e):1}function _t(e,t,n,r){if(50<Ur)throw Ur=0,Ca=null,Error(M(185));pi(e,n,r),(!(re&2)||e!==Ne)&&(e===Ne&&(!(re&2)&&(Fl|=n),_e===4&&nn(e,Pe)),Xe(e,r),n===1&&re===0&&!(t.mode&1)&&(fr=ke()+500,Al&&xn()))}function Xe(e,t){var n=e.callbackNode;nm(e,t);var r=ll(e,e===Ne?Pe:0);if(r===0)n!==null&&hu(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&hu(n),t===1)e.tag===0?tg(oc.bind(null,e)):hp(oc.bind(null,e)),Xm(function(){!(re&6)&&xn()}),n=null;else{switch(Ud(r)){case 1:n=Za;break;case 4:n=Bd;break;case 16:n=il;break;case 536870912:n=$d;break;default:n=il}n=ff(n,of.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function of(e,t){if(Yi=-1,Gi=0,re&6)throw Error(M(327));var n=e.callbackNode;if(ir()&&e.callbackNode!==n)return null;var r=ll(e,e===Ne?Pe:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=bl(e,r);else{t=r;var i=re;re|=2;var l=sf();(Ne!==e||Pe!==t)&&(Ft=null,fr=ke()+500,Nn(e,t));do try{Sg();break}catch(a){af(e,a)}while(!0);ps(),kl.current=l,re=i,Se!==null?t=0:(Ne=null,Pe=0,t=_e)}if(t!==0){if(t===2&&(i=Zo(e),i!==0&&(r=i,t=Na(e,i))),t===1)throw n=ui,Nn(e,0),nn(e,r),Xe(e,ke()),n;if(t===6)nn(e,r);else{if(i=e.current.alternate,!(r&30)&&!kg(i)&&(t=bl(e,r),t===2&&(l=Zo(e),l!==0&&(r=l,t=Na(e,l))),t===1))throw n=ui,Nn(e,0),nn(e,r),Xe(e,ke()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(M(345));case 2:Sn(e,qe,Ft);break;case 3:if(nn(e,r),(r&130023424)===r&&(t=Ns+500-ke(),10<t)){if(ll(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){He(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=aa(Sn.bind(null,e,qe,Ft),t);break}Sn(e,qe,Ft);break;case 4:if(nn(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-bt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=ke()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*yg(r/1960))-r,10<r){e.timeoutHandle=aa(Sn.bind(null,e,qe,Ft),r);break}Sn(e,qe,Ft);break;case 5:Sn(e,qe,Ft);break;default:throw Error(M(329))}}}return Xe(e,ke()),e.callbackNode===n?of.bind(null,e):null}function Na(e,t){var n=Hr;return e.current.memoizedState.isDehydrated&&(Nn(e,t).flags|=256),e=bl(e,t),e!==2&&(t=qe,qe=n,t!==null&&Ea(t)),e}function Ea(e){qe===null?qe=e:qe.push.apply(qe,e)}function kg(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!jt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function nn(e,t){for(t&=~Cs,t&=~Fl,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-bt(t),r=1<<n;e[n]=-1,t&=~r}}function oc(e){if(re&6)throw Error(M(327));ir();var t=ll(e,0);if(!(t&1))return Xe(e,ke()),null;var n=bl(e,t);if(e.tag!==0&&n===2){var r=Zo(e);r!==0&&(t=r,n=Na(e,r))}if(n===1)throw n=ui,Nn(e,0),nn(e,t),Xe(e,ke()),n;if(n===6)throw Error(M(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,Sn(e,qe,Ft),Xe(e,ke()),null}function Es(e,t){var n=re;re|=1;try{return e(t)}finally{re=n,re===0&&(fr=ke()+500,Al&&xn())}}function zn(e){ln!==null&&ln.tag===0&&!(re&6)&&ir();var t=re;re|=1;var n=mt.transition,r=le;try{if(mt.transition=null,le=1,e)return e()}finally{le=r,mt.transition=n,re=t,!(re&6)&&xn()}}function Ts(){rt=Jn.current,pe(Jn)}function Nn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,Gm(n)),Se!==null)for(n=Se.return;n!==null;){var r=n;switch(us(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&cl();break;case 3:dr(),pe(Ye),pe(De),xs();break;case 5:vs(r);break;case 4:dr();break;case 13:pe(me);break;case 19:pe(me);break;case 10:fs(r.type._context);break;case 22:case 23:Ts()}n=n.return}if(Ne=e,Se=e=fn(e.current,null),Pe=rt=t,_e=0,ui=null,Cs=Fl=In=0,qe=Hr=null,jn!==null){for(t=0;t<jn.length;t++)if(n=jn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}jn=null}return e}function af(e,t){do{var n=Se;try{if(ps(),Qi.current=yl,xl){for(var r=ge.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}xl=!1}if(Pn=0,Ce=be=ge=null,Br=!1,oi=0,js.current=null,n===null||n.return===null){_e=1,ui=t,Se=null;break}e:{var l=e,o=n.return,a=n,u=t;if(t=Pe,a.flags|=32768,u!==null&&typeof u=="object"&&typeof u.then=="function"){var c=u,d=a,p=d.tag;if(!(d.mode&1)&&(p===0||p===11||p===15)){var f=d.alternate;f?(d.updateQueue=f.updateQueue,d.memoizedState=f.memoizedState,d.lanes=f.lanes):(d.updateQueue=null,d.memoizedState=null)}var h=qu(o);if(h!==null){h.flags&=-257,Ku(h,o,a,l,t),h.mode&1&&Qu(l,c,t),t=h,u=c;var k=t.updateQueue;if(k===null){var w=new Set;w.add(u),t.updateQueue=w}else k.add(u);break e}else{if(!(t&1)){Qu(l,c,t),Ls();break e}u=Error(M(426))}}else if(fe&&a.mode&1){var I=qu(o);if(I!==null){!(I.flags&65536)&&(I.flags|=256),Ku(I,o,a,l,t),cs(pr(u,a));break e}}l=u=pr(u,a),_e!==4&&(_e=2),Hr===null?Hr=[l]:Hr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var m=Up(l,u,t);Bu(l,m);break e;case 1:a=u;var v=l.type,x=l.stateNode;if(!(l.flags&128)&&(typeof v.getDerivedStateFromError=="function"||x!==null&&typeof x.componentDidCatch=="function"&&(dn===null||!dn.has(x)))){l.flags|=65536,t&=-t,l.lanes|=t;var b=Vp(l,a,t);Bu(l,b);break e}}l=l.return}while(l!==null)}cf(n)}catch(N){t=N,Se===n&&n!==null&&(Se=n=n.return);continue}break}while(!0)}function sf(){var e=kl.current;return kl.current=yl,e===null?yl:e}function Ls(){(_e===0||_e===3||_e===2)&&(_e=4),Ne===null||!(In&268435455)&&!(Fl&268435455)||nn(Ne,Pe)}function bl(e,t){var n=re;re|=2;var r=sf();(Ne!==e||Pe!==t)&&(Ft=null,Nn(e,t));do try{wg();break}catch(i){af(e,i)}while(!0);if(ps(),re=n,kl.current=r,Se!==null)throw Error(M(261));return Ne=null,Pe=0,_e}function wg(){for(;Se!==null;)uf(Se)}function Sg(){for(;Se!==null&&!qh();)uf(Se)}function uf(e){var t=pf(e.alternate,e,rt);e.memoizedProps=e.pendingProps,t===null?cf(e):Se=t,js.current=null}function cf(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=mg(n,t),n!==null){n.flags&=32767,Se=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{_e=6,Se=null;return}}else if(n=hg(n,t,rt),n!==null){Se=n;return}if(t=t.sibling,t!==null){Se=t;return}Se=t=e}while(t!==null);_e===0&&(_e=5)}function Sn(e,t,n){var r=le,i=mt.transition;try{mt.transition=null,le=1,bg(e,t,n,r)}finally{mt.transition=i,le=r}return null}function bg(e,t,n,r){do ir();while(ln!==null);if(re&6)throw Error(M(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(M(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(rm(e,l),e===Ne&&(Se=Ne=null,Pe=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||zi||(zi=!0,ff(il,function(){return ir(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=mt.transition,mt.transition=null;var o=le;le=1;var a=re;re|=4,js.current=null,vg(e,n),rf(n,e),Um(la),ol=!!ia,la=ia=null,e.current=n,xg(n),Kh(),re=a,le=o,mt.transition=l}else e.current=n;if(zi&&(zi=!1,ln=e,Sl=i),l=e.pendingLanes,l===0&&(dn=null),Xh(n.stateNode),Xe(e,ke()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(wl)throw wl=!1,e=ja,ja=null,e;return Sl&1&&e.tag!==0&&ir(),l=e.pendingLanes,l&1?e===Ca?Ur++:(Ur=0,Ca=e):Ur=0,xn(),null}function ir(){if(ln!==null){var e=Ud(Sl),t=mt.transition,n=le;try{if(mt.transition=null,le=16>e?16:e,ln===null)var r=!1;else{if(e=ln,ln=null,Sl=0,re&6)throw Error(M(331));var i=re;for(re|=4,H=e.current;H!==null;){var l=H,o=l.child;if(H.flags&16){var a=l.deletions;if(a!==null){for(var u=0;u<a.length;u++){var c=a[u];for(H=c;H!==null;){var d=H;switch(d.tag){case 0:case 11:case 15:$r(8,d,l)}var p=d.child;if(p!==null)p.return=d,H=p;else for(;H!==null;){d=H;var f=d.sibling,h=d.return;if(ef(d),d===c){H=null;break}if(f!==null){f.return=h,H=f;break}H=h}}}var k=l.alternate;if(k!==null){var w=k.child;if(w!==null){k.child=null;do{var I=w.sibling;w.sibling=null,w=I}while(w!==null)}}H=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,H=o;else e:for(;H!==null;){if(l=H,l.flags&2048)switch(l.tag){case 0:case 11:case 15:$r(9,l,l.return)}var m=l.sibling;if(m!==null){m.return=l.return,H=m;break e}H=l.return}}var v=e.current;for(H=v;H!==null;){o=H;var x=o.child;if(o.subtreeFlags&2064&&x!==null)x.return=o,H=x;else e:for(o=v;H!==null;){if(a=H,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Dl(9,a)}}catch(N){xe(a,a.return,N)}if(a===o){H=null;break e}var b=a.sibling;if(b!==null){b.return=a.return,H=b;break e}H=a.return}}if(re=i,xn(),Pt&&typeof Pt.onPostCommitFiberRoot=="function")try{Pt.onPostCommitFiberRoot(Tl,e)}catch{}r=!0}return r}finally{le=n,mt.transition=t}}return!1}function ac(e,t,n){t=pr(n,t),t=Up(e,t,1),e=cn(e,t,1),t=He(),e!==null&&(pi(e,1,t),Xe(e,t))}function xe(e,t,n){if(e.tag===3)ac(e,e,n);else for(;t!==null;){if(t.tag===3){ac(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(dn===null||!dn.has(r))){e=pr(n,e),e=Vp(t,e,1),t=cn(t,e,1),e=He(),t!==null&&(pi(t,1,e),Xe(t,e));break}}t=t.return}}function _g(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=He(),e.pingedLanes|=e.suspendedLanes&n,Ne===e&&(Pe&n)===n&&(_e===4||_e===3&&(Pe&130023424)===Pe&&500>ke()-Ns?Nn(e,0):Cs|=n),Xe(e,t)}function df(e,t){t===0&&(e.mode&1?(t=bi,bi<<=1,!(bi&130023424)&&(bi=4194304)):t=1);var n=He();e=Wt(e,t),e!==null&&(pi(e,t,n),Xe(e,n))}function jg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),df(e,n)}function Cg(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(M(314))}r!==null&&r.delete(t),df(e,n)}var pf;pf=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Ye.current)Ke=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return Ke=!1,fg(e,t,n);Ke=!!(e.flags&131072)}else Ke=!1,fe&&t.flags&1048576&&mp(t,fl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Ki(e,t),e=t.pendingProps;var i=sr(t,De.current);rr(t,n),i=ks(null,t,r,e,i,n);var l=ws();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Ge(r)?(l=!0,dl(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,ms(t),i.updater=Ml,t.stateNode=i,i._reactInternals=t,ha(t,r,e,n),t=va(null,t,r,!0,l,n)):(t.tag=0,fe&&l&&ss(t),$e(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Ki(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=Eg(r),e=kt(r,e),i){case 0:t=ga(null,t,r,e,n);break e;case 1:t=Xu(null,t,r,e,n);break e;case 11:t=Yu(null,t,r,e,n);break e;case 14:t=Gu(null,t,r,kt(r.type,e),n);break e}throw Error(M(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:kt(r,i),ga(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:kt(r,i),Xu(e,t,r,i,n);case 3:e:{if(Kp(t),e===null)throw Error(M(387));r=t.pendingProps,l=t.memoizedState,i=l.element,wp(e,t),gl(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=pr(Error(M(423)),t),t=Ju(e,t,r,n,i);break e}else if(r!==i){i=pr(Error(M(424)),t),t=Ju(e,t,r,n,i);break e}else for(it=un(t.stateNode.containerInfo.firstChild),ot=t,fe=!0,St=null,n=yp(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(ur(),r===i){t=Qt(e,t,n);break e}$e(e,t,r,n)}t=t.child}return t;case 5:return Sp(t),e===null&&da(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,oa(r,i)?o=null:l!==null&&oa(r,l)&&(t.flags|=32),qp(e,t),$e(e,t,o,n),t.child;case 6:return e===null&&da(t),null;case 13:return Yp(e,t,n);case 4:return gs(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=cr(t,null,r,n):$e(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:kt(r,i),Yu(e,t,r,i,n);case 7:return $e(e,t,t.pendingProps,n),t.child;case 8:return $e(e,t,t.pendingProps.children,n),t.child;case 12:return $e(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,ue(hl,r._currentValue),r._currentValue=o,l!==null)if(jt(l.value,o)){if(l.children===i.children&&!Ye.current){t=Qt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var u=a.firstContext;u!==null;){if(u.context===r){if(l.tag===1){u=Ht(-1,n&-n),u.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?u.next=u:(u.next=d.next,d.next=u),c.pending=u}}l.lanes|=n,u=l.alternate,u!==null&&(u.lanes|=n),pa(l.return,n,t),a.lanes|=n;break}u=u.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(M(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),pa(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}$e(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,rr(t,n),i=gt(i),r=r(i),t.flags|=1,$e(e,t,r,n),t.child;case 14:return r=t.type,i=kt(r,t.pendingProps),i=kt(r.type,i),Gu(e,t,r,i,n);case 15:return Wp(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:kt(r,i),Ki(e,t),t.tag=1,Ge(r)?(e=!0,dl(t)):e=!1,rr(t,n),Hp(t,r,i),ha(t,r,i,n),va(null,t,r,!0,e,n);case 19:return Gp(e,t,n);case 22:return Qp(e,t,n)}throw Error(M(156,t.tag))};function ff(e,t){return Od(e,t)}function Ng(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function ht(e,t,n,r){return new Ng(e,t,n,r)}function Ps(e){return e=e.prototype,!(!e||!e.isReactComponent)}function Eg(e){if(typeof e=="function")return Ps(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Ga)return 11;if(e===Xa)return 14}return 2}function fn(e,t){var n=e.alternate;return n===null?(n=ht(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Xi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")Ps(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Hn:return En(n.children,i,l,t);case Ya:o=8,i|=8;break;case Fo:return e=ht(12,n,t,i|2),e.elementType=Fo,e.lanes=l,e;case Oo:return e=ht(13,n,t,i),e.elementType=Oo,e.lanes=l,e;case Bo:return e=ht(19,n,t,i),e.elementType=Bo,e.lanes=l,e;case Sd:return Ol(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case kd:o=10;break e;case wd:o=9;break e;case Ga:o=11;break e;case Xa:o=14;break e;case Zt:o=16,r=null;break e}throw Error(M(130,e==null?e:typeof e,""))}return t=ht(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function En(e,t,n,r){return e=ht(7,e,r,t),e.lanes=n,e}function Ol(e,t,n,r){return e=ht(22,e,r,t),e.elementType=Sd,e.lanes=n,e.stateNode={isHidden:!1},e}function yo(e,t,n){return e=ht(6,e,null,t),e.lanes=n,e}function ko(e,t,n){return t=ht(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function Tg(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Zl(0),this.expirationTimes=Zl(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Zl(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function Is(e,t,n,r,i,l,o,a,u){return e=new Tg(e,t,n,a,u),t===1?(t=1,l===!0&&(t|=8)):t=0,l=ht(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},ms(l),e}function Lg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:$n,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function hf(e){if(!e)return mn;e=e._reactInternals;e:{if(Rn(e)!==e||e.tag!==1)throw Error(M(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Ge(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(M(171))}if(e.tag===1){var n=e.type;if(Ge(n))return fp(e,n,t)}return t}function mf(e,t,n,r,i,l,o,a,u){return e=Is(n,r,!0,e,i,l,o,a,u),e.context=hf(null),n=e.current,r=He(),i=pn(n),l=Ht(r,i),l.callback=t??null,cn(n,l,i),e.current.lanes=i,pi(e,i,r),Xe(e,r),e}function Bl(e,t,n,r){var i=t.current,l=He(),o=pn(i);return n=hf(n),t.context===null?t.context=n:t.pendingContext=n,t=Ht(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=cn(i,t,o),e!==null&&(_t(e,i,o,l),Wi(e,i,o)),o}function _l(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function sc(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function zs(e,t){sc(e,t),(e=e.alternate)&&sc(e,t)}function Pg(){return null}var gf=typeof reportError=="function"?reportError:function(e){console.error(e)};function As(e){this._internalRoot=e}$l.prototype.render=As.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(M(409));Bl(e,t,null,null)};$l.prototype.unmount=As.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;zn(function(){Bl(null,e,null,null)}),t[Vt]=null}};function $l(e){this._internalRoot=e}$l.prototype.unstable_scheduleHydration=function(e){if(e){var t=Qd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<tn.length&&t!==0&&t<tn[n].priority;n++);tn.splice(n,0,e),n===0&&Kd(e)}};function Rs(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Hl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function uc(){}function Ig(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=_l(o);l.call(c)}}var o=mf(t,r,e,0,null,!1,!1,"",uc);return e._reactRootContainer=o,e[Vt]=o.current,ti(e.nodeType===8?e.parentNode:e),zn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=_l(u);a.call(c)}}var u=Is(e,0,!1,null,null,!1,!1,"",uc);return e._reactRootContainer=u,e[Vt]=u.current,ti(e.nodeType===8?e.parentNode:e),zn(function(){Bl(t,u,n,r)}),u}function Ul(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var u=_l(o);a.call(u)}}Bl(t,o,e,i)}else o=Ig(n,t,e,i,r);return _l(o)}Vd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=zr(t.pendingLanes);n!==0&&(es(t,n|1),Xe(t,ke()),!(re&6)&&(fr=ke()+500,xn()))}break;case 13:zn(function(){var r=Wt(e,1);if(r!==null){var i=He();_t(r,e,1,i)}}),zs(e,1)}};ts=function(e){if(e.tag===13){var t=Wt(e,134217728);if(t!==null){var n=He();_t(t,e,134217728,n)}zs(e,134217728)}};Wd=function(e){if(e.tag===13){var t=pn(e),n=Wt(e,t);if(n!==null){var r=He();_t(n,e,t,r)}zs(e,t)}};Qd=function(){return le};qd=function(e,t){var n=le;try{return le=e,t()}finally{le=n}};Go=function(e,t,n){switch(t){case"input":if(Uo(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=zl(r);if(!i)throw Error(M(90));_d(r),Uo(r,i)}}}break;case"textarea":Cd(e,n);break;case"select":t=n.value,t!=null&&Zn(e,!!n.multiple,t,!1)}};zd=Es;Ad=zn;var zg={usingClientEntryPoint:!1,Events:[hi,Qn,zl,Pd,Id,Es]},Er={findFiberByHostInstance:_n,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},Ag={bundleType:Er.bundleType,version:Er.version,rendererPackageName:Er.rendererPackageName,rendererConfig:Er.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:qt.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=Dd(e),e===null?null:e.stateNode},findFiberByHostInstance:Er.findFiberByHostInstance||Pg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var Ai=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!Ai.isDisabled&&Ai.supportsFiber)try{Tl=Ai.inject(Ag),Pt=Ai}catch{}}st.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=zg;st.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!Rs(t))throw Error(M(200));return Lg(e,t,null,n)};st.createRoot=function(e,t){if(!Rs(e))throw Error(M(299));var n=!1,r="",i=gf;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=Is(e,1,!1,null,null,n,!1,r,i),e[Vt]=t.current,ti(e.nodeType===8?e.parentNode:e),new As(t)};st.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(M(188)):(e=Object.keys(e).join(","),Error(M(268,e)));return e=Dd(t),e=e===null?null:e.stateNode,e};st.flushSync=function(e){return zn(e)};st.hydrate=function(e,t,n){if(!Hl(t))throw Error(M(200));return Ul(null,e,t,!0,n)};st.hydrateRoot=function(e,t,n){if(!Rs(e))throw Error(M(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=gf;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=mf(t,null,e,1,n??null,i,!1,l,o),e[Vt]=t.current,ti(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new $l(t)};st.render=function(e,t,n){if(!Hl(t))throw Error(M(200));return Ul(null,e,t,!1,n)};st.unmountComponentAtNode=function(e){if(!Hl(e))throw Error(M(40));return e._reactRootContainer?(zn(function(){Ul(null,null,e,!1,function(){e._reactRootContainer=null,e[Vt]=null})}),!0):!1};st.unstable_batchedUpdates=Es;st.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Hl(n))throw Error(M(200));if(e==null||e._reactInternals===void 0)throw Error(M(38));return Ul(e,t,n,!1,r)};st.version="18.3.1-next-f1338f8080-20240426";function vf(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(vf)}catch(e){console.error(e)}}vf(),gd.exports=st;var Rg=gd.exports,cc=Rg;Mo.createRoot=cc.createRoot,Mo.hydrateRoot=cc.hydrateRoot;const Mg=new Set(["user","human"]);function Dg(e){return e?Mg.has(e.toLowerCase()):!1}function xf(e){return Dg(e)?"You (Human)":e}const Fg="",Og=({selection:e,onSelect:t,onRefresh:n})=>{const[r,i]=z.useState(null),[l,o]=z.useState(new Set(["all"])),[a,u]=z.useState(!0),[c,d]=z.useState(null),p=async()=>{try{const v=await fetch(`${Fg}/api/hierarchy`);if(!v.ok)throw new Error("Failed to fetch hierarchy");const x=await v.json();i(x),d(null)}catch(v){d(v instanceof Error?v.message:"Unknown error")}finally{u(!1)}};z.useEffect(()=>{p();const v=setInterval(p,5e3);return()=>clearInterval(v)},[]);const f=v=>{o(x=>{const b=new Set(x);return b.has(v)?b.delete(v):b.add(v),b})},h=v=>{var x;if(v.type==="root")t({type:"overview"});else if(v.type==="agent")t({type:"agent",agentId:v.id});else if(v.type==="thread"){const b=(x=r==null?void 0:r.root.children)==null?void 0:x.find(N=>{var S;return(S=N.children)==null?void 0:S.some(C=>C.id===v.id)});t({type:"thread",agentId:b==null?void 0:b.id,threadId:v.id})}},k=v=>v.type==="root"&&e.type==="overview"||v.type==="agent"&&e.type==="agent"&&e.agentId===v.id||v.type==="thread"&&e.threadId===v.id,w=v=>!v||v.length===0?null:s.jsx("span",{className:"badges",children:v.map((x,b)=>s.jsxs("span",{className:`badge badge-${x.type}`,title:`${x.count} ${x.type}`,children:[x.type==="pending"&&"⏳",x.type==="unread"&&"📬",x.type==="running"&&"▶️",x.count]},b))}),I=v=>{if(!v)return null;const x={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return s.jsx("span",{className:"status-indicator",style:{backgroundColor:x[v]||x.idle},title:v})},m=(v,x=0)=>{const b=l.has(v.id),N=v.children&&v.children.length>0,S=k(v);return s.jsxs("div",{className:"tree-node",children:[s.jsxs("div",{className:`tree-node-content ${S?"selected":""} ${v.type}`,style:{paddingLeft:`${x*16+8}px`},onClick:()=>h(v),children:[N&&s.jsx("span",{className:`expand-icon ${b?"expanded":""}`,onClick:C=>{C.stopPropagation(),f(v.id)},children:b?"▼":"▶"}),!N&&s.jsx("span",{className:"expand-icon-placeholder"}),v.type==="agent"&&I(v.status),s.jsx("span",{className:"node-label",children:v.type==="agent"?xf(v.id):v.label}),w(v.badges)]}),N&&b&&s.jsx("div",{className:"tree-children",children:v.children.map(C=>m(C,x+1))})]},v.id)};return a&&!r?s.jsx("div",{className:"hierarchy-tree loading",children:"Loading..."}):c?s.jsxs("div",{className:"hierarchy-tree error",children:[s.jsxs("p",{children:["Error: ",c]}),s.jsx("button",{onClick:p,children:"Retry"})]}):s.jsxs("div",{className:"hierarchy-tree",children:[s.jsxs("div",{className:"tree-header",children:[s.jsx("h3",{children:"Agents"}),s.jsx("button",{className:"refresh-btn",onClick:()=>{p(),n==null||n()},title:"Refresh",children:"↻"})]}),s.jsx("div",{className:"tree-content",children:r&&m(r.root)}),r&&s.jsx("div",{className:"tree-footer",children:s.jsxs("div",{className:"aggregate-stats",children:[s.jsxs("span",{title:"Total agents",children:[r.aggregate.total_agents," agents"]}),s.jsxs("span",{title:"Active",children:[r.aggregate.active_agents," active"]}),r.aggregate.pending_approvals>0&&s.jsxs("span",{className:"pending",title:"Pending approvals",children:[r.aggregate.pending_approvals," pending"]})]})})]})},Bg="_card_shkbn_1",$g="_compact_shkbn_9",Hg="_title_shkbn_13",Ug="_metricsGrid_shkbn_20",Vg="_metricItem_shkbn_26",Wg="_metricLabel_shkbn_32",Qg="_metricValue_shkbn_39",qg="_cost_shkbn_46",Kg="_averages_shkbn_50",Yg="_averagesLabel_shkbn_61",Gg="_avgItem_shkbn_65",Xg="_compactRow_shkbn_72",Jg="_compactLabel_shkbn_80",Zg="_compactValue_shkbn_84",ev="_loading_shkbn_91",tv="_error_shkbn_97",nv="_errorText_shkbn_101",rv="_pending_shkbn_107",iv="_pendingIndicator_shkbn_112",lv="_pendingDot_shkbn_120",ov="_pendingBadge_shkbn_139",q={card:Bg,compact:$g,title:Hg,metricsGrid:Ug,metricItem:Vg,metricLabel:Wg,metricValue:Qg,cost:qg,averages:Kg,averagesLabel:Yg,avgItem:Gg,compactRow:Xg,compactLabel:Jg,compactValue:Zg,loading:ev,error:tv,errorText:nv,pending:rv,pendingIndicator:iv,pendingDot:lv,pendingBadge:ov};function dc(e){return e<1e3?`${e}ms`:e<6e4?`${(e/1e3).toFixed(1)}s`:e<36e5?`${(e/6e4).toFixed(1)}m`:`${(e/36e5).toFixed(1)}h`}function On(e){return e.toLocaleString()}function wo(e){return e===0?"$0.00":e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`}function av(e){return e.pending_tasks>0&&e.total_runs===0}function Ta({scopeType:e,scopeId:t="",title:n,compact:r=!1}){const[i,l]=z.useState(null),[o,a]=z.useState(!0),[u,c]=z.useState(null),d=z.useCallback(async()=>{try{let f="/api/metrics";e!=="global"&&(f=`/api/metrics/${e}/${t}`);const h=await fetch(f);if(!h.ok)throw new Error(`Failed to fetch metrics: ${h.status}`);const k=await h.json();l(k),c(null)}catch(f){c(f instanceof Error?f.message:"Failed to load metrics")}finally{a(!1)}},[e,t]);if(z.useEffect(()=>{d();const f=setInterval(d,3e4);return()=>clearInterval(f)},[d]),o)return s.jsx("div",{className:`${q.card} ${r?q.compact:""}`,children:s.jsx("div",{className:q.loading,children:"Loading metrics..."})});if(u)return s.jsx("div",{className:`${q.card} ${r?q.compact:""} ${q.error}`,children:s.jsx("div",{className:q.errorText,children:u})});if(!i)return null;const p=n||(e==="global"?"Global Metrics":e==="agent"?`Agent: ${t}`:`Thread: ${t.slice(0,12)}...`);if(r){const f=av(i);return s.jsx("div",{className:`${q.card} ${q.compact} ${f?q.pending:""}`,children:s.jsxs("div",{className:q.compactRow,children:[f?s.jsx(s.Fragment,{children:s.jsxs("span",{className:q.pendingIndicator,children:[s.jsx("span",{className:q.pendingDot}),"Running task..."]})}):s.jsxs(s.Fragment,{children:[s.jsx("span",{className:q.compactLabel,children:"Runs:"}),s.jsx("span",{className:q.compactValue,children:On(i.total_runs)}),s.jsx("span",{className:q.compactLabel,children:"Tokens:"}),s.jsx("span",{className:q.compactValue,children:On(i.total_tokens)}),s.jsx("span",{className:q.compactLabel,children:"Cost:"}),s.jsx("span",{className:q.compactValue,children:wo(i.total_cost)})]}),i.pending_tasks>0&&!f&&s.jsxs("span",{className:q.pendingBadge,children:["+",i.pending_tasks," running"]})]})})}return s.jsxs("div",{className:q.card,children:[s.jsx("h3",{className:q.title,children:p}),s.jsxs("div",{className:q.metricsGrid,children:[s.jsxs("div",{className:q.metricItem,children:[s.jsx("span",{className:q.metricLabel,children:"Total Runs"}),s.jsx("span",{className:q.metricValue,children:On(i.total_runs)})]}),s.jsxs("div",{className:q.metricItem,children:[s.jsx("span",{className:q.metricLabel,children:"Total Tokens"}),s.jsx("span",{className:q.metricValue,children:On(i.total_tokens)})]}),s.jsxs("div",{className:q.metricItem,children:[s.jsx("span",{className:q.metricLabel,children:"Total Cost"}),s.jsx("span",{className:`${q.metricValue} ${q.cost}`,children:wo(i.total_cost)})]}),s.jsxs("div",{className:q.metricItem,children:[s.jsx("span",{className:q.metricLabel,children:"Total Duration"}),s.jsx("span",{className:q.metricValue,children:dc(i.total_duration_ms)})]}),s.jsxs("div",{className:q.metricItem,children:[s.jsx("span",{className:q.metricLabel,children:"Files Modified"}),s.jsx("span",{className:q.metricValue,children:On(i.total_files_modified)})]})]}),i.total_runs>0&&s.jsxs("div",{className:q.averages,children:[s.jsx("span",{className:q.averagesLabel,children:"Averages per run:"}),s.jsxs("span",{className:q.avgItem,children:[On(Math.round(i.avg_tokens_per_run))," tokens"]}),s.jsx("span",{className:q.avgItem,children:wo(i.avg_cost_per_run)}),s.jsx("span",{className:q.avgItem,children:dc(Math.round(i.avg_duration_per_run))})]})]})}const sv="_container_1q26w_1",uv="_title_1q26w_9",cv="_header_1q26w_16",dv="_metricLabel_1q26w_25",pv="_total_1q26w_31",fv="_chart_1q26w_37",hv="_barContainer_1q26w_46",mv="_barWrapper_1q26w_55",gv="_bar_1q26w_46",vv="_barValue_1q26w_80",xv="_label_1q26w_89",yv="_loading_1q26w_98",kv="_error_1q26w_99",wv="_empty_1q26w_100",Te={container:sv,title:uv,header:cv,metricLabel:dv,total:pv,chart:fv,barContainer:hv,barWrapper:mv,bar:gv,barValue:vv,label:xv,loading:yv,error:kv,empty:wv};function jl({scopeType:e,scopeId:t,period:n="hour",limit:r=24,metric:i="cost",title:l}){const[o,a]=z.useState([]),[u,c]=z.useState(!0),[d,p]=z.useState(null);z.useEffect(()=>{const x=async()=>{try{c(!0);const N=await fetch(`/api/metrics/trends/${e}/${t}?period=${n}&limit=${r}`);if(!N.ok)throw new Error("Failed to fetch trends");const S=await N.json();a(S||[]),p(null)}catch(N){p(N instanceof Error?N.message:"Unknown error")}finally{c(!1)}};x();const b=setInterval(x,6e4);return()=>clearInterval(b)},[e,t,n,r]);const f=x=>{switch(i){case"cost":return x.cost;case"tokens":return x.tokens;case"duration":return x.duration_ms/1e3;case"runs":return x.runs;default:return x.cost}},h=x=>{switch(i){case"cost":return`$${x.toFixed(2)}`;case"tokens":return x>=1e3?`${(x/1e3).toFixed(1)}k`:x.toString();case"duration":return`${x.toFixed(1)}s`;case"runs":return x.toString();default:return x.toFixed(2)}},k=x=>{const b=new Date(x);return n==="minute"?b.toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}):n==="hour"?b.toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}):b.toLocaleDateString([],{month:"short",day:"numeric"})},w=()=>{switch(i){case"cost":return"Cost ($)";case"tokens":return"Tokens";case"duration":return"Duration (s)";case"runs":return"Runs";default:return""}};if(u&&o.length===0)return s.jsx("div",{className:Te.container,children:s.jsx("div",{className:Te.loading,children:"Loading trends..."})});if(d)return s.jsx("div",{className:Te.container,children:s.jsx("div",{className:Te.error,children:d})});if(o.length===0)return s.jsx("div",{className:Te.container,children:s.jsx("div",{className:Te.empty,children:"No data available"})});const I=o.map(f),m=Math.max(...I,1),v=I.reduce((x,b)=>x+b,0);return s.jsxs("div",{className:Te.container,children:[l&&s.jsx("div",{className:Te.title,children:l}),s.jsxs("div",{className:Te.header,children:[s.jsx("span",{className:Te.metricLabel,children:w()}),s.jsxs("span",{className:Te.total,children:["Total: ",h(v)]})]}),s.jsx("div",{className:Te.chart,children:o.map((x,b)=>{const N=f(x),S=N/m*100;return s.jsxs("div",{className:Te.barContainer,children:[s.jsx("div",{className:Te.barWrapper,children:s.jsx("div",{className:Te.bar,style:{height:`${Math.max(S,2)}%`},title:`${k(x.period_start)}: ${h(N)}`,children:S>30&&s.jsx("span",{className:Te.barValue,children:h(N)})})}),b%Math.ceil(o.length/6)===0&&s.jsx("span",{className:Te.label,children:k(x.period_start)})]},x.period_start)})})]})}const tt=({title:e,value:t,color:n="default",small:r})=>s.jsxs("div",{className:`stat-card stat-${n} ${r?"stat-small":""}`,children:[s.jsx("div",{className:"stat-value",children:t}),s.jsx("div",{className:"stat-title",children:e})]}),Sv=e=>{if(e<1e3)return`${e}ms`;const t=e/1e3;if(t<60)return`${t.toFixed(1)}s`;const n=Math.floor(t/60),r=(t%60).toFixed(0);return`${n}m ${r}s`},bv=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,Ri=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}k`:e.toString(),_v=({agent:e,onClick:t})=>{var o,a,u,c,d;const n=((o=e.children)==null?void 0:o.length)||0,r=((u=(a=e.badges)==null?void 0:a.find(p=>p.type==="pending"))==null?void 0:u.count)||0,i=((d=(c=e.badges)==null?void 0:c.find(p=>p.type==="running"))==null?void 0:d.count)||0,l={active:"#22c55e",running:"#22c55e",pending:"#f59e0b",idle:"#6b7280",error:"#ef4444"};return s.jsxs("div",{className:"agent-card",onClick:t,children:[s.jsxs("div",{className:"agent-card-header",children:[s.jsx("span",{className:"agent-status-dot",style:{backgroundColor:l[e.status||"idle"]}}),s.jsx("span",{className:"agent-name",children:xf(e.id)})]}),s.jsxs("div",{className:"agent-card-stats",children:[s.jsxs("span",{className:"agent-stat",children:[s.jsx("span",{className:"agent-stat-value",children:n}),s.jsx("span",{className:"agent-stat-label",children:"threads"})]}),r>0&&s.jsxs("span",{className:"agent-stat pending",children:[s.jsx("span",{className:"agent-stat-value",children:r}),s.jsx("span",{className:"agent-stat-label",children:"pending"})]}),i>0&&s.jsxs("span",{className:"agent-stat running",children:[s.jsx("span",{className:"agent-stat-value",children:i}),s.jsx("span",{className:"agent-stat-label",children:"running"})]})]})]})},jv=({aggregate:e,agents:t,onSelectAgent:n})=>{const r=e.execution,i=r&&r.total_executions>0,l=i?Math.round(r.successful_executions/r.total_executions*100):0;return s.jsxs("div",{className:"all-agents-overview",children:[s.jsx("div",{className:"overview-header",children:s.jsx("h2",{children:"All Agents Overview"})}),s.jsxs("div",{className:"stats-row",children:[s.jsx(tt,{title:"Total Agents",value:e.total_agents}),s.jsx(tt,{title:"Active",value:e.active_agents,color:"green"}),s.jsx(tt,{title:"Pending Approvals",value:e.pending_approvals,color:"orange"}),s.jsx(tt,{title:"Total Threads",value:e.total_threads,color:"blue"})]}),s.jsxs("div",{className:"metrics-section",children:[s.jsx("h3",{children:"Usage Metrics (Today)"}),s.jsx(Ta,{scopeType:"global",title:"Global Metrics"})]}),s.jsxs("div",{className:"trends-section",children:[s.jsx("h3",{children:"Usage Trends (Last 24 Hours)"}),s.jsxs("div",{className:"trends-grid",children:[s.jsx(jl,{scopeType:"global",scopeId:"",period:"hour",limit:24,metric:"cost",title:"Cost"}),s.jsx(jl,{scopeType:"global",scopeId:"",period:"hour",limit:24,metric:"tokens",title:"Tokens"})]})]}),i&&s.jsxs("div",{className:"execution-stats-section",children:[s.jsx("h3",{children:"Execution Statistics"}),s.jsxs("div",{className:"stats-row",children:[s.jsx(tt,{title:"Total Executions",value:r.total_executions,color:"purple"}),s.jsx(tt,{title:"Success Rate",value:`${l}%`,color:"green"}),s.jsx(tt,{title:"Total Duration",value:Sv(r.total_duration_ms)}),s.jsx(tt,{title:"Total Cost",value:bv(r.total_cost),color:"orange"})]}),s.jsxs("div",{className:"stats-row token-stats",children:[s.jsx(tt,{title:"Input Tokens",value:Ri(r.total_input_tokens),small:!0}),s.jsx(tt,{title:"Output Tokens",value:Ri(r.total_output_tokens),small:!0}),s.jsx(tt,{title:"Cache Read",value:Ri(r.total_cache_read_tokens),small:!0}),s.jsx(tt,{title:"Cache Created",value:Ri(r.total_cache_create_tokens),small:!0}),s.jsx(tt,{title:"Files Created",value:r.total_files_created,small:!0})]})]}),s.jsxs("div",{className:"agents-section",children:[s.jsx("h3",{children:"Agents"}),s.jsxs("div",{className:"agent-cards-grid",children:[t.map(o=>s.jsx(_v,{agent:o,onClick:()=>n(o.id)},o.id)),t.length===0&&s.jsx("div",{className:"no-agents",children:"No agents found. Start an agent to see it here."})]})]})]})},Cv=({items:e})=>s.jsx("nav",{className:"breadcrumb",children:e.map((t,n)=>s.jsxs(Jt.Fragment,{children:[n>0&&s.jsx("span",{className:"breadcrumb-separator",children:"/"}),t.onClick?s.jsx("button",{className:"breadcrumb-link",onClick:t.onClick,children:t.label}):s.jsx("span",{className:"breadcrumb-current",children:t.label})]},n))}),Nv="_badge_11ih7_1",Ev="_small_11ih7_10",Tv="_medium_11ih7_15",Lv="_ailang_11ih7_21",Pv="_claude_11ih7_26",Iv="_stapledon_11ih7_31",zv="_coordinator_11ih7_36",Av="_unknown_11ih7_41",So={badge:Nv,small:Ev,medium:Tv,ailang:Lv,claude:Pv,stapledon:Iv,coordinator:zv,unknown:Av,default:"_default_11ih7_46"},Rv={ailang:{name:"AILANG",colorClass:"ailang"},"claude-code":{name:"Claude",colorClass:"claude"},stapledons_voyage:{name:"Stapledon",colorClass:"stapledon"},coordinator:{name:"Coordinator",colorClass:"coordinator"},unknown:{name:"Unknown",colorClass:"unknown"}};function Mv({workspace:e,size:t="small"}){if(!e)return null;const n=e.toLowerCase(),r=Rv[n]||{name:e.length>12?e.slice(0,12)+"...":e,colorClass:"default"};return s.jsx("span",{className:`${So.badge} ${So[r.colorClass]} ${So[t]}`,title:`Workspace: ${e}`,children:r.name})}const Mt={plus:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),s.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),s.jsx("circle",{cx:"12",cy:"5",r:"2"}),s.jsx("path",{d:"M12 7v4"}),s.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),s.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),s.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),s.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),s.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:s.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),s.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:s.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("polyline",{points:"3 6 5 6 21 6"}),s.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:s.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:s.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),s.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},Dv=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,u]=z.useState(!1),[c,d]=z.useState(""),[p,f]=z.useState(null),[h,k]=z.useState(""),[w,I]=z.useState(null),m=()=>{c.trim()&&(r(c.trim()),d(""),u(!1))},v=j=>{j.key==="Enter"&&!j.shiftKey?(j.preventDefault(),m()):j.key==="Escape"&&(u(!1),d(""))},x=(j,E)=>{E.stopPropagation(),f(j.id),k(j.title)},b=j=>{var E;h.trim()&&h.trim()!==((E=e.find(U=>U.id===j))==null?void 0:E.title)&&l(j,h.trim()),f(null),k("")},N=()=>{f(null),k("")},S=(j,E)=>{j.key==="Enter"?(j.preventDefault(),b(E)):j.key==="Escape"&&N()},C=(j,E)=>{E.stopPropagation(),I(j)},P=(j,E)=>{E.stopPropagation(),i(j),I(null)},D=j=>{j.stopPropagation(),I(null)},A=j=>{const E=new Date(j),V=new Date().getTime()-E.getTime(),W=Math.floor(V/6e4),G=Math.floor(V/36e5),oe=Math.floor(V/864e5);return W<1?"now":W<60?`${W}m`:G<24?`${G}h`:oe<7?`${oe}d`:E.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return s.jsxs("div",{className:"thread-list",children:[s.jsxs("div",{className:"list-header",children:[s.jsx("h2",{children:"Conversations"}),s.jsx("button",{className:"new-thread-btn",onClick:()=>u(!0),title:"New conversation",children:Mt.plus})]}),a&&s.jsxs("div",{className:"new-thread-form",children:[s.jsx("input",{type:"text",value:c,onChange:j=>d(j.target.value),onKeyDown:v,placeholder:"Conversation title...",autoFocus:!0}),s.jsxs("div",{className:"form-actions",children:[s.jsx("button",{className:"cancel-btn",onClick:()=>u(!1),children:"Cancel"}),s.jsx("button",{className:"create-btn",onClick:m,children:"Create"})]})]}),s.jsx("div",{className:"thread-items",children:e.length===0?s.jsxs("div",{className:"empty-state",children:[s.jsx("div",{className:"empty-icon",children:Mt.hash}),s.jsx("p",{children:"No conversations yet"}),s.jsx("button",{className:"start-btn",onClick:()=>u(!0),children:"Start a conversation"})]}):e.map(j=>{const E=o.get(j.id)||0,U=j.id===t,V=p===j.id,W=w===j.id;return s.jsxs("div",{className:`thread-item ${U?"selected":""} ${E>0?"has-unread":""}`,onClick:()=>!V&&n(j.id),children:[s.jsx("div",{className:`status-dot ${j.status}`}),s.jsxs("div",{className:"thread-content",children:[s.jsx("div",{className:"thread-title-row",children:V?s.jsxs("div",{className:"edit-title-form",onClick:G=>G.stopPropagation(),children:[s.jsx("input",{type:"text",value:h,onChange:G=>k(G.target.value),onKeyDown:G=>S(G,j.id),autoFocus:!0}),s.jsx("button",{className:"edit-action save",onClick:()=>b(j.id),title:"Save",children:Mt.check}),s.jsx("button",{className:"edit-action cancel",onClick:N,title:"Cancel",children:Mt.x})]}):s.jsxs(s.Fragment,{children:[s.jsx("span",{className:"thread-title",children:j.title}),s.jsx("span",{className:"thread-time",children:A(j.updated_at)})]})}),s.jsxs("div",{className:"thread-meta",children:[j.target_agent&&s.jsxs("span",{className:"thread-agent",title:`Target: ${j.target_agent}`,children:[Mt.bot,j.target_agent]}),j.workspace&&s.jsx(Mv,{workspace:j.workspace,size:"small"}),s.jsxs("span",{className:"thread-seq",children:["#",j.last_seq]})]})]}),!V&&!W&&s.jsxs("div",{className:"thread-actions",children:[s.jsx("button",{className:"action-btn edit",onClick:G=>x(j,G),title:"Rename",children:Mt.edit}),s.jsx("button",{className:"action-btn delete",onClick:G=>C(j.id,G),title:"Delete",children:Mt.trash})]}),W&&s.jsxs("div",{className:"delete-confirm",onClick:G=>G.stopPropagation(),children:[s.jsx("span",{className:"confirm-text",children:"Delete?"}),s.jsx("button",{className:"confirm-btn yes",onClick:G=>P(j.id,G),title:"Confirm delete",children:Mt.check}),s.jsx("button",{className:"confirm-btn no",onClick:D,title:"Cancel",children:Mt.x})]}),E>0&&!W&&s.jsx("span",{className:"unread-badge",children:E})]},j.id)})}),s.jsx("style",{children:`
        .thread-list {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg-surface);
        }

        /* Header */
        .list-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-4);
          border-bottom: 1px solid var(--border-subtle);
        }

        .list-header h2 {
          font-size: var(--text-sm);
          font-weight: var(--font-semibold);
          color: var(--text-secondary);
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .new-thread-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 28px;
          height: 28px;
          background: var(--bg-elevated);
          color: var(--text-secondary);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .new-thread-btn:hover {
          background: var(--color-primary);
          color: var(--text-inverse);
          border-color: var(--color-primary);
        }

        /* New Thread Form */
        .new-thread-form {
          padding: var(--space-3);
          background: var(--bg-elevated);
          border-bottom: 1px solid var(--border-subtle);
        }

        .new-thread-form input {
          width: 100%;
          padding: var(--space-2) var(--space-3);
          background: var(--bg-base);
          color: var(--text-primary);
          font-size: var(--text-sm);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          margin-bottom: var(--space-2);
        }

        .new-thread-form input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.1);
        }

        .form-actions {
          display: flex;
          justify-content: flex-end;
          gap: var(--space-2);
        }

        .cancel-btn, .create-btn {
          padding: var(--space-1) var(--space-3);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .cancel-btn {
          background: transparent;
          color: var(--text-secondary);
          border: 1px solid var(--border-default);
        }

        .cancel-btn:hover {
          background: var(--bg-hover);
        }

        .create-btn {
          background: var(--color-primary);
          color: var(--text-inverse);
          border: none;
        }

        .create-btn:hover {
          background: var(--color-primary-light);
        }

        /* Thread Items */
        .thread-items {
          flex: 1;
          overflow-y: auto;
        }

        .empty-state {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          padding: var(--space-8);
          text-align: center;
        }

        .empty-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 48px;
          height: 48px;
          background: var(--bg-elevated);
          border-radius: var(--radius-lg);
          color: var(--text-tertiary);
          margin-bottom: var(--space-3);
        }

        .empty-state p {
          font-size: var(--text-sm);
          color: var(--text-tertiary);
          margin-bottom: var(--space-4);
        }

        .start-btn {
          padding: var(--space-2) var(--space-4);
          background: var(--color-primary);
          color: var(--text-inverse);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .start-btn:hover {
          background: var(--color-primary-light);
          transform: translateY(-1px);
        }

        /* Thread Item */
        .thread-item {
          display: flex;
          align-items: flex-start;
          gap: var(--space-3);
          padding: var(--space-3) var(--space-4);
          cursor: pointer;
          transition: all var(--transition-fast);
          border-left: 2px solid transparent;
        }

        .thread-item:hover {
          background: var(--bg-hover);
        }

        .thread-item.selected {
          background: var(--bg-active);
          border-left-color: var(--color-primary);
        }

        .thread-item.has-unread .thread-title {
          font-weight: var(--font-semibold);
          color: var(--text-primary);
        }

        /* Status Dot */
        .status-dot {
          width: 8px;
          height: 8px;
          border-radius: var(--radius-full);
          flex-shrink: 0;
          margin-top: 6px;
        }

        .status-dot.active {
          background: var(--color-success);
          box-shadow: 0 0 6px var(--color-success);
        }

        .status-dot.paused {
          background: var(--color-warning);
        }

        .status-dot.resolved {
          background: var(--color-primary);
        }

        .status-dot.archived {
          background: var(--text-tertiary);
        }

        /* Thread Content */
        .thread-content {
          flex: 1;
          min-width: 0;
        }

        .thread-title-row {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: var(--space-2);
          margin-bottom: var(--space-1);
        }

        .thread-title {
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          color: var(--text-primary);
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .thread-time {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          flex-shrink: 0;
        }

        .thread-meta {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .thread-creator {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .thread-creator svg {
          opacity: 0.7;
        }

        .thread-agent {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          color: var(--color-primary);
          padding: 2px 6px;
          background: rgba(37, 194, 160, 0.1);
          border-radius: var(--radius-sm);
          max-width: 120px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .thread-agent svg {
          flex-shrink: 0;
          opacity: 0.8;
        }

        .thread-seq {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
        }

        /* Unread Badge */
        .unread-badge {
          display: flex;
          align-items: center;
          justify-content: center;
          min-width: 18px;
          height: 18px;
          padding: 0 var(--space-1);
          background: var(--color-primary);
          color: var(--text-inverse);
          font-size: 11px;
          font-weight: var(--font-bold);
          border-radius: var(--radius-full);
          flex-shrink: 0;
        }

        /* Thread Actions */
        .thread-actions {
          display: none;
          align-items: center;
          gap: var(--space-1);
          flex-shrink: 0;
        }

        .thread-item:hover .thread-actions {
          display: flex;
        }

        .action-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 24px;
          height: 24px;
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .action-btn:hover {
          color: var(--text-primary);
          border-color: var(--border-default);
        }

        .action-btn.edit:hover {
          color: var(--color-primary);
          border-color: var(--color-primary);
        }

        .action-btn.delete:hover {
          color: var(--color-error);
          border-color: var(--color-error);
        }

        /* Edit Title Form */
        .edit-title-form {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          flex: 1;
        }

        .edit-title-form input {
          flex: 1;
          padding: var(--space-1) var(--space-2);
          background: var(--bg-base);
          color: var(--text-primary);
          font-size: var(--text-sm);
          border: 1px solid var(--color-primary);
          border-radius: var(--radius-sm);
          outline: none;
        }

        .edit-action {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 22px;
          height: 22px;
          background: transparent;
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .edit-action.save {
          color: var(--color-success);
        }

        .edit-action.save:hover {
          background: rgba(34, 197, 94, 0.1);
        }

        .edit-action.cancel {
          color: var(--text-tertiary);
        }

        .edit-action.cancel:hover {
          color: var(--text-secondary);
          background: var(--bg-hover);
        }

        /* Delete Confirmation */
        .delete-confirm {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          padding: var(--space-1) var(--space-2);
          background: rgba(239, 68, 68, 0.1);
          border-radius: var(--radius-sm);
        }

        .confirm-text {
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          color: var(--color-error);
        }

        .confirm-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 22px;
          height: 22px;
          background: transparent;
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .confirm-btn.yes {
          color: var(--color-error);
        }

        .confirm-btn.yes:hover {
          background: var(--color-error);
          color: white;
        }

        .confirm-btn.no {
          color: var(--text-tertiary);
        }

        .confirm-btn.no:hover {
          color: var(--text-secondary);
          background: var(--bg-hover);
        }
      `})]})};function Fv(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const Ov=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,Bv=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,$v={};function pc(e,t){return($v.jsx?Bv:Ov).test(e)}const Hv=/[ \t\n\f\r]/g;function Uv(e){return typeof e=="object"?e.type==="text"?fc(e.value):!1:fc(e)}function fc(e){return e.replace(Hv,"")===""}class gi{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}gi.prototype.normal={};gi.prototype.property={};gi.prototype.space=void 0;function yf(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new gi(n,r,t)}function La(e){return e.toLowerCase()}class Ze{constructor(t,n){this.attribute=n,this.property=t}}Ze.prototype.attribute="";Ze.prototype.booleanish=!1;Ze.prototype.boolean=!1;Ze.prototype.commaOrSpaceSeparated=!1;Ze.prototype.commaSeparated=!1;Ze.prototype.defined=!1;Ze.prototype.mustUseProperty=!1;Ze.prototype.number=!1;Ze.prototype.overloadedBoolean=!1;Ze.prototype.property="";Ze.prototype.spaceSeparated=!1;Ze.prototype.space=void 0;let Vv=0;const X=Mn(),we=Mn(),Pa=Mn(),F=Mn(),se=Mn(),lr=Mn(),nt=Mn();function Mn(){return 2**++Vv}const Ia=Object.freeze(Object.defineProperty({__proto__:null,boolean:X,booleanish:we,commaOrSpaceSeparated:nt,commaSeparated:lr,number:F,overloadedBoolean:Pa,spaceSeparated:se},Symbol.toStringTag,{value:"Module"})),bo=Object.keys(Ia);class Ms extends Ze{constructor(t,n,r,i){let l=-1;if(super(t,n),hc(this,"space",i),typeof r=="number")for(;++l<bo.length;){const o=bo[l];hc(this,bo[l],(r&Ia[o])===Ia[o])}}}Ms.prototype.defined=!0;function hc(e,t,n){n&&(e[t]=n)}function vr(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new Ms(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[La(r)]=r,n[La(l.attribute)]=r}return new gi(t,n,e.space)}const kf=vr({properties:{ariaActiveDescendant:null,ariaAtomic:we,ariaAutoComplete:null,ariaBusy:we,ariaChecked:we,ariaColCount:F,ariaColIndex:F,ariaColSpan:F,ariaControls:se,ariaCurrent:null,ariaDescribedBy:se,ariaDetails:null,ariaDisabled:we,ariaDropEffect:se,ariaErrorMessage:null,ariaExpanded:we,ariaFlowTo:se,ariaGrabbed:we,ariaHasPopup:null,ariaHidden:we,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:se,ariaLevel:F,ariaLive:null,ariaModal:we,ariaMultiLine:we,ariaMultiSelectable:we,ariaOrientation:null,ariaOwns:se,ariaPlaceholder:null,ariaPosInSet:F,ariaPressed:we,ariaReadOnly:we,ariaRelevant:null,ariaRequired:we,ariaRoleDescription:se,ariaRowCount:F,ariaRowIndex:F,ariaRowSpan:F,ariaSelected:we,ariaSetSize:F,ariaSort:null,ariaValueMax:F,ariaValueMin:F,ariaValueNow:F,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function wf(e,t){return t in e?e[t]:t}function Sf(e,t){return wf(e,t.toLowerCase())}const Wv=vr({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:lr,acceptCharset:se,accessKey:se,action:null,allow:null,allowFullScreen:X,allowPaymentRequest:X,allowUserMedia:X,alt:null,as:null,async:X,autoCapitalize:null,autoComplete:se,autoFocus:X,autoPlay:X,blocking:se,capture:null,charSet:null,checked:X,cite:null,className:se,cols:F,colSpan:null,content:null,contentEditable:we,controls:X,controlsList:se,coords:F|lr,crossOrigin:null,data:null,dateTime:null,decoding:null,default:X,defer:X,dir:null,dirName:null,disabled:X,download:Pa,draggable:we,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:X,formTarget:null,headers:se,height:F,hidden:Pa,high:F,href:null,hrefLang:null,htmlFor:se,httpEquiv:se,id:null,imageSizes:null,imageSrcSet:null,inert:X,inputMode:null,integrity:null,is:null,isMap:X,itemId:null,itemProp:se,itemRef:se,itemScope:X,itemType:se,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:X,low:F,manifest:null,max:null,maxLength:F,media:null,method:null,min:null,minLength:F,multiple:X,muted:X,name:null,nonce:null,noModule:X,noValidate:X,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:X,optimum:F,pattern:null,ping:se,placeholder:null,playsInline:X,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:X,referrerPolicy:null,rel:se,required:X,reversed:X,rows:F,rowSpan:F,sandbox:se,scope:null,scoped:X,seamless:X,selected:X,shadowRootClonable:X,shadowRootDelegatesFocus:X,shadowRootMode:null,shape:null,size:F,sizes:null,slot:null,span:F,spellCheck:we,src:null,srcDoc:null,srcLang:null,srcSet:null,start:F,step:null,style:null,tabIndex:F,target:null,title:null,translate:null,type:null,typeMustMatch:X,useMap:null,value:we,width:F,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:se,axis:null,background:null,bgColor:null,border:F,borderColor:null,bottomMargin:F,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:X,declare:X,event:null,face:null,frame:null,frameBorder:null,hSpace:F,leftMargin:F,link:null,longDesc:null,lowSrc:null,marginHeight:F,marginWidth:F,noResize:X,noHref:X,noShade:X,noWrap:X,object:null,profile:null,prompt:null,rev:null,rightMargin:F,rules:null,scheme:null,scrolling:we,standby:null,summary:null,text:null,topMargin:F,valueType:null,version:null,vAlign:null,vLink:null,vSpace:F,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:X,disableRemotePlayback:X,prefix:null,property:null,results:F,security:null,unselectable:null},space:"html",transform:Sf}),Qv=vr({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:nt,accentHeight:F,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:F,amplitude:F,arabicForm:null,ascent:F,attributeName:null,attributeType:null,azimuth:F,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:F,by:null,calcMode:null,capHeight:F,className:se,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:F,diffuseConstant:F,direction:null,display:null,dur:null,divisor:F,dominantBaseline:null,download:X,dx:null,dy:null,edgeMode:null,editable:null,elevation:F,enableBackground:null,end:null,event:null,exponent:F,externalResourcesRequired:null,fill:null,fillOpacity:F,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:lr,g2:lr,glyphName:lr,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:F,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:F,horizOriginX:F,horizOriginY:F,id:null,ideographic:F,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:F,k:F,k1:F,k2:F,k3:F,k4:F,kernelMatrix:nt,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:F,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:F,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:F,overlineThickness:F,paintOrder:null,panose1:null,path:null,pathLength:F,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:se,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:F,pointsAtY:F,pointsAtZ:F,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:nt,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:nt,rev:nt,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:nt,requiredFeatures:nt,requiredFonts:nt,requiredFormats:nt,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:F,specularExponent:F,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:F,strikethroughThickness:F,string:null,stroke:null,strokeDashArray:nt,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:F,strokeOpacity:F,strokeWidth:null,style:null,surfaceScale:F,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:nt,tabIndex:F,tableValues:null,target:null,targetX:F,targetY:F,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:nt,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:F,underlineThickness:F,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:F,values:null,vAlphabetic:F,vMathematical:F,vectorEffect:null,vHanging:F,vIdeographic:F,version:null,vertAdvY:F,vertOriginX:F,vertOriginY:F,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:F,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:wf}),bf=vr({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),_f=vr({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:Sf}),jf=vr({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),qv={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Kv=/[A-Z]/g,mc=/-[a-z]/g,Yv=/^data[-\w.:]+$/i;function Gv(e,t){const n=La(t);let r=t,i=Ze;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&Yv.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(mc,Jv);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!mc.test(l)){let o=l.replace(Kv,Xv);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=Ms}return new i(r,t)}function Xv(e){return"-"+e.toLowerCase()}function Jv(e){return e.charAt(1).toUpperCase()}const Zv=yf([kf,Wv,bf,_f,jf],"html"),Ds=yf([kf,Qv,bf,_f,jf],"svg");function ex(e){return e.join(" ").trim()}var Fs={},gc=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,tx=/\n/g,nx=/^\s*/,rx=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,ix=/^:\s*/,lx=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,ox=/^[;\s]*/,ax=/^\s+|\s+$/g,sx=`
`,vc="/",xc="*",bn="",ux="comment",cx="declaration";function dx(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(k){var w=k.match(tx);w&&(n+=w.length);var I=k.lastIndexOf(sx);r=~I?k.length-I:r+k.length}function l(){var k={line:n,column:r};return function(w){return w.position=new o(k),c(),w}}function o(k){this.start=k,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(k){var w=new Error(t.source+":"+n+":"+r+": "+k);if(w.reason=k,w.filename=t.source,w.line=n,w.column=r,w.source=e,!t.silent)throw w}function u(k){var w=k.exec(e);if(w){var I=w[0];return i(I),e=e.slice(I.length),w}}function c(){u(nx)}function d(k){var w;for(k=k||[];w=p();)w!==!1&&k.push(w);return k}function p(){var k=l();if(!(vc!=e.charAt(0)||xc!=e.charAt(1))){for(var w=2;bn!=e.charAt(w)&&(xc!=e.charAt(w)||vc!=e.charAt(w+1));)++w;if(w+=2,bn===e.charAt(w-1))return a("End of comment missing");var I=e.slice(2,w-2);return r+=2,i(I),e=e.slice(w),r+=2,k({type:ux,comment:I})}}function f(){var k=l(),w=u(rx);if(w){if(p(),!u(ix))return a("property missing ':'");var I=u(lx),m=k({type:cx,property:yc(w[0].replace(gc,bn)),value:I?yc(I[0].replace(gc,bn)):bn});return u(ox),m}}function h(){var k=[];d(k);for(var w;w=f();)w!==!1&&(k.push(w),d(k));return k}return c(),h()}function yc(e){return e?e.replace(ax,bn):bn}var px=dx,fx=el&&el.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(Fs,"__esModule",{value:!0});Fs.default=mx;const hx=fx(px);function mx(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,hx.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Vl={};Object.defineProperty(Vl,"__esModule",{value:!0});Vl.camelCase=void 0;var gx=/^--[a-zA-Z0-9_-]+$/,vx=/-([a-z])/g,xx=/^[^-]+$/,yx=/^-(webkit|moz|ms|o|khtml)-/,kx=/^-(ms)-/,wx=function(e){return!e||xx.test(e)||gx.test(e)},Sx=function(e,t){return t.toUpperCase()},kc=function(e,t){return"".concat(t,"-")},bx=function(e,t){return t===void 0&&(t={}),wx(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(kx,kc):e=e.replace(yx,kc),e.replace(vx,Sx))};Vl.camelCase=bx;var _x=el&&el.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},jx=_x(Fs),Cx=Vl;function za(e,t){var n={};return!e||typeof e!="string"||(0,jx.default)(e,function(r,i){r&&i&&(n[(0,Cx.camelCase)(r,t)]=i)}),n}za.default=za;var Nx=za;const Ex=$a(Nx),Cf=Nf("end"),Os=Nf("start");function Nf(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function Tx(e){const t=Os(e),n=Cf(e);if(t&&n)return{start:t,end:n}}function Vr(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?wc(e.position):"start"in e||"end"in e?wc(e):"line"in e||"column"in e?Aa(e):""}function Aa(e){return Sc(e&&e.line)+":"+Sc(e&&e.column)}function wc(e){return Aa(e&&e.start)+"-"+Aa(e&&e.end)}function Sc(e){return e&&typeof e=="number"?e:1}class Fe extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const u=r.indexOf(":");u===-1?l.ruleId=r:(l.source=r.slice(0,u),l.ruleId=r.slice(u+1))}if(!l.place&&l.ancestors&&l.ancestors){const u=l.ancestors[l.ancestors.length-1];u&&(l.place=u.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Vr(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}Fe.prototype.file="";Fe.prototype.name="";Fe.prototype.reason="";Fe.prototype.message="";Fe.prototype.stack="";Fe.prototype.column=void 0;Fe.prototype.line=void 0;Fe.prototype.ancestors=void 0;Fe.prototype.cause=void 0;Fe.prototype.fatal=void 0;Fe.prototype.place=void 0;Fe.prototype.ruleId=void 0;Fe.prototype.source=void 0;const Bs={}.hasOwnProperty,Lx=new Map,Px=/[A-Z]/g,Ix=new Set(["table","tbody","thead","tfoot","tr"]),zx=new Set(["td","th"]),Ef="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function Ax(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=Hx(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=$x(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?Ds:Zv,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=Tf(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function Tf(e,t,n){if(t.type==="element")return Rx(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return Mx(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return Fx(e,t,n);if(t.type==="mdxjsEsm")return Dx(e,t);if(t.type==="root")return Ox(e,t,n);if(t.type==="text")return Bx(e,t)}function Rx(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=Ds,e.schema=i),e.ancestors.push(t);const l=Pf(e,t.tagName,!1),o=Ux(e,t);let a=Hs(e,t);return Ix.has(t.tagName)&&(a=a.filter(function(u){return typeof u=="string"?!Uv(u):!0})),Lf(e,o,l,t),$s(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function Mx(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}ci(e,t.position)}function Dx(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);ci(e,t.position)}function Fx(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=Ds,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:Pf(e,t.name,!0),o=Vx(e,t),a=Hs(e,t);return Lf(e,o,l,t),$s(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function Ox(e,t,n){const r={};return $s(r,Hs(e,t)),e.create(t,e.Fragment,r,n)}function Bx(e,t){return t.value}function Lf(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function $s(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function $x(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function Hx(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),u=Os(r);return t(i,l,o,a,{columnNumber:u?u.column-1:void 0,fileName:e,lineNumber:u?u.line:void 0},void 0)}}function Ux(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Bs.call(t.properties,i)){const l=Wx(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&zx.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function Vx(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else ci(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else ci(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Hs(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:Lx;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const u=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(u){const c=i.get(u)||0;o=u+"-"+c,i.set(u,c+1)}}const a=Tf(e,l,o);a!==void 0&&n.push(a)}return n}function Wx(e,t,n){const r=Gv(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?Fv(n):ex(n)),r.property==="style"){let i=typeof n=="object"?n:Qx(e,String(n));return e.stylePropertyNameCase==="css"&&(i=qx(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?qv[r.property]||r.property:r.attribute,n]}}function Qx(e,t){try{return Ex(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new Fe("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=Ef+"#cannot-parse-style-attribute",i}}function Pf(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=pc(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=pc(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Bs.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);ci(e)}function ci(e,t){const n=new Fe("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=Ef+"#cannot-handle-mdx-estrees-without-createevaluater",n}function qx(e){const t={};let n;for(n in e)Bs.call(e,n)&&(t[Kx(n)]=e[n]);return t}function Kx(e){let t=e.replace(Px,Yx);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function Yx(e){return"-"+e.toLowerCase()}const _o={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},Gx={};function Xx(e,t){const n=Gx,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return If(e,r,i)}function If(e,t,n){if(Jx(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return bc(e.children,t,n)}return Array.isArray(e)?bc(e,t,n):""}function bc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=If(e[i],t,n);return r.join("")}function Jx(e){return!!(e&&typeof e=="object")}const _c=document.createElement("i");function Us(e){const t="&"+e+";";_c.innerHTML=t;const n=_c.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function zt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function ft(e,t){return e.length>0?(zt(e,e.length,0,t),e):t}const jc={}.hasOwnProperty;function Zx(e){const t={};let n=-1;for(;++n<e.length;)ey(t,e[n]);return t}function ey(e,t){let n;for(n in t){const i=(jc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){jc.call(i,o)||(i[o]=[]);const a=l[o];ty(i[o],Array.isArray(a)?a:a?[a]:[])}}}function ty(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);zt(e,0,0,r)}function zf(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function or(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const Lt=yn(/[A-Za-z]/),lt=yn(/[\dA-Za-z]/),ny=yn(/[#-'*+\--9=?A-Z^-~]/);function Ra(e){return e!==null&&(e<32||e===127)}const Ma=yn(/\d/),ry=yn(/[\dA-Fa-f]/),iy=yn(/[!-/:-@[-`{-~]/);function Q(e){return e!==null&&e<-2}function Je(e){return e!==null&&(e<0||e===32)}function ie(e){return e===-2||e===-1||e===32}const ly=yn(new RegExp("\\p{P}|\\p{S}","u")),oy=yn(/\s/);function yn(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function xr(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&lt(e.charCodeAt(n+1))&&lt(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function ce(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(u){return ie(u)?(e.enter(n),a(u)):t(u)}function a(u){return ie(u)&&l++<i?(e.consume(u),a):(e.exit(n),t(u))}}const ay={tokenize:sy};function sy(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),ce(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const u=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=u),n=u,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return Q(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const uy={tokenize:cy},Cc={tokenize:dy};function cy(e){const t=this,n=[];let r=0,i,l,o;return a;function a(x){if(r<n.length){const b=n[r];return t.containerState=b[1],e.attempt(b[0].continuation,u,c)(x)}return c(x)}function u(x){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&v();const b=t.events.length;let N=b,S;for(;N--;)if(t.events[N][0]==="exit"&&t.events[N][1].type==="chunkFlow"){S=t.events[N][1].end;break}m(r);let C=b;for(;C<t.events.length;)t.events[C][1].end={...S},C++;return zt(t.events,N+1,0,t.events.slice(b)),t.events.length=C,c(x)}return a(x)}function c(x){if(r===n.length){if(!i)return f(x);if(i.currentConstruct&&i.currentConstruct.concrete)return k(x);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(Cc,d,p)(x)}function d(x){return i&&v(),m(r),f(x)}function p(x){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,k(x)}function f(x){return t.containerState={},e.attempt(Cc,h,k)(x)}function h(x){return r++,n.push([t.currentConstruct,t.containerState]),f(x)}function k(x){if(x===null){i&&v(),m(0),e.consume(x);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),w(x)}function w(x){if(x===null){I(e.exit("chunkFlow"),!0),m(0),e.consume(x);return}return Q(x)?(e.consume(x),I(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(x),w)}function I(x,b){const N=t.sliceStream(x);if(b&&N.push(null),x.previous=l,l&&(l.next=x),l=x,i.defineSkip(x.start),i.write(N),t.parser.lazy[x.start.line]){let S=i.events.length;for(;S--;)if(i.events[S][1].start.offset<o&&(!i.events[S][1].end||i.events[S][1].end.offset>o))return;const C=t.events.length;let P=C,D,A;for(;P--;)if(t.events[P][0]==="exit"&&t.events[P][1].type==="chunkFlow"){if(D){A=t.events[P][1].end;break}D=!0}for(m(r),S=C;S<t.events.length;)t.events[S][1].end={...A},S++;zt(t.events,P+1,0,t.events.slice(C)),t.events.length=S}}function m(x){let b=n.length;for(;b-- >x;){const N=n[b];t.containerState=N[1],N[0].exit.call(t,e)}n.length=x}function v(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function dy(e,t,n){return ce(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function Nc(e){if(e===null||Je(e)||oy(e))return 1;if(ly(e))return 2}function Vs(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const Da={name:"attention",resolveAll:py,tokenize:fy};function py(e,t){let n=-1,r,i,l,o,a,u,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;u=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const p={...e[r][1].end},f={...e[n][1].start};Ec(p,-u),Ec(f,u),o={type:u>1?"strongSequence":"emphasisSequence",start:p,end:{...e[r][1].end}},a={type:u>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:f},l={type:u>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:u>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=ft(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=ft(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=ft(c,Vs(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=ft(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=ft(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,zt(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function fy(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=Nc(r);let l;return o;function o(u){return l=u,e.enter("attentionSequence"),a(u)}function a(u){if(u===l)return e.consume(u),a;const c=e.exit("attentionSequence"),d=Nc(u),p=!d||d===2&&i||n.includes(u),f=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?p:p&&(i||!f)),c._close=!!(l===42?f:f&&(d||!p)),t(u)}}function Ec(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const hy={name:"autolink",tokenize:my};function my(e,t,n){let r=0;return i;function i(h){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(h),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(h){return Lt(h)?(e.consume(h),o):h===64?n(h):c(h)}function o(h){return h===43||h===45||h===46||lt(h)?(r=1,a(h)):c(h)}function a(h){return h===58?(e.consume(h),r=0,u):(h===43||h===45||h===46||lt(h))&&r++<32?(e.consume(h),a):(r=0,c(h))}function u(h){return h===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(h),e.exit("autolinkMarker"),e.exit("autolink"),t):h===null||h===32||h===60||Ra(h)?n(h):(e.consume(h),u)}function c(h){return h===64?(e.consume(h),d):ny(h)?(e.consume(h),c):n(h)}function d(h){return lt(h)?p(h):n(h)}function p(h){return h===46?(e.consume(h),r=0,d):h===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(h),e.exit("autolinkMarker"),e.exit("autolink"),t):f(h)}function f(h){if((h===45||lt(h))&&r++<63){const k=h===45?f:p;return e.consume(h),k}return n(h)}}const Wl={partial:!0,tokenize:gy};function gy(e,t,n){return r;function r(l){return ie(l)?ce(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||Q(l)?t(l):n(l)}}const Af={continuation:{tokenize:xy},exit:yy,name:"blockQuote",tokenize:vy};function vy(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return ie(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function xy(e,t,n){const r=this;return i;function i(o){return ie(o)?ce(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(Af,t,n)(o)}}function yy(e){e.exit("blockQuote")}const Rf={name:"characterEscape",tokenize:ky};function ky(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return iy(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const Mf={name:"characterReference",tokenize:wy};function wy(e,t,n){const r=this;let i=0,l,o;return a;function a(p){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(p),e.exit("characterReferenceMarker"),u}function u(p){return p===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(p),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=lt,d(p))}function c(p){return p===88||p===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(p),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=ry,d):(e.enter("characterReferenceValue"),l=7,o=Ma,d(p))}function d(p){if(p===59&&i){const f=e.exit("characterReferenceValue");return o===lt&&!Us(r.sliceSerialize(f))?n(p):(e.enter("characterReferenceMarker"),e.consume(p),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(p)&&i++<l?(e.consume(p),d):n(p)}}const Tc={partial:!0,tokenize:by},Lc={concrete:!0,name:"codeFenced",tokenize:Sy};function Sy(e,t,n){const r=this,i={partial:!0,tokenize:N};let l=0,o=0,a;return u;function u(S){return c(S)}function c(S){const C=r.events[r.events.length-1];return l=C&&C[1].type==="linePrefix"?C[2].sliceSerialize(C[1],!0).length:0,a=S,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(S)}function d(S){return S===a?(o++,e.consume(S),d):o<3?n(S):(e.exit("codeFencedFenceSequence"),ie(S)?ce(e,p,"whitespace")(S):p(S))}function p(S){return S===null||Q(S)?(e.exit("codeFencedFence"),r.interrupt?t(S):e.check(Tc,w,b)(S)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),f(S))}function f(S){return S===null||Q(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),p(S)):ie(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),ce(e,h,"whitespace")(S)):S===96&&S===a?n(S):(e.consume(S),f)}function h(S){return S===null||Q(S)?p(S):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),k(S))}function k(S){return S===null||Q(S)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),p(S)):S===96&&S===a?n(S):(e.consume(S),k)}function w(S){return e.attempt(i,b,I)(S)}function I(S){return e.enter("lineEnding"),e.consume(S),e.exit("lineEnding"),m}function m(S){return l>0&&ie(S)?ce(e,v,"linePrefix",l+1)(S):v(S)}function v(S){return S===null||Q(S)?e.check(Tc,w,b)(S):(e.enter("codeFlowValue"),x(S))}function x(S){return S===null||Q(S)?(e.exit("codeFlowValue"),v(S)):(e.consume(S),x)}function b(S){return e.exit("codeFenced"),t(S)}function N(S,C,P){let D=0;return A;function A(W){return S.enter("lineEnding"),S.consume(W),S.exit("lineEnding"),j}function j(W){return S.enter("codeFencedFence"),ie(W)?ce(S,E,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(W):E(W)}function E(W){return W===a?(S.enter("codeFencedFenceSequence"),U(W)):P(W)}function U(W){return W===a?(D++,S.consume(W),U):D>=o?(S.exit("codeFencedFenceSequence"),ie(W)?ce(S,V,"whitespace")(W):V(W)):P(W)}function V(W){return W===null||Q(W)?(S.exit("codeFencedFence"),C(W)):P(W)}}}function by(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const jo={name:"codeIndented",tokenize:jy},_y={partial:!0,tokenize:Cy};function jy(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),ce(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?u(c):Q(c)?e.attempt(_y,o,u)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||Q(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function u(c){return e.exit("codeIndented"),t(c)}}function Cy(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):Q(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):ce(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):Q(o)?i(o):n(o)}}const Ny={name:"codeText",previous:Ty,resolve:Ey,tokenize:Ly};function Ey(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function Ty(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function Ly(e,t,n){let r=0,i,l;return o;function o(p){return e.enter("codeText"),e.enter("codeTextSequence"),a(p)}function a(p){return p===96?(e.consume(p),r++,a):(e.exit("codeTextSequence"),u(p))}function u(p){return p===null?n(p):p===32?(e.enter("space"),e.consume(p),e.exit("space"),u):p===96?(l=e.enter("codeTextSequence"),i=0,d(p)):Q(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),u):(e.enter("codeTextData"),c(p))}function c(p){return p===null||p===32||p===96||Q(p)?(e.exit("codeTextData"),u(p)):(e.consume(p),c)}function d(p){return p===96?(e.consume(p),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(p)):(l.type="codeTextData",c(p))}}class Py{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&Tr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),Tr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),Tr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);Tr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);Tr(this.left,n.reverse())}}}function Tr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function Df(e){const t={};let n=-1,r,i,l,o,a,u,c;const d=new Py(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(u=r[1]._tokenizer.events,l=0,l<u.length&&u[l][1].type==="lineEndingBlank"&&(l+=2),l<u.length&&u[l][1].type==="content"))for(;++l<u.length&&u[l][1].type!=="content";)u[l][1].type==="chunkText"&&(u[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,Iy(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return zt(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function Iy(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,u=[],c={};let d,p,f=-1,h=n,k=0,w=0;const I=[w];for(;h;){for(;e.get(++i)[1]!==h;);l.push(i),h._tokenizer||(d=r.sliceStream(h),h.next||d.push(null),p&&o.defineSkip(h.start),h._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),h._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),p=h,h=h.next}for(h=n;++f<a.length;)a[f][0]==="exit"&&a[f-1][0]==="enter"&&a[f][1].type===a[f-1][1].type&&a[f][1].start.line!==a[f][1].end.line&&(w=f+1,I.push(w),h._tokenizer=void 0,h.previous=void 0,h=h.next);for(o.events=[],h?(h._tokenizer=void 0,h.previous=void 0):I.pop(),f=I.length;f--;){const m=a.slice(I[f],I[f+1]),v=l.pop();u.push([v,v+m.length-1]),e.splice(v,2,m)}for(u.reverse(),f=-1;++f<u.length;)c[k+u[f][0]]=k+u[f][1],k+=u[f][1]-u[f][0]-1;return c}const zy={resolve:Ry,tokenize:My},Ay={partial:!0,tokenize:Dy};function Ry(e){return Df(e),e}function My(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):Q(a)?e.check(Ay,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function Dy(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),ce(e,l,"linePrefix")}function l(o){if(o===null||Q(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function Ff(e,t,n,r,i,l,o,a,u){const c=u||Number.POSITIVE_INFINITY;let d=0;return p;function p(m){return m===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(m),e.exit(l),f):m===null||m===32||m===41||Ra(m)?n(m):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),w(m))}function f(m){return m===62?(e.enter(l),e.consume(m),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),h(m))}function h(m){return m===62?(e.exit("chunkString"),e.exit(a),f(m)):m===null||m===60||Q(m)?n(m):(e.consume(m),m===92?k:h)}function k(m){return m===60||m===62||m===92?(e.consume(m),h):h(m)}function w(m){return!d&&(m===null||m===41||Je(m))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(m)):d<c&&m===40?(e.consume(m),d++,w):m===41?(e.consume(m),d--,w):m===null||m===32||m===40||Ra(m)?n(m):(e.consume(m),m===92?I:w)}function I(m){return m===40||m===41||m===92?(e.consume(m),w):w(m)}}function Of(e,t,n,r,i,l){const o=this;let a=0,u;return c;function c(h){return e.enter(r),e.enter(i),e.consume(h),e.exit(i),e.enter(l),d}function d(h){return a>999||h===null||h===91||h===93&&!u||h===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(h):h===93?(e.exit(l),e.enter(i),e.consume(h),e.exit(i),e.exit(r),t):Q(h)?(e.enter("lineEnding"),e.consume(h),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),p(h))}function p(h){return h===null||h===91||h===93||Q(h)||a++>999?(e.exit("chunkString"),d(h)):(e.consume(h),u||(u=!ie(h)),h===92?f:p)}function f(h){return h===91||h===92||h===93?(e.consume(h),a++,p):p(h)}}function Bf(e,t,n,r,i,l){let o;return a;function a(f){return f===34||f===39||f===40?(e.enter(r),e.enter(i),e.consume(f),e.exit(i),o=f===40?41:f,u):n(f)}function u(f){return f===o?(e.enter(i),e.consume(f),e.exit(i),e.exit(r),t):(e.enter(l),c(f))}function c(f){return f===o?(e.exit(l),u(o)):f===null?n(f):Q(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),ce(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(f))}function d(f){return f===o||f===null||Q(f)?(e.exit("chunkString"),c(f)):(e.consume(f),f===92?p:d)}function p(f){return f===o||f===92?(e.consume(f),d):d(f)}}function Wr(e,t){let n;return r;function r(i){return Q(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):ie(i)?ce(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const Fy={name:"definition",tokenize:By},Oy={partial:!0,tokenize:$y};function By(e,t,n){const r=this;let i;return l;function l(h){return e.enter("definition"),o(h)}function o(h){return Of.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(h)}function a(h){return i=or(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),h===58?(e.enter("definitionMarker"),e.consume(h),e.exit("definitionMarker"),u):n(h)}function u(h){return Je(h)?Wr(e,c)(h):c(h)}function c(h){return Ff(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(h)}function d(h){return e.attempt(Oy,p,p)(h)}function p(h){return ie(h)?ce(e,f,"whitespace")(h):f(h)}function f(h){return h===null||Q(h)?(e.exit("definition"),r.parser.defined.push(i),t(h)):n(h)}}function $y(e,t,n){return r;function r(a){return Je(a)?Wr(e,i)(a):n(a)}function i(a){return Bf(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return ie(a)?ce(e,o,"whitespace")(a):o(a)}function o(a){return a===null||Q(a)?t(a):n(a)}}const Hy={name:"hardBreakEscape",tokenize:Uy};function Uy(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return Q(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const Vy={name:"headingAtx",resolve:Wy,tokenize:Qy};function Wy(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},zt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function Qy(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||Je(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),u(d)):d===null||Q(d)?(e.exit("atxHeading"),t(d)):ie(d)?ce(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function u(d){return d===35?(e.consume(d),u):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||Je(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const qy=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],Pc=["pre","script","style","textarea"],Ky={concrete:!0,name:"htmlFlow",resolveTo:Xy,tokenize:Jy},Yy={partial:!0,tokenize:e1},Gy={partial:!0,tokenize:Zy};function Xy(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function Jy(e,t,n){const r=this;let i,l,o,a,u;return c;function c(y){return d(y)}function d(y){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(y),p}function p(y){return y===33?(e.consume(y),f):y===47?(e.consume(y),l=!0,w):y===63?(e.consume(y),i=3,r.interrupt?t:g):Lt(y)?(e.consume(y),o=String.fromCharCode(y),I):n(y)}function f(y){return y===45?(e.consume(y),i=2,h):y===91?(e.consume(y),i=5,a=0,k):Lt(y)?(e.consume(y),i=4,r.interrupt?t:g):n(y)}function h(y){return y===45?(e.consume(y),r.interrupt?t:g):n(y)}function k(y){const J="CDATA[";return y===J.charCodeAt(a++)?(e.consume(y),a===J.length?r.interrupt?t:E:k):n(y)}function w(y){return Lt(y)?(e.consume(y),o=String.fromCharCode(y),I):n(y)}function I(y){if(y===null||y===47||y===62||Je(y)){const J=y===47,he=o.toLowerCase();return!J&&!l&&Pc.includes(he)?(i=1,r.interrupt?t(y):E(y)):qy.includes(o.toLowerCase())?(i=6,J?(e.consume(y),m):r.interrupt?t(y):E(y)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(y):l?v(y):x(y))}return y===45||lt(y)?(e.consume(y),o+=String.fromCharCode(y),I):n(y)}function m(y){return y===62?(e.consume(y),r.interrupt?t:E):n(y)}function v(y){return ie(y)?(e.consume(y),v):A(y)}function x(y){return y===47?(e.consume(y),A):y===58||y===95||Lt(y)?(e.consume(y),b):ie(y)?(e.consume(y),x):A(y)}function b(y){return y===45||y===46||y===58||y===95||lt(y)?(e.consume(y),b):N(y)}function N(y){return y===61?(e.consume(y),S):ie(y)?(e.consume(y),N):x(y)}function S(y){return y===null||y===60||y===61||y===62||y===96?n(y):y===34||y===39?(e.consume(y),u=y,C):ie(y)?(e.consume(y),S):P(y)}function C(y){return y===u?(e.consume(y),u=null,D):y===null||Q(y)?n(y):(e.consume(y),C)}function P(y){return y===null||y===34||y===39||y===47||y===60||y===61||y===62||y===96||Je(y)?N(y):(e.consume(y),P)}function D(y){return y===47||y===62||ie(y)?x(y):n(y)}function A(y){return y===62?(e.consume(y),j):n(y)}function j(y){return y===null||Q(y)?E(y):ie(y)?(e.consume(y),j):n(y)}function E(y){return y===45&&i===2?(e.consume(y),G):y===60&&i===1?(e.consume(y),oe):y===62&&i===4?(e.consume(y),L):y===63&&i===3?(e.consume(y),g):y===93&&i===5?(e.consume(y),B):Q(y)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(Yy,R,U)(y)):y===null||Q(y)?(e.exit("htmlFlowData"),U(y)):(e.consume(y),E)}function U(y){return e.check(Gy,V,R)(y)}function V(y){return e.enter("lineEnding"),e.consume(y),e.exit("lineEnding"),W}function W(y){return y===null||Q(y)?U(y):(e.enter("htmlFlowData"),E(y))}function G(y){return y===45?(e.consume(y),g):E(y)}function oe(y){return y===47?(e.consume(y),o="",_):E(y)}function _(y){if(y===62){const J=o.toLowerCase();return Pc.includes(J)?(e.consume(y),L):E(y)}return Lt(y)&&o.length<8?(e.consume(y),o+=String.fromCharCode(y),_):E(y)}function B(y){return y===93?(e.consume(y),g):E(y)}function g(y){return y===62?(e.consume(y),L):y===45&&i===2?(e.consume(y),g):E(y)}function L(y){return y===null||Q(y)?(e.exit("htmlFlowData"),R(y)):(e.consume(y),L)}function R(y){return e.exit("htmlFlow"),t(y)}}function Zy(e,t,n){const r=this;return i;function i(o){return Q(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function e1(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Wl,t,n)}}const t1={name:"htmlText",tokenize:n1};function n1(e,t,n){const r=this;let i,l,o;return a;function a(g){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(g),u}function u(g){return g===33?(e.consume(g),c):g===47?(e.consume(g),N):g===63?(e.consume(g),x):Lt(g)?(e.consume(g),P):n(g)}function c(g){return g===45?(e.consume(g),d):g===91?(e.consume(g),l=0,k):Lt(g)?(e.consume(g),v):n(g)}function d(g){return g===45?(e.consume(g),h):n(g)}function p(g){return g===null?n(g):g===45?(e.consume(g),f):Q(g)?(o=p,oe(g)):(e.consume(g),p)}function f(g){return g===45?(e.consume(g),h):p(g)}function h(g){return g===62?G(g):g===45?f(g):p(g)}function k(g){const L="CDATA[";return g===L.charCodeAt(l++)?(e.consume(g),l===L.length?w:k):n(g)}function w(g){return g===null?n(g):g===93?(e.consume(g),I):Q(g)?(o=w,oe(g)):(e.consume(g),w)}function I(g){return g===93?(e.consume(g),m):w(g)}function m(g){return g===62?G(g):g===93?(e.consume(g),m):w(g)}function v(g){return g===null||g===62?G(g):Q(g)?(o=v,oe(g)):(e.consume(g),v)}function x(g){return g===null?n(g):g===63?(e.consume(g),b):Q(g)?(o=x,oe(g)):(e.consume(g),x)}function b(g){return g===62?G(g):x(g)}function N(g){return Lt(g)?(e.consume(g),S):n(g)}function S(g){return g===45||lt(g)?(e.consume(g),S):C(g)}function C(g){return Q(g)?(o=C,oe(g)):ie(g)?(e.consume(g),C):G(g)}function P(g){return g===45||lt(g)?(e.consume(g),P):g===47||g===62||Je(g)?D(g):n(g)}function D(g){return g===47?(e.consume(g),G):g===58||g===95||Lt(g)?(e.consume(g),A):Q(g)?(o=D,oe(g)):ie(g)?(e.consume(g),D):G(g)}function A(g){return g===45||g===46||g===58||g===95||lt(g)?(e.consume(g),A):j(g)}function j(g){return g===61?(e.consume(g),E):Q(g)?(o=j,oe(g)):ie(g)?(e.consume(g),j):D(g)}function E(g){return g===null||g===60||g===61||g===62||g===96?n(g):g===34||g===39?(e.consume(g),i=g,U):Q(g)?(o=E,oe(g)):ie(g)?(e.consume(g),E):(e.consume(g),V)}function U(g){return g===i?(e.consume(g),i=void 0,W):g===null?n(g):Q(g)?(o=U,oe(g)):(e.consume(g),U)}function V(g){return g===null||g===34||g===39||g===60||g===61||g===96?n(g):g===47||g===62||Je(g)?D(g):(e.consume(g),V)}function W(g){return g===47||g===62||Je(g)?D(g):n(g)}function G(g){return g===62?(e.consume(g),e.exit("htmlTextData"),e.exit("htmlText"),t):n(g)}function oe(g){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),_}function _(g){return ie(g)?ce(e,B,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(g):B(g)}function B(g){return e.enter("htmlTextData"),o(g)}}const Ws={name:"labelEnd",resolveAll:o1,resolveTo:a1,tokenize:s1},r1={tokenize:u1},i1={tokenize:c1},l1={tokenize:d1};function o1(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&zt(e,0,e.length,n),e}function a1(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const u={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",u,t],["enter",c,t]],a=ft(a,e.slice(l+1,l+r+3)),a=ft(a,[["enter",d,t]]),a=ft(a,Vs(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=ft(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=ft(a,e.slice(o+1)),a=ft(a,[["exit",u,t]]),zt(e,l,e.length,a),e}function s1(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(f){return l?l._inactive?p(f):(o=r.parser.defined.includes(or(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(f),e.exit("labelMarker"),e.exit("labelEnd"),u):n(f)}function u(f){return f===40?e.attempt(r1,d,o?d:p)(f):f===91?e.attempt(i1,d,o?c:p)(f):o?d(f):p(f)}function c(f){return e.attempt(l1,d,p)(f)}function d(f){return t(f)}function p(f){return l._balanced=!0,n(f)}}function u1(e,t,n){return r;function r(p){return e.enter("resource"),e.enter("resourceMarker"),e.consume(p),e.exit("resourceMarker"),i}function i(p){return Je(p)?Wr(e,l)(p):l(p)}function l(p){return p===41?d(p):Ff(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(p)}function o(p){return Je(p)?Wr(e,u)(p):d(p)}function a(p){return n(p)}function u(p){return p===34||p===39||p===40?Bf(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(p):d(p)}function c(p){return Je(p)?Wr(e,d)(p):d(p)}function d(p){return p===41?(e.enter("resourceMarker"),e.consume(p),e.exit("resourceMarker"),e.exit("resource"),t):n(p)}}function c1(e,t,n){const r=this;return i;function i(a){return Of.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(or(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function d1(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const p1={name:"labelStartImage",resolveAll:Ws.resolveAll,tokenize:f1};function f1(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const h1={name:"labelStartLink",resolveAll:Ws.resolveAll,tokenize:m1};function m1(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const Co={name:"lineEnding",tokenize:g1};function g1(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),ce(e,t,"linePrefix")}}const Ji={name:"thematicBreak",tokenize:v1};function v1(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),u(c)):r>=3&&(c===null||Q(c))?(e.exit("thematicBreak"),t(c)):n(c)}function u(c){return c===i?(e.consume(c),r++,u):(e.exit("thematicBreakSequence"),ie(c)?ce(e,a,"whitespace")(c):a(c))}}const Qe={continuation:{tokenize:w1},exit:b1,name:"list",tokenize:k1},x1={partial:!0,tokenize:_1},y1={partial:!0,tokenize:S1};function k1(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(h){const k=r.containerState.type||(h===42||h===43||h===45?"listUnordered":"listOrdered");if(k==="listUnordered"?!r.containerState.marker||h===r.containerState.marker:Ma(h)){if(r.containerState.type||(r.containerState.type=k,e.enter(k,{_container:!0})),k==="listUnordered")return e.enter("listItemPrefix"),h===42||h===45?e.check(Ji,n,c)(h):c(h);if(!r.interrupt||h===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),u(h)}return n(h)}function u(h){return Ma(h)&&++o<10?(e.consume(h),u):(!r.interrupt||o<2)&&(r.containerState.marker?h===r.containerState.marker:h===41||h===46)?(e.exit("listItemValue"),c(h)):n(h)}function c(h){return e.enter("listItemMarker"),e.consume(h),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||h,e.check(Wl,r.interrupt?n:d,e.attempt(x1,f,p))}function d(h){return r.containerState.initialBlankLine=!0,l++,f(h)}function p(h){return ie(h)?(e.enter("listItemPrefixWhitespace"),e.consume(h),e.exit("listItemPrefixWhitespace"),f):n(h)}function f(h){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(h)}}function w1(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Wl,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,ce(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!ie(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(y1,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,ce(e,e.attempt(Qe,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function S1(e,t,n){const r=this;return ce(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function b1(e){e.exit(this.containerState.type)}function _1(e,t,n){const r=this;return ce(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!ie(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const Ic={name:"setextUnderline",resolveTo:j1,tokenize:C1};function j1(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function C1(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,p;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){p=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||p)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),ie(c)?ce(e,u,"lineSuffix")(c):u(c))}function u(c){return c===null||Q(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const N1={tokenize:E1};function E1(e){const t=this,n=e.attempt(Wl,r,e.attempt(this.parser.constructs.flowInitial,i,ce(e,e.attempt(this.parser.constructs.flow,i,e.attempt(zy,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const T1={resolveAll:Hf()},L1=$f("string"),P1=$f("text");function $f(e){return{resolveAll:Hf(e==="text"?I1:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),u}function u(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),u)}function c(d){if(d===null)return!0;const p=i[d];let f=-1;if(p)for(;++f<p.length;){const h=p[f];if(!h.previous||h.previous.call(r,r.previous))return!0}return!1}}}function Hf(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function I1(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,u;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)u=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||u||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const z1={42:Qe,43:Qe,45:Qe,48:Qe,49:Qe,50:Qe,51:Qe,52:Qe,53:Qe,54:Qe,55:Qe,56:Qe,57:Qe,62:Af},A1={91:Fy},R1={[-2]:jo,[-1]:jo,32:jo},M1={35:Vy,42:Ji,45:[Ic,Ji],60:Ky,61:Ic,95:Ji,96:Lc,126:Lc},D1={38:Mf,92:Rf},F1={[-5]:Co,[-4]:Co,[-3]:Co,33:p1,38:Mf,42:Da,60:[hy,t1],91:h1,92:[Hy,Rf],93:Ws,95:Da,96:Ny},O1={null:[Da,T1]},B1={null:[42,95]},$1={null:[]},H1=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:B1,contentInitial:A1,disable:$1,document:z1,flow:M1,flowInitial:R1,insideSpan:O1,string:D1,text:F1},Symbol.toStringTag,{value:"Module"}));function U1(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const u={attempt:C(N),check:C(S),consume:v,enter:x,exit:b,interrupt:C(S,{interrupt:!0})},c={code:null,containerState:{},defineSkip:w,events:[],now:k,parser:e,previous:null,sliceSerialize:f,sliceStream:h,write:p};let d=t.tokenize.call(c,u);return t.resolveAll&&l.push(t),c;function p(j){return o=ft(o,j),I(),o[o.length-1]!==null?[]:(P(t,0),c.events=Vs(l,c.events,c),c.events)}function f(j,E){return W1(h(j),E)}function h(j){return V1(o,j)}function k(){const{_bufferIndex:j,_index:E,line:U,column:V,offset:W}=r;return{_bufferIndex:j,_index:E,line:U,column:V,offset:W}}function w(j){i[j.line]=j.column,A()}function I(){let j;for(;r._index<o.length;){const E=o[r._index];if(typeof E=="string")for(j=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===j&&r._bufferIndex<E.length;)m(E.charCodeAt(r._bufferIndex));else m(E)}}function m(j){d=d(j)}function v(j){Q(j)?(r.line++,r.column=1,r.offset+=j===-3?2:1,A()):j!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=j}function x(j,E){const U=E||{};return U.type=j,U.start=k(),c.events.push(["enter",U,c]),a.push(U),U}function b(j){const E=a.pop();return E.end=k(),c.events.push(["exit",E,c]),E}function N(j,E){P(j,E.from)}function S(j,E){E.restore()}function C(j,E){return U;function U(V,W,G){let oe,_,B,g;return Array.isArray(V)?R(V):"tokenize"in V?R([V]):L(V);function L(ee){return ye;function ye(je){const ne=je!==null&&ee[je],Ee=je!==null&&ee.null,We=[...Array.isArray(ne)?ne:ne?[ne]:[],...Array.isArray(Ee)?Ee:Ee?[Ee]:[]];return R(We)(je)}}function R(ee){return oe=ee,_=0,ee.length===0?G:y(ee[_])}function y(ee){return ye;function ye(je){return g=D(),B=ee,ee.partial||(c.currentConstruct=ee),ee.name&&c.parser.constructs.disable.null.includes(ee.name)?he():ee.tokenize.call(E?Object.assign(Object.create(c),E):c,u,J,he)(je)}}function J(ee){return j(B,g),W}function he(ee){return g.restore(),++_<oe.length?y(oe[_]):G}}}function P(j,E){j.resolveAll&&!l.includes(j)&&l.push(j),j.resolve&&zt(c.events,E,c.events.length-E,j.resolve(c.events.slice(E),c)),j.resolveTo&&(c.events=j.resolveTo(c.events,c))}function D(){const j=k(),E=c.previous,U=c.currentConstruct,V=c.events.length,W=Array.from(a);return{from:V,restore:G};function G(){r=j,c.previous=E,c.currentConstruct=U,c.events.length=V,a=W,A()}}function A(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function V1(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function W1(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function Q1(e){const r={constructs:Zx([H1,...(e||{}).extensions||[]]),content:i(ay),defined:[],document:i(uy),flow:i(N1),lazy:{},string:i(L1),text:i(P1)};return r;function i(l){return o;function o(a){return U1(r,l,a)}}}function q1(e){for(;!Df(e););return e}const zc=/[\0\t\n\r]/g;function K1(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const u=[];let c,d,p,f,h;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),p=0,t="",n&&(l.charCodeAt(0)===65279&&p++,n=void 0);p<l.length;){if(zc.lastIndex=p,c=zc.exec(l),f=c&&c.index!==void 0?c.index:l.length,h=l.charCodeAt(f),!c){t=l.slice(p);break}if(h===10&&p===f&&r)u.push(-3),r=void 0;else switch(r&&(u.push(-5),r=void 0),p<f&&(u.push(l.slice(p,f)),e+=f-p),h){case 0:{u.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,u.push(-2);e++<d;)u.push(-1);break}case 10:{u.push(-4),e=1;break}default:r=!0,e=1}p=f+1}return a&&(r&&u.push(-5),t&&u.push(t),u.push(null)),u}}const Y1=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function G1(e){return e.replace(Y1,X1)}function X1(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return zf(n.slice(l?2:1),l?16:10)}return Us(n)||e}const Uf={}.hasOwnProperty;function J1(e,t,n){return typeof t!="string"&&(n=t,t=void 0),Z1(n)(q1(Q1(n).document().write(K1()(e,t,!0))))}function Z1(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(eu),autolinkProtocol:D,autolinkEmail:D,atxHeading:l(Xs),blockQuote:l(Ee),characterEscape:D,characterReference:D,codeFenced:l(We),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(We,o),codeText:l(Kt,o),codeTextData:D,data:D,codeFlowValue:D,definition:l(Yt),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(th),hardBreakEscape:l(Js),hardBreakTrailing:l(Js),htmlFlow:l(Zs,o),htmlFlowData:D,htmlText:l(Zs,o),htmlTextData:D,image:l(nh),label:o,link:l(eu),listItem:l(rh),listItemValue:f,listOrdered:l(tu,p),listUnordered:l(tu),paragraph:l(ih),reference:y,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Xs),strong:l(lh),thematicBreak:l(ah)},exit:{atxHeading:u(),atxHeadingSequence:N,autolink:u(),autolinkEmail:ne,autolinkProtocol:je,blockQuote:u(),characterEscapeValue:A,characterReferenceMarkerHexadecimal:he,characterReferenceMarkerNumeric:he,characterReferenceValue:ee,characterReference:ye,codeFenced:u(I),codeFencedFence:w,codeFencedFenceInfo:h,codeFencedFenceMeta:k,codeFlowValue:A,codeIndented:u(m),codeText:u(W),codeTextData:A,data:A,definition:u(),definitionDestinationString:b,definitionLabelString:v,definitionTitleString:x,emphasis:u(),hardBreakEscape:u(E),hardBreakTrailing:u(E),htmlFlow:u(U),htmlFlowData:A,htmlText:u(V),htmlTextData:A,image:u(oe),label:B,labelText:_,lineEnding:j,link:u(G),listItem:u(),listOrdered:u(),listUnordered:u(),paragraph:u(),referenceString:J,resourceDestinationString:g,resourceTitleString:L,resource:R,setextHeading:u(P),setextHeadingLineSequence:C,setextHeadingText:S,strong:u(),thematicBreak:u()}};Vf(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(T){let O={type:"root",children:[]};const K={stack:[O],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},te=[];let ae=-1;for(;++ae<T.length;)if(T[ae][1].type==="listOrdered"||T[ae][1].type==="listUnordered")if(T[ae][0]==="enter")te.push(ae);else{const xt=te.pop();ae=i(T,xt,ae)}for(ae=-1;++ae<T.length;){const xt=t[T[ae][0]];Uf.call(xt,T[ae][1].type)&&xt[T[ae][1].type].call(Object.assign({sliceSerialize:T[ae][2].sliceSerialize},K),T[ae][1])}if(K.tokenStack.length>0){const xt=K.tokenStack[K.tokenStack.length-1];(xt[1]||Ac).call(K,void 0,xt[0])}for(O.position={start:Xt(T.length>0?T[0][1].start:{line:1,column:1,offset:0}),end:Xt(T.length>0?T[T.length-2][1].end:{line:1,column:1,offset:0})},ae=-1;++ae<t.transforms.length;)O=t.transforms[ae](O)||O;return O}function i(T,O,K){let te=O-1,ae=-1,xt=!1,kn,At,yr,kr;for(;++te<=K;){const et=T[te];switch(et[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{et[0]==="enter"?ae++:ae--,kr=void 0;break}case"lineEndingBlank":{et[0]==="enter"&&(kn&&!kr&&!ae&&!yr&&(yr=te),kr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:kr=void 0}if(!ae&&et[0]==="enter"&&et[1].type==="listItemPrefix"||ae===-1&&et[0]==="exit"&&(et[1].type==="listUnordered"||et[1].type==="listOrdered")){if(kn){let Dn=te;for(At=void 0;Dn--;){const Rt=T[Dn];if(Rt[1].type==="lineEnding"||Rt[1].type==="lineEndingBlank"){if(Rt[0]==="exit")continue;At&&(T[At][1].type="lineEndingBlank",xt=!0),Rt[1].type="lineEnding",At=Dn}else if(!(Rt[1].type==="linePrefix"||Rt[1].type==="blockQuotePrefix"||Rt[1].type==="blockQuotePrefixWhitespace"||Rt[1].type==="blockQuoteMarker"||Rt[1].type==="listItemIndent"))break}yr&&(!At||yr<At)&&(kn._spread=!0),kn.end=Object.assign({},At?T[At][1].start:et[1].end),T.splice(At||te,0,["exit",kn,et[2]]),te++,K++}if(et[1].type==="listItemPrefix"){const Dn={type:"listItem",_spread:!1,start:Object.assign({},et[1].start),end:void 0};kn=Dn,T.splice(te,0,["enter",Dn,et[2]]),te++,K++,yr=void 0,kr=!0}}}return T[O][1]._spread=xt,K}function l(T,O){return K;function K(te){a.call(this,T(te),te),O&&O.call(this,te)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(T,O,K){this.stack[this.stack.length-1].children.push(T),this.stack.push(T),this.tokenStack.push([O,K||void 0]),T.position={start:Xt(O.start),end:void 0}}function u(T){return O;function O(K){T&&T.call(this,K),c.call(this,K)}}function c(T,O){const K=this.stack.pop(),te=this.tokenStack.pop();if(te)te[0].type!==T.type&&(O?O.call(this,T,te[0]):(te[1]||Ac).call(this,T,te[0]));else throw new Error("Cannot close `"+T.type+"` ("+Vr({start:T.start,end:T.end})+"): it’s not open");K.position.end=Xt(T.end)}function d(){return Xx(this.stack.pop())}function p(){this.data.expectingFirstListItemValue=!0}function f(T){if(this.data.expectingFirstListItemValue){const O=this.stack[this.stack.length-2];O.start=Number.parseInt(this.sliceSerialize(T),10),this.data.expectingFirstListItemValue=void 0}}function h(){const T=this.resume(),O=this.stack[this.stack.length-1];O.lang=T}function k(){const T=this.resume(),O=this.stack[this.stack.length-1];O.meta=T}function w(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function I(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function m(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T.replace(/(\r?\n|\r)$/g,"")}function v(T){const O=this.resume(),K=this.stack[this.stack.length-1];K.label=O,K.identifier=or(this.sliceSerialize(T)).toLowerCase()}function x(){const T=this.resume(),O=this.stack[this.stack.length-1];O.title=T}function b(){const T=this.resume(),O=this.stack[this.stack.length-1];O.url=T}function N(T){const O=this.stack[this.stack.length-1];if(!O.depth){const K=this.sliceSerialize(T).length;O.depth=K}}function S(){this.data.setextHeadingSlurpLineEnding=!0}function C(T){const O=this.stack[this.stack.length-1];O.depth=this.sliceSerialize(T).codePointAt(0)===61?1:2}function P(){this.data.setextHeadingSlurpLineEnding=void 0}function D(T){const K=this.stack[this.stack.length-1].children;let te=K[K.length-1];(!te||te.type!=="text")&&(te=oh(),te.position={start:Xt(T.start),end:void 0},K.push(te)),this.stack.push(te)}function A(T){const O=this.stack.pop();O.value+=this.sliceSerialize(T),O.position.end=Xt(T.end)}function j(T){const O=this.stack[this.stack.length-1];if(this.data.atHardBreak){const K=O.children[O.children.length-1];K.position.end=Xt(T.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(O.type)&&(D.call(this,T),A.call(this,T))}function E(){this.data.atHardBreak=!0}function U(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function V(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function W(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function G(){const T=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";T.type+="Reference",T.referenceType=O,delete T.url,delete T.title}else delete T.identifier,delete T.label;this.data.referenceType=void 0}function oe(){const T=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";T.type+="Reference",T.referenceType=O,delete T.url,delete T.title}else delete T.identifier,delete T.label;this.data.referenceType=void 0}function _(T){const O=this.sliceSerialize(T),K=this.stack[this.stack.length-2];K.label=G1(O),K.identifier=or(O).toLowerCase()}function B(){const T=this.stack[this.stack.length-1],O=this.resume(),K=this.stack[this.stack.length-1];if(this.data.inReference=!0,K.type==="link"){const te=T.children;K.children=te}else K.alt=O}function g(){const T=this.resume(),O=this.stack[this.stack.length-1];O.url=T}function L(){const T=this.resume(),O=this.stack[this.stack.length-1];O.title=T}function R(){this.data.inReference=void 0}function y(){this.data.referenceType="collapsed"}function J(T){const O=this.resume(),K=this.stack[this.stack.length-1];K.label=O,K.identifier=or(this.sliceSerialize(T)).toLowerCase(),this.data.referenceType="full"}function he(T){this.data.characterReferenceType=T.type}function ee(T){const O=this.sliceSerialize(T),K=this.data.characterReferenceType;let te;K?(te=zf(O,K==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):te=Us(O);const ae=this.stack[this.stack.length-1];ae.value+=te}function ye(T){const O=this.stack.pop();O.position.end=Xt(T.end)}function je(T){A.call(this,T);const O=this.stack[this.stack.length-1];O.url=this.sliceSerialize(T)}function ne(T){A.call(this,T);const O=this.stack[this.stack.length-1];O.url="mailto:"+this.sliceSerialize(T)}function Ee(){return{type:"blockquote",children:[]}}function We(){return{type:"code",lang:null,meta:null,value:""}}function Kt(){return{type:"inlineCode",value:""}}function Yt(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function th(){return{type:"emphasis",children:[]}}function Xs(){return{type:"heading",depth:0,children:[]}}function Js(){return{type:"break"}}function Zs(){return{type:"html",value:""}}function nh(){return{type:"image",title:null,url:"",alt:null}}function eu(){return{type:"link",title:null,url:"",children:[]}}function tu(T){return{type:"list",ordered:T.type==="listOrdered",start:null,spread:T._spread,children:[]}}function rh(T){return{type:"listItem",spread:T._spread,checked:null,children:[]}}function ih(){return{type:"paragraph",children:[]}}function lh(){return{type:"strong",children:[]}}function oh(){return{type:"text",value:""}}function ah(){return{type:"thematicBreak"}}}function Xt(e){return{line:e.line,column:e.column,offset:e.offset}}function Vf(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Vf(e,r):e0(e,r)}}function e0(e,t){let n;for(n in t)if(Uf.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function Ac(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Vr({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Vr({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Vr({start:t.start,end:t.end})+") is still open")}function t0(e){const t=this;t.parser=n;function n(r){return J1(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function n0(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function r0(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function i0(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function l0(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function o0(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function a0(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=xr(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const u={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,u);const c={type:"element",tagName:"sup",properties:{},children:[u]};return e.patch(t,c),e.applyData(t,c)}function s0(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function u0(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Wf(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function c0(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Wf(e,t);const i={src:xr(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function d0(e,t){const n={src:xr(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function p0(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function f0(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Wf(e,t);const i={href:xr(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function h0(e,t){const n={href:xr(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function m0(e,t,n){const r=e.all(t),i=n?g0(n):Qf(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let p;d&&d.type==="element"&&d.tagName==="p"?p=d:(p={type:"element",tagName:"p",properties:{},children:[]},r.unshift(p)),p.children.length>0&&p.children.unshift({type:"text",value:" "}),p.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const u=r[r.length-1];u&&(i||u.type!=="element"||u.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function g0(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Qf(n[r])}return t}function Qf(e){const t=e.spread;return t??e.children.length>1}function v0(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function x0(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function y0(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function k0(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function w0(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=Os(t.children[1]),u=Cf(t.children[t.children.length-1]);a&&u&&(o.position={start:a,end:u}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function S0(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let u=-1;const c=[];for(;++u<a;){const p=t.children[u],f={},h=o?o[u]:void 0;h&&(f.align=h);let k={type:"element",tagName:l,properties:f,children:[]};p&&(k.children=e.all(p),e.patch(p,k),k=e.applyData(p,k)),c.push(k)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function b0(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const Rc=9,Mc=32;function _0(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(Dc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(Dc(t.slice(i),i>0,!1)),l.join("")}function Dc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===Rc||l===Mc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===Rc||l===Mc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function j0(e,t){const n={type:"text",value:_0(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function C0(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const N0={blockquote:n0,break:r0,code:i0,delete:l0,emphasis:o0,footnoteReference:a0,heading:s0,html:u0,imageReference:c0,image:d0,inlineCode:p0,linkReference:f0,link:h0,listItem:m0,list:v0,paragraph:x0,root:y0,strong:k0,table:w0,tableCell:b0,tableRow:S0,text:j0,thematicBreak:C0,toml:Mi,yaml:Mi,definition:Mi,footnoteDefinition:Mi};function Mi(){}const qf=-1,Ql=0,Qr=1,Cl=2,Qs=3,qs=4,Ks=5,Ys=6,Kf=7,Yf=8,Fc=typeof self=="object"?self:globalThis,E0=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Ql:case qf:return n(o,i);case Qr:{const a=n([],i);for(const u of o)a.push(r(u));return a}case Cl:{const a=n({},i);for(const[u,c]of o)a[r(u)]=r(c);return a}case Qs:return n(new Date(o),i);case qs:{const{source:a,flags:u}=o;return n(new RegExp(a,u),i)}case Ks:{const a=n(new Map,i);for(const[u,c]of o)a.set(r(u),r(c));return a}case Ys:{const a=n(new Set,i);for(const u of o)a.add(r(u));return a}case Kf:{const{name:a,message:u}=o;return n(new Fc[a](u),i)}case Yf:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new Fc[l](o),i)};return r},Oc=e=>E0(new Map,e)(0),Bn="",{toString:T0}={},{keys:L0}=Object,Lr=e=>{const t=typeof e;if(t!=="object"||!e)return[Ql,t];const n=T0.call(e).slice(8,-1);switch(n){case"Array":return[Qr,Bn];case"Object":return[Cl,Bn];case"Date":return[Qs,Bn];case"RegExp":return[qs,Bn];case"Map":return[Ks,Bn];case"Set":return[Ys,Bn];case"DataView":return[Qr,n]}return n.includes("Array")?[Qr,n]:n.includes("Error")?[Kf,n]:[Cl,n]},Di=([e,t])=>e===Ql&&(t==="function"||t==="symbol"),P0=(e,t,n,r)=>{const i=(o,a)=>{const u=r.push(o)-1;return n.set(a,u),u},l=o=>{if(n.has(o))return n.get(o);let[a,u]=Lr(o);switch(a){case Ql:{let d=o;switch(u){case"bigint":a=Yf,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+u);d=null;break;case"undefined":return i([qf],o)}return i([a,d],o)}case Qr:{if(u){let f=o;return u==="DataView"?f=new Uint8Array(o.buffer):u==="ArrayBuffer"&&(f=new Uint8Array(o)),i([u,[...f]],o)}const d=[],p=i([a,d],o);for(const f of o)d.push(l(f));return p}case Cl:{if(u)switch(u){case"BigInt":return i([u,o.toString()],o);case"Boolean":case"Number":case"String":return i([u,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],p=i([a,d],o);for(const f of L0(o))(e||!Di(Lr(o[f])))&&d.push([l(f),l(o[f])]);return p}case Qs:return i([a,o.toISOString()],o);case qs:{const{source:d,flags:p}=o;return i([a,{source:d,flags:p}],o)}case Ks:{const d=[],p=i([a,d],o);for(const[f,h]of o)(e||!(Di(Lr(f))||Di(Lr(h))))&&d.push([l(f),l(h)]);return p}case Ys:{const d=[],p=i([a,d],o);for(const f of o)(e||!Di(Lr(f)))&&d.push(l(f));return p}}const{message:c}=o;return i([a,{name:u,message:c}],o)};return l},Bc=(e,{json:t,lossy:n}={})=>{const r=[];return P0(!(t||n),!!t,new Map,r)(e),r},Nl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?Oc(Bc(e,t)):structuredClone(e):(e,t)=>Oc(Bc(e,t));function I0(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function z0(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function A0(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||I0,r=e.options.footnoteBackLabel||z0,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let u=-1;for(;++u<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[u]);if(!c)continue;const d=e.all(c),p=String(c.identifier).toUpperCase(),f=xr(p.toLowerCase());let h=0;const k=[],w=e.footnoteCounts.get(p);for(;w!==void 0&&++h<=w;){k.length>0&&k.push({type:"text",value:" "});let v=typeof n=="string"?n:n(u,h);typeof v=="string"&&(v={type:"text",value:v}),k.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+f+(h>1?"-"+h:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(u,h),className:["data-footnote-backref"]},children:Array.isArray(v)?v:[v]})}const I=d[d.length-1];if(I&&I.type==="element"&&I.tagName==="p"){const v=I.children[I.children.length-1];v&&v.type==="text"?v.value+=" ":I.children.push({type:"text",value:" "}),I.children.push(...k)}else d.push(...k);const m={type:"element",tagName:"li",properties:{id:t+"fn-"+f},children:e.wrap(d,!0)};e.patch(c,m),a.push(m)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...Nl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const Gf=function(e){if(e==null)return F0;if(typeof e=="function")return ql(e);if(typeof e=="object")return Array.isArray(e)?R0(e):M0(e);if(typeof e=="string")return D0(e);throw new Error("Expected function, string, or object as test")};function R0(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=Gf(e[n]);return ql(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function M0(e){const t=e;return ql(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function D0(e){return ql(t);function t(n){return n&&n.type===e}}function ql(e){return t;function t(n,r,i){return!!(O0(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function F0(){return!0}function O0(e){return e!==null&&typeof e=="object"&&"type"in e}const Xf=[],B0=!0,$c=!1,$0="skip";function H0(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=Gf(i),o=r?-1:1;a(e,void 0,[])();function a(u,c,d){const p=u&&typeof u=="object"?u:{};if(typeof p.type=="string"){const h=typeof p.tagName=="string"?p.tagName:typeof p.name=="string"?p.name:void 0;Object.defineProperty(f,"name",{value:"node ("+(u.type+(h?"<"+h+">":""))+")"})}return f;function f(){let h=Xf,k,w,I;if((!t||l(u,c,d[d.length-1]||void 0))&&(h=U0(n(u,d)),h[0]===$c))return h;if("children"in u&&u.children){const m=u;if(m.children&&h[0]!==$0)for(w=(r?m.children.length:-1)+o,I=d.concat(m);w>-1&&w<m.children.length;){const v=m.children[w];if(k=a(v,w,I)(),k[0]===$c)return k;w=typeof k[1]=="number"?k[1]:w+o}}return h}}}function U0(e){return Array.isArray(e)?e:typeof e=="number"?[B0,e]:e==null?Xf:[e]}function Jf(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),H0(e,l,a,i);function a(u,c){const d=c[c.length-1],p=d?d.children.indexOf(u):void 0;return o(u,p,d)}}const Fa={}.hasOwnProperty,V0={};function W0(e,t){const n=t||V0,r=new Map,i=new Map,l=new Map,o={...N0,...n.handlers},a={all:c,applyData:q0,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:u,options:n,patch:Q0,wrap:Y0};return Jf(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const p=d.type==="definition"?r:i,f=String(d.identifier).toUpperCase();p.has(f)||p.set(f,d)}}),a;function u(d,p){const f=d.type,h=a.handlers[f];if(Fa.call(a.handlers,f)&&h)return h(a,d,p);if(a.options.passThrough&&a.options.passThrough.includes(f)){if("children"in d){const{children:w,...I}=d,m=Nl(I);return m.children=a.all(d),m}return Nl(d)}return(a.options.unknownHandler||K0)(a,d,p)}function c(d){const p=[];if("children"in d){const f=d.children;let h=-1;for(;++h<f.length;){const k=a.one(f[h],d);if(k){if(h&&f[h-1].type==="break"&&(!Array.isArray(k)&&k.type==="text"&&(k.value=Hc(k.value)),!Array.isArray(k)&&k.type==="element")){const w=k.children[0];w&&w.type==="text"&&(w.value=Hc(w.value))}Array.isArray(k)?p.push(...k):p.push(k)}}}return p}}function Q0(e,t){e.position&&(t.position=Tx(e))}function q0(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,Nl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function K0(e,t){const n=t.data||{},r="value"in t&&!(Fa.call(n,"hProperties")||Fa.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function Y0(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Hc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Uc(e,t){const n=W0(e,t),r=n.one(e,void 0),i=A0(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function G0(e,t){return e&&"run"in e?async function(n,r){const i=Uc(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Uc(n,{file:r,...e||t})}}function Vc(e){if(e)throw e}var Zi=Object.prototype.hasOwnProperty,Zf=Object.prototype.toString,Wc=Object.defineProperty,Qc=Object.getOwnPropertyDescriptor,qc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Zf.call(t)==="[object Array]"},Kc=function(t){if(!t||Zf.call(t)!=="[object Object]")return!1;var n=Zi.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Zi.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Zi.call(t,i)},Yc=function(t,n){Wc&&n.name==="__proto__"?Wc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Gc=function(t,n){if(n==="__proto__")if(Zi.call(t,n)){if(Qc)return Qc(t,n).value}else return;return t[n]},X0=function e(){var t,n,r,i,l,o,a=arguments[0],u=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},u=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});u<c;++u)if(t=arguments[u],t!=null)for(n in t)r=Gc(a,n),i=Gc(t,n),a!==i&&(d&&i&&(Kc(i)||(l=qc(i)))?(l?(l=!1,o=r&&qc(r)?r:[]):o=r&&Kc(r)?r:{},Yc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Yc(a,{name:n,newValue:i}));return a};const No=$a(X0);function Oa(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function J0(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(u,...c){const d=e[++l];let p=-1;if(u){o(u);return}for(;++p<i.length;)(c[p]===null||c[p]===void 0)&&(c[p]=i[p]);i=c,d?Z0(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function Z0(e,t){let n;return r;function r(...o){const a=e.length>o.length;let u;a&&o.push(i);try{u=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(u&&u.then&&typeof u.then=="function"?u.then(l,i):u instanceof Error?i(u):l(u))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const Et={basename:ek,dirname:tk,extname:nk,join:rk,sep:"/"};function ek(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');vi(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function tk(e){if(vi(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function nk(e){vi(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function rk(...e){let t=-1,n;for(;++t<e.length;)vi(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":ik(n)}function ik(e){vi(e);const t=e.codePointAt(0)===47;let n=lk(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function lk(e,t){let n="",r=0,i=-1,l=0,o=-1,a,u;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(u=n.lastIndexOf("/"),u!==n.length-1){u<0?(n="",r=0):(n=n.slice(0,u),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function vi(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const ok={cwd:ak};function ak(){return"/"}function Ba(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function sk(e){if(typeof e=="string")e=new URL(e);else if(!Ba(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return uk(e)}function uk(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const Eo=["history","path","basename","stem","extname","dirname"];class eh{constructor(t){let n;t?Ba(t)?n={path:t}:typeof t=="string"||ck(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":ok.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<Eo.length;){const l=Eo[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)Eo.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?Et.basename(this.path):void 0}set basename(t){Lo(t,"basename"),To(t,"basename"),this.path=Et.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?Et.dirname(this.path):void 0}set dirname(t){Xc(this.basename,"dirname"),this.path=Et.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?Et.extname(this.path):void 0}set extname(t){if(To(t,"extname"),Xc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=Et.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){Ba(t)&&(t=sk(t)),Lo(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?Et.basename(this.path,this.extname):void 0}set stem(t){Lo(t,"stem"),To(t,"stem"),this.path=Et.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new Fe(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function To(e,t){if(e&&e.includes(Et.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+Et.sep+"`")}function Lo(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Xc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function ck(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const dk=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},pk={}.hasOwnProperty;class Gs extends dk{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=J0()}copy(){const t=new Gs;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(No(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(zo("data",this.frozen),this.namespace[t]=n,this):pk.call(this.namespace,t)&&this.namespace[t]||void 0:t?(zo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Fi(t),r=this.parser||this.Parser;return Po("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),Po("process",this.parser||this.Parser),Io("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Fi(t),u=r.parse(a);r.run(u,a,function(d,p,f){if(d||!p||!f)return c(d);const h=p,k=r.stringify(h,f);mk(k)?f.value=k:f.result=k,c(d,f)});function c(d,p){d||!p?o(d):l?l(p):n(void 0,p)}}}processSync(t){let n=!1,r;return this.freeze(),Po("processSync",this.parser||this.Parser),Io("processSync",this.compiler||this.Compiler),this.process(t,i),Zc("processSync","process",n),r;function i(l,o){n=!0,Vc(l),r=o}}run(t,n,r){Jc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const u=Fi(n);i.run(t,u,c);function c(d,p,f){const h=p||t;d?a(d):o?o(h):r(void 0,h,f)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Zc("runSync","run",r),i;function l(o,a){Vc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Fi(n),i=this.compiler||this.Compiler;return Io("stringify",i),Jc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(zo("use",this.frozen),t!=null)if(typeof t=="function")u(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")u(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...p]=c;u(d,p)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=No(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const p=c[d];l(p)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function u(c,d){let p=-1,f=-1;for(;++p<r.length;)if(r[p][0]===c){f=p;break}if(f===-1)r.push([c,...d]);else if(d.length>0){let[h,...k]=d;const w=r[f][1];Oa(w)&&Oa(h)&&(h=No(!0,w,h)),r[f]=[c,h,...k]}}}}const fk=new Gs().freeze();function Po(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function Io(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function zo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Jc(e){if(!Oa(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Zc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Fi(e){return hk(e)?e:new eh(e)}function hk(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function mk(e){return typeof e=="string"||gk(e)}function gk(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const vk="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",ed=[],td={allowDangerousHtml:!0},xk=/^(https?|ircs?|mailto|xmpp)$/i,yk=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function kk(e){const t=wk(e),n=Sk(e);return bk(t.runSync(t.parse(n),n),e)}function wk(e){const t=e.rehypePlugins||ed,n=e.remarkPlugins||ed,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...td}:td;return fk().use(t0).use(n).use(G0,r).use(t)}function Sk(e){const t=e.children||"",n=new eh;return typeof t=="string"&&(n.value=t),n}function bk(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,u=t.urlTransform||_k;for(const d of yk)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+vk+d.id,void 0);return Jf(e,c),Ax(e,{Fragment:s.Fragment,components:i,ignoreInvalidStyle:!0,jsx:s.jsx,jsxs:s.jsxs,passKeys:!0,passNode:!0});function c(d,p,f){if(d.type==="raw"&&f&&typeof p=="number")return o?f.children.splice(p,1):f.children[p]={type:"text",value:d.value},p;if(d.type==="element"){let h;for(h in _o)if(Object.hasOwn(_o,h)&&Object.hasOwn(d.properties,h)){const k=d.properties[h],w=_o[h];(w===null||w.includes(d.tagName))&&(d.properties[h]=u(String(k||""),h,d))}}if(d.type==="element"){let h=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!h&&r&&typeof p=="number"&&(h=!r(d,p,f)),h&&f&&typeof p=="number")return a&&d.children?f.children.splice(p,1,...d.children):f.children.splice(p,1),p}}}function _k(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||xk.test(e.slice(0,t))?e:""}const jk=e=>{if(!e)return null;try{return JSON.parse(e).execution_stats||null}catch{return null}},Ck=e=>{if(e.kind!=="status")return!1;const t=e.content.toLowerCase();return t.includes("running")||t.includes("thinking")||t.includes("executing")||t.includes("processing")},nd=10*1024,Ao=200,Re={send:s.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),s.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),s.jsx("polyline",{points:"14 2 14 8 20 8"}),s.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),s.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("circle",{cx:"12",cy:"12",r:"10"}),s.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),s.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),s.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),s.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),s.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),s.jsx("circle",{cx:"12",cy:"5",r:"2"}),s.jsx("path",{d:"M12 7v4"})]}),check:s.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),s.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),file:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),s.jsx("polyline",{points:"14 2 14 8 20 8"})]}),folder:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),spinner:s.jsx("svg",{className:"spinner-icon",width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M21 12a9 9 0 1 1-6.219-8.56"})})},Nk=e=>{switch(e){case"directive":return Re.directive;case"question":return Re.question;case"status":return Re.status;case"result":return Re.result;case"approval_request":return Re.lock;default:return Re.directive}},Ek=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=z.useRef(null),[a,u]=Jt.useState(""),[c,d]=Jt.useState("directive"),[p,f]=Jt.useState(""),[h,k]=Jt.useState(!1),[w,I]=Jt.useState(new Map),[m,v]=Jt.useState(new Set),[x,b]=z.useState(new Set),[N,S]=z.useState(new Set),C=_=>{const B=(_.match(/\n/g)||[]).length+1;if(!(_.length>nd||B>Ao))return{needsTruncation:!1,truncated:_,fullLength:_.length,lineCount:B};let L=_.slice(0,nd);const R=L.split(`
`);R.length>Ao&&(L=R.slice(0,Ao).join(`
`));const y=L.lastIndexOf(`
`);return y>L.length*.8&&(L=L.slice(0,y)),{needsTruncation:!0,truncated:L,fullLength:_.length,lineCount:B}},P=_=>{b(B=>{const g=new Set(B);return g.has(_)?g.delete(_):g.add(_),g})};z.useEffect(()=>{e!=null&&e.workspace?f(e.workspace):f("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),z.useEffect(()=>{var _;(_=o.current)==null||_.scrollIntoView({behavior:"smooth"})},[t]);const D=_=>{f(_),r&&r(_)},A=()=>{a.trim()&&(n(a,c,p||void 0),u(""))},j=_=>{_.key==="Enter"&&!_.shiftKey&&(_.preventDefault(),A())},E=_=>new Date(_).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),U=_=>_.length>12?`${_.slice(0,8)}...`:_,V=_=>{if(!_.metadata_json)return null;try{return JSON.parse(_.metadata_json).approval_id||null}catch{return null}},W=_=>{const B=w.get(_)||"";i&&(i(_,B),v(g=>new Set(g).add(_)),I(g=>{const L=new Map(g);return L.delete(_),L}))},G=_=>{const B=w.get(_)||"";if(!B.trim()){alert("Please provide a reason for rejection");return}l&&(l(_,B),v(g=>new Set(g).add(_)),I(g=>{const L=new Map(g);return L.delete(_),L}))},oe=(_,B)=>{I(g=>new Map(g).set(_,B))};return e?s.jsxs("div",{className:"conversation-view",children:[s.jsxs("div",{className:"conversation-header",children:[s.jsxs("div",{className:"header-info",children:[s.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&s.jsxs("span",{className:"thread-agent-badge",children:[Re.bot,e.target_agent]})]}),s.jsxs("div",{className:"header-stats",children:[s.jsxs("span",{className:"message-count",children:[t.length," messages"]}),s.jsx("span",{className:"thread-id",title:e.id,children:U(e.id)})]})]}),s.jsxs("div",{className:"messages-container",children:[t.length===0?s.jsxs("div",{className:"empty-messages",children:[s.jsx("div",{className:"empty-icon",children:s.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:s.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),s.jsx("p",{children:"No messages yet"}),s.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((_,B)=>{const g=_.from_type==="human",L=B===0||t[B-1].from_type!==_.from_type,R=x.has(_.id),{needsTruncation:y,truncated:J,fullLength:he,lineCount:ee}=C(_.content),ye=R?_.content:J,je=Ck(_);return s.jsxs("div",{className:`message ${g?"human":"agent"}${je?" running-status":""}`,children:[s.jsx("div",{className:`message-avatar ${L?"visible":""}`,children:L&&(g?Re.user:Re.bot)}),s.jsxs("div",{className:"message-body",children:[L&&s.jsxs("div",{className:"message-meta",children:[s.jsx("span",{className:"sender-name",children:_.from_id}),s.jsxs("span",{className:`kind-badge${je?" running":""}`,children:[je?Re.spinner:Nk(_.kind)," ",_.kind]}),s.jsx("span",{className:"message-time",children:E(_.created_at)})]}),s.jsxs("div",{className:"message-content",children:[_.kind==="result"||!g?s.jsx(kk,{components:{a:({href:ne,children:Ee})=>{let We=ne;return ne&&ne.startsWith("/")&&!ne.startsWith("//")&&(We=`file://${ne}`),s.jsx("a",{href:We,target:"_blank",rel:"noopener noreferrer",children:Ee})},code:({className:ne,children:Ee,...We})=>!ne?s.jsx("code",{className:"inline-code",...We,children:Ee}):s.jsx("code",{className:ne,...We,children:Ee})},children:ye}):ye,y&&s.jsx("div",{className:"truncation-notice",children:s.jsx("button",{className:"expand-btn",onClick:()=>P(_.id),children:R?s.jsx(s.Fragment,{children:"Show less"}):s.jsxs(s.Fragment,{children:["Show more (",Math.round(he/1024),"KB, ",ee," lines)"]})})}),_.kind==="approval_request"&&(()=>{const ne=V(_),Ee=ne&&m.has(ne);return ne?s.jsx("div",{className:"inline-approval",children:Ee?s.jsxs("div",{className:"approval-handled",children:[Re.check,s.jsx("span",{children:"Action taken"})]}):s.jsxs(s.Fragment,{children:[s.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:w.get(ne)||"",onChange:We=>oe(ne,We.target.value)}),s.jsxs("div",{className:"approval-actions",children:[s.jsxs("button",{className:"reject-btn",onClick:()=>G(ne),title:"Reject",children:[Re.x,"Reject"]}),s.jsxs("button",{className:"approve-btn",onClick:()=>W(ne),title:"Approve",children:[Re.check,"Approve"]})]})]})}):null})(),_.kind==="result"&&(()=>{const ne=jk(_.metadata_json);if(!ne||!ne.files_created||ne.files_created.length===0)return null;const Ee=N.has(_.id),We=()=>{S(Kt=>{const Yt=new Set(Kt);return Yt.has(_.id)?Yt.delete(_.id):Yt.add(_.id),Yt})};return s.jsxs("div",{className:"files-created-section",children:[s.jsxs("button",{className:`files-toggle-btn ${Ee?"expanded":""}`,onClick:We,children:[Re.file,s.jsxs("span",{children:["Files Created (",ne.files_created.length,")"]}),ne.workspace&&s.jsxs("span",{className:"workspace-badge",title:ne.workspace,children:[Re.folder,ne.workspace.split("/").pop()]}),s.jsx("span",{className:"toggle-chevron",children:Ee?"▼":"▶"})]}),Ee&&s.jsx("ul",{className:"files-list",children:ne.files_created.map((Kt,Yt)=>s.jsx("li",{className:"file-item",children:s.jsx("a",{href:`file://${ne.workspace?ne.workspace+"/":""}${Kt}`,target:"_blank",rel:"noopener noreferrer",title:Kt,children:Kt})},Yt))})]})})()]}),s.jsx("div",{className:"message-footer",children:s.jsxs("span",{className:"message-seq",children:["#",_.message_seq]})})]})]},_.id)}),s.jsx("div",{ref:o})]}),s.jsxs("div",{className:"input-area",children:[h&&s.jsxs("div",{className:"workspace-input-row",children:[s.jsx("input",{type:"text",value:p,onChange:_=>D(_.target.value),onBlur:()=>{r&&r(p)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),s.jsx("button",{onClick:async()=>{try{const B=await(await fetch("/api/select-folder")).json();!B.cancelled&&B.path&&D(B.path)}catch(_){console.error("Failed to open folder picker:",_)}},className:"workspace-browse",title:"Browse for folder",children:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),s.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),s.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),p&&s.jsx("button",{onClick:()=>{D(""),k(!1)},className:"workspace-clear",children:"Clear"})]}),s.jsxs("div",{className:"input-wrapper",children:[s.jsx("button",{onClick:()=>k(!h),className:`workspace-toggle ${p?"has-workspace":""}`,title:p||"Set working directory for agent tasks",children:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),s.jsxs("select",{value:c,onChange:_=>d(_.target.value),className:"kind-selector",title:c==="directive"?"Directive: A task or instruction for the agent to execute":"Question: A query for information (won't trigger execution)",children:[s.jsx("option",{value:"directive",title:"A task or instruction for the agent to execute",children:"Directive"}),s.jsx("option",{value:"question",title:"A query for information (won't trigger execution)",children:"Question"})]}),s.jsx("textarea",{value:a,onChange:_=>u(_.target.value),onKeyPress:j,placeholder:p?`Message (workspace: ${p.split("/").pop()})`:"Type a message...",rows:1}),s.jsx("button",{onClick:A,className:"send-btn",disabled:!a.trim(),children:Re.send})]}),s.jsxs("div",{className:"input-hint",children:["Press ",s.jsx("kbd",{children:"Enter"})," to send, ",s.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),s.jsx("style",{children:`
        .conversation-view {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg-base);
        }

        /* Header */
        .conversation-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-3) var(--space-4);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
        }

        .header-info {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .thread-title {
          font-size: var(--text-base);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin: 0;
        }

        .thread-agent-badge {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          color: var(--color-primary);
          padding: 2px 8px;
          background: rgba(37, 194, 160, 0.1);
          border-radius: var(--radius-sm);
        }

        .thread-agent-badge svg {
          opacity: 0.8;
        }

        .thread-id {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
        }

        .header-stats {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .message-count {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Messages Container */
        .messages-container {
          flex: 1;
          overflow-y: auto;
          padding: var(--space-4);
        }

        .empty-messages {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          height: 100%;
          text-align: center;
          color: var(--text-tertiary);
        }

        .empty-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 64px;
          height: 64px;
          background: var(--bg-surface);
          border-radius: var(--radius-lg);
          margin-bottom: var(--space-3);
        }

        .empty-messages p {
          font-size: var(--text-sm);
          margin-bottom: var(--space-1);
        }

        .empty-messages .hint {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Message */
        .message {
          display: flex;
          gap: var(--space-3);
          margin-bottom: var(--space-3);
        }

        .message-avatar {
          width: 32px;
          height: 32px;
          display: flex;
          align-items: center;
          justify-content: center;
          border-radius: var(--radius-full);
          flex-shrink: 0;
          visibility: hidden;
        }

        .message-avatar.visible {
          visibility: visible;
        }

        .message.human .message-avatar {
          background: var(--bg-elevated);
          color: var(--text-secondary);
        }

        .message.agent .message-avatar {
          background: rgba(37, 194, 160, 0.15);
          color: var(--color-primary);
        }

        .message-body {
          flex: 1;
          min-width: 0;
        }

        .message-meta {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          margin-bottom: var(--space-1);
        }

        .sender-name {
          font-size: var(--text-sm);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
        }

        .kind-badge {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          padding: 2px var(--space-2);
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
        }

        .message-time {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          margin-left: auto;
        }

        .message-content {
          font-size: var(--text-sm);
          color: var(--text-primary);
          line-height: 1.6;
          word-break: break-word;
          padding: var(--space-3);
          background: var(--bg-surface);
          border-radius: var(--radius-lg);
          border: 1px solid var(--border-subtle);
        }

        /* Markdown styles */
        .message-content h2 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin: 0 0 var(--space-3) 0;
          padding-bottom: var(--space-2);
          border-bottom: 1px solid var(--border-subtle);
        }

        .message-content h3 {
          font-size: var(--text-base);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin: var(--space-4) 0 var(--space-2) 0;
        }

        .message-content p {
          margin: 0 0 var(--space-2) 0;
        }

        .message-content p:last-child {
          margin-bottom: 0;
        }

        .message-content ul, .message-content ol {
          margin: var(--space-2) 0;
          padding-left: var(--space-5);
        }

        .message-content li {
          margin: var(--space-1) 0;
        }

        .message-content pre {
          background: var(--bg-base);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
          padding: var(--space-3);
          overflow-x: auto;
          margin: var(--space-2) 0;
        }

        .message-content pre code {
          background: none;
          padding: 0;
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          color: var(--text-primary);
        }

        .message-content .inline-code {
          background: var(--bg-elevated);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          color: var(--color-primary);
        }

        .message-content a {
          color: var(--color-primary);
          text-decoration: none;
        }

        .message-content a:hover {
          text-decoration: underline;
        }

        .message-content details {
          margin: var(--space-3) 0;
          padding: var(--space-2);
          background: var(--bg-base);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
        }

        .message-content summary {
          cursor: pointer;
          font-weight: var(--font-medium);
          color: var(--text-secondary);
          padding: var(--space-1);
        }

        .message-content summary:hover {
          color: var(--text-primary);
        }

        .message-content strong {
          font-weight: var(--font-semibold);
          color: var(--text-primary);
        }

        .message-content hr {
          border: none;
          border-top: 1px solid var(--border-subtle);
          margin: var(--space-4) 0;
        }

        .message.human .message-content {
          border-left: 2px solid var(--color-info);
        }

        .message.agent .message-content {
          border-left: 2px solid var(--color-primary);
        }

        .message-footer {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          margin-top: var(--space-1);
          padding-left: var(--space-3);
        }

        .message-seq {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
        }

        .delivery-status {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .delivery-status.pending {
          color: var(--color-warning);
        }

        /* Input Area */
        .input-area {
          padding: var(--space-4);
          background: var(--bg-surface);
          border-top: 1px solid var(--border-subtle);
        }

        /* Workspace toggle button in input row */
        .workspace-toggle {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 40px;
          height: 40px;
          padding: 0;
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
          flex-shrink: 0;
        }

        .workspace-toggle:hover {
          color: var(--text-secondary);
          border-color: var(--border-default);
          background: var(--bg-hover);
        }

        .workspace-toggle.has-workspace {
          color: var(--color-primary);
          border-color: var(--color-primary);
          background: rgba(37, 194, 160, 0.15);
        }

        .workspace-toggle.has-workspace:hover {
          background: rgba(37, 194, 160, 0.25);
        }

        .workspace-input-row {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          margin-bottom: var(--space-2);
        }

        .workspace-input {
          flex: 1;
          padding: var(--space-2);
          background: var(--bg-base);
          color: var(--text-primary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-sm);
          transition: all var(--transition-fast);
        }

        .workspace-input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.15);
        }

        .workspace-input::placeholder {
          color: var(--text-tertiary);
        }

        .workspace-browse {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          color: var(--text-secondary);
          font-size: var(--text-xs);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .workspace-browse:hover {
          color: var(--color-primary);
          border-color: var(--color-primary);
          background: rgba(37, 194, 160, 0.1);
        }

        .workspace-clear {
          padding: var(--space-1) var(--space-2);
          background: transparent;
          color: var(--text-tertiary);
          font-size: var(--text-xs);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .workspace-clear:hover {
          color: var(--color-danger);
          border-color: var(--color-danger);
        }

        .input-wrapper {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          background: var(--bg-base);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-lg);
          padding: var(--space-2);
          transition: border-color var(--transition-fast);
        }

        .input-wrapper:focus-within {
          border-color: var(--color-primary);
          box-shadow: 0 0 0 3px rgba(37, 194, 160, 0.1);
        }

        .kind-selector {
          height: 40px;
          padding: 0 var(--space-6) 0 var(--space-3);
          background: var(--bg-elevated);
          color: var(--text-secondary);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
          cursor: pointer;
          appearance: none;
          background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
          background-repeat: no-repeat;
          background-position: right var(--space-2) center;
          flex-shrink: 0;
          transition: all var(--transition-fast);
        }

        .kind-selector:hover {
          border-color: var(--border-default);
          color: var(--text-primary);
        }

        .kind-selector:focus {
          outline: none;
        }

        .input-wrapper textarea {
          flex: 1;
          min-height: 40px;
          max-height: 150px;
          padding: var(--space-2);
          background: transparent;
          color: var(--text-primary);
          font-family: var(--font-sans);
          font-size: var(--text-sm);
          line-height: 1.5;
          border: none;
          resize: none;
        }

        .input-wrapper textarea:focus {
          outline: none;
        }

        .input-wrapper textarea::placeholder {
          color: var(--text-tertiary);
        }

        .send-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 40px;
          height: 40px;
          background: var(--color-primary);
          color: var(--text-inverse);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
          flex-shrink: 0;
        }

        .send-btn:hover:not(:disabled) {
          background: var(--color-primary-light);
          transform: translateY(-1px);
        }

        .send-btn:disabled {
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          cursor: not-allowed;
        }

        .input-hint {
          margin-top: var(--space-2);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          text-align: center;
        }

        .input-hint kbd {
          padding: 2px 6px;
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
          font-family: var(--font-mono);
          font-size: 10px;
        }

        /* Inline Approval UI */
        .inline-approval {
          margin-top: var(--space-3);
          padding: var(--space-3);
          background: var(--bg-elevated);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
        }

        .approval-notes-input {
          width: 100%;
          padding: var(--space-2);
          background: var(--bg-base);
          color: var(--text-primary);
          font-size: var(--text-sm);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-sm);
          margin-bottom: var(--space-2);
        }

        .approval-notes-input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.15);
        }

        .approval-notes-input::placeholder {
          color: var(--text-tertiary);
        }

        .approval-actions {
          display: flex;
          gap: var(--space-2);
          justify-content: flex-end;
        }

        .approve-btn, .reject-btn {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          padding: var(--space-2) var(--space-3);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .approve-btn {
          background: var(--color-success);
          color: var(--text-inverse);
        }

        .approve-btn:hover {
          filter: brightness(1.1);
          transform: translateY(-1px);
        }

        .reject-btn {
          background: var(--bg-surface);
          color: var(--color-danger);
          border: 1px solid var(--color-danger);
        }

        .reject-btn:hover {
          background: var(--color-danger);
          color: var(--text-inverse);
        }

        .approval-handled {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          color: var(--text-tertiary);
          font-size: var(--text-sm);
        }

        .approval-handled svg {
          color: var(--color-success);
        }

        /* Truncation notice */
        .truncation-notice {
          margin-top: var(--space-2);
          padding-top: var(--space-2);
          border-top: 1px dashed var(--border-subtle);
        }

        .expand-btn {
          display: inline-flex;
          align-items: center;
          gap: var(--space-1);
          padding: var(--space-1) var(--space-2);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          color: var(--color-primary);
          background: rgba(37, 194, 160, 0.1);
          border: 1px solid transparent;
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .expand-btn:hover {
          background: rgba(37, 194, 160, 0.2);
          border-color: var(--color-primary);
        }

        /* Files Created Section */
        .files-created-section {
          margin-top: var(--space-3);
        }

        .files-toggle-btn {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          width: 100%;
          padding: var(--space-2) var(--space-3);
          background: var(--bg-base);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          color: var(--text-secondary);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .files-toggle-btn:hover {
          background: var(--bg-hover);
          border-color: var(--border-default);
        }

        .files-toggle-btn.expanded {
          border-bottom-left-radius: 0;
          border-bottom-right-radius: 0;
          border-bottom-color: transparent;
        }

        .files-toggle-btn svg {
          color: var(--color-primary);
          flex-shrink: 0;
        }

        .toggle-chevron {
          margin-left: auto;
          font-size: 10px;
          color: var(--text-tertiary);
        }

        .workspace-badge {
          display: inline-flex;
          align-items: center;
          gap: var(--space-1);
          padding: 2px var(--space-2);
          font-size: var(--text-xs);
          font-weight: var(--font-normal);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
        }

        .workspace-badge svg {
          color: var(--text-tertiary);
          width: 12px;
          height: 12px;
        }

        .files-list {
          margin: 0;
          padding: var(--space-2);
          list-style: none;
          background: var(--bg-base);
          border: 1px solid var(--border-subtle);
          border-top: none;
          border-bottom-left-radius: var(--radius-md);
          border-bottom-right-radius: var(--radius-md);
          max-height: 300px;
          overflow-y: auto;
        }

        .file-item {
          padding: var(--space-1) var(--space-2);
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          border-radius: var(--radius-sm);
          transition: background var(--transition-fast);
        }

        .file-item:hover {
          background: var(--bg-hover);
        }

        .file-item a {
          display: block;
          color: var(--color-info);
          text-decoration: none;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .file-item a:hover {
          text-decoration: underline;
          color: var(--color-primary);
        }

        /* Running Status Animation */
        @keyframes spin {
          from {
            transform: rotate(0deg);
          }
          to {
            transform: rotate(360deg);
          }
        }

        @keyframes pulse-border {
          0%, 100% {
            border-color: var(--color-primary);
            box-shadow: 0 0 8px rgba(37, 194, 160, 0.3);
          }
          50% {
            border-color: var(--color-success);
            box-shadow: 0 0 16px rgba(16, 185, 129, 0.4);
          }
        }

        .spinner-icon {
          animation: spin 1s linear infinite;
        }

        .message.running-status {
          animation: pulse-border 2s ease-in-out infinite;
          border-left: 3px solid var(--color-primary);
        }

        .message.running-status .message-content {
          background: linear-gradient(135deg, rgba(37, 194, 160, 0.05), rgba(16, 185, 129, 0.02));
        }

        .kind-badge.running {
          color: var(--color-primary);
          background: rgba(37, 194, 160, 0.15);
        }

        .kind-badge.running svg {
          color: var(--color-primary);
        }
      `})]}):null};class rd{constructor(){Oe(this,"ws",null);Oe(this,"wsUrl",null);Oe(this,"isConnecting",!1);Oe(this,"reconnectTimeout",null);Oe(this,"reconnectAttempts",0);Oe(this,"maxReconnectAttempts",10);Oe(this,"connectionState","disconnected");Oe(this,"stateListeners",new Set);Oe(this,"messageHandlers",new Set);Oe(this,"batchHandlers",new Set);Oe(this,"errorHandlers",new Set);Oe(this,"taskStreamHandlers",new Set);Oe(this,"subscriptions",new Map);Oe(this,"hookCount",0)}getState(){return{isConnected:this.connectionState==="connected",connectionState:this.connectionState,reconnectAttempts:this.reconnectAttempts}}subscribeToState(t){return this.stateListeners.add(t),t(this.connectionState,this.reconnectAttempts),()=>this.stateListeners.delete(t)}setConnectionState(t){this.connectionState=t,this.stateListeners.forEach(n=>n(t,this.reconnectAttempts))}registerHook(t,n,r){this.hookCount++,console.log(`[WebSocketService] Hook registered, count: ${this.hookCount}`);const i=t?a=>t(a):null,l=n?a=>n(a):null,o=r?a=>r(a):null;return i&&this.messageHandlers.add(i),l&&this.batchHandlers.add(l),o&&this.errorHandlers.add(o),()=>{this.hookCount--,console.log(`[WebSocketService] Hook unregistered, count: ${this.hookCount}`),i&&this.messageHandlers.delete(i),l&&this.batchHandlers.delete(l),o&&this.errorHandlers.delete(o),this.hookCount===0&&(console.log("[WebSocketService] All hooks unregistered, closing connection"),this.disconnect())}}connect(t,n,r=10){this.maxReconnectAttempts=r;const i=`${t}?instance_id=${n}`;if(this.ws&&this.ws.readyState===WebSocket.OPEN&&this.wsUrl===i){console.log("[WebSocketService] Already connected, skipping");return}if(this.isConnecting){console.log("[WebSocketService] Already connecting, skipping");return}if(this.ws&&this.ws.readyState===WebSocket.CONNECTING){console.log("[WebSocketService] Connection pending, skipping");return}this.ws&&this.wsUrl!==i&&(console.log("[WebSocketService] URL changed, closing old connection"),this.ws.close(),this.ws=null),this.isConnecting=!0,this.wsUrl=i,console.log(`[WebSocketService] Creating new WebSocket to ${i}`),this.setConnectionState(this.reconnectAttempts>0?"reconnecting":"connecting");try{this.ws=new WebSocket(i),this.ws.onopen=()=>{console.log("[WebSocketService] Connected"),this.isConnecting=!1,this.reconnectAttempts=0,this.setConnectionState("connected"),this.subscriptions.forEach((l,o)=>{this.subscribe(o,l)})},this.ws.onmessage=l=>{try{const o=JSON.parse(l.data);this.handleEvent(o)}catch(o){console.error("[WebSocketService] Failed to parse message:",o)}},this.ws.onerror=l=>{console.error("[WebSocketService] Error:",l),this.isConnecting=!1},this.ws.onclose=()=>{if(console.log("[WebSocketService] Disconnected"),this.isConnecting=!1,this.setConnectionState("disconnected"),this.hookCount>0&&this.reconnectAttempts<this.maxReconnectAttempts){const l=this.getBackoffDelay(this.reconnectAttempts);console.log(`[WebSocketService] Reconnecting in ${l}ms (attempt ${this.reconnectAttempts+1}/${this.maxReconnectAttempts})`),this.reconnectTimeout=setTimeout(()=>{this.reconnectAttempts++,this.connect(t,n,r)},l)}}}catch(l){console.error("[WebSocketService] Failed to connect:",l),this.isConnecting=!1,this.setConnectionState("disconnected")}}disconnect(){this.reconnectTimeout&&(clearTimeout(this.reconnectTimeout),this.reconnectTimeout=null),this.ws&&(this.ws.close(),this.ws=null),this.wsUrl=null,this.reconnectAttempts=0,this.subscriptions.clear(),this.setConnectionState("disconnected")}send(t){this.ws&&this.ws.readyState===WebSocket.OPEN?this.ws.send(JSON.stringify(t)):console.warn("[WebSocketService] Not connected, cannot send")}handleEvent(t){switch(t.type){case"message":t.data&&this.messageHandlers.forEach(n=>n(t.data));break;case"batch":if(t.data){const n=t.data;this.batchHandlers.forEach(r=>r(n)),n.messages.forEach(r=>{this.messageHandlers.forEach(i=>i(r))})}break;case"error":t.data&&this.errorHandlers.forEach(n=>n(t.data)),console.error("[WebSocketService] Error event:",t.data);break;case"pong":break;case"task_stream":if(t.data){const n=t.data;console.log("[WebSocketService] Task stream event:",n.stream_type,n.task_id),this.taskStreamHandlers.forEach(r=>r(n))}break;default:console.log("[WebSocketService] Unknown event:",t.type)}}getBackoffDelay(t,n=1e3,r=3e4){const i=Math.min(n*Math.pow(2,t),r),l=i*Math.random()*.3;return Math.round(i+l)}subscribe(t,n=0){this.subscriptions.set(t,n),this.send({type:"subscribe",timestamp:Date.now(),data:{thread_id:t,from_seq:n}})}unsubscribe(t){this.subscriptions.delete(t)}acknowledge(t,n){const r=this.subscriptions.get(t)||0;n>r&&this.subscriptions.set(t,n),this.send({type:"ack",timestamp:Date.now(),data:{thread_id:t,ack_seq:n}})}ping(){this.send({type:"ping",timestamp:Date.now()})}subscribeToTaskStream(t){return this.taskStreamHandlers.add(t),console.log("[WebSocketService] Task stream handler registered, count:",this.taskStreamHandlers.size),()=>{this.taskStreamHandlers.delete(t),console.log("[WebSocketService] Task stream handler unregistered, count:",this.taskStreamHandlers.size)}}}function Tk(){return typeof window<"u"?(window.__AILANG_WS_SERVICE__?console.log("[WebSocketService] Reusing existing singleton instance"):(console.log("[WebSocketService] Creating new singleton instance"),window.__AILANG_WS_SERVICE__=new rd),window.__AILANG_WS_SERVICE__):new rd}const ct=Tk();function Lk(e){return ct.subscribeToState(e)}const Pk=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,maxReconnectAttempts:l=10})=>{const[o,a]=z.useState(ct.getState().isConnected),[u,c]=z.useState(null),d=z.useRef(n),p=z.useRef(r),f=z.useRef(i);z.useEffect(()=>{d.current=n},[n]),z.useEffect(()=>{p.current=r},[r]),z.useEffect(()=>{f.current=i},[i]),z.useEffect(()=>{const m=N=>{d.current&&d.current(N)},v=N=>{p.current&&p.current(N)},x=N=>{f.current&&f.current(N)},b=ct.registerHook(m,v,x);return ct.connect(e,t,l),b},[e,t,l]),z.useEffect(()=>ct.subscribeToState((v,x)=>{a(v==="connected"),x>=l?c("Connection lost. Please refresh the page."):c(null)}),[l]),z.useEffect(()=>{if(!o)return;const m=setInterval(()=>{ct.ping()},3e4);return()=>clearInterval(m)},[o]);const h=z.useCallback((m,v=0)=>{ct.subscribe(m,v)},[]),k=z.useCallback(m=>{ct.unsubscribe(m)},[]),w=z.useCallback((m,v)=>{ct.acknowledge(m,v)},[]),I=z.useCallback(()=>{ct.ping()},[]);return{isConnected:o,connectionError:u,subscribe:h,unsubscribe:k,acknowledge:w,ping:I}},Ik=({connected:e})=>s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?s.jsxs(s.Fragment,{children:[s.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),s.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):s.jsxs(s.Fragment,{children:[s.jsx("circle",{cx:"12",cy:"12",r:"10"}),s.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),s.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),zk=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=z.useState([]),[o,a]=z.useState(null),[u,c]=z.useState(new Map),[d,p]=z.useState(new Map),[f,h]=z.useState([]),[k,w]=z.useState(!1),[I,m]=z.useState(""),{isConnected:v,subscribe:x,acknowledge:b}=Pk({url:e,instanceId:t,onMessage:N,onBatch:S});function N(g){const L={id:g.id,thread_id:g.thread_id,message_seq:g.message_seq,created_at:g.created_at,from_type:g.from_type,from_id:g.from_id,to_type:g.to_type,to_id:g.to_id,kind:g.kind,subject:g.subject,content:g.content,metadata_json:g.metadata_json,delivery_state:"visible",business_state:"open"};c(R=>{const y=R.get(L.thread_id)||[];return y.find(J=>J.id===L.id)?R:new Map(R).set(L.thread_id,[...y,L].sort((J,he)=>J.message_seq-he.message_seq))}),L.thread_id!==o&&p(R=>{const y=R.get(L.thread_id)||0;return new Map(R).set(L.thread_id,y+1)}),b(L.thread_id,L.message_seq)}function S(g){g.messages.forEach(L=>{N(L)})}const C=z.useCallback(g=>{if(a(g),p(L=>{const R=new Map(L);return R.delete(g),R}),v){const L=u.get(g)||[],R=L.length>0?Math.max(...L.map(y=>y.message_seq)):0;x(g,R)}},[v,x,u]),P=z.useCallback(async(g,L,R)=>{if(!o)return;const y=R?JSON.stringify({workspace:R}):void 0;try{const J=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:L,content:g,metadata_json:y})});if(!J.ok){console.error("Failed to send message:",await J.text());return}const he=await J.json();c(ee=>{const ye=ee.get(o)||[];return ye.find(je=>je.id===he.id)?ee:new Map(ee).set(o,[...ye,he])})}catch(J){console.error("Error sending message:",J)}},[o,t]);z.useEffect(()=>{(async()=>{try{const L=await fetch("/api/threads");if(!L.ok){console.error("Failed to fetch threads:",await L.text());return}const R=await L.json();l(R),R.length>0&&!o&&a(R[0].id)}catch(L){console.error("Error fetching threads:",L)}})()},[]),z.useEffect(()=>{if(!o)return;const g=o;(async()=>{try{const R=await fetch(`/api/messages?thread_id=${g}`);if(!R.ok){console.error("Failed to fetch messages:",await R.text());return}const y=await R.json();c(J=>{const he=J.get(g)||[],ee=y?[...y]:[];for(const ye of he)ee.find(je=>je.id===ye.id)||ee.push(ye);return ee.sort((ye,je)=>ye.message_seq-je.message_seq),new Map(J).set(g,ee)})}catch(R){console.error("Error fetching messages:",R)}})()},[o]);const D=z.useRef(null);z.useEffect(()=>{n&&n!==D.current&&i.length>0&&(i.some(L=>L.id===n)&&(D.current=n,a(n),p(L=>{const R=new Map(L);return R.delete(n),R})),r&&r())},[n,i,r]);const A=z.useCallback(async g=>{try{const L=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:g,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!L.ok){console.error("Failed to create thread:",await L.text());return}const R=await L.json();l(y=>[R,...y]),a(R.id)}catch(L){console.error("Error creating thread:",L)}},[t]),j=z.useCallback(async()=>{try{const g=await fetch("/api/agents");if(!g.ok){console.error("Failed to fetch agents:",await g.text());return}const L=await g.json();h(L.running||[])}catch(g){console.error("Error fetching agents:",g)}},[]);z.useEffect(()=>{j();const g=setInterval(j,5e3);return()=>clearInterval(g)},[j]);const E=z.useCallback(async()=>{if(I.trim())try{const g=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:I.trim()})});if(!g.ok){const R=await g.text();console.error("Failed to launch agent:",R),alert(`Failed to launch agent: ${R}`);return}const L=await g.json();h(R=>[...R,L]),m(""),w(!1)}catch(g){console.error("Error launching agent:",g)}},[I]),U=z.useCallback(async g=>{try{const L=await fetch(`/api/agents/${g}`,{method:"DELETE"});if(!L.ok){console.error("Failed to stop agent:",await L.text());return}h(R=>R.filter(y=>y.instance_id!==g))}catch(L){console.error("Error stopping agent:",L)}},[]),V=z.useCallback(async g=>{if(o)try{const L=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:g})});if(!L.ok){console.error("Failed to update workspace:",await L.text());return}const R=await L.json();l(y=>y.map(J=>J.id===o?R:J))}catch(L){console.error("Error updating workspace:",L)}},[o]),W=z.useCallback(async g=>{try{const L=await fetch(`/api/threads/${g}`,{method:"DELETE"});if(!L.ok){console.error("Failed to delete thread:",await L.text());return}l(R=>R.filter(y=>y.id!==g)),c(R=>{const y=new Map(R);return y.delete(g),y}),p(R=>{const y=new Map(R);return y.delete(g),y}),o===g&&a(null)}catch(L){console.error("Error deleting thread:",L)}},[o]),G=z.useCallback(async(g,L)=>{try{const R=await fetch(`/api/threads/${g}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:L})});if(!R.ok){console.error("Failed to rename thread:",await R.text());return}const y=await R.json();l(J=>J.map(he=>he.id===g?y:he))}catch(R){console.error("Error renaming thread:",R)}},[]),oe=z.useCallback(async(g,L)=>{try{const R=await fetch(`/api/approvals/${g}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:L})});if(!R.ok){const y=await R.text();console.error("Failed to approve request:",y),alert(`Failed to approve: ${y}`);return}console.log("Approval approved successfully")}catch(R){console.error("Error approving request:",R)}},[]),_=z.useCallback(async(g,L)=>{try{const R=await fetch(`/api/approvals/${g}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:L})});if(!R.ok){const y=await R.text();console.error("Failed to reject request:",y),alert(`Failed to reject: ${y}`);return}console.log("Approval rejected successfully")}catch(R){console.error("Error rejecting request:",R)}},[]),B=o?u.get(o)||[]:[];return s.jsxs("div",{className:"message-center",children:[s.jsxs("div",{className:"status-bar",children:[s.jsxs("div",{className:`status-indicator ${v?"connected":"disconnected"}`,children:[s.jsx(Ik,{connected:v}),s.jsx("span",{children:v?"Connected":"Disconnected"})]}),s.jsxs("div",{className:"status-meta",children:[s.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),s.jsxs("span",{className:"agent-count",children:[f.length," agents"]}),s.jsx("button",{className:"launch-agent-btn",onClick:()=>w(!0),children:"+ Agent"})]})]}),f.length>0&&s.jsx("div",{className:"agents-bar",children:f.map(g=>s.jsxs("div",{className:"agent-chip",children:[s.jsx("span",{className:"agent-pulse"}),s.jsx("span",{className:"agent-name",children:g.instance_id}),s.jsxs("span",{className:"agent-pid",children:["PID ",g.pid]}),s.jsx("button",{className:"agent-stop-btn",onClick:()=>U(g.instance_id),title:"Stop agent",children:"×"})]},g.instance_id))}),k&&s.jsx("div",{className:"modal-overlay",onClick:()=>w(!1),children:s.jsxs("div",{className:"modal-content",onClick:g=>g.stopPropagation(),children:[s.jsx("h3",{children:"Launch New Agent"}),s.jsx("input",{type:"text",value:I,onChange:g=>m(g.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:g=>{g.key==="Enter"&&E(),g.key==="Escape"&&w(!1)}}),s.jsxs("div",{className:"modal-actions",children:[s.jsx("button",{className:"cancel-btn",onClick:()=>w(!1),children:"Cancel"}),s.jsx("button",{className:"launch-btn",onClick:E,children:"Launch"})]})]})}),s.jsxs("div",{className:"center-layout",children:[s.jsx("aside",{className:"threads-panel",children:s.jsx(Dv,{threads:i,selectedThreadId:o,onSelectThread:C,onCreateThread:A,onDeleteThread:W,onRenameThread:G,unreadCounts:d})}),s.jsx("main",{className:"conversation-panel",children:o?s.jsx(Ek,{thread:i.find(g=>g.id===o),messages:B,onSendMessage:P,onWorkspaceChange:V,onApproveRequest:oe,onRejectRequest:_}):s.jsxs("div",{className:"empty-state",children:[s.jsx("div",{className:"empty-icon",children:s.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:s.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),s.jsx("h3",{children:"Select a conversation"}),s.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),s.jsx("style",{children:`
        .message-center {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg-base);
        }

        /* Status Bar */
        .status-bar {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-2) var(--space-4);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
        }

        .status-indicator {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
        }

        .status-indicator.connected {
          color: var(--color-success);
        }

        .status-indicator.connected svg {
          filter: drop-shadow(0 0 4px var(--color-success));
        }

        .status-indicator.disconnected {
          color: var(--color-danger);
        }

        .status-indicator.disconnected svg {
          filter: drop-shadow(0 0 4px var(--color-danger));
        }

        .status-meta {
          display: flex;
          align-items: center;
          gap: var(--space-4);
        }

        .thread-count, .agent-count {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .launch-agent-btn {
          padding: var(--space-1) var(--space-2);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          color: var(--color-primary);
          background: transparent;
          border: 1px solid var(--color-primary);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .launch-agent-btn:hover {
          background: var(--color-primary);
          color: var(--text-inverse);
        }

        /* Running Agents Bar */
        .agents-bar {
          display: flex;
          flex-wrap: wrap;
          gap: var(--space-2);
          padding: var(--space-2) var(--space-4);
          background: var(--bg-elevated);
          border-bottom: 1px solid var(--border-subtle);
        }

        .agent-chip {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-surface);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          font-size: var(--text-xs);
        }

        .agent-pulse {
          width: 8px;
          height: 8px;
          background: var(--color-success);
          border-radius: var(--radius-full);
          animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.6; transform: scale(0.9); }
        }

        .agent-name {
          font-weight: var(--font-medium);
          color: var(--text-primary);
        }

        .agent-pid {
          color: var(--text-tertiary);
          font-family: var(--font-mono);
        }

        .agent-stop-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 16px;
          height: 16px;
          background: transparent;
          color: var(--text-tertiary);
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          font-size: 14px;
          line-height: 1;
          transition: all var(--transition-fast);
        }

        .agent-stop-btn:hover {
          background: var(--color-danger);
          color: var(--text-inverse);
        }

        /* Modal */
        .modal-overlay {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(0, 0, 0, 0.5);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 1000;
        }

        .modal-content {
          background: var(--bg-surface);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-lg);
          padding: var(--space-6);
          width: 400px;
          max-width: 90vw;
        }

        .modal-content h3 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin-bottom: var(--space-4);
        }

        .modal-content input {
          width: 100%;
          padding: var(--space-2) var(--space-3);
          background: var(--bg-base);
          color: var(--text-primary);
          font-size: var(--text-sm);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          margin-bottom: var(--space-4);
        }

        .modal-content input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.1);
        }

        .modal-actions {
          display: flex;
          justify-content: flex-end;
          gap: var(--space-2);
        }

        .modal-actions .cancel-btn {
          padding: var(--space-2) var(--space-4);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          color: var(--text-secondary);
          background: transparent;
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .modal-actions .cancel-btn:hover {
          background: var(--bg-hover);
        }

        .modal-actions .launch-btn {
          padding: var(--space-2) var(--space-4);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          color: var(--text-inverse);
          background: var(--color-primary);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .modal-actions .launch-btn:hover {
          background: var(--color-primary-light);
        }

        /* Layout */
        .center-layout {
          flex: 1;
          display: flex;
          overflow: hidden;
        }

        .threads-panel {
          width: 320px;
          border-right: 1px solid var(--border-subtle);
          flex-shrink: 0;
        }

        .conversation-panel {
          flex: 1;
          display: flex;
          flex-direction: column;
          overflow: hidden;
        }

        /* Empty State */
        .empty-state {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          height: 100%;
          padding: var(--space-8);
          text-align: center;
        }

        .empty-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 80px;
          height: 80px;
          background: var(--bg-surface);
          border-radius: var(--radius-xl);
          margin-bottom: var(--space-4);
          color: var(--text-tertiary);
        }

        .empty-state h3 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin-bottom: var(--space-2);
        }

        .empty-state p {
          font-size: var(--text-sm);
          color: var(--text-tertiary);
          max-width: 300px;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .threads-panel {
            width: 280px;
          }
        }

        @media (max-width: 640px) {
          .center-layout {
            flex-direction: column;
          }

          .threads-panel {
            width: 100%;
            height: 200px;
            border-right: none;
            border-bottom: 1px solid var(--border-subtle);
          }
        }
      `})]})},Be={check:s.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),s.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:s.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:s.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),s.jsx("circle",{cx:"12",cy:"5",r:"2"}),s.jsx("path",{d:"M12 7v4"})]}),dollar:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),s.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("circle",{cx:"12",cy:"12",r:"10"}),s.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:s.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),s.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),s.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},Ak=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=z.useState(!0),[a,u]=z.useState(null),[c,d]=z.useState(new Map),p=m=>{try{return JSON.parse(m)}catch{return null}},f=m=>new Date(m).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),h=m=>{const v=c.get(m)||"";n(m,v),d(new Map(c.set(m,"")))},k=m=>{const v=c.get(m)||"";if(!v.trim()){alert("Please provide a reason for rejection");return}r(m,v),d(new Map(c.set(m,"")))},w=(m,v)=>{d(new Map(c.set(m,v)))},I=e.filter(m=>m.status==="pending");return s.jsxs("div",{className:"approval-queue",children:[s.jsx("div",{className:"queue-header",children:s.jsxs("div",{className:"header-title",children:[s.jsx("h2",{children:"Approval Queue"}),s.jsxs("span",{className:"pending-count",children:[I.length," pending"]})]})}),s.jsxs("div",{className:"approvals-container",children:[I.length===0?s.jsxs("div",{className:"empty-state",children:[s.jsx("div",{className:"empty-icon",children:Be.sparkles}),s.jsx("h3",{children:"All caught up!"}),s.jsx("p",{children:"No pending approvals to review"})]}):s.jsx("div",{className:"approvals-list",children:I.map(m=>{const v=p(m.effect_delta_json),x=a===m.id;return s.jsxs("div",{className:`approval-card impact-${m.impact}`,children:[s.jsxs("div",{className:"card-header",onClick:()=>u(x?null:m.id),children:[s.jsxs("div",{className:"header-left",children:[s.jsx("div",{className:`impact-indicator ${m.impact}`}),s.jsxs("div",{className:"proposal-info",children:[s.jsx("span",{className:"proposal-text",children:m.proposal}),s.jsxs("div",{className:"proposal-meta",children:[m.thread_title&&s.jsxs("span",{className:"meta-item thread-link",onClick:b=>{b.stopPropagation(),i==null||i(m.thread_id)},title:"Go to thread",children:[Be.message,m.thread_title]}),s.jsxs("span",{className:"meta-item",children:[Be.bot,m.instance_id]}),s.jsxs("span",{className:"meta-item",children:[Be.clock,f(m.created_at)]})]})]})]}),s.jsxs("div",{className:"header-right",children:[s.jsxs("span",{className:"cost-badge",children:[Be.dollar,"$",m.estimated_cost.toFixed(2)]}),s.jsx("span",{className:`impact-badge ${m.impact}`,children:m.impact}),s.jsx("button",{className:"expand-btn",children:x?Be.chevronUp:Be.chevronDown})]})]}),x&&s.jsxs("div",{className:"card-details",children:[v&&s.jsxs("div",{className:"detail-section",children:[s.jsx("h4",{children:"Effect Details"}),s.jsxs("div",{className:"detail-grid",children:[s.jsxs("div",{className:"detail-item",children:[s.jsx("span",{className:"detail-label",children:"Capability"}),s.jsx("span",{className:"detail-value code",children:v.cap_type})]}),s.jsxs("div",{className:"detail-item",children:[s.jsx("span",{className:"detail-label",children:"Budget Delta"}),s.jsxs("span",{className:"detail-value",children:["$",v.budget_delta.toFixed(2)]})]}),v.paths.length>0&&s.jsxs("div",{className:"detail-item full-width",children:[s.jsx("span",{className:"detail-label",children:"Paths"}),s.jsx("div",{className:"paths-list",children:v.paths.map((b,N)=>s.jsxs("span",{className:"path-tag",children:[Be.folder,b]},N))})]})]})]}),s.jsxs("div",{className:"detail-section",children:[s.jsx("h4",{children:"Request Info"}),s.jsxs("div",{className:"detail-grid",children:[s.jsxs("div",{className:"detail-item",children:[s.jsx("span",{className:"detail-label",children:"Thread"}),s.jsx("span",{className:"detail-value code",children:m.thread_id})]}),s.jsxs("div",{className:"detail-item",children:[s.jsx("span",{className:"detail-label",children:"Impact Level"}),s.jsx("span",{className:`detail-value impact-text ${m.impact}`,children:m.impact.toUpperCase()})]})]})]}),s.jsxs("div",{className:"review-section",children:[s.jsx("h4",{children:"Review Notes"}),s.jsx("textarea",{value:c.get(m.id)||"",onChange:b=>w(m.id,b.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),s.jsxs("div",{className:"action-buttons",children:[s.jsxs("button",{className:"reject-btn",onClick:()=>k(m.id),children:[Be.x,"Reject"]}),s.jsxs("button",{className:"approve-btn",onClick:()=>h(m.id),children:[Be.check,"Approve"]})]})]})]})]},m.id)})}),t.length>0&&s.jsxs("div",{className:"history-section",children:[s.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[s.jsxs("h3",{children:[l?Be.chevronDown:Be.chevronUp,"Review History"]}),s.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&s.jsx("div",{className:"history-list",children:t.map(m=>{const v=a===`history-${m.id}`;return s.jsxs("div",{className:`history-card ${m.status}`,onClick:()=>u(v?null:`history-${m.id}`),children:[s.jsxs("div",{className:"history-card-header",children:[s.jsxs("div",{className:"history-status",children:[s.jsx("span",{className:`status-icon ${m.status}`,children:m.status==="approved"?Be.check:Be.x}),s.jsxs("div",{className:"history-info",children:[s.jsx("span",{className:"history-proposal",children:m.proposal}),m.thread_title&&s.jsxs("span",{className:"history-thread",onClick:x=>{x.stopPropagation(),i==null||i(m.thread_id)},title:"Go to thread",children:[Be.message,m.thread_title]})]})]}),s.jsxs("div",{className:"history-meta",children:[s.jsx("span",{className:"history-agent",children:m.instance_id}),s.jsx("span",{className:`history-badge ${m.status}`,children:m.status}),s.jsx("span",{className:"history-time",children:m.reviewed_at?f(m.reviewed_at):f(m.created_at)})]})]}),v&&s.jsxs("div",{className:"history-details",children:[s.jsxs("div",{className:"detail-row",children:[s.jsx("span",{className:"detail-label",children:"Reviewed by"}),s.jsx("span",{className:"detail-value",children:m.reviewed_by||"Unknown"})]}),s.jsxs("div",{className:"detail-row",children:[s.jsx("span",{className:"detail-label",children:"Cost"}),s.jsxs("span",{className:"detail-value",children:["$",m.estimated_cost.toFixed(2)]})]}),s.jsxs("div",{className:"detail-row",children:[s.jsx("span",{className:"detail-label",children:"Impact"}),s.jsx("span",{className:`detail-value impact-text ${m.impact}`,children:m.impact.toUpperCase()})]}),m.review_notes&&s.jsxs("div",{className:"detail-row full-width",children:[s.jsx("span",{className:"detail-label",children:"Notes"}),s.jsx("span",{className:"detail-value notes",children:m.review_notes})]})]})]},`history-${m.id}`)})})]})]}),s.jsx("style",{children:`
        .approval-queue {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg-base);
        }

        /* Header */
        .queue-header {
          padding: var(--space-4) var(--space-6);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
        }

        .header-title {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .header-title h2 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
        }

        .pending-count {
          padding: var(--space-1) var(--space-3);
          background: rgba(37, 194, 160, 0.15);
          color: var(--color-primary);
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          border-radius: var(--radius-full);
        }

        /* Container */
        .approvals-container {
          flex: 1;
          overflow-y: auto;
          padding: var(--space-4) var(--space-6);
        }

        /* Empty State */
        .empty-state {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          padding: var(--space-12);
          text-align: center;
        }

        .empty-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 80px;
          height: 80px;
          background: var(--bg-surface);
          border-radius: var(--radius-xl);
          color: var(--color-primary);
          margin-bottom: var(--space-4);
        }

        .empty-state h3 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin-bottom: var(--space-2);
        }

        .empty-state p {
          font-size: var(--text-sm);
          color: var(--text-tertiary);
        }

        /* Approvals List */
        .approvals-list {
          display: flex;
          flex-direction: column;
          gap: var(--space-4);
        }

        /* Approval Card */
        .approval-card {
          background: var(--bg-surface);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-lg);
          overflow: hidden;
          transition: all var(--transition-base);
        }

        .approval-card:hover {
          border-color: var(--border-default);
          box-shadow: var(--shadow-md);
        }

        .approval-card.impact-low {
          border-left: 3px solid var(--color-success);
        }

        .approval-card.impact-medium {
          border-left: 3px solid var(--color-warning);
        }

        .approval-card.impact-high {
          border-left: 3px solid var(--color-danger);
        }

        /* Card Header */
        .card-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-4);
          cursor: pointer;
          transition: background var(--transition-fast);
        }

        .card-header:hover {
          background: var(--bg-hover);
        }

        .header-left {
          display: flex;
          align-items: flex-start;
          gap: var(--space-3);
          flex: 1;
          min-width: 0;
        }

        .impact-indicator {
          width: 10px;
          height: 10px;
          border-radius: var(--radius-full);
          flex-shrink: 0;
          margin-top: 6px;
        }

        .impact-indicator.low {
          background: var(--color-success);
          box-shadow: 0 0 8px var(--color-success);
        }

        .impact-indicator.medium {
          background: var(--color-warning);
          box-shadow: 0 0 8px var(--color-warning);
        }

        .impact-indicator.high {
          background: var(--color-danger);
          box-shadow: 0 0 8px var(--color-danger);
          animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.7; transform: scale(1.2); }
        }

        .proposal-info {
          flex: 1;
          min-width: 0;
        }

        .proposal-text {
          display: block;
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          color: var(--text-primary);
          margin-bottom: var(--space-1);
        }

        .proposal-meta {
          display: flex;
          align-items: center;
          gap: var(--space-4);
        }

        .meta-item {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .meta-item.thread-link {
          color: var(--color-primary);
          cursor: pointer;
          padding: 2px 6px;
          background: rgba(37, 194, 160, 0.1);
          border-radius: var(--radius-sm);
          max-width: 150px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
          transition: all var(--transition-fast);
        }

        .meta-item.thread-link:hover {
          background: rgba(37, 194, 160, 0.2);
          color: var(--color-primary-light);
        }

        .header-right {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          flex-shrink: 0;
        }

        .cost-badge {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          color: var(--text-secondary);
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          border-radius: var(--radius-sm);
        }

        .impact-badge {
          padding: var(--space-1) var(--space-2);
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          text-transform: uppercase;
          letter-spacing: 0.05em;
          border-radius: var(--radius-sm);
        }

        .impact-badge.low {
          background: rgba(16, 185, 129, 0.15);
          color: var(--color-success-light);
        }

        .impact-badge.medium {
          background: rgba(245, 158, 11, 0.15);
          color: var(--color-warning-light);
        }

        .impact-badge.high {
          background: rgba(239, 68, 68, 0.15);
          color: var(--color-danger-light);
        }

        .expand-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 28px;
          height: 28px;
          background: transparent;
          color: var(--text-tertiary);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .expand-btn:hover {
          background: var(--bg-elevated);
          color: var(--text-primary);
        }

        /* Card Details */
        .card-details {
          padding: var(--space-4);
          background: var(--bg-elevated);
          border-top: 1px solid var(--border-subtle);
        }

        .detail-section {
          margin-bottom: var(--space-4);
        }

        .detail-section:last-child {
          margin-bottom: 0;
        }

        .detail-section h4 {
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          color: var(--text-secondary);
          text-transform: uppercase;
          letter-spacing: 0.05em;
          margin-bottom: var(--space-3);
        }

        .detail-grid {
          display: grid;
          grid-template-columns: repeat(2, 1fr);
          gap: var(--space-3);
        }

        .detail-item {
          display: flex;
          flex-direction: column;
          gap: var(--space-1);
        }

        .detail-item.full-width {
          grid-column: span 2;
        }

        .detail-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .detail-value {
          font-size: var(--text-sm);
          color: var(--text-primary);
        }

        .detail-value.code {
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-base);
          border-radius: var(--radius-sm);
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .detail-value.impact-text.low {
          color: var(--color-success);
        }

        .detail-value.impact-text.medium {
          color: var(--color-warning);
        }

        .detail-value.impact-text.high {
          color: var(--color-danger);
        }

        .paths-list {
          display: flex;
          flex-wrap: wrap;
          gap: var(--space-2);
        }

        .path-tag {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-base);
          color: var(--text-secondary);
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          border-radius: var(--radius-sm);
        }

        /* Review Section */
        .review-section {
          padding-top: var(--space-4);
          border-top: 1px solid var(--border-subtle);
        }

        .review-section h4 {
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          color: var(--text-secondary);
          text-transform: uppercase;
          letter-spacing: 0.05em;
          margin-bottom: var(--space-2);
        }

        .review-section textarea {
          width: 100%;
          padding: var(--space-3);
          background: var(--bg-base);
          color: var(--text-primary);
          font-family: var(--font-sans);
          font-size: var(--text-sm);
          line-height: 1.5;
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          resize: vertical;
          margin-bottom: var(--space-3);
        }

        .review-section textarea:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 3px rgba(37, 194, 160, 0.1);
        }

        .review-section textarea::placeholder {
          color: var(--text-tertiary);
        }

        .action-buttons {
          display: flex;
          justify-content: flex-end;
          gap: var(--space-2);
        }

        .reject-btn, .approve-btn {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          padding: var(--space-2) var(--space-4);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .reject-btn {
          background: transparent;
          color: var(--color-danger);
          border: 1px solid var(--color-danger);
        }

        .reject-btn:hover {
          background: var(--color-danger);
          color: white;
        }

        .approve-btn {
          background: var(--color-success);
          color: white;
        }

        .approve-btn:hover {
          background: var(--color-success-light);
          transform: translateY(-1px);
          box-shadow: 0 0 12px rgba(16, 185, 129, 0.4);
        }

        /* History Section */
        .history-section {
          margin-top: var(--space-6);
          border-top: 1px solid var(--border-subtle);
          padding-top: var(--space-4);
        }

        .history-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          cursor: pointer;
          padding: var(--space-2) 0;
          margin-bottom: var(--space-4);
        }

        .history-header h3 {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          font-size: var(--text-sm);
          font-weight: var(--font-semibold);
          color: var(--text-secondary);
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .history-header h3 svg {
          width: 14px;
          height: 14px;
        }

        .history-count {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
        }

        .history-list {
          display: flex;
          flex-direction: column;
          gap: var(--space-2);
        }

        .history-card {
          background: var(--bg-surface);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
          padding: var(--space-3);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .history-card:hover {
          background: var(--bg-hover);
          border-color: var(--border-default);
        }

        .history-card.approved {
          border-left: 3px solid var(--color-success);
        }

        .history-card.rejected {
          border-left: 3px solid var(--color-danger);
        }

        .history-card-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: var(--space-3);
        }

        .history-status {
          display: flex;
          align-items: flex-start;
          gap: var(--space-2);
          flex: 1;
          min-width: 0;
        }

        .history-info {
          display: flex;
          flex-direction: column;
          gap: 2px;
          flex: 1;
          min-width: 0;
        }

        .history-thread {
          display: inline-flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          color: var(--color-primary);
          cursor: pointer;
          max-width: fit-content;
          padding: 1px 4px;
          background: rgba(37, 194, 160, 0.1);
          border-radius: var(--radius-sm);
          transition: all var(--transition-fast);
        }

        .history-thread:hover {
          background: rgba(37, 194, 160, 0.2);
        }

        .status-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 24px;
          height: 24px;
          border-radius: var(--radius-full);
          flex-shrink: 0;
        }

        .status-icon.approved {
          background: rgba(16, 185, 129, 0.15);
          color: var(--color-success);
        }

        .status-icon.rejected {
          background: rgba(239, 68, 68, 0.15);
          color: var(--color-danger);
        }

        .history-proposal {
          font-size: var(--text-sm);
          color: var(--text-primary);
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .history-meta {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          flex-shrink: 0;
        }

        .history-agent {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
        }

        .history-badge {
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          text-transform: uppercase;
          padding: 2px var(--space-2);
          border-radius: var(--radius-sm);
        }

        .history-badge.approved {
          background: rgba(16, 185, 129, 0.15);
          color: var(--color-success);
        }

        .history-badge.rejected {
          background: rgba(239, 68, 68, 0.15);
          color: var(--color-danger);
        }

        .history-time {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .history-details {
          margin-top: var(--space-3);
          padding-top: var(--space-3);
          border-top: 1px solid var(--border-subtle);
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: var(--space-3);
        }

        .detail-row {
          display: flex;
          flex-direction: column;
          gap: var(--space-1);
        }

        .detail-row.full-width {
          grid-column: 1 / -1;
        }

        .detail-row .detail-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .detail-row .detail-value {
          font-size: var(--text-sm);
          color: var(--text-primary);
        }

        .detail-row .detail-value.notes {
          font-size: var(--text-xs);
          color: var(--text-secondary);
          background: var(--bg-elevated);
          padding: var(--space-2);
          border-radius: var(--radius-sm);
          white-space: pre-wrap;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .queue-header,
          .approvals-container {
            padding-left: var(--space-4);
            padding-right: var(--space-4);
          }

          .card-header {
            flex-direction: column;
            align-items: flex-start;
            gap: var(--space-3);
          }

          .header-right {
            width: 100%;
            justify-content: flex-start;
          }

          .detail-grid {
            grid-template-columns: 1fr;
          }

          .detail-item.full-width {
            grid-column: span 1;
          }

          .history-card-header {
            flex-direction: column;
            align-items: flex-start;
          }

          .history-meta {
            width: 100%;
            margin-top: var(--space-2);
          }

          .history-details {
            grid-template-columns: 1fr;
          }
        }
      `})]})},Rk="_indicator_1ctaf_1",Mk="_dot_1ctaf_12",Dk="_connected_1ctaf_19",Fk="_connecting_1ctaf_28",Ok="_disconnected_1ctaf_37",Bk="_pulsing_1ctaf_46",$k="_text_1ctaf_61",Dt={indicator:Rk,dot:Mk,connected:Dk,connecting:Fk,disconnected:Ok,pulsing:Bk,text:$k};function Hk(){const[e,t]=z.useState("disconnected"),[n,r]=z.useState(0);if(z.useEffect(()=>Lk((o,a)=>{t(o),r(a)}),[]),e==="connected")return s.jsx("div",{className:`${Dt.indicator} ${Dt.connected}`,title:"Connected",children:s.jsx("span",{className:Dt.dot})});const i=()=>{switch(e){case"connecting":return"Connecting...";case"reconnecting":return`Reconnecting... (${n})`;case"disconnected":return n>0?"Disconnected":"Offline";default:return"Unknown"}},l=()=>{switch(e){case"connecting":case"reconnecting":return Dt.connecting;case"disconnected":return Dt.disconnected;default:return""}};return s.jsxs("div",{className:`${Dt.indicator} ${l()}`,title:i(),children:[s.jsx("span",{className:`${Dt.dot} ${e==="connecting"||e==="reconnecting"?Dt.pulsing:""}`}),s.jsx("span",{className:Dt.text,children:i()})]})}const Uk="_container_477wx_1",Vk="_header_477wx_8",Wk="_refreshButton_477wx_21",Qk="_section_477wx_41",qk="_subSection_477wx_52",Kk="_statRow_477wx_65",Yk="_label_477wx_72",Gk="_value_477wx_77",Xk="_statusBadge_477wx_83",Jk="_workspaceName_477wx_111",Zk="_providerBadge_477wx_120",ew="_loading_477wx_142",tw="_error_477wx_148",Y={container:Uk,header:Vk,refreshButton:Wk,section:Qk,subSection:qk,statRow:Kk,label:Yk,value:Gk,statusBadge:Xk,workspaceName:Jk,providerBadge:Zk,loading:ew,error:tw};function nw({refreshTrigger:e}){const[t,n]=z.useState(null),[r,i]=z.useState(!0),[l,o]=z.useState(null);z.useEffect(()=>{a()},[e]);const a=async()=>{i(!0),o(null);try{const c=await fetch("/api/statistics");if(!c.ok)throw new Error("Failed to fetch statistics");const d=await c.json();n(d)}catch(c){o(c instanceof Error?c.message:"Unknown error")}finally{i(!1)}};if(r&&!t)return s.jsx("div",{className:Y.loading,children:"Loading statistics..."});if(l)return s.jsxs("div",{className:Y.error,children:["Error: ",l]});if(!t)return null;const u=c=>{if(c==="(no workspace)")return c;const d=c.split("/");return d[d.length-1]||c};return s.jsxs("div",{className:Y.container,children:[s.jsxs("div",{className:Y.header,children:[s.jsx("h3",{children:"Statistics"}),s.jsx("button",{onClick:a,className:Y.refreshButton,disabled:r,children:r?"Refreshing...":"Refresh"})]}),s.jsxs("div",{className:Y.section,children:[s.jsx("h4",{children:"Threads"}),s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.label,children:"Total:"}),s.jsx("span",{className:Y.value,children:t.threads.total})]}),s.jsxs("div",{className:Y.subSection,children:[s.jsx("h5",{children:"By Status"}),Object.entries(t.threads.by_status).map(([c,d])=>s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.statusBadge,"data-status":c,children:c}),s.jsx("span",{className:Y.value,children:d})]},c))]}),s.jsxs("div",{className:Y.subSection,children:[s.jsx("h5",{children:"By Workspace"}),Object.entries(t.threads.by_workspace).sort(([,c],[,d])=>d-c).slice(0,10).map(([c,d])=>s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.workspaceName,title:c,children:u(c)}),s.jsx("span",{className:Y.value,children:d})]},c))]})]}),t.coordinator&&s.jsxs("div",{className:Y.section,children:[s.jsx("h4",{children:"Coordinator Tasks"}),s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.label,children:"Total:"}),s.jsx("span",{className:Y.value,children:t.coordinator.total_tasks})]}),s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.label,children:"Pending:"}),s.jsx("span",{className:Y.value,children:t.coordinator.pending_tasks})]}),s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.label,children:"Running:"}),s.jsx("span",{className:Y.value,children:t.coordinator.running_tasks})]}),s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.label,children:"Completed:"}),s.jsx("span",{className:Y.value,children:t.coordinator.completed_tasks})]}),s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.label,children:"Failed:"}),s.jsx("span",{className:Y.value,children:t.coordinator.failed_tasks})]}),t.coordinator.total_cost>0&&s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.label,children:"Total Cost:"}),s.jsxs("span",{className:Y.value,children:["$",t.coordinator.total_cost.toFixed(4)]})]}),t.coordinator.total_tokens>0&&s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.label,children:"Total Tokens:"}),s.jsx("span",{className:Y.value,children:t.coordinator.total_tokens.toLocaleString()})]}),t.coordinator.by_provider&&Object.keys(t.coordinator.by_provider).length>0&&s.jsxs("div",{className:Y.subSection,children:[s.jsx("h5",{children:"By Provider"}),Object.entries(t.coordinator.by_provider).map(([c,d])=>s.jsxs("div",{className:Y.statRow,children:[s.jsx("span",{className:Y.providerBadge,"data-provider":c,children:c}),s.jsx("span",{className:Y.value,children:d})]},c))]})]})]})}const rw="_taskExecutionPanel_3t8rx_3",iw="_panelHeader_3t8rx_13",lw="_headerLeft_3t8rx_22",ow="_taskTitle_3t8rx_28",aw="_threadId_3t8rx_35",sw="_headerRight_3t8rx_40",uw="_statusBadge_3t8rx_46",cw="_statusPending_3t8rx_53",dw="_statusRunning_3t8rx_58",pw="_statusCompleted_3t8rx_64",fw="_statusFailed_3t8rx_69",hw="_statusApproval_3t8rx_74",mw="_cancelButton_3t8rx_85",gw="_metricsSection_3t8rx_101",vw="_resourceMetrics_3t8rx_106",xw="_resourceMetricsCompact_3t8rx_110",yw="_metricItem_3t8rx_116",kw="_metricsGrid_3t8rx_120",ww="_metricCard_3t8rx_126",Sw="_metricLabel_3t8rx_132",bw="_metricValue_3t8rx_139",_w="_metricBar_3t8rx_146",jw="_metricBarFill_3t8rx_154",Cw="_metricPeak_3t8rx_161",Nw="_metricDetail_3t8rx_162",Ew="_metricsPlaceholder_3t8rx_172",Tw="_approvalSection_3t8rx_179",Lw="_approvalHeader_3t8rx_187",Pw="_approvalIcon_3t8rx_196",Iw="_approvalType_3t8rx_201",zw="_approvalContent_3t8rx_209",Aw="_approvalDescription_3t8rx_213",Rw="_filesChanged_3t8rx_218",Mw="_toggleButton_3t8rx_222",Dw="_fileList_3t8rx_236",Fw="_diffSummary_3t8rx_257",Ow="_approvalActions_3t8rx_269",Bw="_approveButton_3t8rx_275",$w="_rejectButton_3t8rx_276",Hw="_timeout_3t8rx_313",Uw="_logSection_3t8rx_321",Vw="_logHeader_3t8rx_328",Ww="_eventCount_3t8rx_339",Qw="_streamingLog_3t8rx_344",qw="_emptyLog_3t8rx_354",Kw="_logLine_3t8rx_361",Yw="_timestamp_3t8rx_373",Gw="_icon_3t8rx_379",Xw="_toolName_3t8rx_385",Jw="_content_3t8rx_390",Zw="_logStdout_3t8rx_396",e2="_logError_3t8rx_400",t2="_logTool_3t8rx_408",n2="_logResult_3t8rx_421",r2="_logStatus_3t8rx_429",$={taskExecutionPanel:rw,panelHeader:iw,headerLeft:lw,taskTitle:ow,threadId:aw,headerRight:sw,statusBadge:uw,statusPending:cw,statusRunning:dw,statusCompleted:pw,statusFailed:fw,statusApproval:hw,cancelButton:mw,metricsSection:gw,resourceMetrics:vw,resourceMetricsCompact:xw,metricItem:yw,metricsGrid:kw,metricCard:ww,metricLabel:Sw,metricValue:bw,metricBar:_w,metricBarFill:jw,metricPeak:Cw,metricDetail:Nw,metricsPlaceholder:Ew,approvalSection:Tw,approvalHeader:Lw,approvalIcon:Pw,approvalType:Iw,approvalContent:zw,approvalDescription:Aw,filesChanged:Rw,toggleButton:Mw,fileList:Dw,diffSummary:Fw,approvalActions:Ow,approveButton:Bw,rejectButton:$w,timeout:Hw,logSection:Uw,logHeader:Vw,eventCount:Ww,streamingLog:Qw,emptyLog:qw,logLine:Kw,timestamp:Yw,icon:Gw,toolName:Xw,content:Jw,logStdout:Zw,logError:e2,logTool:t2,logResult:n2,logStatus:r2},i2=e=>{switch(e){case"turn_start":return"[START]";case"text":return">";case"tool_use":return"[TOOL]";case"tool_result":return"[RESULT]";case"turn_end":return"[END]";case"error":return"[ERR]";case"status":return"[STATUS]";default:return""}},l2=e=>{switch(e){case"error":return $.logError;case"tool_use":return $.logTool;case"tool_result":return $.logResult;case"turn_start":case"turn_end":return $.logStatus;case"status":return $.logStatus;case"text":default:return $.logStdout}},o2=e=>new Date(e).toLocaleTimeString("en-US",{hour12:!1,hour:"2-digit",minute:"2-digit",second:"2-digit"}),a2=({events:e,maxLines:t=500,autoScroll:n=!0})=>{const r=z.useRef(null),i=z.useRef(!0);z.useEffect(()=>{const o=r.current;if(!o)return;const a=()=>{const{scrollTop:u,scrollHeight:c,clientHeight:d}=o;i.current=c-u-d<50};return o.addEventListener("scroll",a),()=>o.removeEventListener("scroll",a)},[]),z.useEffect(()=>{n&&i.current&&r.current&&(r.current.scrollTop=r.current.scrollHeight)},[e,n]);const l=e.length>t?e.slice(-t):e;return s.jsxs("div",{className:$.streamingLog,ref:r,children:[e.length===0&&s.jsx("div",{className:$.emptyLog,children:"Waiting for task events..."}),l.map((o,a)=>s.jsxs("div",{className:`${$.logLine} ${l2(o.stream_type)}`,children:[s.jsx("span",{className:$.timestamp,children:o2(o.timestamp||Date.now())}),s.jsx("span",{className:$.icon,children:i2(o.stream_type)}),o.tool_name&&s.jsxs("span",{className:$.toolName,children:["[",o.tool_name,"]"]}),s.jsx("span",{className:$.content,children:o.text||o.tool_input||o.tool_output||o.status||o.error_msg||(o.stream_type==="turn_start"?`Turn ${o.turn_num}`:"")})]},`${o.timestamp}-${a}`))]})},Ro=e=>e>=1024?`${(e/1024).toFixed(1)} GB`:`${e.toFixed(0)} MB`,id=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,Oi=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}K`:e.toString(),s2=({metrics:e,compact:t=!1})=>e?t?s.jsxs("div",{className:$.resourceMetricsCompact,children:[s.jsxs("span",{className:$.metricItem,children:["CPU: ",e.cpu_percent.toFixed(0),"%"]}),s.jsxs("span",{className:$.metricItem,children:["RAM: ",Ro(e.memory_mb)]}),s.jsxs("span",{className:$.metricItem,children:["Tokens: ",Oi(e.tokens_in+e.tokens_out)]}),s.jsxs("span",{className:$.metricItem,children:["Cost: ",id(e.cost)]})]}):s.jsx("div",{className:$.resourceMetrics,children:s.jsxs("div",{className:$.metricsGrid,children:[s.jsxs("div",{className:$.metricCard,children:[s.jsx("div",{className:$.metricLabel,children:"CPU"}),s.jsxs("div",{className:$.metricValue,children:[e.cpu_percent.toFixed(1),"%"]}),s.jsx("div",{className:$.metricBar,children:s.jsx("div",{className:$.metricBarFill,style:{width:`${Math.min(100,e.cpu_percent)}%`}})}),s.jsxs("div",{className:$.metricPeak,children:["Peak: ",e.peak_cpu.toFixed(1),"%"]})]}),s.jsxs("div",{className:$.metricCard,children:[s.jsx("div",{className:$.metricLabel,children:"Memory"}),s.jsx("div",{className:$.metricValue,children:Ro(e.memory_mb)}),s.jsx("div",{className:$.metricBar,children:s.jsx("div",{className:$.metricBarFill,style:{width:`${Math.min(100,e.memory_mb/8192*100)}%`}})}),s.jsxs("div",{className:$.metricPeak,children:["Peak: ",Ro(e.peak_memory)]})]}),s.jsxs("div",{className:$.metricCard,children:[s.jsx("div",{className:$.metricLabel,children:"Tokens"}),s.jsx("div",{className:$.metricValue,children:Oi(e.tokens_in+e.tokens_out)}),s.jsxs("div",{className:$.metricDetail,children:[s.jsxs("span",{children:["In: ",Oi(e.tokens_in)]}),s.jsxs("span",{children:["Out: ",Oi(e.tokens_out)]})]})]}),s.jsxs("div",{className:$.metricCard,children:[s.jsx("div",{className:$.metricLabel,children:"Cost"}),s.jsx("div",{className:$.metricValue,children:id(e.cost)}),s.jsx("div",{className:$.metricDetail,children:s.jsx("span",{children:"Running total"})})]})]})}):s.jsx("div",{className:$.resourceMetrics,children:s.jsx("div",{className:$.metricsPlaceholder,children:"No metrics available"})}),u2=({taskId:e,onEvent:t})=>{const[n,r]=z.useState({events:[],metrics:null,status:"pending",pendingApproval:null,isConnected:!1,error:null}),i=z.useRef(t),l=z.useRef(e);z.useEffect(()=>{i.current=t},[t]),z.useEffect(()=>{l.current=e,e&&(r(c=>({...c,events:[]})),fetch(`/api/coordinator/tasks/${e}/events`).then(c=>c.json()).then(c=>{console.log("[useTaskStream] Loaded historical events:",(c==null?void 0:c.length)||0),c&&c.length>0&&r(d=>({...d,events:c.map(p=>({...p,type:"task_stream"}))}))}).catch(c=>{console.error("[useTaskStream] Failed to fetch historical events:",c)}))},[e]),z.useEffect(()=>{const c=ct.subscribeToState(p=>{r(f=>({...f,isConnected:p==="connected"}))}),d=ct.subscribeToTaskStream(p=>{const f=l.current;if(f&&p.task_id!==f)return;console.log("[useTaskStream] Received:",p.stream_type,"for task",p.task_id);const h={...p,type:"task_stream"};r(k=>({...k,events:[...k.events,h].slice(-500)})),p.stream_type==="status"&&(r(k=>({...k,metrics:{task_id:p.task_id,cpu_percent:0,memory_mb:0,tokens_in:p.tokens_in||0,tokens_out:p.tokens_out||0,cost:p.cost||0,peak_cpu:0,peak_memory:0,updated_at:p.timestamp||Date.now()}})),p.status&&r(k=>({...k,status:p.status}))),p.stream_type==="turn_start"&&r(k=>({...k,status:"running"})),i.current&&i.current(p)});return()=>{c(),d()}},[]);const o=z.useCallback(()=>{r(c=>({...c,events:[]}))},[]),a=z.useCallback(async()=>{if(n.pendingApproval)try{const c=await fetch(`/api/coordinator/approve/${n.pendingApproval.id}`,{method:"POST",headers:{"Content-Type":"application/json"}});if(!c.ok)throw new Error(await c.text());r(d=>({...d,pendingApproval:null,status:"running"}))}catch(c){console.error("Failed to approve:",c),r(d=>({...d,error:"Failed to approve"}))}},[n.pendingApproval]),u=z.useCallback(async()=>{if(n.pendingApproval)try{const c=await fetch(`/api/coordinator/reject/${n.pendingApproval.id}`,{method:"POST",headers:{"Content-Type":"application/json"}});if(!c.ok)throw new Error(await c.text());r(d=>({...d,pendingApproval:null,status:"rejected"}))}catch(c){console.error("Failed to reject:",c),r(d=>({...d,error:"Failed to reject"}))}},[n.pendingApproval]);return{...n,clearEvents:o,approve:a,reject:u}},c2=e=>{switch(e){case"pending":return{label:"Pending",className:$.statusPending};case"running":return{label:"Running",className:$.statusRunning};case"completed":return{label:"Completed",className:$.statusCompleted};case"failed":return{label:"Failed",className:$.statusFailed};case"approval_pending":return{label:"Awaiting Approval",className:$.statusApproval};default:return{label:e,className:""}}},d2=({taskId:e,threadId:t,events:n,metrics:r,pendingApproval:i,status:l,onApprove:o,onReject:a,onCancel:u})=>{const c=u2({taskId:e}),d=n??c.events,p=r??c.metrics,f=i??c.pendingApproval,h=l??c.status,[k,w]=z.useState(!1),[I,m]=z.useState(!1),v=c2(h),x=z.useCallback(async()=>{if(f){w(!0);try{o?await o(f.id):await c.approve()}finally{w(!1)}}},[f,o,c]),b=z.useCallback(async()=>{if(f){w(!0);try{a?await a(f.id):await c.reject()}finally{w(!1)}}},[f,a,c]);return z.useEffect(()=>{if(f&&h==="approval_pending")try{const N=new Audio("/notification.mp3");N.volume=.3,N.play().catch(()=>{})}catch{}},[f,h]),s.jsxs("div",{className:$.taskExecutionPanel,children:[s.jsxs("div",{className:$.panelHeader,children:[s.jsxs("div",{className:$.headerLeft,children:[s.jsxs("h3",{className:$.taskTitle,children:["Task: ",e]}),t&&s.jsxs("span",{className:$.threadId,children:["Thread: ",t]})]}),s.jsxs("div",{className:$.headerRight,children:[s.jsx("span",{className:`${$.statusBadge} ${v.className}`,children:v.label}),h==="running"&&u&&s.jsx("button",{className:$.cancelButton,onClick:u,children:"Cancel"})]})]}),s.jsx("div",{className:$.metricsSection,children:s.jsx(s2,{metrics:p})}),f&&s.jsxs("div",{className:$.approvalSection,children:[s.jsxs("div",{className:$.approvalHeader,children:[s.jsx("span",{className:$.approvalIcon,children:"Approval Required"}),s.jsx("span",{className:$.approvalType,children:f.type})]}),s.jsxs("div",{className:$.approvalContent,children:[s.jsx("p",{className:$.approvalDescription,children:f.description}),f.files_changed&&f.files_changed.length>0&&s.jsxs("div",{className:$.filesChanged,children:[s.jsxs("button",{className:$.toggleButton,onClick:()=>m(!I),children:[I?"Hide":"Show"," Changed Files (",f.files_changed.length,")"]}),I&&s.jsx("ul",{className:$.fileList,children:f.files_changed.map((N,S)=>s.jsx("li",{children:N},S))})]}),I&&f.diff_summary&&s.jsx("pre",{className:$.diffSummary,children:f.diff_summary}),s.jsxs("div",{className:$.approvalActions,children:[s.jsx("button",{className:$.approveButton,onClick:x,disabled:k,children:k?"Processing...":"Approve"}),s.jsx("button",{className:$.rejectButton,onClick:b,disabled:k,children:k?"Processing...":"Reject"})]}),f.timeout_at&&s.jsxs("div",{className:$.timeout,children:["Expires: ",new Date(f.timeout_at).toLocaleTimeString()]})]})]}),s.jsxs("div",{className:$.logSection,children:[s.jsxs("div",{className:$.logHeader,children:[s.jsx("span",{children:"Live Output"}),s.jsxs("span",{className:$.eventCount,children:[d.length," events"]})]}),s.jsx(a2,{events:d})]})]})},p2=s.jsx("img",{src:"/logo.png",alt:"AILANG",width:"28",height:"28"}),f2=()=>{const[e,t]=z.useState({type:"overview"}),[n,r]=z.useState(null),[i,l]=z.useState([]),[o,a]=z.useState([]),[u,c]=z.useState(!1),[d,p]=z.useState(""),[f,h]=z.useState("..."),w=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;z.useEffect(()=>{(async()=>{try{const P=await fetch("/api/version");if(P.ok){const D=await P.json();h(D.version||"dev")}}catch(P){console.error("Error fetching version:",P),h("dev")}})()},[]),z.useEffect(()=>{const C=async()=>{try{const D=await fetch("/api/hierarchy");if(D.ok){const A=await D.json();r(A)}}catch(D){console.error("Error fetching hierarchy:",D)}};C();const P=setInterval(C,5e3);return()=>clearInterval(P)},[]),z.useEffect(()=>{const C=async()=>{try{const D=await fetch("/api/approvals?status=pending");if(D.ok){const U=await D.json();l(U)}const[A,j]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),E=[];if(A.ok){const U=await A.json();E.push(...U)}if(j.ok){const U=await j.json();E.push(...U)}E.sort((U,V)=>{const W=U.reviewed_at?new Date(U.reviewed_at).getTime():0;return(V.reviewed_at?new Date(V.reviewed_at).getTime():0)-W}),a(E)}catch(D){console.error("Error fetching approvals:",D)}};C();const P=setInterval(C,5e3);return()=>clearInterval(P)},[]);const I=async(C,P)=>{try{const D=await fetch(`/api/approvals/${C}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:P})});if(!D.ok){console.error("Failed to approve:",await D.text());return}const A=i.find(j=>j.id===C);if(A){const j={...A,status:"approved",reviewed_by:"user",review_notes:P,reviewed_at:Date.now()};a(E=>[j,...E])}l(j=>j.filter(E=>E.id!==C))}catch(D){console.error("Error approving:",D)}},m=async(C,P)=>{try{const D=await fetch(`/api/approvals/${C}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:P})});if(!D.ok){console.error("Failed to reject:",await D.text());return}const A=i.find(j=>j.id===C);if(A){const j={...A,status:"rejected",reviewed_by:"user",review_notes:P,reviewed_at:Date.now()};a(E=>[j,...E])}l(j=>j.filter(E=>E.id!==C))}catch(D){console.error("Error rejecting:",D)}},v=()=>{var P,D,A,j;const C=[{label:"All Agents",onClick:()=>t({type:"overview"})}];if(e.type==="agent"&&e.agentId&&C.push({label:e.agentId}),e.type==="thread"&&e.threadId){e.agentId&&C.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})});const E=(P=n==null?void 0:n.root.children)==null?void 0:P.find(V=>V.id===e.agentId),U=(D=E==null?void 0:E.children)==null?void 0:D.find(V=>V.id===e.threadId);C.push({label:(U==null?void 0:U.label)||"Thread"})}if(e.type==="task"&&e.taskId){if(e.agentId&&C.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})}),e.threadId){const E=(A=n==null?void 0:n.root.children)==null?void 0:A.find(V=>V.id===e.agentId),U=(j=E==null?void 0:E.children)==null?void 0:j.find(V=>V.id===e.threadId);C.push({label:(U==null?void 0:U.label)||"Thread",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:e.threadId})})}C.push({label:`Task ${e.taskId.slice(0,8)}...`})}return C},x=C=>{var D;const P=(D=n==null?void 0:n.root.children)==null?void 0:D.find(A=>{var j;return(j=A.children)==null?void 0:j.some(E=>E.id===C)});t({type:"thread",agentId:P==null?void 0:P.id,threadId:C})},b=async C=>{if(d.trim())try{const P=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:d.trim(),created_by_type:"human",created_by_id:"user",target_agent:C})});if(!P.ok){console.error("Failed to create thread:",await P.text());return}const D=await P.json();p(""),c(!1),t({type:"thread",agentId:C,threadId:D.id})}catch(P){console.error("Error creating thread:",P)}},N=()=>{var C,P,D;if(e.type==="overview"&&n)return s.jsxs("div",{className:"overview-container",children:[s.jsx("div",{className:"overview-main",children:s.jsx(jv,{aggregate:n.aggregate,agents:n.root.children||[],onSelectAgent:A=>t({type:"agent",agentId:A})})}),s.jsx("aside",{className:"overview-sidebar",children:s.jsx(nw,{})})]});if(e.type==="agent"&&e.agentId){const A=(C=n==null?void 0:n.root.children)==null?void 0:C.find(E=>E.id===e.agentId),j=i.filter(E=>{var U;return(U=A==null?void 0:A.children)==null?void 0:U.some(V=>V.id===E.thread_id)});return s.jsxs("div",{className:"agent-view",children:[s.jsxs("div",{className:"agent-view-header",children:[s.jsx("h2",{children:e.agentId}),s.jsxs("span",{className:"agent-thread-count",children:[((P=A==null?void 0:A.children)==null?void 0:P.length)||0," threads"]})]}),s.jsxs("div",{className:"agent-metrics-section",children:[s.jsx("h3",{children:"Agent Metrics"}),s.jsx(Ta,{scopeType:"agent",scopeId:e.agentId,title:""}),s.jsxs("div",{className:"agent-trends-grid",children:[s.jsx(jl,{scopeType:"agent",scopeId:e.agentId,period:"hour",limit:24,metric:"cost",title:"Cost (24h)"}),s.jsx(jl,{scopeType:"agent",scopeId:e.agentId,period:"hour",limit:24,metric:"tokens",title:"Tokens (24h)"})]})]}),s.jsxs("div",{className:"agent-view-content",children:[s.jsxs("div",{className:"agent-threads",children:[s.jsxs("div",{className:"threads-header",children:[s.jsx("h3",{children:"Threads"}),s.jsx("button",{className:"new-thread-btn",onClick:()=>c(!0),title:"New thread",children:"+ New Thread"})]}),u&&s.jsxs("div",{className:"new-thread-form",children:[s.jsx("input",{type:"text",value:d,onChange:E=>p(E.target.value),onKeyDown:E=>{E.key==="Enter"&&b(e.agentId),E.key==="Escape"&&(c(!1),p(""))},placeholder:"Thread title...",autoFocus:!0}),s.jsxs("div",{className:"form-actions",children:[s.jsx("button",{onClick:()=>{c(!1),p("")},children:"Cancel"}),s.jsx("button",{className:"create-btn",onClick:()=>b(e.agentId),children:"Create"})]})]}),(D=A==null?void 0:A.children)==null?void 0:D.map(E=>s.jsxs("div",{className:"thread-card",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:E.id}),children:[s.jsx("span",{className:"thread-title",children:E.label}),E.badges&&E.badges.length>0&&s.jsx("span",{className:"thread-badges",children:E.badges.map((U,V)=>s.jsx("span",{className:`badge badge-${U.type}`,children:U.count},V))})]},E.id)),(!(A!=null&&A.children)||A.children.length===0)&&!u&&s.jsxs("div",{className:"no-threads",children:["No threads yet",s.jsx("button",{className:"start-thread-btn",onClick:()=>c(!0),children:"Start a conversation"})]})]}),j.length>0&&s.jsxs("div",{className:"agent-approvals",children:[s.jsx("h3",{children:"Pending Approvals"}),s.jsx(Ak,{approvals:j,history:[],onApprove:I,onReject:m,onNavigateToThread:x})]})]})]})}return e.type==="thread"&&e.threadId?s.jsxs("div",{className:"thread-view",children:[s.jsx("div",{className:"thread-metrics-bar",children:s.jsx(Ta,{scopeType:"thread",scopeId:e.threadId,title:"Thread Metrics",compact:!0})}),s.jsx("div",{className:"thread-messages-container",children:s.jsx(zk,{websocketUrl:w,instanceId:e.agentId||"default",initialThreadId:e.threadId,onThreadNavigated:()=>{}})})]}):e.type==="task"&&e.taskId?s.jsx("div",{className:"task-view",children:s.jsx(d2,{taskId:e.taskId,threadId:e.threadId,onCancel:()=>{e.threadId?t({type:"thread",agentId:e.agentId,threadId:e.threadId}):t({type:"overview"})}})}):s.jsx("div",{className:"empty-state",children:s.jsx("p",{children:"Select an agent or thread from the sidebar"})})},S=(i==null?void 0:i.filter(C=>C.status==="pending").length)||0;return s.jsxs("div",{className:"app",children:[s.jsxs("header",{className:"app-header",children:[s.jsxs("div",{className:"header-brand",children:[s.jsx("div",{className:"brand-logo",children:p2}),s.jsxs("div",{className:"brand-text",children:[s.jsx("h1",{children:"AILANG"}),s.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),s.jsxs("div",{className:"header-meta",children:[s.jsx(Hk,{}),S>0&&s.jsxs("span",{className:"pending-badge",title:`${S} pending approvals`,children:[S," pending"]}),s.jsxs("a",{href:"https://ailang.sunholo.com",target:"_blank",rel:"noopener noreferrer",className:"docs-link",title:"View documentation",children:[s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"}),s.jsx("polyline",{points:"15 3 21 3 21 9"}),s.jsx("line",{x1:"10",y1:"14",x2:"21",y2:"3"})]}),"Docs"]}),s.jsx("span",{className:"version-tag",children:f})]})]}),s.jsxs("div",{className:"app-body",children:[s.jsx("aside",{className:"app-sidebar",children:s.jsx(Og,{selection:e,onSelect:t})}),s.jsxs("main",{className:"app-main",children:[e.type!=="overview"&&s.jsx(Cv,{items:v()}),s.jsx("div",{className:"main-content",children:N()})]})]}),s.jsx("style",{children:`
        .app {
          display: flex;
          flex-direction: column;
          height: 100vh;
          background: var(--bg-base);
          color: var(--text-primary);
        }

        /* Header */
        .app-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          height: 52px;
          padding: 0 var(--space-4);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
          flex-shrink: 0;
        }

        .header-brand {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .brand-logo {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 36px;
          height: 36px;
        }

        .brand-logo img {
          width: 32px;
          height: 32px;
          object-fit: contain;
        }

        .brand-text h1 {
          font-family: var(--font-heading);
          font-size: var(--text-lg);
          font-weight: 800;
          letter-spacing: -0.02em;
          background: linear-gradient(135deg, var(--gradient-start), var(--gradient-end));
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
          line-height: 1;
          margin-bottom: 2px;
        }

        .brand-subtitle {
          font-family: var(--font-heading);
          font-size: 10px;
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.1em;
        }

        .header-meta {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .docs-link {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: var(--space-1) var(--space-3);
          background: rgba(231, 60, 23, 0.1);
          color: var(--color-primary-light);
          font-family: var(--font-heading);
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          text-decoration: none;
          border-radius: var(--radius-md);
          border: 1px solid rgba(231, 60, 23, 0.2);
          transition: all var(--transition-base);
        }

        .docs-link:hover {
          background: rgba(231, 60, 23, 0.2);
          border-color: rgba(231, 60, 23, 0.4);
          color: var(--color-primary-light);
        }

        .pending-badge {
          padding: var(--space-1) var(--space-2);
          background: rgba(221, 107, 32, 0.15);
          color: var(--sunholo-orange);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          border-radius: var(--radius-full);
        }

        .version-tag {
          padding: var(--space-1) var(--space-2);
          background: linear-gradient(135deg, rgba(231, 60, 23, 0.1), rgba(221, 107, 32, 0.1));
          color: var(--color-primary-light);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          border-radius: var(--radius-full);
          border: 1px solid rgba(231, 60, 23, 0.2);
        }

        /* Body Layout */
        .app-body {
          display: flex;
          flex: 1;
          overflow: hidden;
        }

        .app-sidebar {
          flex-shrink: 0;
          overflow: hidden;
        }

        .app-main {
          flex: 1;
          display: flex;
          flex-direction: column;
          overflow: hidden;
          background: var(--bg-base);
        }

        .main-content {
          flex: 1;
          overflow: auto;
        }

        /* Agent View */
        .agent-view {
          padding: 24px;
          height: 100%;
          overflow-y: auto;
        }

        .agent-view-header {
          display: flex;
          align-items: center;
          gap: 16px;
          margin-bottom: 24px;
        }

        .agent-view-header h2 {
          margin: 0;
          font-size: 24px;
          font-weight: 600;
          color: #cdd6f4;
        }

        .agent-thread-count {
          font-size: 14px;
          color: #6c7086;
        }

        .agent-view-content {
          display: flex;
          flex-direction: column;
          gap: 32px;
        }

        /* Agent Metrics Section */
        .agent-metrics-section {
          margin-bottom: 24px;
          padding: 20px;
          background: linear-gradient(135deg, rgba(59, 130, 246, 0.08), rgba(99, 102, 241, 0.04));
          border: 1px solid rgba(59, 130, 246, 0.2);
          border-radius: 12px;
        }

        .agent-metrics-section h3 {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 600;
          color: #3b82f6;
        }

        .agent-trends-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 16px;
          margin-top: 16px;
        }

        /* Thread View */
        .thread-view {
          display: flex;
          flex-direction: column;
          height: 100%;
          overflow: hidden;
        }

        .thread-metrics-bar {
          flex-shrink: 0;
          padding: 12px 16px;
          background: linear-gradient(135deg, rgba(34, 197, 94, 0.08), rgba(16, 185, 129, 0.04));
          border-bottom: 1px solid rgba(34, 197, 94, 0.2);
        }

        .thread-messages-container {
          flex: 1;
          overflow: hidden;
        }

        /* Task View */
        .task-view {
          height: 100%;
          overflow: hidden;
        }

        .agent-threads h3,
        .agent-approvals h3 {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 600;
          color: #cdd6f4;
        }

        .thread-card {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 12px 16px;
          background: #1e1e2e;
          border: 1px solid #313244;
          border-radius: 8px;
          margin-bottom: 8px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .thread-card:hover {
          border-color: #45475a;
          background: #232336;
        }

        .thread-title {
          font-size: 14px;
          color: #cdd6f4;
        }

        .thread-badges {
          display: flex;
          gap: 6px;
        }

        .badge {
          padding: 2px 8px;
          font-size: 11px;
          border-radius: 10px;
        }

        .badge-pending {
          background: rgba(245, 158, 11, 0.2);
          color: #f59e0b;
        }

        .badge-unread {
          background: rgba(59, 130, 246, 0.2);
          color: #3b82f6;
        }

        .badge-running {
          background: rgba(34, 197, 94, 0.2);
          color: #22c55e;
        }

        .no-threads {
          padding: 20px;
          text-align: center;
          color: #6c7086;
          font-size: 14px;
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 12px;
        }

        .threads-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 16px;
        }

        .threads-header h3 {
          margin: 0;
        }

        .new-thread-btn {
          padding: 6px 12px;
          background: linear-gradient(135deg, var(--gradient-start), var(--gradient-end));
          color: white;
          border: none;
          border-radius: var(--radius-md);
          font-family: var(--font-heading);
          font-size: 13px;
          font-weight: 600;
          cursor: pointer;
          transition: all 0.2s;
          box-shadow: 0 2px 8px rgba(231, 60, 23, 0.3);
        }

        .new-thread-btn:hover {
          background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));
          transform: translateY(-1px);
          box-shadow: 0 4px 12px rgba(231, 60, 23, 0.4);
        }

        .start-thread-btn {
          padding: 8px 16px;
          background: linear-gradient(135deg, var(--gradient-start), var(--gradient-end));
          color: white;
          border: none;
          border-radius: var(--radius-md);
          font-family: var(--font-heading);
          font-size: 13px;
          font-weight: 600;
          cursor: pointer;
          transition: all 0.2s;
          box-shadow: 0 2px 8px rgba(231, 60, 23, 0.3);
        }

        .start-thread-btn:hover {
          background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));
          transform: translateY(-1px);
          box-shadow: 0 4px 12px rgba(231, 60, 23, 0.4);
        }

        .new-thread-form {
          padding: 16px;
          background: #1e1e2e;
          border: 1px solid #313244;
          border-radius: 8px;
          margin-bottom: 12px;
        }

        .new-thread-form input {
          width: 100%;
          padding: 10px 12px;
          background: #11111b;
          border: 1px solid #45475a;
          border-radius: 6px;
          color: #cdd6f4;
          font-size: 14px;
          margin-bottom: 12px;
        }

        .new-thread-form input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 3px rgba(231, 60, 23, 0.1);
        }

        .form-actions {
          display: flex;
          justify-content: flex-end;
          gap: 8px;
        }

        .form-actions button {
          padding: 6px 14px;
          border-radius: 6px;
          font-size: 13px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .form-actions button:first-child {
          background: transparent;
          border: 1px solid #45475a;
          color: #6c7086;
        }

        .form-actions button:first-child:hover {
          background: #313244;
        }

        .form-actions .create-btn {
          background: linear-gradient(135deg, var(--gradient-start), var(--gradient-end));
          border: none;
          color: white;
          font-family: var(--font-heading);
          font-weight: 600;
          box-shadow: 0 2px 8px rgba(231, 60, 23, 0.3);
        }

        .form-actions .create-btn:hover {
          background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));
          box-shadow: 0 4px 12px rgba(231, 60, 23, 0.4);
        }

        .empty-state {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100%;
          color: #6c7086;
          font-size: 14px;
        }

        /* Overview Layout */
        .overview-container {
          display: flex;
          gap: 24px;
          padding: 24px;
          height: 100%;
          overflow-y: auto;
        }

        .overview-main {
          flex: 1;
          min-width: 0;
        }

        .overview-sidebar {
          width: 320px;
          flex-shrink: 0;
        }

        /* Responsive */
        @media (max-width: 1024px) {
          .overview-container {
            flex-direction: column;
          }

          .overview-sidebar {
            width: 100%;
          }
        }

        @media (max-width: 768px) {
          .brand-text {
            display: none;
          }

          .app-sidebar {
            width: 60px;
          }
        }
      `})]})};Mo.createRoot(document.getElementById("root")).render(s.jsx(Jt.StrictMode,{children:s.jsx(f2,{})}));
