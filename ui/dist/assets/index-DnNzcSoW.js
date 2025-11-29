(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Vi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Ea(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Rc={exports:{}},gl={},Fc={exports:{}},Q={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var Zr=Symbol.for("react.element"),Hp=Symbol.for("react.portal"),Vp=Symbol.for("react.fragment"),$p=Symbol.for("react.strict_mode"),Wp=Symbol.for("react.profiler"),Qp=Symbol.for("react.provider"),Kp=Symbol.for("react.context"),qp=Symbol.for("react.forward_ref"),Yp=Symbol.for("react.suspense"),Xp=Symbol.for("react.memo"),Gp=Symbol.for("react.lazy"),Fs=Symbol.iterator;function Jp(e){return e===null||typeof e!="object"?null:(e=Fs&&e[Fs]||e["@@iterator"],typeof e=="function"?e:null)}var Bc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Uc=Object.assign,Hc={};function nr(e,t,n){this.props=e,this.context=t,this.refs=Hc,this.updater=n||Bc}nr.prototype.isReactComponent={};nr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};nr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function Vc(){}Vc.prototype=nr.prototype;function ba(e,t,n){this.props=e,this.context=t,this.refs=Hc,this.updater=n||Bc}var ja=ba.prototype=new Vc;ja.constructor=ba;Uc(ja,nr.prototype);ja.isPureReactComponent=!0;var Bs=Array.isArray,$c=Object.prototype.hasOwnProperty,Na={current:null},Wc={key:!0,ref:!0,__self:!0,__source:!0};function Qc(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)$c.call(t,r)&&!Wc.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),u=0;u<a;u++)s[u]=arguments[u+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:Zr,type:e,key:l,ref:o,props:i,_owner:Na.current}}function Zp(e,t){return{$$typeof:Zr,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function _a(e){return typeof e=="object"&&e!==null&&e.$$typeof===Zr}function eh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Us=/\/+/g;function Dl(e,t){return typeof e=="object"&&e!==null&&e.key!=null?eh(""+e.key):t.toString(36)}function zi(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case Zr:case Hp:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Dl(o,0):r,Bs(i)?(n="",e!=null&&(n=e.replace(Us,"$&/")+"/"),zi(i,t,n,"",function(u){return u})):i!=null&&(_a(i)&&(i=Zp(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Us,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Bs(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Dl(l,a);o+=zi(l,t,n,s,i)}else if(s=Jp(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Dl(l,a++),o+=zi(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function ai(e,t,n){if(e==null)return e;var r=[],i=0;return zi(e,r,"","",function(l){return t.call(n,l,i++)}),r}function th(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Le={current:null},Pi={transition:null},nh={ReactCurrentDispatcher:Le,ReactCurrentBatchConfig:Pi,ReactCurrentOwner:Na};function Kc(){throw Error("act(...) is not supported in production builds of React.")}Q.Children={map:ai,forEach:function(e,t,n){ai(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return ai(e,function(){t++}),t},toArray:function(e){return ai(e,function(t){return t})||[]},only:function(e){if(!_a(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};Q.Component=nr;Q.Fragment=Vp;Q.Profiler=Wp;Q.PureComponent=ba;Q.StrictMode=$p;Q.Suspense=Yp;Q.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=nh;Q.act=Kc;Q.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Uc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Na.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)$c.call(t,s)&&!Wc.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var u=0;u<s;u++)a[u]=arguments[u+2];r.children=a}return{$$typeof:Zr,type:e.type,key:i,ref:l,props:r,_owner:o}};Q.createContext=function(e){return e={$$typeof:Kp,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:Qp,_context:e},e.Consumer=e};Q.createElement=Qc;Q.createFactory=function(e){var t=Qc.bind(null,e);return t.type=e,t};Q.createRef=function(){return{current:null}};Q.forwardRef=function(e){return{$$typeof:qp,render:e}};Q.isValidElement=_a;Q.lazy=function(e){return{$$typeof:Gp,_payload:{_status:-1,_result:e},_init:th}};Q.memo=function(e,t){return{$$typeof:Xp,type:e,compare:t===void 0?null:t}};Q.startTransition=function(e){var t=Pi.transition;Pi.transition={};try{e()}finally{Pi.transition=t}};Q.unstable_act=Kc;Q.useCallback=function(e,t){return Le.current.useCallback(e,t)};Q.useContext=function(e){return Le.current.useContext(e)};Q.useDebugValue=function(){};Q.useDeferredValue=function(e){return Le.current.useDeferredValue(e)};Q.useEffect=function(e,t){return Le.current.useEffect(e,t)};Q.useId=function(){return Le.current.useId()};Q.useImperativeHandle=function(e,t,n){return Le.current.useImperativeHandle(e,t,n)};Q.useInsertionEffect=function(e,t){return Le.current.useInsertionEffect(e,t)};Q.useLayoutEffect=function(e,t){return Le.current.useLayoutEffect(e,t)};Q.useMemo=function(e,t){return Le.current.useMemo(e,t)};Q.useReducer=function(e,t,n){return Le.current.useReducer(e,t,n)};Q.useRef=function(e){return Le.current.useRef(e)};Q.useState=function(e){return Le.current.useState(e)};Q.useSyncExternalStore=function(e,t,n){return Le.current.useSyncExternalStore(e,t,n)};Q.useTransition=function(){return Le.current.useTransition()};Q.version="18.3.1";Fc.exports=Q;var U=Fc.exports;const cn=Ea(U);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var rh=U,ih=Symbol.for("react.element"),lh=Symbol.for("react.fragment"),oh=Object.prototype.hasOwnProperty,ah=rh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,sh={key:!0,ref:!0,__self:!0,__source:!0};function qc(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)oh.call(t,r)&&!sh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:ih,type:e,key:l,ref:o,props:i,_owner:ah.current}}gl.Fragment=lh;gl.jsx=qc;gl.jsxs=qc;Rc.exports=gl;var c=Rc.exports,ko={},Yc={exports:{}},qe={},Xc={exports:{}},Gc={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(T,F){var y=T.length;T.push(F);e:for(;0<y;){var q=y-1>>>1,Z=T[q];if(0<i(Z,F))T[q]=F,T[y]=Z,y=q;else break e}}function n(T){return T.length===0?null:T[0]}function r(T){if(T.length===0)return null;var F=T[0],y=T.pop();if(y!==F){T[0]=y;e:for(var q=0,Z=T.length,x=Z>>>1;q<x;){var ge=2*(q+1)-1,it=T[ge],le=ge+1,pt=T[le];if(0>i(it,y))le<Z&&0>i(pt,it)?(T[q]=pt,T[le]=y,q=le):(T[q]=it,T[ge]=y,q=ge);else if(le<Z&&0>i(pt,y))T[q]=pt,T[le]=y,q=le;else break e}}return F}function i(T,F){var y=T.sortIndex-F.sortIndex;return y!==0?y:T.id-F.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],u=[],f=1,h=null,d=3,p=!1,k=!1,S=!1,b=typeof setTimeout=="function"?setTimeout:null,m=typeof clearTimeout=="function"?clearTimeout:null,g=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function v(T){for(var F=n(u);F!==null;){if(F.callback===null)r(u);else if(F.startTime<=T)r(u),F.sortIndex=F.expirationTime,t(s,F);else break;F=n(u)}}function C(T){if(S=!1,v(T),!k)if(n(s)!==null)k=!0,G(E);else{var F=n(u);F!==null&&te(C,F.startTime-T)}}function E(T,F){k=!1,S&&(S=!1,m(L),L=-1),p=!0;var y=d;try{for(v(F),h=n(s);h!==null&&(!(h.expirationTime>F)||T&&!O());){var q=h.callback;if(typeof q=="function"){h.callback=null,d=h.priorityLevel;var Z=q(h.expirationTime<=F);F=e.unstable_now(),typeof Z=="function"?h.callback=Z:h===n(s)&&r(s),v(F)}else r(s);h=n(s)}if(h!==null)var x=!0;else{var ge=n(u);ge!==null&&te(C,ge.startTime-F),x=!1}return x}finally{h=null,d=y,p=!1}}var w=!1,N=null,L=-1,R=5,D=-1;function O(){return!(e.unstable_now()-D<R)}function _(){if(N!==null){var T=e.unstable_now();D=T;var F=!0;try{F=N(!0,T)}finally{F?M():(w=!1,N=null)}}else w=!1}var M;if(typeof g=="function")M=function(){g(_)};else if(typeof MessageChannel<"u"){var B=new MessageChannel,H=B.port2;B.port1.onmessage=_,M=function(){H.postMessage(null)}}else M=function(){b(_,0)};function G(T){N=T,w||(w=!0,M())}function te(T,F){L=b(function(){T(e.unstable_now())},F)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(T){T.callback=null},e.unstable_continueExecution=function(){k||p||(k=!0,G(E))},e.unstable_forceFrameRate=function(T){0>T||125<T?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):R=0<T?Math.floor(1e3/T):5},e.unstable_getCurrentPriorityLevel=function(){return d},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(T){switch(d){case 1:case 2:case 3:var F=3;break;default:F=d}var y=d;d=F;try{return T()}finally{d=y}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(T,F){switch(T){case 1:case 2:case 3:case 4:case 5:break;default:T=3}var y=d;d=T;try{return F()}finally{d=y}},e.unstable_scheduleCallback=function(T,F,y){var q=e.unstable_now();switch(typeof y=="object"&&y!==null?(y=y.delay,y=typeof y=="number"&&0<y?q+y:q):y=q,T){case 1:var Z=-1;break;case 2:Z=250;break;case 5:Z=1073741823;break;case 4:Z=1e4;break;default:Z=5e3}return Z=y+Z,T={id:f++,callback:F,priorityLevel:T,startTime:y,expirationTime:Z,sortIndex:-1},y>q?(T.sortIndex=y,t(u,T),n(s)===null&&T===n(u)&&(S?(m(L),L=-1):S=!0,te(C,y-q))):(T.sortIndex=Z,t(s,T),k||p||(k=!0,G(E))),T},e.unstable_shouldYield=O,e.unstable_wrapCallback=function(T){var F=d;return function(){var y=d;d=F;try{return T.apply(this,arguments)}finally{d=y}}}})(Gc);Xc.exports=Gc;var uh=Xc.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ch=U,Ke=uh;function z(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var Jc=new Set,Ar={};function wn(e,t){Yn(e,t),Yn(e+"Capture",t)}function Yn(e,t){for(Ar[e]=t,e=0;e<t.length;e++)Jc.add(t[e])}var Lt=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),wo=Object.prototype.hasOwnProperty,dh=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,Hs={},Vs={};function fh(e){return wo.call(Vs,e)?!0:wo.call(Hs,e)?!1:dh.test(e)?Vs[e]=!0:(Hs[e]=!0,!1)}function ph(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function hh(e,t,n,r){if(t===null||typeof t>"u"||ph(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Te(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ce={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ce[e]=new Te(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ce[t]=new Te(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ce[e]=new Te(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ce[e]=new Te(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ce[e]=new Te(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ce[e]=new Te(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ce[e]=new Te(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ce[e]=new Te(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ce[e]=new Te(e,5,!1,e.toLowerCase(),null,!1,!1)});var za=/[\-:]([a-z])/g;function Pa(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(za,Pa);Ce[t]=new Te(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(za,Pa);Ce[t]=new Te(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(za,Pa);Ce[t]=new Te(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ce[e]=new Te(e,1,!1,e.toLowerCase(),null,!1,!1)});Ce.xlinkHref=new Te("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ce[e]=new Te(e,1,!1,e.toLowerCase(),null,!0,!0)});function La(e,t,n,r){var i=Ce.hasOwnProperty(t)?Ce[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(hh(t,n,i,r)&&(n=null),r||i===null?fh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var At=ch.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,si=Symbol.for("react.element"),zn=Symbol.for("react.portal"),Pn=Symbol.for("react.fragment"),Ta=Symbol.for("react.strict_mode"),So=Symbol.for("react.profiler"),Zc=Symbol.for("react.provider"),ed=Symbol.for("react.context"),Ia=Symbol.for("react.forward_ref"),Co=Symbol.for("react.suspense"),Eo=Symbol.for("react.suspense_list"),Ma=Symbol.for("react.memo"),Ft=Symbol.for("react.lazy"),td=Symbol.for("react.offscreen"),$s=Symbol.iterator;function ur(e){return e===null||typeof e!="object"?null:(e=$s&&e[$s]||e["@@iterator"],typeof e=="function"?e:null)}var de=Object.assign,Ol;function xr(e){if(Ol===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Ol=t&&t[1]||""}return`
`+Ol+e}var Rl=!1;function Fl(e,t){if(!e||Rl)return"";Rl=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(u){var r=u}Reflect.construct(e,[],t)}else{try{t.call()}catch(u){r=u}e.call(t.prototype)}else{try{throw Error()}catch(u){r=u}e()}}catch(u){if(u&&r&&typeof u.stack=="string"){for(var i=u.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Rl=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?xr(e):""}function mh(e){switch(e.tag){case 5:return xr(e.type);case 16:return xr("Lazy");case 13:return xr("Suspense");case 19:return xr("SuspenseList");case 0:case 2:case 15:return e=Fl(e.type,!1),e;case 11:return e=Fl(e.type.render,!1),e;case 1:return e=Fl(e.type,!0),e;default:return""}}function bo(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Pn:return"Fragment";case zn:return"Portal";case So:return"Profiler";case Ta:return"StrictMode";case Co:return"Suspense";case Eo:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case ed:return(e.displayName||"Context")+".Consumer";case Zc:return(e._context.displayName||"Context")+".Provider";case Ia:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Ma:return t=e.displayName||null,t!==null?t:bo(e.type)||"Memo";case Ft:t=e._payload,e=e._init;try{return bo(e(t))}catch{}}return null}function gh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return bo(t);case 8:return t===Ta?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function Zt(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function nd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function yh(e){var t=nd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function ui(e){e._valueTracker||(e._valueTracker=yh(e))}function rd(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=nd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function $i(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function jo(e,t){var n=t.checked;return de({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function Ws(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=Zt(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function id(e,t){t=t.checked,t!=null&&La(e,"checked",t,!1)}function No(e,t){id(e,t);var n=Zt(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?_o(e,t.type,n):t.hasOwnProperty("defaultValue")&&_o(e,t.type,Zt(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function Qs(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function _o(e,t,n){(t!=="number"||$i(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var kr=Array.isArray;function Un(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+Zt(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function zo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(z(91));return de({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function Ks(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(z(92));if(kr(n)){if(1<n.length)throw Error(z(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:Zt(n)}}function ld(e,t){var n=Zt(t.value),r=Zt(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function qs(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function od(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Po(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?od(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var ci,ad=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(ci=ci||document.createElement("div"),ci.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=ci.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Dr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Cr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},vh=["Webkit","ms","Moz","O"];Object.keys(Cr).forEach(function(e){vh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Cr[t]=Cr[e]})});function sd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Cr.hasOwnProperty(e)&&Cr[e]?(""+t).trim():t+"px"}function ud(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=sd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var xh=de({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Lo(e,t){if(t){if(xh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(z(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(z(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(z(61))}if(t.style!=null&&typeof t.style!="object")throw Error(z(62))}}function To(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Io=null;function Aa(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Mo=null,Hn=null,Vn=null;function Ys(e){if(e=ni(e)){if(typeof Mo!="function")throw Error(z(280));var t=e.stateNode;t&&(t=wl(t),Mo(e.stateNode,e.type,t))}}function cd(e){Hn?Vn?Vn.push(e):Vn=[e]:Hn=e}function dd(){if(Hn){var e=Hn,t=Vn;if(Vn=Hn=null,Ys(e),t)for(e=0;e<t.length;e++)Ys(t[e])}}function fd(e,t){return e(t)}function pd(){}var Bl=!1;function hd(e,t,n){if(Bl)return e(t,n);Bl=!0;try{return fd(e,t,n)}finally{Bl=!1,(Hn!==null||Vn!==null)&&(pd(),dd())}}function Or(e,t){var n=e.stateNode;if(n===null)return null;var r=wl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(z(231,t,typeof n));return n}var Ao=!1;if(Lt)try{var cr={};Object.defineProperty(cr,"passive",{get:function(){Ao=!0}}),window.addEventListener("test",cr,cr),window.removeEventListener("test",cr,cr)}catch{Ao=!1}function kh(e,t,n,r,i,l,o,a,s){var u=Array.prototype.slice.call(arguments,3);try{t.apply(n,u)}catch(f){this.onError(f)}}var Er=!1,Wi=null,Qi=!1,Do=null,wh={onError:function(e){Er=!0,Wi=e}};function Sh(e,t,n,r,i,l,o,a,s){Er=!1,Wi=null,kh.apply(wh,arguments)}function Ch(e,t,n,r,i,l,o,a,s){if(Sh.apply(this,arguments),Er){if(Er){var u=Wi;Er=!1,Wi=null}else throw Error(z(198));Qi||(Qi=!0,Do=u)}}function Sn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function md(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function Xs(e){if(Sn(e)!==e)throw Error(z(188))}function Eh(e){var t=e.alternate;if(!t){if(t=Sn(e),t===null)throw Error(z(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return Xs(i),e;if(l===r)return Xs(i),t;l=l.sibling}throw Error(z(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(z(189))}}if(n.alternate!==r)throw Error(z(190))}if(n.tag!==3)throw Error(z(188));return n.stateNode.current===n?e:t}function gd(e){return e=Eh(e),e!==null?yd(e):null}function yd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=yd(e);if(t!==null)return t;e=e.sibling}return null}var vd=Ke.unstable_scheduleCallback,Gs=Ke.unstable_cancelCallback,bh=Ke.unstable_shouldYield,jh=Ke.unstable_requestPaint,pe=Ke.unstable_now,Nh=Ke.unstable_getCurrentPriorityLevel,Da=Ke.unstable_ImmediatePriority,xd=Ke.unstable_UserBlockingPriority,Ki=Ke.unstable_NormalPriority,_h=Ke.unstable_LowPriority,kd=Ke.unstable_IdlePriority,yl=null,xt=null;function zh(e){if(xt&&typeof xt.onCommitFiberRoot=="function")try{xt.onCommitFiberRoot(yl,e,void 0,(e.current.flags&128)===128)}catch{}}var ct=Math.clz32?Math.clz32:Th,Ph=Math.log,Lh=Math.LN2;function Th(e){return e>>>=0,e===0?32:31-(Ph(e)/Lh|0)|0}var di=64,fi=4194304;function wr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function qi(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=wr(a):(l&=o,l!==0&&(r=wr(l)))}else o=n&~i,o!==0?r=wr(o):l!==0&&(r=wr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-ct(t),i=1<<n,r|=e[n],t&=~i;return r}function Ih(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Mh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-ct(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Ih(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Oo(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function wd(){var e=di;return di<<=1,!(di&4194240)&&(di=64),e}function Ul(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function ei(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-ct(t),e[t]=n}function Ah(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-ct(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Oa(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-ct(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var J=0;function Sd(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Cd,Ra,Ed,bd,jd,Ro=!1,pi=[],Wt=null,Qt=null,Kt=null,Rr=new Map,Fr=new Map,Ut=[],Dh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function Js(e,t){switch(e){case"focusin":case"focusout":Wt=null;break;case"dragenter":case"dragleave":Qt=null;break;case"mouseover":case"mouseout":Kt=null;break;case"pointerover":case"pointerout":Rr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Fr.delete(t.pointerId)}}function dr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=ni(t),t!==null&&Ra(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Oh(e,t,n,r,i){switch(t){case"focusin":return Wt=dr(Wt,e,t,n,r,i),!0;case"dragenter":return Qt=dr(Qt,e,t,n,r,i),!0;case"mouseover":return Kt=dr(Kt,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Rr.set(l,dr(Rr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Fr.set(l,dr(Fr.get(l)||null,e,t,n,r,i)),!0}return!1}function Nd(e){var t=dn(e.target);if(t!==null){var n=Sn(t);if(n!==null){if(t=n.tag,t===13){if(t=md(n),t!==null){e.blockedOn=t,jd(e.priority,function(){Ed(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Li(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Fo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Io=r,n.target.dispatchEvent(r),Io=null}else return t=ni(n),t!==null&&Ra(t),e.blockedOn=n,!1;t.shift()}return!0}function Zs(e,t,n){Li(e)&&n.delete(t)}function Rh(){Ro=!1,Wt!==null&&Li(Wt)&&(Wt=null),Qt!==null&&Li(Qt)&&(Qt=null),Kt!==null&&Li(Kt)&&(Kt=null),Rr.forEach(Zs),Fr.forEach(Zs)}function fr(e,t){e.blockedOn===t&&(e.blockedOn=null,Ro||(Ro=!0,Ke.unstable_scheduleCallback(Ke.unstable_NormalPriority,Rh)))}function Br(e){function t(i){return fr(i,e)}if(0<pi.length){fr(pi[0],e);for(var n=1;n<pi.length;n++){var r=pi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(Wt!==null&&fr(Wt,e),Qt!==null&&fr(Qt,e),Kt!==null&&fr(Kt,e),Rr.forEach(t),Fr.forEach(t),n=0;n<Ut.length;n++)r=Ut[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Ut.length&&(n=Ut[0],n.blockedOn===null);)Nd(n),n.blockedOn===null&&Ut.shift()}var $n=At.ReactCurrentBatchConfig,Yi=!0;function Fh(e,t,n,r){var i=J,l=$n.transition;$n.transition=null;try{J=1,Fa(e,t,n,r)}finally{J=i,$n.transition=l}}function Bh(e,t,n,r){var i=J,l=$n.transition;$n.transition=null;try{J=4,Fa(e,t,n,r)}finally{J=i,$n.transition=l}}function Fa(e,t,n,r){if(Yi){var i=Fo(e,t,n,r);if(i===null)Gl(e,t,r,Xi,n),Js(e,r);else if(Oh(i,e,t,n,r))r.stopPropagation();else if(Js(e,r),t&4&&-1<Dh.indexOf(e)){for(;i!==null;){var l=ni(i);if(l!==null&&Cd(l),l=Fo(e,t,n,r),l===null&&Gl(e,t,r,Xi,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else Gl(e,t,r,null,n)}}var Xi=null;function Fo(e,t,n,r){if(Xi=null,e=Aa(r),e=dn(e),e!==null)if(t=Sn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=md(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return Xi=e,null}function _d(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(Nh()){case Da:return 1;case xd:return 4;case Ki:case _h:return 16;case kd:return 536870912;default:return 16}default:return 16}}var Vt=null,Ba=null,Ti=null;function zd(){if(Ti)return Ti;var e,t=Ba,n=t.length,r,i="value"in Vt?Vt.value:Vt.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Ti=i.slice(e,1<r?1-r:void 0)}function Ii(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function hi(){return!0}function eu(){return!1}function Ye(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?hi:eu,this.isPropagationStopped=eu,this}return de(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=hi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=hi)},persist:function(){},isPersistent:hi}),t}var rr={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Ua=Ye(rr),ti=de({},rr,{view:0,detail:0}),Uh=Ye(ti),Hl,Vl,pr,vl=de({},ti,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:Ha,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==pr&&(pr&&e.type==="mousemove"?(Hl=e.screenX-pr.screenX,Vl=e.screenY-pr.screenY):Vl=Hl=0,pr=e),Hl)},movementY:function(e){return"movementY"in e?e.movementY:Vl}}),tu=Ye(vl),Hh=de({},vl,{dataTransfer:0}),Vh=Ye(Hh),$h=de({},ti,{relatedTarget:0}),$l=Ye($h),Wh=de({},rr,{animationName:0,elapsedTime:0,pseudoElement:0}),Qh=Ye(Wh),Kh=de({},rr,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),qh=Ye(Kh),Yh=de({},rr,{data:0}),nu=Ye(Yh),Xh={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},Gh={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},Jh={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function Zh(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=Jh[e])?!!t[e]:!1}function Ha(){return Zh}var em=de({},ti,{key:function(e){if(e.key){var t=Xh[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Ii(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?Gh[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:Ha,charCode:function(e){return e.type==="keypress"?Ii(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Ii(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),tm=Ye(em),nm=de({},vl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),ru=Ye(nm),rm=de({},ti,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:Ha}),im=Ye(rm),lm=de({},rr,{propertyName:0,elapsedTime:0,pseudoElement:0}),om=Ye(lm),am=de({},vl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),sm=Ye(am),um=[9,13,27,32],Va=Lt&&"CompositionEvent"in window,br=null;Lt&&"documentMode"in document&&(br=document.documentMode);var cm=Lt&&"TextEvent"in window&&!br,Pd=Lt&&(!Va||br&&8<br&&11>=br),iu=" ",lu=!1;function Ld(e,t){switch(e){case"keyup":return um.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Td(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Ln=!1;function dm(e,t){switch(e){case"compositionend":return Td(t);case"keypress":return t.which!==32?null:(lu=!0,iu);case"textInput":return e=t.data,e===iu&&lu?null:e;default:return null}}function fm(e,t){if(Ln)return e==="compositionend"||!Va&&Ld(e,t)?(e=zd(),Ti=Ba=Vt=null,Ln=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Pd&&t.locale!=="ko"?null:t.data;default:return null}}var pm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function ou(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!pm[e.type]:t==="textarea"}function Id(e,t,n,r){cd(r),t=Gi(t,"onChange"),0<t.length&&(n=new Ua("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var jr=null,Ur=null;function hm(e){$d(e,0)}function xl(e){var t=Mn(e);if(rd(t))return e}function mm(e,t){if(e==="change")return t}var Md=!1;if(Lt){var Wl;if(Lt){var Ql="oninput"in document;if(!Ql){var au=document.createElement("div");au.setAttribute("oninput","return;"),Ql=typeof au.oninput=="function"}Wl=Ql}else Wl=!1;Md=Wl&&(!document.documentMode||9<document.documentMode)}function su(){jr&&(jr.detachEvent("onpropertychange",Ad),Ur=jr=null)}function Ad(e){if(e.propertyName==="value"&&xl(Ur)){var t=[];Id(t,Ur,e,Aa(e)),hd(hm,t)}}function gm(e,t,n){e==="focusin"?(su(),jr=t,Ur=n,jr.attachEvent("onpropertychange",Ad)):e==="focusout"&&su()}function ym(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return xl(Ur)}function vm(e,t){if(e==="click")return xl(t)}function xm(e,t){if(e==="input"||e==="change")return xl(t)}function km(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var ft=typeof Object.is=="function"?Object.is:km;function Hr(e,t){if(ft(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!wo.call(t,i)||!ft(e[i],t[i]))return!1}return!0}function uu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function cu(e,t){var n=uu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=uu(n)}}function Dd(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Dd(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Od(){for(var e=window,t=$i();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=$i(e.document)}return t}function $a(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function wm(e){var t=Od(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Dd(n.ownerDocument.documentElement,n)){if(r!==null&&$a(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=cu(n,l);var o=cu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Sm=Lt&&"documentMode"in document&&11>=document.documentMode,Tn=null,Bo=null,Nr=null,Uo=!1;function du(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Uo||Tn==null||Tn!==$i(r)||(r=Tn,"selectionStart"in r&&$a(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Nr&&Hr(Nr,r)||(Nr=r,r=Gi(Bo,"onSelect"),0<r.length&&(t=new Ua("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Tn)))}function mi(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var In={animationend:mi("Animation","AnimationEnd"),animationiteration:mi("Animation","AnimationIteration"),animationstart:mi("Animation","AnimationStart"),transitionend:mi("Transition","TransitionEnd")},Kl={},Rd={};Lt&&(Rd=document.createElement("div").style,"AnimationEvent"in window||(delete In.animationend.animation,delete In.animationiteration.animation,delete In.animationstart.animation),"TransitionEvent"in window||delete In.transitionend.transition);function kl(e){if(Kl[e])return Kl[e];if(!In[e])return e;var t=In[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Rd)return Kl[e]=t[n];return e}var Fd=kl("animationend"),Bd=kl("animationiteration"),Ud=kl("animationstart"),Hd=kl("transitionend"),Vd=new Map,fu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function tn(e,t){Vd.set(e,t),wn(t,[e])}for(var ql=0;ql<fu.length;ql++){var Yl=fu[ql],Cm=Yl.toLowerCase(),Em=Yl[0].toUpperCase()+Yl.slice(1);tn(Cm,"on"+Em)}tn(Fd,"onAnimationEnd");tn(Bd,"onAnimationIteration");tn(Ud,"onAnimationStart");tn("dblclick","onDoubleClick");tn("focusin","onFocus");tn("focusout","onBlur");tn(Hd,"onTransitionEnd");Yn("onMouseEnter",["mouseout","mouseover"]);Yn("onMouseLeave",["mouseout","mouseover"]);Yn("onPointerEnter",["pointerout","pointerover"]);Yn("onPointerLeave",["pointerout","pointerover"]);wn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));wn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));wn("onBeforeInput",["compositionend","keypress","textInput","paste"]);wn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));wn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));wn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var Sr="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),bm=new Set("cancel close invalid load scroll toggle".split(" ").concat(Sr));function pu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,Ch(r,t,void 0,e),e.currentTarget=null}function $d(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,u=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;pu(i,a,u),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,u=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;pu(i,a,u),l=s}}}if(Qi)throw e=Do,Qi=!1,Do=null,e}function oe(e,t){var n=t[Qo];n===void 0&&(n=t[Qo]=new Set);var r=e+"__bubble";n.has(r)||(Wd(t,e,2,!1),n.add(r))}function Xl(e,t,n){var r=0;t&&(r|=4),Wd(n,e,r,t)}var gi="_reactListening"+Math.random().toString(36).slice(2);function Vr(e){if(!e[gi]){e[gi]=!0,Jc.forEach(function(n){n!=="selectionchange"&&(bm.has(n)||Xl(n,!1,e),Xl(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[gi]||(t[gi]=!0,Xl("selectionchange",!1,t))}}function Wd(e,t,n,r){switch(_d(t)){case 1:var i=Fh;break;case 4:i=Bh;break;default:i=Fa}n=i.bind(null,t,n,e),i=void 0,!Ao||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function Gl(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=dn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}hd(function(){var u=l,f=Aa(n),h=[];e:{var d=Vd.get(e);if(d!==void 0){var p=Ua,k=e;switch(e){case"keypress":if(Ii(n)===0)break e;case"keydown":case"keyup":p=tm;break;case"focusin":k="focus",p=$l;break;case"focusout":k="blur",p=$l;break;case"beforeblur":case"afterblur":p=$l;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":p=tu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":p=Vh;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":p=im;break;case Fd:case Bd:case Ud:p=Qh;break;case Hd:p=om;break;case"scroll":p=Uh;break;case"wheel":p=sm;break;case"copy":case"cut":case"paste":p=qh;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":p=ru}var S=(t&4)!==0,b=!S&&e==="scroll",m=S?d!==null?d+"Capture":null:d;S=[];for(var g=u,v;g!==null;){v=g;var C=v.stateNode;if(v.tag===5&&C!==null&&(v=C,m!==null&&(C=Or(g,m),C!=null&&S.push($r(g,C,v)))),b)break;g=g.return}0<S.length&&(d=new p(d,k,null,n,f),h.push({event:d,listeners:S}))}}if(!(t&7)){e:{if(d=e==="mouseover"||e==="pointerover",p=e==="mouseout"||e==="pointerout",d&&n!==Io&&(k=n.relatedTarget||n.fromElement)&&(dn(k)||k[Tt]))break e;if((p||d)&&(d=f.window===f?f:(d=f.ownerDocument)?d.defaultView||d.parentWindow:window,p?(k=n.relatedTarget||n.toElement,p=u,k=k?dn(k):null,k!==null&&(b=Sn(k),k!==b||k.tag!==5&&k.tag!==6)&&(k=null)):(p=null,k=u),p!==k)){if(S=tu,C="onMouseLeave",m="onMouseEnter",g="mouse",(e==="pointerout"||e==="pointerover")&&(S=ru,C="onPointerLeave",m="onPointerEnter",g="pointer"),b=p==null?d:Mn(p),v=k==null?d:Mn(k),d=new S(C,g+"leave",p,n,f),d.target=b,d.relatedTarget=v,C=null,dn(f)===u&&(S=new S(m,g+"enter",k,n,f),S.target=v,S.relatedTarget=b,C=S),b=C,p&&k)t:{for(S=p,m=k,g=0,v=S;v;v=Nn(v))g++;for(v=0,C=m;C;C=Nn(C))v++;for(;0<g-v;)S=Nn(S),g--;for(;0<v-g;)m=Nn(m),v--;for(;g--;){if(S===m||m!==null&&S===m.alternate)break t;S=Nn(S),m=Nn(m)}S=null}else S=null;p!==null&&hu(h,d,p,S,!1),k!==null&&b!==null&&hu(h,b,k,S,!0)}}e:{if(d=u?Mn(u):window,p=d.nodeName&&d.nodeName.toLowerCase(),p==="select"||p==="input"&&d.type==="file")var E=mm;else if(ou(d))if(Md)E=xm;else{E=ym;var w=gm}else(p=d.nodeName)&&p.toLowerCase()==="input"&&(d.type==="checkbox"||d.type==="radio")&&(E=vm);if(E&&(E=E(e,u))){Id(h,E,n,f);break e}w&&w(e,d,u),e==="focusout"&&(w=d._wrapperState)&&w.controlled&&d.type==="number"&&_o(d,"number",d.value)}switch(w=u?Mn(u):window,e){case"focusin":(ou(w)||w.contentEditable==="true")&&(Tn=w,Bo=u,Nr=null);break;case"focusout":Nr=Bo=Tn=null;break;case"mousedown":Uo=!0;break;case"contextmenu":case"mouseup":case"dragend":Uo=!1,du(h,n,f);break;case"selectionchange":if(Sm)break;case"keydown":case"keyup":du(h,n,f)}var N;if(Va)e:{switch(e){case"compositionstart":var L="onCompositionStart";break e;case"compositionend":L="onCompositionEnd";break e;case"compositionupdate":L="onCompositionUpdate";break e}L=void 0}else Ln?Ld(e,n)&&(L="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(L="onCompositionStart");L&&(Pd&&n.locale!=="ko"&&(Ln||L!=="onCompositionStart"?L==="onCompositionEnd"&&Ln&&(N=zd()):(Vt=f,Ba="value"in Vt?Vt.value:Vt.textContent,Ln=!0)),w=Gi(u,L),0<w.length&&(L=new nu(L,e,null,n,f),h.push({event:L,listeners:w}),N?L.data=N:(N=Td(n),N!==null&&(L.data=N)))),(N=cm?dm(e,n):fm(e,n))&&(u=Gi(u,"onBeforeInput"),0<u.length&&(f=new nu("onBeforeInput","beforeinput",null,n,f),h.push({event:f,listeners:u}),f.data=N))}$d(h,t)})}function $r(e,t,n){return{instance:e,listener:t,currentTarget:n}}function Gi(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Or(e,n),l!=null&&r.unshift($r(e,l,i)),l=Or(e,t),l!=null&&r.push($r(e,l,i))),e=e.return}return r}function Nn(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function hu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,u=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&u!==null&&(a=u,i?(s=Or(n,l),s!=null&&o.unshift($r(n,s,a))):i||(s=Or(n,l),s!=null&&o.push($r(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var jm=/\r\n?/g,Nm=/\u0000|\uFFFD/g;function mu(e){return(typeof e=="string"?e:""+e).replace(jm,`
`).replace(Nm,"")}function yi(e,t,n){if(t=mu(t),mu(e)!==t&&n)throw Error(z(425))}function Ji(){}var Ho=null,Vo=null;function $o(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var Wo=typeof setTimeout=="function"?setTimeout:void 0,_m=typeof clearTimeout=="function"?clearTimeout:void 0,gu=typeof Promise=="function"?Promise:void 0,zm=typeof queueMicrotask=="function"?queueMicrotask:typeof gu<"u"?function(e){return gu.resolve(null).then(e).catch(Pm)}:Wo;function Pm(e){setTimeout(function(){throw e})}function Jl(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Br(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Br(t)}function qt(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function yu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var ir=Math.random().toString(36).slice(2),yt="__reactFiber$"+ir,Wr="__reactProps$"+ir,Tt="__reactContainer$"+ir,Qo="__reactEvents$"+ir,Lm="__reactListeners$"+ir,Tm="__reactHandles$"+ir;function dn(e){var t=e[yt];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Tt]||n[yt]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=yu(e);e!==null;){if(n=e[yt])return n;e=yu(e)}return t}e=n,n=e.parentNode}return null}function ni(e){return e=e[yt]||e[Tt],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function Mn(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(z(33))}function wl(e){return e[Wr]||null}var Ko=[],An=-1;function nn(e){return{current:e}}function ae(e){0>An||(e.current=Ko[An],Ko[An]=null,An--)}function re(e,t){An++,Ko[An]=e.current,e.current=t}var en={},Ne=nn(en),De=nn(!1),gn=en;function Xn(e,t){var n=e.type.contextTypes;if(!n)return en;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Oe(e){return e=e.childContextTypes,e!=null}function Zi(){ae(De),ae(Ne)}function vu(e,t,n){if(Ne.current!==en)throw Error(z(168));re(Ne,t),re(De,n)}function Qd(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(z(108,gh(e)||"Unknown",i));return de({},n,r)}function el(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||en,gn=Ne.current,re(Ne,e),re(De,De.current),!0}function xu(e,t,n){var r=e.stateNode;if(!r)throw Error(z(169));n?(e=Qd(e,t,gn),r.__reactInternalMemoizedMergedChildContext=e,ae(De),ae(Ne),re(Ne,e)):ae(De),re(De,n)}var jt=null,Sl=!1,Zl=!1;function Kd(e){jt===null?jt=[e]:jt.push(e)}function Im(e){Sl=!0,Kd(e)}function rn(){if(!Zl&&jt!==null){Zl=!0;var e=0,t=J;try{var n=jt;for(J=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}jt=null,Sl=!1}catch(i){throw jt!==null&&(jt=jt.slice(e+1)),vd(Da,rn),i}finally{J=t,Zl=!1}}return null}var Dn=[],On=0,tl=null,nl=0,Ge=[],Je=0,yn=null,_t=1,zt="";function an(e,t){Dn[On++]=nl,Dn[On++]=tl,tl=e,nl=t}function qd(e,t,n){Ge[Je++]=_t,Ge[Je++]=zt,Ge[Je++]=yn,yn=e;var r=_t;e=zt;var i=32-ct(r)-1;r&=~(1<<i),n+=1;var l=32-ct(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,_t=1<<32-ct(t)+i|n<<i|r,zt=l+e}else _t=1<<l|n<<i|r,zt=e}function Wa(e){e.return!==null&&(an(e,1),qd(e,1,0))}function Qa(e){for(;e===tl;)tl=Dn[--On],Dn[On]=null,nl=Dn[--On],Dn[On]=null;for(;e===yn;)yn=Ge[--Je],Ge[Je]=null,zt=Ge[--Je],Ge[Je]=null,_t=Ge[--Je],Ge[Je]=null}var Qe=null,$e=null,se=!1,ut=null;function Yd(e,t){var n=et(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function ku(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,Qe=e,$e=qt(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,Qe=e,$e=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=yn!==null?{id:_t,overflow:zt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=et(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,Qe=e,$e=null,!0):!1;default:return!1}}function qo(e){return(e.mode&1)!==0&&(e.flags&128)===0}function Yo(e){if(se){var t=$e;if(t){var n=t;if(!ku(e,t)){if(qo(e))throw Error(z(418));t=qt(n.nextSibling);var r=Qe;t&&ku(e,t)?Yd(r,n):(e.flags=e.flags&-4097|2,se=!1,Qe=e)}}else{if(qo(e))throw Error(z(418));e.flags=e.flags&-4097|2,se=!1,Qe=e}}}function wu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;Qe=e}function vi(e){if(e!==Qe)return!1;if(!se)return wu(e),se=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!$o(e.type,e.memoizedProps)),t&&(t=$e)){if(qo(e))throw Xd(),Error(z(418));for(;t;)Yd(e,t),t=qt(t.nextSibling)}if(wu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(z(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){$e=qt(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}$e=null}}else $e=Qe?qt(e.stateNode.nextSibling):null;return!0}function Xd(){for(var e=$e;e;)e=qt(e.nextSibling)}function Gn(){$e=Qe=null,se=!1}function Ka(e){ut===null?ut=[e]:ut.push(e)}var Mm=At.ReactCurrentBatchConfig;function hr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(z(309));var r=n.stateNode}if(!r)throw Error(z(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(z(284));if(!n._owner)throw Error(z(290,e))}return e}function xi(e,t){throw e=Object.prototype.toString.call(t),Error(z(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Su(e){var t=e._init;return t(e._payload)}function Gd(e){function t(m,g){if(e){var v=m.deletions;v===null?(m.deletions=[g],m.flags|=16):v.push(g)}}function n(m,g){if(!e)return null;for(;g!==null;)t(m,g),g=g.sibling;return null}function r(m,g){for(m=new Map;g!==null;)g.key!==null?m.set(g.key,g):m.set(g.index,g),g=g.sibling;return m}function i(m,g){return m=Jt(m,g),m.index=0,m.sibling=null,m}function l(m,g,v){return m.index=v,e?(v=m.alternate,v!==null?(v=v.index,v<g?(m.flags|=2,g):v):(m.flags|=2,g)):(m.flags|=1048576,g)}function o(m){return e&&m.alternate===null&&(m.flags|=2),m}function a(m,g,v,C){return g===null||g.tag!==6?(g=oo(v,m.mode,C),g.return=m,g):(g=i(g,v),g.return=m,g)}function s(m,g,v,C){var E=v.type;return E===Pn?f(m,g,v.props.children,C,v.key):g!==null&&(g.elementType===E||typeof E=="object"&&E!==null&&E.$$typeof===Ft&&Su(E)===g.type)?(C=i(g,v.props),C.ref=hr(m,g,v),C.return=m,C):(C=Bi(v.type,v.key,v.props,null,m.mode,C),C.ref=hr(m,g,v),C.return=m,C)}function u(m,g,v,C){return g===null||g.tag!==4||g.stateNode.containerInfo!==v.containerInfo||g.stateNode.implementation!==v.implementation?(g=ao(v,m.mode,C),g.return=m,g):(g=i(g,v.children||[]),g.return=m,g)}function f(m,g,v,C,E){return g===null||g.tag!==7?(g=mn(v,m.mode,C,E),g.return=m,g):(g=i(g,v),g.return=m,g)}function h(m,g,v){if(typeof g=="string"&&g!==""||typeof g=="number")return g=oo(""+g,m.mode,v),g.return=m,g;if(typeof g=="object"&&g!==null){switch(g.$$typeof){case si:return v=Bi(g.type,g.key,g.props,null,m.mode,v),v.ref=hr(m,null,g),v.return=m,v;case zn:return g=ao(g,m.mode,v),g.return=m,g;case Ft:var C=g._init;return h(m,C(g._payload),v)}if(kr(g)||ur(g))return g=mn(g,m.mode,v,null),g.return=m,g;xi(m,g)}return null}function d(m,g,v,C){var E=g!==null?g.key:null;if(typeof v=="string"&&v!==""||typeof v=="number")return E!==null?null:a(m,g,""+v,C);if(typeof v=="object"&&v!==null){switch(v.$$typeof){case si:return v.key===E?s(m,g,v,C):null;case zn:return v.key===E?u(m,g,v,C):null;case Ft:return E=v._init,d(m,g,E(v._payload),C)}if(kr(v)||ur(v))return E!==null?null:f(m,g,v,C,null);xi(m,v)}return null}function p(m,g,v,C,E){if(typeof C=="string"&&C!==""||typeof C=="number")return m=m.get(v)||null,a(g,m,""+C,E);if(typeof C=="object"&&C!==null){switch(C.$$typeof){case si:return m=m.get(C.key===null?v:C.key)||null,s(g,m,C,E);case zn:return m=m.get(C.key===null?v:C.key)||null,u(g,m,C,E);case Ft:var w=C._init;return p(m,g,v,w(C._payload),E)}if(kr(C)||ur(C))return m=m.get(v)||null,f(g,m,C,E,null);xi(g,C)}return null}function k(m,g,v,C){for(var E=null,w=null,N=g,L=g=0,R=null;N!==null&&L<v.length;L++){N.index>L?(R=N,N=null):R=N.sibling;var D=d(m,N,v[L],C);if(D===null){N===null&&(N=R);break}e&&N&&D.alternate===null&&t(m,N),g=l(D,g,L),w===null?E=D:w.sibling=D,w=D,N=R}if(L===v.length)return n(m,N),se&&an(m,L),E;if(N===null){for(;L<v.length;L++)N=h(m,v[L],C),N!==null&&(g=l(N,g,L),w===null?E=N:w.sibling=N,w=N);return se&&an(m,L),E}for(N=r(m,N);L<v.length;L++)R=p(N,m,L,v[L],C),R!==null&&(e&&R.alternate!==null&&N.delete(R.key===null?L:R.key),g=l(R,g,L),w===null?E=R:w.sibling=R,w=R);return e&&N.forEach(function(O){return t(m,O)}),se&&an(m,L),E}function S(m,g,v,C){var E=ur(v);if(typeof E!="function")throw Error(z(150));if(v=E.call(v),v==null)throw Error(z(151));for(var w=E=null,N=g,L=g=0,R=null,D=v.next();N!==null&&!D.done;L++,D=v.next()){N.index>L?(R=N,N=null):R=N.sibling;var O=d(m,N,D.value,C);if(O===null){N===null&&(N=R);break}e&&N&&O.alternate===null&&t(m,N),g=l(O,g,L),w===null?E=O:w.sibling=O,w=O,N=R}if(D.done)return n(m,N),se&&an(m,L),E;if(N===null){for(;!D.done;L++,D=v.next())D=h(m,D.value,C),D!==null&&(g=l(D,g,L),w===null?E=D:w.sibling=D,w=D);return se&&an(m,L),E}for(N=r(m,N);!D.done;L++,D=v.next())D=p(N,m,L,D.value,C),D!==null&&(e&&D.alternate!==null&&N.delete(D.key===null?L:D.key),g=l(D,g,L),w===null?E=D:w.sibling=D,w=D);return e&&N.forEach(function(_){return t(m,_)}),se&&an(m,L),E}function b(m,g,v,C){if(typeof v=="object"&&v!==null&&v.type===Pn&&v.key===null&&(v=v.props.children),typeof v=="object"&&v!==null){switch(v.$$typeof){case si:e:{for(var E=v.key,w=g;w!==null;){if(w.key===E){if(E=v.type,E===Pn){if(w.tag===7){n(m,w.sibling),g=i(w,v.props.children),g.return=m,m=g;break e}}else if(w.elementType===E||typeof E=="object"&&E!==null&&E.$$typeof===Ft&&Su(E)===w.type){n(m,w.sibling),g=i(w,v.props),g.ref=hr(m,w,v),g.return=m,m=g;break e}n(m,w);break}else t(m,w);w=w.sibling}v.type===Pn?(g=mn(v.props.children,m.mode,C,v.key),g.return=m,m=g):(C=Bi(v.type,v.key,v.props,null,m.mode,C),C.ref=hr(m,g,v),C.return=m,m=C)}return o(m);case zn:e:{for(w=v.key;g!==null;){if(g.key===w)if(g.tag===4&&g.stateNode.containerInfo===v.containerInfo&&g.stateNode.implementation===v.implementation){n(m,g.sibling),g=i(g,v.children||[]),g.return=m,m=g;break e}else{n(m,g);break}else t(m,g);g=g.sibling}g=ao(v,m.mode,C),g.return=m,m=g}return o(m);case Ft:return w=v._init,b(m,g,w(v._payload),C)}if(kr(v))return k(m,g,v,C);if(ur(v))return S(m,g,v,C);xi(m,v)}return typeof v=="string"&&v!==""||typeof v=="number"?(v=""+v,g!==null&&g.tag===6?(n(m,g.sibling),g=i(g,v),g.return=m,m=g):(n(m,g),g=oo(v,m.mode,C),g.return=m,m=g),o(m)):n(m,g)}return b}var Jn=Gd(!0),Jd=Gd(!1),rl=nn(null),il=null,Rn=null,qa=null;function Ya(){qa=Rn=il=null}function Xa(e){var t=rl.current;ae(rl),e._currentValue=t}function Xo(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Wn(e,t){il=e,qa=Rn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(Ae=!0),e.firstContext=null)}function nt(e){var t=e._currentValue;if(qa!==e)if(e={context:e,memoizedValue:t,next:null},Rn===null){if(il===null)throw Error(z(308));Rn=e,il.dependencies={lanes:0,firstContext:e}}else Rn=Rn.next=e;return t}var fn=null;function Ga(e){fn===null?fn=[e]:fn.push(e)}function Zd(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,Ga(t)):(n.next=i.next,i.next=n),t.interleaved=n,It(e,r)}function It(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Bt=!1;function Ja(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function ef(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Pt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function Yt(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,Y&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,It(e,n)}return i=r.interleaved,i===null?(t.next=t,Ga(r)):(t.next=i.next,i.next=t),r.interleaved=t,It(e,n)}function Mi(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Oa(e,n)}}function Cu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function ll(e,t,n,r){var i=e.updateQueue;Bt=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,u=s.next;s.next=null,o===null?l=u:o.next=u,o=s;var f=e.alternate;f!==null&&(f=f.updateQueue,a=f.lastBaseUpdate,a!==o&&(a===null?f.firstBaseUpdate=u:a.next=u,f.lastBaseUpdate=s))}if(l!==null){var h=i.baseState;o=0,f=u=s=null,a=l;do{var d=a.lane,p=a.eventTime;if((r&d)===d){f!==null&&(f=f.next={eventTime:p,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var k=e,S=a;switch(d=t,p=n,S.tag){case 1:if(k=S.payload,typeof k=="function"){h=k.call(p,h,d);break e}h=k;break e;case 3:k.flags=k.flags&-65537|128;case 0:if(k=S.payload,d=typeof k=="function"?k.call(p,h,d):k,d==null)break e;h=de({},h,d);break e;case 2:Bt=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,d=i.effects,d===null?i.effects=[a]:d.push(a))}else p={eventTime:p,lane:d,tag:a.tag,payload:a.payload,callback:a.callback,next:null},f===null?(u=f=p,s=h):f=f.next=p,o|=d;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;d=a,a=d.next,d.next=null,i.lastBaseUpdate=d,i.shared.pending=null}}while(!0);if(f===null&&(s=h),i.baseState=s,i.firstBaseUpdate=u,i.lastBaseUpdate=f,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);xn|=o,e.lanes=o,e.memoizedState=h}}function Eu(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(z(191,i));i.call(r)}}}var ri={},kt=nn(ri),Qr=nn(ri),Kr=nn(ri);function pn(e){if(e===ri)throw Error(z(174));return e}function Za(e,t){switch(re(Kr,t),re(Qr,e),re(kt,ri),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Po(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Po(t,e)}ae(kt),re(kt,t)}function Zn(){ae(kt),ae(Qr),ae(Kr)}function tf(e){pn(Kr.current);var t=pn(kt.current),n=Po(t,e.type);t!==n&&(re(Qr,e),re(kt,n))}function es(e){Qr.current===e&&(ae(kt),ae(Qr))}var ue=nn(0);function ol(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var eo=[];function ts(){for(var e=0;e<eo.length;e++)eo[e]._workInProgressVersionPrimary=null;eo.length=0}var Ai=At.ReactCurrentDispatcher,to=At.ReactCurrentBatchConfig,vn=0,ce=null,ye=null,xe=null,al=!1,_r=!1,qr=0,Am=0;function Ee(){throw Error(z(321))}function ns(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!ft(e[n],t[n]))return!1;return!0}function rs(e,t,n,r,i,l){if(vn=l,ce=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Ai.current=e===null||e.memoizedState===null?Fm:Bm,e=n(r,i),_r){l=0;do{if(_r=!1,qr=0,25<=l)throw Error(z(301));l+=1,xe=ye=null,t.updateQueue=null,Ai.current=Um,e=n(r,i)}while(_r)}if(Ai.current=sl,t=ye!==null&&ye.next!==null,vn=0,xe=ye=ce=null,al=!1,t)throw Error(z(300));return e}function is(){var e=qr!==0;return qr=0,e}function mt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return xe===null?ce.memoizedState=xe=e:xe=xe.next=e,xe}function rt(){if(ye===null){var e=ce.alternate;e=e!==null?e.memoizedState:null}else e=ye.next;var t=xe===null?ce.memoizedState:xe.next;if(t!==null)xe=t,ye=e;else{if(e===null)throw Error(z(310));ye=e,e={memoizedState:ye.memoizedState,baseState:ye.baseState,baseQueue:ye.baseQueue,queue:ye.queue,next:null},xe===null?ce.memoizedState=xe=e:xe=xe.next=e}return xe}function Yr(e,t){return typeof t=="function"?t(e):t}function no(e){var t=rt(),n=t.queue;if(n===null)throw Error(z(311));n.lastRenderedReducer=e;var r=ye,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,u=l;do{var f=u.lane;if((vn&f)===f)s!==null&&(s=s.next={lane:0,action:u.action,hasEagerState:u.hasEagerState,eagerState:u.eagerState,next:null}),r=u.hasEagerState?u.eagerState:e(r,u.action);else{var h={lane:f,action:u.action,hasEagerState:u.hasEagerState,eagerState:u.eagerState,next:null};s===null?(a=s=h,o=r):s=s.next=h,ce.lanes|=f,xn|=f}u=u.next}while(u!==null&&u!==l);s===null?o=r:s.next=a,ft(r,t.memoizedState)||(Ae=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,ce.lanes|=l,xn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function ro(e){var t=rt(),n=t.queue;if(n===null)throw Error(z(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);ft(l,t.memoizedState)||(Ae=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function nf(){}function rf(e,t){var n=ce,r=rt(),i=t(),l=!ft(r.memoizedState,i);if(l&&(r.memoizedState=i,Ae=!0),r=r.queue,ls(af.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||xe!==null&&xe.memoizedState.tag&1){if(n.flags|=2048,Xr(9,of.bind(null,n,r,i,t),void 0,null),ke===null)throw Error(z(349));vn&30||lf(n,t,i)}return i}function lf(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=ce.updateQueue,t===null?(t={lastEffect:null,stores:null},ce.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function of(e,t,n,r){t.value=n,t.getSnapshot=r,sf(t)&&uf(e)}function af(e,t,n){return n(function(){sf(t)&&uf(e)})}function sf(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!ft(e,n)}catch{return!0}}function uf(e){var t=It(e,1);t!==null&&dt(t,e,1,-1)}function bu(e){var t=mt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:Yr,lastRenderedState:e},t.queue=e,e=e.dispatch=Rm.bind(null,ce,e),[t.memoizedState,e]}function Xr(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=ce.updateQueue,t===null?(t={lastEffect:null,stores:null},ce.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function cf(){return rt().memoizedState}function Di(e,t,n,r){var i=mt();ce.flags|=e,i.memoizedState=Xr(1|t,n,void 0,r===void 0?null:r)}function Cl(e,t,n,r){var i=rt();r=r===void 0?null:r;var l=void 0;if(ye!==null){var o=ye.memoizedState;if(l=o.destroy,r!==null&&ns(r,o.deps)){i.memoizedState=Xr(t,n,l,r);return}}ce.flags|=e,i.memoizedState=Xr(1|t,n,l,r)}function ju(e,t){return Di(8390656,8,e,t)}function ls(e,t){return Cl(2048,8,e,t)}function df(e,t){return Cl(4,2,e,t)}function ff(e,t){return Cl(4,4,e,t)}function pf(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function hf(e,t,n){return n=n!=null?n.concat([e]):null,Cl(4,4,pf.bind(null,t,e),n)}function os(){}function mf(e,t){var n=rt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ns(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function gf(e,t){var n=rt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ns(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function yf(e,t,n){return vn&21?(ft(n,t)||(n=wd(),ce.lanes|=n,xn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,Ae=!0),e.memoizedState=n)}function Dm(e,t){var n=J;J=n!==0&&4>n?n:4,e(!0);var r=to.transition;to.transition={};try{e(!1),t()}finally{J=n,to.transition=r}}function vf(){return rt().memoizedState}function Om(e,t,n){var r=Gt(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},xf(e))kf(t,n);else if(n=Zd(e,t,n,r),n!==null){var i=Pe();dt(n,e,r,i),wf(n,t,r)}}function Rm(e,t,n){var r=Gt(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(xf(e))kf(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,ft(a,o)){var s=t.interleaved;s===null?(i.next=i,Ga(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=Zd(e,t,i,r),n!==null&&(i=Pe(),dt(n,e,r,i),wf(n,t,r))}}function xf(e){var t=e.alternate;return e===ce||t!==null&&t===ce}function kf(e,t){_r=al=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function wf(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Oa(e,n)}}var sl={readContext:nt,useCallback:Ee,useContext:Ee,useEffect:Ee,useImperativeHandle:Ee,useInsertionEffect:Ee,useLayoutEffect:Ee,useMemo:Ee,useReducer:Ee,useRef:Ee,useState:Ee,useDebugValue:Ee,useDeferredValue:Ee,useTransition:Ee,useMutableSource:Ee,useSyncExternalStore:Ee,useId:Ee,unstable_isNewReconciler:!1},Fm={readContext:nt,useCallback:function(e,t){return mt().memoizedState=[e,t===void 0?null:t],e},useContext:nt,useEffect:ju,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Di(4194308,4,pf.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Di(4194308,4,e,t)},useInsertionEffect:function(e,t){return Di(4,2,e,t)},useMemo:function(e,t){var n=mt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=mt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Om.bind(null,ce,e),[r.memoizedState,e]},useRef:function(e){var t=mt();return e={current:e},t.memoizedState=e},useState:bu,useDebugValue:os,useDeferredValue:function(e){return mt().memoizedState=e},useTransition:function(){var e=bu(!1),t=e[0];return e=Dm.bind(null,e[1]),mt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=ce,i=mt();if(se){if(n===void 0)throw Error(z(407));n=n()}else{if(n=t(),ke===null)throw Error(z(349));vn&30||lf(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,ju(af.bind(null,r,l,e),[e]),r.flags|=2048,Xr(9,of.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=mt(),t=ke.identifierPrefix;if(se){var n=zt,r=_t;n=(r&~(1<<32-ct(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=qr++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Am++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},Bm={readContext:nt,useCallback:mf,useContext:nt,useEffect:ls,useImperativeHandle:hf,useInsertionEffect:df,useLayoutEffect:ff,useMemo:gf,useReducer:no,useRef:cf,useState:function(){return no(Yr)},useDebugValue:os,useDeferredValue:function(e){var t=rt();return yf(t,ye.memoizedState,e)},useTransition:function(){var e=no(Yr)[0],t=rt().memoizedState;return[e,t]},useMutableSource:nf,useSyncExternalStore:rf,useId:vf,unstable_isNewReconciler:!1},Um={readContext:nt,useCallback:mf,useContext:nt,useEffect:ls,useImperativeHandle:hf,useInsertionEffect:df,useLayoutEffect:ff,useMemo:gf,useReducer:ro,useRef:cf,useState:function(){return ro(Yr)},useDebugValue:os,useDeferredValue:function(e){var t=rt();return ye===null?t.memoizedState=e:yf(t,ye.memoizedState,e)},useTransition:function(){var e=ro(Yr)[0],t=rt().memoizedState;return[e,t]},useMutableSource:nf,useSyncExternalStore:rf,useId:vf,unstable_isNewReconciler:!1};function at(e,t){if(e&&e.defaultProps){t=de({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function Go(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:de({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var El={isMounted:function(e){return(e=e._reactInternals)?Sn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Pe(),i=Gt(e),l=Pt(r,i);l.payload=t,n!=null&&(l.callback=n),t=Yt(e,l,i),t!==null&&(dt(t,e,i,r),Mi(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Pe(),i=Gt(e),l=Pt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=Yt(e,l,i),t!==null&&(dt(t,e,i,r),Mi(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Pe(),r=Gt(e),i=Pt(n,r);i.tag=2,t!=null&&(i.callback=t),t=Yt(e,i,r),t!==null&&(dt(t,e,r,n),Mi(t,e,r))}};function Nu(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Hr(n,r)||!Hr(i,l):!0}function Sf(e,t,n){var r=!1,i=en,l=t.contextType;return typeof l=="object"&&l!==null?l=nt(l):(i=Oe(t)?gn:Ne.current,r=t.contextTypes,l=(r=r!=null)?Xn(e,i):en),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=El,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function _u(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&El.enqueueReplaceState(t,t.state,null)}function Jo(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},Ja(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=nt(l):(l=Oe(t)?gn:Ne.current,i.context=Xn(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(Go(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&El.enqueueReplaceState(i,i.state,null),ll(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function er(e,t){try{var n="",r=t;do n+=mh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function io(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function Zo(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var Hm=typeof WeakMap=="function"?WeakMap:Map;function Cf(e,t,n){n=Pt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){cl||(cl=!0,ua=r),Zo(e,t)},n}function Ef(e,t,n){n=Pt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){Zo(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){Zo(e,t),typeof r!="function"&&(Xt===null?Xt=new Set([this]):Xt.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function zu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new Hm;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=ng.bind(null,e,t,n),t.then(e,e))}function Pu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Lu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Pt(-1,1),t.tag=2,Yt(n,t,1))),n.lanes|=1),e)}var Vm=At.ReactCurrentOwner,Ae=!1;function ze(e,t,n,r){t.child=e===null?Jd(t,null,n,r):Jn(t,e.child,n,r)}function Tu(e,t,n,r,i){n=n.render;var l=t.ref;return Wn(t,i),r=rs(e,t,n,r,l,i),n=is(),e!==null&&!Ae?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Mt(e,t,i)):(se&&n&&Wa(t),t.flags|=1,ze(e,t,r,i),t.child)}function Iu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!hs(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,bf(e,t,l,r,i)):(e=Bi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Hr,n(o,r)&&e.ref===t.ref)return Mt(e,t,i)}return t.flags|=1,e=Jt(l,r),e.ref=t.ref,e.return=t,t.child=e}function bf(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Hr(l,r)&&e.ref===t.ref)if(Ae=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(Ae=!0);else return t.lanes=e.lanes,Mt(e,t,i)}return ea(e,t,n,r,i)}function jf(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},re(Bn,Ve),Ve|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,re(Bn,Ve),Ve|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,re(Bn,Ve),Ve|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,re(Bn,Ve),Ve|=r;return ze(e,t,i,n),t.child}function Nf(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ea(e,t,n,r,i){var l=Oe(n)?gn:Ne.current;return l=Xn(t,l),Wn(t,i),n=rs(e,t,n,r,l,i),r=is(),e!==null&&!Ae?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Mt(e,t,i)):(se&&r&&Wa(t),t.flags|=1,ze(e,t,n,i),t.child)}function Mu(e,t,n,r,i){if(Oe(n)){var l=!0;el(t)}else l=!1;if(Wn(t,i),t.stateNode===null)Oi(e,t),Sf(t,n,r),Jo(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,u=n.contextType;typeof u=="object"&&u!==null?u=nt(u):(u=Oe(n)?gn:Ne.current,u=Xn(t,u));var f=n.getDerivedStateFromProps,h=typeof f=="function"||typeof o.getSnapshotBeforeUpdate=="function";h||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==u)&&_u(t,o,r,u),Bt=!1;var d=t.memoizedState;o.state=d,ll(t,r,o,i),s=t.memoizedState,a!==r||d!==s||De.current||Bt?(typeof f=="function"&&(Go(t,n,f,r),s=t.memoizedState),(a=Bt||Nu(t,n,a,r,d,s,u))?(h||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=u,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,ef(e,t),a=t.memoizedProps,u=t.type===t.elementType?a:at(t.type,a),o.props=u,h=t.pendingProps,d=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=nt(s):(s=Oe(n)?gn:Ne.current,s=Xn(t,s));var p=n.getDerivedStateFromProps;(f=typeof p=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==h||d!==s)&&_u(t,o,r,s),Bt=!1,d=t.memoizedState,o.state=d,ll(t,r,o,i);var k=t.memoizedState;a!==h||d!==k||De.current||Bt?(typeof p=="function"&&(Go(t,n,p,r),k=t.memoizedState),(u=Bt||Nu(t,n,u,r,d,k,s)||!1)?(f||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,k,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,k,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&d===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&d===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=k),o.props=r,o.state=k,o.context=s,r=u):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&d===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&d===e.memoizedState||(t.flags|=1024),r=!1)}return ta(e,t,n,r,l,i)}function ta(e,t,n,r,i,l){Nf(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&xu(t,n,!1),Mt(e,t,l);r=t.stateNode,Vm.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=Jn(t,e.child,null,l),t.child=Jn(t,null,a,l)):ze(e,t,a,l),t.memoizedState=r.state,i&&xu(t,n,!0),t.child}function _f(e){var t=e.stateNode;t.pendingContext?vu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&vu(e,t.context,!1),Za(e,t.containerInfo)}function Au(e,t,n,r,i){return Gn(),Ka(i),t.flags|=256,ze(e,t,n,r),t.child}var na={dehydrated:null,treeContext:null,retryLane:0};function ra(e){return{baseLanes:e,cachePool:null,transitions:null}}function zf(e,t,n){var r=t.pendingProps,i=ue.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),re(ue,i&1),e===null)return Yo(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Nl(o,r,0,null),e=mn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=ra(n),t.memoizedState=na,e):as(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return $m(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=Jt(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=Jt(a,l):(l=mn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?ra(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=na,r}return l=e.child,e=l.sibling,r=Jt(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function as(e,t){return t=Nl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function ki(e,t,n,r){return r!==null&&Ka(r),Jn(t,e.child,null,n),e=as(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function $m(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=io(Error(z(422))),ki(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Nl({mode:"visible",children:r.children},i,0,null),l=mn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&Jn(t,e.child,null,o),t.child.memoizedState=ra(o),t.memoizedState=na,l);if(!(t.mode&1))return ki(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(z(419)),r=io(l,r,void 0),ki(e,t,o,r)}if(a=(o&e.childLanes)!==0,Ae||a){if(r=ke,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,It(e,i),dt(r,e,i,-1))}return ps(),r=io(Error(z(421))),ki(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=rg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,$e=qt(i.nextSibling),Qe=t,se=!0,ut=null,e!==null&&(Ge[Je++]=_t,Ge[Je++]=zt,Ge[Je++]=yn,_t=e.id,zt=e.overflow,yn=t),t=as(t,r.children),t.flags|=4096,t)}function Du(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),Xo(e.return,t,n)}function lo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Pf(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(ze(e,t,r.children,n),r=ue.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Du(e,n,t);else if(e.tag===19)Du(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(re(ue,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&ol(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),lo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&ol(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}lo(t,!0,n,null,l);break;case"together":lo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Oi(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Mt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),xn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(z(153));if(t.child!==null){for(e=t.child,n=Jt(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=Jt(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function Wm(e,t,n){switch(t.tag){case 3:_f(t),Gn();break;case 5:tf(t);break;case 1:Oe(t.type)&&el(t);break;case 4:Za(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;re(rl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(re(ue,ue.current&1),t.flags|=128,null):n&t.child.childLanes?zf(e,t,n):(re(ue,ue.current&1),e=Mt(e,t,n),e!==null?e.sibling:null);re(ue,ue.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Pf(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),re(ue,ue.current),r)break;return null;case 22:case 23:return t.lanes=0,jf(e,t,n)}return Mt(e,t,n)}var Lf,ia,Tf,If;Lf=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};ia=function(){};Tf=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,pn(kt.current);var l=null;switch(n){case"input":i=jo(e,i),r=jo(e,r),l=[];break;case"select":i=de({},i,{value:void 0}),r=de({},r,{value:void 0}),l=[];break;case"textarea":i=zo(e,i),r=zo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=Ji)}Lo(n,r);var o;n=null;for(u in i)if(!r.hasOwnProperty(u)&&i.hasOwnProperty(u)&&i[u]!=null)if(u==="style"){var a=i[u];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else u!=="dangerouslySetInnerHTML"&&u!=="children"&&u!=="suppressContentEditableWarning"&&u!=="suppressHydrationWarning"&&u!=="autoFocus"&&(Ar.hasOwnProperty(u)?l||(l=[]):(l=l||[]).push(u,null));for(u in r){var s=r[u];if(a=i!=null?i[u]:void 0,r.hasOwnProperty(u)&&s!==a&&(s!=null||a!=null))if(u==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(u,n)),n=s;else u==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(u,s)):u==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(u,""+s):u!=="suppressContentEditableWarning"&&u!=="suppressHydrationWarning"&&(Ar.hasOwnProperty(u)?(s!=null&&u==="onScroll"&&oe("scroll",e),l||a===s||(l=[])):(l=l||[]).push(u,s))}n&&(l=l||[]).push("style",n);var u=l;(t.updateQueue=u)&&(t.flags|=4)}};If=function(e,t,n,r){n!==r&&(t.flags|=4)};function mr(e,t){if(!se)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function be(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function Qm(e,t,n){var r=t.pendingProps;switch(Qa(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return be(t),null;case 1:return Oe(t.type)&&Zi(),be(t),null;case 3:return r=t.stateNode,Zn(),ae(De),ae(Ne),ts(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(vi(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,ut!==null&&(fa(ut),ut=null))),ia(e,t),be(t),null;case 5:es(t);var i=pn(Kr.current);if(n=t.type,e!==null&&t.stateNode!=null)Tf(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(z(166));return be(t),null}if(e=pn(kt.current),vi(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[yt]=t,r[Wr]=l,e=(t.mode&1)!==0,n){case"dialog":oe("cancel",r),oe("close",r);break;case"iframe":case"object":case"embed":oe("load",r);break;case"video":case"audio":for(i=0;i<Sr.length;i++)oe(Sr[i],r);break;case"source":oe("error",r);break;case"img":case"image":case"link":oe("error",r),oe("load",r);break;case"details":oe("toggle",r);break;case"input":Ws(r,l),oe("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},oe("invalid",r);break;case"textarea":Ks(r,l),oe("invalid",r)}Lo(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&yi(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&yi(r.textContent,a,e),i=["children",""+a]):Ar.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&oe("scroll",r)}switch(n){case"input":ui(r),Qs(r,l,!0);break;case"textarea":ui(r),qs(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=Ji)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=od(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[yt]=t,e[Wr]=r,Lf(e,t,!1,!1),t.stateNode=e;e:{switch(o=To(n,r),n){case"dialog":oe("cancel",e),oe("close",e),i=r;break;case"iframe":case"object":case"embed":oe("load",e),i=r;break;case"video":case"audio":for(i=0;i<Sr.length;i++)oe(Sr[i],e);i=r;break;case"source":oe("error",e),i=r;break;case"img":case"image":case"link":oe("error",e),oe("load",e),i=r;break;case"details":oe("toggle",e),i=r;break;case"input":Ws(e,r),i=jo(e,r),oe("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=de({},r,{value:void 0}),oe("invalid",e);break;case"textarea":Ks(e,r),i=zo(e,r),oe("invalid",e);break;default:i=r}Lo(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?ud(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&ad(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Dr(e,s):typeof s=="number"&&Dr(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Ar.hasOwnProperty(l)?s!=null&&l==="onScroll"&&oe("scroll",e):s!=null&&La(e,l,s,o))}switch(n){case"input":ui(e),Qs(e,r,!1);break;case"textarea":ui(e),qs(e);break;case"option":r.value!=null&&e.setAttribute("value",""+Zt(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Un(e,!!r.multiple,l,!1):r.defaultValue!=null&&Un(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=Ji)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return be(t),null;case 6:if(e&&t.stateNode!=null)If(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(z(166));if(n=pn(Kr.current),pn(kt.current),vi(t)){if(r=t.stateNode,n=t.memoizedProps,r[yt]=t,(l=r.nodeValue!==n)&&(e=Qe,e!==null))switch(e.tag){case 3:yi(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&yi(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[yt]=t,t.stateNode=r}return be(t),null;case 13:if(ae(ue),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(se&&$e!==null&&t.mode&1&&!(t.flags&128))Xd(),Gn(),t.flags|=98560,l=!1;else if(l=vi(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(z(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(z(317));l[yt]=t}else Gn(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;be(t),l=!1}else ut!==null&&(fa(ut),ut=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||ue.current&1?ve===0&&(ve=3):ps())),t.updateQueue!==null&&(t.flags|=4),be(t),null);case 4:return Zn(),ia(e,t),e===null&&Vr(t.stateNode.containerInfo),be(t),null;case 10:return Xa(t.type._context),be(t),null;case 17:return Oe(t.type)&&Zi(),be(t),null;case 19:if(ae(ue),l=t.memoizedState,l===null)return be(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)mr(l,!1);else{if(ve!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=ol(e),o!==null){for(t.flags|=128,mr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return re(ue,ue.current&1|2),t.child}e=e.sibling}l.tail!==null&&pe()>tr&&(t.flags|=128,r=!0,mr(l,!1),t.lanes=4194304)}else{if(!r)if(e=ol(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),mr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!se)return be(t),null}else 2*pe()-l.renderingStartTime>tr&&n!==1073741824&&(t.flags|=128,r=!0,mr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=pe(),t.sibling=null,n=ue.current,re(ue,r?n&1|2:n&1),t):(be(t),null);case 22:case 23:return fs(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?Ve&1073741824&&(be(t),t.subtreeFlags&6&&(t.flags|=8192)):be(t),null;case 24:return null;case 25:return null}throw Error(z(156,t.tag))}function Km(e,t){switch(Qa(t),t.tag){case 1:return Oe(t.type)&&Zi(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return Zn(),ae(De),ae(Ne),ts(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return es(t),null;case 13:if(ae(ue),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(z(340));Gn()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return ae(ue),null;case 4:return Zn(),null;case 10:return Xa(t.type._context),null;case 22:case 23:return fs(),null;case 24:return null;default:return null}}var wi=!1,je=!1,qm=typeof WeakSet=="function"?WeakSet:Set,A=null;function Fn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){fe(e,t,r)}else n.current=null}function la(e,t,n){try{n()}catch(r){fe(e,t,r)}}var Ou=!1;function Ym(e,t){if(Ho=Yi,e=Od(),$a(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,u=0,f=0,h=e,d=null;t:for(;;){for(var p;h!==n||i!==0&&h.nodeType!==3||(a=o+i),h!==l||r!==0&&h.nodeType!==3||(s=o+r),h.nodeType===3&&(o+=h.nodeValue.length),(p=h.firstChild)!==null;)d=h,h=p;for(;;){if(h===e)break t;if(d===n&&++u===i&&(a=o),d===l&&++f===r&&(s=o),(p=h.nextSibling)!==null)break;h=d,d=h.parentNode}h=p}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Vo={focusedElem:e,selectionRange:n},Yi=!1,A=t;A!==null;)if(t=A,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,A=e;else for(;A!==null;){t=A;try{var k=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(k!==null){var S=k.memoizedProps,b=k.memoizedState,m=t.stateNode,g=m.getSnapshotBeforeUpdate(t.elementType===t.type?S:at(t.type,S),b);m.__reactInternalSnapshotBeforeUpdate=g}break;case 3:var v=t.stateNode.containerInfo;v.nodeType===1?v.textContent="":v.nodeType===9&&v.documentElement&&v.removeChild(v.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(z(163))}}catch(C){fe(t,t.return,C)}if(e=t.sibling,e!==null){e.return=t.return,A=e;break}A=t.return}return k=Ou,Ou=!1,k}function zr(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&la(t,n,l)}i=i.next}while(i!==r)}}function bl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function oa(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Mf(e){var t=e.alternate;t!==null&&(e.alternate=null,Mf(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[yt],delete t[Wr],delete t[Qo],delete t[Lm],delete t[Tm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Af(e){return e.tag===5||e.tag===3||e.tag===4}function Ru(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Af(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function aa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=Ji));else if(r!==4&&(e=e.child,e!==null))for(aa(e,t,n),e=e.sibling;e!==null;)aa(e,t,n),e=e.sibling}function sa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(sa(e,t,n),e=e.sibling;e!==null;)sa(e,t,n),e=e.sibling}var we=null,st=!1;function Ot(e,t,n){for(n=n.child;n!==null;)Df(e,t,n),n=n.sibling}function Df(e,t,n){if(xt&&typeof xt.onCommitFiberUnmount=="function")try{xt.onCommitFiberUnmount(yl,n)}catch{}switch(n.tag){case 5:je||Fn(n,t);case 6:var r=we,i=st;we=null,Ot(e,t,n),we=r,st=i,we!==null&&(st?(e=we,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):we.removeChild(n.stateNode));break;case 18:we!==null&&(st?(e=we,n=n.stateNode,e.nodeType===8?Jl(e.parentNode,n):e.nodeType===1&&Jl(e,n),Br(e)):Jl(we,n.stateNode));break;case 4:r=we,i=st,we=n.stateNode.containerInfo,st=!0,Ot(e,t,n),we=r,st=i;break;case 0:case 11:case 14:case 15:if(!je&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&la(n,t,o),i=i.next}while(i!==r)}Ot(e,t,n);break;case 1:if(!je&&(Fn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){fe(n,t,a)}Ot(e,t,n);break;case 21:Ot(e,t,n);break;case 22:n.mode&1?(je=(r=je)||n.memoizedState!==null,Ot(e,t,n),je=r):Ot(e,t,n);break;default:Ot(e,t,n)}}function Fu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new qm),t.forEach(function(r){var i=ig.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function ot(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:we=a.stateNode,st=!1;break e;case 3:we=a.stateNode.containerInfo,st=!0;break e;case 4:we=a.stateNode.containerInfo,st=!0;break e}a=a.return}if(we===null)throw Error(z(160));Df(l,o,i),we=null,st=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(u){fe(i,t,u)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Of(t,e),t=t.sibling}function Of(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(ot(t,e),ht(e),r&4){try{zr(3,e,e.return),bl(3,e)}catch(S){fe(e,e.return,S)}try{zr(5,e,e.return)}catch(S){fe(e,e.return,S)}}break;case 1:ot(t,e),ht(e),r&512&&n!==null&&Fn(n,n.return);break;case 5:if(ot(t,e),ht(e),r&512&&n!==null&&Fn(n,n.return),e.flags&32){var i=e.stateNode;try{Dr(i,"")}catch(S){fe(e,e.return,S)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&id(i,l),To(a,o);var u=To(a,l);for(o=0;o<s.length;o+=2){var f=s[o],h=s[o+1];f==="style"?ud(i,h):f==="dangerouslySetInnerHTML"?ad(i,h):f==="children"?Dr(i,h):La(i,f,h,u)}switch(a){case"input":No(i,l);break;case"textarea":ld(i,l);break;case"select":var d=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var p=l.value;p!=null?Un(i,!!l.multiple,p,!1):d!==!!l.multiple&&(l.defaultValue!=null?Un(i,!!l.multiple,l.defaultValue,!0):Un(i,!!l.multiple,l.multiple?[]:"",!1))}i[Wr]=l}catch(S){fe(e,e.return,S)}}break;case 6:if(ot(t,e),ht(e),r&4){if(e.stateNode===null)throw Error(z(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(S){fe(e,e.return,S)}}break;case 3:if(ot(t,e),ht(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Br(t.containerInfo)}catch(S){fe(e,e.return,S)}break;case 4:ot(t,e),ht(e);break;case 13:ot(t,e),ht(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(cs=pe())),r&4&&Fu(e);break;case 22:if(f=n!==null&&n.memoizedState!==null,e.mode&1?(je=(u=je)||f,ot(t,e),je=u):ot(t,e),ht(e),r&8192){if(u=e.memoizedState!==null,(e.stateNode.isHidden=u)&&!f&&e.mode&1)for(A=e,f=e.child;f!==null;){for(h=A=f;A!==null;){switch(d=A,p=d.child,d.tag){case 0:case 11:case 14:case 15:zr(4,d,d.return);break;case 1:Fn(d,d.return);var k=d.stateNode;if(typeof k.componentWillUnmount=="function"){r=d,n=d.return;try{t=r,k.props=t.memoizedProps,k.state=t.memoizedState,k.componentWillUnmount()}catch(S){fe(r,n,S)}}break;case 5:Fn(d,d.return);break;case 22:if(d.memoizedState!==null){Uu(h);continue}}p!==null?(p.return=d,A=p):Uu(h)}f=f.sibling}e:for(f=null,h=e;;){if(h.tag===5){if(f===null){f=h;try{i=h.stateNode,u?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=h.stateNode,s=h.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=sd("display",o))}catch(S){fe(e,e.return,S)}}}else if(h.tag===6){if(f===null)try{h.stateNode.nodeValue=u?"":h.memoizedProps}catch(S){fe(e,e.return,S)}}else if((h.tag!==22&&h.tag!==23||h.memoizedState===null||h===e)&&h.child!==null){h.child.return=h,h=h.child;continue}if(h===e)break e;for(;h.sibling===null;){if(h.return===null||h.return===e)break e;f===h&&(f=null),h=h.return}f===h&&(f=null),h.sibling.return=h.return,h=h.sibling}}break;case 19:ot(t,e),ht(e),r&4&&Fu(e);break;case 21:break;default:ot(t,e),ht(e)}}function ht(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Af(n)){var r=n;break e}n=n.return}throw Error(z(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Dr(i,""),r.flags&=-33);var l=Ru(e);sa(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Ru(e);aa(e,a,o);break;default:throw Error(z(161))}}catch(s){fe(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function Xm(e,t,n){A=e,Rf(e)}function Rf(e,t,n){for(var r=(e.mode&1)!==0;A!==null;){var i=A,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||wi;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||je;a=wi;var u=je;if(wi=o,(je=s)&&!u)for(A=i;A!==null;)o=A,s=o.child,o.tag===22&&o.memoizedState!==null?Hu(i):s!==null?(s.return=o,A=s):Hu(i);for(;l!==null;)A=l,Rf(l),l=l.sibling;A=i,wi=a,je=u}Bu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,A=l):Bu(e)}}function Bu(e){for(;A!==null;){var t=A;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:je||bl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!je)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:at(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Eu(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Eu(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var u=t.alternate;if(u!==null){var f=u.memoizedState;if(f!==null){var h=f.dehydrated;h!==null&&Br(h)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(z(163))}je||t.flags&512&&oa(t)}catch(d){fe(t,t.return,d)}}if(t===e){A=null;break}if(n=t.sibling,n!==null){n.return=t.return,A=n;break}A=t.return}}function Uu(e){for(;A!==null;){var t=A;if(t===e){A=null;break}var n=t.sibling;if(n!==null){n.return=t.return,A=n;break}A=t.return}}function Hu(e){for(;A!==null;){var t=A;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{bl(4,t)}catch(s){fe(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){fe(t,i,s)}}var l=t.return;try{oa(t)}catch(s){fe(t,l,s)}break;case 5:var o=t.return;try{oa(t)}catch(s){fe(t,o,s)}}}catch(s){fe(t,t.return,s)}if(t===e){A=null;break}var a=t.sibling;if(a!==null){a.return=t.return,A=a;break}A=t.return}}var Gm=Math.ceil,ul=At.ReactCurrentDispatcher,ss=At.ReactCurrentOwner,tt=At.ReactCurrentBatchConfig,Y=0,ke=null,me=null,Se=0,Ve=0,Bn=nn(0),ve=0,Gr=null,xn=0,jl=0,us=0,Pr=null,Me=null,cs=0,tr=1/0,bt=null,cl=!1,ua=null,Xt=null,Si=!1,$t=null,dl=0,Lr=0,ca=null,Ri=-1,Fi=0;function Pe(){return Y&6?pe():Ri!==-1?Ri:Ri=pe()}function Gt(e){return e.mode&1?Y&2&&Se!==0?Se&-Se:Mm.transition!==null?(Fi===0&&(Fi=wd()),Fi):(e=J,e!==0||(e=window.event,e=e===void 0?16:_d(e.type)),e):1}function dt(e,t,n,r){if(50<Lr)throw Lr=0,ca=null,Error(z(185));ei(e,n,r),(!(Y&2)||e!==ke)&&(e===ke&&(!(Y&2)&&(jl|=n),ve===4&&Ht(e,Se)),Re(e,r),n===1&&Y===0&&!(t.mode&1)&&(tr=pe()+500,Sl&&rn()))}function Re(e,t){var n=e.callbackNode;Mh(e,t);var r=qi(e,e===ke?Se:0);if(r===0)n!==null&&Gs(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&Gs(n),t===1)e.tag===0?Im(Vu.bind(null,e)):Kd(Vu.bind(null,e)),zm(function(){!(Y&6)&&rn()}),n=null;else{switch(Sd(r)){case 1:n=Da;break;case 4:n=xd;break;case 16:n=Ki;break;case 536870912:n=kd;break;default:n=Ki}n=Qf(n,Ff.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function Ff(e,t){if(Ri=-1,Fi=0,Y&6)throw Error(z(327));var n=e.callbackNode;if(Qn()&&e.callbackNode!==n)return null;var r=qi(e,e===ke?Se:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=fl(e,r);else{t=r;var i=Y;Y|=2;var l=Uf();(ke!==e||Se!==t)&&(bt=null,tr=pe()+500,hn(e,t));do try{eg();break}catch(a){Bf(e,a)}while(!0);Ya(),ul.current=l,Y=i,me!==null?t=0:(ke=null,Se=0,t=ve)}if(t!==0){if(t===2&&(i=Oo(e),i!==0&&(r=i,t=da(e,i))),t===1)throw n=Gr,hn(e,0),Ht(e,r),Re(e,pe()),n;if(t===6)Ht(e,r);else{if(i=e.current.alternate,!(r&30)&&!Jm(i)&&(t=fl(e,r),t===2&&(l=Oo(e),l!==0&&(r=l,t=da(e,l))),t===1))throw n=Gr,hn(e,0),Ht(e,r),Re(e,pe()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(z(345));case 2:sn(e,Me,bt);break;case 3:if(Ht(e,r),(r&130023424)===r&&(t=cs+500-pe(),10<t)){if(qi(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Pe(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=Wo(sn.bind(null,e,Me,bt),t);break}sn(e,Me,bt);break;case 4:if(Ht(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-ct(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=pe()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*Gm(r/1960))-r,10<r){e.timeoutHandle=Wo(sn.bind(null,e,Me,bt),r);break}sn(e,Me,bt);break;case 5:sn(e,Me,bt);break;default:throw Error(z(329))}}}return Re(e,pe()),e.callbackNode===n?Ff.bind(null,e):null}function da(e,t){var n=Pr;return e.current.memoizedState.isDehydrated&&(hn(e,t).flags|=256),e=fl(e,t),e!==2&&(t=Me,Me=n,t!==null&&fa(t)),e}function fa(e){Me===null?Me=e:Me.push.apply(Me,e)}function Jm(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!ft(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Ht(e,t){for(t&=~us,t&=~jl,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-ct(t),r=1<<n;e[n]=-1,t&=~r}}function Vu(e){if(Y&6)throw Error(z(327));Qn();var t=qi(e,0);if(!(t&1))return Re(e,pe()),null;var n=fl(e,t);if(e.tag!==0&&n===2){var r=Oo(e);r!==0&&(t=r,n=da(e,r))}if(n===1)throw n=Gr,hn(e,0),Ht(e,t),Re(e,pe()),n;if(n===6)throw Error(z(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,sn(e,Me,bt),Re(e,pe()),null}function ds(e,t){var n=Y;Y|=1;try{return e(t)}finally{Y=n,Y===0&&(tr=pe()+500,Sl&&rn())}}function kn(e){$t!==null&&$t.tag===0&&!(Y&6)&&Qn();var t=Y;Y|=1;var n=tt.transition,r=J;try{if(tt.transition=null,J=1,e)return e()}finally{J=r,tt.transition=n,Y=t,!(Y&6)&&rn()}}function fs(){Ve=Bn.current,ae(Bn)}function hn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,_m(n)),me!==null)for(n=me.return;n!==null;){var r=n;switch(Qa(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&Zi();break;case 3:Zn(),ae(De),ae(Ne),ts();break;case 5:es(r);break;case 4:Zn();break;case 13:ae(ue);break;case 19:ae(ue);break;case 10:Xa(r.type._context);break;case 22:case 23:fs()}n=n.return}if(ke=e,me=e=Jt(e.current,null),Se=Ve=t,ve=0,Gr=null,us=jl=xn=0,Me=Pr=null,fn!==null){for(t=0;t<fn.length;t++)if(n=fn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}fn=null}return e}function Bf(e,t){do{var n=me;try{if(Ya(),Ai.current=sl,al){for(var r=ce.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}al=!1}if(vn=0,xe=ye=ce=null,_r=!1,qr=0,ss.current=null,n===null||n.return===null){ve=1,Gr=t,me=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=Se,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var u=s,f=a,h=f.tag;if(!(f.mode&1)&&(h===0||h===11||h===15)){var d=f.alternate;d?(f.updateQueue=d.updateQueue,f.memoizedState=d.memoizedState,f.lanes=d.lanes):(f.updateQueue=null,f.memoizedState=null)}var p=Pu(o);if(p!==null){p.flags&=-257,Lu(p,o,a,l,t),p.mode&1&&zu(l,u,t),t=p,s=u;var k=t.updateQueue;if(k===null){var S=new Set;S.add(s),t.updateQueue=S}else k.add(s);break e}else{if(!(t&1)){zu(l,u,t),ps();break e}s=Error(z(426))}}else if(se&&a.mode&1){var b=Pu(o);if(b!==null){!(b.flags&65536)&&(b.flags|=256),Lu(b,o,a,l,t),Ka(er(s,a));break e}}l=s=er(s,a),ve!==4&&(ve=2),Pr===null?Pr=[l]:Pr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var m=Cf(l,s,t);Cu(l,m);break e;case 1:a=s;var g=l.type,v=l.stateNode;if(!(l.flags&128)&&(typeof g.getDerivedStateFromError=="function"||v!==null&&typeof v.componentDidCatch=="function"&&(Xt===null||!Xt.has(v)))){l.flags|=65536,t&=-t,l.lanes|=t;var C=Ef(l,a,t);Cu(l,C);break e}}l=l.return}while(l!==null)}Vf(n)}catch(E){t=E,me===n&&n!==null&&(me=n=n.return);continue}break}while(!0)}function Uf(){var e=ul.current;return ul.current=sl,e===null?sl:e}function ps(){(ve===0||ve===3||ve===2)&&(ve=4),ke===null||!(xn&268435455)&&!(jl&268435455)||Ht(ke,Se)}function fl(e,t){var n=Y;Y|=2;var r=Uf();(ke!==e||Se!==t)&&(bt=null,hn(e,t));do try{Zm();break}catch(i){Bf(e,i)}while(!0);if(Ya(),Y=n,ul.current=r,me!==null)throw Error(z(261));return ke=null,Se=0,ve}function Zm(){for(;me!==null;)Hf(me)}function eg(){for(;me!==null&&!bh();)Hf(me)}function Hf(e){var t=Wf(e.alternate,e,Ve);e.memoizedProps=e.pendingProps,t===null?Vf(e):me=t,ss.current=null}function Vf(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=Km(n,t),n!==null){n.flags&=32767,me=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{ve=6,me=null;return}}else if(n=Qm(n,t,Ve),n!==null){me=n;return}if(t=t.sibling,t!==null){me=t;return}me=t=e}while(t!==null);ve===0&&(ve=5)}function sn(e,t,n){var r=J,i=tt.transition;try{tt.transition=null,J=1,tg(e,t,n,r)}finally{tt.transition=i,J=r}return null}function tg(e,t,n,r){do Qn();while($t!==null);if(Y&6)throw Error(z(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(z(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Ah(e,l),e===ke&&(me=ke=null,Se=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||Si||(Si=!0,Qf(Ki,function(){return Qn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=tt.transition,tt.transition=null;var o=J;J=1;var a=Y;Y|=4,ss.current=null,Ym(e,n),Of(n,e),wm(Vo),Yi=!!Ho,Vo=Ho=null,e.current=n,Xm(n),jh(),Y=a,J=o,tt.transition=l}else e.current=n;if(Si&&(Si=!1,$t=e,dl=i),l=e.pendingLanes,l===0&&(Xt=null),zh(n.stateNode),Re(e,pe()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(cl)throw cl=!1,e=ua,ua=null,e;return dl&1&&e.tag!==0&&Qn(),l=e.pendingLanes,l&1?e===ca?Lr++:(Lr=0,ca=e):Lr=0,rn(),null}function Qn(){if($t!==null){var e=Sd(dl),t=tt.transition,n=J;try{if(tt.transition=null,J=16>e?16:e,$t===null)var r=!1;else{if(e=$t,$t=null,dl=0,Y&6)throw Error(z(331));var i=Y;for(Y|=4,A=e.current;A!==null;){var l=A,o=l.child;if(A.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var u=a[s];for(A=u;A!==null;){var f=A;switch(f.tag){case 0:case 11:case 15:zr(8,f,l)}var h=f.child;if(h!==null)h.return=f,A=h;else for(;A!==null;){f=A;var d=f.sibling,p=f.return;if(Mf(f),f===u){A=null;break}if(d!==null){d.return=p,A=d;break}A=p}}}var k=l.alternate;if(k!==null){var S=k.child;if(S!==null){k.child=null;do{var b=S.sibling;S.sibling=null,S=b}while(S!==null)}}A=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,A=o;else e:for(;A!==null;){if(l=A,l.flags&2048)switch(l.tag){case 0:case 11:case 15:zr(9,l,l.return)}var m=l.sibling;if(m!==null){m.return=l.return,A=m;break e}A=l.return}}var g=e.current;for(A=g;A!==null;){o=A;var v=o.child;if(o.subtreeFlags&2064&&v!==null)v.return=o,A=v;else e:for(o=g;A!==null;){if(a=A,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:bl(9,a)}}catch(E){fe(a,a.return,E)}if(a===o){A=null;break e}var C=a.sibling;if(C!==null){C.return=a.return,A=C;break e}A=a.return}}if(Y=i,rn(),xt&&typeof xt.onPostCommitFiberRoot=="function")try{xt.onPostCommitFiberRoot(yl,e)}catch{}r=!0}return r}finally{J=n,tt.transition=t}}return!1}function $u(e,t,n){t=er(n,t),t=Cf(e,t,1),e=Yt(e,t,1),t=Pe(),e!==null&&(ei(e,1,t),Re(e,t))}function fe(e,t,n){if(e.tag===3)$u(e,e,n);else for(;t!==null;){if(t.tag===3){$u(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(Xt===null||!Xt.has(r))){e=er(n,e),e=Ef(t,e,1),t=Yt(t,e,1),e=Pe(),t!==null&&(ei(t,1,e),Re(t,e));break}}t=t.return}}function ng(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Pe(),e.pingedLanes|=e.suspendedLanes&n,ke===e&&(Se&n)===n&&(ve===4||ve===3&&(Se&130023424)===Se&&500>pe()-cs?hn(e,0):us|=n),Re(e,t)}function $f(e,t){t===0&&(e.mode&1?(t=fi,fi<<=1,!(fi&130023424)&&(fi=4194304)):t=1);var n=Pe();e=It(e,t),e!==null&&(ei(e,t,n),Re(e,n))}function rg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),$f(e,n)}function ig(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(z(314))}r!==null&&r.delete(t),$f(e,n)}var Wf;Wf=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||De.current)Ae=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return Ae=!1,Wm(e,t,n);Ae=!!(e.flags&131072)}else Ae=!1,se&&t.flags&1048576&&qd(t,nl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Oi(e,t),e=t.pendingProps;var i=Xn(t,Ne.current);Wn(t,n),i=rs(null,t,r,e,i,n);var l=is();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Oe(r)?(l=!0,el(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,Ja(t),i.updater=El,t.stateNode=i,i._reactInternals=t,Jo(t,r,e,n),t=ta(null,t,r,!0,l,n)):(t.tag=0,se&&l&&Wa(t),ze(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Oi(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=og(r),e=at(r,e),i){case 0:t=ea(null,t,r,e,n);break e;case 1:t=Mu(null,t,r,e,n);break e;case 11:t=Tu(null,t,r,e,n);break e;case 14:t=Iu(null,t,r,at(r.type,e),n);break e}throw Error(z(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:at(r,i),ea(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:at(r,i),Mu(e,t,r,i,n);case 3:e:{if(_f(t),e===null)throw Error(z(387));r=t.pendingProps,l=t.memoizedState,i=l.element,ef(e,t),ll(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=er(Error(z(423)),t),t=Au(e,t,r,n,i);break e}else if(r!==i){i=er(Error(z(424)),t),t=Au(e,t,r,n,i);break e}else for($e=qt(t.stateNode.containerInfo.firstChild),Qe=t,se=!0,ut=null,n=Jd(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(Gn(),r===i){t=Mt(e,t,n);break e}ze(e,t,r,n)}t=t.child}return t;case 5:return tf(t),e===null&&Yo(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,$o(r,i)?o=null:l!==null&&$o(r,l)&&(t.flags|=32),Nf(e,t),ze(e,t,o,n),t.child;case 6:return e===null&&Yo(t),null;case 13:return zf(e,t,n);case 4:return Za(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=Jn(t,null,r,n):ze(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:at(r,i),Tu(e,t,r,i,n);case 7:return ze(e,t,t.pendingProps,n),t.child;case 8:return ze(e,t,t.pendingProps.children,n),t.child;case 12:return ze(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,re(rl,r._currentValue),r._currentValue=o,l!==null)if(ft(l.value,o)){if(l.children===i.children&&!De.current){t=Mt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Pt(-1,n&-n),s.tag=2;var u=l.updateQueue;if(u!==null){u=u.shared;var f=u.pending;f===null?s.next=s:(s.next=f.next,f.next=s),u.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),Xo(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(z(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),Xo(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}ze(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Wn(t,n),i=nt(i),r=r(i),t.flags|=1,ze(e,t,r,n),t.child;case 14:return r=t.type,i=at(r,t.pendingProps),i=at(r.type,i),Iu(e,t,r,i,n);case 15:return bf(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:at(r,i),Oi(e,t),t.tag=1,Oe(r)?(e=!0,el(t)):e=!1,Wn(t,n),Sf(t,r,i),Jo(t,r,i,n),ta(null,t,r,!0,e,n);case 19:return Pf(e,t,n);case 22:return jf(e,t,n)}throw Error(z(156,t.tag))};function Qf(e,t){return vd(e,t)}function lg(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function et(e,t,n,r){return new lg(e,t,n,r)}function hs(e){return e=e.prototype,!(!e||!e.isReactComponent)}function og(e){if(typeof e=="function")return hs(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Ia)return 11;if(e===Ma)return 14}return 2}function Jt(e,t){var n=e.alternate;return n===null?(n=et(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Bi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")hs(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Pn:return mn(n.children,i,l,t);case Ta:o=8,i|=8;break;case So:return e=et(12,n,t,i|2),e.elementType=So,e.lanes=l,e;case Co:return e=et(13,n,t,i),e.elementType=Co,e.lanes=l,e;case Eo:return e=et(19,n,t,i),e.elementType=Eo,e.lanes=l,e;case td:return Nl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case Zc:o=10;break e;case ed:o=9;break e;case Ia:o=11;break e;case Ma:o=14;break e;case Ft:o=16,r=null;break e}throw Error(z(130,e==null?e:typeof e,""))}return t=et(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function mn(e,t,n,r){return e=et(7,e,r,t),e.lanes=n,e}function Nl(e,t,n,r){return e=et(22,e,r,t),e.elementType=td,e.lanes=n,e.stateNode={isHidden:!1},e}function oo(e,t,n){return e=et(6,e,null,t),e.lanes=n,e}function ao(e,t,n){return t=et(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function ag(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Ul(0),this.expirationTimes=Ul(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Ul(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function ms(e,t,n,r,i,l,o,a,s){return e=new ag(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=et(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},Ja(l),e}function sg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:zn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function Kf(e){if(!e)return en;e=e._reactInternals;e:{if(Sn(e)!==e||e.tag!==1)throw Error(z(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Oe(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(z(171))}if(e.tag===1){var n=e.type;if(Oe(n))return Qd(e,n,t)}return t}function qf(e,t,n,r,i,l,o,a,s){return e=ms(n,r,!0,e,i,l,o,a,s),e.context=Kf(null),n=e.current,r=Pe(),i=Gt(n),l=Pt(r,i),l.callback=t??null,Yt(n,l,i),e.current.lanes=i,ei(e,i,r),Re(e,r),e}function _l(e,t,n,r){var i=t.current,l=Pe(),o=Gt(i);return n=Kf(n),t.context===null?t.context=n:t.pendingContext=n,t=Pt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=Yt(i,t,o),e!==null&&(dt(e,i,o,l),Mi(e,i,o)),o}function pl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function Wu(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function gs(e,t){Wu(e,t),(e=e.alternate)&&Wu(e,t)}function ug(){return null}var Yf=typeof reportError=="function"?reportError:function(e){console.error(e)};function ys(e){this._internalRoot=e}zl.prototype.render=ys.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(z(409));_l(e,t,null,null)};zl.prototype.unmount=ys.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;kn(function(){_l(null,e,null,null)}),t[Tt]=null}};function zl(e){this._internalRoot=e}zl.prototype.unstable_scheduleHydration=function(e){if(e){var t=bd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Ut.length&&t!==0&&t<Ut[n].priority;n++);Ut.splice(n,0,e),n===0&&Nd(e)}};function vs(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Pl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function Qu(){}function cg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var u=pl(o);l.call(u)}}var o=qf(t,r,e,0,null,!1,!1,"",Qu);return e._reactRootContainer=o,e[Tt]=o.current,Vr(e.nodeType===8?e.parentNode:e),kn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var u=pl(s);a.call(u)}}var s=ms(e,0,!1,null,null,!1,!1,"",Qu);return e._reactRootContainer=s,e[Tt]=s.current,Vr(e.nodeType===8?e.parentNode:e),kn(function(){_l(t,s,n,r)}),s}function Ll(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=pl(o);a.call(s)}}_l(t,o,e,i)}else o=cg(n,t,e,i,r);return pl(o)}Cd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=wr(t.pendingLanes);n!==0&&(Oa(t,n|1),Re(t,pe()),!(Y&6)&&(tr=pe()+500,rn()))}break;case 13:kn(function(){var r=It(e,1);if(r!==null){var i=Pe();dt(r,e,1,i)}}),gs(e,1)}};Ra=function(e){if(e.tag===13){var t=It(e,134217728);if(t!==null){var n=Pe();dt(t,e,134217728,n)}gs(e,134217728)}};Ed=function(e){if(e.tag===13){var t=Gt(e),n=It(e,t);if(n!==null){var r=Pe();dt(n,e,t,r)}gs(e,t)}};bd=function(){return J};jd=function(e,t){var n=J;try{return J=e,t()}finally{J=n}};Mo=function(e,t,n){switch(t){case"input":if(No(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=wl(r);if(!i)throw Error(z(90));rd(r),No(r,i)}}}break;case"textarea":ld(e,n);break;case"select":t=n.value,t!=null&&Un(e,!!n.multiple,t,!1)}};fd=ds;pd=kn;var dg={usingClientEntryPoint:!1,Events:[ni,Mn,wl,cd,dd,ds]},gr={findFiberByHostInstance:dn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},fg={bundleType:gr.bundleType,version:gr.version,rendererPackageName:gr.rendererPackageName,rendererConfig:gr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:At.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=gd(e),e===null?null:e.stateNode},findFiberByHostInstance:gr.findFiberByHostInstance||ug,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var Ci=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!Ci.isDisabled&&Ci.supportsFiber)try{yl=Ci.inject(fg),xt=Ci}catch{}}qe.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=dg;qe.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!vs(t))throw Error(z(200));return sg(e,t,null,n)};qe.createRoot=function(e,t){if(!vs(e))throw Error(z(299));var n=!1,r="",i=Yf;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=ms(e,1,!1,null,null,n,!1,r,i),e[Tt]=t.current,Vr(e.nodeType===8?e.parentNode:e),new ys(t)};qe.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(z(188)):(e=Object.keys(e).join(","),Error(z(268,e)));return e=gd(t),e=e===null?null:e.stateNode,e};qe.flushSync=function(e){return kn(e)};qe.hydrate=function(e,t,n){if(!Pl(t))throw Error(z(200));return Ll(null,e,t,!0,n)};qe.hydrateRoot=function(e,t,n){if(!vs(e))throw Error(z(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=Yf;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=qf(t,null,e,1,n??null,i,!1,l,o),e[Tt]=t.current,Vr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new zl(t)};qe.render=function(e,t,n){if(!Pl(t))throw Error(z(200));return Ll(null,e,t,!1,n)};qe.unmountComponentAtNode=function(e){if(!Pl(e))throw Error(z(40));return e._reactRootContainer?(kn(function(){Ll(null,null,e,!1,function(){e._reactRootContainer=null,e[Tt]=null})}),!0):!1};qe.unstable_batchedUpdates=ds;qe.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Pl(n))throw Error(z(200));if(e==null||e._reactInternals===void 0)throw Error(z(38));return Ll(e,t,n,!1,r)};qe.version="18.3.1-next-f1338f8080-20240426";function Xf(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Xf)}catch(e){console.error(e)}}Xf(),Yc.exports=qe;var pg=Yc.exports,Ku=pg;ko.createRoot=Ku.createRoot,ko.hydrateRoot=Ku.hydrateRoot;const Ei={plus:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),c.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),user:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),c.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),c.jsx("circle",{cx:"12",cy:"5",r:"2"}),c.jsx("path",{d:"M12 7v4"}),c.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),c.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),c.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),c.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),c.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]})},hg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,unreadCounts:i})=>{const[l,o]=U.useState(!1),[a,s]=U.useState(""),u=()=>{a.trim()&&(r(a.trim()),s(""),o(!1))},f=d=>{d.key==="Enter"&&!d.shiftKey?(d.preventDefault(),u()):d.key==="Escape"&&(o(!1),s(""))},h=d=>{const p=new Date(d),S=new Date().getTime()-p.getTime(),b=Math.floor(S/6e4),m=Math.floor(S/36e5),g=Math.floor(S/864e5);return b<1?"now":b<60?`${b}m`:m<24?`${m}h`:g<7?`${g}d`:p.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return c.jsxs("div",{className:"thread-list",children:[c.jsxs("div",{className:"list-header",children:[c.jsx("h2",{children:"Conversations"}),c.jsx("button",{className:"new-thread-btn",onClick:()=>o(!0),title:"New conversation",children:Ei.plus})]}),l&&c.jsxs("div",{className:"new-thread-form",children:[c.jsx("input",{type:"text",value:a,onChange:d=>s(d.target.value),onKeyDown:f,placeholder:"Conversation title...",autoFocus:!0}),c.jsxs("div",{className:"form-actions",children:[c.jsx("button",{className:"cancel-btn",onClick:()=>o(!1),children:"Cancel"}),c.jsx("button",{className:"create-btn",onClick:u,children:"Create"})]})]}),c.jsx("div",{className:"thread-items",children:e.length===0?c.jsxs("div",{className:"empty-state",children:[c.jsx("div",{className:"empty-icon",children:Ei.hash}),c.jsx("p",{children:"No conversations yet"}),c.jsx("button",{className:"start-btn",onClick:()=>o(!0),children:"Start a conversation"})]}):e.map(d=>{const p=i.get(d.id)||0,k=d.id===t;return c.jsxs("div",{className:`thread-item ${k?"selected":""} ${p>0?"has-unread":""}`,onClick:()=>n(d.id),children:[c.jsx("div",{className:`status-dot ${d.status}`}),c.jsxs("div",{className:"thread-content",children:[c.jsxs("div",{className:"thread-title-row",children:[c.jsx("span",{className:"thread-title",children:d.title}),c.jsx("span",{className:"thread-time",children:h(d.updated_at)})]}),c.jsxs("div",{className:"thread-meta",children:[c.jsxs("span",{className:"thread-creator",children:[d.created_by_type==="human"?Ei.user:Ei.bot,d.created_by_id]}),c.jsxs("span",{className:"thread-seq",children:["#",d.last_seq]})]})]}),p>0&&c.jsx("span",{className:"unread-badge",children:p})]},d.id)})}),c.jsx("style",{children:`
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
      `})]})};function mg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const gg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,yg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,vg={};function qu(e,t){return(vg.jsx?yg:gg).test(e)}const xg=/[ \t\n\f\r]/g;function kg(e){return typeof e=="object"?e.type==="text"?Yu(e.value):!1:Yu(e)}function Yu(e){return e.replace(xg,"")===""}class ii{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}ii.prototype.normal={};ii.prototype.property={};ii.prototype.space=void 0;function Gf(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new ii(n,r,t)}function pa(e){return e.toLowerCase()}class Be{constructor(t,n){this.attribute=n,this.property=t}}Be.prototype.attribute="";Be.prototype.booleanish=!1;Be.prototype.boolean=!1;Be.prototype.commaOrSpaceSeparated=!1;Be.prototype.commaSeparated=!1;Be.prototype.defined=!1;Be.prototype.mustUseProperty=!1;Be.prototype.number=!1;Be.prototype.overloadedBoolean=!1;Be.prototype.property="";Be.prototype.spaceSeparated=!1;Be.prototype.space=void 0;let wg=0;const W=Cn(),he=Cn(),ha=Cn(),P=Cn(),ne=Cn(),Kn=Cn(),He=Cn();function Cn(){return 2**++wg}const ma=Object.freeze(Object.defineProperty({__proto__:null,boolean:W,booleanish:he,commaOrSpaceSeparated:He,commaSeparated:Kn,number:P,overloadedBoolean:ha,spaceSeparated:ne},Symbol.toStringTag,{value:"Module"})),so=Object.keys(ma);class xs extends Be{constructor(t,n,r,i){let l=-1;if(super(t,n),Xu(this,"space",i),typeof r=="number")for(;++l<so.length;){const o=so[l];Xu(this,so[l],(r&ma[o])===ma[o])}}}xs.prototype.defined=!0;function Xu(e,t,n){n&&(e[t]=n)}function lr(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new xs(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[pa(r)]=r,n[pa(l.attribute)]=r}return new ii(t,n,e.space)}const Jf=lr({properties:{ariaActiveDescendant:null,ariaAtomic:he,ariaAutoComplete:null,ariaBusy:he,ariaChecked:he,ariaColCount:P,ariaColIndex:P,ariaColSpan:P,ariaControls:ne,ariaCurrent:null,ariaDescribedBy:ne,ariaDetails:null,ariaDisabled:he,ariaDropEffect:ne,ariaErrorMessage:null,ariaExpanded:he,ariaFlowTo:ne,ariaGrabbed:he,ariaHasPopup:null,ariaHidden:he,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:ne,ariaLevel:P,ariaLive:null,ariaModal:he,ariaMultiLine:he,ariaMultiSelectable:he,ariaOrientation:null,ariaOwns:ne,ariaPlaceholder:null,ariaPosInSet:P,ariaPressed:he,ariaReadOnly:he,ariaRelevant:null,ariaRequired:he,ariaRoleDescription:ne,ariaRowCount:P,ariaRowIndex:P,ariaRowSpan:P,ariaSelected:he,ariaSetSize:P,ariaSort:null,ariaValueMax:P,ariaValueMin:P,ariaValueNow:P,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function Zf(e,t){return t in e?e[t]:t}function ep(e,t){return Zf(e,t.toLowerCase())}const Sg=lr({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:Kn,acceptCharset:ne,accessKey:ne,action:null,allow:null,allowFullScreen:W,allowPaymentRequest:W,allowUserMedia:W,alt:null,as:null,async:W,autoCapitalize:null,autoComplete:ne,autoFocus:W,autoPlay:W,blocking:ne,capture:null,charSet:null,checked:W,cite:null,className:ne,cols:P,colSpan:null,content:null,contentEditable:he,controls:W,controlsList:ne,coords:P|Kn,crossOrigin:null,data:null,dateTime:null,decoding:null,default:W,defer:W,dir:null,dirName:null,disabled:W,download:ha,draggable:he,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:W,formTarget:null,headers:ne,height:P,hidden:ha,high:P,href:null,hrefLang:null,htmlFor:ne,httpEquiv:ne,id:null,imageSizes:null,imageSrcSet:null,inert:W,inputMode:null,integrity:null,is:null,isMap:W,itemId:null,itemProp:ne,itemRef:ne,itemScope:W,itemType:ne,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:W,low:P,manifest:null,max:null,maxLength:P,media:null,method:null,min:null,minLength:P,multiple:W,muted:W,name:null,nonce:null,noModule:W,noValidate:W,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:W,optimum:P,pattern:null,ping:ne,placeholder:null,playsInline:W,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:W,referrerPolicy:null,rel:ne,required:W,reversed:W,rows:P,rowSpan:P,sandbox:ne,scope:null,scoped:W,seamless:W,selected:W,shadowRootClonable:W,shadowRootDelegatesFocus:W,shadowRootMode:null,shape:null,size:P,sizes:null,slot:null,span:P,spellCheck:he,src:null,srcDoc:null,srcLang:null,srcSet:null,start:P,step:null,style:null,tabIndex:P,target:null,title:null,translate:null,type:null,typeMustMatch:W,useMap:null,value:he,width:P,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:ne,axis:null,background:null,bgColor:null,border:P,borderColor:null,bottomMargin:P,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:W,declare:W,event:null,face:null,frame:null,frameBorder:null,hSpace:P,leftMargin:P,link:null,longDesc:null,lowSrc:null,marginHeight:P,marginWidth:P,noResize:W,noHref:W,noShade:W,noWrap:W,object:null,profile:null,prompt:null,rev:null,rightMargin:P,rules:null,scheme:null,scrolling:he,standby:null,summary:null,text:null,topMargin:P,valueType:null,version:null,vAlign:null,vLink:null,vSpace:P,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:W,disableRemotePlayback:W,prefix:null,property:null,results:P,security:null,unselectable:null},space:"html",transform:ep}),Cg=lr({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:He,accentHeight:P,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:P,amplitude:P,arabicForm:null,ascent:P,attributeName:null,attributeType:null,azimuth:P,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:P,by:null,calcMode:null,capHeight:P,className:ne,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:P,diffuseConstant:P,direction:null,display:null,dur:null,divisor:P,dominantBaseline:null,download:W,dx:null,dy:null,edgeMode:null,editable:null,elevation:P,enableBackground:null,end:null,event:null,exponent:P,externalResourcesRequired:null,fill:null,fillOpacity:P,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:Kn,g2:Kn,glyphName:Kn,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:P,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:P,horizOriginX:P,horizOriginY:P,id:null,ideographic:P,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:P,k:P,k1:P,k2:P,k3:P,k4:P,kernelMatrix:He,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:P,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:P,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:P,overlineThickness:P,paintOrder:null,panose1:null,path:null,pathLength:P,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:ne,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:P,pointsAtY:P,pointsAtZ:P,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:He,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:He,rev:He,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:He,requiredFeatures:He,requiredFonts:He,requiredFormats:He,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:P,specularExponent:P,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:P,strikethroughThickness:P,string:null,stroke:null,strokeDashArray:He,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:P,strokeOpacity:P,strokeWidth:null,style:null,surfaceScale:P,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:He,tabIndex:P,tableValues:null,target:null,targetX:P,targetY:P,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:He,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:P,underlineThickness:P,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:P,values:null,vAlphabetic:P,vMathematical:P,vectorEffect:null,vHanging:P,vIdeographic:P,version:null,vertAdvY:P,vertOriginX:P,vertOriginY:P,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:P,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:Zf}),tp=lr({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),np=lr({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:ep}),rp=lr({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),Eg={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},bg=/[A-Z]/g,Gu=/-[a-z]/g,jg=/^data[-\w.:]+$/i;function Ng(e,t){const n=pa(t);let r=t,i=Be;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&jg.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(Gu,zg);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!Gu.test(l)){let o=l.replace(bg,_g);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=xs}return new i(r,t)}function _g(e){return"-"+e.toLowerCase()}function zg(e){return e.charAt(1).toUpperCase()}const Pg=Gf([Jf,Sg,tp,np,rp],"html"),ks=Gf([Jf,Cg,tp,np,rp],"svg");function Lg(e){return e.join(" ").trim()}var ws={},Ju=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Tg=/\n/g,Ig=/^\s*/,Mg=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Ag=/^:\s*/,Dg=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Og=/^[;\s]*/,Rg=/^\s+|\s+$/g,Fg=`
`,Zu="/",ec="*",un="",Bg="comment",Ug="declaration";function Hg(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(k){var S=k.match(Tg);S&&(n+=S.length);var b=k.lastIndexOf(Fg);r=~b?k.length-b:r+k.length}function l(){var k={line:n,column:r};return function(S){return S.position=new o(k),u(),S}}function o(k){this.start=k,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(k){var S=new Error(t.source+":"+n+":"+r+": "+k);if(S.reason=k,S.filename=t.source,S.line=n,S.column=r,S.source=e,!t.silent)throw S}function s(k){var S=k.exec(e);if(S){var b=S[0];return i(b),e=e.slice(b.length),S}}function u(){s(Ig)}function f(k){var S;for(k=k||[];S=h();)S!==!1&&k.push(S);return k}function h(){var k=l();if(!(Zu!=e.charAt(0)||ec!=e.charAt(1))){for(var S=2;un!=e.charAt(S)&&(ec!=e.charAt(S)||Zu!=e.charAt(S+1));)++S;if(S+=2,un===e.charAt(S-1))return a("End of comment missing");var b=e.slice(2,S-2);return r+=2,i(b),e=e.slice(S),r+=2,k({type:Bg,comment:b})}}function d(){var k=l(),S=s(Mg);if(S){if(h(),!s(Ag))return a("property missing ':'");var b=s(Dg),m=k({type:Ug,property:tc(S[0].replace(Ju,un)),value:b?tc(b[0].replace(Ju,un)):un});return s(Og),m}}function p(){var k=[];f(k);for(var S;S=d();)S!==!1&&(k.push(S),f(k));return k}return u(),p()}function tc(e){return e?e.replace(Rg,un):un}var Vg=Hg,$g=Vi&&Vi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(ws,"__esModule",{value:!0});ws.default=Qg;const Wg=$g(Vg);function Qg(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Wg.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Tl={};Object.defineProperty(Tl,"__esModule",{value:!0});Tl.camelCase=void 0;var Kg=/^--[a-zA-Z0-9_-]+$/,qg=/-([a-z])/g,Yg=/^[^-]+$/,Xg=/^-(webkit|moz|ms|o|khtml)-/,Gg=/^-(ms)-/,Jg=function(e){return!e||Yg.test(e)||Kg.test(e)},Zg=function(e,t){return t.toUpperCase()},nc=function(e,t){return"".concat(t,"-")},ey=function(e,t){return t===void 0&&(t={}),Jg(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(Gg,nc):e=e.replace(Xg,nc),e.replace(qg,Zg))};Tl.camelCase=ey;var ty=Vi&&Vi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},ny=ty(ws),ry=Tl;function ga(e,t){var n={};return!e||typeof e!="string"||(0,ny.default)(e,function(r,i){r&&i&&(n[(0,ry.camelCase)(r,t)]=i)}),n}ga.default=ga;var iy=ga;const ly=Ea(iy),ip=lp("end"),Ss=lp("start");function lp(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function oy(e){const t=Ss(e),n=ip(e);if(t&&n)return{start:t,end:n}}function Tr(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?rc(e.position):"start"in e||"end"in e?rc(e):"line"in e||"column"in e?ya(e):""}function ya(e){return ic(e&&e.line)+":"+ic(e&&e.column)}function rc(e){return ya(e&&e.start)+"-"+ya(e&&e.end)}function ic(e){return e&&typeof e=="number"?e:1}class _e extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Tr(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}_e.prototype.file="";_e.prototype.name="";_e.prototype.reason="";_e.prototype.message="";_e.prototype.stack="";_e.prototype.column=void 0;_e.prototype.line=void 0;_e.prototype.ancestors=void 0;_e.prototype.cause=void 0;_e.prototype.fatal=void 0;_e.prototype.place=void 0;_e.prototype.ruleId=void 0;_e.prototype.source=void 0;const Cs={}.hasOwnProperty,ay=new Map,sy=/[A-Z]/g,uy=new Set(["table","tbody","thead","tfoot","tr"]),cy=new Set(["td","th"]),op="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function dy(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=xy(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=vy(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?ks:Pg,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=ap(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function ap(e,t,n){if(t.type==="element")return fy(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return py(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return my(e,t,n);if(t.type==="mdxjsEsm")return hy(e,t);if(t.type==="root")return gy(e,t,n);if(t.type==="text")return yy(e,t)}function fy(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=ks,e.schema=i),e.ancestors.push(t);const l=up(e,t.tagName,!1),o=ky(e,t);let a=bs(e,t);return uy.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!kg(s):!0})),sp(e,o,l,t),Es(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function py(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}Jr(e,t.position)}function hy(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);Jr(e,t.position)}function my(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=ks,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:up(e,t.name,!0),o=wy(e,t),a=bs(e,t);return sp(e,o,l,t),Es(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function gy(e,t,n){const r={};return Es(r,bs(e,t)),e.create(t,e.Fragment,r,n)}function yy(e,t){return t.value}function sp(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Es(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function vy(e,t,n){return r;function r(i,l,o,a){const u=Array.isArray(o.children)?n:t;return a?u(l,o,a):u(l,o)}}function xy(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=Ss(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function ky(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Cs.call(t.properties,i)){const l=Sy(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&cy.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function wy(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else Jr(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else Jr(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function bs(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:ay;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const u=i.get(s)||0;o=s+"-"+u,i.set(s,u+1)}}const a=ap(e,l,o);a!==void 0&&n.push(a)}return n}function Sy(e,t,n){const r=Ng(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?mg(n):Lg(n)),r.property==="style"){let i=typeof n=="object"?n:Cy(e,String(n));return e.stylePropertyNameCase==="css"&&(i=Ey(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?Eg[r.property]||r.property:r.attribute,n]}}function Cy(e,t){try{return ly(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new _e("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=op+"#cannot-parse-style-attribute",i}}function up(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=qu(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=qu(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Cs.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);Jr(e)}function Jr(e,t){const n=new _e("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=op+"#cannot-handle-mdx-estrees-without-createevaluater",n}function Ey(e){const t={};let n;for(n in e)Cs.call(e,n)&&(t[by(n)]=e[n]);return t}function by(e){let t=e.replace(sy,jy);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function jy(e){return"-"+e.toLowerCase()}const uo={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},Ny={};function _y(e,t){const n=Ny,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return cp(e,r,i)}function cp(e,t,n){if(zy(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return lc(e.children,t,n)}return Array.isArray(e)?lc(e,t,n):""}function lc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=cp(e[i],t,n);return r.join("")}function zy(e){return!!(e&&typeof e=="object")}const oc=document.createElement("i");function js(e){const t="&"+e+";";oc.innerHTML=t;const n=oc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function wt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function Ze(e,t){return e.length>0?(wt(e,e.length,0,t),e):t}const ac={}.hasOwnProperty;function Py(e){const t={};let n=-1;for(;++n<e.length;)Ly(t,e[n]);return t}function Ly(e,t){let n;for(n in t){const i=(ac.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){ac.call(i,o)||(i[o]=[]);const a=l[o];Ty(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Ty(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);wt(e,0,0,r)}function dp(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function qn(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const vt=ln(/[A-Za-z]/),We=ln(/[\dA-Za-z]/),Iy=ln(/[#-'*+\--9=?A-Z^-~]/);function va(e){return e!==null&&(e<32||e===127)}const xa=ln(/\d/),My=ln(/[\dA-Fa-f]/),Ay=ln(/[!-/:-@[-`{-~]/);function V(e){return e!==null&&e<-2}function Fe(e){return e!==null&&(e<0||e===32)}function X(e){return e===-2||e===-1||e===32}const Dy=ln(new RegExp("\\p{P}|\\p{S}","u")),Oy=ln(/\s/);function ln(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function or(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&We(e.charCodeAt(n+1))&&We(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function ie(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return X(s)?(e.enter(n),a(s)):t(s)}function a(s){return X(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const Ry={tokenize:Fy};function Fy(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),ie(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return V(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const By={tokenize:Uy},sc={tokenize:Hy};function Uy(e){const t=this,n=[];let r=0,i,l,o;return a;function a(v){if(r<n.length){const C=n[r];return t.containerState=C[1],e.attempt(C[0].continuation,s,u)(v)}return u(v)}function s(v){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&g();const C=t.events.length;let E=C,w;for(;E--;)if(t.events[E][0]==="exit"&&t.events[E][1].type==="chunkFlow"){w=t.events[E][1].end;break}m(r);let N=C;for(;N<t.events.length;)t.events[N][1].end={...w},N++;return wt(t.events,E+1,0,t.events.slice(C)),t.events.length=N,u(v)}return a(v)}function u(v){if(r===n.length){if(!i)return d(v);if(i.currentConstruct&&i.currentConstruct.concrete)return k(v);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(sc,f,h)(v)}function f(v){return i&&g(),m(r),d(v)}function h(v){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,k(v)}function d(v){return t.containerState={},e.attempt(sc,p,k)(v)}function p(v){return r++,n.push([t.currentConstruct,t.containerState]),d(v)}function k(v){if(v===null){i&&g(),m(0),e.consume(v);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),S(v)}function S(v){if(v===null){b(e.exit("chunkFlow"),!0),m(0),e.consume(v);return}return V(v)?(e.consume(v),b(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(v),S)}function b(v,C){const E=t.sliceStream(v);if(C&&E.push(null),v.previous=l,l&&(l.next=v),l=v,i.defineSkip(v.start),i.write(E),t.parser.lazy[v.start.line]){let w=i.events.length;for(;w--;)if(i.events[w][1].start.offset<o&&(!i.events[w][1].end||i.events[w][1].end.offset>o))return;const N=t.events.length;let L=N,R,D;for(;L--;)if(t.events[L][0]==="exit"&&t.events[L][1].type==="chunkFlow"){if(R){D=t.events[L][1].end;break}R=!0}for(m(r),w=N;w<t.events.length;)t.events[w][1].end={...D},w++;wt(t.events,L+1,0,t.events.slice(N)),t.events.length=w}}function m(v){let C=n.length;for(;C-- >v;){const E=n[C];t.containerState=E[1],E[0].exit.call(t,e)}n.length=v}function g(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Hy(e,t,n){return ie(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function uc(e){if(e===null||Fe(e)||Oy(e))return 1;if(Dy(e))return 2}function Ns(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const ka={name:"attention",resolveAll:Vy,tokenize:$y};function Vy(e,t){let n=-1,r,i,l,o,a,s,u,f;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const h={...e[r][1].end},d={...e[n][1].start};cc(h,-s),cc(d,s),o={type:s>1?"strongSequence":"emphasisSequence",start:h,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:d},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},u=[],e[r][1].end.offset-e[r][1].start.offset&&(u=Ze(u,[["enter",e[r][1],t],["exit",e[r][1],t]])),u=Ze(u,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),u=Ze(u,Ns(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),u=Ze(u,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(f=2,u=Ze(u,[["enter",e[n][1],t],["exit",e[n][1],t]])):f=0,wt(e,r-1,n-r+3,u),n=r+u.length-f-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function $y(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=uc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const u=e.exit("attentionSequence"),f=uc(s),h=!f||f===2&&i||n.includes(s),d=!i||i===2&&f||n.includes(r);return u._open=!!(l===42?h:h&&(i||!d)),u._close=!!(l===42?d:d&&(f||!h)),t(s)}}function cc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Wy={name:"autolink",tokenize:Qy};function Qy(e,t,n){let r=0;return i;function i(p){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(p){return vt(p)?(e.consume(p),o):p===64?n(p):u(p)}function o(p){return p===43||p===45||p===46||We(p)?(r=1,a(p)):u(p)}function a(p){return p===58?(e.consume(p),r=0,s):(p===43||p===45||p===46||We(p))&&r++<32?(e.consume(p),a):(r=0,u(p))}function s(p){return p===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):p===null||p===32||p===60||va(p)?n(p):(e.consume(p),s)}function u(p){return p===64?(e.consume(p),f):Iy(p)?(e.consume(p),u):n(p)}function f(p){return We(p)?h(p):n(p)}function h(p){return p===46?(e.consume(p),r=0,f):p===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):d(p)}function d(p){if((p===45||We(p))&&r++<63){const k=p===45?d:h;return e.consume(p),k}return n(p)}}const Il={partial:!0,tokenize:Ky};function Ky(e,t,n){return r;function r(l){return X(l)?ie(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||V(l)?t(l):n(l)}}const fp={continuation:{tokenize:Yy},exit:Xy,name:"blockQuote",tokenize:qy};function qy(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return X(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function Yy(e,t,n){const r=this;return i;function i(o){return X(o)?ie(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(fp,t,n)(o)}}function Xy(e){e.exit("blockQuote")}const pp={name:"characterEscape",tokenize:Gy};function Gy(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return Ay(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const hp={name:"characterReference",tokenize:Jy};function Jy(e,t,n){const r=this;let i=0,l,o;return a;function a(h){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(h),e.exit("characterReferenceMarker"),s}function s(h){return h===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(h),e.exit("characterReferenceMarkerNumeric"),u):(e.enter("characterReferenceValue"),l=31,o=We,f(h))}function u(h){return h===88||h===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(h),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=My,f):(e.enter("characterReferenceValue"),l=7,o=xa,f(h))}function f(h){if(h===59&&i){const d=e.exit("characterReferenceValue");return o===We&&!js(r.sliceSerialize(d))?n(h):(e.enter("characterReferenceMarker"),e.consume(h),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(h)&&i++<l?(e.consume(h),f):n(h)}}const dc={partial:!0,tokenize:ev},fc={concrete:!0,name:"codeFenced",tokenize:Zy};function Zy(e,t,n){const r=this,i={partial:!0,tokenize:E};let l=0,o=0,a;return s;function s(w){return u(w)}function u(w){const N=r.events[r.events.length-1];return l=N&&N[1].type==="linePrefix"?N[2].sliceSerialize(N[1],!0).length:0,a=w,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),f(w)}function f(w){return w===a?(o++,e.consume(w),f):o<3?n(w):(e.exit("codeFencedFenceSequence"),X(w)?ie(e,h,"whitespace")(w):h(w))}function h(w){return w===null||V(w)?(e.exit("codeFencedFence"),r.interrupt?t(w):e.check(dc,S,C)(w)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),d(w))}function d(w){return w===null||V(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),h(w)):X(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),ie(e,p,"whitespace")(w)):w===96&&w===a?n(w):(e.consume(w),d)}function p(w){return w===null||V(w)?h(w):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),k(w))}function k(w){return w===null||V(w)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),h(w)):w===96&&w===a?n(w):(e.consume(w),k)}function S(w){return e.attempt(i,C,b)(w)}function b(w){return e.enter("lineEnding"),e.consume(w),e.exit("lineEnding"),m}function m(w){return l>0&&X(w)?ie(e,g,"linePrefix",l+1)(w):g(w)}function g(w){return w===null||V(w)?e.check(dc,S,C)(w):(e.enter("codeFlowValue"),v(w))}function v(w){return w===null||V(w)?(e.exit("codeFlowValue"),g(w)):(e.consume(w),v)}function C(w){return e.exit("codeFenced"),t(w)}function E(w,N,L){let R=0;return D;function D(H){return w.enter("lineEnding"),w.consume(H),w.exit("lineEnding"),O}function O(H){return w.enter("codeFencedFence"),X(H)?ie(w,_,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(H):_(H)}function _(H){return H===a?(w.enter("codeFencedFenceSequence"),M(H)):L(H)}function M(H){return H===a?(R++,w.consume(H),M):R>=o?(w.exit("codeFencedFenceSequence"),X(H)?ie(w,B,"whitespace")(H):B(H)):L(H)}function B(H){return H===null||V(H)?(w.exit("codeFencedFence"),N(H)):L(H)}}}function ev(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const co={name:"codeIndented",tokenize:nv},tv={partial:!0,tokenize:rv};function nv(e,t,n){const r=this;return i;function i(u){return e.enter("codeIndented"),ie(e,l,"linePrefix",5)(u)}function l(u){const f=r.events[r.events.length-1];return f&&f[1].type==="linePrefix"&&f[2].sliceSerialize(f[1],!0).length>=4?o(u):n(u)}function o(u){return u===null?s(u):V(u)?e.attempt(tv,o,s)(u):(e.enter("codeFlowValue"),a(u))}function a(u){return u===null||V(u)?(e.exit("codeFlowValue"),o(u)):(e.consume(u),a)}function s(u){return e.exit("codeIndented"),t(u)}}function rv(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):V(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):ie(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):V(o)?i(o):n(o)}}const iv={name:"codeText",previous:ov,resolve:lv,tokenize:av};function lv(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function ov(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function av(e,t,n){let r=0,i,l;return o;function o(h){return e.enter("codeText"),e.enter("codeTextSequence"),a(h)}function a(h){return h===96?(e.consume(h),r++,a):(e.exit("codeTextSequence"),s(h))}function s(h){return h===null?n(h):h===32?(e.enter("space"),e.consume(h),e.exit("space"),s):h===96?(l=e.enter("codeTextSequence"),i=0,f(h)):V(h)?(e.enter("lineEnding"),e.consume(h),e.exit("lineEnding"),s):(e.enter("codeTextData"),u(h))}function u(h){return h===null||h===32||h===96||V(h)?(e.exit("codeTextData"),s(h)):(e.consume(h),u)}function f(h){return h===96?(e.consume(h),i++,f):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(h)):(l.type="codeTextData",u(h))}}class sv{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&yr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),yr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),yr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);yr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);yr(this.left,n.reverse())}}}function yr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function mp(e){const t={};let n=-1,r,i,l,o,a,s,u;const f=new sv(e);for(;++n<f.length;){for(;n in t;)n=t[n];if(r=f.get(n),n&&r[1].type==="chunkFlow"&&f.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,uv(f,n)),n=t[n],u=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=f.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(f.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...f.get(i)[1].start},a=f.slice(i,n),a.unshift(r),f.splice(i,n-i+1,a))}}return wt(e,0,Number.POSITIVE_INFINITY,f.slice(0)),!u}function uv(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],u={};let f,h,d=-1,p=n,k=0,S=0;const b=[S];for(;p;){for(;e.get(++i)[1]!==p;);l.push(i),p._tokenizer||(f=r.sliceStream(p),p.next||f.push(null),h&&o.defineSkip(p.start),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(f),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),h=p,p=p.next}for(p=n;++d<a.length;)a[d][0]==="exit"&&a[d-1][0]==="enter"&&a[d][1].type===a[d-1][1].type&&a[d][1].start.line!==a[d][1].end.line&&(S=d+1,b.push(S),p._tokenizer=void 0,p.previous=void 0,p=p.next);for(o.events=[],p?(p._tokenizer=void 0,p.previous=void 0):b.pop(),d=b.length;d--;){const m=a.slice(b[d],b[d+1]),g=l.pop();s.push([g,g+m.length-1]),e.splice(g,2,m)}for(s.reverse(),d=-1;++d<s.length;)u[k+s[d][0]]=k+s[d][1],k+=s[d][1]-s[d][0]-1;return u}const cv={resolve:fv,tokenize:pv},dv={partial:!0,tokenize:hv};function fv(e){return mp(e),e}function pv(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):V(a)?e.check(dv,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function hv(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),ie(e,l,"linePrefix")}function l(o){if(o===null||V(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function gp(e,t,n,r,i,l,o,a,s){const u=s||Number.POSITIVE_INFINITY;let f=0;return h;function h(m){return m===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(m),e.exit(l),d):m===null||m===32||m===41||va(m)?n(m):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),S(m))}function d(m){return m===62?(e.enter(l),e.consume(m),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),p(m))}function p(m){return m===62?(e.exit("chunkString"),e.exit(a),d(m)):m===null||m===60||V(m)?n(m):(e.consume(m),m===92?k:p)}function k(m){return m===60||m===62||m===92?(e.consume(m),p):p(m)}function S(m){return!f&&(m===null||m===41||Fe(m))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(m)):f<u&&m===40?(e.consume(m),f++,S):m===41?(e.consume(m),f--,S):m===null||m===32||m===40||va(m)?n(m):(e.consume(m),m===92?b:S)}function b(m){return m===40||m===41||m===92?(e.consume(m),S):S(m)}}function yp(e,t,n,r,i,l){const o=this;let a=0,s;return u;function u(p){return e.enter(r),e.enter(i),e.consume(p),e.exit(i),e.enter(l),f}function f(p){return a>999||p===null||p===91||p===93&&!s||p===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(p):p===93?(e.exit(l),e.enter(i),e.consume(p),e.exit(i),e.exit(r),t):V(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),f):(e.enter("chunkString",{contentType:"string"}),h(p))}function h(p){return p===null||p===91||p===93||V(p)||a++>999?(e.exit("chunkString"),f(p)):(e.consume(p),s||(s=!X(p)),p===92?d:h)}function d(p){return p===91||p===92||p===93?(e.consume(p),a++,h):h(p)}}function vp(e,t,n,r,i,l){let o;return a;function a(d){return d===34||d===39||d===40?(e.enter(r),e.enter(i),e.consume(d),e.exit(i),o=d===40?41:d,s):n(d)}function s(d){return d===o?(e.enter(i),e.consume(d),e.exit(i),e.exit(r),t):(e.enter(l),u(d))}function u(d){return d===o?(e.exit(l),s(o)):d===null?n(d):V(d)?(e.enter("lineEnding"),e.consume(d),e.exit("lineEnding"),ie(e,u,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),f(d))}function f(d){return d===o||d===null||V(d)?(e.exit("chunkString"),u(d)):(e.consume(d),d===92?h:f)}function h(d){return d===o||d===92?(e.consume(d),f):f(d)}}function Ir(e,t){let n;return r;function r(i){return V(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):X(i)?ie(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const mv={name:"definition",tokenize:yv},gv={partial:!0,tokenize:vv};function yv(e,t,n){const r=this;let i;return l;function l(p){return e.enter("definition"),o(p)}function o(p){return yp.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(p)}function a(p){return i=qn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),p===58?(e.enter("definitionMarker"),e.consume(p),e.exit("definitionMarker"),s):n(p)}function s(p){return Fe(p)?Ir(e,u)(p):u(p)}function u(p){return gp(e,f,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(p)}function f(p){return e.attempt(gv,h,h)(p)}function h(p){return X(p)?ie(e,d,"whitespace")(p):d(p)}function d(p){return p===null||V(p)?(e.exit("definition"),r.parser.defined.push(i),t(p)):n(p)}}function vv(e,t,n){return r;function r(a){return Fe(a)?Ir(e,i)(a):n(a)}function i(a){return vp(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return X(a)?ie(e,o,"whitespace")(a):o(a)}function o(a){return a===null||V(a)?t(a):n(a)}}const xv={name:"hardBreakEscape",tokenize:kv};function kv(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return V(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const wv={name:"headingAtx",resolve:Sv,tokenize:Cv};function Sv(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},wt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function Cv(e,t,n){let r=0;return i;function i(f){return e.enter("atxHeading"),l(f)}function l(f){return e.enter("atxHeadingSequence"),o(f)}function o(f){return f===35&&r++<6?(e.consume(f),o):f===null||Fe(f)?(e.exit("atxHeadingSequence"),a(f)):n(f)}function a(f){return f===35?(e.enter("atxHeadingSequence"),s(f)):f===null||V(f)?(e.exit("atxHeading"),t(f)):X(f)?ie(e,a,"whitespace")(f):(e.enter("atxHeadingText"),u(f))}function s(f){return f===35?(e.consume(f),s):(e.exit("atxHeadingSequence"),a(f))}function u(f){return f===null||f===35||Fe(f)?(e.exit("atxHeadingText"),a(f)):(e.consume(f),u)}}const Ev=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],pc=["pre","script","style","textarea"],bv={concrete:!0,name:"htmlFlow",resolveTo:_v,tokenize:zv},jv={partial:!0,tokenize:Lv},Nv={partial:!0,tokenize:Pv};function _v(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function zv(e,t,n){const r=this;let i,l,o,a,s;return u;function u(x){return f(x)}function f(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),h}function h(x){return x===33?(e.consume(x),d):x===47?(e.consume(x),l=!0,S):x===63?(e.consume(x),i=3,r.interrupt?t:y):vt(x)?(e.consume(x),o=String.fromCharCode(x),b):n(x)}function d(x){return x===45?(e.consume(x),i=2,p):x===91?(e.consume(x),i=5,a=0,k):vt(x)?(e.consume(x),i=4,r.interrupt?t:y):n(x)}function p(x){return x===45?(e.consume(x),r.interrupt?t:y):n(x)}function k(x){const ge="CDATA[";return x===ge.charCodeAt(a++)?(e.consume(x),a===ge.length?r.interrupt?t:_:k):n(x)}function S(x){return vt(x)?(e.consume(x),o=String.fromCharCode(x),b):n(x)}function b(x){if(x===null||x===47||x===62||Fe(x)){const ge=x===47,it=o.toLowerCase();return!ge&&!l&&pc.includes(it)?(i=1,r.interrupt?t(x):_(x)):Ev.includes(o.toLowerCase())?(i=6,ge?(e.consume(x),m):r.interrupt?t(x):_(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?g(x):v(x))}return x===45||We(x)?(e.consume(x),o+=String.fromCharCode(x),b):n(x)}function m(x){return x===62?(e.consume(x),r.interrupt?t:_):n(x)}function g(x){return X(x)?(e.consume(x),g):D(x)}function v(x){return x===47?(e.consume(x),D):x===58||x===95||vt(x)?(e.consume(x),C):X(x)?(e.consume(x),v):D(x)}function C(x){return x===45||x===46||x===58||x===95||We(x)?(e.consume(x),C):E(x)}function E(x){return x===61?(e.consume(x),w):X(x)?(e.consume(x),E):v(x)}function w(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,N):X(x)?(e.consume(x),w):L(x)}function N(x){return x===s?(e.consume(x),s=null,R):x===null||V(x)?n(x):(e.consume(x),N)}function L(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||Fe(x)?E(x):(e.consume(x),L)}function R(x){return x===47||x===62||X(x)?v(x):n(x)}function D(x){return x===62?(e.consume(x),O):n(x)}function O(x){return x===null||V(x)?_(x):X(x)?(e.consume(x),O):n(x)}function _(x){return x===45&&i===2?(e.consume(x),G):x===60&&i===1?(e.consume(x),te):x===62&&i===4?(e.consume(x),q):x===63&&i===3?(e.consume(x),y):x===93&&i===5?(e.consume(x),F):V(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(jv,Z,M)(x)):x===null||V(x)?(e.exit("htmlFlowData"),M(x)):(e.consume(x),_)}function M(x){return e.check(Nv,B,Z)(x)}function B(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),H}function H(x){return x===null||V(x)?M(x):(e.enter("htmlFlowData"),_(x))}function G(x){return x===45?(e.consume(x),y):_(x)}function te(x){return x===47?(e.consume(x),o="",T):_(x)}function T(x){if(x===62){const ge=o.toLowerCase();return pc.includes(ge)?(e.consume(x),q):_(x)}return vt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),T):_(x)}function F(x){return x===93?(e.consume(x),y):_(x)}function y(x){return x===62?(e.consume(x),q):x===45&&i===2?(e.consume(x),y):_(x)}function q(x){return x===null||V(x)?(e.exit("htmlFlowData"),Z(x)):(e.consume(x),q)}function Z(x){return e.exit("htmlFlow"),t(x)}}function Pv(e,t,n){const r=this;return i;function i(o){return V(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Lv(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Il,t,n)}}const Tv={name:"htmlText",tokenize:Iv};function Iv(e,t,n){const r=this;let i,l,o;return a;function a(y){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(y),s}function s(y){return y===33?(e.consume(y),u):y===47?(e.consume(y),E):y===63?(e.consume(y),v):vt(y)?(e.consume(y),L):n(y)}function u(y){return y===45?(e.consume(y),f):y===91?(e.consume(y),l=0,k):vt(y)?(e.consume(y),g):n(y)}function f(y){return y===45?(e.consume(y),p):n(y)}function h(y){return y===null?n(y):y===45?(e.consume(y),d):V(y)?(o=h,te(y)):(e.consume(y),h)}function d(y){return y===45?(e.consume(y),p):h(y)}function p(y){return y===62?G(y):y===45?d(y):h(y)}function k(y){const q="CDATA[";return y===q.charCodeAt(l++)?(e.consume(y),l===q.length?S:k):n(y)}function S(y){return y===null?n(y):y===93?(e.consume(y),b):V(y)?(o=S,te(y)):(e.consume(y),S)}function b(y){return y===93?(e.consume(y),m):S(y)}function m(y){return y===62?G(y):y===93?(e.consume(y),m):S(y)}function g(y){return y===null||y===62?G(y):V(y)?(o=g,te(y)):(e.consume(y),g)}function v(y){return y===null?n(y):y===63?(e.consume(y),C):V(y)?(o=v,te(y)):(e.consume(y),v)}function C(y){return y===62?G(y):v(y)}function E(y){return vt(y)?(e.consume(y),w):n(y)}function w(y){return y===45||We(y)?(e.consume(y),w):N(y)}function N(y){return V(y)?(o=N,te(y)):X(y)?(e.consume(y),N):G(y)}function L(y){return y===45||We(y)?(e.consume(y),L):y===47||y===62||Fe(y)?R(y):n(y)}function R(y){return y===47?(e.consume(y),G):y===58||y===95||vt(y)?(e.consume(y),D):V(y)?(o=R,te(y)):X(y)?(e.consume(y),R):G(y)}function D(y){return y===45||y===46||y===58||y===95||We(y)?(e.consume(y),D):O(y)}function O(y){return y===61?(e.consume(y),_):V(y)?(o=O,te(y)):X(y)?(e.consume(y),O):R(y)}function _(y){return y===null||y===60||y===61||y===62||y===96?n(y):y===34||y===39?(e.consume(y),i=y,M):V(y)?(o=_,te(y)):X(y)?(e.consume(y),_):(e.consume(y),B)}function M(y){return y===i?(e.consume(y),i=void 0,H):y===null?n(y):V(y)?(o=M,te(y)):(e.consume(y),M)}function B(y){return y===null||y===34||y===39||y===60||y===61||y===96?n(y):y===47||y===62||Fe(y)?R(y):(e.consume(y),B)}function H(y){return y===47||y===62||Fe(y)?R(y):n(y)}function G(y){return y===62?(e.consume(y),e.exit("htmlTextData"),e.exit("htmlText"),t):n(y)}function te(y){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(y),e.exit("lineEnding"),T}function T(y){return X(y)?ie(e,F,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(y):F(y)}function F(y){return e.enter("htmlTextData"),o(y)}}const _s={name:"labelEnd",resolveAll:Ov,resolveTo:Rv,tokenize:Fv},Mv={tokenize:Bv},Av={tokenize:Uv},Dv={tokenize:Hv};function Ov(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&wt(e,0,e.length,n),e}function Rv(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},u={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},f={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",u,t]],a=Ze(a,e.slice(l+1,l+r+3)),a=Ze(a,[["enter",f,t]]),a=Ze(a,Ns(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=Ze(a,[["exit",f,t],e[o-2],e[o-1],["exit",u,t]]),a=Ze(a,e.slice(o+1)),a=Ze(a,[["exit",s,t]]),wt(e,l,e.length,a),e}function Fv(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(d){return l?l._inactive?h(d):(o=r.parser.defined.includes(qn(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(d),e.exit("labelMarker"),e.exit("labelEnd"),s):n(d)}function s(d){return d===40?e.attempt(Mv,f,o?f:h)(d):d===91?e.attempt(Av,f,o?u:h)(d):o?f(d):h(d)}function u(d){return e.attempt(Dv,f,h)(d)}function f(d){return t(d)}function h(d){return l._balanced=!0,n(d)}}function Bv(e,t,n){return r;function r(h){return e.enter("resource"),e.enter("resourceMarker"),e.consume(h),e.exit("resourceMarker"),i}function i(h){return Fe(h)?Ir(e,l)(h):l(h)}function l(h){return h===41?f(h):gp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(h)}function o(h){return Fe(h)?Ir(e,s)(h):f(h)}function a(h){return n(h)}function s(h){return h===34||h===39||h===40?vp(e,u,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(h):f(h)}function u(h){return Fe(h)?Ir(e,f)(h):f(h)}function f(h){return h===41?(e.enter("resourceMarker"),e.consume(h),e.exit("resourceMarker"),e.exit("resource"),t):n(h)}}function Uv(e,t,n){const r=this;return i;function i(a){return yp.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(qn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Hv(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Vv={name:"labelStartImage",resolveAll:_s.resolveAll,tokenize:$v};function $v(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Wv={name:"labelStartLink",resolveAll:_s.resolveAll,tokenize:Qv};function Qv(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const fo={name:"lineEnding",tokenize:Kv};function Kv(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),ie(e,t,"linePrefix")}}const Ui={name:"thematicBreak",tokenize:qv};function qv(e,t,n){let r=0,i;return l;function l(u){return e.enter("thematicBreak"),o(u)}function o(u){return i=u,a(u)}function a(u){return u===i?(e.enter("thematicBreakSequence"),s(u)):r>=3&&(u===null||V(u))?(e.exit("thematicBreak"),t(u)):n(u)}function s(u){return u===i?(e.consume(u),r++,s):(e.exit("thematicBreakSequence"),X(u)?ie(e,a,"whitespace")(u):a(u))}}const Ie={continuation:{tokenize:Jv},exit:ex,name:"list",tokenize:Gv},Yv={partial:!0,tokenize:tx},Xv={partial:!0,tokenize:Zv};function Gv(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(p){const k=r.containerState.type||(p===42||p===43||p===45?"listUnordered":"listOrdered");if(k==="listUnordered"?!r.containerState.marker||p===r.containerState.marker:xa(p)){if(r.containerState.type||(r.containerState.type=k,e.enter(k,{_container:!0})),k==="listUnordered")return e.enter("listItemPrefix"),p===42||p===45?e.check(Ui,n,u)(p):u(p);if(!r.interrupt||p===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(p)}return n(p)}function s(p){return xa(p)&&++o<10?(e.consume(p),s):(!r.interrupt||o<2)&&(r.containerState.marker?p===r.containerState.marker:p===41||p===46)?(e.exit("listItemValue"),u(p)):n(p)}function u(p){return e.enter("listItemMarker"),e.consume(p),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||p,e.check(Il,r.interrupt?n:f,e.attempt(Yv,d,h))}function f(p){return r.containerState.initialBlankLine=!0,l++,d(p)}function h(p){return X(p)?(e.enter("listItemPrefixWhitespace"),e.consume(p),e.exit("listItemPrefixWhitespace"),d):n(p)}function d(p){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(p)}}function Jv(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Il,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,ie(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!X(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Xv,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,ie(e,e.attempt(Ie,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function Zv(e,t,n){const r=this;return ie(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function ex(e){e.exit(this.containerState.type)}function tx(e,t,n){const r=this;return ie(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!X(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const hc={name:"setextUnderline",resolveTo:nx,tokenize:rx};function nx(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function rx(e,t,n){const r=this;let i;return l;function l(u){let f=r.events.length,h;for(;f--;)if(r.events[f][1].type!=="lineEnding"&&r.events[f][1].type!=="linePrefix"&&r.events[f][1].type!=="content"){h=r.events[f][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||h)?(e.enter("setextHeadingLine"),i=u,o(u)):n(u)}function o(u){return e.enter("setextHeadingLineSequence"),a(u)}function a(u){return u===i?(e.consume(u),a):(e.exit("setextHeadingLineSequence"),X(u)?ie(e,s,"lineSuffix")(u):s(u))}function s(u){return u===null||V(u)?(e.exit("setextHeadingLine"),t(u)):n(u)}}const ix={tokenize:lx};function lx(e){const t=this,n=e.attempt(Il,r,e.attempt(this.parser.constructs.flowInitial,i,ie(e,e.attempt(this.parser.constructs.flow,i,e.attempt(cv,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const ox={resolveAll:kp()},ax=xp("string"),sx=xp("text");function xp(e){return{resolveAll:kp(e==="text"?ux:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(f){return u(f)?l(f):a(f)}function a(f){if(f===null){n.consume(f);return}return n.enter("data"),n.consume(f),s}function s(f){return u(f)?(n.exit("data"),l(f)):(n.consume(f),s)}function u(f){if(f===null)return!0;const h=i[f];let d=-1;if(h)for(;++d<h.length;){const p=h[d];if(!p.previous||p.previous.call(r,r.previous))return!0}return!1}}}function kp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function ux(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const u=i[l];if(typeof u=="string"){for(o=u.length;u.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(u===-2)s=!0,a++;else if(u!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const u={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...u.start},r.start.offset===r.end.offset?Object.assign(r,u):(e.splice(n,0,["enter",u,t],["exit",u,t]),n+=2)}n++}return e}const cx={42:Ie,43:Ie,45:Ie,48:Ie,49:Ie,50:Ie,51:Ie,52:Ie,53:Ie,54:Ie,55:Ie,56:Ie,57:Ie,62:fp},dx={91:mv},fx={[-2]:co,[-1]:co,32:co},px={35:wv,42:Ui,45:[hc,Ui],60:bv,61:hc,95:Ui,96:fc,126:fc},hx={38:hp,92:pp},mx={[-5]:fo,[-4]:fo,[-3]:fo,33:Vv,38:hp,42:ka,60:[Wy,Tv],91:Wv,92:[xv,pp],93:_s,95:ka,96:iv},gx={null:[ka,ox]},yx={null:[42,95]},vx={null:[]},xx=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:yx,contentInitial:dx,disable:vx,document:cx,flow:px,flowInitial:fx,insideSpan:gx,string:hx,text:mx},Symbol.toStringTag,{value:"Module"}));function kx(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:N(E),check:N(w),consume:g,enter:v,exit:C,interrupt:N(w,{interrupt:!0})},u={code:null,containerState:{},defineSkip:S,events:[],now:k,parser:e,previous:null,sliceSerialize:d,sliceStream:p,write:h};let f=t.tokenize.call(u,s);return t.resolveAll&&l.push(t),u;function h(O){return o=Ze(o,O),b(),o[o.length-1]!==null?[]:(L(t,0),u.events=Ns(l,u.events,u),u.events)}function d(O,_){return Sx(p(O),_)}function p(O){return wx(o,O)}function k(){const{_bufferIndex:O,_index:_,line:M,column:B,offset:H}=r;return{_bufferIndex:O,_index:_,line:M,column:B,offset:H}}function S(O){i[O.line]=O.column,D()}function b(){let O;for(;r._index<o.length;){const _=o[r._index];if(typeof _=="string")for(O=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===O&&r._bufferIndex<_.length;)m(_.charCodeAt(r._bufferIndex));else m(_)}}function m(O){f=f(O)}function g(O){V(O)?(r.line++,r.column=1,r.offset+=O===-3?2:1,D()):O!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),u.previous=O}function v(O,_){const M=_||{};return M.type=O,M.start=k(),u.events.push(["enter",M,u]),a.push(M),M}function C(O){const _=a.pop();return _.end=k(),u.events.push(["exit",_,u]),_}function E(O,_){L(O,_.from)}function w(O,_){_.restore()}function N(O,_){return M;function M(B,H,G){let te,T,F,y;return Array.isArray(B)?Z(B):"tokenize"in B?Z([B]):q(B);function q(le){return pt;function pt(Dt){const En=Dt!==null&&le[Dt],bn=Dt!==null&&le.null,oi=[...Array.isArray(En)?En:En?[En]:[],...Array.isArray(bn)?bn:bn?[bn]:[]];return Z(oi)(Dt)}}function Z(le){return te=le,T=0,le.length===0?G:x(le[T])}function x(le){return pt;function pt(Dt){return y=R(),F=le,le.partial||(u.currentConstruct=le),le.name&&u.parser.constructs.disable.null.includes(le.name)?it():le.tokenize.call(_?Object.assign(Object.create(u),_):u,s,ge,it)(Dt)}}function ge(le){return O(F,y),H}function it(le){return y.restore(),++T<te.length?x(te[T]):G}}}function L(O,_){O.resolveAll&&!l.includes(O)&&l.push(O),O.resolve&&wt(u.events,_,u.events.length-_,O.resolve(u.events.slice(_),u)),O.resolveTo&&(u.events=O.resolveTo(u.events,u))}function R(){const O=k(),_=u.previous,M=u.currentConstruct,B=u.events.length,H=Array.from(a);return{from:B,restore:G};function G(){r=O,u.previous=_,u.currentConstruct=M,u.events.length=B,a=H,D()}}function D(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function wx(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function Sx(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function Cx(e){const r={constructs:Py([xx,...(e||{}).extensions||[]]),content:i(Ry),defined:[],document:i(By),flow:i(ix),lazy:{},string:i(ax),text:i(sx)};return r;function i(l){return o;function o(a){return kx(r,l,a)}}}function Ex(e){for(;!mp(e););return e}const mc=/[\0\t\n\r]/g;function bx(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let u,f,h,d,p;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),h=0,t="",n&&(l.charCodeAt(0)===65279&&h++,n=void 0);h<l.length;){if(mc.lastIndex=h,u=mc.exec(l),d=u&&u.index!==void 0?u.index:l.length,p=l.charCodeAt(d),!u){t=l.slice(h);break}if(p===10&&h===d&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),h<d&&(s.push(l.slice(h,d)),e+=d-h),p){case 0:{s.push(65533),e++;break}case 9:{for(f=Math.ceil(e/4)*4,s.push(-2);e++<f;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}h=d+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const jx=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function Nx(e){return e.replace(jx,_x)}function _x(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return dp(n.slice(l?2:1),l?16:10)}return js(n)||e}const wp={}.hasOwnProperty;function zx(e,t,n){return typeof t!="string"&&(n=t,t=void 0),Px(n)(Ex(Cx(n).document().write(bx()(e,t,!0))))}function Px(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Os),autolinkProtocol:R,autolinkEmail:R,atxHeading:l(Ms),blockQuote:l(bn),characterEscape:R,characterReference:R,codeFenced:l(oi),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(oi,o),codeText:l(Ip,o),codeTextData:R,data:R,codeFlowValue:R,definition:l(Mp),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Ap),hardBreakEscape:l(As),hardBreakTrailing:l(As),htmlFlow:l(Ds,o),htmlFlowData:R,htmlText:l(Ds,o),htmlTextData:R,image:l(Dp),label:o,link:l(Os),listItem:l(Op),listItemValue:d,listOrdered:l(Rs,h),listUnordered:l(Rs),paragraph:l(Rp),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Ms),strong:l(Fp),thematicBreak:l(Up)},exit:{atxHeading:s(),atxHeadingSequence:E,autolink:s(),autolinkEmail:En,autolinkProtocol:Dt,blockQuote:s(),characterEscapeValue:D,characterReferenceMarkerHexadecimal:it,characterReferenceMarkerNumeric:it,characterReferenceValue:le,characterReference:pt,codeFenced:s(b),codeFencedFence:S,codeFencedFenceInfo:p,codeFencedFenceMeta:k,codeFlowValue:D,codeIndented:s(m),codeText:s(H),codeTextData:D,data:D,definition:s(),definitionDestinationString:C,definitionLabelString:g,definitionTitleString:v,emphasis:s(),hardBreakEscape:s(_),hardBreakTrailing:s(_),htmlFlow:s(M),htmlFlowData:D,htmlText:s(B),htmlTextData:D,image:s(te),label:F,labelText:T,lineEnding:O,link:s(G),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:ge,resourceDestinationString:y,resourceTitleString:q,resource:Z,setextHeading:s(L),setextHeadingLineSequence:N,setextHeadingText:w,strong:s(),thematicBreak:s()}};Sp(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(j){let I={type:"root",children:[]};const $={stack:[I],tokenStack:[],config:t,enter:a,exit:u,buffer:o,resume:f,data:n},K=[];let ee=-1;for(;++ee<j.length;)if(j[ee][1].type==="listOrdered"||j[ee][1].type==="listUnordered")if(j[ee][0]==="enter")K.push(ee);else{const lt=K.pop();ee=i(j,lt,ee)}for(ee=-1;++ee<j.length;){const lt=t[j[ee][0]];wp.call(lt,j[ee][1].type)&&lt[j[ee][1].type].call(Object.assign({sliceSerialize:j[ee][2].sliceSerialize},$),j[ee][1])}if($.tokenStack.length>0){const lt=$.tokenStack[$.tokenStack.length-1];(lt[1]||gc).call($,void 0,lt[0])}for(I.position={start:Rt(j.length>0?j[0][1].start:{line:1,column:1,offset:0}),end:Rt(j.length>0?j[j.length-2][1].end:{line:1,column:1,offset:0})},ee=-1;++ee<t.transforms.length;)I=t.transforms[ee](I)||I;return I}function i(j,I,$){let K=I-1,ee=-1,lt=!1,on,St,ar,sr;for(;++K<=$;){const Ue=j[K];switch(Ue[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Ue[0]==="enter"?ee++:ee--,sr=void 0;break}case"lineEndingBlank":{Ue[0]==="enter"&&(on&&!sr&&!ee&&!ar&&(ar=K),sr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:sr=void 0}if(!ee&&Ue[0]==="enter"&&Ue[1].type==="listItemPrefix"||ee===-1&&Ue[0]==="exit"&&(Ue[1].type==="listUnordered"||Ue[1].type==="listOrdered")){if(on){let jn=K;for(St=void 0;jn--;){const Ct=j[jn];if(Ct[1].type==="lineEnding"||Ct[1].type==="lineEndingBlank"){if(Ct[0]==="exit")continue;St&&(j[St][1].type="lineEndingBlank",lt=!0),Ct[1].type="lineEnding",St=jn}else if(!(Ct[1].type==="linePrefix"||Ct[1].type==="blockQuotePrefix"||Ct[1].type==="blockQuotePrefixWhitespace"||Ct[1].type==="blockQuoteMarker"||Ct[1].type==="listItemIndent"))break}ar&&(!St||ar<St)&&(on._spread=!0),on.end=Object.assign({},St?j[St][1].start:Ue[1].end),j.splice(St||K,0,["exit",on,Ue[2]]),K++,$++}if(Ue[1].type==="listItemPrefix"){const jn={type:"listItem",_spread:!1,start:Object.assign({},Ue[1].start),end:void 0};on=jn,j.splice(K,0,["enter",jn,Ue[2]]),K++,$++,ar=void 0,sr=!0}}}return j[I][1]._spread=lt,$}function l(j,I){return $;function $(K){a.call(this,j(K),K),I&&I.call(this,K)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(j,I,$){this.stack[this.stack.length-1].children.push(j),this.stack.push(j),this.tokenStack.push([I,$||void 0]),j.position={start:Rt(I.start),end:void 0}}function s(j){return I;function I($){j&&j.call(this,$),u.call(this,$)}}function u(j,I){const $=this.stack.pop(),K=this.tokenStack.pop();if(K)K[0].type!==j.type&&(I?I.call(this,j,K[0]):(K[1]||gc).call(this,j,K[0]));else throw new Error("Cannot close `"+j.type+"` ("+Tr({start:j.start,end:j.end})+"): it’s not open");$.position.end=Rt(j.end)}function f(){return _y(this.stack.pop())}function h(){this.data.expectingFirstListItemValue=!0}function d(j){if(this.data.expectingFirstListItemValue){const I=this.stack[this.stack.length-2];I.start=Number.parseInt(this.sliceSerialize(j),10),this.data.expectingFirstListItemValue=void 0}}function p(){const j=this.resume(),I=this.stack[this.stack.length-1];I.lang=j}function k(){const j=this.resume(),I=this.stack[this.stack.length-1];I.meta=j}function S(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function b(){const j=this.resume(),I=this.stack[this.stack.length-1];I.value=j.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function m(){const j=this.resume(),I=this.stack[this.stack.length-1];I.value=j.replace(/(\r?\n|\r)$/g,"")}function g(j){const I=this.resume(),$=this.stack[this.stack.length-1];$.label=I,$.identifier=qn(this.sliceSerialize(j)).toLowerCase()}function v(){const j=this.resume(),I=this.stack[this.stack.length-1];I.title=j}function C(){const j=this.resume(),I=this.stack[this.stack.length-1];I.url=j}function E(j){const I=this.stack[this.stack.length-1];if(!I.depth){const $=this.sliceSerialize(j).length;I.depth=$}}function w(){this.data.setextHeadingSlurpLineEnding=!0}function N(j){const I=this.stack[this.stack.length-1];I.depth=this.sliceSerialize(j).codePointAt(0)===61?1:2}function L(){this.data.setextHeadingSlurpLineEnding=void 0}function R(j){const $=this.stack[this.stack.length-1].children;let K=$[$.length-1];(!K||K.type!=="text")&&(K=Bp(),K.position={start:Rt(j.start),end:void 0},$.push(K)),this.stack.push(K)}function D(j){const I=this.stack.pop();I.value+=this.sliceSerialize(j),I.position.end=Rt(j.end)}function O(j){const I=this.stack[this.stack.length-1];if(this.data.atHardBreak){const $=I.children[I.children.length-1];$.position.end=Rt(j.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(I.type)&&(R.call(this,j),D.call(this,j))}function _(){this.data.atHardBreak=!0}function M(){const j=this.resume(),I=this.stack[this.stack.length-1];I.value=j}function B(){const j=this.resume(),I=this.stack[this.stack.length-1];I.value=j}function H(){const j=this.resume(),I=this.stack[this.stack.length-1];I.value=j}function G(){const j=this.stack[this.stack.length-1];if(this.data.inReference){const I=this.data.referenceType||"shortcut";j.type+="Reference",j.referenceType=I,delete j.url,delete j.title}else delete j.identifier,delete j.label;this.data.referenceType=void 0}function te(){const j=this.stack[this.stack.length-1];if(this.data.inReference){const I=this.data.referenceType||"shortcut";j.type+="Reference",j.referenceType=I,delete j.url,delete j.title}else delete j.identifier,delete j.label;this.data.referenceType=void 0}function T(j){const I=this.sliceSerialize(j),$=this.stack[this.stack.length-2];$.label=Nx(I),$.identifier=qn(I).toLowerCase()}function F(){const j=this.stack[this.stack.length-1],I=this.resume(),$=this.stack[this.stack.length-1];if(this.data.inReference=!0,$.type==="link"){const K=j.children;$.children=K}else $.alt=I}function y(){const j=this.resume(),I=this.stack[this.stack.length-1];I.url=j}function q(){const j=this.resume(),I=this.stack[this.stack.length-1];I.title=j}function Z(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function ge(j){const I=this.resume(),$=this.stack[this.stack.length-1];$.label=I,$.identifier=qn(this.sliceSerialize(j)).toLowerCase(),this.data.referenceType="full"}function it(j){this.data.characterReferenceType=j.type}function le(j){const I=this.sliceSerialize(j),$=this.data.characterReferenceType;let K;$?(K=dp(I,$==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):K=js(I);const ee=this.stack[this.stack.length-1];ee.value+=K}function pt(j){const I=this.stack.pop();I.position.end=Rt(j.end)}function Dt(j){D.call(this,j);const I=this.stack[this.stack.length-1];I.url=this.sliceSerialize(j)}function En(j){D.call(this,j);const I=this.stack[this.stack.length-1];I.url="mailto:"+this.sliceSerialize(j)}function bn(){return{type:"blockquote",children:[]}}function oi(){return{type:"code",lang:null,meta:null,value:""}}function Ip(){return{type:"inlineCode",value:""}}function Mp(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Ap(){return{type:"emphasis",children:[]}}function Ms(){return{type:"heading",depth:0,children:[]}}function As(){return{type:"break"}}function Ds(){return{type:"html",value:""}}function Dp(){return{type:"image",title:null,url:"",alt:null}}function Os(){return{type:"link",title:null,url:"",children:[]}}function Rs(j){return{type:"list",ordered:j.type==="listOrdered",start:null,spread:j._spread,children:[]}}function Op(j){return{type:"listItem",spread:j._spread,checked:null,children:[]}}function Rp(){return{type:"paragraph",children:[]}}function Fp(){return{type:"strong",children:[]}}function Bp(){return{type:"text",value:""}}function Up(){return{type:"thematicBreak"}}}function Rt(e){return{line:e.line,column:e.column,offset:e.offset}}function Sp(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Sp(e,r):Lx(e,r)}}function Lx(e,t){let n;for(n in t)if(wp.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function gc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Tr({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Tr({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Tr({start:t.start,end:t.end})+") is still open")}function Tx(e){const t=this;t.parser=n;function n(r){return zx(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function Ix(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function Mx(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function Ax(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function Dx(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Ox(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Rx(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=or(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const u={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,u),e.applyData(t,u)}function Fx(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Bx(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Cp(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function Ux(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Cp(e,t);const i={src:or(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function Hx(e,t){const n={src:or(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function Vx(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function $x(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Cp(e,t);const i={href:or(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function Wx(e,t){const n={href:or(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function Qx(e,t,n){const r=e.all(t),i=n?Kx(n):Ep(t),l={},o=[];if(typeof t.checked=="boolean"){const f=r[0];let h;f&&f.type==="element"&&f.tagName==="p"?h=f:(h={type:"element",tagName:"p",properties:{},children:[]},r.unshift(h)),h.children.length>0&&h.children.unshift({type:"text",value:" "}),h.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const f=r[a];(i||a!==0||f.type!=="element"||f.tagName!=="p")&&o.push({type:"text",value:`
`}),f.type==="element"&&f.tagName==="p"&&!i?o.push(...f.children):o.push(f)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const u={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,u),e.applyData(t,u)}function Kx(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Ep(n[r])}return t}function Ep(e){const t=e.spread;return t??e.children.length>1}function qx(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function Yx(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Xx(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function Gx(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Jx(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=Ss(t.children[1]),s=ip(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function Zx(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const u=[];for(;++s<a;){const h=t.children[s],d={},p=o?o[s]:void 0;p&&(d.align=p);let k={type:"element",tagName:l,properties:d,children:[]};h&&(k.children=e.all(h),e.patch(h,k),k=e.applyData(h,k)),u.push(k)}const f={type:"element",tagName:"tr",properties:{},children:e.wrap(u,!0)};return e.patch(t,f),e.applyData(t,f)}function e1(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const yc=9,vc=32;function t1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(xc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(xc(t.slice(i),i>0,!1)),l.join("")}function xc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===yc||l===vc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===yc||l===vc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function n1(e,t){const n={type:"text",value:t1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function r1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const i1={blockquote:Ix,break:Mx,code:Ax,delete:Dx,emphasis:Ox,footnoteReference:Rx,heading:Fx,html:Bx,imageReference:Ux,image:Hx,inlineCode:Vx,linkReference:$x,link:Wx,listItem:Qx,list:qx,paragraph:Yx,root:Xx,strong:Gx,table:Jx,tableCell:e1,tableRow:Zx,text:n1,thematicBreak:r1,toml:bi,yaml:bi,definition:bi,footnoteDefinition:bi};function bi(){}const bp=-1,Ml=0,Mr=1,hl=2,zs=3,Ps=4,Ls=5,Ts=6,jp=7,Np=8,kc=typeof self=="object"?self:globalThis,l1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Ml:case bp:return n(o,i);case Mr:{const a=n([],i);for(const s of o)a.push(r(s));return a}case hl:{const a=n({},i);for(const[s,u]of o)a[r(s)]=r(u);return a}case zs:return n(new Date(o),i);case Ps:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Ls:{const a=n(new Map,i);for(const[s,u]of o)a.set(r(s),r(u));return a}case Ts:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case jp:{const{name:a,message:s}=o;return n(new kc[a](s),i)}case Np:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new kc[l](o),i)};return r},wc=e=>l1(new Map,e)(0),_n="",{toString:o1}={},{keys:a1}=Object,vr=e=>{const t=typeof e;if(t!=="object"||!e)return[Ml,t];const n=o1.call(e).slice(8,-1);switch(n){case"Array":return[Mr,_n];case"Object":return[hl,_n];case"Date":return[zs,_n];case"RegExp":return[Ps,_n];case"Map":return[Ls,_n];case"Set":return[Ts,_n];case"DataView":return[Mr,n]}return n.includes("Array")?[Mr,n]:n.includes("Error")?[jp,n]:[hl,n]},ji=([e,t])=>e===Ml&&(t==="function"||t==="symbol"),s1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=vr(o);switch(a){case Ml:{let f=o;switch(s){case"bigint":a=Np,f=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);f=null;break;case"undefined":return i([bp],o)}return i([a,f],o)}case Mr:{if(s){let d=o;return s==="DataView"?d=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(d=new Uint8Array(o)),i([s,[...d]],o)}const f=[],h=i([a,f],o);for(const d of o)f.push(l(d));return h}case hl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const f=[],h=i([a,f],o);for(const d of a1(o))(e||!ji(vr(o[d])))&&f.push([l(d),l(o[d])]);return h}case zs:return i([a,o.toISOString()],o);case Ps:{const{source:f,flags:h}=o;return i([a,{source:f,flags:h}],o)}case Ls:{const f=[],h=i([a,f],o);for(const[d,p]of o)(e||!(ji(vr(d))||ji(vr(p))))&&f.push([l(d),l(p)]);return h}case Ts:{const f=[],h=i([a,f],o);for(const d of o)(e||!ji(vr(d)))&&f.push(l(d));return h}}const{message:u}=o;return i([a,{name:s,message:u}],o)};return l},Sc=(e,{json:t,lossy:n}={})=>{const r=[];return s1(!(t||n),!!t,new Map,r)(e),r},ml=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?wc(Sc(e,t)):structuredClone(e):(e,t)=>wc(Sc(e,t));function u1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function c1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function d1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||u1,r=e.options.footnoteBackLabel||c1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const u=e.footnoteById.get(e.footnoteOrder[s]);if(!u)continue;const f=e.all(u),h=String(u.identifier).toUpperCase(),d=or(h.toLowerCase());let p=0;const k=[],S=e.footnoteCounts.get(h);for(;S!==void 0&&++p<=S;){k.length>0&&k.push({type:"text",value:" "});let g=typeof n=="string"?n:n(s,p);typeof g=="string"&&(g={type:"text",value:g}),k.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+d+(p>1?"-"+p:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,p),className:["data-footnote-backref"]},children:Array.isArray(g)?g:[g]})}const b=f[f.length-1];if(b&&b.type==="element"&&b.tagName==="p"){const g=b.children[b.children.length-1];g&&g.type==="text"?g.value+=" ":b.children.push({type:"text",value:" "}),b.children.push(...k)}else f.push(...k);const m={type:"element",tagName:"li",properties:{id:t+"fn-"+d},children:e.wrap(f,!0)};e.patch(u,m),a.push(m)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...ml(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const _p=function(e){if(e==null)return m1;if(typeof e=="function")return Al(e);if(typeof e=="object")return Array.isArray(e)?f1(e):p1(e);if(typeof e=="string")return h1(e);throw new Error("Expected function, string, or object as test")};function f1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=_p(e[n]);return Al(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function p1(e){const t=e;return Al(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function h1(e){return Al(t);function t(n){return n&&n.type===e}}function Al(e){return t;function t(n,r,i){return!!(g1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function m1(){return!0}function g1(e){return e!==null&&typeof e=="object"&&"type"in e}const zp=[],y1=!0,Cc=!1,v1="skip";function x1(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=_p(i),o=r?-1:1;a(e,void 0,[])();function a(s,u,f){const h=s&&typeof s=="object"?s:{};if(typeof h.type=="string"){const p=typeof h.tagName=="string"?h.tagName:typeof h.name=="string"?h.name:void 0;Object.defineProperty(d,"name",{value:"node ("+(s.type+(p?"<"+p+">":""))+")"})}return d;function d(){let p=zp,k,S,b;if((!t||l(s,u,f[f.length-1]||void 0))&&(p=k1(n(s,f)),p[0]===Cc))return p;if("children"in s&&s.children){const m=s;if(m.children&&p[0]!==v1)for(S=(r?m.children.length:-1)+o,b=f.concat(m);S>-1&&S<m.children.length;){const g=m.children[S];if(k=a(g,S,b)(),k[0]===Cc)return k;S=typeof k[1]=="number"?k[1]:S+o}}return p}}}function k1(e){return Array.isArray(e)?e:typeof e=="number"?[y1,e]:e==null?zp:[e]}function Pp(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),x1(e,l,a,i);function a(s,u){const f=u[u.length-1],h=f?f.children.indexOf(s):void 0;return o(s,h,f)}}const wa={}.hasOwnProperty,w1={};function S1(e,t){const n=t||w1,r=new Map,i=new Map,l=new Map,o={...i1,...n.handlers},a={all:u,applyData:E1,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:C1,wrap:j1};return Pp(e,function(f){if(f.type==="definition"||f.type==="footnoteDefinition"){const h=f.type==="definition"?r:i,d=String(f.identifier).toUpperCase();h.has(d)||h.set(d,f)}}),a;function s(f,h){const d=f.type,p=a.handlers[d];if(wa.call(a.handlers,d)&&p)return p(a,f,h);if(a.options.passThrough&&a.options.passThrough.includes(d)){if("children"in f){const{children:S,...b}=f,m=ml(b);return m.children=a.all(f),m}return ml(f)}return(a.options.unknownHandler||b1)(a,f,h)}function u(f){const h=[];if("children"in f){const d=f.children;let p=-1;for(;++p<d.length;){const k=a.one(d[p],f);if(k){if(p&&d[p-1].type==="break"&&(!Array.isArray(k)&&k.type==="text"&&(k.value=Ec(k.value)),!Array.isArray(k)&&k.type==="element")){const S=k.children[0];S&&S.type==="text"&&(S.value=Ec(S.value))}Array.isArray(k)?h.push(...k):h.push(k)}}}return h}}function C1(e,t){e.position&&(t.position=oy(e))}function E1(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,ml(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function b1(e,t){const n=t.data||{},r="value"in t&&!(wa.call(n,"hProperties")||wa.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function j1(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Ec(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function bc(e,t){const n=S1(e,t),r=n.one(e,void 0),i=d1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function N1(e,t){return e&&"run"in e?async function(n,r){const i=bc(n,{file:r,...t});await e.run(i,r)}:function(n,r){return bc(n,{file:r,...e||t})}}function jc(e){if(e)throw e}var Hi=Object.prototype.hasOwnProperty,Lp=Object.prototype.toString,Nc=Object.defineProperty,_c=Object.getOwnPropertyDescriptor,zc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Lp.call(t)==="[object Array]"},Pc=function(t){if(!t||Lp.call(t)!=="[object Object]")return!1;var n=Hi.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Hi.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Hi.call(t,i)},Lc=function(t,n){Nc&&n.name==="__proto__"?Nc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Tc=function(t,n){if(n==="__proto__")if(Hi.call(t,n)){if(_c)return _c(t,n).value}else return;return t[n]},_1=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,u=arguments.length,f=!1;for(typeof a=="boolean"&&(f=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<u;++s)if(t=arguments[s],t!=null)for(n in t)r=Tc(a,n),i=Tc(t,n),a!==i&&(f&&i&&(Pc(i)||(l=zc(i)))?(l?(l=!1,o=r&&zc(r)?r:[]):o=r&&Pc(r)?r:{},Lc(a,{name:n,newValue:e(f,o,i)})):typeof i<"u"&&Lc(a,{name:n,newValue:i}));return a};const po=Ea(_1);function Sa(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function z1(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...u){const f=e[++l];let h=-1;if(s){o(s);return}for(;++h<i.length;)(u[h]===null||u[h]===void 0)&&(u[h]=i[h]);i=u,f?P1(f,a)(...u):o(null,...u)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function P1(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(u){const f=u;if(a&&n)throw f;return i(f)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const gt={basename:L1,dirname:T1,extname:I1,join:M1,sep:"/"};function L1(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');li(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function T1(e){if(li(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function I1(e){li(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function M1(...e){let t=-1,n;for(;++t<e.length;)li(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":A1(n)}function A1(e){li(e);const t=e.codePointAt(0)===47;let n=D1(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function D1(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function li(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const O1={cwd:R1};function R1(){return"/"}function Ca(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function F1(e){if(typeof e=="string")e=new URL(e);else if(!Ca(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return B1(e)}function B1(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const ho=["history","path","basename","stem","extname","dirname"];class Tp{constructor(t){let n;t?Ca(t)?n={path:t}:typeof t=="string"||U1(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":O1.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<ho.length;){const l=ho[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)ho.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?gt.basename(this.path):void 0}set basename(t){go(t,"basename"),mo(t,"basename"),this.path=gt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?gt.dirname(this.path):void 0}set dirname(t){Ic(this.basename,"dirname"),this.path=gt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?gt.extname(this.path):void 0}set extname(t){if(mo(t,"extname"),Ic(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=gt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){Ca(t)&&(t=F1(t)),go(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?gt.basename(this.path,this.extname):void 0}set stem(t){go(t,"stem"),mo(t,"stem"),this.path=gt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new _e(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function mo(e,t){if(e&&e.includes(gt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+gt.sep+"`")}function go(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Ic(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function U1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const H1=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},V1={}.hasOwnProperty;class Is extends H1{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=z1()}copy(){const t=new Is;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(po(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(xo("data",this.frozen),this.namespace[t]=n,this):V1.call(this.namespace,t)&&this.namespace[t]||void 0:t?(xo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Ni(t),r=this.parser||this.Parser;return yo("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),yo("process",this.parser||this.Parser),vo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Ni(t),s=r.parse(a);r.run(s,a,function(f,h,d){if(f||!h||!d)return u(f);const p=h,k=r.stringify(p,d);Q1(k)?d.value=k:d.result=k,u(f,d)});function u(f,h){f||!h?o(f):l?l(h):n(void 0,h)}}}processSync(t){let n=!1,r;return this.freeze(),yo("processSync",this.parser||this.Parser),vo("processSync",this.compiler||this.Compiler),this.process(t,i),Ac("processSync","process",n),r;function i(l,o){n=!0,jc(l),r=o}}run(t,n,r){Mc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Ni(n);i.run(t,s,u);function u(f,h,d){const p=h||t;f?a(f):o?o(p):r(void 0,p,d)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Ac("runSync","run",r),i;function l(o,a){jc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Ni(n),i=this.compiler||this.Compiler;return vo("stringify",i),Mc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(xo("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(u){if(typeof u=="function")s(u,[]);else if(typeof u=="object")if(Array.isArray(u)){const[f,...h]=u;s(f,h)}else o(u);else throw new TypeError("Expected usable value, not `"+u+"`")}function o(u){if(!("plugins"in u)&&!("settings"in u))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(u.plugins),u.settings&&(i.settings=po(!0,i.settings,u.settings))}function a(u){let f=-1;if(u!=null)if(Array.isArray(u))for(;++f<u.length;){const h=u[f];l(h)}else throw new TypeError("Expected a list of plugins, not `"+u+"`")}function s(u,f){let h=-1,d=-1;for(;++h<r.length;)if(r[h][0]===u){d=h;break}if(d===-1)r.push([u,...f]);else if(f.length>0){let[p,...k]=f;const S=r[d][1];Sa(S)&&Sa(p)&&(p=po(!0,S,p)),r[d]=[u,p,...k]}}}}const $1=new Is().freeze();function yo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function vo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function xo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Mc(e){if(!Sa(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Ac(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Ni(e){return W1(e)?e:new Tp(e)}function W1(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function Q1(e){return typeof e=="string"||K1(e)}function K1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const q1="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Dc=[],Oc={allowDangerousHtml:!0},Y1=/^(https?|ircs?|mailto|xmpp)$/i,X1=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function G1(e){const t=J1(e),n=Z1(e);return e0(t.runSync(t.parse(n),n),e)}function J1(e){const t=e.rehypePlugins||Dc,n=e.remarkPlugins||Dc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Oc}:Oc;return $1().use(Tx).use(n).use(N1,r).use(t)}function Z1(e){const t=e.children||"",n=new Tp;return typeof t=="string"&&(n.value=t),n}function e0(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||t0;for(const f of X1)Object.hasOwn(t,f.from)&&(""+f.from+(f.to?"use `"+f.to+"` instead":"remove it")+q1+f.id,void 0);return Pp(e,u),dy(e,{Fragment:c.Fragment,components:i,ignoreInvalidStyle:!0,jsx:c.jsx,jsxs:c.jsxs,passKeys:!0,passNode:!0});function u(f,h,d){if(f.type==="raw"&&d&&typeof h=="number")return o?d.children.splice(h,1):d.children[h]={type:"text",value:f.value},h;if(f.type==="element"){let p;for(p in uo)if(Object.hasOwn(uo,p)&&Object.hasOwn(f.properties,p)){const k=f.properties[p],S=uo[p];(S===null||S.includes(f.tagName))&&(f.properties[p]=s(String(k||""),p,f))}}if(f.type==="element"){let p=n?!n.includes(f.tagName):l?l.includes(f.tagName):!1;if(!p&&r&&typeof h=="number"&&(p=!r(f,h,d)),p&&d&&typeof h=="number")return a&&f.children?d.children.splice(h,1,...f.children):d.children.splice(h,1),h}}}function t0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||Y1.test(e.slice(0,t))?e:""}const Nt={send:c.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),c.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),c.jsx("polyline",{points:"14 2 14 8 20 8"}),c.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),c.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),c.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),c.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),c.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),c.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),c.jsx("circle",{cx:"12",cy:"5",r:"2"}),c.jsx("path",{d:"M12 7v4"})]})},n0=e=>{switch(e){case"directive":return Nt.directive;case"question":return Nt.question;case"status":return Nt.status;case"result":return Nt.result;case"approval_request":return Nt.lock;default:return Nt.directive}},r0=({threadId:e,messages:t,onSendMessage:n})=>{const r=U.useRef(null),[i,l]=cn.useState(""),[o,a]=cn.useState("directive"),[s,u]=cn.useState(""),[f,h]=cn.useState(!1);U.useEffect(()=>{var b;(b=r.current)==null||b.scrollIntoView({behavior:"smooth"})},[t]);const d=()=>{i.trim()&&(n(i,o,s||void 0),l(""))},p=b=>{b.key==="Enter"&&!b.shiftKey&&(b.preventDefault(),d())},k=b=>new Date(b).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),S=b=>b.length>12?`${b.slice(0,8)}...`:b;return c.jsxs("div",{className:"conversation-view",children:[c.jsxs("div",{className:"conversation-header",children:[c.jsxs("div",{className:"header-info",children:[c.jsx("span",{className:"thread-label",children:"Thread"}),c.jsx("span",{className:"thread-id",title:e,children:S(e)})]}),c.jsx("div",{className:"header-stats",children:c.jsxs("span",{className:"message-count",children:[t.length," messages"]})})]}),c.jsxs("div",{className:"messages-container",children:[t.length===0?c.jsxs("div",{className:"empty-messages",children:[c.jsx("div",{className:"empty-icon",children:c.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:c.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),c.jsx("p",{children:"No messages yet"}),c.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((b,m)=>{const g=b.from_type==="human",v=m===0||t[m-1].from_type!==b.from_type;return c.jsxs("div",{className:`message ${g?"human":"agent"}`,children:[c.jsx("div",{className:`message-avatar ${v?"visible":""}`,children:v&&(g?Nt.user:Nt.bot)}),c.jsxs("div",{className:"message-body",children:[v&&c.jsxs("div",{className:"message-meta",children:[c.jsx("span",{className:"sender-name",children:b.from_id}),c.jsxs("span",{className:"kind-badge",children:[n0(b.kind)," ",b.kind]}),c.jsx("span",{className:"message-time",children:k(b.created_at)})]}),c.jsx("div",{className:"message-content",children:b.kind==="result"||!g?c.jsx(G1,{components:{a:({href:C,children:E})=>c.jsx("a",{href:C,target:"_blank",rel:"noopener noreferrer",children:E}),code:({className:C,children:E,...w})=>!C?c.jsx("code",{className:"inline-code",...w,children:E}):c.jsx("code",{className:C,...w,children:E})},children:b.content}):b.content}),c.jsxs("div",{className:"message-footer",children:[c.jsxs("span",{className:"message-seq",children:["#",b.message_seq]}),b.delivery_state!=="acked"&&c.jsx("span",{className:`delivery-status ${b.delivery_state}`,children:b.delivery_state==="pending"?"sending...":"delivered"})]})]})]},b.id)}),c.jsx("div",{ref:r})]}),c.jsxs("div",{className:"input-area",children:[c.jsxs("div",{className:"workspace-row",children:[c.jsxs("button",{onClick:()=>h(!f),className:`workspace-toggle ${s?"has-workspace":""}`,title:s||"Set working directory",children:[c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),c.jsx("span",{children:s?"Workspace set":"Set workspace"})]}),s&&c.jsx("span",{className:"workspace-path",title:s,children:s.length>40?`...${s.slice(-37)}`:s})]}),f&&c.jsxs("div",{className:"workspace-input-row",children:[c.jsx("input",{type:"text",value:s,onChange:b=>u(b.target.value),placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),s&&c.jsx("button",{onClick:()=>{u(""),h(!1)},className:"workspace-clear",children:"Clear"})]}),c.jsxs("div",{className:"input-wrapper",children:[c.jsxs("select",{value:o,onChange:b=>a(b.target.value),className:"kind-selector",children:[c.jsx("option",{value:"directive",children:"Directive"}),c.jsx("option",{value:"question",children:"Question"})]}),c.jsx("textarea",{value:i,onChange:b=>l(b.target.value),onKeyPress:p,placeholder:"Type a message...",rows:1}),c.jsx("button",{onClick:d,className:"send-btn",disabled:!i.trim(),children:Nt.send})]}),c.jsxs("div",{className:"input-hint",children:["Press ",c.jsx("kbd",{children:"Enter"})," to send, ",c.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),c.jsx("style",{children:`
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
          gap: var(--space-2);
        }

        .thread-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .thread-id {
          font-size: var(--text-sm);
          font-family: var(--font-mono);
          color: var(--text-secondary);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
        }

        .header-stats {
          display: flex;
          gap: var(--space-4);
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

        /* Workspace selector */
        .workspace-row {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          margin-bottom: var(--space-2);
        }

        .workspace-toggle {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          font-size: var(--text-xs);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .workspace-toggle:hover {
          color: var(--text-secondary);
          border-color: var(--border-default);
        }

        .workspace-toggle.has-workspace {
          color: var(--color-primary);
          border-color: var(--color-primary);
          background: rgba(37, 194, 160, 0.1);
        }

        .workspace-path {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
          max-width: 300px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
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
      `})]})},i0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,reconnectInterval:l=5e3})=>{const o=U.useRef(null),[a,s]=U.useState(!1),[u,f]=U.useState(null),h=U.useRef(null),d=U.useRef(new Map),p=U.useCallback(()=>{try{const C=`${e}?instance_id=${t}`;o.current=new WebSocket(C),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),f(null),d.current.forEach((E,w)=>{b(w,E)})},o.current.onmessage=E=>{try{const w=JSON.parse(E.data);k(w)}catch(w){console.error("Failed to parse WebSocket message:",w)}},o.current.onerror=E=>{console.error("WebSocket error:",E),f("Connection error")},o.current.onclose=()=>{console.log("WebSocket disconnected"),s(!1),h.current=setTimeout(()=>{console.log("Attempting to reconnect..."),p()},l)}}catch(C){console.error("Failed to connect to WebSocket:",C),f("Failed to connect")}},[e,t,l]),k=U.useCallback(C=>{switch(C.type){case"message":n&&C.data&&n(C.data);break;case"batch":if(r&&C.data){const E=C.data;r(E),n&&E.messages.forEach(w=>n(w))}break;case"error":i&&C.data&&i(C.data),console.error("WebSocket error event:",C.data);break;case"pong":break;default:console.log("Unknown event type:",C.type)}},[n,r,i]),S=U.useCallback(C=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(C)):console.warn("WebSocket not connected, cannot send event")},[]),b=U.useCallback((C,E=0)=>{d.current.set(C,E);const w={type:"subscribe",timestamp:Date.now(),data:{thread_id:C,from_seq:E}};S(w)},[S]),m=U.useCallback((C,E)=>{const w=d.current.get(C)||0;E>w&&d.current.set(C,E);const N={type:"ack",timestamp:Date.now(),data:{thread_id:C,ack_seq:E}};S(N)},[S]),g=U.useCallback(()=>{const C={type:"ping",timestamp:Date.now()};S(C)},[S]),v=U.useCallback(C=>{d.current.delete(C)},[]);return U.useEffect(()=>(p(),()=>{h.current&&clearTimeout(h.current),o.current&&o.current.close()}),[p]),U.useEffect(()=>{if(!a)return;const C=setInterval(()=>{g()},3e4);return()=>clearInterval(C)},[a,g]),{isConnected:a,connectionError:u,subscribe:b,unsubscribe:v,acknowledge:m,ping:g}},l0=({connected:e})=>c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?c.jsxs(c.Fragment,{children:[c.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),c.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):c.jsxs(c.Fragment,{children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),c.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),o0=({websocketUrl:e,instanceId:t})=>{const[n,r]=U.useState([]),[i,l]=U.useState(null),[o,a]=U.useState(new Map),[s,u]=U.useState(new Map),[f,h]=U.useState([]),[d,p]=U.useState(!1),[k,S]=U.useState(""),{isConnected:b,subscribe:m,acknowledge:g}=i0({url:e,instanceId:t,onMessage:v,onBatch:C});function v(_){const M={id:_.id,thread_id:_.thread_id,message_seq:_.message_seq,created_at:_.created_at,from_type:_.from_type,from_id:_.from_id,to_type:_.to_type,to_id:_.to_id,kind:_.kind,subject:_.subject,content:_.content,metadata_json:_.metadata_json,delivery_state:"visible",business_state:"open"};a(B=>{const H=B.get(M.thread_id)||[];return H.find(G=>G.id===M.id)?B:new Map(B).set(M.thread_id,[...H,M].sort((G,te)=>G.message_seq-te.message_seq))}),M.thread_id!==i&&u(B=>{const H=B.get(M.thread_id)||0;return new Map(B).set(M.thread_id,H+1)}),g(M.thread_id,M.message_seq)}function C(_){_.messages.forEach(M=>{v(M)})}const E=U.useCallback(_=>{if(l(_),u(M=>{const B=new Map(M);return B.delete(_),B}),b){const M=o.get(_)||[],B=M.length>0?Math.max(...M.map(H=>H.message_seq)):0;m(_,B)}},[b,m,o]),w=U.useCallback(async(_,M,B)=>{if(!i)return;const H=B?JSON.stringify({workspace:B}):void 0;try{const G=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:i,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:M,content:_,metadata_json:H})});if(!G.ok){console.error("Failed to send message:",await G.text());return}const te=await G.json();a(T=>{const F=T.get(i)||[];return F.find(y=>y.id===te.id)?T:new Map(T).set(i,[...F,te])})}catch(G){console.error("Error sending message:",G)}},[i,t]);U.useEffect(()=>{(async()=>{try{const M=await fetch("/api/threads");if(!M.ok){console.error("Failed to fetch threads:",await M.text());return}const B=await M.json();r(B)}catch(M){console.error("Error fetching threads:",M)}})()},[]);const N=U.useCallback(async _=>{try{const M=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:_,created_by_type:"human",created_by_id:"user"})});if(!M.ok){console.error("Failed to create thread:",await M.text());return}const B=await M.json();r(H=>[B,...H]),l(B.id)}catch(M){console.error("Error creating thread:",M)}},[]),L=U.useCallback(async()=>{try{const _=await fetch("/api/agents");if(!_.ok){console.error("Failed to fetch agents:",await _.text());return}const M=await _.json();h(M.running||[])}catch(_){console.error("Error fetching agents:",_)}},[]);U.useEffect(()=>{L();const _=setInterval(L,5e3);return()=>clearInterval(_)},[L]);const R=U.useCallback(async()=>{if(k.trim())try{const _=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:k.trim()})});if(!_.ok){const B=await _.text();console.error("Failed to launch agent:",B),alert(`Failed to launch agent: ${B}`);return}const M=await _.json();h(B=>[...B,M]),S(""),p(!1)}catch(_){console.error("Error launching agent:",_)}},[k]),D=U.useCallback(async _=>{try{const M=await fetch(`/api/agents/${_}`,{method:"DELETE"});if(!M.ok){console.error("Failed to stop agent:",await M.text());return}h(B=>B.filter(H=>H.instance_id!==_))}catch(M){console.error("Error stopping agent:",M)}},[]),O=i?o.get(i)||[]:[];return c.jsxs("div",{className:"message-center",children:[c.jsxs("div",{className:"status-bar",children:[c.jsxs("div",{className:`status-indicator ${b?"connected":"disconnected"}`,children:[c.jsx(l0,{connected:b}),c.jsx("span",{children:b?"Connected":"Disconnected"})]}),c.jsxs("div",{className:"status-meta",children:[c.jsxs("span",{className:"thread-count",children:[n.length," threads"]}),c.jsxs("span",{className:"agent-count",children:[f.length," agents"]}),c.jsx("button",{className:"launch-agent-btn",onClick:()=>p(!0),children:"+ Agent"})]})]}),f.length>0&&c.jsx("div",{className:"agents-bar",children:f.map(_=>c.jsxs("div",{className:"agent-chip",children:[c.jsx("span",{className:"agent-pulse"}),c.jsx("span",{className:"agent-name",children:_.instance_id}),c.jsxs("span",{className:"agent-pid",children:["PID ",_.pid]}),c.jsx("button",{className:"agent-stop-btn",onClick:()=>D(_.instance_id),title:"Stop agent",children:"×"})]},_.instance_id))}),d&&c.jsx("div",{className:"modal-overlay",onClick:()=>p(!1),children:c.jsxs("div",{className:"modal-content",onClick:_=>_.stopPropagation(),children:[c.jsx("h3",{children:"Launch New Agent"}),c.jsx("input",{type:"text",value:k,onChange:_=>S(_.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:_=>{_.key==="Enter"&&R(),_.key==="Escape"&&p(!1)}}),c.jsxs("div",{className:"modal-actions",children:[c.jsx("button",{className:"cancel-btn",onClick:()=>p(!1),children:"Cancel"}),c.jsx("button",{className:"launch-btn",onClick:R,children:"Launch"})]})]})}),c.jsxs("div",{className:"center-layout",children:[c.jsx("aside",{className:"threads-panel",children:c.jsx(hg,{threads:n,selectedThreadId:i,onSelectThread:E,onCreateThread:N,unreadCounts:s})}),c.jsx("main",{className:"conversation-panel",children:i?c.jsx(r0,{threadId:i,messages:O,onSendMessage:w}):c.jsxs("div",{className:"empty-state",children:[c.jsx("div",{className:"empty-icon",children:c.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:c.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),c.jsx("h3",{children:"Select a conversation"}),c.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),c.jsx("style",{children:`
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
      `})]})},Et={check:c.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),c.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:c.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:c.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),c.jsx("circle",{cx:"12",cy:"5",r:"2"}),c.jsx("path",{d:"M12 7v4"})]}),dollar:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),c.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("polyline",{points:"12 6 12 12 16 14"})]}),sparkles:c.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),c.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),c.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},a0=({approvals:e,onApprove:t,onReject:n})=>{const[r,i]=U.useState(null),[l,o]=U.useState(new Map),a=p=>{try{return JSON.parse(p)}catch{return null}},s=p=>new Date(p).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),u=p=>{const k=l.get(p)||"";t(p,k),o(new Map(l.set(p,"")))},f=p=>{const k=l.get(p)||"";if(!k.trim()){alert("Please provide a reason for rejection");return}n(p,k),o(new Map(l.set(p,"")))},h=(p,k)=>{o(new Map(l.set(p,k)))},d=e.filter(p=>p.status==="pending");return c.jsxs("div",{className:"approval-queue",children:[c.jsx("div",{className:"queue-header",children:c.jsxs("div",{className:"header-title",children:[c.jsx("h2",{children:"Approval Queue"}),c.jsxs("span",{className:"pending-count",children:[d.length," pending"]})]})}),c.jsx("div",{className:"approvals-container",children:d.length===0?c.jsxs("div",{className:"empty-state",children:[c.jsx("div",{className:"empty-icon",children:Et.sparkles}),c.jsx("h3",{children:"All caught up!"}),c.jsx("p",{children:"No pending approvals to review"})]}):c.jsx("div",{className:"approvals-list",children:d.map(p=>{const k=a(p.effect_delta_json),S=r===p.id;return c.jsxs("div",{className:`approval-card impact-${p.impact}`,children:[c.jsxs("div",{className:"card-header",onClick:()=>i(S?null:p.id),children:[c.jsxs("div",{className:"header-left",children:[c.jsx("div",{className:`impact-indicator ${p.impact}`}),c.jsxs("div",{className:"proposal-info",children:[c.jsx("span",{className:"proposal-text",children:p.proposal}),c.jsxs("div",{className:"proposal-meta",children:[c.jsxs("span",{className:"meta-item",children:[Et.bot,p.instance_id]}),c.jsxs("span",{className:"meta-item",children:[Et.clock,s(p.created_at)]})]})]})]}),c.jsxs("div",{className:"header-right",children:[c.jsxs("span",{className:"cost-badge",children:[Et.dollar,"$",p.estimated_cost.toFixed(2)]}),c.jsx("span",{className:`impact-badge ${p.impact}`,children:p.impact}),c.jsx("button",{className:"expand-btn",children:S?Et.chevronUp:Et.chevronDown})]})]}),S&&c.jsxs("div",{className:"card-details",children:[k&&c.jsxs("div",{className:"detail-section",children:[c.jsx("h4",{children:"Effect Details"}),c.jsxs("div",{className:"detail-grid",children:[c.jsxs("div",{className:"detail-item",children:[c.jsx("span",{className:"detail-label",children:"Capability"}),c.jsx("span",{className:"detail-value code",children:k.cap_type})]}),c.jsxs("div",{className:"detail-item",children:[c.jsx("span",{className:"detail-label",children:"Budget Delta"}),c.jsxs("span",{className:"detail-value",children:["$",k.budget_delta.toFixed(2)]})]}),k.paths.length>0&&c.jsxs("div",{className:"detail-item full-width",children:[c.jsx("span",{className:"detail-label",children:"Paths"}),c.jsx("div",{className:"paths-list",children:k.paths.map((b,m)=>c.jsxs("span",{className:"path-tag",children:[Et.folder,b]},m))})]})]})]}),c.jsxs("div",{className:"detail-section",children:[c.jsx("h4",{children:"Request Info"}),c.jsxs("div",{className:"detail-grid",children:[c.jsxs("div",{className:"detail-item",children:[c.jsx("span",{className:"detail-label",children:"Thread"}),c.jsx("span",{className:"detail-value code",children:p.thread_id})]}),c.jsxs("div",{className:"detail-item",children:[c.jsx("span",{className:"detail-label",children:"Impact Level"}),c.jsx("span",{className:`detail-value impact-text ${p.impact}`,children:p.impact.toUpperCase()})]})]})]}),c.jsxs("div",{className:"review-section",children:[c.jsx("h4",{children:"Review Notes"}),c.jsx("textarea",{value:l.get(p.id)||"",onChange:b=>h(p.id,b.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),c.jsxs("div",{className:"action-buttons",children:[c.jsxs("button",{className:"reject-btn",onClick:()=>f(p.id),children:[Et.x,"Reject"]}),c.jsxs("button",{className:"approve-btn",onClick:()=>u(p.id),children:[Et.check,"Approve"]})]})]})]})]},p.id)})})}),c.jsx("style",{children:`
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
        }
      `})]})},Xe={cpu:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"4",y:"4",width:"16",height:"16",rx:"2"}),c.jsx("rect",{x:"9",y:"9",width:"6",height:"6"}),c.jsx("line",{x1:"9",y1:"1",x2:"9",y2:"4"}),c.jsx("line",{x1:"15",y1:"1",x2:"15",y2:"4"}),c.jsx("line",{x1:"9",y1:"20",x2:"9",y2:"23"}),c.jsx("line",{x1:"15",y1:"20",x2:"15",y2:"23"}),c.jsx("line",{x1:"20",y1:"9",x2:"23",y2:"9"}),c.jsx("line",{x1:"20",y1:"14",x2:"23",y2:"14"}),c.jsx("line",{x1:"1",y1:"9",x2:"4",y2:"9"}),c.jsx("line",{x1:"1",y1:"14",x2:"4",y2:"14"})]}),memory:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"2",y:"6",width:"20",height:"12",rx:"2"}),c.jsx("line",{x1:"6",y1:"10",x2:"6",y2:"14"}),c.jsx("line",{x1:"10",y1:"10",x2:"10",y2:"14"}),c.jsx("line",{x1:"14",y1:"10",x2:"14",y2:"14"}),c.jsx("line",{x1:"18",y1:"10",x2:"18",y2:"14"})]}),clock:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("polyline",{points:"12 6 12 12 16 14"})]}),dollar:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),c.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),activity:c.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"22 12 18 12 15 21 9 3 6 12 2 12"})}),stop:c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("rect",{x:"3",y:"3",width:"18",height:"18",rx:"2"})}),warning:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"}),c.jsx("line",{x1:"12",y1:"9",x2:"12",y2:"13"}),c.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),check:c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"20 6 9 17 4 12"})})},s0=()=>{const[e,t]=U.useState(null),[n,r]=U.useState(null),[i,l]=U.useState(null),o=U.useCallback(async()=>{try{const d=await fetch("/api/monitor");if(!d.ok)throw new Error(`Failed to fetch: ${d.statusText}`);const p=await d.json();t(p),l(new Date),r(null)}catch(d){r(d instanceof Error?d.message:"Unknown error")}},[]);U.useEffect(()=>{o();const d=setInterval(o,2e3);return()=>clearInterval(d)},[o]);const a=async d=>{try{await fetch(`/api/agents/${d}`,{method:"DELETE"}),o()}catch(p){console.error("Failed to stop process:",p)}},s=d=>{if(d<0)return"Unknown";if(d<60)return`${d}s`;if(d<3600){const S=Math.floor(d/60),b=d%60;return`${S}m ${b}s`}const p=Math.floor(d/3600),k=Math.floor(d%3600/60);return`${p}h ${k}m`},u=d=>d===0?"$0.00":d<.01?`$${d.toFixed(4)}`:`$${d.toFixed(2)}`,f=d=>{switch(d){case"running":return"var(--color-success)";case"completed":return"var(--color-primary)";case"failed":return"var(--color-danger)";case"orphan":return"var(--color-warning)";default:return"var(--text-tertiary)"}},h=d=>d.cpu_percent>80||d.duration_sec>300;return c.jsxs("div",{className:"monitor",children:[c.jsxs("div",{className:"monitor-summary",children:[c.jsxs("div",{className:"summary-item",children:[c.jsx("span",{className:"summary-icon",children:Xe.activity}),c.jsx("span",{className:"summary-value",children:(e==null?void 0:e.summary.total_processes)||0}),c.jsx("span",{className:"summary-label",children:"Running"})]}),c.jsxs("div",{className:"summary-item",children:[c.jsx("span",{className:"summary-icon",children:Xe.cpu}),c.jsxs("span",{className:"summary-value",children:[(e==null?void 0:e.summary.total_cpu_percent.toFixed(1))||"0.0","%"]}),c.jsx("span",{className:"summary-label",children:"CPU"})]}),c.jsxs("div",{className:"summary-item",children:[c.jsx("span",{className:"summary-icon",children:Xe.memory}),c.jsxs("span",{className:"summary-value",children:[(e==null?void 0:e.summary.total_memory_mb.toFixed(0))||"0"," MB"]}),c.jsx("span",{className:"summary-label",children:"Memory"})]}),c.jsxs("div",{className:"summary-item",children:[c.jsx("span",{className:"summary-icon",children:Xe.dollar}),c.jsx("span",{className:"summary-value",children:u((e==null?void 0:e.summary.total_cost)||0)}),c.jsx("span",{className:"summary-label",children:"Cost"})]}),((e==null?void 0:e.summary.warning_count)||0)>0&&c.jsxs("div",{className:"summary-item warning",children:[c.jsx("span",{className:"summary-icon",children:Xe.warning}),c.jsx("span",{className:"summary-value",children:e==null?void 0:e.summary.warning_count}),c.jsx("span",{className:"summary-label",children:"Warnings"})]}),c.jsx("div",{className:"summary-spacer"}),c.jsxs("div",{className:"summary-update",children:["Last update: ",i?i.toLocaleTimeString():"Never"]})]}),c.jsxs("div",{className:"process-grid",children:[n&&c.jsxs("div",{className:"error-card",children:[c.jsx("span",{className:"error-icon",children:Xe.warning}),c.jsx("span",{children:n})]}),(!(e!=null&&e.processes)||e.processes.length===0)&&!n&&c.jsxs("div",{className:"empty-state",children:[c.jsx("span",{className:"empty-icon",children:Xe.activity}),c.jsx("h3",{children:"No Active Processes"}),c.jsx("p",{children:"Spawn an agent from the Messages tab to see it here."})]}),e==null?void 0:e.processes.map(d=>c.jsxs("div",{className:`process-card ${h(d)?"warning":""}`,children:[c.jsxs("div",{className:"process-header",children:[c.jsxs("div",{className:"process-status",children:[c.jsx("span",{className:"status-dot",style:{background:f(d.status)}}),c.jsx("span",{className:"process-name",children:d.instance_id})]}),d.status==="running"&&c.jsx("button",{className:"stop-btn",onClick:()=>a(d.instance_id),title:"Stop process",children:Xe.stop}),d.status==="completed"&&c.jsxs("span",{className:"status-badge completed",children:[Xe.check," Done"]})]}),c.jsxs("div",{className:"process-metrics",children:[c.jsxs("div",{className:"metric",children:[c.jsx("span",{className:"metric-icon",children:Xe.cpu}),c.jsxs("span",{className:`metric-value ${d.cpu_percent>80?"high":""}`,children:[d.cpu_percent.toFixed(1),"%"]}),c.jsx("span",{className:"metric-label",children:"CPU"})]}),c.jsxs("div",{className:"metric",children:[c.jsx("span",{className:"metric-icon",children:Xe.memory}),c.jsxs("span",{className:"metric-value",children:[d.memory_mb.toFixed(0)," MB"]}),c.jsx("span",{className:"metric-label",children:"Memory"})]}),c.jsxs("div",{className:"metric",children:[c.jsx("span",{className:"metric-icon",children:Xe.clock}),c.jsx("span",{className:`metric-value ${d.duration_sec>300?"high":""}`,children:s(d.duration_sec)}),c.jsx("span",{className:"metric-label",children:"Duration"})]})]}),c.jsxs("div",{className:"process-footer",children:[c.jsxs("span",{className:"process-pid",children:["PID: ",d.pid]}),d.turns&&c.jsxs("span",{className:"process-turns",children:[d.turns," turns"]}),d.cost!==void 0&&d.cost>0&&c.jsx("span",{className:"process-cost",children:u(d.cost)})]})]},d.instance_id))]}),c.jsx("style",{children:`
        .monitor {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg-base);
        }

        /* Summary Bar */
        .monitor-summary {
          display: flex;
          align-items: center;
          gap: var(--space-6);
          padding: var(--space-4) var(--space-6);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
        }

        .summary-item {
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }

        .summary-item.warning {
          color: var(--color-warning);
        }

        .summary-icon {
          color: var(--text-tertiary);
          display: flex;
        }

        .summary-item.warning .summary-icon {
          color: var(--color-warning);
        }

        .summary-value {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          font-family: var(--font-mono);
          color: var(--text-primary);
        }

        .summary-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .summary-spacer {
          flex: 1;
        }

        .summary-update {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Process Grid */
        .process-grid {
          flex: 1;
          overflow-y: auto;
          padding: var(--space-6);
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
          gap: var(--space-4);
          align-content: start;
        }

        .error-card {
          grid-column: 1 / -1;
          display: flex;
          align-items: center;
          gap: var(--space-3);
          padding: var(--space-4);
          background: rgba(248, 81, 73, 0.1);
          border: 1px solid var(--color-danger);
          border-radius: var(--radius-md);
          color: var(--color-danger);
        }

        .error-icon {
          display: flex;
        }

        .empty-state {
          grid-column: 1 / -1;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          padding: var(--space-12);
          text-align: center;
          color: var(--text-tertiary);
        }

        .empty-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 64px;
          height: 64px;
          background: var(--bg-elevated);
          border-radius: var(--radius-lg);
          margin-bottom: var(--space-4);
        }

        .empty-icon svg {
          width: 32px;
          height: 32px;
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

        /* Process Card */
        .process-card {
          background: var(--bg-surface);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-lg);
          padding: var(--space-4);
          transition: all var(--transition-fast);
        }

        .process-card:hover {
          border-color: var(--border-default);
          box-shadow: var(--shadow-md);
        }

        .process-card.warning {
          border-color: var(--color-warning);
          background: rgba(210, 153, 34, 0.05);
        }

        .process-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: var(--space-4);
        }

        .process-status {
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }

        .status-dot {
          width: 8px;
          height: 8px;
          border-radius: var(--radius-full);
          animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.5; }
        }

        .process-name {
          font-size: var(--text-sm);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          font-family: var(--font-mono);
        }

        .stop-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 28px;
          height: 28px;
          background: transparent;
          color: var(--text-tertiary);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .stop-btn:hover {
          background: var(--color-danger);
          color: white;
          border-color: var(--color-danger);
        }

        .status-badge {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          padding: var(--space-1) var(--space-2);
          border-radius: var(--radius-sm);
        }

        .status-badge.completed {
          background: rgba(37, 194, 160, 0.1);
          color: var(--color-primary);
        }

        /* Metrics */
        .process-metrics {
          display: flex;
          gap: var(--space-4);
          margin-bottom: var(--space-4);
        }

        .metric {
          display: flex;
          flex-direction: column;
          align-items: center;
          flex: 1;
          padding: var(--space-2);
          background: var(--bg-base);
          border-radius: var(--radius-md);
        }

        .metric-icon {
          color: var(--text-tertiary);
          margin-bottom: var(--space-1);
        }

        .metric-value {
          font-size: var(--text-base);
          font-weight: var(--font-semibold);
          font-family: var(--font-mono);
          color: var(--text-primary);
        }

        .metric-value.high {
          color: var(--color-warning);
        }

        .metric-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Footer */
        .process-footer {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          padding-top: var(--space-3);
          border-top: 1px solid var(--border-subtle);
        }

        .process-pid {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
        }

        .process-turns {
          font-size: var(--text-xs);
          color: var(--text-secondary);
        }

        .process-cost {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--color-primary);
          margin-left: auto;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .monitor-summary {
            flex-wrap: wrap;
            gap: var(--space-3);
          }

          .process-grid {
            padding: var(--space-4);
            grid-template-columns: 1fr;
          }
        }
      `})]})},_i={messages:c.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),shield:c.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"})}),activity:c.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"22 12 18 12 15 21 9 3 6 12 2 12"})}),logo:c.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("path",{d:"M12 6v12M6 12h12"}),c.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]})},u0=()=>{const[e,t]=U.useState("messages"),[n,r]=U.useState([]),[i,l]=U.useState("my-agent"),[o,a]=U.useState([]),[s,u]=U.useState(""),[f,h]=U.useState(!1),p=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;cn.useEffect(()=>{const E=async()=>{try{const N=await fetch("/api/agents");if(N.ok){const L=await N.json();a(L),L.length>0&&!i&&l(L[0].id)}}catch(N){console.error("Error fetching agents:",N)}};E();const w=setInterval(E,1e4);return()=>clearInterval(w)},[]);const k=E=>{const w=E.target.value;w==="__custom__"?h(!0):(l(w),h(!1))},S=()=>{s.trim()&&(l(s.trim()),h(!1),u(""))},b=E=>E.last_active?Date.now()-E.last_active<3e4:!1,m=E=>b(E)?"●":"○",g=async(E,w)=>{try{const N=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:w})});if(!N.ok){console.error("Failed to approve:",await N.text());return}r(L=>L.map(R=>R.id===E?{...R,status:"approved",reviewed_by:"user",review_notes:w}:R))}catch(N){console.error("Error approving:",N)}},v=async(E,w)=>{try{const N=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:w})});if(!N.ok){console.error("Failed to reject:",await N.text());return}r(L=>L.map(R=>R.id===E?{...R,status:"rejected",reviewed_by:"user",review_notes:w}:R))}catch(N){console.error("Error rejecting:",N)}};cn.useEffect(()=>{const E=async()=>{try{const N=await fetch("/api/approvals?status=pending");if(!N.ok){console.error("Failed to fetch approvals:",await N.text());return}const L=await N.json();r(L)}catch(N){console.error("Error fetching approvals:",N)}};E();const w=setInterval(E,5e3);return()=>clearInterval(w)},[]);const C=(n==null?void 0:n.filter(E=>E.status==="pending").length)||0;return c.jsxs("div",{className:"app",children:[c.jsxs("header",{className:"app-header",children:[c.jsxs("div",{className:"header-brand",children:[c.jsx("div",{className:"brand-logo",children:_i.logo}),c.jsxs("div",{className:"brand-text",children:[c.jsx("h1",{children:"AILANG"}),c.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),c.jsxs("nav",{className:"header-nav",children:[c.jsxs("button",{className:`nav-tab ${e==="messages"?"active":""}`,onClick:()=>t("messages"),children:[c.jsx("span",{className:"nav-icon",children:_i.messages}),c.jsx("span",{className:"nav-label",children:"Messages"})]}),c.jsxs("button",{className:`nav-tab ${e==="approvals"?"active":""}`,onClick:()=>t("approvals"),children:[c.jsx("span",{className:"nav-icon",children:_i.shield}),c.jsx("span",{className:"nav-label",children:"Approvals"}),C>0&&c.jsx("span",{className:"nav-badge",children:C})]}),c.jsxs("button",{className:`nav-tab ${e==="monitor"?"active":""}`,onClick:()=>t("monitor"),children:[c.jsx("span",{className:"nav-icon",children:_i.activity}),c.jsx("span",{className:"nav-label",children:"Monitor"})]})]}),c.jsxs("div",{className:"header-meta",children:[c.jsxs("div",{className:"agent-selector",children:[c.jsx("label",{className:"agent-label",children:"Target:"}),f?c.jsxs("div",{className:"custom-agent-input",children:[c.jsx("input",{type:"text",value:s,onChange:E=>u(E.target.value),onKeyDown:E=>E.key==="Enter"&&S(),className:"agent-input",placeholder:"agent-id",autoFocus:!0}),c.jsx("button",{onClick:S,className:"agent-apply",children:"Add"}),c.jsx("button",{onClick:()=>h(!1),className:"agent-cancel",children:"Cancel"})]}):c.jsxs(c.Fragment,{children:[c.jsxs("select",{value:i,onChange:k,className:"agent-select",children:[o.map(E=>c.jsxs("option",{value:E.id,children:[m(E)," ",E.id]},E.id)),!o.find(E=>E.id===i)&&i&&c.jsxs("option",{value:i,children:["○ ",i]}),c.jsx("option",{value:"__custom__",children:"+ Add custom..."})]}),o.find(E=>E.id===i)&&c.jsx("span",{className:`agent-status ${b(o.find(E=>E.id===i))?"active":"inactive"}`,children:b(o.find(E=>E.id===i))?"Online":"Offline"})]})]}),c.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),c.jsxs("main",{className:"app-content",children:[e==="messages"&&c.jsx(o0,{websocketUrl:p,instanceId:i}),e==="approvals"&&c.jsx(a0,{approvals:n,onApprove:g,onReject:v}),e==="monitor"&&c.jsx(s0,{})]}),c.jsx("style",{children:`
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
          height: 60px;
          padding: 0 var(--space-6);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
          flex-shrink: 0;
        }

        /* Brand */
        .header-brand {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .brand-logo {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 40px;
          height: 40px;
          background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));
          border-radius: var(--radius-lg);
          color: var(--text-inverse);
          box-shadow: var(--shadow-glow);
        }

        .brand-text h1 {
          font-size: var(--text-lg);
          font-weight: var(--font-bold);
          letter-spacing: -0.02em;
          color: var(--text-primary);
          line-height: 1;
          margin-bottom: 2px;
        }

        .brand-subtitle {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.1em;
        }

        /* Navigation */
        .header-nav {
          display: flex;
          gap: var(--space-1);
          background: var(--bg-base);
          padding: var(--space-1);
          border-radius: var(--radius-lg);
        }

        .nav-tab {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          padding: var(--space-2) var(--space-4);
          background: transparent;
          color: var(--text-secondary);
          font-family: var(--font-sans);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
          position: relative;
        }

        .nav-tab:hover {
          color: var(--text-primary);
          background: var(--bg-hover);
        }

        .nav-tab.active {
          color: var(--color-primary);
          background: var(--bg-elevated);
        }

        .nav-tab.active::after {
          content: '';
          position: absolute;
          bottom: -1px;
          left: 50%;
          transform: translateX(-50%);
          width: 20px;
          height: 2px;
          background: var(--color-primary);
          border-radius: var(--radius-full);
        }

        .nav-icon {
          display: flex;
          align-items: center;
        }

        .nav-label {
          display: block;
        }

        .nav-badge {
          display: flex;
          align-items: center;
          justify-content: center;
          min-width: 18px;
          height: 18px;
          padding: 0 var(--space-1);
          background: var(--color-danger);
          color: white;
          font-size: 11px;
          font-weight: var(--font-bold);
          border-radius: var(--radius-full);
          animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.8; }
        }

        /* Header Meta */
        .header-meta {
          display: flex;
          align-items: center;
          gap: var(--space-4);
        }

        .agent-selector {
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }

        .agent-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          white-space: nowrap;
        }

        .custom-agent-input {
          display: flex;
          align-items: center;
          gap: var(--space-1);
        }

        .agent-input {
          padding: var(--space-1) var(--space-2);
          background: var(--bg-base);
          color: var(--text-primary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-sm);
          width: 120px;
          transition: all var(--transition-fast);
        }

        .agent-input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.15);
        }

        .agent-select {
          padding: var(--space-1) var(--space-3);
          padding-right: var(--space-6);
          background: var(--bg-base);
          color: var(--text-primary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-sm);
          cursor: pointer;
          appearance: none;
          background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
          background-repeat: no-repeat;
          background-position: right var(--space-2) center;
          min-width: 140px;
          transition: all var(--transition-fast);
        }

        .agent-select:hover {
          border-color: var(--color-primary);
        }

        .agent-select:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.15);
        }

        .agent-apply {
          padding: var(--space-1) var(--space-2);
          background: var(--color-primary);
          color: var(--text-inverse);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .agent-apply:hover {
          background: var(--color-primary-light);
        }

        .agent-cancel {
          padding: var(--space-1) var(--space-2);
          background: transparent;
          color: var(--text-secondary);
          font-size: var(--text-xs);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .agent-cancel:hover {
          background: var(--bg-hover);
          color: var(--text-primary);
        }

        .agent-status {
          font-size: var(--text-xs);
          padding: 2px var(--space-2);
          border-radius: var(--radius-full);
          font-weight: var(--font-medium);
        }

        .agent-status.active {
          background: rgba(46, 160, 67, 0.15);
          color: var(--color-success);
        }

        .agent-status.inactive {
          background: var(--bg-elevated);
          color: var(--text-tertiary);
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

        /* Content */
        .app-content {
          flex: 1;
          overflow: hidden;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .app-header {
            padding: 0 var(--space-4);
          }

          .brand-text {
            display: none;
          }

          .nav-label {
            display: none;
          }

          .nav-tab {
            padding: var(--space-2) var(--space-3);
          }

          .version-tag {
            display: none;
          }
        }
      `})]})};ko.createRoot(document.getElementById("root")).render(c.jsx(cn.StrictMode,{children:c.jsx(u0,{})}));
