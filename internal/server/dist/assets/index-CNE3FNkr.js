(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Hi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function ja(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Oc={exports:{}},gl={},Fc={exports:{}},X={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ei=Symbol.for("react.element"),$f=Symbol.for("react.portal"),Hf=Symbol.for("react.fragment"),Vf=Symbol.for("react.strict_mode"),Wf=Symbol.for("react.profiler"),Qf=Symbol.for("react.provider"),Kf=Symbol.for("react.context"),qf=Symbol.for("react.forward_ref"),Yf=Symbol.for("react.suspense"),Xf=Symbol.for("react.memo"),Gf=Symbol.for("react.lazy"),Fs=Symbol.iterator;function Jf(e){return e===null||typeof e!="object"?null:(e=Fs&&e[Fs]||e["@@iterator"],typeof e=="function"?e:null)}var Bc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Uc=Object.assign,$c={};function rr(e,t,n){this.props=e,this.context=t,this.refs=$c,this.updater=n||Bc}rr.prototype.isReactComponent={};rr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};rr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function Hc(){}Hc.prototype=rr.prototype;function Ca(e,t,n){this.props=e,this.context=t,this.refs=$c,this.updater=n||Bc}var Ea=Ca.prototype=new Hc;Ea.constructor=Ca;Uc(Ea,rr.prototype);Ea.isPureReactComponent=!0;var Bs=Array.isArray,Vc=Object.prototype.hasOwnProperty,Na={current:null},Wc={key:!0,ref:!0,__self:!0,__source:!0};function Qc(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)Vc.call(t,r)&&!Wc.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),c=0;c<a;c++)s[c]=arguments[c+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:ei,type:e,key:l,ref:o,props:i,_owner:Na.current}}function Zf(e,t){return{$$typeof:ei,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function _a(e){return typeof e=="object"&&e!==null&&e.$$typeof===ei}function eh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Us=/\/+/g;function Dl(e,t){return typeof e=="object"&&e!==null&&e.key!=null?eh(""+e.key):t.toString(36)}function zi(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case ei:case $f:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Dl(o,0):r,Bs(i)?(n="",e!=null&&(n=e.replace(Us,"$&/")+"/"),zi(i,t,n,"",function(c){return c})):i!=null&&(_a(i)&&(i=Zf(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Us,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Bs(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Dl(l,a);o+=zi(l,t,n,s,i)}else if(s=Jf(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Dl(l,a++),o+=zi(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function si(e,t,n){if(e==null)return e;var r=[],i=0;return zi(e,r,"","",function(l){return t.call(n,l,i++)}),r}function th(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Me={current:null},Ti={transition:null},nh={ReactCurrentDispatcher:Me,ReactCurrentBatchConfig:Ti,ReactCurrentOwner:Na};function Kc(){throw Error("act(...) is not supported in production builds of React.")}X.Children={map:si,forEach:function(e,t,n){si(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return si(e,function(){t++}),t},toArray:function(e){return si(e,function(t){return t})||[]},only:function(e){if(!_a(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};X.Component=rr;X.Fragment=Hf;X.Profiler=Wf;X.PureComponent=Ca;X.StrictMode=Vf;X.Suspense=Yf;X.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=nh;X.act=Kc;X.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Uc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Na.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)Vc.call(t,s)&&!Wc.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var c=0;c<s;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:ei,type:e.type,key:i,ref:l,props:r,_owner:o}};X.createContext=function(e){return e={$$typeof:Kf,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:Qf,_context:e},e.Consumer=e};X.createElement=Qc;X.createFactory=function(e){var t=Qc.bind(null,e);return t.type=e,t};X.createRef=function(){return{current:null}};X.forwardRef=function(e){return{$$typeof:qf,render:e}};X.isValidElement=_a;X.lazy=function(e){return{$$typeof:Gf,_payload:{_status:-1,_result:e},_init:th}};X.memo=function(e,t){return{$$typeof:Xf,type:e,compare:t===void 0?null:t}};X.startTransition=function(e){var t=Ti.transition;Ti.transition={};try{e()}finally{Ti.transition=t}};X.unstable_act=Kc;X.useCallback=function(e,t){return Me.current.useCallback(e,t)};X.useContext=function(e){return Me.current.useContext(e)};X.useDebugValue=function(){};X.useDeferredValue=function(e){return Me.current.useDeferredValue(e)};X.useEffect=function(e,t){return Me.current.useEffect(e,t)};X.useId=function(){return Me.current.useId()};X.useImperativeHandle=function(e,t,n){return Me.current.useImperativeHandle(e,t,n)};X.useInsertionEffect=function(e,t){return Me.current.useInsertionEffect(e,t)};X.useLayoutEffect=function(e,t){return Me.current.useLayoutEffect(e,t)};X.useMemo=function(e,t){return Me.current.useMemo(e,t)};X.useReducer=function(e,t,n){return Me.current.useReducer(e,t,n)};X.useRef=function(e){return Me.current.useRef(e)};X.useState=function(e){return Me.current.useState(e)};X.useSyncExternalStore=function(e,t,n){return Me.current.useSyncExternalStore(e,t,n)};X.useTransition=function(){return Me.current.useTransition()};X.version="18.3.1";Fc.exports=X;var H=Fc.exports;const _t=ja(H);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var rh=H,ih=Symbol.for("react.element"),lh=Symbol.for("react.fragment"),oh=Object.prototype.hasOwnProperty,ah=rh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,sh={key:!0,ref:!0,__self:!0,__source:!0};function qc(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)oh.call(t,r)&&!sh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:ih,type:e,key:l,ref:o,props:i,_owner:ah.current}}gl.Fragment=lh;gl.jsx=qc;gl.jsxs=qc;Oc.exports=gl;var u=Oc.exports,ko={},Yc={exports:{}},Je={},Xc={exports:{}},Gc={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(A,N){var g=A.length;A.push(N);e:for(;0<g;){var D=g-1>>>1,W=A[D];if(0<i(W,N))A[D]=N,A[g]=W,g=D;else break e}}function n(A){return A.length===0?null:A[0]}function r(A){if(A.length===0)return null;var N=A[0],g=A.pop();if(g!==N){A[0]=g;e:for(var D=0,W=A.length,x=W>>>1;D<x;){var ne=2*(D+1)-1,Te=A[ne],te=ne+1,et=A[te];if(0>i(Te,g))te<W&&0>i(et,Te)?(A[D]=et,A[te]=g,D=te):(A[D]=Te,A[ne]=g,D=ne);else if(te<W&&0>i(et,g))A[D]=et,A[te]=g,D=te;else break e}}return N}function i(A,N){var g=A.sortIndex-N.sortIndex;return g!==0?g:A.id-N.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],c=[],d=1,p=null,m=3,f=!1,k=!1,w=!1,P=typeof setTimeout=="function"?setTimeout:null,h=typeof clearTimeout=="function"?clearTimeout:null,v=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(A){for(var N=n(c);N!==null;){if(N.callback===null)r(c);else if(N.startTime<=A)r(c),N.sortIndex=N.expirationTime,t(s,N);else break;N=n(c)}}function b(A){if(w=!1,y(A),!k)if(n(s)!==null)k=!0,U(_);else{var N=n(c);N!==null&&K(b,N.startTime-A)}}function _(A,N){k=!1,w&&(w=!1,h(C),C=-1),f=!0;var g=m;try{for(y(N),p=n(s);p!==null&&(!(p.expirationTime>N)||A&&!j());){var D=p.callback;if(typeof D=="function"){p.callback=null,m=p.priorityLevel;var W=D(p.expirationTime<=N);N=e.unstable_now(),typeof W=="function"?p.callback=W:p===n(s)&&r(s),y(N)}else r(s);p=n(s)}if(p!==null)var x=!0;else{var ne=n(c);ne!==null&&K(b,ne.startTime-N),x=!1}return x}finally{p=null,m=g,f=!1}}var S=!1,L=null,C=-1,T=5,R=-1;function j(){return!(e.unstable_now()-R<T)}function E(){if(L!==null){var A=e.unstable_now();R=A;var N=!0;try{N=L(!0,A)}finally{N?F():(S=!1,L=null)}}else S=!1}var F;if(typeof v=="function")F=function(){v(E)};else if(typeof MessageChannel<"u"){var V=new MessageChannel,B=V.port2;V.port1.onmessage=E,F=function(){B.postMessage(null)}}else F=function(){P(E,0)};function U(A){L=A,S||(S=!0,F())}function K(A,N){C=P(function(){A(e.unstable_now())},N)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(A){A.callback=null},e.unstable_continueExecution=function(){k||f||(k=!0,U(_))},e.unstable_forceFrameRate=function(A){0>A||125<A?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):T=0<A?Math.floor(1e3/A):5},e.unstable_getCurrentPriorityLevel=function(){return m},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(A){switch(m){case 1:case 2:case 3:var N=3;break;default:N=m}var g=m;m=N;try{return A()}finally{m=g}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(A,N){switch(A){case 1:case 2:case 3:case 4:case 5:break;default:A=3}var g=m;m=A;try{return N()}finally{m=g}},e.unstable_scheduleCallback=function(A,N,g){var D=e.unstable_now();switch(typeof g=="object"&&g!==null?(g=g.delay,g=typeof g=="number"&&0<g?D+g:D):g=D,A){case 1:var W=-1;break;case 2:W=250;break;case 5:W=1073741823;break;case 4:W=1e4;break;default:W=5e3}return W=g+W,A={id:d++,callback:N,priorityLevel:A,startTime:g,expirationTime:W,sortIndex:-1},g>D?(A.sortIndex=g,t(c,A),n(s)===null&&A===n(c)&&(w?(h(C),C=-1):w=!0,K(b,g-D))):(A.sortIndex=W,t(s,A),k||f||(k=!0,U(_))),A},e.unstable_shouldYield=j,e.unstable_wrapCallback=function(A){var N=m;return function(){var g=m;m=N;try{return A.apply(this,arguments)}finally{m=g}}}})(Gc);Xc.exports=Gc;var uh=Xc.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ch=H,Ge=uh;function I(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var Jc=new Set,Dr={};function Sn(e,t){Xn(e,t),Xn(e+"Capture",t)}function Xn(e,t){for(Dr[e]=t,e=0;e<t.length;e++)Jc.add(t[e])}var It=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),wo=Object.prototype.hasOwnProperty,dh=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,$s={},Hs={};function ph(e){return wo.call(Hs,e)?!0:wo.call($s,e)?!1:dh.test(e)?Hs[e]=!0:($s[e]=!0,!1)}function fh(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function hh(e,t,n,r){if(t===null||typeof t>"u"||fh(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Ae(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var je={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){je[e]=new Ae(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];je[t]=new Ae(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){je[e]=new Ae(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){je[e]=new Ae(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){je[e]=new Ae(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){je[e]=new Ae(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){je[e]=new Ae(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){je[e]=new Ae(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){je[e]=new Ae(e,5,!1,e.toLowerCase(),null,!1,!1)});var za=/[\-:]([a-z])/g;function Ta(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(za,Ta);je[t]=new Ae(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(za,Ta);je[t]=new Ae(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(za,Ta);je[t]=new Ae(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){je[e]=new Ae(e,1,!1,e.toLowerCase(),null,!1,!1)});je.xlinkHref=new Ae("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){je[e]=new Ae(e,1,!1,e.toLowerCase(),null,!0,!0)});function La(e,t,n,r){var i=je.hasOwnProperty(t)?je[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(hh(t,n,i,r)&&(n=null),r||i===null?ph(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Rt=ch.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,ui=Symbol.for("react.element"),Tn=Symbol.for("react.portal"),Ln=Symbol.for("react.fragment"),Pa=Symbol.for("react.strict_mode"),So=Symbol.for("react.profiler"),Zc=Symbol.for("react.provider"),ed=Symbol.for("react.context"),Ia=Symbol.for("react.forward_ref"),bo=Symbol.for("react.suspense"),jo=Symbol.for("react.suspense_list"),Ma=Symbol.for("react.memo"),Ut=Symbol.for("react.lazy"),td=Symbol.for("react.offscreen"),Vs=Symbol.iterator;function cr(e){return e===null||typeof e!="object"?null:(e=Vs&&e[Vs]||e["@@iterator"],typeof e=="function"?e:null)}var pe=Object.assign,Rl;function kr(e){if(Rl===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Rl=t&&t[1]||""}return`
`+Rl+e}var Ol=!1;function Fl(e,t){if(!e||Ol)return"";Ol=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Ol=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?kr(e):""}function mh(e){switch(e.tag){case 5:return kr(e.type);case 16:return kr("Lazy");case 13:return kr("Suspense");case 19:return kr("SuspenseList");case 0:case 2:case 15:return e=Fl(e.type,!1),e;case 11:return e=Fl(e.type.render,!1),e;case 1:return e=Fl(e.type,!0),e;default:return""}}function Co(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Ln:return"Fragment";case Tn:return"Portal";case So:return"Profiler";case Pa:return"StrictMode";case bo:return"Suspense";case jo:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case ed:return(e.displayName||"Context")+".Consumer";case Zc:return(e._context.displayName||"Context")+".Provider";case Ia:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Ma:return t=e.displayName||null,t!==null?t:Co(e.type)||"Memo";case Ut:t=e._payload,e=e._init;try{return Co(e(t))}catch{}}return null}function gh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Co(t);case 8:return t===Pa?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function tn(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function nd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function vh(e){var t=nd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function ci(e){e._valueTracker||(e._valueTracker=vh(e))}function rd(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=nd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Vi(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function Eo(e,t){var n=t.checked;return pe({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function Ws(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=tn(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function id(e,t){t=t.checked,t!=null&&La(e,"checked",t,!1)}function No(e,t){id(e,t);var n=tn(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?_o(e,t.type,n):t.hasOwnProperty("defaultValue")&&_o(e,t.type,tn(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function Qs(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function _o(e,t,n){(t!=="number"||Vi(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var wr=Array.isArray;function $n(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+tn(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function zo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(I(91));return pe({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function Ks(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(I(92));if(wr(n)){if(1<n.length)throw Error(I(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:tn(n)}}function ld(e,t){var n=tn(t.value),r=tn(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function qs(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function od(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function To(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?od(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var di,ad=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(di=di||document.createElement("div"),di.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=di.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Rr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var jr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},yh=["Webkit","ms","Moz","O"];Object.keys(jr).forEach(function(e){yh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),jr[t]=jr[e]})});function sd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||jr.hasOwnProperty(e)&&jr[e]?(""+t).trim():t+"px"}function ud(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=sd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var xh=pe({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Lo(e,t){if(t){if(xh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(I(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(I(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(I(61))}if(t.style!=null&&typeof t.style!="object")throw Error(I(62))}}function Po(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Io=null;function Aa(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Mo=null,Hn=null,Vn=null;function Ys(e){if(e=ri(e)){if(typeof Mo!="function")throw Error(I(280));var t=e.stateNode;t&&(t=wl(t),Mo(e.stateNode,e.type,t))}}function cd(e){Hn?Vn?Vn.push(e):Vn=[e]:Hn=e}function dd(){if(Hn){var e=Hn,t=Vn;if(Vn=Hn=null,Ys(e),t)for(e=0;e<t.length;e++)Ys(t[e])}}function pd(e,t){return e(t)}function fd(){}var Bl=!1;function hd(e,t,n){if(Bl)return e(t,n);Bl=!0;try{return pd(e,t,n)}finally{Bl=!1,(Hn!==null||Vn!==null)&&(fd(),dd())}}function Or(e,t){var n=e.stateNode;if(n===null)return null;var r=wl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(I(231,t,typeof n));return n}var Ao=!1;if(It)try{var dr={};Object.defineProperty(dr,"passive",{get:function(){Ao=!0}}),window.addEventListener("test",dr,dr),window.removeEventListener("test",dr,dr)}catch{Ao=!1}function kh(e,t,n,r,i,l,o,a,s){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Cr=!1,Wi=null,Qi=!1,Do=null,wh={onError:function(e){Cr=!0,Wi=e}};function Sh(e,t,n,r,i,l,o,a,s){Cr=!1,Wi=null,kh.apply(wh,arguments)}function bh(e,t,n,r,i,l,o,a,s){if(Sh.apply(this,arguments),Cr){if(Cr){var c=Wi;Cr=!1,Wi=null}else throw Error(I(198));Qi||(Qi=!0,Do=c)}}function bn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function md(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function Xs(e){if(bn(e)!==e)throw Error(I(188))}function jh(e){var t=e.alternate;if(!t){if(t=bn(e),t===null)throw Error(I(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return Xs(i),e;if(l===r)return Xs(i),t;l=l.sibling}throw Error(I(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(I(189))}}if(n.alternate!==r)throw Error(I(190))}if(n.tag!==3)throw Error(I(188));return n.stateNode.current===n?e:t}function gd(e){return e=jh(e),e!==null?vd(e):null}function vd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=vd(e);if(t!==null)return t;e=e.sibling}return null}var yd=Ge.unstable_scheduleCallback,Gs=Ge.unstable_cancelCallback,Ch=Ge.unstable_shouldYield,Eh=Ge.unstable_requestPaint,he=Ge.unstable_now,Nh=Ge.unstable_getCurrentPriorityLevel,Da=Ge.unstable_ImmediatePriority,xd=Ge.unstable_UserBlockingPriority,Ki=Ge.unstable_NormalPriority,_h=Ge.unstable_LowPriority,kd=Ge.unstable_IdlePriority,vl=null,wt=null;function zh(e){if(wt&&typeof wt.onCommitFiberRoot=="function")try{wt.onCommitFiberRoot(vl,e,void 0,(e.current.flags&128)===128)}catch{}}var ft=Math.clz32?Math.clz32:Ph,Th=Math.log,Lh=Math.LN2;function Ph(e){return e>>>=0,e===0?32:31-(Th(e)/Lh|0)|0}var pi=64,fi=4194304;function Sr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function qi(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=Sr(a):(l&=o,l!==0&&(r=Sr(l)))}else o=n&~i,o!==0?r=Sr(o):l!==0&&(r=Sr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-ft(t),i=1<<n,r|=e[n],t&=~i;return r}function Ih(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Mh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-ft(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Ih(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Ro(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function wd(){var e=pi;return pi<<=1,!(pi&4194240)&&(pi=64),e}function Ul(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function ti(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-ft(t),e[t]=n}function Ah(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-ft(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Ra(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-ft(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var ee=0;function Sd(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var bd,Oa,jd,Cd,Ed,Oo=!1,hi=[],Kt=null,qt=null,Yt=null,Fr=new Map,Br=new Map,Ht=[],Dh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function Js(e,t){switch(e){case"focusin":case"focusout":Kt=null;break;case"dragenter":case"dragleave":qt=null;break;case"mouseover":case"mouseout":Yt=null;break;case"pointerover":case"pointerout":Fr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Br.delete(t.pointerId)}}function pr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=ri(t),t!==null&&Oa(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Rh(e,t,n,r,i){switch(t){case"focusin":return Kt=pr(Kt,e,t,n,r,i),!0;case"dragenter":return qt=pr(qt,e,t,n,r,i),!0;case"mouseover":return Yt=pr(Yt,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Fr.set(l,pr(Fr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Br.set(l,pr(Br.get(l)||null,e,t,n,r,i)),!0}return!1}function Nd(e){var t=pn(e.target);if(t!==null){var n=bn(t);if(n!==null){if(t=n.tag,t===13){if(t=md(n),t!==null){e.blockedOn=t,Ed(e.priority,function(){jd(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Li(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Fo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Io=r,n.target.dispatchEvent(r),Io=null}else return t=ri(n),t!==null&&Oa(t),e.blockedOn=n,!1;t.shift()}return!0}function Zs(e,t,n){Li(e)&&n.delete(t)}function Oh(){Oo=!1,Kt!==null&&Li(Kt)&&(Kt=null),qt!==null&&Li(qt)&&(qt=null),Yt!==null&&Li(Yt)&&(Yt=null),Fr.forEach(Zs),Br.forEach(Zs)}function fr(e,t){e.blockedOn===t&&(e.blockedOn=null,Oo||(Oo=!0,Ge.unstable_scheduleCallback(Ge.unstable_NormalPriority,Oh)))}function Ur(e){function t(i){return fr(i,e)}if(0<hi.length){fr(hi[0],e);for(var n=1;n<hi.length;n++){var r=hi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(Kt!==null&&fr(Kt,e),qt!==null&&fr(qt,e),Yt!==null&&fr(Yt,e),Fr.forEach(t),Br.forEach(t),n=0;n<Ht.length;n++)r=Ht[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Ht.length&&(n=Ht[0],n.blockedOn===null);)Nd(n),n.blockedOn===null&&Ht.shift()}var Wn=Rt.ReactCurrentBatchConfig,Yi=!0;function Fh(e,t,n,r){var i=ee,l=Wn.transition;Wn.transition=null;try{ee=1,Fa(e,t,n,r)}finally{ee=i,Wn.transition=l}}function Bh(e,t,n,r){var i=ee,l=Wn.transition;Wn.transition=null;try{ee=4,Fa(e,t,n,r)}finally{ee=i,Wn.transition=l}}function Fa(e,t,n,r){if(Yi){var i=Fo(e,t,n,r);if(i===null)Gl(e,t,r,Xi,n),Js(e,r);else if(Rh(i,e,t,n,r))r.stopPropagation();else if(Js(e,r),t&4&&-1<Dh.indexOf(e)){for(;i!==null;){var l=ri(i);if(l!==null&&bd(l),l=Fo(e,t,n,r),l===null&&Gl(e,t,r,Xi,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else Gl(e,t,r,null,n)}}var Xi=null;function Fo(e,t,n,r){if(Xi=null,e=Aa(r),e=pn(e),e!==null)if(t=bn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=md(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return Xi=e,null}function _d(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(Nh()){case Da:return 1;case xd:return 4;case Ki:case _h:return 16;case kd:return 536870912;default:return 16}default:return 16}}var Wt=null,Ba=null,Pi=null;function zd(){if(Pi)return Pi;var e,t=Ba,n=t.length,r,i="value"in Wt?Wt.value:Wt.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Pi=i.slice(e,1<r?1-r:void 0)}function Ii(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function mi(){return!0}function eu(){return!1}function Ze(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?mi:eu,this.isPropagationStopped=eu,this}return pe(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=mi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=mi)},persist:function(){},isPersistent:mi}),t}var ir={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Ua=Ze(ir),ni=pe({},ir,{view:0,detail:0}),Uh=Ze(ni),$l,Hl,hr,yl=pe({},ni,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:$a,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==hr&&(hr&&e.type==="mousemove"?($l=e.screenX-hr.screenX,Hl=e.screenY-hr.screenY):Hl=$l=0,hr=e),$l)},movementY:function(e){return"movementY"in e?e.movementY:Hl}}),tu=Ze(yl),$h=pe({},yl,{dataTransfer:0}),Hh=Ze($h),Vh=pe({},ni,{relatedTarget:0}),Vl=Ze(Vh),Wh=pe({},ir,{animationName:0,elapsedTime:0,pseudoElement:0}),Qh=Ze(Wh),Kh=pe({},ir,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),qh=Ze(Kh),Yh=pe({},ir,{data:0}),nu=Ze(Yh),Xh={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},Gh={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},Jh={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function Zh(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=Jh[e])?!!t[e]:!1}function $a(){return Zh}var em=pe({},ni,{key:function(e){if(e.key){var t=Xh[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Ii(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?Gh[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:$a,charCode:function(e){return e.type==="keypress"?Ii(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Ii(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),tm=Ze(em),nm=pe({},yl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),ru=Ze(nm),rm=pe({},ni,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:$a}),im=Ze(rm),lm=pe({},ir,{propertyName:0,elapsedTime:0,pseudoElement:0}),om=Ze(lm),am=pe({},yl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),sm=Ze(am),um=[9,13,27,32],Ha=It&&"CompositionEvent"in window,Er=null;It&&"documentMode"in document&&(Er=document.documentMode);var cm=It&&"TextEvent"in window&&!Er,Td=It&&(!Ha||Er&&8<Er&&11>=Er),iu=" ",lu=!1;function Ld(e,t){switch(e){case"keyup":return um.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Pd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Pn=!1;function dm(e,t){switch(e){case"compositionend":return Pd(t);case"keypress":return t.which!==32?null:(lu=!0,iu);case"textInput":return e=t.data,e===iu&&lu?null:e;default:return null}}function pm(e,t){if(Pn)return e==="compositionend"||!Ha&&Ld(e,t)?(e=zd(),Pi=Ba=Wt=null,Pn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Td&&t.locale!=="ko"?null:t.data;default:return null}}var fm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function ou(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!fm[e.type]:t==="textarea"}function Id(e,t,n,r){cd(r),t=Gi(t,"onChange"),0<t.length&&(n=new Ua("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Nr=null,$r=null;function hm(e){Vd(e,0)}function xl(e){var t=An(e);if(rd(t))return e}function mm(e,t){if(e==="change")return t}var Md=!1;if(It){var Wl;if(It){var Ql="oninput"in document;if(!Ql){var au=document.createElement("div");au.setAttribute("oninput","return;"),Ql=typeof au.oninput=="function"}Wl=Ql}else Wl=!1;Md=Wl&&(!document.documentMode||9<document.documentMode)}function su(){Nr&&(Nr.detachEvent("onpropertychange",Ad),$r=Nr=null)}function Ad(e){if(e.propertyName==="value"&&xl($r)){var t=[];Id(t,$r,e,Aa(e)),hd(hm,t)}}function gm(e,t,n){e==="focusin"?(su(),Nr=t,$r=n,Nr.attachEvent("onpropertychange",Ad)):e==="focusout"&&su()}function vm(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return xl($r)}function ym(e,t){if(e==="click")return xl(t)}function xm(e,t){if(e==="input"||e==="change")return xl(t)}function km(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var mt=typeof Object.is=="function"?Object.is:km;function Hr(e,t){if(mt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!wo.call(t,i)||!mt(e[i],t[i]))return!1}return!0}function uu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function cu(e,t){var n=uu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=uu(n)}}function Dd(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Dd(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Rd(){for(var e=window,t=Vi();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Vi(e.document)}return t}function Va(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function wm(e){var t=Rd(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Dd(n.ownerDocument.documentElement,n)){if(r!==null&&Va(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=cu(n,l);var o=cu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Sm=It&&"documentMode"in document&&11>=document.documentMode,In=null,Bo=null,_r=null,Uo=!1;function du(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Uo||In==null||In!==Vi(r)||(r=In,"selectionStart"in r&&Va(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),_r&&Hr(_r,r)||(_r=r,r=Gi(Bo,"onSelect"),0<r.length&&(t=new Ua("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=In)))}function gi(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Mn={animationend:gi("Animation","AnimationEnd"),animationiteration:gi("Animation","AnimationIteration"),animationstart:gi("Animation","AnimationStart"),transitionend:gi("Transition","TransitionEnd")},Kl={},Od={};It&&(Od=document.createElement("div").style,"AnimationEvent"in window||(delete Mn.animationend.animation,delete Mn.animationiteration.animation,delete Mn.animationstart.animation),"TransitionEvent"in window||delete Mn.transitionend.transition);function kl(e){if(Kl[e])return Kl[e];if(!Mn[e])return e;var t=Mn[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Od)return Kl[e]=t[n];return e}var Fd=kl("animationend"),Bd=kl("animationiteration"),Ud=kl("animationstart"),$d=kl("transitionend"),Hd=new Map,pu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function rn(e,t){Hd.set(e,t),Sn(t,[e])}for(var ql=0;ql<pu.length;ql++){var Yl=pu[ql],bm=Yl.toLowerCase(),jm=Yl[0].toUpperCase()+Yl.slice(1);rn(bm,"on"+jm)}rn(Fd,"onAnimationEnd");rn(Bd,"onAnimationIteration");rn(Ud,"onAnimationStart");rn("dblclick","onDoubleClick");rn("focusin","onFocus");rn("focusout","onBlur");rn($d,"onTransitionEnd");Xn("onMouseEnter",["mouseout","mouseover"]);Xn("onMouseLeave",["mouseout","mouseover"]);Xn("onPointerEnter",["pointerout","pointerover"]);Xn("onPointerLeave",["pointerout","pointerover"]);Sn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));Sn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));Sn("onBeforeInput",["compositionend","keypress","textInput","paste"]);Sn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));Sn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));Sn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var br="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Cm=new Set("cancel close invalid load scroll toggle".split(" ").concat(br));function fu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,bh(r,t,void 0,e),e.currentTarget=null}function Vd(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,c=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;fu(i,a,c),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,c=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;fu(i,a,c),l=s}}}if(Qi)throw e=Do,Qi=!1,Do=null,e}function ae(e,t){var n=t[Qo];n===void 0&&(n=t[Qo]=new Set);var r=e+"__bubble";n.has(r)||(Wd(t,e,2,!1),n.add(r))}function Xl(e,t,n){var r=0;t&&(r|=4),Wd(n,e,r,t)}var vi="_reactListening"+Math.random().toString(36).slice(2);function Vr(e){if(!e[vi]){e[vi]=!0,Jc.forEach(function(n){n!=="selectionchange"&&(Cm.has(n)||Xl(n,!1,e),Xl(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[vi]||(t[vi]=!0,Xl("selectionchange",!1,t))}}function Wd(e,t,n,r){switch(_d(t)){case 1:var i=Fh;break;case 4:i=Bh;break;default:i=Fa}n=i.bind(null,t,n,e),i=void 0,!Ao||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function Gl(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=pn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}hd(function(){var c=l,d=Aa(n),p=[];e:{var m=Hd.get(e);if(m!==void 0){var f=Ua,k=e;switch(e){case"keypress":if(Ii(n)===0)break e;case"keydown":case"keyup":f=tm;break;case"focusin":k="focus",f=Vl;break;case"focusout":k="blur",f=Vl;break;case"beforeblur":case"afterblur":f=Vl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":f=tu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":f=Hh;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":f=im;break;case Fd:case Bd:case Ud:f=Qh;break;case $d:f=om;break;case"scroll":f=Uh;break;case"wheel":f=sm;break;case"copy":case"cut":case"paste":f=qh;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":f=ru}var w=(t&4)!==0,P=!w&&e==="scroll",h=w?m!==null?m+"Capture":null:m;w=[];for(var v=c,y;v!==null;){y=v;var b=y.stateNode;if(y.tag===5&&b!==null&&(y=b,h!==null&&(b=Or(v,h),b!=null&&w.push(Wr(v,b,y)))),P)break;v=v.return}0<w.length&&(m=new f(m,k,null,n,d),p.push({event:m,listeners:w}))}}if(!(t&7)){e:{if(m=e==="mouseover"||e==="pointerover",f=e==="mouseout"||e==="pointerout",m&&n!==Io&&(k=n.relatedTarget||n.fromElement)&&(pn(k)||k[Mt]))break e;if((f||m)&&(m=d.window===d?d:(m=d.ownerDocument)?m.defaultView||m.parentWindow:window,f?(k=n.relatedTarget||n.toElement,f=c,k=k?pn(k):null,k!==null&&(P=bn(k),k!==P||k.tag!==5&&k.tag!==6)&&(k=null)):(f=null,k=c),f!==k)){if(w=tu,b="onMouseLeave",h="onMouseEnter",v="mouse",(e==="pointerout"||e==="pointerover")&&(w=ru,b="onPointerLeave",h="onPointerEnter",v="pointer"),P=f==null?m:An(f),y=k==null?m:An(k),m=new w(b,v+"leave",f,n,d),m.target=P,m.relatedTarget=y,b=null,pn(d)===c&&(w=new w(h,v+"enter",k,n,d),w.target=y,w.relatedTarget=P,b=w),P=b,f&&k)t:{for(w=f,h=k,v=0,y=w;y;y=_n(y))v++;for(y=0,b=h;b;b=_n(b))y++;for(;0<v-y;)w=_n(w),v--;for(;0<y-v;)h=_n(h),y--;for(;v--;){if(w===h||h!==null&&w===h.alternate)break t;w=_n(w),h=_n(h)}w=null}else w=null;f!==null&&hu(p,m,f,w,!1),k!==null&&P!==null&&hu(p,P,k,w,!0)}}e:{if(m=c?An(c):window,f=m.nodeName&&m.nodeName.toLowerCase(),f==="select"||f==="input"&&m.type==="file")var _=mm;else if(ou(m))if(Md)_=xm;else{_=vm;var S=gm}else(f=m.nodeName)&&f.toLowerCase()==="input"&&(m.type==="checkbox"||m.type==="radio")&&(_=ym);if(_&&(_=_(e,c))){Id(p,_,n,d);break e}S&&S(e,m,c),e==="focusout"&&(S=m._wrapperState)&&S.controlled&&m.type==="number"&&_o(m,"number",m.value)}switch(S=c?An(c):window,e){case"focusin":(ou(S)||S.contentEditable==="true")&&(In=S,Bo=c,_r=null);break;case"focusout":_r=Bo=In=null;break;case"mousedown":Uo=!0;break;case"contextmenu":case"mouseup":case"dragend":Uo=!1,du(p,n,d);break;case"selectionchange":if(Sm)break;case"keydown":case"keyup":du(p,n,d)}var L;if(Ha)e:{switch(e){case"compositionstart":var C="onCompositionStart";break e;case"compositionend":C="onCompositionEnd";break e;case"compositionupdate":C="onCompositionUpdate";break e}C=void 0}else Pn?Ld(e,n)&&(C="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(C="onCompositionStart");C&&(Td&&n.locale!=="ko"&&(Pn||C!=="onCompositionStart"?C==="onCompositionEnd"&&Pn&&(L=zd()):(Wt=d,Ba="value"in Wt?Wt.value:Wt.textContent,Pn=!0)),S=Gi(c,C),0<S.length&&(C=new nu(C,e,null,n,d),p.push({event:C,listeners:S}),L?C.data=L:(L=Pd(n),L!==null&&(C.data=L)))),(L=cm?dm(e,n):pm(e,n))&&(c=Gi(c,"onBeforeInput"),0<c.length&&(d=new nu("onBeforeInput","beforeinput",null,n,d),p.push({event:d,listeners:c}),d.data=L))}Vd(p,t)})}function Wr(e,t,n){return{instance:e,listener:t,currentTarget:n}}function Gi(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Or(e,n),l!=null&&r.unshift(Wr(e,l,i)),l=Or(e,t),l!=null&&r.push(Wr(e,l,i))),e=e.return}return r}function _n(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function hu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,c=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&c!==null&&(a=c,i?(s=Or(n,l),s!=null&&o.unshift(Wr(n,s,a))):i||(s=Or(n,l),s!=null&&o.push(Wr(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Em=/\r\n?/g,Nm=/\u0000|\uFFFD/g;function mu(e){return(typeof e=="string"?e:""+e).replace(Em,`
`).replace(Nm,"")}function yi(e,t,n){if(t=mu(t),mu(e)!==t&&n)throw Error(I(425))}function Ji(){}var $o=null,Ho=null;function Vo(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var Wo=typeof setTimeout=="function"?setTimeout:void 0,_m=typeof clearTimeout=="function"?clearTimeout:void 0,gu=typeof Promise=="function"?Promise:void 0,zm=typeof queueMicrotask=="function"?queueMicrotask:typeof gu<"u"?function(e){return gu.resolve(null).then(e).catch(Tm)}:Wo;function Tm(e){setTimeout(function(){throw e})}function Jl(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Ur(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Ur(t)}function Xt(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function vu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var lr=Math.random().toString(36).slice(2),xt="__reactFiber$"+lr,Qr="__reactProps$"+lr,Mt="__reactContainer$"+lr,Qo="__reactEvents$"+lr,Lm="__reactListeners$"+lr,Pm="__reactHandles$"+lr;function pn(e){var t=e[xt];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Mt]||n[xt]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=vu(e);e!==null;){if(n=e[xt])return n;e=vu(e)}return t}e=n,n=e.parentNode}return null}function ri(e){return e=e[xt]||e[Mt],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function An(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(I(33))}function wl(e){return e[Qr]||null}var Ko=[],Dn=-1;function ln(e){return{current:e}}function se(e){0>Dn||(e.current=Ko[Dn],Ko[Dn]=null,Dn--)}function le(e,t){Dn++,Ko[Dn]=e.current,e.current=t}var nn={},_e=ln(nn),Fe=ln(!1),vn=nn;function Gn(e,t){var n=e.type.contextTypes;if(!n)return nn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Be(e){return e=e.childContextTypes,e!=null}function Zi(){se(Fe),se(_e)}function yu(e,t,n){if(_e.current!==nn)throw Error(I(168));le(_e,t),le(Fe,n)}function Qd(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(I(108,gh(e)||"Unknown",i));return pe({},n,r)}function el(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||nn,vn=_e.current,le(_e,e),le(Fe,Fe.current),!0}function xu(e,t,n){var r=e.stateNode;if(!r)throw Error(I(169));n?(e=Qd(e,t,vn),r.__reactInternalMemoizedMergedChildContext=e,se(Fe),se(_e),le(_e,e)):se(Fe),le(Fe,n)}var zt=null,Sl=!1,Zl=!1;function Kd(e){zt===null?zt=[e]:zt.push(e)}function Im(e){Sl=!0,Kd(e)}function on(){if(!Zl&&zt!==null){Zl=!0;var e=0,t=ee;try{var n=zt;for(ee=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}zt=null,Sl=!1}catch(i){throw zt!==null&&(zt=zt.slice(e+1)),yd(Da,on),i}finally{ee=t,Zl=!1}}return null}var Rn=[],On=0,tl=null,nl=0,tt=[],nt=0,yn=null,Tt=1,Lt="";function un(e,t){Rn[On++]=nl,Rn[On++]=tl,tl=e,nl=t}function qd(e,t,n){tt[nt++]=Tt,tt[nt++]=Lt,tt[nt++]=yn,yn=e;var r=Tt;e=Lt;var i=32-ft(r)-1;r&=~(1<<i),n+=1;var l=32-ft(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,Tt=1<<32-ft(t)+i|n<<i|r,Lt=l+e}else Tt=1<<l|n<<i|r,Lt=e}function Wa(e){e.return!==null&&(un(e,1),qd(e,1,0))}function Qa(e){for(;e===tl;)tl=Rn[--On],Rn[On]=null,nl=Rn[--On],Rn[On]=null;for(;e===yn;)yn=tt[--nt],tt[nt]=null,Lt=tt[--nt],tt[nt]=null,Tt=tt[--nt],tt[nt]=null}var Xe=null,qe=null,ue=!1,pt=null;function Yd(e,t){var n=it(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function ku(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,Xe=e,qe=Xt(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,Xe=e,qe=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=yn!==null?{id:Tt,overflow:Lt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=it(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,Xe=e,qe=null,!0):!1;default:return!1}}function qo(e){return(e.mode&1)!==0&&(e.flags&128)===0}function Yo(e){if(ue){var t=qe;if(t){var n=t;if(!ku(e,t)){if(qo(e))throw Error(I(418));t=Xt(n.nextSibling);var r=Xe;t&&ku(e,t)?Yd(r,n):(e.flags=e.flags&-4097|2,ue=!1,Xe=e)}}else{if(qo(e))throw Error(I(418));e.flags=e.flags&-4097|2,ue=!1,Xe=e}}}function wu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;Xe=e}function xi(e){if(e!==Xe)return!1;if(!ue)return wu(e),ue=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!Vo(e.type,e.memoizedProps)),t&&(t=qe)){if(qo(e))throw Xd(),Error(I(418));for(;t;)Yd(e,t),t=Xt(t.nextSibling)}if(wu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(I(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){qe=Xt(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}qe=null}}else qe=Xe?Xt(e.stateNode.nextSibling):null;return!0}function Xd(){for(var e=qe;e;)e=Xt(e.nextSibling)}function Jn(){qe=Xe=null,ue=!1}function Ka(e){pt===null?pt=[e]:pt.push(e)}var Mm=Rt.ReactCurrentBatchConfig;function mr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(I(309));var r=n.stateNode}if(!r)throw Error(I(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(I(284));if(!n._owner)throw Error(I(290,e))}return e}function ki(e,t){throw e=Object.prototype.toString.call(t),Error(I(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Su(e){var t=e._init;return t(e._payload)}function Gd(e){function t(h,v){if(e){var y=h.deletions;y===null?(h.deletions=[v],h.flags|=16):y.push(v)}}function n(h,v){if(!e)return null;for(;v!==null;)t(h,v),v=v.sibling;return null}function r(h,v){for(h=new Map;v!==null;)v.key!==null?h.set(v.key,v):h.set(v.index,v),v=v.sibling;return h}function i(h,v){return h=en(h,v),h.index=0,h.sibling=null,h}function l(h,v,y){return h.index=y,e?(y=h.alternate,y!==null?(y=y.index,y<v?(h.flags|=2,v):y):(h.flags|=2,v)):(h.flags|=1048576,v)}function o(h){return e&&h.alternate===null&&(h.flags|=2),h}function a(h,v,y,b){return v===null||v.tag!==6?(v=oo(y,h.mode,b),v.return=h,v):(v=i(v,y),v.return=h,v)}function s(h,v,y,b){var _=y.type;return _===Ln?d(h,v,y.props.children,b,y.key):v!==null&&(v.elementType===_||typeof _=="object"&&_!==null&&_.$$typeof===Ut&&Su(_)===v.type)?(b=i(v,y.props),b.ref=mr(h,v,y),b.return=h,b):(b=Bi(y.type,y.key,y.props,null,h.mode,b),b.ref=mr(h,v,y),b.return=h,b)}function c(h,v,y,b){return v===null||v.tag!==4||v.stateNode.containerInfo!==y.containerInfo||v.stateNode.implementation!==y.implementation?(v=ao(y,h.mode,b),v.return=h,v):(v=i(v,y.children||[]),v.return=h,v)}function d(h,v,y,b,_){return v===null||v.tag!==7?(v=gn(y,h.mode,b,_),v.return=h,v):(v=i(v,y),v.return=h,v)}function p(h,v,y){if(typeof v=="string"&&v!==""||typeof v=="number")return v=oo(""+v,h.mode,y),v.return=h,v;if(typeof v=="object"&&v!==null){switch(v.$$typeof){case ui:return y=Bi(v.type,v.key,v.props,null,h.mode,y),y.ref=mr(h,null,v),y.return=h,y;case Tn:return v=ao(v,h.mode,y),v.return=h,v;case Ut:var b=v._init;return p(h,b(v._payload),y)}if(wr(v)||cr(v))return v=gn(v,h.mode,y,null),v.return=h,v;ki(h,v)}return null}function m(h,v,y,b){var _=v!==null?v.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return _!==null?null:a(h,v,""+y,b);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case ui:return y.key===_?s(h,v,y,b):null;case Tn:return y.key===_?c(h,v,y,b):null;case Ut:return _=y._init,m(h,v,_(y._payload),b)}if(wr(y)||cr(y))return _!==null?null:d(h,v,y,b,null);ki(h,y)}return null}function f(h,v,y,b,_){if(typeof b=="string"&&b!==""||typeof b=="number")return h=h.get(y)||null,a(v,h,""+b,_);if(typeof b=="object"&&b!==null){switch(b.$$typeof){case ui:return h=h.get(b.key===null?y:b.key)||null,s(v,h,b,_);case Tn:return h=h.get(b.key===null?y:b.key)||null,c(v,h,b,_);case Ut:var S=b._init;return f(h,v,y,S(b._payload),_)}if(wr(b)||cr(b))return h=h.get(y)||null,d(v,h,b,_,null);ki(v,b)}return null}function k(h,v,y,b){for(var _=null,S=null,L=v,C=v=0,T=null;L!==null&&C<y.length;C++){L.index>C?(T=L,L=null):T=L.sibling;var R=m(h,L,y[C],b);if(R===null){L===null&&(L=T);break}e&&L&&R.alternate===null&&t(h,L),v=l(R,v,C),S===null?_=R:S.sibling=R,S=R,L=T}if(C===y.length)return n(h,L),ue&&un(h,C),_;if(L===null){for(;C<y.length;C++)L=p(h,y[C],b),L!==null&&(v=l(L,v,C),S===null?_=L:S.sibling=L,S=L);return ue&&un(h,C),_}for(L=r(h,L);C<y.length;C++)T=f(L,h,C,y[C],b),T!==null&&(e&&T.alternate!==null&&L.delete(T.key===null?C:T.key),v=l(T,v,C),S===null?_=T:S.sibling=T,S=T);return e&&L.forEach(function(j){return t(h,j)}),ue&&un(h,C),_}function w(h,v,y,b){var _=cr(y);if(typeof _!="function")throw Error(I(150));if(y=_.call(y),y==null)throw Error(I(151));for(var S=_=null,L=v,C=v=0,T=null,R=y.next();L!==null&&!R.done;C++,R=y.next()){L.index>C?(T=L,L=null):T=L.sibling;var j=m(h,L,R.value,b);if(j===null){L===null&&(L=T);break}e&&L&&j.alternate===null&&t(h,L),v=l(j,v,C),S===null?_=j:S.sibling=j,S=j,L=T}if(R.done)return n(h,L),ue&&un(h,C),_;if(L===null){for(;!R.done;C++,R=y.next())R=p(h,R.value,b),R!==null&&(v=l(R,v,C),S===null?_=R:S.sibling=R,S=R);return ue&&un(h,C),_}for(L=r(h,L);!R.done;C++,R=y.next())R=f(L,h,C,R.value,b),R!==null&&(e&&R.alternate!==null&&L.delete(R.key===null?C:R.key),v=l(R,v,C),S===null?_=R:S.sibling=R,S=R);return e&&L.forEach(function(E){return t(h,E)}),ue&&un(h,C),_}function P(h,v,y,b){if(typeof y=="object"&&y!==null&&y.type===Ln&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case ui:e:{for(var _=y.key,S=v;S!==null;){if(S.key===_){if(_=y.type,_===Ln){if(S.tag===7){n(h,S.sibling),v=i(S,y.props.children),v.return=h,h=v;break e}}else if(S.elementType===_||typeof _=="object"&&_!==null&&_.$$typeof===Ut&&Su(_)===S.type){n(h,S.sibling),v=i(S,y.props),v.ref=mr(h,S,y),v.return=h,h=v;break e}n(h,S);break}else t(h,S);S=S.sibling}y.type===Ln?(v=gn(y.props.children,h.mode,b,y.key),v.return=h,h=v):(b=Bi(y.type,y.key,y.props,null,h.mode,b),b.ref=mr(h,v,y),b.return=h,h=b)}return o(h);case Tn:e:{for(S=y.key;v!==null;){if(v.key===S)if(v.tag===4&&v.stateNode.containerInfo===y.containerInfo&&v.stateNode.implementation===y.implementation){n(h,v.sibling),v=i(v,y.children||[]),v.return=h,h=v;break e}else{n(h,v);break}else t(h,v);v=v.sibling}v=ao(y,h.mode,b),v.return=h,h=v}return o(h);case Ut:return S=y._init,P(h,v,S(y._payload),b)}if(wr(y))return k(h,v,y,b);if(cr(y))return w(h,v,y,b);ki(h,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,v!==null&&v.tag===6?(n(h,v.sibling),v=i(v,y),v.return=h,h=v):(n(h,v),v=oo(y,h.mode,b),v.return=h,h=v),o(h)):n(h,v)}return P}var Zn=Gd(!0),Jd=Gd(!1),rl=ln(null),il=null,Fn=null,qa=null;function Ya(){qa=Fn=il=null}function Xa(e){var t=rl.current;se(rl),e._currentValue=t}function Xo(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Qn(e,t){il=e,qa=Fn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(Oe=!0),e.firstContext=null)}function ot(e){var t=e._currentValue;if(qa!==e)if(e={context:e,memoizedValue:t,next:null},Fn===null){if(il===null)throw Error(I(308));Fn=e,il.dependencies={lanes:0,firstContext:e}}else Fn=Fn.next=e;return t}var fn=null;function Ga(e){fn===null?fn=[e]:fn.push(e)}function Zd(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,Ga(t)):(n.next=i.next,i.next=n),t.interleaved=n,At(e,r)}function At(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var $t=!1;function Ja(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function ep(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Pt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function Gt(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,J&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,At(e,n)}return i=r.interleaved,i===null?(t.next=t,Ga(r)):(t.next=i.next,i.next=t),r.interleaved=t,At(e,n)}function Mi(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ra(e,n)}}function bu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function ll(e,t,n,r){var i=e.updateQueue;$t=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,c=s.next;s.next=null,o===null?l=c:o.next=c,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=s))}if(l!==null){var p=i.baseState;o=0,d=c=s=null,a=l;do{var m=a.lane,f=a.eventTime;if((r&m)===m){d!==null&&(d=d.next={eventTime:f,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var k=e,w=a;switch(m=t,f=n,w.tag){case 1:if(k=w.payload,typeof k=="function"){p=k.call(f,p,m);break e}p=k;break e;case 3:k.flags=k.flags&-65537|128;case 0:if(k=w.payload,m=typeof k=="function"?k.call(f,p,m):k,m==null)break e;p=pe({},p,m);break e;case 2:$t=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,m=i.effects,m===null?i.effects=[a]:m.push(a))}else f={eventTime:f,lane:m,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=f,s=p):d=d.next=f,o|=m;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;m=a,a=m.next,m.next=null,i.lastBaseUpdate=m,i.shared.pending=null}}while(!0);if(d===null&&(s=p),i.baseState=s,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);kn|=o,e.lanes=o,e.memoizedState=p}}function ju(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(I(191,i));i.call(r)}}}var ii={},St=ln(ii),Kr=ln(ii),qr=ln(ii);function hn(e){if(e===ii)throw Error(I(174));return e}function Za(e,t){switch(le(qr,t),le(Kr,e),le(St,ii),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:To(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=To(t,e)}se(St),le(St,t)}function er(){se(St),se(Kr),se(qr)}function tp(e){hn(qr.current);var t=hn(St.current),n=To(t,e.type);t!==n&&(le(Kr,e),le(St,n))}function es(e){Kr.current===e&&(se(St),se(Kr))}var ce=ln(0);function ol(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var eo=[];function ts(){for(var e=0;e<eo.length;e++)eo[e]._workInProgressVersionPrimary=null;eo.length=0}var Ai=Rt.ReactCurrentDispatcher,to=Rt.ReactCurrentBatchConfig,xn=0,de=null,ye=null,ke=null,al=!1,zr=!1,Yr=0,Am=0;function Ce(){throw Error(I(321))}function ns(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!mt(e[n],t[n]))return!1;return!0}function rs(e,t,n,r,i,l){if(xn=l,de=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Ai.current=e===null||e.memoizedState===null?Fm:Bm,e=n(r,i),zr){l=0;do{if(zr=!1,Yr=0,25<=l)throw Error(I(301));l+=1,ke=ye=null,t.updateQueue=null,Ai.current=Um,e=n(r,i)}while(zr)}if(Ai.current=sl,t=ye!==null&&ye.next!==null,xn=0,ke=ye=de=null,al=!1,t)throw Error(I(300));return e}function is(){var e=Yr!==0;return Yr=0,e}function vt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return ke===null?de.memoizedState=ke=e:ke=ke.next=e,ke}function at(){if(ye===null){var e=de.alternate;e=e!==null?e.memoizedState:null}else e=ye.next;var t=ke===null?de.memoizedState:ke.next;if(t!==null)ke=t,ye=e;else{if(e===null)throw Error(I(310));ye=e,e={memoizedState:ye.memoizedState,baseState:ye.baseState,baseQueue:ye.baseQueue,queue:ye.queue,next:null},ke===null?de.memoizedState=ke=e:ke=ke.next=e}return ke}function Xr(e,t){return typeof t=="function"?t(e):t}function no(e){var t=at(),n=t.queue;if(n===null)throw Error(I(311));n.lastRenderedReducer=e;var r=ye,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,c=l;do{var d=c.lane;if((xn&d)===d)s!==null&&(s=s.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var p={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};s===null?(a=s=p,o=r):s=s.next=p,de.lanes|=d,kn|=d}c=c.next}while(c!==null&&c!==l);s===null?o=r:s.next=a,mt(r,t.memoizedState)||(Oe=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,de.lanes|=l,kn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function ro(e){var t=at(),n=t.queue;if(n===null)throw Error(I(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);mt(l,t.memoizedState)||(Oe=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function np(){}function rp(e,t){var n=de,r=at(),i=t(),l=!mt(r.memoizedState,i);if(l&&(r.memoizedState=i,Oe=!0),r=r.queue,ls(op.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||ke!==null&&ke.memoizedState.tag&1){if(n.flags|=2048,Gr(9,lp.bind(null,n,r,i,t),void 0,null),we===null)throw Error(I(349));xn&30||ip(n,t,i)}return i}function ip(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=de.updateQueue,t===null?(t={lastEffect:null,stores:null},de.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function lp(e,t,n,r){t.value=n,t.getSnapshot=r,ap(t)&&sp(e)}function op(e,t,n){return n(function(){ap(t)&&sp(e)})}function ap(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!mt(e,n)}catch{return!0}}function sp(e){var t=At(e,1);t!==null&&ht(t,e,1,-1)}function Cu(e){var t=vt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:Xr,lastRenderedState:e},t.queue=e,e=e.dispatch=Om.bind(null,de,e),[t.memoizedState,e]}function Gr(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=de.updateQueue,t===null?(t={lastEffect:null,stores:null},de.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function up(){return at().memoizedState}function Di(e,t,n,r){var i=vt();de.flags|=e,i.memoizedState=Gr(1|t,n,void 0,r===void 0?null:r)}function bl(e,t,n,r){var i=at();r=r===void 0?null:r;var l=void 0;if(ye!==null){var o=ye.memoizedState;if(l=o.destroy,r!==null&&ns(r,o.deps)){i.memoizedState=Gr(t,n,l,r);return}}de.flags|=e,i.memoizedState=Gr(1|t,n,l,r)}function Eu(e,t){return Di(8390656,8,e,t)}function ls(e,t){return bl(2048,8,e,t)}function cp(e,t){return bl(4,2,e,t)}function dp(e,t){return bl(4,4,e,t)}function pp(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function fp(e,t,n){return n=n!=null?n.concat([e]):null,bl(4,4,pp.bind(null,t,e),n)}function os(){}function hp(e,t){var n=at();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ns(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function mp(e,t){var n=at();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ns(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function gp(e,t,n){return xn&21?(mt(n,t)||(n=wd(),de.lanes|=n,kn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,Oe=!0),e.memoizedState=n)}function Dm(e,t){var n=ee;ee=n!==0&&4>n?n:4,e(!0);var r=to.transition;to.transition={};try{e(!1),t()}finally{ee=n,to.transition=r}}function vp(){return at().memoizedState}function Rm(e,t,n){var r=Zt(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},yp(e))xp(t,n);else if(n=Zd(e,t,n,r),n!==null){var i=Ie();ht(n,e,r,i),kp(n,t,r)}}function Om(e,t,n){var r=Zt(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(yp(e))xp(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,mt(a,o)){var s=t.interleaved;s===null?(i.next=i,Ga(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=Zd(e,t,i,r),n!==null&&(i=Ie(),ht(n,e,r,i),kp(n,t,r))}}function yp(e){var t=e.alternate;return e===de||t!==null&&t===de}function xp(e,t){zr=al=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function kp(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ra(e,n)}}var sl={readContext:ot,useCallback:Ce,useContext:Ce,useEffect:Ce,useImperativeHandle:Ce,useInsertionEffect:Ce,useLayoutEffect:Ce,useMemo:Ce,useReducer:Ce,useRef:Ce,useState:Ce,useDebugValue:Ce,useDeferredValue:Ce,useTransition:Ce,useMutableSource:Ce,useSyncExternalStore:Ce,useId:Ce,unstable_isNewReconciler:!1},Fm={readContext:ot,useCallback:function(e,t){return vt().memoizedState=[e,t===void 0?null:t],e},useContext:ot,useEffect:Eu,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Di(4194308,4,pp.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Di(4194308,4,e,t)},useInsertionEffect:function(e,t){return Di(4,2,e,t)},useMemo:function(e,t){var n=vt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=vt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Rm.bind(null,de,e),[r.memoizedState,e]},useRef:function(e){var t=vt();return e={current:e},t.memoizedState=e},useState:Cu,useDebugValue:os,useDeferredValue:function(e){return vt().memoizedState=e},useTransition:function(){var e=Cu(!1),t=e[0];return e=Dm.bind(null,e[1]),vt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=de,i=vt();if(ue){if(n===void 0)throw Error(I(407));n=n()}else{if(n=t(),we===null)throw Error(I(349));xn&30||ip(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Eu(op.bind(null,r,l,e),[e]),r.flags|=2048,Gr(9,lp.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=vt(),t=we.identifierPrefix;if(ue){var n=Lt,r=Tt;n=(r&~(1<<32-ft(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=Yr++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Am++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},Bm={readContext:ot,useCallback:hp,useContext:ot,useEffect:ls,useImperativeHandle:fp,useInsertionEffect:cp,useLayoutEffect:dp,useMemo:mp,useReducer:no,useRef:up,useState:function(){return no(Xr)},useDebugValue:os,useDeferredValue:function(e){var t=at();return gp(t,ye.memoizedState,e)},useTransition:function(){var e=no(Xr)[0],t=at().memoizedState;return[e,t]},useMutableSource:np,useSyncExternalStore:rp,useId:vp,unstable_isNewReconciler:!1},Um={readContext:ot,useCallback:hp,useContext:ot,useEffect:ls,useImperativeHandle:fp,useInsertionEffect:cp,useLayoutEffect:dp,useMemo:mp,useReducer:ro,useRef:up,useState:function(){return ro(Xr)},useDebugValue:os,useDeferredValue:function(e){var t=at();return ye===null?t.memoizedState=e:gp(t,ye.memoizedState,e)},useTransition:function(){var e=ro(Xr)[0],t=at().memoizedState;return[e,t]},useMutableSource:np,useSyncExternalStore:rp,useId:vp,unstable_isNewReconciler:!1};function ct(e,t){if(e&&e.defaultProps){t=pe({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function Go(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:pe({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var jl={isMounted:function(e){return(e=e._reactInternals)?bn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Ie(),i=Zt(e),l=Pt(r,i);l.payload=t,n!=null&&(l.callback=n),t=Gt(e,l,i),t!==null&&(ht(t,e,i,r),Mi(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Ie(),i=Zt(e),l=Pt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=Gt(e,l,i),t!==null&&(ht(t,e,i,r),Mi(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Ie(),r=Zt(e),i=Pt(n,r);i.tag=2,t!=null&&(i.callback=t),t=Gt(e,i,r),t!==null&&(ht(t,e,r,n),Mi(t,e,r))}};function Nu(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Hr(n,r)||!Hr(i,l):!0}function wp(e,t,n){var r=!1,i=nn,l=t.contextType;return typeof l=="object"&&l!==null?l=ot(l):(i=Be(t)?vn:_e.current,r=t.contextTypes,l=(r=r!=null)?Gn(e,i):nn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=jl,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function _u(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&jl.enqueueReplaceState(t,t.state,null)}function Jo(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},Ja(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=ot(l):(l=Be(t)?vn:_e.current,i.context=Gn(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(Go(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&jl.enqueueReplaceState(i,i.state,null),ll(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function tr(e,t){try{var n="",r=t;do n+=mh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function io(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function Zo(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var $m=typeof WeakMap=="function"?WeakMap:Map;function Sp(e,t,n){n=Pt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){cl||(cl=!0,ua=r),Zo(e,t)},n}function bp(e,t,n){n=Pt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){Zo(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){Zo(e,t),typeof r!="function"&&(Jt===null?Jt=new Set([this]):Jt.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function zu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new $m;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=ng.bind(null,e,t,n),t.then(e,e))}function Tu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Lu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Pt(-1,1),t.tag=2,Gt(n,t,1))),n.lanes|=1),e)}var Hm=Rt.ReactCurrentOwner,Oe=!1;function Pe(e,t,n,r){t.child=e===null?Jd(t,null,n,r):Zn(t,e.child,n,r)}function Pu(e,t,n,r,i){n=n.render;var l=t.ref;return Qn(t,i),r=rs(e,t,n,r,l,i),n=is(),e!==null&&!Oe?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Dt(e,t,i)):(ue&&n&&Wa(t),t.flags|=1,Pe(e,t,r,i),t.child)}function Iu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!hs(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,jp(e,t,l,r,i)):(e=Bi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Hr,n(o,r)&&e.ref===t.ref)return Dt(e,t,i)}return t.flags|=1,e=en(l,r),e.ref=t.ref,e.return=t,t.child=e}function jp(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Hr(l,r)&&e.ref===t.ref)if(Oe=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(Oe=!0);else return t.lanes=e.lanes,Dt(e,t,i)}return ea(e,t,n,r,i)}function Cp(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},le(Un,Qe),Qe|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,le(Un,Qe),Qe|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,le(Un,Qe),Qe|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,le(Un,Qe),Qe|=r;return Pe(e,t,i,n),t.child}function Ep(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ea(e,t,n,r,i){var l=Be(n)?vn:_e.current;return l=Gn(t,l),Qn(t,i),n=rs(e,t,n,r,l,i),r=is(),e!==null&&!Oe?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Dt(e,t,i)):(ue&&r&&Wa(t),t.flags|=1,Pe(e,t,n,i),t.child)}function Mu(e,t,n,r,i){if(Be(n)){var l=!0;el(t)}else l=!1;if(Qn(t,i),t.stateNode===null)Ri(e,t),wp(t,n,r),Jo(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=ot(c):(c=Be(n)?vn:_e.current,c=Gn(t,c));var d=n.getDerivedStateFromProps,p=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";p||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==c)&&_u(t,o,r,c),$t=!1;var m=t.memoizedState;o.state=m,ll(t,r,o,i),s=t.memoizedState,a!==r||m!==s||Fe.current||$t?(typeof d=="function"&&(Go(t,n,d,r),s=t.memoizedState),(a=$t||Nu(t,n,a,r,m,s,c))?(p||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,ep(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:ct(t.type,a),o.props=c,p=t.pendingProps,m=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=ot(s):(s=Be(n)?vn:_e.current,s=Gn(t,s));var f=n.getDerivedStateFromProps;(d=typeof f=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==p||m!==s)&&_u(t,o,r,s),$t=!1,m=t.memoizedState,o.state=m,ll(t,r,o,i);var k=t.memoizedState;a!==p||m!==k||Fe.current||$t?(typeof f=="function"&&(Go(t,n,f,r),k=t.memoizedState),(c=$t||Nu(t,n,c,r,m,k,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,k,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,k,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=k),o.props=r,o.state=k,o.context=s,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=1024),r=!1)}return ta(e,t,n,r,l,i)}function ta(e,t,n,r,i,l){Ep(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&xu(t,n,!1),Dt(e,t,l);r=t.stateNode,Hm.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=Zn(t,e.child,null,l),t.child=Zn(t,null,a,l)):Pe(e,t,a,l),t.memoizedState=r.state,i&&xu(t,n,!0),t.child}function Np(e){var t=e.stateNode;t.pendingContext?yu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&yu(e,t.context,!1),Za(e,t.containerInfo)}function Au(e,t,n,r,i){return Jn(),Ka(i),t.flags|=256,Pe(e,t,n,r),t.child}var na={dehydrated:null,treeContext:null,retryLane:0};function ra(e){return{baseLanes:e,cachePool:null,transitions:null}}function _p(e,t,n){var r=t.pendingProps,i=ce.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),le(ce,i&1),e===null)return Yo(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Nl(o,r,0,null),e=gn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=ra(n),t.memoizedState=na,e):as(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return Vm(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=en(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=en(a,l):(l=gn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?ra(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=na,r}return l=e.child,e=l.sibling,r=en(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function as(e,t){return t=Nl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function wi(e,t,n,r){return r!==null&&Ka(r),Zn(t,e.child,null,n),e=as(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function Vm(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=io(Error(I(422))),wi(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Nl({mode:"visible",children:r.children},i,0,null),l=gn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&Zn(t,e.child,null,o),t.child.memoizedState=ra(o),t.memoizedState=na,l);if(!(t.mode&1))return wi(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(I(419)),r=io(l,r,void 0),wi(e,t,o,r)}if(a=(o&e.childLanes)!==0,Oe||a){if(r=we,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,At(e,i),ht(r,e,i,-1))}return fs(),r=io(Error(I(421))),wi(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=rg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,qe=Xt(i.nextSibling),Xe=t,ue=!0,pt=null,e!==null&&(tt[nt++]=Tt,tt[nt++]=Lt,tt[nt++]=yn,Tt=e.id,Lt=e.overflow,yn=t),t=as(t,r.children),t.flags|=4096,t)}function Du(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),Xo(e.return,t,n)}function lo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function zp(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(Pe(e,t,r.children,n),r=ce.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Du(e,n,t);else if(e.tag===19)Du(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(le(ce,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&ol(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),lo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&ol(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}lo(t,!0,n,null,l);break;case"together":lo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Ri(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Dt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),kn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(I(153));if(t.child!==null){for(e=t.child,n=en(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=en(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function Wm(e,t,n){switch(t.tag){case 3:Np(t),Jn();break;case 5:tp(t);break;case 1:Be(t.type)&&el(t);break;case 4:Za(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;le(rl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(le(ce,ce.current&1),t.flags|=128,null):n&t.child.childLanes?_p(e,t,n):(le(ce,ce.current&1),e=Dt(e,t,n),e!==null?e.sibling:null);le(ce,ce.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return zp(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),le(ce,ce.current),r)break;return null;case 22:case 23:return t.lanes=0,Cp(e,t,n)}return Dt(e,t,n)}var Tp,ia,Lp,Pp;Tp=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};ia=function(){};Lp=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,hn(St.current);var l=null;switch(n){case"input":i=Eo(e,i),r=Eo(e,r),l=[];break;case"select":i=pe({},i,{value:void 0}),r=pe({},r,{value:void 0}),l=[];break;case"textarea":i=zo(e,i),r=zo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=Ji)}Lo(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(Dr.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var s=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&s!==a&&(s!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(c,n)),n=s;else c==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(c,s)):c==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(c,""+s):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(Dr.hasOwnProperty(c)?(s!=null&&c==="onScroll"&&ae("scroll",e),l||a===s||(l=[])):(l=l||[]).push(c,s))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};Pp=function(e,t,n,r){n!==r&&(t.flags|=4)};function gr(e,t){if(!ue)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function Ee(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function Qm(e,t,n){var r=t.pendingProps;switch(Qa(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return Ee(t),null;case 1:return Be(t.type)&&Zi(),Ee(t),null;case 3:return r=t.stateNode,er(),se(Fe),se(_e),ts(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(xi(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,pt!==null&&(pa(pt),pt=null))),ia(e,t),Ee(t),null;case 5:es(t);var i=hn(qr.current);if(n=t.type,e!==null&&t.stateNode!=null)Lp(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(I(166));return Ee(t),null}if(e=hn(St.current),xi(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[xt]=t,r[Qr]=l,e=(t.mode&1)!==0,n){case"dialog":ae("cancel",r),ae("close",r);break;case"iframe":case"object":case"embed":ae("load",r);break;case"video":case"audio":for(i=0;i<br.length;i++)ae(br[i],r);break;case"source":ae("error",r);break;case"img":case"image":case"link":ae("error",r),ae("load",r);break;case"details":ae("toggle",r);break;case"input":Ws(r,l),ae("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},ae("invalid",r);break;case"textarea":Ks(r,l),ae("invalid",r)}Lo(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&yi(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&yi(r.textContent,a,e),i=["children",""+a]):Dr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&ae("scroll",r)}switch(n){case"input":ci(r),Qs(r,l,!0);break;case"textarea":ci(r),qs(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=Ji)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=od(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[xt]=t,e[Qr]=r,Tp(e,t,!1,!1),t.stateNode=e;e:{switch(o=Po(n,r),n){case"dialog":ae("cancel",e),ae("close",e),i=r;break;case"iframe":case"object":case"embed":ae("load",e),i=r;break;case"video":case"audio":for(i=0;i<br.length;i++)ae(br[i],e);i=r;break;case"source":ae("error",e),i=r;break;case"img":case"image":case"link":ae("error",e),ae("load",e),i=r;break;case"details":ae("toggle",e),i=r;break;case"input":Ws(e,r),i=Eo(e,r),ae("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=pe({},r,{value:void 0}),ae("invalid",e);break;case"textarea":Ks(e,r),i=zo(e,r),ae("invalid",e);break;default:i=r}Lo(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?ud(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&ad(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Rr(e,s):typeof s=="number"&&Rr(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Dr.hasOwnProperty(l)?s!=null&&l==="onScroll"&&ae("scroll",e):s!=null&&La(e,l,s,o))}switch(n){case"input":ci(e),Qs(e,r,!1);break;case"textarea":ci(e),qs(e);break;case"option":r.value!=null&&e.setAttribute("value",""+tn(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?$n(e,!!r.multiple,l,!1):r.defaultValue!=null&&$n(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=Ji)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return Ee(t),null;case 6:if(e&&t.stateNode!=null)Pp(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(I(166));if(n=hn(qr.current),hn(St.current),xi(t)){if(r=t.stateNode,n=t.memoizedProps,r[xt]=t,(l=r.nodeValue!==n)&&(e=Xe,e!==null))switch(e.tag){case 3:yi(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&yi(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[xt]=t,t.stateNode=r}return Ee(t),null;case 13:if(se(ce),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(ue&&qe!==null&&t.mode&1&&!(t.flags&128))Xd(),Jn(),t.flags|=98560,l=!1;else if(l=xi(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(I(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(I(317));l[xt]=t}else Jn(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;Ee(t),l=!1}else pt!==null&&(pa(pt),pt=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||ce.current&1?xe===0&&(xe=3):fs())),t.updateQueue!==null&&(t.flags|=4),Ee(t),null);case 4:return er(),ia(e,t),e===null&&Vr(t.stateNode.containerInfo),Ee(t),null;case 10:return Xa(t.type._context),Ee(t),null;case 17:return Be(t.type)&&Zi(),Ee(t),null;case 19:if(se(ce),l=t.memoizedState,l===null)return Ee(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)gr(l,!1);else{if(xe!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=ol(e),o!==null){for(t.flags|=128,gr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return le(ce,ce.current&1|2),t.child}e=e.sibling}l.tail!==null&&he()>nr&&(t.flags|=128,r=!0,gr(l,!1),t.lanes=4194304)}else{if(!r)if(e=ol(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),gr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!ue)return Ee(t),null}else 2*he()-l.renderingStartTime>nr&&n!==1073741824&&(t.flags|=128,r=!0,gr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=he(),t.sibling=null,n=ce.current,le(ce,r?n&1|2:n&1),t):(Ee(t),null);case 22:case 23:return ps(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?Qe&1073741824&&(Ee(t),t.subtreeFlags&6&&(t.flags|=8192)):Ee(t),null;case 24:return null;case 25:return null}throw Error(I(156,t.tag))}function Km(e,t){switch(Qa(t),t.tag){case 1:return Be(t.type)&&Zi(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return er(),se(Fe),se(_e),ts(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return es(t),null;case 13:if(se(ce),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(I(340));Jn()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return se(ce),null;case 4:return er(),null;case 10:return Xa(t.type._context),null;case 22:case 23:return ps(),null;case 24:return null;default:return null}}var Si=!1,Ne=!1,qm=typeof WeakSet=="function"?WeakSet:Set,$=null;function Bn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){fe(e,t,r)}else n.current=null}function la(e,t,n){try{n()}catch(r){fe(e,t,r)}}var Ru=!1;function Ym(e,t){if($o=Yi,e=Rd(),Va(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,c=0,d=0,p=e,m=null;t:for(;;){for(var f;p!==n||i!==0&&p.nodeType!==3||(a=o+i),p!==l||r!==0&&p.nodeType!==3||(s=o+r),p.nodeType===3&&(o+=p.nodeValue.length),(f=p.firstChild)!==null;)m=p,p=f;for(;;){if(p===e)break t;if(m===n&&++c===i&&(a=o),m===l&&++d===r&&(s=o),(f=p.nextSibling)!==null)break;p=m,m=p.parentNode}p=f}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Ho={focusedElem:e,selectionRange:n},Yi=!1,$=t;$!==null;)if(t=$,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,$=e;else for(;$!==null;){t=$;try{var k=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(k!==null){var w=k.memoizedProps,P=k.memoizedState,h=t.stateNode,v=h.getSnapshotBeforeUpdate(t.elementType===t.type?w:ct(t.type,w),P);h.__reactInternalSnapshotBeforeUpdate=v}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(I(163))}}catch(b){fe(t,t.return,b)}if(e=t.sibling,e!==null){e.return=t.return,$=e;break}$=t.return}return k=Ru,Ru=!1,k}function Tr(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&la(t,n,l)}i=i.next}while(i!==r)}}function Cl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function oa(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Ip(e){var t=e.alternate;t!==null&&(e.alternate=null,Ip(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[xt],delete t[Qr],delete t[Qo],delete t[Lm],delete t[Pm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Mp(e){return e.tag===5||e.tag===3||e.tag===4}function Ou(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Mp(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function aa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=Ji));else if(r!==4&&(e=e.child,e!==null))for(aa(e,t,n),e=e.sibling;e!==null;)aa(e,t,n),e=e.sibling}function sa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(sa(e,t,n),e=e.sibling;e!==null;)sa(e,t,n),e=e.sibling}var Se=null,dt=!1;function Ft(e,t,n){for(n=n.child;n!==null;)Ap(e,t,n),n=n.sibling}function Ap(e,t,n){if(wt&&typeof wt.onCommitFiberUnmount=="function")try{wt.onCommitFiberUnmount(vl,n)}catch{}switch(n.tag){case 5:Ne||Bn(n,t);case 6:var r=Se,i=dt;Se=null,Ft(e,t,n),Se=r,dt=i,Se!==null&&(dt?(e=Se,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):Se.removeChild(n.stateNode));break;case 18:Se!==null&&(dt?(e=Se,n=n.stateNode,e.nodeType===8?Jl(e.parentNode,n):e.nodeType===1&&Jl(e,n),Ur(e)):Jl(Se,n.stateNode));break;case 4:r=Se,i=dt,Se=n.stateNode.containerInfo,dt=!0,Ft(e,t,n),Se=r,dt=i;break;case 0:case 11:case 14:case 15:if(!Ne&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&la(n,t,o),i=i.next}while(i!==r)}Ft(e,t,n);break;case 1:if(!Ne&&(Bn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){fe(n,t,a)}Ft(e,t,n);break;case 21:Ft(e,t,n);break;case 22:n.mode&1?(Ne=(r=Ne)||n.memoizedState!==null,Ft(e,t,n),Ne=r):Ft(e,t,n);break;default:Ft(e,t,n)}}function Fu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new qm),t.forEach(function(r){var i=ig.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function ut(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:Se=a.stateNode,dt=!1;break e;case 3:Se=a.stateNode.containerInfo,dt=!0;break e;case 4:Se=a.stateNode.containerInfo,dt=!0;break e}a=a.return}if(Se===null)throw Error(I(160));Ap(l,o,i),Se=null,dt=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(c){fe(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Dp(t,e),t=t.sibling}function Dp(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(ut(t,e),gt(e),r&4){try{Tr(3,e,e.return),Cl(3,e)}catch(w){fe(e,e.return,w)}try{Tr(5,e,e.return)}catch(w){fe(e,e.return,w)}}break;case 1:ut(t,e),gt(e),r&512&&n!==null&&Bn(n,n.return);break;case 5:if(ut(t,e),gt(e),r&512&&n!==null&&Bn(n,n.return),e.flags&32){var i=e.stateNode;try{Rr(i,"")}catch(w){fe(e,e.return,w)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&id(i,l),Po(a,o);var c=Po(a,l);for(o=0;o<s.length;o+=2){var d=s[o],p=s[o+1];d==="style"?ud(i,p):d==="dangerouslySetInnerHTML"?ad(i,p):d==="children"?Rr(i,p):La(i,d,p,c)}switch(a){case"input":No(i,l);break;case"textarea":ld(i,l);break;case"select":var m=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var f=l.value;f!=null?$n(i,!!l.multiple,f,!1):m!==!!l.multiple&&(l.defaultValue!=null?$n(i,!!l.multiple,l.defaultValue,!0):$n(i,!!l.multiple,l.multiple?[]:"",!1))}i[Qr]=l}catch(w){fe(e,e.return,w)}}break;case 6:if(ut(t,e),gt(e),r&4){if(e.stateNode===null)throw Error(I(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(w){fe(e,e.return,w)}}break;case 3:if(ut(t,e),gt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Ur(t.containerInfo)}catch(w){fe(e,e.return,w)}break;case 4:ut(t,e),gt(e);break;case 13:ut(t,e),gt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(cs=he())),r&4&&Fu(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Ne=(c=Ne)||d,ut(t,e),Ne=c):ut(t,e),gt(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for($=e,d=e.child;d!==null;){for(p=$=d;$!==null;){switch(m=$,f=m.child,m.tag){case 0:case 11:case 14:case 15:Tr(4,m,m.return);break;case 1:Bn(m,m.return);var k=m.stateNode;if(typeof k.componentWillUnmount=="function"){r=m,n=m.return;try{t=r,k.props=t.memoizedProps,k.state=t.memoizedState,k.componentWillUnmount()}catch(w){fe(r,n,w)}}break;case 5:Bn(m,m.return);break;case 22:if(m.memoizedState!==null){Uu(p);continue}}f!==null?(f.return=m,$=f):Uu(p)}d=d.sibling}e:for(d=null,p=e;;){if(p.tag===5){if(d===null){d=p;try{i=p.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=p.stateNode,s=p.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=sd("display",o))}catch(w){fe(e,e.return,w)}}}else if(p.tag===6){if(d===null)try{p.stateNode.nodeValue=c?"":p.memoizedProps}catch(w){fe(e,e.return,w)}}else if((p.tag!==22&&p.tag!==23||p.memoizedState===null||p===e)&&p.child!==null){p.child.return=p,p=p.child;continue}if(p===e)break e;for(;p.sibling===null;){if(p.return===null||p.return===e)break e;d===p&&(d=null),p=p.return}d===p&&(d=null),p.sibling.return=p.return,p=p.sibling}}break;case 19:ut(t,e),gt(e),r&4&&Fu(e);break;case 21:break;default:ut(t,e),gt(e)}}function gt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Mp(n)){var r=n;break e}n=n.return}throw Error(I(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Rr(i,""),r.flags&=-33);var l=Ou(e);sa(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Ou(e);aa(e,a,o);break;default:throw Error(I(161))}}catch(s){fe(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function Xm(e,t,n){$=e,Rp(e)}function Rp(e,t,n){for(var r=(e.mode&1)!==0;$!==null;){var i=$,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||Si;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||Ne;a=Si;var c=Ne;if(Si=o,(Ne=s)&&!c)for($=i;$!==null;)o=$,s=o.child,o.tag===22&&o.memoizedState!==null?$u(i):s!==null?(s.return=o,$=s):$u(i);for(;l!==null;)$=l,Rp(l),l=l.sibling;$=i,Si=a,Ne=c}Bu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,$=l):Bu(e)}}function Bu(e){for(;$!==null;){var t=$;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Ne||Cl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Ne)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:ct(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&ju(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}ju(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var p=d.dehydrated;p!==null&&Ur(p)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(I(163))}Ne||t.flags&512&&oa(t)}catch(m){fe(t,t.return,m)}}if(t===e){$=null;break}if(n=t.sibling,n!==null){n.return=t.return,$=n;break}$=t.return}}function Uu(e){for(;$!==null;){var t=$;if(t===e){$=null;break}var n=t.sibling;if(n!==null){n.return=t.return,$=n;break}$=t.return}}function $u(e){for(;$!==null;){var t=$;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Cl(4,t)}catch(s){fe(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){fe(t,i,s)}}var l=t.return;try{oa(t)}catch(s){fe(t,l,s)}break;case 5:var o=t.return;try{oa(t)}catch(s){fe(t,o,s)}}}catch(s){fe(t,t.return,s)}if(t===e){$=null;break}var a=t.sibling;if(a!==null){a.return=t.return,$=a;break}$=t.return}}var Gm=Math.ceil,ul=Rt.ReactCurrentDispatcher,ss=Rt.ReactCurrentOwner,lt=Rt.ReactCurrentBatchConfig,J=0,we=null,ve=null,be=0,Qe=0,Un=ln(0),xe=0,Jr=null,kn=0,El=0,us=0,Lr=null,Re=null,cs=0,nr=1/0,Nt=null,cl=!1,ua=null,Jt=null,bi=!1,Qt=null,dl=0,Pr=0,ca=null,Oi=-1,Fi=0;function Ie(){return J&6?he():Oi!==-1?Oi:Oi=he()}function Zt(e){return e.mode&1?J&2&&be!==0?be&-be:Mm.transition!==null?(Fi===0&&(Fi=wd()),Fi):(e=ee,e!==0||(e=window.event,e=e===void 0?16:_d(e.type)),e):1}function ht(e,t,n,r){if(50<Pr)throw Pr=0,ca=null,Error(I(185));ti(e,n,r),(!(J&2)||e!==we)&&(e===we&&(!(J&2)&&(El|=n),xe===4&&Vt(e,be)),Ue(e,r),n===1&&J===0&&!(t.mode&1)&&(nr=he()+500,Sl&&on()))}function Ue(e,t){var n=e.callbackNode;Mh(e,t);var r=qi(e,e===we?be:0);if(r===0)n!==null&&Gs(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&Gs(n),t===1)e.tag===0?Im(Hu.bind(null,e)):Kd(Hu.bind(null,e)),zm(function(){!(J&6)&&on()}),n=null;else{switch(Sd(r)){case 1:n=Da;break;case 4:n=xd;break;case 16:n=Ki;break;case 536870912:n=kd;break;default:n=Ki}n=Wp(n,Op.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function Op(e,t){if(Oi=-1,Fi=0,J&6)throw Error(I(327));var n=e.callbackNode;if(Kn()&&e.callbackNode!==n)return null;var r=qi(e,e===we?be:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=pl(e,r);else{t=r;var i=J;J|=2;var l=Bp();(we!==e||be!==t)&&(Nt=null,nr=he()+500,mn(e,t));do try{eg();break}catch(a){Fp(e,a)}while(!0);Ya(),ul.current=l,J=i,ve!==null?t=0:(we=null,be=0,t=xe)}if(t!==0){if(t===2&&(i=Ro(e),i!==0&&(r=i,t=da(e,i))),t===1)throw n=Jr,mn(e,0),Vt(e,r),Ue(e,he()),n;if(t===6)Vt(e,r);else{if(i=e.current.alternate,!(r&30)&&!Jm(i)&&(t=pl(e,r),t===2&&(l=Ro(e),l!==0&&(r=l,t=da(e,l))),t===1))throw n=Jr,mn(e,0),Vt(e,r),Ue(e,he()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(I(345));case 2:cn(e,Re,Nt);break;case 3:if(Vt(e,r),(r&130023424)===r&&(t=cs+500-he(),10<t)){if(qi(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Ie(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=Wo(cn.bind(null,e,Re,Nt),t);break}cn(e,Re,Nt);break;case 4:if(Vt(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-ft(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=he()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*Gm(r/1960))-r,10<r){e.timeoutHandle=Wo(cn.bind(null,e,Re,Nt),r);break}cn(e,Re,Nt);break;case 5:cn(e,Re,Nt);break;default:throw Error(I(329))}}}return Ue(e,he()),e.callbackNode===n?Op.bind(null,e):null}function da(e,t){var n=Lr;return e.current.memoizedState.isDehydrated&&(mn(e,t).flags|=256),e=pl(e,t),e!==2&&(t=Re,Re=n,t!==null&&pa(t)),e}function pa(e){Re===null?Re=e:Re.push.apply(Re,e)}function Jm(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!mt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Vt(e,t){for(t&=~us,t&=~El,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-ft(t),r=1<<n;e[n]=-1,t&=~r}}function Hu(e){if(J&6)throw Error(I(327));Kn();var t=qi(e,0);if(!(t&1))return Ue(e,he()),null;var n=pl(e,t);if(e.tag!==0&&n===2){var r=Ro(e);r!==0&&(t=r,n=da(e,r))}if(n===1)throw n=Jr,mn(e,0),Vt(e,t),Ue(e,he()),n;if(n===6)throw Error(I(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,cn(e,Re,Nt),Ue(e,he()),null}function ds(e,t){var n=J;J|=1;try{return e(t)}finally{J=n,J===0&&(nr=he()+500,Sl&&on())}}function wn(e){Qt!==null&&Qt.tag===0&&!(J&6)&&Kn();var t=J;J|=1;var n=lt.transition,r=ee;try{if(lt.transition=null,ee=1,e)return e()}finally{ee=r,lt.transition=n,J=t,!(J&6)&&on()}}function ps(){Qe=Un.current,se(Un)}function mn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,_m(n)),ve!==null)for(n=ve.return;n!==null;){var r=n;switch(Qa(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&Zi();break;case 3:er(),se(Fe),se(_e),ts();break;case 5:es(r);break;case 4:er();break;case 13:se(ce);break;case 19:se(ce);break;case 10:Xa(r.type._context);break;case 22:case 23:ps()}n=n.return}if(we=e,ve=e=en(e.current,null),be=Qe=t,xe=0,Jr=null,us=El=kn=0,Re=Lr=null,fn!==null){for(t=0;t<fn.length;t++)if(n=fn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}fn=null}return e}function Fp(e,t){do{var n=ve;try{if(Ya(),Ai.current=sl,al){for(var r=de.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}al=!1}if(xn=0,ke=ye=de=null,zr=!1,Yr=0,ss.current=null,n===null||n.return===null){xe=1,Jr=t,ve=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=be,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var c=s,d=a,p=d.tag;if(!(d.mode&1)&&(p===0||p===11||p===15)){var m=d.alternate;m?(d.updateQueue=m.updateQueue,d.memoizedState=m.memoizedState,d.lanes=m.lanes):(d.updateQueue=null,d.memoizedState=null)}var f=Tu(o);if(f!==null){f.flags&=-257,Lu(f,o,a,l,t),f.mode&1&&zu(l,c,t),t=f,s=c;var k=t.updateQueue;if(k===null){var w=new Set;w.add(s),t.updateQueue=w}else k.add(s);break e}else{if(!(t&1)){zu(l,c,t),fs();break e}s=Error(I(426))}}else if(ue&&a.mode&1){var P=Tu(o);if(P!==null){!(P.flags&65536)&&(P.flags|=256),Lu(P,o,a,l,t),Ka(tr(s,a));break e}}l=s=tr(s,a),xe!==4&&(xe=2),Lr===null?Lr=[l]:Lr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var h=Sp(l,s,t);bu(l,h);break e;case 1:a=s;var v=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof v.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(Jt===null||!Jt.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var b=bp(l,a,t);bu(l,b);break e}}l=l.return}while(l!==null)}$p(n)}catch(_){t=_,ve===n&&n!==null&&(ve=n=n.return);continue}break}while(!0)}function Bp(){var e=ul.current;return ul.current=sl,e===null?sl:e}function fs(){(xe===0||xe===3||xe===2)&&(xe=4),we===null||!(kn&268435455)&&!(El&268435455)||Vt(we,be)}function pl(e,t){var n=J;J|=2;var r=Bp();(we!==e||be!==t)&&(Nt=null,mn(e,t));do try{Zm();break}catch(i){Fp(e,i)}while(!0);if(Ya(),J=n,ul.current=r,ve!==null)throw Error(I(261));return we=null,be=0,xe}function Zm(){for(;ve!==null;)Up(ve)}function eg(){for(;ve!==null&&!Ch();)Up(ve)}function Up(e){var t=Vp(e.alternate,e,Qe);e.memoizedProps=e.pendingProps,t===null?$p(e):ve=t,ss.current=null}function $p(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=Km(n,t),n!==null){n.flags&=32767,ve=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{xe=6,ve=null;return}}else if(n=Qm(n,t,Qe),n!==null){ve=n;return}if(t=t.sibling,t!==null){ve=t;return}ve=t=e}while(t!==null);xe===0&&(xe=5)}function cn(e,t,n){var r=ee,i=lt.transition;try{lt.transition=null,ee=1,tg(e,t,n,r)}finally{lt.transition=i,ee=r}return null}function tg(e,t,n,r){do Kn();while(Qt!==null);if(J&6)throw Error(I(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(I(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Ah(e,l),e===we&&(ve=we=null,be=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||bi||(bi=!0,Wp(Ki,function(){return Kn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=lt.transition,lt.transition=null;var o=ee;ee=1;var a=J;J|=4,ss.current=null,Ym(e,n),Dp(n,e),wm(Ho),Yi=!!$o,Ho=$o=null,e.current=n,Xm(n),Eh(),J=a,ee=o,lt.transition=l}else e.current=n;if(bi&&(bi=!1,Qt=e,dl=i),l=e.pendingLanes,l===0&&(Jt=null),zh(n.stateNode),Ue(e,he()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(cl)throw cl=!1,e=ua,ua=null,e;return dl&1&&e.tag!==0&&Kn(),l=e.pendingLanes,l&1?e===ca?Pr++:(Pr=0,ca=e):Pr=0,on(),null}function Kn(){if(Qt!==null){var e=Sd(dl),t=lt.transition,n=ee;try{if(lt.transition=null,ee=16>e?16:e,Qt===null)var r=!1;else{if(e=Qt,Qt=null,dl=0,J&6)throw Error(I(331));var i=J;for(J|=4,$=e.current;$!==null;){var l=$,o=l.child;if($.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var c=a[s];for($=c;$!==null;){var d=$;switch(d.tag){case 0:case 11:case 15:Tr(8,d,l)}var p=d.child;if(p!==null)p.return=d,$=p;else for(;$!==null;){d=$;var m=d.sibling,f=d.return;if(Ip(d),d===c){$=null;break}if(m!==null){m.return=f,$=m;break}$=f}}}var k=l.alternate;if(k!==null){var w=k.child;if(w!==null){k.child=null;do{var P=w.sibling;w.sibling=null,w=P}while(w!==null)}}$=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,$=o;else e:for(;$!==null;){if(l=$,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Tr(9,l,l.return)}var h=l.sibling;if(h!==null){h.return=l.return,$=h;break e}$=l.return}}var v=e.current;for($=v;$!==null;){o=$;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,$=y;else e:for(o=v;$!==null;){if(a=$,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Cl(9,a)}}catch(_){fe(a,a.return,_)}if(a===o){$=null;break e}var b=a.sibling;if(b!==null){b.return=a.return,$=b;break e}$=a.return}}if(J=i,on(),wt&&typeof wt.onPostCommitFiberRoot=="function")try{wt.onPostCommitFiberRoot(vl,e)}catch{}r=!0}return r}finally{ee=n,lt.transition=t}}return!1}function Vu(e,t,n){t=tr(n,t),t=Sp(e,t,1),e=Gt(e,t,1),t=Ie(),e!==null&&(ti(e,1,t),Ue(e,t))}function fe(e,t,n){if(e.tag===3)Vu(e,e,n);else for(;t!==null;){if(t.tag===3){Vu(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(Jt===null||!Jt.has(r))){e=tr(n,e),e=bp(t,e,1),t=Gt(t,e,1),e=Ie(),t!==null&&(ti(t,1,e),Ue(t,e));break}}t=t.return}}function ng(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Ie(),e.pingedLanes|=e.suspendedLanes&n,we===e&&(be&n)===n&&(xe===4||xe===3&&(be&130023424)===be&&500>he()-cs?mn(e,0):us|=n),Ue(e,t)}function Hp(e,t){t===0&&(e.mode&1?(t=fi,fi<<=1,!(fi&130023424)&&(fi=4194304)):t=1);var n=Ie();e=At(e,t),e!==null&&(ti(e,t,n),Ue(e,n))}function rg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),Hp(e,n)}function ig(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(I(314))}r!==null&&r.delete(t),Hp(e,n)}var Vp;Vp=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Fe.current)Oe=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return Oe=!1,Wm(e,t,n);Oe=!!(e.flags&131072)}else Oe=!1,ue&&t.flags&1048576&&qd(t,nl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Ri(e,t),e=t.pendingProps;var i=Gn(t,_e.current);Qn(t,n),i=rs(null,t,r,e,i,n);var l=is();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Be(r)?(l=!0,el(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,Ja(t),i.updater=jl,t.stateNode=i,i._reactInternals=t,Jo(t,r,e,n),t=ta(null,t,r,!0,l,n)):(t.tag=0,ue&&l&&Wa(t),Pe(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Ri(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=og(r),e=ct(r,e),i){case 0:t=ea(null,t,r,e,n);break e;case 1:t=Mu(null,t,r,e,n);break e;case 11:t=Pu(null,t,r,e,n);break e;case 14:t=Iu(null,t,r,ct(r.type,e),n);break e}throw Error(I(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ct(r,i),ea(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ct(r,i),Mu(e,t,r,i,n);case 3:e:{if(Np(t),e===null)throw Error(I(387));r=t.pendingProps,l=t.memoizedState,i=l.element,ep(e,t),ll(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=tr(Error(I(423)),t),t=Au(e,t,r,n,i);break e}else if(r!==i){i=tr(Error(I(424)),t),t=Au(e,t,r,n,i);break e}else for(qe=Xt(t.stateNode.containerInfo.firstChild),Xe=t,ue=!0,pt=null,n=Jd(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(Jn(),r===i){t=Dt(e,t,n);break e}Pe(e,t,r,n)}t=t.child}return t;case 5:return tp(t),e===null&&Yo(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,Vo(r,i)?o=null:l!==null&&Vo(r,l)&&(t.flags|=32),Ep(e,t),Pe(e,t,o,n),t.child;case 6:return e===null&&Yo(t),null;case 13:return _p(e,t,n);case 4:return Za(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=Zn(t,null,r,n):Pe(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ct(r,i),Pu(e,t,r,i,n);case 7:return Pe(e,t,t.pendingProps,n),t.child;case 8:return Pe(e,t,t.pendingProps.children,n),t.child;case 12:return Pe(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,le(rl,r._currentValue),r._currentValue=o,l!==null)if(mt(l.value,o)){if(l.children===i.children&&!Fe.current){t=Dt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Pt(-1,n&-n),s.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?s.next=s:(s.next=d.next,d.next=s),c.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),Xo(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(I(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),Xo(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}Pe(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Qn(t,n),i=ot(i),r=r(i),t.flags|=1,Pe(e,t,r,n),t.child;case 14:return r=t.type,i=ct(r,t.pendingProps),i=ct(r.type,i),Iu(e,t,r,i,n);case 15:return jp(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ct(r,i),Ri(e,t),t.tag=1,Be(r)?(e=!0,el(t)):e=!1,Qn(t,n),wp(t,r,i),Jo(t,r,i,n),ta(null,t,r,!0,e,n);case 19:return zp(e,t,n);case 22:return Cp(e,t,n)}throw Error(I(156,t.tag))};function Wp(e,t){return yd(e,t)}function lg(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function it(e,t,n,r){return new lg(e,t,n,r)}function hs(e){return e=e.prototype,!(!e||!e.isReactComponent)}function og(e){if(typeof e=="function")return hs(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Ia)return 11;if(e===Ma)return 14}return 2}function en(e,t){var n=e.alternate;return n===null?(n=it(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Bi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")hs(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Ln:return gn(n.children,i,l,t);case Pa:o=8,i|=8;break;case So:return e=it(12,n,t,i|2),e.elementType=So,e.lanes=l,e;case bo:return e=it(13,n,t,i),e.elementType=bo,e.lanes=l,e;case jo:return e=it(19,n,t,i),e.elementType=jo,e.lanes=l,e;case td:return Nl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case Zc:o=10;break e;case ed:o=9;break e;case Ia:o=11;break e;case Ma:o=14;break e;case Ut:o=16,r=null;break e}throw Error(I(130,e==null?e:typeof e,""))}return t=it(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function gn(e,t,n,r){return e=it(7,e,r,t),e.lanes=n,e}function Nl(e,t,n,r){return e=it(22,e,r,t),e.elementType=td,e.lanes=n,e.stateNode={isHidden:!1},e}function oo(e,t,n){return e=it(6,e,null,t),e.lanes=n,e}function ao(e,t,n){return t=it(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function ag(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Ul(0),this.expirationTimes=Ul(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Ul(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function ms(e,t,n,r,i,l,o,a,s){return e=new ag(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=it(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},Ja(l),e}function sg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:Tn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function Qp(e){if(!e)return nn;e=e._reactInternals;e:{if(bn(e)!==e||e.tag!==1)throw Error(I(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Be(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(I(171))}if(e.tag===1){var n=e.type;if(Be(n))return Qd(e,n,t)}return t}function Kp(e,t,n,r,i,l,o,a,s){return e=ms(n,r,!0,e,i,l,o,a,s),e.context=Qp(null),n=e.current,r=Ie(),i=Zt(n),l=Pt(r,i),l.callback=t??null,Gt(n,l,i),e.current.lanes=i,ti(e,i,r),Ue(e,r),e}function _l(e,t,n,r){var i=t.current,l=Ie(),o=Zt(i);return n=Qp(n),t.context===null?t.context=n:t.pendingContext=n,t=Pt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=Gt(i,t,o),e!==null&&(ht(e,i,o,l),Mi(e,i,o)),o}function fl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function Wu(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function gs(e,t){Wu(e,t),(e=e.alternate)&&Wu(e,t)}function ug(){return null}var qp=typeof reportError=="function"?reportError:function(e){console.error(e)};function vs(e){this._internalRoot=e}zl.prototype.render=vs.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(I(409));_l(e,t,null,null)};zl.prototype.unmount=vs.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;wn(function(){_l(null,e,null,null)}),t[Mt]=null}};function zl(e){this._internalRoot=e}zl.prototype.unstable_scheduleHydration=function(e){if(e){var t=Cd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Ht.length&&t!==0&&t<Ht[n].priority;n++);Ht.splice(n,0,e),n===0&&Nd(e)}};function ys(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Tl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function Qu(){}function cg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=fl(o);l.call(c)}}var o=Kp(t,r,e,0,null,!1,!1,"",Qu);return e._reactRootContainer=o,e[Mt]=o.current,Vr(e.nodeType===8?e.parentNode:e),wn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=fl(s);a.call(c)}}var s=ms(e,0,!1,null,null,!1,!1,"",Qu);return e._reactRootContainer=s,e[Mt]=s.current,Vr(e.nodeType===8?e.parentNode:e),wn(function(){_l(t,s,n,r)}),s}function Ll(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=fl(o);a.call(s)}}_l(t,o,e,i)}else o=cg(n,t,e,i,r);return fl(o)}bd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=Sr(t.pendingLanes);n!==0&&(Ra(t,n|1),Ue(t,he()),!(J&6)&&(nr=he()+500,on()))}break;case 13:wn(function(){var r=At(e,1);if(r!==null){var i=Ie();ht(r,e,1,i)}}),gs(e,1)}};Oa=function(e){if(e.tag===13){var t=At(e,134217728);if(t!==null){var n=Ie();ht(t,e,134217728,n)}gs(e,134217728)}};jd=function(e){if(e.tag===13){var t=Zt(e),n=At(e,t);if(n!==null){var r=Ie();ht(n,e,t,r)}gs(e,t)}};Cd=function(){return ee};Ed=function(e,t){var n=ee;try{return ee=e,t()}finally{ee=n}};Mo=function(e,t,n){switch(t){case"input":if(No(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=wl(r);if(!i)throw Error(I(90));rd(r),No(r,i)}}}break;case"textarea":ld(e,n);break;case"select":t=n.value,t!=null&&$n(e,!!n.multiple,t,!1)}};pd=ds;fd=wn;var dg={usingClientEntryPoint:!1,Events:[ri,An,wl,cd,dd,ds]},vr={findFiberByHostInstance:pn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},pg={bundleType:vr.bundleType,version:vr.version,rendererPackageName:vr.rendererPackageName,rendererConfig:vr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Rt.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=gd(e),e===null?null:e.stateNode},findFiberByHostInstance:vr.findFiberByHostInstance||ug,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var ji=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!ji.isDisabled&&ji.supportsFiber)try{vl=ji.inject(pg),wt=ji}catch{}}Je.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=dg;Je.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!ys(t))throw Error(I(200));return sg(e,t,null,n)};Je.createRoot=function(e,t){if(!ys(e))throw Error(I(299));var n=!1,r="",i=qp;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=ms(e,1,!1,null,null,n,!1,r,i),e[Mt]=t.current,Vr(e.nodeType===8?e.parentNode:e),new vs(t)};Je.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(I(188)):(e=Object.keys(e).join(","),Error(I(268,e)));return e=gd(t),e=e===null?null:e.stateNode,e};Je.flushSync=function(e){return wn(e)};Je.hydrate=function(e,t,n){if(!Tl(t))throw Error(I(200));return Ll(null,e,t,!0,n)};Je.hydrateRoot=function(e,t,n){if(!ys(e))throw Error(I(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=qp;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=Kp(t,null,e,1,n??null,i,!1,l,o),e[Mt]=t.current,Vr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new zl(t)};Je.render=function(e,t,n){if(!Tl(t))throw Error(I(200));return Ll(null,e,t,!1,n)};Je.unmountComponentAtNode=function(e){if(!Tl(e))throw Error(I(40));return e._reactRootContainer?(wn(function(){Ll(null,null,e,!1,function(){e._reactRootContainer=null,e[Mt]=null})}),!0):!1};Je.unstable_batchedUpdates=ds;Je.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Tl(n))throw Error(I(200));if(e==null||e._reactInternals===void 0)throw Error(I(38));return Ll(e,t,n,!1,r)};Je.version="18.3.1-next-f1338f8080-20240426";function Yp(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Yp)}catch(e){console.error(e)}}Yp(),Yc.exports=Je;var fg=Yc.exports,Ku=fg;ko.createRoot=Ku.createRoot,ko.hydrateRoot=Ku.hydrateRoot;const Et={plus:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),u.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"}),u.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),u.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),u.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),u.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),u.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),u.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("polyline",{points:"3 6 5 6 21 6"}),u.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:u.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},hg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=H.useState(!1),[c,d]=H.useState(""),[p,m]=H.useState(null),[f,k]=H.useState(""),[w,P]=H.useState(null),h=()=>{c.trim()&&(r(c.trim()),d(""),s(!1))},v=j=>{j.key==="Enter"&&!j.shiftKey?(j.preventDefault(),h()):j.key==="Escape"&&(s(!1),d(""))},y=(j,E)=>{E.stopPropagation(),m(j.id),k(j.title)},b=j=>{var E;f.trim()&&f.trim()!==((E=e.find(F=>F.id===j))==null?void 0:E.title)&&l(j,f.trim()),m(null),k("")},_=()=>{m(null),k("")},S=(j,E)=>{j.key==="Enter"?(j.preventDefault(),b(E)):j.key==="Escape"&&_()},L=(j,E)=>{E.stopPropagation(),P(j)},C=(j,E)=>{E.stopPropagation(),i(j),P(null)},T=j=>{j.stopPropagation(),P(null)},R=j=>{const E=new Date(j),V=new Date().getTime()-E.getTime(),B=Math.floor(V/6e4),U=Math.floor(V/36e5),K=Math.floor(V/864e5);return B<1?"now":B<60?`${B}m`:U<24?`${U}h`:K<7?`${K}d`:E.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return u.jsxs("div",{className:"thread-list",children:[u.jsxs("div",{className:"list-header",children:[u.jsx("h2",{children:"Conversations"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:Et.plus})]}),a&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:c,onChange:j=>d(j.target.value),onKeyDown:v,placeholder:"Conversation title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:h,children:"Create"})]})]}),u.jsx("div",{className:"thread-items",children:e.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Et.hash}),u.jsx("p",{children:"No conversations yet"}),u.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(j=>{const E=o.get(j.id)||0,F=j.id===t,V=p===j.id,B=w===j.id;return u.jsxs("div",{className:`thread-item ${F?"selected":""} ${E>0?"has-unread":""}`,onClick:()=>!V&&n(j.id),children:[u.jsx("div",{className:`status-dot ${j.status}`}),u.jsxs("div",{className:"thread-content",children:[u.jsx("div",{className:"thread-title-row",children:V?u.jsxs("div",{className:"edit-title-form",onClick:U=>U.stopPropagation(),children:[u.jsx("input",{type:"text",value:f,onChange:U=>k(U.target.value),onKeyDown:U=>S(U,j.id),autoFocus:!0}),u.jsx("button",{className:"edit-action save",onClick:()=>b(j.id),title:"Save",children:Et.check}),u.jsx("button",{className:"edit-action cancel",onClick:_,title:"Cancel",children:Et.x})]}):u.jsxs(u.Fragment,{children:[u.jsx("span",{className:"thread-title",children:j.title}),u.jsx("span",{className:"thread-time",children:R(j.updated_at)})]})}),u.jsxs("div",{className:"thread-meta",children:[j.target_agent&&u.jsxs("span",{className:"thread-agent",title:`Target: ${j.target_agent}`,children:[Et.bot,j.target_agent]}),u.jsxs("span",{className:"thread-seq",children:["#",j.last_seq]})]})]}),!V&&!B&&u.jsxs("div",{className:"thread-actions",children:[u.jsx("button",{className:"action-btn edit",onClick:U=>y(j,U),title:"Rename",children:Et.edit}),u.jsx("button",{className:"action-btn delete",onClick:U=>L(j.id,U),title:"Delete",children:Et.trash})]}),B&&u.jsxs("div",{className:"delete-confirm",onClick:U=>U.stopPropagation(),children:[u.jsx("span",{className:"confirm-text",children:"Delete?"}),u.jsx("button",{className:"confirm-btn yes",onClick:U=>C(j.id,U),title:"Confirm delete",children:Et.check}),u.jsx("button",{className:"confirm-btn no",onClick:T,title:"Cancel",children:Et.x})]}),E>0&&!B&&u.jsx("span",{className:"unread-badge",children:E})]},j.id)})}),u.jsx("style",{children:`
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
      `})]})};function mg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const gg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,vg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,yg={};function qu(e,t){return(yg.jsx?vg:gg).test(e)}const xg=/[ \t\n\f\r]/g;function kg(e){return typeof e=="object"?e.type==="text"?Yu(e.value):!1:Yu(e)}function Yu(e){return e.replace(xg,"")===""}class li{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}li.prototype.normal={};li.prototype.property={};li.prototype.space=void 0;function Xp(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new li(n,r,t)}function fa(e){return e.toLowerCase()}class He{constructor(t,n){this.attribute=n,this.property=t}}He.prototype.attribute="";He.prototype.booleanish=!1;He.prototype.boolean=!1;He.prototype.commaOrSpaceSeparated=!1;He.prototype.commaSeparated=!1;He.prototype.defined=!1;He.prototype.mustUseProperty=!1;He.prototype.number=!1;He.prototype.overloadedBoolean=!1;He.prototype.property="";He.prototype.spaceSeparated=!1;He.prototype.space=void 0;let wg=0;const Y=jn(),ge=jn(),ha=jn(),M=jn(),ie=jn(),qn=jn(),We=jn();function jn(){return 2**++wg}const ma=Object.freeze(Object.defineProperty({__proto__:null,boolean:Y,booleanish:ge,commaOrSpaceSeparated:We,commaSeparated:qn,number:M,overloadedBoolean:ha,spaceSeparated:ie},Symbol.toStringTag,{value:"Module"})),so=Object.keys(ma);class xs extends He{constructor(t,n,r,i){let l=-1;if(super(t,n),Xu(this,"space",i),typeof r=="number")for(;++l<so.length;){const o=so[l];Xu(this,so[l],(r&ma[o])===ma[o])}}}xs.prototype.defined=!0;function Xu(e,t,n){n&&(e[t]=n)}function or(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new xs(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[fa(r)]=r,n[fa(l.attribute)]=r}return new li(t,n,e.space)}const Gp=or({properties:{ariaActiveDescendant:null,ariaAtomic:ge,ariaAutoComplete:null,ariaBusy:ge,ariaChecked:ge,ariaColCount:M,ariaColIndex:M,ariaColSpan:M,ariaControls:ie,ariaCurrent:null,ariaDescribedBy:ie,ariaDetails:null,ariaDisabled:ge,ariaDropEffect:ie,ariaErrorMessage:null,ariaExpanded:ge,ariaFlowTo:ie,ariaGrabbed:ge,ariaHasPopup:null,ariaHidden:ge,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:ie,ariaLevel:M,ariaLive:null,ariaModal:ge,ariaMultiLine:ge,ariaMultiSelectable:ge,ariaOrientation:null,ariaOwns:ie,ariaPlaceholder:null,ariaPosInSet:M,ariaPressed:ge,ariaReadOnly:ge,ariaRelevant:null,ariaRequired:ge,ariaRoleDescription:ie,ariaRowCount:M,ariaRowIndex:M,ariaRowSpan:M,ariaSelected:ge,ariaSetSize:M,ariaSort:null,ariaValueMax:M,ariaValueMin:M,ariaValueNow:M,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function Jp(e,t){return t in e?e[t]:t}function Zp(e,t){return Jp(e,t.toLowerCase())}const Sg=or({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:qn,acceptCharset:ie,accessKey:ie,action:null,allow:null,allowFullScreen:Y,allowPaymentRequest:Y,allowUserMedia:Y,alt:null,as:null,async:Y,autoCapitalize:null,autoComplete:ie,autoFocus:Y,autoPlay:Y,blocking:ie,capture:null,charSet:null,checked:Y,cite:null,className:ie,cols:M,colSpan:null,content:null,contentEditable:ge,controls:Y,controlsList:ie,coords:M|qn,crossOrigin:null,data:null,dateTime:null,decoding:null,default:Y,defer:Y,dir:null,dirName:null,disabled:Y,download:ha,draggable:ge,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:Y,formTarget:null,headers:ie,height:M,hidden:ha,high:M,href:null,hrefLang:null,htmlFor:ie,httpEquiv:ie,id:null,imageSizes:null,imageSrcSet:null,inert:Y,inputMode:null,integrity:null,is:null,isMap:Y,itemId:null,itemProp:ie,itemRef:ie,itemScope:Y,itemType:ie,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:Y,low:M,manifest:null,max:null,maxLength:M,media:null,method:null,min:null,minLength:M,multiple:Y,muted:Y,name:null,nonce:null,noModule:Y,noValidate:Y,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:Y,optimum:M,pattern:null,ping:ie,placeholder:null,playsInline:Y,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:Y,referrerPolicy:null,rel:ie,required:Y,reversed:Y,rows:M,rowSpan:M,sandbox:ie,scope:null,scoped:Y,seamless:Y,selected:Y,shadowRootClonable:Y,shadowRootDelegatesFocus:Y,shadowRootMode:null,shape:null,size:M,sizes:null,slot:null,span:M,spellCheck:ge,src:null,srcDoc:null,srcLang:null,srcSet:null,start:M,step:null,style:null,tabIndex:M,target:null,title:null,translate:null,type:null,typeMustMatch:Y,useMap:null,value:ge,width:M,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:ie,axis:null,background:null,bgColor:null,border:M,borderColor:null,bottomMargin:M,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:Y,declare:Y,event:null,face:null,frame:null,frameBorder:null,hSpace:M,leftMargin:M,link:null,longDesc:null,lowSrc:null,marginHeight:M,marginWidth:M,noResize:Y,noHref:Y,noShade:Y,noWrap:Y,object:null,profile:null,prompt:null,rev:null,rightMargin:M,rules:null,scheme:null,scrolling:ge,standby:null,summary:null,text:null,topMargin:M,valueType:null,version:null,vAlign:null,vLink:null,vSpace:M,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:Y,disableRemotePlayback:Y,prefix:null,property:null,results:M,security:null,unselectable:null},space:"html",transform:Zp}),bg=or({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:We,accentHeight:M,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:M,amplitude:M,arabicForm:null,ascent:M,attributeName:null,attributeType:null,azimuth:M,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:M,by:null,calcMode:null,capHeight:M,className:ie,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:M,diffuseConstant:M,direction:null,display:null,dur:null,divisor:M,dominantBaseline:null,download:Y,dx:null,dy:null,edgeMode:null,editable:null,elevation:M,enableBackground:null,end:null,event:null,exponent:M,externalResourcesRequired:null,fill:null,fillOpacity:M,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:qn,g2:qn,glyphName:qn,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:M,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:M,horizOriginX:M,horizOriginY:M,id:null,ideographic:M,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:M,k:M,k1:M,k2:M,k3:M,k4:M,kernelMatrix:We,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:M,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:M,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:M,overlineThickness:M,paintOrder:null,panose1:null,path:null,pathLength:M,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:ie,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:M,pointsAtY:M,pointsAtZ:M,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:We,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:We,rev:We,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:We,requiredFeatures:We,requiredFonts:We,requiredFormats:We,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:M,specularExponent:M,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:M,strikethroughThickness:M,string:null,stroke:null,strokeDashArray:We,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:M,strokeOpacity:M,strokeWidth:null,style:null,surfaceScale:M,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:We,tabIndex:M,tableValues:null,target:null,targetX:M,targetY:M,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:We,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:M,underlineThickness:M,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:M,values:null,vAlphabetic:M,vMathematical:M,vectorEffect:null,vHanging:M,vIdeographic:M,version:null,vertAdvY:M,vertOriginX:M,vertOriginY:M,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:M,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:Jp}),ef=or({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),tf=or({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:Zp}),nf=or({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),jg={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Cg=/[A-Z]/g,Gu=/-[a-z]/g,Eg=/^data[-\w.:]+$/i;function Ng(e,t){const n=fa(t);let r=t,i=He;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&Eg.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(Gu,zg);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!Gu.test(l)){let o=l.replace(Cg,_g);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=xs}return new i(r,t)}function _g(e){return"-"+e.toLowerCase()}function zg(e){return e.charAt(1).toUpperCase()}const Tg=Xp([Gp,Sg,ef,tf,nf],"html"),ks=Xp([Gp,bg,ef,tf,nf],"svg");function Lg(e){return e.join(" ").trim()}var ws={},Ju=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Pg=/\n/g,Ig=/^\s*/,Mg=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Ag=/^:\s*/,Dg=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Rg=/^[;\s]*/,Og=/^\s+|\s+$/g,Fg=`
`,Zu="/",ec="*",dn="",Bg="comment",Ug="declaration";function $g(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(k){var w=k.match(Pg);w&&(n+=w.length);var P=k.lastIndexOf(Fg);r=~P?k.length-P:r+k.length}function l(){var k={line:n,column:r};return function(w){return w.position=new o(k),c(),w}}function o(k){this.start=k,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(k){var w=new Error(t.source+":"+n+":"+r+": "+k);if(w.reason=k,w.filename=t.source,w.line=n,w.column=r,w.source=e,!t.silent)throw w}function s(k){var w=k.exec(e);if(w){var P=w[0];return i(P),e=e.slice(P.length),w}}function c(){s(Ig)}function d(k){var w;for(k=k||[];w=p();)w!==!1&&k.push(w);return k}function p(){var k=l();if(!(Zu!=e.charAt(0)||ec!=e.charAt(1))){for(var w=2;dn!=e.charAt(w)&&(ec!=e.charAt(w)||Zu!=e.charAt(w+1));)++w;if(w+=2,dn===e.charAt(w-1))return a("End of comment missing");var P=e.slice(2,w-2);return r+=2,i(P),e=e.slice(w),r+=2,k({type:Bg,comment:P})}}function m(){var k=l(),w=s(Mg);if(w){if(p(),!s(Ag))return a("property missing ':'");var P=s(Dg),h=k({type:Ug,property:tc(w[0].replace(Ju,dn)),value:P?tc(P[0].replace(Ju,dn)):dn});return s(Rg),h}}function f(){var k=[];d(k);for(var w;w=m();)w!==!1&&(k.push(w),d(k));return k}return c(),f()}function tc(e){return e?e.replace(Og,dn):dn}var Hg=$g,Vg=Hi&&Hi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(ws,"__esModule",{value:!0});ws.default=Qg;const Wg=Vg(Hg);function Qg(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Wg.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Pl={};Object.defineProperty(Pl,"__esModule",{value:!0});Pl.camelCase=void 0;var Kg=/^--[a-zA-Z0-9_-]+$/,qg=/-([a-z])/g,Yg=/^[^-]+$/,Xg=/^-(webkit|moz|ms|o|khtml)-/,Gg=/^-(ms)-/,Jg=function(e){return!e||Yg.test(e)||Kg.test(e)},Zg=function(e,t){return t.toUpperCase()},nc=function(e,t){return"".concat(t,"-")},ev=function(e,t){return t===void 0&&(t={}),Jg(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(Gg,nc):e=e.replace(Xg,nc),e.replace(qg,Zg))};Pl.camelCase=ev;var tv=Hi&&Hi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},nv=tv(ws),rv=Pl;function ga(e,t){var n={};return!e||typeof e!="string"||(0,nv.default)(e,function(r,i){r&&i&&(n[(0,rv.camelCase)(r,t)]=i)}),n}ga.default=ga;var iv=ga;const lv=ja(iv),rf=lf("end"),Ss=lf("start");function lf(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function ov(e){const t=Ss(e),n=rf(e);if(t&&n)return{start:t,end:n}}function Ir(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?rc(e.position):"start"in e||"end"in e?rc(e):"line"in e||"column"in e?va(e):""}function va(e){return ic(e&&e.line)+":"+ic(e&&e.column)}function rc(e){return va(e&&e.start)+"-"+va(e&&e.end)}function ic(e){return e&&typeof e=="number"?e:1}class ze extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Ir(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}ze.prototype.file="";ze.prototype.name="";ze.prototype.reason="";ze.prototype.message="";ze.prototype.stack="";ze.prototype.column=void 0;ze.prototype.line=void 0;ze.prototype.ancestors=void 0;ze.prototype.cause=void 0;ze.prototype.fatal=void 0;ze.prototype.place=void 0;ze.prototype.ruleId=void 0;ze.prototype.source=void 0;const bs={}.hasOwnProperty,av=new Map,sv=/[A-Z]/g,uv=new Set(["table","tbody","thead","tfoot","tr"]),cv=new Set(["td","th"]),of="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function dv(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=xv(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=yv(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?ks:Tg,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=af(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function af(e,t,n){if(t.type==="element")return pv(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return fv(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return mv(e,t,n);if(t.type==="mdxjsEsm")return hv(e,t);if(t.type==="root")return gv(e,t,n);if(t.type==="text")return vv(e,t)}function pv(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=ks,e.schema=i),e.ancestors.push(t);const l=uf(e,t.tagName,!1),o=kv(e,t);let a=Cs(e,t);return uv.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!kg(s):!0})),sf(e,o,l,t),js(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function fv(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}Zr(e,t.position)}function hv(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);Zr(e,t.position)}function mv(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=ks,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:uf(e,t.name,!0),o=wv(e,t),a=Cs(e,t);return sf(e,o,l,t),js(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function gv(e,t,n){const r={};return js(r,Cs(e,t)),e.create(t,e.Fragment,r,n)}function vv(e,t){return t.value}function sf(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function js(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function yv(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function xv(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=Ss(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function kv(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&bs.call(t.properties,i)){const l=Sv(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&cv.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function wv(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else Zr(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else Zr(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Cs(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:av;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const c=i.get(s)||0;o=s+"-"+c,i.set(s,c+1)}}const a=af(e,l,o);a!==void 0&&n.push(a)}return n}function Sv(e,t,n){const r=Ng(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?mg(n):Lg(n)),r.property==="style"){let i=typeof n=="object"?n:bv(e,String(n));return e.stylePropertyNameCase==="css"&&(i=jv(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?jg[r.property]||r.property:r.attribute,n]}}function bv(e,t){try{return lv(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new ze("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=of+"#cannot-parse-style-attribute",i}}function uf(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=qu(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=qu(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return bs.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);Zr(e)}function Zr(e,t){const n=new ze("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=of+"#cannot-handle-mdx-estrees-without-createevaluater",n}function jv(e){const t={};let n;for(n in e)bs.call(e,n)&&(t[Cv(n)]=e[n]);return t}function Cv(e){let t=e.replace(sv,Ev);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function Ev(e){return"-"+e.toLowerCase()}const uo={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},Nv={};function _v(e,t){const n=Nv,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return cf(e,r,i)}function cf(e,t,n){if(zv(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return lc(e.children,t,n)}return Array.isArray(e)?lc(e,t,n):""}function lc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=cf(e[i],t,n);return r.join("")}function zv(e){return!!(e&&typeof e=="object")}const oc=document.createElement("i");function Es(e){const t="&"+e+";";oc.innerHTML=t;const n=oc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function bt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function rt(e,t){return e.length>0?(bt(e,e.length,0,t),e):t}const ac={}.hasOwnProperty;function Tv(e){const t={};let n=-1;for(;++n<e.length;)Lv(t,e[n]);return t}function Lv(e,t){let n;for(n in t){const i=(ac.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){ac.call(i,o)||(i[o]=[]);const a=l[o];Pv(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Pv(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);bt(e,0,0,r)}function df(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function Yn(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const kt=an(/[A-Za-z]/),Ye=an(/[\dA-Za-z]/),Iv=an(/[#-'*+\--9=?A-Z^-~]/);function ya(e){return e!==null&&(e<32||e===127)}const xa=an(/\d/),Mv=an(/[\dA-Fa-f]/),Av=an(/[!-/:-@[-`{-~]/);function Q(e){return e!==null&&e<-2}function $e(e){return e!==null&&(e<0||e===32)}function Z(e){return e===-2||e===-1||e===32}const Dv=an(new RegExp("\\p{P}|\\p{S}","u")),Rv=an(/\s/);function an(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function ar(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&Ye(e.charCodeAt(n+1))&&Ye(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function oe(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return Z(s)?(e.enter(n),a(s)):t(s)}function a(s){return Z(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const Ov={tokenize:Fv};function Fv(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),oe(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return Q(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const Bv={tokenize:Uv},sc={tokenize:$v};function Uv(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const b=n[r];return t.containerState=b[1],e.attempt(b[0].continuation,s,c)(y)}return c(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&v();const b=t.events.length;let _=b,S;for(;_--;)if(t.events[_][0]==="exit"&&t.events[_][1].type==="chunkFlow"){S=t.events[_][1].end;break}h(r);let L=b;for(;L<t.events.length;)t.events[L][1].end={...S},L++;return bt(t.events,_+1,0,t.events.slice(b)),t.events.length=L,c(y)}return a(y)}function c(y){if(r===n.length){if(!i)return m(y);if(i.currentConstruct&&i.currentConstruct.concrete)return k(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(sc,d,p)(y)}function d(y){return i&&v(),h(r),m(y)}function p(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,k(y)}function m(y){return t.containerState={},e.attempt(sc,f,k)(y)}function f(y){return r++,n.push([t.currentConstruct,t.containerState]),m(y)}function k(y){if(y===null){i&&v(),h(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),w(y)}function w(y){if(y===null){P(e.exit("chunkFlow"),!0),h(0),e.consume(y);return}return Q(y)?(e.consume(y),P(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),w)}function P(y,b){const _=t.sliceStream(y);if(b&&_.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(_),t.parser.lazy[y.start.line]){let S=i.events.length;for(;S--;)if(i.events[S][1].start.offset<o&&(!i.events[S][1].end||i.events[S][1].end.offset>o))return;const L=t.events.length;let C=L,T,R;for(;C--;)if(t.events[C][0]==="exit"&&t.events[C][1].type==="chunkFlow"){if(T){R=t.events[C][1].end;break}T=!0}for(h(r),S=L;S<t.events.length;)t.events[S][1].end={...R},S++;bt(t.events,C+1,0,t.events.slice(L)),t.events.length=S}}function h(y){let b=n.length;for(;b-- >y;){const _=n[b];t.containerState=_[1],_[0].exit.call(t,e)}n.length=y}function v(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function $v(e,t,n){return oe(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function uc(e){if(e===null||$e(e)||Rv(e))return 1;if(Dv(e))return 2}function Ns(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const ka={name:"attention",resolveAll:Hv,tokenize:Vv};function Hv(e,t){let n=-1,r,i,l,o,a,s,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const p={...e[r][1].end},m={...e[n][1].start};cc(p,-s),cc(m,s),o={type:s>1?"strongSequence":"emphasisSequence",start:p,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:m},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=rt(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=rt(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=rt(c,Ns(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=rt(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=rt(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,bt(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Vv(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=uc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const c=e.exit("attentionSequence"),d=uc(s),p=!d||d===2&&i||n.includes(s),m=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?p:p&&(i||!m)),c._close=!!(l===42?m:m&&(d||!p)),t(s)}}function cc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Wv={name:"autolink",tokenize:Qv};function Qv(e,t,n){let r=0;return i;function i(f){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(f){return kt(f)?(e.consume(f),o):f===64?n(f):c(f)}function o(f){return f===43||f===45||f===46||Ye(f)?(r=1,a(f)):c(f)}function a(f){return f===58?(e.consume(f),r=0,s):(f===43||f===45||f===46||Ye(f))&&r++<32?(e.consume(f),a):(r=0,c(f))}function s(f){return f===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.exit("autolink"),t):f===null||f===32||f===60||ya(f)?n(f):(e.consume(f),s)}function c(f){return f===64?(e.consume(f),d):Iv(f)?(e.consume(f),c):n(f)}function d(f){return Ye(f)?p(f):n(f)}function p(f){return f===46?(e.consume(f),r=0,d):f===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.exit("autolink"),t):m(f)}function m(f){if((f===45||Ye(f))&&r++<63){const k=f===45?m:p;return e.consume(f),k}return n(f)}}const Il={partial:!0,tokenize:Kv};function Kv(e,t,n){return r;function r(l){return Z(l)?oe(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||Q(l)?t(l):n(l)}}const pf={continuation:{tokenize:Yv},exit:Xv,name:"blockQuote",tokenize:qv};function qv(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return Z(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function Yv(e,t,n){const r=this;return i;function i(o){return Z(o)?oe(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(pf,t,n)(o)}}function Xv(e){e.exit("blockQuote")}const ff={name:"characterEscape",tokenize:Gv};function Gv(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return Av(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const hf={name:"characterReference",tokenize:Jv};function Jv(e,t,n){const r=this;let i=0,l,o;return a;function a(p){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(p),e.exit("characterReferenceMarker"),s}function s(p){return p===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(p),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=Ye,d(p))}function c(p){return p===88||p===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(p),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=Mv,d):(e.enter("characterReferenceValue"),l=7,o=xa,d(p))}function d(p){if(p===59&&i){const m=e.exit("characterReferenceValue");return o===Ye&&!Es(r.sliceSerialize(m))?n(p):(e.enter("characterReferenceMarker"),e.consume(p),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(p)&&i++<l?(e.consume(p),d):n(p)}}const dc={partial:!0,tokenize:ey},pc={concrete:!0,name:"codeFenced",tokenize:Zv};function Zv(e,t,n){const r=this,i={partial:!0,tokenize:_};let l=0,o=0,a;return s;function s(S){return c(S)}function c(S){const L=r.events[r.events.length-1];return l=L&&L[1].type==="linePrefix"?L[2].sliceSerialize(L[1],!0).length:0,a=S,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(S)}function d(S){return S===a?(o++,e.consume(S),d):o<3?n(S):(e.exit("codeFencedFenceSequence"),Z(S)?oe(e,p,"whitespace")(S):p(S))}function p(S){return S===null||Q(S)?(e.exit("codeFencedFence"),r.interrupt?t(S):e.check(dc,w,b)(S)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),m(S))}function m(S){return S===null||Q(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),p(S)):Z(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),oe(e,f,"whitespace")(S)):S===96&&S===a?n(S):(e.consume(S),m)}function f(S){return S===null||Q(S)?p(S):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),k(S))}function k(S){return S===null||Q(S)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),p(S)):S===96&&S===a?n(S):(e.consume(S),k)}function w(S){return e.attempt(i,b,P)(S)}function P(S){return e.enter("lineEnding"),e.consume(S),e.exit("lineEnding"),h}function h(S){return l>0&&Z(S)?oe(e,v,"linePrefix",l+1)(S):v(S)}function v(S){return S===null||Q(S)?e.check(dc,w,b)(S):(e.enter("codeFlowValue"),y(S))}function y(S){return S===null||Q(S)?(e.exit("codeFlowValue"),v(S)):(e.consume(S),y)}function b(S){return e.exit("codeFenced"),t(S)}function _(S,L,C){let T=0;return R;function R(B){return S.enter("lineEnding"),S.consume(B),S.exit("lineEnding"),j}function j(B){return S.enter("codeFencedFence"),Z(B)?oe(S,E,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(B):E(B)}function E(B){return B===a?(S.enter("codeFencedFenceSequence"),F(B)):C(B)}function F(B){return B===a?(T++,S.consume(B),F):T>=o?(S.exit("codeFencedFenceSequence"),Z(B)?oe(S,V,"whitespace")(B):V(B)):C(B)}function V(B){return B===null||Q(B)?(S.exit("codeFencedFence"),L(B)):C(B)}}}function ey(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const co={name:"codeIndented",tokenize:ny},ty={partial:!0,tokenize:ry};function ny(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),oe(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?s(c):Q(c)?e.attempt(ty,o,s)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||Q(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function s(c){return e.exit("codeIndented"),t(c)}}function ry(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):Q(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):oe(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):Q(o)?i(o):n(o)}}const iy={name:"codeText",previous:oy,resolve:ly,tokenize:ay};function ly(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function oy(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function ay(e,t,n){let r=0,i,l;return o;function o(p){return e.enter("codeText"),e.enter("codeTextSequence"),a(p)}function a(p){return p===96?(e.consume(p),r++,a):(e.exit("codeTextSequence"),s(p))}function s(p){return p===null?n(p):p===32?(e.enter("space"),e.consume(p),e.exit("space"),s):p===96?(l=e.enter("codeTextSequence"),i=0,d(p)):Q(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),s):(e.enter("codeTextData"),c(p))}function c(p){return p===null||p===32||p===96||Q(p)?(e.exit("codeTextData"),s(p)):(e.consume(p),c)}function d(p){return p===96?(e.consume(p),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(p)):(l.type="codeTextData",c(p))}}class sy{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&yr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),yr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),yr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);yr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);yr(this.left,n.reverse())}}}function yr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function mf(e){const t={};let n=-1,r,i,l,o,a,s,c;const d=new sy(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,uy(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return bt(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function uy(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],c={};let d,p,m=-1,f=n,k=0,w=0;const P=[w];for(;f;){for(;e.get(++i)[1]!==f;);l.push(i),f._tokenizer||(d=r.sliceStream(f),f.next||d.push(null),p&&o.defineSkip(f.start),f._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),f._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),p=f,f=f.next}for(f=n;++m<a.length;)a[m][0]==="exit"&&a[m-1][0]==="enter"&&a[m][1].type===a[m-1][1].type&&a[m][1].start.line!==a[m][1].end.line&&(w=m+1,P.push(w),f._tokenizer=void 0,f.previous=void 0,f=f.next);for(o.events=[],f?(f._tokenizer=void 0,f.previous=void 0):P.pop(),m=P.length;m--;){const h=a.slice(P[m],P[m+1]),v=l.pop();s.push([v,v+h.length-1]),e.splice(v,2,h)}for(s.reverse(),m=-1;++m<s.length;)c[k+s[m][0]]=k+s[m][1],k+=s[m][1]-s[m][0]-1;return c}const cy={resolve:py,tokenize:fy},dy={partial:!0,tokenize:hy};function py(e){return mf(e),e}function fy(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):Q(a)?e.check(dy,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function hy(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),oe(e,l,"linePrefix")}function l(o){if(o===null||Q(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function gf(e,t,n,r,i,l,o,a,s){const c=s||Number.POSITIVE_INFINITY;let d=0;return p;function p(h){return h===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(h),e.exit(l),m):h===null||h===32||h===41||ya(h)?n(h):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),w(h))}function m(h){return h===62?(e.enter(l),e.consume(h),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),f(h))}function f(h){return h===62?(e.exit("chunkString"),e.exit(a),m(h)):h===null||h===60||Q(h)?n(h):(e.consume(h),h===92?k:f)}function k(h){return h===60||h===62||h===92?(e.consume(h),f):f(h)}function w(h){return!d&&(h===null||h===41||$e(h))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(h)):d<c&&h===40?(e.consume(h),d++,w):h===41?(e.consume(h),d--,w):h===null||h===32||h===40||ya(h)?n(h):(e.consume(h),h===92?P:w)}function P(h){return h===40||h===41||h===92?(e.consume(h),w):w(h)}}function vf(e,t,n,r,i,l){const o=this;let a=0,s;return c;function c(f){return e.enter(r),e.enter(i),e.consume(f),e.exit(i),e.enter(l),d}function d(f){return a>999||f===null||f===91||f===93&&!s||f===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(f):f===93?(e.exit(l),e.enter(i),e.consume(f),e.exit(i),e.exit(r),t):Q(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),p(f))}function p(f){return f===null||f===91||f===93||Q(f)||a++>999?(e.exit("chunkString"),d(f)):(e.consume(f),s||(s=!Z(f)),f===92?m:p)}function m(f){return f===91||f===92||f===93?(e.consume(f),a++,p):p(f)}}function yf(e,t,n,r,i,l){let o;return a;function a(m){return m===34||m===39||m===40?(e.enter(r),e.enter(i),e.consume(m),e.exit(i),o=m===40?41:m,s):n(m)}function s(m){return m===o?(e.enter(i),e.consume(m),e.exit(i),e.exit(r),t):(e.enter(l),c(m))}function c(m){return m===o?(e.exit(l),s(o)):m===null?n(m):Q(m)?(e.enter("lineEnding"),e.consume(m),e.exit("lineEnding"),oe(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(m))}function d(m){return m===o||m===null||Q(m)?(e.exit("chunkString"),c(m)):(e.consume(m),m===92?p:d)}function p(m){return m===o||m===92?(e.consume(m),d):d(m)}}function Mr(e,t){let n;return r;function r(i){return Q(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):Z(i)?oe(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const my={name:"definition",tokenize:vy},gy={partial:!0,tokenize:yy};function vy(e,t,n){const r=this;let i;return l;function l(f){return e.enter("definition"),o(f)}function o(f){return vf.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(f)}function a(f){return i=Yn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),f===58?(e.enter("definitionMarker"),e.consume(f),e.exit("definitionMarker"),s):n(f)}function s(f){return $e(f)?Mr(e,c)(f):c(f)}function c(f){return gf(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(f)}function d(f){return e.attempt(gy,p,p)(f)}function p(f){return Z(f)?oe(e,m,"whitespace")(f):m(f)}function m(f){return f===null||Q(f)?(e.exit("definition"),r.parser.defined.push(i),t(f)):n(f)}}function yy(e,t,n){return r;function r(a){return $e(a)?Mr(e,i)(a):n(a)}function i(a){return yf(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return Z(a)?oe(e,o,"whitespace")(a):o(a)}function o(a){return a===null||Q(a)?t(a):n(a)}}const xy={name:"hardBreakEscape",tokenize:ky};function ky(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return Q(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const wy={name:"headingAtx",resolve:Sy,tokenize:by};function Sy(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},bt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function by(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||$e(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||Q(d)?(e.exit("atxHeading"),t(d)):Z(d)?oe(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||$e(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const jy=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],fc=["pre","script","style","textarea"],Cy={concrete:!0,name:"htmlFlow",resolveTo:_y,tokenize:zy},Ey={partial:!0,tokenize:Ly},Ny={partial:!0,tokenize:Ty};function _y(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function zy(e,t,n){const r=this;let i,l,o,a,s;return c;function c(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),p}function p(x){return x===33?(e.consume(x),m):x===47?(e.consume(x),l=!0,w):x===63?(e.consume(x),i=3,r.interrupt?t:g):kt(x)?(e.consume(x),o=String.fromCharCode(x),P):n(x)}function m(x){return x===45?(e.consume(x),i=2,f):x===91?(e.consume(x),i=5,a=0,k):kt(x)?(e.consume(x),i=4,r.interrupt?t:g):n(x)}function f(x){return x===45?(e.consume(x),r.interrupt?t:g):n(x)}function k(x){const ne="CDATA[";return x===ne.charCodeAt(a++)?(e.consume(x),a===ne.length?r.interrupt?t:E:k):n(x)}function w(x){return kt(x)?(e.consume(x),o=String.fromCharCode(x),P):n(x)}function P(x){if(x===null||x===47||x===62||$e(x)){const ne=x===47,Te=o.toLowerCase();return!ne&&!l&&fc.includes(Te)?(i=1,r.interrupt?t(x):E(x)):jy.includes(o.toLowerCase())?(i=6,ne?(e.consume(x),h):r.interrupt?t(x):E(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?v(x):y(x))}return x===45||Ye(x)?(e.consume(x),o+=String.fromCharCode(x),P):n(x)}function h(x){return x===62?(e.consume(x),r.interrupt?t:E):n(x)}function v(x){return Z(x)?(e.consume(x),v):R(x)}function y(x){return x===47?(e.consume(x),R):x===58||x===95||kt(x)?(e.consume(x),b):Z(x)?(e.consume(x),y):R(x)}function b(x){return x===45||x===46||x===58||x===95||Ye(x)?(e.consume(x),b):_(x)}function _(x){return x===61?(e.consume(x),S):Z(x)?(e.consume(x),_):y(x)}function S(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,L):Z(x)?(e.consume(x),S):C(x)}function L(x){return x===s?(e.consume(x),s=null,T):x===null||Q(x)?n(x):(e.consume(x),L)}function C(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||$e(x)?_(x):(e.consume(x),C)}function T(x){return x===47||x===62||Z(x)?y(x):n(x)}function R(x){return x===62?(e.consume(x),j):n(x)}function j(x){return x===null||Q(x)?E(x):Z(x)?(e.consume(x),j):n(x)}function E(x){return x===45&&i===2?(e.consume(x),U):x===60&&i===1?(e.consume(x),K):x===62&&i===4?(e.consume(x),D):x===63&&i===3?(e.consume(x),g):x===93&&i===5?(e.consume(x),N):Q(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(Ey,W,F)(x)):x===null||Q(x)?(e.exit("htmlFlowData"),F(x)):(e.consume(x),E)}function F(x){return e.check(Ny,V,W)(x)}function V(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),B}function B(x){return x===null||Q(x)?F(x):(e.enter("htmlFlowData"),E(x))}function U(x){return x===45?(e.consume(x),g):E(x)}function K(x){return x===47?(e.consume(x),o="",A):E(x)}function A(x){if(x===62){const ne=o.toLowerCase();return fc.includes(ne)?(e.consume(x),D):E(x)}return kt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),A):E(x)}function N(x){return x===93?(e.consume(x),g):E(x)}function g(x){return x===62?(e.consume(x),D):x===45&&i===2?(e.consume(x),g):E(x)}function D(x){return x===null||Q(x)?(e.exit("htmlFlowData"),W(x)):(e.consume(x),D)}function W(x){return e.exit("htmlFlow"),t(x)}}function Ty(e,t,n){const r=this;return i;function i(o){return Q(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Ly(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Il,t,n)}}const Py={name:"htmlText",tokenize:Iy};function Iy(e,t,n){const r=this;let i,l,o;return a;function a(g){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(g),s}function s(g){return g===33?(e.consume(g),c):g===47?(e.consume(g),_):g===63?(e.consume(g),y):kt(g)?(e.consume(g),C):n(g)}function c(g){return g===45?(e.consume(g),d):g===91?(e.consume(g),l=0,k):kt(g)?(e.consume(g),v):n(g)}function d(g){return g===45?(e.consume(g),f):n(g)}function p(g){return g===null?n(g):g===45?(e.consume(g),m):Q(g)?(o=p,K(g)):(e.consume(g),p)}function m(g){return g===45?(e.consume(g),f):p(g)}function f(g){return g===62?U(g):g===45?m(g):p(g)}function k(g){const D="CDATA[";return g===D.charCodeAt(l++)?(e.consume(g),l===D.length?w:k):n(g)}function w(g){return g===null?n(g):g===93?(e.consume(g),P):Q(g)?(o=w,K(g)):(e.consume(g),w)}function P(g){return g===93?(e.consume(g),h):w(g)}function h(g){return g===62?U(g):g===93?(e.consume(g),h):w(g)}function v(g){return g===null||g===62?U(g):Q(g)?(o=v,K(g)):(e.consume(g),v)}function y(g){return g===null?n(g):g===63?(e.consume(g),b):Q(g)?(o=y,K(g)):(e.consume(g),y)}function b(g){return g===62?U(g):y(g)}function _(g){return kt(g)?(e.consume(g),S):n(g)}function S(g){return g===45||Ye(g)?(e.consume(g),S):L(g)}function L(g){return Q(g)?(o=L,K(g)):Z(g)?(e.consume(g),L):U(g)}function C(g){return g===45||Ye(g)?(e.consume(g),C):g===47||g===62||$e(g)?T(g):n(g)}function T(g){return g===47?(e.consume(g),U):g===58||g===95||kt(g)?(e.consume(g),R):Q(g)?(o=T,K(g)):Z(g)?(e.consume(g),T):U(g)}function R(g){return g===45||g===46||g===58||g===95||Ye(g)?(e.consume(g),R):j(g)}function j(g){return g===61?(e.consume(g),E):Q(g)?(o=j,K(g)):Z(g)?(e.consume(g),j):T(g)}function E(g){return g===null||g===60||g===61||g===62||g===96?n(g):g===34||g===39?(e.consume(g),i=g,F):Q(g)?(o=E,K(g)):Z(g)?(e.consume(g),E):(e.consume(g),V)}function F(g){return g===i?(e.consume(g),i=void 0,B):g===null?n(g):Q(g)?(o=F,K(g)):(e.consume(g),F)}function V(g){return g===null||g===34||g===39||g===60||g===61||g===96?n(g):g===47||g===62||$e(g)?T(g):(e.consume(g),V)}function B(g){return g===47||g===62||$e(g)?T(g):n(g)}function U(g){return g===62?(e.consume(g),e.exit("htmlTextData"),e.exit("htmlText"),t):n(g)}function K(g){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),A}function A(g){return Z(g)?oe(e,N,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(g):N(g)}function N(g){return e.enter("htmlTextData"),o(g)}}const _s={name:"labelEnd",resolveAll:Ry,resolveTo:Oy,tokenize:Fy},My={tokenize:By},Ay={tokenize:Uy},Dy={tokenize:$y};function Ry(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&bt(e,0,e.length,n),e}function Oy(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",c,t]],a=rt(a,e.slice(l+1,l+r+3)),a=rt(a,[["enter",d,t]]),a=rt(a,Ns(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=rt(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=rt(a,e.slice(o+1)),a=rt(a,[["exit",s,t]]),bt(e,l,e.length,a),e}function Fy(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(m){return l?l._inactive?p(m):(o=r.parser.defined.includes(Yn(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(m),e.exit("labelMarker"),e.exit("labelEnd"),s):n(m)}function s(m){return m===40?e.attempt(My,d,o?d:p)(m):m===91?e.attempt(Ay,d,o?c:p)(m):o?d(m):p(m)}function c(m){return e.attempt(Dy,d,p)(m)}function d(m){return t(m)}function p(m){return l._balanced=!0,n(m)}}function By(e,t,n){return r;function r(p){return e.enter("resource"),e.enter("resourceMarker"),e.consume(p),e.exit("resourceMarker"),i}function i(p){return $e(p)?Mr(e,l)(p):l(p)}function l(p){return p===41?d(p):gf(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(p)}function o(p){return $e(p)?Mr(e,s)(p):d(p)}function a(p){return n(p)}function s(p){return p===34||p===39||p===40?yf(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(p):d(p)}function c(p){return $e(p)?Mr(e,d)(p):d(p)}function d(p){return p===41?(e.enter("resourceMarker"),e.consume(p),e.exit("resourceMarker"),e.exit("resource"),t):n(p)}}function Uy(e,t,n){const r=this;return i;function i(a){return vf.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(Yn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function $y(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Hy={name:"labelStartImage",resolveAll:_s.resolveAll,tokenize:Vy};function Vy(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Wy={name:"labelStartLink",resolveAll:_s.resolveAll,tokenize:Qy};function Qy(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const po={name:"lineEnding",tokenize:Ky};function Ky(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),oe(e,t,"linePrefix")}}const Ui={name:"thematicBreak",tokenize:qy};function qy(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),s(c)):r>=3&&(c===null||Q(c))?(e.exit("thematicBreak"),t(c)):n(c)}function s(c){return c===i?(e.consume(c),r++,s):(e.exit("thematicBreakSequence"),Z(c)?oe(e,a,"whitespace")(c):a(c))}}const De={continuation:{tokenize:Jy},exit:ex,name:"list",tokenize:Gy},Yy={partial:!0,tokenize:tx},Xy={partial:!0,tokenize:Zy};function Gy(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(f){const k=r.containerState.type||(f===42||f===43||f===45?"listUnordered":"listOrdered");if(k==="listUnordered"?!r.containerState.marker||f===r.containerState.marker:xa(f)){if(r.containerState.type||(r.containerState.type=k,e.enter(k,{_container:!0})),k==="listUnordered")return e.enter("listItemPrefix"),f===42||f===45?e.check(Ui,n,c)(f):c(f);if(!r.interrupt||f===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(f)}return n(f)}function s(f){return xa(f)&&++o<10?(e.consume(f),s):(!r.interrupt||o<2)&&(r.containerState.marker?f===r.containerState.marker:f===41||f===46)?(e.exit("listItemValue"),c(f)):n(f)}function c(f){return e.enter("listItemMarker"),e.consume(f),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||f,e.check(Il,r.interrupt?n:d,e.attempt(Yy,m,p))}function d(f){return r.containerState.initialBlankLine=!0,l++,m(f)}function p(f){return Z(f)?(e.enter("listItemPrefixWhitespace"),e.consume(f),e.exit("listItemPrefixWhitespace"),m):n(f)}function m(f){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(f)}}function Jy(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Il,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,oe(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!Z(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Xy,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,oe(e,e.attempt(De,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function Zy(e,t,n){const r=this;return oe(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function ex(e){e.exit(this.containerState.type)}function tx(e,t,n){const r=this;return oe(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!Z(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const hc={name:"setextUnderline",resolveTo:nx,tokenize:rx};function nx(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function rx(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,p;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){p=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||p)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),Z(c)?oe(e,s,"lineSuffix")(c):s(c))}function s(c){return c===null||Q(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const ix={tokenize:lx};function lx(e){const t=this,n=e.attempt(Il,r,e.attempt(this.parser.constructs.flowInitial,i,oe(e,e.attempt(this.parser.constructs.flow,i,e.attempt(cy,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const ox={resolveAll:kf()},ax=xf("string"),sx=xf("text");function xf(e){return{resolveAll:kf(e==="text"?ux:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function c(d){if(d===null)return!0;const p=i[d];let m=-1;if(p)for(;++m<p.length;){const f=p[m];if(!f.previous||f.previous.call(r,r.previous))return!0}return!1}}}function kf(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function ux(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)s=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const cx={42:De,43:De,45:De,48:De,49:De,50:De,51:De,52:De,53:De,54:De,55:De,56:De,57:De,62:pf},dx={91:my},px={[-2]:co,[-1]:co,32:co},fx={35:wy,42:Ui,45:[hc,Ui],60:Cy,61:hc,95:Ui,96:pc,126:pc},hx={38:hf,92:ff},mx={[-5]:po,[-4]:po,[-3]:po,33:Hy,38:hf,42:ka,60:[Wv,Py],91:Wy,92:[xy,ff],93:_s,95:ka,96:iy},gx={null:[ka,ox]},vx={null:[42,95]},yx={null:[]},xx=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:vx,contentInitial:dx,disable:yx,document:cx,flow:fx,flowInitial:px,insideSpan:gx,string:hx,text:mx},Symbol.toStringTag,{value:"Module"}));function kx(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:L(_),check:L(S),consume:v,enter:y,exit:b,interrupt:L(S,{interrupt:!0})},c={code:null,containerState:{},defineSkip:w,events:[],now:k,parser:e,previous:null,sliceSerialize:m,sliceStream:f,write:p};let d=t.tokenize.call(c,s);return t.resolveAll&&l.push(t),c;function p(j){return o=rt(o,j),P(),o[o.length-1]!==null?[]:(C(t,0),c.events=Ns(l,c.events,c),c.events)}function m(j,E){return Sx(f(j),E)}function f(j){return wx(o,j)}function k(){const{_bufferIndex:j,_index:E,line:F,column:V,offset:B}=r;return{_bufferIndex:j,_index:E,line:F,column:V,offset:B}}function w(j){i[j.line]=j.column,R()}function P(){let j;for(;r._index<o.length;){const E=o[r._index];if(typeof E=="string")for(j=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===j&&r._bufferIndex<E.length;)h(E.charCodeAt(r._bufferIndex));else h(E)}}function h(j){d=d(j)}function v(j){Q(j)?(r.line++,r.column=1,r.offset+=j===-3?2:1,R()):j!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=j}function y(j,E){const F=E||{};return F.type=j,F.start=k(),c.events.push(["enter",F,c]),a.push(F),F}function b(j){const E=a.pop();return E.end=k(),c.events.push(["exit",E,c]),E}function _(j,E){C(j,E.from)}function S(j,E){E.restore()}function L(j,E){return F;function F(V,B,U){let K,A,N,g;return Array.isArray(V)?W(V):"tokenize"in V?W([V]):D(V);function D(te){return et;function et(Ot){const Cn=Ot!==null&&te[Ot],En=Ot!==null&&te.null,ai=[...Array.isArray(Cn)?Cn:Cn?[Cn]:[],...Array.isArray(En)?En:En?[En]:[]];return W(ai)(Ot)}}function W(te){return K=te,A=0,te.length===0?U:x(te[A])}function x(te){return et;function et(Ot){return g=T(),N=te,te.partial||(c.currentConstruct=te),te.name&&c.parser.constructs.disable.null.includes(te.name)?Te():te.tokenize.call(E?Object.assign(Object.create(c),E):c,s,ne,Te)(Ot)}}function ne(te){return j(N,g),B}function Te(te){return g.restore(),++A<K.length?x(K[A]):U}}}function C(j,E){j.resolveAll&&!l.includes(j)&&l.push(j),j.resolve&&bt(c.events,E,c.events.length-E,j.resolve(c.events.slice(E),c)),j.resolveTo&&(c.events=j.resolveTo(c.events,c))}function T(){const j=k(),E=c.previous,F=c.currentConstruct,V=c.events.length,B=Array.from(a);return{from:V,restore:U};function U(){r=j,c.previous=E,c.currentConstruct=F,c.events.length=V,a=B,R()}}function R(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function wx(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function Sx(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function bx(e){const r={constructs:Tv([xx,...(e||{}).extensions||[]]),content:i(Ov),defined:[],document:i(Bv),flow:i(ix),lazy:{},string:i(ax),text:i(sx)};return r;function i(l){return o;function o(a){return kx(r,l,a)}}}function jx(e){for(;!mf(e););return e}const mc=/[\0\t\n\r]/g;function Cx(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let c,d,p,m,f;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),p=0,t="",n&&(l.charCodeAt(0)===65279&&p++,n=void 0);p<l.length;){if(mc.lastIndex=p,c=mc.exec(l),m=c&&c.index!==void 0?c.index:l.length,f=l.charCodeAt(m),!c){t=l.slice(p);break}if(f===10&&p===m&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),p<m&&(s.push(l.slice(p,m)),e+=m-p),f){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}p=m+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const Ex=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function Nx(e){return e.replace(Ex,_x)}function _x(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return df(n.slice(l?2:1),l?16:10)}return Es(n)||e}const wf={}.hasOwnProperty;function zx(e,t,n){return typeof t!="string"&&(n=t,t=void 0),Tx(n)(jx(bx(n).document().write(Cx()(e,t,!0))))}function Tx(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Rs),autolinkProtocol:T,autolinkEmail:T,atxHeading:l(Ms),blockQuote:l(En),characterEscape:T,characterReference:T,codeFenced:l(ai),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(ai,o),codeText:l(If,o),codeTextData:T,data:T,codeFlowValue:T,definition:l(Mf),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Af),hardBreakEscape:l(As),hardBreakTrailing:l(As),htmlFlow:l(Ds,o),htmlFlowData:T,htmlText:l(Ds,o),htmlTextData:T,image:l(Df),label:o,link:l(Rs),listItem:l(Rf),listItemValue:m,listOrdered:l(Os,p),listUnordered:l(Os),paragraph:l(Of),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Ms),strong:l(Ff),thematicBreak:l(Uf)},exit:{atxHeading:s(),atxHeadingSequence:_,autolink:s(),autolinkEmail:Cn,autolinkProtocol:Ot,blockQuote:s(),characterEscapeValue:R,characterReferenceMarkerHexadecimal:Te,characterReferenceMarkerNumeric:Te,characterReferenceValue:te,characterReference:et,codeFenced:s(P),codeFencedFence:w,codeFencedFenceInfo:f,codeFencedFenceMeta:k,codeFlowValue:R,codeIndented:s(h),codeText:s(B),codeTextData:R,data:R,definition:s(),definitionDestinationString:b,definitionLabelString:v,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(E),hardBreakTrailing:s(E),htmlFlow:s(F),htmlFlowData:R,htmlText:s(V),htmlTextData:R,image:s(K),label:N,labelText:A,lineEnding:j,link:s(U),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:ne,resourceDestinationString:g,resourceTitleString:D,resource:W,setextHeading:s(C),setextHeadingLineSequence:L,setextHeadingText:S,strong:s(),thematicBreak:s()}};Sf(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(z){let O={type:"root",children:[]};const q={stack:[O],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},G=[];let re=-1;for(;++re<z.length;)if(z[re][1].type==="listOrdered"||z[re][1].type==="listUnordered")if(z[re][0]==="enter")G.push(re);else{const st=G.pop();re=i(z,st,re)}for(re=-1;++re<z.length;){const st=t[z[re][0]];wf.call(st,z[re][1].type)&&st[z[re][1].type].call(Object.assign({sliceSerialize:z[re][2].sliceSerialize},q),z[re][1])}if(q.tokenStack.length>0){const st=q.tokenStack[q.tokenStack.length-1];(st[1]||gc).call(q,void 0,st[0])}for(O.position={start:Bt(z.length>0?z[0][1].start:{line:1,column:1,offset:0}),end:Bt(z.length>0?z[z.length-2][1].end:{line:1,column:1,offset:0})},re=-1;++re<t.transforms.length;)O=t.transforms[re](O)||O;return O}function i(z,O,q){let G=O-1,re=-1,st=!1,sn,jt,sr,ur;for(;++G<=q;){const Ve=z[G];switch(Ve[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Ve[0]==="enter"?re++:re--,ur=void 0;break}case"lineEndingBlank":{Ve[0]==="enter"&&(sn&&!ur&&!re&&!sr&&(sr=G),ur=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:ur=void 0}if(!re&&Ve[0]==="enter"&&Ve[1].type==="listItemPrefix"||re===-1&&Ve[0]==="exit"&&(Ve[1].type==="listUnordered"||Ve[1].type==="listOrdered")){if(sn){let Nn=G;for(jt=void 0;Nn--;){const Ct=z[Nn];if(Ct[1].type==="lineEnding"||Ct[1].type==="lineEndingBlank"){if(Ct[0]==="exit")continue;jt&&(z[jt][1].type="lineEndingBlank",st=!0),Ct[1].type="lineEnding",jt=Nn}else if(!(Ct[1].type==="linePrefix"||Ct[1].type==="blockQuotePrefix"||Ct[1].type==="blockQuotePrefixWhitespace"||Ct[1].type==="blockQuoteMarker"||Ct[1].type==="listItemIndent"))break}sr&&(!jt||sr<jt)&&(sn._spread=!0),sn.end=Object.assign({},jt?z[jt][1].start:Ve[1].end),z.splice(jt||G,0,["exit",sn,Ve[2]]),G++,q++}if(Ve[1].type==="listItemPrefix"){const Nn={type:"listItem",_spread:!1,start:Object.assign({},Ve[1].start),end:void 0};sn=Nn,z.splice(G,0,["enter",Nn,Ve[2]]),G++,q++,sr=void 0,ur=!0}}}return z[O][1]._spread=st,q}function l(z,O){return q;function q(G){a.call(this,z(G),G),O&&O.call(this,G)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(z,O,q){this.stack[this.stack.length-1].children.push(z),this.stack.push(z),this.tokenStack.push([O,q||void 0]),z.position={start:Bt(O.start),end:void 0}}function s(z){return O;function O(q){z&&z.call(this,q),c.call(this,q)}}function c(z,O){const q=this.stack.pop(),G=this.tokenStack.pop();if(G)G[0].type!==z.type&&(O?O.call(this,z,G[0]):(G[1]||gc).call(this,z,G[0]));else throw new Error("Cannot close `"+z.type+"` ("+Ir({start:z.start,end:z.end})+"): it’s not open");q.position.end=Bt(z.end)}function d(){return _v(this.stack.pop())}function p(){this.data.expectingFirstListItemValue=!0}function m(z){if(this.data.expectingFirstListItemValue){const O=this.stack[this.stack.length-2];O.start=Number.parseInt(this.sliceSerialize(z),10),this.data.expectingFirstListItemValue=void 0}}function f(){const z=this.resume(),O=this.stack[this.stack.length-1];O.lang=z}function k(){const z=this.resume(),O=this.stack[this.stack.length-1];O.meta=z}function w(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function P(){const z=this.resume(),O=this.stack[this.stack.length-1];O.value=z.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function h(){const z=this.resume(),O=this.stack[this.stack.length-1];O.value=z.replace(/(\r?\n|\r)$/g,"")}function v(z){const O=this.resume(),q=this.stack[this.stack.length-1];q.label=O,q.identifier=Yn(this.sliceSerialize(z)).toLowerCase()}function y(){const z=this.resume(),O=this.stack[this.stack.length-1];O.title=z}function b(){const z=this.resume(),O=this.stack[this.stack.length-1];O.url=z}function _(z){const O=this.stack[this.stack.length-1];if(!O.depth){const q=this.sliceSerialize(z).length;O.depth=q}}function S(){this.data.setextHeadingSlurpLineEnding=!0}function L(z){const O=this.stack[this.stack.length-1];O.depth=this.sliceSerialize(z).codePointAt(0)===61?1:2}function C(){this.data.setextHeadingSlurpLineEnding=void 0}function T(z){const q=this.stack[this.stack.length-1].children;let G=q[q.length-1];(!G||G.type!=="text")&&(G=Bf(),G.position={start:Bt(z.start),end:void 0},q.push(G)),this.stack.push(G)}function R(z){const O=this.stack.pop();O.value+=this.sliceSerialize(z),O.position.end=Bt(z.end)}function j(z){const O=this.stack[this.stack.length-1];if(this.data.atHardBreak){const q=O.children[O.children.length-1];q.position.end=Bt(z.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(O.type)&&(T.call(this,z),R.call(this,z))}function E(){this.data.atHardBreak=!0}function F(){const z=this.resume(),O=this.stack[this.stack.length-1];O.value=z}function V(){const z=this.resume(),O=this.stack[this.stack.length-1];O.value=z}function B(){const z=this.resume(),O=this.stack[this.stack.length-1];O.value=z}function U(){const z=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";z.type+="Reference",z.referenceType=O,delete z.url,delete z.title}else delete z.identifier,delete z.label;this.data.referenceType=void 0}function K(){const z=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";z.type+="Reference",z.referenceType=O,delete z.url,delete z.title}else delete z.identifier,delete z.label;this.data.referenceType=void 0}function A(z){const O=this.sliceSerialize(z),q=this.stack[this.stack.length-2];q.label=Nx(O),q.identifier=Yn(O).toLowerCase()}function N(){const z=this.stack[this.stack.length-1],O=this.resume(),q=this.stack[this.stack.length-1];if(this.data.inReference=!0,q.type==="link"){const G=z.children;q.children=G}else q.alt=O}function g(){const z=this.resume(),O=this.stack[this.stack.length-1];O.url=z}function D(){const z=this.resume(),O=this.stack[this.stack.length-1];O.title=z}function W(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function ne(z){const O=this.resume(),q=this.stack[this.stack.length-1];q.label=O,q.identifier=Yn(this.sliceSerialize(z)).toLowerCase(),this.data.referenceType="full"}function Te(z){this.data.characterReferenceType=z.type}function te(z){const O=this.sliceSerialize(z),q=this.data.characterReferenceType;let G;q?(G=df(O,q==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):G=Es(O);const re=this.stack[this.stack.length-1];re.value+=G}function et(z){const O=this.stack.pop();O.position.end=Bt(z.end)}function Ot(z){R.call(this,z);const O=this.stack[this.stack.length-1];O.url=this.sliceSerialize(z)}function Cn(z){R.call(this,z);const O=this.stack[this.stack.length-1];O.url="mailto:"+this.sliceSerialize(z)}function En(){return{type:"blockquote",children:[]}}function ai(){return{type:"code",lang:null,meta:null,value:""}}function If(){return{type:"inlineCode",value:""}}function Mf(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Af(){return{type:"emphasis",children:[]}}function Ms(){return{type:"heading",depth:0,children:[]}}function As(){return{type:"break"}}function Ds(){return{type:"html",value:""}}function Df(){return{type:"image",title:null,url:"",alt:null}}function Rs(){return{type:"link",title:null,url:"",children:[]}}function Os(z){return{type:"list",ordered:z.type==="listOrdered",start:null,spread:z._spread,children:[]}}function Rf(z){return{type:"listItem",spread:z._spread,checked:null,children:[]}}function Of(){return{type:"paragraph",children:[]}}function Ff(){return{type:"strong",children:[]}}function Bf(){return{type:"text",value:""}}function Uf(){return{type:"thematicBreak"}}}function Bt(e){return{line:e.line,column:e.column,offset:e.offset}}function Sf(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Sf(e,r):Lx(e,r)}}function Lx(e,t){let n;for(n in t)if(wf.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function gc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Ir({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Ir({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Ir({start:t.start,end:t.end})+") is still open")}function Px(e){const t=this;t.parser=n;function n(r){return zx(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function Ix(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function Mx(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function Ax(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function Dx(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Rx(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Ox(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=ar(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const c={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,c),e.applyData(t,c)}function Fx(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Bx(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function bf(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function Ux(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return bf(e,t);const i={src:ar(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function $x(e,t){const n={src:ar(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function Hx(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function Vx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return bf(e,t);const i={href:ar(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function Wx(e,t){const n={href:ar(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function Qx(e,t,n){const r=e.all(t),i=n?Kx(n):jf(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let p;d&&d.type==="element"&&d.tagName==="p"?p=d:(p={type:"element",tagName:"p",properties:{},children:[]},r.unshift(p)),p.children.length>0&&p.children.unshift({type:"text",value:" "}),p.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function Kx(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=jf(n[r])}return t}function jf(e){const t=e.spread;return t??e.children.length>1}function qx(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function Yx(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Xx(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function Gx(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Jx(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=Ss(t.children[1]),s=rf(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function Zx(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const c=[];for(;++s<a;){const p=t.children[s],m={},f=o?o[s]:void 0;f&&(m.align=f);let k={type:"element",tagName:l,properties:m,children:[]};p&&(k.children=e.all(p),e.patch(p,k),k=e.applyData(p,k)),c.push(k)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function e1(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const vc=9,yc=32;function t1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(xc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(xc(t.slice(i),i>0,!1)),l.join("")}function xc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===vc||l===yc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===vc||l===yc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function n1(e,t){const n={type:"text",value:t1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function r1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const i1={blockquote:Ix,break:Mx,code:Ax,delete:Dx,emphasis:Rx,footnoteReference:Ox,heading:Fx,html:Bx,imageReference:Ux,image:$x,inlineCode:Hx,linkReference:Vx,link:Wx,listItem:Qx,list:qx,paragraph:Yx,root:Xx,strong:Gx,table:Jx,tableCell:e1,tableRow:Zx,text:n1,thematicBreak:r1,toml:Ci,yaml:Ci,definition:Ci,footnoteDefinition:Ci};function Ci(){}const Cf=-1,Ml=0,Ar=1,hl=2,zs=3,Ts=4,Ls=5,Ps=6,Ef=7,Nf=8,kc=typeof self=="object"?self:globalThis,l1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Ml:case Cf:return n(o,i);case Ar:{const a=n([],i);for(const s of o)a.push(r(s));return a}case hl:{const a=n({},i);for(const[s,c]of o)a[r(s)]=r(c);return a}case zs:return n(new Date(o),i);case Ts:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Ls:{const a=n(new Map,i);for(const[s,c]of o)a.set(r(s),r(c));return a}case Ps:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Ef:{const{name:a,message:s}=o;return n(new kc[a](s),i)}case Nf:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new kc[l](o),i)};return r},wc=e=>l1(new Map,e)(0),zn="",{toString:o1}={},{keys:a1}=Object,xr=e=>{const t=typeof e;if(t!=="object"||!e)return[Ml,t];const n=o1.call(e).slice(8,-1);switch(n){case"Array":return[Ar,zn];case"Object":return[hl,zn];case"Date":return[zs,zn];case"RegExp":return[Ts,zn];case"Map":return[Ls,zn];case"Set":return[Ps,zn];case"DataView":return[Ar,n]}return n.includes("Array")?[Ar,n]:n.includes("Error")?[Ef,n]:[hl,n]},Ei=([e,t])=>e===Ml&&(t==="function"||t==="symbol"),s1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=xr(o);switch(a){case Ml:{let d=o;switch(s){case"bigint":a=Nf,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([Cf],o)}return i([a,d],o)}case Ar:{if(s){let m=o;return s==="DataView"?m=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(m=new Uint8Array(o)),i([s,[...m]],o)}const d=[],p=i([a,d],o);for(const m of o)d.push(l(m));return p}case hl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],p=i([a,d],o);for(const m of a1(o))(e||!Ei(xr(o[m])))&&d.push([l(m),l(o[m])]);return p}case zs:return i([a,o.toISOString()],o);case Ts:{const{source:d,flags:p}=o;return i([a,{source:d,flags:p}],o)}case Ls:{const d=[],p=i([a,d],o);for(const[m,f]of o)(e||!(Ei(xr(m))||Ei(xr(f))))&&d.push([l(m),l(f)]);return p}case Ps:{const d=[],p=i([a,d],o);for(const m of o)(e||!Ei(xr(m)))&&d.push(l(m));return p}}const{message:c}=o;return i([a,{name:s,message:c}],o)};return l},Sc=(e,{json:t,lossy:n}={})=>{const r=[];return s1(!(t||n),!!t,new Map,r)(e),r},ml=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?wc(Sc(e,t)):structuredClone(e):(e,t)=>wc(Sc(e,t));function u1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function c1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function d1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||u1,r=e.options.footnoteBackLabel||c1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[s]);if(!c)continue;const d=e.all(c),p=String(c.identifier).toUpperCase(),m=ar(p.toLowerCase());let f=0;const k=[],w=e.footnoteCounts.get(p);for(;w!==void 0&&++f<=w;){k.length>0&&k.push({type:"text",value:" "});let v=typeof n=="string"?n:n(s,f);typeof v=="string"&&(v={type:"text",value:v}),k.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+m+(f>1?"-"+f:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,f),className:["data-footnote-backref"]},children:Array.isArray(v)?v:[v]})}const P=d[d.length-1];if(P&&P.type==="element"&&P.tagName==="p"){const v=P.children[P.children.length-1];v&&v.type==="text"?v.value+=" ":P.children.push({type:"text",value:" "}),P.children.push(...k)}else d.push(...k);const h={type:"element",tagName:"li",properties:{id:t+"fn-"+m},children:e.wrap(d,!0)};e.patch(c,h),a.push(h)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...ml(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const _f=function(e){if(e==null)return m1;if(typeof e=="function")return Al(e);if(typeof e=="object")return Array.isArray(e)?p1(e):f1(e);if(typeof e=="string")return h1(e);throw new Error("Expected function, string, or object as test")};function p1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=_f(e[n]);return Al(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function f1(e){const t=e;return Al(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function h1(e){return Al(t);function t(n){return n&&n.type===e}}function Al(e){return t;function t(n,r,i){return!!(g1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function m1(){return!0}function g1(e){return e!==null&&typeof e=="object"&&"type"in e}const zf=[],v1=!0,bc=!1,y1="skip";function x1(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=_f(i),o=r?-1:1;a(e,void 0,[])();function a(s,c,d){const p=s&&typeof s=="object"?s:{};if(typeof p.type=="string"){const f=typeof p.tagName=="string"?p.tagName:typeof p.name=="string"?p.name:void 0;Object.defineProperty(m,"name",{value:"node ("+(s.type+(f?"<"+f+">":""))+")"})}return m;function m(){let f=zf,k,w,P;if((!t||l(s,c,d[d.length-1]||void 0))&&(f=k1(n(s,d)),f[0]===bc))return f;if("children"in s&&s.children){const h=s;if(h.children&&f[0]!==y1)for(w=(r?h.children.length:-1)+o,P=d.concat(h);w>-1&&w<h.children.length;){const v=h.children[w];if(k=a(v,w,P)(),k[0]===bc)return k;w=typeof k[1]=="number"?k[1]:w+o}}return f}}}function k1(e){return Array.isArray(e)?e:typeof e=="number"?[v1,e]:e==null?zf:[e]}function Tf(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),x1(e,l,a,i);function a(s,c){const d=c[c.length-1],p=d?d.children.indexOf(s):void 0;return o(s,p,d)}}const wa={}.hasOwnProperty,w1={};function S1(e,t){const n=t||w1,r=new Map,i=new Map,l=new Map,o={...i1,...n.handlers},a={all:c,applyData:j1,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:b1,wrap:E1};return Tf(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const p=d.type==="definition"?r:i,m=String(d.identifier).toUpperCase();p.has(m)||p.set(m,d)}}),a;function s(d,p){const m=d.type,f=a.handlers[m];if(wa.call(a.handlers,m)&&f)return f(a,d,p);if(a.options.passThrough&&a.options.passThrough.includes(m)){if("children"in d){const{children:w,...P}=d,h=ml(P);return h.children=a.all(d),h}return ml(d)}return(a.options.unknownHandler||C1)(a,d,p)}function c(d){const p=[];if("children"in d){const m=d.children;let f=-1;for(;++f<m.length;){const k=a.one(m[f],d);if(k){if(f&&m[f-1].type==="break"&&(!Array.isArray(k)&&k.type==="text"&&(k.value=jc(k.value)),!Array.isArray(k)&&k.type==="element")){const w=k.children[0];w&&w.type==="text"&&(w.value=jc(w.value))}Array.isArray(k)?p.push(...k):p.push(k)}}}return p}}function b1(e,t){e.position&&(t.position=ov(e))}function j1(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,ml(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function C1(e,t){const n=t.data||{},r="value"in t&&!(wa.call(n,"hProperties")||wa.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function E1(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function jc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Cc(e,t){const n=S1(e,t),r=n.one(e,void 0),i=d1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function N1(e,t){return e&&"run"in e?async function(n,r){const i=Cc(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Cc(n,{file:r,...e||t})}}function Ec(e){if(e)throw e}var $i=Object.prototype.hasOwnProperty,Lf=Object.prototype.toString,Nc=Object.defineProperty,_c=Object.getOwnPropertyDescriptor,zc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Lf.call(t)==="[object Array]"},Tc=function(t){if(!t||Lf.call(t)!=="[object Object]")return!1;var n=$i.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&$i.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||$i.call(t,i)},Lc=function(t,n){Nc&&n.name==="__proto__"?Nc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Pc=function(t,n){if(n==="__proto__")if($i.call(t,n)){if(_c)return _c(t,n).value}else return;return t[n]},_1=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<c;++s)if(t=arguments[s],t!=null)for(n in t)r=Pc(a,n),i=Pc(t,n),a!==i&&(d&&i&&(Tc(i)||(l=zc(i)))?(l?(l=!1,o=r&&zc(r)?r:[]):o=r&&Tc(r)?r:{},Lc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Lc(a,{name:n,newValue:i}));return a};const fo=ja(_1);function Sa(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function z1(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...c){const d=e[++l];let p=-1;if(s){o(s);return}for(;++p<i.length;)(c[p]===null||c[p]===void 0)&&(c[p]=i[p]);i=c,d?T1(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function T1(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const yt={basename:L1,dirname:P1,extname:I1,join:M1,sep:"/"};function L1(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');oi(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function P1(e){if(oi(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function I1(e){oi(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function M1(...e){let t=-1,n;for(;++t<e.length;)oi(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":A1(n)}function A1(e){oi(e);const t=e.codePointAt(0)===47;let n=D1(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function D1(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function oi(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const R1={cwd:O1};function O1(){return"/"}function ba(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function F1(e){if(typeof e=="string")e=new URL(e);else if(!ba(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return B1(e)}function B1(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const ho=["history","path","basename","stem","extname","dirname"];class Pf{constructor(t){let n;t?ba(t)?n={path:t}:typeof t=="string"||U1(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":R1.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<ho.length;){const l=ho[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)ho.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?yt.basename(this.path):void 0}set basename(t){go(t,"basename"),mo(t,"basename"),this.path=yt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?yt.dirname(this.path):void 0}set dirname(t){Ic(this.basename,"dirname"),this.path=yt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?yt.extname(this.path):void 0}set extname(t){if(mo(t,"extname"),Ic(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=yt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){ba(t)&&(t=F1(t)),go(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?yt.basename(this.path,this.extname):void 0}set stem(t){go(t,"stem"),mo(t,"stem"),this.path=yt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new ze(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function mo(e,t){if(e&&e.includes(yt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+yt.sep+"`")}function go(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Ic(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function U1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const $1=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},H1={}.hasOwnProperty;class Is extends $1{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=z1()}copy(){const t=new Is;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(fo(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(xo("data",this.frozen),this.namespace[t]=n,this):H1.call(this.namespace,t)&&this.namespace[t]||void 0:t?(xo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Ni(t),r=this.parser||this.Parser;return vo("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),vo("process",this.parser||this.Parser),yo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Ni(t),s=r.parse(a);r.run(s,a,function(d,p,m){if(d||!p||!m)return c(d);const f=p,k=r.stringify(f,m);Q1(k)?m.value=k:m.result=k,c(d,m)});function c(d,p){d||!p?o(d):l?l(p):n(void 0,p)}}}processSync(t){let n=!1,r;return this.freeze(),vo("processSync",this.parser||this.Parser),yo("processSync",this.compiler||this.Compiler),this.process(t,i),Ac("processSync","process",n),r;function i(l,o){n=!0,Ec(l),r=o}}run(t,n,r){Mc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Ni(n);i.run(t,s,c);function c(d,p,m){const f=p||t;d?a(d):o?o(f):r(void 0,f,m)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Ac("runSync","run",r),i;function l(o,a){Ec(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Ni(n),i=this.compiler||this.Compiler;return yo("stringify",i),Mc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(xo("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")s(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...p]=c;s(d,p)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=fo(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const p=c[d];l(p)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function s(c,d){let p=-1,m=-1;for(;++p<r.length;)if(r[p][0]===c){m=p;break}if(m===-1)r.push([c,...d]);else if(d.length>0){let[f,...k]=d;const w=r[m][1];Sa(w)&&Sa(f)&&(f=fo(!0,w,f)),r[m]=[c,f,...k]}}}}const V1=new Is().freeze();function vo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function yo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function xo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Mc(e){if(!Sa(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Ac(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Ni(e){return W1(e)?e:new Pf(e)}function W1(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function Q1(e){return typeof e=="string"||K1(e)}function K1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const q1="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Dc=[],Rc={allowDangerousHtml:!0},Y1=/^(https?|ircs?|mailto|xmpp)$/i,X1=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function G1(e){const t=J1(e),n=Z1(e);return e0(t.runSync(t.parse(n),n),e)}function J1(e){const t=e.rehypePlugins||Dc,n=e.remarkPlugins||Dc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Rc}:Rc;return V1().use(Px).use(n).use(N1,r).use(t)}function Z1(e){const t=e.children||"",n=new Pf;return typeof t=="string"&&(n.value=t),n}function e0(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||t0;for(const d of X1)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+q1+d.id,void 0);return Tf(e,c),dv(e,{Fragment:u.Fragment,components:i,ignoreInvalidStyle:!0,jsx:u.jsx,jsxs:u.jsxs,passKeys:!0,passNode:!0});function c(d,p,m){if(d.type==="raw"&&m&&typeof p=="number")return o?m.children.splice(p,1):m.children[p]={type:"text",value:d.value},p;if(d.type==="element"){let f;for(f in uo)if(Object.hasOwn(uo,f)&&Object.hasOwn(d.properties,f)){const k=d.properties[f],w=uo[f];(w===null||w.includes(d.tagName))&&(d.properties[f]=s(String(k||""),f,d))}}if(d.type==="element"){let f=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!f&&r&&typeof p=="number"&&(f=!r(d,p,m)),f&&m&&typeof p=="number")return a&&d.children?m.children.splice(p,1,...d.children):m.children.splice(p,1),p}}}function t0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||Y1.test(e.slice(0,t))?e:""}const Ke={send:u.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),u.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),u.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),u.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},n0=e=>{switch(e){case"directive":return Ke.directive;case"question":return Ke.question;case"status":return Ke.status;case"result":return Ke.result;case"approval_request":return Ke.lock;default:return Ke.directive}},r0=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=H.useRef(null),[a,s]=_t.useState(""),[c,d]=_t.useState("directive"),[p,m]=_t.useState(""),[f,k]=_t.useState(!1),[w,P]=_t.useState(new Map),[h,v]=_t.useState(new Set);H.useEffect(()=>{e!=null&&e.workspace?m(e.workspace):m("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),H.useEffect(()=>{var E;(E=o.current)==null||E.scrollIntoView({behavior:"smooth"})},[t]);const y=E=>{m(E),r&&r(E)},b=()=>{a.trim()&&(n(a,c,p||void 0),s(""))},_=E=>{E.key==="Enter"&&!E.shiftKey&&(E.preventDefault(),b())},S=E=>new Date(E).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),L=E=>E.length>12?`${E.slice(0,8)}...`:E,C=E=>{if(!E.metadata_json)return null;try{return JSON.parse(E.metadata_json).approval_id||null}catch{return null}},T=E=>{const F=w.get(E)||"";i&&(i(E,F),v(V=>new Set(V).add(E)),P(V=>{const B=new Map(V);return B.delete(E),B}))},R=E=>{const F=w.get(E)||"";if(!F.trim()){alert("Please provide a reason for rejection");return}l&&(l(E,F),v(V=>new Set(V).add(E)),P(V=>{const B=new Map(V);return B.delete(E),B}))},j=(E,F)=>{P(V=>new Map(V).set(E,F))};return e?u.jsxs("div",{className:"conversation-view",children:[u.jsxs("div",{className:"conversation-header",children:[u.jsxs("div",{className:"header-info",children:[u.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&u.jsxs("span",{className:"thread-agent-badge",children:[Ke.bot,e.target_agent]})]}),u.jsxs("div",{className:"header-stats",children:[u.jsxs("span",{className:"message-count",children:[t.length," messages"]}),u.jsx("span",{className:"thread-id",title:e.id,children:L(e.id)})]})]}),u.jsxs("div",{className:"messages-container",children:[t.length===0?u.jsxs("div",{className:"empty-messages",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("p",{children:"No messages yet"}),u.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((E,F)=>{const V=E.from_type==="human",B=F===0||t[F-1].from_type!==E.from_type;return u.jsxs("div",{className:`message ${V?"human":"agent"}`,children:[u.jsx("div",{className:`message-avatar ${B?"visible":""}`,children:B&&(V?Ke.user:Ke.bot)}),u.jsxs("div",{className:"message-body",children:[B&&u.jsxs("div",{className:"message-meta",children:[u.jsx("span",{className:"sender-name",children:E.from_id}),u.jsxs("span",{className:"kind-badge",children:[n0(E.kind)," ",E.kind]}),u.jsx("span",{className:"message-time",children:S(E.created_at)})]}),u.jsxs("div",{className:"message-content",children:[E.kind==="result"||!V?u.jsx(G1,{components:{a:({href:U,children:K})=>{let A=U;return U&&U.startsWith("/")&&!U.startsWith("//")&&(A=`file://${U}`),u.jsx("a",{href:A,target:"_blank",rel:"noopener noreferrer",children:K})},code:({className:U,children:K,...A})=>!U?u.jsx("code",{className:"inline-code",...A,children:K}):u.jsx("code",{className:U,...A,children:K})},children:E.content}):E.content,E.kind==="approval_request"&&(()=>{const U=C(E),K=U&&h.has(U);return U?u.jsx("div",{className:"inline-approval",children:K?u.jsxs("div",{className:"approval-handled",children:[Ke.check,u.jsx("span",{children:"Action taken"})]}):u.jsxs(u.Fragment,{children:[u.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:w.get(U)||"",onChange:A=>j(U,A.target.value)}),u.jsxs("div",{className:"approval-actions",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>R(U),title:"Reject",children:[Ke.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>T(U),title:"Approve",children:[Ke.check,"Approve"]})]})]})}):null})()]}),u.jsxs("div",{className:"message-footer",children:[u.jsxs("span",{className:"message-seq",children:["#",E.message_seq]}),E.delivery_state!=="acked"&&u.jsx("span",{className:`delivery-status ${E.delivery_state}`,children:E.delivery_state==="pending"?"sending...":"delivered"})]})]})]},E.id)}),u.jsx("div",{ref:o})]}),u.jsxs("div",{className:"input-area",children:[f&&u.jsxs("div",{className:"workspace-input-row",children:[u.jsx("input",{type:"text",value:p,onChange:E=>y(E.target.value),onBlur:()=>{r&&r(p)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),u.jsx("button",{onClick:async()=>{try{const F=await(await fetch("/api/select-folder")).json();!F.cancelled&&F.path&&y(F.path)}catch(E){console.error("Failed to open folder picker:",E)}},className:"workspace-browse",title:"Browse for folder",children:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),u.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),u.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),p&&u.jsx("button",{onClick:()=>{y(""),k(!1)},className:"workspace-clear",children:"Clear"})]}),u.jsxs("div",{className:"input-wrapper",children:[u.jsx("button",{onClick:()=>k(!f),className:`workspace-toggle ${p?"has-workspace":""}`,title:p||"Set working directory",children:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),u.jsxs("select",{value:c,onChange:E=>d(E.target.value),className:"kind-selector",children:[u.jsx("option",{value:"directive",children:"Directive"}),u.jsx("option",{value:"question",children:"Question"})]}),u.jsx("textarea",{value:a,onChange:E=>s(E.target.value),onKeyPress:_,placeholder:p?`Message (workspace: ${p.split("/").pop()})`:"Type a message...",rows:1}),u.jsx("button",{onClick:b,className:"send-btn",disabled:!a.trim(),children:Ke.send})]}),u.jsxs("div",{className:"input-hint",children:["Press ",u.jsx("kbd",{children:"Enter"})," to send, ",u.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),u.jsx("style",{children:`
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
      `})]}):null},i0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,reconnectInterval:l=5e3})=>{const o=H.useRef(null),[a,s]=H.useState(!1),[c,d]=H.useState(null),p=H.useRef(null),m=H.useRef(new Map),f=H.useCallback(()=>{try{const b=`${e}?instance_id=${t}`;o.current=new WebSocket(b),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),d(null),m.current.forEach((_,S)=>{P(S,_)})},o.current.onmessage=_=>{try{const S=JSON.parse(_.data);k(S)}catch(S){console.error("Failed to parse WebSocket message:",S)}},o.current.onerror=_=>{console.error("WebSocket error:",_),d("Connection error")},o.current.onclose=()=>{console.log("WebSocket disconnected"),s(!1),p.current=setTimeout(()=>{console.log("Attempting to reconnect..."),f()},l)}}catch(b){console.error("Failed to connect to WebSocket:",b),d("Failed to connect")}},[e,t,l]),k=H.useCallback(b=>{switch(b.type){case"message":n&&b.data&&n(b.data);break;case"batch":if(r&&b.data){const _=b.data;r(_),n&&_.messages.forEach(S=>n(S))}break;case"error":i&&b.data&&i(b.data),console.error("WebSocket error event:",b.data);break;case"pong":break;default:console.log("Unknown event type:",b.type)}},[n,r,i]),w=H.useCallback(b=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(b)):console.warn("WebSocket not connected, cannot send event")},[]),P=H.useCallback((b,_=0)=>{m.current.set(b,_);const S={type:"subscribe",timestamp:Date.now(),data:{thread_id:b,from_seq:_}};w(S)},[w]),h=H.useCallback((b,_)=>{const S=m.current.get(b)||0;_>S&&m.current.set(b,_);const L={type:"ack",timestamp:Date.now(),data:{thread_id:b,ack_seq:_}};w(L)},[w]),v=H.useCallback(()=>{const b={type:"ping",timestamp:Date.now()};w(b)},[w]),y=H.useCallback(b=>{m.current.delete(b)},[]);return H.useEffect(()=>(f(),()=>{p.current&&clearTimeout(p.current),o.current&&o.current.close()}),[f]),H.useEffect(()=>{if(!a)return;const b=setInterval(()=>{v()},3e4);return()=>clearInterval(b)},[a,v]),{isConnected:a,connectionError:c,subscribe:P,unsubscribe:y,acknowledge:h,ping:v}},l0=({connected:e})=>u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?u.jsxs(u.Fragment,{children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):u.jsxs(u.Fragment,{children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),u.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),o0=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=H.useState([]),[o,a]=H.useState(null),[s,c]=H.useState(new Map),[d,p]=H.useState(new Map),[m,f]=H.useState([]),[k,w]=H.useState(!1),[P,h]=H.useState(""),{isConnected:v,subscribe:y,acknowledge:b}=i0({url:e,instanceId:t,onMessage:_,onBatch:S});function _(N){const g={id:N.id,thread_id:N.thread_id,message_seq:N.message_seq,created_at:N.created_at,from_type:N.from_type,from_id:N.from_id,to_type:N.to_type,to_id:N.to_id,kind:N.kind,subject:N.subject,content:N.content,metadata_json:N.metadata_json,delivery_state:"visible",business_state:"open"};c(D=>{const W=D.get(g.thread_id)||[];return W.find(x=>x.id===g.id)?D:new Map(D).set(g.thread_id,[...W,g].sort((x,ne)=>x.message_seq-ne.message_seq))}),g.thread_id!==o&&p(D=>{const W=D.get(g.thread_id)||0;return new Map(D).set(g.thread_id,W+1)}),b(g.thread_id,g.message_seq)}function S(N){N.messages.forEach(g=>{_(g)})}const L=H.useCallback(N=>{if(a(N),p(g=>{const D=new Map(g);return D.delete(N),D}),v){const g=s.get(N)||[],D=g.length>0?Math.max(...g.map(W=>W.message_seq)):0;y(N,D)}},[v,y,s]),C=H.useCallback(async(N,g,D)=>{if(!o)return;const W=D?JSON.stringify({workspace:D}):void 0;try{const x=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:g,content:N,metadata_json:W})});if(!x.ok){console.error("Failed to send message:",await x.text());return}const ne=await x.json();c(Te=>{const te=Te.get(o)||[];return te.find(et=>et.id===ne.id)?Te:new Map(Te).set(o,[...te,ne])})}catch(x){console.error("Error sending message:",x)}},[o,t]);H.useEffect(()=>{(async()=>{try{const g=await fetch("/api/threads");if(!g.ok){console.error("Failed to fetch threads:",await g.text());return}const D=await g.json();l(D),D.length>0&&!o&&a(D[0].id)}catch(g){console.error("Error fetching threads:",g)}})()},[]),H.useEffect(()=>{n&&i.length>0&&(i.some(g=>g.id===n)&&(a(n),p(g=>{const D=new Map(g);return D.delete(n),D})),r&&r())},[n,i,r]);const T=H.useCallback(async N=>{try{const g=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:N,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!g.ok){console.error("Failed to create thread:",await g.text());return}const D=await g.json();l(W=>[D,...W]),a(D.id)}catch(g){console.error("Error creating thread:",g)}},[t]),R=H.useCallback(async()=>{try{const N=await fetch("/api/agents");if(!N.ok){console.error("Failed to fetch agents:",await N.text());return}const g=await N.json();f(g.running||[])}catch(N){console.error("Error fetching agents:",N)}},[]);H.useEffect(()=>{R();const N=setInterval(R,5e3);return()=>clearInterval(N)},[R]);const j=H.useCallback(async()=>{if(P.trim())try{const N=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:P.trim()})});if(!N.ok){const D=await N.text();console.error("Failed to launch agent:",D),alert(`Failed to launch agent: ${D}`);return}const g=await N.json();f(D=>[...D,g]),h(""),w(!1)}catch(N){console.error("Error launching agent:",N)}},[P]),E=H.useCallback(async N=>{try{const g=await fetch(`/api/agents/${N}`,{method:"DELETE"});if(!g.ok){console.error("Failed to stop agent:",await g.text());return}f(D=>D.filter(W=>W.instance_id!==N))}catch(g){console.error("Error stopping agent:",g)}},[]),F=H.useCallback(async N=>{if(o)try{const g=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:N})});if(!g.ok){console.error("Failed to update workspace:",await g.text());return}const D=await g.json();l(W=>W.map(x=>x.id===o?D:x))}catch(g){console.error("Error updating workspace:",g)}},[o]),V=H.useCallback(async N=>{try{const g=await fetch(`/api/threads/${N}`,{method:"DELETE"});if(!g.ok){console.error("Failed to delete thread:",await g.text());return}l(D=>D.filter(W=>W.id!==N)),c(D=>{const W=new Map(D);return W.delete(N),W}),p(D=>{const W=new Map(D);return W.delete(N),W}),o===N&&a(null)}catch(g){console.error("Error deleting thread:",g)}},[o]),B=H.useCallback(async(N,g)=>{try{const D=await fetch(`/api/threads/${N}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:g})});if(!D.ok){console.error("Failed to rename thread:",await D.text());return}const W=await D.json();l(x=>x.map(ne=>ne.id===N?W:ne))}catch(D){console.error("Error renaming thread:",D)}},[]),U=H.useCallback(async(N,g)=>{try{const D=await fetch(`/api/approvals/${N}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!D.ok){const W=await D.text();console.error("Failed to approve request:",W),alert(`Failed to approve: ${W}`);return}console.log("Approval approved successfully")}catch(D){console.error("Error approving request:",D)}},[]),K=H.useCallback(async(N,g)=>{try{const D=await fetch(`/api/approvals/${N}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!D.ok){const W=await D.text();console.error("Failed to reject request:",W),alert(`Failed to reject: ${W}`);return}console.log("Approval rejected successfully")}catch(D){console.error("Error rejecting request:",D)}},[]),A=o?s.get(o)||[]:[];return u.jsxs("div",{className:"message-center",children:[u.jsxs("div",{className:"status-bar",children:[u.jsxs("div",{className:`status-indicator ${v?"connected":"disconnected"}`,children:[u.jsx(l0,{connected:v}),u.jsx("span",{children:v?"Connected":"Disconnected"})]}),u.jsxs("div",{className:"status-meta",children:[u.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),u.jsxs("span",{className:"agent-count",children:[m.length," agents"]}),u.jsx("button",{className:"launch-agent-btn",onClick:()=>w(!0),children:"+ Agent"})]})]}),m.length>0&&u.jsx("div",{className:"agents-bar",children:m.map(N=>u.jsxs("div",{className:"agent-chip",children:[u.jsx("span",{className:"agent-pulse"}),u.jsx("span",{className:"agent-name",children:N.instance_id}),u.jsxs("span",{className:"agent-pid",children:["PID ",N.pid]}),u.jsx("button",{className:"agent-stop-btn",onClick:()=>E(N.instance_id),title:"Stop agent",children:"×"})]},N.instance_id))}),k&&u.jsx("div",{className:"modal-overlay",onClick:()=>w(!1),children:u.jsxs("div",{className:"modal-content",onClick:N=>N.stopPropagation(),children:[u.jsx("h3",{children:"Launch New Agent"}),u.jsx("input",{type:"text",value:P,onChange:N=>h(N.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:N=>{N.key==="Enter"&&j(),N.key==="Escape"&&w(!1)}}),u.jsxs("div",{className:"modal-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>w(!1),children:"Cancel"}),u.jsx("button",{className:"launch-btn",onClick:j,children:"Launch"})]})]})}),u.jsxs("div",{className:"center-layout",children:[u.jsx("aside",{className:"threads-panel",children:u.jsx(hg,{threads:i,selectedThreadId:o,onSelectThread:L,onCreateThread:T,onDeleteThread:V,onRenameThread:B,unreadCounts:d})}),u.jsx("main",{className:"conversation-panel",children:o?u.jsx(r0,{thread:i.find(N=>N.id===o),messages:A,onSendMessage:C,onWorkspaceChange:F,onApproveRequest:U,onRejectRequest:K}):u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("h3",{children:"Select a conversation"}),u.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),u.jsx("style",{children:`
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
      `})]})},Le={check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),dollar:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:u.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),u.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),u.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},a0=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=H.useState(!0),[a,s]=H.useState(null),[c,d]=H.useState(new Map),p=h=>{try{return JSON.parse(h)}catch{return null}},m=h=>new Date(h).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),f=h=>{const v=c.get(h)||"";n(h,v),d(new Map(c.set(h,"")))},k=h=>{const v=c.get(h)||"";if(!v.trim()){alert("Please provide a reason for rejection");return}r(h,v),d(new Map(c.set(h,"")))},w=(h,v)=>{d(new Map(c.set(h,v)))},P=e.filter(h=>h.status==="pending");return u.jsxs("div",{className:"approval-queue",children:[u.jsx("div",{className:"queue-header",children:u.jsxs("div",{className:"header-title",children:[u.jsx("h2",{children:"Approval Queue"}),u.jsxs("span",{className:"pending-count",children:[P.length," pending"]})]})}),u.jsxs("div",{className:"approvals-container",children:[P.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Le.sparkles}),u.jsx("h3",{children:"All caught up!"}),u.jsx("p",{children:"No pending approvals to review"})]}):u.jsx("div",{className:"approvals-list",children:P.map(h=>{const v=p(h.effect_delta_json),y=a===h.id;return u.jsxs("div",{className:`approval-card impact-${h.impact}`,children:[u.jsxs("div",{className:"card-header",onClick:()=>s(y?null:h.id),children:[u.jsxs("div",{className:"header-left",children:[u.jsx("div",{className:`impact-indicator ${h.impact}`}),u.jsxs("div",{className:"proposal-info",children:[u.jsx("span",{className:"proposal-text",children:h.proposal}),u.jsxs("div",{className:"proposal-meta",children:[h.thread_title&&u.jsxs("span",{className:"meta-item thread-link",onClick:b=>{b.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Le.message,h.thread_title]}),u.jsxs("span",{className:"meta-item",children:[Le.bot,h.instance_id]}),u.jsxs("span",{className:"meta-item",children:[Le.clock,m(h.created_at)]})]})]})]}),u.jsxs("div",{className:"header-right",children:[u.jsxs("span",{className:"cost-badge",children:[Le.dollar,"$",h.estimated_cost.toFixed(2)]}),u.jsx("span",{className:`impact-badge ${h.impact}`,children:h.impact}),u.jsx("button",{className:"expand-btn",children:y?Le.chevronUp:Le.chevronDown})]})]}),y&&u.jsxs("div",{className:"card-details",children:[v&&u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Effect Details"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Capability"}),u.jsx("span",{className:"detail-value code",children:v.cap_type})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Budget Delta"}),u.jsxs("span",{className:"detail-value",children:["$",v.budget_delta.toFixed(2)]})]}),v.paths.length>0&&u.jsxs("div",{className:"detail-item full-width",children:[u.jsx("span",{className:"detail-label",children:"Paths"}),u.jsx("div",{className:"paths-list",children:v.paths.map((b,_)=>u.jsxs("span",{className:"path-tag",children:[Le.folder,b]},_))})]})]})]}),u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Request Info"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Thread"}),u.jsx("span",{className:"detail-value code",children:h.thread_id})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Impact Level"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]})]})]}),u.jsxs("div",{className:"review-section",children:[u.jsx("h4",{children:"Review Notes"}),u.jsx("textarea",{value:c.get(h.id)||"",onChange:b=>w(h.id,b.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),u.jsxs("div",{className:"action-buttons",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>k(h.id),children:[Le.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>f(h.id),children:[Le.check,"Approve"]})]})]})]})]},h.id)})}),t.length>0&&u.jsxs("div",{className:"history-section",children:[u.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[u.jsxs("h3",{children:[l?Le.chevronDown:Le.chevronUp,"Review History"]}),u.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&u.jsx("div",{className:"history-list",children:t.map(h=>{const v=a===`history-${h.id}`;return u.jsxs("div",{className:`history-card ${h.status}`,onClick:()=>s(v?null:`history-${h.id}`),children:[u.jsxs("div",{className:"history-card-header",children:[u.jsxs("div",{className:"history-status",children:[u.jsx("span",{className:`status-icon ${h.status}`,children:h.status==="approved"?Le.check:Le.x}),u.jsxs("div",{className:"history-info",children:[u.jsx("span",{className:"history-proposal",children:h.proposal}),h.thread_title&&u.jsxs("span",{className:"history-thread",onClick:y=>{y.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Le.message,h.thread_title]})]})]}),u.jsxs("div",{className:"history-meta",children:[u.jsx("span",{className:"history-agent",children:h.instance_id}),u.jsx("span",{className:`history-badge ${h.status}`,children:h.status}),u.jsx("span",{className:"history-time",children:h.reviewed_at?m(h.reviewed_at):m(h.created_at)})]})]}),v&&u.jsxs("div",{className:"history-details",children:[u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Reviewed by"}),u.jsx("span",{className:"detail-value",children:h.reviewed_by||"Unknown"})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Cost"}),u.jsxs("span",{className:"detail-value",children:["$",h.estimated_cost.toFixed(2)]})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Impact"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]}),h.review_notes&&u.jsxs("div",{className:"detail-row full-width",children:[u.jsx("span",{className:"detail-label",children:"Notes"}),u.jsx("span",{className:"detail-value notes",children:h.review_notes})]})]})]},`history-${h.id}`)})})]})]}),u.jsx("style",{children:`
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
      `})]})},me={cpu:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"4",y:"4",width:"16",height:"16",rx:"2"}),u.jsx("rect",{x:"9",y:"9",width:"6",height:"6"}),u.jsx("line",{x1:"9",y1:"1",x2:"9",y2:"4"}),u.jsx("line",{x1:"15",y1:"1",x2:"15",y2:"4"}),u.jsx("line",{x1:"9",y1:"20",x2:"9",y2:"23"}),u.jsx("line",{x1:"15",y1:"20",x2:"15",y2:"23"}),u.jsx("line",{x1:"20",y1:"9",x2:"23",y2:"9"}),u.jsx("line",{x1:"20",y1:"14",x2:"23",y2:"14"}),u.jsx("line",{x1:"1",y1:"9",x2:"4",y2:"9"}),u.jsx("line",{x1:"1",y1:"14",x2:"4",y2:"14"})]}),memory:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"2",y:"6",width:"20",height:"12",rx:"2"}),u.jsx("line",{x1:"6",y1:"10",x2:"6",y2:"14"}),u.jsx("line",{x1:"10",y1:"10",x2:"10",y2:"14"}),u.jsx("line",{x1:"14",y1:"10",x2:"14",y2:"14"}),u.jsx("line",{x1:"18",y1:"10",x2:"18",y2:"14"})]}),clock:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),dollar:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),activity:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"22 12 18 12 15 21 9 3 6 12 2 12"})}),tokens:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"}),u.jsx("line",{x1:"10",y1:"9",x2:"8",y2:"9"})]}),message:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),stop:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("rect",{x:"3",y:"3",width:"18",height:"18",rx:"2"})}),warning:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"}),u.jsx("line",{x1:"12",y1:"9",x2:"12",y2:"13"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),check:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})})},s0=()=>{const[e,t]=H.useState(null),[n,r]=H.useState(null),[i,l]=H.useState(null),[o,a]=H.useState(new Map),[s,c]=H.useState({prevCost:0,prevTokensIn:0,prevTokensOut:0,timestamp:0}),d=H.useCallback(async()=>{try{const C=await fetch("/api/monitor");if(!C.ok)throw new Error(`Failed to fetch: ${C.statusText}`);const T=await C.json();t(T),l(new Date),r(null)}catch(C){r(C instanceof Error?C.message:"Unknown error")}},[]);H.useEffect(()=>{d();const C=setInterval(d,2e3);return()=>clearInterval(C)},[d]),H.useEffect(()=>{const T=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;let R=null,j=null;const E=()=>{R=new WebSocket(T),R.onmessage=F=>{try{const V=JSON.parse(F.data);if(V.type==="telemetry"){const B=V.data;a(U=>{const K=new Map(U);return K.set(B.instance_id,B),K})}}catch{}},R.onclose=()=>{j=setTimeout(E,3e3)},R.onerror=()=>{R==null||R.close()}};return E(),()=>{j&&clearTimeout(j),R==null||R.close()}},[]);const p=async C=>{try{await fetch(`/api/agents/${C}`,{method:"DELETE"}),d()}catch(T){console.error("Failed to stop process:",T)}},m=C=>{if(C<0)return"Unknown";if(C<60)return`${C}s`;if(C<3600){const j=Math.floor(C/60),E=C%60;return`${j}m ${E}s`}const T=Math.floor(C/3600),R=Math.floor(C%3600/60);return`${T}h ${R}m`},f=C=>C===0?"$0.00":C<.01?`$${C.toFixed(4)}`:`$${C.toFixed(2)}`,k=C=>{switch(C){case"running":return"var(--color-success)";case"completed":return"var(--color-primary)";case"failed":return"var(--color-danger)";case"orphan":return"var(--color-warning)";default:return"var(--text-tertiary)"}},w=C=>C.cpu_percent>80||C.duration_sec>300,P=C=>{const T=o.get(C.instance_id);return T?{...C,turns:T.turns,tokens_in:T.tokens_in,tokens_out:T.tokens_out,cost:T.cost,hasLiveTelemetry:!0}:{...C,hasLiveTelemetry:!1}},h=Array.from(o.values()).reduce((C,T)=>({tokens_in:C.tokens_in+T.tokens_in,tokens_out:C.tokens_out+T.tokens_out,cost:C.cost+T.cost,turns:C.turns+T.turns}),{tokens_in:0,tokens_out:0,cost:0,turns:0}),v=h.cost>0?h.cost:(e==null?void 0:e.summary.total_cost)||0,y={in:h.tokens_in,out:h.tokens_out},b=o.size>0;H.useEffect(()=>{if(b){const C=Date.now();C-s.timestamp>2e3&&c({prevCost:v,prevTokensIn:y.in,prevTokensOut:y.out,timestamp:C})}},[v,y.in,y.out,b,s.timestamp]);const _=v>s.prevCost&&s.prevCost>0?"up":null,S=y.in+y.out>s.prevTokensIn+s.prevTokensOut&&s.prevTokensIn+s.prevTokensOut>0?"up":null,L=C=>C>=1e6?`${(C/1e6).toFixed(1)}M`:C>=1e3?`${(C/1e3).toFixed(1)}K`:C.toString();return u.jsxs("div",{className:"monitor",children:[u.jsxs("div",{className:"monitor-summary",children:[u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:me.activity}),u.jsx("span",{className:"summary-value",children:(e==null?void 0:e.summary.total_processes)||0}),u.jsx("span",{className:"summary-label",children:"Running"})]}),u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:me.cpu}),u.jsxs("span",{className:"summary-value",children:[(e==null?void 0:e.summary.total_cpu_percent.toFixed(1))||"0.0","%"]}),u.jsx("span",{className:"summary-label",children:"CPU"})]}),u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:me.memory}),u.jsxs("span",{className:"summary-value",children:[(e==null?void 0:e.summary.total_memory_mb.toFixed(0))||"0"," MB"]}),u.jsx("span",{className:"summary-label",children:"Memory"})]}),u.jsxs("div",{className:`summary-item ${b?"live":""}`,children:[u.jsx("span",{className:"summary-icon",children:me.dollar}),u.jsxs("span",{className:"summary-value",children:[f(v),_==="up"&&u.jsx("span",{className:"trend-up",children:"▲"})]}),u.jsxs("span",{className:"summary-label",children:["Cost ",b&&u.jsx("span",{className:"live-indicator",children:"●"})]})]}),u.jsxs("div",{className:`summary-item ${b?"live":""}`,children:[u.jsx("span",{className:"summary-icon",children:me.tokens}),u.jsxs("span",{className:"summary-value",children:[L(y.in),"↓ / ",L(y.out),"↑",S==="up"&&u.jsx("span",{className:"trend-up",children:"▲"})]}),u.jsxs("span",{className:"summary-label",children:["Tokens ",b&&u.jsx("span",{className:"live-indicator",children:"●"})]})]}),h.turns>0&&u.jsxs("div",{className:"summary-item live",children:[u.jsx("span",{className:"summary-icon",children:me.message}),u.jsx("span",{className:"summary-value",children:h.turns}),u.jsxs("span",{className:"summary-label",children:["Turns ",u.jsx("span",{className:"live-indicator",children:"●"})]})]}),((e==null?void 0:e.summary.warning_count)||0)>0&&u.jsxs("div",{className:"summary-item warning",children:[u.jsx("span",{className:"summary-icon",children:me.warning}),u.jsx("span",{className:"summary-value",children:e==null?void 0:e.summary.warning_count}),u.jsx("span",{className:"summary-label",children:"Warnings"})]}),u.jsx("div",{className:"summary-spacer"}),u.jsxs("div",{className:"summary-update",children:[b&&u.jsx("span",{className:"live-badge-summary",children:"LIVE"}),"Last update: ",i?i.toLocaleTimeString():"Never"]})]}),u.jsxs("div",{className:"process-grid",children:[n&&u.jsxs("div",{className:"error-card",children:[u.jsx("span",{className:"error-icon",children:me.warning}),u.jsx("span",{children:n})]}),(!(e!=null&&e.processes)||e.processes.length===0)&&!n&&u.jsxs("div",{className:"empty-state",children:[u.jsx("span",{className:"empty-icon",children:me.activity}),u.jsx("h3",{children:"No Active Processes"}),u.jsx("p",{children:"Spawn an agent from the Messages tab to see it here."})]}),e==null?void 0:e.processes.map(C=>{const T=P(C);return u.jsxs("div",{className:`process-card ${w(T)?"warning":""} ${T.hasLiveTelemetry?"live":""}`,children:[u.jsxs("div",{className:"process-header",children:[u.jsxs("div",{className:"process-status",children:[u.jsx("span",{className:"status-dot",style:{background:k(T.status)}}),u.jsx("span",{className:"process-name",children:T.instance_id}),T.hasLiveTelemetry&&u.jsx("span",{className:"live-badge",children:"LIVE"})]}),T.status==="running"&&u.jsx("button",{className:"stop-btn",onClick:()=>p(T.instance_id),title:"Stop process",children:me.stop}),T.status==="completed"&&u.jsxs("span",{className:"status-badge completed",children:[me.check," Done"]})]}),u.jsxs("div",{className:"process-metrics",children:[u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:me.cpu}),u.jsxs("span",{className:`metric-value ${T.cpu_percent>80?"high":""}`,children:[T.cpu_percent.toFixed(1),"%"]}),u.jsx("span",{className:"metric-label",children:"CPU"})]}),u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:me.memory}),u.jsxs("span",{className:"metric-value",children:[T.memory_mb.toFixed(0)," MB"]}),u.jsx("span",{className:"metric-label",children:"Memory"})]}),u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:me.clock}),u.jsx("span",{className:`metric-value ${T.duration_sec>300?"high":""}`,children:m(T.duration_sec)}),u.jsx("span",{className:"metric-label",children:"Duration"})]})]}),T.hasLiveTelemetry&&u.jsxs("div",{className:"process-telemetry",children:[u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:me.message}),u.jsx("span",{className:"telemetry-value",children:T.turns||0}),u.jsx("span",{className:"telemetry-label",children:"Turns"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:me.tokens}),u.jsx("span",{className:"telemetry-value",children:L(T.tokens_in||0)}),u.jsx("span",{className:"telemetry-label",children:"In"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:me.tokens}),u.jsx("span",{className:"telemetry-value",children:L(T.tokens_out||0)}),u.jsx("span",{className:"telemetry-label",children:"Out"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:me.dollar}),u.jsx("span",{className:"telemetry-value cost",children:f(T.cost||0)}),u.jsx("span",{className:"telemetry-label",children:"Cost"})]})]}),u.jsxs("div",{className:"process-footer",children:[u.jsxs("span",{className:"process-pid",children:["PID: ",T.pid]}),T.source&&u.jsx("span",{className:`source-badge ${T.source}`,children:T.source}),T.command&&u.jsx("span",{className:"process-command",title:T.full_cmd,children:T.command}),!T.hasLiveTelemetry&&T.turns&&u.jsxs("span",{className:"process-turns",children:[T.turns," turns"]}),!T.hasLiveTelemetry&&T.cost!==void 0&&T.cost>0&&u.jsx("span",{className:"process-cost",children:f(T.cost)})]})]},T.instance_id)}),(e==null?void 0:e.history)&&e.history.length>0&&u.jsxs(u.Fragment,{children:[u.jsx("div",{className:"history-divider",children:u.jsx("span",{children:"Recent History"})}),e.history.map(C=>u.jsxs("div",{className:`process-card history ${C.status==="failed"?"failed":""}`,children:[u.jsx("div",{className:"process-header",children:u.jsxs("div",{className:"process-status",children:[u.jsx("span",{className:"status-dot",style:{background:k(C.status)}}),u.jsx("span",{className:"process-name",children:C.instance_id}),u.jsxs("span",{className:`status-badge ${C.status}`,children:[C.status==="completed"?me.check:me.warning,C.status]})]})}),u.jsxs("div",{className:"process-footer",children:[u.jsxs("span",{className:"process-pid",children:["PID: ",C.pid]}),C.source&&u.jsx("span",{className:`source-badge ${C.source}`,children:C.source}),C.command&&u.jsx("span",{className:"process-command",title:C.full_cmd,children:C.command}),u.jsx("span",{className:"process-duration",children:m(C.duration_sec)}),C.cost!==void 0&&C.cost>0&&u.jsx("span",{className:"process-cost",children:f(C.cost)})]})]},`history-${C.instance_id}-${C.stopped_at}`))]})]}),u.jsx("style",{children:`
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
          display: flex;
          align-items: center;
          gap: 4px;
        }

        .summary-item.live .summary-value {
          color: var(--color-primary);
        }

        .trend-up {
          color: var(--color-success);
          font-size: 10px;
          margin-left: 4px;
          animation: trend-flash 0.5s ease-out;
        }

        @keyframes trend-flash {
          0% { opacity: 0; transform: scale(1.5); }
          100% { opacity: 1; transform: scale(1); }
        }

        .live-indicator {
          color: var(--color-primary);
          font-size: 8px;
          animation: live-blink 1s ease-in-out infinite;
        }

        @keyframes live-blink {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.3; }
        }

        .live-badge-summary {
          display: inline-block;
          font-size: 9px;
          font-weight: var(--font-bold);
          color: var(--color-primary);
          background: rgba(37, 194, 160, 0.15);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          margin-right: var(--space-2);
          animation: live-blink 1s ease-in-out infinite;
        }

        .summary-spacer {
          flex: 1;
        }

        .summary-update {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          display: flex;
          align-items: center;
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

        .process-card.live {
          border-color: var(--color-primary);
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

        .live-badge {
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          color: var(--color-primary);
          background: rgba(37, 194, 160, 0.15);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          animation: live-pulse 1.5s ease-in-out infinite;
        }

        @keyframes live-pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.6; }
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

        /* Telemetry Row */
        .process-telemetry {
          display: flex;
          gap: var(--space-3);
          padding: var(--space-3);
          background: rgba(37, 194, 160, 0.05);
          border-radius: var(--radius-md);
          margin-bottom: var(--space-4);
        }

        .telemetry-item {
          display: flex;
          flex-direction: column;
          align-items: center;
          flex: 1;
        }

        .telemetry-icon {
          color: var(--color-primary);
          margin-bottom: var(--space-1);
          opacity: 0.7;
        }

        .telemetry-value {
          font-size: var(--text-sm);
          font-weight: var(--font-semibold);
          font-family: var(--font-mono);
          color: var(--text-primary);
        }

        .telemetry-value.cost {
          color: var(--color-primary);
        }

        .telemetry-label {
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

        /* Source Badge */
        .source-badge {
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          text-transform: uppercase;
        }

        .source-badge.ui {
          background: rgba(59, 130, 246, 0.15);
          color: #3b82f6;
        }

        .source-badge.eval {
          background: rgba(168, 85, 247, 0.15);
          color: #a855f7;
        }

        .source-badge.cli {
          background: rgba(100, 116, 139, 0.15);
          color: var(--text-secondary);
        }

        .source-badge.agent {
          background: rgba(37, 194, 160, 0.15);
          color: var(--color-primary);
        }

        .process-command {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-secondary);
          max-width: 150px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .process-duration {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
        }

        /* History Section */
        .history-divider {
          grid-column: 1 / -1;
          display: flex;
          align-items: center;
          gap: var(--space-3);
          margin: var(--space-4) 0;
        }

        .history-divider::before,
        .history-divider::after {
          content: '';
          flex: 1;
          height: 1px;
          background: var(--border-subtle);
        }

        .history-divider span {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .process-card.history {
          opacity: 0.7;
          background: var(--bg-base);
        }

        .process-card.history:hover {
          opacity: 1;
        }

        .process-card.history.failed {
          border-color: var(--color-danger);
          background: rgba(248, 81, 73, 0.05);
        }

        .status-badge.failed {
          background: rgba(248, 81, 73, 0.1);
          color: var(--color-danger);
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
      `})]})},_i={messages:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),shield:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"})}),activity:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"22 12 18 12 15 21 9 3 6 12 2 12"})}),logo:u.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M12 6v12M6 12h12"}),u.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]})},u0=()=>{const[e,t]=H.useState("messages"),[n,r]=H.useState([]),[i,l]=H.useState([]),[o,a]=H.useState("my-agent"),[s,c]=H.useState([]),[d,p]=H.useState(""),[m,f]=H.useState(!1),[k,w]=H.useState(null),h=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;_t.useEffect(()=>{const j=async()=>{try{const F=await fetch("/api/agents");if(F.ok){const V=await F.json();c(V),V.length>0&&!o&&a(V[0].id)}}catch(F){console.error("Error fetching agents:",F)}};j();const E=setInterval(j,1e4);return()=>clearInterval(E)},[]);const v=j=>{const E=j.target.value;E==="__custom__"?f(!0):(a(E),f(!1))},y=()=>{d.trim()&&(a(d.trim()),f(!1),p(""))},b=j=>j.last_active?Date.now()-j.last_active<3e4:!1,_=j=>b(j)?"●":"○",S=async(j,E)=>{try{const F=await fetch(`/api/approvals/${j}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:E})});if(!F.ok){console.error("Failed to approve:",await F.text());return}const V=n.find(B=>B.id===j);if(V){const B={...V,status:"approved",reviewed_by:"user",review_notes:E,reviewed_at:Date.now()};l(U=>[B,...U])}r(B=>B.filter(U=>U.id!==j))}catch(F){console.error("Error approving:",F)}},L=async(j,E)=>{try{const F=await fetch(`/api/approvals/${j}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:E})});if(!F.ok){console.error("Failed to reject:",await F.text());return}const V=n.find(B=>B.id===j);if(V){const B={...V,status:"rejected",reviewed_by:"user",review_notes:E,reviewed_at:Date.now()};l(U=>[B,...U])}r(B=>B.filter(U=>U.id!==j))}catch(F){console.error("Error rejecting:",F)}};_t.useEffect(()=>{const j=async()=>{try{const F=await fetch("/api/approvals?status=pending");if(F.ok){const K=await F.json();r(K)}const[V,B]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),U=[];if(V.ok){const K=await V.json();U.push(...K)}if(B.ok){const K=await B.json();U.push(...K)}U.sort((K,A)=>{const N=K.reviewed_at?new Date(K.reviewed_at).getTime():0;return(A.reviewed_at?new Date(A.reviewed_at).getTime():0)-N}),l(U)}catch(F){console.error("Error fetching approvals:",F)}};j();const E=setInterval(j,5e3);return()=>clearInterval(E)},[]);const C=(n==null?void 0:n.filter(j=>j.status==="pending").length)||0,T=j=>{w(j),t("messages")},R=()=>{w(null)};return u.jsxs("div",{className:"app",children:[u.jsxs("header",{className:"app-header",children:[u.jsxs("div",{className:"header-brand",children:[u.jsx("div",{className:"brand-logo",children:_i.logo}),u.jsxs("div",{className:"brand-text",children:[u.jsx("h1",{children:"AILANG"}),u.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),u.jsxs("nav",{className:"header-nav",children:[u.jsxs("button",{className:`nav-tab ${e==="messages"?"active":""}`,onClick:()=>t("messages"),children:[u.jsx("span",{className:"nav-icon",children:_i.messages}),u.jsx("span",{className:"nav-label",children:"Messages"})]}),u.jsxs("button",{className:`nav-tab ${e==="approvals"?"active":""}`,onClick:()=>t("approvals"),children:[u.jsx("span",{className:"nav-icon",children:_i.shield}),u.jsx("span",{className:"nav-label",children:"Approvals"}),C>0&&u.jsx("span",{className:"nav-badge",children:C})]}),u.jsxs("button",{className:`nav-tab ${e==="monitor"?"active":""}`,onClick:()=>t("monitor"),children:[u.jsx("span",{className:"nav-icon",children:_i.activity}),u.jsx("span",{className:"nav-label",children:"Monitor"})]})]}),u.jsxs("div",{className:"header-meta",children:[u.jsxs("div",{className:"agent-selector",children:[u.jsx("label",{className:"agent-label",children:"Target:"}),m?u.jsxs("div",{className:"custom-agent-input",children:[u.jsx("input",{type:"text",value:d,onChange:j=>p(j.target.value),onKeyDown:j=>j.key==="Enter"&&y(),className:"agent-input",placeholder:"agent-id",autoFocus:!0}),u.jsx("button",{onClick:y,className:"agent-apply",children:"Add"}),u.jsx("button",{onClick:()=>f(!1),className:"agent-cancel",children:"Cancel"})]}):u.jsxs(u.Fragment,{children:[u.jsxs("select",{value:o,onChange:v,className:"agent-select",children:[s.map(j=>u.jsxs("option",{value:j.id,children:[_(j)," ",j.id]},j.id)),!s.find(j=>j.id===o)&&o&&u.jsxs("option",{value:o,children:["○ ",o]}),u.jsx("option",{value:"__custom__",children:"+ Add custom..."})]}),s.find(j=>j.id===o)&&u.jsx("span",{className:`agent-status ${b(s.find(j=>j.id===o))?"active":"inactive"}`,children:b(s.find(j=>j.id===o))?"Online":"Offline"})]})]}),u.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),u.jsxs("main",{className:"app-content",children:[e==="messages"&&u.jsx(o0,{websocketUrl:h,instanceId:o,initialThreadId:k,onThreadNavigated:R}),e==="approvals"&&u.jsx(a0,{approvals:n,history:i,onApprove:S,onReject:L,onNavigateToThread:T}),e==="monitor"&&u.jsx(s0,{})]}),u.jsx("style",{children:`
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
      `})]})};ko.createRoot(document.getElementById("root")).render(u.jsx(_t.StrictMode,{children:u.jsx(u0,{})}));
