var oh=Object.defineProperty;var ah=(e,t,n)=>t in e?oh(e,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):e[t]=n;var Ve=(e,t,n)=>ah(e,typeof t!="symbol"?t+"":t,n);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Zi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Oa(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var rd={exports:{}},Nl={},id={exports:{}},J={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ci=Symbol.for("react.element"),sh=Symbol.for("react.portal"),uh=Symbol.for("react.fragment"),ch=Symbol.for("react.strict_mode"),dh=Symbol.for("react.profiler"),fh=Symbol.for("react.provider"),ph=Symbol.for("react.context"),hh=Symbol.for("react.forward_ref"),mh=Symbol.for("react.suspense"),gh=Symbol.for("react.memo"),vh=Symbol.for("react.lazy"),eu=Symbol.iterator;function xh(e){return e===null||typeof e!="object"?null:(e=eu&&e[eu]||e["@@iterator"],typeof e=="function"?e:null)}var ld={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},od=Object.assign,ad={};function pr(e,t,n){this.props=e,this.context=t,this.refs=ad,this.updater=n||ld}pr.prototype.isReactComponent={};pr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};pr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function sd(){}sd.prototype=pr.prototype;function Ba(e,t,n){this.props=e,this.context=t,this.refs=ad,this.updater=n||ld}var $a=Ba.prototype=new sd;$a.constructor=Ba;od($a,pr.prototype);$a.isPureReactComponent=!0;var tu=Array.isArray,ud=Object.prototype.hasOwnProperty,Ha={current:null},cd={key:!0,ref:!0,__self:!0,__source:!0};function dd(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)ud.call(t,r)&&!cd.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var u=Array(a),c=0;c<a;c++)u[c]=arguments[c+2];i.children=u}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:ci,type:e,key:l,ref:o,props:i,_owner:Ha.current}}function yh(e,t){return{$$typeof:ci,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function Ua(e){return typeof e=="object"&&e!==null&&e.$$typeof===ci}function kh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var nu=/\/+/g;function ql(e,t){return typeof e=="object"&&e!==null&&e.key!=null?kh(""+e.key):t.toString(36)}function Oi(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case ci:case sh:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+ql(o,0):r,tu(i)?(n="",e!=null&&(n=e.replace(nu,"$&/")+"/"),Oi(i,t,n,"",function(c){return c})):i!=null&&(Ua(i)&&(i=yh(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(nu,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",tu(e))for(var a=0;a<e.length;a++){l=e[a];var u=r+ql(l,a);o+=Oi(l,t,n,u,i)}else if(u=xh(e),typeof u=="function")for(e=u.call(e),a=0;!(l=e.next()).done;)l=l.value,u=r+ql(l,a++),o+=Oi(l,t,n,u,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function vi(e,t,n){if(e==null)return e;var r=[],i=0;return Oi(e,r,"","",function(l){return t.call(n,l,i++)}),r}function wh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var $e={current:null},Bi={transition:null},Sh={ReactCurrentDispatcher:$e,ReactCurrentBatchConfig:Bi,ReactCurrentOwner:Ha};function fd(){throw Error("act(...) is not supported in production builds of React.")}J.Children={map:vi,forEach:function(e,t,n){vi(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return vi(e,function(){t++}),t},toArray:function(e){return vi(e,function(t){return t})||[]},only:function(e){if(!Ua(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};J.Component=pr;J.Fragment=uh;J.Profiler=dh;J.PureComponent=Ba;J.StrictMode=ch;J.Suspense=mh;J.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=Sh;J.act=fd;J.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=od({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Ha.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(u in t)ud.call(t,u)&&!cd.hasOwnProperty(u)&&(r[u]=t[u]===void 0&&a!==void 0?a[u]:t[u])}var u=arguments.length-2;if(u===1)r.children=n;else if(1<u){a=Array(u);for(var c=0;c<u;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:ci,type:e.type,key:i,ref:l,props:r,_owner:o}};J.createContext=function(e){return e={$$typeof:ph,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:fh,_context:e},e.Consumer=e};J.createElement=dd;J.createFactory=function(e){var t=dd.bind(null,e);return t.type=e,t};J.createRef=function(){return{current:null}};J.forwardRef=function(e){return{$$typeof:hh,render:e}};J.isValidElement=Ua;J.lazy=function(e){return{$$typeof:vh,_payload:{_status:-1,_result:e},_init:wh}};J.memo=function(e,t){return{$$typeof:gh,type:e,compare:t===void 0?null:t}};J.startTransition=function(e){var t=Bi.transition;Bi.transition={};try{e()}finally{Bi.transition=t}};J.unstable_act=fd;J.useCallback=function(e,t){return $e.current.useCallback(e,t)};J.useContext=function(e){return $e.current.useContext(e)};J.useDebugValue=function(){};J.useDeferredValue=function(e){return $e.current.useDeferredValue(e)};J.useEffect=function(e,t){return $e.current.useEffect(e,t)};J.useId=function(){return $e.current.useId()};J.useImperativeHandle=function(e,t,n){return $e.current.useImperativeHandle(e,t,n)};J.useInsertionEffect=function(e,t){return $e.current.useInsertionEffect(e,t)};J.useLayoutEffect=function(e,t){return $e.current.useLayoutEffect(e,t)};J.useMemo=function(e,t){return $e.current.useMemo(e,t)};J.useReducer=function(e,t,n){return $e.current.useReducer(e,t,n)};J.useRef=function(e){return $e.current.useRef(e)};J.useState=function(e){return $e.current.useState(e)};J.useSyncExternalStore=function(e,t,n){return $e.current.useSyncExternalStore(e,t,n)};J.useTransition=function(){return $e.current.useTransition()};J.version="18.3.1";id.exports=J;var F=id.exports;const Xt=Oa(F);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var bh=F,_h=Symbol.for("react.element"),jh=Symbol.for("react.fragment"),Ch=Object.prototype.hasOwnProperty,Nh=bh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,Eh={key:!0,ref:!0,__self:!0,__source:!0};function pd(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)Ch.call(t,r)&&!Eh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:_h,type:e,key:l,ref:o,props:i,_owner:Nh.current}}Nl.Fragment=jh;Nl.jsx=pd;Nl.jsxs=pd;rd.exports=Nl;var s=rd.exports,Mo={},hd={exports:{}},at={},md={exports:{}},gd={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(_,B){var m=_.length;_.push(B);e:for(;0<m;){var L=m-1>>>1,M=_[L];if(0<i(M,B))_[L]=B,_[m]=M,m=L;else break e}}function n(_){return _.length===0?null:_[0]}function r(_){if(_.length===0)return null;var B=_[0],m=_.pop();if(m!==B){_[0]=m;e:for(var L=0,M=_.length,y=M>>>1;L<y;){var X=2*(L+1)-1,pe=_[X],Z=X+1,xe=_[Z];if(0>i(pe,m))Z<M&&0>i(xe,pe)?(_[L]=xe,_[Z]=m,L=Z):(_[L]=pe,_[X]=m,L=X);else if(Z<M&&0>i(xe,m))_[L]=xe,_[Z]=m,L=Z;else break e}}return B}function i(_,B){var m=_.sortIndex-B.sortIndex;return m!==0?m:_.id-B.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var u=[],c=[],d=1,f=null,g=3,p=!1,k=!1,w=!1,z=typeof setTimeout=="function"?setTimeout:null,h=typeof clearTimeout=="function"?clearTimeout:null,v=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function x(_){for(var B=n(c);B!==null;){if(B.callback===null)r(c);else if(B.startTime<=_)r(c),B.sortIndex=B.expirationTime,t(u,B);else break;B=n(c)}}function b(_){if(w=!1,x(_),!k)if(n(u)!==null)k=!0,K(N);else{var B=n(c);B!==null&&le(b,B.startTime-_)}}function N(_,B){k=!1,w&&(w=!1,h(I),I=-1),p=!0;var m=g;try{for(x(B),f=n(u);f!==null&&(!(f.expirationTime>B)||_&&!j());){var L=f.callback;if(typeof L=="function"){f.callback=null,g=f.priorityLevel;var M=L(f.expirationTime<=B);B=e.unstable_now(),typeof M=="function"?f.callback=M:f===n(u)&&r(u),x(B)}else r(u);f=n(u)}if(f!==null)var y=!0;else{var X=n(c);X!==null&&le(b,X.startTime-B),y=!1}return y}finally{f=null,g=m,p=!1}}var S=!1,C=null,I=-1,R=5,P=-1;function j(){return!(e.unstable_now()-P<R)}function E(){if(C!==null){var _=e.unstable_now();P=_;var B=!0;try{B=C(!0,_)}finally{B?U():(S=!1,C=null)}}else S=!1}var U;if(typeof v=="function")U=function(){v(E)};else if(typeof MessageChannel<"u"){var V=new MessageChannel,W=V.port2;V.port1.onmessage=E,U=function(){W.postMessage(null)}}else U=function(){z(E,0)};function K(_){C=_,S||(S=!0,U())}function le(_,B){I=z(function(){_(e.unstable_now())},B)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(_){_.callback=null},e.unstable_continueExecution=function(){k||p||(k=!0,K(N))},e.unstable_forceFrameRate=function(_){0>_||125<_?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):R=0<_?Math.floor(1e3/_):5},e.unstable_getCurrentPriorityLevel=function(){return g},e.unstable_getFirstCallbackNode=function(){return n(u)},e.unstable_next=function(_){switch(g){case 1:case 2:case 3:var B=3;break;default:B=g}var m=g;g=B;try{return _()}finally{g=m}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(_,B){switch(_){case 1:case 2:case 3:case 4:case 5:break;default:_=3}var m=g;g=_;try{return B()}finally{g=m}},e.unstable_scheduleCallback=function(_,B,m){var L=e.unstable_now();switch(typeof m=="object"&&m!==null?(m=m.delay,m=typeof m=="number"&&0<m?L+m:L):m=L,_){case 1:var M=-1;break;case 2:M=250;break;case 5:M=1073741823;break;case 4:M=1e4;break;default:M=5e3}return M=m+M,_={id:d++,callback:B,priorityLevel:_,startTime:m,expirationTime:M,sortIndex:-1},m>L?(_.sortIndex=m,t(c,_),n(u)===null&&_===n(c)&&(w?(h(I),I=-1):w=!0,le(b,m-L))):(_.sortIndex=M,t(u,_),k||p||(k=!0,K(N))),_},e.unstable_shouldYield=j,e.unstable_wrapCallback=function(_){var B=g;return function(){var m=g;g=B;try{return _.apply(this,arguments)}finally{g=m}}}})(gd);md.exports=gd;var Th=md.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var Lh=F,ot=Th;function A(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var vd=new Set,Qr={};function Pn(e,t){or(e,t),or(e+"Capture",t)}function or(e,t){for(Qr[e]=t,e=0;e<t.length;e++)vd.add(t[e])}var Ht=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),Ao=Object.prototype.hasOwnProperty,Ih=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,ru={},iu={};function zh(e){return Ao.call(iu,e)?!0:Ao.call(ru,e)?!1:Ih.test(e)?iu[e]=!0:(ru[e]=!0,!1)}function Ph(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function Mh(e,t,n,r){if(t===null||typeof t>"u"||Ph(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function He(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ie={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ie[e]=new He(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ie[t]=new He(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ie[e]=new He(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ie[e]=new He(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ie[e]=new He(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ie[e]=new He(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ie[e]=new He(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ie[e]=new He(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ie[e]=new He(e,5,!1,e.toLowerCase(),null,!1,!1)});var Va=/[\-:]([a-z])/g;function Wa(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(Va,Wa);Ie[t]=new He(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(Va,Wa);Ie[t]=new He(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(Va,Wa);Ie[t]=new He(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ie[e]=new He(e,1,!1,e.toLowerCase(),null,!1,!1)});Ie.xlinkHref=new He("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ie[e]=new He(e,1,!1,e.toLowerCase(),null,!0,!0)});function Qa(e,t,n,r){var i=Ie.hasOwnProperty(t)?Ie[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(Mh(t,n,i,r)&&(n=null),r||i===null?zh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Qt=Lh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,xi=Symbol.for("react.element"),Bn=Symbol.for("react.portal"),$n=Symbol.for("react.fragment"),qa=Symbol.for("react.strict_mode"),Ro=Symbol.for("react.profiler"),xd=Symbol.for("react.provider"),yd=Symbol.for("react.context"),Ka=Symbol.for("react.forward_ref"),Do=Symbol.for("react.suspense"),Fo=Symbol.for("react.suspense_list"),Ya=Symbol.for("react.memo"),Jt=Symbol.for("react.lazy"),kd=Symbol.for("react.offscreen"),lu=Symbol.iterator;function kr(e){return e===null||typeof e!="object"?null:(e=lu&&e[lu]||e["@@iterator"],typeof e=="function"?e:null)}var ge=Object.assign,Kl;function Lr(e){if(Kl===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Kl=t&&t[1]||""}return`
`+Kl+e}var Yl=!1;function Gl(e,t){if(!e||Yl)return"";Yl=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var u=`
`+i[o].replace(" at new "," at ");return e.displayName&&u.includes("<anonymous>")&&(u=u.replace("<anonymous>",e.displayName)),u}while(1<=o&&0<=a);break}}}finally{Yl=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?Lr(e):""}function Ah(e){switch(e.tag){case 5:return Lr(e.type);case 16:return Lr("Lazy");case 13:return Lr("Suspense");case 19:return Lr("SuspenseList");case 0:case 2:case 15:return e=Gl(e.type,!1),e;case 11:return e=Gl(e.type.render,!1),e;case 1:return e=Gl(e.type,!0),e;default:return""}}function Oo(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case $n:return"Fragment";case Bn:return"Portal";case Ro:return"Profiler";case qa:return"StrictMode";case Do:return"Suspense";case Fo:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case yd:return(e.displayName||"Context")+".Consumer";case xd:return(e._context.displayName||"Context")+".Provider";case Ka:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Ya:return t=e.displayName||null,t!==null?t:Oo(e.type)||"Memo";case Jt:t=e._payload,e=e._init;try{return Oo(e(t))}catch{}}return null}function Rh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Oo(t);case 8:return t===qa?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function pn(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function wd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function Dh(e){var t=wd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function yi(e){e._valueTracker||(e._valueTracker=Dh(e))}function Sd(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=wd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function el(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function Bo(e,t){var n=t.checked;return ge({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function ou(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=pn(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function bd(e,t){t=t.checked,t!=null&&Qa(e,"checked",t,!1)}function $o(e,t){bd(e,t);var n=pn(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?Ho(e,t.type,n):t.hasOwnProperty("defaultValue")&&Ho(e,t.type,pn(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function au(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function Ho(e,t,n){(t!=="number"||el(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var Ir=Array.isArray;function Jn(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+pn(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function Uo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(A(91));return ge({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function su(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(A(92));if(Ir(n)){if(1<n.length)throw Error(A(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:pn(n)}}function _d(e,t){var n=pn(t.value),r=pn(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function uu(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function jd(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Vo(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?jd(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var ki,Cd=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(ki=ki||document.createElement("div"),ki.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=ki.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function qr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Mr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},Fh=["Webkit","ms","Moz","O"];Object.keys(Mr).forEach(function(e){Fh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Mr[t]=Mr[e]})});function Nd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Mr.hasOwnProperty(e)&&Mr[e]?(""+t).trim():t+"px"}function Ed(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=Nd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var Oh=ge({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Wo(e,t){if(t){if(Oh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(A(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(A(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(A(61))}if(t.style!=null&&typeof t.style!="object")throw Error(A(62))}}function Qo(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var qo=null;function Ga(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Ko=null,Zn=null,er=null;function cu(e){if(e=pi(e)){if(typeof Ko!="function")throw Error(A(280));var t=e.stateNode;t&&(t=zl(t),Ko(e.stateNode,e.type,t))}}function Td(e){Zn?er?er.push(e):er=[e]:Zn=e}function Ld(){if(Zn){var e=Zn,t=er;if(er=Zn=null,cu(e),t)for(e=0;e<t.length;e++)cu(t[e])}}function Id(e,t){return e(t)}function zd(){}var Xl=!1;function Pd(e,t,n){if(Xl)return e(t,n);Xl=!0;try{return Id(e,t,n)}finally{Xl=!1,(Zn!==null||er!==null)&&(zd(),Ld())}}function Kr(e,t){var n=e.stateNode;if(n===null)return null;var r=zl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(A(231,t,typeof n));return n}var Yo=!1;if(Ht)try{var wr={};Object.defineProperty(wr,"passive",{get:function(){Yo=!0}}),window.addEventListener("test",wr,wr),window.removeEventListener("test",wr,wr)}catch{Yo=!1}function Bh(e,t,n,r,i,l,o,a,u){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Ar=!1,tl=null,nl=!1,Go=null,$h={onError:function(e){Ar=!0,tl=e}};function Hh(e,t,n,r,i,l,o,a,u){Ar=!1,tl=null,Bh.apply($h,arguments)}function Uh(e,t,n,r,i,l,o,a,u){if(Hh.apply(this,arguments),Ar){if(Ar){var c=tl;Ar=!1,tl=null}else throw Error(A(198));nl||(nl=!0,Go=c)}}function Mn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function Md(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function du(e){if(Mn(e)!==e)throw Error(A(188))}function Vh(e){var t=e.alternate;if(!t){if(t=Mn(e),t===null)throw Error(A(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return du(i),e;if(l===r)return du(i),t;l=l.sibling}throw Error(A(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(A(189))}}if(n.alternate!==r)throw Error(A(190))}if(n.tag!==3)throw Error(A(188));return n.stateNode.current===n?e:t}function Ad(e){return e=Vh(e),e!==null?Rd(e):null}function Rd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=Rd(e);if(t!==null)return t;e=e.sibling}return null}var Dd=ot.unstable_scheduleCallback,fu=ot.unstable_cancelCallback,Wh=ot.unstable_shouldYield,Qh=ot.unstable_requestPaint,ye=ot.unstable_now,qh=ot.unstable_getCurrentPriorityLevel,Xa=ot.unstable_ImmediatePriority,Fd=ot.unstable_UserBlockingPriority,rl=ot.unstable_NormalPriority,Kh=ot.unstable_LowPriority,Od=ot.unstable_IdlePriority,El=null,Lt=null;function Yh(e){if(Lt&&typeof Lt.onCommitFiberRoot=="function")try{Lt.onCommitFiberRoot(El,e,void 0,(e.current.flags&128)===128)}catch{}}var wt=Math.clz32?Math.clz32:Jh,Gh=Math.log,Xh=Math.LN2;function Jh(e){return e>>>=0,e===0?32:31-(Gh(e)/Xh|0)|0}var wi=64,Si=4194304;function zr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function il(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=zr(a):(l&=o,l!==0&&(r=zr(l)))}else o=n&~i,o!==0?r=zr(o):l!==0&&(r=zr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-wt(t),i=1<<n,r|=e[n],t&=~i;return r}function Zh(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function em(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-wt(l),a=1<<o,u=i[o];u===-1?(!(a&n)||a&r)&&(i[o]=Zh(a,t)):u<=t&&(e.expiredLanes|=a),l&=~a}}function Xo(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function Bd(){var e=wi;return wi<<=1,!(wi&4194240)&&(wi=64),e}function Jl(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function di(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-wt(t),e[t]=n}function tm(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-wt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Ja(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-wt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var ie=0;function $d(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Hd,Za,Ud,Vd,Wd,Jo=!1,bi=[],ln=null,on=null,an=null,Yr=new Map,Gr=new Map,en=[],nm="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function pu(e,t){switch(e){case"focusin":case"focusout":ln=null;break;case"dragenter":case"dragleave":on=null;break;case"mouseover":case"mouseout":an=null;break;case"pointerover":case"pointerout":Yr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Gr.delete(t.pointerId)}}function Sr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=pi(t),t!==null&&Za(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function rm(e,t,n,r,i){switch(t){case"focusin":return ln=Sr(ln,e,t,n,r,i),!0;case"dragenter":return on=Sr(on,e,t,n,r,i),!0;case"mouseover":return an=Sr(an,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Yr.set(l,Sr(Yr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Gr.set(l,Sr(Gr.get(l)||null,e,t,n,r,i)),!0}return!1}function Qd(e){var t=bn(e.target);if(t!==null){var n=Mn(t);if(n!==null){if(t=n.tag,t===13){if(t=Md(n),t!==null){e.blockedOn=t,Wd(e.priority,function(){Ud(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function $i(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Zo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);qo=r,n.target.dispatchEvent(r),qo=null}else return t=pi(n),t!==null&&Za(t),e.blockedOn=n,!1;t.shift()}return!0}function hu(e,t,n){$i(e)&&n.delete(t)}function im(){Jo=!1,ln!==null&&$i(ln)&&(ln=null),on!==null&&$i(on)&&(on=null),an!==null&&$i(an)&&(an=null),Yr.forEach(hu),Gr.forEach(hu)}function br(e,t){e.blockedOn===t&&(e.blockedOn=null,Jo||(Jo=!0,ot.unstable_scheduleCallback(ot.unstable_NormalPriority,im)))}function Xr(e){function t(i){return br(i,e)}if(0<bi.length){br(bi[0],e);for(var n=1;n<bi.length;n++){var r=bi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(ln!==null&&br(ln,e),on!==null&&br(on,e),an!==null&&br(an,e),Yr.forEach(t),Gr.forEach(t),n=0;n<en.length;n++)r=en[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<en.length&&(n=en[0],n.blockedOn===null);)Qd(n),n.blockedOn===null&&en.shift()}var tr=Qt.ReactCurrentBatchConfig,ll=!0;function lm(e,t,n,r){var i=ie,l=tr.transition;tr.transition=null;try{ie=1,es(e,t,n,r)}finally{ie=i,tr.transition=l}}function om(e,t,n,r){var i=ie,l=tr.transition;tr.transition=null;try{ie=4,es(e,t,n,r)}finally{ie=i,tr.transition=l}}function es(e,t,n,r){if(ll){var i=Zo(e,t,n,r);if(i===null)so(e,t,r,ol,n),pu(e,r);else if(rm(i,e,t,n,r))r.stopPropagation();else if(pu(e,r),t&4&&-1<nm.indexOf(e)){for(;i!==null;){var l=pi(i);if(l!==null&&Hd(l),l=Zo(e,t,n,r),l===null&&so(e,t,r,ol,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else so(e,t,r,null,n)}}var ol=null;function Zo(e,t,n,r){if(ol=null,e=Ga(r),e=bn(e),e!==null)if(t=Mn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=Md(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return ol=e,null}function qd(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(qh()){case Xa:return 1;case Fd:return 4;case rl:case Kh:return 16;case Od:return 536870912;default:return 16}default:return 16}}var nn=null,ts=null,Hi=null;function Kd(){if(Hi)return Hi;var e,t=ts,n=t.length,r,i="value"in nn?nn.value:nn.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Hi=i.slice(e,1<r?1-r:void 0)}function Ui(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function _i(){return!0}function mu(){return!1}function st(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?_i:mu,this.isPropagationStopped=mu,this}return ge(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=_i)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=_i)},persist:function(){},isPersistent:_i}),t}var hr={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},ns=st(hr),fi=ge({},hr,{view:0,detail:0}),am=st(fi),Zl,eo,_r,Tl=ge({},fi,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:rs,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==_r&&(_r&&e.type==="mousemove"?(Zl=e.screenX-_r.screenX,eo=e.screenY-_r.screenY):eo=Zl=0,_r=e),Zl)},movementY:function(e){return"movementY"in e?e.movementY:eo}}),gu=st(Tl),sm=ge({},Tl,{dataTransfer:0}),um=st(sm),cm=ge({},fi,{relatedTarget:0}),to=st(cm),dm=ge({},hr,{animationName:0,elapsedTime:0,pseudoElement:0}),fm=st(dm),pm=ge({},hr,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),hm=st(pm),mm=ge({},hr,{data:0}),vu=st(mm),gm={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},vm={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},xm={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function ym(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=xm[e])?!!t[e]:!1}function rs(){return ym}var km=ge({},fi,{key:function(e){if(e.key){var t=gm[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Ui(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?vm[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:rs,charCode:function(e){return e.type==="keypress"?Ui(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Ui(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),wm=st(km),Sm=ge({},Tl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),xu=st(Sm),bm=ge({},fi,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:rs}),_m=st(bm),jm=ge({},hr,{propertyName:0,elapsedTime:0,pseudoElement:0}),Cm=st(jm),Nm=ge({},Tl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),Em=st(Nm),Tm=[9,13,27,32],is=Ht&&"CompositionEvent"in window,Rr=null;Ht&&"documentMode"in document&&(Rr=document.documentMode);var Lm=Ht&&"TextEvent"in window&&!Rr,Yd=Ht&&(!is||Rr&&8<Rr&&11>=Rr),yu=" ",ku=!1;function Gd(e,t){switch(e){case"keyup":return Tm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Xd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Hn=!1;function Im(e,t){switch(e){case"compositionend":return Xd(t);case"keypress":return t.which!==32?null:(ku=!0,yu);case"textInput":return e=t.data,e===yu&&ku?null:e;default:return null}}function zm(e,t){if(Hn)return e==="compositionend"||!is&&Gd(e,t)?(e=Kd(),Hi=ts=nn=null,Hn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Yd&&t.locale!=="ko"?null:t.data;default:return null}}var Pm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function wu(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!Pm[e.type]:t==="textarea"}function Jd(e,t,n,r){Td(r),t=al(t,"onChange"),0<t.length&&(n=new ns("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Dr=null,Jr=null;function Mm(e){cf(e,0)}function Ll(e){var t=Wn(e);if(Sd(t))return e}function Am(e,t){if(e==="change")return t}var Zd=!1;if(Ht){var no;if(Ht){var ro="oninput"in document;if(!ro){var Su=document.createElement("div");Su.setAttribute("oninput","return;"),ro=typeof Su.oninput=="function"}no=ro}else no=!1;Zd=no&&(!document.documentMode||9<document.documentMode)}function bu(){Dr&&(Dr.detachEvent("onpropertychange",ef),Jr=Dr=null)}function ef(e){if(e.propertyName==="value"&&Ll(Jr)){var t=[];Jd(t,Jr,e,Ga(e)),Pd(Mm,t)}}function Rm(e,t,n){e==="focusin"?(bu(),Dr=t,Jr=n,Dr.attachEvent("onpropertychange",ef)):e==="focusout"&&bu()}function Dm(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return Ll(Jr)}function Fm(e,t){if(e==="click")return Ll(t)}function Om(e,t){if(e==="input"||e==="change")return Ll(t)}function Bm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var bt=typeof Object.is=="function"?Object.is:Bm;function Zr(e,t){if(bt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!Ao.call(t,i)||!bt(e[i],t[i]))return!1}return!0}function _u(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function ju(e,t){var n=_u(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=_u(n)}}function tf(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?tf(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function nf(){for(var e=window,t=el();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=el(e.document)}return t}function ls(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function $m(e){var t=nf(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&tf(n.ownerDocument.documentElement,n)){if(r!==null&&ls(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=ju(n,l);var o=ju(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Hm=Ht&&"documentMode"in document&&11>=document.documentMode,Un=null,ea=null,Fr=null,ta=!1;function Cu(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;ta||Un==null||Un!==el(r)||(r=Un,"selectionStart"in r&&ls(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Fr&&Zr(Fr,r)||(Fr=r,r=al(ea,"onSelect"),0<r.length&&(t=new ns("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Un)))}function ji(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Vn={animationend:ji("Animation","AnimationEnd"),animationiteration:ji("Animation","AnimationIteration"),animationstart:ji("Animation","AnimationStart"),transitionend:ji("Transition","TransitionEnd")},io={},rf={};Ht&&(rf=document.createElement("div").style,"AnimationEvent"in window||(delete Vn.animationend.animation,delete Vn.animationiteration.animation,delete Vn.animationstart.animation),"TransitionEvent"in window||delete Vn.transitionend.transition);function Il(e){if(io[e])return io[e];if(!Vn[e])return e;var t=Vn[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in rf)return io[e]=t[n];return e}var lf=Il("animationend"),of=Il("animationiteration"),af=Il("animationstart"),sf=Il("transitionend"),uf=new Map,Nu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function mn(e,t){uf.set(e,t),Pn(t,[e])}for(var lo=0;lo<Nu.length;lo++){var oo=Nu[lo],Um=oo.toLowerCase(),Vm=oo[0].toUpperCase()+oo.slice(1);mn(Um,"on"+Vm)}mn(lf,"onAnimationEnd");mn(of,"onAnimationIteration");mn(af,"onAnimationStart");mn("dblclick","onDoubleClick");mn("focusin","onFocus");mn("focusout","onBlur");mn(sf,"onTransitionEnd");or("onMouseEnter",["mouseout","mouseover"]);or("onMouseLeave",["mouseout","mouseover"]);or("onPointerEnter",["pointerout","pointerover"]);or("onPointerLeave",["pointerout","pointerover"]);Pn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));Pn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));Pn("onBeforeInput",["compositionend","keypress","textInput","paste"]);Pn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));Pn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));Pn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var Pr="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Wm=new Set("cancel close invalid load scroll toggle".split(" ").concat(Pr));function Eu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,Uh(r,t,void 0,e),e.currentTarget=null}function cf(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],u=a.instance,c=a.currentTarget;if(a=a.listener,u!==l&&i.isPropagationStopped())break e;Eu(i,a,c),l=u}else for(o=0;o<r.length;o++){if(a=r[o],u=a.instance,c=a.currentTarget,a=a.listener,u!==l&&i.isPropagationStopped())break e;Eu(i,a,c),l=u}}}if(nl)throw e=Go,nl=!1,Go=null,e}function ce(e,t){var n=t[oa];n===void 0&&(n=t[oa]=new Set);var r=e+"__bubble";n.has(r)||(df(t,e,2,!1),n.add(r))}function ao(e,t,n){var r=0;t&&(r|=4),df(n,e,r,t)}var Ci="_reactListening"+Math.random().toString(36).slice(2);function ei(e){if(!e[Ci]){e[Ci]=!0,vd.forEach(function(n){n!=="selectionchange"&&(Wm.has(n)||ao(n,!1,e),ao(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[Ci]||(t[Ci]=!0,ao("selectionchange",!1,t))}}function df(e,t,n,r){switch(qd(t)){case 1:var i=lm;break;case 4:i=om;break;default:i=es}n=i.bind(null,t,n,e),i=void 0,!Yo||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function so(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var u=o.tag;if((u===3||u===4)&&(u=o.stateNode.containerInfo,u===i||u.nodeType===8&&u.parentNode===i))return;o=o.return}for(;a!==null;){if(o=bn(a),o===null)return;if(u=o.tag,u===5||u===6){r=l=o;continue e}a=a.parentNode}}r=r.return}Pd(function(){var c=l,d=Ga(n),f=[];e:{var g=uf.get(e);if(g!==void 0){var p=ns,k=e;switch(e){case"keypress":if(Ui(n)===0)break e;case"keydown":case"keyup":p=wm;break;case"focusin":k="focus",p=to;break;case"focusout":k="blur",p=to;break;case"beforeblur":case"afterblur":p=to;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":p=gu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":p=um;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":p=_m;break;case lf:case of:case af:p=fm;break;case sf:p=Cm;break;case"scroll":p=am;break;case"wheel":p=Em;break;case"copy":case"cut":case"paste":p=hm;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":p=xu}var w=(t&4)!==0,z=!w&&e==="scroll",h=w?g!==null?g+"Capture":null:g;w=[];for(var v=c,x;v!==null;){x=v;var b=x.stateNode;if(x.tag===5&&b!==null&&(x=b,h!==null&&(b=Kr(v,h),b!=null&&w.push(ti(v,b,x)))),z)break;v=v.return}0<w.length&&(g=new p(g,k,null,n,d),f.push({event:g,listeners:w}))}}if(!(t&7)){e:{if(g=e==="mouseover"||e==="pointerover",p=e==="mouseout"||e==="pointerout",g&&n!==qo&&(k=n.relatedTarget||n.fromElement)&&(bn(k)||k[Ut]))break e;if((p||g)&&(g=d.window===d?d:(g=d.ownerDocument)?g.defaultView||g.parentWindow:window,p?(k=n.relatedTarget||n.toElement,p=c,k=k?bn(k):null,k!==null&&(z=Mn(k),k!==z||k.tag!==5&&k.tag!==6)&&(k=null)):(p=null,k=c),p!==k)){if(w=gu,b="onMouseLeave",h="onMouseEnter",v="mouse",(e==="pointerout"||e==="pointerover")&&(w=xu,b="onPointerLeave",h="onPointerEnter",v="pointer"),z=p==null?g:Wn(p),x=k==null?g:Wn(k),g=new w(b,v+"leave",p,n,d),g.target=z,g.relatedTarget=x,b=null,bn(d)===c&&(w=new w(h,v+"enter",k,n,d),w.target=x,w.relatedTarget=z,b=w),z=b,p&&k)t:{for(w=p,h=k,v=0,x=w;x;x=Dn(x))v++;for(x=0,b=h;b;b=Dn(b))x++;for(;0<v-x;)w=Dn(w),v--;for(;0<x-v;)h=Dn(h),x--;for(;v--;){if(w===h||h!==null&&w===h.alternate)break t;w=Dn(w),h=Dn(h)}w=null}else w=null;p!==null&&Tu(f,g,p,w,!1),k!==null&&z!==null&&Tu(f,z,k,w,!0)}}e:{if(g=c?Wn(c):window,p=g.nodeName&&g.nodeName.toLowerCase(),p==="select"||p==="input"&&g.type==="file")var N=Am;else if(wu(g))if(Zd)N=Om;else{N=Dm;var S=Rm}else(p=g.nodeName)&&p.toLowerCase()==="input"&&(g.type==="checkbox"||g.type==="radio")&&(N=Fm);if(N&&(N=N(e,c))){Jd(f,N,n,d);break e}S&&S(e,g,c),e==="focusout"&&(S=g._wrapperState)&&S.controlled&&g.type==="number"&&Ho(g,"number",g.value)}switch(S=c?Wn(c):window,e){case"focusin":(wu(S)||S.contentEditable==="true")&&(Un=S,ea=c,Fr=null);break;case"focusout":Fr=ea=Un=null;break;case"mousedown":ta=!0;break;case"contextmenu":case"mouseup":case"dragend":ta=!1,Cu(f,n,d);break;case"selectionchange":if(Hm)break;case"keydown":case"keyup":Cu(f,n,d)}var C;if(is)e:{switch(e){case"compositionstart":var I="onCompositionStart";break e;case"compositionend":I="onCompositionEnd";break e;case"compositionupdate":I="onCompositionUpdate";break e}I=void 0}else Hn?Gd(e,n)&&(I="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(I="onCompositionStart");I&&(Yd&&n.locale!=="ko"&&(Hn||I!=="onCompositionStart"?I==="onCompositionEnd"&&Hn&&(C=Kd()):(nn=d,ts="value"in nn?nn.value:nn.textContent,Hn=!0)),S=al(c,I),0<S.length&&(I=new vu(I,e,null,n,d),f.push({event:I,listeners:S}),C?I.data=C:(C=Xd(n),C!==null&&(I.data=C)))),(C=Lm?Im(e,n):zm(e,n))&&(c=al(c,"onBeforeInput"),0<c.length&&(d=new vu("onBeforeInput","beforeinput",null,n,d),f.push({event:d,listeners:c}),d.data=C))}cf(f,t)})}function ti(e,t,n){return{instance:e,listener:t,currentTarget:n}}function al(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Kr(e,n),l!=null&&r.unshift(ti(e,l,i)),l=Kr(e,t),l!=null&&r.push(ti(e,l,i))),e=e.return}return r}function Dn(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function Tu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,u=a.alternate,c=a.stateNode;if(u!==null&&u===r)break;a.tag===5&&c!==null&&(a=c,i?(u=Kr(n,l),u!=null&&o.unshift(ti(n,u,a))):i||(u=Kr(n,l),u!=null&&o.push(ti(n,u,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Qm=/\r\n?/g,qm=/\u0000|\uFFFD/g;function Lu(e){return(typeof e=="string"?e:""+e).replace(Qm,`
`).replace(qm,"")}function Ni(e,t,n){if(t=Lu(t),Lu(e)!==t&&n)throw Error(A(425))}function sl(){}var na=null,ra=null;function ia(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var la=typeof setTimeout=="function"?setTimeout:void 0,Km=typeof clearTimeout=="function"?clearTimeout:void 0,Iu=typeof Promise=="function"?Promise:void 0,Ym=typeof queueMicrotask=="function"?queueMicrotask:typeof Iu<"u"?function(e){return Iu.resolve(null).then(e).catch(Gm)}:la;function Gm(e){setTimeout(function(){throw e})}function uo(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Xr(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Xr(t)}function sn(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function zu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var mr=Math.random().toString(36).slice(2),Et="__reactFiber$"+mr,ni="__reactProps$"+mr,Ut="__reactContainer$"+mr,oa="__reactEvents$"+mr,Xm="__reactListeners$"+mr,Jm="__reactHandles$"+mr;function bn(e){var t=e[Et];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Ut]||n[Et]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=zu(e);e!==null;){if(n=e[Et])return n;e=zu(e)}return t}e=n,n=e.parentNode}return null}function pi(e){return e=e[Et]||e[Ut],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function Wn(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(A(33))}function zl(e){return e[ni]||null}var aa=[],Qn=-1;function gn(e){return{current:e}}function de(e){0>Qn||(e.current=aa[Qn],aa[Qn]=null,Qn--)}function se(e,t){Qn++,aa[Qn]=e.current,e.current=t}var hn={},Re=gn(hn),Ke=gn(!1),En=hn;function ar(e,t){var n=e.type.contextTypes;if(!n)return hn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Ye(e){return e=e.childContextTypes,e!=null}function ul(){de(Ke),de(Re)}function Pu(e,t,n){if(Re.current!==hn)throw Error(A(168));se(Re,t),se(Ke,n)}function ff(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(A(108,Rh(e)||"Unknown",i));return ge({},n,r)}function cl(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||hn,En=Re.current,se(Re,e),se(Ke,Ke.current),!0}function Mu(e,t,n){var r=e.stateNode;if(!r)throw Error(A(169));n?(e=ff(e,t,En),r.__reactInternalMemoizedMergedChildContext=e,de(Ke),de(Re),se(Re,e)):de(Ke),se(Ke,n)}var Ft=null,Pl=!1,co=!1;function pf(e){Ft===null?Ft=[e]:Ft.push(e)}function Zm(e){Pl=!0,pf(e)}function vn(){if(!co&&Ft!==null){co=!0;var e=0,t=ie;try{var n=Ft;for(ie=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}Ft=null,Pl=!1}catch(i){throw Ft!==null&&(Ft=Ft.slice(e+1)),Dd(Xa,vn),i}finally{ie=t,co=!1}}return null}var qn=[],Kn=0,dl=null,fl=0,ut=[],ct=0,Tn=null,Ot=1,Bt="";function kn(e,t){qn[Kn++]=fl,qn[Kn++]=dl,dl=e,fl=t}function hf(e,t,n){ut[ct++]=Ot,ut[ct++]=Bt,ut[ct++]=Tn,Tn=e;var r=Ot;e=Bt;var i=32-wt(r)-1;r&=~(1<<i),n+=1;var l=32-wt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,Ot=1<<32-wt(t)+i|n<<i|r,Bt=l+e}else Ot=1<<l|n<<i|r,Bt=e}function os(e){e.return!==null&&(kn(e,1),hf(e,1,0))}function as(e){for(;e===dl;)dl=qn[--Kn],qn[Kn]=null,fl=qn[--Kn],qn[Kn]=null;for(;e===Tn;)Tn=ut[--ct],ut[ct]=null,Bt=ut[--ct],ut[ct]=null,Ot=ut[--ct],ut[ct]=null}var lt=null,rt=null,fe=!1,kt=null;function mf(e,t){var n=ft(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function Au(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,lt=e,rt=sn(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,lt=e,rt=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=Tn!==null?{id:Ot,overflow:Bt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=ft(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,lt=e,rt=null,!0):!1;default:return!1}}function sa(e){return(e.mode&1)!==0&&(e.flags&128)===0}function ua(e){if(fe){var t=rt;if(t){var n=t;if(!Au(e,t)){if(sa(e))throw Error(A(418));t=sn(n.nextSibling);var r=lt;t&&Au(e,t)?mf(r,n):(e.flags=e.flags&-4097|2,fe=!1,lt=e)}}else{if(sa(e))throw Error(A(418));e.flags=e.flags&-4097|2,fe=!1,lt=e}}}function Ru(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;lt=e}function Ei(e){if(e!==lt)return!1;if(!fe)return Ru(e),fe=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!ia(e.type,e.memoizedProps)),t&&(t=rt)){if(sa(e))throw gf(),Error(A(418));for(;t;)mf(e,t),t=sn(t.nextSibling)}if(Ru(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(A(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){rt=sn(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}rt=null}}else rt=lt?sn(e.stateNode.nextSibling):null;return!0}function gf(){for(var e=rt;e;)e=sn(e.nextSibling)}function sr(){rt=lt=null,fe=!1}function ss(e){kt===null?kt=[e]:kt.push(e)}var eg=Qt.ReactCurrentBatchConfig;function jr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(A(309));var r=n.stateNode}if(!r)throw Error(A(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(A(284));if(!n._owner)throw Error(A(290,e))}return e}function Ti(e,t){throw e=Object.prototype.toString.call(t),Error(A(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Du(e){var t=e._init;return t(e._payload)}function vf(e){function t(h,v){if(e){var x=h.deletions;x===null?(h.deletions=[v],h.flags|=16):x.push(v)}}function n(h,v){if(!e)return null;for(;v!==null;)t(h,v),v=v.sibling;return null}function r(h,v){for(h=new Map;v!==null;)v.key!==null?h.set(v.key,v):h.set(v.index,v),v=v.sibling;return h}function i(h,v){return h=fn(h,v),h.index=0,h.sibling=null,h}function l(h,v,x){return h.index=x,e?(x=h.alternate,x!==null?(x=x.index,x<v?(h.flags|=2,v):x):(h.flags|=2,v)):(h.flags|=1048576,v)}function o(h){return e&&h.alternate===null&&(h.flags|=2),h}function a(h,v,x,b){return v===null||v.tag!==6?(v=xo(x,h.mode,b),v.return=h,v):(v=i(v,x),v.return=h,v)}function u(h,v,x,b){var N=x.type;return N===$n?d(h,v,x.props.children,b,x.key):v!==null&&(v.elementType===N||typeof N=="object"&&N!==null&&N.$$typeof===Jt&&Du(N)===v.type)?(b=i(v,x.props),b.ref=jr(h,v,x),b.return=h,b):(b=Gi(x.type,x.key,x.props,null,h.mode,b),b.ref=jr(h,v,x),b.return=h,b)}function c(h,v,x,b){return v===null||v.tag!==4||v.stateNode.containerInfo!==x.containerInfo||v.stateNode.implementation!==x.implementation?(v=yo(x,h.mode,b),v.return=h,v):(v=i(v,x.children||[]),v.return=h,v)}function d(h,v,x,b,N){return v===null||v.tag!==7?(v=Nn(x,h.mode,b,N),v.return=h,v):(v=i(v,x),v.return=h,v)}function f(h,v,x){if(typeof v=="string"&&v!==""||typeof v=="number")return v=xo(""+v,h.mode,x),v.return=h,v;if(typeof v=="object"&&v!==null){switch(v.$$typeof){case xi:return x=Gi(v.type,v.key,v.props,null,h.mode,x),x.ref=jr(h,null,v),x.return=h,x;case Bn:return v=yo(v,h.mode,x),v.return=h,v;case Jt:var b=v._init;return f(h,b(v._payload),x)}if(Ir(v)||kr(v))return v=Nn(v,h.mode,x,null),v.return=h,v;Ti(h,v)}return null}function g(h,v,x,b){var N=v!==null?v.key:null;if(typeof x=="string"&&x!==""||typeof x=="number")return N!==null?null:a(h,v,""+x,b);if(typeof x=="object"&&x!==null){switch(x.$$typeof){case xi:return x.key===N?u(h,v,x,b):null;case Bn:return x.key===N?c(h,v,x,b):null;case Jt:return N=x._init,g(h,v,N(x._payload),b)}if(Ir(x)||kr(x))return N!==null?null:d(h,v,x,b,null);Ti(h,x)}return null}function p(h,v,x,b,N){if(typeof b=="string"&&b!==""||typeof b=="number")return h=h.get(x)||null,a(v,h,""+b,N);if(typeof b=="object"&&b!==null){switch(b.$$typeof){case xi:return h=h.get(b.key===null?x:b.key)||null,u(v,h,b,N);case Bn:return h=h.get(b.key===null?x:b.key)||null,c(v,h,b,N);case Jt:var S=b._init;return p(h,v,x,S(b._payload),N)}if(Ir(b)||kr(b))return h=h.get(x)||null,d(v,h,b,N,null);Ti(v,b)}return null}function k(h,v,x,b){for(var N=null,S=null,C=v,I=v=0,R=null;C!==null&&I<x.length;I++){C.index>I?(R=C,C=null):R=C.sibling;var P=g(h,C,x[I],b);if(P===null){C===null&&(C=R);break}e&&C&&P.alternate===null&&t(h,C),v=l(P,v,I),S===null?N=P:S.sibling=P,S=P,C=R}if(I===x.length)return n(h,C),fe&&kn(h,I),N;if(C===null){for(;I<x.length;I++)C=f(h,x[I],b),C!==null&&(v=l(C,v,I),S===null?N=C:S.sibling=C,S=C);return fe&&kn(h,I),N}for(C=r(h,C);I<x.length;I++)R=p(C,h,I,x[I],b),R!==null&&(e&&R.alternate!==null&&C.delete(R.key===null?I:R.key),v=l(R,v,I),S===null?N=R:S.sibling=R,S=R);return e&&C.forEach(function(j){return t(h,j)}),fe&&kn(h,I),N}function w(h,v,x,b){var N=kr(x);if(typeof N!="function")throw Error(A(150));if(x=N.call(x),x==null)throw Error(A(151));for(var S=N=null,C=v,I=v=0,R=null,P=x.next();C!==null&&!P.done;I++,P=x.next()){C.index>I?(R=C,C=null):R=C.sibling;var j=g(h,C,P.value,b);if(j===null){C===null&&(C=R);break}e&&C&&j.alternate===null&&t(h,C),v=l(j,v,I),S===null?N=j:S.sibling=j,S=j,C=R}if(P.done)return n(h,C),fe&&kn(h,I),N;if(C===null){for(;!P.done;I++,P=x.next())P=f(h,P.value,b),P!==null&&(v=l(P,v,I),S===null?N=P:S.sibling=P,S=P);return fe&&kn(h,I),N}for(C=r(h,C);!P.done;I++,P=x.next())P=p(C,h,I,P.value,b),P!==null&&(e&&P.alternate!==null&&C.delete(P.key===null?I:P.key),v=l(P,v,I),S===null?N=P:S.sibling=P,S=P);return e&&C.forEach(function(E){return t(h,E)}),fe&&kn(h,I),N}function z(h,v,x,b){if(typeof x=="object"&&x!==null&&x.type===$n&&x.key===null&&(x=x.props.children),typeof x=="object"&&x!==null){switch(x.$$typeof){case xi:e:{for(var N=x.key,S=v;S!==null;){if(S.key===N){if(N=x.type,N===$n){if(S.tag===7){n(h,S.sibling),v=i(S,x.props.children),v.return=h,h=v;break e}}else if(S.elementType===N||typeof N=="object"&&N!==null&&N.$$typeof===Jt&&Du(N)===S.type){n(h,S.sibling),v=i(S,x.props),v.ref=jr(h,S,x),v.return=h,h=v;break e}n(h,S);break}else t(h,S);S=S.sibling}x.type===$n?(v=Nn(x.props.children,h.mode,b,x.key),v.return=h,h=v):(b=Gi(x.type,x.key,x.props,null,h.mode,b),b.ref=jr(h,v,x),b.return=h,h=b)}return o(h);case Bn:e:{for(S=x.key;v!==null;){if(v.key===S)if(v.tag===4&&v.stateNode.containerInfo===x.containerInfo&&v.stateNode.implementation===x.implementation){n(h,v.sibling),v=i(v,x.children||[]),v.return=h,h=v;break e}else{n(h,v);break}else t(h,v);v=v.sibling}v=yo(x,h.mode,b),v.return=h,h=v}return o(h);case Jt:return S=x._init,z(h,v,S(x._payload),b)}if(Ir(x))return k(h,v,x,b);if(kr(x))return w(h,v,x,b);Ti(h,x)}return typeof x=="string"&&x!==""||typeof x=="number"?(x=""+x,v!==null&&v.tag===6?(n(h,v.sibling),v=i(v,x),v.return=h,h=v):(n(h,v),v=xo(x,h.mode,b),v.return=h,h=v),o(h)):n(h,v)}return z}var ur=vf(!0),xf=vf(!1),pl=gn(null),hl=null,Yn=null,us=null;function cs(){us=Yn=hl=null}function ds(e){var t=pl.current;de(pl),e._currentValue=t}function ca(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function nr(e,t){hl=e,us=Yn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(qe=!0),e.firstContext=null)}function ht(e){var t=e._currentValue;if(us!==e)if(e={context:e,memoizedValue:t,next:null},Yn===null){if(hl===null)throw Error(A(308));Yn=e,hl.dependencies={lanes:0,firstContext:e}}else Yn=Yn.next=e;return t}var _n=null;function fs(e){_n===null?_n=[e]:_n.push(e)}function yf(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,fs(t)):(n.next=i.next,i.next=n),t.interleaved=n,Vt(e,r)}function Vt(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Zt=!1;function ps(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function kf(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function $t(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function un(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,ne&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Vt(e,n)}return i=r.interleaved,i===null?(t.next=t,fs(r)):(t.next=i.next,i.next=t),r.interleaved=t,Vt(e,n)}function Vi(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ja(e,n)}}function Fu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function ml(e,t,n,r){var i=e.updateQueue;Zt=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var u=a,c=u.next;u.next=null,o===null?l=c:o.next=c,o=u;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=u))}if(l!==null){var f=i.baseState;o=0,d=c=u=null,a=l;do{var g=a.lane,p=a.eventTime;if((r&g)===g){d!==null&&(d=d.next={eventTime:p,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var k=e,w=a;switch(g=t,p=n,w.tag){case 1:if(k=w.payload,typeof k=="function"){f=k.call(p,f,g);break e}f=k;break e;case 3:k.flags=k.flags&-65537|128;case 0:if(k=w.payload,g=typeof k=="function"?k.call(p,f,g):k,g==null)break e;f=ge({},f,g);break e;case 2:Zt=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,g=i.effects,g===null?i.effects=[a]:g.push(a))}else p={eventTime:p,lane:g,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=p,u=f):d=d.next=p,o|=g;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;g=a,a=g.next,g.next=null,i.lastBaseUpdate=g,i.shared.pending=null}}while(!0);if(d===null&&(u=f),i.baseState=u,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);In|=o,e.lanes=o,e.memoizedState=f}}function Ou(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(A(191,i));i.call(r)}}}var hi={},It=gn(hi),ri=gn(hi),ii=gn(hi);function jn(e){if(e===hi)throw Error(A(174));return e}function hs(e,t){switch(se(ii,t),se(ri,e),se(It,hi),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Vo(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Vo(t,e)}de(It),se(It,t)}function cr(){de(It),de(ri),de(ii)}function wf(e){jn(ii.current);var t=jn(It.current),n=Vo(t,e.type);t!==n&&(se(ri,e),se(It,n))}function ms(e){ri.current===e&&(de(It),de(ri))}var he=gn(0);function gl(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var fo=[];function gs(){for(var e=0;e<fo.length;e++)fo[e]._workInProgressVersionPrimary=null;fo.length=0}var Wi=Qt.ReactCurrentDispatcher,po=Qt.ReactCurrentBatchConfig,Ln=0,me=null,Se=null,je=null,vl=!1,Or=!1,li=0,tg=0;function ze(){throw Error(A(321))}function vs(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!bt(e[n],t[n]))return!1;return!0}function xs(e,t,n,r,i,l){if(Ln=l,me=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Wi.current=e===null||e.memoizedState===null?lg:og,e=n(r,i),Or){l=0;do{if(Or=!1,li=0,25<=l)throw Error(A(301));l+=1,je=Se=null,t.updateQueue=null,Wi.current=ag,e=n(r,i)}while(Or)}if(Wi.current=xl,t=Se!==null&&Se.next!==null,Ln=0,je=Se=me=null,vl=!1,t)throw Error(A(300));return e}function ys(){var e=li!==0;return li=0,e}function Ct(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return je===null?me.memoizedState=je=e:je=je.next=e,je}function mt(){if(Se===null){var e=me.alternate;e=e!==null?e.memoizedState:null}else e=Se.next;var t=je===null?me.memoizedState:je.next;if(t!==null)je=t,Se=e;else{if(e===null)throw Error(A(310));Se=e,e={memoizedState:Se.memoizedState,baseState:Se.baseState,baseQueue:Se.baseQueue,queue:Se.queue,next:null},je===null?me.memoizedState=je=e:je=je.next=e}return je}function oi(e,t){return typeof t=="function"?t(e):t}function ho(e){var t=mt(),n=t.queue;if(n===null)throw Error(A(311));n.lastRenderedReducer=e;var r=Se,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,u=null,c=l;do{var d=c.lane;if((Ln&d)===d)u!==null&&(u=u.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var f={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};u===null?(a=u=f,o=r):u=u.next=f,me.lanes|=d,In|=d}c=c.next}while(c!==null&&c!==l);u===null?o=r:u.next=a,bt(r,t.memoizedState)||(qe=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=u,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,me.lanes|=l,In|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function mo(e){var t=mt(),n=t.queue;if(n===null)throw Error(A(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);bt(l,t.memoizedState)||(qe=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function Sf(){}function bf(e,t){var n=me,r=mt(),i=t(),l=!bt(r.memoizedState,i);if(l&&(r.memoizedState=i,qe=!0),r=r.queue,ks(Cf.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||je!==null&&je.memoizedState.tag&1){if(n.flags|=2048,ai(9,jf.bind(null,n,r,i,t),void 0,null),Ce===null)throw Error(A(349));Ln&30||_f(n,t,i)}return i}function _f(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=me.updateQueue,t===null?(t={lastEffect:null,stores:null},me.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function jf(e,t,n,r){t.value=n,t.getSnapshot=r,Nf(t)&&Ef(e)}function Cf(e,t,n){return n(function(){Nf(t)&&Ef(e)})}function Nf(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!bt(e,n)}catch{return!0}}function Ef(e){var t=Vt(e,1);t!==null&&St(t,e,1,-1)}function Bu(e){var t=Ct();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:oi,lastRenderedState:e},t.queue=e,e=e.dispatch=ig.bind(null,me,e),[t.memoizedState,e]}function ai(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=me.updateQueue,t===null?(t={lastEffect:null,stores:null},me.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function Tf(){return mt().memoizedState}function Qi(e,t,n,r){var i=Ct();me.flags|=e,i.memoizedState=ai(1|t,n,void 0,r===void 0?null:r)}function Ml(e,t,n,r){var i=mt();r=r===void 0?null:r;var l=void 0;if(Se!==null){var o=Se.memoizedState;if(l=o.destroy,r!==null&&vs(r,o.deps)){i.memoizedState=ai(t,n,l,r);return}}me.flags|=e,i.memoizedState=ai(1|t,n,l,r)}function $u(e,t){return Qi(8390656,8,e,t)}function ks(e,t){return Ml(2048,8,e,t)}function Lf(e,t){return Ml(4,2,e,t)}function If(e,t){return Ml(4,4,e,t)}function zf(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function Pf(e,t,n){return n=n!=null?n.concat([e]):null,Ml(4,4,zf.bind(null,t,e),n)}function ws(){}function Mf(e,t){var n=mt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&vs(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function Af(e,t){var n=mt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&vs(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function Rf(e,t,n){return Ln&21?(bt(n,t)||(n=Bd(),me.lanes|=n,In|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,qe=!0),e.memoizedState=n)}function ng(e,t){var n=ie;ie=n!==0&&4>n?n:4,e(!0);var r=po.transition;po.transition={};try{e(!1),t()}finally{ie=n,po.transition=r}}function Df(){return mt().memoizedState}function rg(e,t,n){var r=dn(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},Ff(e))Of(t,n);else if(n=yf(e,t,n,r),n!==null){var i=Be();St(n,e,r,i),Bf(n,t,r)}}function ig(e,t,n){var r=dn(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(Ff(e))Of(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,bt(a,o)){var u=t.interleaved;u===null?(i.next=i,fs(t)):(i.next=u.next,u.next=i),t.interleaved=i;return}}catch{}finally{}n=yf(e,t,i,r),n!==null&&(i=Be(),St(n,e,r,i),Bf(n,t,r))}}function Ff(e){var t=e.alternate;return e===me||t!==null&&t===me}function Of(e,t){Or=vl=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function Bf(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ja(e,n)}}var xl={readContext:ht,useCallback:ze,useContext:ze,useEffect:ze,useImperativeHandle:ze,useInsertionEffect:ze,useLayoutEffect:ze,useMemo:ze,useReducer:ze,useRef:ze,useState:ze,useDebugValue:ze,useDeferredValue:ze,useTransition:ze,useMutableSource:ze,useSyncExternalStore:ze,useId:ze,unstable_isNewReconciler:!1},lg={readContext:ht,useCallback:function(e,t){return Ct().memoizedState=[e,t===void 0?null:t],e},useContext:ht,useEffect:$u,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Qi(4194308,4,zf.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Qi(4194308,4,e,t)},useInsertionEffect:function(e,t){return Qi(4,2,e,t)},useMemo:function(e,t){var n=Ct();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=Ct();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=rg.bind(null,me,e),[r.memoizedState,e]},useRef:function(e){var t=Ct();return e={current:e},t.memoizedState=e},useState:Bu,useDebugValue:ws,useDeferredValue:function(e){return Ct().memoizedState=e},useTransition:function(){var e=Bu(!1),t=e[0];return e=ng.bind(null,e[1]),Ct().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=me,i=Ct();if(fe){if(n===void 0)throw Error(A(407));n=n()}else{if(n=t(),Ce===null)throw Error(A(349));Ln&30||_f(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,$u(Cf.bind(null,r,l,e),[e]),r.flags|=2048,ai(9,jf.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=Ct(),t=Ce.identifierPrefix;if(fe){var n=Bt,r=Ot;n=(r&~(1<<32-wt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=li++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=tg++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},og={readContext:ht,useCallback:Mf,useContext:ht,useEffect:ks,useImperativeHandle:Pf,useInsertionEffect:Lf,useLayoutEffect:If,useMemo:Af,useReducer:ho,useRef:Tf,useState:function(){return ho(oi)},useDebugValue:ws,useDeferredValue:function(e){var t=mt();return Rf(t,Se.memoizedState,e)},useTransition:function(){var e=ho(oi)[0],t=mt().memoizedState;return[e,t]},useMutableSource:Sf,useSyncExternalStore:bf,useId:Df,unstable_isNewReconciler:!1},ag={readContext:ht,useCallback:Mf,useContext:ht,useEffect:ks,useImperativeHandle:Pf,useInsertionEffect:Lf,useLayoutEffect:If,useMemo:Af,useReducer:mo,useRef:Tf,useState:function(){return mo(oi)},useDebugValue:ws,useDeferredValue:function(e){var t=mt();return Se===null?t.memoizedState=e:Rf(t,Se.memoizedState,e)},useTransition:function(){var e=mo(oi)[0],t=mt().memoizedState;return[e,t]},useMutableSource:Sf,useSyncExternalStore:bf,useId:Df,unstable_isNewReconciler:!1};function xt(e,t){if(e&&e.defaultProps){t=ge({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function da(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:ge({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var Al={isMounted:function(e){return(e=e._reactInternals)?Mn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Be(),i=dn(e),l=$t(r,i);l.payload=t,n!=null&&(l.callback=n),t=un(e,l,i),t!==null&&(St(t,e,i,r),Vi(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Be(),i=dn(e),l=$t(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=un(e,l,i),t!==null&&(St(t,e,i,r),Vi(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Be(),r=dn(e),i=$t(n,r);i.tag=2,t!=null&&(i.callback=t),t=un(e,i,r),t!==null&&(St(t,e,r,n),Vi(t,e,r))}};function Hu(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Zr(n,r)||!Zr(i,l):!0}function $f(e,t,n){var r=!1,i=hn,l=t.contextType;return typeof l=="object"&&l!==null?l=ht(l):(i=Ye(t)?En:Re.current,r=t.contextTypes,l=(r=r!=null)?ar(e,i):hn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=Al,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function Uu(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&Al.enqueueReplaceState(t,t.state,null)}function fa(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},ps(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=ht(l):(l=Ye(t)?En:Re.current,i.context=ar(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(da(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&Al.enqueueReplaceState(i,i.state,null),ml(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function dr(e,t){try{var n="",r=t;do n+=Ah(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function go(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function pa(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var sg=typeof WeakMap=="function"?WeakMap:Map;function Hf(e,t,n){n=$t(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){kl||(kl=!0,ba=r),pa(e,t)},n}function Uf(e,t,n){n=$t(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){pa(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){pa(e,t),typeof r!="function"&&(cn===null?cn=new Set([this]):cn.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function Vu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new sg;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=Sg.bind(null,e,t,n),t.then(e,e))}function Wu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Qu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=$t(-1,1),t.tag=2,un(n,t,1))),n.lanes|=1),e)}var ug=Qt.ReactCurrentOwner,qe=!1;function Oe(e,t,n,r){t.child=e===null?xf(t,null,n,r):ur(t,e.child,n,r)}function qu(e,t,n,r,i){n=n.render;var l=t.ref;return nr(t,i),r=xs(e,t,n,r,l,i),n=ys(),e!==null&&!qe?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Wt(e,t,i)):(fe&&n&&os(t),t.flags|=1,Oe(e,t,r,i),t.child)}function Ku(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!Ts(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,Vf(e,t,l,r,i)):(e=Gi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Zr,n(o,r)&&e.ref===t.ref)return Wt(e,t,i)}return t.flags|=1,e=fn(l,r),e.ref=t.ref,e.return=t,t.child=e}function Vf(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Zr(l,r)&&e.ref===t.ref)if(qe=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(qe=!0);else return t.lanes=e.lanes,Wt(e,t,i)}return ha(e,t,n,r,i)}function Wf(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},se(Xn,nt),nt|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,se(Xn,nt),nt|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,se(Xn,nt),nt|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,se(Xn,nt),nt|=r;return Oe(e,t,i,n),t.child}function Qf(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ha(e,t,n,r,i){var l=Ye(n)?En:Re.current;return l=ar(t,l),nr(t,i),n=xs(e,t,n,r,l,i),r=ys(),e!==null&&!qe?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Wt(e,t,i)):(fe&&r&&os(t),t.flags|=1,Oe(e,t,n,i),t.child)}function Yu(e,t,n,r,i){if(Ye(n)){var l=!0;cl(t)}else l=!1;if(nr(t,i),t.stateNode===null)qi(e,t),$f(t,n,r),fa(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var u=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=ht(c):(c=Ye(n)?En:Re.current,c=ar(t,c));var d=n.getDerivedStateFromProps,f=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";f||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||u!==c)&&Uu(t,o,r,c),Zt=!1;var g=t.memoizedState;o.state=g,ml(t,r,o,i),u=t.memoizedState,a!==r||g!==u||Ke.current||Zt?(typeof d=="function"&&(da(t,n,d,r),u=t.memoizedState),(a=Zt||Hu(t,n,a,r,g,u,c))?(f||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=u),o.props=r,o.state=u,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,kf(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:xt(t.type,a),o.props=c,f=t.pendingProps,g=o.context,u=n.contextType,typeof u=="object"&&u!==null?u=ht(u):(u=Ye(n)?En:Re.current,u=ar(t,u));var p=n.getDerivedStateFromProps;(d=typeof p=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==f||g!==u)&&Uu(t,o,r,u),Zt=!1,g=t.memoizedState,o.state=g,ml(t,r,o,i);var k=t.memoizedState;a!==f||g!==k||Ke.current||Zt?(typeof p=="function"&&(da(t,n,p,r),k=t.memoizedState),(c=Zt||Hu(t,n,c,r,g,k,u)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,k,u),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,k,u)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=k),o.props=r,o.state=k,o.context=u,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=1024),r=!1)}return ma(e,t,n,r,l,i)}function ma(e,t,n,r,i,l){Qf(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&Mu(t,n,!1),Wt(e,t,l);r=t.stateNode,ug.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=ur(t,e.child,null,l),t.child=ur(t,null,a,l)):Oe(e,t,a,l),t.memoizedState=r.state,i&&Mu(t,n,!0),t.child}function qf(e){var t=e.stateNode;t.pendingContext?Pu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&Pu(e,t.context,!1),hs(e,t.containerInfo)}function Gu(e,t,n,r,i){return sr(),ss(i),t.flags|=256,Oe(e,t,n,r),t.child}var ga={dehydrated:null,treeContext:null,retryLane:0};function va(e){return{baseLanes:e,cachePool:null,transitions:null}}function Kf(e,t,n){var r=t.pendingProps,i=he.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),se(he,i&1),e===null)return ua(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Fl(o,r,0,null),e=Nn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=va(n),t.memoizedState=ga,e):Ss(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return cg(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var u={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=u,t.deletions=null):(r=fn(i,u),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=fn(a,l):(l=Nn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?va(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=ga,r}return l=e.child,e=l.sibling,r=fn(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function Ss(e,t){return t=Fl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function Li(e,t,n,r){return r!==null&&ss(r),ur(t,e.child,null,n),e=Ss(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function cg(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=go(Error(A(422))),Li(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Fl({mode:"visible",children:r.children},i,0,null),l=Nn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&ur(t,e.child,null,o),t.child.memoizedState=va(o),t.memoizedState=ga,l);if(!(t.mode&1))return Li(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(A(419)),r=go(l,r,void 0),Li(e,t,o,r)}if(a=(o&e.childLanes)!==0,qe||a){if(r=Ce,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Vt(e,i),St(r,e,i,-1))}return Es(),r=go(Error(A(421))),Li(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=bg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,rt=sn(i.nextSibling),lt=t,fe=!0,kt=null,e!==null&&(ut[ct++]=Ot,ut[ct++]=Bt,ut[ct++]=Tn,Ot=e.id,Bt=e.overflow,Tn=t),t=Ss(t,r.children),t.flags|=4096,t)}function Xu(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),ca(e.return,t,n)}function vo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Yf(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(Oe(e,t,r.children,n),r=he.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Xu(e,n,t);else if(e.tag===19)Xu(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(se(he,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&gl(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),vo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&gl(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}vo(t,!0,n,null,l);break;case"together":vo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function qi(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Wt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),In|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(A(153));if(t.child!==null){for(e=t.child,n=fn(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=fn(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function dg(e,t,n){switch(t.tag){case 3:qf(t),sr();break;case 5:wf(t);break;case 1:Ye(t.type)&&cl(t);break;case 4:hs(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;se(pl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(se(he,he.current&1),t.flags|=128,null):n&t.child.childLanes?Kf(e,t,n):(se(he,he.current&1),e=Wt(e,t,n),e!==null?e.sibling:null);se(he,he.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Yf(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),se(he,he.current),r)break;return null;case 22:case 23:return t.lanes=0,Wf(e,t,n)}return Wt(e,t,n)}var Gf,xa,Xf,Jf;Gf=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};xa=function(){};Xf=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,jn(It.current);var l=null;switch(n){case"input":i=Bo(e,i),r=Bo(e,r),l=[];break;case"select":i=ge({},i,{value:void 0}),r=ge({},r,{value:void 0}),l=[];break;case"textarea":i=Uo(e,i),r=Uo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=sl)}Wo(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(Qr.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var u=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&u!==a&&(u!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||u&&u.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in u)u.hasOwnProperty(o)&&a[o]!==u[o]&&(n||(n={}),n[o]=u[o])}else n||(l||(l=[]),l.push(c,n)),n=u;else c==="dangerouslySetInnerHTML"?(u=u?u.__html:void 0,a=a?a.__html:void 0,u!=null&&a!==u&&(l=l||[]).push(c,u)):c==="children"?typeof u!="string"&&typeof u!="number"||(l=l||[]).push(c,""+u):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(Qr.hasOwnProperty(c)?(u!=null&&c==="onScroll"&&ce("scroll",e),l||a===u||(l=[])):(l=l||[]).push(c,u))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};Jf=function(e,t,n,r){n!==r&&(t.flags|=4)};function Cr(e,t){if(!fe)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function Pe(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function fg(e,t,n){var r=t.pendingProps;switch(as(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return Pe(t),null;case 1:return Ye(t.type)&&ul(),Pe(t),null;case 3:return r=t.stateNode,cr(),de(Ke),de(Re),gs(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(Ei(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,kt!==null&&(Ca(kt),kt=null))),xa(e,t),Pe(t),null;case 5:ms(t);var i=jn(ii.current);if(n=t.type,e!==null&&t.stateNode!=null)Xf(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(A(166));return Pe(t),null}if(e=jn(It.current),Ei(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[Et]=t,r[ni]=l,e=(t.mode&1)!==0,n){case"dialog":ce("cancel",r),ce("close",r);break;case"iframe":case"object":case"embed":ce("load",r);break;case"video":case"audio":for(i=0;i<Pr.length;i++)ce(Pr[i],r);break;case"source":ce("error",r);break;case"img":case"image":case"link":ce("error",r),ce("load",r);break;case"details":ce("toggle",r);break;case"input":ou(r,l),ce("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},ce("invalid",r);break;case"textarea":su(r,l),ce("invalid",r)}Wo(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&Ni(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&Ni(r.textContent,a,e),i=["children",""+a]):Qr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&ce("scroll",r)}switch(n){case"input":yi(r),au(r,l,!0);break;case"textarea":yi(r),uu(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=sl)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=jd(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[Et]=t,e[ni]=r,Gf(e,t,!1,!1),t.stateNode=e;e:{switch(o=Qo(n,r),n){case"dialog":ce("cancel",e),ce("close",e),i=r;break;case"iframe":case"object":case"embed":ce("load",e),i=r;break;case"video":case"audio":for(i=0;i<Pr.length;i++)ce(Pr[i],e);i=r;break;case"source":ce("error",e),i=r;break;case"img":case"image":case"link":ce("error",e),ce("load",e),i=r;break;case"details":ce("toggle",e),i=r;break;case"input":ou(e,r),i=Bo(e,r),ce("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=ge({},r,{value:void 0}),ce("invalid",e);break;case"textarea":su(e,r),i=Uo(e,r),ce("invalid",e);break;default:i=r}Wo(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var u=a[l];l==="style"?Ed(e,u):l==="dangerouslySetInnerHTML"?(u=u?u.__html:void 0,u!=null&&Cd(e,u)):l==="children"?typeof u=="string"?(n!=="textarea"||u!=="")&&qr(e,u):typeof u=="number"&&qr(e,""+u):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Qr.hasOwnProperty(l)?u!=null&&l==="onScroll"&&ce("scroll",e):u!=null&&Qa(e,l,u,o))}switch(n){case"input":yi(e),au(e,r,!1);break;case"textarea":yi(e),uu(e);break;case"option":r.value!=null&&e.setAttribute("value",""+pn(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Jn(e,!!r.multiple,l,!1):r.defaultValue!=null&&Jn(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=sl)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return Pe(t),null;case 6:if(e&&t.stateNode!=null)Jf(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(A(166));if(n=jn(ii.current),jn(It.current),Ei(t)){if(r=t.stateNode,n=t.memoizedProps,r[Et]=t,(l=r.nodeValue!==n)&&(e=lt,e!==null))switch(e.tag){case 3:Ni(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&Ni(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[Et]=t,t.stateNode=r}return Pe(t),null;case 13:if(de(he),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(fe&&rt!==null&&t.mode&1&&!(t.flags&128))gf(),sr(),t.flags|=98560,l=!1;else if(l=Ei(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(A(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(A(317));l[Et]=t}else sr(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;Pe(t),l=!1}else kt!==null&&(Ca(kt),kt=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||he.current&1?be===0&&(be=3):Es())),t.updateQueue!==null&&(t.flags|=4),Pe(t),null);case 4:return cr(),xa(e,t),e===null&&ei(t.stateNode.containerInfo),Pe(t),null;case 10:return ds(t.type._context),Pe(t),null;case 17:return Ye(t.type)&&ul(),Pe(t),null;case 19:if(de(he),l=t.memoizedState,l===null)return Pe(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)Cr(l,!1);else{if(be!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=gl(e),o!==null){for(t.flags|=128,Cr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return se(he,he.current&1|2),t.child}e=e.sibling}l.tail!==null&&ye()>fr&&(t.flags|=128,r=!0,Cr(l,!1),t.lanes=4194304)}else{if(!r)if(e=gl(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),Cr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!fe)return Pe(t),null}else 2*ye()-l.renderingStartTime>fr&&n!==1073741824&&(t.flags|=128,r=!0,Cr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=ye(),t.sibling=null,n=he.current,se(he,r?n&1|2:n&1),t):(Pe(t),null);case 22:case 23:return Ns(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?nt&1073741824&&(Pe(t),t.subtreeFlags&6&&(t.flags|=8192)):Pe(t),null;case 24:return null;case 25:return null}throw Error(A(156,t.tag))}function pg(e,t){switch(as(t),t.tag){case 1:return Ye(t.type)&&ul(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return cr(),de(Ke),de(Re),gs(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return ms(t),null;case 13:if(de(he),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(A(340));sr()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return de(he),null;case 4:return cr(),null;case 10:return ds(t.type._context),null;case 22:case 23:return Ns(),null;case 24:return null;default:return null}}var Ii=!1,Ae=!1,hg=typeof WeakSet=="function"?WeakSet:Set,H=null;function Gn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){ve(e,t,r)}else n.current=null}function ya(e,t,n){try{n()}catch(r){ve(e,t,r)}}var Ju=!1;function mg(e,t){if(na=ll,e=nf(),ls(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,u=-1,c=0,d=0,f=e,g=null;t:for(;;){for(var p;f!==n||i!==0&&f.nodeType!==3||(a=o+i),f!==l||r!==0&&f.nodeType!==3||(u=o+r),f.nodeType===3&&(o+=f.nodeValue.length),(p=f.firstChild)!==null;)g=f,f=p;for(;;){if(f===e)break t;if(g===n&&++c===i&&(a=o),g===l&&++d===r&&(u=o),(p=f.nextSibling)!==null)break;f=g,g=f.parentNode}f=p}n=a===-1||u===-1?null:{start:a,end:u}}else n=null}n=n||{start:0,end:0}}else n=null;for(ra={focusedElem:e,selectionRange:n},ll=!1,H=t;H!==null;)if(t=H,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,H=e;else for(;H!==null;){t=H;try{var k=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(k!==null){var w=k.memoizedProps,z=k.memoizedState,h=t.stateNode,v=h.getSnapshotBeforeUpdate(t.elementType===t.type?w:xt(t.type,w),z);h.__reactInternalSnapshotBeforeUpdate=v}break;case 3:var x=t.stateNode.containerInfo;x.nodeType===1?x.textContent="":x.nodeType===9&&x.documentElement&&x.removeChild(x.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(A(163))}}catch(b){ve(t,t.return,b)}if(e=t.sibling,e!==null){e.return=t.return,H=e;break}H=t.return}return k=Ju,Ju=!1,k}function Br(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&ya(t,n,l)}i=i.next}while(i!==r)}}function Rl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function ka(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Zf(e){var t=e.alternate;t!==null&&(e.alternate=null,Zf(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[Et],delete t[ni],delete t[oa],delete t[Xm],delete t[Jm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function ep(e){return e.tag===5||e.tag===3||e.tag===4}function Zu(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||ep(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function wa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=sl));else if(r!==4&&(e=e.child,e!==null))for(wa(e,t,n),e=e.sibling;e!==null;)wa(e,t,n),e=e.sibling}function Sa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(Sa(e,t,n),e=e.sibling;e!==null;)Sa(e,t,n),e=e.sibling}var Te=null,yt=!1;function Yt(e,t,n){for(n=n.child;n!==null;)tp(e,t,n),n=n.sibling}function tp(e,t,n){if(Lt&&typeof Lt.onCommitFiberUnmount=="function")try{Lt.onCommitFiberUnmount(El,n)}catch{}switch(n.tag){case 5:Ae||Gn(n,t);case 6:var r=Te,i=yt;Te=null,Yt(e,t,n),Te=r,yt=i,Te!==null&&(yt?(e=Te,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):Te.removeChild(n.stateNode));break;case 18:Te!==null&&(yt?(e=Te,n=n.stateNode,e.nodeType===8?uo(e.parentNode,n):e.nodeType===1&&uo(e,n),Xr(e)):uo(Te,n.stateNode));break;case 4:r=Te,i=yt,Te=n.stateNode.containerInfo,yt=!0,Yt(e,t,n),Te=r,yt=i;break;case 0:case 11:case 14:case 15:if(!Ae&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&ya(n,t,o),i=i.next}while(i!==r)}Yt(e,t,n);break;case 1:if(!Ae&&(Gn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){ve(n,t,a)}Yt(e,t,n);break;case 21:Yt(e,t,n);break;case 22:n.mode&1?(Ae=(r=Ae)||n.memoizedState!==null,Yt(e,t,n),Ae=r):Yt(e,t,n);break;default:Yt(e,t,n)}}function ec(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new hg),t.forEach(function(r){var i=_g.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function vt(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:Te=a.stateNode,yt=!1;break e;case 3:Te=a.stateNode.containerInfo,yt=!0;break e;case 4:Te=a.stateNode.containerInfo,yt=!0;break e}a=a.return}if(Te===null)throw Error(A(160));tp(l,o,i),Te=null,yt=!1;var u=i.alternate;u!==null&&(u.return=null),i.return=null}catch(c){ve(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)np(t,e),t=t.sibling}function np(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(vt(t,e),_t(e),r&4){try{Br(3,e,e.return),Rl(3,e)}catch(w){ve(e,e.return,w)}try{Br(5,e,e.return)}catch(w){ve(e,e.return,w)}}break;case 1:vt(t,e),_t(e),r&512&&n!==null&&Gn(n,n.return);break;case 5:if(vt(t,e),_t(e),r&512&&n!==null&&Gn(n,n.return),e.flags&32){var i=e.stateNode;try{qr(i,"")}catch(w){ve(e,e.return,w)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,u=e.updateQueue;if(e.updateQueue=null,u!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&bd(i,l),Qo(a,o);var c=Qo(a,l);for(o=0;o<u.length;o+=2){var d=u[o],f=u[o+1];d==="style"?Ed(i,f):d==="dangerouslySetInnerHTML"?Cd(i,f):d==="children"?qr(i,f):Qa(i,d,f,c)}switch(a){case"input":$o(i,l);break;case"textarea":_d(i,l);break;case"select":var g=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var p=l.value;p!=null?Jn(i,!!l.multiple,p,!1):g!==!!l.multiple&&(l.defaultValue!=null?Jn(i,!!l.multiple,l.defaultValue,!0):Jn(i,!!l.multiple,l.multiple?[]:"",!1))}i[ni]=l}catch(w){ve(e,e.return,w)}}break;case 6:if(vt(t,e),_t(e),r&4){if(e.stateNode===null)throw Error(A(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(w){ve(e,e.return,w)}}break;case 3:if(vt(t,e),_t(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Xr(t.containerInfo)}catch(w){ve(e,e.return,w)}break;case 4:vt(t,e),_t(e);break;case 13:vt(t,e),_t(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(js=ye())),r&4&&ec(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Ae=(c=Ae)||d,vt(t,e),Ae=c):vt(t,e),_t(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for(H=e,d=e.child;d!==null;){for(f=H=d;H!==null;){switch(g=H,p=g.child,g.tag){case 0:case 11:case 14:case 15:Br(4,g,g.return);break;case 1:Gn(g,g.return);var k=g.stateNode;if(typeof k.componentWillUnmount=="function"){r=g,n=g.return;try{t=r,k.props=t.memoizedProps,k.state=t.memoizedState,k.componentWillUnmount()}catch(w){ve(r,n,w)}}break;case 5:Gn(g,g.return);break;case 22:if(g.memoizedState!==null){nc(f);continue}}p!==null?(p.return=g,H=p):nc(f)}d=d.sibling}e:for(d=null,f=e;;){if(f.tag===5){if(d===null){d=f;try{i=f.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=f.stateNode,u=f.memoizedProps.style,o=u!=null&&u.hasOwnProperty("display")?u.display:null,a.style.display=Nd("display",o))}catch(w){ve(e,e.return,w)}}}else if(f.tag===6){if(d===null)try{f.stateNode.nodeValue=c?"":f.memoizedProps}catch(w){ve(e,e.return,w)}}else if((f.tag!==22&&f.tag!==23||f.memoizedState===null||f===e)&&f.child!==null){f.child.return=f,f=f.child;continue}if(f===e)break e;for(;f.sibling===null;){if(f.return===null||f.return===e)break e;d===f&&(d=null),f=f.return}d===f&&(d=null),f.sibling.return=f.return,f=f.sibling}}break;case 19:vt(t,e),_t(e),r&4&&ec(e);break;case 21:break;default:vt(t,e),_t(e)}}function _t(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(ep(n)){var r=n;break e}n=n.return}throw Error(A(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(qr(i,""),r.flags&=-33);var l=Zu(e);Sa(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Zu(e);wa(e,a,o);break;default:throw Error(A(161))}}catch(u){ve(e,e.return,u)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function gg(e,t,n){H=e,rp(e)}function rp(e,t,n){for(var r=(e.mode&1)!==0;H!==null;){var i=H,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||Ii;if(!o){var a=i.alternate,u=a!==null&&a.memoizedState!==null||Ae;a=Ii;var c=Ae;if(Ii=o,(Ae=u)&&!c)for(H=i;H!==null;)o=H,u=o.child,o.tag===22&&o.memoizedState!==null?rc(i):u!==null?(u.return=o,H=u):rc(i);for(;l!==null;)H=l,rp(l),l=l.sibling;H=i,Ii=a,Ae=c}tc(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,H=l):tc(e)}}function tc(e){for(;H!==null;){var t=H;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Ae||Rl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Ae)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:xt(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Ou(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Ou(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var u=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":u.autoFocus&&n.focus();break;case"img":u.src&&(n.src=u.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var f=d.dehydrated;f!==null&&Xr(f)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(A(163))}Ae||t.flags&512&&ka(t)}catch(g){ve(t,t.return,g)}}if(t===e){H=null;break}if(n=t.sibling,n!==null){n.return=t.return,H=n;break}H=t.return}}function nc(e){for(;H!==null;){var t=H;if(t===e){H=null;break}var n=t.sibling;if(n!==null){n.return=t.return,H=n;break}H=t.return}}function rc(e){for(;H!==null;){var t=H;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Rl(4,t)}catch(u){ve(t,n,u)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(u){ve(t,i,u)}}var l=t.return;try{ka(t)}catch(u){ve(t,l,u)}break;case 5:var o=t.return;try{ka(t)}catch(u){ve(t,o,u)}}}catch(u){ve(t,t.return,u)}if(t===e){H=null;break}var a=t.sibling;if(a!==null){a.return=t.return,H=a;break}H=t.return}}var vg=Math.ceil,yl=Qt.ReactCurrentDispatcher,bs=Qt.ReactCurrentOwner,pt=Qt.ReactCurrentBatchConfig,ne=0,Ce=null,we=null,Le=0,nt=0,Xn=gn(0),be=0,si=null,In=0,Dl=0,_s=0,$r=null,Qe=null,js=0,fr=1/0,Dt=null,kl=!1,ba=null,cn=null,zi=!1,rn=null,wl=0,Hr=0,_a=null,Ki=-1,Yi=0;function Be(){return ne&6?ye():Ki!==-1?Ki:Ki=ye()}function dn(e){return e.mode&1?ne&2&&Le!==0?Le&-Le:eg.transition!==null?(Yi===0&&(Yi=Bd()),Yi):(e=ie,e!==0||(e=window.event,e=e===void 0?16:qd(e.type)),e):1}function St(e,t,n,r){if(50<Hr)throw Hr=0,_a=null,Error(A(185));di(e,n,r),(!(ne&2)||e!==Ce)&&(e===Ce&&(!(ne&2)&&(Dl|=n),be===4&&tn(e,Le)),Ge(e,r),n===1&&ne===0&&!(t.mode&1)&&(fr=ye()+500,Pl&&vn()))}function Ge(e,t){var n=e.callbackNode;em(e,t);var r=il(e,e===Ce?Le:0);if(r===0)n!==null&&fu(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&fu(n),t===1)e.tag===0?Zm(ic.bind(null,e)):pf(ic.bind(null,e)),Ym(function(){!(ne&6)&&vn()}),n=null;else{switch($d(r)){case 1:n=Xa;break;case 4:n=Fd;break;case 16:n=rl;break;case 536870912:n=Od;break;default:n=rl}n=dp(n,ip.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function ip(e,t){if(Ki=-1,Yi=0,ne&6)throw Error(A(327));var n=e.callbackNode;if(rr()&&e.callbackNode!==n)return null;var r=il(e,e===Ce?Le:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=Sl(e,r);else{t=r;var i=ne;ne|=2;var l=op();(Ce!==e||Le!==t)&&(Dt=null,fr=ye()+500,Cn(e,t));do try{kg();break}catch(a){lp(e,a)}while(!0);cs(),yl.current=l,ne=i,we!==null?t=0:(Ce=null,Le=0,t=be)}if(t!==0){if(t===2&&(i=Xo(e),i!==0&&(r=i,t=ja(e,i))),t===1)throw n=si,Cn(e,0),tn(e,r),Ge(e,ye()),n;if(t===6)tn(e,r);else{if(i=e.current.alternate,!(r&30)&&!xg(i)&&(t=Sl(e,r),t===2&&(l=Xo(e),l!==0&&(r=l,t=ja(e,l))),t===1))throw n=si,Cn(e,0),tn(e,r),Ge(e,ye()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(A(345));case 2:wn(e,Qe,Dt);break;case 3:if(tn(e,r),(r&130023424)===r&&(t=js+500-ye(),10<t)){if(il(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Be(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=la(wn.bind(null,e,Qe,Dt),t);break}wn(e,Qe,Dt);break;case 4:if(tn(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-wt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=ye()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*vg(r/1960))-r,10<r){e.timeoutHandle=la(wn.bind(null,e,Qe,Dt),r);break}wn(e,Qe,Dt);break;case 5:wn(e,Qe,Dt);break;default:throw Error(A(329))}}}return Ge(e,ye()),e.callbackNode===n?ip.bind(null,e):null}function ja(e,t){var n=$r;return e.current.memoizedState.isDehydrated&&(Cn(e,t).flags|=256),e=Sl(e,t),e!==2&&(t=Qe,Qe=n,t!==null&&Ca(t)),e}function Ca(e){Qe===null?Qe=e:Qe.push.apply(Qe,e)}function xg(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!bt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function tn(e,t){for(t&=~_s,t&=~Dl,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-wt(t),r=1<<n;e[n]=-1,t&=~r}}function ic(e){if(ne&6)throw Error(A(327));rr();var t=il(e,0);if(!(t&1))return Ge(e,ye()),null;var n=Sl(e,t);if(e.tag!==0&&n===2){var r=Xo(e);r!==0&&(t=r,n=ja(e,r))}if(n===1)throw n=si,Cn(e,0),tn(e,t),Ge(e,ye()),n;if(n===6)throw Error(A(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,wn(e,Qe,Dt),Ge(e,ye()),null}function Cs(e,t){var n=ne;ne|=1;try{return e(t)}finally{ne=n,ne===0&&(fr=ye()+500,Pl&&vn())}}function zn(e){rn!==null&&rn.tag===0&&!(ne&6)&&rr();var t=ne;ne|=1;var n=pt.transition,r=ie;try{if(pt.transition=null,ie=1,e)return e()}finally{ie=r,pt.transition=n,ne=t,!(ne&6)&&vn()}}function Ns(){nt=Xn.current,de(Xn)}function Cn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,Km(n)),we!==null)for(n=we.return;n!==null;){var r=n;switch(as(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&ul();break;case 3:cr(),de(Ke),de(Re),gs();break;case 5:ms(r);break;case 4:cr();break;case 13:de(he);break;case 19:de(he);break;case 10:ds(r.type._context);break;case 22:case 23:Ns()}n=n.return}if(Ce=e,we=e=fn(e.current,null),Le=nt=t,be=0,si=null,_s=Dl=In=0,Qe=$r=null,_n!==null){for(t=0;t<_n.length;t++)if(n=_n[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}_n=null}return e}function lp(e,t){do{var n=we;try{if(cs(),Wi.current=xl,vl){for(var r=me.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}vl=!1}if(Ln=0,je=Se=me=null,Or=!1,li=0,bs.current=null,n===null||n.return===null){be=1,si=t,we=null;break}e:{var l=e,o=n.return,a=n,u=t;if(t=Le,a.flags|=32768,u!==null&&typeof u=="object"&&typeof u.then=="function"){var c=u,d=a,f=d.tag;if(!(d.mode&1)&&(f===0||f===11||f===15)){var g=d.alternate;g?(d.updateQueue=g.updateQueue,d.memoizedState=g.memoizedState,d.lanes=g.lanes):(d.updateQueue=null,d.memoizedState=null)}var p=Wu(o);if(p!==null){p.flags&=-257,Qu(p,o,a,l,t),p.mode&1&&Vu(l,c,t),t=p,u=c;var k=t.updateQueue;if(k===null){var w=new Set;w.add(u),t.updateQueue=w}else k.add(u);break e}else{if(!(t&1)){Vu(l,c,t),Es();break e}u=Error(A(426))}}else if(fe&&a.mode&1){var z=Wu(o);if(z!==null){!(z.flags&65536)&&(z.flags|=256),Qu(z,o,a,l,t),ss(dr(u,a));break e}}l=u=dr(u,a),be!==4&&(be=2),$r===null?$r=[l]:$r.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var h=Hf(l,u,t);Fu(l,h);break e;case 1:a=u;var v=l.type,x=l.stateNode;if(!(l.flags&128)&&(typeof v.getDerivedStateFromError=="function"||x!==null&&typeof x.componentDidCatch=="function"&&(cn===null||!cn.has(x)))){l.flags|=65536,t&=-t,l.lanes|=t;var b=Uf(l,a,t);Fu(l,b);break e}}l=l.return}while(l!==null)}sp(n)}catch(N){t=N,we===n&&n!==null&&(we=n=n.return);continue}break}while(!0)}function op(){var e=yl.current;return yl.current=xl,e===null?xl:e}function Es(){(be===0||be===3||be===2)&&(be=4),Ce===null||!(In&268435455)&&!(Dl&268435455)||tn(Ce,Le)}function Sl(e,t){var n=ne;ne|=2;var r=op();(Ce!==e||Le!==t)&&(Dt=null,Cn(e,t));do try{yg();break}catch(i){lp(e,i)}while(!0);if(cs(),ne=n,yl.current=r,we!==null)throw Error(A(261));return Ce=null,Le=0,be}function yg(){for(;we!==null;)ap(we)}function kg(){for(;we!==null&&!Wh();)ap(we)}function ap(e){var t=cp(e.alternate,e,nt);e.memoizedProps=e.pendingProps,t===null?sp(e):we=t,bs.current=null}function sp(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=pg(n,t),n!==null){n.flags&=32767,we=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{be=6,we=null;return}}else if(n=fg(n,t,nt),n!==null){we=n;return}if(t=t.sibling,t!==null){we=t;return}we=t=e}while(t!==null);be===0&&(be=5)}function wn(e,t,n){var r=ie,i=pt.transition;try{pt.transition=null,ie=1,wg(e,t,n,r)}finally{pt.transition=i,ie=r}return null}function wg(e,t,n,r){do rr();while(rn!==null);if(ne&6)throw Error(A(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(A(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(tm(e,l),e===Ce&&(we=Ce=null,Le=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||zi||(zi=!0,dp(rl,function(){return rr(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=pt.transition,pt.transition=null;var o=ie;ie=1;var a=ne;ne|=4,bs.current=null,mg(e,n),np(n,e),$m(ra),ll=!!na,ra=na=null,e.current=n,gg(n),Qh(),ne=a,ie=o,pt.transition=l}else e.current=n;if(zi&&(zi=!1,rn=e,wl=i),l=e.pendingLanes,l===0&&(cn=null),Yh(n.stateNode),Ge(e,ye()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(kl)throw kl=!1,e=ba,ba=null,e;return wl&1&&e.tag!==0&&rr(),l=e.pendingLanes,l&1?e===_a?Hr++:(Hr=0,_a=e):Hr=0,vn(),null}function rr(){if(rn!==null){var e=$d(wl),t=pt.transition,n=ie;try{if(pt.transition=null,ie=16>e?16:e,rn===null)var r=!1;else{if(e=rn,rn=null,wl=0,ne&6)throw Error(A(331));var i=ne;for(ne|=4,H=e.current;H!==null;){var l=H,o=l.child;if(H.flags&16){var a=l.deletions;if(a!==null){for(var u=0;u<a.length;u++){var c=a[u];for(H=c;H!==null;){var d=H;switch(d.tag){case 0:case 11:case 15:Br(8,d,l)}var f=d.child;if(f!==null)f.return=d,H=f;else for(;H!==null;){d=H;var g=d.sibling,p=d.return;if(Zf(d),d===c){H=null;break}if(g!==null){g.return=p,H=g;break}H=p}}}var k=l.alternate;if(k!==null){var w=k.child;if(w!==null){k.child=null;do{var z=w.sibling;w.sibling=null,w=z}while(w!==null)}}H=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,H=o;else e:for(;H!==null;){if(l=H,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Br(9,l,l.return)}var h=l.sibling;if(h!==null){h.return=l.return,H=h;break e}H=l.return}}var v=e.current;for(H=v;H!==null;){o=H;var x=o.child;if(o.subtreeFlags&2064&&x!==null)x.return=o,H=x;else e:for(o=v;H!==null;){if(a=H,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Rl(9,a)}}catch(N){ve(a,a.return,N)}if(a===o){H=null;break e}var b=a.sibling;if(b!==null){b.return=a.return,H=b;break e}H=a.return}}if(ne=i,vn(),Lt&&typeof Lt.onPostCommitFiberRoot=="function")try{Lt.onPostCommitFiberRoot(El,e)}catch{}r=!0}return r}finally{ie=n,pt.transition=t}}return!1}function lc(e,t,n){t=dr(n,t),t=Hf(e,t,1),e=un(e,t,1),t=Be(),e!==null&&(di(e,1,t),Ge(e,t))}function ve(e,t,n){if(e.tag===3)lc(e,e,n);else for(;t!==null;){if(t.tag===3){lc(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(cn===null||!cn.has(r))){e=dr(n,e),e=Uf(t,e,1),t=un(t,e,1),e=Be(),t!==null&&(di(t,1,e),Ge(t,e));break}}t=t.return}}function Sg(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Be(),e.pingedLanes|=e.suspendedLanes&n,Ce===e&&(Le&n)===n&&(be===4||be===3&&(Le&130023424)===Le&&500>ye()-js?Cn(e,0):_s|=n),Ge(e,t)}function up(e,t){t===0&&(e.mode&1?(t=Si,Si<<=1,!(Si&130023424)&&(Si=4194304)):t=1);var n=Be();e=Vt(e,t),e!==null&&(di(e,t,n),Ge(e,n))}function bg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),up(e,n)}function _g(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(A(314))}r!==null&&r.delete(t),up(e,n)}var cp;cp=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Ke.current)qe=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return qe=!1,dg(e,t,n);qe=!!(e.flags&131072)}else qe=!1,fe&&t.flags&1048576&&hf(t,fl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;qi(e,t),e=t.pendingProps;var i=ar(t,Re.current);nr(t,n),i=xs(null,t,r,e,i,n);var l=ys();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Ye(r)?(l=!0,cl(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,ps(t),i.updater=Al,t.stateNode=i,i._reactInternals=t,fa(t,r,e,n),t=ma(null,t,r,!0,l,n)):(t.tag=0,fe&&l&&os(t),Oe(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(qi(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=Cg(r),e=xt(r,e),i){case 0:t=ha(null,t,r,e,n);break e;case 1:t=Yu(null,t,r,e,n);break e;case 11:t=qu(null,t,r,e,n);break e;case 14:t=Ku(null,t,r,xt(r.type,e),n);break e}throw Error(A(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:xt(r,i),ha(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:xt(r,i),Yu(e,t,r,i,n);case 3:e:{if(qf(t),e===null)throw Error(A(387));r=t.pendingProps,l=t.memoizedState,i=l.element,kf(e,t),ml(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=dr(Error(A(423)),t),t=Gu(e,t,r,n,i);break e}else if(r!==i){i=dr(Error(A(424)),t),t=Gu(e,t,r,n,i);break e}else for(rt=sn(t.stateNode.containerInfo.firstChild),lt=t,fe=!0,kt=null,n=xf(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(sr(),r===i){t=Wt(e,t,n);break e}Oe(e,t,r,n)}t=t.child}return t;case 5:return wf(t),e===null&&ua(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,ia(r,i)?o=null:l!==null&&ia(r,l)&&(t.flags|=32),Qf(e,t),Oe(e,t,o,n),t.child;case 6:return e===null&&ua(t),null;case 13:return Kf(e,t,n);case 4:return hs(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=ur(t,null,r,n):Oe(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:xt(r,i),qu(e,t,r,i,n);case 7:return Oe(e,t,t.pendingProps,n),t.child;case 8:return Oe(e,t,t.pendingProps.children,n),t.child;case 12:return Oe(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,se(pl,r._currentValue),r._currentValue=o,l!==null)if(bt(l.value,o)){if(l.children===i.children&&!Ke.current){t=Wt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var u=a.firstContext;u!==null;){if(u.context===r){if(l.tag===1){u=$t(-1,n&-n),u.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?u.next=u:(u.next=d.next,d.next=u),c.pending=u}}l.lanes|=n,u=l.alternate,u!==null&&(u.lanes|=n),ca(l.return,n,t),a.lanes|=n;break}u=u.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(A(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),ca(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}Oe(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,nr(t,n),i=ht(i),r=r(i),t.flags|=1,Oe(e,t,r,n),t.child;case 14:return r=t.type,i=xt(r,t.pendingProps),i=xt(r.type,i),Ku(e,t,r,i,n);case 15:return Vf(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:xt(r,i),qi(e,t),t.tag=1,Ye(r)?(e=!0,cl(t)):e=!1,nr(t,n),$f(t,r,i),fa(t,r,i,n),ma(null,t,r,!0,e,n);case 19:return Yf(e,t,n);case 22:return Wf(e,t,n)}throw Error(A(156,t.tag))};function dp(e,t){return Dd(e,t)}function jg(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function ft(e,t,n,r){return new jg(e,t,n,r)}function Ts(e){return e=e.prototype,!(!e||!e.isReactComponent)}function Cg(e){if(typeof e=="function")return Ts(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Ka)return 11;if(e===Ya)return 14}return 2}function fn(e,t){var n=e.alternate;return n===null?(n=ft(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Gi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")Ts(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case $n:return Nn(n.children,i,l,t);case qa:o=8,i|=8;break;case Ro:return e=ft(12,n,t,i|2),e.elementType=Ro,e.lanes=l,e;case Do:return e=ft(13,n,t,i),e.elementType=Do,e.lanes=l,e;case Fo:return e=ft(19,n,t,i),e.elementType=Fo,e.lanes=l,e;case kd:return Fl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case xd:o=10;break e;case yd:o=9;break e;case Ka:o=11;break e;case Ya:o=14;break e;case Jt:o=16,r=null;break e}throw Error(A(130,e==null?e:typeof e,""))}return t=ft(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function Nn(e,t,n,r){return e=ft(7,e,r,t),e.lanes=n,e}function Fl(e,t,n,r){return e=ft(22,e,r,t),e.elementType=kd,e.lanes=n,e.stateNode={isHidden:!1},e}function xo(e,t,n){return e=ft(6,e,null,t),e.lanes=n,e}function yo(e,t,n){return t=ft(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function Ng(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Jl(0),this.expirationTimes=Jl(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Jl(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function Ls(e,t,n,r,i,l,o,a,u){return e=new Ng(e,t,n,a,u),t===1?(t=1,l===!0&&(t|=8)):t=0,l=ft(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},ps(l),e}function Eg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:Bn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function fp(e){if(!e)return hn;e=e._reactInternals;e:{if(Mn(e)!==e||e.tag!==1)throw Error(A(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Ye(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(A(171))}if(e.tag===1){var n=e.type;if(Ye(n))return ff(e,n,t)}return t}function pp(e,t,n,r,i,l,o,a,u){return e=Ls(n,r,!0,e,i,l,o,a,u),e.context=fp(null),n=e.current,r=Be(),i=dn(n),l=$t(r,i),l.callback=t??null,un(n,l,i),e.current.lanes=i,di(e,i,r),Ge(e,r),e}function Ol(e,t,n,r){var i=t.current,l=Be(),o=dn(i);return n=fp(n),t.context===null?t.context=n:t.pendingContext=n,t=$t(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=un(i,t,o),e!==null&&(St(e,i,o,l),Vi(e,i,o)),o}function bl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function oc(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function Is(e,t){oc(e,t),(e=e.alternate)&&oc(e,t)}function Tg(){return null}var hp=typeof reportError=="function"?reportError:function(e){console.error(e)};function zs(e){this._internalRoot=e}Bl.prototype.render=zs.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(A(409));Ol(e,t,null,null)};Bl.prototype.unmount=zs.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;zn(function(){Ol(null,e,null,null)}),t[Ut]=null}};function Bl(e){this._internalRoot=e}Bl.prototype.unstable_scheduleHydration=function(e){if(e){var t=Vd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<en.length&&t!==0&&t<en[n].priority;n++);en.splice(n,0,e),n===0&&Qd(e)}};function Ps(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function $l(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function ac(){}function Lg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=bl(o);l.call(c)}}var o=pp(t,r,e,0,null,!1,!1,"",ac);return e._reactRootContainer=o,e[Ut]=o.current,ei(e.nodeType===8?e.parentNode:e),zn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=bl(u);a.call(c)}}var u=Ls(e,0,!1,null,null,!1,!1,"",ac);return e._reactRootContainer=u,e[Ut]=u.current,ei(e.nodeType===8?e.parentNode:e),zn(function(){Ol(t,u,n,r)}),u}function Hl(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var u=bl(o);a.call(u)}}Ol(t,o,e,i)}else o=Lg(n,t,e,i,r);return bl(o)}Hd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=zr(t.pendingLanes);n!==0&&(Ja(t,n|1),Ge(t,ye()),!(ne&6)&&(fr=ye()+500,vn()))}break;case 13:zn(function(){var r=Vt(e,1);if(r!==null){var i=Be();St(r,e,1,i)}}),Is(e,1)}};Za=function(e){if(e.tag===13){var t=Vt(e,134217728);if(t!==null){var n=Be();St(t,e,134217728,n)}Is(e,134217728)}};Ud=function(e){if(e.tag===13){var t=dn(e),n=Vt(e,t);if(n!==null){var r=Be();St(n,e,t,r)}Is(e,t)}};Vd=function(){return ie};Wd=function(e,t){var n=ie;try{return ie=e,t()}finally{ie=n}};Ko=function(e,t,n){switch(t){case"input":if($o(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=zl(r);if(!i)throw Error(A(90));Sd(r),$o(r,i)}}}break;case"textarea":_d(e,n);break;case"select":t=n.value,t!=null&&Jn(e,!!n.multiple,t,!1)}};Id=Cs;zd=zn;var Ig={usingClientEntryPoint:!1,Events:[pi,Wn,zl,Td,Ld,Cs]},Nr={findFiberByHostInstance:bn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},zg={bundleType:Nr.bundleType,version:Nr.version,rendererPackageName:Nr.rendererPackageName,rendererConfig:Nr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Qt.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=Ad(e),e===null?null:e.stateNode},findFiberByHostInstance:Nr.findFiberByHostInstance||Tg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var Pi=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!Pi.isDisabled&&Pi.supportsFiber)try{El=Pi.inject(zg),Lt=Pi}catch{}}at.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=Ig;at.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!Ps(t))throw Error(A(200));return Eg(e,t,null,n)};at.createRoot=function(e,t){if(!Ps(e))throw Error(A(299));var n=!1,r="",i=hp;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=Ls(e,1,!1,null,null,n,!1,r,i),e[Ut]=t.current,ei(e.nodeType===8?e.parentNode:e),new zs(t)};at.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(A(188)):(e=Object.keys(e).join(","),Error(A(268,e)));return e=Ad(t),e=e===null?null:e.stateNode,e};at.flushSync=function(e){return zn(e)};at.hydrate=function(e,t,n){if(!$l(t))throw Error(A(200));return Hl(null,e,t,!0,n)};at.hydrateRoot=function(e,t,n){if(!Ps(e))throw Error(A(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=hp;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=pp(t,null,e,1,n??null,i,!1,l,o),e[Ut]=t.current,ei(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new Bl(t)};at.render=function(e,t,n){if(!$l(t))throw Error(A(200));return Hl(null,e,t,!1,n)};at.unmountComponentAtNode=function(e){if(!$l(e))throw Error(A(40));return e._reactRootContainer?(zn(function(){Hl(null,null,e,!1,function(){e._reactRootContainer=null,e[Ut]=null})}),!0):!1};at.unstable_batchedUpdates=Cs;at.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!$l(n))throw Error(A(200));if(e==null||e._reactInternals===void 0)throw Error(A(38));return Hl(e,t,n,!1,r)};at.version="18.3.1-next-f1338f8080-20240426";function mp(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(mp)}catch(e){console.error(e)}}mp(),hd.exports=at;var Pg=hd.exports,sc=Pg;Mo.createRoot=sc.createRoot,Mo.hydrateRoot=sc.hydrateRoot;const Mg=new Set(["user","human"]);function Ag(e){return e?Mg.has(e.toLowerCase()):!1}function gp(e){return Ag(e)?"You (Human)":e}const Rg="",Dg=({selection:e,onSelect:t,onRefresh:n})=>{const[r,i]=F.useState(null),[l,o]=F.useState(new Set(["all"])),[a,u]=F.useState(!0),[c,d]=F.useState(null),f=async()=>{try{const v=await fetch(`${Rg}/api/hierarchy`);if(!v.ok)throw new Error("Failed to fetch hierarchy");const x=await v.json();i(x),d(null)}catch(v){d(v instanceof Error?v.message:"Unknown error")}finally{u(!1)}};F.useEffect(()=>{f();const v=setInterval(f,5e3);return()=>clearInterval(v)},[]);const g=v=>{o(x=>{const b=new Set(x);return b.has(v)?b.delete(v):b.add(v),b})},p=v=>{var x;if(v.type==="root")t({type:"overview"});else if(v.type==="agent")t({type:"agent",agentId:v.id});else if(v.type==="thread"){const b=(x=r==null?void 0:r.root.children)==null?void 0:x.find(N=>{var S;return(S=N.children)==null?void 0:S.some(C=>C.id===v.id)});t({type:"thread",agentId:b==null?void 0:b.id,threadId:v.id})}},k=v=>v.type==="root"&&e.type==="overview"||v.type==="agent"&&e.type==="agent"&&e.agentId===v.id||v.type==="thread"&&e.threadId===v.id,w=v=>!v||v.length===0?null:s.jsx("span",{className:"badges",children:v.map((x,b)=>s.jsxs("span",{className:`badge badge-${x.type}`,title:`${x.count} ${x.type}`,children:[x.type==="pending"&&"⏳",x.type==="unread"&&"📬",x.type==="running"&&"▶️",x.count]},b))}),z=v=>{if(!v)return null;const x={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return s.jsx("span",{className:"status-indicator",style:{backgroundColor:x[v]||x.idle},title:v})},h=(v,x=0)=>{const b=l.has(v.id),N=v.children&&v.children.length>0,S=k(v);return s.jsxs("div",{className:"tree-node",children:[s.jsxs("div",{className:`tree-node-content ${S?"selected":""} ${v.type}`,style:{paddingLeft:`${x*16+8}px`},onClick:()=>p(v),children:[N&&s.jsx("span",{className:`expand-icon ${b?"expanded":""}`,onClick:C=>{C.stopPropagation(),g(v.id)},children:b?"▼":"▶"}),!N&&s.jsx("span",{className:"expand-icon-placeholder"}),v.type==="agent"&&z(v.status),s.jsx("span",{className:"node-label",children:v.type==="agent"?gp(v.id):v.label}),w(v.badges)]}),N&&b&&s.jsx("div",{className:"tree-children",children:v.children.map(C=>h(C,x+1))})]},v.id)};return a&&!r?s.jsx("div",{className:"hierarchy-tree loading",children:"Loading..."}):c?s.jsxs("div",{className:"hierarchy-tree error",children:[s.jsxs("p",{children:["Error: ",c]}),s.jsx("button",{onClick:f,children:"Retry"})]}):s.jsxs("div",{className:"hierarchy-tree",children:[s.jsxs("div",{className:"tree-header",children:[s.jsx("h3",{children:"Agents"}),s.jsx("button",{className:"refresh-btn",onClick:()=>{f(),n==null||n()},title:"Refresh",children:"↻"})]}),s.jsx("div",{className:"tree-content",children:r&&h(r.root)}),r&&s.jsx("div",{className:"tree-footer",children:s.jsxs("div",{className:"aggregate-stats",children:[s.jsxs("span",{title:"Total agents",children:[r.aggregate.total_agents," agents"]}),s.jsxs("span",{title:"Active",children:[r.aggregate.active_agents," active"]}),r.aggregate.pending_approvals>0&&s.jsxs("span",{className:"pending",title:"Pending approvals",children:[r.aggregate.pending_approvals," pending"]})]})})]})},Fg="_card_1d3of_1",Og="_compact_1d3of_9",Bg="_title_1d3of_13",$g="_metricsGrid_1d3of_20",Hg="_metricItem_1d3of_26",Ug="_metricLabel_1d3of_32",Vg="_metricValue_1d3of_39",Wg="_cost_1d3of_46",Qg="_averages_1d3of_50",qg="_averagesLabel_1d3of_61",Kg="_avgItem_1d3of_65",Yg="_compactRow_1d3of_72",Gg="_compactLabel_1d3of_80",Xg="_compactValue_1d3of_84",Jg="_loading_1d3of_91",Zg="_error_1d3of_97",ev="_errorText_1d3of_101",G={card:Fg,compact:Og,title:Bg,metricsGrid:$g,metricItem:Hg,metricLabel:Ug,metricValue:Vg,cost:Wg,averages:Qg,averagesLabel:qg,avgItem:Kg,compactRow:Yg,compactLabel:Gg,compactValue:Xg,loading:Jg,error:Zg,errorText:ev};function uc(e){return e<1e3?`${e}ms`:e<6e4?`${(e/1e3).toFixed(1)}s`:e<36e5?`${(e/6e4).toFixed(1)}m`:`${(e/36e5).toFixed(1)}h`}function Fn(e){return e.toLocaleString()}function ko(e){return e===0?"$0.00":e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`}function Na({scopeType:e,scopeId:t="",title:n,compact:r=!1}){const[i,l]=F.useState(null),[o,a]=F.useState(!0),[u,c]=F.useState(null),d=F.useCallback(async()=>{try{let g="/api/metrics";e!=="global"&&(g=`/api/metrics/${e}/${t}`);const p=await fetch(g);if(!p.ok)throw new Error(`Failed to fetch metrics: ${p.status}`);const k=await p.json();l(k),c(null)}catch(g){c(g instanceof Error?g.message:"Failed to load metrics")}finally{a(!1)}},[e,t]);if(F.useEffect(()=>{d();const g=setInterval(d,3e4);return()=>clearInterval(g)},[d]),o)return s.jsx("div",{className:`${G.card} ${r?G.compact:""}`,children:s.jsx("div",{className:G.loading,children:"Loading metrics..."})});if(u)return s.jsx("div",{className:`${G.card} ${r?G.compact:""} ${G.error}`,children:s.jsx("div",{className:G.errorText,children:u})});if(!i)return null;const f=n||(e==="global"?"Global Metrics":e==="agent"?`Agent: ${t}`:`Thread: ${t.slice(0,12)}...`);return r?s.jsx("div",{className:`${G.card} ${G.compact}`,children:s.jsxs("div",{className:G.compactRow,children:[s.jsx("span",{className:G.compactLabel,children:"Runs:"}),s.jsx("span",{className:G.compactValue,children:Fn(i.total_runs)}),s.jsx("span",{className:G.compactLabel,children:"Tokens:"}),s.jsx("span",{className:G.compactValue,children:Fn(i.total_tokens)}),s.jsx("span",{className:G.compactLabel,children:"Cost:"}),s.jsx("span",{className:G.compactValue,children:ko(i.total_cost)})]})}):s.jsxs("div",{className:G.card,children:[s.jsx("h3",{className:G.title,children:f}),s.jsxs("div",{className:G.metricsGrid,children:[s.jsxs("div",{className:G.metricItem,children:[s.jsx("span",{className:G.metricLabel,children:"Total Runs"}),s.jsx("span",{className:G.metricValue,children:Fn(i.total_runs)})]}),s.jsxs("div",{className:G.metricItem,children:[s.jsx("span",{className:G.metricLabel,children:"Total Tokens"}),s.jsx("span",{className:G.metricValue,children:Fn(i.total_tokens)})]}),s.jsxs("div",{className:G.metricItem,children:[s.jsx("span",{className:G.metricLabel,children:"Total Cost"}),s.jsx("span",{className:`${G.metricValue} ${G.cost}`,children:ko(i.total_cost)})]}),s.jsxs("div",{className:G.metricItem,children:[s.jsx("span",{className:G.metricLabel,children:"Total Duration"}),s.jsx("span",{className:G.metricValue,children:uc(i.total_duration_ms)})]}),s.jsxs("div",{className:G.metricItem,children:[s.jsx("span",{className:G.metricLabel,children:"Files Modified"}),s.jsx("span",{className:G.metricValue,children:Fn(i.total_files_modified)})]})]}),i.total_runs>0&&s.jsxs("div",{className:G.averages,children:[s.jsx("span",{className:G.averagesLabel,children:"Averages per run:"}),s.jsxs("span",{className:G.avgItem,children:[Fn(Math.round(i.avg_tokens_per_run))," tokens"]}),s.jsx("span",{className:G.avgItem,children:ko(i.avg_cost_per_run)}),s.jsx("span",{className:G.avgItem,children:uc(Math.round(i.avg_duration_per_run))})]})]})}const tv="_container_1q26w_1",nv="_title_1q26w_9",rv="_header_1q26w_16",iv="_metricLabel_1q26w_25",lv="_total_1q26w_31",ov="_chart_1q26w_37",av="_barContainer_1q26w_46",sv="_barWrapper_1q26w_55",uv="_bar_1q26w_46",cv="_barValue_1q26w_80",dv="_label_1q26w_89",fv="_loading_1q26w_98",pv="_error_1q26w_99",hv="_empty_1q26w_100",Ee={container:tv,title:nv,header:rv,metricLabel:iv,total:lv,chart:ov,barContainer:av,barWrapper:sv,bar:uv,barValue:cv,label:dv,loading:fv,error:pv,empty:hv};function _l({scopeType:e,scopeId:t,period:n="hour",limit:r=24,metric:i="cost",title:l}){const[o,a]=F.useState([]),[u,c]=F.useState(!0),[d,f]=F.useState(null);F.useEffect(()=>{const x=async()=>{try{c(!0);const N=await fetch(`/api/metrics/trends/${e}/${t}?period=${n}&limit=${r}`);if(!N.ok)throw new Error("Failed to fetch trends");const S=await N.json();a(S||[]),f(null)}catch(N){f(N instanceof Error?N.message:"Unknown error")}finally{c(!1)}};x();const b=setInterval(x,6e4);return()=>clearInterval(b)},[e,t,n,r]);const g=x=>{switch(i){case"cost":return x.cost;case"tokens":return x.tokens;case"duration":return x.duration_ms/1e3;case"runs":return x.runs;default:return x.cost}},p=x=>{switch(i){case"cost":return`$${x.toFixed(2)}`;case"tokens":return x>=1e3?`${(x/1e3).toFixed(1)}k`:x.toString();case"duration":return`${x.toFixed(1)}s`;case"runs":return x.toString();default:return x.toFixed(2)}},k=x=>{const b=new Date(x);return n==="minute"?b.toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}):n==="hour"?b.toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}):b.toLocaleDateString([],{month:"short",day:"numeric"})},w=()=>{switch(i){case"cost":return"Cost ($)";case"tokens":return"Tokens";case"duration":return"Duration (s)";case"runs":return"Runs";default:return""}};if(u&&o.length===0)return s.jsx("div",{className:Ee.container,children:s.jsx("div",{className:Ee.loading,children:"Loading trends..."})});if(d)return s.jsx("div",{className:Ee.container,children:s.jsx("div",{className:Ee.error,children:d})});if(o.length===0)return s.jsx("div",{className:Ee.container,children:s.jsx("div",{className:Ee.empty,children:"No data available"})});const z=o.map(g),h=Math.max(...z,1),v=z.reduce((x,b)=>x+b,0);return s.jsxs("div",{className:Ee.container,children:[l&&s.jsx("div",{className:Ee.title,children:l}),s.jsxs("div",{className:Ee.header,children:[s.jsx("span",{className:Ee.metricLabel,children:w()}),s.jsxs("span",{className:Ee.total,children:["Total: ",p(v)]})]}),s.jsx("div",{className:Ee.chart,children:o.map((x,b)=>{const N=g(x),S=N/h*100;return s.jsxs("div",{className:Ee.barContainer,children:[s.jsx("div",{className:Ee.barWrapper,children:s.jsx("div",{className:Ee.bar,style:{height:`${Math.max(S,2)}%`},title:`${k(x.period_start)}: ${p(N)}`,children:S>30&&s.jsx("span",{className:Ee.barValue,children:p(N)})})}),b%Math.ceil(o.length/6)===0&&s.jsx("span",{className:Ee.label,children:k(x.period_start)})]},x.period_start)})})]})}const et=({title:e,value:t,color:n="default",small:r})=>s.jsxs("div",{className:`stat-card stat-${n} ${r?"stat-small":""}`,children:[s.jsx("div",{className:"stat-value",children:t}),s.jsx("div",{className:"stat-title",children:e})]}),mv=e=>{if(e<1e3)return`${e}ms`;const t=e/1e3;if(t<60)return`${t.toFixed(1)}s`;const n=Math.floor(t/60),r=(t%60).toFixed(0);return`${n}m ${r}s`},gv=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,Mi=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}k`:e.toString(),vv=({agent:e,onClick:t})=>{var o,a,u,c,d;const n=((o=e.children)==null?void 0:o.length)||0,r=((u=(a=e.badges)==null?void 0:a.find(f=>f.type==="pending"))==null?void 0:u.count)||0,i=((d=(c=e.badges)==null?void 0:c.find(f=>f.type==="running"))==null?void 0:d.count)||0,l={active:"#22c55e",running:"#22c55e",pending:"#f59e0b",idle:"#6b7280",error:"#ef4444"};return s.jsxs("div",{className:"agent-card",onClick:t,children:[s.jsxs("div",{className:"agent-card-header",children:[s.jsx("span",{className:"agent-status-dot",style:{backgroundColor:l[e.status||"idle"]}}),s.jsx("span",{className:"agent-name",children:gp(e.id)})]}),s.jsxs("div",{className:"agent-card-stats",children:[s.jsxs("span",{className:"agent-stat",children:[s.jsx("span",{className:"agent-stat-value",children:n}),s.jsx("span",{className:"agent-stat-label",children:"threads"})]}),r>0&&s.jsxs("span",{className:"agent-stat pending",children:[s.jsx("span",{className:"agent-stat-value",children:r}),s.jsx("span",{className:"agent-stat-label",children:"pending"})]}),i>0&&s.jsxs("span",{className:"agent-stat running",children:[s.jsx("span",{className:"agent-stat-value",children:i}),s.jsx("span",{className:"agent-stat-label",children:"running"})]})]})]})},xv=({aggregate:e,agents:t,onSelectAgent:n})=>{const r=e.execution,i=r&&r.total_executions>0,l=i?Math.round(r.successful_executions/r.total_executions*100):0;return s.jsxs("div",{className:"all-agents-overview",children:[s.jsx("div",{className:"overview-header",children:s.jsx("h2",{children:"All Agents Overview"})}),s.jsxs("div",{className:"stats-row",children:[s.jsx(et,{title:"Total Agents",value:e.total_agents}),s.jsx(et,{title:"Active",value:e.active_agents,color:"green"}),s.jsx(et,{title:"Pending Approvals",value:e.pending_approvals,color:"orange"}),s.jsx(et,{title:"Total Threads",value:e.total_threads,color:"blue"})]}),s.jsxs("div",{className:"metrics-section",children:[s.jsx("h3",{children:"Usage Metrics (Today)"}),s.jsx(Na,{scopeType:"global",title:"Global Metrics"})]}),s.jsxs("div",{className:"trends-section",children:[s.jsx("h3",{children:"Usage Trends (Last 24 Hours)"}),s.jsxs("div",{className:"trends-grid",children:[s.jsx(_l,{scopeType:"global",scopeId:"",period:"hour",limit:24,metric:"cost",title:"Cost"}),s.jsx(_l,{scopeType:"global",scopeId:"",period:"hour",limit:24,metric:"tokens",title:"Tokens"})]})]}),i&&s.jsxs("div",{className:"execution-stats-section",children:[s.jsx("h3",{children:"Execution Statistics"}),s.jsxs("div",{className:"stats-row",children:[s.jsx(et,{title:"Total Executions",value:r.total_executions,color:"purple"}),s.jsx(et,{title:"Success Rate",value:`${l}%`,color:"green"}),s.jsx(et,{title:"Total Duration",value:mv(r.total_duration_ms)}),s.jsx(et,{title:"Total Cost",value:gv(r.total_cost),color:"orange"})]}),s.jsxs("div",{className:"stats-row token-stats",children:[s.jsx(et,{title:"Input Tokens",value:Mi(r.total_input_tokens),small:!0}),s.jsx(et,{title:"Output Tokens",value:Mi(r.total_output_tokens),small:!0}),s.jsx(et,{title:"Cache Read",value:Mi(r.total_cache_read_tokens),small:!0}),s.jsx(et,{title:"Cache Created",value:Mi(r.total_cache_create_tokens),small:!0}),s.jsx(et,{title:"Files Created",value:r.total_files_created,small:!0})]})]}),s.jsxs("div",{className:"agents-section",children:[s.jsx("h3",{children:"Agents"}),s.jsxs("div",{className:"agent-cards-grid",children:[t.map(o=>s.jsx(vv,{agent:o,onClick:()=>n(o.id)},o.id)),t.length===0&&s.jsx("div",{className:"no-agents",children:"No agents found. Start an agent to see it here."})]})]})]})},yv=({items:e})=>s.jsx("nav",{className:"breadcrumb",children:e.map((t,n)=>s.jsxs(Xt.Fragment,{children:[n>0&&s.jsx("span",{className:"breadcrumb-separator",children:"/"}),t.onClick?s.jsx("button",{className:"breadcrumb-link",onClick:t.onClick,children:t.label}):s.jsx("span",{className:"breadcrumb-current",children:t.label})]},n))}),At={plus:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),s.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),s.jsx("circle",{cx:"12",cy:"5",r:"2"}),s.jsx("path",{d:"M12 7v4"}),s.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),s.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),s.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),s.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),s.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:s.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),s.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:s.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("polyline",{points:"3 6 5 6 21 6"}),s.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:s.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:s.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),s.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},kv=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,u]=F.useState(!1),[c,d]=F.useState(""),[f,g]=F.useState(null),[p,k]=F.useState(""),[w,z]=F.useState(null),h=()=>{c.trim()&&(r(c.trim()),d(""),u(!1))},v=j=>{j.key==="Enter"&&!j.shiftKey?(j.preventDefault(),h()):j.key==="Escape"&&(u(!1),d(""))},x=(j,E)=>{E.stopPropagation(),g(j.id),k(j.title)},b=j=>{var E;p.trim()&&p.trim()!==((E=e.find(U=>U.id===j))==null?void 0:E.title)&&l(j,p.trim()),g(null),k("")},N=()=>{g(null),k("")},S=(j,E)=>{j.key==="Enter"?(j.preventDefault(),b(E)):j.key==="Escape"&&N()},C=(j,E)=>{E.stopPropagation(),z(j)},I=(j,E)=>{E.stopPropagation(),i(j),z(null)},R=j=>{j.stopPropagation(),z(null)},P=j=>{const E=new Date(j),V=new Date().getTime()-E.getTime(),W=Math.floor(V/6e4),K=Math.floor(V/36e5),le=Math.floor(V/864e5);return W<1?"now":W<60?`${W}m`:K<24?`${K}h`:le<7?`${le}d`:E.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return s.jsxs("div",{className:"thread-list",children:[s.jsxs("div",{className:"list-header",children:[s.jsx("h2",{children:"Conversations"}),s.jsx("button",{className:"new-thread-btn",onClick:()=>u(!0),title:"New conversation",children:At.plus})]}),a&&s.jsxs("div",{className:"new-thread-form",children:[s.jsx("input",{type:"text",value:c,onChange:j=>d(j.target.value),onKeyDown:v,placeholder:"Conversation title...",autoFocus:!0}),s.jsxs("div",{className:"form-actions",children:[s.jsx("button",{className:"cancel-btn",onClick:()=>u(!1),children:"Cancel"}),s.jsx("button",{className:"create-btn",onClick:h,children:"Create"})]})]}),s.jsx("div",{className:"thread-items",children:e.length===0?s.jsxs("div",{className:"empty-state",children:[s.jsx("div",{className:"empty-icon",children:At.hash}),s.jsx("p",{children:"No conversations yet"}),s.jsx("button",{className:"start-btn",onClick:()=>u(!0),children:"Start a conversation"})]}):e.map(j=>{const E=o.get(j.id)||0,U=j.id===t,V=f===j.id,W=w===j.id;return s.jsxs("div",{className:`thread-item ${U?"selected":""} ${E>0?"has-unread":""}`,onClick:()=>!V&&n(j.id),children:[s.jsx("div",{className:`status-dot ${j.status}`}),s.jsxs("div",{className:"thread-content",children:[s.jsx("div",{className:"thread-title-row",children:V?s.jsxs("div",{className:"edit-title-form",onClick:K=>K.stopPropagation(),children:[s.jsx("input",{type:"text",value:p,onChange:K=>k(K.target.value),onKeyDown:K=>S(K,j.id),autoFocus:!0}),s.jsx("button",{className:"edit-action save",onClick:()=>b(j.id),title:"Save",children:At.check}),s.jsx("button",{className:"edit-action cancel",onClick:N,title:"Cancel",children:At.x})]}):s.jsxs(s.Fragment,{children:[s.jsx("span",{className:"thread-title",children:j.title}),s.jsx("span",{className:"thread-time",children:P(j.updated_at)})]})}),s.jsxs("div",{className:"thread-meta",children:[j.target_agent&&s.jsxs("span",{className:"thread-agent",title:`Target: ${j.target_agent}`,children:[At.bot,j.target_agent]}),s.jsxs("span",{className:"thread-seq",children:["#",j.last_seq]})]})]}),!V&&!W&&s.jsxs("div",{className:"thread-actions",children:[s.jsx("button",{className:"action-btn edit",onClick:K=>x(j,K),title:"Rename",children:At.edit}),s.jsx("button",{className:"action-btn delete",onClick:K=>C(j.id,K),title:"Delete",children:At.trash})]}),W&&s.jsxs("div",{className:"delete-confirm",onClick:K=>K.stopPropagation(),children:[s.jsx("span",{className:"confirm-text",children:"Delete?"}),s.jsx("button",{className:"confirm-btn yes",onClick:K=>I(j.id,K),title:"Confirm delete",children:At.check}),s.jsx("button",{className:"confirm-btn no",onClick:R,title:"Cancel",children:At.x})]}),E>0&&!W&&s.jsx("span",{className:"unread-badge",children:E})]},j.id)})}),s.jsx("style",{children:`
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
      `})]})};function wv(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const Sv=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,bv=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,_v={};function cc(e,t){return(_v.jsx?bv:Sv).test(e)}const jv=/[ \t\n\f\r]/g;function Cv(e){return typeof e=="object"?e.type==="text"?dc(e.value):!1:dc(e)}function dc(e){return e.replace(jv,"")===""}class mi{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}mi.prototype.normal={};mi.prototype.property={};mi.prototype.space=void 0;function vp(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new mi(n,r,t)}function Ea(e){return e.toLowerCase()}class Je{constructor(t,n){this.attribute=n,this.property=t}}Je.prototype.attribute="";Je.prototype.booleanish=!1;Je.prototype.boolean=!1;Je.prototype.commaOrSpaceSeparated=!1;Je.prototype.commaSeparated=!1;Je.prototype.defined=!1;Je.prototype.mustUseProperty=!1;Je.prototype.number=!1;Je.prototype.overloadedBoolean=!1;Je.prototype.property="";Je.prototype.spaceSeparated=!1;Je.prototype.space=void 0;let Nv=0;const Y=An(),ke=An(),Ta=An(),D=An(),ae=An(),ir=An(),tt=An();function An(){return 2**++Nv}const La=Object.freeze(Object.defineProperty({__proto__:null,boolean:Y,booleanish:ke,commaOrSpaceSeparated:tt,commaSeparated:ir,number:D,overloadedBoolean:Ta,spaceSeparated:ae},Symbol.toStringTag,{value:"Module"})),wo=Object.keys(La);class Ms extends Je{constructor(t,n,r,i){let l=-1;if(super(t,n),fc(this,"space",i),typeof r=="number")for(;++l<wo.length;){const o=wo[l];fc(this,wo[l],(r&La[o])===La[o])}}}Ms.prototype.defined=!0;function fc(e,t,n){n&&(e[t]=n)}function gr(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new Ms(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[Ea(r)]=r,n[Ea(l.attribute)]=r}return new mi(t,n,e.space)}const xp=gr({properties:{ariaActiveDescendant:null,ariaAtomic:ke,ariaAutoComplete:null,ariaBusy:ke,ariaChecked:ke,ariaColCount:D,ariaColIndex:D,ariaColSpan:D,ariaControls:ae,ariaCurrent:null,ariaDescribedBy:ae,ariaDetails:null,ariaDisabled:ke,ariaDropEffect:ae,ariaErrorMessage:null,ariaExpanded:ke,ariaFlowTo:ae,ariaGrabbed:ke,ariaHasPopup:null,ariaHidden:ke,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:ae,ariaLevel:D,ariaLive:null,ariaModal:ke,ariaMultiLine:ke,ariaMultiSelectable:ke,ariaOrientation:null,ariaOwns:ae,ariaPlaceholder:null,ariaPosInSet:D,ariaPressed:ke,ariaReadOnly:ke,ariaRelevant:null,ariaRequired:ke,ariaRoleDescription:ae,ariaRowCount:D,ariaRowIndex:D,ariaRowSpan:D,ariaSelected:ke,ariaSetSize:D,ariaSort:null,ariaValueMax:D,ariaValueMin:D,ariaValueNow:D,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function yp(e,t){return t in e?e[t]:t}function kp(e,t){return yp(e,t.toLowerCase())}const Ev=gr({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:ir,acceptCharset:ae,accessKey:ae,action:null,allow:null,allowFullScreen:Y,allowPaymentRequest:Y,allowUserMedia:Y,alt:null,as:null,async:Y,autoCapitalize:null,autoComplete:ae,autoFocus:Y,autoPlay:Y,blocking:ae,capture:null,charSet:null,checked:Y,cite:null,className:ae,cols:D,colSpan:null,content:null,contentEditable:ke,controls:Y,controlsList:ae,coords:D|ir,crossOrigin:null,data:null,dateTime:null,decoding:null,default:Y,defer:Y,dir:null,dirName:null,disabled:Y,download:Ta,draggable:ke,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:Y,formTarget:null,headers:ae,height:D,hidden:Ta,high:D,href:null,hrefLang:null,htmlFor:ae,httpEquiv:ae,id:null,imageSizes:null,imageSrcSet:null,inert:Y,inputMode:null,integrity:null,is:null,isMap:Y,itemId:null,itemProp:ae,itemRef:ae,itemScope:Y,itemType:ae,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:Y,low:D,manifest:null,max:null,maxLength:D,media:null,method:null,min:null,minLength:D,multiple:Y,muted:Y,name:null,nonce:null,noModule:Y,noValidate:Y,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:Y,optimum:D,pattern:null,ping:ae,placeholder:null,playsInline:Y,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:Y,referrerPolicy:null,rel:ae,required:Y,reversed:Y,rows:D,rowSpan:D,sandbox:ae,scope:null,scoped:Y,seamless:Y,selected:Y,shadowRootClonable:Y,shadowRootDelegatesFocus:Y,shadowRootMode:null,shape:null,size:D,sizes:null,slot:null,span:D,spellCheck:ke,src:null,srcDoc:null,srcLang:null,srcSet:null,start:D,step:null,style:null,tabIndex:D,target:null,title:null,translate:null,type:null,typeMustMatch:Y,useMap:null,value:ke,width:D,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:ae,axis:null,background:null,bgColor:null,border:D,borderColor:null,bottomMargin:D,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:Y,declare:Y,event:null,face:null,frame:null,frameBorder:null,hSpace:D,leftMargin:D,link:null,longDesc:null,lowSrc:null,marginHeight:D,marginWidth:D,noResize:Y,noHref:Y,noShade:Y,noWrap:Y,object:null,profile:null,prompt:null,rev:null,rightMargin:D,rules:null,scheme:null,scrolling:ke,standby:null,summary:null,text:null,topMargin:D,valueType:null,version:null,vAlign:null,vLink:null,vSpace:D,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:Y,disableRemotePlayback:Y,prefix:null,property:null,results:D,security:null,unselectable:null},space:"html",transform:kp}),Tv=gr({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:tt,accentHeight:D,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:D,amplitude:D,arabicForm:null,ascent:D,attributeName:null,attributeType:null,azimuth:D,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:D,by:null,calcMode:null,capHeight:D,className:ae,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:D,diffuseConstant:D,direction:null,display:null,dur:null,divisor:D,dominantBaseline:null,download:Y,dx:null,dy:null,edgeMode:null,editable:null,elevation:D,enableBackground:null,end:null,event:null,exponent:D,externalResourcesRequired:null,fill:null,fillOpacity:D,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:ir,g2:ir,glyphName:ir,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:D,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:D,horizOriginX:D,horizOriginY:D,id:null,ideographic:D,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:D,k:D,k1:D,k2:D,k3:D,k4:D,kernelMatrix:tt,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:D,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:D,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:D,overlineThickness:D,paintOrder:null,panose1:null,path:null,pathLength:D,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:ae,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:D,pointsAtY:D,pointsAtZ:D,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:tt,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:tt,rev:tt,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:tt,requiredFeatures:tt,requiredFonts:tt,requiredFormats:tt,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:D,specularExponent:D,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:D,strikethroughThickness:D,string:null,stroke:null,strokeDashArray:tt,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:D,strokeOpacity:D,strokeWidth:null,style:null,surfaceScale:D,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:tt,tabIndex:D,tableValues:null,target:null,targetX:D,targetY:D,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:tt,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:D,underlineThickness:D,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:D,values:null,vAlphabetic:D,vMathematical:D,vectorEffect:null,vHanging:D,vIdeographic:D,version:null,vertAdvY:D,vertOriginX:D,vertOriginY:D,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:D,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:yp}),wp=gr({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),Sp=gr({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:kp}),bp=gr({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),Lv={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Iv=/[A-Z]/g,pc=/-[a-z]/g,zv=/^data[-\w.:]+$/i;function Pv(e,t){const n=Ea(t);let r=t,i=Je;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&zv.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(pc,Av);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!pc.test(l)){let o=l.replace(Iv,Mv);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=Ms}return new i(r,t)}function Mv(e){return"-"+e.toLowerCase()}function Av(e){return e.charAt(1).toUpperCase()}const Rv=vp([xp,Ev,wp,Sp,bp],"html"),As=vp([xp,Tv,wp,Sp,bp],"svg");function Dv(e){return e.join(" ").trim()}var Rs={},hc=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Fv=/\n/g,Ov=/^\s*/,Bv=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,$v=/^:\s*/,Hv=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Uv=/^[;\s]*/,Vv=/^\s+|\s+$/g,Wv=`
`,mc="/",gc="*",Sn="",Qv="comment",qv="declaration";function Kv(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(k){var w=k.match(Fv);w&&(n+=w.length);var z=k.lastIndexOf(Wv);r=~z?k.length-z:r+k.length}function l(){var k={line:n,column:r};return function(w){return w.position=new o(k),c(),w}}function o(k){this.start=k,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(k){var w=new Error(t.source+":"+n+":"+r+": "+k);if(w.reason=k,w.filename=t.source,w.line=n,w.column=r,w.source=e,!t.silent)throw w}function u(k){var w=k.exec(e);if(w){var z=w[0];return i(z),e=e.slice(z.length),w}}function c(){u(Ov)}function d(k){var w;for(k=k||[];w=f();)w!==!1&&k.push(w);return k}function f(){var k=l();if(!(mc!=e.charAt(0)||gc!=e.charAt(1))){for(var w=2;Sn!=e.charAt(w)&&(gc!=e.charAt(w)||mc!=e.charAt(w+1));)++w;if(w+=2,Sn===e.charAt(w-1))return a("End of comment missing");var z=e.slice(2,w-2);return r+=2,i(z),e=e.slice(w),r+=2,k({type:Qv,comment:z})}}function g(){var k=l(),w=u(Bv);if(w){if(f(),!u($v))return a("property missing ':'");var z=u(Hv),h=k({type:qv,property:vc(w[0].replace(hc,Sn)),value:z?vc(z[0].replace(hc,Sn)):Sn});return u(Uv),h}}function p(){var k=[];d(k);for(var w;w=g();)w!==!1&&(k.push(w),d(k));return k}return c(),p()}function vc(e){return e?e.replace(Vv,Sn):Sn}var Yv=Kv,Gv=Zi&&Zi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(Rs,"__esModule",{value:!0});Rs.default=Jv;const Xv=Gv(Yv);function Jv(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Xv.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Ul={};Object.defineProperty(Ul,"__esModule",{value:!0});Ul.camelCase=void 0;var Zv=/^--[a-zA-Z0-9_-]+$/,ex=/-([a-z])/g,tx=/^[^-]+$/,nx=/^-(webkit|moz|ms|o|khtml)-/,rx=/^-(ms)-/,ix=function(e){return!e||tx.test(e)||Zv.test(e)},lx=function(e,t){return t.toUpperCase()},xc=function(e,t){return"".concat(t,"-")},ox=function(e,t){return t===void 0&&(t={}),ix(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(rx,xc):e=e.replace(nx,xc),e.replace(ex,lx))};Ul.camelCase=ox;var ax=Zi&&Zi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},sx=ax(Rs),ux=Ul;function Ia(e,t){var n={};return!e||typeof e!="string"||(0,sx.default)(e,function(r,i){r&&i&&(n[(0,ux.camelCase)(r,t)]=i)}),n}Ia.default=Ia;var cx=Ia;const dx=Oa(cx),_p=jp("end"),Ds=jp("start");function jp(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function fx(e){const t=Ds(e),n=_p(e);if(t&&n)return{start:t,end:n}}function Ur(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?yc(e.position):"start"in e||"end"in e?yc(e):"line"in e||"column"in e?za(e):""}function za(e){return kc(e&&e.line)+":"+kc(e&&e.column)}function yc(e){return za(e&&e.start)+"-"+za(e&&e.end)}function kc(e){return e&&typeof e=="number"?e:1}class De extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const u=r.indexOf(":");u===-1?l.ruleId=r:(l.source=r.slice(0,u),l.ruleId=r.slice(u+1))}if(!l.place&&l.ancestors&&l.ancestors){const u=l.ancestors[l.ancestors.length-1];u&&(l.place=u.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Ur(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}De.prototype.file="";De.prototype.name="";De.prototype.reason="";De.prototype.message="";De.prototype.stack="";De.prototype.column=void 0;De.prototype.line=void 0;De.prototype.ancestors=void 0;De.prototype.cause=void 0;De.prototype.fatal=void 0;De.prototype.place=void 0;De.prototype.ruleId=void 0;De.prototype.source=void 0;const Fs={}.hasOwnProperty,px=new Map,hx=/[A-Z]/g,mx=new Set(["table","tbody","thead","tfoot","tr"]),gx=new Set(["td","th"]),Cp="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function vx(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=jx(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=_x(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?As:Rv,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=Np(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function Np(e,t,n){if(t.type==="element")return xx(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return yx(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return wx(e,t,n);if(t.type==="mdxjsEsm")return kx(e,t);if(t.type==="root")return Sx(e,t,n);if(t.type==="text")return bx(e,t)}function xx(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=As,e.schema=i),e.ancestors.push(t);const l=Tp(e,t.tagName,!1),o=Cx(e,t);let a=Bs(e,t);return mx.has(t.tagName)&&(a=a.filter(function(u){return typeof u=="string"?!Cv(u):!0})),Ep(e,o,l,t),Os(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function yx(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}ui(e,t.position)}function kx(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);ui(e,t.position)}function wx(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=As,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:Tp(e,t.name,!0),o=Nx(e,t),a=Bs(e,t);return Ep(e,o,l,t),Os(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function Sx(e,t,n){const r={};return Os(r,Bs(e,t)),e.create(t,e.Fragment,r,n)}function bx(e,t){return t.value}function Ep(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Os(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function _x(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function jx(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),u=Ds(r);return t(i,l,o,a,{columnNumber:u?u.column-1:void 0,fileName:e,lineNumber:u?u.line:void 0},void 0)}}function Cx(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Fs.call(t.properties,i)){const l=Ex(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&gx.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function Nx(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else ui(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else ui(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Bs(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:px;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const u=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(u){const c=i.get(u)||0;o=u+"-"+c,i.set(u,c+1)}}const a=Np(e,l,o);a!==void 0&&n.push(a)}return n}function Ex(e,t,n){const r=Pv(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?wv(n):Dv(n)),r.property==="style"){let i=typeof n=="object"?n:Tx(e,String(n));return e.stylePropertyNameCase==="css"&&(i=Lx(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?Lv[r.property]||r.property:r.attribute,n]}}function Tx(e,t){try{return dx(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new De("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=Cp+"#cannot-parse-style-attribute",i}}function Tp(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=cc(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=cc(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Fs.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);ui(e)}function ui(e,t){const n=new De("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=Cp+"#cannot-handle-mdx-estrees-without-createevaluater",n}function Lx(e){const t={};let n;for(n in e)Fs.call(e,n)&&(t[Ix(n)]=e[n]);return t}function Ix(e){let t=e.replace(hx,zx);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function zx(e){return"-"+e.toLowerCase()}const So={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},Px={};function Mx(e,t){const n=Px,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return Lp(e,r,i)}function Lp(e,t,n){if(Ax(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return wc(e.children,t,n)}return Array.isArray(e)?wc(e,t,n):""}function wc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=Lp(e[i],t,n);return r.join("")}function Ax(e){return!!(e&&typeof e=="object")}const Sc=document.createElement("i");function $s(e){const t="&"+e+";";Sc.innerHTML=t;const n=Sc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function zt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function dt(e,t){return e.length>0?(zt(e,e.length,0,t),e):t}const bc={}.hasOwnProperty;function Rx(e){const t={};let n=-1;for(;++n<e.length;)Dx(t,e[n]);return t}function Dx(e,t){let n;for(n in t){const i=(bc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){bc.call(i,o)||(i[o]=[]);const a=l[o];Fx(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Fx(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);zt(e,0,0,r)}function Ip(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function lr(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const Tt=xn(/[A-Za-z]/),it=xn(/[\dA-Za-z]/),Ox=xn(/[#-'*+\--9=?A-Z^-~]/);function Pa(e){return e!==null&&(e<32||e===127)}const Ma=xn(/\d/),Bx=xn(/[\dA-Fa-f]/),$x=xn(/[!-/:-@[-`{-~]/);function Q(e){return e!==null&&e<-2}function Xe(e){return e!==null&&(e<0||e===32)}function re(e){return e===-2||e===-1||e===32}const Hx=xn(new RegExp("\\p{P}|\\p{S}","u")),Ux=xn(/\s/);function xn(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function vr(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&it(e.charCodeAt(n+1))&&it(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function ue(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(u){return re(u)?(e.enter(n),a(u)):t(u)}function a(u){return re(u)&&l++<i?(e.consume(u),a):(e.exit(n),t(u))}}const Vx={tokenize:Wx};function Wx(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),ue(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const u=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=u),n=u,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return Q(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const Qx={tokenize:qx},_c={tokenize:Kx};function qx(e){const t=this,n=[];let r=0,i,l,o;return a;function a(x){if(r<n.length){const b=n[r];return t.containerState=b[1],e.attempt(b[0].continuation,u,c)(x)}return c(x)}function u(x){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&v();const b=t.events.length;let N=b,S;for(;N--;)if(t.events[N][0]==="exit"&&t.events[N][1].type==="chunkFlow"){S=t.events[N][1].end;break}h(r);let C=b;for(;C<t.events.length;)t.events[C][1].end={...S},C++;return zt(t.events,N+1,0,t.events.slice(b)),t.events.length=C,c(x)}return a(x)}function c(x){if(r===n.length){if(!i)return g(x);if(i.currentConstruct&&i.currentConstruct.concrete)return k(x);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(_c,d,f)(x)}function d(x){return i&&v(),h(r),g(x)}function f(x){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,k(x)}function g(x){return t.containerState={},e.attempt(_c,p,k)(x)}function p(x){return r++,n.push([t.currentConstruct,t.containerState]),g(x)}function k(x){if(x===null){i&&v(),h(0),e.consume(x);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),w(x)}function w(x){if(x===null){z(e.exit("chunkFlow"),!0),h(0),e.consume(x);return}return Q(x)?(e.consume(x),z(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(x),w)}function z(x,b){const N=t.sliceStream(x);if(b&&N.push(null),x.previous=l,l&&(l.next=x),l=x,i.defineSkip(x.start),i.write(N),t.parser.lazy[x.start.line]){let S=i.events.length;for(;S--;)if(i.events[S][1].start.offset<o&&(!i.events[S][1].end||i.events[S][1].end.offset>o))return;const C=t.events.length;let I=C,R,P;for(;I--;)if(t.events[I][0]==="exit"&&t.events[I][1].type==="chunkFlow"){if(R){P=t.events[I][1].end;break}R=!0}for(h(r),S=C;S<t.events.length;)t.events[S][1].end={...P},S++;zt(t.events,I+1,0,t.events.slice(C)),t.events.length=S}}function h(x){let b=n.length;for(;b-- >x;){const N=n[b];t.containerState=N[1],N[0].exit.call(t,e)}n.length=x}function v(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Kx(e,t,n){return ue(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function jc(e){if(e===null||Xe(e)||Ux(e))return 1;if(Hx(e))return 2}function Hs(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const Aa={name:"attention",resolveAll:Yx,tokenize:Gx};function Yx(e,t){let n=-1,r,i,l,o,a,u,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;u=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const f={...e[r][1].end},g={...e[n][1].start};Cc(f,-u),Cc(g,u),o={type:u>1?"strongSequence":"emphasisSequence",start:f,end:{...e[r][1].end}},a={type:u>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:g},l={type:u>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:u>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=dt(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=dt(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=dt(c,Hs(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=dt(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=dt(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,zt(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Gx(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=jc(r);let l;return o;function o(u){return l=u,e.enter("attentionSequence"),a(u)}function a(u){if(u===l)return e.consume(u),a;const c=e.exit("attentionSequence"),d=jc(u),f=!d||d===2&&i||n.includes(u),g=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?f:f&&(i||!g)),c._close=!!(l===42?g:g&&(d||!f)),t(u)}}function Cc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Xx={name:"autolink",tokenize:Jx};function Jx(e,t,n){let r=0;return i;function i(p){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(p){return Tt(p)?(e.consume(p),o):p===64?n(p):c(p)}function o(p){return p===43||p===45||p===46||it(p)?(r=1,a(p)):c(p)}function a(p){return p===58?(e.consume(p),r=0,u):(p===43||p===45||p===46||it(p))&&r++<32?(e.consume(p),a):(r=0,c(p))}function u(p){return p===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):p===null||p===32||p===60||Pa(p)?n(p):(e.consume(p),u)}function c(p){return p===64?(e.consume(p),d):Ox(p)?(e.consume(p),c):n(p)}function d(p){return it(p)?f(p):n(p)}function f(p){return p===46?(e.consume(p),r=0,d):p===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):g(p)}function g(p){if((p===45||it(p))&&r++<63){const k=p===45?g:f;return e.consume(p),k}return n(p)}}const Vl={partial:!0,tokenize:Zx};function Zx(e,t,n){return r;function r(l){return re(l)?ue(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||Q(l)?t(l):n(l)}}const zp={continuation:{tokenize:ty},exit:ny,name:"blockQuote",tokenize:ey};function ey(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return re(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function ty(e,t,n){const r=this;return i;function i(o){return re(o)?ue(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(zp,t,n)(o)}}function ny(e){e.exit("blockQuote")}const Pp={name:"characterEscape",tokenize:ry};function ry(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return $x(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const Mp={name:"characterReference",tokenize:iy};function iy(e,t,n){const r=this;let i=0,l,o;return a;function a(f){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),u}function u(f){return f===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(f),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=it,d(f))}function c(f){return f===88||f===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(f),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=Bx,d):(e.enter("characterReferenceValue"),l=7,o=Ma,d(f))}function d(f){if(f===59&&i){const g=e.exit("characterReferenceValue");return o===it&&!$s(r.sliceSerialize(g))?n(f):(e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(f)&&i++<l?(e.consume(f),d):n(f)}}const Nc={partial:!0,tokenize:oy},Ec={concrete:!0,name:"codeFenced",tokenize:ly};function ly(e,t,n){const r=this,i={partial:!0,tokenize:N};let l=0,o=0,a;return u;function u(S){return c(S)}function c(S){const C=r.events[r.events.length-1];return l=C&&C[1].type==="linePrefix"?C[2].sliceSerialize(C[1],!0).length:0,a=S,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(S)}function d(S){return S===a?(o++,e.consume(S),d):o<3?n(S):(e.exit("codeFencedFenceSequence"),re(S)?ue(e,f,"whitespace")(S):f(S))}function f(S){return S===null||Q(S)?(e.exit("codeFencedFence"),r.interrupt?t(S):e.check(Nc,w,b)(S)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),g(S))}function g(S){return S===null||Q(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),f(S)):re(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),ue(e,p,"whitespace")(S)):S===96&&S===a?n(S):(e.consume(S),g)}function p(S){return S===null||Q(S)?f(S):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),k(S))}function k(S){return S===null||Q(S)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),f(S)):S===96&&S===a?n(S):(e.consume(S),k)}function w(S){return e.attempt(i,b,z)(S)}function z(S){return e.enter("lineEnding"),e.consume(S),e.exit("lineEnding"),h}function h(S){return l>0&&re(S)?ue(e,v,"linePrefix",l+1)(S):v(S)}function v(S){return S===null||Q(S)?e.check(Nc,w,b)(S):(e.enter("codeFlowValue"),x(S))}function x(S){return S===null||Q(S)?(e.exit("codeFlowValue"),v(S)):(e.consume(S),x)}function b(S){return e.exit("codeFenced"),t(S)}function N(S,C,I){let R=0;return P;function P(W){return S.enter("lineEnding"),S.consume(W),S.exit("lineEnding"),j}function j(W){return S.enter("codeFencedFence"),re(W)?ue(S,E,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(W):E(W)}function E(W){return W===a?(S.enter("codeFencedFenceSequence"),U(W)):I(W)}function U(W){return W===a?(R++,S.consume(W),U):R>=o?(S.exit("codeFencedFenceSequence"),re(W)?ue(S,V,"whitespace")(W):V(W)):I(W)}function V(W){return W===null||Q(W)?(S.exit("codeFencedFence"),C(W)):I(W)}}}function oy(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const bo={name:"codeIndented",tokenize:sy},ay={partial:!0,tokenize:uy};function sy(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),ue(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?u(c):Q(c)?e.attempt(ay,o,u)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||Q(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function u(c){return e.exit("codeIndented"),t(c)}}function uy(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):Q(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):ue(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):Q(o)?i(o):n(o)}}const cy={name:"codeText",previous:fy,resolve:dy,tokenize:py};function dy(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function fy(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function py(e,t,n){let r=0,i,l;return o;function o(f){return e.enter("codeText"),e.enter("codeTextSequence"),a(f)}function a(f){return f===96?(e.consume(f),r++,a):(e.exit("codeTextSequence"),u(f))}function u(f){return f===null?n(f):f===32?(e.enter("space"),e.consume(f),e.exit("space"),u):f===96?(l=e.enter("codeTextSequence"),i=0,d(f)):Q(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),u):(e.enter("codeTextData"),c(f))}function c(f){return f===null||f===32||f===96||Q(f)?(e.exit("codeTextData"),u(f)):(e.consume(f),c)}function d(f){return f===96?(e.consume(f),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(f)):(l.type="codeTextData",c(f))}}class hy{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&Er(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),Er(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),Er(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);Er(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);Er(this.left,n.reverse())}}}function Er(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function Ap(e){const t={};let n=-1,r,i,l,o,a,u,c;const d=new hy(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(u=r[1]._tokenizer.events,l=0,l<u.length&&u[l][1].type==="lineEndingBlank"&&(l+=2),l<u.length&&u[l][1].type==="content"))for(;++l<u.length&&u[l][1].type!=="content";)u[l][1].type==="chunkText"&&(u[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,my(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return zt(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function my(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,u=[],c={};let d,f,g=-1,p=n,k=0,w=0;const z=[w];for(;p;){for(;e.get(++i)[1]!==p;);l.push(i),p._tokenizer||(d=r.sliceStream(p),p.next||d.push(null),f&&o.defineSkip(p.start),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),f=p,p=p.next}for(p=n;++g<a.length;)a[g][0]==="exit"&&a[g-1][0]==="enter"&&a[g][1].type===a[g-1][1].type&&a[g][1].start.line!==a[g][1].end.line&&(w=g+1,z.push(w),p._tokenizer=void 0,p.previous=void 0,p=p.next);for(o.events=[],p?(p._tokenizer=void 0,p.previous=void 0):z.pop(),g=z.length;g--;){const h=a.slice(z[g],z[g+1]),v=l.pop();u.push([v,v+h.length-1]),e.splice(v,2,h)}for(u.reverse(),g=-1;++g<u.length;)c[k+u[g][0]]=k+u[g][1],k+=u[g][1]-u[g][0]-1;return c}const gy={resolve:xy,tokenize:yy},vy={partial:!0,tokenize:ky};function xy(e){return Ap(e),e}function yy(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):Q(a)?e.check(vy,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function ky(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),ue(e,l,"linePrefix")}function l(o){if(o===null||Q(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function Rp(e,t,n,r,i,l,o,a,u){const c=u||Number.POSITIVE_INFINITY;let d=0;return f;function f(h){return h===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(h),e.exit(l),g):h===null||h===32||h===41||Pa(h)?n(h):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),w(h))}function g(h){return h===62?(e.enter(l),e.consume(h),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),p(h))}function p(h){return h===62?(e.exit("chunkString"),e.exit(a),g(h)):h===null||h===60||Q(h)?n(h):(e.consume(h),h===92?k:p)}function k(h){return h===60||h===62||h===92?(e.consume(h),p):p(h)}function w(h){return!d&&(h===null||h===41||Xe(h))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(h)):d<c&&h===40?(e.consume(h),d++,w):h===41?(e.consume(h),d--,w):h===null||h===32||h===40||Pa(h)?n(h):(e.consume(h),h===92?z:w)}function z(h){return h===40||h===41||h===92?(e.consume(h),w):w(h)}}function Dp(e,t,n,r,i,l){const o=this;let a=0,u;return c;function c(p){return e.enter(r),e.enter(i),e.consume(p),e.exit(i),e.enter(l),d}function d(p){return a>999||p===null||p===91||p===93&&!u||p===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(p):p===93?(e.exit(l),e.enter(i),e.consume(p),e.exit(i),e.exit(r),t):Q(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),f(p))}function f(p){return p===null||p===91||p===93||Q(p)||a++>999?(e.exit("chunkString"),d(p)):(e.consume(p),u||(u=!re(p)),p===92?g:f)}function g(p){return p===91||p===92||p===93?(e.consume(p),a++,f):f(p)}}function Fp(e,t,n,r,i,l){let o;return a;function a(g){return g===34||g===39||g===40?(e.enter(r),e.enter(i),e.consume(g),e.exit(i),o=g===40?41:g,u):n(g)}function u(g){return g===o?(e.enter(i),e.consume(g),e.exit(i),e.exit(r),t):(e.enter(l),c(g))}function c(g){return g===o?(e.exit(l),u(o)):g===null?n(g):Q(g)?(e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),ue(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(g))}function d(g){return g===o||g===null||Q(g)?(e.exit("chunkString"),c(g)):(e.consume(g),g===92?f:d)}function f(g){return g===o||g===92?(e.consume(g),d):d(g)}}function Vr(e,t){let n;return r;function r(i){return Q(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):re(i)?ue(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const wy={name:"definition",tokenize:by},Sy={partial:!0,tokenize:_y};function by(e,t,n){const r=this;let i;return l;function l(p){return e.enter("definition"),o(p)}function o(p){return Dp.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(p)}function a(p){return i=lr(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),p===58?(e.enter("definitionMarker"),e.consume(p),e.exit("definitionMarker"),u):n(p)}function u(p){return Xe(p)?Vr(e,c)(p):c(p)}function c(p){return Rp(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(p)}function d(p){return e.attempt(Sy,f,f)(p)}function f(p){return re(p)?ue(e,g,"whitespace")(p):g(p)}function g(p){return p===null||Q(p)?(e.exit("definition"),r.parser.defined.push(i),t(p)):n(p)}}function _y(e,t,n){return r;function r(a){return Xe(a)?Vr(e,i)(a):n(a)}function i(a){return Fp(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return re(a)?ue(e,o,"whitespace")(a):o(a)}function o(a){return a===null||Q(a)?t(a):n(a)}}const jy={name:"hardBreakEscape",tokenize:Cy};function Cy(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return Q(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const Ny={name:"headingAtx",resolve:Ey,tokenize:Ty};function Ey(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},zt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function Ty(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||Xe(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),u(d)):d===null||Q(d)?(e.exit("atxHeading"),t(d)):re(d)?ue(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function u(d){return d===35?(e.consume(d),u):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||Xe(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const Ly=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],Tc=["pre","script","style","textarea"],Iy={concrete:!0,name:"htmlFlow",resolveTo:My,tokenize:Ay},zy={partial:!0,tokenize:Dy},Py={partial:!0,tokenize:Ry};function My(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function Ay(e,t,n){const r=this;let i,l,o,a,u;return c;function c(y){return d(y)}function d(y){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(y),f}function f(y){return y===33?(e.consume(y),g):y===47?(e.consume(y),l=!0,w):y===63?(e.consume(y),i=3,r.interrupt?t:m):Tt(y)?(e.consume(y),o=String.fromCharCode(y),z):n(y)}function g(y){return y===45?(e.consume(y),i=2,p):y===91?(e.consume(y),i=5,a=0,k):Tt(y)?(e.consume(y),i=4,r.interrupt?t:m):n(y)}function p(y){return y===45?(e.consume(y),r.interrupt?t:m):n(y)}function k(y){const X="CDATA[";return y===X.charCodeAt(a++)?(e.consume(y),a===X.length?r.interrupt?t:E:k):n(y)}function w(y){return Tt(y)?(e.consume(y),o=String.fromCharCode(y),z):n(y)}function z(y){if(y===null||y===47||y===62||Xe(y)){const X=y===47,pe=o.toLowerCase();return!X&&!l&&Tc.includes(pe)?(i=1,r.interrupt?t(y):E(y)):Ly.includes(o.toLowerCase())?(i=6,X?(e.consume(y),h):r.interrupt?t(y):E(y)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(y):l?v(y):x(y))}return y===45||it(y)?(e.consume(y),o+=String.fromCharCode(y),z):n(y)}function h(y){return y===62?(e.consume(y),r.interrupt?t:E):n(y)}function v(y){return re(y)?(e.consume(y),v):P(y)}function x(y){return y===47?(e.consume(y),P):y===58||y===95||Tt(y)?(e.consume(y),b):re(y)?(e.consume(y),x):P(y)}function b(y){return y===45||y===46||y===58||y===95||it(y)?(e.consume(y),b):N(y)}function N(y){return y===61?(e.consume(y),S):re(y)?(e.consume(y),N):x(y)}function S(y){return y===null||y===60||y===61||y===62||y===96?n(y):y===34||y===39?(e.consume(y),u=y,C):re(y)?(e.consume(y),S):I(y)}function C(y){return y===u?(e.consume(y),u=null,R):y===null||Q(y)?n(y):(e.consume(y),C)}function I(y){return y===null||y===34||y===39||y===47||y===60||y===61||y===62||y===96||Xe(y)?N(y):(e.consume(y),I)}function R(y){return y===47||y===62||re(y)?x(y):n(y)}function P(y){return y===62?(e.consume(y),j):n(y)}function j(y){return y===null||Q(y)?E(y):re(y)?(e.consume(y),j):n(y)}function E(y){return y===45&&i===2?(e.consume(y),K):y===60&&i===1?(e.consume(y),le):y===62&&i===4?(e.consume(y),L):y===63&&i===3?(e.consume(y),m):y===93&&i===5?(e.consume(y),B):Q(y)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(zy,M,U)(y)):y===null||Q(y)?(e.exit("htmlFlowData"),U(y)):(e.consume(y),E)}function U(y){return e.check(Py,V,M)(y)}function V(y){return e.enter("lineEnding"),e.consume(y),e.exit("lineEnding"),W}function W(y){return y===null||Q(y)?U(y):(e.enter("htmlFlowData"),E(y))}function K(y){return y===45?(e.consume(y),m):E(y)}function le(y){return y===47?(e.consume(y),o="",_):E(y)}function _(y){if(y===62){const X=o.toLowerCase();return Tc.includes(X)?(e.consume(y),L):E(y)}return Tt(y)&&o.length<8?(e.consume(y),o+=String.fromCharCode(y),_):E(y)}function B(y){return y===93?(e.consume(y),m):E(y)}function m(y){return y===62?(e.consume(y),L):y===45&&i===2?(e.consume(y),m):E(y)}function L(y){return y===null||Q(y)?(e.exit("htmlFlowData"),M(y)):(e.consume(y),L)}function M(y){return e.exit("htmlFlow"),t(y)}}function Ry(e,t,n){const r=this;return i;function i(o){return Q(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Dy(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Vl,t,n)}}const Fy={name:"htmlText",tokenize:Oy};function Oy(e,t,n){const r=this;let i,l,o;return a;function a(m){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(m),u}function u(m){return m===33?(e.consume(m),c):m===47?(e.consume(m),N):m===63?(e.consume(m),x):Tt(m)?(e.consume(m),I):n(m)}function c(m){return m===45?(e.consume(m),d):m===91?(e.consume(m),l=0,k):Tt(m)?(e.consume(m),v):n(m)}function d(m){return m===45?(e.consume(m),p):n(m)}function f(m){return m===null?n(m):m===45?(e.consume(m),g):Q(m)?(o=f,le(m)):(e.consume(m),f)}function g(m){return m===45?(e.consume(m),p):f(m)}function p(m){return m===62?K(m):m===45?g(m):f(m)}function k(m){const L="CDATA[";return m===L.charCodeAt(l++)?(e.consume(m),l===L.length?w:k):n(m)}function w(m){return m===null?n(m):m===93?(e.consume(m),z):Q(m)?(o=w,le(m)):(e.consume(m),w)}function z(m){return m===93?(e.consume(m),h):w(m)}function h(m){return m===62?K(m):m===93?(e.consume(m),h):w(m)}function v(m){return m===null||m===62?K(m):Q(m)?(o=v,le(m)):(e.consume(m),v)}function x(m){return m===null?n(m):m===63?(e.consume(m),b):Q(m)?(o=x,le(m)):(e.consume(m),x)}function b(m){return m===62?K(m):x(m)}function N(m){return Tt(m)?(e.consume(m),S):n(m)}function S(m){return m===45||it(m)?(e.consume(m),S):C(m)}function C(m){return Q(m)?(o=C,le(m)):re(m)?(e.consume(m),C):K(m)}function I(m){return m===45||it(m)?(e.consume(m),I):m===47||m===62||Xe(m)?R(m):n(m)}function R(m){return m===47?(e.consume(m),K):m===58||m===95||Tt(m)?(e.consume(m),P):Q(m)?(o=R,le(m)):re(m)?(e.consume(m),R):K(m)}function P(m){return m===45||m===46||m===58||m===95||it(m)?(e.consume(m),P):j(m)}function j(m){return m===61?(e.consume(m),E):Q(m)?(o=j,le(m)):re(m)?(e.consume(m),j):R(m)}function E(m){return m===null||m===60||m===61||m===62||m===96?n(m):m===34||m===39?(e.consume(m),i=m,U):Q(m)?(o=E,le(m)):re(m)?(e.consume(m),E):(e.consume(m),V)}function U(m){return m===i?(e.consume(m),i=void 0,W):m===null?n(m):Q(m)?(o=U,le(m)):(e.consume(m),U)}function V(m){return m===null||m===34||m===39||m===60||m===61||m===96?n(m):m===47||m===62||Xe(m)?R(m):(e.consume(m),V)}function W(m){return m===47||m===62||Xe(m)?R(m):n(m)}function K(m){return m===62?(e.consume(m),e.exit("htmlTextData"),e.exit("htmlText"),t):n(m)}function le(m){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(m),e.exit("lineEnding"),_}function _(m){return re(m)?ue(e,B,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(m):B(m)}function B(m){return e.enter("htmlTextData"),o(m)}}const Us={name:"labelEnd",resolveAll:Uy,resolveTo:Vy,tokenize:Wy},By={tokenize:Qy},$y={tokenize:qy},Hy={tokenize:Ky};function Uy(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&zt(e,0,e.length,n),e}function Vy(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const u={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",u,t],["enter",c,t]],a=dt(a,e.slice(l+1,l+r+3)),a=dt(a,[["enter",d,t]]),a=dt(a,Hs(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=dt(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=dt(a,e.slice(o+1)),a=dt(a,[["exit",u,t]]),zt(e,l,e.length,a),e}function Wy(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(g){return l?l._inactive?f(g):(o=r.parser.defined.includes(lr(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(g),e.exit("labelMarker"),e.exit("labelEnd"),u):n(g)}function u(g){return g===40?e.attempt(By,d,o?d:f)(g):g===91?e.attempt($y,d,o?c:f)(g):o?d(g):f(g)}function c(g){return e.attempt(Hy,d,f)(g)}function d(g){return t(g)}function f(g){return l._balanced=!0,n(g)}}function Qy(e,t,n){return r;function r(f){return e.enter("resource"),e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),i}function i(f){return Xe(f)?Vr(e,l)(f):l(f)}function l(f){return f===41?d(f):Rp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(f)}function o(f){return Xe(f)?Vr(e,u)(f):d(f)}function a(f){return n(f)}function u(f){return f===34||f===39||f===40?Fp(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(f):d(f)}function c(f){return Xe(f)?Vr(e,d)(f):d(f)}function d(f){return f===41?(e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),e.exit("resource"),t):n(f)}}function qy(e,t,n){const r=this;return i;function i(a){return Dp.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(lr(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Ky(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Yy={name:"labelStartImage",resolveAll:Us.resolveAll,tokenize:Gy};function Gy(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Xy={name:"labelStartLink",resolveAll:Us.resolveAll,tokenize:Jy};function Jy(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const _o={name:"lineEnding",tokenize:Zy};function Zy(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),ue(e,t,"linePrefix")}}const Xi={name:"thematicBreak",tokenize:e1};function e1(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),u(c)):r>=3&&(c===null||Q(c))?(e.exit("thematicBreak"),t(c)):n(c)}function u(c){return c===i?(e.consume(c),r++,u):(e.exit("thematicBreakSequence"),re(c)?ue(e,a,"whitespace")(c):a(c))}}const We={continuation:{tokenize:i1},exit:o1,name:"list",tokenize:r1},t1={partial:!0,tokenize:a1},n1={partial:!0,tokenize:l1};function r1(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(p){const k=r.containerState.type||(p===42||p===43||p===45?"listUnordered":"listOrdered");if(k==="listUnordered"?!r.containerState.marker||p===r.containerState.marker:Ma(p)){if(r.containerState.type||(r.containerState.type=k,e.enter(k,{_container:!0})),k==="listUnordered")return e.enter("listItemPrefix"),p===42||p===45?e.check(Xi,n,c)(p):c(p);if(!r.interrupt||p===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),u(p)}return n(p)}function u(p){return Ma(p)&&++o<10?(e.consume(p),u):(!r.interrupt||o<2)&&(r.containerState.marker?p===r.containerState.marker:p===41||p===46)?(e.exit("listItemValue"),c(p)):n(p)}function c(p){return e.enter("listItemMarker"),e.consume(p),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||p,e.check(Vl,r.interrupt?n:d,e.attempt(t1,g,f))}function d(p){return r.containerState.initialBlankLine=!0,l++,g(p)}function f(p){return re(p)?(e.enter("listItemPrefixWhitespace"),e.consume(p),e.exit("listItemPrefixWhitespace"),g):n(p)}function g(p){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(p)}}function i1(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Vl,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,ue(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!re(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(n1,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,ue(e,e.attempt(We,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function l1(e,t,n){const r=this;return ue(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function o1(e){e.exit(this.containerState.type)}function a1(e,t,n){const r=this;return ue(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!re(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const Lc={name:"setextUnderline",resolveTo:s1,tokenize:u1};function s1(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function u1(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,f;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){f=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||f)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),re(c)?ue(e,u,"lineSuffix")(c):u(c))}function u(c){return c===null||Q(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const c1={tokenize:d1};function d1(e){const t=this,n=e.attempt(Vl,r,e.attempt(this.parser.constructs.flowInitial,i,ue(e,e.attempt(this.parser.constructs.flow,i,e.attempt(gy,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const f1={resolveAll:Bp()},p1=Op("string"),h1=Op("text");function Op(e){return{resolveAll:Bp(e==="text"?m1:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),u}function u(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),u)}function c(d){if(d===null)return!0;const f=i[d];let g=-1;if(f)for(;++g<f.length;){const p=f[g];if(!p.previous||p.previous.call(r,r.previous))return!0}return!1}}}function Bp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function m1(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,u;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)u=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||u||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const g1={42:We,43:We,45:We,48:We,49:We,50:We,51:We,52:We,53:We,54:We,55:We,56:We,57:We,62:zp},v1={91:wy},x1={[-2]:bo,[-1]:bo,32:bo},y1={35:Ny,42:Xi,45:[Lc,Xi],60:Iy,61:Lc,95:Xi,96:Ec,126:Ec},k1={38:Mp,92:Pp},w1={[-5]:_o,[-4]:_o,[-3]:_o,33:Yy,38:Mp,42:Aa,60:[Xx,Fy],91:Xy,92:[jy,Pp],93:Us,95:Aa,96:cy},S1={null:[Aa,f1]},b1={null:[42,95]},_1={null:[]},j1=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:b1,contentInitial:v1,disable:_1,document:g1,flow:y1,flowInitial:x1,insideSpan:S1,string:k1,text:w1},Symbol.toStringTag,{value:"Module"}));function C1(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const u={attempt:C(N),check:C(S),consume:v,enter:x,exit:b,interrupt:C(S,{interrupt:!0})},c={code:null,containerState:{},defineSkip:w,events:[],now:k,parser:e,previous:null,sliceSerialize:g,sliceStream:p,write:f};let d=t.tokenize.call(c,u);return t.resolveAll&&l.push(t),c;function f(j){return o=dt(o,j),z(),o[o.length-1]!==null?[]:(I(t,0),c.events=Hs(l,c.events,c),c.events)}function g(j,E){return E1(p(j),E)}function p(j){return N1(o,j)}function k(){const{_bufferIndex:j,_index:E,line:U,column:V,offset:W}=r;return{_bufferIndex:j,_index:E,line:U,column:V,offset:W}}function w(j){i[j.line]=j.column,P()}function z(){let j;for(;r._index<o.length;){const E=o[r._index];if(typeof E=="string")for(j=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===j&&r._bufferIndex<E.length;)h(E.charCodeAt(r._bufferIndex));else h(E)}}function h(j){d=d(j)}function v(j){Q(j)?(r.line++,r.column=1,r.offset+=j===-3?2:1,P()):j!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=j}function x(j,E){const U=E||{};return U.type=j,U.start=k(),c.events.push(["enter",U,c]),a.push(U),U}function b(j){const E=a.pop();return E.end=k(),c.events.push(["exit",E,c]),E}function N(j,E){I(j,E.from)}function S(j,E){E.restore()}function C(j,E){return U;function U(V,W,K){let le,_,B,m;return Array.isArray(V)?M(V):"tokenize"in V?M([V]):L(V);function L(Z){return xe;function xe(_e){const te=_e!==null&&Z[_e],Ne=_e!==null&&Z.null,Ue=[...Array.isArray(te)?te:te?[te]:[],...Array.isArray(Ne)?Ne:Ne?[Ne]:[]];return M(Ue)(_e)}}function M(Z){return le=Z,_=0,Z.length===0?K:y(Z[_])}function y(Z){return xe;function xe(_e){return m=R(),B=Z,Z.partial||(c.currentConstruct=Z),Z.name&&c.parser.constructs.disable.null.includes(Z.name)?pe():Z.tokenize.call(E?Object.assign(Object.create(c),E):c,u,X,pe)(_e)}}function X(Z){return j(B,m),W}function pe(Z){return m.restore(),++_<le.length?y(le[_]):K}}}function I(j,E){j.resolveAll&&!l.includes(j)&&l.push(j),j.resolve&&zt(c.events,E,c.events.length-E,j.resolve(c.events.slice(E),c)),j.resolveTo&&(c.events=j.resolveTo(c.events,c))}function R(){const j=k(),E=c.previous,U=c.currentConstruct,V=c.events.length,W=Array.from(a);return{from:V,restore:K};function K(){r=j,c.previous=E,c.currentConstruct=U,c.events.length=V,a=W,P()}}function P(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function N1(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function E1(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function T1(e){const r={constructs:Rx([j1,...(e||{}).extensions||[]]),content:i(Vx),defined:[],document:i(Qx),flow:i(c1),lazy:{},string:i(p1),text:i(h1)};return r;function i(l){return o;function o(a){return C1(r,l,a)}}}function L1(e){for(;!Ap(e););return e}const Ic=/[\0\t\n\r]/g;function I1(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const u=[];let c,d,f,g,p;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),f=0,t="",n&&(l.charCodeAt(0)===65279&&f++,n=void 0);f<l.length;){if(Ic.lastIndex=f,c=Ic.exec(l),g=c&&c.index!==void 0?c.index:l.length,p=l.charCodeAt(g),!c){t=l.slice(f);break}if(p===10&&f===g&&r)u.push(-3),r=void 0;else switch(r&&(u.push(-5),r=void 0),f<g&&(u.push(l.slice(f,g)),e+=g-f),p){case 0:{u.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,u.push(-2);e++<d;)u.push(-1);break}case 10:{u.push(-4),e=1;break}default:r=!0,e=1}f=g+1}return a&&(r&&u.push(-5),t&&u.push(t),u.push(null)),u}}const z1=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function P1(e){return e.replace(z1,M1)}function M1(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return Ip(n.slice(l?2:1),l?16:10)}return $s(n)||e}const $p={}.hasOwnProperty;function A1(e,t,n){return typeof t!="string"&&(n=t,t=void 0),R1(n)(L1(T1(n).document().write(I1()(e,t,!0))))}function R1(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Js),autolinkProtocol:R,autolinkEmail:R,atxHeading:l(Ys),blockQuote:l(Ne),characterEscape:R,characterReference:R,codeFenced:l(Ue),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(Ue,o),codeText:l(qt,o),codeTextData:R,data:R,codeFlowValue:R,definition:l(Kt),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Zp),hardBreakEscape:l(Gs),hardBreakTrailing:l(Gs),htmlFlow:l(Xs,o),htmlFlowData:R,htmlText:l(Xs,o),htmlTextData:R,image:l(eh),label:o,link:l(Js),listItem:l(th),listItemValue:g,listOrdered:l(Zs,f),listUnordered:l(Zs),paragraph:l(nh),reference:y,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Ys),strong:l(rh),thematicBreak:l(lh)},exit:{atxHeading:u(),atxHeadingSequence:N,autolink:u(),autolinkEmail:te,autolinkProtocol:_e,blockQuote:u(),characterEscapeValue:P,characterReferenceMarkerHexadecimal:pe,characterReferenceMarkerNumeric:pe,characterReferenceValue:Z,characterReference:xe,codeFenced:u(z),codeFencedFence:w,codeFencedFenceInfo:p,codeFencedFenceMeta:k,codeFlowValue:P,codeIndented:u(h),codeText:u(W),codeTextData:P,data:P,definition:u(),definitionDestinationString:b,definitionLabelString:v,definitionTitleString:x,emphasis:u(),hardBreakEscape:u(E),hardBreakTrailing:u(E),htmlFlow:u(U),htmlFlowData:P,htmlText:u(V),htmlTextData:P,image:u(le),label:B,labelText:_,lineEnding:j,link:u(K),listItem:u(),listOrdered:u(),listUnordered:u(),paragraph:u(),referenceString:X,resourceDestinationString:m,resourceTitleString:L,resource:M,setextHeading:u(I),setextHeadingLineSequence:C,setextHeadingText:S,strong:u(),thematicBreak:u()}};Hp(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(T){let O={type:"root",children:[]};const q={stack:[O],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},ee=[];let oe=-1;for(;++oe<T.length;)if(T[oe][1].type==="listOrdered"||T[oe][1].type==="listUnordered")if(T[oe][0]==="enter")ee.push(oe);else{const gt=ee.pop();oe=i(T,gt,oe)}for(oe=-1;++oe<T.length;){const gt=t[T[oe][0]];$p.call(gt,T[oe][1].type)&&gt[T[oe][1].type].call(Object.assign({sliceSerialize:T[oe][2].sliceSerialize},q),T[oe][1])}if(q.tokenStack.length>0){const gt=q.tokenStack[q.tokenStack.length-1];(gt[1]||zc).call(q,void 0,gt[0])}for(O.position={start:Gt(T.length>0?T[0][1].start:{line:1,column:1,offset:0}),end:Gt(T.length>0?T[T.length-2][1].end:{line:1,column:1,offset:0})},oe=-1;++oe<t.transforms.length;)O=t.transforms[oe](O)||O;return O}function i(T,O,q){let ee=O-1,oe=-1,gt=!1,yn,Pt,xr,yr;for(;++ee<=q;){const Ze=T[ee];switch(Ze[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Ze[0]==="enter"?oe++:oe--,yr=void 0;break}case"lineEndingBlank":{Ze[0]==="enter"&&(yn&&!yr&&!oe&&!xr&&(xr=ee),yr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:yr=void 0}if(!oe&&Ze[0]==="enter"&&Ze[1].type==="listItemPrefix"||oe===-1&&Ze[0]==="exit"&&(Ze[1].type==="listUnordered"||Ze[1].type==="listOrdered")){if(yn){let Rn=ee;for(Pt=void 0;Rn--;){const Mt=T[Rn];if(Mt[1].type==="lineEnding"||Mt[1].type==="lineEndingBlank"){if(Mt[0]==="exit")continue;Pt&&(T[Pt][1].type="lineEndingBlank",gt=!0),Mt[1].type="lineEnding",Pt=Rn}else if(!(Mt[1].type==="linePrefix"||Mt[1].type==="blockQuotePrefix"||Mt[1].type==="blockQuotePrefixWhitespace"||Mt[1].type==="blockQuoteMarker"||Mt[1].type==="listItemIndent"))break}xr&&(!Pt||xr<Pt)&&(yn._spread=!0),yn.end=Object.assign({},Pt?T[Pt][1].start:Ze[1].end),T.splice(Pt||ee,0,["exit",yn,Ze[2]]),ee++,q++}if(Ze[1].type==="listItemPrefix"){const Rn={type:"listItem",_spread:!1,start:Object.assign({},Ze[1].start),end:void 0};yn=Rn,T.splice(ee,0,["enter",Rn,Ze[2]]),ee++,q++,xr=void 0,yr=!0}}}return T[O][1]._spread=gt,q}function l(T,O){return q;function q(ee){a.call(this,T(ee),ee),O&&O.call(this,ee)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(T,O,q){this.stack[this.stack.length-1].children.push(T),this.stack.push(T),this.tokenStack.push([O,q||void 0]),T.position={start:Gt(O.start),end:void 0}}function u(T){return O;function O(q){T&&T.call(this,q),c.call(this,q)}}function c(T,O){const q=this.stack.pop(),ee=this.tokenStack.pop();if(ee)ee[0].type!==T.type&&(O?O.call(this,T,ee[0]):(ee[1]||zc).call(this,T,ee[0]));else throw new Error("Cannot close `"+T.type+"` ("+Ur({start:T.start,end:T.end})+"): it’s not open");q.position.end=Gt(T.end)}function d(){return Mx(this.stack.pop())}function f(){this.data.expectingFirstListItemValue=!0}function g(T){if(this.data.expectingFirstListItemValue){const O=this.stack[this.stack.length-2];O.start=Number.parseInt(this.sliceSerialize(T),10),this.data.expectingFirstListItemValue=void 0}}function p(){const T=this.resume(),O=this.stack[this.stack.length-1];O.lang=T}function k(){const T=this.resume(),O=this.stack[this.stack.length-1];O.meta=T}function w(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function z(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function h(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T.replace(/(\r?\n|\r)$/g,"")}function v(T){const O=this.resume(),q=this.stack[this.stack.length-1];q.label=O,q.identifier=lr(this.sliceSerialize(T)).toLowerCase()}function x(){const T=this.resume(),O=this.stack[this.stack.length-1];O.title=T}function b(){const T=this.resume(),O=this.stack[this.stack.length-1];O.url=T}function N(T){const O=this.stack[this.stack.length-1];if(!O.depth){const q=this.sliceSerialize(T).length;O.depth=q}}function S(){this.data.setextHeadingSlurpLineEnding=!0}function C(T){const O=this.stack[this.stack.length-1];O.depth=this.sliceSerialize(T).codePointAt(0)===61?1:2}function I(){this.data.setextHeadingSlurpLineEnding=void 0}function R(T){const q=this.stack[this.stack.length-1].children;let ee=q[q.length-1];(!ee||ee.type!=="text")&&(ee=ih(),ee.position={start:Gt(T.start),end:void 0},q.push(ee)),this.stack.push(ee)}function P(T){const O=this.stack.pop();O.value+=this.sliceSerialize(T),O.position.end=Gt(T.end)}function j(T){const O=this.stack[this.stack.length-1];if(this.data.atHardBreak){const q=O.children[O.children.length-1];q.position.end=Gt(T.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(O.type)&&(R.call(this,T),P.call(this,T))}function E(){this.data.atHardBreak=!0}function U(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function V(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function W(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function K(){const T=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";T.type+="Reference",T.referenceType=O,delete T.url,delete T.title}else delete T.identifier,delete T.label;this.data.referenceType=void 0}function le(){const T=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";T.type+="Reference",T.referenceType=O,delete T.url,delete T.title}else delete T.identifier,delete T.label;this.data.referenceType=void 0}function _(T){const O=this.sliceSerialize(T),q=this.stack[this.stack.length-2];q.label=P1(O),q.identifier=lr(O).toLowerCase()}function B(){const T=this.stack[this.stack.length-1],O=this.resume(),q=this.stack[this.stack.length-1];if(this.data.inReference=!0,q.type==="link"){const ee=T.children;q.children=ee}else q.alt=O}function m(){const T=this.resume(),O=this.stack[this.stack.length-1];O.url=T}function L(){const T=this.resume(),O=this.stack[this.stack.length-1];O.title=T}function M(){this.data.inReference=void 0}function y(){this.data.referenceType="collapsed"}function X(T){const O=this.resume(),q=this.stack[this.stack.length-1];q.label=O,q.identifier=lr(this.sliceSerialize(T)).toLowerCase(),this.data.referenceType="full"}function pe(T){this.data.characterReferenceType=T.type}function Z(T){const O=this.sliceSerialize(T),q=this.data.characterReferenceType;let ee;q?(ee=Ip(O,q==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):ee=$s(O);const oe=this.stack[this.stack.length-1];oe.value+=ee}function xe(T){const O=this.stack.pop();O.position.end=Gt(T.end)}function _e(T){P.call(this,T);const O=this.stack[this.stack.length-1];O.url=this.sliceSerialize(T)}function te(T){P.call(this,T);const O=this.stack[this.stack.length-1];O.url="mailto:"+this.sliceSerialize(T)}function Ne(){return{type:"blockquote",children:[]}}function Ue(){return{type:"code",lang:null,meta:null,value:""}}function qt(){return{type:"inlineCode",value:""}}function Kt(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Zp(){return{type:"emphasis",children:[]}}function Ys(){return{type:"heading",depth:0,children:[]}}function Gs(){return{type:"break"}}function Xs(){return{type:"html",value:""}}function eh(){return{type:"image",title:null,url:"",alt:null}}function Js(){return{type:"link",title:null,url:"",children:[]}}function Zs(T){return{type:"list",ordered:T.type==="listOrdered",start:null,spread:T._spread,children:[]}}function th(T){return{type:"listItem",spread:T._spread,checked:null,children:[]}}function nh(){return{type:"paragraph",children:[]}}function rh(){return{type:"strong",children:[]}}function ih(){return{type:"text",value:""}}function lh(){return{type:"thematicBreak"}}}function Gt(e){return{line:e.line,column:e.column,offset:e.offset}}function Hp(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Hp(e,r):D1(e,r)}}function D1(e,t){let n;for(n in t)if($p.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function zc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Ur({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Ur({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Ur({start:t.start,end:t.end})+") is still open")}function F1(e){const t=this;t.parser=n;function n(r){return A1(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function O1(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function B1(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function $1(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function H1(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function U1(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function V1(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=vr(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const u={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,u);const c={type:"element",tagName:"sup",properties:{},children:[u]};return e.patch(t,c),e.applyData(t,c)}function W1(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Q1(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Up(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function q1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Up(e,t);const i={src:vr(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function K1(e,t){const n={src:vr(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function Y1(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function G1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Up(e,t);const i={href:vr(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function X1(e,t){const n={href:vr(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function J1(e,t,n){const r=e.all(t),i=n?Z1(n):Vp(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let f;d&&d.type==="element"&&d.tagName==="p"?f=d:(f={type:"element",tagName:"p",properties:{},children:[]},r.unshift(f)),f.children.length>0&&f.children.unshift({type:"text",value:" "}),f.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const u=r[r.length-1];u&&(i||u.type!=="element"||u.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function Z1(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Vp(n[r])}return t}function Vp(e){const t=e.spread;return t??e.children.length>1}function e0(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function t0(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function n0(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function r0(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function i0(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=Ds(t.children[1]),u=_p(t.children[t.children.length-1]);a&&u&&(o.position={start:a,end:u}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function l0(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let u=-1;const c=[];for(;++u<a;){const f=t.children[u],g={},p=o?o[u]:void 0;p&&(g.align=p);let k={type:"element",tagName:l,properties:g,children:[]};f&&(k.children=e.all(f),e.patch(f,k),k=e.applyData(f,k)),c.push(k)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function o0(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const Pc=9,Mc=32;function a0(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(Ac(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(Ac(t.slice(i),i>0,!1)),l.join("")}function Ac(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===Pc||l===Mc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===Pc||l===Mc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function s0(e,t){const n={type:"text",value:a0(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function u0(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const c0={blockquote:O1,break:B1,code:$1,delete:H1,emphasis:U1,footnoteReference:V1,heading:W1,html:Q1,imageReference:q1,image:K1,inlineCode:Y1,linkReference:G1,link:X1,listItem:J1,list:e0,paragraph:t0,root:n0,strong:r0,table:i0,tableCell:o0,tableRow:l0,text:s0,thematicBreak:u0,toml:Ai,yaml:Ai,definition:Ai,footnoteDefinition:Ai};function Ai(){}const Wp=-1,Wl=0,Wr=1,jl=2,Vs=3,Ws=4,Qs=5,qs=6,Qp=7,qp=8,Rc=typeof self=="object"?self:globalThis,d0=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Wl:case Wp:return n(o,i);case Wr:{const a=n([],i);for(const u of o)a.push(r(u));return a}case jl:{const a=n({},i);for(const[u,c]of o)a[r(u)]=r(c);return a}case Vs:return n(new Date(o),i);case Ws:{const{source:a,flags:u}=o;return n(new RegExp(a,u),i)}case Qs:{const a=n(new Map,i);for(const[u,c]of o)a.set(r(u),r(c));return a}case qs:{const a=n(new Set,i);for(const u of o)a.add(r(u));return a}case Qp:{const{name:a,message:u}=o;return n(new Rc[a](u),i)}case qp:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new Rc[l](o),i)};return r},Dc=e=>d0(new Map,e)(0),On="",{toString:f0}={},{keys:p0}=Object,Tr=e=>{const t=typeof e;if(t!=="object"||!e)return[Wl,t];const n=f0.call(e).slice(8,-1);switch(n){case"Array":return[Wr,On];case"Object":return[jl,On];case"Date":return[Vs,On];case"RegExp":return[Ws,On];case"Map":return[Qs,On];case"Set":return[qs,On];case"DataView":return[Wr,n]}return n.includes("Array")?[Wr,n]:n.includes("Error")?[Qp,n]:[jl,n]},Ri=([e,t])=>e===Wl&&(t==="function"||t==="symbol"),h0=(e,t,n,r)=>{const i=(o,a)=>{const u=r.push(o)-1;return n.set(a,u),u},l=o=>{if(n.has(o))return n.get(o);let[a,u]=Tr(o);switch(a){case Wl:{let d=o;switch(u){case"bigint":a=qp,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+u);d=null;break;case"undefined":return i([Wp],o)}return i([a,d],o)}case Wr:{if(u){let g=o;return u==="DataView"?g=new Uint8Array(o.buffer):u==="ArrayBuffer"&&(g=new Uint8Array(o)),i([u,[...g]],o)}const d=[],f=i([a,d],o);for(const g of o)d.push(l(g));return f}case jl:{if(u)switch(u){case"BigInt":return i([u,o.toString()],o);case"Boolean":case"Number":case"String":return i([u,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],f=i([a,d],o);for(const g of p0(o))(e||!Ri(Tr(o[g])))&&d.push([l(g),l(o[g])]);return f}case Vs:return i([a,o.toISOString()],o);case Ws:{const{source:d,flags:f}=o;return i([a,{source:d,flags:f}],o)}case Qs:{const d=[],f=i([a,d],o);for(const[g,p]of o)(e||!(Ri(Tr(g))||Ri(Tr(p))))&&d.push([l(g),l(p)]);return f}case qs:{const d=[],f=i([a,d],o);for(const g of o)(e||!Ri(Tr(g)))&&d.push(l(g));return f}}const{message:c}=o;return i([a,{name:u,message:c}],o)};return l},Fc=(e,{json:t,lossy:n}={})=>{const r=[];return h0(!(t||n),!!t,new Map,r)(e),r},Cl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?Dc(Fc(e,t)):structuredClone(e):(e,t)=>Dc(Fc(e,t));function m0(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function g0(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function v0(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||m0,r=e.options.footnoteBackLabel||g0,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let u=-1;for(;++u<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[u]);if(!c)continue;const d=e.all(c),f=String(c.identifier).toUpperCase(),g=vr(f.toLowerCase());let p=0;const k=[],w=e.footnoteCounts.get(f);for(;w!==void 0&&++p<=w;){k.length>0&&k.push({type:"text",value:" "});let v=typeof n=="string"?n:n(u,p);typeof v=="string"&&(v={type:"text",value:v}),k.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+g+(p>1?"-"+p:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(u,p),className:["data-footnote-backref"]},children:Array.isArray(v)?v:[v]})}const z=d[d.length-1];if(z&&z.type==="element"&&z.tagName==="p"){const v=z.children[z.children.length-1];v&&v.type==="text"?v.value+=" ":z.children.push({type:"text",value:" "}),z.children.push(...k)}else d.push(...k);const h={type:"element",tagName:"li",properties:{id:t+"fn-"+g},children:e.wrap(d,!0)};e.patch(c,h),a.push(h)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...Cl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const Kp=function(e){if(e==null)return w0;if(typeof e=="function")return Ql(e);if(typeof e=="object")return Array.isArray(e)?x0(e):y0(e);if(typeof e=="string")return k0(e);throw new Error("Expected function, string, or object as test")};function x0(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=Kp(e[n]);return Ql(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function y0(e){const t=e;return Ql(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function k0(e){return Ql(t);function t(n){return n&&n.type===e}}function Ql(e){return t;function t(n,r,i){return!!(S0(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function w0(){return!0}function S0(e){return e!==null&&typeof e=="object"&&"type"in e}const Yp=[],b0=!0,Oc=!1,_0="skip";function j0(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=Kp(i),o=r?-1:1;a(e,void 0,[])();function a(u,c,d){const f=u&&typeof u=="object"?u:{};if(typeof f.type=="string"){const p=typeof f.tagName=="string"?f.tagName:typeof f.name=="string"?f.name:void 0;Object.defineProperty(g,"name",{value:"node ("+(u.type+(p?"<"+p+">":""))+")"})}return g;function g(){let p=Yp,k,w,z;if((!t||l(u,c,d[d.length-1]||void 0))&&(p=C0(n(u,d)),p[0]===Oc))return p;if("children"in u&&u.children){const h=u;if(h.children&&p[0]!==_0)for(w=(r?h.children.length:-1)+o,z=d.concat(h);w>-1&&w<h.children.length;){const v=h.children[w];if(k=a(v,w,z)(),k[0]===Oc)return k;w=typeof k[1]=="number"?k[1]:w+o}}return p}}}function C0(e){return Array.isArray(e)?e:typeof e=="number"?[b0,e]:e==null?Yp:[e]}function Gp(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),j0(e,l,a,i);function a(u,c){const d=c[c.length-1],f=d?d.children.indexOf(u):void 0;return o(u,f,d)}}const Ra={}.hasOwnProperty,N0={};function E0(e,t){const n=t||N0,r=new Map,i=new Map,l=new Map,o={...c0,...n.handlers},a={all:c,applyData:L0,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:u,options:n,patch:T0,wrap:z0};return Gp(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const f=d.type==="definition"?r:i,g=String(d.identifier).toUpperCase();f.has(g)||f.set(g,d)}}),a;function u(d,f){const g=d.type,p=a.handlers[g];if(Ra.call(a.handlers,g)&&p)return p(a,d,f);if(a.options.passThrough&&a.options.passThrough.includes(g)){if("children"in d){const{children:w,...z}=d,h=Cl(z);return h.children=a.all(d),h}return Cl(d)}return(a.options.unknownHandler||I0)(a,d,f)}function c(d){const f=[];if("children"in d){const g=d.children;let p=-1;for(;++p<g.length;){const k=a.one(g[p],d);if(k){if(p&&g[p-1].type==="break"&&(!Array.isArray(k)&&k.type==="text"&&(k.value=Bc(k.value)),!Array.isArray(k)&&k.type==="element")){const w=k.children[0];w&&w.type==="text"&&(w.value=Bc(w.value))}Array.isArray(k)?f.push(...k):f.push(k)}}}return f}}function T0(e,t){e.position&&(t.position=fx(e))}function L0(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,Cl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function I0(e,t){const n=t.data||{},r="value"in t&&!(Ra.call(n,"hProperties")||Ra.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function z0(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Bc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function $c(e,t){const n=E0(e,t),r=n.one(e,void 0),i=v0(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function P0(e,t){return e&&"run"in e?async function(n,r){const i=$c(n,{file:r,...t});await e.run(i,r)}:function(n,r){return $c(n,{file:r,...e||t})}}function Hc(e){if(e)throw e}var Ji=Object.prototype.hasOwnProperty,Xp=Object.prototype.toString,Uc=Object.defineProperty,Vc=Object.getOwnPropertyDescriptor,Wc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Xp.call(t)==="[object Array]"},Qc=function(t){if(!t||Xp.call(t)!=="[object Object]")return!1;var n=Ji.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Ji.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Ji.call(t,i)},qc=function(t,n){Uc&&n.name==="__proto__"?Uc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Kc=function(t,n){if(n==="__proto__")if(Ji.call(t,n)){if(Vc)return Vc(t,n).value}else return;return t[n]},M0=function e(){var t,n,r,i,l,o,a=arguments[0],u=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},u=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});u<c;++u)if(t=arguments[u],t!=null)for(n in t)r=Kc(a,n),i=Kc(t,n),a!==i&&(d&&i&&(Qc(i)||(l=Wc(i)))?(l?(l=!1,o=r&&Wc(r)?r:[]):o=r&&Qc(r)?r:{},qc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&qc(a,{name:n,newValue:i}));return a};const jo=Oa(M0);function Da(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function A0(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(u,...c){const d=e[++l];let f=-1;if(u){o(u);return}for(;++f<i.length;)(c[f]===null||c[f]===void 0)&&(c[f]=i[f]);i=c,d?R0(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function R0(e,t){let n;return r;function r(...o){const a=e.length>o.length;let u;a&&o.push(i);try{u=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(u&&u.then&&typeof u.then=="function"?u.then(l,i):u instanceof Error?i(u):l(u))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const Nt={basename:D0,dirname:F0,extname:O0,join:B0,sep:"/"};function D0(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');gi(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function F0(e){if(gi(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function O0(e){gi(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function B0(...e){let t=-1,n;for(;++t<e.length;)gi(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":$0(n)}function $0(e){gi(e);const t=e.codePointAt(0)===47;let n=H0(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function H0(e,t){let n="",r=0,i=-1,l=0,o=-1,a,u;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(u=n.lastIndexOf("/"),u!==n.length-1){u<0?(n="",r=0):(n=n.slice(0,u),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function gi(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const U0={cwd:V0};function V0(){return"/"}function Fa(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function W0(e){if(typeof e=="string")e=new URL(e);else if(!Fa(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return Q0(e)}function Q0(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const Co=["history","path","basename","stem","extname","dirname"];class Jp{constructor(t){let n;t?Fa(t)?n={path:t}:typeof t=="string"||q0(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":U0.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<Co.length;){const l=Co[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)Co.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?Nt.basename(this.path):void 0}set basename(t){Eo(t,"basename"),No(t,"basename"),this.path=Nt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?Nt.dirname(this.path):void 0}set dirname(t){Yc(this.basename,"dirname"),this.path=Nt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?Nt.extname(this.path):void 0}set extname(t){if(No(t,"extname"),Yc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=Nt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){Fa(t)&&(t=W0(t)),Eo(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?Nt.basename(this.path,this.extname):void 0}set stem(t){Eo(t,"stem"),No(t,"stem"),this.path=Nt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new De(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function No(e,t){if(e&&e.includes(Nt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+Nt.sep+"`")}function Eo(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Yc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function q0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const K0=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},Y0={}.hasOwnProperty;class Ks extends K0{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=A0()}copy(){const t=new Ks;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(jo(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(Io("data",this.frozen),this.namespace[t]=n,this):Y0.call(this.namespace,t)&&this.namespace[t]||void 0:t?(Io("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Di(t),r=this.parser||this.Parser;return To("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),To("process",this.parser||this.Parser),Lo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Di(t),u=r.parse(a);r.run(u,a,function(d,f,g){if(d||!f||!g)return c(d);const p=f,k=r.stringify(p,g);J0(k)?g.value=k:g.result=k,c(d,g)});function c(d,f){d||!f?o(d):l?l(f):n(void 0,f)}}}processSync(t){let n=!1,r;return this.freeze(),To("processSync",this.parser||this.Parser),Lo("processSync",this.compiler||this.Compiler),this.process(t,i),Xc("processSync","process",n),r;function i(l,o){n=!0,Hc(l),r=o}}run(t,n,r){Gc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const u=Di(n);i.run(t,u,c);function c(d,f,g){const p=f||t;d?a(d):o?o(p):r(void 0,p,g)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Xc("runSync","run",r),i;function l(o,a){Hc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Di(n),i=this.compiler||this.Compiler;return Lo("stringify",i),Gc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(Io("use",this.frozen),t!=null)if(typeof t=="function")u(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")u(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...f]=c;u(d,f)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=jo(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const f=c[d];l(f)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function u(c,d){let f=-1,g=-1;for(;++f<r.length;)if(r[f][0]===c){g=f;break}if(g===-1)r.push([c,...d]);else if(d.length>0){let[p,...k]=d;const w=r[g][1];Da(w)&&Da(p)&&(p=jo(!0,w,p)),r[g]=[c,p,...k]}}}}const G0=new Ks().freeze();function To(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function Lo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function Io(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Gc(e){if(!Da(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Xc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Di(e){return X0(e)?e:new Jp(e)}function X0(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function J0(e){return typeof e=="string"||Z0(e)}function Z0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const ek="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Jc=[],Zc={allowDangerousHtml:!0},tk=/^(https?|ircs?|mailto|xmpp)$/i,nk=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function rk(e){const t=ik(e),n=lk(e);return ok(t.runSync(t.parse(n),n),e)}function ik(e){const t=e.rehypePlugins||Jc,n=e.remarkPlugins||Jc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Zc}:Zc;return G0().use(F1).use(n).use(P0,r).use(t)}function lk(e){const t=e.children||"",n=new Jp;return typeof t=="string"&&(n.value=t),n}function ok(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,u=t.urlTransform||ak;for(const d of nk)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+ek+d.id,void 0);return Gp(e,c),vx(e,{Fragment:s.Fragment,components:i,ignoreInvalidStyle:!0,jsx:s.jsx,jsxs:s.jsxs,passKeys:!0,passNode:!0});function c(d,f,g){if(d.type==="raw"&&g&&typeof f=="number")return o?g.children.splice(f,1):g.children[f]={type:"text",value:d.value},f;if(d.type==="element"){let p;for(p in So)if(Object.hasOwn(So,p)&&Object.hasOwn(d.properties,p)){const k=d.properties[p],w=So[p];(w===null||w.includes(d.tagName))&&(d.properties[p]=u(String(k||""),p,d))}}if(d.type==="element"){let p=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!p&&r&&typeof f=="number"&&(p=!r(d,f,g)),p&&g&&typeof f=="number")return a&&d.children?g.children.splice(f,1,...d.children):g.children.splice(f,1),f}}}function ak(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||tk.test(e.slice(0,t))?e:""}const sk=e=>{if(!e)return null;try{return JSON.parse(e).execution_stats||null}catch{return null}},uk=e=>{if(e.kind!=="status")return!1;const t=e.content.toLowerCase();return t.includes("running")||t.includes("thinking")||t.includes("executing")||t.includes("processing")},ed=10*1024,zo=200,Me={send:s.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),s.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),s.jsx("polyline",{points:"14 2 14 8 20 8"}),s.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),s.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("circle",{cx:"12",cy:"12",r:"10"}),s.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),s.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),s.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),s.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),s.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),s.jsx("circle",{cx:"12",cy:"5",r:"2"}),s.jsx("path",{d:"M12 7v4"})]}),check:s.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),s.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),file:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),s.jsx("polyline",{points:"14 2 14 8 20 8"})]}),folder:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),spinner:s.jsx("svg",{className:"spinner-icon",width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M21 12a9 9 0 1 1-6.219-8.56"})})},ck=e=>{switch(e){case"directive":return Me.directive;case"question":return Me.question;case"status":return Me.status;case"result":return Me.result;case"approval_request":return Me.lock;default:return Me.directive}},dk=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=F.useRef(null),[a,u]=Xt.useState(""),[c,d]=Xt.useState("directive"),[f,g]=Xt.useState(""),[p,k]=Xt.useState(!1),[w,z]=Xt.useState(new Map),[h,v]=Xt.useState(new Set),[x,b]=F.useState(new Set),[N,S]=F.useState(new Set),C=_=>{const B=(_.match(/\n/g)||[]).length+1;if(!(_.length>ed||B>zo))return{needsTruncation:!1,truncated:_,fullLength:_.length,lineCount:B};let L=_.slice(0,ed);const M=L.split(`
`);M.length>zo&&(L=M.slice(0,zo).join(`
`));const y=L.lastIndexOf(`
`);return y>L.length*.8&&(L=L.slice(0,y)),{needsTruncation:!0,truncated:L,fullLength:_.length,lineCount:B}},I=_=>{b(B=>{const m=new Set(B);return m.has(_)?m.delete(_):m.add(_),m})};F.useEffect(()=>{e!=null&&e.workspace?g(e.workspace):g("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),F.useEffect(()=>{var _;(_=o.current)==null||_.scrollIntoView({behavior:"smooth"})},[t]);const R=_=>{g(_),r&&r(_)},P=()=>{a.trim()&&(n(a,c,f||void 0),u(""))},j=_=>{_.key==="Enter"&&!_.shiftKey&&(_.preventDefault(),P())},E=_=>new Date(_).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),U=_=>_.length>12?`${_.slice(0,8)}...`:_,V=_=>{if(!_.metadata_json)return null;try{return JSON.parse(_.metadata_json).approval_id||null}catch{return null}},W=_=>{const B=w.get(_)||"";i&&(i(_,B),v(m=>new Set(m).add(_)),z(m=>{const L=new Map(m);return L.delete(_),L}))},K=_=>{const B=w.get(_)||"";if(!B.trim()){alert("Please provide a reason for rejection");return}l&&(l(_,B),v(m=>new Set(m).add(_)),z(m=>{const L=new Map(m);return L.delete(_),L}))},le=(_,B)=>{z(m=>new Map(m).set(_,B))};return e?s.jsxs("div",{className:"conversation-view",children:[s.jsxs("div",{className:"conversation-header",children:[s.jsxs("div",{className:"header-info",children:[s.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&s.jsxs("span",{className:"thread-agent-badge",children:[Me.bot,e.target_agent]})]}),s.jsxs("div",{className:"header-stats",children:[s.jsxs("span",{className:"message-count",children:[t.length," messages"]}),s.jsx("span",{className:"thread-id",title:e.id,children:U(e.id)})]})]}),s.jsxs("div",{className:"messages-container",children:[t.length===0?s.jsxs("div",{className:"empty-messages",children:[s.jsx("div",{className:"empty-icon",children:s.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:s.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),s.jsx("p",{children:"No messages yet"}),s.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((_,B)=>{const m=_.from_type==="human",L=B===0||t[B-1].from_type!==_.from_type,M=x.has(_.id),{needsTruncation:y,truncated:X,fullLength:pe,lineCount:Z}=C(_.content),xe=M?_.content:X,_e=uk(_);return s.jsxs("div",{className:`message ${m?"human":"agent"}${_e?" running-status":""}`,children:[s.jsx("div",{className:`message-avatar ${L?"visible":""}`,children:L&&(m?Me.user:Me.bot)}),s.jsxs("div",{className:"message-body",children:[L&&s.jsxs("div",{className:"message-meta",children:[s.jsx("span",{className:"sender-name",children:_.from_id}),s.jsxs("span",{className:`kind-badge${_e?" running":""}`,children:[_e?Me.spinner:ck(_.kind)," ",_.kind]}),s.jsx("span",{className:"message-time",children:E(_.created_at)})]}),s.jsxs("div",{className:"message-content",children:[_.kind==="result"||!m?s.jsx(rk,{components:{a:({href:te,children:Ne})=>{let Ue=te;return te&&te.startsWith("/")&&!te.startsWith("//")&&(Ue=`file://${te}`),s.jsx("a",{href:Ue,target:"_blank",rel:"noopener noreferrer",children:Ne})},code:({className:te,children:Ne,...Ue})=>!te?s.jsx("code",{className:"inline-code",...Ue,children:Ne}):s.jsx("code",{className:te,...Ue,children:Ne})},children:xe}):xe,y&&s.jsx("div",{className:"truncation-notice",children:s.jsx("button",{className:"expand-btn",onClick:()=>I(_.id),children:M?s.jsx(s.Fragment,{children:"Show less"}):s.jsxs(s.Fragment,{children:["Show more (",Math.round(pe/1024),"KB, ",Z," lines)"]})})}),_.kind==="approval_request"&&(()=>{const te=V(_),Ne=te&&h.has(te);return te?s.jsx("div",{className:"inline-approval",children:Ne?s.jsxs("div",{className:"approval-handled",children:[Me.check,s.jsx("span",{children:"Action taken"})]}):s.jsxs(s.Fragment,{children:[s.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:w.get(te)||"",onChange:Ue=>le(te,Ue.target.value)}),s.jsxs("div",{className:"approval-actions",children:[s.jsxs("button",{className:"reject-btn",onClick:()=>K(te),title:"Reject",children:[Me.x,"Reject"]}),s.jsxs("button",{className:"approve-btn",onClick:()=>W(te),title:"Approve",children:[Me.check,"Approve"]})]})]})}):null})(),_.kind==="result"&&(()=>{const te=sk(_.metadata_json);if(!te||!te.files_created||te.files_created.length===0)return null;const Ne=N.has(_.id),Ue=()=>{S(qt=>{const Kt=new Set(qt);return Kt.has(_.id)?Kt.delete(_.id):Kt.add(_.id),Kt})};return s.jsxs("div",{className:"files-created-section",children:[s.jsxs("button",{className:`files-toggle-btn ${Ne?"expanded":""}`,onClick:Ue,children:[Me.file,s.jsxs("span",{children:["Files Created (",te.files_created.length,")"]}),te.workspace&&s.jsxs("span",{className:"workspace-badge",title:te.workspace,children:[Me.folder,te.workspace.split("/").pop()]}),s.jsx("span",{className:"toggle-chevron",children:Ne?"▼":"▶"})]}),Ne&&s.jsx("ul",{className:"files-list",children:te.files_created.map((qt,Kt)=>s.jsx("li",{className:"file-item",children:s.jsx("a",{href:`file://${te.workspace?te.workspace+"/":""}${qt}`,target:"_blank",rel:"noopener noreferrer",title:qt,children:qt})},Kt))})]})})()]}),s.jsx("div",{className:"message-footer",children:s.jsxs("span",{className:"message-seq",children:["#",_.message_seq]})})]})]},_.id)}),s.jsx("div",{ref:o})]}),s.jsxs("div",{className:"input-area",children:[p&&s.jsxs("div",{className:"workspace-input-row",children:[s.jsx("input",{type:"text",value:f,onChange:_=>R(_.target.value),onBlur:()=>{r&&r(f)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),s.jsx("button",{onClick:async()=>{try{const B=await(await fetch("/api/select-folder")).json();!B.cancelled&&B.path&&R(B.path)}catch(_){console.error("Failed to open folder picker:",_)}},className:"workspace-browse",title:"Browse for folder",children:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),s.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),s.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),f&&s.jsx("button",{onClick:()=>{R(""),k(!1)},className:"workspace-clear",children:"Clear"})]}),s.jsxs("div",{className:"input-wrapper",children:[s.jsx("button",{onClick:()=>k(!p),className:`workspace-toggle ${f?"has-workspace":""}`,title:f||"Set working directory for agent tasks",children:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),s.jsxs("select",{value:c,onChange:_=>d(_.target.value),className:"kind-selector",title:c==="directive"?"Directive: A task or instruction for the agent to execute":"Question: A query for information (won't trigger execution)",children:[s.jsx("option",{value:"directive",title:"A task or instruction for the agent to execute",children:"Directive"}),s.jsx("option",{value:"question",title:"A query for information (won't trigger execution)",children:"Question"})]}),s.jsx("textarea",{value:a,onChange:_=>u(_.target.value),onKeyPress:j,placeholder:f?`Message (workspace: ${f.split("/").pop()})`:"Type a message...",rows:1}),s.jsx("button",{onClick:P,className:"send-btn",disabled:!a.trim(),children:Me.send})]}),s.jsxs("div",{className:"input-hint",children:["Press ",s.jsx("kbd",{children:"Enter"})," to send, ",s.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),s.jsx("style",{children:`
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
      `})]}):null};class td{constructor(){Ve(this,"ws",null);Ve(this,"wsUrl",null);Ve(this,"isConnecting",!1);Ve(this,"reconnectTimeout",null);Ve(this,"reconnectAttempts",0);Ve(this,"maxReconnectAttempts",10);Ve(this,"connectionState","disconnected");Ve(this,"stateListeners",new Set);Ve(this,"messageHandlers",new Set);Ve(this,"batchHandlers",new Set);Ve(this,"errorHandlers",new Set);Ve(this,"subscriptions",new Map);Ve(this,"hookCount",0)}getState(){return{isConnected:this.connectionState==="connected",connectionState:this.connectionState,reconnectAttempts:this.reconnectAttempts}}subscribeToState(t){return this.stateListeners.add(t),t(this.connectionState,this.reconnectAttempts),()=>this.stateListeners.delete(t)}setConnectionState(t){this.connectionState=t,this.stateListeners.forEach(n=>n(t,this.reconnectAttempts))}registerHook(t,n,r){this.hookCount++,console.log(`[WebSocketService] Hook registered, count: ${this.hookCount}`);const i=t?a=>t(a):null,l=n?a=>n(a):null,o=r?a=>r(a):null;return i&&this.messageHandlers.add(i),l&&this.batchHandlers.add(l),o&&this.errorHandlers.add(o),()=>{this.hookCount--,console.log(`[WebSocketService] Hook unregistered, count: ${this.hookCount}`),i&&this.messageHandlers.delete(i),l&&this.batchHandlers.delete(l),o&&this.errorHandlers.delete(o),this.hookCount===0&&(console.log("[WebSocketService] All hooks unregistered, closing connection"),this.disconnect())}}connect(t,n,r=10){this.maxReconnectAttempts=r;const i=`${t}?instance_id=${n}`;if(this.ws&&this.ws.readyState===WebSocket.OPEN&&this.wsUrl===i){console.log("[WebSocketService] Already connected, skipping");return}if(this.isConnecting){console.log("[WebSocketService] Already connecting, skipping");return}if(this.ws&&this.ws.readyState===WebSocket.CONNECTING){console.log("[WebSocketService] Connection pending, skipping");return}this.ws&&this.wsUrl!==i&&(console.log("[WebSocketService] URL changed, closing old connection"),this.ws.close(),this.ws=null),this.isConnecting=!0,this.wsUrl=i,console.log(`[WebSocketService] Creating new WebSocket to ${i}`),this.setConnectionState(this.reconnectAttempts>0?"reconnecting":"connecting");try{this.ws=new WebSocket(i),this.ws.onopen=()=>{console.log("[WebSocketService] Connected"),this.isConnecting=!1,this.reconnectAttempts=0,this.setConnectionState("connected"),this.subscriptions.forEach((l,o)=>{this.subscribe(o,l)})},this.ws.onmessage=l=>{try{const o=JSON.parse(l.data);this.handleEvent(o)}catch(o){console.error("[WebSocketService] Failed to parse message:",o)}},this.ws.onerror=l=>{console.error("[WebSocketService] Error:",l),this.isConnecting=!1},this.ws.onclose=()=>{if(console.log("[WebSocketService] Disconnected"),this.isConnecting=!1,this.setConnectionState("disconnected"),this.hookCount>0&&this.reconnectAttempts<this.maxReconnectAttempts){const l=this.getBackoffDelay(this.reconnectAttempts);console.log(`[WebSocketService] Reconnecting in ${l}ms (attempt ${this.reconnectAttempts+1}/${this.maxReconnectAttempts})`),this.reconnectTimeout=setTimeout(()=>{this.reconnectAttempts++,this.connect(t,n,r)},l)}}}catch(l){console.error("[WebSocketService] Failed to connect:",l),this.isConnecting=!1,this.setConnectionState("disconnected")}}disconnect(){this.reconnectTimeout&&(clearTimeout(this.reconnectTimeout),this.reconnectTimeout=null),this.ws&&(this.ws.close(),this.ws=null),this.wsUrl=null,this.reconnectAttempts=0,this.subscriptions.clear(),this.setConnectionState("disconnected")}send(t){this.ws&&this.ws.readyState===WebSocket.OPEN?this.ws.send(JSON.stringify(t)):console.warn("[WebSocketService] Not connected, cannot send")}handleEvent(t){switch(t.type){case"message":t.data&&this.messageHandlers.forEach(n=>n(t.data));break;case"batch":if(t.data){const n=t.data;this.batchHandlers.forEach(r=>r(n)),n.messages.forEach(r=>{this.messageHandlers.forEach(i=>i(r))})}break;case"error":t.data&&this.errorHandlers.forEach(n=>n(t.data)),console.error("[WebSocketService] Error event:",t.data);break;case"pong":break;default:console.log("[WebSocketService] Unknown event:",t.type)}}getBackoffDelay(t,n=1e3,r=3e4){const i=Math.min(n*Math.pow(2,t),r),l=i*Math.random()*.3;return Math.round(i+l)}subscribe(t,n=0){this.subscriptions.set(t,n),this.send({type:"subscribe",timestamp:Date.now(),data:{thread_id:t,from_seq:n}})}unsubscribe(t){this.subscriptions.delete(t)}acknowledge(t,n){const r=this.subscriptions.get(t)||0;n>r&&this.subscriptions.set(t,n),this.send({type:"ack",timestamp:Date.now(),data:{thread_id:t,ack_seq:n}})}ping(){this.send({type:"ping",timestamp:Date.now()})}}function fk(){return typeof window<"u"?(window.__AILANG_WS_SERVICE__?console.log("[WebSocketService] Reusing existing singleton instance"):(console.log("[WebSocketService] Creating new singleton instance"),window.__AILANG_WS_SERVICE__=new td),window.__AILANG_WS_SERVICE__):new td}const jt=fk();function pk(e){return jt.subscribeToState(e)}const hk=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,maxReconnectAttempts:l=10})=>{const[o,a]=F.useState(jt.getState().isConnected),[u,c]=F.useState(null),d=F.useRef(n),f=F.useRef(r),g=F.useRef(i);F.useEffect(()=>{d.current=n},[n]),F.useEffect(()=>{f.current=r},[r]),F.useEffect(()=>{g.current=i},[i]),F.useEffect(()=>{const h=N=>{d.current&&d.current(N)},v=N=>{f.current&&f.current(N)},x=N=>{g.current&&g.current(N)},b=jt.registerHook(h,v,x);return jt.connect(e,t,l),b},[e,t,l]),F.useEffect(()=>jt.subscribeToState((v,x)=>{a(v==="connected"),x>=l?c("Connection lost. Please refresh the page."):c(null)}),[l]),F.useEffect(()=>{if(!o)return;const h=setInterval(()=>{jt.ping()},3e4);return()=>clearInterval(h)},[o]);const p=F.useCallback((h,v=0)=>{jt.subscribe(h,v)},[]),k=F.useCallback(h=>{jt.unsubscribe(h)},[]),w=F.useCallback((h,v)=>{jt.acknowledge(h,v)},[]),z=F.useCallback(()=>{jt.ping()},[]);return{isConnected:o,connectionError:u,subscribe:p,unsubscribe:k,acknowledge:w,ping:z}},mk=({connected:e})=>s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?s.jsxs(s.Fragment,{children:[s.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),s.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):s.jsxs(s.Fragment,{children:[s.jsx("circle",{cx:"12",cy:"12",r:"10"}),s.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),s.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),gk=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=F.useState([]),[o,a]=F.useState(null),[u,c]=F.useState(new Map),[d,f]=F.useState(new Map),[g,p]=F.useState([]),[k,w]=F.useState(!1),[z,h]=F.useState(""),{isConnected:v,subscribe:x,acknowledge:b}=hk({url:e,instanceId:t,onMessage:N,onBatch:S});function N(m){const L={id:m.id,thread_id:m.thread_id,message_seq:m.message_seq,created_at:m.created_at,from_type:m.from_type,from_id:m.from_id,to_type:m.to_type,to_id:m.to_id,kind:m.kind,subject:m.subject,content:m.content,metadata_json:m.metadata_json,delivery_state:"visible",business_state:"open"};c(M=>{const y=M.get(L.thread_id)||[];return y.find(X=>X.id===L.id)?M:new Map(M).set(L.thread_id,[...y,L].sort((X,pe)=>X.message_seq-pe.message_seq))}),L.thread_id!==o&&f(M=>{const y=M.get(L.thread_id)||0;return new Map(M).set(L.thread_id,y+1)}),b(L.thread_id,L.message_seq)}function S(m){m.messages.forEach(L=>{N(L)})}const C=F.useCallback(m=>{if(a(m),f(L=>{const M=new Map(L);return M.delete(m),M}),v){const L=u.get(m)||[],M=L.length>0?Math.max(...L.map(y=>y.message_seq)):0;x(m,M)}},[v,x,u]),I=F.useCallback(async(m,L,M)=>{if(!o)return;const y=M?JSON.stringify({workspace:M}):void 0;try{const X=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:L,content:m,metadata_json:y})});if(!X.ok){console.error("Failed to send message:",await X.text());return}const pe=await X.json();c(Z=>{const xe=Z.get(o)||[];return xe.find(_e=>_e.id===pe.id)?Z:new Map(Z).set(o,[...xe,pe])})}catch(X){console.error("Error sending message:",X)}},[o,t]);F.useEffect(()=>{(async()=>{try{const L=await fetch("/api/threads");if(!L.ok){console.error("Failed to fetch threads:",await L.text());return}const M=await L.json();l(M),M.length>0&&!o&&a(M[0].id)}catch(L){console.error("Error fetching threads:",L)}})()},[]),F.useEffect(()=>{if(!o)return;const m=o;(async()=>{try{const M=await fetch(`/api/messages?thread_id=${m}`);if(!M.ok){console.error("Failed to fetch messages:",await M.text());return}const y=await M.json();c(X=>{const pe=X.get(m)||[],Z=y?[...y]:[];for(const xe of pe)Z.find(_e=>_e.id===xe.id)||Z.push(xe);return Z.sort((xe,_e)=>xe.message_seq-_e.message_seq),new Map(X).set(m,Z)})}catch(M){console.error("Error fetching messages:",M)}})()},[o]);const R=F.useRef(null);F.useEffect(()=>{n&&n!==R.current&&i.length>0&&(i.some(L=>L.id===n)&&(R.current=n,a(n),f(L=>{const M=new Map(L);return M.delete(n),M})),r&&r())},[n,i,r]);const P=F.useCallback(async m=>{try{const L=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:m,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!L.ok){console.error("Failed to create thread:",await L.text());return}const M=await L.json();l(y=>[M,...y]),a(M.id)}catch(L){console.error("Error creating thread:",L)}},[t]),j=F.useCallback(async()=>{try{const m=await fetch("/api/agents");if(!m.ok){console.error("Failed to fetch agents:",await m.text());return}const L=await m.json();p(L.running||[])}catch(m){console.error("Error fetching agents:",m)}},[]);F.useEffect(()=>{j();const m=setInterval(j,5e3);return()=>clearInterval(m)},[j]);const E=F.useCallback(async()=>{if(z.trim())try{const m=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:z.trim()})});if(!m.ok){const M=await m.text();console.error("Failed to launch agent:",M),alert(`Failed to launch agent: ${M}`);return}const L=await m.json();p(M=>[...M,L]),h(""),w(!1)}catch(m){console.error("Error launching agent:",m)}},[z]),U=F.useCallback(async m=>{try{const L=await fetch(`/api/agents/${m}`,{method:"DELETE"});if(!L.ok){console.error("Failed to stop agent:",await L.text());return}p(M=>M.filter(y=>y.instance_id!==m))}catch(L){console.error("Error stopping agent:",L)}},[]),V=F.useCallback(async m=>{if(o)try{const L=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:m})});if(!L.ok){console.error("Failed to update workspace:",await L.text());return}const M=await L.json();l(y=>y.map(X=>X.id===o?M:X))}catch(L){console.error("Error updating workspace:",L)}},[o]),W=F.useCallback(async m=>{try{const L=await fetch(`/api/threads/${m}`,{method:"DELETE"});if(!L.ok){console.error("Failed to delete thread:",await L.text());return}l(M=>M.filter(y=>y.id!==m)),c(M=>{const y=new Map(M);return y.delete(m),y}),f(M=>{const y=new Map(M);return y.delete(m),y}),o===m&&a(null)}catch(L){console.error("Error deleting thread:",L)}},[o]),K=F.useCallback(async(m,L)=>{try{const M=await fetch(`/api/threads/${m}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:L})});if(!M.ok){console.error("Failed to rename thread:",await M.text());return}const y=await M.json();l(X=>X.map(pe=>pe.id===m?y:pe))}catch(M){console.error("Error renaming thread:",M)}},[]),le=F.useCallback(async(m,L)=>{try{const M=await fetch(`/api/approvals/${m}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:L})});if(!M.ok){const y=await M.text();console.error("Failed to approve request:",y),alert(`Failed to approve: ${y}`);return}console.log("Approval approved successfully")}catch(M){console.error("Error approving request:",M)}},[]),_=F.useCallback(async(m,L)=>{try{const M=await fetch(`/api/approvals/${m}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:L})});if(!M.ok){const y=await M.text();console.error("Failed to reject request:",y),alert(`Failed to reject: ${y}`);return}console.log("Approval rejected successfully")}catch(M){console.error("Error rejecting request:",M)}},[]),B=o?u.get(o)||[]:[];return s.jsxs("div",{className:"message-center",children:[s.jsxs("div",{className:"status-bar",children:[s.jsxs("div",{className:`status-indicator ${v?"connected":"disconnected"}`,children:[s.jsx(mk,{connected:v}),s.jsx("span",{children:v?"Connected":"Disconnected"})]}),s.jsxs("div",{className:"status-meta",children:[s.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),s.jsxs("span",{className:"agent-count",children:[g.length," agents"]}),s.jsx("button",{className:"launch-agent-btn",onClick:()=>w(!0),children:"+ Agent"})]})]}),g.length>0&&s.jsx("div",{className:"agents-bar",children:g.map(m=>s.jsxs("div",{className:"agent-chip",children:[s.jsx("span",{className:"agent-pulse"}),s.jsx("span",{className:"agent-name",children:m.instance_id}),s.jsxs("span",{className:"agent-pid",children:["PID ",m.pid]}),s.jsx("button",{className:"agent-stop-btn",onClick:()=>U(m.instance_id),title:"Stop agent",children:"×"})]},m.instance_id))}),k&&s.jsx("div",{className:"modal-overlay",onClick:()=>w(!1),children:s.jsxs("div",{className:"modal-content",onClick:m=>m.stopPropagation(),children:[s.jsx("h3",{children:"Launch New Agent"}),s.jsx("input",{type:"text",value:z,onChange:m=>h(m.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:m=>{m.key==="Enter"&&E(),m.key==="Escape"&&w(!1)}}),s.jsxs("div",{className:"modal-actions",children:[s.jsx("button",{className:"cancel-btn",onClick:()=>w(!1),children:"Cancel"}),s.jsx("button",{className:"launch-btn",onClick:E,children:"Launch"})]})]})}),s.jsxs("div",{className:"center-layout",children:[s.jsx("aside",{className:"threads-panel",children:s.jsx(kv,{threads:i,selectedThreadId:o,onSelectThread:C,onCreateThread:P,onDeleteThread:W,onRenameThread:K,unreadCounts:d})}),s.jsx("main",{className:"conversation-panel",children:o?s.jsx(dk,{thread:i.find(m=>m.id===o),messages:B,onSendMessage:I,onWorkspaceChange:V,onApproveRequest:le,onRejectRequest:_}):s.jsxs("div",{className:"empty-state",children:[s.jsx("div",{className:"empty-icon",children:s.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:s.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),s.jsx("h3",{children:"Select a conversation"}),s.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),s.jsx("style",{children:`
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
      `})]})},Fe={check:s.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),s.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:s.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:s.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),s.jsx("circle",{cx:"12",cy:"5",r:"2"}),s.jsx("path",{d:"M12 7v4"})]}),dollar:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),s.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:s.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("circle",{cx:"12",cy:"12",r:"10"}),s.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:s.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:s.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:s.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),s.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),s.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},vk=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=F.useState(!0),[a,u]=F.useState(null),[c,d]=F.useState(new Map),f=h=>{try{return JSON.parse(h)}catch{return null}},g=h=>new Date(h).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),p=h=>{const v=c.get(h)||"";n(h,v),d(new Map(c.set(h,"")))},k=h=>{const v=c.get(h)||"";if(!v.trim()){alert("Please provide a reason for rejection");return}r(h,v),d(new Map(c.set(h,"")))},w=(h,v)=>{d(new Map(c.set(h,v)))},z=e.filter(h=>h.status==="pending");return s.jsxs("div",{className:"approval-queue",children:[s.jsx("div",{className:"queue-header",children:s.jsxs("div",{className:"header-title",children:[s.jsx("h2",{children:"Approval Queue"}),s.jsxs("span",{className:"pending-count",children:[z.length," pending"]})]})}),s.jsxs("div",{className:"approvals-container",children:[z.length===0?s.jsxs("div",{className:"empty-state",children:[s.jsx("div",{className:"empty-icon",children:Fe.sparkles}),s.jsx("h3",{children:"All caught up!"}),s.jsx("p",{children:"No pending approvals to review"})]}):s.jsx("div",{className:"approvals-list",children:z.map(h=>{const v=f(h.effect_delta_json),x=a===h.id;return s.jsxs("div",{className:`approval-card impact-${h.impact}`,children:[s.jsxs("div",{className:"card-header",onClick:()=>u(x?null:h.id),children:[s.jsxs("div",{className:"header-left",children:[s.jsx("div",{className:`impact-indicator ${h.impact}`}),s.jsxs("div",{className:"proposal-info",children:[s.jsx("span",{className:"proposal-text",children:h.proposal}),s.jsxs("div",{className:"proposal-meta",children:[h.thread_title&&s.jsxs("span",{className:"meta-item thread-link",onClick:b=>{b.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Fe.message,h.thread_title]}),s.jsxs("span",{className:"meta-item",children:[Fe.bot,h.instance_id]}),s.jsxs("span",{className:"meta-item",children:[Fe.clock,g(h.created_at)]})]})]})]}),s.jsxs("div",{className:"header-right",children:[s.jsxs("span",{className:"cost-badge",children:[Fe.dollar,"$",h.estimated_cost.toFixed(2)]}),s.jsx("span",{className:`impact-badge ${h.impact}`,children:h.impact}),s.jsx("button",{className:"expand-btn",children:x?Fe.chevronUp:Fe.chevronDown})]})]}),x&&s.jsxs("div",{className:"card-details",children:[v&&s.jsxs("div",{className:"detail-section",children:[s.jsx("h4",{children:"Effect Details"}),s.jsxs("div",{className:"detail-grid",children:[s.jsxs("div",{className:"detail-item",children:[s.jsx("span",{className:"detail-label",children:"Capability"}),s.jsx("span",{className:"detail-value code",children:v.cap_type})]}),s.jsxs("div",{className:"detail-item",children:[s.jsx("span",{className:"detail-label",children:"Budget Delta"}),s.jsxs("span",{className:"detail-value",children:["$",v.budget_delta.toFixed(2)]})]}),v.paths.length>0&&s.jsxs("div",{className:"detail-item full-width",children:[s.jsx("span",{className:"detail-label",children:"Paths"}),s.jsx("div",{className:"paths-list",children:v.paths.map((b,N)=>s.jsxs("span",{className:"path-tag",children:[Fe.folder,b]},N))})]})]})]}),s.jsxs("div",{className:"detail-section",children:[s.jsx("h4",{children:"Request Info"}),s.jsxs("div",{className:"detail-grid",children:[s.jsxs("div",{className:"detail-item",children:[s.jsx("span",{className:"detail-label",children:"Thread"}),s.jsx("span",{className:"detail-value code",children:h.thread_id})]}),s.jsxs("div",{className:"detail-item",children:[s.jsx("span",{className:"detail-label",children:"Impact Level"}),s.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]})]})]}),s.jsxs("div",{className:"review-section",children:[s.jsx("h4",{children:"Review Notes"}),s.jsx("textarea",{value:c.get(h.id)||"",onChange:b=>w(h.id,b.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),s.jsxs("div",{className:"action-buttons",children:[s.jsxs("button",{className:"reject-btn",onClick:()=>k(h.id),children:[Fe.x,"Reject"]}),s.jsxs("button",{className:"approve-btn",onClick:()=>p(h.id),children:[Fe.check,"Approve"]})]})]})]})]},h.id)})}),t.length>0&&s.jsxs("div",{className:"history-section",children:[s.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[s.jsxs("h3",{children:[l?Fe.chevronDown:Fe.chevronUp,"Review History"]}),s.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&s.jsx("div",{className:"history-list",children:t.map(h=>{const v=a===`history-${h.id}`;return s.jsxs("div",{className:`history-card ${h.status}`,onClick:()=>u(v?null:`history-${h.id}`),children:[s.jsxs("div",{className:"history-card-header",children:[s.jsxs("div",{className:"history-status",children:[s.jsx("span",{className:`status-icon ${h.status}`,children:h.status==="approved"?Fe.check:Fe.x}),s.jsxs("div",{className:"history-info",children:[s.jsx("span",{className:"history-proposal",children:h.proposal}),h.thread_title&&s.jsxs("span",{className:"history-thread",onClick:x=>{x.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Fe.message,h.thread_title]})]})]}),s.jsxs("div",{className:"history-meta",children:[s.jsx("span",{className:"history-agent",children:h.instance_id}),s.jsx("span",{className:`history-badge ${h.status}`,children:h.status}),s.jsx("span",{className:"history-time",children:h.reviewed_at?g(h.reviewed_at):g(h.created_at)})]})]}),v&&s.jsxs("div",{className:"history-details",children:[s.jsxs("div",{className:"detail-row",children:[s.jsx("span",{className:"detail-label",children:"Reviewed by"}),s.jsx("span",{className:"detail-value",children:h.reviewed_by||"Unknown"})]}),s.jsxs("div",{className:"detail-row",children:[s.jsx("span",{className:"detail-label",children:"Cost"}),s.jsxs("span",{className:"detail-value",children:["$",h.estimated_cost.toFixed(2)]})]}),s.jsxs("div",{className:"detail-row",children:[s.jsx("span",{className:"detail-label",children:"Impact"}),s.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]}),h.review_notes&&s.jsxs("div",{className:"detail-row full-width",children:[s.jsx("span",{className:"detail-label",children:"Notes"}),s.jsx("span",{className:"detail-value notes",children:h.review_notes})]})]})]},`history-${h.id}`)})})]})]}),s.jsx("style",{children:`
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
      `})]})},xk="_indicator_1ctaf_1",yk="_dot_1ctaf_12",kk="_connected_1ctaf_19",wk="_connecting_1ctaf_28",Sk="_disconnected_1ctaf_37",bk="_pulsing_1ctaf_46",_k="_text_1ctaf_61",Rt={indicator:xk,dot:yk,connected:kk,connecting:wk,disconnected:Sk,pulsing:bk,text:_k};function jk(){const[e,t]=F.useState("disconnected"),[n,r]=F.useState(0);if(F.useEffect(()=>pk((o,a)=>{t(o),r(a)}),[]),e==="connected")return s.jsx("div",{className:`${Rt.indicator} ${Rt.connected}`,title:"Connected",children:s.jsx("span",{className:Rt.dot})});const i=()=>{switch(e){case"connecting":return"Connecting...";case"reconnecting":return`Reconnecting... (${n})`;case"disconnected":return n>0?"Disconnected":"Offline";default:return"Unknown"}},l=()=>{switch(e){case"connecting":case"reconnecting":return Rt.connecting;case"disconnected":return Rt.disconnected;default:return""}};return s.jsxs("div",{className:`${Rt.indicator} ${l()}`,title:i(),children:[s.jsx("span",{className:`${Rt.dot} ${e==="connecting"||e==="reconnecting"?Rt.pulsing:""}`}),s.jsx("span",{className:Rt.text,children:i()})]})}const Ck="_taskExecutionPanel_3t8rx_3",Nk="_panelHeader_3t8rx_13",Ek="_headerLeft_3t8rx_22",Tk="_taskTitle_3t8rx_28",Lk="_threadId_3t8rx_35",Ik="_headerRight_3t8rx_40",zk="_statusBadge_3t8rx_46",Pk="_statusPending_3t8rx_53",Mk="_statusRunning_3t8rx_58",Ak="_statusCompleted_3t8rx_64",Rk="_statusFailed_3t8rx_69",Dk="_statusApproval_3t8rx_74",Fk="_cancelButton_3t8rx_85",Ok="_metricsSection_3t8rx_101",Bk="_resourceMetrics_3t8rx_106",$k="_resourceMetricsCompact_3t8rx_110",Hk="_metricItem_3t8rx_116",Uk="_metricsGrid_3t8rx_120",Vk="_metricCard_3t8rx_126",Wk="_metricLabel_3t8rx_132",Qk="_metricValue_3t8rx_139",qk="_metricBar_3t8rx_146",Kk="_metricBarFill_3t8rx_154",Yk="_metricPeak_3t8rx_161",Gk="_metricDetail_3t8rx_162",Xk="_metricsPlaceholder_3t8rx_172",Jk="_approvalSection_3t8rx_179",Zk="_approvalHeader_3t8rx_187",ew="_approvalIcon_3t8rx_196",tw="_approvalType_3t8rx_201",nw="_approvalContent_3t8rx_209",rw="_approvalDescription_3t8rx_213",iw="_filesChanged_3t8rx_218",lw="_toggleButton_3t8rx_222",ow="_fileList_3t8rx_236",aw="_diffSummary_3t8rx_257",sw="_approvalActions_3t8rx_269",uw="_approveButton_3t8rx_275",cw="_rejectButton_3t8rx_276",dw="_timeout_3t8rx_313",fw="_logSection_3t8rx_321",pw="_logHeader_3t8rx_328",hw="_eventCount_3t8rx_339",mw="_streamingLog_3t8rx_344",gw="_emptyLog_3t8rx_354",vw="_logLine_3t8rx_361",xw="_timestamp_3t8rx_373",yw="_icon_3t8rx_379",kw="_toolName_3t8rx_385",ww="_content_3t8rx_390",Sw="_logStdout_3t8rx_396",bw="_logError_3t8rx_400",_w="_logTool_3t8rx_408",jw="_logThinking_3t8rx_412",Cw="_logResult_3t8rx_421",Nw="_logStatus_3t8rx_429",Ew="_logMetrics_3t8rx_433",$={taskExecutionPanel:Ck,panelHeader:Nk,headerLeft:Ek,taskTitle:Tk,threadId:Lk,headerRight:Ik,statusBadge:zk,statusPending:Pk,statusRunning:Mk,statusCompleted:Ak,statusFailed:Rk,statusApproval:Dk,cancelButton:Fk,metricsSection:Ok,resourceMetrics:Bk,resourceMetricsCompact:$k,metricItem:Hk,metricsGrid:Uk,metricCard:Vk,metricLabel:Wk,metricValue:Qk,metricBar:qk,metricBarFill:Kk,metricPeak:Yk,metricDetail:Gk,metricsPlaceholder:Xk,approvalSection:Jk,approvalHeader:Zk,approvalIcon:ew,approvalType:tw,approvalContent:nw,approvalDescription:rw,filesChanged:iw,toggleButton:lw,fileList:ow,diffSummary:aw,approvalActions:sw,approveButton:uw,rejectButton:cw,timeout:dw,logSection:fw,logHeader:pw,eventCount:hw,streamingLog:mw,emptyLog:gw,logLine:vw,timestamp:xw,icon:yw,toolName:kw,content:ww,logStdout:Sw,logError:bw,logTool:_w,logThinking:jw,logResult:Cw,logStatus:Nw,logMetrics:Ew},Tw=e=>{switch(e){case"log":case"stdout":return">";case"stderr":return"!";case"tool_use":return"[T]";case"thinking":return"...";case"result":return"[R]";case"error":return"[E]";case"status":return"[S]";case"metrics":return"[M]";default:return""}},Lw=e=>{switch(e){case"stderr":case"error":return $.logError;case"tool_use":return $.logTool;case"thinking":return $.logThinking;case"result":return $.logResult;case"status":return $.logStatus;case"metrics":return $.logMetrics;default:return $.logStdout}},Iw=e=>new Date(e).toLocaleTimeString("en-US",{hour12:!1,hour:"2-digit",minute:"2-digit",second:"2-digit"}),zw=({events:e,maxLines:t=500,autoScroll:n=!0})=>{const r=F.useRef(null),i=F.useRef(!0);F.useEffect(()=>{const o=r.current;if(!o)return;const a=()=>{const{scrollTop:u,scrollHeight:c,clientHeight:d}=o;i.current=c-u-d<50};return o.addEventListener("scroll",a),()=>o.removeEventListener("scroll",a)},[]),F.useEffect(()=>{n&&i.current&&r.current&&(r.current.scrollTop=r.current.scrollHeight)},[e,n]);const l=e.length>t?e.slice(-t):e;return s.jsxs("div",{className:$.streamingLog,ref:r,children:[e.length===0&&s.jsx("div",{className:$.emptyLog,children:"Waiting for task events..."}),l.map((o,a)=>s.jsxs("div",{className:`${$.logLine} ${Lw(o.event_type)}`,children:[s.jsx("span",{className:$.timestamp,children:Iw(o.timestamp)}),s.jsx("span",{className:$.icon,children:Tw(o.event_type)}),o.tool_name&&s.jsxs("span",{className:$.toolName,children:["[",o.tool_name,"]"]}),s.jsx("span",{className:$.content,children:o.content||o.tool_input||o.status||""})]},`${o.timestamp}-${a}`))]})},Po=e=>e>=1024?`${(e/1024).toFixed(1)} GB`:`${e.toFixed(0)} MB`,nd=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,Fi=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}K`:e.toString(),Pw=({metrics:e,compact:t=!1})=>e?t?s.jsxs("div",{className:$.resourceMetricsCompact,children:[s.jsxs("span",{className:$.metricItem,children:["CPU: ",e.cpu_percent.toFixed(0),"%"]}),s.jsxs("span",{className:$.metricItem,children:["RAM: ",Po(e.memory_mb)]}),s.jsxs("span",{className:$.metricItem,children:["Tokens: ",Fi(e.tokens_in+e.tokens_out)]}),s.jsxs("span",{className:$.metricItem,children:["Cost: ",nd(e.cost)]})]}):s.jsx("div",{className:$.resourceMetrics,children:s.jsxs("div",{className:$.metricsGrid,children:[s.jsxs("div",{className:$.metricCard,children:[s.jsx("div",{className:$.metricLabel,children:"CPU"}),s.jsxs("div",{className:$.metricValue,children:[e.cpu_percent.toFixed(1),"%"]}),s.jsx("div",{className:$.metricBar,children:s.jsx("div",{className:$.metricBarFill,style:{width:`${Math.min(100,e.cpu_percent)}%`}})}),s.jsxs("div",{className:$.metricPeak,children:["Peak: ",e.peak_cpu.toFixed(1),"%"]})]}),s.jsxs("div",{className:$.metricCard,children:[s.jsx("div",{className:$.metricLabel,children:"Memory"}),s.jsx("div",{className:$.metricValue,children:Po(e.memory_mb)}),s.jsx("div",{className:$.metricBar,children:s.jsx("div",{className:$.metricBarFill,style:{width:`${Math.min(100,e.memory_mb/8192*100)}%`}})}),s.jsxs("div",{className:$.metricPeak,children:["Peak: ",Po(e.peak_memory)]})]}),s.jsxs("div",{className:$.metricCard,children:[s.jsx("div",{className:$.metricLabel,children:"Tokens"}),s.jsx("div",{className:$.metricValue,children:Fi(e.tokens_in+e.tokens_out)}),s.jsxs("div",{className:$.metricDetail,children:[s.jsxs("span",{children:["In: ",Fi(e.tokens_in)]}),s.jsxs("span",{children:["Out: ",Fi(e.tokens_out)]})]})]}),s.jsxs("div",{className:$.metricCard,children:[s.jsx("div",{className:$.metricLabel,children:"Cost"}),s.jsx("div",{className:$.metricValue,children:nd(e.cost)}),s.jsx("div",{className:$.metricDetail,children:s.jsx("span",{children:"Running total"})})]})]})}):s.jsx("div",{className:$.resourceMetrics,children:s.jsx("div",{className:$.metricsPlaceholder,children:"No metrics available"})}),Mw=e=>{switch(e){case"pending":return{label:"Pending",className:$.statusPending};case"running":return{label:"Running",className:$.statusRunning};case"completed":return{label:"Completed",className:$.statusCompleted};case"failed":return{label:"Failed",className:$.statusFailed};case"approval_pending":return{label:"Awaiting Approval",className:$.statusApproval};default:return{label:e,className:""}}},Aw=({taskId:e,threadId:t,events:n,metrics:r,pendingApproval:i,status:l,onApprove:o,onReject:a,onCancel:u})=>{const[c,d]=F.useState(!1),[f,g]=F.useState(!1),p=Mw(l),k=F.useCallback(async()=>{if(!(!i||!o)){d(!0);try{await o(i.id)}finally{d(!1)}}},[i,o]),w=F.useCallback(async()=>{if(!(!i||!a)){d(!0);try{await a(i.id)}finally{d(!1)}}},[i,a]);return F.useEffect(()=>{if(i&&l==="approval_pending")try{const z=new Audio("/notification.mp3");z.volume=.3,z.play().catch(()=>{})}catch{}},[i,l]),s.jsxs("div",{className:$.taskExecutionPanel,children:[s.jsxs("div",{className:$.panelHeader,children:[s.jsxs("div",{className:$.headerLeft,children:[s.jsxs("h3",{className:$.taskTitle,children:["Task: ",e]}),t&&s.jsxs("span",{className:$.threadId,children:["Thread: ",t]})]}),s.jsxs("div",{className:$.headerRight,children:[s.jsx("span",{className:`${$.statusBadge} ${p.className}`,children:p.label}),l==="running"&&u&&s.jsx("button",{className:$.cancelButton,onClick:u,children:"Cancel"})]})]}),s.jsx("div",{className:$.metricsSection,children:s.jsx(Pw,{metrics:r})}),i&&s.jsxs("div",{className:$.approvalSection,children:[s.jsxs("div",{className:$.approvalHeader,children:[s.jsx("span",{className:$.approvalIcon,children:"Approval Required"}),s.jsx("span",{className:$.approvalType,children:i.type})]}),s.jsxs("div",{className:$.approvalContent,children:[s.jsx("p",{className:$.approvalDescription,children:i.description}),i.files_changed&&i.files_changed.length>0&&s.jsxs("div",{className:$.filesChanged,children:[s.jsxs("button",{className:$.toggleButton,onClick:()=>g(!f),children:[f?"Hide":"Show"," Changed Files (",i.files_changed.length,")"]}),f&&s.jsx("ul",{className:$.fileList,children:i.files_changed.map((z,h)=>s.jsx("li",{children:z},h))})]}),f&&i.diff_summary&&s.jsx("pre",{className:$.diffSummary,children:i.diff_summary}),s.jsxs("div",{className:$.approvalActions,children:[s.jsx("button",{className:$.approveButton,onClick:k,disabled:c,children:c?"Processing...":"Approve"}),s.jsx("button",{className:$.rejectButton,onClick:w,disabled:c,children:c?"Processing...":"Reject"})]}),i.timeout_at&&s.jsxs("div",{className:$.timeout,children:["Expires: ",new Date(i.timeout_at).toLocaleTimeString()]})]})]}),s.jsxs("div",{className:$.logSection,children:[s.jsxs("div",{className:$.logHeader,children:[s.jsx("span",{children:"Live Output"}),s.jsxs("span",{className:$.eventCount,children:[n.length," events"]})]}),s.jsx(zw,{events:n})]})]})},Rw=s.jsx("img",{src:"/logo.png",alt:"AILANG",width:"28",height:"28"}),Dw=()=>{const[e,t]=F.useState({type:"overview"}),[n,r]=F.useState(null),[i,l]=F.useState([]),[o,a]=F.useState([]),[u,c]=F.useState(!1),[d,f]=F.useState(""),[g,p]=F.useState("..."),w=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;F.useEffect(()=>{(async()=>{try{const I=await fetch("/api/version");if(I.ok){const R=await I.json();p(R.version||"dev")}}catch(I){console.error("Error fetching version:",I),p("dev")}})()},[]),F.useEffect(()=>{const C=async()=>{try{const R=await fetch("/api/hierarchy");if(R.ok){const P=await R.json();r(P)}}catch(R){console.error("Error fetching hierarchy:",R)}};C();const I=setInterval(C,5e3);return()=>clearInterval(I)},[]),F.useEffect(()=>{const C=async()=>{try{const R=await fetch("/api/approvals?status=pending");if(R.ok){const U=await R.json();l(U)}const[P,j]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),E=[];if(P.ok){const U=await P.json();E.push(...U)}if(j.ok){const U=await j.json();E.push(...U)}E.sort((U,V)=>{const W=U.reviewed_at?new Date(U.reviewed_at).getTime():0;return(V.reviewed_at?new Date(V.reviewed_at).getTime():0)-W}),a(E)}catch(R){console.error("Error fetching approvals:",R)}};C();const I=setInterval(C,5e3);return()=>clearInterval(I)},[]);const z=async(C,I)=>{try{const R=await fetch(`/api/approvals/${C}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:I})});if(!R.ok){console.error("Failed to approve:",await R.text());return}const P=i.find(j=>j.id===C);if(P){const j={...P,status:"approved",reviewed_by:"user",review_notes:I,reviewed_at:Date.now()};a(E=>[j,...E])}l(j=>j.filter(E=>E.id!==C))}catch(R){console.error("Error approving:",R)}},h=async(C,I)=>{try{const R=await fetch(`/api/approvals/${C}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:I})});if(!R.ok){console.error("Failed to reject:",await R.text());return}const P=i.find(j=>j.id===C);if(P){const j={...P,status:"rejected",reviewed_by:"user",review_notes:I,reviewed_at:Date.now()};a(E=>[j,...E])}l(j=>j.filter(E=>E.id!==C))}catch(R){console.error("Error rejecting:",R)}},v=()=>{var I,R,P,j;const C=[{label:"All Agents",onClick:()=>t({type:"overview"})}];if(e.type==="agent"&&e.agentId&&C.push({label:e.agentId}),e.type==="thread"&&e.threadId){e.agentId&&C.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})});const E=(I=n==null?void 0:n.root.children)==null?void 0:I.find(V=>V.id===e.agentId),U=(R=E==null?void 0:E.children)==null?void 0:R.find(V=>V.id===e.threadId);C.push({label:(U==null?void 0:U.label)||"Thread"})}if(e.type==="task"&&e.taskId){if(e.agentId&&C.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})}),e.threadId){const E=(P=n==null?void 0:n.root.children)==null?void 0:P.find(V=>V.id===e.agentId),U=(j=E==null?void 0:E.children)==null?void 0:j.find(V=>V.id===e.threadId);C.push({label:(U==null?void 0:U.label)||"Thread",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:e.threadId})})}C.push({label:`Task ${e.taskId.slice(0,8)}...`})}return C},x=C=>{var R;const I=(R=n==null?void 0:n.root.children)==null?void 0:R.find(P=>{var j;return(j=P.children)==null?void 0:j.some(E=>E.id===C)});t({type:"thread",agentId:I==null?void 0:I.id,threadId:C})},b=async C=>{if(d.trim())try{const I=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:d.trim(),created_by_type:"human",created_by_id:"user",target_agent:C})});if(!I.ok){console.error("Failed to create thread:",await I.text());return}const R=await I.json();f(""),c(!1),t({type:"thread",agentId:C,threadId:R.id})}catch(I){console.error("Error creating thread:",I)}},N=()=>{var C,I,R;if(e.type==="overview"&&n)return s.jsx(xv,{aggregate:n.aggregate,agents:n.root.children||[],onSelectAgent:P=>t({type:"agent",agentId:P})});if(e.type==="agent"&&e.agentId){const P=(C=n==null?void 0:n.root.children)==null?void 0:C.find(E=>E.id===e.agentId),j=i.filter(E=>{var U;return(U=P==null?void 0:P.children)==null?void 0:U.some(V=>V.id===E.thread_id)});return s.jsxs("div",{className:"agent-view",children:[s.jsxs("div",{className:"agent-view-header",children:[s.jsx("h2",{children:e.agentId}),s.jsxs("span",{className:"agent-thread-count",children:[((I=P==null?void 0:P.children)==null?void 0:I.length)||0," threads"]})]}),s.jsxs("div",{className:"agent-metrics-section",children:[s.jsx("h3",{children:"Agent Metrics"}),s.jsx(Na,{scopeType:"agent",scopeId:e.agentId,title:""}),s.jsxs("div",{className:"agent-trends-grid",children:[s.jsx(_l,{scopeType:"agent",scopeId:e.agentId,period:"hour",limit:24,metric:"cost",title:"Cost (24h)"}),s.jsx(_l,{scopeType:"agent",scopeId:e.agentId,period:"hour",limit:24,metric:"tokens",title:"Tokens (24h)"})]})]}),s.jsxs("div",{className:"agent-view-content",children:[s.jsxs("div",{className:"agent-threads",children:[s.jsxs("div",{className:"threads-header",children:[s.jsx("h3",{children:"Threads"}),s.jsx("button",{className:"new-thread-btn",onClick:()=>c(!0),title:"New thread",children:"+ New Thread"})]}),u&&s.jsxs("div",{className:"new-thread-form",children:[s.jsx("input",{type:"text",value:d,onChange:E=>f(E.target.value),onKeyDown:E=>{E.key==="Enter"&&b(e.agentId),E.key==="Escape"&&(c(!1),f(""))},placeholder:"Thread title...",autoFocus:!0}),s.jsxs("div",{className:"form-actions",children:[s.jsx("button",{onClick:()=>{c(!1),f("")},children:"Cancel"}),s.jsx("button",{className:"create-btn",onClick:()=>b(e.agentId),children:"Create"})]})]}),(R=P==null?void 0:P.children)==null?void 0:R.map(E=>s.jsxs("div",{className:"thread-card",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:E.id}),children:[s.jsx("span",{className:"thread-title",children:E.label}),E.badges&&E.badges.length>0&&s.jsx("span",{className:"thread-badges",children:E.badges.map((U,V)=>s.jsx("span",{className:`badge badge-${U.type}`,children:U.count},V))})]},E.id)),(!(P!=null&&P.children)||P.children.length===0)&&!u&&s.jsxs("div",{className:"no-threads",children:["No threads yet",s.jsx("button",{className:"start-thread-btn",onClick:()=>c(!0),children:"Start a conversation"})]})]}),j.length>0&&s.jsxs("div",{className:"agent-approvals",children:[s.jsx("h3",{children:"Pending Approvals"}),s.jsx(vk,{approvals:j,history:[],onApprove:z,onReject:h,onNavigateToThread:x})]})]})]})}return e.type==="thread"&&e.threadId?s.jsxs("div",{className:"thread-view",children:[s.jsx("div",{className:"thread-metrics-bar",children:s.jsx(Na,{scopeType:"thread",scopeId:e.threadId,title:"Thread Metrics",compact:!0})}),s.jsx("div",{className:"thread-messages-container",children:s.jsx(gk,{websocketUrl:w,instanceId:e.agentId||"default",initialThreadId:e.threadId,onThreadNavigated:()=>{}})})]}):e.type==="task"&&e.taskId?s.jsx("div",{className:"task-view",children:s.jsx(Aw,{taskId:e.taskId,threadId:e.threadId,onCancel:()=>{e.threadId?t({type:"thread",agentId:e.agentId,threadId:e.threadId}):t({type:"overview"})}})}):s.jsx("div",{className:"empty-state",children:s.jsx("p",{children:"Select an agent or thread from the sidebar"})})},S=(i==null?void 0:i.filter(C=>C.status==="pending").length)||0;return s.jsxs("div",{className:"app",children:[s.jsxs("header",{className:"app-header",children:[s.jsxs("div",{className:"header-brand",children:[s.jsx("div",{className:"brand-logo",children:Rw}),s.jsxs("div",{className:"brand-text",children:[s.jsx("h1",{children:"AILANG"}),s.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),s.jsxs("div",{className:"header-meta",children:[s.jsx(jk,{}),S>0&&s.jsxs("span",{className:"pending-badge",title:`${S} pending approvals`,children:[S," pending"]}),s.jsxs("a",{href:"https://ailang.sunholo.com",target:"_blank",rel:"noopener noreferrer",className:"docs-link",title:"View documentation",children:[s.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[s.jsx("path",{d:"M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"}),s.jsx("polyline",{points:"15 3 21 3 21 9"}),s.jsx("line",{x1:"10",y1:"14",x2:"21",y2:"3"})]}),"Docs"]}),s.jsx("span",{className:"version-tag",children:g})]})]}),s.jsxs("div",{className:"app-body",children:[s.jsx("aside",{className:"app-sidebar",children:s.jsx(Dg,{selection:e,onSelect:t})}),s.jsxs("main",{className:"app-main",children:[e.type!=="overview"&&s.jsx(yv,{items:v()}),s.jsx("div",{className:"main-content",children:N()})]})]}),s.jsx("style",{children:`
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

        /* Responsive */
        @media (max-width: 768px) {
          .brand-text {
            display: none;
          }

          .app-sidebar {
            width: 60px;
          }
        }
      `})]})};Mo.createRoot(document.getElementById("root")).render(s.jsx(Xt.StrictMode,{children:s.jsx(Dw,{})}));
