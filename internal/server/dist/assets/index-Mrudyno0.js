(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Wi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Na(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Hc={exports:{}},yl={},$c={exports:{}},K={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ri=Symbol.for("react.element"),Vp=Symbol.for("react.portal"),Wp=Symbol.for("react.fragment"),Qp=Symbol.for("react.strict_mode"),qp=Symbol.for("react.profiler"),Kp=Symbol.for("react.provider"),Yp=Symbol.for("react.context"),Xp=Symbol.for("react.forward_ref"),Gp=Symbol.for("react.suspense"),Jp=Symbol.for("react.memo"),Zp=Symbol.for("react.lazy"),Hs=Symbol.iterator;function eh(e){return e===null||typeof e!="object"?null:(e=Hs&&e[Hs]||e["@@iterator"],typeof e=="function"?e:null)}var Vc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Wc=Object.assign,Qc={};function or(e,t,n){this.props=e,this.context=t,this.refs=Qc,this.updater=n||Vc}or.prototype.isReactComponent={};or.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};or.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function qc(){}qc.prototype=or.prototype;function _a(e,t,n){this.props=e,this.context=t,this.refs=Qc,this.updater=n||Vc}var za=_a.prototype=new qc;za.constructor=_a;Wc(za,or.prototype);za.isPureReactComponent=!0;var $s=Array.isArray,Kc=Object.prototype.hasOwnProperty,Pa={current:null},Yc={key:!0,ref:!0,__self:!0,__source:!0};function Xc(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)Kc.call(t,r)&&!Yc.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),c=0;c<a;c++)s[c]=arguments[c+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:ri,type:e,key:l,ref:o,props:i,_owner:Pa.current}}function th(e,t){return{$$typeof:ri,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function Ta(e){return typeof e=="object"&&e!==null&&e.$$typeof===ri}function nh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Vs=/\/+/g;function Ol(e,t){return typeof e=="object"&&e!==null&&e.key!=null?nh(""+e.key):t.toString(36)}function Ti(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case ri:case Vp:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Ol(o,0):r,$s(i)?(n="",e!=null&&(n=e.replace(Vs,"$&/")+"/"),Ti(i,t,n,"",function(c){return c})):i!=null&&(Ta(i)&&(i=th(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Vs,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",$s(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Ol(l,a);o+=Ti(l,t,n,s,i)}else if(s=eh(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Ol(l,a++),o+=Ti(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function ci(e,t,n){if(e==null)return e;var r=[],i=0;return Ti(e,r,"","",function(l){return t.call(n,l,i++)}),r}function rh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Re={current:null},Li={transition:null},ih={ReactCurrentDispatcher:Re,ReactCurrentBatchConfig:Li,ReactCurrentOwner:Pa};function Gc(){throw Error("act(...) is not supported in production builds of React.")}K.Children={map:ci,forEach:function(e,t,n){ci(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return ci(e,function(){t++}),t},toArray:function(e){return ci(e,function(t){return t})||[]},only:function(e){if(!Ta(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};K.Component=or;K.Fragment=Wp;K.Profiler=qp;K.PureComponent=_a;K.StrictMode=Qp;K.Suspense=Gp;K.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=ih;K.act=Gc;K.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Wc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Pa.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)Kc.call(t,s)&&!Yc.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var c=0;c<s;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:ri,type:e.type,key:i,ref:l,props:r,_owner:o}};K.createContext=function(e){return e={$$typeof:Yp,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:Kp,_context:e},e.Consumer=e};K.createElement=Xc;K.createFactory=function(e){var t=Xc.bind(null,e);return t.type=e,t};K.createRef=function(){return{current:null}};K.forwardRef=function(e){return{$$typeof:Xp,render:e}};K.isValidElement=Ta;K.lazy=function(e){return{$$typeof:Zp,_payload:{_status:-1,_result:e},_init:rh}};K.memo=function(e,t){return{$$typeof:Jp,type:e,compare:t===void 0?null:t}};K.startTransition=function(e){var t=Li.transition;Li.transition={};try{e()}finally{Li.transition=t}};K.unstable_act=Gc;K.useCallback=function(e,t){return Re.current.useCallback(e,t)};K.useContext=function(e){return Re.current.useContext(e)};K.useDebugValue=function(){};K.useDeferredValue=function(e){return Re.current.useDeferredValue(e)};K.useEffect=function(e,t){return Re.current.useEffect(e,t)};K.useId=function(){return Re.current.useId()};K.useImperativeHandle=function(e,t,n){return Re.current.useImperativeHandle(e,t,n)};K.useInsertionEffect=function(e,t){return Re.current.useInsertionEffect(e,t)};K.useLayoutEffect=function(e,t){return Re.current.useLayoutEffect(e,t)};K.useMemo=function(e,t){return Re.current.useMemo(e,t)};K.useReducer=function(e,t,n){return Re.current.useReducer(e,t,n)};K.useRef=function(e){return Re.current.useRef(e)};K.useState=function(e){return Re.current.useState(e)};K.useSyncExternalStore=function(e,t,n){return Re.current.useSyncExternalStore(e,t,n)};K.useTransition=function(){return Re.current.useTransition()};K.version="18.3.1";$c.exports=K;var B=$c.exports;const Wt=Na(B);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var lh=B,oh=Symbol.for("react.element"),ah=Symbol.for("react.fragment"),sh=Object.prototype.hasOwnProperty,uh=lh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,ch={key:!0,ref:!0,__self:!0,__source:!0};function Jc(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)sh.call(t,r)&&!ch.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:oh,type:e,key:l,ref:o,props:i,_owner:uh.current}}yl.Fragment=ah;yl.jsx=Jc;yl.jsxs=Jc;Hc.exports=yl;var u=Hc.exports,bo={},Zc={exports:{}},nt={},ed={exports:{}},td={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(C,E){var v=C.length;C.push(E);e:for(;0<v;){var T=v-1>>>1,F=C[T];if(0<i(F,E))C[T]=E,C[v]=F,v=T;else break e}}function n(C){return C.length===0?null:C[0]}function r(C){if(C.length===0)return null;var E=C[0],v=C.pop();if(v!==E){C[0]=v;e:for(var T=0,F=C.length,x=F>>>1;T<x;){var te=2*(T+1)-1,Se=C[te],ee=te+1,Ie=C[ee];if(0>i(Se,v))ee<F&&0>i(Ie,Se)?(C[T]=Ie,C[ee]=v,T=ee):(C[T]=Se,C[te]=v,T=te);else if(ee<F&&0>i(Ie,v))C[T]=Ie,C[ee]=v,T=ee;else break e}}return E}function i(C,E){var v=C.sortIndex-E.sortIndex;return v!==0?v:C.id-E.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],c=[],d=1,f=null,g=3,m=!1,S=!1,b=!1,N=typeof setTimeout=="function"?setTimeout:null,p=typeof clearTimeout=="function"?clearTimeout:null,h=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(C){for(var E=n(c);E!==null;){if(E.callback===null)r(c);else if(E.startTime<=C)r(c),E.sortIndex=E.expirationTime,t(s,E);else break;E=n(c)}}function k(C){if(b=!1,y(C),!S)if(n(s)!==null)S=!0,W(j);else{var E=n(c);E!==null&&re(k,E.startTime-C)}}function j(C,E){S=!1,b&&(b=!1,p(A),A=-1),m=!0;var v=g;try{for(y(E),f=n(s);f!==null&&(!(f.expirationTime>E)||C&&!z());){var T=f.callback;if(typeof T=="function"){f.callback=null,g=f.priorityLevel;var F=T(f.expirationTime<=E);E=e.unstable_now(),typeof F=="function"?f.callback=F:f===n(s)&&r(s),y(E)}else r(s);f=n(s)}if(f!==null)var x=!0;else{var te=n(c);te!==null&&re(k,te.startTime-E),x=!1}return x}finally{f=null,g=v,m=!1}}var w=!1,P=null,A=-1,U=5,R=-1;function z(){return!(e.unstable_now()-R<U)}function M(){if(P!==null){var C=e.unstable_now();R=C;var E=!0;try{E=P(!0,C)}finally{E?q():(w=!1,P=null)}}else w=!1}var q;if(typeof h=="function")q=function(){h(M)};else if(typeof MessageChannel<"u"){var Y=new MessageChannel,H=Y.port2;Y.port1.onmessage=M,q=function(){H.postMessage(null)}}else q=function(){N(M,0)};function W(C){P=C,w||(w=!0,q())}function re(C,E){A=N(function(){C(e.unstable_now())},E)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(C){C.callback=null},e.unstable_continueExecution=function(){S||m||(S=!0,W(j))},e.unstable_forceFrameRate=function(C){0>C||125<C?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):U=0<C?Math.floor(1e3/C):5},e.unstable_getCurrentPriorityLevel=function(){return g},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(C){switch(g){case 1:case 2:case 3:var E=3;break;default:E=g}var v=g;g=E;try{return C()}finally{g=v}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(C,E){switch(C){case 1:case 2:case 3:case 4:case 5:break;default:C=3}var v=g;g=C;try{return E()}finally{g=v}},e.unstable_scheduleCallback=function(C,E,v){var T=e.unstable_now();switch(typeof v=="object"&&v!==null?(v=v.delay,v=typeof v=="number"&&0<v?T+v:T):v=T,C){case 1:var F=-1;break;case 2:F=250;break;case 5:F=1073741823;break;case 4:F=1e4;break;default:F=5e3}return F=v+F,C={id:d++,callback:E,priorityLevel:C,startTime:v,expirationTime:F,sortIndex:-1},v>T?(C.sortIndex=v,t(c,C),n(s)===null&&C===n(c)&&(b?(p(A),A=-1):b=!0,re(k,v-T))):(C.sortIndex=F,t(s,C),S||m||(S=!0,W(j))),C},e.unstable_shouldYield=z,e.unstable_wrapCallback=function(C){var E=g;return function(){var v=g;g=E;try{return C.apply(this,arguments)}finally{g=v}}}})(td);ed.exports=td;var dh=ed.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var fh=B,tt=dh;function L(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var nd=new Set,Fr={};function Nn(e,t){Zn(e,t),Zn(e+"Capture",t)}function Zn(e,t){for(Fr[e]=t,e=0;e<t.length;e++)nd.add(t[e])}var Dt=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),Co=Object.prototype.hasOwnProperty,ph=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,Ws={},Qs={};function hh(e){return Co.call(Qs,e)?!0:Co.call(Ws,e)?!1:ph.test(e)?Qs[e]=!0:(Ws[e]=!0,!1)}function mh(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function gh(e,t,n,r){if(t===null||typeof t>"u"||mh(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Oe(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ee={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ee[e]=new Oe(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ee[t]=new Oe(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ee[e]=new Oe(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ee[e]=new Oe(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ee[e]=new Oe(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ee[e]=new Oe(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ee[e]=new Oe(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ee[e]=new Oe(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ee[e]=new Oe(e,5,!1,e.toLowerCase(),null,!1,!1)});var La=/[\-:]([a-z])/g;function Ia(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(La,Ia);Ee[t]=new Oe(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(La,Ia);Ee[t]=new Oe(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(La,Ia);Ee[t]=new Oe(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ee[e]=new Oe(e,1,!1,e.toLowerCase(),null,!1,!1)});Ee.xlinkHref=new Oe("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ee[e]=new Oe(e,1,!1,e.toLowerCase(),null,!0,!0)});function Aa(e,t,n,r){var i=Ee.hasOwnProperty(t)?Ee[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(gh(t,n,i,r)&&(n=null),r||i===null?hh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Bt=fh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,di=Symbol.for("react.element"),In=Symbol.for("react.portal"),An=Symbol.for("react.fragment"),Ma=Symbol.for("react.strict_mode"),jo=Symbol.for("react.profiler"),rd=Symbol.for("react.provider"),id=Symbol.for("react.context"),Da=Symbol.for("react.forward_ref"),Eo=Symbol.for("react.suspense"),No=Symbol.for("react.suspense_list"),Ra=Symbol.for("react.memo"),Qt=Symbol.for("react.lazy"),ld=Symbol.for("react.offscreen"),qs=Symbol.iterator;function pr(e){return e===null||typeof e!="object"?null:(e=qs&&e[qs]||e["@@iterator"],typeof e=="function"?e:null)}var pe=Object.assign,Fl;function br(e){if(Fl===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Fl=t&&t[1]||""}return`
`+Fl+e}var Bl=!1;function Ul(e,t){if(!e||Bl)return"";Bl=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Bl=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?br(e):""}function vh(e){switch(e.tag){case 5:return br(e.type);case 16:return br("Lazy");case 13:return br("Suspense");case 19:return br("SuspenseList");case 0:case 2:case 15:return e=Ul(e.type,!1),e;case 11:return e=Ul(e.type.render,!1),e;case 1:return e=Ul(e.type,!0),e;default:return""}}function _o(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case An:return"Fragment";case In:return"Portal";case jo:return"Profiler";case Ma:return"StrictMode";case Eo:return"Suspense";case No:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case id:return(e.displayName||"Context")+".Consumer";case rd:return(e._context.displayName||"Context")+".Provider";case Da:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Ra:return t=e.displayName||null,t!==null?t:_o(e.type)||"Memo";case Qt:t=e._payload,e=e._init;try{return _o(e(t))}catch{}}return null}function yh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return _o(t);case 8:return t===Ma?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function an(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function od(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function xh(e){var t=od(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function fi(e){e._valueTracker||(e._valueTracker=xh(e))}function ad(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=od(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Qi(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function zo(e,t){var n=t.checked;return pe({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function Ks(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=an(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function sd(e,t){t=t.checked,t!=null&&Aa(e,"checked",t,!1)}function Po(e,t){sd(e,t);var n=an(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?To(e,t.type,n):t.hasOwnProperty("defaultValue")&&To(e,t.type,an(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function Ys(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function To(e,t,n){(t!=="number"||Qi(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var Cr=Array.isArray;function Wn(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+an(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function Lo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(L(91));return pe({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function Xs(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(L(92));if(Cr(n)){if(1<n.length)throw Error(L(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:an(n)}}function ud(e,t){var n=an(t.value),r=an(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function Gs(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function cd(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Io(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?cd(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var pi,dd=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(pi=pi||document.createElement("div"),pi.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=pi.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Br(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Nr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},kh=["Webkit","ms","Moz","O"];Object.keys(Nr).forEach(function(e){kh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Nr[t]=Nr[e]})});function fd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Nr.hasOwnProperty(e)&&Nr[e]?(""+t).trim():t+"px"}function pd(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=fd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var wh=pe({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Ao(e,t){if(t){if(wh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(L(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(L(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(L(61))}if(t.style!=null&&typeof t.style!="object")throw Error(L(62))}}function Mo(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Do=null;function Oa(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Ro=null,Qn=null,qn=null;function Js(e){if(e=oi(e)){if(typeof Ro!="function")throw Error(L(280));var t=e.stateNode;t&&(t=bl(t),Ro(e.stateNode,e.type,t))}}function hd(e){Qn?qn?qn.push(e):qn=[e]:Qn=e}function md(){if(Qn){var e=Qn,t=qn;if(qn=Qn=null,Js(e),t)for(e=0;e<t.length;e++)Js(t[e])}}function gd(e,t){return e(t)}function vd(){}var Hl=!1;function yd(e,t,n){if(Hl)return e(t,n);Hl=!0;try{return gd(e,t,n)}finally{Hl=!1,(Qn!==null||qn!==null)&&(vd(),md())}}function Ur(e,t){var n=e.stateNode;if(n===null)return null;var r=bl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(L(231,t,typeof n));return n}var Oo=!1;if(Dt)try{var hr={};Object.defineProperty(hr,"passive",{get:function(){Oo=!0}}),window.addEventListener("test",hr,hr),window.removeEventListener("test",hr,hr)}catch{Oo=!1}function Sh(e,t,n,r,i,l,o,a,s){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var _r=!1,qi=null,Ki=!1,Fo=null,bh={onError:function(e){_r=!0,qi=e}};function Ch(e,t,n,r,i,l,o,a,s){_r=!1,qi=null,Sh.apply(bh,arguments)}function jh(e,t,n,r,i,l,o,a,s){if(Ch.apply(this,arguments),_r){if(_r){var c=qi;_r=!1,qi=null}else throw Error(L(198));Ki||(Ki=!0,Fo=c)}}function _n(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function xd(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function Zs(e){if(_n(e)!==e)throw Error(L(188))}function Eh(e){var t=e.alternate;if(!t){if(t=_n(e),t===null)throw Error(L(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return Zs(i),e;if(l===r)return Zs(i),t;l=l.sibling}throw Error(L(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(L(189))}}if(n.alternate!==r)throw Error(L(190))}if(n.tag!==3)throw Error(L(188));return n.stateNode.current===n?e:t}function kd(e){return e=Eh(e),e!==null?wd(e):null}function wd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=wd(e);if(t!==null)return t;e=e.sibling}return null}var Sd=tt.unstable_scheduleCallback,eu=tt.unstable_cancelCallback,Nh=tt.unstable_shouldYield,_h=tt.unstable_requestPaint,me=tt.unstable_now,zh=tt.unstable_getCurrentPriorityLevel,Fa=tt.unstable_ImmediatePriority,bd=tt.unstable_UserBlockingPriority,Yi=tt.unstable_NormalPriority,Ph=tt.unstable_LowPriority,Cd=tt.unstable_IdlePriority,xl=null,jt=null;function Th(e){if(jt&&typeof jt.onCommitFiberRoot=="function")try{jt.onCommitFiberRoot(xl,e,void 0,(e.current.flags&128)===128)}catch{}}var vt=Math.clz32?Math.clz32:Ah,Lh=Math.log,Ih=Math.LN2;function Ah(e){return e>>>=0,e===0?32:31-(Lh(e)/Ih|0)|0}var hi=64,mi=4194304;function jr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function Xi(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=jr(a):(l&=o,l!==0&&(r=jr(l)))}else o=n&~i,o!==0?r=jr(o):l!==0&&(r=jr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-vt(t),i=1<<n,r|=e[n],t&=~i;return r}function Mh(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Dh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-vt(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Mh(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Bo(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function jd(){var e=hi;return hi<<=1,!(hi&4194240)&&(hi=64),e}function $l(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function ii(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-vt(t),e[t]=n}function Rh(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-vt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Ba(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-vt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var ne=0;function Ed(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Nd,Ua,_d,zd,Pd,Uo=!1,gi=[],Jt=null,Zt=null,en=null,Hr=new Map,$r=new Map,Kt=[],Oh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function tu(e,t){switch(e){case"focusin":case"focusout":Jt=null;break;case"dragenter":case"dragleave":Zt=null;break;case"mouseover":case"mouseout":en=null;break;case"pointerover":case"pointerout":Hr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":$r.delete(t.pointerId)}}function mr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=oi(t),t!==null&&Ua(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Fh(e,t,n,r,i){switch(t){case"focusin":return Jt=mr(Jt,e,t,n,r,i),!0;case"dragenter":return Zt=mr(Zt,e,t,n,r,i),!0;case"mouseover":return en=mr(en,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Hr.set(l,mr(Hr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,$r.set(l,mr($r.get(l)||null,e,t,n,r,i)),!0}return!1}function Td(e){var t=vn(e.target);if(t!==null){var n=_n(t);if(n!==null){if(t=n.tag,t===13){if(t=xd(n),t!==null){e.blockedOn=t,Pd(e.priority,function(){_d(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Ii(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Ho(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Do=r,n.target.dispatchEvent(r),Do=null}else return t=oi(n),t!==null&&Ua(t),e.blockedOn=n,!1;t.shift()}return!0}function nu(e,t,n){Ii(e)&&n.delete(t)}function Bh(){Uo=!1,Jt!==null&&Ii(Jt)&&(Jt=null),Zt!==null&&Ii(Zt)&&(Zt=null),en!==null&&Ii(en)&&(en=null),Hr.forEach(nu),$r.forEach(nu)}function gr(e,t){e.blockedOn===t&&(e.blockedOn=null,Uo||(Uo=!0,tt.unstable_scheduleCallback(tt.unstable_NormalPriority,Bh)))}function Vr(e){function t(i){return gr(i,e)}if(0<gi.length){gr(gi[0],e);for(var n=1;n<gi.length;n++){var r=gi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(Jt!==null&&gr(Jt,e),Zt!==null&&gr(Zt,e),en!==null&&gr(en,e),Hr.forEach(t),$r.forEach(t),n=0;n<Kt.length;n++)r=Kt[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Kt.length&&(n=Kt[0],n.blockedOn===null);)Td(n),n.blockedOn===null&&Kt.shift()}var Kn=Bt.ReactCurrentBatchConfig,Gi=!0;function Uh(e,t,n,r){var i=ne,l=Kn.transition;Kn.transition=null;try{ne=1,Ha(e,t,n,r)}finally{ne=i,Kn.transition=l}}function Hh(e,t,n,r){var i=ne,l=Kn.transition;Kn.transition=null;try{ne=4,Ha(e,t,n,r)}finally{ne=i,Kn.transition=l}}function Ha(e,t,n,r){if(Gi){var i=Ho(e,t,n,r);if(i===null)Zl(e,t,r,Ji,n),tu(e,r);else if(Fh(i,e,t,n,r))r.stopPropagation();else if(tu(e,r),t&4&&-1<Oh.indexOf(e)){for(;i!==null;){var l=oi(i);if(l!==null&&Nd(l),l=Ho(e,t,n,r),l===null&&Zl(e,t,r,Ji,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else Zl(e,t,r,null,n)}}var Ji=null;function Ho(e,t,n,r){if(Ji=null,e=Oa(r),e=vn(e),e!==null)if(t=_n(e),t===null)e=null;else if(n=t.tag,n===13){if(e=xd(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return Ji=e,null}function Ld(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(zh()){case Fa:return 1;case bd:return 4;case Yi:case Ph:return 16;case Cd:return 536870912;default:return 16}default:return 16}}var Xt=null,$a=null,Ai=null;function Id(){if(Ai)return Ai;var e,t=$a,n=t.length,r,i="value"in Xt?Xt.value:Xt.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Ai=i.slice(e,1<r?1-r:void 0)}function Mi(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function vi(){return!0}function ru(){return!1}function rt(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?vi:ru,this.isPropagationStopped=ru,this}return pe(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=vi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=vi)},persist:function(){},isPersistent:vi}),t}var ar={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Va=rt(ar),li=pe({},ar,{view:0,detail:0}),$h=rt(li),Vl,Wl,vr,kl=pe({},li,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:Wa,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==vr&&(vr&&e.type==="mousemove"?(Vl=e.screenX-vr.screenX,Wl=e.screenY-vr.screenY):Wl=Vl=0,vr=e),Vl)},movementY:function(e){return"movementY"in e?e.movementY:Wl}}),iu=rt(kl),Vh=pe({},kl,{dataTransfer:0}),Wh=rt(Vh),Qh=pe({},li,{relatedTarget:0}),Ql=rt(Qh),qh=pe({},ar,{animationName:0,elapsedTime:0,pseudoElement:0}),Kh=rt(qh),Yh=pe({},ar,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),Xh=rt(Yh),Gh=pe({},ar,{data:0}),lu=rt(Gh),Jh={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},Zh={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},em={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function tm(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=em[e])?!!t[e]:!1}function Wa(){return tm}var nm=pe({},li,{key:function(e){if(e.key){var t=Jh[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Mi(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?Zh[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:Wa,charCode:function(e){return e.type==="keypress"?Mi(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Mi(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),rm=rt(nm),im=pe({},kl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),ou=rt(im),lm=pe({},li,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:Wa}),om=rt(lm),am=pe({},ar,{propertyName:0,elapsedTime:0,pseudoElement:0}),sm=rt(am),um=pe({},kl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),cm=rt(um),dm=[9,13,27,32],Qa=Dt&&"CompositionEvent"in window,zr=null;Dt&&"documentMode"in document&&(zr=document.documentMode);var fm=Dt&&"TextEvent"in window&&!zr,Ad=Dt&&(!Qa||zr&&8<zr&&11>=zr),au=" ",su=!1;function Md(e,t){switch(e){case"keyup":return dm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Dd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Mn=!1;function pm(e,t){switch(e){case"compositionend":return Dd(t);case"keypress":return t.which!==32?null:(su=!0,au);case"textInput":return e=t.data,e===au&&su?null:e;default:return null}}function hm(e,t){if(Mn)return e==="compositionend"||!Qa&&Md(e,t)?(e=Id(),Ai=$a=Xt=null,Mn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Ad&&t.locale!=="ko"?null:t.data;default:return null}}var mm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function uu(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!mm[e.type]:t==="textarea"}function Rd(e,t,n,r){hd(r),t=Zi(t,"onChange"),0<t.length&&(n=new Va("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Pr=null,Wr=null;function gm(e){Kd(e,0)}function wl(e){var t=On(e);if(ad(t))return e}function vm(e,t){if(e==="change")return t}var Od=!1;if(Dt){var ql;if(Dt){var Kl="oninput"in document;if(!Kl){var cu=document.createElement("div");cu.setAttribute("oninput","return;"),Kl=typeof cu.oninput=="function"}ql=Kl}else ql=!1;Od=ql&&(!document.documentMode||9<document.documentMode)}function du(){Pr&&(Pr.detachEvent("onpropertychange",Fd),Wr=Pr=null)}function Fd(e){if(e.propertyName==="value"&&wl(Wr)){var t=[];Rd(t,Wr,e,Oa(e)),yd(gm,t)}}function ym(e,t,n){e==="focusin"?(du(),Pr=t,Wr=n,Pr.attachEvent("onpropertychange",Fd)):e==="focusout"&&du()}function xm(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return wl(Wr)}function km(e,t){if(e==="click")return wl(t)}function wm(e,t){if(e==="input"||e==="change")return wl(t)}function Sm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var xt=typeof Object.is=="function"?Object.is:Sm;function Qr(e,t){if(xt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!Co.call(t,i)||!xt(e[i],t[i]))return!1}return!0}function fu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function pu(e,t){var n=fu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=fu(n)}}function Bd(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Bd(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Ud(){for(var e=window,t=Qi();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Qi(e.document)}return t}function qa(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function bm(e){var t=Ud(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Bd(n.ownerDocument.documentElement,n)){if(r!==null&&qa(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=pu(n,l);var o=pu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Cm=Dt&&"documentMode"in document&&11>=document.documentMode,Dn=null,$o=null,Tr=null,Vo=!1;function hu(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Vo||Dn==null||Dn!==Qi(r)||(r=Dn,"selectionStart"in r&&qa(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Tr&&Qr(Tr,r)||(Tr=r,r=Zi($o,"onSelect"),0<r.length&&(t=new Va("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Dn)))}function yi(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Rn={animationend:yi("Animation","AnimationEnd"),animationiteration:yi("Animation","AnimationIteration"),animationstart:yi("Animation","AnimationStart"),transitionend:yi("Transition","TransitionEnd")},Yl={},Hd={};Dt&&(Hd=document.createElement("div").style,"AnimationEvent"in window||(delete Rn.animationend.animation,delete Rn.animationiteration.animation,delete Rn.animationstart.animation),"TransitionEvent"in window||delete Rn.transitionend.transition);function Sl(e){if(Yl[e])return Yl[e];if(!Rn[e])return e;var t=Rn[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Hd)return Yl[e]=t[n];return e}var $d=Sl("animationend"),Vd=Sl("animationiteration"),Wd=Sl("animationstart"),Qd=Sl("transitionend"),qd=new Map,mu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function un(e,t){qd.set(e,t),Nn(t,[e])}for(var Xl=0;Xl<mu.length;Xl++){var Gl=mu[Xl],jm=Gl.toLowerCase(),Em=Gl[0].toUpperCase()+Gl.slice(1);un(jm,"on"+Em)}un($d,"onAnimationEnd");un(Vd,"onAnimationIteration");un(Wd,"onAnimationStart");un("dblclick","onDoubleClick");un("focusin","onFocus");un("focusout","onBlur");un(Qd,"onTransitionEnd");Zn("onMouseEnter",["mouseout","mouseover"]);Zn("onMouseLeave",["mouseout","mouseover"]);Zn("onPointerEnter",["pointerout","pointerover"]);Zn("onPointerLeave",["pointerout","pointerover"]);Nn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));Nn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));Nn("onBeforeInput",["compositionend","keypress","textInput","paste"]);Nn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));Nn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));Nn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var Er="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Nm=new Set("cancel close invalid load scroll toggle".split(" ").concat(Er));function gu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,jh(r,t,void 0,e),e.currentTarget=null}function Kd(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,c=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;gu(i,a,c),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,c=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;gu(i,a,c),l=s}}}if(Ki)throw e=Fo,Ki=!1,Fo=null,e}function se(e,t){var n=t[Yo];n===void 0&&(n=t[Yo]=new Set);var r=e+"__bubble";n.has(r)||(Yd(t,e,2,!1),n.add(r))}function Jl(e,t,n){var r=0;t&&(r|=4),Yd(n,e,r,t)}var xi="_reactListening"+Math.random().toString(36).slice(2);function qr(e){if(!e[xi]){e[xi]=!0,nd.forEach(function(n){n!=="selectionchange"&&(Nm.has(n)||Jl(n,!1,e),Jl(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[xi]||(t[xi]=!0,Jl("selectionchange",!1,t))}}function Yd(e,t,n,r){switch(Ld(t)){case 1:var i=Uh;break;case 4:i=Hh;break;default:i=Ha}n=i.bind(null,t,n,e),i=void 0,!Oo||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function Zl(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=vn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}yd(function(){var c=l,d=Oa(n),f=[];e:{var g=qd.get(e);if(g!==void 0){var m=Va,S=e;switch(e){case"keypress":if(Mi(n)===0)break e;case"keydown":case"keyup":m=rm;break;case"focusin":S="focus",m=Ql;break;case"focusout":S="blur",m=Ql;break;case"beforeblur":case"afterblur":m=Ql;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":m=iu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":m=Wh;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":m=om;break;case $d:case Vd:case Wd:m=Kh;break;case Qd:m=sm;break;case"scroll":m=$h;break;case"wheel":m=cm;break;case"copy":case"cut":case"paste":m=Xh;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":m=ou}var b=(t&4)!==0,N=!b&&e==="scroll",p=b?g!==null?g+"Capture":null:g;b=[];for(var h=c,y;h!==null;){y=h;var k=y.stateNode;if(y.tag===5&&k!==null&&(y=k,p!==null&&(k=Ur(h,p),k!=null&&b.push(Kr(h,k,y)))),N)break;h=h.return}0<b.length&&(g=new m(g,S,null,n,d),f.push({event:g,listeners:b}))}}if(!(t&7)){e:{if(g=e==="mouseover"||e==="pointerover",m=e==="mouseout"||e==="pointerout",g&&n!==Do&&(S=n.relatedTarget||n.fromElement)&&(vn(S)||S[Rt]))break e;if((m||g)&&(g=d.window===d?d:(g=d.ownerDocument)?g.defaultView||g.parentWindow:window,m?(S=n.relatedTarget||n.toElement,m=c,S=S?vn(S):null,S!==null&&(N=_n(S),S!==N||S.tag!==5&&S.tag!==6)&&(S=null)):(m=null,S=c),m!==S)){if(b=iu,k="onMouseLeave",p="onMouseEnter",h="mouse",(e==="pointerout"||e==="pointerover")&&(b=ou,k="onPointerLeave",p="onPointerEnter",h="pointer"),N=m==null?g:On(m),y=S==null?g:On(S),g=new b(k,h+"leave",m,n,d),g.target=N,g.relatedTarget=y,k=null,vn(d)===c&&(b=new b(p,h+"enter",S,n,d),b.target=y,b.relatedTarget=N,k=b),N=k,m&&S)t:{for(b=m,p=S,h=0,y=b;y;y=Tn(y))h++;for(y=0,k=p;k;k=Tn(k))y++;for(;0<h-y;)b=Tn(b),h--;for(;0<y-h;)p=Tn(p),y--;for(;h--;){if(b===p||p!==null&&b===p.alternate)break t;b=Tn(b),p=Tn(p)}b=null}else b=null;m!==null&&vu(f,g,m,b,!1),S!==null&&N!==null&&vu(f,N,S,b,!0)}}e:{if(g=c?On(c):window,m=g.nodeName&&g.nodeName.toLowerCase(),m==="select"||m==="input"&&g.type==="file")var j=vm;else if(uu(g))if(Od)j=wm;else{j=xm;var w=ym}else(m=g.nodeName)&&m.toLowerCase()==="input"&&(g.type==="checkbox"||g.type==="radio")&&(j=km);if(j&&(j=j(e,c))){Rd(f,j,n,d);break e}w&&w(e,g,c),e==="focusout"&&(w=g._wrapperState)&&w.controlled&&g.type==="number"&&To(g,"number",g.value)}switch(w=c?On(c):window,e){case"focusin":(uu(w)||w.contentEditable==="true")&&(Dn=w,$o=c,Tr=null);break;case"focusout":Tr=$o=Dn=null;break;case"mousedown":Vo=!0;break;case"contextmenu":case"mouseup":case"dragend":Vo=!1,hu(f,n,d);break;case"selectionchange":if(Cm)break;case"keydown":case"keyup":hu(f,n,d)}var P;if(Qa)e:{switch(e){case"compositionstart":var A="onCompositionStart";break e;case"compositionend":A="onCompositionEnd";break e;case"compositionupdate":A="onCompositionUpdate";break e}A=void 0}else Mn?Md(e,n)&&(A="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(A="onCompositionStart");A&&(Ad&&n.locale!=="ko"&&(Mn||A!=="onCompositionStart"?A==="onCompositionEnd"&&Mn&&(P=Id()):(Xt=d,$a="value"in Xt?Xt.value:Xt.textContent,Mn=!0)),w=Zi(c,A),0<w.length&&(A=new lu(A,e,null,n,d),f.push({event:A,listeners:w}),P?A.data=P:(P=Dd(n),P!==null&&(A.data=P)))),(P=fm?pm(e,n):hm(e,n))&&(c=Zi(c,"onBeforeInput"),0<c.length&&(d=new lu("onBeforeInput","beforeinput",null,n,d),f.push({event:d,listeners:c}),d.data=P))}Kd(f,t)})}function Kr(e,t,n){return{instance:e,listener:t,currentTarget:n}}function Zi(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Ur(e,n),l!=null&&r.unshift(Kr(e,l,i)),l=Ur(e,t),l!=null&&r.push(Kr(e,l,i))),e=e.return}return r}function Tn(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function vu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,c=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&c!==null&&(a=c,i?(s=Ur(n,l),s!=null&&o.unshift(Kr(n,s,a))):i||(s=Ur(n,l),s!=null&&o.push(Kr(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var _m=/\r\n?/g,zm=/\u0000|\uFFFD/g;function yu(e){return(typeof e=="string"?e:""+e).replace(_m,`
`).replace(zm,"")}function ki(e,t,n){if(t=yu(t),yu(e)!==t&&n)throw Error(L(425))}function el(){}var Wo=null,Qo=null;function qo(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var Ko=typeof setTimeout=="function"?setTimeout:void 0,Pm=typeof clearTimeout=="function"?clearTimeout:void 0,xu=typeof Promise=="function"?Promise:void 0,Tm=typeof queueMicrotask=="function"?queueMicrotask:typeof xu<"u"?function(e){return xu.resolve(null).then(e).catch(Lm)}:Ko;function Lm(e){setTimeout(function(){throw e})}function eo(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Vr(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Vr(t)}function tn(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function ku(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var sr=Math.random().toString(36).slice(2),bt="__reactFiber$"+sr,Yr="__reactProps$"+sr,Rt="__reactContainer$"+sr,Yo="__reactEvents$"+sr,Im="__reactListeners$"+sr,Am="__reactHandles$"+sr;function vn(e){var t=e[bt];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Rt]||n[bt]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=ku(e);e!==null;){if(n=e[bt])return n;e=ku(e)}return t}e=n,n=e.parentNode}return null}function oi(e){return e=e[bt]||e[Rt],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function On(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(L(33))}function bl(e){return e[Yr]||null}var Xo=[],Fn=-1;function cn(e){return{current:e}}function ue(e){0>Fn||(e.current=Xo[Fn],Xo[Fn]=null,Fn--)}function oe(e,t){Fn++,Xo[Fn]=e.current,e.current=t}var sn={},Te=cn(sn),$e=cn(!1),Sn=sn;function er(e,t){var n=e.type.contextTypes;if(!n)return sn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Ve(e){return e=e.childContextTypes,e!=null}function tl(){ue($e),ue(Te)}function wu(e,t,n){if(Te.current!==sn)throw Error(L(168));oe(Te,t),oe($e,n)}function Xd(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(L(108,yh(e)||"Unknown",i));return pe({},n,r)}function nl(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||sn,Sn=Te.current,oe(Te,e),oe($e,$e.current),!0}function Su(e,t,n){var r=e.stateNode;if(!r)throw Error(L(169));n?(e=Xd(e,t,Sn),r.__reactInternalMemoizedMergedChildContext=e,ue($e),ue(Te),oe(Te,e)):ue($e),oe($e,n)}var Lt=null,Cl=!1,to=!1;function Gd(e){Lt===null?Lt=[e]:Lt.push(e)}function Mm(e){Cl=!0,Gd(e)}function dn(){if(!to&&Lt!==null){to=!0;var e=0,t=ne;try{var n=Lt;for(ne=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}Lt=null,Cl=!1}catch(i){throw Lt!==null&&(Lt=Lt.slice(e+1)),Sd(Fa,dn),i}finally{ne=t,to=!1}}return null}var Bn=[],Un=0,rl=null,il=0,lt=[],ot=0,bn=null,It=1,At="";function hn(e,t){Bn[Un++]=il,Bn[Un++]=rl,rl=e,il=t}function Jd(e,t,n){lt[ot++]=It,lt[ot++]=At,lt[ot++]=bn,bn=e;var r=It;e=At;var i=32-vt(r)-1;r&=~(1<<i),n+=1;var l=32-vt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,It=1<<32-vt(t)+i|n<<i|r,At=l+e}else It=1<<l|n<<i|r,At=e}function Ka(e){e.return!==null&&(hn(e,1),Jd(e,1,0))}function Ya(e){for(;e===rl;)rl=Bn[--Un],Bn[Un]=null,il=Bn[--Un],Bn[Un]=null;for(;e===bn;)bn=lt[--ot],lt[ot]=null,At=lt[--ot],lt[ot]=null,It=lt[--ot],lt[ot]=null}var et=null,Je=null,ce=!1,gt=null;function Zd(e,t){var n=st(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function bu(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,et=e,Je=tn(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,et=e,Je=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=bn!==null?{id:It,overflow:At}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=st(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,et=e,Je=null,!0):!1;default:return!1}}function Go(e){return(e.mode&1)!==0&&(e.flags&128)===0}function Jo(e){if(ce){var t=Je;if(t){var n=t;if(!bu(e,t)){if(Go(e))throw Error(L(418));t=tn(n.nextSibling);var r=et;t&&bu(e,t)?Zd(r,n):(e.flags=e.flags&-4097|2,ce=!1,et=e)}}else{if(Go(e))throw Error(L(418));e.flags=e.flags&-4097|2,ce=!1,et=e}}}function Cu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;et=e}function wi(e){if(e!==et)return!1;if(!ce)return Cu(e),ce=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!qo(e.type,e.memoizedProps)),t&&(t=Je)){if(Go(e))throw ef(),Error(L(418));for(;t;)Zd(e,t),t=tn(t.nextSibling)}if(Cu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(L(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){Je=tn(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}Je=null}}else Je=et?tn(e.stateNode.nextSibling):null;return!0}function ef(){for(var e=Je;e;)e=tn(e.nextSibling)}function tr(){Je=et=null,ce=!1}function Xa(e){gt===null?gt=[e]:gt.push(e)}var Dm=Bt.ReactCurrentBatchConfig;function yr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(L(309));var r=n.stateNode}if(!r)throw Error(L(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(L(284));if(!n._owner)throw Error(L(290,e))}return e}function Si(e,t){throw e=Object.prototype.toString.call(t),Error(L(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function ju(e){var t=e._init;return t(e._payload)}function tf(e){function t(p,h){if(e){var y=p.deletions;y===null?(p.deletions=[h],p.flags|=16):y.push(h)}}function n(p,h){if(!e)return null;for(;h!==null;)t(p,h),h=h.sibling;return null}function r(p,h){for(p=new Map;h!==null;)h.key!==null?p.set(h.key,h):p.set(h.index,h),h=h.sibling;return p}function i(p,h){return p=on(p,h),p.index=0,p.sibling=null,p}function l(p,h,y){return p.index=y,e?(y=p.alternate,y!==null?(y=y.index,y<h?(p.flags|=2,h):y):(p.flags|=2,h)):(p.flags|=1048576,h)}function o(p){return e&&p.alternate===null&&(p.flags|=2),p}function a(p,h,y,k){return h===null||h.tag!==6?(h=so(y,p.mode,k),h.return=p,h):(h=i(h,y),h.return=p,h)}function s(p,h,y,k){var j=y.type;return j===An?d(p,h,y.props.children,k,y.key):h!==null&&(h.elementType===j||typeof j=="object"&&j!==null&&j.$$typeof===Qt&&ju(j)===h.type)?(k=i(h,y.props),k.ref=yr(p,h,y),k.return=p,k):(k=Hi(y.type,y.key,y.props,null,p.mode,k),k.ref=yr(p,h,y),k.return=p,k)}function c(p,h,y,k){return h===null||h.tag!==4||h.stateNode.containerInfo!==y.containerInfo||h.stateNode.implementation!==y.implementation?(h=uo(y,p.mode,k),h.return=p,h):(h=i(h,y.children||[]),h.return=p,h)}function d(p,h,y,k,j){return h===null||h.tag!==7?(h=wn(y,p.mode,k,j),h.return=p,h):(h=i(h,y),h.return=p,h)}function f(p,h,y){if(typeof h=="string"&&h!==""||typeof h=="number")return h=so(""+h,p.mode,y),h.return=p,h;if(typeof h=="object"&&h!==null){switch(h.$$typeof){case di:return y=Hi(h.type,h.key,h.props,null,p.mode,y),y.ref=yr(p,null,h),y.return=p,y;case In:return h=uo(h,p.mode,y),h.return=p,h;case Qt:var k=h._init;return f(p,k(h._payload),y)}if(Cr(h)||pr(h))return h=wn(h,p.mode,y,null),h.return=p,h;Si(p,h)}return null}function g(p,h,y,k){var j=h!==null?h.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return j!==null?null:a(p,h,""+y,k);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case di:return y.key===j?s(p,h,y,k):null;case In:return y.key===j?c(p,h,y,k):null;case Qt:return j=y._init,g(p,h,j(y._payload),k)}if(Cr(y)||pr(y))return j!==null?null:d(p,h,y,k,null);Si(p,y)}return null}function m(p,h,y,k,j){if(typeof k=="string"&&k!==""||typeof k=="number")return p=p.get(y)||null,a(h,p,""+k,j);if(typeof k=="object"&&k!==null){switch(k.$$typeof){case di:return p=p.get(k.key===null?y:k.key)||null,s(h,p,k,j);case In:return p=p.get(k.key===null?y:k.key)||null,c(h,p,k,j);case Qt:var w=k._init;return m(p,h,y,w(k._payload),j)}if(Cr(k)||pr(k))return p=p.get(y)||null,d(h,p,k,j,null);Si(h,k)}return null}function S(p,h,y,k){for(var j=null,w=null,P=h,A=h=0,U=null;P!==null&&A<y.length;A++){P.index>A?(U=P,P=null):U=P.sibling;var R=g(p,P,y[A],k);if(R===null){P===null&&(P=U);break}e&&P&&R.alternate===null&&t(p,P),h=l(R,h,A),w===null?j=R:w.sibling=R,w=R,P=U}if(A===y.length)return n(p,P),ce&&hn(p,A),j;if(P===null){for(;A<y.length;A++)P=f(p,y[A],k),P!==null&&(h=l(P,h,A),w===null?j=P:w.sibling=P,w=P);return ce&&hn(p,A),j}for(P=r(p,P);A<y.length;A++)U=m(P,p,A,y[A],k),U!==null&&(e&&U.alternate!==null&&P.delete(U.key===null?A:U.key),h=l(U,h,A),w===null?j=U:w.sibling=U,w=U);return e&&P.forEach(function(z){return t(p,z)}),ce&&hn(p,A),j}function b(p,h,y,k){var j=pr(y);if(typeof j!="function")throw Error(L(150));if(y=j.call(y),y==null)throw Error(L(151));for(var w=j=null,P=h,A=h=0,U=null,R=y.next();P!==null&&!R.done;A++,R=y.next()){P.index>A?(U=P,P=null):U=P.sibling;var z=g(p,P,R.value,k);if(z===null){P===null&&(P=U);break}e&&P&&z.alternate===null&&t(p,P),h=l(z,h,A),w===null?j=z:w.sibling=z,w=z,P=U}if(R.done)return n(p,P),ce&&hn(p,A),j;if(P===null){for(;!R.done;A++,R=y.next())R=f(p,R.value,k),R!==null&&(h=l(R,h,A),w===null?j=R:w.sibling=R,w=R);return ce&&hn(p,A),j}for(P=r(p,P);!R.done;A++,R=y.next())R=m(P,p,A,R.value,k),R!==null&&(e&&R.alternate!==null&&P.delete(R.key===null?A:R.key),h=l(R,h,A),w===null?j=R:w.sibling=R,w=R);return e&&P.forEach(function(M){return t(p,M)}),ce&&hn(p,A),j}function N(p,h,y,k){if(typeof y=="object"&&y!==null&&y.type===An&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case di:e:{for(var j=y.key,w=h;w!==null;){if(w.key===j){if(j=y.type,j===An){if(w.tag===7){n(p,w.sibling),h=i(w,y.props.children),h.return=p,p=h;break e}}else if(w.elementType===j||typeof j=="object"&&j!==null&&j.$$typeof===Qt&&ju(j)===w.type){n(p,w.sibling),h=i(w,y.props),h.ref=yr(p,w,y),h.return=p,p=h;break e}n(p,w);break}else t(p,w);w=w.sibling}y.type===An?(h=wn(y.props.children,p.mode,k,y.key),h.return=p,p=h):(k=Hi(y.type,y.key,y.props,null,p.mode,k),k.ref=yr(p,h,y),k.return=p,p=k)}return o(p);case In:e:{for(w=y.key;h!==null;){if(h.key===w)if(h.tag===4&&h.stateNode.containerInfo===y.containerInfo&&h.stateNode.implementation===y.implementation){n(p,h.sibling),h=i(h,y.children||[]),h.return=p,p=h;break e}else{n(p,h);break}else t(p,h);h=h.sibling}h=uo(y,p.mode,k),h.return=p,p=h}return o(p);case Qt:return w=y._init,N(p,h,w(y._payload),k)}if(Cr(y))return S(p,h,y,k);if(pr(y))return b(p,h,y,k);Si(p,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,h!==null&&h.tag===6?(n(p,h.sibling),h=i(h,y),h.return=p,p=h):(n(p,h),h=so(y,p.mode,k),h.return=p,p=h),o(p)):n(p,h)}return N}var nr=tf(!0),nf=tf(!1),ll=cn(null),ol=null,Hn=null,Ga=null;function Ja(){Ga=Hn=ol=null}function Za(e){var t=ll.current;ue(ll),e._currentValue=t}function Zo(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Yn(e,t){ol=e,Ga=Hn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(He=!0),e.firstContext=null)}function ct(e){var t=e._currentValue;if(Ga!==e)if(e={context:e,memoizedValue:t,next:null},Hn===null){if(ol===null)throw Error(L(308));Hn=e,ol.dependencies={lanes:0,firstContext:e}}else Hn=Hn.next=e;return t}var yn=null;function es(e){yn===null?yn=[e]:yn.push(e)}function rf(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,es(t)):(n.next=i.next,i.next=n),t.interleaved=n,Ot(e,r)}function Ot(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var qt=!1;function ts(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function lf(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Mt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function nn(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,J&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Ot(e,n)}return i=r.interleaved,i===null?(t.next=t,es(r)):(t.next=i.next,i.next=t),r.interleaved=t,Ot(e,n)}function Di(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ba(e,n)}}function Eu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function al(e,t,n,r){var i=e.updateQueue;qt=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,c=s.next;s.next=null,o===null?l=c:o.next=c,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=s))}if(l!==null){var f=i.baseState;o=0,d=c=s=null,a=l;do{var g=a.lane,m=a.eventTime;if((r&g)===g){d!==null&&(d=d.next={eventTime:m,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var S=e,b=a;switch(g=t,m=n,b.tag){case 1:if(S=b.payload,typeof S=="function"){f=S.call(m,f,g);break e}f=S;break e;case 3:S.flags=S.flags&-65537|128;case 0:if(S=b.payload,g=typeof S=="function"?S.call(m,f,g):S,g==null)break e;f=pe({},f,g);break e;case 2:qt=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,g=i.effects,g===null?i.effects=[a]:g.push(a))}else m={eventTime:m,lane:g,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=m,s=f):d=d.next=m,o|=g;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;g=a,a=g.next,g.next=null,i.lastBaseUpdate=g,i.shared.pending=null}}while(!0);if(d===null&&(s=f),i.baseState=s,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);jn|=o,e.lanes=o,e.memoizedState=f}}function Nu(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(L(191,i));i.call(r)}}}var ai={},Et=cn(ai),Xr=cn(ai),Gr=cn(ai);function xn(e){if(e===ai)throw Error(L(174));return e}function ns(e,t){switch(oe(Gr,t),oe(Xr,e),oe(Et,ai),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Io(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Io(t,e)}ue(Et),oe(Et,t)}function rr(){ue(Et),ue(Xr),ue(Gr)}function of(e){xn(Gr.current);var t=xn(Et.current),n=Io(t,e.type);t!==n&&(oe(Xr,e),oe(Et,n))}function rs(e){Xr.current===e&&(ue(Et),ue(Xr))}var de=cn(0);function sl(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var no=[];function is(){for(var e=0;e<no.length;e++)no[e]._workInProgressVersionPrimary=null;no.length=0}var Ri=Bt.ReactCurrentDispatcher,ro=Bt.ReactCurrentBatchConfig,Cn=0,fe=null,ye=null,ke=null,ul=!1,Lr=!1,Jr=0,Rm=0;function Ne(){throw Error(L(321))}function ls(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!xt(e[n],t[n]))return!1;return!0}function os(e,t,n,r,i,l){if(Cn=l,fe=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Ri.current=e===null||e.memoizedState===null?Um:Hm,e=n(r,i),Lr){l=0;do{if(Lr=!1,Jr=0,25<=l)throw Error(L(301));l+=1,ke=ye=null,t.updateQueue=null,Ri.current=$m,e=n(r,i)}while(Lr)}if(Ri.current=cl,t=ye!==null&&ye.next!==null,Cn=0,ke=ye=fe=null,ul=!1,t)throw Error(L(300));return e}function as(){var e=Jr!==0;return Jr=0,e}function wt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return ke===null?fe.memoizedState=ke=e:ke=ke.next=e,ke}function dt(){if(ye===null){var e=fe.alternate;e=e!==null?e.memoizedState:null}else e=ye.next;var t=ke===null?fe.memoizedState:ke.next;if(t!==null)ke=t,ye=e;else{if(e===null)throw Error(L(310));ye=e,e={memoizedState:ye.memoizedState,baseState:ye.baseState,baseQueue:ye.baseQueue,queue:ye.queue,next:null},ke===null?fe.memoizedState=ke=e:ke=ke.next=e}return ke}function Zr(e,t){return typeof t=="function"?t(e):t}function io(e){var t=dt(),n=t.queue;if(n===null)throw Error(L(311));n.lastRenderedReducer=e;var r=ye,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,c=l;do{var d=c.lane;if((Cn&d)===d)s!==null&&(s=s.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var f={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};s===null?(a=s=f,o=r):s=s.next=f,fe.lanes|=d,jn|=d}c=c.next}while(c!==null&&c!==l);s===null?o=r:s.next=a,xt(r,t.memoizedState)||(He=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,fe.lanes|=l,jn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function lo(e){var t=dt(),n=t.queue;if(n===null)throw Error(L(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);xt(l,t.memoizedState)||(He=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function af(){}function sf(e,t){var n=fe,r=dt(),i=t(),l=!xt(r.memoizedState,i);if(l&&(r.memoizedState=i,He=!0),r=r.queue,ss(df.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||ke!==null&&ke.memoizedState.tag&1){if(n.flags|=2048,ei(9,cf.bind(null,n,r,i,t),void 0,null),we===null)throw Error(L(349));Cn&30||uf(n,t,i)}return i}function uf(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=fe.updateQueue,t===null?(t={lastEffect:null,stores:null},fe.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function cf(e,t,n,r){t.value=n,t.getSnapshot=r,ff(t)&&pf(e)}function df(e,t,n){return n(function(){ff(t)&&pf(e)})}function ff(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!xt(e,n)}catch{return!0}}function pf(e){var t=Ot(e,1);t!==null&&yt(t,e,1,-1)}function _u(e){var t=wt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:Zr,lastRenderedState:e},t.queue=e,e=e.dispatch=Bm.bind(null,fe,e),[t.memoizedState,e]}function ei(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=fe.updateQueue,t===null?(t={lastEffect:null,stores:null},fe.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function hf(){return dt().memoizedState}function Oi(e,t,n,r){var i=wt();fe.flags|=e,i.memoizedState=ei(1|t,n,void 0,r===void 0?null:r)}function jl(e,t,n,r){var i=dt();r=r===void 0?null:r;var l=void 0;if(ye!==null){var o=ye.memoizedState;if(l=o.destroy,r!==null&&ls(r,o.deps)){i.memoizedState=ei(t,n,l,r);return}}fe.flags|=e,i.memoizedState=ei(1|t,n,l,r)}function zu(e,t){return Oi(8390656,8,e,t)}function ss(e,t){return jl(2048,8,e,t)}function mf(e,t){return jl(4,2,e,t)}function gf(e,t){return jl(4,4,e,t)}function vf(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function yf(e,t,n){return n=n!=null?n.concat([e]):null,jl(4,4,vf.bind(null,t,e),n)}function us(){}function xf(e,t){var n=dt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ls(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function kf(e,t){var n=dt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ls(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function wf(e,t,n){return Cn&21?(xt(n,t)||(n=jd(),fe.lanes|=n,jn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,He=!0),e.memoizedState=n)}function Om(e,t){var n=ne;ne=n!==0&&4>n?n:4,e(!0);var r=ro.transition;ro.transition={};try{e(!1),t()}finally{ne=n,ro.transition=r}}function Sf(){return dt().memoizedState}function Fm(e,t,n){var r=ln(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},bf(e))Cf(t,n);else if(n=rf(e,t,n,r),n!==null){var i=De();yt(n,e,r,i),jf(n,t,r)}}function Bm(e,t,n){var r=ln(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(bf(e))Cf(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,xt(a,o)){var s=t.interleaved;s===null?(i.next=i,es(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=rf(e,t,i,r),n!==null&&(i=De(),yt(n,e,r,i),jf(n,t,r))}}function bf(e){var t=e.alternate;return e===fe||t!==null&&t===fe}function Cf(e,t){Lr=ul=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function jf(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ba(e,n)}}var cl={readContext:ct,useCallback:Ne,useContext:Ne,useEffect:Ne,useImperativeHandle:Ne,useInsertionEffect:Ne,useLayoutEffect:Ne,useMemo:Ne,useReducer:Ne,useRef:Ne,useState:Ne,useDebugValue:Ne,useDeferredValue:Ne,useTransition:Ne,useMutableSource:Ne,useSyncExternalStore:Ne,useId:Ne,unstable_isNewReconciler:!1},Um={readContext:ct,useCallback:function(e,t){return wt().memoizedState=[e,t===void 0?null:t],e},useContext:ct,useEffect:zu,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Oi(4194308,4,vf.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Oi(4194308,4,e,t)},useInsertionEffect:function(e,t){return Oi(4,2,e,t)},useMemo:function(e,t){var n=wt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=wt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Fm.bind(null,fe,e),[r.memoizedState,e]},useRef:function(e){var t=wt();return e={current:e},t.memoizedState=e},useState:_u,useDebugValue:us,useDeferredValue:function(e){return wt().memoizedState=e},useTransition:function(){var e=_u(!1),t=e[0];return e=Om.bind(null,e[1]),wt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=fe,i=wt();if(ce){if(n===void 0)throw Error(L(407));n=n()}else{if(n=t(),we===null)throw Error(L(349));Cn&30||uf(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,zu(df.bind(null,r,l,e),[e]),r.flags|=2048,ei(9,cf.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=wt(),t=we.identifierPrefix;if(ce){var n=At,r=It;n=(r&~(1<<32-vt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=Jr++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Rm++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},Hm={readContext:ct,useCallback:xf,useContext:ct,useEffect:ss,useImperativeHandle:yf,useInsertionEffect:mf,useLayoutEffect:gf,useMemo:kf,useReducer:io,useRef:hf,useState:function(){return io(Zr)},useDebugValue:us,useDeferredValue:function(e){var t=dt();return wf(t,ye.memoizedState,e)},useTransition:function(){var e=io(Zr)[0],t=dt().memoizedState;return[e,t]},useMutableSource:af,useSyncExternalStore:sf,useId:Sf,unstable_isNewReconciler:!1},$m={readContext:ct,useCallback:xf,useContext:ct,useEffect:ss,useImperativeHandle:yf,useInsertionEffect:mf,useLayoutEffect:gf,useMemo:kf,useReducer:lo,useRef:hf,useState:function(){return lo(Zr)},useDebugValue:us,useDeferredValue:function(e){var t=dt();return ye===null?t.memoizedState=e:wf(t,ye.memoizedState,e)},useTransition:function(){var e=lo(Zr)[0],t=dt().memoizedState;return[e,t]},useMutableSource:af,useSyncExternalStore:sf,useId:Sf,unstable_isNewReconciler:!1};function ht(e,t){if(e&&e.defaultProps){t=pe({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function ea(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:pe({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var El={isMounted:function(e){return(e=e._reactInternals)?_n(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=De(),i=ln(e),l=Mt(r,i);l.payload=t,n!=null&&(l.callback=n),t=nn(e,l,i),t!==null&&(yt(t,e,i,r),Di(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=De(),i=ln(e),l=Mt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=nn(e,l,i),t!==null&&(yt(t,e,i,r),Di(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=De(),r=ln(e),i=Mt(n,r);i.tag=2,t!=null&&(i.callback=t),t=nn(e,i,r),t!==null&&(yt(t,e,r,n),Di(t,e,r))}};function Pu(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Qr(n,r)||!Qr(i,l):!0}function Ef(e,t,n){var r=!1,i=sn,l=t.contextType;return typeof l=="object"&&l!==null?l=ct(l):(i=Ve(t)?Sn:Te.current,r=t.contextTypes,l=(r=r!=null)?er(e,i):sn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=El,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function Tu(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&El.enqueueReplaceState(t,t.state,null)}function ta(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},ts(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=ct(l):(l=Ve(t)?Sn:Te.current,i.context=er(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(ea(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&El.enqueueReplaceState(i,i.state,null),al(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function ir(e,t){try{var n="",r=t;do n+=vh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function oo(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function na(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var Vm=typeof WeakMap=="function"?WeakMap:Map;function Nf(e,t,n){n=Mt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){fl||(fl=!0,fa=r),na(e,t)},n}function _f(e,t,n){n=Mt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){na(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){na(e,t),typeof r!="function"&&(rn===null?rn=new Set([this]):rn.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function Lu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new Vm;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=ig.bind(null,e,t,n),t.then(e,e))}function Iu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Au(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Mt(-1,1),t.tag=2,nn(n,t,1))),n.lanes|=1),e)}var Wm=Bt.ReactCurrentOwner,He=!1;function Me(e,t,n,r){t.child=e===null?nf(t,null,n,r):nr(t,e.child,n,r)}function Mu(e,t,n,r,i){n=n.render;var l=t.ref;return Yn(t,i),r=os(e,t,n,r,l,i),n=as(),e!==null&&!He?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Ft(e,t,i)):(ce&&n&&Ka(t),t.flags|=1,Me(e,t,r,i),t.child)}function Du(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!vs(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,zf(e,t,l,r,i)):(e=Hi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Qr,n(o,r)&&e.ref===t.ref)return Ft(e,t,i)}return t.flags|=1,e=on(l,r),e.ref=t.ref,e.return=t,t.child=e}function zf(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Qr(l,r)&&e.ref===t.ref)if(He=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(He=!0);else return t.lanes=e.lanes,Ft(e,t,i)}return ra(e,t,n,r,i)}function Pf(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},oe(Vn,Ge),Ge|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,oe(Vn,Ge),Ge|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,oe(Vn,Ge),Ge|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,oe(Vn,Ge),Ge|=r;return Me(e,t,i,n),t.child}function Tf(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ra(e,t,n,r,i){var l=Ve(n)?Sn:Te.current;return l=er(t,l),Yn(t,i),n=os(e,t,n,r,l,i),r=as(),e!==null&&!He?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Ft(e,t,i)):(ce&&r&&Ka(t),t.flags|=1,Me(e,t,n,i),t.child)}function Ru(e,t,n,r,i){if(Ve(n)){var l=!0;nl(t)}else l=!1;if(Yn(t,i),t.stateNode===null)Fi(e,t),Ef(t,n,r),ta(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=ct(c):(c=Ve(n)?Sn:Te.current,c=er(t,c));var d=n.getDerivedStateFromProps,f=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";f||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==c)&&Tu(t,o,r,c),qt=!1;var g=t.memoizedState;o.state=g,al(t,r,o,i),s=t.memoizedState,a!==r||g!==s||$e.current||qt?(typeof d=="function"&&(ea(t,n,d,r),s=t.memoizedState),(a=qt||Pu(t,n,a,r,g,s,c))?(f||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,lf(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:ht(t.type,a),o.props=c,f=t.pendingProps,g=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=ct(s):(s=Ve(n)?Sn:Te.current,s=er(t,s));var m=n.getDerivedStateFromProps;(d=typeof m=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==f||g!==s)&&Tu(t,o,r,s),qt=!1,g=t.memoizedState,o.state=g,al(t,r,o,i);var S=t.memoizedState;a!==f||g!==S||$e.current||qt?(typeof m=="function"&&(ea(t,n,m,r),S=t.memoizedState),(c=qt||Pu(t,n,c,r,g,S,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,S,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,S,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=S),o.props=r,o.state=S,o.context=s,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=1024),r=!1)}return ia(e,t,n,r,l,i)}function ia(e,t,n,r,i,l){Tf(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&Su(t,n,!1),Ft(e,t,l);r=t.stateNode,Wm.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=nr(t,e.child,null,l),t.child=nr(t,null,a,l)):Me(e,t,a,l),t.memoizedState=r.state,i&&Su(t,n,!0),t.child}function Lf(e){var t=e.stateNode;t.pendingContext?wu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&wu(e,t.context,!1),ns(e,t.containerInfo)}function Ou(e,t,n,r,i){return tr(),Xa(i),t.flags|=256,Me(e,t,n,r),t.child}var la={dehydrated:null,treeContext:null,retryLane:0};function oa(e){return{baseLanes:e,cachePool:null,transitions:null}}function If(e,t,n){var r=t.pendingProps,i=de.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),oe(de,i&1),e===null)return Jo(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=zl(o,r,0,null),e=wn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=oa(n),t.memoizedState=la,e):cs(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return Qm(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=on(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=on(a,l):(l=wn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?oa(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=la,r}return l=e.child,e=l.sibling,r=on(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function cs(e,t){return t=zl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function bi(e,t,n,r){return r!==null&&Xa(r),nr(t,e.child,null,n),e=cs(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function Qm(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=oo(Error(L(422))),bi(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=zl({mode:"visible",children:r.children},i,0,null),l=wn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&nr(t,e.child,null,o),t.child.memoizedState=oa(o),t.memoizedState=la,l);if(!(t.mode&1))return bi(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(L(419)),r=oo(l,r,void 0),bi(e,t,o,r)}if(a=(o&e.childLanes)!==0,He||a){if(r=we,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Ot(e,i),yt(r,e,i,-1))}return gs(),r=oo(Error(L(421))),bi(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=lg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,Je=tn(i.nextSibling),et=t,ce=!0,gt=null,e!==null&&(lt[ot++]=It,lt[ot++]=At,lt[ot++]=bn,It=e.id,At=e.overflow,bn=t),t=cs(t,r.children),t.flags|=4096,t)}function Fu(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),Zo(e.return,t,n)}function ao(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Af(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(Me(e,t,r.children,n),r=de.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Fu(e,n,t);else if(e.tag===19)Fu(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(oe(de,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&sl(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),ao(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&sl(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}ao(t,!0,n,null,l);break;case"together":ao(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Fi(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Ft(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),jn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(L(153));if(t.child!==null){for(e=t.child,n=on(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=on(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function qm(e,t,n){switch(t.tag){case 3:Lf(t),tr();break;case 5:of(t);break;case 1:Ve(t.type)&&nl(t);break;case 4:ns(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;oe(ll,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(oe(de,de.current&1),t.flags|=128,null):n&t.child.childLanes?If(e,t,n):(oe(de,de.current&1),e=Ft(e,t,n),e!==null?e.sibling:null);oe(de,de.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Af(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),oe(de,de.current),r)break;return null;case 22:case 23:return t.lanes=0,Pf(e,t,n)}return Ft(e,t,n)}var Mf,aa,Df,Rf;Mf=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};aa=function(){};Df=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,xn(Et.current);var l=null;switch(n){case"input":i=zo(e,i),r=zo(e,r),l=[];break;case"select":i=pe({},i,{value:void 0}),r=pe({},r,{value:void 0}),l=[];break;case"textarea":i=Lo(e,i),r=Lo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=el)}Ao(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(Fr.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var s=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&s!==a&&(s!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(c,n)),n=s;else c==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(c,s)):c==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(c,""+s):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(Fr.hasOwnProperty(c)?(s!=null&&c==="onScroll"&&se("scroll",e),l||a===s||(l=[])):(l=l||[]).push(c,s))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};Rf=function(e,t,n,r){n!==r&&(t.flags|=4)};function xr(e,t){if(!ce)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function _e(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function Km(e,t,n){var r=t.pendingProps;switch(Ya(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return _e(t),null;case 1:return Ve(t.type)&&tl(),_e(t),null;case 3:return r=t.stateNode,rr(),ue($e),ue(Te),is(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(wi(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,gt!==null&&(ma(gt),gt=null))),aa(e,t),_e(t),null;case 5:rs(t);var i=xn(Gr.current);if(n=t.type,e!==null&&t.stateNode!=null)Df(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(L(166));return _e(t),null}if(e=xn(Et.current),wi(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[bt]=t,r[Yr]=l,e=(t.mode&1)!==0,n){case"dialog":se("cancel",r),se("close",r);break;case"iframe":case"object":case"embed":se("load",r);break;case"video":case"audio":for(i=0;i<Er.length;i++)se(Er[i],r);break;case"source":se("error",r);break;case"img":case"image":case"link":se("error",r),se("load",r);break;case"details":se("toggle",r);break;case"input":Ks(r,l),se("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},se("invalid",r);break;case"textarea":Xs(r,l),se("invalid",r)}Ao(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&ki(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&ki(r.textContent,a,e),i=["children",""+a]):Fr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&se("scroll",r)}switch(n){case"input":fi(r),Ys(r,l,!0);break;case"textarea":fi(r),Gs(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=el)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=cd(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[bt]=t,e[Yr]=r,Mf(e,t,!1,!1),t.stateNode=e;e:{switch(o=Mo(n,r),n){case"dialog":se("cancel",e),se("close",e),i=r;break;case"iframe":case"object":case"embed":se("load",e),i=r;break;case"video":case"audio":for(i=0;i<Er.length;i++)se(Er[i],e);i=r;break;case"source":se("error",e),i=r;break;case"img":case"image":case"link":se("error",e),se("load",e),i=r;break;case"details":se("toggle",e),i=r;break;case"input":Ks(e,r),i=zo(e,r),se("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=pe({},r,{value:void 0}),se("invalid",e);break;case"textarea":Xs(e,r),i=Lo(e,r),se("invalid",e);break;default:i=r}Ao(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?pd(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&dd(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Br(e,s):typeof s=="number"&&Br(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Fr.hasOwnProperty(l)?s!=null&&l==="onScroll"&&se("scroll",e):s!=null&&Aa(e,l,s,o))}switch(n){case"input":fi(e),Ys(e,r,!1);break;case"textarea":fi(e),Gs(e);break;case"option":r.value!=null&&e.setAttribute("value",""+an(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Wn(e,!!r.multiple,l,!1):r.defaultValue!=null&&Wn(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=el)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return _e(t),null;case 6:if(e&&t.stateNode!=null)Rf(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(L(166));if(n=xn(Gr.current),xn(Et.current),wi(t)){if(r=t.stateNode,n=t.memoizedProps,r[bt]=t,(l=r.nodeValue!==n)&&(e=et,e!==null))switch(e.tag){case 3:ki(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&ki(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[bt]=t,t.stateNode=r}return _e(t),null;case 13:if(ue(de),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(ce&&Je!==null&&t.mode&1&&!(t.flags&128))ef(),tr(),t.flags|=98560,l=!1;else if(l=wi(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(L(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(L(317));l[bt]=t}else tr(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;_e(t),l=!1}else gt!==null&&(ma(gt),gt=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||de.current&1?xe===0&&(xe=3):gs())),t.updateQueue!==null&&(t.flags|=4),_e(t),null);case 4:return rr(),aa(e,t),e===null&&qr(t.stateNode.containerInfo),_e(t),null;case 10:return Za(t.type._context),_e(t),null;case 17:return Ve(t.type)&&tl(),_e(t),null;case 19:if(ue(de),l=t.memoizedState,l===null)return _e(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)xr(l,!1);else{if(xe!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=sl(e),o!==null){for(t.flags|=128,xr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return oe(de,de.current&1|2),t.child}e=e.sibling}l.tail!==null&&me()>lr&&(t.flags|=128,r=!0,xr(l,!1),t.lanes=4194304)}else{if(!r)if(e=sl(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),xr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!ce)return _e(t),null}else 2*me()-l.renderingStartTime>lr&&n!==1073741824&&(t.flags|=128,r=!0,xr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=me(),t.sibling=null,n=de.current,oe(de,r?n&1|2:n&1),t):(_e(t),null);case 22:case 23:return ms(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?Ge&1073741824&&(_e(t),t.subtreeFlags&6&&(t.flags|=8192)):_e(t),null;case 24:return null;case 25:return null}throw Error(L(156,t.tag))}function Ym(e,t){switch(Ya(t),t.tag){case 1:return Ve(t.type)&&tl(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return rr(),ue($e),ue(Te),is(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return rs(t),null;case 13:if(ue(de),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(L(340));tr()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return ue(de),null;case 4:return rr(),null;case 10:return Za(t.type._context),null;case 22:case 23:return ms(),null;case 24:return null;default:return null}}var Ci=!1,Pe=!1,Xm=typeof WeakSet=="function"?WeakSet:Set,O=null;function $n(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){he(e,t,r)}else n.current=null}function sa(e,t,n){try{n()}catch(r){he(e,t,r)}}var Bu=!1;function Gm(e,t){if(Wo=Gi,e=Ud(),qa(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,c=0,d=0,f=e,g=null;t:for(;;){for(var m;f!==n||i!==0&&f.nodeType!==3||(a=o+i),f!==l||r!==0&&f.nodeType!==3||(s=o+r),f.nodeType===3&&(o+=f.nodeValue.length),(m=f.firstChild)!==null;)g=f,f=m;for(;;){if(f===e)break t;if(g===n&&++c===i&&(a=o),g===l&&++d===r&&(s=o),(m=f.nextSibling)!==null)break;f=g,g=f.parentNode}f=m}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Qo={focusedElem:e,selectionRange:n},Gi=!1,O=t;O!==null;)if(t=O,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,O=e;else for(;O!==null;){t=O;try{var S=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(S!==null){var b=S.memoizedProps,N=S.memoizedState,p=t.stateNode,h=p.getSnapshotBeforeUpdate(t.elementType===t.type?b:ht(t.type,b),N);p.__reactInternalSnapshotBeforeUpdate=h}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(L(163))}}catch(k){he(t,t.return,k)}if(e=t.sibling,e!==null){e.return=t.return,O=e;break}O=t.return}return S=Bu,Bu=!1,S}function Ir(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&sa(t,n,l)}i=i.next}while(i!==r)}}function Nl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function ua(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Of(e){var t=e.alternate;t!==null&&(e.alternate=null,Of(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[bt],delete t[Yr],delete t[Yo],delete t[Im],delete t[Am])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Ff(e){return e.tag===5||e.tag===3||e.tag===4}function Uu(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Ff(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function ca(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=el));else if(r!==4&&(e=e.child,e!==null))for(ca(e,t,n),e=e.sibling;e!==null;)ca(e,t,n),e=e.sibling}function da(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(da(e,t,n),e=e.sibling;e!==null;)da(e,t,n),e=e.sibling}var Ce=null,mt=!1;function $t(e,t,n){for(n=n.child;n!==null;)Bf(e,t,n),n=n.sibling}function Bf(e,t,n){if(jt&&typeof jt.onCommitFiberUnmount=="function")try{jt.onCommitFiberUnmount(xl,n)}catch{}switch(n.tag){case 5:Pe||$n(n,t);case 6:var r=Ce,i=mt;Ce=null,$t(e,t,n),Ce=r,mt=i,Ce!==null&&(mt?(e=Ce,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):Ce.removeChild(n.stateNode));break;case 18:Ce!==null&&(mt?(e=Ce,n=n.stateNode,e.nodeType===8?eo(e.parentNode,n):e.nodeType===1&&eo(e,n),Vr(e)):eo(Ce,n.stateNode));break;case 4:r=Ce,i=mt,Ce=n.stateNode.containerInfo,mt=!0,$t(e,t,n),Ce=r,mt=i;break;case 0:case 11:case 14:case 15:if(!Pe&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&sa(n,t,o),i=i.next}while(i!==r)}$t(e,t,n);break;case 1:if(!Pe&&($n(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){he(n,t,a)}$t(e,t,n);break;case 21:$t(e,t,n);break;case 22:n.mode&1?(Pe=(r=Pe)||n.memoizedState!==null,$t(e,t,n),Pe=r):$t(e,t,n);break;default:$t(e,t,n)}}function Hu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new Xm),t.forEach(function(r){var i=og.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function pt(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:Ce=a.stateNode,mt=!1;break e;case 3:Ce=a.stateNode.containerInfo,mt=!0;break e;case 4:Ce=a.stateNode.containerInfo,mt=!0;break e}a=a.return}if(Ce===null)throw Error(L(160));Bf(l,o,i),Ce=null,mt=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(c){he(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Uf(t,e),t=t.sibling}function Uf(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(pt(t,e),kt(e),r&4){try{Ir(3,e,e.return),Nl(3,e)}catch(b){he(e,e.return,b)}try{Ir(5,e,e.return)}catch(b){he(e,e.return,b)}}break;case 1:pt(t,e),kt(e),r&512&&n!==null&&$n(n,n.return);break;case 5:if(pt(t,e),kt(e),r&512&&n!==null&&$n(n,n.return),e.flags&32){var i=e.stateNode;try{Br(i,"")}catch(b){he(e,e.return,b)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&sd(i,l),Mo(a,o);var c=Mo(a,l);for(o=0;o<s.length;o+=2){var d=s[o],f=s[o+1];d==="style"?pd(i,f):d==="dangerouslySetInnerHTML"?dd(i,f):d==="children"?Br(i,f):Aa(i,d,f,c)}switch(a){case"input":Po(i,l);break;case"textarea":ud(i,l);break;case"select":var g=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var m=l.value;m!=null?Wn(i,!!l.multiple,m,!1):g!==!!l.multiple&&(l.defaultValue!=null?Wn(i,!!l.multiple,l.defaultValue,!0):Wn(i,!!l.multiple,l.multiple?[]:"",!1))}i[Yr]=l}catch(b){he(e,e.return,b)}}break;case 6:if(pt(t,e),kt(e),r&4){if(e.stateNode===null)throw Error(L(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(b){he(e,e.return,b)}}break;case 3:if(pt(t,e),kt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Vr(t.containerInfo)}catch(b){he(e,e.return,b)}break;case 4:pt(t,e),kt(e);break;case 13:pt(t,e),kt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(ps=me())),r&4&&Hu(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Pe=(c=Pe)||d,pt(t,e),Pe=c):pt(t,e),kt(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for(O=e,d=e.child;d!==null;){for(f=O=d;O!==null;){switch(g=O,m=g.child,g.tag){case 0:case 11:case 14:case 15:Ir(4,g,g.return);break;case 1:$n(g,g.return);var S=g.stateNode;if(typeof S.componentWillUnmount=="function"){r=g,n=g.return;try{t=r,S.props=t.memoizedProps,S.state=t.memoizedState,S.componentWillUnmount()}catch(b){he(r,n,b)}}break;case 5:$n(g,g.return);break;case 22:if(g.memoizedState!==null){Vu(f);continue}}m!==null?(m.return=g,O=m):Vu(f)}d=d.sibling}e:for(d=null,f=e;;){if(f.tag===5){if(d===null){d=f;try{i=f.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=f.stateNode,s=f.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=fd("display",o))}catch(b){he(e,e.return,b)}}}else if(f.tag===6){if(d===null)try{f.stateNode.nodeValue=c?"":f.memoizedProps}catch(b){he(e,e.return,b)}}else if((f.tag!==22&&f.tag!==23||f.memoizedState===null||f===e)&&f.child!==null){f.child.return=f,f=f.child;continue}if(f===e)break e;for(;f.sibling===null;){if(f.return===null||f.return===e)break e;d===f&&(d=null),f=f.return}d===f&&(d=null),f.sibling.return=f.return,f=f.sibling}}break;case 19:pt(t,e),kt(e),r&4&&Hu(e);break;case 21:break;default:pt(t,e),kt(e)}}function kt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Ff(n)){var r=n;break e}n=n.return}throw Error(L(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Br(i,""),r.flags&=-33);var l=Uu(e);da(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Uu(e);ca(e,a,o);break;default:throw Error(L(161))}}catch(s){he(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function Jm(e,t,n){O=e,Hf(e)}function Hf(e,t,n){for(var r=(e.mode&1)!==0;O!==null;){var i=O,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||Ci;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||Pe;a=Ci;var c=Pe;if(Ci=o,(Pe=s)&&!c)for(O=i;O!==null;)o=O,s=o.child,o.tag===22&&o.memoizedState!==null?Wu(i):s!==null?(s.return=o,O=s):Wu(i);for(;l!==null;)O=l,Hf(l),l=l.sibling;O=i,Ci=a,Pe=c}$u(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,O=l):$u(e)}}function $u(e){for(;O!==null;){var t=O;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Pe||Nl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Pe)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:ht(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Nu(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Nu(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var f=d.dehydrated;f!==null&&Vr(f)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(L(163))}Pe||t.flags&512&&ua(t)}catch(g){he(t,t.return,g)}}if(t===e){O=null;break}if(n=t.sibling,n!==null){n.return=t.return,O=n;break}O=t.return}}function Vu(e){for(;O!==null;){var t=O;if(t===e){O=null;break}var n=t.sibling;if(n!==null){n.return=t.return,O=n;break}O=t.return}}function Wu(e){for(;O!==null;){var t=O;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Nl(4,t)}catch(s){he(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){he(t,i,s)}}var l=t.return;try{ua(t)}catch(s){he(t,l,s)}break;case 5:var o=t.return;try{ua(t)}catch(s){he(t,o,s)}}}catch(s){he(t,t.return,s)}if(t===e){O=null;break}var a=t.sibling;if(a!==null){a.return=t.return,O=a;break}O=t.return}}var Zm=Math.ceil,dl=Bt.ReactCurrentDispatcher,ds=Bt.ReactCurrentOwner,ut=Bt.ReactCurrentBatchConfig,J=0,we=null,ve=null,je=0,Ge=0,Vn=cn(0),xe=0,ti=null,jn=0,_l=0,fs=0,Ar=null,Ue=null,ps=0,lr=1/0,Tt=null,fl=!1,fa=null,rn=null,ji=!1,Gt=null,pl=0,Mr=0,pa=null,Bi=-1,Ui=0;function De(){return J&6?me():Bi!==-1?Bi:Bi=me()}function ln(e){return e.mode&1?J&2&&je!==0?je&-je:Dm.transition!==null?(Ui===0&&(Ui=jd()),Ui):(e=ne,e!==0||(e=window.event,e=e===void 0?16:Ld(e.type)),e):1}function yt(e,t,n,r){if(50<Mr)throw Mr=0,pa=null,Error(L(185));ii(e,n,r),(!(J&2)||e!==we)&&(e===we&&(!(J&2)&&(_l|=n),xe===4&&Yt(e,je)),We(e,r),n===1&&J===0&&!(t.mode&1)&&(lr=me()+500,Cl&&dn()))}function We(e,t){var n=e.callbackNode;Dh(e,t);var r=Xi(e,e===we?je:0);if(r===0)n!==null&&eu(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&eu(n),t===1)e.tag===0?Mm(Qu.bind(null,e)):Gd(Qu.bind(null,e)),Tm(function(){!(J&6)&&dn()}),n=null;else{switch(Ed(r)){case 1:n=Fa;break;case 4:n=bd;break;case 16:n=Yi;break;case 536870912:n=Cd;break;default:n=Yi}n=Xf(n,$f.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function $f(e,t){if(Bi=-1,Ui=0,J&6)throw Error(L(327));var n=e.callbackNode;if(Xn()&&e.callbackNode!==n)return null;var r=Xi(e,e===we?je:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=hl(e,r);else{t=r;var i=J;J|=2;var l=Wf();(we!==e||je!==t)&&(Tt=null,lr=me()+500,kn(e,t));do try{ng();break}catch(a){Vf(e,a)}while(!0);Ja(),dl.current=l,J=i,ve!==null?t=0:(we=null,je=0,t=xe)}if(t!==0){if(t===2&&(i=Bo(e),i!==0&&(r=i,t=ha(e,i))),t===1)throw n=ti,kn(e,0),Yt(e,r),We(e,me()),n;if(t===6)Yt(e,r);else{if(i=e.current.alternate,!(r&30)&&!eg(i)&&(t=hl(e,r),t===2&&(l=Bo(e),l!==0&&(r=l,t=ha(e,l))),t===1))throw n=ti,kn(e,0),Yt(e,r),We(e,me()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(L(345));case 2:mn(e,Ue,Tt);break;case 3:if(Yt(e,r),(r&130023424)===r&&(t=ps+500-me(),10<t)){if(Xi(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){De(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=Ko(mn.bind(null,e,Ue,Tt),t);break}mn(e,Ue,Tt);break;case 4:if(Yt(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-vt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=me()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*Zm(r/1960))-r,10<r){e.timeoutHandle=Ko(mn.bind(null,e,Ue,Tt),r);break}mn(e,Ue,Tt);break;case 5:mn(e,Ue,Tt);break;default:throw Error(L(329))}}}return We(e,me()),e.callbackNode===n?$f.bind(null,e):null}function ha(e,t){var n=Ar;return e.current.memoizedState.isDehydrated&&(kn(e,t).flags|=256),e=hl(e,t),e!==2&&(t=Ue,Ue=n,t!==null&&ma(t)),e}function ma(e){Ue===null?Ue=e:Ue.push.apply(Ue,e)}function eg(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!xt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Yt(e,t){for(t&=~fs,t&=~_l,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-vt(t),r=1<<n;e[n]=-1,t&=~r}}function Qu(e){if(J&6)throw Error(L(327));Xn();var t=Xi(e,0);if(!(t&1))return We(e,me()),null;var n=hl(e,t);if(e.tag!==0&&n===2){var r=Bo(e);r!==0&&(t=r,n=ha(e,r))}if(n===1)throw n=ti,kn(e,0),Yt(e,t),We(e,me()),n;if(n===6)throw Error(L(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,mn(e,Ue,Tt),We(e,me()),null}function hs(e,t){var n=J;J|=1;try{return e(t)}finally{J=n,J===0&&(lr=me()+500,Cl&&dn())}}function En(e){Gt!==null&&Gt.tag===0&&!(J&6)&&Xn();var t=J;J|=1;var n=ut.transition,r=ne;try{if(ut.transition=null,ne=1,e)return e()}finally{ne=r,ut.transition=n,J=t,!(J&6)&&dn()}}function ms(){Ge=Vn.current,ue(Vn)}function kn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,Pm(n)),ve!==null)for(n=ve.return;n!==null;){var r=n;switch(Ya(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&tl();break;case 3:rr(),ue($e),ue(Te),is();break;case 5:rs(r);break;case 4:rr();break;case 13:ue(de);break;case 19:ue(de);break;case 10:Za(r.type._context);break;case 22:case 23:ms()}n=n.return}if(we=e,ve=e=on(e.current,null),je=Ge=t,xe=0,ti=null,fs=_l=jn=0,Ue=Ar=null,yn!==null){for(t=0;t<yn.length;t++)if(n=yn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}yn=null}return e}function Vf(e,t){do{var n=ve;try{if(Ja(),Ri.current=cl,ul){for(var r=fe.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}ul=!1}if(Cn=0,ke=ye=fe=null,Lr=!1,Jr=0,ds.current=null,n===null||n.return===null){xe=1,ti=t,ve=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=je,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var c=s,d=a,f=d.tag;if(!(d.mode&1)&&(f===0||f===11||f===15)){var g=d.alternate;g?(d.updateQueue=g.updateQueue,d.memoizedState=g.memoizedState,d.lanes=g.lanes):(d.updateQueue=null,d.memoizedState=null)}var m=Iu(o);if(m!==null){m.flags&=-257,Au(m,o,a,l,t),m.mode&1&&Lu(l,c,t),t=m,s=c;var S=t.updateQueue;if(S===null){var b=new Set;b.add(s),t.updateQueue=b}else S.add(s);break e}else{if(!(t&1)){Lu(l,c,t),gs();break e}s=Error(L(426))}}else if(ce&&a.mode&1){var N=Iu(o);if(N!==null){!(N.flags&65536)&&(N.flags|=256),Au(N,o,a,l,t),Xa(ir(s,a));break e}}l=s=ir(s,a),xe!==4&&(xe=2),Ar===null?Ar=[l]:Ar.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var p=Nf(l,s,t);Eu(l,p);break e;case 1:a=s;var h=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof h.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(rn===null||!rn.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var k=_f(l,a,t);Eu(l,k);break e}}l=l.return}while(l!==null)}qf(n)}catch(j){t=j,ve===n&&n!==null&&(ve=n=n.return);continue}break}while(!0)}function Wf(){var e=dl.current;return dl.current=cl,e===null?cl:e}function gs(){(xe===0||xe===3||xe===2)&&(xe=4),we===null||!(jn&268435455)&&!(_l&268435455)||Yt(we,je)}function hl(e,t){var n=J;J|=2;var r=Wf();(we!==e||je!==t)&&(Tt=null,kn(e,t));do try{tg();break}catch(i){Vf(e,i)}while(!0);if(Ja(),J=n,dl.current=r,ve!==null)throw Error(L(261));return we=null,je=0,xe}function tg(){for(;ve!==null;)Qf(ve)}function ng(){for(;ve!==null&&!Nh();)Qf(ve)}function Qf(e){var t=Yf(e.alternate,e,Ge);e.memoizedProps=e.pendingProps,t===null?qf(e):ve=t,ds.current=null}function qf(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=Ym(n,t),n!==null){n.flags&=32767,ve=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{xe=6,ve=null;return}}else if(n=Km(n,t,Ge),n!==null){ve=n;return}if(t=t.sibling,t!==null){ve=t;return}ve=t=e}while(t!==null);xe===0&&(xe=5)}function mn(e,t,n){var r=ne,i=ut.transition;try{ut.transition=null,ne=1,rg(e,t,n,r)}finally{ut.transition=i,ne=r}return null}function rg(e,t,n,r){do Xn();while(Gt!==null);if(J&6)throw Error(L(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(L(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Rh(e,l),e===we&&(ve=we=null,je=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||ji||(ji=!0,Xf(Yi,function(){return Xn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=ut.transition,ut.transition=null;var o=ne;ne=1;var a=J;J|=4,ds.current=null,Gm(e,n),Uf(n,e),bm(Qo),Gi=!!Wo,Qo=Wo=null,e.current=n,Jm(n),_h(),J=a,ne=o,ut.transition=l}else e.current=n;if(ji&&(ji=!1,Gt=e,pl=i),l=e.pendingLanes,l===0&&(rn=null),Th(n.stateNode),We(e,me()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(fl)throw fl=!1,e=fa,fa=null,e;return pl&1&&e.tag!==0&&Xn(),l=e.pendingLanes,l&1?e===pa?Mr++:(Mr=0,pa=e):Mr=0,dn(),null}function Xn(){if(Gt!==null){var e=Ed(pl),t=ut.transition,n=ne;try{if(ut.transition=null,ne=16>e?16:e,Gt===null)var r=!1;else{if(e=Gt,Gt=null,pl=0,J&6)throw Error(L(331));var i=J;for(J|=4,O=e.current;O!==null;){var l=O,o=l.child;if(O.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var c=a[s];for(O=c;O!==null;){var d=O;switch(d.tag){case 0:case 11:case 15:Ir(8,d,l)}var f=d.child;if(f!==null)f.return=d,O=f;else for(;O!==null;){d=O;var g=d.sibling,m=d.return;if(Of(d),d===c){O=null;break}if(g!==null){g.return=m,O=g;break}O=m}}}var S=l.alternate;if(S!==null){var b=S.child;if(b!==null){S.child=null;do{var N=b.sibling;b.sibling=null,b=N}while(b!==null)}}O=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,O=o;else e:for(;O!==null;){if(l=O,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Ir(9,l,l.return)}var p=l.sibling;if(p!==null){p.return=l.return,O=p;break e}O=l.return}}var h=e.current;for(O=h;O!==null;){o=O;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,O=y;else e:for(o=h;O!==null;){if(a=O,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Nl(9,a)}}catch(j){he(a,a.return,j)}if(a===o){O=null;break e}var k=a.sibling;if(k!==null){k.return=a.return,O=k;break e}O=a.return}}if(J=i,dn(),jt&&typeof jt.onPostCommitFiberRoot=="function")try{jt.onPostCommitFiberRoot(xl,e)}catch{}r=!0}return r}finally{ne=n,ut.transition=t}}return!1}function qu(e,t,n){t=ir(n,t),t=Nf(e,t,1),e=nn(e,t,1),t=De(),e!==null&&(ii(e,1,t),We(e,t))}function he(e,t,n){if(e.tag===3)qu(e,e,n);else for(;t!==null;){if(t.tag===3){qu(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(rn===null||!rn.has(r))){e=ir(n,e),e=_f(t,e,1),t=nn(t,e,1),e=De(),t!==null&&(ii(t,1,e),We(t,e));break}}t=t.return}}function ig(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=De(),e.pingedLanes|=e.suspendedLanes&n,we===e&&(je&n)===n&&(xe===4||xe===3&&(je&130023424)===je&&500>me()-ps?kn(e,0):fs|=n),We(e,t)}function Kf(e,t){t===0&&(e.mode&1?(t=mi,mi<<=1,!(mi&130023424)&&(mi=4194304)):t=1);var n=De();e=Ot(e,t),e!==null&&(ii(e,t,n),We(e,n))}function lg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),Kf(e,n)}function og(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(L(314))}r!==null&&r.delete(t),Kf(e,n)}var Yf;Yf=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||$e.current)He=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return He=!1,qm(e,t,n);He=!!(e.flags&131072)}else He=!1,ce&&t.flags&1048576&&Jd(t,il,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Fi(e,t),e=t.pendingProps;var i=er(t,Te.current);Yn(t,n),i=os(null,t,r,e,i,n);var l=as();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Ve(r)?(l=!0,nl(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,ts(t),i.updater=El,t.stateNode=i,i._reactInternals=t,ta(t,r,e,n),t=ia(null,t,r,!0,l,n)):(t.tag=0,ce&&l&&Ka(t),Me(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Fi(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=sg(r),e=ht(r,e),i){case 0:t=ra(null,t,r,e,n);break e;case 1:t=Ru(null,t,r,e,n);break e;case 11:t=Mu(null,t,r,e,n);break e;case 14:t=Du(null,t,r,ht(r.type,e),n);break e}throw Error(L(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ht(r,i),ra(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ht(r,i),Ru(e,t,r,i,n);case 3:e:{if(Lf(t),e===null)throw Error(L(387));r=t.pendingProps,l=t.memoizedState,i=l.element,lf(e,t),al(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=ir(Error(L(423)),t),t=Ou(e,t,r,n,i);break e}else if(r!==i){i=ir(Error(L(424)),t),t=Ou(e,t,r,n,i);break e}else for(Je=tn(t.stateNode.containerInfo.firstChild),et=t,ce=!0,gt=null,n=nf(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(tr(),r===i){t=Ft(e,t,n);break e}Me(e,t,r,n)}t=t.child}return t;case 5:return of(t),e===null&&Jo(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,qo(r,i)?o=null:l!==null&&qo(r,l)&&(t.flags|=32),Tf(e,t),Me(e,t,o,n),t.child;case 6:return e===null&&Jo(t),null;case 13:return If(e,t,n);case 4:return ns(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=nr(t,null,r,n):Me(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ht(r,i),Mu(e,t,r,i,n);case 7:return Me(e,t,t.pendingProps,n),t.child;case 8:return Me(e,t,t.pendingProps.children,n),t.child;case 12:return Me(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,oe(ll,r._currentValue),r._currentValue=o,l!==null)if(xt(l.value,o)){if(l.children===i.children&&!$e.current){t=Ft(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Mt(-1,n&-n),s.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?s.next=s:(s.next=d.next,d.next=s),c.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),Zo(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(L(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),Zo(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}Me(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Yn(t,n),i=ct(i),r=r(i),t.flags|=1,Me(e,t,r,n),t.child;case 14:return r=t.type,i=ht(r,t.pendingProps),i=ht(r.type,i),Du(e,t,r,i,n);case 15:return zf(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ht(r,i),Fi(e,t),t.tag=1,Ve(r)?(e=!0,nl(t)):e=!1,Yn(t,n),Ef(t,r,i),ta(t,r,i,n),ia(null,t,r,!0,e,n);case 19:return Af(e,t,n);case 22:return Pf(e,t,n)}throw Error(L(156,t.tag))};function Xf(e,t){return Sd(e,t)}function ag(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function st(e,t,n,r){return new ag(e,t,n,r)}function vs(e){return e=e.prototype,!(!e||!e.isReactComponent)}function sg(e){if(typeof e=="function")return vs(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Da)return 11;if(e===Ra)return 14}return 2}function on(e,t){var n=e.alternate;return n===null?(n=st(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Hi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")vs(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case An:return wn(n.children,i,l,t);case Ma:o=8,i|=8;break;case jo:return e=st(12,n,t,i|2),e.elementType=jo,e.lanes=l,e;case Eo:return e=st(13,n,t,i),e.elementType=Eo,e.lanes=l,e;case No:return e=st(19,n,t,i),e.elementType=No,e.lanes=l,e;case ld:return zl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case rd:o=10;break e;case id:o=9;break e;case Da:o=11;break e;case Ra:o=14;break e;case Qt:o=16,r=null;break e}throw Error(L(130,e==null?e:typeof e,""))}return t=st(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function wn(e,t,n,r){return e=st(7,e,r,t),e.lanes=n,e}function zl(e,t,n,r){return e=st(22,e,r,t),e.elementType=ld,e.lanes=n,e.stateNode={isHidden:!1},e}function so(e,t,n){return e=st(6,e,null,t),e.lanes=n,e}function uo(e,t,n){return t=st(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function ug(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=$l(0),this.expirationTimes=$l(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=$l(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function ys(e,t,n,r,i,l,o,a,s){return e=new ug(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=st(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},ts(l),e}function cg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:In,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function Gf(e){if(!e)return sn;e=e._reactInternals;e:{if(_n(e)!==e||e.tag!==1)throw Error(L(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Ve(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(L(171))}if(e.tag===1){var n=e.type;if(Ve(n))return Xd(e,n,t)}return t}function Jf(e,t,n,r,i,l,o,a,s){return e=ys(n,r,!0,e,i,l,o,a,s),e.context=Gf(null),n=e.current,r=De(),i=ln(n),l=Mt(r,i),l.callback=t??null,nn(n,l,i),e.current.lanes=i,ii(e,i,r),We(e,r),e}function Pl(e,t,n,r){var i=t.current,l=De(),o=ln(i);return n=Gf(n),t.context===null?t.context=n:t.pendingContext=n,t=Mt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=nn(i,t,o),e!==null&&(yt(e,i,o,l),Di(e,i,o)),o}function ml(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function Ku(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function xs(e,t){Ku(e,t),(e=e.alternate)&&Ku(e,t)}function dg(){return null}var Zf=typeof reportError=="function"?reportError:function(e){console.error(e)};function ks(e){this._internalRoot=e}Tl.prototype.render=ks.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(L(409));Pl(e,t,null,null)};Tl.prototype.unmount=ks.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;En(function(){Pl(null,e,null,null)}),t[Rt]=null}};function Tl(e){this._internalRoot=e}Tl.prototype.unstable_scheduleHydration=function(e){if(e){var t=zd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Kt.length&&t!==0&&t<Kt[n].priority;n++);Kt.splice(n,0,e),n===0&&Td(e)}};function ws(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Ll(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function Yu(){}function fg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=ml(o);l.call(c)}}var o=Jf(t,r,e,0,null,!1,!1,"",Yu);return e._reactRootContainer=o,e[Rt]=o.current,qr(e.nodeType===8?e.parentNode:e),En(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=ml(s);a.call(c)}}var s=ys(e,0,!1,null,null,!1,!1,"",Yu);return e._reactRootContainer=s,e[Rt]=s.current,qr(e.nodeType===8?e.parentNode:e),En(function(){Pl(t,s,n,r)}),s}function Il(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=ml(o);a.call(s)}}Pl(t,o,e,i)}else o=fg(n,t,e,i,r);return ml(o)}Nd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=jr(t.pendingLanes);n!==0&&(Ba(t,n|1),We(t,me()),!(J&6)&&(lr=me()+500,dn()))}break;case 13:En(function(){var r=Ot(e,1);if(r!==null){var i=De();yt(r,e,1,i)}}),xs(e,1)}};Ua=function(e){if(e.tag===13){var t=Ot(e,134217728);if(t!==null){var n=De();yt(t,e,134217728,n)}xs(e,134217728)}};_d=function(e){if(e.tag===13){var t=ln(e),n=Ot(e,t);if(n!==null){var r=De();yt(n,e,t,r)}xs(e,t)}};zd=function(){return ne};Pd=function(e,t){var n=ne;try{return ne=e,t()}finally{ne=n}};Ro=function(e,t,n){switch(t){case"input":if(Po(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=bl(r);if(!i)throw Error(L(90));ad(r),Po(r,i)}}}break;case"textarea":ud(e,n);break;case"select":t=n.value,t!=null&&Wn(e,!!n.multiple,t,!1)}};gd=hs;vd=En;var pg={usingClientEntryPoint:!1,Events:[oi,On,bl,hd,md,hs]},kr={findFiberByHostInstance:vn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},hg={bundleType:kr.bundleType,version:kr.version,rendererPackageName:kr.rendererPackageName,rendererConfig:kr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Bt.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=kd(e),e===null?null:e.stateNode},findFiberByHostInstance:kr.findFiberByHostInstance||dg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var Ei=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!Ei.isDisabled&&Ei.supportsFiber)try{xl=Ei.inject(hg),jt=Ei}catch{}}nt.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=pg;nt.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!ws(t))throw Error(L(200));return cg(e,t,null,n)};nt.createRoot=function(e,t){if(!ws(e))throw Error(L(299));var n=!1,r="",i=Zf;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=ys(e,1,!1,null,null,n,!1,r,i),e[Rt]=t.current,qr(e.nodeType===8?e.parentNode:e),new ks(t)};nt.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(L(188)):(e=Object.keys(e).join(","),Error(L(268,e)));return e=kd(t),e=e===null?null:e.stateNode,e};nt.flushSync=function(e){return En(e)};nt.hydrate=function(e,t,n){if(!Ll(t))throw Error(L(200));return Il(null,e,t,!0,n)};nt.hydrateRoot=function(e,t,n){if(!ws(e))throw Error(L(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=Zf;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=Jf(t,null,e,1,n??null,i,!1,l,o),e[Rt]=t.current,qr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new Tl(t)};nt.render=function(e,t,n){if(!Ll(t))throw Error(L(200));return Il(null,e,t,!1,n)};nt.unmountComponentAtNode=function(e){if(!Ll(e))throw Error(L(40));return e._reactRootContainer?(En(function(){Il(null,null,e,!1,function(){e._reactRootContainer=null,e[Rt]=null})}),!0):!1};nt.unstable_batchedUpdates=hs;nt.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Ll(n))throw Error(L(200));if(e==null||e._reactInternals===void 0)throw Error(L(38));return Il(e,t,n,!1,r)};nt.version="18.3.1-next-f1338f8080-20240426";function ep(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(ep)}catch(e){console.error(e)}}ep(),Zc.exports=nt;var mg=Zc.exports,Xu=mg;bo.createRoot=Xu.createRoot,bo.hydrateRoot=Xu.hydrateRoot;const gg="",vg=({selection:e,onSelect:t,onRefresh:n})=>{const[r,i]=B.useState(null),[l,o]=B.useState(new Set(["all"])),[a,s]=B.useState(!0),[c,d]=B.useState(null),f=async()=>{try{const h=await fetch(`${gg}/api/hierarchy`);if(!h.ok)throw new Error("Failed to fetch hierarchy");const y=await h.json();i(y),d(null)}catch(h){d(h instanceof Error?h.message:"Unknown error")}finally{s(!1)}};B.useEffect(()=>{f();const h=setInterval(f,5e3);return()=>clearInterval(h)},[]);const g=h=>{o(y=>{const k=new Set(y);return k.has(h)?k.delete(h):k.add(h),k})},m=h=>{var y;if(h.type==="root")t({type:"overview"});else if(h.type==="agent")t({type:"agent",agentId:h.id});else if(h.type==="thread"){const k=(y=r==null?void 0:r.root.children)==null?void 0:y.find(j=>{var w;return(w=j.children)==null?void 0:w.some(P=>P.id===h.id)});t({type:"thread",agentId:k==null?void 0:k.id,threadId:h.id})}},S=h=>h.type==="root"&&e.type==="overview"||h.type==="agent"&&e.type==="agent"&&e.agentId===h.id||h.type==="thread"&&e.threadId===h.id,b=h=>!h||h.length===0?null:u.jsx("span",{className:"badges",children:h.map((y,k)=>u.jsxs("span",{className:`badge badge-${y.type}`,title:`${y.count} ${y.type}`,children:[y.type==="pending"&&"⏳",y.type==="unread"&&"📬",y.type==="running"&&"▶️",y.count]},k))}),N=h=>{if(!h)return null;const y={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsx("span",{className:"status-indicator",style:{backgroundColor:y[h]||y.idle},title:h})},p=(h,y=0)=>{const k=l.has(h.id),j=h.children&&h.children.length>0,w=S(h);return u.jsxs("div",{className:"tree-node",children:[u.jsxs("div",{className:`tree-node-content ${w?"selected":""} ${h.type}`,style:{paddingLeft:`${y*16+8}px`},onClick:()=>m(h),children:[j&&u.jsx("span",{className:`expand-icon ${k?"expanded":""}`,onClick:P=>{P.stopPropagation(),g(h.id)},children:k?"▼":"▶"}),!j&&u.jsx("span",{className:"expand-icon-placeholder"}),h.type==="agent"&&N(h.status),u.jsx("span",{className:"node-label",children:h.label}),b(h.badges)]}),j&&k&&u.jsx("div",{className:"tree-children",children:h.children.map(P=>p(P,y+1))})]},h.id)};return a&&!r?u.jsx("div",{className:"hierarchy-tree loading",children:"Loading..."}):c?u.jsxs("div",{className:"hierarchy-tree error",children:[u.jsxs("p",{children:["Error: ",c]}),u.jsx("button",{onClick:f,children:"Retry"})]}):u.jsxs("div",{className:"hierarchy-tree",children:[u.jsxs("div",{className:"tree-header",children:[u.jsx("h3",{children:"Agents"}),u.jsx("button",{className:"refresh-btn",onClick:()=>{f(),n==null||n()},title:"Refresh",children:"\\u21BB"})]}),u.jsx("div",{className:"tree-content",children:r&&p(r.root)}),r&&u.jsx("div",{className:"tree-footer",children:u.jsxs("div",{className:"aggregate-stats",children:[u.jsxs("span",{title:"Total agents",children:[r.aggregate.total_agents," agents"]}),u.jsxs("span",{title:"Active",children:[r.aggregate.active_agents," active"]}),r.aggregate.pending_approvals>0&&u.jsxs("span",{className:"pending",title:"Pending approvals",children:[r.aggregate.pending_approvals," pending"]})]})})]})},Ye=({title:e,value:t,color:n="default",small:r})=>u.jsxs("div",{className:`stat-card stat-${n} ${r?"stat-small":""}`,children:[u.jsx("div",{className:"stat-value",children:t}),u.jsx("div",{className:"stat-title",children:e})]}),yg=e=>{if(e<1e3)return`${e}ms`;const t=e/1e3;if(t<60)return`${t.toFixed(1)}s`;const n=Math.floor(t/60),r=(t%60).toFixed(0);return`${n}m ${r}s`},xg=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,Ni=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}k`:e.toString(),kg=({agent:e,onClick:t})=>{var o,a,s,c,d;const n=((o=e.children)==null?void 0:o.length)||0,r=((s=(a=e.badges)==null?void 0:a.find(f=>f.type==="pending"))==null?void 0:s.count)||0,i=((d=(c=e.badges)==null?void 0:c.find(f=>f.type==="running"))==null?void 0:d.count)||0,l={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsxs("div",{className:"agent-card",onClick:t,children:[u.jsxs("div",{className:"agent-card-header",children:[u.jsx("span",{className:"agent-status-dot",style:{backgroundColor:l[e.status||"idle"]}}),u.jsx("span",{className:"agent-name",children:e.label})]}),u.jsxs("div",{className:"agent-card-stats",children:[u.jsxs("span",{className:"agent-stat",children:[u.jsx("span",{className:"agent-stat-value",children:n}),u.jsx("span",{className:"agent-stat-label",children:"threads"})]}),r>0&&u.jsxs("span",{className:"agent-stat pending",children:[u.jsx("span",{className:"agent-stat-value",children:r}),u.jsx("span",{className:"agent-stat-label",children:"pending"})]}),i>0&&u.jsxs("span",{className:"agent-stat running",children:[u.jsx("span",{className:"agent-stat-value",children:i}),u.jsx("span",{className:"agent-stat-label",children:"running"})]})]})]})},wg=({aggregate:e,agents:t,onSelectAgent:n})=>{const r=e.execution,i=r&&r.total_executions>0,l=i?Math.round(r.successful_executions/r.total_executions*100):0;return u.jsxs("div",{className:"all-agents-overview",children:[u.jsx("div",{className:"overview-header",children:u.jsx("h2",{children:"All Agents Overview"})}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Ye,{title:"Total Agents",value:e.total_agents}),u.jsx(Ye,{title:"Active",value:e.active_agents,color:"green"}),u.jsx(Ye,{title:"Pending Approvals",value:e.pending_approvals,color:"orange"}),u.jsx(Ye,{title:"Total Threads",value:e.total_threads,color:"blue"})]}),i&&u.jsxs("div",{className:"execution-stats-section",children:[u.jsx("h3",{children:"Execution Statistics"}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Ye,{title:"Total Executions",value:r.total_executions,color:"purple"}),u.jsx(Ye,{title:"Success Rate",value:`${l}%`,color:"green"}),u.jsx(Ye,{title:"Total Duration",value:yg(r.total_duration_ms)}),u.jsx(Ye,{title:"Total Cost",value:xg(r.total_cost),color:"orange"})]}),u.jsxs("div",{className:"stats-row token-stats",children:[u.jsx(Ye,{title:"Input Tokens",value:Ni(r.total_input_tokens),small:!0}),u.jsx(Ye,{title:"Output Tokens",value:Ni(r.total_output_tokens),small:!0}),u.jsx(Ye,{title:"Cache Read",value:Ni(r.total_cache_read_tokens),small:!0}),u.jsx(Ye,{title:"Cache Created",value:Ni(r.total_cache_create_tokens),small:!0}),u.jsx(Ye,{title:"Files Created",value:r.total_files_created,small:!0})]})]}),u.jsxs("div",{className:"agents-section",children:[u.jsx("h3",{children:"Agents"}),u.jsxs("div",{className:"agent-cards-grid",children:[t.map(o=>u.jsx(kg,{agent:o,onClick:()=>n(o.id)},o.id)),t.length===0&&u.jsx("div",{className:"no-agents",children:"No agents found. Start an agent to see it here."})]})]})]})},Sg=({items:e})=>u.jsx("nav",{className:"breadcrumb",children:e.map((t,n)=>u.jsxs(Wt.Fragment,{children:[n>0&&u.jsx("span",{className:"breadcrumb-separator",children:"/"}),t.onClick?u.jsx("button",{className:"breadcrumb-link",onClick:t.onClick,children:t.label}):u.jsx("span",{className:"breadcrumb-current",children:t.label})]},n))}),Pt={plus:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),u.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"}),u.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),u.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),u.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),u.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),u.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),u.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("polyline",{points:"3 6 5 6 21 6"}),u.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:u.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},bg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=B.useState(!1),[c,d]=B.useState(""),[f,g]=B.useState(null),[m,S]=B.useState(""),[b,N]=B.useState(null),p=()=>{c.trim()&&(r(c.trim()),d(""),s(!1))},h=z=>{z.key==="Enter"&&!z.shiftKey?(z.preventDefault(),p()):z.key==="Escape"&&(s(!1),d(""))},y=(z,M)=>{M.stopPropagation(),g(z.id),S(z.title)},k=z=>{var M;m.trim()&&m.trim()!==((M=e.find(q=>q.id===z))==null?void 0:M.title)&&l(z,m.trim()),g(null),S("")},j=()=>{g(null),S("")},w=(z,M)=>{z.key==="Enter"?(z.preventDefault(),k(M)):z.key==="Escape"&&j()},P=(z,M)=>{M.stopPropagation(),N(z)},A=(z,M)=>{M.stopPropagation(),i(z),N(null)},U=z=>{z.stopPropagation(),N(null)},R=z=>{const M=new Date(z),Y=new Date().getTime()-M.getTime(),H=Math.floor(Y/6e4),W=Math.floor(Y/36e5),re=Math.floor(Y/864e5);return H<1?"now":H<60?`${H}m`:W<24?`${W}h`:re<7?`${re}d`:M.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return u.jsxs("div",{className:"thread-list",children:[u.jsxs("div",{className:"list-header",children:[u.jsx("h2",{children:"Conversations"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:Pt.plus})]}),a&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:c,onChange:z=>d(z.target.value),onKeyDown:h,placeholder:"Conversation title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:p,children:"Create"})]})]}),u.jsx("div",{className:"thread-items",children:e.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Pt.hash}),u.jsx("p",{children:"No conversations yet"}),u.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(z=>{const M=o.get(z.id)||0,q=z.id===t,Y=f===z.id,H=b===z.id;return u.jsxs("div",{className:`thread-item ${q?"selected":""} ${M>0?"has-unread":""}`,onClick:()=>!Y&&n(z.id),children:[u.jsx("div",{className:`status-dot ${z.status}`}),u.jsxs("div",{className:"thread-content",children:[u.jsx("div",{className:"thread-title-row",children:Y?u.jsxs("div",{className:"edit-title-form",onClick:W=>W.stopPropagation(),children:[u.jsx("input",{type:"text",value:m,onChange:W=>S(W.target.value),onKeyDown:W=>w(W,z.id),autoFocus:!0}),u.jsx("button",{className:"edit-action save",onClick:()=>k(z.id),title:"Save",children:Pt.check}),u.jsx("button",{className:"edit-action cancel",onClick:j,title:"Cancel",children:Pt.x})]}):u.jsxs(u.Fragment,{children:[u.jsx("span",{className:"thread-title",children:z.title}),u.jsx("span",{className:"thread-time",children:R(z.updated_at)})]})}),u.jsxs("div",{className:"thread-meta",children:[z.target_agent&&u.jsxs("span",{className:"thread-agent",title:`Target: ${z.target_agent}`,children:[Pt.bot,z.target_agent]}),u.jsxs("span",{className:"thread-seq",children:["#",z.last_seq]})]})]}),!Y&&!H&&u.jsxs("div",{className:"thread-actions",children:[u.jsx("button",{className:"action-btn edit",onClick:W=>y(z,W),title:"Rename",children:Pt.edit}),u.jsx("button",{className:"action-btn delete",onClick:W=>P(z.id,W),title:"Delete",children:Pt.trash})]}),H&&u.jsxs("div",{className:"delete-confirm",onClick:W=>W.stopPropagation(),children:[u.jsx("span",{className:"confirm-text",children:"Delete?"}),u.jsx("button",{className:"confirm-btn yes",onClick:W=>A(z.id,W),title:"Confirm delete",children:Pt.check}),u.jsx("button",{className:"confirm-btn no",onClick:U,title:"Cancel",children:Pt.x})]}),M>0&&!H&&u.jsx("span",{className:"unread-badge",children:M})]},z.id)})}),u.jsx("style",{children:`
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
      `})]})};function Cg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const jg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,Eg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,Ng={};function Gu(e,t){return(Ng.jsx?Eg:jg).test(e)}const _g=/[ \t\n\f\r]/g;function zg(e){return typeof e=="object"?e.type==="text"?Ju(e.value):!1:Ju(e)}function Ju(e){return e.replace(_g,"")===""}class si{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}si.prototype.normal={};si.prototype.property={};si.prototype.space=void 0;function tp(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new si(n,r,t)}function ga(e){return e.toLowerCase()}class qe{constructor(t,n){this.attribute=n,this.property=t}}qe.prototype.attribute="";qe.prototype.booleanish=!1;qe.prototype.boolean=!1;qe.prototype.commaOrSpaceSeparated=!1;qe.prototype.commaSeparated=!1;qe.prototype.defined=!1;qe.prototype.mustUseProperty=!1;qe.prototype.number=!1;qe.prototype.overloadedBoolean=!1;qe.prototype.property="";qe.prototype.spaceSeparated=!1;qe.prototype.space=void 0;let Pg=0;const Q=zn(),ge=zn(),va=zn(),I=zn(),le=zn(),Gn=zn(),Xe=zn();function zn(){return 2**++Pg}const ya=Object.freeze(Object.defineProperty({__proto__:null,boolean:Q,booleanish:ge,commaOrSpaceSeparated:Xe,commaSeparated:Gn,number:I,overloadedBoolean:va,spaceSeparated:le},Symbol.toStringTag,{value:"Module"})),co=Object.keys(ya);class Ss extends qe{constructor(t,n,r,i){let l=-1;if(super(t,n),Zu(this,"space",i),typeof r=="number")for(;++l<co.length;){const o=co[l];Zu(this,co[l],(r&ya[o])===ya[o])}}}Ss.prototype.defined=!0;function Zu(e,t,n){n&&(e[t]=n)}function ur(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new Ss(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[ga(r)]=r,n[ga(l.attribute)]=r}return new si(t,n,e.space)}const np=ur({properties:{ariaActiveDescendant:null,ariaAtomic:ge,ariaAutoComplete:null,ariaBusy:ge,ariaChecked:ge,ariaColCount:I,ariaColIndex:I,ariaColSpan:I,ariaControls:le,ariaCurrent:null,ariaDescribedBy:le,ariaDetails:null,ariaDisabled:ge,ariaDropEffect:le,ariaErrorMessage:null,ariaExpanded:ge,ariaFlowTo:le,ariaGrabbed:ge,ariaHasPopup:null,ariaHidden:ge,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:le,ariaLevel:I,ariaLive:null,ariaModal:ge,ariaMultiLine:ge,ariaMultiSelectable:ge,ariaOrientation:null,ariaOwns:le,ariaPlaceholder:null,ariaPosInSet:I,ariaPressed:ge,ariaReadOnly:ge,ariaRelevant:null,ariaRequired:ge,ariaRoleDescription:le,ariaRowCount:I,ariaRowIndex:I,ariaRowSpan:I,ariaSelected:ge,ariaSetSize:I,ariaSort:null,ariaValueMax:I,ariaValueMin:I,ariaValueNow:I,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function rp(e,t){return t in e?e[t]:t}function ip(e,t){return rp(e,t.toLowerCase())}const Tg=ur({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:Gn,acceptCharset:le,accessKey:le,action:null,allow:null,allowFullScreen:Q,allowPaymentRequest:Q,allowUserMedia:Q,alt:null,as:null,async:Q,autoCapitalize:null,autoComplete:le,autoFocus:Q,autoPlay:Q,blocking:le,capture:null,charSet:null,checked:Q,cite:null,className:le,cols:I,colSpan:null,content:null,contentEditable:ge,controls:Q,controlsList:le,coords:I|Gn,crossOrigin:null,data:null,dateTime:null,decoding:null,default:Q,defer:Q,dir:null,dirName:null,disabled:Q,download:va,draggable:ge,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:Q,formTarget:null,headers:le,height:I,hidden:va,high:I,href:null,hrefLang:null,htmlFor:le,httpEquiv:le,id:null,imageSizes:null,imageSrcSet:null,inert:Q,inputMode:null,integrity:null,is:null,isMap:Q,itemId:null,itemProp:le,itemRef:le,itemScope:Q,itemType:le,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:Q,low:I,manifest:null,max:null,maxLength:I,media:null,method:null,min:null,minLength:I,multiple:Q,muted:Q,name:null,nonce:null,noModule:Q,noValidate:Q,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:Q,optimum:I,pattern:null,ping:le,placeholder:null,playsInline:Q,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:Q,referrerPolicy:null,rel:le,required:Q,reversed:Q,rows:I,rowSpan:I,sandbox:le,scope:null,scoped:Q,seamless:Q,selected:Q,shadowRootClonable:Q,shadowRootDelegatesFocus:Q,shadowRootMode:null,shape:null,size:I,sizes:null,slot:null,span:I,spellCheck:ge,src:null,srcDoc:null,srcLang:null,srcSet:null,start:I,step:null,style:null,tabIndex:I,target:null,title:null,translate:null,type:null,typeMustMatch:Q,useMap:null,value:ge,width:I,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:le,axis:null,background:null,bgColor:null,border:I,borderColor:null,bottomMargin:I,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:Q,declare:Q,event:null,face:null,frame:null,frameBorder:null,hSpace:I,leftMargin:I,link:null,longDesc:null,lowSrc:null,marginHeight:I,marginWidth:I,noResize:Q,noHref:Q,noShade:Q,noWrap:Q,object:null,profile:null,prompt:null,rev:null,rightMargin:I,rules:null,scheme:null,scrolling:ge,standby:null,summary:null,text:null,topMargin:I,valueType:null,version:null,vAlign:null,vLink:null,vSpace:I,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:Q,disableRemotePlayback:Q,prefix:null,property:null,results:I,security:null,unselectable:null},space:"html",transform:ip}),Lg=ur({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:Xe,accentHeight:I,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:I,amplitude:I,arabicForm:null,ascent:I,attributeName:null,attributeType:null,azimuth:I,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:I,by:null,calcMode:null,capHeight:I,className:le,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:I,diffuseConstant:I,direction:null,display:null,dur:null,divisor:I,dominantBaseline:null,download:Q,dx:null,dy:null,edgeMode:null,editable:null,elevation:I,enableBackground:null,end:null,event:null,exponent:I,externalResourcesRequired:null,fill:null,fillOpacity:I,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:Gn,g2:Gn,glyphName:Gn,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:I,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:I,horizOriginX:I,horizOriginY:I,id:null,ideographic:I,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:I,k:I,k1:I,k2:I,k3:I,k4:I,kernelMatrix:Xe,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:I,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:I,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:I,overlineThickness:I,paintOrder:null,panose1:null,path:null,pathLength:I,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:le,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:I,pointsAtY:I,pointsAtZ:I,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:Xe,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:Xe,rev:Xe,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:Xe,requiredFeatures:Xe,requiredFonts:Xe,requiredFormats:Xe,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:I,specularExponent:I,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:I,strikethroughThickness:I,string:null,stroke:null,strokeDashArray:Xe,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:I,strokeOpacity:I,strokeWidth:null,style:null,surfaceScale:I,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:Xe,tabIndex:I,tableValues:null,target:null,targetX:I,targetY:I,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:Xe,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:I,underlineThickness:I,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:I,values:null,vAlphabetic:I,vMathematical:I,vectorEffect:null,vHanging:I,vIdeographic:I,version:null,vertAdvY:I,vertOriginX:I,vertOriginY:I,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:I,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:rp}),lp=ur({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),op=ur({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:ip}),ap=ur({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),Ig={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Ag=/[A-Z]/g,ec=/-[a-z]/g,Mg=/^data[-\w.:]+$/i;function Dg(e,t){const n=ga(t);let r=t,i=qe;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&Mg.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(ec,Og);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!ec.test(l)){let o=l.replace(Ag,Rg);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=Ss}return new i(r,t)}function Rg(e){return"-"+e.toLowerCase()}function Og(e){return e.charAt(1).toUpperCase()}const Fg=tp([np,Tg,lp,op,ap],"html"),bs=tp([np,Lg,lp,op,ap],"svg");function Bg(e){return e.join(" ").trim()}var Cs={},tc=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Ug=/\n/g,Hg=/^\s*/,$g=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Vg=/^:\s*/,Wg=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Qg=/^[;\s]*/,qg=/^\s+|\s+$/g,Kg=`
`,nc="/",rc="*",gn="",Yg="comment",Xg="declaration";function Gg(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(S){var b=S.match(Ug);b&&(n+=b.length);var N=S.lastIndexOf(Kg);r=~N?S.length-N:r+S.length}function l(){var S={line:n,column:r};return function(b){return b.position=new o(S),c(),b}}function o(S){this.start=S,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(S){var b=new Error(t.source+":"+n+":"+r+": "+S);if(b.reason=S,b.filename=t.source,b.line=n,b.column=r,b.source=e,!t.silent)throw b}function s(S){var b=S.exec(e);if(b){var N=b[0];return i(N),e=e.slice(N.length),b}}function c(){s(Hg)}function d(S){var b;for(S=S||[];b=f();)b!==!1&&S.push(b);return S}function f(){var S=l();if(!(nc!=e.charAt(0)||rc!=e.charAt(1))){for(var b=2;gn!=e.charAt(b)&&(rc!=e.charAt(b)||nc!=e.charAt(b+1));)++b;if(b+=2,gn===e.charAt(b-1))return a("End of comment missing");var N=e.slice(2,b-2);return r+=2,i(N),e=e.slice(b),r+=2,S({type:Yg,comment:N})}}function g(){var S=l(),b=s($g);if(b){if(f(),!s(Vg))return a("property missing ':'");var N=s(Wg),p=S({type:Xg,property:ic(b[0].replace(tc,gn)),value:N?ic(N[0].replace(tc,gn)):gn});return s(Qg),p}}function m(){var S=[];d(S);for(var b;b=g();)b!==!1&&(S.push(b),d(S));return S}return c(),m()}function ic(e){return e?e.replace(qg,gn):gn}var Jg=Gg,Zg=Wi&&Wi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(Cs,"__esModule",{value:!0});Cs.default=tv;const ev=Zg(Jg);function tv(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,ev.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Al={};Object.defineProperty(Al,"__esModule",{value:!0});Al.camelCase=void 0;var nv=/^--[a-zA-Z0-9_-]+$/,rv=/-([a-z])/g,iv=/^[^-]+$/,lv=/^-(webkit|moz|ms|o|khtml)-/,ov=/^-(ms)-/,av=function(e){return!e||iv.test(e)||nv.test(e)},sv=function(e,t){return t.toUpperCase()},lc=function(e,t){return"".concat(t,"-")},uv=function(e,t){return t===void 0&&(t={}),av(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(ov,lc):e=e.replace(lv,lc),e.replace(rv,sv))};Al.camelCase=uv;var cv=Wi&&Wi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},dv=cv(Cs),fv=Al;function xa(e,t){var n={};return!e||typeof e!="string"||(0,dv.default)(e,function(r,i){r&&i&&(n[(0,fv.camelCase)(r,t)]=i)}),n}xa.default=xa;var pv=xa;const hv=Na(pv),sp=up("end"),js=up("start");function up(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function mv(e){const t=js(e),n=sp(e);if(t&&n)return{start:t,end:n}}function Dr(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?oc(e.position):"start"in e||"end"in e?oc(e):"line"in e||"column"in e?ka(e):""}function ka(e){return ac(e&&e.line)+":"+ac(e&&e.column)}function oc(e){return ka(e&&e.start)+"-"+ka(e&&e.end)}function ac(e){return e&&typeof e=="number"?e:1}class Le extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Dr(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}Le.prototype.file="";Le.prototype.name="";Le.prototype.reason="";Le.prototype.message="";Le.prototype.stack="";Le.prototype.column=void 0;Le.prototype.line=void 0;Le.prototype.ancestors=void 0;Le.prototype.cause=void 0;Le.prototype.fatal=void 0;Le.prototype.place=void 0;Le.prototype.ruleId=void 0;Le.prototype.source=void 0;const Es={}.hasOwnProperty,gv=new Map,vv=/[A-Z]/g,yv=new Set(["table","tbody","thead","tfoot","tr"]),xv=new Set(["td","th"]),cp="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function kv(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=_v(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=Nv(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?bs:Fg,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=dp(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function dp(e,t,n){if(t.type==="element")return wv(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return Sv(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return Cv(e,t,n);if(t.type==="mdxjsEsm")return bv(e,t);if(t.type==="root")return jv(e,t,n);if(t.type==="text")return Ev(e,t)}function wv(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=bs,e.schema=i),e.ancestors.push(t);const l=pp(e,t.tagName,!1),o=zv(e,t);let a=_s(e,t);return yv.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!zg(s):!0})),fp(e,o,l,t),Ns(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function Sv(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}ni(e,t.position)}function bv(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);ni(e,t.position)}function Cv(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=bs,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:pp(e,t.name,!0),o=Pv(e,t),a=_s(e,t);return fp(e,o,l,t),Ns(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function jv(e,t,n){const r={};return Ns(r,_s(e,t)),e.create(t,e.Fragment,r,n)}function Ev(e,t){return t.value}function fp(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Ns(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function Nv(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function _v(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=js(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function zv(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Es.call(t.properties,i)){const l=Tv(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&xv.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function Pv(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else ni(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else ni(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function _s(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:gv;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const c=i.get(s)||0;o=s+"-"+c,i.set(s,c+1)}}const a=dp(e,l,o);a!==void 0&&n.push(a)}return n}function Tv(e,t,n){const r=Dg(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?Cg(n):Bg(n)),r.property==="style"){let i=typeof n=="object"?n:Lv(e,String(n));return e.stylePropertyNameCase==="css"&&(i=Iv(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?Ig[r.property]||r.property:r.attribute,n]}}function Lv(e,t){try{return hv(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new Le("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=cp+"#cannot-parse-style-attribute",i}}function pp(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=Gu(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=Gu(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Es.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);ni(e)}function ni(e,t){const n=new Le("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=cp+"#cannot-handle-mdx-estrees-without-createevaluater",n}function Iv(e){const t={};let n;for(n in e)Es.call(e,n)&&(t[Av(n)]=e[n]);return t}function Av(e){let t=e.replace(vv,Mv);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function Mv(e){return"-"+e.toLowerCase()}const fo={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},Dv={};function Rv(e,t){const n=Dv,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return hp(e,r,i)}function hp(e,t,n){if(Ov(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return sc(e.children,t,n)}return Array.isArray(e)?sc(e,t,n):""}function sc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=hp(e[i],t,n);return r.join("")}function Ov(e){return!!(e&&typeof e=="object")}const uc=document.createElement("i");function zs(e){const t="&"+e+";";uc.innerHTML=t;const n=uc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function Nt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function at(e,t){return e.length>0?(Nt(e,e.length,0,t),e):t}const cc={}.hasOwnProperty;function Fv(e){const t={};let n=-1;for(;++n<e.length;)Bv(t,e[n]);return t}function Bv(e,t){let n;for(n in t){const i=(cc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){cc.call(i,o)||(i[o]=[]);const a=l[o];Uv(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Uv(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);Nt(e,0,0,r)}function mp(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function Jn(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const Ct=fn(/[A-Za-z]/),Ze=fn(/[\dA-Za-z]/),Hv=fn(/[#-'*+\--9=?A-Z^-~]/);function wa(e){return e!==null&&(e<32||e===127)}const Sa=fn(/\d/),$v=fn(/[\dA-Fa-f]/),Vv=fn(/[!-/:-@[-`{-~]/);function $(e){return e!==null&&e<-2}function Qe(e){return e!==null&&(e<0||e===32)}function Z(e){return e===-2||e===-1||e===32}const Wv=fn(new RegExp("\\p{P}|\\p{S}","u")),Qv=fn(/\s/);function fn(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function cr(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&Ze(e.charCodeAt(n+1))&&Ze(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function ae(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return Z(s)?(e.enter(n),a(s)):t(s)}function a(s){return Z(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const qv={tokenize:Kv};function Kv(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),ae(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return $(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const Yv={tokenize:Xv},dc={tokenize:Gv};function Xv(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const k=n[r];return t.containerState=k[1],e.attempt(k[0].continuation,s,c)(y)}return c(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&h();const k=t.events.length;let j=k,w;for(;j--;)if(t.events[j][0]==="exit"&&t.events[j][1].type==="chunkFlow"){w=t.events[j][1].end;break}p(r);let P=k;for(;P<t.events.length;)t.events[P][1].end={...w},P++;return Nt(t.events,j+1,0,t.events.slice(k)),t.events.length=P,c(y)}return a(y)}function c(y){if(r===n.length){if(!i)return g(y);if(i.currentConstruct&&i.currentConstruct.concrete)return S(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(dc,d,f)(y)}function d(y){return i&&h(),p(r),g(y)}function f(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,S(y)}function g(y){return t.containerState={},e.attempt(dc,m,S)(y)}function m(y){return r++,n.push([t.currentConstruct,t.containerState]),g(y)}function S(y){if(y===null){i&&h(),p(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),b(y)}function b(y){if(y===null){N(e.exit("chunkFlow"),!0),p(0),e.consume(y);return}return $(y)?(e.consume(y),N(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),b)}function N(y,k){const j=t.sliceStream(y);if(k&&j.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(j),t.parser.lazy[y.start.line]){let w=i.events.length;for(;w--;)if(i.events[w][1].start.offset<o&&(!i.events[w][1].end||i.events[w][1].end.offset>o))return;const P=t.events.length;let A=P,U,R;for(;A--;)if(t.events[A][0]==="exit"&&t.events[A][1].type==="chunkFlow"){if(U){R=t.events[A][1].end;break}U=!0}for(p(r),w=P;w<t.events.length;)t.events[w][1].end={...R},w++;Nt(t.events,A+1,0,t.events.slice(P)),t.events.length=w}}function p(y){let k=n.length;for(;k-- >y;){const j=n[k];t.containerState=j[1],j[0].exit.call(t,e)}n.length=y}function h(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Gv(e,t,n){return ae(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function fc(e){if(e===null||Qe(e)||Qv(e))return 1;if(Wv(e))return 2}function Ps(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const ba={name:"attention",resolveAll:Jv,tokenize:Zv};function Jv(e,t){let n=-1,r,i,l,o,a,s,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const f={...e[r][1].end},g={...e[n][1].start};pc(f,-s),pc(g,s),o={type:s>1?"strongSequence":"emphasisSequence",start:f,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:g},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=at(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=at(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=at(c,Ps(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=at(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=at(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,Nt(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Zv(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=fc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const c=e.exit("attentionSequence"),d=fc(s),f=!d||d===2&&i||n.includes(s),g=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?f:f&&(i||!g)),c._close=!!(l===42?g:g&&(d||!f)),t(s)}}function pc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const ey={name:"autolink",tokenize:ty};function ty(e,t,n){let r=0;return i;function i(m){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(m),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(m){return Ct(m)?(e.consume(m),o):m===64?n(m):c(m)}function o(m){return m===43||m===45||m===46||Ze(m)?(r=1,a(m)):c(m)}function a(m){return m===58?(e.consume(m),r=0,s):(m===43||m===45||m===46||Ze(m))&&r++<32?(e.consume(m),a):(r=0,c(m))}function s(m){return m===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(m),e.exit("autolinkMarker"),e.exit("autolink"),t):m===null||m===32||m===60||wa(m)?n(m):(e.consume(m),s)}function c(m){return m===64?(e.consume(m),d):Hv(m)?(e.consume(m),c):n(m)}function d(m){return Ze(m)?f(m):n(m)}function f(m){return m===46?(e.consume(m),r=0,d):m===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(m),e.exit("autolinkMarker"),e.exit("autolink"),t):g(m)}function g(m){if((m===45||Ze(m))&&r++<63){const S=m===45?g:f;return e.consume(m),S}return n(m)}}const Ml={partial:!0,tokenize:ny};function ny(e,t,n){return r;function r(l){return Z(l)?ae(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||$(l)?t(l):n(l)}}const gp={continuation:{tokenize:iy},exit:ly,name:"blockQuote",tokenize:ry};function ry(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return Z(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function iy(e,t,n){const r=this;return i;function i(o){return Z(o)?ae(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(gp,t,n)(o)}}function ly(e){e.exit("blockQuote")}const vp={name:"characterEscape",tokenize:oy};function oy(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return Vv(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const yp={name:"characterReference",tokenize:ay};function ay(e,t,n){const r=this;let i=0,l,o;return a;function a(f){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),s}function s(f){return f===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(f),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=Ze,d(f))}function c(f){return f===88||f===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(f),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=$v,d):(e.enter("characterReferenceValue"),l=7,o=Sa,d(f))}function d(f){if(f===59&&i){const g=e.exit("characterReferenceValue");return o===Ze&&!zs(r.sliceSerialize(g))?n(f):(e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(f)&&i++<l?(e.consume(f),d):n(f)}}const hc={partial:!0,tokenize:uy},mc={concrete:!0,name:"codeFenced",tokenize:sy};function sy(e,t,n){const r=this,i={partial:!0,tokenize:j};let l=0,o=0,a;return s;function s(w){return c(w)}function c(w){const P=r.events[r.events.length-1];return l=P&&P[1].type==="linePrefix"?P[2].sliceSerialize(P[1],!0).length:0,a=w,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(w)}function d(w){return w===a?(o++,e.consume(w),d):o<3?n(w):(e.exit("codeFencedFenceSequence"),Z(w)?ae(e,f,"whitespace")(w):f(w))}function f(w){return w===null||$(w)?(e.exit("codeFencedFence"),r.interrupt?t(w):e.check(hc,b,k)(w)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),g(w))}function g(w){return w===null||$(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),f(w)):Z(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),ae(e,m,"whitespace")(w)):w===96&&w===a?n(w):(e.consume(w),g)}function m(w){return w===null||$(w)?f(w):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),S(w))}function S(w){return w===null||$(w)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),f(w)):w===96&&w===a?n(w):(e.consume(w),S)}function b(w){return e.attempt(i,k,N)(w)}function N(w){return e.enter("lineEnding"),e.consume(w),e.exit("lineEnding"),p}function p(w){return l>0&&Z(w)?ae(e,h,"linePrefix",l+1)(w):h(w)}function h(w){return w===null||$(w)?e.check(hc,b,k)(w):(e.enter("codeFlowValue"),y(w))}function y(w){return w===null||$(w)?(e.exit("codeFlowValue"),h(w)):(e.consume(w),y)}function k(w){return e.exit("codeFenced"),t(w)}function j(w,P,A){let U=0;return R;function R(H){return w.enter("lineEnding"),w.consume(H),w.exit("lineEnding"),z}function z(H){return w.enter("codeFencedFence"),Z(H)?ae(w,M,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(H):M(H)}function M(H){return H===a?(w.enter("codeFencedFenceSequence"),q(H)):A(H)}function q(H){return H===a?(U++,w.consume(H),q):U>=o?(w.exit("codeFencedFenceSequence"),Z(H)?ae(w,Y,"whitespace")(H):Y(H)):A(H)}function Y(H){return H===null||$(H)?(w.exit("codeFencedFence"),P(H)):A(H)}}}function uy(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const po={name:"codeIndented",tokenize:dy},cy={partial:!0,tokenize:fy};function dy(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),ae(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?s(c):$(c)?e.attempt(cy,o,s)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||$(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function s(c){return e.exit("codeIndented"),t(c)}}function fy(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):$(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):ae(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):$(o)?i(o):n(o)}}const py={name:"codeText",previous:my,resolve:hy,tokenize:gy};function hy(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function my(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function gy(e,t,n){let r=0,i,l;return o;function o(f){return e.enter("codeText"),e.enter("codeTextSequence"),a(f)}function a(f){return f===96?(e.consume(f),r++,a):(e.exit("codeTextSequence"),s(f))}function s(f){return f===null?n(f):f===32?(e.enter("space"),e.consume(f),e.exit("space"),s):f===96?(l=e.enter("codeTextSequence"),i=0,d(f)):$(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),s):(e.enter("codeTextData"),c(f))}function c(f){return f===null||f===32||f===96||$(f)?(e.exit("codeTextData"),s(f)):(e.consume(f),c)}function d(f){return f===96?(e.consume(f),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(f)):(l.type="codeTextData",c(f))}}class vy{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&wr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),wr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),wr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);wr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);wr(this.left,n.reverse())}}}function wr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function xp(e){const t={};let n=-1,r,i,l,o,a,s,c;const d=new vy(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,yy(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return Nt(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function yy(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],c={};let d,f,g=-1,m=n,S=0,b=0;const N=[b];for(;m;){for(;e.get(++i)[1]!==m;);l.push(i),m._tokenizer||(d=r.sliceStream(m),m.next||d.push(null),f&&o.defineSkip(m.start),m._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),m._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),f=m,m=m.next}for(m=n;++g<a.length;)a[g][0]==="exit"&&a[g-1][0]==="enter"&&a[g][1].type===a[g-1][1].type&&a[g][1].start.line!==a[g][1].end.line&&(b=g+1,N.push(b),m._tokenizer=void 0,m.previous=void 0,m=m.next);for(o.events=[],m?(m._tokenizer=void 0,m.previous=void 0):N.pop(),g=N.length;g--;){const p=a.slice(N[g],N[g+1]),h=l.pop();s.push([h,h+p.length-1]),e.splice(h,2,p)}for(s.reverse(),g=-1;++g<s.length;)c[S+s[g][0]]=S+s[g][1],S+=s[g][1]-s[g][0]-1;return c}const xy={resolve:wy,tokenize:Sy},ky={partial:!0,tokenize:by};function wy(e){return xp(e),e}function Sy(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):$(a)?e.check(ky,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function by(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),ae(e,l,"linePrefix")}function l(o){if(o===null||$(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function kp(e,t,n,r,i,l,o,a,s){const c=s||Number.POSITIVE_INFINITY;let d=0;return f;function f(p){return p===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(p),e.exit(l),g):p===null||p===32||p===41||wa(p)?n(p):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),b(p))}function g(p){return p===62?(e.enter(l),e.consume(p),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),m(p))}function m(p){return p===62?(e.exit("chunkString"),e.exit(a),g(p)):p===null||p===60||$(p)?n(p):(e.consume(p),p===92?S:m)}function S(p){return p===60||p===62||p===92?(e.consume(p),m):m(p)}function b(p){return!d&&(p===null||p===41||Qe(p))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(p)):d<c&&p===40?(e.consume(p),d++,b):p===41?(e.consume(p),d--,b):p===null||p===32||p===40||wa(p)?n(p):(e.consume(p),p===92?N:b)}function N(p){return p===40||p===41||p===92?(e.consume(p),b):b(p)}}function wp(e,t,n,r,i,l){const o=this;let a=0,s;return c;function c(m){return e.enter(r),e.enter(i),e.consume(m),e.exit(i),e.enter(l),d}function d(m){return a>999||m===null||m===91||m===93&&!s||m===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(m):m===93?(e.exit(l),e.enter(i),e.consume(m),e.exit(i),e.exit(r),t):$(m)?(e.enter("lineEnding"),e.consume(m),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),f(m))}function f(m){return m===null||m===91||m===93||$(m)||a++>999?(e.exit("chunkString"),d(m)):(e.consume(m),s||(s=!Z(m)),m===92?g:f)}function g(m){return m===91||m===92||m===93?(e.consume(m),a++,f):f(m)}}function Sp(e,t,n,r,i,l){let o;return a;function a(g){return g===34||g===39||g===40?(e.enter(r),e.enter(i),e.consume(g),e.exit(i),o=g===40?41:g,s):n(g)}function s(g){return g===o?(e.enter(i),e.consume(g),e.exit(i),e.exit(r),t):(e.enter(l),c(g))}function c(g){return g===o?(e.exit(l),s(o)):g===null?n(g):$(g)?(e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),ae(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(g))}function d(g){return g===o||g===null||$(g)?(e.exit("chunkString"),c(g)):(e.consume(g),g===92?f:d)}function f(g){return g===o||g===92?(e.consume(g),d):d(g)}}function Rr(e,t){let n;return r;function r(i){return $(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):Z(i)?ae(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const Cy={name:"definition",tokenize:Ey},jy={partial:!0,tokenize:Ny};function Ey(e,t,n){const r=this;let i;return l;function l(m){return e.enter("definition"),o(m)}function o(m){return wp.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(m)}function a(m){return i=Jn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),m===58?(e.enter("definitionMarker"),e.consume(m),e.exit("definitionMarker"),s):n(m)}function s(m){return Qe(m)?Rr(e,c)(m):c(m)}function c(m){return kp(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(m)}function d(m){return e.attempt(jy,f,f)(m)}function f(m){return Z(m)?ae(e,g,"whitespace")(m):g(m)}function g(m){return m===null||$(m)?(e.exit("definition"),r.parser.defined.push(i),t(m)):n(m)}}function Ny(e,t,n){return r;function r(a){return Qe(a)?Rr(e,i)(a):n(a)}function i(a){return Sp(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return Z(a)?ae(e,o,"whitespace")(a):o(a)}function o(a){return a===null||$(a)?t(a):n(a)}}const _y={name:"hardBreakEscape",tokenize:zy};function zy(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return $(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const Py={name:"headingAtx",resolve:Ty,tokenize:Ly};function Ty(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},Nt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function Ly(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||Qe(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||$(d)?(e.exit("atxHeading"),t(d)):Z(d)?ae(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||Qe(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const Iy=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],gc=["pre","script","style","textarea"],Ay={concrete:!0,name:"htmlFlow",resolveTo:Ry,tokenize:Oy},My={partial:!0,tokenize:By},Dy={partial:!0,tokenize:Fy};function Ry(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function Oy(e,t,n){const r=this;let i,l,o,a,s;return c;function c(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),f}function f(x){return x===33?(e.consume(x),g):x===47?(e.consume(x),l=!0,b):x===63?(e.consume(x),i=3,r.interrupt?t:v):Ct(x)?(e.consume(x),o=String.fromCharCode(x),N):n(x)}function g(x){return x===45?(e.consume(x),i=2,m):x===91?(e.consume(x),i=5,a=0,S):Ct(x)?(e.consume(x),i=4,r.interrupt?t:v):n(x)}function m(x){return x===45?(e.consume(x),r.interrupt?t:v):n(x)}function S(x){const te="CDATA[";return x===te.charCodeAt(a++)?(e.consume(x),a===te.length?r.interrupt?t:M:S):n(x)}function b(x){return Ct(x)?(e.consume(x),o=String.fromCharCode(x),N):n(x)}function N(x){if(x===null||x===47||x===62||Qe(x)){const te=x===47,Se=o.toLowerCase();return!te&&!l&&gc.includes(Se)?(i=1,r.interrupt?t(x):M(x)):Iy.includes(o.toLowerCase())?(i=6,te?(e.consume(x),p):r.interrupt?t(x):M(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?h(x):y(x))}return x===45||Ze(x)?(e.consume(x),o+=String.fromCharCode(x),N):n(x)}function p(x){return x===62?(e.consume(x),r.interrupt?t:M):n(x)}function h(x){return Z(x)?(e.consume(x),h):R(x)}function y(x){return x===47?(e.consume(x),R):x===58||x===95||Ct(x)?(e.consume(x),k):Z(x)?(e.consume(x),y):R(x)}function k(x){return x===45||x===46||x===58||x===95||Ze(x)?(e.consume(x),k):j(x)}function j(x){return x===61?(e.consume(x),w):Z(x)?(e.consume(x),j):y(x)}function w(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,P):Z(x)?(e.consume(x),w):A(x)}function P(x){return x===s?(e.consume(x),s=null,U):x===null||$(x)?n(x):(e.consume(x),P)}function A(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||Qe(x)?j(x):(e.consume(x),A)}function U(x){return x===47||x===62||Z(x)?y(x):n(x)}function R(x){return x===62?(e.consume(x),z):n(x)}function z(x){return x===null||$(x)?M(x):Z(x)?(e.consume(x),z):n(x)}function M(x){return x===45&&i===2?(e.consume(x),W):x===60&&i===1?(e.consume(x),re):x===62&&i===4?(e.consume(x),T):x===63&&i===3?(e.consume(x),v):x===93&&i===5?(e.consume(x),E):$(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(My,F,q)(x)):x===null||$(x)?(e.exit("htmlFlowData"),q(x)):(e.consume(x),M)}function q(x){return e.check(Dy,Y,F)(x)}function Y(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),H}function H(x){return x===null||$(x)?q(x):(e.enter("htmlFlowData"),M(x))}function W(x){return x===45?(e.consume(x),v):M(x)}function re(x){return x===47?(e.consume(x),o="",C):M(x)}function C(x){if(x===62){const te=o.toLowerCase();return gc.includes(te)?(e.consume(x),T):M(x)}return Ct(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),C):M(x)}function E(x){return x===93?(e.consume(x),v):M(x)}function v(x){return x===62?(e.consume(x),T):x===45&&i===2?(e.consume(x),v):M(x)}function T(x){return x===null||$(x)?(e.exit("htmlFlowData"),F(x)):(e.consume(x),T)}function F(x){return e.exit("htmlFlow"),t(x)}}function Fy(e,t,n){const r=this;return i;function i(o){return $(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function By(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Ml,t,n)}}const Uy={name:"htmlText",tokenize:Hy};function Hy(e,t,n){const r=this;let i,l,o;return a;function a(v){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(v),s}function s(v){return v===33?(e.consume(v),c):v===47?(e.consume(v),j):v===63?(e.consume(v),y):Ct(v)?(e.consume(v),A):n(v)}function c(v){return v===45?(e.consume(v),d):v===91?(e.consume(v),l=0,S):Ct(v)?(e.consume(v),h):n(v)}function d(v){return v===45?(e.consume(v),m):n(v)}function f(v){return v===null?n(v):v===45?(e.consume(v),g):$(v)?(o=f,re(v)):(e.consume(v),f)}function g(v){return v===45?(e.consume(v),m):f(v)}function m(v){return v===62?W(v):v===45?g(v):f(v)}function S(v){const T="CDATA[";return v===T.charCodeAt(l++)?(e.consume(v),l===T.length?b:S):n(v)}function b(v){return v===null?n(v):v===93?(e.consume(v),N):$(v)?(o=b,re(v)):(e.consume(v),b)}function N(v){return v===93?(e.consume(v),p):b(v)}function p(v){return v===62?W(v):v===93?(e.consume(v),p):b(v)}function h(v){return v===null||v===62?W(v):$(v)?(o=h,re(v)):(e.consume(v),h)}function y(v){return v===null?n(v):v===63?(e.consume(v),k):$(v)?(o=y,re(v)):(e.consume(v),y)}function k(v){return v===62?W(v):y(v)}function j(v){return Ct(v)?(e.consume(v),w):n(v)}function w(v){return v===45||Ze(v)?(e.consume(v),w):P(v)}function P(v){return $(v)?(o=P,re(v)):Z(v)?(e.consume(v),P):W(v)}function A(v){return v===45||Ze(v)?(e.consume(v),A):v===47||v===62||Qe(v)?U(v):n(v)}function U(v){return v===47?(e.consume(v),W):v===58||v===95||Ct(v)?(e.consume(v),R):$(v)?(o=U,re(v)):Z(v)?(e.consume(v),U):W(v)}function R(v){return v===45||v===46||v===58||v===95||Ze(v)?(e.consume(v),R):z(v)}function z(v){return v===61?(e.consume(v),M):$(v)?(o=z,re(v)):Z(v)?(e.consume(v),z):U(v)}function M(v){return v===null||v===60||v===61||v===62||v===96?n(v):v===34||v===39?(e.consume(v),i=v,q):$(v)?(o=M,re(v)):Z(v)?(e.consume(v),M):(e.consume(v),Y)}function q(v){return v===i?(e.consume(v),i=void 0,H):v===null?n(v):$(v)?(o=q,re(v)):(e.consume(v),q)}function Y(v){return v===null||v===34||v===39||v===60||v===61||v===96?n(v):v===47||v===62||Qe(v)?U(v):(e.consume(v),Y)}function H(v){return v===47||v===62||Qe(v)?U(v):n(v)}function W(v){return v===62?(e.consume(v),e.exit("htmlTextData"),e.exit("htmlText"),t):n(v)}function re(v){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(v),e.exit("lineEnding"),C}function C(v){return Z(v)?ae(e,E,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(v):E(v)}function E(v){return e.enter("htmlTextData"),o(v)}}const Ts={name:"labelEnd",resolveAll:Qy,resolveTo:qy,tokenize:Ky},$y={tokenize:Yy},Vy={tokenize:Xy},Wy={tokenize:Gy};function Qy(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&Nt(e,0,e.length,n),e}function qy(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",c,t]],a=at(a,e.slice(l+1,l+r+3)),a=at(a,[["enter",d,t]]),a=at(a,Ps(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=at(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=at(a,e.slice(o+1)),a=at(a,[["exit",s,t]]),Nt(e,l,e.length,a),e}function Ky(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(g){return l?l._inactive?f(g):(o=r.parser.defined.includes(Jn(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(g),e.exit("labelMarker"),e.exit("labelEnd"),s):n(g)}function s(g){return g===40?e.attempt($y,d,o?d:f)(g):g===91?e.attempt(Vy,d,o?c:f)(g):o?d(g):f(g)}function c(g){return e.attempt(Wy,d,f)(g)}function d(g){return t(g)}function f(g){return l._balanced=!0,n(g)}}function Yy(e,t,n){return r;function r(f){return e.enter("resource"),e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),i}function i(f){return Qe(f)?Rr(e,l)(f):l(f)}function l(f){return f===41?d(f):kp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(f)}function o(f){return Qe(f)?Rr(e,s)(f):d(f)}function a(f){return n(f)}function s(f){return f===34||f===39||f===40?Sp(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(f):d(f)}function c(f){return Qe(f)?Rr(e,d)(f):d(f)}function d(f){return f===41?(e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),e.exit("resource"),t):n(f)}}function Xy(e,t,n){const r=this;return i;function i(a){return wp.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(Jn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Gy(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Jy={name:"labelStartImage",resolveAll:Ts.resolveAll,tokenize:Zy};function Zy(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const ex={name:"labelStartLink",resolveAll:Ts.resolveAll,tokenize:tx};function tx(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const ho={name:"lineEnding",tokenize:nx};function nx(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),ae(e,t,"linePrefix")}}const $i={name:"thematicBreak",tokenize:rx};function rx(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),s(c)):r>=3&&(c===null||$(c))?(e.exit("thematicBreak"),t(c)):n(c)}function s(c){return c===i?(e.consume(c),r++,s):(e.exit("thematicBreakSequence"),Z(c)?ae(e,a,"whitespace")(c):a(c))}}const Be={continuation:{tokenize:ax},exit:ux,name:"list",tokenize:ox},ix={partial:!0,tokenize:cx},lx={partial:!0,tokenize:sx};function ox(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(m){const S=r.containerState.type||(m===42||m===43||m===45?"listUnordered":"listOrdered");if(S==="listUnordered"?!r.containerState.marker||m===r.containerState.marker:Sa(m)){if(r.containerState.type||(r.containerState.type=S,e.enter(S,{_container:!0})),S==="listUnordered")return e.enter("listItemPrefix"),m===42||m===45?e.check($i,n,c)(m):c(m);if(!r.interrupt||m===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(m)}return n(m)}function s(m){return Sa(m)&&++o<10?(e.consume(m),s):(!r.interrupt||o<2)&&(r.containerState.marker?m===r.containerState.marker:m===41||m===46)?(e.exit("listItemValue"),c(m)):n(m)}function c(m){return e.enter("listItemMarker"),e.consume(m),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||m,e.check(Ml,r.interrupt?n:d,e.attempt(ix,g,f))}function d(m){return r.containerState.initialBlankLine=!0,l++,g(m)}function f(m){return Z(m)?(e.enter("listItemPrefixWhitespace"),e.consume(m),e.exit("listItemPrefixWhitespace"),g):n(m)}function g(m){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(m)}}function ax(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Ml,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,ae(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!Z(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(lx,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,ae(e,e.attempt(Be,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function sx(e,t,n){const r=this;return ae(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function ux(e){e.exit(this.containerState.type)}function cx(e,t,n){const r=this;return ae(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!Z(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const vc={name:"setextUnderline",resolveTo:dx,tokenize:fx};function dx(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function fx(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,f;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){f=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||f)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),Z(c)?ae(e,s,"lineSuffix")(c):s(c))}function s(c){return c===null||$(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const px={tokenize:hx};function hx(e){const t=this,n=e.attempt(Ml,r,e.attempt(this.parser.constructs.flowInitial,i,ae(e,e.attempt(this.parser.constructs.flow,i,e.attempt(xy,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const mx={resolveAll:Cp()},gx=bp("string"),vx=bp("text");function bp(e){return{resolveAll:Cp(e==="text"?yx:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function c(d){if(d===null)return!0;const f=i[d];let g=-1;if(f)for(;++g<f.length;){const m=f[g];if(!m.previous||m.previous.call(r,r.previous))return!0}return!1}}}function Cp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function yx(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)s=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const xx={42:Be,43:Be,45:Be,48:Be,49:Be,50:Be,51:Be,52:Be,53:Be,54:Be,55:Be,56:Be,57:Be,62:gp},kx={91:Cy},wx={[-2]:po,[-1]:po,32:po},Sx={35:Py,42:$i,45:[vc,$i],60:Ay,61:vc,95:$i,96:mc,126:mc},bx={38:yp,92:vp},Cx={[-5]:ho,[-4]:ho,[-3]:ho,33:Jy,38:yp,42:ba,60:[ey,Uy],91:ex,92:[_y,vp],93:Ts,95:ba,96:py},jx={null:[ba,mx]},Ex={null:[42,95]},Nx={null:[]},_x=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:Ex,contentInitial:kx,disable:Nx,document:xx,flow:Sx,flowInitial:wx,insideSpan:jx,string:bx,text:Cx},Symbol.toStringTag,{value:"Module"}));function zx(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:P(j),check:P(w),consume:h,enter:y,exit:k,interrupt:P(w,{interrupt:!0})},c={code:null,containerState:{},defineSkip:b,events:[],now:S,parser:e,previous:null,sliceSerialize:g,sliceStream:m,write:f};let d=t.tokenize.call(c,s);return t.resolveAll&&l.push(t),c;function f(z){return o=at(o,z),N(),o[o.length-1]!==null?[]:(A(t,0),c.events=Ps(l,c.events,c),c.events)}function g(z,M){return Tx(m(z),M)}function m(z){return Px(o,z)}function S(){const{_bufferIndex:z,_index:M,line:q,column:Y,offset:H}=r;return{_bufferIndex:z,_index:M,line:q,column:Y,offset:H}}function b(z){i[z.line]=z.column,R()}function N(){let z;for(;r._index<o.length;){const M=o[r._index];if(typeof M=="string")for(z=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===z&&r._bufferIndex<M.length;)p(M.charCodeAt(r._bufferIndex));else p(M)}}function p(z){d=d(z)}function h(z){$(z)?(r.line++,r.column=1,r.offset+=z===-3?2:1,R()):z!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=z}function y(z,M){const q=M||{};return q.type=z,q.start=S(),c.events.push(["enter",q,c]),a.push(q),q}function k(z){const M=a.pop();return M.end=S(),c.events.push(["exit",M,c]),M}function j(z,M){A(z,M.from)}function w(z,M){M.restore()}function P(z,M){return q;function q(Y,H,W){let re,C,E,v;return Array.isArray(Y)?F(Y):"tokenize"in Y?F([Y]):T(Y);function T(ee){return Ie;function Ie(it){const G=it!==null&&ee[it],be=it!==null&&ee.null,Fe=[...Array.isArray(G)?G:G?[G]:[],...Array.isArray(be)?be:be?[be]:[]];return F(Fe)(it)}}function F(ee){return re=ee,C=0,ee.length===0?W:x(ee[C])}function x(ee){return Ie;function Ie(it){return v=U(),E=ee,ee.partial||(c.currentConstruct=ee),ee.name&&c.parser.constructs.disable.null.includes(ee.name)?Se():ee.tokenize.call(M?Object.assign(Object.create(c),M):c,s,te,Se)(it)}}function te(ee){return z(E,v),H}function Se(ee){return v.restore(),++C<re.length?x(re[C]):W}}}function A(z,M){z.resolveAll&&!l.includes(z)&&l.push(z),z.resolve&&Nt(c.events,M,c.events.length-M,z.resolve(c.events.slice(M),c)),z.resolveTo&&(c.events=z.resolveTo(c.events,c))}function U(){const z=S(),M=c.previous,q=c.currentConstruct,Y=c.events.length,H=Array.from(a);return{from:Y,restore:W};function W(){r=z,c.previous=M,c.currentConstruct=q,c.events.length=Y,a=H,R()}}function R(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function Px(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function Tx(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function Lx(e){const r={constructs:Fv([_x,...(e||{}).extensions||[]]),content:i(qv),defined:[],document:i(Yv),flow:i(px),lazy:{},string:i(gx),text:i(vx)};return r;function i(l){return o;function o(a){return zx(r,l,a)}}}function Ix(e){for(;!xp(e););return e}const yc=/[\0\t\n\r]/g;function Ax(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let c,d,f,g,m;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),f=0,t="",n&&(l.charCodeAt(0)===65279&&f++,n=void 0);f<l.length;){if(yc.lastIndex=f,c=yc.exec(l),g=c&&c.index!==void 0?c.index:l.length,m=l.charCodeAt(g),!c){t=l.slice(f);break}if(m===10&&f===g&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),f<g&&(s.push(l.slice(f,g)),e+=g-f),m){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}f=g+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const Mx=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function Dx(e){return e.replace(Mx,Rx)}function Rx(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return mp(n.slice(l?2:1),l?16:10)}return zs(n)||e}const jp={}.hasOwnProperty;function Ox(e,t,n){return typeof t!="string"&&(n=t,t=void 0),Fx(n)(Ix(Lx(n).document().write(Ax()(e,t,!0))))}function Fx(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Bs),autolinkProtocol:U,autolinkEmail:U,atxHeading:l(Rs),blockQuote:l(be),characterEscape:U,characterReference:U,codeFenced:l(Fe),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(Fe,o),codeText:l(Ut,o),codeTextData:U,data:U,codeFlowValue:U,definition:l(Ht),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Rp),hardBreakEscape:l(Os),hardBreakTrailing:l(Os),htmlFlow:l(Fs,o),htmlFlowData:U,htmlText:l(Fs,o),htmlTextData:U,image:l(Op),label:o,link:l(Bs),listItem:l(Fp),listItemValue:g,listOrdered:l(Us,f),listUnordered:l(Us),paragraph:l(Bp),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Rs),strong:l(Up),thematicBreak:l($p)},exit:{atxHeading:s(),atxHeadingSequence:j,autolink:s(),autolinkEmail:G,autolinkProtocol:it,blockQuote:s(),characterEscapeValue:R,characterReferenceMarkerHexadecimal:Se,characterReferenceMarkerNumeric:Se,characterReferenceValue:ee,characterReference:Ie,codeFenced:s(N),codeFencedFence:b,codeFencedFenceInfo:m,codeFencedFenceMeta:S,codeFlowValue:R,codeIndented:s(p),codeText:s(H),codeTextData:R,data:R,definition:s(),definitionDestinationString:k,definitionLabelString:h,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(M),hardBreakTrailing:s(M),htmlFlow:s(q),htmlFlowData:R,htmlText:s(Y),htmlTextData:R,image:s(re),label:E,labelText:C,lineEnding:z,link:s(W),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:te,resourceDestinationString:v,resourceTitleString:T,resource:F,setextHeading:s(A),setextHeadingLineSequence:P,setextHeadingText:w,strong:s(),thematicBreak:s()}};Ep(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(_){let D={type:"root",children:[]};const V={stack:[D],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},X=[];let ie=-1;for(;++ie<_.length;)if(_[ie][1].type==="listOrdered"||_[ie][1].type==="listUnordered")if(_[ie][0]==="enter")X.push(ie);else{const ft=X.pop();ie=i(_,ft,ie)}for(ie=-1;++ie<_.length;){const ft=t[_[ie][0]];jp.call(ft,_[ie][1].type)&&ft[_[ie][1].type].call(Object.assign({sliceSerialize:_[ie][2].sliceSerialize},V),_[ie][1])}if(V.tokenStack.length>0){const ft=V.tokenStack[V.tokenStack.length-1];(ft[1]||xc).call(V,void 0,ft[0])}for(D.position={start:Vt(_.length>0?_[0][1].start:{line:1,column:1,offset:0}),end:Vt(_.length>0?_[_.length-2][1].end:{line:1,column:1,offset:0})},ie=-1;++ie<t.transforms.length;)D=t.transforms[ie](D)||D;return D}function i(_,D,V){let X=D-1,ie=-1,ft=!1,pn,_t,dr,fr;for(;++X<=V;){const Ke=_[X];switch(Ke[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Ke[0]==="enter"?ie++:ie--,fr=void 0;break}case"lineEndingBlank":{Ke[0]==="enter"&&(pn&&!fr&&!ie&&!dr&&(dr=X),fr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:fr=void 0}if(!ie&&Ke[0]==="enter"&&Ke[1].type==="listItemPrefix"||ie===-1&&Ke[0]==="exit"&&(Ke[1].type==="listUnordered"||Ke[1].type==="listOrdered")){if(pn){let Pn=X;for(_t=void 0;Pn--;){const zt=_[Pn];if(zt[1].type==="lineEnding"||zt[1].type==="lineEndingBlank"){if(zt[0]==="exit")continue;_t&&(_[_t][1].type="lineEndingBlank",ft=!0),zt[1].type="lineEnding",_t=Pn}else if(!(zt[1].type==="linePrefix"||zt[1].type==="blockQuotePrefix"||zt[1].type==="blockQuotePrefixWhitespace"||zt[1].type==="blockQuoteMarker"||zt[1].type==="listItemIndent"))break}dr&&(!_t||dr<_t)&&(pn._spread=!0),pn.end=Object.assign({},_t?_[_t][1].start:Ke[1].end),_.splice(_t||X,0,["exit",pn,Ke[2]]),X++,V++}if(Ke[1].type==="listItemPrefix"){const Pn={type:"listItem",_spread:!1,start:Object.assign({},Ke[1].start),end:void 0};pn=Pn,_.splice(X,0,["enter",Pn,Ke[2]]),X++,V++,dr=void 0,fr=!0}}}return _[D][1]._spread=ft,V}function l(_,D){return V;function V(X){a.call(this,_(X),X),D&&D.call(this,X)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(_,D,V){this.stack[this.stack.length-1].children.push(_),this.stack.push(_),this.tokenStack.push([D,V||void 0]),_.position={start:Vt(D.start),end:void 0}}function s(_){return D;function D(V){_&&_.call(this,V),c.call(this,V)}}function c(_,D){const V=this.stack.pop(),X=this.tokenStack.pop();if(X)X[0].type!==_.type&&(D?D.call(this,_,X[0]):(X[1]||xc).call(this,_,X[0]));else throw new Error("Cannot close `"+_.type+"` ("+Dr({start:_.start,end:_.end})+"): it’s not open");V.position.end=Vt(_.end)}function d(){return Rv(this.stack.pop())}function f(){this.data.expectingFirstListItemValue=!0}function g(_){if(this.data.expectingFirstListItemValue){const D=this.stack[this.stack.length-2];D.start=Number.parseInt(this.sliceSerialize(_),10),this.data.expectingFirstListItemValue=void 0}}function m(){const _=this.resume(),D=this.stack[this.stack.length-1];D.lang=_}function S(){const _=this.resume(),D=this.stack[this.stack.length-1];D.meta=_}function b(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function N(){const _=this.resume(),D=this.stack[this.stack.length-1];D.value=_.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function p(){const _=this.resume(),D=this.stack[this.stack.length-1];D.value=_.replace(/(\r?\n|\r)$/g,"")}function h(_){const D=this.resume(),V=this.stack[this.stack.length-1];V.label=D,V.identifier=Jn(this.sliceSerialize(_)).toLowerCase()}function y(){const _=this.resume(),D=this.stack[this.stack.length-1];D.title=_}function k(){const _=this.resume(),D=this.stack[this.stack.length-1];D.url=_}function j(_){const D=this.stack[this.stack.length-1];if(!D.depth){const V=this.sliceSerialize(_).length;D.depth=V}}function w(){this.data.setextHeadingSlurpLineEnding=!0}function P(_){const D=this.stack[this.stack.length-1];D.depth=this.sliceSerialize(_).codePointAt(0)===61?1:2}function A(){this.data.setextHeadingSlurpLineEnding=void 0}function U(_){const V=this.stack[this.stack.length-1].children;let X=V[V.length-1];(!X||X.type!=="text")&&(X=Hp(),X.position={start:Vt(_.start),end:void 0},V.push(X)),this.stack.push(X)}function R(_){const D=this.stack.pop();D.value+=this.sliceSerialize(_),D.position.end=Vt(_.end)}function z(_){const D=this.stack[this.stack.length-1];if(this.data.atHardBreak){const V=D.children[D.children.length-1];V.position.end=Vt(_.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(D.type)&&(U.call(this,_),R.call(this,_))}function M(){this.data.atHardBreak=!0}function q(){const _=this.resume(),D=this.stack[this.stack.length-1];D.value=_}function Y(){const _=this.resume(),D=this.stack[this.stack.length-1];D.value=_}function H(){const _=this.resume(),D=this.stack[this.stack.length-1];D.value=_}function W(){const _=this.stack[this.stack.length-1];if(this.data.inReference){const D=this.data.referenceType||"shortcut";_.type+="Reference",_.referenceType=D,delete _.url,delete _.title}else delete _.identifier,delete _.label;this.data.referenceType=void 0}function re(){const _=this.stack[this.stack.length-1];if(this.data.inReference){const D=this.data.referenceType||"shortcut";_.type+="Reference",_.referenceType=D,delete _.url,delete _.title}else delete _.identifier,delete _.label;this.data.referenceType=void 0}function C(_){const D=this.sliceSerialize(_),V=this.stack[this.stack.length-2];V.label=Dx(D),V.identifier=Jn(D).toLowerCase()}function E(){const _=this.stack[this.stack.length-1],D=this.resume(),V=this.stack[this.stack.length-1];if(this.data.inReference=!0,V.type==="link"){const X=_.children;V.children=X}else V.alt=D}function v(){const _=this.resume(),D=this.stack[this.stack.length-1];D.url=_}function T(){const _=this.resume(),D=this.stack[this.stack.length-1];D.title=_}function F(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function te(_){const D=this.resume(),V=this.stack[this.stack.length-1];V.label=D,V.identifier=Jn(this.sliceSerialize(_)).toLowerCase(),this.data.referenceType="full"}function Se(_){this.data.characterReferenceType=_.type}function ee(_){const D=this.sliceSerialize(_),V=this.data.characterReferenceType;let X;V?(X=mp(D,V==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):X=zs(D);const ie=this.stack[this.stack.length-1];ie.value+=X}function Ie(_){const D=this.stack.pop();D.position.end=Vt(_.end)}function it(_){R.call(this,_);const D=this.stack[this.stack.length-1];D.url=this.sliceSerialize(_)}function G(_){R.call(this,_);const D=this.stack[this.stack.length-1];D.url="mailto:"+this.sliceSerialize(_)}function be(){return{type:"blockquote",children:[]}}function Fe(){return{type:"code",lang:null,meta:null,value:""}}function Ut(){return{type:"inlineCode",value:""}}function Ht(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Rp(){return{type:"emphasis",children:[]}}function Rs(){return{type:"heading",depth:0,children:[]}}function Os(){return{type:"break"}}function Fs(){return{type:"html",value:""}}function Op(){return{type:"image",title:null,url:"",alt:null}}function Bs(){return{type:"link",title:null,url:"",children:[]}}function Us(_){return{type:"list",ordered:_.type==="listOrdered",start:null,spread:_._spread,children:[]}}function Fp(_){return{type:"listItem",spread:_._spread,checked:null,children:[]}}function Bp(){return{type:"paragraph",children:[]}}function Up(){return{type:"strong",children:[]}}function Hp(){return{type:"text",value:""}}function $p(){return{type:"thematicBreak"}}}function Vt(e){return{line:e.line,column:e.column,offset:e.offset}}function Ep(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Ep(e,r):Bx(e,r)}}function Bx(e,t){let n;for(n in t)if(jp.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function xc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Dr({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Dr({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Dr({start:t.start,end:t.end})+") is still open")}function Ux(e){const t=this;t.parser=n;function n(r){return Ox(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function Hx(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function $x(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function Vx(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function Wx(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Qx(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function qx(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=cr(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const c={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,c),e.applyData(t,c)}function Kx(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Yx(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Np(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function Xx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Np(e,t);const i={src:cr(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function Gx(e,t){const n={src:cr(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function Jx(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function Zx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Np(e,t);const i={href:cr(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function e1(e,t){const n={href:cr(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function t1(e,t,n){const r=e.all(t),i=n?n1(n):_p(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let f;d&&d.type==="element"&&d.tagName==="p"?f=d:(f={type:"element",tagName:"p",properties:{},children:[]},r.unshift(f)),f.children.length>0&&f.children.unshift({type:"text",value:" "}),f.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function n1(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=_p(n[r])}return t}function _p(e){const t=e.spread;return t??e.children.length>1}function r1(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function i1(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function l1(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function o1(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function a1(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=js(t.children[1]),s=sp(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function s1(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const c=[];for(;++s<a;){const f=t.children[s],g={},m=o?o[s]:void 0;m&&(g.align=m);let S={type:"element",tagName:l,properties:g,children:[]};f&&(S.children=e.all(f),e.patch(f,S),S=e.applyData(f,S)),c.push(S)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function u1(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const kc=9,wc=32;function c1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(Sc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(Sc(t.slice(i),i>0,!1)),l.join("")}function Sc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===kc||l===wc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===kc||l===wc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function d1(e,t){const n={type:"text",value:c1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function f1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const p1={blockquote:Hx,break:$x,code:Vx,delete:Wx,emphasis:Qx,footnoteReference:qx,heading:Kx,html:Yx,imageReference:Xx,image:Gx,inlineCode:Jx,linkReference:Zx,link:e1,listItem:t1,list:r1,paragraph:i1,root:l1,strong:o1,table:a1,tableCell:u1,tableRow:s1,text:d1,thematicBreak:f1,toml:_i,yaml:_i,definition:_i,footnoteDefinition:_i};function _i(){}const zp=-1,Dl=0,Or=1,gl=2,Ls=3,Is=4,As=5,Ms=6,Pp=7,Tp=8,bc=typeof self=="object"?self:globalThis,h1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Dl:case zp:return n(o,i);case Or:{const a=n([],i);for(const s of o)a.push(r(s));return a}case gl:{const a=n({},i);for(const[s,c]of o)a[r(s)]=r(c);return a}case Ls:return n(new Date(o),i);case Is:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case As:{const a=n(new Map,i);for(const[s,c]of o)a.set(r(s),r(c));return a}case Ms:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Pp:{const{name:a,message:s}=o;return n(new bc[a](s),i)}case Tp:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new bc[l](o),i)};return r},Cc=e=>h1(new Map,e)(0),Ln="",{toString:m1}={},{keys:g1}=Object,Sr=e=>{const t=typeof e;if(t!=="object"||!e)return[Dl,t];const n=m1.call(e).slice(8,-1);switch(n){case"Array":return[Or,Ln];case"Object":return[gl,Ln];case"Date":return[Ls,Ln];case"RegExp":return[Is,Ln];case"Map":return[As,Ln];case"Set":return[Ms,Ln];case"DataView":return[Or,n]}return n.includes("Array")?[Or,n]:n.includes("Error")?[Pp,n]:[gl,n]},zi=([e,t])=>e===Dl&&(t==="function"||t==="symbol"),v1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=Sr(o);switch(a){case Dl:{let d=o;switch(s){case"bigint":a=Tp,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([zp],o)}return i([a,d],o)}case Or:{if(s){let g=o;return s==="DataView"?g=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(g=new Uint8Array(o)),i([s,[...g]],o)}const d=[],f=i([a,d],o);for(const g of o)d.push(l(g));return f}case gl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],f=i([a,d],o);for(const g of g1(o))(e||!zi(Sr(o[g])))&&d.push([l(g),l(o[g])]);return f}case Ls:return i([a,o.toISOString()],o);case Is:{const{source:d,flags:f}=o;return i([a,{source:d,flags:f}],o)}case As:{const d=[],f=i([a,d],o);for(const[g,m]of o)(e||!(zi(Sr(g))||zi(Sr(m))))&&d.push([l(g),l(m)]);return f}case Ms:{const d=[],f=i([a,d],o);for(const g of o)(e||!zi(Sr(g)))&&d.push(l(g));return f}}const{message:c}=o;return i([a,{name:s,message:c}],o)};return l},jc=(e,{json:t,lossy:n}={})=>{const r=[];return v1(!(t||n),!!t,new Map,r)(e),r},vl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?Cc(jc(e,t)):structuredClone(e):(e,t)=>Cc(jc(e,t));function y1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function x1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function k1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||y1,r=e.options.footnoteBackLabel||x1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[s]);if(!c)continue;const d=e.all(c),f=String(c.identifier).toUpperCase(),g=cr(f.toLowerCase());let m=0;const S=[],b=e.footnoteCounts.get(f);for(;b!==void 0&&++m<=b;){S.length>0&&S.push({type:"text",value:" "});let h=typeof n=="string"?n:n(s,m);typeof h=="string"&&(h={type:"text",value:h}),S.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+g+(m>1?"-"+m:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,m),className:["data-footnote-backref"]},children:Array.isArray(h)?h:[h]})}const N=d[d.length-1];if(N&&N.type==="element"&&N.tagName==="p"){const h=N.children[N.children.length-1];h&&h.type==="text"?h.value+=" ":N.children.push({type:"text",value:" "}),N.children.push(...S)}else d.push(...S);const p={type:"element",tagName:"li",properties:{id:t+"fn-"+g},children:e.wrap(d,!0)};e.patch(c,p),a.push(p)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...vl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const Lp=function(e){if(e==null)return C1;if(typeof e=="function")return Rl(e);if(typeof e=="object")return Array.isArray(e)?w1(e):S1(e);if(typeof e=="string")return b1(e);throw new Error("Expected function, string, or object as test")};function w1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=Lp(e[n]);return Rl(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function S1(e){const t=e;return Rl(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function b1(e){return Rl(t);function t(n){return n&&n.type===e}}function Rl(e){return t;function t(n,r,i){return!!(j1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function C1(){return!0}function j1(e){return e!==null&&typeof e=="object"&&"type"in e}const Ip=[],E1=!0,Ec=!1,N1="skip";function _1(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=Lp(i),o=r?-1:1;a(e,void 0,[])();function a(s,c,d){const f=s&&typeof s=="object"?s:{};if(typeof f.type=="string"){const m=typeof f.tagName=="string"?f.tagName:typeof f.name=="string"?f.name:void 0;Object.defineProperty(g,"name",{value:"node ("+(s.type+(m?"<"+m+">":""))+")"})}return g;function g(){let m=Ip,S,b,N;if((!t||l(s,c,d[d.length-1]||void 0))&&(m=z1(n(s,d)),m[0]===Ec))return m;if("children"in s&&s.children){const p=s;if(p.children&&m[0]!==N1)for(b=(r?p.children.length:-1)+o,N=d.concat(p);b>-1&&b<p.children.length;){const h=p.children[b];if(S=a(h,b,N)(),S[0]===Ec)return S;b=typeof S[1]=="number"?S[1]:b+o}}return m}}}function z1(e){return Array.isArray(e)?e:typeof e=="number"?[E1,e]:e==null?Ip:[e]}function Ap(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),_1(e,l,a,i);function a(s,c){const d=c[c.length-1],f=d?d.children.indexOf(s):void 0;return o(s,f,d)}}const Ca={}.hasOwnProperty,P1={};function T1(e,t){const n=t||P1,r=new Map,i=new Map,l=new Map,o={...p1,...n.handlers},a={all:c,applyData:I1,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:L1,wrap:M1};return Ap(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const f=d.type==="definition"?r:i,g=String(d.identifier).toUpperCase();f.has(g)||f.set(g,d)}}),a;function s(d,f){const g=d.type,m=a.handlers[g];if(Ca.call(a.handlers,g)&&m)return m(a,d,f);if(a.options.passThrough&&a.options.passThrough.includes(g)){if("children"in d){const{children:b,...N}=d,p=vl(N);return p.children=a.all(d),p}return vl(d)}return(a.options.unknownHandler||A1)(a,d,f)}function c(d){const f=[];if("children"in d){const g=d.children;let m=-1;for(;++m<g.length;){const S=a.one(g[m],d);if(S){if(m&&g[m-1].type==="break"&&(!Array.isArray(S)&&S.type==="text"&&(S.value=Nc(S.value)),!Array.isArray(S)&&S.type==="element")){const b=S.children[0];b&&b.type==="text"&&(b.value=Nc(b.value))}Array.isArray(S)?f.push(...S):f.push(S)}}}return f}}function L1(e,t){e.position&&(t.position=mv(e))}function I1(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,vl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function A1(e,t){const n=t.data||{},r="value"in t&&!(Ca.call(n,"hProperties")||Ca.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function M1(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Nc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function _c(e,t){const n=T1(e,t),r=n.one(e,void 0),i=k1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function D1(e,t){return e&&"run"in e?async function(n,r){const i=_c(n,{file:r,...t});await e.run(i,r)}:function(n,r){return _c(n,{file:r,...e||t})}}function zc(e){if(e)throw e}var Vi=Object.prototype.hasOwnProperty,Mp=Object.prototype.toString,Pc=Object.defineProperty,Tc=Object.getOwnPropertyDescriptor,Lc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Mp.call(t)==="[object Array]"},Ic=function(t){if(!t||Mp.call(t)!=="[object Object]")return!1;var n=Vi.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Vi.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Vi.call(t,i)},Ac=function(t,n){Pc&&n.name==="__proto__"?Pc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Mc=function(t,n){if(n==="__proto__")if(Vi.call(t,n)){if(Tc)return Tc(t,n).value}else return;return t[n]},R1=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<c;++s)if(t=arguments[s],t!=null)for(n in t)r=Mc(a,n),i=Mc(t,n),a!==i&&(d&&i&&(Ic(i)||(l=Lc(i)))?(l?(l=!1,o=r&&Lc(r)?r:[]):o=r&&Ic(r)?r:{},Ac(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Ac(a,{name:n,newValue:i}));return a};const mo=Na(R1);function ja(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function O1(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...c){const d=e[++l];let f=-1;if(s){o(s);return}for(;++f<i.length;)(c[f]===null||c[f]===void 0)&&(c[f]=i[f]);i=c,d?F1(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function F1(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const St={basename:B1,dirname:U1,extname:H1,join:$1,sep:"/"};function B1(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');ui(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function U1(e){if(ui(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function H1(e){ui(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function $1(...e){let t=-1,n;for(;++t<e.length;)ui(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":V1(n)}function V1(e){ui(e);const t=e.codePointAt(0)===47;let n=W1(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function W1(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function ui(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const Q1={cwd:q1};function q1(){return"/"}function Ea(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function K1(e){if(typeof e=="string")e=new URL(e);else if(!Ea(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return Y1(e)}function Y1(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const go=["history","path","basename","stem","extname","dirname"];class Dp{constructor(t){let n;t?Ea(t)?n={path:t}:typeof t=="string"||X1(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":Q1.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<go.length;){const l=go[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)go.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?St.basename(this.path):void 0}set basename(t){yo(t,"basename"),vo(t,"basename"),this.path=St.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?St.dirname(this.path):void 0}set dirname(t){Dc(this.basename,"dirname"),this.path=St.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?St.extname(this.path):void 0}set extname(t){if(vo(t,"extname"),Dc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=St.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){Ea(t)&&(t=K1(t)),yo(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?St.basename(this.path,this.extname):void 0}set stem(t){yo(t,"stem"),vo(t,"stem"),this.path=St.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new Le(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function vo(e,t){if(e&&e.includes(St.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+St.sep+"`")}function yo(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Dc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function X1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const G1=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},J1={}.hasOwnProperty;class Ds extends G1{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=O1()}copy(){const t=new Ds;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(mo(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(wo("data",this.frozen),this.namespace[t]=n,this):J1.call(this.namespace,t)&&this.namespace[t]||void 0:t?(wo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Pi(t),r=this.parser||this.Parser;return xo("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),xo("process",this.parser||this.Parser),ko("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Pi(t),s=r.parse(a);r.run(s,a,function(d,f,g){if(d||!f||!g)return c(d);const m=f,S=r.stringify(m,g);t0(S)?g.value=S:g.result=S,c(d,g)});function c(d,f){d||!f?o(d):l?l(f):n(void 0,f)}}}processSync(t){let n=!1,r;return this.freeze(),xo("processSync",this.parser||this.Parser),ko("processSync",this.compiler||this.Compiler),this.process(t,i),Oc("processSync","process",n),r;function i(l,o){n=!0,zc(l),r=o}}run(t,n,r){Rc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Pi(n);i.run(t,s,c);function c(d,f,g){const m=f||t;d?a(d):o?o(m):r(void 0,m,g)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Oc("runSync","run",r),i;function l(o,a){zc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Pi(n),i=this.compiler||this.Compiler;return ko("stringify",i),Rc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(wo("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")s(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...f]=c;s(d,f)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=mo(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const f=c[d];l(f)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function s(c,d){let f=-1,g=-1;for(;++f<r.length;)if(r[f][0]===c){g=f;break}if(g===-1)r.push([c,...d]);else if(d.length>0){let[m,...S]=d;const b=r[g][1];ja(b)&&ja(m)&&(m=mo(!0,b,m)),r[g]=[c,m,...S]}}}}const Z1=new Ds().freeze();function xo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function ko(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function wo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Rc(e){if(!ja(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Oc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Pi(e){return e0(e)?e:new Dp(e)}function e0(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function t0(e){return typeof e=="string"||n0(e)}function n0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const r0="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Fc=[],Bc={allowDangerousHtml:!0},i0=/^(https?|ircs?|mailto|xmpp)$/i,l0=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function o0(e){const t=a0(e),n=s0(e);return u0(t.runSync(t.parse(n),n),e)}function a0(e){const t=e.rehypePlugins||Fc,n=e.remarkPlugins||Fc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Bc}:Bc;return Z1().use(Ux).use(n).use(D1,r).use(t)}function s0(e){const t=e.children||"",n=new Dp;return typeof t=="string"&&(n.value=t),n}function u0(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||c0;for(const d of l0)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+r0+d.id,void 0);return Ap(e,c),kv(e,{Fragment:u.Fragment,components:i,ignoreInvalidStyle:!0,jsx:u.jsx,jsxs:u.jsxs,passKeys:!0,passNode:!0});function c(d,f,g){if(d.type==="raw"&&g&&typeof f=="number")return o?g.children.splice(f,1):g.children[f]={type:"text",value:d.value},f;if(d.type==="element"){let m;for(m in fo)if(Object.hasOwn(fo,m)&&Object.hasOwn(d.properties,m)){const S=d.properties[m],b=fo[m];(b===null||b.includes(d.tagName))&&(d.properties[m]=s(String(S||""),m,d))}}if(d.type==="element"){let m=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!m&&r&&typeof f=="number"&&(m=!r(d,f,g)),m&&g&&typeof f=="number")return a&&d.children?g.children.splice(f,1,...d.children):g.children.splice(f,1),f}}}function c0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||i0.test(e.slice(0,t))?e:""}const d0=e=>{if(!e)return null;try{return JSON.parse(e).execution_stats||null}catch{return null}},f0=e=>{if(e.kind!=="status")return!1;const t=e.content.toLowerCase();return t.includes("running")||t.includes("thinking")||t.includes("executing")||t.includes("processing")},Uc=10*1024,So=200,ze={send:u.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),u.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),u.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),u.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),file:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),spinner:u.jsx("svg",{className:"spinner-icon",width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 12a9 9 0 1 1-6.219-8.56"})})},p0=e=>{switch(e){case"directive":return ze.directive;case"question":return ze.question;case"status":return ze.status;case"result":return ze.result;case"approval_request":return ze.lock;default:return ze.directive}},h0=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=B.useRef(null),[a,s]=Wt.useState(""),[c,d]=Wt.useState("directive"),[f,g]=Wt.useState(""),[m,S]=Wt.useState(!1),[b,N]=Wt.useState(new Map),[p,h]=Wt.useState(new Set),[y,k]=B.useState(new Set),[j,w]=B.useState(new Set),P=C=>{const E=(C.match(/\n/g)||[]).length+1;if(!(C.length>Uc||E>So))return{needsTruncation:!1,truncated:C,fullLength:C.length,lineCount:E};let T=C.slice(0,Uc);const F=T.split(`
`);F.length>So&&(T=F.slice(0,So).join(`
`));const x=T.lastIndexOf(`
`);return x>T.length*.8&&(T=T.slice(0,x)),{needsTruncation:!0,truncated:T,fullLength:C.length,lineCount:E}},A=C=>{k(E=>{const v=new Set(E);return v.has(C)?v.delete(C):v.add(C),v})};B.useEffect(()=>{e!=null&&e.workspace?g(e.workspace):g("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),B.useEffect(()=>{var C;(C=o.current)==null||C.scrollIntoView({behavior:"smooth"})},[t]);const U=C=>{g(C),r&&r(C)},R=()=>{a.trim()&&(n(a,c,f||void 0),s(""))},z=C=>{C.key==="Enter"&&!C.shiftKey&&(C.preventDefault(),R())},M=C=>new Date(C).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),q=C=>C.length>12?`${C.slice(0,8)}...`:C,Y=C=>{if(!C.metadata_json)return null;try{return JSON.parse(C.metadata_json).approval_id||null}catch{return null}},H=C=>{const E=b.get(C)||"";i&&(i(C,E),h(v=>new Set(v).add(C)),N(v=>{const T=new Map(v);return T.delete(C),T}))},W=C=>{const E=b.get(C)||"";if(!E.trim()){alert("Please provide a reason for rejection");return}l&&(l(C,E),h(v=>new Set(v).add(C)),N(v=>{const T=new Map(v);return T.delete(C),T}))},re=(C,E)=>{N(v=>new Map(v).set(C,E))};return e?u.jsxs("div",{className:"conversation-view",children:[u.jsxs("div",{className:"conversation-header",children:[u.jsxs("div",{className:"header-info",children:[u.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&u.jsxs("span",{className:"thread-agent-badge",children:[ze.bot,e.target_agent]})]}),u.jsxs("div",{className:"header-stats",children:[u.jsxs("span",{className:"message-count",children:[t.length," messages"]}),u.jsx("span",{className:"thread-id",title:e.id,children:q(e.id)})]})]}),u.jsxs("div",{className:"messages-container",children:[t.length===0?u.jsxs("div",{className:"empty-messages",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("p",{children:"No messages yet"}),u.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((C,E)=>{const v=C.from_type==="human",T=E===0||t[E-1].from_type!==C.from_type,F=y.has(C.id),{needsTruncation:x,truncated:te,fullLength:Se,lineCount:ee}=P(C.content),Ie=F?C.content:te,it=f0(C);return u.jsxs("div",{className:`message ${v?"human":"agent"}${it?" running-status":""}`,children:[u.jsx("div",{className:`message-avatar ${T?"visible":""}`,children:T&&(v?ze.user:ze.bot)}),u.jsxs("div",{className:"message-body",children:[T&&u.jsxs("div",{className:"message-meta",children:[u.jsx("span",{className:"sender-name",children:C.from_id}),u.jsxs("span",{className:`kind-badge${it?" running":""}`,children:[it?ze.spinner:p0(C.kind)," ",C.kind]}),u.jsx("span",{className:"message-time",children:M(C.created_at)})]}),u.jsxs("div",{className:"message-content",children:[C.kind==="result"||!v?u.jsx(o0,{components:{a:({href:G,children:be})=>{let Fe=G;return G&&G.startsWith("/")&&!G.startsWith("//")&&(Fe=`file://${G}`),u.jsx("a",{href:Fe,target:"_blank",rel:"noopener noreferrer",children:be})},code:({className:G,children:be,...Fe})=>!G?u.jsx("code",{className:"inline-code",...Fe,children:be}):u.jsx("code",{className:G,...Fe,children:be})},children:Ie}):Ie,x&&u.jsx("div",{className:"truncation-notice",children:u.jsx("button",{className:"expand-btn",onClick:()=>A(C.id),children:F?u.jsx(u.Fragment,{children:"Show less"}):u.jsxs(u.Fragment,{children:["Show more (",Math.round(Se/1024),"KB, ",ee," lines)"]})})}),C.kind==="approval_request"&&(()=>{const G=Y(C),be=G&&p.has(G);return G?u.jsx("div",{className:"inline-approval",children:be?u.jsxs("div",{className:"approval-handled",children:[ze.check,u.jsx("span",{children:"Action taken"})]}):u.jsxs(u.Fragment,{children:[u.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:b.get(G)||"",onChange:Fe=>re(G,Fe.target.value)}),u.jsxs("div",{className:"approval-actions",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>W(G),title:"Reject",children:[ze.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>H(G),title:"Approve",children:[ze.check,"Approve"]})]})]})}):null})(),C.kind==="result"&&(()=>{const G=d0(C.metadata_json);if(!G||!G.files_created||G.files_created.length===0)return null;const be=j.has(C.id),Fe=()=>{w(Ut=>{const Ht=new Set(Ut);return Ht.has(C.id)?Ht.delete(C.id):Ht.add(C.id),Ht})};return u.jsxs("div",{className:"files-created-section",children:[u.jsxs("button",{className:`files-toggle-btn ${be?"expanded":""}`,onClick:Fe,children:[ze.file,u.jsxs("span",{children:["Files Created (",G.files_created.length,")"]}),G.workspace&&u.jsxs("span",{className:"workspace-badge",title:G.workspace,children:[ze.folder,G.workspace.split("/").pop()]}),u.jsx("span",{className:"toggle-chevron",children:be?"▼":"▶"})]}),be&&u.jsx("ul",{className:"files-list",children:G.files_created.map((Ut,Ht)=>u.jsx("li",{className:"file-item",children:u.jsx("a",{href:`file://${G.workspace?G.workspace+"/":""}${Ut}`,target:"_blank",rel:"noopener noreferrer",title:Ut,children:Ut})},Ht))})]})})()]}),u.jsxs("div",{className:"message-footer",children:[u.jsxs("span",{className:"message-seq",children:["#",C.message_seq]}),C.delivery_state!=="acked"&&u.jsx("span",{className:`delivery-status ${C.delivery_state}`,children:C.delivery_state==="pending"?"sending...":"delivered"})]})]})]},C.id)}),u.jsx("div",{ref:o})]}),u.jsxs("div",{className:"input-area",children:[m&&u.jsxs("div",{className:"workspace-input-row",children:[u.jsx("input",{type:"text",value:f,onChange:C=>U(C.target.value),onBlur:()=>{r&&r(f)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),u.jsx("button",{onClick:async()=>{try{const E=await(await fetch("/api/select-folder")).json();!E.cancelled&&E.path&&U(E.path)}catch(C){console.error("Failed to open folder picker:",C)}},className:"workspace-browse",title:"Browse for folder",children:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),u.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),u.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),f&&u.jsx("button",{onClick:()=>{U(""),S(!1)},className:"workspace-clear",children:"Clear"})]}),u.jsxs("div",{className:"input-wrapper",children:[u.jsx("button",{onClick:()=>S(!m),className:`workspace-toggle ${f?"has-workspace":""}`,title:f||"Set working directory",children:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),u.jsxs("select",{value:c,onChange:C=>d(C.target.value),className:"kind-selector",children:[u.jsx("option",{value:"directive",children:"Directive"}),u.jsx("option",{value:"question",children:"Question"})]}),u.jsx("textarea",{value:a,onChange:C=>s(C.target.value),onKeyPress:z,placeholder:f?`Message (workspace: ${f.split("/").pop()})`:"Type a message...",rows:1}),u.jsx("button",{onClick:R,className:"send-btn",disabled:!a.trim(),children:ze.send})]}),u.jsxs("div",{className:"input-hint",children:["Press ",u.jsx("kbd",{children:"Enter"})," to send, ",u.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),u.jsx("style",{children:`
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
      `})]}):null},m0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,reconnectInterval:l=5e3})=>{const o=B.useRef(null),[a,s]=B.useState(!1),[c,d]=B.useState(null),f=B.useRef(null),g=B.useRef(new Map),m=B.useCallback(()=>{try{const k=`${e}?instance_id=${t}`;o.current=new WebSocket(k),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),d(null),g.current.forEach((j,w)=>{N(w,j)})},o.current.onmessage=j=>{try{const w=JSON.parse(j.data);S(w)}catch(w){console.error("Failed to parse WebSocket message:",w)}},o.current.onerror=j=>{console.error("WebSocket error:",j),d("Connection error")},o.current.onclose=()=>{console.log("WebSocket disconnected"),s(!1),f.current=setTimeout(()=>{console.log("Attempting to reconnect..."),m()},l)}}catch(k){console.error("Failed to connect to WebSocket:",k),d("Failed to connect")}},[e,t,l]),S=B.useCallback(k=>{switch(k.type){case"message":n&&k.data&&n(k.data);break;case"batch":if(r&&k.data){const j=k.data;r(j),n&&j.messages.forEach(w=>n(w))}break;case"error":i&&k.data&&i(k.data),console.error("WebSocket error event:",k.data);break;case"pong":break;default:console.log("Unknown event type:",k.type)}},[n,r,i]),b=B.useCallback(k=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(k)):console.warn("WebSocket not connected, cannot send event")},[]),N=B.useCallback((k,j=0)=>{g.current.set(k,j);const w={type:"subscribe",timestamp:Date.now(),data:{thread_id:k,from_seq:j}};b(w)},[b]),p=B.useCallback((k,j)=>{const w=g.current.get(k)||0;j>w&&g.current.set(k,j);const P={type:"ack",timestamp:Date.now(),data:{thread_id:k,ack_seq:j}};b(P)},[b]),h=B.useCallback(()=>{const k={type:"ping",timestamp:Date.now()};b(k)},[b]),y=B.useCallback(k=>{g.current.delete(k)},[]);return B.useEffect(()=>(m(),()=>{f.current&&clearTimeout(f.current),o.current&&o.current.close()}),[m]),B.useEffect(()=>{if(!a)return;const k=setInterval(()=>{h()},3e4);return()=>clearInterval(k)},[a,h]),{isConnected:a,connectionError:c,subscribe:N,unsubscribe:y,acknowledge:p,ping:h}},g0=({connected:e})=>u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?u.jsxs(u.Fragment,{children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):u.jsxs(u.Fragment,{children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),u.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),v0=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=B.useState([]),[o,a]=B.useState(null),[s,c]=B.useState(new Map),[d,f]=B.useState(new Map),[g,m]=B.useState([]),[S,b]=B.useState(!1),[N,p]=B.useState(""),{isConnected:h,subscribe:y,acknowledge:k}=m0({url:e,instanceId:t,onMessage:j,onBatch:w});function j(E){const v={id:E.id,thread_id:E.thread_id,message_seq:E.message_seq,created_at:E.created_at,from_type:E.from_type,from_id:E.from_id,to_type:E.to_type,to_id:E.to_id,kind:E.kind,subject:E.subject,content:E.content,metadata_json:E.metadata_json,delivery_state:"visible",business_state:"open"};c(T=>{const F=T.get(v.thread_id)||[];return F.find(x=>x.id===v.id)?T:new Map(T).set(v.thread_id,[...F,v].sort((x,te)=>x.message_seq-te.message_seq))}),v.thread_id!==o&&f(T=>{const F=T.get(v.thread_id)||0;return new Map(T).set(v.thread_id,F+1)}),k(v.thread_id,v.message_seq)}function w(E){E.messages.forEach(v=>{j(v)})}const P=B.useCallback(E=>{if(a(E),f(v=>{const T=new Map(v);return T.delete(E),T}),h){const v=s.get(E)||[],T=v.length>0?Math.max(...v.map(F=>F.message_seq)):0;y(E,T)}},[h,y,s]),A=B.useCallback(async(E,v,T)=>{if(!o)return;const F=T?JSON.stringify({workspace:T}):void 0;try{const x=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:v,content:E,metadata_json:F})});if(!x.ok){console.error("Failed to send message:",await x.text());return}const te=await x.json();c(Se=>{const ee=Se.get(o)||[];return ee.find(Ie=>Ie.id===te.id)?Se:new Map(Se).set(o,[...ee,te])})}catch(x){console.error("Error sending message:",x)}},[o,t]);B.useEffect(()=>{(async()=>{try{const v=await fetch("/api/threads");if(!v.ok){console.error("Failed to fetch threads:",await v.text());return}const T=await v.json();l(T),T.length>0&&!o&&a(T[0].id)}catch(v){console.error("Error fetching threads:",v)}})()},[]),B.useEffect(()=>{n&&i.length>0&&(i.some(v=>v.id===n)&&(a(n),f(v=>{const T=new Map(v);return T.delete(n),T})),r&&r())},[n,i,r]);const U=B.useCallback(async E=>{try{const v=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:E,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!v.ok){console.error("Failed to create thread:",await v.text());return}const T=await v.json();l(F=>[T,...F]),a(T.id)}catch(v){console.error("Error creating thread:",v)}},[t]),R=B.useCallback(async()=>{try{const E=await fetch("/api/agents");if(!E.ok){console.error("Failed to fetch agents:",await E.text());return}const v=await E.json();m(v.running||[])}catch(E){console.error("Error fetching agents:",E)}},[]);B.useEffect(()=>{R();const E=setInterval(R,5e3);return()=>clearInterval(E)},[R]);const z=B.useCallback(async()=>{if(N.trim())try{const E=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:N.trim()})});if(!E.ok){const T=await E.text();console.error("Failed to launch agent:",T),alert(`Failed to launch agent: ${T}`);return}const v=await E.json();m(T=>[...T,v]),p(""),b(!1)}catch(E){console.error("Error launching agent:",E)}},[N]),M=B.useCallback(async E=>{try{const v=await fetch(`/api/agents/${E}`,{method:"DELETE"});if(!v.ok){console.error("Failed to stop agent:",await v.text());return}m(T=>T.filter(F=>F.instance_id!==E))}catch(v){console.error("Error stopping agent:",v)}},[]),q=B.useCallback(async E=>{if(o)try{const v=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:E})});if(!v.ok){console.error("Failed to update workspace:",await v.text());return}const T=await v.json();l(F=>F.map(x=>x.id===o?T:x))}catch(v){console.error("Error updating workspace:",v)}},[o]),Y=B.useCallback(async E=>{try{const v=await fetch(`/api/threads/${E}`,{method:"DELETE"});if(!v.ok){console.error("Failed to delete thread:",await v.text());return}l(T=>T.filter(F=>F.id!==E)),c(T=>{const F=new Map(T);return F.delete(E),F}),f(T=>{const F=new Map(T);return F.delete(E),F}),o===E&&a(null)}catch(v){console.error("Error deleting thread:",v)}},[o]),H=B.useCallback(async(E,v)=>{try{const T=await fetch(`/api/threads/${E}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:v})});if(!T.ok){console.error("Failed to rename thread:",await T.text());return}const F=await T.json();l(x=>x.map(te=>te.id===E?F:te))}catch(T){console.error("Error renaming thread:",T)}},[]),W=B.useCallback(async(E,v)=>{try{const T=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:v})});if(!T.ok){const F=await T.text();console.error("Failed to approve request:",F),alert(`Failed to approve: ${F}`);return}console.log("Approval approved successfully")}catch(T){console.error("Error approving request:",T)}},[]),re=B.useCallback(async(E,v)=>{try{const T=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:v})});if(!T.ok){const F=await T.text();console.error("Failed to reject request:",F),alert(`Failed to reject: ${F}`);return}console.log("Approval rejected successfully")}catch(T){console.error("Error rejecting request:",T)}},[]),C=o?s.get(o)||[]:[];return u.jsxs("div",{className:"message-center",children:[u.jsxs("div",{className:"status-bar",children:[u.jsxs("div",{className:`status-indicator ${h?"connected":"disconnected"}`,children:[u.jsx(g0,{connected:h}),u.jsx("span",{children:h?"Connected":"Disconnected"})]}),u.jsxs("div",{className:"status-meta",children:[u.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),u.jsxs("span",{className:"agent-count",children:[g.length," agents"]}),u.jsx("button",{className:"launch-agent-btn",onClick:()=>b(!0),children:"+ Agent"})]})]}),g.length>0&&u.jsx("div",{className:"agents-bar",children:g.map(E=>u.jsxs("div",{className:"agent-chip",children:[u.jsx("span",{className:"agent-pulse"}),u.jsx("span",{className:"agent-name",children:E.instance_id}),u.jsxs("span",{className:"agent-pid",children:["PID ",E.pid]}),u.jsx("button",{className:"agent-stop-btn",onClick:()=>M(E.instance_id),title:"Stop agent",children:"×"})]},E.instance_id))}),S&&u.jsx("div",{className:"modal-overlay",onClick:()=>b(!1),children:u.jsxs("div",{className:"modal-content",onClick:E=>E.stopPropagation(),children:[u.jsx("h3",{children:"Launch New Agent"}),u.jsx("input",{type:"text",value:N,onChange:E=>p(E.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:E=>{E.key==="Enter"&&z(),E.key==="Escape"&&b(!1)}}),u.jsxs("div",{className:"modal-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>b(!1),children:"Cancel"}),u.jsx("button",{className:"launch-btn",onClick:z,children:"Launch"})]})]})}),u.jsxs("div",{className:"center-layout",children:[u.jsx("aside",{className:"threads-panel",children:u.jsx(bg,{threads:i,selectedThreadId:o,onSelectThread:P,onCreateThread:U,onDeleteThread:Y,onRenameThread:H,unreadCounts:d})}),u.jsx("main",{className:"conversation-panel",children:o?u.jsx(h0,{thread:i.find(E=>E.id===o),messages:C,onSendMessage:A,onWorkspaceChange:q,onApproveRequest:W,onRejectRequest:re}):u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("h3",{children:"Select a conversation"}),u.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),u.jsx("style",{children:`
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
      `})]})},Ae={check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),dollar:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:u.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),u.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),u.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},y0=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=B.useState(!0),[a,s]=B.useState(null),[c,d]=B.useState(new Map),f=p=>{try{return JSON.parse(p)}catch{return null}},g=p=>new Date(p).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),m=p=>{const h=c.get(p)||"";n(p,h),d(new Map(c.set(p,"")))},S=p=>{const h=c.get(p)||"";if(!h.trim()){alert("Please provide a reason for rejection");return}r(p,h),d(new Map(c.set(p,"")))},b=(p,h)=>{d(new Map(c.set(p,h)))},N=e.filter(p=>p.status==="pending");return u.jsxs("div",{className:"approval-queue",children:[u.jsx("div",{className:"queue-header",children:u.jsxs("div",{className:"header-title",children:[u.jsx("h2",{children:"Approval Queue"}),u.jsxs("span",{className:"pending-count",children:[N.length," pending"]})]})}),u.jsxs("div",{className:"approvals-container",children:[N.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Ae.sparkles}),u.jsx("h3",{children:"All caught up!"}),u.jsx("p",{children:"No pending approvals to review"})]}):u.jsx("div",{className:"approvals-list",children:N.map(p=>{const h=f(p.effect_delta_json),y=a===p.id;return u.jsxs("div",{className:`approval-card impact-${p.impact}`,children:[u.jsxs("div",{className:"card-header",onClick:()=>s(y?null:p.id),children:[u.jsxs("div",{className:"header-left",children:[u.jsx("div",{className:`impact-indicator ${p.impact}`}),u.jsxs("div",{className:"proposal-info",children:[u.jsx("span",{className:"proposal-text",children:p.proposal}),u.jsxs("div",{className:"proposal-meta",children:[p.thread_title&&u.jsxs("span",{className:"meta-item thread-link",onClick:k=>{k.stopPropagation(),i==null||i(p.thread_id)},title:"Go to thread",children:[Ae.message,p.thread_title]}),u.jsxs("span",{className:"meta-item",children:[Ae.bot,p.instance_id]}),u.jsxs("span",{className:"meta-item",children:[Ae.clock,g(p.created_at)]})]})]})]}),u.jsxs("div",{className:"header-right",children:[u.jsxs("span",{className:"cost-badge",children:[Ae.dollar,"$",p.estimated_cost.toFixed(2)]}),u.jsx("span",{className:`impact-badge ${p.impact}`,children:p.impact}),u.jsx("button",{className:"expand-btn",children:y?Ae.chevronUp:Ae.chevronDown})]})]}),y&&u.jsxs("div",{className:"card-details",children:[h&&u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Effect Details"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Capability"}),u.jsx("span",{className:"detail-value code",children:h.cap_type})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Budget Delta"}),u.jsxs("span",{className:"detail-value",children:["$",h.budget_delta.toFixed(2)]})]}),h.paths.length>0&&u.jsxs("div",{className:"detail-item full-width",children:[u.jsx("span",{className:"detail-label",children:"Paths"}),u.jsx("div",{className:"paths-list",children:h.paths.map((k,j)=>u.jsxs("span",{className:"path-tag",children:[Ae.folder,k]},j))})]})]})]}),u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Request Info"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Thread"}),u.jsx("span",{className:"detail-value code",children:p.thread_id})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Impact Level"}),u.jsx("span",{className:`detail-value impact-text ${p.impact}`,children:p.impact.toUpperCase()})]})]})]}),u.jsxs("div",{className:"review-section",children:[u.jsx("h4",{children:"Review Notes"}),u.jsx("textarea",{value:c.get(p.id)||"",onChange:k=>b(p.id,k.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),u.jsxs("div",{className:"action-buttons",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>S(p.id),children:[Ae.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>m(p.id),children:[Ae.check,"Approve"]})]})]})]})]},p.id)})}),t.length>0&&u.jsxs("div",{className:"history-section",children:[u.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[u.jsxs("h3",{children:[l?Ae.chevronDown:Ae.chevronUp,"Review History"]}),u.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&u.jsx("div",{className:"history-list",children:t.map(p=>{const h=a===`history-${p.id}`;return u.jsxs("div",{className:`history-card ${p.status}`,onClick:()=>s(h?null:`history-${p.id}`),children:[u.jsxs("div",{className:"history-card-header",children:[u.jsxs("div",{className:"history-status",children:[u.jsx("span",{className:`status-icon ${p.status}`,children:p.status==="approved"?Ae.check:Ae.x}),u.jsxs("div",{className:"history-info",children:[u.jsx("span",{className:"history-proposal",children:p.proposal}),p.thread_title&&u.jsxs("span",{className:"history-thread",onClick:y=>{y.stopPropagation(),i==null||i(p.thread_id)},title:"Go to thread",children:[Ae.message,p.thread_title]})]})]}),u.jsxs("div",{className:"history-meta",children:[u.jsx("span",{className:"history-agent",children:p.instance_id}),u.jsx("span",{className:`history-badge ${p.status}`,children:p.status}),u.jsx("span",{className:"history-time",children:p.reviewed_at?g(p.reviewed_at):g(p.created_at)})]})]}),h&&u.jsxs("div",{className:"history-details",children:[u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Reviewed by"}),u.jsx("span",{className:"detail-value",children:p.reviewed_by||"Unknown"})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Cost"}),u.jsxs("span",{className:"detail-value",children:["$",p.estimated_cost.toFixed(2)]})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Impact"}),u.jsx("span",{className:`detail-value impact-text ${p.impact}`,children:p.impact.toUpperCase()})]}),p.review_notes&&u.jsxs("div",{className:"detail-row full-width",children:[u.jsx("span",{className:"detail-label",children:"Notes"}),u.jsx("span",{className:"detail-value notes",children:p.review_notes})]})]})]},`history-${p.id}`)})})]})]}),u.jsx("style",{children:`
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
      `})]})},x0=u.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M12 6v12M6 12h12"}),u.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]}),k0=()=>{const[e,t]=B.useState({type:"overview"}),[n,r]=B.useState(null),[i,l]=B.useState([]),[o,a]=B.useState([]),c=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;B.useEffect(()=>{const N=async()=>{try{const h=await fetch("/api/hierarchy");if(h.ok){const y=await h.json();r(y)}}catch(h){console.error("Error fetching hierarchy:",h)}};N();const p=setInterval(N,5e3);return()=>clearInterval(p)},[]),B.useEffect(()=>{const N=async()=>{try{const h=await fetch("/api/approvals?status=pending");if(h.ok){const w=await h.json();l(w)}const[y,k]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),j=[];if(y.ok){const w=await y.json();j.push(...w)}if(k.ok){const w=await k.json();j.push(...w)}j.sort((w,P)=>{const A=w.reviewed_at?new Date(w.reviewed_at).getTime():0;return(P.reviewed_at?new Date(P.reviewed_at).getTime():0)-A}),a(j)}catch(h){console.error("Error fetching approvals:",h)}};N();const p=setInterval(N,5e3);return()=>clearInterval(p)},[]);const d=async(N,p)=>{try{const h=await fetch(`/api/approvals/${N}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:p})});if(!h.ok){console.error("Failed to approve:",await h.text());return}const y=i.find(k=>k.id===N);if(y){const k={...y,status:"approved",reviewed_by:"user",review_notes:p,reviewed_at:Date.now()};a(j=>[k,...j])}l(k=>k.filter(j=>j.id!==N))}catch(h){console.error("Error approving:",h)}},f=async(N,p)=>{try{const h=await fetch(`/api/approvals/${N}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:p})});if(!h.ok){console.error("Failed to reject:",await h.text());return}const y=i.find(k=>k.id===N);if(y){const k={...y,status:"rejected",reviewed_by:"user",review_notes:p,reviewed_at:Date.now()};a(j=>[k,...j])}l(k=>k.filter(j=>j.id!==N))}catch(h){console.error("Error rejecting:",h)}},g=()=>{var p,h;const N=[{label:"All Agents",onClick:()=>t({type:"overview"})}];if(e.type==="agent"&&e.agentId&&N.push({label:e.agentId}),e.type==="thread"&&e.threadId){e.agentId&&N.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})});const y=(p=n==null?void 0:n.root.children)==null?void 0:p.find(j=>j.id===e.agentId),k=(h=y==null?void 0:y.children)==null?void 0:h.find(j=>j.id===e.threadId);N.push({label:(k==null?void 0:k.label)||"Thread"})}return N},m=N=>{var h;const p=(h=n==null?void 0:n.root.children)==null?void 0:h.find(y=>{var k;return(k=y.children)==null?void 0:k.some(j=>j.id===N)});t({type:"thread",agentId:p==null?void 0:p.id,threadId:N})},S=()=>{var N,p,h;if(e.type==="overview"&&n)return u.jsx(wg,{aggregate:n.aggregate,agents:n.root.children||[],onSelectAgent:y=>t({type:"agent",agentId:y})});if(e.type==="agent"&&e.agentId){const y=(N=n==null?void 0:n.root.children)==null?void 0:N.find(j=>j.id===e.agentId),k=i.filter(j=>{var w;return(w=y==null?void 0:y.children)==null?void 0:w.some(P=>P.id===j.thread_id)});return u.jsxs("div",{className:"agent-view",children:[u.jsxs("div",{className:"agent-view-header",children:[u.jsx("h2",{children:e.agentId}),u.jsxs("span",{className:"agent-thread-count",children:[((p=y==null?void 0:y.children)==null?void 0:p.length)||0," threads"]})]}),u.jsxs("div",{className:"agent-view-content",children:[u.jsxs("div",{className:"agent-threads",children:[u.jsx("h3",{children:"Threads"}),(h=y==null?void 0:y.children)==null?void 0:h.map(j=>u.jsxs("div",{className:"thread-card",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:j.id}),children:[u.jsx("span",{className:"thread-title",children:j.label}),j.badges&&j.badges.length>0&&u.jsx("span",{className:"thread-badges",children:j.badges.map((w,P)=>u.jsx("span",{className:`badge badge-${w.type}`,children:w.count},P))})]},j.id)),(!(y!=null&&y.children)||y.children.length===0)&&u.jsx("div",{className:"no-threads",children:"No threads yet"})]}),k.length>0&&u.jsxs("div",{className:"agent-approvals",children:[u.jsx("h3",{children:"Pending Approvals"}),u.jsx(y0,{approvals:k,history:[],onApprove:d,onReject:f,onNavigateToThread:m})]})]})]})}return e.type==="thread"&&e.threadId?u.jsx(v0,{websocketUrl:c,instanceId:e.agentId||"default",initialThreadId:e.threadId,onThreadNavigated:()=>{}}):u.jsx("div",{className:"empty-state",children:u.jsx("p",{children:"Select an agent or thread from the sidebar"})})},b=(i==null?void 0:i.filter(N=>N.status==="pending").length)||0;return u.jsxs("div",{className:"app",children:[u.jsxs("header",{className:"app-header",children:[u.jsxs("div",{className:"header-brand",children:[u.jsx("div",{className:"brand-logo",children:x0}),u.jsxs("div",{className:"brand-text",children:[u.jsx("h1",{children:"AILANG"}),u.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),u.jsxs("div",{className:"header-meta",children:[b>0&&u.jsxs("span",{className:"pending-badge",title:`${b} pending approvals`,children:[b," pending"]}),u.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),u.jsxs("div",{className:"app-body",children:[u.jsx("aside",{className:"app-sidebar",children:u.jsx(vg,{selection:e,onSelect:t})}),u.jsxs("main",{className:"app-main",children:[e.type!=="overview"&&u.jsx(Sg,{items:g()}),u.jsx("div",{className:"main-content",children:S()})]})]}),u.jsx("style",{children:`
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
      `})]})};bo.createRoot(document.getElementById("root")).render(u.jsx(Wt.StrictMode,{children:u.jsx(k0,{})}));
