var nh=Object.defineProperty;var rh=(e,t,n)=>t in e?nh(e,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):e[t]=n;var He=(e,t,n)=>rh(e,typeof t!="symbol"?t+"":t,n);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Gi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Da(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Zc={exports:{}},jl={},ed={exports:{}},G={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ui=Symbol.for("react.element"),ih=Symbol.for("react.portal"),lh=Symbol.for("react.fragment"),oh=Symbol.for("react.strict_mode"),ah=Symbol.for("react.profiler"),sh=Symbol.for("react.provider"),uh=Symbol.for("react.context"),ch=Symbol.for("react.forward_ref"),dh=Symbol.for("react.suspense"),fh=Symbol.for("react.memo"),ph=Symbol.for("react.lazy"),Gs=Symbol.iterator;function hh(e){return e===null||typeof e!="object"?null:(e=Gs&&e[Gs]||e["@@iterator"],typeof e=="function"?e:null)}var td={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},nd=Object.assign,rd={};function fr(e,t,n){this.props=e,this.context=t,this.refs=rd,this.updater=n||td}fr.prototype.isReactComponent={};fr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};fr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function id(){}id.prototype=fr.prototype;function Ra(e,t,n){this.props=e,this.context=t,this.refs=rd,this.updater=n||td}var Fa=Ra.prototype=new id;Fa.constructor=Ra;nd(Fa,fr.prototype);Fa.isPureReactComponent=!0;var Js=Array.isArray,ld=Object.prototype.hasOwnProperty,Oa={current:null},od={key:!0,ref:!0,__self:!0,__source:!0};function ad(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)ld.call(t,r)&&!od.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),c=0;c<a;c++)s[c]=arguments[c+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:ui,type:e,key:l,ref:o,props:i,_owner:Oa.current}}function mh(e,t){return{$$typeof:ui,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function Ba(e){return typeof e=="object"&&e!==null&&e.$$typeof===ui}function gh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Zs=/\/+/g;function Wl(e,t){return typeof e=="object"&&e!==null&&e.key!=null?gh(""+e.key):t.toString(36)}function Ri(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case ui:case ih:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Wl(o,0):r,Js(i)?(n="",e!=null&&(n=e.replace(Zs,"$&/")+"/"),Ri(i,t,n,"",function(c){return c})):i!=null&&(Ba(i)&&(i=mh(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Zs,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Js(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Wl(l,a);o+=Ri(l,t,n,s,i)}else if(s=hh(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Wl(l,a++),o+=Ri(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function gi(e,t,n){if(e==null)return e;var r=[],i=0;return Ri(e,r,"","",function(l){return t.call(n,l,i++)}),r}function vh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Be={current:null},Fi={transition:null},yh={ReactCurrentDispatcher:Be,ReactCurrentBatchConfig:Fi,ReactCurrentOwner:Oa};function sd(){throw Error("act(...) is not supported in production builds of React.")}G.Children={map:gi,forEach:function(e,t,n){gi(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return gi(e,function(){t++}),t},toArray:function(e){return gi(e,function(t){return t})||[]},only:function(e){if(!Ba(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};G.Component=fr;G.Fragment=lh;G.Profiler=ah;G.PureComponent=Ra;G.StrictMode=oh;G.Suspense=dh;G.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=yh;G.act=sd;G.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=nd({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Oa.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)ld.call(t,s)&&!od.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var c=0;c<s;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:ui,type:e.type,key:i,ref:l,props:r,_owner:o}};G.createContext=function(e){return e={$$typeof:uh,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:sh,_context:e},e.Consumer=e};G.createElement=ad;G.createFactory=function(e){var t=ad.bind(null,e);return t.type=e,t};G.createRef=function(){return{current:null}};G.forwardRef=function(e){return{$$typeof:ch,render:e}};G.isValidElement=Ba;G.lazy=function(e){return{$$typeof:ph,_payload:{_status:-1,_result:e},_init:vh}};G.memo=function(e,t){return{$$typeof:fh,type:e,compare:t===void 0?null:t}};G.startTransition=function(e){var t=Fi.transition;Fi.transition={};try{e()}finally{Fi.transition=t}};G.unstable_act=sd;G.useCallback=function(e,t){return Be.current.useCallback(e,t)};G.useContext=function(e){return Be.current.useContext(e)};G.useDebugValue=function(){};G.useDeferredValue=function(e){return Be.current.useDeferredValue(e)};G.useEffect=function(e,t){return Be.current.useEffect(e,t)};G.useId=function(){return Be.current.useId()};G.useImperativeHandle=function(e,t,n){return Be.current.useImperativeHandle(e,t,n)};G.useInsertionEffect=function(e,t){return Be.current.useInsertionEffect(e,t)};G.useLayoutEffect=function(e,t){return Be.current.useLayoutEffect(e,t)};G.useMemo=function(e,t){return Be.current.useMemo(e,t)};G.useReducer=function(e,t,n){return Be.current.useReducer(e,t,n)};G.useRef=function(e){return Be.current.useRef(e)};G.useState=function(e){return Be.current.useState(e)};G.useSyncExternalStore=function(e,t,n){return Be.current.useSyncExternalStore(e,t,n)};G.useTransition=function(){return Be.current.useTransition()};G.version="18.3.1";ed.exports=G;var F=ed.exports;const Xt=Da(F);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var xh=F,kh=Symbol.for("react.element"),wh=Symbol.for("react.fragment"),Sh=Object.prototype.hasOwnProperty,bh=xh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,Ch={key:!0,ref:!0,__self:!0,__source:!0};function ud(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)Sh.call(t,r)&&!Ch.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:kh,type:e,key:l,ref:o,props:i,_owner:bh.current}}jl.Fragment=wh;jl.jsx=ud;jl.jsxs=ud;Zc.exports=jl;var u=Zc.exports,Lo={},cd={exports:{}},ot={},dd={exports:{}},fd={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(C,B){var m=C.length;C.push(B);e:for(;0<m;){var z=m-1>>>1,A=C[z];if(0<i(A,B))C[z]=B,C[m]=A,m=z;else break e}}function n(C){return C.length===0?null:C[0]}function r(C){if(C.length===0)return null;var B=C[0],m=C.pop();if(m!==B){C[0]=m;e:for(var z=0,A=C.length,x=A>>>1;z<x;){var X=2*(z+1)-1,fe=C[X],J=X+1,ve=C[J];if(0>i(fe,m))J<A&&0>i(ve,fe)?(C[z]=ve,C[J]=m,z=J):(C[z]=fe,C[X]=m,z=X);else if(J<A&&0>i(ve,m))C[z]=ve,C[J]=m,z=J;else break e}}return B}function i(C,B){var m=C.sortIndex-B.sortIndex;return m!==0?m:C.id-B.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],c=[],d=1,f=null,g=3,p=!1,k=!1,w=!1,I=typeof setTimeout=="function"?setTimeout:null,h=typeof clearTimeout=="function"?clearTimeout:null,v=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(C){for(var B=n(c);B!==null;){if(B.callback===null)r(c);else if(B.startTime<=C)r(c),B.sortIndex=B.expirationTime,t(s,B);else break;B=n(c)}}function b(C){if(w=!1,y(C),!k)if(n(s)!==null)k=!0,q(_);else{var B=n(c);B!==null&&ie(b,B.startTime-C)}}function _(C,B){k=!1,w&&(w=!1,h(L),L=-1),p=!0;var m=g;try{for(y(B),f=n(s);f!==null&&(!(f.expirationTime>B)||C&&!j());){var z=f.callback;if(typeof z=="function"){f.callback=null,g=f.priorityLevel;var A=z(f.expirationTime<=B);B=e.unstable_now(),typeof A=="function"?f.callback=A:f===n(s)&&r(s),y(B)}else r(s);f=n(s)}if(f!==null)var x=!0;else{var X=n(c);X!==null&&ie(b,X.startTime-B),x=!1}return x}finally{f=null,g=m,p=!1}}var S=!1,E=null,L=-1,D=5,P=-1;function j(){return!(e.unstable_now()-P<D)}function T(){if(E!==null){var C=e.unstable_now();P=C;var B=!0;try{B=E(!0,C)}finally{B?U():(S=!1,E=null)}}else S=!1}var U;if(typeof v=="function")U=function(){v(T)};else if(typeof MessageChannel<"u"){var Q=new MessageChannel,H=Q.port2;Q.port1.onmessage=T,U=function(){H.postMessage(null)}}else U=function(){I(T,0)};function q(C){E=C,S||(S=!0,U())}function ie(C,B){L=I(function(){C(e.unstable_now())},B)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(C){C.callback=null},e.unstable_continueExecution=function(){k||p||(k=!0,q(_))},e.unstable_forceFrameRate=function(C){0>C||125<C?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):D=0<C?Math.floor(1e3/C):5},e.unstable_getCurrentPriorityLevel=function(){return g},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(C){switch(g){case 1:case 2:case 3:var B=3;break;default:B=g}var m=g;g=B;try{return C()}finally{g=m}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(C,B){switch(C){case 1:case 2:case 3:case 4:case 5:break;default:C=3}var m=g;g=C;try{return B()}finally{g=m}},e.unstable_scheduleCallback=function(C,B,m){var z=e.unstable_now();switch(typeof m=="object"&&m!==null?(m=m.delay,m=typeof m=="number"&&0<m?z+m:z):m=z,C){case 1:var A=-1;break;case 2:A=250;break;case 5:A=1073741823;break;case 4:A=1e4;break;default:A=5e3}return A=m+A,C={id:d++,callback:B,priorityLevel:C,startTime:m,expirationTime:A,sortIndex:-1},m>z?(C.sortIndex=m,t(c,C),n(s)===null&&C===n(c)&&(w?(h(L),L=-1):w=!0,ie(b,m-z))):(C.sortIndex=A,t(s,C),k||p||(k=!0,q(_))),C},e.unstable_shouldYield=j,e.unstable_wrapCallback=function(C){var B=g;return function(){var m=g;g=B;try{return C.apply(this,arguments)}finally{g=m}}}})(fd);dd.exports=fd;var jh=dd.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var Eh=F,lt=jh;function M(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var pd=new Set,Wr={};function Pn(e,t){lr(e,t),lr(e+"Capture",t)}function lr(e,t){for(Wr[e]=t,e=0;e<t.length;e++)pd.add(t[e])}var $t=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),Po=Object.prototype.hasOwnProperty,_h=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,eu={},tu={};function Nh(e){return Po.call(tu,e)?!0:Po.call(eu,e)?!1:_h.test(e)?tu[e]=!0:(eu[e]=!0,!1)}function Th(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function zh(e,t,n,r){if(t===null||typeof t>"u"||Th(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function $e(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var ze={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){ze[e]=new $e(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];ze[t]=new $e(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){ze[e]=new $e(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){ze[e]=new $e(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){ze[e]=new $e(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){ze[e]=new $e(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){ze[e]=new $e(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){ze[e]=new $e(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){ze[e]=new $e(e,5,!1,e.toLowerCase(),null,!1,!1)});var $a=/[\-:]([a-z])/g;function Ua(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace($a,Ua);ze[t]=new $e(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace($a,Ua);ze[t]=new $e(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace($a,Ua);ze[t]=new $e(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){ze[e]=new $e(e,1,!1,e.toLowerCase(),null,!1,!1)});ze.xlinkHref=new $e("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){ze[e]=new $e(e,1,!1,e.toLowerCase(),null,!0,!0)});function Ha(e,t,n,r){var i=ze.hasOwnProperty(t)?ze[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(zh(t,n,i,r)&&(n=null),r||i===null?Nh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Wt=Eh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,vi=Symbol.for("react.element"),On=Symbol.for("react.portal"),Bn=Symbol.for("react.fragment"),Va=Symbol.for("react.strict_mode"),Io=Symbol.for("react.profiler"),hd=Symbol.for("react.provider"),md=Symbol.for("react.context"),Wa=Symbol.for("react.forward_ref"),Ao=Symbol.for("react.suspense"),Mo=Symbol.for("react.suspense_list"),Qa=Symbol.for("react.memo"),Gt=Symbol.for("react.lazy"),gd=Symbol.for("react.offscreen"),nu=Symbol.iterator;function xr(e){return e===null||typeof e!="object"?null:(e=nu&&e[nu]||e["@@iterator"],typeof e=="function"?e:null)}var me=Object.assign,Ql;function Tr(e){if(Ql===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Ql=t&&t[1]||""}return`
`+Ql+e}var ql=!1;function Kl(e,t){if(!e||ql)return"";ql=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{ql=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?Tr(e):""}function Lh(e){switch(e.tag){case 5:return Tr(e.type);case 16:return Tr("Lazy");case 13:return Tr("Suspense");case 19:return Tr("SuspenseList");case 0:case 2:case 15:return e=Kl(e.type,!1),e;case 11:return e=Kl(e.type.render,!1),e;case 1:return e=Kl(e.type,!0),e;default:return""}}function Do(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Bn:return"Fragment";case On:return"Portal";case Io:return"Profiler";case Va:return"StrictMode";case Ao:return"Suspense";case Mo:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case md:return(e.displayName||"Context")+".Consumer";case hd:return(e._context.displayName||"Context")+".Provider";case Wa:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Qa:return t=e.displayName||null,t!==null?t:Do(e.type)||"Memo";case Gt:t=e._payload,e=e._init;try{return Do(e(t))}catch{}}return null}function Ph(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Do(t);case 8:return t===Va?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function fn(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function vd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function Ih(e){var t=vd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function yi(e){e._valueTracker||(e._valueTracker=Ih(e))}function yd(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=vd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Ji(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function Ro(e,t){var n=t.checked;return me({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function ru(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=fn(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function xd(e,t){t=t.checked,t!=null&&Ha(e,"checked",t,!1)}function Fo(e,t){xd(e,t);var n=fn(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?Oo(e,t.type,n):t.hasOwnProperty("defaultValue")&&Oo(e,t.type,fn(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function iu(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function Oo(e,t,n){(t!=="number"||Ji(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var zr=Array.isArray;function Gn(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+fn(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function Bo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(M(91));return me({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function lu(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(M(92));if(zr(n)){if(1<n.length)throw Error(M(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:fn(n)}}function kd(e,t){var n=fn(t.value),r=fn(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function ou(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function wd(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function $o(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?wd(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var xi,Sd=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(xi=xi||document.createElement("div"),xi.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=xi.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Qr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Ir={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},Ah=["Webkit","ms","Moz","O"];Object.keys(Ir).forEach(function(e){Ah.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Ir[t]=Ir[e]})});function bd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Ir.hasOwnProperty(e)&&Ir[e]?(""+t).trim():t+"px"}function Cd(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=bd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var Mh=me({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Uo(e,t){if(t){if(Mh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(M(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(M(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(M(61))}if(t.style!=null&&typeof t.style!="object")throw Error(M(62))}}function Ho(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Vo=null;function qa(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Wo=null,Jn=null,Zn=null;function au(e){if(e=fi(e)){if(typeof Wo!="function")throw Error(M(280));var t=e.stateNode;t&&(t=zl(t),Wo(e.stateNode,e.type,t))}}function jd(e){Jn?Zn?Zn.push(e):Zn=[e]:Jn=e}function Ed(){if(Jn){var e=Jn,t=Zn;if(Zn=Jn=null,au(e),t)for(e=0;e<t.length;e++)au(t[e])}}function _d(e,t){return e(t)}function Nd(){}var Yl=!1;function Td(e,t,n){if(Yl)return e(t,n);Yl=!0;try{return _d(e,t,n)}finally{Yl=!1,(Jn!==null||Zn!==null)&&(Nd(),Ed())}}function qr(e,t){var n=e.stateNode;if(n===null)return null;var r=zl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(M(231,t,typeof n));return n}var Qo=!1;if($t)try{var kr={};Object.defineProperty(kr,"passive",{get:function(){Qo=!0}}),window.addEventListener("test",kr,kr),window.removeEventListener("test",kr,kr)}catch{Qo=!1}function Dh(e,t,n,r,i,l,o,a,s){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Ar=!1,Zi=null,el=!1,qo=null,Rh={onError:function(e){Ar=!0,Zi=e}};function Fh(e,t,n,r,i,l,o,a,s){Ar=!1,Zi=null,Dh.apply(Rh,arguments)}function Oh(e,t,n,r,i,l,o,a,s){if(Fh.apply(this,arguments),Ar){if(Ar){var c=Zi;Ar=!1,Zi=null}else throw Error(M(198));el||(el=!0,qo=c)}}function In(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function zd(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function su(e){if(In(e)!==e)throw Error(M(188))}function Bh(e){var t=e.alternate;if(!t){if(t=In(e),t===null)throw Error(M(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return su(i),e;if(l===r)return su(i),t;l=l.sibling}throw Error(M(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(M(189))}}if(n.alternate!==r)throw Error(M(190))}if(n.tag!==3)throw Error(M(188));return n.stateNode.current===n?e:t}function Ld(e){return e=Bh(e),e!==null?Pd(e):null}function Pd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=Pd(e);if(t!==null)return t;e=e.sibling}return null}var Id=lt.unstable_scheduleCallback,uu=lt.unstable_cancelCallback,$h=lt.unstable_shouldYield,Uh=lt.unstable_requestPaint,ye=lt.unstable_now,Hh=lt.unstable_getCurrentPriorityLevel,Ka=lt.unstable_ImmediatePriority,Ad=lt.unstable_UserBlockingPriority,tl=lt.unstable_NormalPriority,Vh=lt.unstable_LowPriority,Md=lt.unstable_IdlePriority,El=null,Tt=null;function Wh(e){if(Tt&&typeof Tt.onCommitFiberRoot=="function")try{Tt.onCommitFiberRoot(El,e,void 0,(e.current.flags&128)===128)}catch{}}var kt=Math.clz32?Math.clz32:Kh,Qh=Math.log,qh=Math.LN2;function Kh(e){return e>>>=0,e===0?32:31-(Qh(e)/qh|0)|0}var ki=64,wi=4194304;function Lr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function nl(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=Lr(a):(l&=o,l!==0&&(r=Lr(l)))}else o=n&~i,o!==0?r=Lr(o):l!==0&&(r=Lr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-kt(t),i=1<<n,r|=e[n],t&=~i;return r}function Yh(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Xh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-kt(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Yh(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Ko(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function Dd(){var e=ki;return ki<<=1,!(ki&4194240)&&(ki=64),e}function Xl(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function ci(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-kt(t),e[t]=n}function Gh(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-kt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Ya(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-kt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var re=0;function Rd(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Fd,Xa,Od,Bd,$d,Yo=!1,Si=[],rn=null,ln=null,on=null,Kr=new Map,Yr=new Map,Zt=[],Jh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function cu(e,t){switch(e){case"focusin":case"focusout":rn=null;break;case"dragenter":case"dragleave":ln=null;break;case"mouseover":case"mouseout":on=null;break;case"pointerover":case"pointerout":Kr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Yr.delete(t.pointerId)}}function wr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=fi(t),t!==null&&Xa(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Zh(e,t,n,r,i){switch(t){case"focusin":return rn=wr(rn,e,t,n,r,i),!0;case"dragenter":return ln=wr(ln,e,t,n,r,i),!0;case"mouseover":return on=wr(on,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Kr.set(l,wr(Kr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Yr.set(l,wr(Yr.get(l)||null,e,t,n,r,i)),!0}return!1}function Ud(e){var t=Sn(e.target);if(t!==null){var n=In(t);if(n!==null){if(t=n.tag,t===13){if(t=zd(n),t!==null){e.blockedOn=t,$d(e.priority,function(){Od(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Oi(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Xo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Vo=r,n.target.dispatchEvent(r),Vo=null}else return t=fi(n),t!==null&&Xa(t),e.blockedOn=n,!1;t.shift()}return!0}function du(e,t,n){Oi(e)&&n.delete(t)}function em(){Yo=!1,rn!==null&&Oi(rn)&&(rn=null),ln!==null&&Oi(ln)&&(ln=null),on!==null&&Oi(on)&&(on=null),Kr.forEach(du),Yr.forEach(du)}function Sr(e,t){e.blockedOn===t&&(e.blockedOn=null,Yo||(Yo=!0,lt.unstable_scheduleCallback(lt.unstable_NormalPriority,em)))}function Xr(e){function t(i){return Sr(i,e)}if(0<Si.length){Sr(Si[0],e);for(var n=1;n<Si.length;n++){var r=Si[n];r.blockedOn===e&&(r.blockedOn=null)}}for(rn!==null&&Sr(rn,e),ln!==null&&Sr(ln,e),on!==null&&Sr(on,e),Kr.forEach(t),Yr.forEach(t),n=0;n<Zt.length;n++)r=Zt[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Zt.length&&(n=Zt[0],n.blockedOn===null);)Ud(n),n.blockedOn===null&&Zt.shift()}var er=Wt.ReactCurrentBatchConfig,rl=!0;function tm(e,t,n,r){var i=re,l=er.transition;er.transition=null;try{re=1,Ga(e,t,n,r)}finally{re=i,er.transition=l}}function nm(e,t,n,r){var i=re,l=er.transition;er.transition=null;try{re=4,Ga(e,t,n,r)}finally{re=i,er.transition=l}}function Ga(e,t,n,r){if(rl){var i=Xo(e,t,n,r);if(i===null)oo(e,t,r,il,n),cu(e,r);else if(Zh(i,e,t,n,r))r.stopPropagation();else if(cu(e,r),t&4&&-1<Jh.indexOf(e)){for(;i!==null;){var l=fi(i);if(l!==null&&Fd(l),l=Xo(e,t,n,r),l===null&&oo(e,t,r,il,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else oo(e,t,r,null,n)}}var il=null;function Xo(e,t,n,r){if(il=null,e=qa(r),e=Sn(e),e!==null)if(t=In(e),t===null)e=null;else if(n=t.tag,n===13){if(e=zd(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return il=e,null}function Hd(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(Hh()){case Ka:return 1;case Ad:return 4;case tl:case Vh:return 16;case Md:return 536870912;default:return 16}default:return 16}}var tn=null,Ja=null,Bi=null;function Vd(){if(Bi)return Bi;var e,t=Ja,n=t.length,r,i="value"in tn?tn.value:tn.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Bi=i.slice(e,1<r?1-r:void 0)}function $i(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function bi(){return!0}function fu(){return!1}function at(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?bi:fu,this.isPropagationStopped=fu,this}return me(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=bi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=bi)},persist:function(){},isPersistent:bi}),t}var pr={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Za=at(pr),di=me({},pr,{view:0,detail:0}),rm=at(di),Gl,Jl,br,_l=me({},di,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:es,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==br&&(br&&e.type==="mousemove"?(Gl=e.screenX-br.screenX,Jl=e.screenY-br.screenY):Jl=Gl=0,br=e),Gl)},movementY:function(e){return"movementY"in e?e.movementY:Jl}}),pu=at(_l),im=me({},_l,{dataTransfer:0}),lm=at(im),om=me({},di,{relatedTarget:0}),Zl=at(om),am=me({},pr,{animationName:0,elapsedTime:0,pseudoElement:0}),sm=at(am),um=me({},pr,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),cm=at(um),dm=me({},pr,{data:0}),hu=at(dm),fm={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},pm={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},hm={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function mm(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=hm[e])?!!t[e]:!1}function es(){return mm}var gm=me({},di,{key:function(e){if(e.key){var t=fm[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=$i(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?pm[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:es,charCode:function(e){return e.type==="keypress"?$i(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?$i(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),vm=at(gm),ym=me({},_l,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),mu=at(ym),xm=me({},di,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:es}),km=at(xm),wm=me({},pr,{propertyName:0,elapsedTime:0,pseudoElement:0}),Sm=at(wm),bm=me({},_l,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),Cm=at(bm),jm=[9,13,27,32],ts=$t&&"CompositionEvent"in window,Mr=null;$t&&"documentMode"in document&&(Mr=document.documentMode);var Em=$t&&"TextEvent"in window&&!Mr,Wd=$t&&(!ts||Mr&&8<Mr&&11>=Mr),gu=" ",vu=!1;function Qd(e,t){switch(e){case"keyup":return jm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function qd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var $n=!1;function _m(e,t){switch(e){case"compositionend":return qd(t);case"keypress":return t.which!==32?null:(vu=!0,gu);case"textInput":return e=t.data,e===gu&&vu?null:e;default:return null}}function Nm(e,t){if($n)return e==="compositionend"||!ts&&Qd(e,t)?(e=Vd(),Bi=Ja=tn=null,$n=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Wd&&t.locale!=="ko"?null:t.data;default:return null}}var Tm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function yu(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!Tm[e.type]:t==="textarea"}function Kd(e,t,n,r){jd(r),t=ll(t,"onChange"),0<t.length&&(n=new Za("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Dr=null,Gr=null;function zm(e){of(e,0)}function Nl(e){var t=Vn(e);if(yd(t))return e}function Lm(e,t){if(e==="change")return t}var Yd=!1;if($t){var eo;if($t){var to="oninput"in document;if(!to){var xu=document.createElement("div");xu.setAttribute("oninput","return;"),to=typeof xu.oninput=="function"}eo=to}else eo=!1;Yd=eo&&(!document.documentMode||9<document.documentMode)}function ku(){Dr&&(Dr.detachEvent("onpropertychange",Xd),Gr=Dr=null)}function Xd(e){if(e.propertyName==="value"&&Nl(Gr)){var t=[];Kd(t,Gr,e,qa(e)),Td(zm,t)}}function Pm(e,t,n){e==="focusin"?(ku(),Dr=t,Gr=n,Dr.attachEvent("onpropertychange",Xd)):e==="focusout"&&ku()}function Im(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return Nl(Gr)}function Am(e,t){if(e==="click")return Nl(t)}function Mm(e,t){if(e==="input"||e==="change")return Nl(t)}function Dm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var St=typeof Object.is=="function"?Object.is:Dm;function Jr(e,t){if(St(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!Po.call(t,i)||!St(e[i],t[i]))return!1}return!0}function wu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function Su(e,t){var n=wu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=wu(n)}}function Gd(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Gd(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Jd(){for(var e=window,t=Ji();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Ji(e.document)}return t}function ns(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function Rm(e){var t=Jd(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Gd(n.ownerDocument.documentElement,n)){if(r!==null&&ns(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=Su(n,l);var o=Su(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Fm=$t&&"documentMode"in document&&11>=document.documentMode,Un=null,Go=null,Rr=null,Jo=!1;function bu(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Jo||Un==null||Un!==Ji(r)||(r=Un,"selectionStart"in r&&ns(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Rr&&Jr(Rr,r)||(Rr=r,r=ll(Go,"onSelect"),0<r.length&&(t=new Za("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Un)))}function Ci(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Hn={animationend:Ci("Animation","AnimationEnd"),animationiteration:Ci("Animation","AnimationIteration"),animationstart:Ci("Animation","AnimationStart"),transitionend:Ci("Transition","TransitionEnd")},no={},Zd={};$t&&(Zd=document.createElement("div").style,"AnimationEvent"in window||(delete Hn.animationend.animation,delete Hn.animationiteration.animation,delete Hn.animationstart.animation),"TransitionEvent"in window||delete Hn.transitionend.transition);function Tl(e){if(no[e])return no[e];if(!Hn[e])return e;var t=Hn[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Zd)return no[e]=t[n];return e}var ef=Tl("animationend"),tf=Tl("animationiteration"),nf=Tl("animationstart"),rf=Tl("transitionend"),lf=new Map,Cu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function hn(e,t){lf.set(e,t),Pn(t,[e])}for(var ro=0;ro<Cu.length;ro++){var io=Cu[ro],Om=io.toLowerCase(),Bm=io[0].toUpperCase()+io.slice(1);hn(Om,"on"+Bm)}hn(ef,"onAnimationEnd");hn(tf,"onAnimationIteration");hn(nf,"onAnimationStart");hn("dblclick","onDoubleClick");hn("focusin","onFocus");hn("focusout","onBlur");hn(rf,"onTransitionEnd");lr("onMouseEnter",["mouseout","mouseover"]);lr("onMouseLeave",["mouseout","mouseover"]);lr("onPointerEnter",["pointerout","pointerover"]);lr("onPointerLeave",["pointerout","pointerover"]);Pn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));Pn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));Pn("onBeforeInput",["compositionend","keypress","textInput","paste"]);Pn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));Pn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));Pn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var Pr="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),$m=new Set("cancel close invalid load scroll toggle".split(" ").concat(Pr));function ju(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,Oh(r,t,void 0,e),e.currentTarget=null}function of(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,c=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;ju(i,a,c),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,c=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;ju(i,a,c),l=s}}}if(el)throw e=qo,el=!1,qo=null,e}function ue(e,t){var n=t[ra];n===void 0&&(n=t[ra]=new Set);var r=e+"__bubble";n.has(r)||(af(t,e,2,!1),n.add(r))}function lo(e,t,n){var r=0;t&&(r|=4),af(n,e,r,t)}var ji="_reactListening"+Math.random().toString(36).slice(2);function Zr(e){if(!e[ji]){e[ji]=!0,pd.forEach(function(n){n!=="selectionchange"&&($m.has(n)||lo(n,!1,e),lo(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[ji]||(t[ji]=!0,lo("selectionchange",!1,t))}}function af(e,t,n,r){switch(Hd(t)){case 1:var i=tm;break;case 4:i=nm;break;default:i=Ga}n=i.bind(null,t,n,e),i=void 0,!Qo||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function oo(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=Sn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}Td(function(){var c=l,d=qa(n),f=[];e:{var g=lf.get(e);if(g!==void 0){var p=Za,k=e;switch(e){case"keypress":if($i(n)===0)break e;case"keydown":case"keyup":p=vm;break;case"focusin":k="focus",p=Zl;break;case"focusout":k="blur",p=Zl;break;case"beforeblur":case"afterblur":p=Zl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":p=pu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":p=lm;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":p=km;break;case ef:case tf:case nf:p=sm;break;case rf:p=Sm;break;case"scroll":p=rm;break;case"wheel":p=Cm;break;case"copy":case"cut":case"paste":p=cm;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":p=mu}var w=(t&4)!==0,I=!w&&e==="scroll",h=w?g!==null?g+"Capture":null:g;w=[];for(var v=c,y;v!==null;){y=v;var b=y.stateNode;if(y.tag===5&&b!==null&&(y=b,h!==null&&(b=qr(v,h),b!=null&&w.push(ei(v,b,y)))),I)break;v=v.return}0<w.length&&(g=new p(g,k,null,n,d),f.push({event:g,listeners:w}))}}if(!(t&7)){e:{if(g=e==="mouseover"||e==="pointerover",p=e==="mouseout"||e==="pointerout",g&&n!==Vo&&(k=n.relatedTarget||n.fromElement)&&(Sn(k)||k[Ut]))break e;if((p||g)&&(g=d.window===d?d:(g=d.ownerDocument)?g.defaultView||g.parentWindow:window,p?(k=n.relatedTarget||n.toElement,p=c,k=k?Sn(k):null,k!==null&&(I=In(k),k!==I||k.tag!==5&&k.tag!==6)&&(k=null)):(p=null,k=c),p!==k)){if(w=pu,b="onMouseLeave",h="onMouseEnter",v="mouse",(e==="pointerout"||e==="pointerover")&&(w=mu,b="onPointerLeave",h="onPointerEnter",v="pointer"),I=p==null?g:Vn(p),y=k==null?g:Vn(k),g=new w(b,v+"leave",p,n,d),g.target=I,g.relatedTarget=y,b=null,Sn(d)===c&&(w=new w(h,v+"enter",k,n,d),w.target=y,w.relatedTarget=I,b=w),I=b,p&&k)t:{for(w=p,h=k,v=0,y=w;y;y=Dn(y))v++;for(y=0,b=h;b;b=Dn(b))y++;for(;0<v-y;)w=Dn(w),v--;for(;0<y-v;)h=Dn(h),y--;for(;v--;){if(w===h||h!==null&&w===h.alternate)break t;w=Dn(w),h=Dn(h)}w=null}else w=null;p!==null&&Eu(f,g,p,w,!1),k!==null&&I!==null&&Eu(f,I,k,w,!0)}}e:{if(g=c?Vn(c):window,p=g.nodeName&&g.nodeName.toLowerCase(),p==="select"||p==="input"&&g.type==="file")var _=Lm;else if(yu(g))if(Yd)_=Mm;else{_=Im;var S=Pm}else(p=g.nodeName)&&p.toLowerCase()==="input"&&(g.type==="checkbox"||g.type==="radio")&&(_=Am);if(_&&(_=_(e,c))){Kd(f,_,n,d);break e}S&&S(e,g,c),e==="focusout"&&(S=g._wrapperState)&&S.controlled&&g.type==="number"&&Oo(g,"number",g.value)}switch(S=c?Vn(c):window,e){case"focusin":(yu(S)||S.contentEditable==="true")&&(Un=S,Go=c,Rr=null);break;case"focusout":Rr=Go=Un=null;break;case"mousedown":Jo=!0;break;case"contextmenu":case"mouseup":case"dragend":Jo=!1,bu(f,n,d);break;case"selectionchange":if(Fm)break;case"keydown":case"keyup":bu(f,n,d)}var E;if(ts)e:{switch(e){case"compositionstart":var L="onCompositionStart";break e;case"compositionend":L="onCompositionEnd";break e;case"compositionupdate":L="onCompositionUpdate";break e}L=void 0}else $n?Qd(e,n)&&(L="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(L="onCompositionStart");L&&(Wd&&n.locale!=="ko"&&($n||L!=="onCompositionStart"?L==="onCompositionEnd"&&$n&&(E=Vd()):(tn=d,Ja="value"in tn?tn.value:tn.textContent,$n=!0)),S=ll(c,L),0<S.length&&(L=new hu(L,e,null,n,d),f.push({event:L,listeners:S}),E?L.data=E:(E=qd(n),E!==null&&(L.data=E)))),(E=Em?_m(e,n):Nm(e,n))&&(c=ll(c,"onBeforeInput"),0<c.length&&(d=new hu("onBeforeInput","beforeinput",null,n,d),f.push({event:d,listeners:c}),d.data=E))}of(f,t)})}function ei(e,t,n){return{instance:e,listener:t,currentTarget:n}}function ll(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=qr(e,n),l!=null&&r.unshift(ei(e,l,i)),l=qr(e,t),l!=null&&r.push(ei(e,l,i))),e=e.return}return r}function Dn(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function Eu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,c=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&c!==null&&(a=c,i?(s=qr(n,l),s!=null&&o.unshift(ei(n,s,a))):i||(s=qr(n,l),s!=null&&o.push(ei(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Um=/\r\n?/g,Hm=/\u0000|\uFFFD/g;function _u(e){return(typeof e=="string"?e:""+e).replace(Um,`
`).replace(Hm,"")}function Ei(e,t,n){if(t=_u(t),_u(e)!==t&&n)throw Error(M(425))}function ol(){}var Zo=null,ea=null;function ta(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var na=typeof setTimeout=="function"?setTimeout:void 0,Vm=typeof clearTimeout=="function"?clearTimeout:void 0,Nu=typeof Promise=="function"?Promise:void 0,Wm=typeof queueMicrotask=="function"?queueMicrotask:typeof Nu<"u"?function(e){return Nu.resolve(null).then(e).catch(Qm)}:na;function Qm(e){setTimeout(function(){throw e})}function ao(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Xr(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Xr(t)}function an(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function Tu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var hr=Math.random().toString(36).slice(2),_t="__reactFiber$"+hr,ti="__reactProps$"+hr,Ut="__reactContainer$"+hr,ra="__reactEvents$"+hr,qm="__reactListeners$"+hr,Km="__reactHandles$"+hr;function Sn(e){var t=e[_t];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Ut]||n[_t]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=Tu(e);e!==null;){if(n=e[_t])return n;e=Tu(e)}return t}e=n,n=e.parentNode}return null}function fi(e){return e=e[_t]||e[Ut],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function Vn(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(M(33))}function zl(e){return e[ti]||null}var ia=[],Wn=-1;function mn(e){return{current:e}}function ce(e){0>Wn||(e.current=ia[Wn],ia[Wn]=null,Wn--)}function ae(e,t){Wn++,ia[Wn]=e.current,e.current=t}var pn={},Me=mn(pn),qe=mn(!1),_n=pn;function or(e,t){var n=e.type.contextTypes;if(!n)return pn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Ke(e){return e=e.childContextTypes,e!=null}function al(){ce(qe),ce(Me)}function zu(e,t,n){if(Me.current!==pn)throw Error(M(168));ae(Me,t),ae(qe,n)}function sf(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(M(108,Ph(e)||"Unknown",i));return me({},n,r)}function sl(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||pn,_n=Me.current,ae(Me,e),ae(qe,qe.current),!0}function Lu(e,t,n){var r=e.stateNode;if(!r)throw Error(M(169));n?(e=sf(e,t,_n),r.__reactInternalMemoizedMergedChildContext=e,ce(qe),ce(Me),ae(Me,e)):ce(qe),ae(qe,n)}var Rt=null,Ll=!1,so=!1;function uf(e){Rt===null?Rt=[e]:Rt.push(e)}function Ym(e){Ll=!0,uf(e)}function gn(){if(!so&&Rt!==null){so=!0;var e=0,t=re;try{var n=Rt;for(re=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}Rt=null,Ll=!1}catch(i){throw Rt!==null&&(Rt=Rt.slice(e+1)),Id(Ka,gn),i}finally{re=t,so=!1}}return null}var Qn=[],qn=0,ul=null,cl=0,st=[],ut=0,Nn=null,Ft=1,Ot="";function xn(e,t){Qn[qn++]=cl,Qn[qn++]=ul,ul=e,cl=t}function cf(e,t,n){st[ut++]=Ft,st[ut++]=Ot,st[ut++]=Nn,Nn=e;var r=Ft;e=Ot;var i=32-kt(r)-1;r&=~(1<<i),n+=1;var l=32-kt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,Ft=1<<32-kt(t)+i|n<<i|r,Ot=l+e}else Ft=1<<l|n<<i|r,Ot=e}function rs(e){e.return!==null&&(xn(e,1),cf(e,1,0))}function is(e){for(;e===ul;)ul=Qn[--qn],Qn[qn]=null,cl=Qn[--qn],Qn[qn]=null;for(;e===Nn;)Nn=st[--ut],st[ut]=null,Ot=st[--ut],st[ut]=null,Ft=st[--ut],st[ut]=null}var it=null,nt=null,de=!1,xt=null;function df(e,t){var n=dt(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function Pu(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,it=e,nt=an(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,it=e,nt=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=Nn!==null?{id:Ft,overflow:Ot}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=dt(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,it=e,nt=null,!0):!1;default:return!1}}function la(e){return(e.mode&1)!==0&&(e.flags&128)===0}function oa(e){if(de){var t=nt;if(t){var n=t;if(!Pu(e,t)){if(la(e))throw Error(M(418));t=an(n.nextSibling);var r=it;t&&Pu(e,t)?df(r,n):(e.flags=e.flags&-4097|2,de=!1,it=e)}}else{if(la(e))throw Error(M(418));e.flags=e.flags&-4097|2,de=!1,it=e}}}function Iu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;it=e}function _i(e){if(e!==it)return!1;if(!de)return Iu(e),de=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!ta(e.type,e.memoizedProps)),t&&(t=nt)){if(la(e))throw ff(),Error(M(418));for(;t;)df(e,t),t=an(t.nextSibling)}if(Iu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(M(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){nt=an(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}nt=null}}else nt=it?an(e.stateNode.nextSibling):null;return!0}function ff(){for(var e=nt;e;)e=an(e.nextSibling)}function ar(){nt=it=null,de=!1}function ls(e){xt===null?xt=[e]:xt.push(e)}var Xm=Wt.ReactCurrentBatchConfig;function Cr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(M(309));var r=n.stateNode}if(!r)throw Error(M(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(M(284));if(!n._owner)throw Error(M(290,e))}return e}function Ni(e,t){throw e=Object.prototype.toString.call(t),Error(M(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Au(e){var t=e._init;return t(e._payload)}function pf(e){function t(h,v){if(e){var y=h.deletions;y===null?(h.deletions=[v],h.flags|=16):y.push(v)}}function n(h,v){if(!e)return null;for(;v!==null;)t(h,v),v=v.sibling;return null}function r(h,v){for(h=new Map;v!==null;)v.key!==null?h.set(v.key,v):h.set(v.index,v),v=v.sibling;return h}function i(h,v){return h=dn(h,v),h.index=0,h.sibling=null,h}function l(h,v,y){return h.index=y,e?(y=h.alternate,y!==null?(y=y.index,y<v?(h.flags|=2,v):y):(h.flags|=2,v)):(h.flags|=1048576,v)}function o(h){return e&&h.alternate===null&&(h.flags|=2),h}function a(h,v,y,b){return v===null||v.tag!==6?(v=go(y,h.mode,b),v.return=h,v):(v=i(v,y),v.return=h,v)}function s(h,v,y,b){var _=y.type;return _===Bn?d(h,v,y.props.children,b,y.key):v!==null&&(v.elementType===_||typeof _=="object"&&_!==null&&_.$$typeof===Gt&&Au(_)===v.type)?(b=i(v,y.props),b.ref=Cr(h,v,y),b.return=h,b):(b=Ki(y.type,y.key,y.props,null,h.mode,b),b.ref=Cr(h,v,y),b.return=h,b)}function c(h,v,y,b){return v===null||v.tag!==4||v.stateNode.containerInfo!==y.containerInfo||v.stateNode.implementation!==y.implementation?(v=vo(y,h.mode,b),v.return=h,v):(v=i(v,y.children||[]),v.return=h,v)}function d(h,v,y,b,_){return v===null||v.tag!==7?(v=En(y,h.mode,b,_),v.return=h,v):(v=i(v,y),v.return=h,v)}function f(h,v,y){if(typeof v=="string"&&v!==""||typeof v=="number")return v=go(""+v,h.mode,y),v.return=h,v;if(typeof v=="object"&&v!==null){switch(v.$$typeof){case vi:return y=Ki(v.type,v.key,v.props,null,h.mode,y),y.ref=Cr(h,null,v),y.return=h,y;case On:return v=vo(v,h.mode,y),v.return=h,v;case Gt:var b=v._init;return f(h,b(v._payload),y)}if(zr(v)||xr(v))return v=En(v,h.mode,y,null),v.return=h,v;Ni(h,v)}return null}function g(h,v,y,b){var _=v!==null?v.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return _!==null?null:a(h,v,""+y,b);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case vi:return y.key===_?s(h,v,y,b):null;case On:return y.key===_?c(h,v,y,b):null;case Gt:return _=y._init,g(h,v,_(y._payload),b)}if(zr(y)||xr(y))return _!==null?null:d(h,v,y,b,null);Ni(h,y)}return null}function p(h,v,y,b,_){if(typeof b=="string"&&b!==""||typeof b=="number")return h=h.get(y)||null,a(v,h,""+b,_);if(typeof b=="object"&&b!==null){switch(b.$$typeof){case vi:return h=h.get(b.key===null?y:b.key)||null,s(v,h,b,_);case On:return h=h.get(b.key===null?y:b.key)||null,c(v,h,b,_);case Gt:var S=b._init;return p(h,v,y,S(b._payload),_)}if(zr(b)||xr(b))return h=h.get(y)||null,d(v,h,b,_,null);Ni(v,b)}return null}function k(h,v,y,b){for(var _=null,S=null,E=v,L=v=0,D=null;E!==null&&L<y.length;L++){E.index>L?(D=E,E=null):D=E.sibling;var P=g(h,E,y[L],b);if(P===null){E===null&&(E=D);break}e&&E&&P.alternate===null&&t(h,E),v=l(P,v,L),S===null?_=P:S.sibling=P,S=P,E=D}if(L===y.length)return n(h,E),de&&xn(h,L),_;if(E===null){for(;L<y.length;L++)E=f(h,y[L],b),E!==null&&(v=l(E,v,L),S===null?_=E:S.sibling=E,S=E);return de&&xn(h,L),_}for(E=r(h,E);L<y.length;L++)D=p(E,h,L,y[L],b),D!==null&&(e&&D.alternate!==null&&E.delete(D.key===null?L:D.key),v=l(D,v,L),S===null?_=D:S.sibling=D,S=D);return e&&E.forEach(function(j){return t(h,j)}),de&&xn(h,L),_}function w(h,v,y,b){var _=xr(y);if(typeof _!="function")throw Error(M(150));if(y=_.call(y),y==null)throw Error(M(151));for(var S=_=null,E=v,L=v=0,D=null,P=y.next();E!==null&&!P.done;L++,P=y.next()){E.index>L?(D=E,E=null):D=E.sibling;var j=g(h,E,P.value,b);if(j===null){E===null&&(E=D);break}e&&E&&j.alternate===null&&t(h,E),v=l(j,v,L),S===null?_=j:S.sibling=j,S=j,E=D}if(P.done)return n(h,E),de&&xn(h,L),_;if(E===null){for(;!P.done;L++,P=y.next())P=f(h,P.value,b),P!==null&&(v=l(P,v,L),S===null?_=P:S.sibling=P,S=P);return de&&xn(h,L),_}for(E=r(h,E);!P.done;L++,P=y.next())P=p(E,h,L,P.value,b),P!==null&&(e&&P.alternate!==null&&E.delete(P.key===null?L:P.key),v=l(P,v,L),S===null?_=P:S.sibling=P,S=P);return e&&E.forEach(function(T){return t(h,T)}),de&&xn(h,L),_}function I(h,v,y,b){if(typeof y=="object"&&y!==null&&y.type===Bn&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case vi:e:{for(var _=y.key,S=v;S!==null;){if(S.key===_){if(_=y.type,_===Bn){if(S.tag===7){n(h,S.sibling),v=i(S,y.props.children),v.return=h,h=v;break e}}else if(S.elementType===_||typeof _=="object"&&_!==null&&_.$$typeof===Gt&&Au(_)===S.type){n(h,S.sibling),v=i(S,y.props),v.ref=Cr(h,S,y),v.return=h,h=v;break e}n(h,S);break}else t(h,S);S=S.sibling}y.type===Bn?(v=En(y.props.children,h.mode,b,y.key),v.return=h,h=v):(b=Ki(y.type,y.key,y.props,null,h.mode,b),b.ref=Cr(h,v,y),b.return=h,h=b)}return o(h);case On:e:{for(S=y.key;v!==null;){if(v.key===S)if(v.tag===4&&v.stateNode.containerInfo===y.containerInfo&&v.stateNode.implementation===y.implementation){n(h,v.sibling),v=i(v,y.children||[]),v.return=h,h=v;break e}else{n(h,v);break}else t(h,v);v=v.sibling}v=vo(y,h.mode,b),v.return=h,h=v}return o(h);case Gt:return S=y._init,I(h,v,S(y._payload),b)}if(zr(y))return k(h,v,y,b);if(xr(y))return w(h,v,y,b);Ni(h,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,v!==null&&v.tag===6?(n(h,v.sibling),v=i(v,y),v.return=h,h=v):(n(h,v),v=go(y,h.mode,b),v.return=h,h=v),o(h)):n(h,v)}return I}var sr=pf(!0),hf=pf(!1),dl=mn(null),fl=null,Kn=null,os=null;function as(){os=Kn=fl=null}function ss(e){var t=dl.current;ce(dl),e._currentValue=t}function aa(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function tr(e,t){fl=e,os=Kn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(Qe=!0),e.firstContext=null)}function pt(e){var t=e._currentValue;if(os!==e)if(e={context:e,memoizedValue:t,next:null},Kn===null){if(fl===null)throw Error(M(308));Kn=e,fl.dependencies={lanes:0,firstContext:e}}else Kn=Kn.next=e;return t}var bn=null;function us(e){bn===null?bn=[e]:bn.push(e)}function mf(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,us(t)):(n.next=i.next,i.next=n),t.interleaved=n,Ht(e,r)}function Ht(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Jt=!1;function cs(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function gf(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Bt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function sn(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,te&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Ht(e,n)}return i=r.interleaved,i===null?(t.next=t,us(r)):(t.next=i.next,i.next=t),r.interleaved=t,Ht(e,n)}function Ui(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ya(e,n)}}function Mu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function pl(e,t,n,r){var i=e.updateQueue;Jt=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,c=s.next;s.next=null,o===null?l=c:o.next=c,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=s))}if(l!==null){var f=i.baseState;o=0,d=c=s=null,a=l;do{var g=a.lane,p=a.eventTime;if((r&g)===g){d!==null&&(d=d.next={eventTime:p,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var k=e,w=a;switch(g=t,p=n,w.tag){case 1:if(k=w.payload,typeof k=="function"){f=k.call(p,f,g);break e}f=k;break e;case 3:k.flags=k.flags&-65537|128;case 0:if(k=w.payload,g=typeof k=="function"?k.call(p,f,g):k,g==null)break e;f=me({},f,g);break e;case 2:Jt=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,g=i.effects,g===null?i.effects=[a]:g.push(a))}else p={eventTime:p,lane:g,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=p,s=f):d=d.next=p,o|=g;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;g=a,a=g.next,g.next=null,i.lastBaseUpdate=g,i.shared.pending=null}}while(!0);if(d===null&&(s=f),i.baseState=s,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);zn|=o,e.lanes=o,e.memoizedState=f}}function Du(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(M(191,i));i.call(r)}}}var pi={},zt=mn(pi),ni=mn(pi),ri=mn(pi);function Cn(e){if(e===pi)throw Error(M(174));return e}function ds(e,t){switch(ae(ri,t),ae(ni,e),ae(zt,pi),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:$o(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=$o(t,e)}ce(zt),ae(zt,t)}function ur(){ce(zt),ce(ni),ce(ri)}function vf(e){Cn(ri.current);var t=Cn(zt.current),n=$o(t,e.type);t!==n&&(ae(ni,e),ae(zt,n))}function fs(e){ni.current===e&&(ce(zt),ce(ni))}var pe=mn(0);function hl(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var uo=[];function ps(){for(var e=0;e<uo.length;e++)uo[e]._workInProgressVersionPrimary=null;uo.length=0}var Hi=Wt.ReactCurrentDispatcher,co=Wt.ReactCurrentBatchConfig,Tn=0,he=null,we=null,Ce=null,ml=!1,Fr=!1,ii=0,Gm=0;function Le(){throw Error(M(321))}function hs(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!St(e[n],t[n]))return!1;return!0}function ms(e,t,n,r,i,l){if(Tn=l,he=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Hi.current=e===null||e.memoizedState===null?tg:ng,e=n(r,i),Fr){l=0;do{if(Fr=!1,ii=0,25<=l)throw Error(M(301));l+=1,Ce=we=null,t.updateQueue=null,Hi.current=rg,e=n(r,i)}while(Fr)}if(Hi.current=gl,t=we!==null&&we.next!==null,Tn=0,Ce=we=he=null,ml=!1,t)throw Error(M(300));return e}function gs(){var e=ii!==0;return ii=0,e}function jt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return Ce===null?he.memoizedState=Ce=e:Ce=Ce.next=e,Ce}function ht(){if(we===null){var e=he.alternate;e=e!==null?e.memoizedState:null}else e=we.next;var t=Ce===null?he.memoizedState:Ce.next;if(t!==null)Ce=t,we=e;else{if(e===null)throw Error(M(310));we=e,e={memoizedState:we.memoizedState,baseState:we.baseState,baseQueue:we.baseQueue,queue:we.queue,next:null},Ce===null?he.memoizedState=Ce=e:Ce=Ce.next=e}return Ce}function li(e,t){return typeof t=="function"?t(e):t}function fo(e){var t=ht(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=we,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,c=l;do{var d=c.lane;if((Tn&d)===d)s!==null&&(s=s.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var f={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};s===null?(a=s=f,o=r):s=s.next=f,he.lanes|=d,zn|=d}c=c.next}while(c!==null&&c!==l);s===null?o=r:s.next=a,St(r,t.memoizedState)||(Qe=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,he.lanes|=l,zn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function po(e){var t=ht(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);St(l,t.memoizedState)||(Qe=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function yf(){}function xf(e,t){var n=he,r=ht(),i=t(),l=!St(r.memoizedState,i);if(l&&(r.memoizedState=i,Qe=!0),r=r.queue,vs(Sf.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||Ce!==null&&Ce.memoizedState.tag&1){if(n.flags|=2048,oi(9,wf.bind(null,n,r,i,t),void 0,null),je===null)throw Error(M(349));Tn&30||kf(n,t,i)}return i}function kf(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=he.updateQueue,t===null?(t={lastEffect:null,stores:null},he.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function wf(e,t,n,r){t.value=n,t.getSnapshot=r,bf(t)&&Cf(e)}function Sf(e,t,n){return n(function(){bf(t)&&Cf(e)})}function bf(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!St(e,n)}catch{return!0}}function Cf(e){var t=Ht(e,1);t!==null&&wt(t,e,1,-1)}function Ru(e){var t=jt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:li,lastRenderedState:e},t.queue=e,e=e.dispatch=eg.bind(null,he,e),[t.memoizedState,e]}function oi(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=he.updateQueue,t===null?(t={lastEffect:null,stores:null},he.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function jf(){return ht().memoizedState}function Vi(e,t,n,r){var i=jt();he.flags|=e,i.memoizedState=oi(1|t,n,void 0,r===void 0?null:r)}function Pl(e,t,n,r){var i=ht();r=r===void 0?null:r;var l=void 0;if(we!==null){var o=we.memoizedState;if(l=o.destroy,r!==null&&hs(r,o.deps)){i.memoizedState=oi(t,n,l,r);return}}he.flags|=e,i.memoizedState=oi(1|t,n,l,r)}function Fu(e,t){return Vi(8390656,8,e,t)}function vs(e,t){return Pl(2048,8,e,t)}function Ef(e,t){return Pl(4,2,e,t)}function _f(e,t){return Pl(4,4,e,t)}function Nf(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function Tf(e,t,n){return n=n!=null?n.concat([e]):null,Pl(4,4,Nf.bind(null,t,e),n)}function ys(){}function zf(e,t){var n=ht();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&hs(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function Lf(e,t){var n=ht();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&hs(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function Pf(e,t,n){return Tn&21?(St(n,t)||(n=Dd(),he.lanes|=n,zn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,Qe=!0),e.memoizedState=n)}function Jm(e,t){var n=re;re=n!==0&&4>n?n:4,e(!0);var r=co.transition;co.transition={};try{e(!1),t()}finally{re=n,co.transition=r}}function If(){return ht().memoizedState}function Zm(e,t,n){var r=cn(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},Af(e))Mf(t,n);else if(n=mf(e,t,n,r),n!==null){var i=Oe();wt(n,e,r,i),Df(n,t,r)}}function eg(e,t,n){var r=cn(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(Af(e))Mf(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,St(a,o)){var s=t.interleaved;s===null?(i.next=i,us(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=mf(e,t,i,r),n!==null&&(i=Oe(),wt(n,e,r,i),Df(n,t,r))}}function Af(e){var t=e.alternate;return e===he||t!==null&&t===he}function Mf(e,t){Fr=ml=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function Df(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ya(e,n)}}var gl={readContext:pt,useCallback:Le,useContext:Le,useEffect:Le,useImperativeHandle:Le,useInsertionEffect:Le,useLayoutEffect:Le,useMemo:Le,useReducer:Le,useRef:Le,useState:Le,useDebugValue:Le,useDeferredValue:Le,useTransition:Le,useMutableSource:Le,useSyncExternalStore:Le,useId:Le,unstable_isNewReconciler:!1},tg={readContext:pt,useCallback:function(e,t){return jt().memoizedState=[e,t===void 0?null:t],e},useContext:pt,useEffect:Fu,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Vi(4194308,4,Nf.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Vi(4194308,4,e,t)},useInsertionEffect:function(e,t){return Vi(4,2,e,t)},useMemo:function(e,t){var n=jt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=jt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Zm.bind(null,he,e),[r.memoizedState,e]},useRef:function(e){var t=jt();return e={current:e},t.memoizedState=e},useState:Ru,useDebugValue:ys,useDeferredValue:function(e){return jt().memoizedState=e},useTransition:function(){var e=Ru(!1),t=e[0];return e=Jm.bind(null,e[1]),jt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=he,i=jt();if(de){if(n===void 0)throw Error(M(407));n=n()}else{if(n=t(),je===null)throw Error(M(349));Tn&30||kf(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Fu(Sf.bind(null,r,l,e),[e]),r.flags|=2048,oi(9,wf.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=jt(),t=je.identifierPrefix;if(de){var n=Ot,r=Ft;n=(r&~(1<<32-kt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=ii++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Gm++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},ng={readContext:pt,useCallback:zf,useContext:pt,useEffect:vs,useImperativeHandle:Tf,useInsertionEffect:Ef,useLayoutEffect:_f,useMemo:Lf,useReducer:fo,useRef:jf,useState:function(){return fo(li)},useDebugValue:ys,useDeferredValue:function(e){var t=ht();return Pf(t,we.memoizedState,e)},useTransition:function(){var e=fo(li)[0],t=ht().memoizedState;return[e,t]},useMutableSource:yf,useSyncExternalStore:xf,useId:If,unstable_isNewReconciler:!1},rg={readContext:pt,useCallback:zf,useContext:pt,useEffect:vs,useImperativeHandle:Tf,useInsertionEffect:Ef,useLayoutEffect:_f,useMemo:Lf,useReducer:po,useRef:jf,useState:function(){return po(li)},useDebugValue:ys,useDeferredValue:function(e){var t=ht();return we===null?t.memoizedState=e:Pf(t,we.memoizedState,e)},useTransition:function(){var e=po(li)[0],t=ht().memoizedState;return[e,t]},useMutableSource:yf,useSyncExternalStore:xf,useId:If,unstable_isNewReconciler:!1};function vt(e,t){if(e&&e.defaultProps){t=me({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function sa(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:me({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var Il={isMounted:function(e){return(e=e._reactInternals)?In(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Oe(),i=cn(e),l=Bt(r,i);l.payload=t,n!=null&&(l.callback=n),t=sn(e,l,i),t!==null&&(wt(t,e,i,r),Ui(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Oe(),i=cn(e),l=Bt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=sn(e,l,i),t!==null&&(wt(t,e,i,r),Ui(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Oe(),r=cn(e),i=Bt(n,r);i.tag=2,t!=null&&(i.callback=t),t=sn(e,i,r),t!==null&&(wt(t,e,r,n),Ui(t,e,r))}};function Ou(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Jr(n,r)||!Jr(i,l):!0}function Rf(e,t,n){var r=!1,i=pn,l=t.contextType;return typeof l=="object"&&l!==null?l=pt(l):(i=Ke(t)?_n:Me.current,r=t.contextTypes,l=(r=r!=null)?or(e,i):pn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=Il,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function Bu(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&Il.enqueueReplaceState(t,t.state,null)}function ua(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},cs(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=pt(l):(l=Ke(t)?_n:Me.current,i.context=or(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(sa(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&Il.enqueueReplaceState(i,i.state,null),pl(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function cr(e,t){try{var n="",r=t;do n+=Lh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function ho(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function ca(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var ig=typeof WeakMap=="function"?WeakMap:Map;function Ff(e,t,n){n=Bt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){yl||(yl=!0,ka=r),ca(e,t)},n}function Of(e,t,n){n=Bt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){ca(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){ca(e,t),typeof r!="function"&&(un===null?un=new Set([this]):un.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function $u(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new ig;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=yg.bind(null,e,t,n),t.then(e,e))}function Uu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Hu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Bt(-1,1),t.tag=2,sn(n,t,1))),n.lanes|=1),e)}var lg=Wt.ReactCurrentOwner,Qe=!1;function Fe(e,t,n,r){t.child=e===null?hf(t,null,n,r):sr(t,e.child,n,r)}function Vu(e,t,n,r,i){n=n.render;var l=t.ref;return tr(t,i),r=ms(e,t,n,r,l,i),n=gs(),e!==null&&!Qe?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Vt(e,t,i)):(de&&n&&rs(t),t.flags|=1,Fe(e,t,r,i),t.child)}function Wu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!Es(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,Bf(e,t,l,r,i)):(e=Ki(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Jr,n(o,r)&&e.ref===t.ref)return Vt(e,t,i)}return t.flags|=1,e=dn(l,r),e.ref=t.ref,e.return=t,t.child=e}function Bf(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Jr(l,r)&&e.ref===t.ref)if(Qe=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(Qe=!0);else return t.lanes=e.lanes,Vt(e,t,i)}return da(e,t,n,r,i)}function $f(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},ae(Xn,tt),tt|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,ae(Xn,tt),tt|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,ae(Xn,tt),tt|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,ae(Xn,tt),tt|=r;return Fe(e,t,i,n),t.child}function Uf(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function da(e,t,n,r,i){var l=Ke(n)?_n:Me.current;return l=or(t,l),tr(t,i),n=ms(e,t,n,r,l,i),r=gs(),e!==null&&!Qe?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Vt(e,t,i)):(de&&r&&rs(t),t.flags|=1,Fe(e,t,n,i),t.child)}function Qu(e,t,n,r,i){if(Ke(n)){var l=!0;sl(t)}else l=!1;if(tr(t,i),t.stateNode===null)Wi(e,t),Rf(t,n,r),ua(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=pt(c):(c=Ke(n)?_n:Me.current,c=or(t,c));var d=n.getDerivedStateFromProps,f=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";f||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==c)&&Bu(t,o,r,c),Jt=!1;var g=t.memoizedState;o.state=g,pl(t,r,o,i),s=t.memoizedState,a!==r||g!==s||qe.current||Jt?(typeof d=="function"&&(sa(t,n,d,r),s=t.memoizedState),(a=Jt||Ou(t,n,a,r,g,s,c))?(f||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,gf(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:vt(t.type,a),o.props=c,f=t.pendingProps,g=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=pt(s):(s=Ke(n)?_n:Me.current,s=or(t,s));var p=n.getDerivedStateFromProps;(d=typeof p=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==f||g!==s)&&Bu(t,o,r,s),Jt=!1,g=t.memoizedState,o.state=g,pl(t,r,o,i);var k=t.memoizedState;a!==f||g!==k||qe.current||Jt?(typeof p=="function"&&(sa(t,n,p,r),k=t.memoizedState),(c=Jt||Ou(t,n,c,r,g,k,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,k,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,k,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=k),o.props=r,o.state=k,o.context=s,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=1024),r=!1)}return fa(e,t,n,r,l,i)}function fa(e,t,n,r,i,l){Uf(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&Lu(t,n,!1),Vt(e,t,l);r=t.stateNode,lg.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=sr(t,e.child,null,l),t.child=sr(t,null,a,l)):Fe(e,t,a,l),t.memoizedState=r.state,i&&Lu(t,n,!0),t.child}function Hf(e){var t=e.stateNode;t.pendingContext?zu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&zu(e,t.context,!1),ds(e,t.containerInfo)}function qu(e,t,n,r,i){return ar(),ls(i),t.flags|=256,Fe(e,t,n,r),t.child}var pa={dehydrated:null,treeContext:null,retryLane:0};function ha(e){return{baseLanes:e,cachePool:null,transitions:null}}function Vf(e,t,n){var r=t.pendingProps,i=pe.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),ae(pe,i&1),e===null)return oa(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Dl(o,r,0,null),e=En(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=ha(n),t.memoizedState=pa,e):xs(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return og(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=dn(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=dn(a,l):(l=En(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?ha(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=pa,r}return l=e.child,e=l.sibling,r=dn(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function xs(e,t){return t=Dl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function Ti(e,t,n,r){return r!==null&&ls(r),sr(t,e.child,null,n),e=xs(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function og(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=ho(Error(M(422))),Ti(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Dl({mode:"visible",children:r.children},i,0,null),l=En(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&sr(t,e.child,null,o),t.child.memoizedState=ha(o),t.memoizedState=pa,l);if(!(t.mode&1))return Ti(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(M(419)),r=ho(l,r,void 0),Ti(e,t,o,r)}if(a=(o&e.childLanes)!==0,Qe||a){if(r=je,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Ht(e,i),wt(r,e,i,-1))}return js(),r=ho(Error(M(421))),Ti(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=xg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,nt=an(i.nextSibling),it=t,de=!0,xt=null,e!==null&&(st[ut++]=Ft,st[ut++]=Ot,st[ut++]=Nn,Ft=e.id,Ot=e.overflow,Nn=t),t=xs(t,r.children),t.flags|=4096,t)}function Ku(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),aa(e.return,t,n)}function mo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Wf(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(Fe(e,t,r.children,n),r=pe.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Ku(e,n,t);else if(e.tag===19)Ku(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(ae(pe,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&hl(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),mo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&hl(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}mo(t,!0,n,null,l);break;case"together":mo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Wi(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Vt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),zn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(M(153));if(t.child!==null){for(e=t.child,n=dn(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=dn(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function ag(e,t,n){switch(t.tag){case 3:Hf(t),ar();break;case 5:vf(t);break;case 1:Ke(t.type)&&sl(t);break;case 4:ds(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;ae(dl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(ae(pe,pe.current&1),t.flags|=128,null):n&t.child.childLanes?Vf(e,t,n):(ae(pe,pe.current&1),e=Vt(e,t,n),e!==null?e.sibling:null);ae(pe,pe.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Wf(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),ae(pe,pe.current),r)break;return null;case 22:case 23:return t.lanes=0,$f(e,t,n)}return Vt(e,t,n)}var Qf,ma,qf,Kf;Qf=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};ma=function(){};qf=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,Cn(zt.current);var l=null;switch(n){case"input":i=Ro(e,i),r=Ro(e,r),l=[];break;case"select":i=me({},i,{value:void 0}),r=me({},r,{value:void 0}),l=[];break;case"textarea":i=Bo(e,i),r=Bo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=ol)}Uo(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(Wr.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var s=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&s!==a&&(s!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(c,n)),n=s;else c==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(c,s)):c==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(c,""+s):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(Wr.hasOwnProperty(c)?(s!=null&&c==="onScroll"&&ue("scroll",e),l||a===s||(l=[])):(l=l||[]).push(c,s))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};Kf=function(e,t,n,r){n!==r&&(t.flags|=4)};function jr(e,t){if(!de)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function Pe(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function sg(e,t,n){var r=t.pendingProps;switch(is(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return Pe(t),null;case 1:return Ke(t.type)&&al(),Pe(t),null;case 3:return r=t.stateNode,ur(),ce(qe),ce(Me),ps(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(_i(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,xt!==null&&(ba(xt),xt=null))),ma(e,t),Pe(t),null;case 5:fs(t);var i=Cn(ri.current);if(n=t.type,e!==null&&t.stateNode!=null)qf(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(M(166));return Pe(t),null}if(e=Cn(zt.current),_i(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[_t]=t,r[ti]=l,e=(t.mode&1)!==0,n){case"dialog":ue("cancel",r),ue("close",r);break;case"iframe":case"object":case"embed":ue("load",r);break;case"video":case"audio":for(i=0;i<Pr.length;i++)ue(Pr[i],r);break;case"source":ue("error",r);break;case"img":case"image":case"link":ue("error",r),ue("load",r);break;case"details":ue("toggle",r);break;case"input":ru(r,l),ue("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},ue("invalid",r);break;case"textarea":lu(r,l),ue("invalid",r)}Uo(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&Ei(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&Ei(r.textContent,a,e),i=["children",""+a]):Wr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&ue("scroll",r)}switch(n){case"input":yi(r),iu(r,l,!0);break;case"textarea":yi(r),ou(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=ol)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=wd(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[_t]=t,e[ti]=r,Qf(e,t,!1,!1),t.stateNode=e;e:{switch(o=Ho(n,r),n){case"dialog":ue("cancel",e),ue("close",e),i=r;break;case"iframe":case"object":case"embed":ue("load",e),i=r;break;case"video":case"audio":for(i=0;i<Pr.length;i++)ue(Pr[i],e);i=r;break;case"source":ue("error",e),i=r;break;case"img":case"image":case"link":ue("error",e),ue("load",e),i=r;break;case"details":ue("toggle",e),i=r;break;case"input":ru(e,r),i=Ro(e,r),ue("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=me({},r,{value:void 0}),ue("invalid",e);break;case"textarea":lu(e,r),i=Bo(e,r),ue("invalid",e);break;default:i=r}Uo(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?Cd(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&Sd(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Qr(e,s):typeof s=="number"&&Qr(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Wr.hasOwnProperty(l)?s!=null&&l==="onScroll"&&ue("scroll",e):s!=null&&Ha(e,l,s,o))}switch(n){case"input":yi(e),iu(e,r,!1);break;case"textarea":yi(e),ou(e);break;case"option":r.value!=null&&e.setAttribute("value",""+fn(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Gn(e,!!r.multiple,l,!1):r.defaultValue!=null&&Gn(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=ol)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return Pe(t),null;case 6:if(e&&t.stateNode!=null)Kf(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(M(166));if(n=Cn(ri.current),Cn(zt.current),_i(t)){if(r=t.stateNode,n=t.memoizedProps,r[_t]=t,(l=r.nodeValue!==n)&&(e=it,e!==null))switch(e.tag){case 3:Ei(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&Ei(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[_t]=t,t.stateNode=r}return Pe(t),null;case 13:if(ce(pe),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(de&&nt!==null&&t.mode&1&&!(t.flags&128))ff(),ar(),t.flags|=98560,l=!1;else if(l=_i(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(M(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(M(317));l[_t]=t}else ar(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;Pe(t),l=!1}else xt!==null&&(ba(xt),xt=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||pe.current&1?Se===0&&(Se=3):js())),t.updateQueue!==null&&(t.flags|=4),Pe(t),null);case 4:return ur(),ma(e,t),e===null&&Zr(t.stateNode.containerInfo),Pe(t),null;case 10:return ss(t.type._context),Pe(t),null;case 17:return Ke(t.type)&&al(),Pe(t),null;case 19:if(ce(pe),l=t.memoizedState,l===null)return Pe(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)jr(l,!1);else{if(Se!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=hl(e),o!==null){for(t.flags|=128,jr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return ae(pe,pe.current&1|2),t.child}e=e.sibling}l.tail!==null&&ye()>dr&&(t.flags|=128,r=!0,jr(l,!1),t.lanes=4194304)}else{if(!r)if(e=hl(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),jr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!de)return Pe(t),null}else 2*ye()-l.renderingStartTime>dr&&n!==1073741824&&(t.flags|=128,r=!0,jr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=ye(),t.sibling=null,n=pe.current,ae(pe,r?n&1|2:n&1),t):(Pe(t),null);case 22:case 23:return Cs(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?tt&1073741824&&(Pe(t),t.subtreeFlags&6&&(t.flags|=8192)):Pe(t),null;case 24:return null;case 25:return null}throw Error(M(156,t.tag))}function ug(e,t){switch(is(t),t.tag){case 1:return Ke(t.type)&&al(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return ur(),ce(qe),ce(Me),ps(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return fs(t),null;case 13:if(ce(pe),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(M(340));ar()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return ce(pe),null;case 4:return ur(),null;case 10:return ss(t.type._context),null;case 22:case 23:return Cs(),null;case 24:return null;default:return null}}var zi=!1,Ae=!1,cg=typeof WeakSet=="function"?WeakSet:Set,$=null;function Yn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){ge(e,t,r)}else n.current=null}function ga(e,t,n){try{n()}catch(r){ge(e,t,r)}}var Yu=!1;function dg(e,t){if(Zo=rl,e=Jd(),ns(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,c=0,d=0,f=e,g=null;t:for(;;){for(var p;f!==n||i!==0&&f.nodeType!==3||(a=o+i),f!==l||r!==0&&f.nodeType!==3||(s=o+r),f.nodeType===3&&(o+=f.nodeValue.length),(p=f.firstChild)!==null;)g=f,f=p;for(;;){if(f===e)break t;if(g===n&&++c===i&&(a=o),g===l&&++d===r&&(s=o),(p=f.nextSibling)!==null)break;f=g,g=f.parentNode}f=p}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(ea={focusedElem:e,selectionRange:n},rl=!1,$=t;$!==null;)if(t=$,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,$=e;else for(;$!==null;){t=$;try{var k=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(k!==null){var w=k.memoizedProps,I=k.memoizedState,h=t.stateNode,v=h.getSnapshotBeforeUpdate(t.elementType===t.type?w:vt(t.type,w),I);h.__reactInternalSnapshotBeforeUpdate=v}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(M(163))}}catch(b){ge(t,t.return,b)}if(e=t.sibling,e!==null){e.return=t.return,$=e;break}$=t.return}return k=Yu,Yu=!1,k}function Or(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&ga(t,n,l)}i=i.next}while(i!==r)}}function Al(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function va(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Yf(e){var t=e.alternate;t!==null&&(e.alternate=null,Yf(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[_t],delete t[ti],delete t[ra],delete t[qm],delete t[Km])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Xf(e){return e.tag===5||e.tag===3||e.tag===4}function Xu(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Xf(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function ya(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=ol));else if(r!==4&&(e=e.child,e!==null))for(ya(e,t,n),e=e.sibling;e!==null;)ya(e,t,n),e=e.sibling}function xa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(xa(e,t,n),e=e.sibling;e!==null;)xa(e,t,n),e=e.sibling}var Ne=null,yt=!1;function Kt(e,t,n){for(n=n.child;n!==null;)Gf(e,t,n),n=n.sibling}function Gf(e,t,n){if(Tt&&typeof Tt.onCommitFiberUnmount=="function")try{Tt.onCommitFiberUnmount(El,n)}catch{}switch(n.tag){case 5:Ae||Yn(n,t);case 6:var r=Ne,i=yt;Ne=null,Kt(e,t,n),Ne=r,yt=i,Ne!==null&&(yt?(e=Ne,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):Ne.removeChild(n.stateNode));break;case 18:Ne!==null&&(yt?(e=Ne,n=n.stateNode,e.nodeType===8?ao(e.parentNode,n):e.nodeType===1&&ao(e,n),Xr(e)):ao(Ne,n.stateNode));break;case 4:r=Ne,i=yt,Ne=n.stateNode.containerInfo,yt=!0,Kt(e,t,n),Ne=r,yt=i;break;case 0:case 11:case 14:case 15:if(!Ae&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&ga(n,t,o),i=i.next}while(i!==r)}Kt(e,t,n);break;case 1:if(!Ae&&(Yn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){ge(n,t,a)}Kt(e,t,n);break;case 21:Kt(e,t,n);break;case 22:n.mode&1?(Ae=(r=Ae)||n.memoizedState!==null,Kt(e,t,n),Ae=r):Kt(e,t,n);break;default:Kt(e,t,n)}}function Gu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new cg),t.forEach(function(r){var i=kg.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function gt(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:Ne=a.stateNode,yt=!1;break e;case 3:Ne=a.stateNode.containerInfo,yt=!0;break e;case 4:Ne=a.stateNode.containerInfo,yt=!0;break e}a=a.return}if(Ne===null)throw Error(M(160));Gf(l,o,i),Ne=null,yt=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(c){ge(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Jf(t,e),t=t.sibling}function Jf(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(gt(t,e),bt(e),r&4){try{Or(3,e,e.return),Al(3,e)}catch(w){ge(e,e.return,w)}try{Or(5,e,e.return)}catch(w){ge(e,e.return,w)}}break;case 1:gt(t,e),bt(e),r&512&&n!==null&&Yn(n,n.return);break;case 5:if(gt(t,e),bt(e),r&512&&n!==null&&Yn(n,n.return),e.flags&32){var i=e.stateNode;try{Qr(i,"")}catch(w){ge(e,e.return,w)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&xd(i,l),Ho(a,o);var c=Ho(a,l);for(o=0;o<s.length;o+=2){var d=s[o],f=s[o+1];d==="style"?Cd(i,f):d==="dangerouslySetInnerHTML"?Sd(i,f):d==="children"?Qr(i,f):Ha(i,d,f,c)}switch(a){case"input":Fo(i,l);break;case"textarea":kd(i,l);break;case"select":var g=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var p=l.value;p!=null?Gn(i,!!l.multiple,p,!1):g!==!!l.multiple&&(l.defaultValue!=null?Gn(i,!!l.multiple,l.defaultValue,!0):Gn(i,!!l.multiple,l.multiple?[]:"",!1))}i[ti]=l}catch(w){ge(e,e.return,w)}}break;case 6:if(gt(t,e),bt(e),r&4){if(e.stateNode===null)throw Error(M(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(w){ge(e,e.return,w)}}break;case 3:if(gt(t,e),bt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Xr(t.containerInfo)}catch(w){ge(e,e.return,w)}break;case 4:gt(t,e),bt(e);break;case 13:gt(t,e),bt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(Ss=ye())),r&4&&Gu(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Ae=(c=Ae)||d,gt(t,e),Ae=c):gt(t,e),bt(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for($=e,d=e.child;d!==null;){for(f=$=d;$!==null;){switch(g=$,p=g.child,g.tag){case 0:case 11:case 14:case 15:Or(4,g,g.return);break;case 1:Yn(g,g.return);var k=g.stateNode;if(typeof k.componentWillUnmount=="function"){r=g,n=g.return;try{t=r,k.props=t.memoizedProps,k.state=t.memoizedState,k.componentWillUnmount()}catch(w){ge(r,n,w)}}break;case 5:Yn(g,g.return);break;case 22:if(g.memoizedState!==null){Zu(f);continue}}p!==null?(p.return=g,$=p):Zu(f)}d=d.sibling}e:for(d=null,f=e;;){if(f.tag===5){if(d===null){d=f;try{i=f.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=f.stateNode,s=f.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=bd("display",o))}catch(w){ge(e,e.return,w)}}}else if(f.tag===6){if(d===null)try{f.stateNode.nodeValue=c?"":f.memoizedProps}catch(w){ge(e,e.return,w)}}else if((f.tag!==22&&f.tag!==23||f.memoizedState===null||f===e)&&f.child!==null){f.child.return=f,f=f.child;continue}if(f===e)break e;for(;f.sibling===null;){if(f.return===null||f.return===e)break e;d===f&&(d=null),f=f.return}d===f&&(d=null),f.sibling.return=f.return,f=f.sibling}}break;case 19:gt(t,e),bt(e),r&4&&Gu(e);break;case 21:break;default:gt(t,e),bt(e)}}function bt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Xf(n)){var r=n;break e}n=n.return}throw Error(M(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Qr(i,""),r.flags&=-33);var l=Xu(e);xa(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Xu(e);ya(e,a,o);break;default:throw Error(M(161))}}catch(s){ge(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function fg(e,t,n){$=e,Zf(e)}function Zf(e,t,n){for(var r=(e.mode&1)!==0;$!==null;){var i=$,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||zi;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||Ae;a=zi;var c=Ae;if(zi=o,(Ae=s)&&!c)for($=i;$!==null;)o=$,s=o.child,o.tag===22&&o.memoizedState!==null?ec(i):s!==null?(s.return=o,$=s):ec(i);for(;l!==null;)$=l,Zf(l),l=l.sibling;$=i,zi=a,Ae=c}Ju(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,$=l):Ju(e)}}function Ju(e){for(;$!==null;){var t=$;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Ae||Al(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Ae)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:vt(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Du(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Du(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var f=d.dehydrated;f!==null&&Xr(f)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(M(163))}Ae||t.flags&512&&va(t)}catch(g){ge(t,t.return,g)}}if(t===e){$=null;break}if(n=t.sibling,n!==null){n.return=t.return,$=n;break}$=t.return}}function Zu(e){for(;$!==null;){var t=$;if(t===e){$=null;break}var n=t.sibling;if(n!==null){n.return=t.return,$=n;break}$=t.return}}function ec(e){for(;$!==null;){var t=$;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Al(4,t)}catch(s){ge(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){ge(t,i,s)}}var l=t.return;try{va(t)}catch(s){ge(t,l,s)}break;case 5:var o=t.return;try{va(t)}catch(s){ge(t,o,s)}}}catch(s){ge(t,t.return,s)}if(t===e){$=null;break}var a=t.sibling;if(a!==null){a.return=t.return,$=a;break}$=t.return}}var pg=Math.ceil,vl=Wt.ReactCurrentDispatcher,ks=Wt.ReactCurrentOwner,ft=Wt.ReactCurrentBatchConfig,te=0,je=null,ke=null,Te=0,tt=0,Xn=mn(0),Se=0,ai=null,zn=0,Ml=0,ws=0,Br=null,We=null,Ss=0,dr=1/0,Dt=null,yl=!1,ka=null,un=null,Li=!1,nn=null,xl=0,$r=0,wa=null,Qi=-1,qi=0;function Oe(){return te&6?ye():Qi!==-1?Qi:Qi=ye()}function cn(e){return e.mode&1?te&2&&Te!==0?Te&-Te:Xm.transition!==null?(qi===0&&(qi=Dd()),qi):(e=re,e!==0||(e=window.event,e=e===void 0?16:Hd(e.type)),e):1}function wt(e,t,n,r){if(50<$r)throw $r=0,wa=null,Error(M(185));ci(e,n,r),(!(te&2)||e!==je)&&(e===je&&(!(te&2)&&(Ml|=n),Se===4&&en(e,Te)),Ye(e,r),n===1&&te===0&&!(t.mode&1)&&(dr=ye()+500,Ll&&gn()))}function Ye(e,t){var n=e.callbackNode;Xh(e,t);var r=nl(e,e===je?Te:0);if(r===0)n!==null&&uu(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&uu(n),t===1)e.tag===0?Ym(tc.bind(null,e)):uf(tc.bind(null,e)),Wm(function(){!(te&6)&&gn()}),n=null;else{switch(Rd(r)){case 1:n=Ka;break;case 4:n=Ad;break;case 16:n=tl;break;case 536870912:n=Md;break;default:n=tl}n=ap(n,ep.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function ep(e,t){if(Qi=-1,qi=0,te&6)throw Error(M(327));var n=e.callbackNode;if(nr()&&e.callbackNode!==n)return null;var r=nl(e,e===je?Te:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=kl(e,r);else{t=r;var i=te;te|=2;var l=np();(je!==e||Te!==t)&&(Dt=null,dr=ye()+500,jn(e,t));do try{gg();break}catch(a){tp(e,a)}while(!0);as(),vl.current=l,te=i,ke!==null?t=0:(je=null,Te=0,t=Se)}if(t!==0){if(t===2&&(i=Ko(e),i!==0&&(r=i,t=Sa(e,i))),t===1)throw n=ai,jn(e,0),en(e,r),Ye(e,ye()),n;if(t===6)en(e,r);else{if(i=e.current.alternate,!(r&30)&&!hg(i)&&(t=kl(e,r),t===2&&(l=Ko(e),l!==0&&(r=l,t=Sa(e,l))),t===1))throw n=ai,jn(e,0),en(e,r),Ye(e,ye()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(M(345));case 2:kn(e,We,Dt);break;case 3:if(en(e,r),(r&130023424)===r&&(t=Ss+500-ye(),10<t)){if(nl(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Oe(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=na(kn.bind(null,e,We,Dt),t);break}kn(e,We,Dt);break;case 4:if(en(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-kt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=ye()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*pg(r/1960))-r,10<r){e.timeoutHandle=na(kn.bind(null,e,We,Dt),r);break}kn(e,We,Dt);break;case 5:kn(e,We,Dt);break;default:throw Error(M(329))}}}return Ye(e,ye()),e.callbackNode===n?ep.bind(null,e):null}function Sa(e,t){var n=Br;return e.current.memoizedState.isDehydrated&&(jn(e,t).flags|=256),e=kl(e,t),e!==2&&(t=We,We=n,t!==null&&ba(t)),e}function ba(e){We===null?We=e:We.push.apply(We,e)}function hg(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!St(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function en(e,t){for(t&=~ws,t&=~Ml,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-kt(t),r=1<<n;e[n]=-1,t&=~r}}function tc(e){if(te&6)throw Error(M(327));nr();var t=nl(e,0);if(!(t&1))return Ye(e,ye()),null;var n=kl(e,t);if(e.tag!==0&&n===2){var r=Ko(e);r!==0&&(t=r,n=Sa(e,r))}if(n===1)throw n=ai,jn(e,0),en(e,t),Ye(e,ye()),n;if(n===6)throw Error(M(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,kn(e,We,Dt),Ye(e,ye()),null}function bs(e,t){var n=te;te|=1;try{return e(t)}finally{te=n,te===0&&(dr=ye()+500,Ll&&gn())}}function Ln(e){nn!==null&&nn.tag===0&&!(te&6)&&nr();var t=te;te|=1;var n=ft.transition,r=re;try{if(ft.transition=null,re=1,e)return e()}finally{re=r,ft.transition=n,te=t,!(te&6)&&gn()}}function Cs(){tt=Xn.current,ce(Xn)}function jn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,Vm(n)),ke!==null)for(n=ke.return;n!==null;){var r=n;switch(is(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&al();break;case 3:ur(),ce(qe),ce(Me),ps();break;case 5:fs(r);break;case 4:ur();break;case 13:ce(pe);break;case 19:ce(pe);break;case 10:ss(r.type._context);break;case 22:case 23:Cs()}n=n.return}if(je=e,ke=e=dn(e.current,null),Te=tt=t,Se=0,ai=null,ws=Ml=zn=0,We=Br=null,bn!==null){for(t=0;t<bn.length;t++)if(n=bn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}bn=null}return e}function tp(e,t){do{var n=ke;try{if(as(),Hi.current=gl,ml){for(var r=he.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}ml=!1}if(Tn=0,Ce=we=he=null,Fr=!1,ii=0,ks.current=null,n===null||n.return===null){Se=1,ai=t,ke=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=Te,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var c=s,d=a,f=d.tag;if(!(d.mode&1)&&(f===0||f===11||f===15)){var g=d.alternate;g?(d.updateQueue=g.updateQueue,d.memoizedState=g.memoizedState,d.lanes=g.lanes):(d.updateQueue=null,d.memoizedState=null)}var p=Uu(o);if(p!==null){p.flags&=-257,Hu(p,o,a,l,t),p.mode&1&&$u(l,c,t),t=p,s=c;var k=t.updateQueue;if(k===null){var w=new Set;w.add(s),t.updateQueue=w}else k.add(s);break e}else{if(!(t&1)){$u(l,c,t),js();break e}s=Error(M(426))}}else if(de&&a.mode&1){var I=Uu(o);if(I!==null){!(I.flags&65536)&&(I.flags|=256),Hu(I,o,a,l,t),ls(cr(s,a));break e}}l=s=cr(s,a),Se!==4&&(Se=2),Br===null?Br=[l]:Br.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var h=Ff(l,s,t);Mu(l,h);break e;case 1:a=s;var v=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof v.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(un===null||!un.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var b=Of(l,a,t);Mu(l,b);break e}}l=l.return}while(l!==null)}ip(n)}catch(_){t=_,ke===n&&n!==null&&(ke=n=n.return);continue}break}while(!0)}function np(){var e=vl.current;return vl.current=gl,e===null?gl:e}function js(){(Se===0||Se===3||Se===2)&&(Se=4),je===null||!(zn&268435455)&&!(Ml&268435455)||en(je,Te)}function kl(e,t){var n=te;te|=2;var r=np();(je!==e||Te!==t)&&(Dt=null,jn(e,t));do try{mg();break}catch(i){tp(e,i)}while(!0);if(as(),te=n,vl.current=r,ke!==null)throw Error(M(261));return je=null,Te=0,Se}function mg(){for(;ke!==null;)rp(ke)}function gg(){for(;ke!==null&&!$h();)rp(ke)}function rp(e){var t=op(e.alternate,e,tt);e.memoizedProps=e.pendingProps,t===null?ip(e):ke=t,ks.current=null}function ip(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=ug(n,t),n!==null){n.flags&=32767,ke=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{Se=6,ke=null;return}}else if(n=sg(n,t,tt),n!==null){ke=n;return}if(t=t.sibling,t!==null){ke=t;return}ke=t=e}while(t!==null);Se===0&&(Se=5)}function kn(e,t,n){var r=re,i=ft.transition;try{ft.transition=null,re=1,vg(e,t,n,r)}finally{ft.transition=i,re=r}return null}function vg(e,t,n,r){do nr();while(nn!==null);if(te&6)throw Error(M(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(M(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Gh(e,l),e===je&&(ke=je=null,Te=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||Li||(Li=!0,ap(tl,function(){return nr(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=ft.transition,ft.transition=null;var o=re;re=1;var a=te;te|=4,ks.current=null,dg(e,n),Jf(n,e),Rm(ea),rl=!!Zo,ea=Zo=null,e.current=n,fg(n),Uh(),te=a,re=o,ft.transition=l}else e.current=n;if(Li&&(Li=!1,nn=e,xl=i),l=e.pendingLanes,l===0&&(un=null),Wh(n.stateNode),Ye(e,ye()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(yl)throw yl=!1,e=ka,ka=null,e;return xl&1&&e.tag!==0&&nr(),l=e.pendingLanes,l&1?e===wa?$r++:($r=0,wa=e):$r=0,gn(),null}function nr(){if(nn!==null){var e=Rd(xl),t=ft.transition,n=re;try{if(ft.transition=null,re=16>e?16:e,nn===null)var r=!1;else{if(e=nn,nn=null,xl=0,te&6)throw Error(M(331));var i=te;for(te|=4,$=e.current;$!==null;){var l=$,o=l.child;if($.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var c=a[s];for($=c;$!==null;){var d=$;switch(d.tag){case 0:case 11:case 15:Or(8,d,l)}var f=d.child;if(f!==null)f.return=d,$=f;else for(;$!==null;){d=$;var g=d.sibling,p=d.return;if(Yf(d),d===c){$=null;break}if(g!==null){g.return=p,$=g;break}$=p}}}var k=l.alternate;if(k!==null){var w=k.child;if(w!==null){k.child=null;do{var I=w.sibling;w.sibling=null,w=I}while(w!==null)}}$=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,$=o;else e:for(;$!==null;){if(l=$,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Or(9,l,l.return)}var h=l.sibling;if(h!==null){h.return=l.return,$=h;break e}$=l.return}}var v=e.current;for($=v;$!==null;){o=$;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,$=y;else e:for(o=v;$!==null;){if(a=$,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Al(9,a)}}catch(_){ge(a,a.return,_)}if(a===o){$=null;break e}var b=a.sibling;if(b!==null){b.return=a.return,$=b;break e}$=a.return}}if(te=i,gn(),Tt&&typeof Tt.onPostCommitFiberRoot=="function")try{Tt.onPostCommitFiberRoot(El,e)}catch{}r=!0}return r}finally{re=n,ft.transition=t}}return!1}function nc(e,t,n){t=cr(n,t),t=Ff(e,t,1),e=sn(e,t,1),t=Oe(),e!==null&&(ci(e,1,t),Ye(e,t))}function ge(e,t,n){if(e.tag===3)nc(e,e,n);else for(;t!==null;){if(t.tag===3){nc(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(un===null||!un.has(r))){e=cr(n,e),e=Of(t,e,1),t=sn(t,e,1),e=Oe(),t!==null&&(ci(t,1,e),Ye(t,e));break}}t=t.return}}function yg(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Oe(),e.pingedLanes|=e.suspendedLanes&n,je===e&&(Te&n)===n&&(Se===4||Se===3&&(Te&130023424)===Te&&500>ye()-Ss?jn(e,0):ws|=n),Ye(e,t)}function lp(e,t){t===0&&(e.mode&1?(t=wi,wi<<=1,!(wi&130023424)&&(wi=4194304)):t=1);var n=Oe();e=Ht(e,t),e!==null&&(ci(e,t,n),Ye(e,n))}function xg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),lp(e,n)}function kg(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(M(314))}r!==null&&r.delete(t),lp(e,n)}var op;op=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||qe.current)Qe=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return Qe=!1,ag(e,t,n);Qe=!!(e.flags&131072)}else Qe=!1,de&&t.flags&1048576&&cf(t,cl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Wi(e,t),e=t.pendingProps;var i=or(t,Me.current);tr(t,n),i=ms(null,t,r,e,i,n);var l=gs();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Ke(r)?(l=!0,sl(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,cs(t),i.updater=Il,t.stateNode=i,i._reactInternals=t,ua(t,r,e,n),t=fa(null,t,r,!0,l,n)):(t.tag=0,de&&l&&rs(t),Fe(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Wi(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=Sg(r),e=vt(r,e),i){case 0:t=da(null,t,r,e,n);break e;case 1:t=Qu(null,t,r,e,n);break e;case 11:t=Vu(null,t,r,e,n);break e;case 14:t=Wu(null,t,r,vt(r.type,e),n);break e}throw Error(M(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:vt(r,i),da(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:vt(r,i),Qu(e,t,r,i,n);case 3:e:{if(Hf(t),e===null)throw Error(M(387));r=t.pendingProps,l=t.memoizedState,i=l.element,gf(e,t),pl(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=cr(Error(M(423)),t),t=qu(e,t,r,n,i);break e}else if(r!==i){i=cr(Error(M(424)),t),t=qu(e,t,r,n,i);break e}else for(nt=an(t.stateNode.containerInfo.firstChild),it=t,de=!0,xt=null,n=hf(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(ar(),r===i){t=Vt(e,t,n);break e}Fe(e,t,r,n)}t=t.child}return t;case 5:return vf(t),e===null&&oa(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,ta(r,i)?o=null:l!==null&&ta(r,l)&&(t.flags|=32),Uf(e,t),Fe(e,t,o,n),t.child;case 6:return e===null&&oa(t),null;case 13:return Vf(e,t,n);case 4:return ds(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=sr(t,null,r,n):Fe(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:vt(r,i),Vu(e,t,r,i,n);case 7:return Fe(e,t,t.pendingProps,n),t.child;case 8:return Fe(e,t,t.pendingProps.children,n),t.child;case 12:return Fe(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,ae(dl,r._currentValue),r._currentValue=o,l!==null)if(St(l.value,o)){if(l.children===i.children&&!qe.current){t=Vt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Bt(-1,n&-n),s.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?s.next=s:(s.next=d.next,d.next=s),c.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),aa(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(M(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),aa(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}Fe(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,tr(t,n),i=pt(i),r=r(i),t.flags|=1,Fe(e,t,r,n),t.child;case 14:return r=t.type,i=vt(r,t.pendingProps),i=vt(r.type,i),Wu(e,t,r,i,n);case 15:return Bf(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:vt(r,i),Wi(e,t),t.tag=1,Ke(r)?(e=!0,sl(t)):e=!1,tr(t,n),Rf(t,r,i),ua(t,r,i,n),fa(null,t,r,!0,e,n);case 19:return Wf(e,t,n);case 22:return $f(e,t,n)}throw Error(M(156,t.tag))};function ap(e,t){return Id(e,t)}function wg(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function dt(e,t,n,r){return new wg(e,t,n,r)}function Es(e){return e=e.prototype,!(!e||!e.isReactComponent)}function Sg(e){if(typeof e=="function")return Es(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Wa)return 11;if(e===Qa)return 14}return 2}function dn(e,t){var n=e.alternate;return n===null?(n=dt(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Ki(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")Es(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Bn:return En(n.children,i,l,t);case Va:o=8,i|=8;break;case Io:return e=dt(12,n,t,i|2),e.elementType=Io,e.lanes=l,e;case Ao:return e=dt(13,n,t,i),e.elementType=Ao,e.lanes=l,e;case Mo:return e=dt(19,n,t,i),e.elementType=Mo,e.lanes=l,e;case gd:return Dl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case hd:o=10;break e;case md:o=9;break e;case Wa:o=11;break e;case Qa:o=14;break e;case Gt:o=16,r=null;break e}throw Error(M(130,e==null?e:typeof e,""))}return t=dt(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function En(e,t,n,r){return e=dt(7,e,r,t),e.lanes=n,e}function Dl(e,t,n,r){return e=dt(22,e,r,t),e.elementType=gd,e.lanes=n,e.stateNode={isHidden:!1},e}function go(e,t,n){return e=dt(6,e,null,t),e.lanes=n,e}function vo(e,t,n){return t=dt(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function bg(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Xl(0),this.expirationTimes=Xl(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Xl(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function _s(e,t,n,r,i,l,o,a,s){return e=new bg(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=dt(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},cs(l),e}function Cg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:On,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function sp(e){if(!e)return pn;e=e._reactInternals;e:{if(In(e)!==e||e.tag!==1)throw Error(M(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Ke(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(M(171))}if(e.tag===1){var n=e.type;if(Ke(n))return sf(e,n,t)}return t}function up(e,t,n,r,i,l,o,a,s){return e=_s(n,r,!0,e,i,l,o,a,s),e.context=sp(null),n=e.current,r=Oe(),i=cn(n),l=Bt(r,i),l.callback=t??null,sn(n,l,i),e.current.lanes=i,ci(e,i,r),Ye(e,r),e}function Rl(e,t,n,r){var i=t.current,l=Oe(),o=cn(i);return n=sp(n),t.context===null?t.context=n:t.pendingContext=n,t=Bt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=sn(i,t,o),e!==null&&(wt(e,i,o,l),Ui(e,i,o)),o}function wl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function rc(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function Ns(e,t){rc(e,t),(e=e.alternate)&&rc(e,t)}function jg(){return null}var cp=typeof reportError=="function"?reportError:function(e){console.error(e)};function Ts(e){this._internalRoot=e}Fl.prototype.render=Ts.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(M(409));Rl(e,t,null,null)};Fl.prototype.unmount=Ts.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;Ln(function(){Rl(null,e,null,null)}),t[Ut]=null}};function Fl(e){this._internalRoot=e}Fl.prototype.unstable_scheduleHydration=function(e){if(e){var t=Bd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Zt.length&&t!==0&&t<Zt[n].priority;n++);Zt.splice(n,0,e),n===0&&Ud(e)}};function zs(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Ol(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function ic(){}function Eg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=wl(o);l.call(c)}}var o=up(t,r,e,0,null,!1,!1,"",ic);return e._reactRootContainer=o,e[Ut]=o.current,Zr(e.nodeType===8?e.parentNode:e),Ln(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=wl(s);a.call(c)}}var s=_s(e,0,!1,null,null,!1,!1,"",ic);return e._reactRootContainer=s,e[Ut]=s.current,Zr(e.nodeType===8?e.parentNode:e),Ln(function(){Rl(t,s,n,r)}),s}function Bl(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=wl(o);a.call(s)}}Rl(t,o,e,i)}else o=Eg(n,t,e,i,r);return wl(o)}Fd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=Lr(t.pendingLanes);n!==0&&(Ya(t,n|1),Ye(t,ye()),!(te&6)&&(dr=ye()+500,gn()))}break;case 13:Ln(function(){var r=Ht(e,1);if(r!==null){var i=Oe();wt(r,e,1,i)}}),Ns(e,1)}};Xa=function(e){if(e.tag===13){var t=Ht(e,134217728);if(t!==null){var n=Oe();wt(t,e,134217728,n)}Ns(e,134217728)}};Od=function(e){if(e.tag===13){var t=cn(e),n=Ht(e,t);if(n!==null){var r=Oe();wt(n,e,t,r)}Ns(e,t)}};Bd=function(){return re};$d=function(e,t){var n=re;try{return re=e,t()}finally{re=n}};Wo=function(e,t,n){switch(t){case"input":if(Fo(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=zl(r);if(!i)throw Error(M(90));yd(r),Fo(r,i)}}}break;case"textarea":kd(e,n);break;case"select":t=n.value,t!=null&&Gn(e,!!n.multiple,t,!1)}};_d=bs;Nd=Ln;var _g={usingClientEntryPoint:!1,Events:[fi,Vn,zl,jd,Ed,bs]},Er={findFiberByHostInstance:Sn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},Ng={bundleType:Er.bundleType,version:Er.version,rendererPackageName:Er.rendererPackageName,rendererConfig:Er.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Wt.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=Ld(e),e===null?null:e.stateNode},findFiberByHostInstance:Er.findFiberByHostInstance||jg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var Pi=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!Pi.isDisabled&&Pi.supportsFiber)try{El=Pi.inject(Ng),Tt=Pi}catch{}}ot.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=_g;ot.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!zs(t))throw Error(M(200));return Cg(e,t,null,n)};ot.createRoot=function(e,t){if(!zs(e))throw Error(M(299));var n=!1,r="",i=cp;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=_s(e,1,!1,null,null,n,!1,r,i),e[Ut]=t.current,Zr(e.nodeType===8?e.parentNode:e),new Ts(t)};ot.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(M(188)):(e=Object.keys(e).join(","),Error(M(268,e)));return e=Ld(t),e=e===null?null:e.stateNode,e};ot.flushSync=function(e){return Ln(e)};ot.hydrate=function(e,t,n){if(!Ol(t))throw Error(M(200));return Bl(null,e,t,!0,n)};ot.hydrateRoot=function(e,t,n){if(!zs(e))throw Error(M(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=cp;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=up(t,null,e,1,n??null,i,!1,l,o),e[Ut]=t.current,Zr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new Fl(t)};ot.render=function(e,t,n){if(!Ol(t))throw Error(M(200));return Bl(null,e,t,!1,n)};ot.unmountComponentAtNode=function(e){if(!Ol(e))throw Error(M(40));return e._reactRootContainer?(Ln(function(){Bl(null,null,e,!1,function(){e._reactRootContainer=null,e[Ut]=null})}),!0):!1};ot.unstable_batchedUpdates=bs;ot.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Ol(n))throw Error(M(200));if(e==null||e._reactInternals===void 0)throw Error(M(38));return Bl(e,t,n,!1,r)};ot.version="18.3.1-next-f1338f8080-20240426";function dp(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(dp)}catch(e){console.error(e)}}dp(),cd.exports=ot;var Tg=cd.exports,lc=Tg;Lo.createRoot=lc.createRoot,Lo.hydrateRoot=lc.hydrateRoot;const zg=new Set(["user","human"]);function Lg(e){return e?zg.has(e.toLowerCase()):!1}function fp(e){return Lg(e)?"You (Human)":e}const Pg="",Ig=({selection:e,onSelect:t,onRefresh:n})=>{const[r,i]=F.useState(null),[l,o]=F.useState(new Set(["all"])),[a,s]=F.useState(!0),[c,d]=F.useState(null),f=async()=>{try{const v=await fetch(`${Pg}/api/hierarchy`);if(!v.ok)throw new Error("Failed to fetch hierarchy");const y=await v.json();i(y),d(null)}catch(v){d(v instanceof Error?v.message:"Unknown error")}finally{s(!1)}};F.useEffect(()=>{f();const v=setInterval(f,5e3);return()=>clearInterval(v)},[]);const g=v=>{o(y=>{const b=new Set(y);return b.has(v)?b.delete(v):b.add(v),b})},p=v=>{var y;if(v.type==="root")t({type:"overview"});else if(v.type==="agent")t({type:"agent",agentId:v.id});else if(v.type==="thread"){const b=(y=r==null?void 0:r.root.children)==null?void 0:y.find(_=>{var S;return(S=_.children)==null?void 0:S.some(E=>E.id===v.id)});t({type:"thread",agentId:b==null?void 0:b.id,threadId:v.id})}},k=v=>v.type==="root"&&e.type==="overview"||v.type==="agent"&&e.type==="agent"&&e.agentId===v.id||v.type==="thread"&&e.threadId===v.id,w=v=>!v||v.length===0?null:u.jsx("span",{className:"badges",children:v.map((y,b)=>u.jsxs("span",{className:`badge badge-${y.type}`,title:`${y.count} ${y.type}`,children:[y.type==="pending"&&"⏳",y.type==="unread"&&"📬",y.type==="running"&&"▶️",y.count]},b))}),I=v=>{if(!v)return null;const y={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsx("span",{className:"status-indicator",style:{backgroundColor:y[v]||y.idle},title:v})},h=(v,y=0)=>{const b=l.has(v.id),_=v.children&&v.children.length>0,S=k(v);return u.jsxs("div",{className:"tree-node",children:[u.jsxs("div",{className:`tree-node-content ${S?"selected":""} ${v.type}`,style:{paddingLeft:`${y*16+8}px`},onClick:()=>p(v),children:[_&&u.jsx("span",{className:`expand-icon ${b?"expanded":""}`,onClick:E=>{E.stopPropagation(),g(v.id)},children:b?"▼":"▶"}),!_&&u.jsx("span",{className:"expand-icon-placeholder"}),v.type==="agent"&&I(v.status),u.jsx("span",{className:"node-label",children:v.type==="agent"?fp(v.id):v.label}),w(v.badges)]}),_&&b&&u.jsx("div",{className:"tree-children",children:v.children.map(E=>h(E,y+1))})]},v.id)};return a&&!r?u.jsx("div",{className:"hierarchy-tree loading",children:"Loading..."}):c?u.jsxs("div",{className:"hierarchy-tree error",children:[u.jsxs("p",{children:["Error: ",c]}),u.jsx("button",{onClick:f,children:"Retry"})]}):u.jsxs("div",{className:"hierarchy-tree",children:[u.jsxs("div",{className:"tree-header",children:[u.jsx("h3",{children:"Agents"}),u.jsx("button",{className:"refresh-btn",onClick:()=>{f(),n==null||n()},title:"Refresh",children:"↻"})]}),u.jsx("div",{className:"tree-content",children:r&&h(r.root)}),r&&u.jsx("div",{className:"tree-footer",children:u.jsxs("div",{className:"aggregate-stats",children:[u.jsxs("span",{title:"Total agents",children:[r.aggregate.total_agents," agents"]}),u.jsxs("span",{title:"Active",children:[r.aggregate.active_agents," active"]}),r.aggregate.pending_approvals>0&&u.jsxs("span",{className:"pending",title:"Pending approvals",children:[r.aggregate.pending_approvals," pending"]})]})})]})},Ag="_card_1d3of_1",Mg="_compact_1d3of_9",Dg="_title_1d3of_13",Rg="_metricsGrid_1d3of_20",Fg="_metricItem_1d3of_26",Og="_metricLabel_1d3of_32",Bg="_metricValue_1d3of_39",$g="_cost_1d3of_46",Ug="_averages_1d3of_50",Hg="_averagesLabel_1d3of_61",Vg="_avgItem_1d3of_65",Wg="_compactRow_1d3of_72",Qg="_compactLabel_1d3of_80",qg="_compactValue_1d3of_84",Kg="_loading_1d3of_91",Yg="_error_1d3of_97",Xg="_errorText_1d3of_101",Y={card:Ag,compact:Mg,title:Dg,metricsGrid:Rg,metricItem:Fg,metricLabel:Og,metricValue:Bg,cost:$g,averages:Ug,averagesLabel:Hg,avgItem:Vg,compactRow:Wg,compactLabel:Qg,compactValue:qg,loading:Kg,error:Yg,errorText:Xg};function oc(e){return e<1e3?`${e}ms`:e<6e4?`${(e/1e3).toFixed(1)}s`:e<36e5?`${(e/6e4).toFixed(1)}m`:`${(e/36e5).toFixed(1)}h`}function Rn(e){return e.toLocaleString()}function yo(e){return e===0?"$0.00":e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`}function Ca({scopeType:e,scopeId:t="",title:n,compact:r=!1}){const[i,l]=F.useState(null),[o,a]=F.useState(!0),[s,c]=F.useState(null),d=F.useCallback(async()=>{try{let g="/api/metrics";e!=="global"&&(g=`/api/metrics/${e}/${t}`);const p=await fetch(g);if(!p.ok)throw new Error(`Failed to fetch metrics: ${p.status}`);const k=await p.json();l(k),c(null)}catch(g){c(g instanceof Error?g.message:"Failed to load metrics")}finally{a(!1)}},[e,t]);if(F.useEffect(()=>{d();const g=setInterval(d,3e4);return()=>clearInterval(g)},[d]),o)return u.jsx("div",{className:`${Y.card} ${r?Y.compact:""}`,children:u.jsx("div",{className:Y.loading,children:"Loading metrics..."})});if(s)return u.jsx("div",{className:`${Y.card} ${r?Y.compact:""} ${Y.error}`,children:u.jsx("div",{className:Y.errorText,children:s})});if(!i)return null;const f=n||(e==="global"?"Global Metrics":e==="agent"?`Agent: ${t}`:`Thread: ${t.slice(0,12)}...`);return r?u.jsx("div",{className:`${Y.card} ${Y.compact}`,children:u.jsxs("div",{className:Y.compactRow,children:[u.jsx("span",{className:Y.compactLabel,children:"Runs:"}),u.jsx("span",{className:Y.compactValue,children:Rn(i.total_runs)}),u.jsx("span",{className:Y.compactLabel,children:"Tokens:"}),u.jsx("span",{className:Y.compactValue,children:Rn(i.total_tokens)}),u.jsx("span",{className:Y.compactLabel,children:"Cost:"}),u.jsx("span",{className:Y.compactValue,children:yo(i.total_cost)})]})}):u.jsxs("div",{className:Y.card,children:[u.jsx("h3",{className:Y.title,children:f}),u.jsxs("div",{className:Y.metricsGrid,children:[u.jsxs("div",{className:Y.metricItem,children:[u.jsx("span",{className:Y.metricLabel,children:"Total Runs"}),u.jsx("span",{className:Y.metricValue,children:Rn(i.total_runs)})]}),u.jsxs("div",{className:Y.metricItem,children:[u.jsx("span",{className:Y.metricLabel,children:"Total Tokens"}),u.jsx("span",{className:Y.metricValue,children:Rn(i.total_tokens)})]}),u.jsxs("div",{className:Y.metricItem,children:[u.jsx("span",{className:Y.metricLabel,children:"Total Cost"}),u.jsx("span",{className:`${Y.metricValue} ${Y.cost}`,children:yo(i.total_cost)})]}),u.jsxs("div",{className:Y.metricItem,children:[u.jsx("span",{className:Y.metricLabel,children:"Total Duration"}),u.jsx("span",{className:Y.metricValue,children:oc(i.total_duration_ms)})]}),u.jsxs("div",{className:Y.metricItem,children:[u.jsx("span",{className:Y.metricLabel,children:"Files Modified"}),u.jsx("span",{className:Y.metricValue,children:Rn(i.total_files_modified)})]})]}),i.total_runs>0&&u.jsxs("div",{className:Y.averages,children:[u.jsx("span",{className:Y.averagesLabel,children:"Averages per run:"}),u.jsxs("span",{className:Y.avgItem,children:[Rn(Math.round(i.avg_tokens_per_run))," tokens"]}),u.jsx("span",{className:Y.avgItem,children:yo(i.avg_cost_per_run)}),u.jsx("span",{className:Y.avgItem,children:oc(Math.round(i.avg_duration_per_run))})]})]})}const Gg="_container_1q26w_1",Jg="_title_1q26w_9",Zg="_header_1q26w_16",ev="_metricLabel_1q26w_25",tv="_total_1q26w_31",nv="_chart_1q26w_37",rv="_barContainer_1q26w_46",iv="_barWrapper_1q26w_55",lv="_bar_1q26w_46",ov="_barValue_1q26w_80",av="_label_1q26w_89",sv="_loading_1q26w_98",uv="_error_1q26w_99",cv="_empty_1q26w_100",_e={container:Gg,title:Jg,header:Zg,metricLabel:ev,total:tv,chart:nv,barContainer:rv,barWrapper:iv,bar:lv,barValue:ov,label:av,loading:sv,error:uv,empty:cv};function Sl({scopeType:e,scopeId:t,period:n="hour",limit:r=24,metric:i="cost",title:l}){const[o,a]=F.useState([]),[s,c]=F.useState(!0),[d,f]=F.useState(null);F.useEffect(()=>{const y=async()=>{try{c(!0);const _=await fetch(`/api/metrics/trends/${e}/${t}?period=${n}&limit=${r}`);if(!_.ok)throw new Error("Failed to fetch trends");const S=await _.json();a(S||[]),f(null)}catch(_){f(_ instanceof Error?_.message:"Unknown error")}finally{c(!1)}};y();const b=setInterval(y,6e4);return()=>clearInterval(b)},[e,t,n,r]);const g=y=>{switch(i){case"cost":return y.cost;case"tokens":return y.tokens;case"duration":return y.duration_ms/1e3;case"runs":return y.runs;default:return y.cost}},p=y=>{switch(i){case"cost":return`$${y.toFixed(2)}`;case"tokens":return y>=1e3?`${(y/1e3).toFixed(1)}k`:y.toString();case"duration":return`${y.toFixed(1)}s`;case"runs":return y.toString();default:return y.toFixed(2)}},k=y=>{const b=new Date(y);return n==="minute"?b.toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}):n==="hour"?b.toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}):b.toLocaleDateString([],{month:"short",day:"numeric"})},w=()=>{switch(i){case"cost":return"Cost ($)";case"tokens":return"Tokens";case"duration":return"Duration (s)";case"runs":return"Runs";default:return""}};if(s&&o.length===0)return u.jsx("div",{className:_e.container,children:u.jsx("div",{className:_e.loading,children:"Loading trends..."})});if(d)return u.jsx("div",{className:_e.container,children:u.jsx("div",{className:_e.error,children:d})});if(o.length===0)return u.jsx("div",{className:_e.container,children:u.jsx("div",{className:_e.empty,children:"No data available"})});const I=o.map(g),h=Math.max(...I,1),v=I.reduce((y,b)=>y+b,0);return u.jsxs("div",{className:_e.container,children:[l&&u.jsx("div",{className:_e.title,children:l}),u.jsxs("div",{className:_e.header,children:[u.jsx("span",{className:_e.metricLabel,children:w()}),u.jsxs("span",{className:_e.total,children:["Total: ",p(v)]})]}),u.jsx("div",{className:_e.chart,children:o.map((y,b)=>{const _=g(y),S=_/h*100;return u.jsxs("div",{className:_e.barContainer,children:[u.jsx("div",{className:_e.barWrapper,children:u.jsx("div",{className:_e.bar,style:{height:`${Math.max(S,2)}%`},title:`${k(y.period_start)}: ${p(_)}`,children:S>30&&u.jsx("span",{className:_e.barValue,children:p(_)})})}),b%Math.ceil(o.length/6)===0&&u.jsx("span",{className:_e.label,children:k(y.period_start)})]},y.period_start)})})]})}const Ze=({title:e,value:t,color:n="default",small:r})=>u.jsxs("div",{className:`stat-card stat-${n} ${r?"stat-small":""}`,children:[u.jsx("div",{className:"stat-value",children:t}),u.jsx("div",{className:"stat-title",children:e})]}),dv=e=>{if(e<1e3)return`${e}ms`;const t=e/1e3;if(t<60)return`${t.toFixed(1)}s`;const n=Math.floor(t/60),r=(t%60).toFixed(0);return`${n}m ${r}s`},fv=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,Ii=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}k`:e.toString(),pv=({agent:e,onClick:t})=>{var o,a,s,c,d;const n=((o=e.children)==null?void 0:o.length)||0,r=((s=(a=e.badges)==null?void 0:a.find(f=>f.type==="pending"))==null?void 0:s.count)||0,i=((d=(c=e.badges)==null?void 0:c.find(f=>f.type==="running"))==null?void 0:d.count)||0,l={active:"#22c55e",running:"#22c55e",pending:"#f59e0b",idle:"#6b7280",error:"#ef4444"};return u.jsxs("div",{className:"agent-card",onClick:t,children:[u.jsxs("div",{className:"agent-card-header",children:[u.jsx("span",{className:"agent-status-dot",style:{backgroundColor:l[e.status||"idle"]}}),u.jsx("span",{className:"agent-name",children:fp(e.id)})]}),u.jsxs("div",{className:"agent-card-stats",children:[u.jsxs("span",{className:"agent-stat",children:[u.jsx("span",{className:"agent-stat-value",children:n}),u.jsx("span",{className:"agent-stat-label",children:"threads"})]}),r>0&&u.jsxs("span",{className:"agent-stat pending",children:[u.jsx("span",{className:"agent-stat-value",children:r}),u.jsx("span",{className:"agent-stat-label",children:"pending"})]}),i>0&&u.jsxs("span",{className:"agent-stat running",children:[u.jsx("span",{className:"agent-stat-value",children:i}),u.jsx("span",{className:"agent-stat-label",children:"running"})]})]})]})},hv=({aggregate:e,agents:t,onSelectAgent:n})=>{const r=e.execution,i=r&&r.total_executions>0,l=i?Math.round(r.successful_executions/r.total_executions*100):0;return u.jsxs("div",{className:"all-agents-overview",children:[u.jsx("div",{className:"overview-header",children:u.jsx("h2",{children:"All Agents Overview"})}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Ze,{title:"Total Agents",value:e.total_agents}),u.jsx(Ze,{title:"Active",value:e.active_agents,color:"green"}),u.jsx(Ze,{title:"Pending Approvals",value:e.pending_approvals,color:"orange"}),u.jsx(Ze,{title:"Total Threads",value:e.total_threads,color:"blue"})]}),u.jsxs("div",{className:"metrics-section",children:[u.jsx("h3",{children:"Usage Metrics (Today)"}),u.jsx(Ca,{scopeType:"global",title:"Global Metrics"})]}),u.jsxs("div",{className:"trends-section",children:[u.jsx("h3",{children:"Usage Trends (Last 24 Hours)"}),u.jsxs("div",{className:"trends-grid",children:[u.jsx(Sl,{scopeType:"global",scopeId:"",period:"hour",limit:24,metric:"cost",title:"Cost"}),u.jsx(Sl,{scopeType:"global",scopeId:"",period:"hour",limit:24,metric:"tokens",title:"Tokens"})]})]}),i&&u.jsxs("div",{className:"execution-stats-section",children:[u.jsx("h3",{children:"Execution Statistics"}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Ze,{title:"Total Executions",value:r.total_executions,color:"purple"}),u.jsx(Ze,{title:"Success Rate",value:`${l}%`,color:"green"}),u.jsx(Ze,{title:"Total Duration",value:dv(r.total_duration_ms)}),u.jsx(Ze,{title:"Total Cost",value:fv(r.total_cost),color:"orange"})]}),u.jsxs("div",{className:"stats-row token-stats",children:[u.jsx(Ze,{title:"Input Tokens",value:Ii(r.total_input_tokens),small:!0}),u.jsx(Ze,{title:"Output Tokens",value:Ii(r.total_output_tokens),small:!0}),u.jsx(Ze,{title:"Cache Read",value:Ii(r.total_cache_read_tokens),small:!0}),u.jsx(Ze,{title:"Cache Created",value:Ii(r.total_cache_create_tokens),small:!0}),u.jsx(Ze,{title:"Files Created",value:r.total_files_created,small:!0})]})]}),u.jsxs("div",{className:"agents-section",children:[u.jsx("h3",{children:"Agents"}),u.jsxs("div",{className:"agent-cards-grid",children:[t.map(o=>u.jsx(pv,{agent:o,onClick:()=>n(o.id)},o.id)),t.length===0&&u.jsx("div",{className:"no-agents",children:"No agents found. Start an agent to see it here."})]})]})]})},mv=({items:e})=>u.jsx("nav",{className:"breadcrumb",children:e.map((t,n)=>u.jsxs(Xt.Fragment,{children:[n>0&&u.jsx("span",{className:"breadcrumb-separator",children:"/"}),t.onClick?u.jsx("button",{className:"breadcrumb-link",onClick:t.onClick,children:t.label}):u.jsx("span",{className:"breadcrumb-current",children:t.label})]},n))}),At={plus:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),u.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"}),u.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),u.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),u.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),u.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),u.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),u.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("polyline",{points:"3 6 5 6 21 6"}),u.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:u.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},gv=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=F.useState(!1),[c,d]=F.useState(""),[f,g]=F.useState(null),[p,k]=F.useState(""),[w,I]=F.useState(null),h=()=>{c.trim()&&(r(c.trim()),d(""),s(!1))},v=j=>{j.key==="Enter"&&!j.shiftKey?(j.preventDefault(),h()):j.key==="Escape"&&(s(!1),d(""))},y=(j,T)=>{T.stopPropagation(),g(j.id),k(j.title)},b=j=>{var T;p.trim()&&p.trim()!==((T=e.find(U=>U.id===j))==null?void 0:T.title)&&l(j,p.trim()),g(null),k("")},_=()=>{g(null),k("")},S=(j,T)=>{j.key==="Enter"?(j.preventDefault(),b(T)):j.key==="Escape"&&_()},E=(j,T)=>{T.stopPropagation(),I(j)},L=(j,T)=>{T.stopPropagation(),i(j),I(null)},D=j=>{j.stopPropagation(),I(null)},P=j=>{const T=new Date(j),Q=new Date().getTime()-T.getTime(),H=Math.floor(Q/6e4),q=Math.floor(Q/36e5),ie=Math.floor(Q/864e5);return H<1?"now":H<60?`${H}m`:q<24?`${q}h`:ie<7?`${ie}d`:T.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return u.jsxs("div",{className:"thread-list",children:[u.jsxs("div",{className:"list-header",children:[u.jsx("h2",{children:"Conversations"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:At.plus})]}),a&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:c,onChange:j=>d(j.target.value),onKeyDown:v,placeholder:"Conversation title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:h,children:"Create"})]})]}),u.jsx("div",{className:"thread-items",children:e.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:At.hash}),u.jsx("p",{children:"No conversations yet"}),u.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(j=>{const T=o.get(j.id)||0,U=j.id===t,Q=f===j.id,H=w===j.id;return u.jsxs("div",{className:`thread-item ${U?"selected":""} ${T>0?"has-unread":""}`,onClick:()=>!Q&&n(j.id),children:[u.jsx("div",{className:`status-dot ${j.status}`}),u.jsxs("div",{className:"thread-content",children:[u.jsx("div",{className:"thread-title-row",children:Q?u.jsxs("div",{className:"edit-title-form",onClick:q=>q.stopPropagation(),children:[u.jsx("input",{type:"text",value:p,onChange:q=>k(q.target.value),onKeyDown:q=>S(q,j.id),autoFocus:!0}),u.jsx("button",{className:"edit-action save",onClick:()=>b(j.id),title:"Save",children:At.check}),u.jsx("button",{className:"edit-action cancel",onClick:_,title:"Cancel",children:At.x})]}):u.jsxs(u.Fragment,{children:[u.jsx("span",{className:"thread-title",children:j.title}),u.jsx("span",{className:"thread-time",children:P(j.updated_at)})]})}),u.jsxs("div",{className:"thread-meta",children:[j.target_agent&&u.jsxs("span",{className:"thread-agent",title:`Target: ${j.target_agent}`,children:[At.bot,j.target_agent]}),u.jsxs("span",{className:"thread-seq",children:["#",j.last_seq]})]})]}),!Q&&!H&&u.jsxs("div",{className:"thread-actions",children:[u.jsx("button",{className:"action-btn edit",onClick:q=>y(j,q),title:"Rename",children:At.edit}),u.jsx("button",{className:"action-btn delete",onClick:q=>E(j.id,q),title:"Delete",children:At.trash})]}),H&&u.jsxs("div",{className:"delete-confirm",onClick:q=>q.stopPropagation(),children:[u.jsx("span",{className:"confirm-text",children:"Delete?"}),u.jsx("button",{className:"confirm-btn yes",onClick:q=>L(j.id,q),title:"Confirm delete",children:At.check}),u.jsx("button",{className:"confirm-btn no",onClick:D,title:"Cancel",children:At.x})]}),T>0&&!H&&u.jsx("span",{className:"unread-badge",children:T})]},j.id)})}),u.jsx("style",{children:`
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
      `})]})};function vv(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const yv=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,xv=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,kv={};function ac(e,t){return(kv.jsx?xv:yv).test(e)}const wv=/[ \t\n\f\r]/g;function Sv(e){return typeof e=="object"?e.type==="text"?sc(e.value):!1:sc(e)}function sc(e){return e.replace(wv,"")===""}class hi{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}hi.prototype.normal={};hi.prototype.property={};hi.prototype.space=void 0;function pp(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new hi(n,r,t)}function ja(e){return e.toLowerCase()}class Ge{constructor(t,n){this.attribute=n,this.property=t}}Ge.prototype.attribute="";Ge.prototype.booleanish=!1;Ge.prototype.boolean=!1;Ge.prototype.commaOrSpaceSeparated=!1;Ge.prototype.commaSeparated=!1;Ge.prototype.defined=!1;Ge.prototype.mustUseProperty=!1;Ge.prototype.number=!1;Ge.prototype.overloadedBoolean=!1;Ge.prototype.property="";Ge.prototype.spaceSeparated=!1;Ge.prototype.space=void 0;let bv=0;const K=An(),xe=An(),Ea=An(),R=An(),oe=An(),rr=An(),et=An();function An(){return 2**++bv}const _a=Object.freeze(Object.defineProperty({__proto__:null,boolean:K,booleanish:xe,commaOrSpaceSeparated:et,commaSeparated:rr,number:R,overloadedBoolean:Ea,spaceSeparated:oe},Symbol.toStringTag,{value:"Module"})),xo=Object.keys(_a);class Ls extends Ge{constructor(t,n,r,i){let l=-1;if(super(t,n),uc(this,"space",i),typeof r=="number")for(;++l<xo.length;){const o=xo[l];uc(this,xo[l],(r&_a[o])===_a[o])}}}Ls.prototype.defined=!0;function uc(e,t,n){n&&(e[t]=n)}function mr(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new Ls(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[ja(r)]=r,n[ja(l.attribute)]=r}return new hi(t,n,e.space)}const hp=mr({properties:{ariaActiveDescendant:null,ariaAtomic:xe,ariaAutoComplete:null,ariaBusy:xe,ariaChecked:xe,ariaColCount:R,ariaColIndex:R,ariaColSpan:R,ariaControls:oe,ariaCurrent:null,ariaDescribedBy:oe,ariaDetails:null,ariaDisabled:xe,ariaDropEffect:oe,ariaErrorMessage:null,ariaExpanded:xe,ariaFlowTo:oe,ariaGrabbed:xe,ariaHasPopup:null,ariaHidden:xe,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:oe,ariaLevel:R,ariaLive:null,ariaModal:xe,ariaMultiLine:xe,ariaMultiSelectable:xe,ariaOrientation:null,ariaOwns:oe,ariaPlaceholder:null,ariaPosInSet:R,ariaPressed:xe,ariaReadOnly:xe,ariaRelevant:null,ariaRequired:xe,ariaRoleDescription:oe,ariaRowCount:R,ariaRowIndex:R,ariaRowSpan:R,ariaSelected:xe,ariaSetSize:R,ariaSort:null,ariaValueMax:R,ariaValueMin:R,ariaValueNow:R,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function mp(e,t){return t in e?e[t]:t}function gp(e,t){return mp(e,t.toLowerCase())}const Cv=mr({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:rr,acceptCharset:oe,accessKey:oe,action:null,allow:null,allowFullScreen:K,allowPaymentRequest:K,allowUserMedia:K,alt:null,as:null,async:K,autoCapitalize:null,autoComplete:oe,autoFocus:K,autoPlay:K,blocking:oe,capture:null,charSet:null,checked:K,cite:null,className:oe,cols:R,colSpan:null,content:null,contentEditable:xe,controls:K,controlsList:oe,coords:R|rr,crossOrigin:null,data:null,dateTime:null,decoding:null,default:K,defer:K,dir:null,dirName:null,disabled:K,download:Ea,draggable:xe,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:K,formTarget:null,headers:oe,height:R,hidden:Ea,high:R,href:null,hrefLang:null,htmlFor:oe,httpEquiv:oe,id:null,imageSizes:null,imageSrcSet:null,inert:K,inputMode:null,integrity:null,is:null,isMap:K,itemId:null,itemProp:oe,itemRef:oe,itemScope:K,itemType:oe,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:K,low:R,manifest:null,max:null,maxLength:R,media:null,method:null,min:null,minLength:R,multiple:K,muted:K,name:null,nonce:null,noModule:K,noValidate:K,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:K,optimum:R,pattern:null,ping:oe,placeholder:null,playsInline:K,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:K,referrerPolicy:null,rel:oe,required:K,reversed:K,rows:R,rowSpan:R,sandbox:oe,scope:null,scoped:K,seamless:K,selected:K,shadowRootClonable:K,shadowRootDelegatesFocus:K,shadowRootMode:null,shape:null,size:R,sizes:null,slot:null,span:R,spellCheck:xe,src:null,srcDoc:null,srcLang:null,srcSet:null,start:R,step:null,style:null,tabIndex:R,target:null,title:null,translate:null,type:null,typeMustMatch:K,useMap:null,value:xe,width:R,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:oe,axis:null,background:null,bgColor:null,border:R,borderColor:null,bottomMargin:R,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:K,declare:K,event:null,face:null,frame:null,frameBorder:null,hSpace:R,leftMargin:R,link:null,longDesc:null,lowSrc:null,marginHeight:R,marginWidth:R,noResize:K,noHref:K,noShade:K,noWrap:K,object:null,profile:null,prompt:null,rev:null,rightMargin:R,rules:null,scheme:null,scrolling:xe,standby:null,summary:null,text:null,topMargin:R,valueType:null,version:null,vAlign:null,vLink:null,vSpace:R,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:K,disableRemotePlayback:K,prefix:null,property:null,results:R,security:null,unselectable:null},space:"html",transform:gp}),jv=mr({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:et,accentHeight:R,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:R,amplitude:R,arabicForm:null,ascent:R,attributeName:null,attributeType:null,azimuth:R,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:R,by:null,calcMode:null,capHeight:R,className:oe,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:R,diffuseConstant:R,direction:null,display:null,dur:null,divisor:R,dominantBaseline:null,download:K,dx:null,dy:null,edgeMode:null,editable:null,elevation:R,enableBackground:null,end:null,event:null,exponent:R,externalResourcesRequired:null,fill:null,fillOpacity:R,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:rr,g2:rr,glyphName:rr,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:R,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:R,horizOriginX:R,horizOriginY:R,id:null,ideographic:R,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:R,k:R,k1:R,k2:R,k3:R,k4:R,kernelMatrix:et,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:R,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:R,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:R,overlineThickness:R,paintOrder:null,panose1:null,path:null,pathLength:R,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:oe,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:R,pointsAtY:R,pointsAtZ:R,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:et,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:et,rev:et,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:et,requiredFeatures:et,requiredFonts:et,requiredFormats:et,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:R,specularExponent:R,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:R,strikethroughThickness:R,string:null,stroke:null,strokeDashArray:et,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:R,strokeOpacity:R,strokeWidth:null,style:null,surfaceScale:R,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:et,tabIndex:R,tableValues:null,target:null,targetX:R,targetY:R,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:et,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:R,underlineThickness:R,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:R,values:null,vAlphabetic:R,vMathematical:R,vectorEffect:null,vHanging:R,vIdeographic:R,version:null,vertAdvY:R,vertOriginX:R,vertOriginY:R,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:R,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:mp}),vp=mr({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),yp=mr({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:gp}),xp=mr({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),Ev={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},_v=/[A-Z]/g,cc=/-[a-z]/g,Nv=/^data[-\w.:]+$/i;function Tv(e,t){const n=ja(t);let r=t,i=Ge;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&Nv.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(cc,Lv);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!cc.test(l)){let o=l.replace(_v,zv);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=Ls}return new i(r,t)}function zv(e){return"-"+e.toLowerCase()}function Lv(e){return e.charAt(1).toUpperCase()}const Pv=pp([hp,Cv,vp,yp,xp],"html"),Ps=pp([hp,jv,vp,yp,xp],"svg");function Iv(e){return e.join(" ").trim()}var Is={},dc=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Av=/\n/g,Mv=/^\s*/,Dv=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Rv=/^:\s*/,Fv=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Ov=/^[;\s]*/,Bv=/^\s+|\s+$/g,$v=`
`,fc="/",pc="*",wn="",Uv="comment",Hv="declaration";function Vv(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(k){var w=k.match(Av);w&&(n+=w.length);var I=k.lastIndexOf($v);r=~I?k.length-I:r+k.length}function l(){var k={line:n,column:r};return function(w){return w.position=new o(k),c(),w}}function o(k){this.start=k,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(k){var w=new Error(t.source+":"+n+":"+r+": "+k);if(w.reason=k,w.filename=t.source,w.line=n,w.column=r,w.source=e,!t.silent)throw w}function s(k){var w=k.exec(e);if(w){var I=w[0];return i(I),e=e.slice(I.length),w}}function c(){s(Mv)}function d(k){var w;for(k=k||[];w=f();)w!==!1&&k.push(w);return k}function f(){var k=l();if(!(fc!=e.charAt(0)||pc!=e.charAt(1))){for(var w=2;wn!=e.charAt(w)&&(pc!=e.charAt(w)||fc!=e.charAt(w+1));)++w;if(w+=2,wn===e.charAt(w-1))return a("End of comment missing");var I=e.slice(2,w-2);return r+=2,i(I),e=e.slice(w),r+=2,k({type:Uv,comment:I})}}function g(){var k=l(),w=s(Dv);if(w){if(f(),!s(Rv))return a("property missing ':'");var I=s(Fv),h=k({type:Hv,property:hc(w[0].replace(dc,wn)),value:I?hc(I[0].replace(dc,wn)):wn});return s(Ov),h}}function p(){var k=[];d(k);for(var w;w=g();)w!==!1&&(k.push(w),d(k));return k}return c(),p()}function hc(e){return e?e.replace(Bv,wn):wn}var Wv=Vv,Qv=Gi&&Gi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(Is,"__esModule",{value:!0});Is.default=Kv;const qv=Qv(Wv);function Kv(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,qv.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var $l={};Object.defineProperty($l,"__esModule",{value:!0});$l.camelCase=void 0;var Yv=/^--[a-zA-Z0-9_-]+$/,Xv=/-([a-z])/g,Gv=/^[^-]+$/,Jv=/^-(webkit|moz|ms|o|khtml)-/,Zv=/^-(ms)-/,ey=function(e){return!e||Gv.test(e)||Yv.test(e)},ty=function(e,t){return t.toUpperCase()},mc=function(e,t){return"".concat(t,"-")},ny=function(e,t){return t===void 0&&(t={}),ey(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(Zv,mc):e=e.replace(Jv,mc),e.replace(Xv,ty))};$l.camelCase=ny;var ry=Gi&&Gi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},iy=ry(Is),ly=$l;function Na(e,t){var n={};return!e||typeof e!="string"||(0,iy.default)(e,function(r,i){r&&i&&(n[(0,ly.camelCase)(r,t)]=i)}),n}Na.default=Na;var oy=Na;const ay=Da(oy),kp=wp("end"),As=wp("start");function wp(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function sy(e){const t=As(e),n=kp(e);if(t&&n)return{start:t,end:n}}function Ur(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?gc(e.position):"start"in e||"end"in e?gc(e):"line"in e||"column"in e?Ta(e):""}function Ta(e){return vc(e&&e.line)+":"+vc(e&&e.column)}function gc(e){return Ta(e&&e.start)+"-"+Ta(e&&e.end)}function vc(e){return e&&typeof e=="number"?e:1}class De extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Ur(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}De.prototype.file="";De.prototype.name="";De.prototype.reason="";De.prototype.message="";De.prototype.stack="";De.prototype.column=void 0;De.prototype.line=void 0;De.prototype.ancestors=void 0;De.prototype.cause=void 0;De.prototype.fatal=void 0;De.prototype.place=void 0;De.prototype.ruleId=void 0;De.prototype.source=void 0;const Ms={}.hasOwnProperty,uy=new Map,cy=/[A-Z]/g,dy=new Set(["table","tbody","thead","tfoot","tr"]),fy=new Set(["td","th"]),Sp="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function py(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=wy(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=ky(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?Ps:Pv,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=bp(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function bp(e,t,n){if(t.type==="element")return hy(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return my(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return vy(e,t,n);if(t.type==="mdxjsEsm")return gy(e,t);if(t.type==="root")return yy(e,t,n);if(t.type==="text")return xy(e,t)}function hy(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=Ps,e.schema=i),e.ancestors.push(t);const l=jp(e,t.tagName,!1),o=Sy(e,t);let a=Rs(e,t);return dy.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!Sv(s):!0})),Cp(e,o,l,t),Ds(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function my(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}si(e,t.position)}function gy(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);si(e,t.position)}function vy(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=Ps,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:jp(e,t.name,!0),o=by(e,t),a=Rs(e,t);return Cp(e,o,l,t),Ds(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function yy(e,t,n){const r={};return Ds(r,Rs(e,t)),e.create(t,e.Fragment,r,n)}function xy(e,t){return t.value}function Cp(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Ds(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function ky(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function wy(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=As(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function Sy(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Ms.call(t.properties,i)){const l=Cy(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&fy.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function by(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else si(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else si(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Rs(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:uy;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const c=i.get(s)||0;o=s+"-"+c,i.set(s,c+1)}}const a=bp(e,l,o);a!==void 0&&n.push(a)}return n}function Cy(e,t,n){const r=Tv(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?vv(n):Iv(n)),r.property==="style"){let i=typeof n=="object"?n:jy(e,String(n));return e.stylePropertyNameCase==="css"&&(i=Ey(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?Ev[r.property]||r.property:r.attribute,n]}}function jy(e,t){try{return ay(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new De("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=Sp+"#cannot-parse-style-attribute",i}}function jp(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=ac(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=ac(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Ms.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);si(e)}function si(e,t){const n=new De("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=Sp+"#cannot-handle-mdx-estrees-without-createevaluater",n}function Ey(e){const t={};let n;for(n in e)Ms.call(e,n)&&(t[_y(n)]=e[n]);return t}function _y(e){let t=e.replace(cy,Ny);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function Ny(e){return"-"+e.toLowerCase()}const ko={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},Ty={};function zy(e,t){const n=Ty,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return Ep(e,r,i)}function Ep(e,t,n){if(Ly(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return yc(e.children,t,n)}return Array.isArray(e)?yc(e,t,n):""}function yc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=Ep(e[i],t,n);return r.join("")}function Ly(e){return!!(e&&typeof e=="object")}const xc=document.createElement("i");function Fs(e){const t="&"+e+";";xc.innerHTML=t;const n=xc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function Lt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function ct(e,t){return e.length>0?(Lt(e,e.length,0,t),e):t}const kc={}.hasOwnProperty;function Py(e){const t={};let n=-1;for(;++n<e.length;)Iy(t,e[n]);return t}function Iy(e,t){let n;for(n in t){const i=(kc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){kc.call(i,o)||(i[o]=[]);const a=l[o];Ay(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Ay(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);Lt(e,0,0,r)}function _p(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function ir(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const Nt=vn(/[A-Za-z]/),rt=vn(/[\dA-Za-z]/),My=vn(/[#-'*+\--9=?A-Z^-~]/);function za(e){return e!==null&&(e<32||e===127)}const La=vn(/\d/),Dy=vn(/[\dA-Fa-f]/),Ry=vn(/[!-/:-@[-`{-~]/);function V(e){return e!==null&&e<-2}function Xe(e){return e!==null&&(e<0||e===32)}function ne(e){return e===-2||e===-1||e===32}const Fy=vn(new RegExp("\\p{P}|\\p{S}","u")),Oy=vn(/\s/);function vn(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function gr(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&rt(e.charCodeAt(n+1))&&rt(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function se(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return ne(s)?(e.enter(n),a(s)):t(s)}function a(s){return ne(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const By={tokenize:$y};function $y(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),se(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return V(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const Uy={tokenize:Hy},wc={tokenize:Vy};function Hy(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const b=n[r];return t.containerState=b[1],e.attempt(b[0].continuation,s,c)(y)}return c(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&v();const b=t.events.length;let _=b,S;for(;_--;)if(t.events[_][0]==="exit"&&t.events[_][1].type==="chunkFlow"){S=t.events[_][1].end;break}h(r);let E=b;for(;E<t.events.length;)t.events[E][1].end={...S},E++;return Lt(t.events,_+1,0,t.events.slice(b)),t.events.length=E,c(y)}return a(y)}function c(y){if(r===n.length){if(!i)return g(y);if(i.currentConstruct&&i.currentConstruct.concrete)return k(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(wc,d,f)(y)}function d(y){return i&&v(),h(r),g(y)}function f(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,k(y)}function g(y){return t.containerState={},e.attempt(wc,p,k)(y)}function p(y){return r++,n.push([t.currentConstruct,t.containerState]),g(y)}function k(y){if(y===null){i&&v(),h(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),w(y)}function w(y){if(y===null){I(e.exit("chunkFlow"),!0),h(0),e.consume(y);return}return V(y)?(e.consume(y),I(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),w)}function I(y,b){const _=t.sliceStream(y);if(b&&_.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(_),t.parser.lazy[y.start.line]){let S=i.events.length;for(;S--;)if(i.events[S][1].start.offset<o&&(!i.events[S][1].end||i.events[S][1].end.offset>o))return;const E=t.events.length;let L=E,D,P;for(;L--;)if(t.events[L][0]==="exit"&&t.events[L][1].type==="chunkFlow"){if(D){P=t.events[L][1].end;break}D=!0}for(h(r),S=E;S<t.events.length;)t.events[S][1].end={...P},S++;Lt(t.events,L+1,0,t.events.slice(E)),t.events.length=S}}function h(y){let b=n.length;for(;b-- >y;){const _=n[b];t.containerState=_[1],_[0].exit.call(t,e)}n.length=y}function v(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Vy(e,t,n){return se(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function Sc(e){if(e===null||Xe(e)||Oy(e))return 1;if(Fy(e))return 2}function Os(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const Pa={name:"attention",resolveAll:Wy,tokenize:Qy};function Wy(e,t){let n=-1,r,i,l,o,a,s,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const f={...e[r][1].end},g={...e[n][1].start};bc(f,-s),bc(g,s),o={type:s>1?"strongSequence":"emphasisSequence",start:f,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:g},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=ct(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=ct(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=ct(c,Os(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=ct(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=ct(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,Lt(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Qy(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=Sc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const c=e.exit("attentionSequence"),d=Sc(s),f=!d||d===2&&i||n.includes(s),g=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?f:f&&(i||!g)),c._close=!!(l===42?g:g&&(d||!f)),t(s)}}function bc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const qy={name:"autolink",tokenize:Ky};function Ky(e,t,n){let r=0;return i;function i(p){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(p){return Nt(p)?(e.consume(p),o):p===64?n(p):c(p)}function o(p){return p===43||p===45||p===46||rt(p)?(r=1,a(p)):c(p)}function a(p){return p===58?(e.consume(p),r=0,s):(p===43||p===45||p===46||rt(p))&&r++<32?(e.consume(p),a):(r=0,c(p))}function s(p){return p===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):p===null||p===32||p===60||za(p)?n(p):(e.consume(p),s)}function c(p){return p===64?(e.consume(p),d):My(p)?(e.consume(p),c):n(p)}function d(p){return rt(p)?f(p):n(p)}function f(p){return p===46?(e.consume(p),r=0,d):p===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):g(p)}function g(p){if((p===45||rt(p))&&r++<63){const k=p===45?g:f;return e.consume(p),k}return n(p)}}const Ul={partial:!0,tokenize:Yy};function Yy(e,t,n){return r;function r(l){return ne(l)?se(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||V(l)?t(l):n(l)}}const Np={continuation:{tokenize:Gy},exit:Jy,name:"blockQuote",tokenize:Xy};function Xy(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return ne(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function Gy(e,t,n){const r=this;return i;function i(o){return ne(o)?se(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(Np,t,n)(o)}}function Jy(e){e.exit("blockQuote")}const Tp={name:"characterEscape",tokenize:Zy};function Zy(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return Ry(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const zp={name:"characterReference",tokenize:ex};function ex(e,t,n){const r=this;let i=0,l,o;return a;function a(f){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),s}function s(f){return f===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(f),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=rt,d(f))}function c(f){return f===88||f===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(f),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=Dy,d):(e.enter("characterReferenceValue"),l=7,o=La,d(f))}function d(f){if(f===59&&i){const g=e.exit("characterReferenceValue");return o===rt&&!Fs(r.sliceSerialize(g))?n(f):(e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(f)&&i++<l?(e.consume(f),d):n(f)}}const Cc={partial:!0,tokenize:nx},jc={concrete:!0,name:"codeFenced",tokenize:tx};function tx(e,t,n){const r=this,i={partial:!0,tokenize:_};let l=0,o=0,a;return s;function s(S){return c(S)}function c(S){const E=r.events[r.events.length-1];return l=E&&E[1].type==="linePrefix"?E[2].sliceSerialize(E[1],!0).length:0,a=S,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(S)}function d(S){return S===a?(o++,e.consume(S),d):o<3?n(S):(e.exit("codeFencedFenceSequence"),ne(S)?se(e,f,"whitespace")(S):f(S))}function f(S){return S===null||V(S)?(e.exit("codeFencedFence"),r.interrupt?t(S):e.check(Cc,w,b)(S)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),g(S))}function g(S){return S===null||V(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),f(S)):ne(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),se(e,p,"whitespace")(S)):S===96&&S===a?n(S):(e.consume(S),g)}function p(S){return S===null||V(S)?f(S):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),k(S))}function k(S){return S===null||V(S)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),f(S)):S===96&&S===a?n(S):(e.consume(S),k)}function w(S){return e.attempt(i,b,I)(S)}function I(S){return e.enter("lineEnding"),e.consume(S),e.exit("lineEnding"),h}function h(S){return l>0&&ne(S)?se(e,v,"linePrefix",l+1)(S):v(S)}function v(S){return S===null||V(S)?e.check(Cc,w,b)(S):(e.enter("codeFlowValue"),y(S))}function y(S){return S===null||V(S)?(e.exit("codeFlowValue"),v(S)):(e.consume(S),y)}function b(S){return e.exit("codeFenced"),t(S)}function _(S,E,L){let D=0;return P;function P(H){return S.enter("lineEnding"),S.consume(H),S.exit("lineEnding"),j}function j(H){return S.enter("codeFencedFence"),ne(H)?se(S,T,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(H):T(H)}function T(H){return H===a?(S.enter("codeFencedFenceSequence"),U(H)):L(H)}function U(H){return H===a?(D++,S.consume(H),U):D>=o?(S.exit("codeFencedFenceSequence"),ne(H)?se(S,Q,"whitespace")(H):Q(H)):L(H)}function Q(H){return H===null||V(H)?(S.exit("codeFencedFence"),E(H)):L(H)}}}function nx(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const wo={name:"codeIndented",tokenize:ix},rx={partial:!0,tokenize:lx};function ix(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),se(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?s(c):V(c)?e.attempt(rx,o,s)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||V(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function s(c){return e.exit("codeIndented"),t(c)}}function lx(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):V(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):se(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):V(o)?i(o):n(o)}}const ox={name:"codeText",previous:sx,resolve:ax,tokenize:ux};function ax(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function sx(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function ux(e,t,n){let r=0,i,l;return o;function o(f){return e.enter("codeText"),e.enter("codeTextSequence"),a(f)}function a(f){return f===96?(e.consume(f),r++,a):(e.exit("codeTextSequence"),s(f))}function s(f){return f===null?n(f):f===32?(e.enter("space"),e.consume(f),e.exit("space"),s):f===96?(l=e.enter("codeTextSequence"),i=0,d(f)):V(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),s):(e.enter("codeTextData"),c(f))}function c(f){return f===null||f===32||f===96||V(f)?(e.exit("codeTextData"),s(f)):(e.consume(f),c)}function d(f){return f===96?(e.consume(f),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(f)):(l.type="codeTextData",c(f))}}class cx{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&_r(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),_r(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),_r(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);_r(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);_r(this.left,n.reverse())}}}function _r(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function Lp(e){const t={};let n=-1,r,i,l,o,a,s,c;const d=new cx(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,dx(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return Lt(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function dx(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],c={};let d,f,g=-1,p=n,k=0,w=0;const I=[w];for(;p;){for(;e.get(++i)[1]!==p;);l.push(i),p._tokenizer||(d=r.sliceStream(p),p.next||d.push(null),f&&o.defineSkip(p.start),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),f=p,p=p.next}for(p=n;++g<a.length;)a[g][0]==="exit"&&a[g-1][0]==="enter"&&a[g][1].type===a[g-1][1].type&&a[g][1].start.line!==a[g][1].end.line&&(w=g+1,I.push(w),p._tokenizer=void 0,p.previous=void 0,p=p.next);for(o.events=[],p?(p._tokenizer=void 0,p.previous=void 0):I.pop(),g=I.length;g--;){const h=a.slice(I[g],I[g+1]),v=l.pop();s.push([v,v+h.length-1]),e.splice(v,2,h)}for(s.reverse(),g=-1;++g<s.length;)c[k+s[g][0]]=k+s[g][1],k+=s[g][1]-s[g][0]-1;return c}const fx={resolve:hx,tokenize:mx},px={partial:!0,tokenize:gx};function hx(e){return Lp(e),e}function mx(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):V(a)?e.check(px,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function gx(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),se(e,l,"linePrefix")}function l(o){if(o===null||V(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function Pp(e,t,n,r,i,l,o,a,s){const c=s||Number.POSITIVE_INFINITY;let d=0;return f;function f(h){return h===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(h),e.exit(l),g):h===null||h===32||h===41||za(h)?n(h):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),w(h))}function g(h){return h===62?(e.enter(l),e.consume(h),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),p(h))}function p(h){return h===62?(e.exit("chunkString"),e.exit(a),g(h)):h===null||h===60||V(h)?n(h):(e.consume(h),h===92?k:p)}function k(h){return h===60||h===62||h===92?(e.consume(h),p):p(h)}function w(h){return!d&&(h===null||h===41||Xe(h))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(h)):d<c&&h===40?(e.consume(h),d++,w):h===41?(e.consume(h),d--,w):h===null||h===32||h===40||za(h)?n(h):(e.consume(h),h===92?I:w)}function I(h){return h===40||h===41||h===92?(e.consume(h),w):w(h)}}function Ip(e,t,n,r,i,l){const o=this;let a=0,s;return c;function c(p){return e.enter(r),e.enter(i),e.consume(p),e.exit(i),e.enter(l),d}function d(p){return a>999||p===null||p===91||p===93&&!s||p===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(p):p===93?(e.exit(l),e.enter(i),e.consume(p),e.exit(i),e.exit(r),t):V(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),f(p))}function f(p){return p===null||p===91||p===93||V(p)||a++>999?(e.exit("chunkString"),d(p)):(e.consume(p),s||(s=!ne(p)),p===92?g:f)}function g(p){return p===91||p===92||p===93?(e.consume(p),a++,f):f(p)}}function Ap(e,t,n,r,i,l){let o;return a;function a(g){return g===34||g===39||g===40?(e.enter(r),e.enter(i),e.consume(g),e.exit(i),o=g===40?41:g,s):n(g)}function s(g){return g===o?(e.enter(i),e.consume(g),e.exit(i),e.exit(r),t):(e.enter(l),c(g))}function c(g){return g===o?(e.exit(l),s(o)):g===null?n(g):V(g)?(e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),se(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(g))}function d(g){return g===o||g===null||V(g)?(e.exit("chunkString"),c(g)):(e.consume(g),g===92?f:d)}function f(g){return g===o||g===92?(e.consume(g),d):d(g)}}function Hr(e,t){let n;return r;function r(i){return V(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):ne(i)?se(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const vx={name:"definition",tokenize:xx},yx={partial:!0,tokenize:kx};function xx(e,t,n){const r=this;let i;return l;function l(p){return e.enter("definition"),o(p)}function o(p){return Ip.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(p)}function a(p){return i=ir(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),p===58?(e.enter("definitionMarker"),e.consume(p),e.exit("definitionMarker"),s):n(p)}function s(p){return Xe(p)?Hr(e,c)(p):c(p)}function c(p){return Pp(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(p)}function d(p){return e.attempt(yx,f,f)(p)}function f(p){return ne(p)?se(e,g,"whitespace")(p):g(p)}function g(p){return p===null||V(p)?(e.exit("definition"),r.parser.defined.push(i),t(p)):n(p)}}function kx(e,t,n){return r;function r(a){return Xe(a)?Hr(e,i)(a):n(a)}function i(a){return Ap(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return ne(a)?se(e,o,"whitespace")(a):o(a)}function o(a){return a===null||V(a)?t(a):n(a)}}const wx={name:"hardBreakEscape",tokenize:Sx};function Sx(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return V(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const bx={name:"headingAtx",resolve:Cx,tokenize:jx};function Cx(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},Lt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function jx(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||Xe(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||V(d)?(e.exit("atxHeading"),t(d)):ne(d)?se(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||Xe(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const Ex=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],Ec=["pre","script","style","textarea"],_x={concrete:!0,name:"htmlFlow",resolveTo:zx,tokenize:Lx},Nx={partial:!0,tokenize:Ix},Tx={partial:!0,tokenize:Px};function zx(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function Lx(e,t,n){const r=this;let i,l,o,a,s;return c;function c(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),f}function f(x){return x===33?(e.consume(x),g):x===47?(e.consume(x),l=!0,w):x===63?(e.consume(x),i=3,r.interrupt?t:m):Nt(x)?(e.consume(x),o=String.fromCharCode(x),I):n(x)}function g(x){return x===45?(e.consume(x),i=2,p):x===91?(e.consume(x),i=5,a=0,k):Nt(x)?(e.consume(x),i=4,r.interrupt?t:m):n(x)}function p(x){return x===45?(e.consume(x),r.interrupt?t:m):n(x)}function k(x){const X="CDATA[";return x===X.charCodeAt(a++)?(e.consume(x),a===X.length?r.interrupt?t:T:k):n(x)}function w(x){return Nt(x)?(e.consume(x),o=String.fromCharCode(x),I):n(x)}function I(x){if(x===null||x===47||x===62||Xe(x)){const X=x===47,fe=o.toLowerCase();return!X&&!l&&Ec.includes(fe)?(i=1,r.interrupt?t(x):T(x)):Ex.includes(o.toLowerCase())?(i=6,X?(e.consume(x),h):r.interrupt?t(x):T(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?v(x):y(x))}return x===45||rt(x)?(e.consume(x),o+=String.fromCharCode(x),I):n(x)}function h(x){return x===62?(e.consume(x),r.interrupt?t:T):n(x)}function v(x){return ne(x)?(e.consume(x),v):P(x)}function y(x){return x===47?(e.consume(x),P):x===58||x===95||Nt(x)?(e.consume(x),b):ne(x)?(e.consume(x),y):P(x)}function b(x){return x===45||x===46||x===58||x===95||rt(x)?(e.consume(x),b):_(x)}function _(x){return x===61?(e.consume(x),S):ne(x)?(e.consume(x),_):y(x)}function S(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,E):ne(x)?(e.consume(x),S):L(x)}function E(x){return x===s?(e.consume(x),s=null,D):x===null||V(x)?n(x):(e.consume(x),E)}function L(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||Xe(x)?_(x):(e.consume(x),L)}function D(x){return x===47||x===62||ne(x)?y(x):n(x)}function P(x){return x===62?(e.consume(x),j):n(x)}function j(x){return x===null||V(x)?T(x):ne(x)?(e.consume(x),j):n(x)}function T(x){return x===45&&i===2?(e.consume(x),q):x===60&&i===1?(e.consume(x),ie):x===62&&i===4?(e.consume(x),z):x===63&&i===3?(e.consume(x),m):x===93&&i===5?(e.consume(x),B):V(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(Nx,A,U)(x)):x===null||V(x)?(e.exit("htmlFlowData"),U(x)):(e.consume(x),T)}function U(x){return e.check(Tx,Q,A)(x)}function Q(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),H}function H(x){return x===null||V(x)?U(x):(e.enter("htmlFlowData"),T(x))}function q(x){return x===45?(e.consume(x),m):T(x)}function ie(x){return x===47?(e.consume(x),o="",C):T(x)}function C(x){if(x===62){const X=o.toLowerCase();return Ec.includes(X)?(e.consume(x),z):T(x)}return Nt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),C):T(x)}function B(x){return x===93?(e.consume(x),m):T(x)}function m(x){return x===62?(e.consume(x),z):x===45&&i===2?(e.consume(x),m):T(x)}function z(x){return x===null||V(x)?(e.exit("htmlFlowData"),A(x)):(e.consume(x),z)}function A(x){return e.exit("htmlFlow"),t(x)}}function Px(e,t,n){const r=this;return i;function i(o){return V(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Ix(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Ul,t,n)}}const Ax={name:"htmlText",tokenize:Mx};function Mx(e,t,n){const r=this;let i,l,o;return a;function a(m){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(m),s}function s(m){return m===33?(e.consume(m),c):m===47?(e.consume(m),_):m===63?(e.consume(m),y):Nt(m)?(e.consume(m),L):n(m)}function c(m){return m===45?(e.consume(m),d):m===91?(e.consume(m),l=0,k):Nt(m)?(e.consume(m),v):n(m)}function d(m){return m===45?(e.consume(m),p):n(m)}function f(m){return m===null?n(m):m===45?(e.consume(m),g):V(m)?(o=f,ie(m)):(e.consume(m),f)}function g(m){return m===45?(e.consume(m),p):f(m)}function p(m){return m===62?q(m):m===45?g(m):f(m)}function k(m){const z="CDATA[";return m===z.charCodeAt(l++)?(e.consume(m),l===z.length?w:k):n(m)}function w(m){return m===null?n(m):m===93?(e.consume(m),I):V(m)?(o=w,ie(m)):(e.consume(m),w)}function I(m){return m===93?(e.consume(m),h):w(m)}function h(m){return m===62?q(m):m===93?(e.consume(m),h):w(m)}function v(m){return m===null||m===62?q(m):V(m)?(o=v,ie(m)):(e.consume(m),v)}function y(m){return m===null?n(m):m===63?(e.consume(m),b):V(m)?(o=y,ie(m)):(e.consume(m),y)}function b(m){return m===62?q(m):y(m)}function _(m){return Nt(m)?(e.consume(m),S):n(m)}function S(m){return m===45||rt(m)?(e.consume(m),S):E(m)}function E(m){return V(m)?(o=E,ie(m)):ne(m)?(e.consume(m),E):q(m)}function L(m){return m===45||rt(m)?(e.consume(m),L):m===47||m===62||Xe(m)?D(m):n(m)}function D(m){return m===47?(e.consume(m),q):m===58||m===95||Nt(m)?(e.consume(m),P):V(m)?(o=D,ie(m)):ne(m)?(e.consume(m),D):q(m)}function P(m){return m===45||m===46||m===58||m===95||rt(m)?(e.consume(m),P):j(m)}function j(m){return m===61?(e.consume(m),T):V(m)?(o=j,ie(m)):ne(m)?(e.consume(m),j):D(m)}function T(m){return m===null||m===60||m===61||m===62||m===96?n(m):m===34||m===39?(e.consume(m),i=m,U):V(m)?(o=T,ie(m)):ne(m)?(e.consume(m),T):(e.consume(m),Q)}function U(m){return m===i?(e.consume(m),i=void 0,H):m===null?n(m):V(m)?(o=U,ie(m)):(e.consume(m),U)}function Q(m){return m===null||m===34||m===39||m===60||m===61||m===96?n(m):m===47||m===62||Xe(m)?D(m):(e.consume(m),Q)}function H(m){return m===47||m===62||Xe(m)?D(m):n(m)}function q(m){return m===62?(e.consume(m),e.exit("htmlTextData"),e.exit("htmlText"),t):n(m)}function ie(m){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(m),e.exit("lineEnding"),C}function C(m){return ne(m)?se(e,B,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(m):B(m)}function B(m){return e.enter("htmlTextData"),o(m)}}const Bs={name:"labelEnd",resolveAll:Ox,resolveTo:Bx,tokenize:$x},Dx={tokenize:Ux},Rx={tokenize:Hx},Fx={tokenize:Vx};function Ox(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&Lt(e,0,e.length,n),e}function Bx(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",c,t]],a=ct(a,e.slice(l+1,l+r+3)),a=ct(a,[["enter",d,t]]),a=ct(a,Os(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=ct(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=ct(a,e.slice(o+1)),a=ct(a,[["exit",s,t]]),Lt(e,l,e.length,a),e}function $x(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(g){return l?l._inactive?f(g):(o=r.parser.defined.includes(ir(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(g),e.exit("labelMarker"),e.exit("labelEnd"),s):n(g)}function s(g){return g===40?e.attempt(Dx,d,o?d:f)(g):g===91?e.attempt(Rx,d,o?c:f)(g):o?d(g):f(g)}function c(g){return e.attempt(Fx,d,f)(g)}function d(g){return t(g)}function f(g){return l._balanced=!0,n(g)}}function Ux(e,t,n){return r;function r(f){return e.enter("resource"),e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),i}function i(f){return Xe(f)?Hr(e,l)(f):l(f)}function l(f){return f===41?d(f):Pp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(f)}function o(f){return Xe(f)?Hr(e,s)(f):d(f)}function a(f){return n(f)}function s(f){return f===34||f===39||f===40?Ap(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(f):d(f)}function c(f){return Xe(f)?Hr(e,d)(f):d(f)}function d(f){return f===41?(e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),e.exit("resource"),t):n(f)}}function Hx(e,t,n){const r=this;return i;function i(a){return Ip.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(ir(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Vx(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Wx={name:"labelStartImage",resolveAll:Bs.resolveAll,tokenize:Qx};function Qx(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const qx={name:"labelStartLink",resolveAll:Bs.resolveAll,tokenize:Kx};function Kx(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const So={name:"lineEnding",tokenize:Yx};function Yx(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),se(e,t,"linePrefix")}}const Yi={name:"thematicBreak",tokenize:Xx};function Xx(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),s(c)):r>=3&&(c===null||V(c))?(e.exit("thematicBreak"),t(c)):n(c)}function s(c){return c===i?(e.consume(c),r++,s):(e.exit("thematicBreakSequence"),ne(c)?se(e,a,"whitespace")(c):a(c))}}const Ve={continuation:{tokenize:e1},exit:n1,name:"list",tokenize:Zx},Gx={partial:!0,tokenize:r1},Jx={partial:!0,tokenize:t1};function Zx(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(p){const k=r.containerState.type||(p===42||p===43||p===45?"listUnordered":"listOrdered");if(k==="listUnordered"?!r.containerState.marker||p===r.containerState.marker:La(p)){if(r.containerState.type||(r.containerState.type=k,e.enter(k,{_container:!0})),k==="listUnordered")return e.enter("listItemPrefix"),p===42||p===45?e.check(Yi,n,c)(p):c(p);if(!r.interrupt||p===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(p)}return n(p)}function s(p){return La(p)&&++o<10?(e.consume(p),s):(!r.interrupt||o<2)&&(r.containerState.marker?p===r.containerState.marker:p===41||p===46)?(e.exit("listItemValue"),c(p)):n(p)}function c(p){return e.enter("listItemMarker"),e.consume(p),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||p,e.check(Ul,r.interrupt?n:d,e.attempt(Gx,g,f))}function d(p){return r.containerState.initialBlankLine=!0,l++,g(p)}function f(p){return ne(p)?(e.enter("listItemPrefixWhitespace"),e.consume(p),e.exit("listItemPrefixWhitespace"),g):n(p)}function g(p){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(p)}}function e1(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Ul,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,se(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!ne(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Jx,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,se(e,e.attempt(Ve,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function t1(e,t,n){const r=this;return se(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function n1(e){e.exit(this.containerState.type)}function r1(e,t,n){const r=this;return se(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!ne(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const _c={name:"setextUnderline",resolveTo:i1,tokenize:l1};function i1(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function l1(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,f;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){f=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||f)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),ne(c)?se(e,s,"lineSuffix")(c):s(c))}function s(c){return c===null||V(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const o1={tokenize:a1};function a1(e){const t=this,n=e.attempt(Ul,r,e.attempt(this.parser.constructs.flowInitial,i,se(e,e.attempt(this.parser.constructs.flow,i,e.attempt(fx,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const s1={resolveAll:Dp()},u1=Mp("string"),c1=Mp("text");function Mp(e){return{resolveAll:Dp(e==="text"?d1:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function c(d){if(d===null)return!0;const f=i[d];let g=-1;if(f)for(;++g<f.length;){const p=f[g];if(!p.previous||p.previous.call(r,r.previous))return!0}return!1}}}function Dp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function d1(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)s=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const f1={42:Ve,43:Ve,45:Ve,48:Ve,49:Ve,50:Ve,51:Ve,52:Ve,53:Ve,54:Ve,55:Ve,56:Ve,57:Ve,62:Np},p1={91:vx},h1={[-2]:wo,[-1]:wo,32:wo},m1={35:bx,42:Yi,45:[_c,Yi],60:_x,61:_c,95:Yi,96:jc,126:jc},g1={38:zp,92:Tp},v1={[-5]:So,[-4]:So,[-3]:So,33:Wx,38:zp,42:Pa,60:[qy,Ax],91:qx,92:[wx,Tp],93:Bs,95:Pa,96:ox},y1={null:[Pa,s1]},x1={null:[42,95]},k1={null:[]},w1=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:x1,contentInitial:p1,disable:k1,document:f1,flow:m1,flowInitial:h1,insideSpan:y1,string:g1,text:v1},Symbol.toStringTag,{value:"Module"}));function S1(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:E(_),check:E(S),consume:v,enter:y,exit:b,interrupt:E(S,{interrupt:!0})},c={code:null,containerState:{},defineSkip:w,events:[],now:k,parser:e,previous:null,sliceSerialize:g,sliceStream:p,write:f};let d=t.tokenize.call(c,s);return t.resolveAll&&l.push(t),c;function f(j){return o=ct(o,j),I(),o[o.length-1]!==null?[]:(L(t,0),c.events=Os(l,c.events,c),c.events)}function g(j,T){return C1(p(j),T)}function p(j){return b1(o,j)}function k(){const{_bufferIndex:j,_index:T,line:U,column:Q,offset:H}=r;return{_bufferIndex:j,_index:T,line:U,column:Q,offset:H}}function w(j){i[j.line]=j.column,P()}function I(){let j;for(;r._index<o.length;){const T=o[r._index];if(typeof T=="string")for(j=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===j&&r._bufferIndex<T.length;)h(T.charCodeAt(r._bufferIndex));else h(T)}}function h(j){d=d(j)}function v(j){V(j)?(r.line++,r.column=1,r.offset+=j===-3?2:1,P()):j!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=j}function y(j,T){const U=T||{};return U.type=j,U.start=k(),c.events.push(["enter",U,c]),a.push(U),U}function b(j){const T=a.pop();return T.end=k(),c.events.push(["exit",T,c]),T}function _(j,T){L(j,T.from)}function S(j,T){T.restore()}function E(j,T){return U;function U(Q,H,q){let ie,C,B,m;return Array.isArray(Q)?A(Q):"tokenize"in Q?A([Q]):z(Q);function z(J){return ve;function ve(be){const ee=be!==null&&J[be],Ee=be!==null&&J.null,Ue=[...Array.isArray(ee)?ee:ee?[ee]:[],...Array.isArray(Ee)?Ee:Ee?[Ee]:[]];return A(Ue)(be)}}function A(J){return ie=J,C=0,J.length===0?q:x(J[C])}function x(J){return ve;function ve(be){return m=D(),B=J,J.partial||(c.currentConstruct=J),J.name&&c.parser.constructs.disable.null.includes(J.name)?fe():J.tokenize.call(T?Object.assign(Object.create(c),T):c,s,X,fe)(be)}}function X(J){return j(B,m),H}function fe(J){return m.restore(),++C<ie.length?x(ie[C]):q}}}function L(j,T){j.resolveAll&&!l.includes(j)&&l.push(j),j.resolve&&Lt(c.events,T,c.events.length-T,j.resolve(c.events.slice(T),c)),j.resolveTo&&(c.events=j.resolveTo(c.events,c))}function D(){const j=k(),T=c.previous,U=c.currentConstruct,Q=c.events.length,H=Array.from(a);return{from:Q,restore:q};function q(){r=j,c.previous=T,c.currentConstruct=U,c.events.length=Q,a=H,P()}}function P(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function b1(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function C1(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function j1(e){const r={constructs:Py([w1,...(e||{}).extensions||[]]),content:i(By),defined:[],document:i(Uy),flow:i(o1),lazy:{},string:i(u1),text:i(c1)};return r;function i(l){return o;function o(a){return S1(r,l,a)}}}function E1(e){for(;!Lp(e););return e}const Nc=/[\0\t\n\r]/g;function _1(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let c,d,f,g,p;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),f=0,t="",n&&(l.charCodeAt(0)===65279&&f++,n=void 0);f<l.length;){if(Nc.lastIndex=f,c=Nc.exec(l),g=c&&c.index!==void 0?c.index:l.length,p=l.charCodeAt(g),!c){t=l.slice(f);break}if(p===10&&f===g&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),f<g&&(s.push(l.slice(f,g)),e+=g-f),p){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}f=g+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const N1=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function T1(e){return e.replace(N1,z1)}function z1(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return _p(n.slice(l?2:1),l?16:10)}return Fs(n)||e}const Rp={}.hasOwnProperty;function L1(e,t,n){return typeof t!="string"&&(n=t,t=void 0),P1(n)(E1(j1(n).document().write(_1()(e,t,!0))))}function P1(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Ys),autolinkProtocol:D,autolinkEmail:D,atxHeading:l(Qs),blockQuote:l(Ee),characterEscape:D,characterReference:D,codeFenced:l(Ue),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(Ue,o),codeText:l(Qt,o),codeTextData:D,data:D,codeFlowValue:D,definition:l(qt),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Yp),hardBreakEscape:l(qs),hardBreakTrailing:l(qs),htmlFlow:l(Ks,o),htmlFlowData:D,htmlText:l(Ks,o),htmlTextData:D,image:l(Xp),label:o,link:l(Ys),listItem:l(Gp),listItemValue:g,listOrdered:l(Xs,f),listUnordered:l(Xs),paragraph:l(Jp),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Qs),strong:l(Zp),thematicBreak:l(th)},exit:{atxHeading:s(),atxHeadingSequence:_,autolink:s(),autolinkEmail:ee,autolinkProtocol:be,blockQuote:s(),characterEscapeValue:P,characterReferenceMarkerHexadecimal:fe,characterReferenceMarkerNumeric:fe,characterReferenceValue:J,characterReference:ve,codeFenced:s(I),codeFencedFence:w,codeFencedFenceInfo:p,codeFencedFenceMeta:k,codeFlowValue:P,codeIndented:s(h),codeText:s(H),codeTextData:P,data:P,definition:s(),definitionDestinationString:b,definitionLabelString:v,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(T),hardBreakTrailing:s(T),htmlFlow:s(U),htmlFlowData:P,htmlText:s(Q),htmlTextData:P,image:s(ie),label:B,labelText:C,lineEnding:j,link:s(q),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:X,resourceDestinationString:m,resourceTitleString:z,resource:A,setextHeading:s(L),setextHeadingLineSequence:E,setextHeadingText:S,strong:s(),thematicBreak:s()}};Fp(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(N){let O={type:"root",children:[]};const W={stack:[O],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},Z=[];let le=-1;for(;++le<N.length;)if(N[le][1].type==="listOrdered"||N[le][1].type==="listUnordered")if(N[le][0]==="enter")Z.push(le);else{const mt=Z.pop();le=i(N,mt,le)}for(le=-1;++le<N.length;){const mt=t[N[le][0]];Rp.call(mt,N[le][1].type)&&mt[N[le][1].type].call(Object.assign({sliceSerialize:N[le][2].sliceSerialize},W),N[le][1])}if(W.tokenStack.length>0){const mt=W.tokenStack[W.tokenStack.length-1];(mt[1]||Tc).call(W,void 0,mt[0])}for(O.position={start:Yt(N.length>0?N[0][1].start:{line:1,column:1,offset:0}),end:Yt(N.length>0?N[N.length-2][1].end:{line:1,column:1,offset:0})},le=-1;++le<t.transforms.length;)O=t.transforms[le](O)||O;return O}function i(N,O,W){let Z=O-1,le=-1,mt=!1,yn,Pt,vr,yr;for(;++Z<=W;){const Je=N[Z];switch(Je[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Je[0]==="enter"?le++:le--,yr=void 0;break}case"lineEndingBlank":{Je[0]==="enter"&&(yn&&!yr&&!le&&!vr&&(vr=Z),yr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:yr=void 0}if(!le&&Je[0]==="enter"&&Je[1].type==="listItemPrefix"||le===-1&&Je[0]==="exit"&&(Je[1].type==="listUnordered"||Je[1].type==="listOrdered")){if(yn){let Mn=Z;for(Pt=void 0;Mn--;){const It=N[Mn];if(It[1].type==="lineEnding"||It[1].type==="lineEndingBlank"){if(It[0]==="exit")continue;Pt&&(N[Pt][1].type="lineEndingBlank",mt=!0),It[1].type="lineEnding",Pt=Mn}else if(!(It[1].type==="linePrefix"||It[1].type==="blockQuotePrefix"||It[1].type==="blockQuotePrefixWhitespace"||It[1].type==="blockQuoteMarker"||It[1].type==="listItemIndent"))break}vr&&(!Pt||vr<Pt)&&(yn._spread=!0),yn.end=Object.assign({},Pt?N[Pt][1].start:Je[1].end),N.splice(Pt||Z,0,["exit",yn,Je[2]]),Z++,W++}if(Je[1].type==="listItemPrefix"){const Mn={type:"listItem",_spread:!1,start:Object.assign({},Je[1].start),end:void 0};yn=Mn,N.splice(Z,0,["enter",Mn,Je[2]]),Z++,W++,vr=void 0,yr=!0}}}return N[O][1]._spread=mt,W}function l(N,O){return W;function W(Z){a.call(this,N(Z),Z),O&&O.call(this,Z)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(N,O,W){this.stack[this.stack.length-1].children.push(N),this.stack.push(N),this.tokenStack.push([O,W||void 0]),N.position={start:Yt(O.start),end:void 0}}function s(N){return O;function O(W){N&&N.call(this,W),c.call(this,W)}}function c(N,O){const W=this.stack.pop(),Z=this.tokenStack.pop();if(Z)Z[0].type!==N.type&&(O?O.call(this,N,Z[0]):(Z[1]||Tc).call(this,N,Z[0]));else throw new Error("Cannot close `"+N.type+"` ("+Ur({start:N.start,end:N.end})+"): it’s not open");W.position.end=Yt(N.end)}function d(){return zy(this.stack.pop())}function f(){this.data.expectingFirstListItemValue=!0}function g(N){if(this.data.expectingFirstListItemValue){const O=this.stack[this.stack.length-2];O.start=Number.parseInt(this.sliceSerialize(N),10),this.data.expectingFirstListItemValue=void 0}}function p(){const N=this.resume(),O=this.stack[this.stack.length-1];O.lang=N}function k(){const N=this.resume(),O=this.stack[this.stack.length-1];O.meta=N}function w(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function I(){const N=this.resume(),O=this.stack[this.stack.length-1];O.value=N.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function h(){const N=this.resume(),O=this.stack[this.stack.length-1];O.value=N.replace(/(\r?\n|\r)$/g,"")}function v(N){const O=this.resume(),W=this.stack[this.stack.length-1];W.label=O,W.identifier=ir(this.sliceSerialize(N)).toLowerCase()}function y(){const N=this.resume(),O=this.stack[this.stack.length-1];O.title=N}function b(){const N=this.resume(),O=this.stack[this.stack.length-1];O.url=N}function _(N){const O=this.stack[this.stack.length-1];if(!O.depth){const W=this.sliceSerialize(N).length;O.depth=W}}function S(){this.data.setextHeadingSlurpLineEnding=!0}function E(N){const O=this.stack[this.stack.length-1];O.depth=this.sliceSerialize(N).codePointAt(0)===61?1:2}function L(){this.data.setextHeadingSlurpLineEnding=void 0}function D(N){const W=this.stack[this.stack.length-1].children;let Z=W[W.length-1];(!Z||Z.type!=="text")&&(Z=eh(),Z.position={start:Yt(N.start),end:void 0},W.push(Z)),this.stack.push(Z)}function P(N){const O=this.stack.pop();O.value+=this.sliceSerialize(N),O.position.end=Yt(N.end)}function j(N){const O=this.stack[this.stack.length-1];if(this.data.atHardBreak){const W=O.children[O.children.length-1];W.position.end=Yt(N.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(O.type)&&(D.call(this,N),P.call(this,N))}function T(){this.data.atHardBreak=!0}function U(){const N=this.resume(),O=this.stack[this.stack.length-1];O.value=N}function Q(){const N=this.resume(),O=this.stack[this.stack.length-1];O.value=N}function H(){const N=this.resume(),O=this.stack[this.stack.length-1];O.value=N}function q(){const N=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";N.type+="Reference",N.referenceType=O,delete N.url,delete N.title}else delete N.identifier,delete N.label;this.data.referenceType=void 0}function ie(){const N=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";N.type+="Reference",N.referenceType=O,delete N.url,delete N.title}else delete N.identifier,delete N.label;this.data.referenceType=void 0}function C(N){const O=this.sliceSerialize(N),W=this.stack[this.stack.length-2];W.label=T1(O),W.identifier=ir(O).toLowerCase()}function B(){const N=this.stack[this.stack.length-1],O=this.resume(),W=this.stack[this.stack.length-1];if(this.data.inReference=!0,W.type==="link"){const Z=N.children;W.children=Z}else W.alt=O}function m(){const N=this.resume(),O=this.stack[this.stack.length-1];O.url=N}function z(){const N=this.resume(),O=this.stack[this.stack.length-1];O.title=N}function A(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function X(N){const O=this.resume(),W=this.stack[this.stack.length-1];W.label=O,W.identifier=ir(this.sliceSerialize(N)).toLowerCase(),this.data.referenceType="full"}function fe(N){this.data.characterReferenceType=N.type}function J(N){const O=this.sliceSerialize(N),W=this.data.characterReferenceType;let Z;W?(Z=_p(O,W==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):Z=Fs(O);const le=this.stack[this.stack.length-1];le.value+=Z}function ve(N){const O=this.stack.pop();O.position.end=Yt(N.end)}function be(N){P.call(this,N);const O=this.stack[this.stack.length-1];O.url=this.sliceSerialize(N)}function ee(N){P.call(this,N);const O=this.stack[this.stack.length-1];O.url="mailto:"+this.sliceSerialize(N)}function Ee(){return{type:"blockquote",children:[]}}function Ue(){return{type:"code",lang:null,meta:null,value:""}}function Qt(){return{type:"inlineCode",value:""}}function qt(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Yp(){return{type:"emphasis",children:[]}}function Qs(){return{type:"heading",depth:0,children:[]}}function qs(){return{type:"break"}}function Ks(){return{type:"html",value:""}}function Xp(){return{type:"image",title:null,url:"",alt:null}}function Ys(){return{type:"link",title:null,url:"",children:[]}}function Xs(N){return{type:"list",ordered:N.type==="listOrdered",start:null,spread:N._spread,children:[]}}function Gp(N){return{type:"listItem",spread:N._spread,checked:null,children:[]}}function Jp(){return{type:"paragraph",children:[]}}function Zp(){return{type:"strong",children:[]}}function eh(){return{type:"text",value:""}}function th(){return{type:"thematicBreak"}}}function Yt(e){return{line:e.line,column:e.column,offset:e.offset}}function Fp(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Fp(e,r):I1(e,r)}}function I1(e,t){let n;for(n in t)if(Rp.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function Tc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Ur({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Ur({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Ur({start:t.start,end:t.end})+") is still open")}function A1(e){const t=this;t.parser=n;function n(r){return L1(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function M1(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function D1(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function R1(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function F1(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function O1(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function B1(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=gr(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const c={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,c),e.applyData(t,c)}function $1(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function U1(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Op(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function H1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Op(e,t);const i={src:gr(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function V1(e,t){const n={src:gr(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function W1(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function Q1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Op(e,t);const i={href:gr(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function q1(e,t){const n={href:gr(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function K1(e,t,n){const r=e.all(t),i=n?Y1(n):Bp(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let f;d&&d.type==="element"&&d.tagName==="p"?f=d:(f={type:"element",tagName:"p",properties:{},children:[]},r.unshift(f)),f.children.length>0&&f.children.unshift({type:"text",value:" "}),f.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function Y1(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Bp(n[r])}return t}function Bp(e){const t=e.spread;return t??e.children.length>1}function X1(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function G1(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function J1(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function Z1(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function e0(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=As(t.children[1]),s=kp(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function t0(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const c=[];for(;++s<a;){const f=t.children[s],g={},p=o?o[s]:void 0;p&&(g.align=p);let k={type:"element",tagName:l,properties:g,children:[]};f&&(k.children=e.all(f),e.patch(f,k),k=e.applyData(f,k)),c.push(k)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function n0(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const zc=9,Lc=32;function r0(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(Pc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(Pc(t.slice(i),i>0,!1)),l.join("")}function Pc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===zc||l===Lc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===zc||l===Lc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function i0(e,t){const n={type:"text",value:r0(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function l0(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const o0={blockquote:M1,break:D1,code:R1,delete:F1,emphasis:O1,footnoteReference:B1,heading:$1,html:U1,imageReference:H1,image:V1,inlineCode:W1,linkReference:Q1,link:q1,listItem:K1,list:X1,paragraph:G1,root:J1,strong:Z1,table:e0,tableCell:n0,tableRow:t0,text:i0,thematicBreak:l0,toml:Ai,yaml:Ai,definition:Ai,footnoteDefinition:Ai};function Ai(){}const $p=-1,Hl=0,Vr=1,bl=2,$s=3,Us=4,Hs=5,Vs=6,Up=7,Hp=8,Ic=typeof self=="object"?self:globalThis,a0=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Hl:case $p:return n(o,i);case Vr:{const a=n([],i);for(const s of o)a.push(r(s));return a}case bl:{const a=n({},i);for(const[s,c]of o)a[r(s)]=r(c);return a}case $s:return n(new Date(o),i);case Us:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Hs:{const a=n(new Map,i);for(const[s,c]of o)a.set(r(s),r(c));return a}case Vs:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Up:{const{name:a,message:s}=o;return n(new Ic[a](s),i)}case Hp:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new Ic[l](o),i)};return r},Ac=e=>a0(new Map,e)(0),Fn="",{toString:s0}={},{keys:u0}=Object,Nr=e=>{const t=typeof e;if(t!=="object"||!e)return[Hl,t];const n=s0.call(e).slice(8,-1);switch(n){case"Array":return[Vr,Fn];case"Object":return[bl,Fn];case"Date":return[$s,Fn];case"RegExp":return[Us,Fn];case"Map":return[Hs,Fn];case"Set":return[Vs,Fn];case"DataView":return[Vr,n]}return n.includes("Array")?[Vr,n]:n.includes("Error")?[Up,n]:[bl,n]},Mi=([e,t])=>e===Hl&&(t==="function"||t==="symbol"),c0=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=Nr(o);switch(a){case Hl:{let d=o;switch(s){case"bigint":a=Hp,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([$p],o)}return i([a,d],o)}case Vr:{if(s){let g=o;return s==="DataView"?g=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(g=new Uint8Array(o)),i([s,[...g]],o)}const d=[],f=i([a,d],o);for(const g of o)d.push(l(g));return f}case bl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],f=i([a,d],o);for(const g of u0(o))(e||!Mi(Nr(o[g])))&&d.push([l(g),l(o[g])]);return f}case $s:return i([a,o.toISOString()],o);case Us:{const{source:d,flags:f}=o;return i([a,{source:d,flags:f}],o)}case Hs:{const d=[],f=i([a,d],o);for(const[g,p]of o)(e||!(Mi(Nr(g))||Mi(Nr(p))))&&d.push([l(g),l(p)]);return f}case Vs:{const d=[],f=i([a,d],o);for(const g of o)(e||!Mi(Nr(g)))&&d.push(l(g));return f}}const{message:c}=o;return i([a,{name:s,message:c}],o)};return l},Mc=(e,{json:t,lossy:n}={})=>{const r=[];return c0(!(t||n),!!t,new Map,r)(e),r},Cl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?Ac(Mc(e,t)):structuredClone(e):(e,t)=>Ac(Mc(e,t));function d0(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function f0(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function p0(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||d0,r=e.options.footnoteBackLabel||f0,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[s]);if(!c)continue;const d=e.all(c),f=String(c.identifier).toUpperCase(),g=gr(f.toLowerCase());let p=0;const k=[],w=e.footnoteCounts.get(f);for(;w!==void 0&&++p<=w;){k.length>0&&k.push({type:"text",value:" "});let v=typeof n=="string"?n:n(s,p);typeof v=="string"&&(v={type:"text",value:v}),k.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+g+(p>1?"-"+p:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,p),className:["data-footnote-backref"]},children:Array.isArray(v)?v:[v]})}const I=d[d.length-1];if(I&&I.type==="element"&&I.tagName==="p"){const v=I.children[I.children.length-1];v&&v.type==="text"?v.value+=" ":I.children.push({type:"text",value:" "}),I.children.push(...k)}else d.push(...k);const h={type:"element",tagName:"li",properties:{id:t+"fn-"+g},children:e.wrap(d,!0)};e.patch(c,h),a.push(h)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...Cl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const Vp=function(e){if(e==null)return v0;if(typeof e=="function")return Vl(e);if(typeof e=="object")return Array.isArray(e)?h0(e):m0(e);if(typeof e=="string")return g0(e);throw new Error("Expected function, string, or object as test")};function h0(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=Vp(e[n]);return Vl(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function m0(e){const t=e;return Vl(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function g0(e){return Vl(t);function t(n){return n&&n.type===e}}function Vl(e){return t;function t(n,r,i){return!!(y0(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function v0(){return!0}function y0(e){return e!==null&&typeof e=="object"&&"type"in e}const Wp=[],x0=!0,Dc=!1,k0="skip";function w0(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=Vp(i),o=r?-1:1;a(e,void 0,[])();function a(s,c,d){const f=s&&typeof s=="object"?s:{};if(typeof f.type=="string"){const p=typeof f.tagName=="string"?f.tagName:typeof f.name=="string"?f.name:void 0;Object.defineProperty(g,"name",{value:"node ("+(s.type+(p?"<"+p+">":""))+")"})}return g;function g(){let p=Wp,k,w,I;if((!t||l(s,c,d[d.length-1]||void 0))&&(p=S0(n(s,d)),p[0]===Dc))return p;if("children"in s&&s.children){const h=s;if(h.children&&p[0]!==k0)for(w=(r?h.children.length:-1)+o,I=d.concat(h);w>-1&&w<h.children.length;){const v=h.children[w];if(k=a(v,w,I)(),k[0]===Dc)return k;w=typeof k[1]=="number"?k[1]:w+o}}return p}}}function S0(e){return Array.isArray(e)?e:typeof e=="number"?[x0,e]:e==null?Wp:[e]}function Qp(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),w0(e,l,a,i);function a(s,c){const d=c[c.length-1],f=d?d.children.indexOf(s):void 0;return o(s,f,d)}}const Ia={}.hasOwnProperty,b0={};function C0(e,t){const n=t||b0,r=new Map,i=new Map,l=new Map,o={...o0,...n.handlers},a={all:c,applyData:E0,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:j0,wrap:N0};return Qp(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const f=d.type==="definition"?r:i,g=String(d.identifier).toUpperCase();f.has(g)||f.set(g,d)}}),a;function s(d,f){const g=d.type,p=a.handlers[g];if(Ia.call(a.handlers,g)&&p)return p(a,d,f);if(a.options.passThrough&&a.options.passThrough.includes(g)){if("children"in d){const{children:w,...I}=d,h=Cl(I);return h.children=a.all(d),h}return Cl(d)}return(a.options.unknownHandler||_0)(a,d,f)}function c(d){const f=[];if("children"in d){const g=d.children;let p=-1;for(;++p<g.length;){const k=a.one(g[p],d);if(k){if(p&&g[p-1].type==="break"&&(!Array.isArray(k)&&k.type==="text"&&(k.value=Rc(k.value)),!Array.isArray(k)&&k.type==="element")){const w=k.children[0];w&&w.type==="text"&&(w.value=Rc(w.value))}Array.isArray(k)?f.push(...k):f.push(k)}}}return f}}function j0(e,t){e.position&&(t.position=sy(e))}function E0(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,Cl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function _0(e,t){const n=t.data||{},r="value"in t&&!(Ia.call(n,"hProperties")||Ia.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function N0(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Rc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Fc(e,t){const n=C0(e,t),r=n.one(e,void 0),i=p0(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function T0(e,t){return e&&"run"in e?async function(n,r){const i=Fc(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Fc(n,{file:r,...e||t})}}function Oc(e){if(e)throw e}var Xi=Object.prototype.hasOwnProperty,qp=Object.prototype.toString,Bc=Object.defineProperty,$c=Object.getOwnPropertyDescriptor,Uc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):qp.call(t)==="[object Array]"},Hc=function(t){if(!t||qp.call(t)!=="[object Object]")return!1;var n=Xi.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Xi.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Xi.call(t,i)},Vc=function(t,n){Bc&&n.name==="__proto__"?Bc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Wc=function(t,n){if(n==="__proto__")if(Xi.call(t,n)){if($c)return $c(t,n).value}else return;return t[n]},z0=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<c;++s)if(t=arguments[s],t!=null)for(n in t)r=Wc(a,n),i=Wc(t,n),a!==i&&(d&&i&&(Hc(i)||(l=Uc(i)))?(l?(l=!1,o=r&&Uc(r)?r:[]):o=r&&Hc(r)?r:{},Vc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Vc(a,{name:n,newValue:i}));return a};const bo=Da(z0);function Aa(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function L0(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...c){const d=e[++l];let f=-1;if(s){o(s);return}for(;++f<i.length;)(c[f]===null||c[f]===void 0)&&(c[f]=i[f]);i=c,d?P0(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function P0(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const Et={basename:I0,dirname:A0,extname:M0,join:D0,sep:"/"};function I0(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');mi(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function A0(e){if(mi(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function M0(e){mi(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function D0(...e){let t=-1,n;for(;++t<e.length;)mi(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":R0(n)}function R0(e){mi(e);const t=e.codePointAt(0)===47;let n=F0(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function F0(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function mi(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const O0={cwd:B0};function B0(){return"/"}function Ma(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function $0(e){if(typeof e=="string")e=new URL(e);else if(!Ma(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return U0(e)}function U0(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const Co=["history","path","basename","stem","extname","dirname"];class Kp{constructor(t){let n;t?Ma(t)?n={path:t}:typeof t=="string"||H0(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":O0.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<Co.length;){const l=Co[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)Co.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?Et.basename(this.path):void 0}set basename(t){Eo(t,"basename"),jo(t,"basename"),this.path=Et.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?Et.dirname(this.path):void 0}set dirname(t){Qc(this.basename,"dirname"),this.path=Et.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?Et.extname(this.path):void 0}set extname(t){if(jo(t,"extname"),Qc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=Et.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){Ma(t)&&(t=$0(t)),Eo(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?Et.basename(this.path,this.extname):void 0}set stem(t){Eo(t,"stem"),jo(t,"stem"),this.path=Et.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new De(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function jo(e,t){if(e&&e.includes(Et.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+Et.sep+"`")}function Eo(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Qc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function H0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const V0=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},W0={}.hasOwnProperty;class Ws extends V0{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=L0()}copy(){const t=new Ws;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(bo(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(To("data",this.frozen),this.namespace[t]=n,this):W0.call(this.namespace,t)&&this.namespace[t]||void 0:t?(To("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Di(t),r=this.parser||this.Parser;return _o("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),_o("process",this.parser||this.Parser),No("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Di(t),s=r.parse(a);r.run(s,a,function(d,f,g){if(d||!f||!g)return c(d);const p=f,k=r.stringify(p,g);K0(k)?g.value=k:g.result=k,c(d,g)});function c(d,f){d||!f?o(d):l?l(f):n(void 0,f)}}}processSync(t){let n=!1,r;return this.freeze(),_o("processSync",this.parser||this.Parser),No("processSync",this.compiler||this.Compiler),this.process(t,i),Kc("processSync","process",n),r;function i(l,o){n=!0,Oc(l),r=o}}run(t,n,r){qc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Di(n);i.run(t,s,c);function c(d,f,g){const p=f||t;d?a(d):o?o(p):r(void 0,p,g)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Kc("runSync","run",r),i;function l(o,a){Oc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Di(n),i=this.compiler||this.Compiler;return No("stringify",i),qc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(To("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")s(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...f]=c;s(d,f)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=bo(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const f=c[d];l(f)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function s(c,d){let f=-1,g=-1;for(;++f<r.length;)if(r[f][0]===c){g=f;break}if(g===-1)r.push([c,...d]);else if(d.length>0){let[p,...k]=d;const w=r[g][1];Aa(w)&&Aa(p)&&(p=bo(!0,w,p)),r[g]=[c,p,...k]}}}}const Q0=new Ws().freeze();function _o(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function No(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function To(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function qc(e){if(!Aa(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Kc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Di(e){return q0(e)?e:new Kp(e)}function q0(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function K0(e){return typeof e=="string"||Y0(e)}function Y0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const X0="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Yc=[],Xc={allowDangerousHtml:!0},G0=/^(https?|ircs?|mailto|xmpp)$/i,J0=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function Z0(e){const t=ek(e),n=tk(e);return nk(t.runSync(t.parse(n),n),e)}function ek(e){const t=e.rehypePlugins||Yc,n=e.remarkPlugins||Yc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Xc}:Xc;return Q0().use(A1).use(n).use(T0,r).use(t)}function tk(e){const t=e.children||"",n=new Kp;return typeof t=="string"&&(n.value=t),n}function nk(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||rk;for(const d of J0)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+X0+d.id,void 0);return Qp(e,c),py(e,{Fragment:u.Fragment,components:i,ignoreInvalidStyle:!0,jsx:u.jsx,jsxs:u.jsxs,passKeys:!0,passNode:!0});function c(d,f,g){if(d.type==="raw"&&g&&typeof f=="number")return o?g.children.splice(f,1):g.children[f]={type:"text",value:d.value},f;if(d.type==="element"){let p;for(p in ko)if(Object.hasOwn(ko,p)&&Object.hasOwn(d.properties,p)){const k=d.properties[p],w=ko[p];(w===null||w.includes(d.tagName))&&(d.properties[p]=s(String(k||""),p,d))}}if(d.type==="element"){let p=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!p&&r&&typeof f=="number"&&(p=!r(d,f,g)),p&&g&&typeof f=="number")return a&&d.children?g.children.splice(f,1,...d.children):g.children.splice(f,1),f}}}function rk(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||G0.test(e.slice(0,t))?e:""}const ik=e=>{if(!e)return null;try{return JSON.parse(e).execution_stats||null}catch{return null}},lk=e=>{if(e.kind!=="status")return!1;const t=e.content.toLowerCase();return t.includes("running")||t.includes("thinking")||t.includes("executing")||t.includes("processing")},Gc=10*1024,zo=200,Ie={send:u.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),u.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),u.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),u.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),file:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),spinner:u.jsx("svg",{className:"spinner-icon",width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 12a9 9 0 1 1-6.219-8.56"})})},ok=e=>{switch(e){case"directive":return Ie.directive;case"question":return Ie.question;case"status":return Ie.status;case"result":return Ie.result;case"approval_request":return Ie.lock;default:return Ie.directive}},ak=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=F.useRef(null),[a,s]=Xt.useState(""),[c,d]=Xt.useState("directive"),[f,g]=Xt.useState(""),[p,k]=Xt.useState(!1),[w,I]=Xt.useState(new Map),[h,v]=Xt.useState(new Set),[y,b]=F.useState(new Set),[_,S]=F.useState(new Set),E=C=>{const B=(C.match(/\n/g)||[]).length+1;if(!(C.length>Gc||B>zo))return{needsTruncation:!1,truncated:C,fullLength:C.length,lineCount:B};let z=C.slice(0,Gc);const A=z.split(`
`);A.length>zo&&(z=A.slice(0,zo).join(`
`));const x=z.lastIndexOf(`
`);return x>z.length*.8&&(z=z.slice(0,x)),{needsTruncation:!0,truncated:z,fullLength:C.length,lineCount:B}},L=C=>{b(B=>{const m=new Set(B);return m.has(C)?m.delete(C):m.add(C),m})};F.useEffect(()=>{e!=null&&e.workspace?g(e.workspace):g("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),F.useEffect(()=>{var C;(C=o.current)==null||C.scrollIntoView({behavior:"smooth"})},[t]);const D=C=>{g(C),r&&r(C)},P=()=>{a.trim()&&(n(a,c,f||void 0),s(""))},j=C=>{C.key==="Enter"&&!C.shiftKey&&(C.preventDefault(),P())},T=C=>new Date(C).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),U=C=>C.length>12?`${C.slice(0,8)}...`:C,Q=C=>{if(!C.metadata_json)return null;try{return JSON.parse(C.metadata_json).approval_id||null}catch{return null}},H=C=>{const B=w.get(C)||"";i&&(i(C,B),v(m=>new Set(m).add(C)),I(m=>{const z=new Map(m);return z.delete(C),z}))},q=C=>{const B=w.get(C)||"";if(!B.trim()){alert("Please provide a reason for rejection");return}l&&(l(C,B),v(m=>new Set(m).add(C)),I(m=>{const z=new Map(m);return z.delete(C),z}))},ie=(C,B)=>{I(m=>new Map(m).set(C,B))};return e?u.jsxs("div",{className:"conversation-view",children:[u.jsxs("div",{className:"conversation-header",children:[u.jsxs("div",{className:"header-info",children:[u.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&u.jsxs("span",{className:"thread-agent-badge",children:[Ie.bot,e.target_agent]})]}),u.jsxs("div",{className:"header-stats",children:[u.jsxs("span",{className:"message-count",children:[t.length," messages"]}),u.jsx("span",{className:"thread-id",title:e.id,children:U(e.id)})]})]}),u.jsxs("div",{className:"messages-container",children:[t.length===0?u.jsxs("div",{className:"empty-messages",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("p",{children:"No messages yet"}),u.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((C,B)=>{const m=C.from_type==="human",z=B===0||t[B-1].from_type!==C.from_type,A=y.has(C.id),{needsTruncation:x,truncated:X,fullLength:fe,lineCount:J}=E(C.content),ve=A?C.content:X,be=lk(C);return u.jsxs("div",{className:`message ${m?"human":"agent"}${be?" running-status":""}`,children:[u.jsx("div",{className:`message-avatar ${z?"visible":""}`,children:z&&(m?Ie.user:Ie.bot)}),u.jsxs("div",{className:"message-body",children:[z&&u.jsxs("div",{className:"message-meta",children:[u.jsx("span",{className:"sender-name",children:C.from_id}),u.jsxs("span",{className:`kind-badge${be?" running":""}`,children:[be?Ie.spinner:ok(C.kind)," ",C.kind]}),u.jsx("span",{className:"message-time",children:T(C.created_at)})]}),u.jsxs("div",{className:"message-content",children:[C.kind==="result"||!m?u.jsx(Z0,{components:{a:({href:ee,children:Ee})=>{let Ue=ee;return ee&&ee.startsWith("/")&&!ee.startsWith("//")&&(Ue=`file://${ee}`),u.jsx("a",{href:Ue,target:"_blank",rel:"noopener noreferrer",children:Ee})},code:({className:ee,children:Ee,...Ue})=>!ee?u.jsx("code",{className:"inline-code",...Ue,children:Ee}):u.jsx("code",{className:ee,...Ue,children:Ee})},children:ve}):ve,x&&u.jsx("div",{className:"truncation-notice",children:u.jsx("button",{className:"expand-btn",onClick:()=>L(C.id),children:A?u.jsx(u.Fragment,{children:"Show less"}):u.jsxs(u.Fragment,{children:["Show more (",Math.round(fe/1024),"KB, ",J," lines)"]})})}),C.kind==="approval_request"&&(()=>{const ee=Q(C),Ee=ee&&h.has(ee);return ee?u.jsx("div",{className:"inline-approval",children:Ee?u.jsxs("div",{className:"approval-handled",children:[Ie.check,u.jsx("span",{children:"Action taken"})]}):u.jsxs(u.Fragment,{children:[u.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:w.get(ee)||"",onChange:Ue=>ie(ee,Ue.target.value)}),u.jsxs("div",{className:"approval-actions",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>q(ee),title:"Reject",children:[Ie.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>H(ee),title:"Approve",children:[Ie.check,"Approve"]})]})]})}):null})(),C.kind==="result"&&(()=>{const ee=ik(C.metadata_json);if(!ee||!ee.files_created||ee.files_created.length===0)return null;const Ee=_.has(C.id),Ue=()=>{S(Qt=>{const qt=new Set(Qt);return qt.has(C.id)?qt.delete(C.id):qt.add(C.id),qt})};return u.jsxs("div",{className:"files-created-section",children:[u.jsxs("button",{className:`files-toggle-btn ${Ee?"expanded":""}`,onClick:Ue,children:[Ie.file,u.jsxs("span",{children:["Files Created (",ee.files_created.length,")"]}),ee.workspace&&u.jsxs("span",{className:"workspace-badge",title:ee.workspace,children:[Ie.folder,ee.workspace.split("/").pop()]}),u.jsx("span",{className:"toggle-chevron",children:Ee?"▼":"▶"})]}),Ee&&u.jsx("ul",{className:"files-list",children:ee.files_created.map((Qt,qt)=>u.jsx("li",{className:"file-item",children:u.jsx("a",{href:`file://${ee.workspace?ee.workspace+"/":""}${Qt}`,target:"_blank",rel:"noopener noreferrer",title:Qt,children:Qt})},qt))})]})})()]}),u.jsx("div",{className:"message-footer",children:u.jsxs("span",{className:"message-seq",children:["#",C.message_seq]})})]})]},C.id)}),u.jsx("div",{ref:o})]}),u.jsxs("div",{className:"input-area",children:[p&&u.jsxs("div",{className:"workspace-input-row",children:[u.jsx("input",{type:"text",value:f,onChange:C=>D(C.target.value),onBlur:()=>{r&&r(f)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),u.jsx("button",{onClick:async()=>{try{const B=await(await fetch("/api/select-folder")).json();!B.cancelled&&B.path&&D(B.path)}catch(C){console.error("Failed to open folder picker:",C)}},className:"workspace-browse",title:"Browse for folder",children:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),u.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),u.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),f&&u.jsx("button",{onClick:()=>{D(""),k(!1)},className:"workspace-clear",children:"Clear"})]}),u.jsxs("div",{className:"input-wrapper",children:[u.jsx("button",{onClick:()=>k(!p),className:`workspace-toggle ${f?"has-workspace":""}`,title:f||"Set working directory",children:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),u.jsxs("select",{value:c,onChange:C=>d(C.target.value),className:"kind-selector",children:[u.jsx("option",{value:"directive",children:"Directive"}),u.jsx("option",{value:"question",children:"Question"})]}),u.jsx("textarea",{value:a,onChange:C=>s(C.target.value),onKeyPress:j,placeholder:f?`Message (workspace: ${f.split("/").pop()})`:"Type a message...",rows:1}),u.jsx("button",{onClick:P,className:"send-btn",disabled:!a.trim(),children:Ie.send})]}),u.jsxs("div",{className:"input-hint",children:["Press ",u.jsx("kbd",{children:"Enter"})," to send, ",u.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),u.jsx("style",{children:`
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
          width: 36px;
          height: 36px;
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
          align-items: flex-end;
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
          padding: var(--space-2) var(--space-3);
          padding-right: var(--space-6);
          background: var(--bg-elevated);
          color: var(--text-secondary);
          font-size: var(--text-xs);
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          appearance: none;
          background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
          background-repeat: no-repeat;
          background-position: right var(--space-2) center;
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
      `})]}):null};class Jc{constructor(){He(this,"ws",null);He(this,"wsUrl",null);He(this,"isConnecting",!1);He(this,"reconnectTimeout",null);He(this,"reconnectAttempts",0);He(this,"maxReconnectAttempts",10);He(this,"connectionState","disconnected");He(this,"stateListeners",new Set);He(this,"messageHandlers",new Set);He(this,"batchHandlers",new Set);He(this,"errorHandlers",new Set);He(this,"subscriptions",new Map);He(this,"hookCount",0)}getState(){return{isConnected:this.connectionState==="connected",connectionState:this.connectionState,reconnectAttempts:this.reconnectAttempts}}subscribeToState(t){return this.stateListeners.add(t),t(this.connectionState,this.reconnectAttempts),()=>this.stateListeners.delete(t)}setConnectionState(t){this.connectionState=t,this.stateListeners.forEach(n=>n(t,this.reconnectAttempts))}registerHook(t,n,r){this.hookCount++,console.log(`[WebSocketService] Hook registered, count: ${this.hookCount}`);const i=t?a=>t(a):null,l=n?a=>n(a):null,o=r?a=>r(a):null;return i&&this.messageHandlers.add(i),l&&this.batchHandlers.add(l),o&&this.errorHandlers.add(o),()=>{this.hookCount--,console.log(`[WebSocketService] Hook unregistered, count: ${this.hookCount}`),i&&this.messageHandlers.delete(i),l&&this.batchHandlers.delete(l),o&&this.errorHandlers.delete(o),this.hookCount===0&&(console.log("[WebSocketService] All hooks unregistered, closing connection"),this.disconnect())}}connect(t,n,r=10){this.maxReconnectAttempts=r;const i=`${t}?instance_id=${n}`;if(this.ws&&this.ws.readyState===WebSocket.OPEN&&this.wsUrl===i){console.log("[WebSocketService] Already connected, skipping");return}if(this.isConnecting){console.log("[WebSocketService] Already connecting, skipping");return}if(this.ws&&this.ws.readyState===WebSocket.CONNECTING){console.log("[WebSocketService] Connection pending, skipping");return}this.ws&&this.wsUrl!==i&&(console.log("[WebSocketService] URL changed, closing old connection"),this.ws.close(),this.ws=null),this.isConnecting=!0,this.wsUrl=i,console.log(`[WebSocketService] Creating new WebSocket to ${i}`),this.setConnectionState(this.reconnectAttempts>0?"reconnecting":"connecting");try{this.ws=new WebSocket(i),this.ws.onopen=()=>{console.log("[WebSocketService] Connected"),this.isConnecting=!1,this.reconnectAttempts=0,this.setConnectionState("connected"),this.subscriptions.forEach((l,o)=>{this.subscribe(o,l)})},this.ws.onmessage=l=>{try{const o=JSON.parse(l.data);this.handleEvent(o)}catch(o){console.error("[WebSocketService] Failed to parse message:",o)}},this.ws.onerror=l=>{console.error("[WebSocketService] Error:",l),this.isConnecting=!1},this.ws.onclose=()=>{if(console.log("[WebSocketService] Disconnected"),this.isConnecting=!1,this.setConnectionState("disconnected"),this.hookCount>0&&this.reconnectAttempts<this.maxReconnectAttempts){const l=this.getBackoffDelay(this.reconnectAttempts);console.log(`[WebSocketService] Reconnecting in ${l}ms (attempt ${this.reconnectAttempts+1}/${this.maxReconnectAttempts})`),this.reconnectTimeout=setTimeout(()=>{this.reconnectAttempts++,this.connect(t,n,r)},l)}}}catch(l){console.error("[WebSocketService] Failed to connect:",l),this.isConnecting=!1,this.setConnectionState("disconnected")}}disconnect(){this.reconnectTimeout&&(clearTimeout(this.reconnectTimeout),this.reconnectTimeout=null),this.ws&&(this.ws.close(),this.ws=null),this.wsUrl=null,this.reconnectAttempts=0,this.subscriptions.clear(),this.setConnectionState("disconnected")}send(t){this.ws&&this.ws.readyState===WebSocket.OPEN?this.ws.send(JSON.stringify(t)):console.warn("[WebSocketService] Not connected, cannot send")}handleEvent(t){switch(t.type){case"message":t.data&&this.messageHandlers.forEach(n=>n(t.data));break;case"batch":if(t.data){const n=t.data;this.batchHandlers.forEach(r=>r(n)),n.messages.forEach(r=>{this.messageHandlers.forEach(i=>i(r))})}break;case"error":t.data&&this.errorHandlers.forEach(n=>n(t.data)),console.error("[WebSocketService] Error event:",t.data);break;case"pong":break;default:console.log("[WebSocketService] Unknown event:",t.type)}}getBackoffDelay(t,n=1e3,r=3e4){const i=Math.min(n*Math.pow(2,t),r),l=i*Math.random()*.3;return Math.round(i+l)}subscribe(t,n=0){this.subscriptions.set(t,n),this.send({type:"subscribe",timestamp:Date.now(),data:{thread_id:t,from_seq:n}})}unsubscribe(t){this.subscriptions.delete(t)}acknowledge(t,n){const r=this.subscriptions.get(t)||0;n>r&&this.subscriptions.set(t,n),this.send({type:"ack",timestamp:Date.now(),data:{thread_id:t,ack_seq:n}})}ping(){this.send({type:"ping",timestamp:Date.now()})}}function sk(){return typeof window<"u"?(window.__AILANG_WS_SERVICE__?console.log("[WebSocketService] Reusing existing singleton instance"):(console.log("[WebSocketService] Creating new singleton instance"),window.__AILANG_WS_SERVICE__=new Jc),window.__AILANG_WS_SERVICE__):new Jc}const Ct=sk();function uk(e){return Ct.subscribeToState(e)}const ck=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,maxReconnectAttempts:l=10})=>{const[o,a]=F.useState(Ct.getState().isConnected),[s,c]=F.useState(null),d=F.useRef(n),f=F.useRef(r),g=F.useRef(i);F.useEffect(()=>{d.current=n},[n]),F.useEffect(()=>{f.current=r},[r]),F.useEffect(()=>{g.current=i},[i]),F.useEffect(()=>{const h=_=>{d.current&&d.current(_)},v=_=>{f.current&&f.current(_)},y=_=>{g.current&&g.current(_)},b=Ct.registerHook(h,v,y);return Ct.connect(e,t,l),b},[e,t,l]),F.useEffect(()=>Ct.subscribeToState((v,y)=>{a(v==="connected"),y>=l?c("Connection lost. Please refresh the page."):c(null)}),[l]),F.useEffect(()=>{if(!o)return;const h=setInterval(()=>{Ct.ping()},3e4);return()=>clearInterval(h)},[o]);const p=F.useCallback((h,v=0)=>{Ct.subscribe(h,v)},[]),k=F.useCallback(h=>{Ct.unsubscribe(h)},[]),w=F.useCallback((h,v)=>{Ct.acknowledge(h,v)},[]),I=F.useCallback(()=>{Ct.ping()},[]);return{isConnected:o,connectionError:s,subscribe:p,unsubscribe:k,acknowledge:w,ping:I}},dk=({connected:e})=>u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?u.jsxs(u.Fragment,{children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):u.jsxs(u.Fragment,{children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),u.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),fk=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=F.useState([]),[o,a]=F.useState(null),[s,c]=F.useState(new Map),[d,f]=F.useState(new Map),[g,p]=F.useState([]),[k,w]=F.useState(!1),[I,h]=F.useState(""),{isConnected:v,subscribe:y,acknowledge:b}=ck({url:e,instanceId:t,onMessage:_,onBatch:S});function _(m){const z={id:m.id,thread_id:m.thread_id,message_seq:m.message_seq,created_at:m.created_at,from_type:m.from_type,from_id:m.from_id,to_type:m.to_type,to_id:m.to_id,kind:m.kind,subject:m.subject,content:m.content,metadata_json:m.metadata_json,delivery_state:"visible",business_state:"open"};c(A=>{const x=A.get(z.thread_id)||[];return x.find(X=>X.id===z.id)?A:new Map(A).set(z.thread_id,[...x,z].sort((X,fe)=>X.message_seq-fe.message_seq))}),z.thread_id!==o&&f(A=>{const x=A.get(z.thread_id)||0;return new Map(A).set(z.thread_id,x+1)}),b(z.thread_id,z.message_seq)}function S(m){m.messages.forEach(z=>{_(z)})}const E=F.useCallback(m=>{if(a(m),f(z=>{const A=new Map(z);return A.delete(m),A}),v){const z=s.get(m)||[],A=z.length>0?Math.max(...z.map(x=>x.message_seq)):0;y(m,A)}},[v,y,s]),L=F.useCallback(async(m,z,A)=>{if(!o)return;const x=A?JSON.stringify({workspace:A}):void 0;try{const X=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:z,content:m,metadata_json:x})});if(!X.ok){console.error("Failed to send message:",await X.text());return}const fe=await X.json();c(J=>{const ve=J.get(o)||[];return ve.find(be=>be.id===fe.id)?J:new Map(J).set(o,[...ve,fe])})}catch(X){console.error("Error sending message:",X)}},[o,t]);F.useEffect(()=>{(async()=>{try{const z=await fetch("/api/threads");if(!z.ok){console.error("Failed to fetch threads:",await z.text());return}const A=await z.json();l(A),A.length>0&&!o&&a(A[0].id)}catch(z){console.error("Error fetching threads:",z)}})()},[]),F.useEffect(()=>{if(!o)return;const m=o;(async()=>{try{const A=await fetch(`/api/messages?thread_id=${m}`);if(!A.ok){console.error("Failed to fetch messages:",await A.text());return}const x=await A.json();c(X=>{const fe=X.get(m)||[],J=x?[...x]:[];for(const ve of fe)J.find(be=>be.id===ve.id)||J.push(ve);return J.sort((ve,be)=>ve.message_seq-be.message_seq),new Map(X).set(m,J)})}catch(A){console.error("Error fetching messages:",A)}})()},[o]);const D=F.useRef(null);F.useEffect(()=>{n&&n!==D.current&&i.length>0&&(i.some(z=>z.id===n)&&(D.current=n,a(n),f(z=>{const A=new Map(z);return A.delete(n),A})),r&&r())},[n,i,r]);const P=F.useCallback(async m=>{try{const z=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:m,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!z.ok){console.error("Failed to create thread:",await z.text());return}const A=await z.json();l(x=>[A,...x]),a(A.id)}catch(z){console.error("Error creating thread:",z)}},[t]),j=F.useCallback(async()=>{try{const m=await fetch("/api/agents");if(!m.ok){console.error("Failed to fetch agents:",await m.text());return}const z=await m.json();p(z.running||[])}catch(m){console.error("Error fetching agents:",m)}},[]);F.useEffect(()=>{j();const m=setInterval(j,5e3);return()=>clearInterval(m)},[j]);const T=F.useCallback(async()=>{if(I.trim())try{const m=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:I.trim()})});if(!m.ok){const A=await m.text();console.error("Failed to launch agent:",A),alert(`Failed to launch agent: ${A}`);return}const z=await m.json();p(A=>[...A,z]),h(""),w(!1)}catch(m){console.error("Error launching agent:",m)}},[I]),U=F.useCallback(async m=>{try{const z=await fetch(`/api/agents/${m}`,{method:"DELETE"});if(!z.ok){console.error("Failed to stop agent:",await z.text());return}p(A=>A.filter(x=>x.instance_id!==m))}catch(z){console.error("Error stopping agent:",z)}},[]),Q=F.useCallback(async m=>{if(o)try{const z=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:m})});if(!z.ok){console.error("Failed to update workspace:",await z.text());return}const A=await z.json();l(x=>x.map(X=>X.id===o?A:X))}catch(z){console.error("Error updating workspace:",z)}},[o]),H=F.useCallback(async m=>{try{const z=await fetch(`/api/threads/${m}`,{method:"DELETE"});if(!z.ok){console.error("Failed to delete thread:",await z.text());return}l(A=>A.filter(x=>x.id!==m)),c(A=>{const x=new Map(A);return x.delete(m),x}),f(A=>{const x=new Map(A);return x.delete(m),x}),o===m&&a(null)}catch(z){console.error("Error deleting thread:",z)}},[o]),q=F.useCallback(async(m,z)=>{try{const A=await fetch(`/api/threads/${m}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:z})});if(!A.ok){console.error("Failed to rename thread:",await A.text());return}const x=await A.json();l(X=>X.map(fe=>fe.id===m?x:fe))}catch(A){console.error("Error renaming thread:",A)}},[]),ie=F.useCallback(async(m,z)=>{try{const A=await fetch(`/api/approvals/${m}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:z})});if(!A.ok){const x=await A.text();console.error("Failed to approve request:",x),alert(`Failed to approve: ${x}`);return}console.log("Approval approved successfully")}catch(A){console.error("Error approving request:",A)}},[]),C=F.useCallback(async(m,z)=>{try{const A=await fetch(`/api/approvals/${m}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:z})});if(!A.ok){const x=await A.text();console.error("Failed to reject request:",x),alert(`Failed to reject: ${x}`);return}console.log("Approval rejected successfully")}catch(A){console.error("Error rejecting request:",A)}},[]),B=o?s.get(o)||[]:[];return u.jsxs("div",{className:"message-center",children:[u.jsxs("div",{className:"status-bar",children:[u.jsxs("div",{className:`status-indicator ${v?"connected":"disconnected"}`,children:[u.jsx(dk,{connected:v}),u.jsx("span",{children:v?"Connected":"Disconnected"})]}),u.jsxs("div",{className:"status-meta",children:[u.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),u.jsxs("span",{className:"agent-count",children:[g.length," agents"]}),u.jsx("button",{className:"launch-agent-btn",onClick:()=>w(!0),children:"+ Agent"})]})]}),g.length>0&&u.jsx("div",{className:"agents-bar",children:g.map(m=>u.jsxs("div",{className:"agent-chip",children:[u.jsx("span",{className:"agent-pulse"}),u.jsx("span",{className:"agent-name",children:m.instance_id}),u.jsxs("span",{className:"agent-pid",children:["PID ",m.pid]}),u.jsx("button",{className:"agent-stop-btn",onClick:()=>U(m.instance_id),title:"Stop agent",children:"×"})]},m.instance_id))}),k&&u.jsx("div",{className:"modal-overlay",onClick:()=>w(!1),children:u.jsxs("div",{className:"modal-content",onClick:m=>m.stopPropagation(),children:[u.jsx("h3",{children:"Launch New Agent"}),u.jsx("input",{type:"text",value:I,onChange:m=>h(m.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:m=>{m.key==="Enter"&&T(),m.key==="Escape"&&w(!1)}}),u.jsxs("div",{className:"modal-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>w(!1),children:"Cancel"}),u.jsx("button",{className:"launch-btn",onClick:T,children:"Launch"})]})]})}),u.jsxs("div",{className:"center-layout",children:[u.jsx("aside",{className:"threads-panel",children:u.jsx(gv,{threads:i,selectedThreadId:o,onSelectThread:E,onCreateThread:P,onDeleteThread:H,onRenameThread:q,unreadCounts:d})}),u.jsx("main",{className:"conversation-panel",children:o?u.jsx(ak,{thread:i.find(m=>m.id===o),messages:B,onSendMessage:L,onWorkspaceChange:Q,onApproveRequest:ie,onRejectRequest:C}):u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("h3",{children:"Select a conversation"}),u.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),u.jsx("style",{children:`
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
      `})]})},Re={check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),dollar:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:u.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),u.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),u.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},pk=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=F.useState(!0),[a,s]=F.useState(null),[c,d]=F.useState(new Map),f=h=>{try{return JSON.parse(h)}catch{return null}},g=h=>new Date(h).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),p=h=>{const v=c.get(h)||"";n(h,v),d(new Map(c.set(h,"")))},k=h=>{const v=c.get(h)||"";if(!v.trim()){alert("Please provide a reason for rejection");return}r(h,v),d(new Map(c.set(h,"")))},w=(h,v)=>{d(new Map(c.set(h,v)))},I=e.filter(h=>h.status==="pending");return u.jsxs("div",{className:"approval-queue",children:[u.jsx("div",{className:"queue-header",children:u.jsxs("div",{className:"header-title",children:[u.jsx("h2",{children:"Approval Queue"}),u.jsxs("span",{className:"pending-count",children:[I.length," pending"]})]})}),u.jsxs("div",{className:"approvals-container",children:[I.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Re.sparkles}),u.jsx("h3",{children:"All caught up!"}),u.jsx("p",{children:"No pending approvals to review"})]}):u.jsx("div",{className:"approvals-list",children:I.map(h=>{const v=f(h.effect_delta_json),y=a===h.id;return u.jsxs("div",{className:`approval-card impact-${h.impact}`,children:[u.jsxs("div",{className:"card-header",onClick:()=>s(y?null:h.id),children:[u.jsxs("div",{className:"header-left",children:[u.jsx("div",{className:`impact-indicator ${h.impact}`}),u.jsxs("div",{className:"proposal-info",children:[u.jsx("span",{className:"proposal-text",children:h.proposal}),u.jsxs("div",{className:"proposal-meta",children:[h.thread_title&&u.jsxs("span",{className:"meta-item thread-link",onClick:b=>{b.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Re.message,h.thread_title]}),u.jsxs("span",{className:"meta-item",children:[Re.bot,h.instance_id]}),u.jsxs("span",{className:"meta-item",children:[Re.clock,g(h.created_at)]})]})]})]}),u.jsxs("div",{className:"header-right",children:[u.jsxs("span",{className:"cost-badge",children:[Re.dollar,"$",h.estimated_cost.toFixed(2)]}),u.jsx("span",{className:`impact-badge ${h.impact}`,children:h.impact}),u.jsx("button",{className:"expand-btn",children:y?Re.chevronUp:Re.chevronDown})]})]}),y&&u.jsxs("div",{className:"card-details",children:[v&&u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Effect Details"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Capability"}),u.jsx("span",{className:"detail-value code",children:v.cap_type})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Budget Delta"}),u.jsxs("span",{className:"detail-value",children:["$",v.budget_delta.toFixed(2)]})]}),v.paths.length>0&&u.jsxs("div",{className:"detail-item full-width",children:[u.jsx("span",{className:"detail-label",children:"Paths"}),u.jsx("div",{className:"paths-list",children:v.paths.map((b,_)=>u.jsxs("span",{className:"path-tag",children:[Re.folder,b]},_))})]})]})]}),u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Request Info"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Thread"}),u.jsx("span",{className:"detail-value code",children:h.thread_id})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Impact Level"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]})]})]}),u.jsxs("div",{className:"review-section",children:[u.jsx("h4",{children:"Review Notes"}),u.jsx("textarea",{value:c.get(h.id)||"",onChange:b=>w(h.id,b.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),u.jsxs("div",{className:"action-buttons",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>k(h.id),children:[Re.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>p(h.id),children:[Re.check,"Approve"]})]})]})]})]},h.id)})}),t.length>0&&u.jsxs("div",{className:"history-section",children:[u.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[u.jsxs("h3",{children:[l?Re.chevronDown:Re.chevronUp,"Review History"]}),u.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&u.jsx("div",{className:"history-list",children:t.map(h=>{const v=a===`history-${h.id}`;return u.jsxs("div",{className:`history-card ${h.status}`,onClick:()=>s(v?null:`history-${h.id}`),children:[u.jsxs("div",{className:"history-card-header",children:[u.jsxs("div",{className:"history-status",children:[u.jsx("span",{className:`status-icon ${h.status}`,children:h.status==="approved"?Re.check:Re.x}),u.jsxs("div",{className:"history-info",children:[u.jsx("span",{className:"history-proposal",children:h.proposal}),h.thread_title&&u.jsxs("span",{className:"history-thread",onClick:y=>{y.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Re.message,h.thread_title]})]})]}),u.jsxs("div",{className:"history-meta",children:[u.jsx("span",{className:"history-agent",children:h.instance_id}),u.jsx("span",{className:`history-badge ${h.status}`,children:h.status}),u.jsx("span",{className:"history-time",children:h.reviewed_at?g(h.reviewed_at):g(h.created_at)})]})]}),v&&u.jsxs("div",{className:"history-details",children:[u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Reviewed by"}),u.jsx("span",{className:"detail-value",children:h.reviewed_by||"Unknown"})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Cost"}),u.jsxs("span",{className:"detail-value",children:["$",h.estimated_cost.toFixed(2)]})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Impact"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]}),h.review_notes&&u.jsxs("div",{className:"detail-row full-width",children:[u.jsx("span",{className:"detail-label",children:"Notes"}),u.jsx("span",{className:"detail-value notes",children:h.review_notes})]})]})]},`history-${h.id}`)})})]})]}),u.jsx("style",{children:`
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
      `})]})},hk="_indicator_1ctaf_1",mk="_dot_1ctaf_12",gk="_connected_1ctaf_19",vk="_connecting_1ctaf_28",yk="_disconnected_1ctaf_37",xk="_pulsing_1ctaf_46",kk="_text_1ctaf_61",Mt={indicator:hk,dot:mk,connected:gk,connecting:vk,disconnected:yk,pulsing:xk,text:kk};function wk(){const[e,t]=F.useState("disconnected"),[n,r]=F.useState(0);if(F.useEffect(()=>uk((o,a)=>{t(o),r(a)}),[]),e==="connected")return u.jsx("div",{className:`${Mt.indicator} ${Mt.connected}`,title:"Connected",children:u.jsx("span",{className:Mt.dot})});const i=()=>{switch(e){case"connecting":return"Connecting...";case"reconnecting":return`Reconnecting... (${n})`;case"disconnected":return n>0?"Disconnected":"Offline";default:return"Unknown"}},l=()=>{switch(e){case"connecting":case"reconnecting":return Mt.connecting;case"disconnected":return Mt.disconnected;default:return""}};return u.jsxs("div",{className:`${Mt.indicator} ${l()}`,title:i(),children:[u.jsx("span",{className:`${Mt.dot} ${e==="connecting"||e==="reconnecting"?Mt.pulsing:""}`}),u.jsx("span",{className:Mt.text,children:i()})]})}const Sk=u.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M12 6v12M6 12h12"}),u.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]}),bk=()=>{const[e,t]=F.useState({type:"overview"}),[n,r]=F.useState(null),[i,l]=F.useState([]),[o,a]=F.useState([]),[s,c]=F.useState(!1),[d,f]=F.useState(""),[g,p]=F.useState("..."),w=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;F.useEffect(()=>{(async()=>{try{const L=await fetch("/api/version");if(L.ok){const D=await L.json();p(D.version||"dev")}}catch(L){console.error("Error fetching version:",L),p("dev")}})()},[]),F.useEffect(()=>{const E=async()=>{try{const D=await fetch("/api/hierarchy");if(D.ok){const P=await D.json();r(P)}}catch(D){console.error("Error fetching hierarchy:",D)}};E();const L=setInterval(E,5e3);return()=>clearInterval(L)},[]),F.useEffect(()=>{const E=async()=>{try{const D=await fetch("/api/approvals?status=pending");if(D.ok){const U=await D.json();l(U)}const[P,j]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),T=[];if(P.ok){const U=await P.json();T.push(...U)}if(j.ok){const U=await j.json();T.push(...U)}T.sort((U,Q)=>{const H=U.reviewed_at?new Date(U.reviewed_at).getTime():0;return(Q.reviewed_at?new Date(Q.reviewed_at).getTime():0)-H}),a(T)}catch(D){console.error("Error fetching approvals:",D)}};E();const L=setInterval(E,5e3);return()=>clearInterval(L)},[]);const I=async(E,L)=>{try{const D=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:L})});if(!D.ok){console.error("Failed to approve:",await D.text());return}const P=i.find(j=>j.id===E);if(P){const j={...P,status:"approved",reviewed_by:"user",review_notes:L,reviewed_at:Date.now()};a(T=>[j,...T])}l(j=>j.filter(T=>T.id!==E))}catch(D){console.error("Error approving:",D)}},h=async(E,L)=>{try{const D=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:L})});if(!D.ok){console.error("Failed to reject:",await D.text());return}const P=i.find(j=>j.id===E);if(P){const j={...P,status:"rejected",reviewed_by:"user",review_notes:L,reviewed_at:Date.now()};a(T=>[j,...T])}l(j=>j.filter(T=>T.id!==E))}catch(D){console.error("Error rejecting:",D)}},v=()=>{var L,D;const E=[{label:"All Agents",onClick:()=>t({type:"overview"})}];if(e.type==="agent"&&e.agentId&&E.push({label:e.agentId}),e.type==="thread"&&e.threadId){e.agentId&&E.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})});const P=(L=n==null?void 0:n.root.children)==null?void 0:L.find(T=>T.id===e.agentId),j=(D=P==null?void 0:P.children)==null?void 0:D.find(T=>T.id===e.threadId);E.push({label:(j==null?void 0:j.label)||"Thread"})}return E},y=E=>{var D;const L=(D=n==null?void 0:n.root.children)==null?void 0:D.find(P=>{var j;return(j=P.children)==null?void 0:j.some(T=>T.id===E)});t({type:"thread",agentId:L==null?void 0:L.id,threadId:E})},b=async E=>{if(d.trim())try{const L=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:d.trim(),created_by_type:"human",created_by_id:"user",target_agent:E})});if(!L.ok){console.error("Failed to create thread:",await L.text());return}const D=await L.json();f(""),c(!1),t({type:"thread",agentId:E,threadId:D.id})}catch(L){console.error("Error creating thread:",L)}},_=()=>{var E,L,D;if(e.type==="overview"&&n)return u.jsx(hv,{aggregate:n.aggregate,agents:n.root.children||[],onSelectAgent:P=>t({type:"agent",agentId:P})});if(e.type==="agent"&&e.agentId){const P=(E=n==null?void 0:n.root.children)==null?void 0:E.find(T=>T.id===e.agentId),j=i.filter(T=>{var U;return(U=P==null?void 0:P.children)==null?void 0:U.some(Q=>Q.id===T.thread_id)});return u.jsxs("div",{className:"agent-view",children:[u.jsxs("div",{className:"agent-view-header",children:[u.jsx("h2",{children:e.agentId}),u.jsxs("span",{className:"agent-thread-count",children:[((L=P==null?void 0:P.children)==null?void 0:L.length)||0," threads"]})]}),u.jsxs("div",{className:"agent-metrics-section",children:[u.jsx("h3",{children:"Agent Metrics"}),u.jsx(Ca,{scopeType:"agent",scopeId:e.agentId,title:""}),u.jsxs("div",{className:"agent-trends-grid",children:[u.jsx(Sl,{scopeType:"agent",scopeId:e.agentId,period:"hour",limit:24,metric:"cost",title:"Cost (24h)"}),u.jsx(Sl,{scopeType:"agent",scopeId:e.agentId,period:"hour",limit:24,metric:"tokens",title:"Tokens (24h)"})]})]}),u.jsxs("div",{className:"agent-view-content",children:[u.jsxs("div",{className:"agent-threads",children:[u.jsxs("div",{className:"threads-header",children:[u.jsx("h3",{children:"Threads"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>c(!0),title:"New thread",children:"+ New Thread"})]}),s&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:d,onChange:T=>f(T.target.value),onKeyDown:T=>{T.key==="Enter"&&b(e.agentId),T.key==="Escape"&&(c(!1),f(""))},placeholder:"Thread title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{onClick:()=>{c(!1),f("")},children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:()=>b(e.agentId),children:"Create"})]})]}),(D=P==null?void 0:P.children)==null?void 0:D.map(T=>u.jsxs("div",{className:"thread-card",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:T.id}),children:[u.jsx("span",{className:"thread-title",children:T.label}),T.badges&&T.badges.length>0&&u.jsx("span",{className:"thread-badges",children:T.badges.map((U,Q)=>u.jsx("span",{className:`badge badge-${U.type}`,children:U.count},Q))})]},T.id)),(!(P!=null&&P.children)||P.children.length===0)&&!s&&u.jsxs("div",{className:"no-threads",children:["No threads yet",u.jsx("button",{className:"start-thread-btn",onClick:()=>c(!0),children:"Start a conversation"})]})]}),j.length>0&&u.jsxs("div",{className:"agent-approvals",children:[u.jsx("h3",{children:"Pending Approvals"}),u.jsx(pk,{approvals:j,history:[],onApprove:I,onReject:h,onNavigateToThread:y})]})]})]})}return e.type==="thread"&&e.threadId?u.jsxs("div",{className:"thread-view",children:[u.jsx("div",{className:"thread-metrics-bar",children:u.jsx(Ca,{scopeType:"thread",scopeId:e.threadId,title:"Thread Metrics",compact:!0})}),u.jsx("div",{className:"thread-messages-container",children:u.jsx(fk,{websocketUrl:w,instanceId:e.agentId||"default",initialThreadId:e.threadId,onThreadNavigated:()=>{}})})]}):u.jsx("div",{className:"empty-state",children:u.jsx("p",{children:"Select an agent or thread from the sidebar"})})},S=(i==null?void 0:i.filter(E=>E.status==="pending").length)||0;return u.jsxs("div",{className:"app",children:[u.jsxs("header",{className:"app-header",children:[u.jsxs("div",{className:"header-brand",children:[u.jsx("div",{className:"brand-logo",children:Sk}),u.jsxs("div",{className:"brand-text",children:[u.jsx("h1",{children:"AILANG"}),u.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),u.jsxs("div",{className:"header-meta",children:[u.jsx(wk,{}),S>0&&u.jsxs("span",{className:"pending-badge",title:`${S} pending approvals`,children:[S," pending"]}),u.jsx("span",{className:"version-tag",children:g})]})]}),u.jsxs("div",{className:"app-body",children:[u.jsx("aside",{className:"app-sidebar",children:u.jsx(Ig,{selection:e,onSelect:t})}),u.jsxs("main",{className:"app-main",children:[e.type!=="overview"&&u.jsx(mv,{items:v()}),u.jsx("div",{className:"main-content",children:_()})]})]}),u.jsx("style",{children:`
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
          background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));
          border-radius: var(--radius-md);
          color: var(--text-inverse);
        }

        .brand-text h1 {
          font-size: var(--text-base);
          font-weight: var(--font-bold);
          letter-spacing: -0.02em;
          color: var(--text-primary);
          line-height: 1;
          margin-bottom: 2px;
        }

        .brand-subtitle {
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

        .pending-badge {
          padding: var(--space-1) var(--space-2);
          background: rgba(245, 158, 11, 0.15);
          color: #f59e0b;
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          border-radius: var(--radius-full);
        }

        .version-tag {
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border-radius: var(--radius-sm);
          border: 1px solid var(--border-subtle);
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
          background: var(--color-primary);
          color: white;
          border: none;
          border-radius: 6px;
          font-size: 13px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .new-thread-btn:hover {
          background: var(--color-primary-dark);
        }

        .start-thread-btn {
          padding: 8px 16px;
          background: var(--color-primary);
          color: white;
          border: none;
          border-radius: 6px;
          font-size: 13px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .start-thread-btn:hover {
          background: var(--color-primary-dark);
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
          background: var(--color-primary);
          border: none;
          color: white;
        }

        .form-actions .create-btn:hover {
          background: var(--color-primary-dark);
        }

        .empty-state {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100%;
          color: #6c7086;
          font-size: 14px;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .brand-text {
            display: none;
          }

          .app-sidebar {
            width: 60px;
          }
        }
      `})]})};Lo.createRoot(document.getElementById("root")).render(u.jsx(Xt.StrictMode,{children:u.jsx(bk,{})}));
