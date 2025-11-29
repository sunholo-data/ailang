(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Hi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function ja(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Oc={exports:{}},gl={},Fc={exports:{}},Y={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ei=Symbol.for("react.element"),$p=Symbol.for("react.portal"),Hp=Symbol.for("react.fragment"),Vp=Symbol.for("react.strict_mode"),Wp=Symbol.for("react.profiler"),Qp=Symbol.for("react.provider"),Kp=Symbol.for("react.context"),qp=Symbol.for("react.forward_ref"),Yp=Symbol.for("react.suspense"),Xp=Symbol.for("react.memo"),Gp=Symbol.for("react.lazy"),Fs=Symbol.iterator;function Jp(e){return e===null||typeof e!="object"?null:(e=Fs&&e[Fs]||e["@@iterator"],typeof e=="function"?e:null)}var Bc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Uc=Object.assign,$c={};function rr(e,t,n){this.props=e,this.context=t,this.refs=$c,this.updater=n||Bc}rr.prototype.isReactComponent={};rr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};rr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function Hc(){}Hc.prototype=rr.prototype;function Ca(e,t,n){this.props=e,this.context=t,this.refs=$c,this.updater=n||Bc}var Ea=Ca.prototype=new Hc;Ea.constructor=Ca;Uc(Ea,rr.prototype);Ea.isPureReactComponent=!0;var Bs=Array.isArray,Vc=Object.prototype.hasOwnProperty,Na={current:null},Wc={key:!0,ref:!0,__self:!0,__source:!0};function Qc(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)Vc.call(t,r)&&!Wc.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),c=0;c<a;c++)s[c]=arguments[c+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:ei,type:e,key:l,ref:o,props:i,_owner:Na.current}}function Zp(e,t){return{$$typeof:ei,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function _a(e){return typeof e=="object"&&e!==null&&e.$$typeof===ei}function eh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Us=/\/+/g;function Al(e,t){return typeof e=="object"&&e!==null&&e.key!=null?eh(""+e.key):t.toString(36)}function zi(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case ei:case $p:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Al(o,0):r,Bs(i)?(n="",e!=null&&(n=e.replace(Us,"$&/")+"/"),zi(i,t,n,"",function(c){return c})):i!=null&&(_a(i)&&(i=Zp(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Us,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Bs(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Al(l,a);o+=zi(l,t,n,s,i)}else if(s=Jp(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Al(l,a++),o+=zi(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function si(e,t,n){if(e==null)return e;var r=[],i=0;return zi(e,r,"","",function(l){return t.call(n,l,i++)}),r}function th(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Pe={current:null},Ti={transition:null},nh={ReactCurrentDispatcher:Pe,ReactCurrentBatchConfig:Ti,ReactCurrentOwner:Na};function Kc(){throw Error("act(...) is not supported in production builds of React.")}Y.Children={map:si,forEach:function(e,t,n){si(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return si(e,function(){t++}),t},toArray:function(e){return si(e,function(t){return t})||[]},only:function(e){if(!_a(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};Y.Component=rr;Y.Fragment=Hp;Y.Profiler=Wp;Y.PureComponent=Ca;Y.StrictMode=Vp;Y.Suspense=Yp;Y.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=nh;Y.act=Kc;Y.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Uc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Na.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)Vc.call(t,s)&&!Wc.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var c=0;c<s;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:ei,type:e.type,key:i,ref:l,props:r,_owner:o}};Y.createContext=function(e){return e={$$typeof:Kp,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:Qp,_context:e},e.Consumer=e};Y.createElement=Qc;Y.createFactory=function(e){var t=Qc.bind(null,e);return t.type=e,t};Y.createRef=function(){return{current:null}};Y.forwardRef=function(e){return{$$typeof:qp,render:e}};Y.isValidElement=_a;Y.lazy=function(e){return{$$typeof:Gp,_payload:{_status:-1,_result:e},_init:th}};Y.memo=function(e,t){return{$$typeof:Xp,type:e,compare:t===void 0?null:t}};Y.startTransition=function(e){var t=Ti.transition;Ti.transition={};try{e()}finally{Ti.transition=t}};Y.unstable_act=Kc;Y.useCallback=function(e,t){return Pe.current.useCallback(e,t)};Y.useContext=function(e){return Pe.current.useContext(e)};Y.useDebugValue=function(){};Y.useDeferredValue=function(e){return Pe.current.useDeferredValue(e)};Y.useEffect=function(e,t){return Pe.current.useEffect(e,t)};Y.useId=function(){return Pe.current.useId()};Y.useImperativeHandle=function(e,t,n){return Pe.current.useImperativeHandle(e,t,n)};Y.useInsertionEffect=function(e,t){return Pe.current.useInsertionEffect(e,t)};Y.useLayoutEffect=function(e,t){return Pe.current.useLayoutEffect(e,t)};Y.useMemo=function(e,t){return Pe.current.useMemo(e,t)};Y.useReducer=function(e,t,n){return Pe.current.useReducer(e,t,n)};Y.useRef=function(e){return Pe.current.useRef(e)};Y.useState=function(e){return Pe.current.useState(e)};Y.useSyncExternalStore=function(e,t,n){return Pe.current.useSyncExternalStore(e,t,n)};Y.useTransition=function(){return Pe.current.useTransition()};Y.version="18.3.1";Fc.exports=Y;var $=Fc.exports;const dn=ja($);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var rh=$,ih=Symbol.for("react.element"),lh=Symbol.for("react.fragment"),oh=Object.prototype.hasOwnProperty,ah=rh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,sh={key:!0,ref:!0,__self:!0,__source:!0};function qc(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)oh.call(t,r)&&!sh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:ih,type:e,key:l,ref:o,props:i,_owner:ah.current}}gl.Fragment=lh;gl.jsx=qc;gl.jsxs=qc;Oc.exports=gl;var u=Oc.exports,ko={},Yc={exports:{}},Xe={},Xc={exports:{}},Gc={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(P,B){var v=P.length;P.push(B);e:for(;0<v;){var q=v-1>>>1,J=P[q];if(0<i(J,B))P[q]=B,P[v]=J,v=q;else break e}}function n(P){return P.length===0?null:P[0]}function r(P){if(P.length===0)return null;var B=P[0],v=P.pop();if(v!==B){P[0]=v;e:for(var q=0,J=P.length,x=J>>>1;q<x;){var ve=2*(q+1)-1,lt=P[ve],le=ve+1,ht=P[le];if(0>i(lt,v))le<J&&0>i(ht,lt)?(P[q]=ht,P[le]=v,q=le):(P[q]=lt,P[ve]=v,q=ve);else if(le<J&&0>i(ht,v))P[q]=ht,P[le]=v,q=le;else break e}}return B}function i(P,B){var v=P.sortIndex-B.sortIndex;return v!==0?v:P.id-B.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],c=[],d=1,f=null,h=3,p=!1,k=!1,S=!1,C=typeof setTimeout=="function"?setTimeout:null,g=typeof clearTimeout=="function"?clearTimeout:null,m=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(P){for(var B=n(c);B!==null;){if(B.callback===null)r(c);else if(B.startTime<=P)r(c),B.sortIndex=B.expirationTime,t(s,B);else break;B=n(c)}}function j(P){if(S=!1,y(P),!k)if(n(s)!==null)k=!0,D(z);else{var B=n(c);B!==null&&F(j,B.startTime-P)}}function z(P,B){k=!1,S&&(S=!1,g(b),b=-1),p=!0;var v=h;try{for(y(B),f=n(s);f!==null&&(!(f.expirationTime>B)||P&&!_());){var q=f.callback;if(typeof q=="function"){f.callback=null,h=f.priorityLevel;var J=q(f.expirationTime<=B);B=e.unstable_now(),typeof J=="function"?f.callback=J:f===n(s)&&r(s),y(B)}else r(s);f=n(s)}if(f!==null)var x=!0;else{var ve=n(c);ve!==null&&F(j,ve.startTime-B),x=!1}return x}finally{f=null,h=v,p=!1}}var w=!1,E=null,b=-1,N=5,A=-1;function _(){return!(e.unstable_now()-A<N)}function I(){if(E!==null){var P=e.unstable_now();A=P;var B=!0;try{B=E(!0,P)}finally{B?H():(w=!1,E=null)}}else w=!1}var H;if(typeof m=="function")H=function(){m(I)};else if(typeof MessageChannel<"u"){var K=new MessageChannel,L=K.port2;K.port1.onmessage=I,H=function(){L.postMessage(null)}}else H=function(){C(I,0)};function D(P){E=P,w||(w=!0,H())}function F(P,B){b=C(function(){P(e.unstable_now())},B)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(P){P.callback=null},e.unstable_continueExecution=function(){k||p||(k=!0,D(z))},e.unstable_forceFrameRate=function(P){0>P||125<P?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):N=0<P?Math.floor(1e3/P):5},e.unstable_getCurrentPriorityLevel=function(){return h},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(P){switch(h){case 1:case 2:case 3:var B=3;break;default:B=h}var v=h;h=B;try{return P()}finally{h=v}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(P,B){switch(P){case 1:case 2:case 3:case 4:case 5:break;default:P=3}var v=h;h=P;try{return B()}finally{h=v}},e.unstable_scheduleCallback=function(P,B,v){var q=e.unstable_now();switch(typeof v=="object"&&v!==null?(v=v.delay,v=typeof v=="number"&&0<v?q+v:q):v=q,P){case 1:var J=-1;break;case 2:J=250;break;case 5:J=1073741823;break;case 4:J=1e4;break;default:J=5e3}return J=v+J,P={id:d++,callback:B,priorityLevel:P,startTime:v,expirationTime:J,sortIndex:-1},v>q?(P.sortIndex=v,t(c,P),n(s)===null&&P===n(c)&&(S?(g(b),b=-1):S=!0,F(j,v-q))):(P.sortIndex=J,t(s,P),k||p||(k=!0,D(z))),P},e.unstable_shouldYield=_,e.unstable_wrapCallback=function(P){var B=h;return function(){var v=h;h=B;try{return P.apply(this,arguments)}finally{h=v}}}})(Gc);Xc.exports=Gc;var uh=Xc.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ch=$,Ye=uh;function M(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var Jc=new Set,Ar={};function Sn(e,t){Xn(e,t),Xn(e+"Capture",t)}function Xn(e,t){for(Ar[e]=t,e=0;e<t.length;e++)Jc.add(t[e])}var Pt=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),wo=Object.prototype.hasOwnProperty,dh=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,$s={},Hs={};function fh(e){return wo.call(Hs,e)?!0:wo.call($s,e)?!1:dh.test(e)?Hs[e]=!0:($s[e]=!0,!1)}function ph(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function hh(e,t,n,r){if(t===null||typeof t>"u"||ph(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Ie(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var je={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){je[e]=new Ie(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];je[t]=new Ie(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){je[e]=new Ie(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){je[e]=new Ie(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){je[e]=new Ie(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){je[e]=new Ie(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){je[e]=new Ie(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){je[e]=new Ie(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){je[e]=new Ie(e,5,!1,e.toLowerCase(),null,!1,!1)});var za=/[\-:]([a-z])/g;function Ta(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(za,Ta);je[t]=new Ie(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(za,Ta);je[t]=new Ie(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(za,Ta);je[t]=new Ie(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){je[e]=new Ie(e,1,!1,e.toLowerCase(),null,!1,!1)});je.xlinkHref=new Ie("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){je[e]=new Ie(e,1,!1,e.toLowerCase(),null,!0,!0)});function La(e,t,n,r){var i=je.hasOwnProperty(t)?je[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(hh(t,n,i,r)&&(n=null),r||i===null?fh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var At=ch.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,ui=Symbol.for("react.element"),Tn=Symbol.for("react.portal"),Ln=Symbol.for("react.fragment"),Pa=Symbol.for("react.strict_mode"),So=Symbol.for("react.profiler"),Zc=Symbol.for("react.provider"),ed=Symbol.for("react.context"),Ia=Symbol.for("react.forward_ref"),bo=Symbol.for("react.suspense"),jo=Symbol.for("react.suspense_list"),Da=Symbol.for("react.memo"),Bt=Symbol.for("react.lazy"),td=Symbol.for("react.offscreen"),Vs=Symbol.iterator;function cr(e){return e===null||typeof e!="object"?null:(e=Vs&&e[Vs]||e["@@iterator"],typeof e=="function"?e:null)}var de=Object.assign,Rl;function kr(e){if(Rl===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Rl=t&&t[1]||""}return`
`+Rl+e}var Ol=!1;function Fl(e,t){if(!e||Ol)return"";Ol=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Ol=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?kr(e):""}function mh(e){switch(e.tag){case 5:return kr(e.type);case 16:return kr("Lazy");case 13:return kr("Suspense");case 19:return kr("SuspenseList");case 0:case 2:case 15:return e=Fl(e.type,!1),e;case 11:return e=Fl(e.type.render,!1),e;case 1:return e=Fl(e.type,!0),e;default:return""}}function Co(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Ln:return"Fragment";case Tn:return"Portal";case So:return"Profiler";case Pa:return"StrictMode";case bo:return"Suspense";case jo:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case ed:return(e.displayName||"Context")+".Consumer";case Zc:return(e._context.displayName||"Context")+".Provider";case Ia:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Da:return t=e.displayName||null,t!==null?t:Co(e.type)||"Memo";case Bt:t=e._payload,e=e._init;try{return Co(e(t))}catch{}}return null}function gh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Co(t);case 8:return t===Pa?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function en(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function nd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function vh(e){var t=nd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function ci(e){e._valueTracker||(e._valueTracker=vh(e))}function rd(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=nd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Vi(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function Eo(e,t){var n=t.checked;return de({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function Ws(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=en(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function id(e,t){t=t.checked,t!=null&&La(e,"checked",t,!1)}function No(e,t){id(e,t);var n=en(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?_o(e,t.type,n):t.hasOwnProperty("defaultValue")&&_o(e,t.type,en(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function Qs(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function _o(e,t,n){(t!=="number"||Vi(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var wr=Array.isArray;function $n(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+en(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function zo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(M(91));return de({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function Ks(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(M(92));if(wr(n)){if(1<n.length)throw Error(M(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:en(n)}}function ld(e,t){var n=en(t.value),r=en(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function qs(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function od(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function To(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?od(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var di,ad=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(di=di||document.createElement("div"),di.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=di.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Rr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var jr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},yh=["Webkit","ms","Moz","O"];Object.keys(jr).forEach(function(e){yh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),jr[t]=jr[e]})});function sd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||jr.hasOwnProperty(e)&&jr[e]?(""+t).trim():t+"px"}function ud(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=sd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var xh=de({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Lo(e,t){if(t){if(xh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(M(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(M(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(M(61))}if(t.style!=null&&typeof t.style!="object")throw Error(M(62))}}function Po(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Io=null;function Ma(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Do=null,Hn=null,Vn=null;function Ys(e){if(e=ri(e)){if(typeof Do!="function")throw Error(M(280));var t=e.stateNode;t&&(t=wl(t),Do(e.stateNode,e.type,t))}}function cd(e){Hn?Vn?Vn.push(e):Vn=[e]:Hn=e}function dd(){if(Hn){var e=Hn,t=Vn;if(Vn=Hn=null,Ys(e),t)for(e=0;e<t.length;e++)Ys(t[e])}}function fd(e,t){return e(t)}function pd(){}var Bl=!1;function hd(e,t,n){if(Bl)return e(t,n);Bl=!0;try{return fd(e,t,n)}finally{Bl=!1,(Hn!==null||Vn!==null)&&(pd(),dd())}}function Or(e,t){var n=e.stateNode;if(n===null)return null;var r=wl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(M(231,t,typeof n));return n}var Mo=!1;if(Pt)try{var dr={};Object.defineProperty(dr,"passive",{get:function(){Mo=!0}}),window.addEventListener("test",dr,dr),window.removeEventListener("test",dr,dr)}catch{Mo=!1}function kh(e,t,n,r,i,l,o,a,s){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Cr=!1,Wi=null,Qi=!1,Ao=null,wh={onError:function(e){Cr=!0,Wi=e}};function Sh(e,t,n,r,i,l,o,a,s){Cr=!1,Wi=null,kh.apply(wh,arguments)}function bh(e,t,n,r,i,l,o,a,s){if(Sh.apply(this,arguments),Cr){if(Cr){var c=Wi;Cr=!1,Wi=null}else throw Error(M(198));Qi||(Qi=!0,Ao=c)}}function bn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function md(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function Xs(e){if(bn(e)!==e)throw Error(M(188))}function jh(e){var t=e.alternate;if(!t){if(t=bn(e),t===null)throw Error(M(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return Xs(i),e;if(l===r)return Xs(i),t;l=l.sibling}throw Error(M(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(M(189))}}if(n.alternate!==r)throw Error(M(190))}if(n.tag!==3)throw Error(M(188));return n.stateNode.current===n?e:t}function gd(e){return e=jh(e),e!==null?vd(e):null}function vd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=vd(e);if(t!==null)return t;e=e.sibling}return null}var yd=Ye.unstable_scheduleCallback,Gs=Ye.unstable_cancelCallback,Ch=Ye.unstable_shouldYield,Eh=Ye.unstable_requestPaint,pe=Ye.unstable_now,Nh=Ye.unstable_getCurrentPriorityLevel,Aa=Ye.unstable_ImmediatePriority,xd=Ye.unstable_UserBlockingPriority,Ki=Ye.unstable_NormalPriority,_h=Ye.unstable_LowPriority,kd=Ye.unstable_IdlePriority,vl=null,wt=null;function zh(e){if(wt&&typeof wt.onCommitFiberRoot=="function")try{wt.onCommitFiberRoot(vl,e,void 0,(e.current.flags&128)===128)}catch{}}var dt=Math.clz32?Math.clz32:Ph,Th=Math.log,Lh=Math.LN2;function Ph(e){return e>>>=0,e===0?32:31-(Th(e)/Lh|0)|0}var fi=64,pi=4194304;function Sr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function qi(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=Sr(a):(l&=o,l!==0&&(r=Sr(l)))}else o=n&~i,o!==0?r=Sr(o):l!==0&&(r=Sr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-dt(t),i=1<<n,r|=e[n],t&=~i;return r}function Ih(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Dh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-dt(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Ih(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Ro(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function wd(){var e=fi;return fi<<=1,!(fi&4194240)&&(fi=64),e}function Ul(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function ti(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-dt(t),e[t]=n}function Mh(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-dt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Ra(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-dt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var ee=0;function Sd(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var bd,Oa,jd,Cd,Ed,Oo=!1,hi=[],Qt=null,Kt=null,qt=null,Fr=new Map,Br=new Map,$t=[],Ah="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function Js(e,t){switch(e){case"focusin":case"focusout":Qt=null;break;case"dragenter":case"dragleave":Kt=null;break;case"mouseover":case"mouseout":qt=null;break;case"pointerover":case"pointerout":Fr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Br.delete(t.pointerId)}}function fr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=ri(t),t!==null&&Oa(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Rh(e,t,n,r,i){switch(t){case"focusin":return Qt=fr(Qt,e,t,n,r,i),!0;case"dragenter":return Kt=fr(Kt,e,t,n,r,i),!0;case"mouseover":return qt=fr(qt,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Fr.set(l,fr(Fr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Br.set(l,fr(Br.get(l)||null,e,t,n,r,i)),!0}return!1}function Nd(e){var t=fn(e.target);if(t!==null){var n=bn(t);if(n!==null){if(t=n.tag,t===13){if(t=md(n),t!==null){e.blockedOn=t,Ed(e.priority,function(){jd(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Li(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Fo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Io=r,n.target.dispatchEvent(r),Io=null}else return t=ri(n),t!==null&&Oa(t),e.blockedOn=n,!1;t.shift()}return!0}function Zs(e,t,n){Li(e)&&n.delete(t)}function Oh(){Oo=!1,Qt!==null&&Li(Qt)&&(Qt=null),Kt!==null&&Li(Kt)&&(Kt=null),qt!==null&&Li(qt)&&(qt=null),Fr.forEach(Zs),Br.forEach(Zs)}function pr(e,t){e.blockedOn===t&&(e.blockedOn=null,Oo||(Oo=!0,Ye.unstable_scheduleCallback(Ye.unstable_NormalPriority,Oh)))}function Ur(e){function t(i){return pr(i,e)}if(0<hi.length){pr(hi[0],e);for(var n=1;n<hi.length;n++){var r=hi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(Qt!==null&&pr(Qt,e),Kt!==null&&pr(Kt,e),qt!==null&&pr(qt,e),Fr.forEach(t),Br.forEach(t),n=0;n<$t.length;n++)r=$t[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<$t.length&&(n=$t[0],n.blockedOn===null);)Nd(n),n.blockedOn===null&&$t.shift()}var Wn=At.ReactCurrentBatchConfig,Yi=!0;function Fh(e,t,n,r){var i=ee,l=Wn.transition;Wn.transition=null;try{ee=1,Fa(e,t,n,r)}finally{ee=i,Wn.transition=l}}function Bh(e,t,n,r){var i=ee,l=Wn.transition;Wn.transition=null;try{ee=4,Fa(e,t,n,r)}finally{ee=i,Wn.transition=l}}function Fa(e,t,n,r){if(Yi){var i=Fo(e,t,n,r);if(i===null)Gl(e,t,r,Xi,n),Js(e,r);else if(Rh(i,e,t,n,r))r.stopPropagation();else if(Js(e,r),t&4&&-1<Ah.indexOf(e)){for(;i!==null;){var l=ri(i);if(l!==null&&bd(l),l=Fo(e,t,n,r),l===null&&Gl(e,t,r,Xi,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else Gl(e,t,r,null,n)}}var Xi=null;function Fo(e,t,n,r){if(Xi=null,e=Ma(r),e=fn(e),e!==null)if(t=bn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=md(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return Xi=e,null}function _d(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(Nh()){case Aa:return 1;case xd:return 4;case Ki:case _h:return 16;case kd:return 536870912;default:return 16}default:return 16}}var Vt=null,Ba=null,Pi=null;function zd(){if(Pi)return Pi;var e,t=Ba,n=t.length,r,i="value"in Vt?Vt.value:Vt.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Pi=i.slice(e,1<r?1-r:void 0)}function Ii(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function mi(){return!0}function eu(){return!1}function Ge(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?mi:eu,this.isPropagationStopped=eu,this}return de(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=mi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=mi)},persist:function(){},isPersistent:mi}),t}var ir={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Ua=Ge(ir),ni=de({},ir,{view:0,detail:0}),Uh=Ge(ni),$l,Hl,hr,yl=de({},ni,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:$a,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==hr&&(hr&&e.type==="mousemove"?($l=e.screenX-hr.screenX,Hl=e.screenY-hr.screenY):Hl=$l=0,hr=e),$l)},movementY:function(e){return"movementY"in e?e.movementY:Hl}}),tu=Ge(yl),$h=de({},yl,{dataTransfer:0}),Hh=Ge($h),Vh=de({},ni,{relatedTarget:0}),Vl=Ge(Vh),Wh=de({},ir,{animationName:0,elapsedTime:0,pseudoElement:0}),Qh=Ge(Wh),Kh=de({},ir,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),qh=Ge(Kh),Yh=de({},ir,{data:0}),nu=Ge(Yh),Xh={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},Gh={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},Jh={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function Zh(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=Jh[e])?!!t[e]:!1}function $a(){return Zh}var em=de({},ni,{key:function(e){if(e.key){var t=Xh[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Ii(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?Gh[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:$a,charCode:function(e){return e.type==="keypress"?Ii(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Ii(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),tm=Ge(em),nm=de({},yl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),ru=Ge(nm),rm=de({},ni,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:$a}),im=Ge(rm),lm=de({},ir,{propertyName:0,elapsedTime:0,pseudoElement:0}),om=Ge(lm),am=de({},yl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),sm=Ge(am),um=[9,13,27,32],Ha=Pt&&"CompositionEvent"in window,Er=null;Pt&&"documentMode"in document&&(Er=document.documentMode);var cm=Pt&&"TextEvent"in window&&!Er,Td=Pt&&(!Ha||Er&&8<Er&&11>=Er),iu=" ",lu=!1;function Ld(e,t){switch(e){case"keyup":return um.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Pd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Pn=!1;function dm(e,t){switch(e){case"compositionend":return Pd(t);case"keypress":return t.which!==32?null:(lu=!0,iu);case"textInput":return e=t.data,e===iu&&lu?null:e;default:return null}}function fm(e,t){if(Pn)return e==="compositionend"||!Ha&&Ld(e,t)?(e=zd(),Pi=Ba=Vt=null,Pn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Td&&t.locale!=="ko"?null:t.data;default:return null}}var pm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function ou(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!pm[e.type]:t==="textarea"}function Id(e,t,n,r){cd(r),t=Gi(t,"onChange"),0<t.length&&(n=new Ua("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Nr=null,$r=null;function hm(e){Vd(e,0)}function xl(e){var t=Mn(e);if(rd(t))return e}function mm(e,t){if(e==="change")return t}var Dd=!1;if(Pt){var Wl;if(Pt){var Ql="oninput"in document;if(!Ql){var au=document.createElement("div");au.setAttribute("oninput","return;"),Ql=typeof au.oninput=="function"}Wl=Ql}else Wl=!1;Dd=Wl&&(!document.documentMode||9<document.documentMode)}function su(){Nr&&(Nr.detachEvent("onpropertychange",Md),$r=Nr=null)}function Md(e){if(e.propertyName==="value"&&xl($r)){var t=[];Id(t,$r,e,Ma(e)),hd(hm,t)}}function gm(e,t,n){e==="focusin"?(su(),Nr=t,$r=n,Nr.attachEvent("onpropertychange",Md)):e==="focusout"&&su()}function vm(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return xl($r)}function ym(e,t){if(e==="click")return xl(t)}function xm(e,t){if(e==="input"||e==="change")return xl(t)}function km(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var pt=typeof Object.is=="function"?Object.is:km;function Hr(e,t){if(pt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!wo.call(t,i)||!pt(e[i],t[i]))return!1}return!0}function uu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function cu(e,t){var n=uu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=uu(n)}}function Ad(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Ad(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Rd(){for(var e=window,t=Vi();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Vi(e.document)}return t}function Va(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function wm(e){var t=Rd(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Ad(n.ownerDocument.documentElement,n)){if(r!==null&&Va(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=cu(n,l);var o=cu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Sm=Pt&&"documentMode"in document&&11>=document.documentMode,In=null,Bo=null,_r=null,Uo=!1;function du(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Uo||In==null||In!==Vi(r)||(r=In,"selectionStart"in r&&Va(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),_r&&Hr(_r,r)||(_r=r,r=Gi(Bo,"onSelect"),0<r.length&&(t=new Ua("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=In)))}function gi(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Dn={animationend:gi("Animation","AnimationEnd"),animationiteration:gi("Animation","AnimationIteration"),animationstart:gi("Animation","AnimationStart"),transitionend:gi("Transition","TransitionEnd")},Kl={},Od={};Pt&&(Od=document.createElement("div").style,"AnimationEvent"in window||(delete Dn.animationend.animation,delete Dn.animationiteration.animation,delete Dn.animationstart.animation),"TransitionEvent"in window||delete Dn.transitionend.transition);function kl(e){if(Kl[e])return Kl[e];if(!Dn[e])return e;var t=Dn[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Od)return Kl[e]=t[n];return e}var Fd=kl("animationend"),Bd=kl("animationiteration"),Ud=kl("animationstart"),$d=kl("transitionend"),Hd=new Map,fu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function nn(e,t){Hd.set(e,t),Sn(t,[e])}for(var ql=0;ql<fu.length;ql++){var Yl=fu[ql],bm=Yl.toLowerCase(),jm=Yl[0].toUpperCase()+Yl.slice(1);nn(bm,"on"+jm)}nn(Fd,"onAnimationEnd");nn(Bd,"onAnimationIteration");nn(Ud,"onAnimationStart");nn("dblclick","onDoubleClick");nn("focusin","onFocus");nn("focusout","onBlur");nn($d,"onTransitionEnd");Xn("onMouseEnter",["mouseout","mouseover"]);Xn("onMouseLeave",["mouseout","mouseover"]);Xn("onPointerEnter",["pointerout","pointerover"]);Xn("onPointerLeave",["pointerout","pointerover"]);Sn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));Sn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));Sn("onBeforeInput",["compositionend","keypress","textInput","paste"]);Sn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));Sn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));Sn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var br="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Cm=new Set("cancel close invalid load scroll toggle".split(" ").concat(br));function pu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,bh(r,t,void 0,e),e.currentTarget=null}function Vd(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,c=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;pu(i,a,c),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,c=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;pu(i,a,c),l=s}}}if(Qi)throw e=Ao,Qi=!1,Ao=null,e}function oe(e,t){var n=t[Qo];n===void 0&&(n=t[Qo]=new Set);var r=e+"__bubble";n.has(r)||(Wd(t,e,2,!1),n.add(r))}function Xl(e,t,n){var r=0;t&&(r|=4),Wd(n,e,r,t)}var vi="_reactListening"+Math.random().toString(36).slice(2);function Vr(e){if(!e[vi]){e[vi]=!0,Jc.forEach(function(n){n!=="selectionchange"&&(Cm.has(n)||Xl(n,!1,e),Xl(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[vi]||(t[vi]=!0,Xl("selectionchange",!1,t))}}function Wd(e,t,n,r){switch(_d(t)){case 1:var i=Fh;break;case 4:i=Bh;break;default:i=Fa}n=i.bind(null,t,n,e),i=void 0,!Mo||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function Gl(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=fn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}hd(function(){var c=l,d=Ma(n),f=[];e:{var h=Hd.get(e);if(h!==void 0){var p=Ua,k=e;switch(e){case"keypress":if(Ii(n)===0)break e;case"keydown":case"keyup":p=tm;break;case"focusin":k="focus",p=Vl;break;case"focusout":k="blur",p=Vl;break;case"beforeblur":case"afterblur":p=Vl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":p=tu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":p=Hh;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":p=im;break;case Fd:case Bd:case Ud:p=Qh;break;case $d:p=om;break;case"scroll":p=Uh;break;case"wheel":p=sm;break;case"copy":case"cut":case"paste":p=qh;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":p=ru}var S=(t&4)!==0,C=!S&&e==="scroll",g=S?h!==null?h+"Capture":null:h;S=[];for(var m=c,y;m!==null;){y=m;var j=y.stateNode;if(y.tag===5&&j!==null&&(y=j,g!==null&&(j=Or(m,g),j!=null&&S.push(Wr(m,j,y)))),C)break;m=m.return}0<S.length&&(h=new p(h,k,null,n,d),f.push({event:h,listeners:S}))}}if(!(t&7)){e:{if(h=e==="mouseover"||e==="pointerover",p=e==="mouseout"||e==="pointerout",h&&n!==Io&&(k=n.relatedTarget||n.fromElement)&&(fn(k)||k[It]))break e;if((p||h)&&(h=d.window===d?d:(h=d.ownerDocument)?h.defaultView||h.parentWindow:window,p?(k=n.relatedTarget||n.toElement,p=c,k=k?fn(k):null,k!==null&&(C=bn(k),k!==C||k.tag!==5&&k.tag!==6)&&(k=null)):(p=null,k=c),p!==k)){if(S=tu,j="onMouseLeave",g="onMouseEnter",m="mouse",(e==="pointerout"||e==="pointerover")&&(S=ru,j="onPointerLeave",g="onPointerEnter",m="pointer"),C=p==null?h:Mn(p),y=k==null?h:Mn(k),h=new S(j,m+"leave",p,n,d),h.target=C,h.relatedTarget=y,j=null,fn(d)===c&&(S=new S(g,m+"enter",k,n,d),S.target=y,S.relatedTarget=C,j=S),C=j,p&&k)t:{for(S=p,g=k,m=0,y=S;y;y=_n(y))m++;for(y=0,j=g;j;j=_n(j))y++;for(;0<m-y;)S=_n(S),m--;for(;0<y-m;)g=_n(g),y--;for(;m--;){if(S===g||g!==null&&S===g.alternate)break t;S=_n(S),g=_n(g)}S=null}else S=null;p!==null&&hu(f,h,p,S,!1),k!==null&&C!==null&&hu(f,C,k,S,!0)}}e:{if(h=c?Mn(c):window,p=h.nodeName&&h.nodeName.toLowerCase(),p==="select"||p==="input"&&h.type==="file")var z=mm;else if(ou(h))if(Dd)z=xm;else{z=vm;var w=gm}else(p=h.nodeName)&&p.toLowerCase()==="input"&&(h.type==="checkbox"||h.type==="radio")&&(z=ym);if(z&&(z=z(e,c))){Id(f,z,n,d);break e}w&&w(e,h,c),e==="focusout"&&(w=h._wrapperState)&&w.controlled&&h.type==="number"&&_o(h,"number",h.value)}switch(w=c?Mn(c):window,e){case"focusin":(ou(w)||w.contentEditable==="true")&&(In=w,Bo=c,_r=null);break;case"focusout":_r=Bo=In=null;break;case"mousedown":Uo=!0;break;case"contextmenu":case"mouseup":case"dragend":Uo=!1,du(f,n,d);break;case"selectionchange":if(Sm)break;case"keydown":case"keyup":du(f,n,d)}var E;if(Ha)e:{switch(e){case"compositionstart":var b="onCompositionStart";break e;case"compositionend":b="onCompositionEnd";break e;case"compositionupdate":b="onCompositionUpdate";break e}b=void 0}else Pn?Ld(e,n)&&(b="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(b="onCompositionStart");b&&(Td&&n.locale!=="ko"&&(Pn||b!=="onCompositionStart"?b==="onCompositionEnd"&&Pn&&(E=zd()):(Vt=d,Ba="value"in Vt?Vt.value:Vt.textContent,Pn=!0)),w=Gi(c,b),0<w.length&&(b=new nu(b,e,null,n,d),f.push({event:b,listeners:w}),E?b.data=E:(E=Pd(n),E!==null&&(b.data=E)))),(E=cm?dm(e,n):fm(e,n))&&(c=Gi(c,"onBeforeInput"),0<c.length&&(d=new nu("onBeforeInput","beforeinput",null,n,d),f.push({event:d,listeners:c}),d.data=E))}Vd(f,t)})}function Wr(e,t,n){return{instance:e,listener:t,currentTarget:n}}function Gi(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Or(e,n),l!=null&&r.unshift(Wr(e,l,i)),l=Or(e,t),l!=null&&r.push(Wr(e,l,i))),e=e.return}return r}function _n(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function hu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,c=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&c!==null&&(a=c,i?(s=Or(n,l),s!=null&&o.unshift(Wr(n,s,a))):i||(s=Or(n,l),s!=null&&o.push(Wr(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Em=/\r\n?/g,Nm=/\u0000|\uFFFD/g;function mu(e){return(typeof e=="string"?e:""+e).replace(Em,`
`).replace(Nm,"")}function yi(e,t,n){if(t=mu(t),mu(e)!==t&&n)throw Error(M(425))}function Ji(){}var $o=null,Ho=null;function Vo(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var Wo=typeof setTimeout=="function"?setTimeout:void 0,_m=typeof clearTimeout=="function"?clearTimeout:void 0,gu=typeof Promise=="function"?Promise:void 0,zm=typeof queueMicrotask=="function"?queueMicrotask:typeof gu<"u"?function(e){return gu.resolve(null).then(e).catch(Tm)}:Wo;function Tm(e){setTimeout(function(){throw e})}function Jl(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Ur(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Ur(t)}function Yt(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function vu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var lr=Math.random().toString(36).slice(2),xt="__reactFiber$"+lr,Qr="__reactProps$"+lr,It="__reactContainer$"+lr,Qo="__reactEvents$"+lr,Lm="__reactListeners$"+lr,Pm="__reactHandles$"+lr;function fn(e){var t=e[xt];if(t)return t;for(var n=e.parentNode;n;){if(t=n[It]||n[xt]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=vu(e);e!==null;){if(n=e[xt])return n;e=vu(e)}return t}e=n,n=e.parentNode}return null}function ri(e){return e=e[xt]||e[It],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function Mn(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(M(33))}function wl(e){return e[Qr]||null}var Ko=[],An=-1;function rn(e){return{current:e}}function ae(e){0>An||(e.current=Ko[An],Ko[An]=null,An--)}function re(e,t){An++,Ko[An]=e.current,e.current=t}var tn={},_e=rn(tn),Re=rn(!1),vn=tn;function Gn(e,t){var n=e.type.contextTypes;if(!n)return tn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Oe(e){return e=e.childContextTypes,e!=null}function Zi(){ae(Re),ae(_e)}function yu(e,t,n){if(_e.current!==tn)throw Error(M(168));re(_e,t),re(Re,n)}function Qd(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(M(108,gh(e)||"Unknown",i));return de({},n,r)}function el(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||tn,vn=_e.current,re(_e,e),re(Re,Re.current),!0}function xu(e,t,n){var r=e.stateNode;if(!r)throw Error(M(169));n?(e=Qd(e,t,vn),r.__reactInternalMemoizedMergedChildContext=e,ae(Re),ae(_e),re(_e,e)):ae(Re),re(Re,n)}var _t=null,Sl=!1,Zl=!1;function Kd(e){_t===null?_t=[e]:_t.push(e)}function Im(e){Sl=!0,Kd(e)}function ln(){if(!Zl&&_t!==null){Zl=!0;var e=0,t=ee;try{var n=_t;for(ee=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}_t=null,Sl=!1}catch(i){throw _t!==null&&(_t=_t.slice(e+1)),yd(Aa,ln),i}finally{ee=t,Zl=!1}}return null}var Rn=[],On=0,tl=null,nl=0,Je=[],Ze=0,yn=null,zt=1,Tt="";function sn(e,t){Rn[On++]=nl,Rn[On++]=tl,tl=e,nl=t}function qd(e,t,n){Je[Ze++]=zt,Je[Ze++]=Tt,Je[Ze++]=yn,yn=e;var r=zt;e=Tt;var i=32-dt(r)-1;r&=~(1<<i),n+=1;var l=32-dt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,zt=1<<32-dt(t)+i|n<<i|r,Tt=l+e}else zt=1<<l|n<<i|r,Tt=e}function Wa(e){e.return!==null&&(sn(e,1),qd(e,1,0))}function Qa(e){for(;e===tl;)tl=Rn[--On],Rn[On]=null,nl=Rn[--On],Rn[On]=null;for(;e===yn;)yn=Je[--Ze],Je[Ze]=null,Tt=Je[--Ze],Je[Ze]=null,zt=Je[--Ze],Je[Ze]=null}var qe=null,Qe=null,se=!1,ct=null;function Yd(e,t){var n=tt(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function ku(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,qe=e,Qe=Yt(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,qe=e,Qe=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=yn!==null?{id:zt,overflow:Tt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=tt(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,qe=e,Qe=null,!0):!1;default:return!1}}function qo(e){return(e.mode&1)!==0&&(e.flags&128)===0}function Yo(e){if(se){var t=Qe;if(t){var n=t;if(!ku(e,t)){if(qo(e))throw Error(M(418));t=Yt(n.nextSibling);var r=qe;t&&ku(e,t)?Yd(r,n):(e.flags=e.flags&-4097|2,se=!1,qe=e)}}else{if(qo(e))throw Error(M(418));e.flags=e.flags&-4097|2,se=!1,qe=e}}}function wu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;qe=e}function xi(e){if(e!==qe)return!1;if(!se)return wu(e),se=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!Vo(e.type,e.memoizedProps)),t&&(t=Qe)){if(qo(e))throw Xd(),Error(M(418));for(;t;)Yd(e,t),t=Yt(t.nextSibling)}if(wu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(M(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){Qe=Yt(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}Qe=null}}else Qe=qe?Yt(e.stateNode.nextSibling):null;return!0}function Xd(){for(var e=Qe;e;)e=Yt(e.nextSibling)}function Jn(){Qe=qe=null,se=!1}function Ka(e){ct===null?ct=[e]:ct.push(e)}var Dm=At.ReactCurrentBatchConfig;function mr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(M(309));var r=n.stateNode}if(!r)throw Error(M(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(M(284));if(!n._owner)throw Error(M(290,e))}return e}function ki(e,t){throw e=Object.prototype.toString.call(t),Error(M(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Su(e){var t=e._init;return t(e._payload)}function Gd(e){function t(g,m){if(e){var y=g.deletions;y===null?(g.deletions=[m],g.flags|=16):y.push(m)}}function n(g,m){if(!e)return null;for(;m!==null;)t(g,m),m=m.sibling;return null}function r(g,m){for(g=new Map;m!==null;)m.key!==null?g.set(m.key,m):g.set(m.index,m),m=m.sibling;return g}function i(g,m){return g=Zt(g,m),g.index=0,g.sibling=null,g}function l(g,m,y){return g.index=y,e?(y=g.alternate,y!==null?(y=y.index,y<m?(g.flags|=2,m):y):(g.flags|=2,m)):(g.flags|=1048576,m)}function o(g){return e&&g.alternate===null&&(g.flags|=2),g}function a(g,m,y,j){return m===null||m.tag!==6?(m=oo(y,g.mode,j),m.return=g,m):(m=i(m,y),m.return=g,m)}function s(g,m,y,j){var z=y.type;return z===Ln?d(g,m,y.props.children,j,y.key):m!==null&&(m.elementType===z||typeof z=="object"&&z!==null&&z.$$typeof===Bt&&Su(z)===m.type)?(j=i(m,y.props),j.ref=mr(g,m,y),j.return=g,j):(j=Bi(y.type,y.key,y.props,null,g.mode,j),j.ref=mr(g,m,y),j.return=g,j)}function c(g,m,y,j){return m===null||m.tag!==4||m.stateNode.containerInfo!==y.containerInfo||m.stateNode.implementation!==y.implementation?(m=ao(y,g.mode,j),m.return=g,m):(m=i(m,y.children||[]),m.return=g,m)}function d(g,m,y,j,z){return m===null||m.tag!==7?(m=gn(y,g.mode,j,z),m.return=g,m):(m=i(m,y),m.return=g,m)}function f(g,m,y){if(typeof m=="string"&&m!==""||typeof m=="number")return m=oo(""+m,g.mode,y),m.return=g,m;if(typeof m=="object"&&m!==null){switch(m.$$typeof){case ui:return y=Bi(m.type,m.key,m.props,null,g.mode,y),y.ref=mr(g,null,m),y.return=g,y;case Tn:return m=ao(m,g.mode,y),m.return=g,m;case Bt:var j=m._init;return f(g,j(m._payload),y)}if(wr(m)||cr(m))return m=gn(m,g.mode,y,null),m.return=g,m;ki(g,m)}return null}function h(g,m,y,j){var z=m!==null?m.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return z!==null?null:a(g,m,""+y,j);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case ui:return y.key===z?s(g,m,y,j):null;case Tn:return y.key===z?c(g,m,y,j):null;case Bt:return z=y._init,h(g,m,z(y._payload),j)}if(wr(y)||cr(y))return z!==null?null:d(g,m,y,j,null);ki(g,y)}return null}function p(g,m,y,j,z){if(typeof j=="string"&&j!==""||typeof j=="number")return g=g.get(y)||null,a(m,g,""+j,z);if(typeof j=="object"&&j!==null){switch(j.$$typeof){case ui:return g=g.get(j.key===null?y:j.key)||null,s(m,g,j,z);case Tn:return g=g.get(j.key===null?y:j.key)||null,c(m,g,j,z);case Bt:var w=j._init;return p(g,m,y,w(j._payload),z)}if(wr(j)||cr(j))return g=g.get(y)||null,d(m,g,j,z,null);ki(m,j)}return null}function k(g,m,y,j){for(var z=null,w=null,E=m,b=m=0,N=null;E!==null&&b<y.length;b++){E.index>b?(N=E,E=null):N=E.sibling;var A=h(g,E,y[b],j);if(A===null){E===null&&(E=N);break}e&&E&&A.alternate===null&&t(g,E),m=l(A,m,b),w===null?z=A:w.sibling=A,w=A,E=N}if(b===y.length)return n(g,E),se&&sn(g,b),z;if(E===null){for(;b<y.length;b++)E=f(g,y[b],j),E!==null&&(m=l(E,m,b),w===null?z=E:w.sibling=E,w=E);return se&&sn(g,b),z}for(E=r(g,E);b<y.length;b++)N=p(E,g,b,y[b],j),N!==null&&(e&&N.alternate!==null&&E.delete(N.key===null?b:N.key),m=l(N,m,b),w===null?z=N:w.sibling=N,w=N);return e&&E.forEach(function(_){return t(g,_)}),se&&sn(g,b),z}function S(g,m,y,j){var z=cr(y);if(typeof z!="function")throw Error(M(150));if(y=z.call(y),y==null)throw Error(M(151));for(var w=z=null,E=m,b=m=0,N=null,A=y.next();E!==null&&!A.done;b++,A=y.next()){E.index>b?(N=E,E=null):N=E.sibling;var _=h(g,E,A.value,j);if(_===null){E===null&&(E=N);break}e&&E&&_.alternate===null&&t(g,E),m=l(_,m,b),w===null?z=_:w.sibling=_,w=_,E=N}if(A.done)return n(g,E),se&&sn(g,b),z;if(E===null){for(;!A.done;b++,A=y.next())A=f(g,A.value,j),A!==null&&(m=l(A,m,b),w===null?z=A:w.sibling=A,w=A);return se&&sn(g,b),z}for(E=r(g,E);!A.done;b++,A=y.next())A=p(E,g,b,A.value,j),A!==null&&(e&&A.alternate!==null&&E.delete(A.key===null?b:A.key),m=l(A,m,b),w===null?z=A:w.sibling=A,w=A);return e&&E.forEach(function(I){return t(g,I)}),se&&sn(g,b),z}function C(g,m,y,j){if(typeof y=="object"&&y!==null&&y.type===Ln&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case ui:e:{for(var z=y.key,w=m;w!==null;){if(w.key===z){if(z=y.type,z===Ln){if(w.tag===7){n(g,w.sibling),m=i(w,y.props.children),m.return=g,g=m;break e}}else if(w.elementType===z||typeof z=="object"&&z!==null&&z.$$typeof===Bt&&Su(z)===w.type){n(g,w.sibling),m=i(w,y.props),m.ref=mr(g,w,y),m.return=g,g=m;break e}n(g,w);break}else t(g,w);w=w.sibling}y.type===Ln?(m=gn(y.props.children,g.mode,j,y.key),m.return=g,g=m):(j=Bi(y.type,y.key,y.props,null,g.mode,j),j.ref=mr(g,m,y),j.return=g,g=j)}return o(g);case Tn:e:{for(w=y.key;m!==null;){if(m.key===w)if(m.tag===4&&m.stateNode.containerInfo===y.containerInfo&&m.stateNode.implementation===y.implementation){n(g,m.sibling),m=i(m,y.children||[]),m.return=g,g=m;break e}else{n(g,m);break}else t(g,m);m=m.sibling}m=ao(y,g.mode,j),m.return=g,g=m}return o(g);case Bt:return w=y._init,C(g,m,w(y._payload),j)}if(wr(y))return k(g,m,y,j);if(cr(y))return S(g,m,y,j);ki(g,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,m!==null&&m.tag===6?(n(g,m.sibling),m=i(m,y),m.return=g,g=m):(n(g,m),m=oo(y,g.mode,j),m.return=g,g=m),o(g)):n(g,m)}return C}var Zn=Gd(!0),Jd=Gd(!1),rl=rn(null),il=null,Fn=null,qa=null;function Ya(){qa=Fn=il=null}function Xa(e){var t=rl.current;ae(rl),e._currentValue=t}function Xo(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Qn(e,t){il=e,qa=Fn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(Ae=!0),e.firstContext=null)}function rt(e){var t=e._currentValue;if(qa!==e)if(e={context:e,memoizedValue:t,next:null},Fn===null){if(il===null)throw Error(M(308));Fn=e,il.dependencies={lanes:0,firstContext:e}}else Fn=Fn.next=e;return t}var pn=null;function Ga(e){pn===null?pn=[e]:pn.push(e)}function Zd(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,Ga(t)):(n.next=i.next,i.next=n),t.interleaved=n,Dt(e,r)}function Dt(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Ut=!1;function Ja(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function ef(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Lt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function Xt(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,G&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Dt(e,n)}return i=r.interleaved,i===null?(t.next=t,Ga(r)):(t.next=i.next,i.next=t),r.interleaved=t,Dt(e,n)}function Di(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ra(e,n)}}function bu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function ll(e,t,n,r){var i=e.updateQueue;Ut=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,c=s.next;s.next=null,o===null?l=c:o.next=c,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=s))}if(l!==null){var f=i.baseState;o=0,d=c=s=null,a=l;do{var h=a.lane,p=a.eventTime;if((r&h)===h){d!==null&&(d=d.next={eventTime:p,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var k=e,S=a;switch(h=t,p=n,S.tag){case 1:if(k=S.payload,typeof k=="function"){f=k.call(p,f,h);break e}f=k;break e;case 3:k.flags=k.flags&-65537|128;case 0:if(k=S.payload,h=typeof k=="function"?k.call(p,f,h):k,h==null)break e;f=de({},f,h);break e;case 2:Ut=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,h=i.effects,h===null?i.effects=[a]:h.push(a))}else p={eventTime:p,lane:h,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=p,s=f):d=d.next=p,o|=h;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;h=a,a=h.next,h.next=null,i.lastBaseUpdate=h,i.shared.pending=null}}while(!0);if(d===null&&(s=f),i.baseState=s,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);kn|=o,e.lanes=o,e.memoizedState=f}}function ju(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(M(191,i));i.call(r)}}}var ii={},St=rn(ii),Kr=rn(ii),qr=rn(ii);function hn(e){if(e===ii)throw Error(M(174));return e}function Za(e,t){switch(re(qr,t),re(Kr,e),re(St,ii),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:To(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=To(t,e)}ae(St),re(St,t)}function er(){ae(St),ae(Kr),ae(qr)}function tf(e){hn(qr.current);var t=hn(St.current),n=To(t,e.type);t!==n&&(re(Kr,e),re(St,n))}function es(e){Kr.current===e&&(ae(St),ae(Kr))}var ue=rn(0);function ol(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var eo=[];function ts(){for(var e=0;e<eo.length;e++)eo[e]._workInProgressVersionPrimary=null;eo.length=0}var Mi=At.ReactCurrentDispatcher,to=At.ReactCurrentBatchConfig,xn=0,ce=null,ye=null,ke=null,al=!1,zr=!1,Yr=0,Mm=0;function Ce(){throw Error(M(321))}function ns(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!pt(e[n],t[n]))return!1;return!0}function rs(e,t,n,r,i,l){if(xn=l,ce=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Mi.current=e===null||e.memoizedState===null?Fm:Bm,e=n(r,i),zr){l=0;do{if(zr=!1,Yr=0,25<=l)throw Error(M(301));l+=1,ke=ye=null,t.updateQueue=null,Mi.current=Um,e=n(r,i)}while(zr)}if(Mi.current=sl,t=ye!==null&&ye.next!==null,xn=0,ke=ye=ce=null,al=!1,t)throw Error(M(300));return e}function is(){var e=Yr!==0;return Yr=0,e}function gt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return ke===null?ce.memoizedState=ke=e:ke=ke.next=e,ke}function it(){if(ye===null){var e=ce.alternate;e=e!==null?e.memoizedState:null}else e=ye.next;var t=ke===null?ce.memoizedState:ke.next;if(t!==null)ke=t,ye=e;else{if(e===null)throw Error(M(310));ye=e,e={memoizedState:ye.memoizedState,baseState:ye.baseState,baseQueue:ye.baseQueue,queue:ye.queue,next:null},ke===null?ce.memoizedState=ke=e:ke=ke.next=e}return ke}function Xr(e,t){return typeof t=="function"?t(e):t}function no(e){var t=it(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=ye,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,c=l;do{var d=c.lane;if((xn&d)===d)s!==null&&(s=s.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var f={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};s===null?(a=s=f,o=r):s=s.next=f,ce.lanes|=d,kn|=d}c=c.next}while(c!==null&&c!==l);s===null?o=r:s.next=a,pt(r,t.memoizedState)||(Ae=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,ce.lanes|=l,kn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function ro(e){var t=it(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);pt(l,t.memoizedState)||(Ae=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function nf(){}function rf(e,t){var n=ce,r=it(),i=t(),l=!pt(r.memoizedState,i);if(l&&(r.memoizedState=i,Ae=!0),r=r.queue,ls(af.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||ke!==null&&ke.memoizedState.tag&1){if(n.flags|=2048,Gr(9,of.bind(null,n,r,i,t),void 0,null),we===null)throw Error(M(349));xn&30||lf(n,t,i)}return i}function lf(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=ce.updateQueue,t===null?(t={lastEffect:null,stores:null},ce.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function of(e,t,n,r){t.value=n,t.getSnapshot=r,sf(t)&&uf(e)}function af(e,t,n){return n(function(){sf(t)&&uf(e)})}function sf(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!pt(e,n)}catch{return!0}}function uf(e){var t=Dt(e,1);t!==null&&ft(t,e,1,-1)}function Cu(e){var t=gt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:Xr,lastRenderedState:e},t.queue=e,e=e.dispatch=Om.bind(null,ce,e),[t.memoizedState,e]}function Gr(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=ce.updateQueue,t===null?(t={lastEffect:null,stores:null},ce.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function cf(){return it().memoizedState}function Ai(e,t,n,r){var i=gt();ce.flags|=e,i.memoizedState=Gr(1|t,n,void 0,r===void 0?null:r)}function bl(e,t,n,r){var i=it();r=r===void 0?null:r;var l=void 0;if(ye!==null){var o=ye.memoizedState;if(l=o.destroy,r!==null&&ns(r,o.deps)){i.memoizedState=Gr(t,n,l,r);return}}ce.flags|=e,i.memoizedState=Gr(1|t,n,l,r)}function Eu(e,t){return Ai(8390656,8,e,t)}function ls(e,t){return bl(2048,8,e,t)}function df(e,t){return bl(4,2,e,t)}function ff(e,t){return bl(4,4,e,t)}function pf(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function hf(e,t,n){return n=n!=null?n.concat([e]):null,bl(4,4,pf.bind(null,t,e),n)}function os(){}function mf(e,t){var n=it();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ns(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function gf(e,t){var n=it();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ns(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function vf(e,t,n){return xn&21?(pt(n,t)||(n=wd(),ce.lanes|=n,kn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,Ae=!0),e.memoizedState=n)}function Am(e,t){var n=ee;ee=n!==0&&4>n?n:4,e(!0);var r=to.transition;to.transition={};try{e(!1),t()}finally{ee=n,to.transition=r}}function yf(){return it().memoizedState}function Rm(e,t,n){var r=Jt(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},xf(e))kf(t,n);else if(n=Zd(e,t,n,r),n!==null){var i=Le();ft(n,e,r,i),wf(n,t,r)}}function Om(e,t,n){var r=Jt(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(xf(e))kf(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,pt(a,o)){var s=t.interleaved;s===null?(i.next=i,Ga(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=Zd(e,t,i,r),n!==null&&(i=Le(),ft(n,e,r,i),wf(n,t,r))}}function xf(e){var t=e.alternate;return e===ce||t!==null&&t===ce}function kf(e,t){zr=al=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function wf(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ra(e,n)}}var sl={readContext:rt,useCallback:Ce,useContext:Ce,useEffect:Ce,useImperativeHandle:Ce,useInsertionEffect:Ce,useLayoutEffect:Ce,useMemo:Ce,useReducer:Ce,useRef:Ce,useState:Ce,useDebugValue:Ce,useDeferredValue:Ce,useTransition:Ce,useMutableSource:Ce,useSyncExternalStore:Ce,useId:Ce,unstable_isNewReconciler:!1},Fm={readContext:rt,useCallback:function(e,t){return gt().memoizedState=[e,t===void 0?null:t],e},useContext:rt,useEffect:Eu,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Ai(4194308,4,pf.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Ai(4194308,4,e,t)},useInsertionEffect:function(e,t){return Ai(4,2,e,t)},useMemo:function(e,t){var n=gt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=gt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Rm.bind(null,ce,e),[r.memoizedState,e]},useRef:function(e){var t=gt();return e={current:e},t.memoizedState=e},useState:Cu,useDebugValue:os,useDeferredValue:function(e){return gt().memoizedState=e},useTransition:function(){var e=Cu(!1),t=e[0];return e=Am.bind(null,e[1]),gt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=ce,i=gt();if(se){if(n===void 0)throw Error(M(407));n=n()}else{if(n=t(),we===null)throw Error(M(349));xn&30||lf(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Eu(af.bind(null,r,l,e),[e]),r.flags|=2048,Gr(9,of.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=gt(),t=we.identifierPrefix;if(se){var n=Tt,r=zt;n=(r&~(1<<32-dt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=Yr++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Mm++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},Bm={readContext:rt,useCallback:mf,useContext:rt,useEffect:ls,useImperativeHandle:hf,useInsertionEffect:df,useLayoutEffect:ff,useMemo:gf,useReducer:no,useRef:cf,useState:function(){return no(Xr)},useDebugValue:os,useDeferredValue:function(e){var t=it();return vf(t,ye.memoizedState,e)},useTransition:function(){var e=no(Xr)[0],t=it().memoizedState;return[e,t]},useMutableSource:nf,useSyncExternalStore:rf,useId:yf,unstable_isNewReconciler:!1},Um={readContext:rt,useCallback:mf,useContext:rt,useEffect:ls,useImperativeHandle:hf,useInsertionEffect:df,useLayoutEffect:ff,useMemo:gf,useReducer:ro,useRef:cf,useState:function(){return ro(Xr)},useDebugValue:os,useDeferredValue:function(e){var t=it();return ye===null?t.memoizedState=e:vf(t,ye.memoizedState,e)},useTransition:function(){var e=ro(Xr)[0],t=it().memoizedState;return[e,t]},useMutableSource:nf,useSyncExternalStore:rf,useId:yf,unstable_isNewReconciler:!1};function st(e,t){if(e&&e.defaultProps){t=de({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function Go(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:de({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var jl={isMounted:function(e){return(e=e._reactInternals)?bn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Le(),i=Jt(e),l=Lt(r,i);l.payload=t,n!=null&&(l.callback=n),t=Xt(e,l,i),t!==null&&(ft(t,e,i,r),Di(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Le(),i=Jt(e),l=Lt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=Xt(e,l,i),t!==null&&(ft(t,e,i,r),Di(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Le(),r=Jt(e),i=Lt(n,r);i.tag=2,t!=null&&(i.callback=t),t=Xt(e,i,r),t!==null&&(ft(t,e,r,n),Di(t,e,r))}};function Nu(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Hr(n,r)||!Hr(i,l):!0}function Sf(e,t,n){var r=!1,i=tn,l=t.contextType;return typeof l=="object"&&l!==null?l=rt(l):(i=Oe(t)?vn:_e.current,r=t.contextTypes,l=(r=r!=null)?Gn(e,i):tn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=jl,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function _u(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&jl.enqueueReplaceState(t,t.state,null)}function Jo(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},Ja(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=rt(l):(l=Oe(t)?vn:_e.current,i.context=Gn(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(Go(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&jl.enqueueReplaceState(i,i.state,null),ll(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function tr(e,t){try{var n="",r=t;do n+=mh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function io(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function Zo(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var $m=typeof WeakMap=="function"?WeakMap:Map;function bf(e,t,n){n=Lt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){cl||(cl=!0,ua=r),Zo(e,t)},n}function jf(e,t,n){n=Lt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){Zo(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){Zo(e,t),typeof r!="function"&&(Gt===null?Gt=new Set([this]):Gt.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function zu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new $m;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=ng.bind(null,e,t,n),t.then(e,e))}function Tu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Lu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Lt(-1,1),t.tag=2,Xt(n,t,1))),n.lanes|=1),e)}var Hm=At.ReactCurrentOwner,Ae=!1;function Te(e,t,n,r){t.child=e===null?Jd(t,null,n,r):Zn(t,e.child,n,r)}function Pu(e,t,n,r,i){n=n.render;var l=t.ref;return Qn(t,i),r=rs(e,t,n,r,l,i),n=is(),e!==null&&!Ae?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Mt(e,t,i)):(se&&n&&Wa(t),t.flags|=1,Te(e,t,r,i),t.child)}function Iu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!hs(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,Cf(e,t,l,r,i)):(e=Bi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Hr,n(o,r)&&e.ref===t.ref)return Mt(e,t,i)}return t.flags|=1,e=Zt(l,r),e.ref=t.ref,e.return=t,t.child=e}function Cf(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Hr(l,r)&&e.ref===t.ref)if(Ae=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(Ae=!0);else return t.lanes=e.lanes,Mt(e,t,i)}return ea(e,t,n,r,i)}function Ef(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},re(Un,We),We|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,re(Un,We),We|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,re(Un,We),We|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,re(Un,We),We|=r;return Te(e,t,i,n),t.child}function Nf(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ea(e,t,n,r,i){var l=Oe(n)?vn:_e.current;return l=Gn(t,l),Qn(t,i),n=rs(e,t,n,r,l,i),r=is(),e!==null&&!Ae?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Mt(e,t,i)):(se&&r&&Wa(t),t.flags|=1,Te(e,t,n,i),t.child)}function Du(e,t,n,r,i){if(Oe(n)){var l=!0;el(t)}else l=!1;if(Qn(t,i),t.stateNode===null)Ri(e,t),Sf(t,n,r),Jo(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=rt(c):(c=Oe(n)?vn:_e.current,c=Gn(t,c));var d=n.getDerivedStateFromProps,f=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";f||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==c)&&_u(t,o,r,c),Ut=!1;var h=t.memoizedState;o.state=h,ll(t,r,o,i),s=t.memoizedState,a!==r||h!==s||Re.current||Ut?(typeof d=="function"&&(Go(t,n,d,r),s=t.memoizedState),(a=Ut||Nu(t,n,a,r,h,s,c))?(f||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,ef(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:st(t.type,a),o.props=c,f=t.pendingProps,h=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=rt(s):(s=Oe(n)?vn:_e.current,s=Gn(t,s));var p=n.getDerivedStateFromProps;(d=typeof p=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==f||h!==s)&&_u(t,o,r,s),Ut=!1,h=t.memoizedState,o.state=h,ll(t,r,o,i);var k=t.memoizedState;a!==f||h!==k||Re.current||Ut?(typeof p=="function"&&(Go(t,n,p,r),k=t.memoizedState),(c=Ut||Nu(t,n,c,r,h,k,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,k,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,k,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&h===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&h===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=k),o.props=r,o.state=k,o.context=s,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&h===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&h===e.memoizedState||(t.flags|=1024),r=!1)}return ta(e,t,n,r,l,i)}function ta(e,t,n,r,i,l){Nf(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&xu(t,n,!1),Mt(e,t,l);r=t.stateNode,Hm.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=Zn(t,e.child,null,l),t.child=Zn(t,null,a,l)):Te(e,t,a,l),t.memoizedState=r.state,i&&xu(t,n,!0),t.child}function _f(e){var t=e.stateNode;t.pendingContext?yu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&yu(e,t.context,!1),Za(e,t.containerInfo)}function Mu(e,t,n,r,i){return Jn(),Ka(i),t.flags|=256,Te(e,t,n,r),t.child}var na={dehydrated:null,treeContext:null,retryLane:0};function ra(e){return{baseLanes:e,cachePool:null,transitions:null}}function zf(e,t,n){var r=t.pendingProps,i=ue.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),re(ue,i&1),e===null)return Yo(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Nl(o,r,0,null),e=gn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=ra(n),t.memoizedState=na,e):as(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return Vm(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=Zt(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=Zt(a,l):(l=gn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?ra(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=na,r}return l=e.child,e=l.sibling,r=Zt(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function as(e,t){return t=Nl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function wi(e,t,n,r){return r!==null&&Ka(r),Zn(t,e.child,null,n),e=as(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function Vm(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=io(Error(M(422))),wi(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Nl({mode:"visible",children:r.children},i,0,null),l=gn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&Zn(t,e.child,null,o),t.child.memoizedState=ra(o),t.memoizedState=na,l);if(!(t.mode&1))return wi(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(M(419)),r=io(l,r,void 0),wi(e,t,o,r)}if(a=(o&e.childLanes)!==0,Ae||a){if(r=we,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Dt(e,i),ft(r,e,i,-1))}return ps(),r=io(Error(M(421))),wi(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=rg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,Qe=Yt(i.nextSibling),qe=t,se=!0,ct=null,e!==null&&(Je[Ze++]=zt,Je[Ze++]=Tt,Je[Ze++]=yn,zt=e.id,Tt=e.overflow,yn=t),t=as(t,r.children),t.flags|=4096,t)}function Au(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),Xo(e.return,t,n)}function lo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Tf(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(Te(e,t,r.children,n),r=ue.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Au(e,n,t);else if(e.tag===19)Au(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(re(ue,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&ol(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),lo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&ol(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}lo(t,!0,n,null,l);break;case"together":lo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Ri(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Mt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),kn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(M(153));if(t.child!==null){for(e=t.child,n=Zt(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=Zt(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function Wm(e,t,n){switch(t.tag){case 3:_f(t),Jn();break;case 5:tf(t);break;case 1:Oe(t.type)&&el(t);break;case 4:Za(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;re(rl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(re(ue,ue.current&1),t.flags|=128,null):n&t.child.childLanes?zf(e,t,n):(re(ue,ue.current&1),e=Mt(e,t,n),e!==null?e.sibling:null);re(ue,ue.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Tf(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),re(ue,ue.current),r)break;return null;case 22:case 23:return t.lanes=0,Ef(e,t,n)}return Mt(e,t,n)}var Lf,ia,Pf,If;Lf=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};ia=function(){};Pf=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,hn(St.current);var l=null;switch(n){case"input":i=Eo(e,i),r=Eo(e,r),l=[];break;case"select":i=de({},i,{value:void 0}),r=de({},r,{value:void 0}),l=[];break;case"textarea":i=zo(e,i),r=zo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=Ji)}Lo(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(Ar.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var s=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&s!==a&&(s!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(c,n)),n=s;else c==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(c,s)):c==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(c,""+s):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(Ar.hasOwnProperty(c)?(s!=null&&c==="onScroll"&&oe("scroll",e),l||a===s||(l=[])):(l=l||[]).push(c,s))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};If=function(e,t,n,r){n!==r&&(t.flags|=4)};function gr(e,t){if(!se)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function Ee(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function Qm(e,t,n){var r=t.pendingProps;switch(Qa(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return Ee(t),null;case 1:return Oe(t.type)&&Zi(),Ee(t),null;case 3:return r=t.stateNode,er(),ae(Re),ae(_e),ts(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(xi(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,ct!==null&&(fa(ct),ct=null))),ia(e,t),Ee(t),null;case 5:es(t);var i=hn(qr.current);if(n=t.type,e!==null&&t.stateNode!=null)Pf(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(M(166));return Ee(t),null}if(e=hn(St.current),xi(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[xt]=t,r[Qr]=l,e=(t.mode&1)!==0,n){case"dialog":oe("cancel",r),oe("close",r);break;case"iframe":case"object":case"embed":oe("load",r);break;case"video":case"audio":for(i=0;i<br.length;i++)oe(br[i],r);break;case"source":oe("error",r);break;case"img":case"image":case"link":oe("error",r),oe("load",r);break;case"details":oe("toggle",r);break;case"input":Ws(r,l),oe("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},oe("invalid",r);break;case"textarea":Ks(r,l),oe("invalid",r)}Lo(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&yi(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&yi(r.textContent,a,e),i=["children",""+a]):Ar.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&oe("scroll",r)}switch(n){case"input":ci(r),Qs(r,l,!0);break;case"textarea":ci(r),qs(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=Ji)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=od(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[xt]=t,e[Qr]=r,Lf(e,t,!1,!1),t.stateNode=e;e:{switch(o=Po(n,r),n){case"dialog":oe("cancel",e),oe("close",e),i=r;break;case"iframe":case"object":case"embed":oe("load",e),i=r;break;case"video":case"audio":for(i=0;i<br.length;i++)oe(br[i],e);i=r;break;case"source":oe("error",e),i=r;break;case"img":case"image":case"link":oe("error",e),oe("load",e),i=r;break;case"details":oe("toggle",e),i=r;break;case"input":Ws(e,r),i=Eo(e,r),oe("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=de({},r,{value:void 0}),oe("invalid",e);break;case"textarea":Ks(e,r),i=zo(e,r),oe("invalid",e);break;default:i=r}Lo(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?ud(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&ad(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Rr(e,s):typeof s=="number"&&Rr(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Ar.hasOwnProperty(l)?s!=null&&l==="onScroll"&&oe("scroll",e):s!=null&&La(e,l,s,o))}switch(n){case"input":ci(e),Qs(e,r,!1);break;case"textarea":ci(e),qs(e);break;case"option":r.value!=null&&e.setAttribute("value",""+en(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?$n(e,!!r.multiple,l,!1):r.defaultValue!=null&&$n(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=Ji)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return Ee(t),null;case 6:if(e&&t.stateNode!=null)If(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(M(166));if(n=hn(qr.current),hn(St.current),xi(t)){if(r=t.stateNode,n=t.memoizedProps,r[xt]=t,(l=r.nodeValue!==n)&&(e=qe,e!==null))switch(e.tag){case 3:yi(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&yi(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[xt]=t,t.stateNode=r}return Ee(t),null;case 13:if(ae(ue),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(se&&Qe!==null&&t.mode&1&&!(t.flags&128))Xd(),Jn(),t.flags|=98560,l=!1;else if(l=xi(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(M(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(M(317));l[xt]=t}else Jn(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;Ee(t),l=!1}else ct!==null&&(fa(ct),ct=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||ue.current&1?xe===0&&(xe=3):ps())),t.updateQueue!==null&&(t.flags|=4),Ee(t),null);case 4:return er(),ia(e,t),e===null&&Vr(t.stateNode.containerInfo),Ee(t),null;case 10:return Xa(t.type._context),Ee(t),null;case 17:return Oe(t.type)&&Zi(),Ee(t),null;case 19:if(ae(ue),l=t.memoizedState,l===null)return Ee(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)gr(l,!1);else{if(xe!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=ol(e),o!==null){for(t.flags|=128,gr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return re(ue,ue.current&1|2),t.child}e=e.sibling}l.tail!==null&&pe()>nr&&(t.flags|=128,r=!0,gr(l,!1),t.lanes=4194304)}else{if(!r)if(e=ol(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),gr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!se)return Ee(t),null}else 2*pe()-l.renderingStartTime>nr&&n!==1073741824&&(t.flags|=128,r=!0,gr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=pe(),t.sibling=null,n=ue.current,re(ue,r?n&1|2:n&1),t):(Ee(t),null);case 22:case 23:return fs(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?We&1073741824&&(Ee(t),t.subtreeFlags&6&&(t.flags|=8192)):Ee(t),null;case 24:return null;case 25:return null}throw Error(M(156,t.tag))}function Km(e,t){switch(Qa(t),t.tag){case 1:return Oe(t.type)&&Zi(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return er(),ae(Re),ae(_e),ts(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return es(t),null;case 13:if(ae(ue),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(M(340));Jn()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return ae(ue),null;case 4:return er(),null;case 10:return Xa(t.type._context),null;case 22:case 23:return fs(),null;case 24:return null;default:return null}}var Si=!1,Ne=!1,qm=typeof WeakSet=="function"?WeakSet:Set,U=null;function Bn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){fe(e,t,r)}else n.current=null}function la(e,t,n){try{n()}catch(r){fe(e,t,r)}}var Ru=!1;function Ym(e,t){if($o=Yi,e=Rd(),Va(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,c=0,d=0,f=e,h=null;t:for(;;){for(var p;f!==n||i!==0&&f.nodeType!==3||(a=o+i),f!==l||r!==0&&f.nodeType!==3||(s=o+r),f.nodeType===3&&(o+=f.nodeValue.length),(p=f.firstChild)!==null;)h=f,f=p;for(;;){if(f===e)break t;if(h===n&&++c===i&&(a=o),h===l&&++d===r&&(s=o),(p=f.nextSibling)!==null)break;f=h,h=f.parentNode}f=p}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Ho={focusedElem:e,selectionRange:n},Yi=!1,U=t;U!==null;)if(t=U,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,U=e;else for(;U!==null;){t=U;try{var k=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(k!==null){var S=k.memoizedProps,C=k.memoizedState,g=t.stateNode,m=g.getSnapshotBeforeUpdate(t.elementType===t.type?S:st(t.type,S),C);g.__reactInternalSnapshotBeforeUpdate=m}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(M(163))}}catch(j){fe(t,t.return,j)}if(e=t.sibling,e!==null){e.return=t.return,U=e;break}U=t.return}return k=Ru,Ru=!1,k}function Tr(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&la(t,n,l)}i=i.next}while(i!==r)}}function Cl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function oa(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Df(e){var t=e.alternate;t!==null&&(e.alternate=null,Df(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[xt],delete t[Qr],delete t[Qo],delete t[Lm],delete t[Pm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Mf(e){return e.tag===5||e.tag===3||e.tag===4}function Ou(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Mf(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function aa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=Ji));else if(r!==4&&(e=e.child,e!==null))for(aa(e,t,n),e=e.sibling;e!==null;)aa(e,t,n),e=e.sibling}function sa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(sa(e,t,n),e=e.sibling;e!==null;)sa(e,t,n),e=e.sibling}var Se=null,ut=!1;function Ot(e,t,n){for(n=n.child;n!==null;)Af(e,t,n),n=n.sibling}function Af(e,t,n){if(wt&&typeof wt.onCommitFiberUnmount=="function")try{wt.onCommitFiberUnmount(vl,n)}catch{}switch(n.tag){case 5:Ne||Bn(n,t);case 6:var r=Se,i=ut;Se=null,Ot(e,t,n),Se=r,ut=i,Se!==null&&(ut?(e=Se,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):Se.removeChild(n.stateNode));break;case 18:Se!==null&&(ut?(e=Se,n=n.stateNode,e.nodeType===8?Jl(e.parentNode,n):e.nodeType===1&&Jl(e,n),Ur(e)):Jl(Se,n.stateNode));break;case 4:r=Se,i=ut,Se=n.stateNode.containerInfo,ut=!0,Ot(e,t,n),Se=r,ut=i;break;case 0:case 11:case 14:case 15:if(!Ne&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&la(n,t,o),i=i.next}while(i!==r)}Ot(e,t,n);break;case 1:if(!Ne&&(Bn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){fe(n,t,a)}Ot(e,t,n);break;case 21:Ot(e,t,n);break;case 22:n.mode&1?(Ne=(r=Ne)||n.memoizedState!==null,Ot(e,t,n),Ne=r):Ot(e,t,n);break;default:Ot(e,t,n)}}function Fu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new qm),t.forEach(function(r){var i=ig.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function at(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:Se=a.stateNode,ut=!1;break e;case 3:Se=a.stateNode.containerInfo,ut=!0;break e;case 4:Se=a.stateNode.containerInfo,ut=!0;break e}a=a.return}if(Se===null)throw Error(M(160));Af(l,o,i),Se=null,ut=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(c){fe(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Rf(t,e),t=t.sibling}function Rf(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(at(t,e),mt(e),r&4){try{Tr(3,e,e.return),Cl(3,e)}catch(S){fe(e,e.return,S)}try{Tr(5,e,e.return)}catch(S){fe(e,e.return,S)}}break;case 1:at(t,e),mt(e),r&512&&n!==null&&Bn(n,n.return);break;case 5:if(at(t,e),mt(e),r&512&&n!==null&&Bn(n,n.return),e.flags&32){var i=e.stateNode;try{Rr(i,"")}catch(S){fe(e,e.return,S)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&id(i,l),Po(a,o);var c=Po(a,l);for(o=0;o<s.length;o+=2){var d=s[o],f=s[o+1];d==="style"?ud(i,f):d==="dangerouslySetInnerHTML"?ad(i,f):d==="children"?Rr(i,f):La(i,d,f,c)}switch(a){case"input":No(i,l);break;case"textarea":ld(i,l);break;case"select":var h=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var p=l.value;p!=null?$n(i,!!l.multiple,p,!1):h!==!!l.multiple&&(l.defaultValue!=null?$n(i,!!l.multiple,l.defaultValue,!0):$n(i,!!l.multiple,l.multiple?[]:"",!1))}i[Qr]=l}catch(S){fe(e,e.return,S)}}break;case 6:if(at(t,e),mt(e),r&4){if(e.stateNode===null)throw Error(M(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(S){fe(e,e.return,S)}}break;case 3:if(at(t,e),mt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Ur(t.containerInfo)}catch(S){fe(e,e.return,S)}break;case 4:at(t,e),mt(e);break;case 13:at(t,e),mt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(cs=pe())),r&4&&Fu(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Ne=(c=Ne)||d,at(t,e),Ne=c):at(t,e),mt(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for(U=e,d=e.child;d!==null;){for(f=U=d;U!==null;){switch(h=U,p=h.child,h.tag){case 0:case 11:case 14:case 15:Tr(4,h,h.return);break;case 1:Bn(h,h.return);var k=h.stateNode;if(typeof k.componentWillUnmount=="function"){r=h,n=h.return;try{t=r,k.props=t.memoizedProps,k.state=t.memoizedState,k.componentWillUnmount()}catch(S){fe(r,n,S)}}break;case 5:Bn(h,h.return);break;case 22:if(h.memoizedState!==null){Uu(f);continue}}p!==null?(p.return=h,U=p):Uu(f)}d=d.sibling}e:for(d=null,f=e;;){if(f.tag===5){if(d===null){d=f;try{i=f.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=f.stateNode,s=f.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=sd("display",o))}catch(S){fe(e,e.return,S)}}}else if(f.tag===6){if(d===null)try{f.stateNode.nodeValue=c?"":f.memoizedProps}catch(S){fe(e,e.return,S)}}else if((f.tag!==22&&f.tag!==23||f.memoizedState===null||f===e)&&f.child!==null){f.child.return=f,f=f.child;continue}if(f===e)break e;for(;f.sibling===null;){if(f.return===null||f.return===e)break e;d===f&&(d=null),f=f.return}d===f&&(d=null),f.sibling.return=f.return,f=f.sibling}}break;case 19:at(t,e),mt(e),r&4&&Fu(e);break;case 21:break;default:at(t,e),mt(e)}}function mt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Mf(n)){var r=n;break e}n=n.return}throw Error(M(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Rr(i,""),r.flags&=-33);var l=Ou(e);sa(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Ou(e);aa(e,a,o);break;default:throw Error(M(161))}}catch(s){fe(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function Xm(e,t,n){U=e,Of(e)}function Of(e,t,n){for(var r=(e.mode&1)!==0;U!==null;){var i=U,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||Si;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||Ne;a=Si;var c=Ne;if(Si=o,(Ne=s)&&!c)for(U=i;U!==null;)o=U,s=o.child,o.tag===22&&o.memoizedState!==null?$u(i):s!==null?(s.return=o,U=s):$u(i);for(;l!==null;)U=l,Of(l),l=l.sibling;U=i,Si=a,Ne=c}Bu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,U=l):Bu(e)}}function Bu(e){for(;U!==null;){var t=U;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Ne||Cl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Ne)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:st(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&ju(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}ju(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var f=d.dehydrated;f!==null&&Ur(f)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(M(163))}Ne||t.flags&512&&oa(t)}catch(h){fe(t,t.return,h)}}if(t===e){U=null;break}if(n=t.sibling,n!==null){n.return=t.return,U=n;break}U=t.return}}function Uu(e){for(;U!==null;){var t=U;if(t===e){U=null;break}var n=t.sibling;if(n!==null){n.return=t.return,U=n;break}U=t.return}}function $u(e){for(;U!==null;){var t=U;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Cl(4,t)}catch(s){fe(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){fe(t,i,s)}}var l=t.return;try{oa(t)}catch(s){fe(t,l,s)}break;case 5:var o=t.return;try{oa(t)}catch(s){fe(t,o,s)}}}catch(s){fe(t,t.return,s)}if(t===e){U=null;break}var a=t.sibling;if(a!==null){a.return=t.return,U=a;break}U=t.return}}var Gm=Math.ceil,ul=At.ReactCurrentDispatcher,ss=At.ReactCurrentOwner,nt=At.ReactCurrentBatchConfig,G=0,we=null,ge=null,be=0,We=0,Un=rn(0),xe=0,Jr=null,kn=0,El=0,us=0,Lr=null,Me=null,cs=0,nr=1/0,Nt=null,cl=!1,ua=null,Gt=null,bi=!1,Wt=null,dl=0,Pr=0,ca=null,Oi=-1,Fi=0;function Le(){return G&6?pe():Oi!==-1?Oi:Oi=pe()}function Jt(e){return e.mode&1?G&2&&be!==0?be&-be:Dm.transition!==null?(Fi===0&&(Fi=wd()),Fi):(e=ee,e!==0||(e=window.event,e=e===void 0?16:_d(e.type)),e):1}function ft(e,t,n,r){if(50<Pr)throw Pr=0,ca=null,Error(M(185));ti(e,n,r),(!(G&2)||e!==we)&&(e===we&&(!(G&2)&&(El|=n),xe===4&&Ht(e,be)),Fe(e,r),n===1&&G===0&&!(t.mode&1)&&(nr=pe()+500,Sl&&ln()))}function Fe(e,t){var n=e.callbackNode;Dh(e,t);var r=qi(e,e===we?be:0);if(r===0)n!==null&&Gs(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&Gs(n),t===1)e.tag===0?Im(Hu.bind(null,e)):Kd(Hu.bind(null,e)),zm(function(){!(G&6)&&ln()}),n=null;else{switch(Sd(r)){case 1:n=Aa;break;case 4:n=xd;break;case 16:n=Ki;break;case 536870912:n=kd;break;default:n=Ki}n=Qf(n,Ff.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function Ff(e,t){if(Oi=-1,Fi=0,G&6)throw Error(M(327));var n=e.callbackNode;if(Kn()&&e.callbackNode!==n)return null;var r=qi(e,e===we?be:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=fl(e,r);else{t=r;var i=G;G|=2;var l=Uf();(we!==e||be!==t)&&(Nt=null,nr=pe()+500,mn(e,t));do try{eg();break}catch(a){Bf(e,a)}while(!0);Ya(),ul.current=l,G=i,ge!==null?t=0:(we=null,be=0,t=xe)}if(t!==0){if(t===2&&(i=Ro(e),i!==0&&(r=i,t=da(e,i))),t===1)throw n=Jr,mn(e,0),Ht(e,r),Fe(e,pe()),n;if(t===6)Ht(e,r);else{if(i=e.current.alternate,!(r&30)&&!Jm(i)&&(t=fl(e,r),t===2&&(l=Ro(e),l!==0&&(r=l,t=da(e,l))),t===1))throw n=Jr,mn(e,0),Ht(e,r),Fe(e,pe()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(M(345));case 2:un(e,Me,Nt);break;case 3:if(Ht(e,r),(r&130023424)===r&&(t=cs+500-pe(),10<t)){if(qi(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Le(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=Wo(un.bind(null,e,Me,Nt),t);break}un(e,Me,Nt);break;case 4:if(Ht(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-dt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=pe()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*Gm(r/1960))-r,10<r){e.timeoutHandle=Wo(un.bind(null,e,Me,Nt),r);break}un(e,Me,Nt);break;case 5:un(e,Me,Nt);break;default:throw Error(M(329))}}}return Fe(e,pe()),e.callbackNode===n?Ff.bind(null,e):null}function da(e,t){var n=Lr;return e.current.memoizedState.isDehydrated&&(mn(e,t).flags|=256),e=fl(e,t),e!==2&&(t=Me,Me=n,t!==null&&fa(t)),e}function fa(e){Me===null?Me=e:Me.push.apply(Me,e)}function Jm(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!pt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Ht(e,t){for(t&=~us,t&=~El,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-dt(t),r=1<<n;e[n]=-1,t&=~r}}function Hu(e){if(G&6)throw Error(M(327));Kn();var t=qi(e,0);if(!(t&1))return Fe(e,pe()),null;var n=fl(e,t);if(e.tag!==0&&n===2){var r=Ro(e);r!==0&&(t=r,n=da(e,r))}if(n===1)throw n=Jr,mn(e,0),Ht(e,t),Fe(e,pe()),n;if(n===6)throw Error(M(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,un(e,Me,Nt),Fe(e,pe()),null}function ds(e,t){var n=G;G|=1;try{return e(t)}finally{G=n,G===0&&(nr=pe()+500,Sl&&ln())}}function wn(e){Wt!==null&&Wt.tag===0&&!(G&6)&&Kn();var t=G;G|=1;var n=nt.transition,r=ee;try{if(nt.transition=null,ee=1,e)return e()}finally{ee=r,nt.transition=n,G=t,!(G&6)&&ln()}}function fs(){We=Un.current,ae(Un)}function mn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,_m(n)),ge!==null)for(n=ge.return;n!==null;){var r=n;switch(Qa(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&Zi();break;case 3:er(),ae(Re),ae(_e),ts();break;case 5:es(r);break;case 4:er();break;case 13:ae(ue);break;case 19:ae(ue);break;case 10:Xa(r.type._context);break;case 22:case 23:fs()}n=n.return}if(we=e,ge=e=Zt(e.current,null),be=We=t,xe=0,Jr=null,us=El=kn=0,Me=Lr=null,pn!==null){for(t=0;t<pn.length;t++)if(n=pn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}pn=null}return e}function Bf(e,t){do{var n=ge;try{if(Ya(),Mi.current=sl,al){for(var r=ce.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}al=!1}if(xn=0,ke=ye=ce=null,zr=!1,Yr=0,ss.current=null,n===null||n.return===null){xe=1,Jr=t,ge=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=be,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var c=s,d=a,f=d.tag;if(!(d.mode&1)&&(f===0||f===11||f===15)){var h=d.alternate;h?(d.updateQueue=h.updateQueue,d.memoizedState=h.memoizedState,d.lanes=h.lanes):(d.updateQueue=null,d.memoizedState=null)}var p=Tu(o);if(p!==null){p.flags&=-257,Lu(p,o,a,l,t),p.mode&1&&zu(l,c,t),t=p,s=c;var k=t.updateQueue;if(k===null){var S=new Set;S.add(s),t.updateQueue=S}else k.add(s);break e}else{if(!(t&1)){zu(l,c,t),ps();break e}s=Error(M(426))}}else if(se&&a.mode&1){var C=Tu(o);if(C!==null){!(C.flags&65536)&&(C.flags|=256),Lu(C,o,a,l,t),Ka(tr(s,a));break e}}l=s=tr(s,a),xe!==4&&(xe=2),Lr===null?Lr=[l]:Lr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var g=bf(l,s,t);bu(l,g);break e;case 1:a=s;var m=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof m.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(Gt===null||!Gt.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var j=jf(l,a,t);bu(l,j);break e}}l=l.return}while(l!==null)}Hf(n)}catch(z){t=z,ge===n&&n!==null&&(ge=n=n.return);continue}break}while(!0)}function Uf(){var e=ul.current;return ul.current=sl,e===null?sl:e}function ps(){(xe===0||xe===3||xe===2)&&(xe=4),we===null||!(kn&268435455)&&!(El&268435455)||Ht(we,be)}function fl(e,t){var n=G;G|=2;var r=Uf();(we!==e||be!==t)&&(Nt=null,mn(e,t));do try{Zm();break}catch(i){Bf(e,i)}while(!0);if(Ya(),G=n,ul.current=r,ge!==null)throw Error(M(261));return we=null,be=0,xe}function Zm(){for(;ge!==null;)$f(ge)}function eg(){for(;ge!==null&&!Ch();)$f(ge)}function $f(e){var t=Wf(e.alternate,e,We);e.memoizedProps=e.pendingProps,t===null?Hf(e):ge=t,ss.current=null}function Hf(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=Km(n,t),n!==null){n.flags&=32767,ge=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{xe=6,ge=null;return}}else if(n=Qm(n,t,We),n!==null){ge=n;return}if(t=t.sibling,t!==null){ge=t;return}ge=t=e}while(t!==null);xe===0&&(xe=5)}function un(e,t,n){var r=ee,i=nt.transition;try{nt.transition=null,ee=1,tg(e,t,n,r)}finally{nt.transition=i,ee=r}return null}function tg(e,t,n,r){do Kn();while(Wt!==null);if(G&6)throw Error(M(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(M(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Mh(e,l),e===we&&(ge=we=null,be=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||bi||(bi=!0,Qf(Ki,function(){return Kn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=nt.transition,nt.transition=null;var o=ee;ee=1;var a=G;G|=4,ss.current=null,Ym(e,n),Rf(n,e),wm(Ho),Yi=!!$o,Ho=$o=null,e.current=n,Xm(n),Eh(),G=a,ee=o,nt.transition=l}else e.current=n;if(bi&&(bi=!1,Wt=e,dl=i),l=e.pendingLanes,l===0&&(Gt=null),zh(n.stateNode),Fe(e,pe()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(cl)throw cl=!1,e=ua,ua=null,e;return dl&1&&e.tag!==0&&Kn(),l=e.pendingLanes,l&1?e===ca?Pr++:(Pr=0,ca=e):Pr=0,ln(),null}function Kn(){if(Wt!==null){var e=Sd(dl),t=nt.transition,n=ee;try{if(nt.transition=null,ee=16>e?16:e,Wt===null)var r=!1;else{if(e=Wt,Wt=null,dl=0,G&6)throw Error(M(331));var i=G;for(G|=4,U=e.current;U!==null;){var l=U,o=l.child;if(U.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var c=a[s];for(U=c;U!==null;){var d=U;switch(d.tag){case 0:case 11:case 15:Tr(8,d,l)}var f=d.child;if(f!==null)f.return=d,U=f;else for(;U!==null;){d=U;var h=d.sibling,p=d.return;if(Df(d),d===c){U=null;break}if(h!==null){h.return=p,U=h;break}U=p}}}var k=l.alternate;if(k!==null){var S=k.child;if(S!==null){k.child=null;do{var C=S.sibling;S.sibling=null,S=C}while(S!==null)}}U=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,U=o;else e:for(;U!==null;){if(l=U,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Tr(9,l,l.return)}var g=l.sibling;if(g!==null){g.return=l.return,U=g;break e}U=l.return}}var m=e.current;for(U=m;U!==null;){o=U;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,U=y;else e:for(o=m;U!==null;){if(a=U,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Cl(9,a)}}catch(z){fe(a,a.return,z)}if(a===o){U=null;break e}var j=a.sibling;if(j!==null){j.return=a.return,U=j;break e}U=a.return}}if(G=i,ln(),wt&&typeof wt.onPostCommitFiberRoot=="function")try{wt.onPostCommitFiberRoot(vl,e)}catch{}r=!0}return r}finally{ee=n,nt.transition=t}}return!1}function Vu(e,t,n){t=tr(n,t),t=bf(e,t,1),e=Xt(e,t,1),t=Le(),e!==null&&(ti(e,1,t),Fe(e,t))}function fe(e,t,n){if(e.tag===3)Vu(e,e,n);else for(;t!==null;){if(t.tag===3){Vu(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(Gt===null||!Gt.has(r))){e=tr(n,e),e=jf(t,e,1),t=Xt(t,e,1),e=Le(),t!==null&&(ti(t,1,e),Fe(t,e));break}}t=t.return}}function ng(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Le(),e.pingedLanes|=e.suspendedLanes&n,we===e&&(be&n)===n&&(xe===4||xe===3&&(be&130023424)===be&&500>pe()-cs?mn(e,0):us|=n),Fe(e,t)}function Vf(e,t){t===0&&(e.mode&1?(t=pi,pi<<=1,!(pi&130023424)&&(pi=4194304)):t=1);var n=Le();e=Dt(e,t),e!==null&&(ti(e,t,n),Fe(e,n))}function rg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),Vf(e,n)}function ig(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(M(314))}r!==null&&r.delete(t),Vf(e,n)}var Wf;Wf=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Re.current)Ae=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return Ae=!1,Wm(e,t,n);Ae=!!(e.flags&131072)}else Ae=!1,se&&t.flags&1048576&&qd(t,nl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Ri(e,t),e=t.pendingProps;var i=Gn(t,_e.current);Qn(t,n),i=rs(null,t,r,e,i,n);var l=is();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Oe(r)?(l=!0,el(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,Ja(t),i.updater=jl,t.stateNode=i,i._reactInternals=t,Jo(t,r,e,n),t=ta(null,t,r,!0,l,n)):(t.tag=0,se&&l&&Wa(t),Te(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Ri(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=og(r),e=st(r,e),i){case 0:t=ea(null,t,r,e,n);break e;case 1:t=Du(null,t,r,e,n);break e;case 11:t=Pu(null,t,r,e,n);break e;case 14:t=Iu(null,t,r,st(r.type,e),n);break e}throw Error(M(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:st(r,i),ea(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:st(r,i),Du(e,t,r,i,n);case 3:e:{if(_f(t),e===null)throw Error(M(387));r=t.pendingProps,l=t.memoizedState,i=l.element,ef(e,t),ll(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=tr(Error(M(423)),t),t=Mu(e,t,r,n,i);break e}else if(r!==i){i=tr(Error(M(424)),t),t=Mu(e,t,r,n,i);break e}else for(Qe=Yt(t.stateNode.containerInfo.firstChild),qe=t,se=!0,ct=null,n=Jd(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(Jn(),r===i){t=Mt(e,t,n);break e}Te(e,t,r,n)}t=t.child}return t;case 5:return tf(t),e===null&&Yo(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,Vo(r,i)?o=null:l!==null&&Vo(r,l)&&(t.flags|=32),Nf(e,t),Te(e,t,o,n),t.child;case 6:return e===null&&Yo(t),null;case 13:return zf(e,t,n);case 4:return Za(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=Zn(t,null,r,n):Te(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:st(r,i),Pu(e,t,r,i,n);case 7:return Te(e,t,t.pendingProps,n),t.child;case 8:return Te(e,t,t.pendingProps.children,n),t.child;case 12:return Te(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,re(rl,r._currentValue),r._currentValue=o,l!==null)if(pt(l.value,o)){if(l.children===i.children&&!Re.current){t=Mt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Lt(-1,n&-n),s.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?s.next=s:(s.next=d.next,d.next=s),c.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),Xo(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(M(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),Xo(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}Te(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Qn(t,n),i=rt(i),r=r(i),t.flags|=1,Te(e,t,r,n),t.child;case 14:return r=t.type,i=st(r,t.pendingProps),i=st(r.type,i),Iu(e,t,r,i,n);case 15:return Cf(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:st(r,i),Ri(e,t),t.tag=1,Oe(r)?(e=!0,el(t)):e=!1,Qn(t,n),Sf(t,r,i),Jo(t,r,i,n),ta(null,t,r,!0,e,n);case 19:return Tf(e,t,n);case 22:return Ef(e,t,n)}throw Error(M(156,t.tag))};function Qf(e,t){return yd(e,t)}function lg(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function tt(e,t,n,r){return new lg(e,t,n,r)}function hs(e){return e=e.prototype,!(!e||!e.isReactComponent)}function og(e){if(typeof e=="function")return hs(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Ia)return 11;if(e===Da)return 14}return 2}function Zt(e,t){var n=e.alternate;return n===null?(n=tt(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Bi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")hs(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Ln:return gn(n.children,i,l,t);case Pa:o=8,i|=8;break;case So:return e=tt(12,n,t,i|2),e.elementType=So,e.lanes=l,e;case bo:return e=tt(13,n,t,i),e.elementType=bo,e.lanes=l,e;case jo:return e=tt(19,n,t,i),e.elementType=jo,e.lanes=l,e;case td:return Nl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case Zc:o=10;break e;case ed:o=9;break e;case Ia:o=11;break e;case Da:o=14;break e;case Bt:o=16,r=null;break e}throw Error(M(130,e==null?e:typeof e,""))}return t=tt(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function gn(e,t,n,r){return e=tt(7,e,r,t),e.lanes=n,e}function Nl(e,t,n,r){return e=tt(22,e,r,t),e.elementType=td,e.lanes=n,e.stateNode={isHidden:!1},e}function oo(e,t,n){return e=tt(6,e,null,t),e.lanes=n,e}function ao(e,t,n){return t=tt(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function ag(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Ul(0),this.expirationTimes=Ul(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Ul(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function ms(e,t,n,r,i,l,o,a,s){return e=new ag(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=tt(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},Ja(l),e}function sg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:Tn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function Kf(e){if(!e)return tn;e=e._reactInternals;e:{if(bn(e)!==e||e.tag!==1)throw Error(M(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Oe(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(M(171))}if(e.tag===1){var n=e.type;if(Oe(n))return Qd(e,n,t)}return t}function qf(e,t,n,r,i,l,o,a,s){return e=ms(n,r,!0,e,i,l,o,a,s),e.context=Kf(null),n=e.current,r=Le(),i=Jt(n),l=Lt(r,i),l.callback=t??null,Xt(n,l,i),e.current.lanes=i,ti(e,i,r),Fe(e,r),e}function _l(e,t,n,r){var i=t.current,l=Le(),o=Jt(i);return n=Kf(n),t.context===null?t.context=n:t.pendingContext=n,t=Lt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=Xt(i,t,o),e!==null&&(ft(e,i,o,l),Di(e,i,o)),o}function pl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function Wu(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function gs(e,t){Wu(e,t),(e=e.alternate)&&Wu(e,t)}function ug(){return null}var Yf=typeof reportError=="function"?reportError:function(e){console.error(e)};function vs(e){this._internalRoot=e}zl.prototype.render=vs.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(M(409));_l(e,t,null,null)};zl.prototype.unmount=vs.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;wn(function(){_l(null,e,null,null)}),t[It]=null}};function zl(e){this._internalRoot=e}zl.prototype.unstable_scheduleHydration=function(e){if(e){var t=Cd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<$t.length&&t!==0&&t<$t[n].priority;n++);$t.splice(n,0,e),n===0&&Nd(e)}};function ys(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Tl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function Qu(){}function cg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=pl(o);l.call(c)}}var o=qf(t,r,e,0,null,!1,!1,"",Qu);return e._reactRootContainer=o,e[It]=o.current,Vr(e.nodeType===8?e.parentNode:e),wn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=pl(s);a.call(c)}}var s=ms(e,0,!1,null,null,!1,!1,"",Qu);return e._reactRootContainer=s,e[It]=s.current,Vr(e.nodeType===8?e.parentNode:e),wn(function(){_l(t,s,n,r)}),s}function Ll(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=pl(o);a.call(s)}}_l(t,o,e,i)}else o=cg(n,t,e,i,r);return pl(o)}bd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=Sr(t.pendingLanes);n!==0&&(Ra(t,n|1),Fe(t,pe()),!(G&6)&&(nr=pe()+500,ln()))}break;case 13:wn(function(){var r=Dt(e,1);if(r!==null){var i=Le();ft(r,e,1,i)}}),gs(e,1)}};Oa=function(e){if(e.tag===13){var t=Dt(e,134217728);if(t!==null){var n=Le();ft(t,e,134217728,n)}gs(e,134217728)}};jd=function(e){if(e.tag===13){var t=Jt(e),n=Dt(e,t);if(n!==null){var r=Le();ft(n,e,t,r)}gs(e,t)}};Cd=function(){return ee};Ed=function(e,t){var n=ee;try{return ee=e,t()}finally{ee=n}};Do=function(e,t,n){switch(t){case"input":if(No(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=wl(r);if(!i)throw Error(M(90));rd(r),No(r,i)}}}break;case"textarea":ld(e,n);break;case"select":t=n.value,t!=null&&$n(e,!!n.multiple,t,!1)}};fd=ds;pd=wn;var dg={usingClientEntryPoint:!1,Events:[ri,Mn,wl,cd,dd,ds]},vr={findFiberByHostInstance:fn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},fg={bundleType:vr.bundleType,version:vr.version,rendererPackageName:vr.rendererPackageName,rendererConfig:vr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:At.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=gd(e),e===null?null:e.stateNode},findFiberByHostInstance:vr.findFiberByHostInstance||ug,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var ji=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!ji.isDisabled&&ji.supportsFiber)try{vl=ji.inject(fg),wt=ji}catch{}}Xe.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=dg;Xe.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!ys(t))throw Error(M(200));return sg(e,t,null,n)};Xe.createRoot=function(e,t){if(!ys(e))throw Error(M(299));var n=!1,r="",i=Yf;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=ms(e,1,!1,null,null,n,!1,r,i),e[It]=t.current,Vr(e.nodeType===8?e.parentNode:e),new vs(t)};Xe.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(M(188)):(e=Object.keys(e).join(","),Error(M(268,e)));return e=gd(t),e=e===null?null:e.stateNode,e};Xe.flushSync=function(e){return wn(e)};Xe.hydrate=function(e,t,n){if(!Tl(t))throw Error(M(200));return Ll(null,e,t,!0,n)};Xe.hydrateRoot=function(e,t,n){if(!ys(e))throw Error(M(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=Yf;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=qf(t,null,e,1,n??null,i,!1,l,o),e[It]=t.current,Vr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new zl(t)};Xe.render=function(e,t,n){if(!Tl(t))throw Error(M(200));return Ll(null,e,t,!1,n)};Xe.unmountComponentAtNode=function(e){if(!Tl(e))throw Error(M(40));return e._reactRootContainer?(wn(function(){Ll(null,null,e,!1,function(){e._reactRootContainer=null,e[It]=null})}),!0):!1};Xe.unstable_batchedUpdates=ds;Xe.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Tl(n))throw Error(M(200));if(e==null||e._reactInternals===void 0)throw Error(M(38));return Ll(e,t,n,!1,r)};Xe.version="18.3.1-next-f1338f8080-20240426";function Xf(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Xf)}catch(e){console.error(e)}}Xf(),Yc.exports=Xe;var pg=Yc.exports,Ku=pg;ko.createRoot=Ku.createRoot,ko.hydrateRoot=Ku.hydrateRoot;const Et={plus:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),u.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"}),u.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),u.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),u.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),u.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),u.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),u.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("polyline",{points:"3 6 5 6 21 6"}),u.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:u.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},hg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=$.useState(!1),[c,d]=$.useState(""),[f,h]=$.useState(null),[p,k]=$.useState(""),[S,C]=$.useState(null),g=()=>{c.trim()&&(r(c.trim()),d(""),s(!1))},m=_=>{_.key==="Enter"&&!_.shiftKey?(_.preventDefault(),g()):_.key==="Escape"&&(s(!1),d(""))},y=(_,I)=>{I.stopPropagation(),h(_.id),k(_.title)},j=_=>{var I;p.trim()&&p.trim()!==((I=e.find(H=>H.id===_))==null?void 0:I.title)&&l(_,p.trim()),h(null),k("")},z=()=>{h(null),k("")},w=(_,I)=>{_.key==="Enter"?(_.preventDefault(),j(I)):_.key==="Escape"&&z()},E=(_,I)=>{I.stopPropagation(),C(_)},b=(_,I)=>{I.stopPropagation(),i(_),C(null)},N=_=>{_.stopPropagation(),C(null)},A=_=>{const I=new Date(_),K=new Date().getTime()-I.getTime(),L=Math.floor(K/6e4),D=Math.floor(K/36e5),F=Math.floor(K/864e5);return L<1?"now":L<60?`${L}m`:D<24?`${D}h`:F<7?`${F}d`:I.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return u.jsxs("div",{className:"thread-list",children:[u.jsxs("div",{className:"list-header",children:[u.jsx("h2",{children:"Conversations"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:Et.plus})]}),a&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:c,onChange:_=>d(_.target.value),onKeyDown:m,placeholder:"Conversation title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:g,children:"Create"})]})]}),u.jsx("div",{className:"thread-items",children:e.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Et.hash}),u.jsx("p",{children:"No conversations yet"}),u.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(_=>{const I=o.get(_.id)||0,H=_.id===t,K=f===_.id,L=S===_.id;return u.jsxs("div",{className:`thread-item ${H?"selected":""} ${I>0?"has-unread":""}`,onClick:()=>!K&&n(_.id),children:[u.jsx("div",{className:`status-dot ${_.status}`}),u.jsxs("div",{className:"thread-content",children:[u.jsx("div",{className:"thread-title-row",children:K?u.jsxs("div",{className:"edit-title-form",onClick:D=>D.stopPropagation(),children:[u.jsx("input",{type:"text",value:p,onChange:D=>k(D.target.value),onKeyDown:D=>w(D,_.id),autoFocus:!0}),u.jsx("button",{className:"edit-action save",onClick:()=>j(_.id),title:"Save",children:Et.check}),u.jsx("button",{className:"edit-action cancel",onClick:z,title:"Cancel",children:Et.x})]}):u.jsxs(u.Fragment,{children:[u.jsx("span",{className:"thread-title",children:_.title}),u.jsx("span",{className:"thread-time",children:A(_.updated_at)})]})}),u.jsxs("div",{className:"thread-meta",children:[_.target_agent&&u.jsxs("span",{className:"thread-agent",title:`Target: ${_.target_agent}`,children:[Et.bot,_.target_agent]}),u.jsxs("span",{className:"thread-seq",children:["#",_.last_seq]})]})]}),!K&&!L&&u.jsxs("div",{className:"thread-actions",children:[u.jsx("button",{className:"action-btn edit",onClick:D=>y(_,D),title:"Rename",children:Et.edit}),u.jsx("button",{className:"action-btn delete",onClick:D=>E(_.id,D),title:"Delete",children:Et.trash})]}),L&&u.jsxs("div",{className:"delete-confirm",onClick:D=>D.stopPropagation(),children:[u.jsx("span",{className:"confirm-text",children:"Delete?"}),u.jsx("button",{className:"confirm-btn yes",onClick:D=>b(_.id,D),title:"Confirm delete",children:Et.check}),u.jsx("button",{className:"confirm-btn no",onClick:N,title:"Cancel",children:Et.x})]}),I>0&&!L&&u.jsx("span",{className:"unread-badge",children:I})]},_.id)})}),u.jsx("style",{children:`
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
      `})]})};function mg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const gg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,vg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,yg={};function qu(e,t){return(yg.jsx?vg:gg).test(e)}const xg=/[ \t\n\f\r]/g;function kg(e){return typeof e=="object"?e.type==="text"?Yu(e.value):!1:Yu(e)}function Yu(e){return e.replace(xg,"")===""}class li{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}li.prototype.normal={};li.prototype.property={};li.prototype.space=void 0;function Gf(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new li(n,r,t)}function pa(e){return e.toLowerCase()}class Ue{constructor(t,n){this.attribute=n,this.property=t}}Ue.prototype.attribute="";Ue.prototype.booleanish=!1;Ue.prototype.boolean=!1;Ue.prototype.commaOrSpaceSeparated=!1;Ue.prototype.commaSeparated=!1;Ue.prototype.defined=!1;Ue.prototype.mustUseProperty=!1;Ue.prototype.number=!1;Ue.prototype.overloadedBoolean=!1;Ue.prototype.property="";Ue.prototype.spaceSeparated=!1;Ue.prototype.space=void 0;let wg=0;const Q=jn(),me=jn(),ha=jn(),R=jn(),ne=jn(),qn=jn(),Ve=jn();function jn(){return 2**++wg}const ma=Object.freeze(Object.defineProperty({__proto__:null,boolean:Q,booleanish:me,commaOrSpaceSeparated:Ve,commaSeparated:qn,number:R,overloadedBoolean:ha,spaceSeparated:ne},Symbol.toStringTag,{value:"Module"})),so=Object.keys(ma);class xs extends Ue{constructor(t,n,r,i){let l=-1;if(super(t,n),Xu(this,"space",i),typeof r=="number")for(;++l<so.length;){const o=so[l];Xu(this,so[l],(r&ma[o])===ma[o])}}}xs.prototype.defined=!0;function Xu(e,t,n){n&&(e[t]=n)}function or(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new xs(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[pa(r)]=r,n[pa(l.attribute)]=r}return new li(t,n,e.space)}const Jf=or({properties:{ariaActiveDescendant:null,ariaAtomic:me,ariaAutoComplete:null,ariaBusy:me,ariaChecked:me,ariaColCount:R,ariaColIndex:R,ariaColSpan:R,ariaControls:ne,ariaCurrent:null,ariaDescribedBy:ne,ariaDetails:null,ariaDisabled:me,ariaDropEffect:ne,ariaErrorMessage:null,ariaExpanded:me,ariaFlowTo:ne,ariaGrabbed:me,ariaHasPopup:null,ariaHidden:me,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:ne,ariaLevel:R,ariaLive:null,ariaModal:me,ariaMultiLine:me,ariaMultiSelectable:me,ariaOrientation:null,ariaOwns:ne,ariaPlaceholder:null,ariaPosInSet:R,ariaPressed:me,ariaReadOnly:me,ariaRelevant:null,ariaRequired:me,ariaRoleDescription:ne,ariaRowCount:R,ariaRowIndex:R,ariaRowSpan:R,ariaSelected:me,ariaSetSize:R,ariaSort:null,ariaValueMax:R,ariaValueMin:R,ariaValueNow:R,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function Zf(e,t){return t in e?e[t]:t}function ep(e,t){return Zf(e,t.toLowerCase())}const Sg=or({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:qn,acceptCharset:ne,accessKey:ne,action:null,allow:null,allowFullScreen:Q,allowPaymentRequest:Q,allowUserMedia:Q,alt:null,as:null,async:Q,autoCapitalize:null,autoComplete:ne,autoFocus:Q,autoPlay:Q,blocking:ne,capture:null,charSet:null,checked:Q,cite:null,className:ne,cols:R,colSpan:null,content:null,contentEditable:me,controls:Q,controlsList:ne,coords:R|qn,crossOrigin:null,data:null,dateTime:null,decoding:null,default:Q,defer:Q,dir:null,dirName:null,disabled:Q,download:ha,draggable:me,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:Q,formTarget:null,headers:ne,height:R,hidden:ha,high:R,href:null,hrefLang:null,htmlFor:ne,httpEquiv:ne,id:null,imageSizes:null,imageSrcSet:null,inert:Q,inputMode:null,integrity:null,is:null,isMap:Q,itemId:null,itemProp:ne,itemRef:ne,itemScope:Q,itemType:ne,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:Q,low:R,manifest:null,max:null,maxLength:R,media:null,method:null,min:null,minLength:R,multiple:Q,muted:Q,name:null,nonce:null,noModule:Q,noValidate:Q,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:Q,optimum:R,pattern:null,ping:ne,placeholder:null,playsInline:Q,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:Q,referrerPolicy:null,rel:ne,required:Q,reversed:Q,rows:R,rowSpan:R,sandbox:ne,scope:null,scoped:Q,seamless:Q,selected:Q,shadowRootClonable:Q,shadowRootDelegatesFocus:Q,shadowRootMode:null,shape:null,size:R,sizes:null,slot:null,span:R,spellCheck:me,src:null,srcDoc:null,srcLang:null,srcSet:null,start:R,step:null,style:null,tabIndex:R,target:null,title:null,translate:null,type:null,typeMustMatch:Q,useMap:null,value:me,width:R,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:ne,axis:null,background:null,bgColor:null,border:R,borderColor:null,bottomMargin:R,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:Q,declare:Q,event:null,face:null,frame:null,frameBorder:null,hSpace:R,leftMargin:R,link:null,longDesc:null,lowSrc:null,marginHeight:R,marginWidth:R,noResize:Q,noHref:Q,noShade:Q,noWrap:Q,object:null,profile:null,prompt:null,rev:null,rightMargin:R,rules:null,scheme:null,scrolling:me,standby:null,summary:null,text:null,topMargin:R,valueType:null,version:null,vAlign:null,vLink:null,vSpace:R,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:Q,disableRemotePlayback:Q,prefix:null,property:null,results:R,security:null,unselectable:null},space:"html",transform:ep}),bg=or({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:Ve,accentHeight:R,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:R,amplitude:R,arabicForm:null,ascent:R,attributeName:null,attributeType:null,azimuth:R,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:R,by:null,calcMode:null,capHeight:R,className:ne,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:R,diffuseConstant:R,direction:null,display:null,dur:null,divisor:R,dominantBaseline:null,download:Q,dx:null,dy:null,edgeMode:null,editable:null,elevation:R,enableBackground:null,end:null,event:null,exponent:R,externalResourcesRequired:null,fill:null,fillOpacity:R,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:qn,g2:qn,glyphName:qn,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:R,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:R,horizOriginX:R,horizOriginY:R,id:null,ideographic:R,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:R,k:R,k1:R,k2:R,k3:R,k4:R,kernelMatrix:Ve,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:R,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:R,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:R,overlineThickness:R,paintOrder:null,panose1:null,path:null,pathLength:R,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:ne,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:R,pointsAtY:R,pointsAtZ:R,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:Ve,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:Ve,rev:Ve,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:Ve,requiredFeatures:Ve,requiredFonts:Ve,requiredFormats:Ve,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:R,specularExponent:R,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:R,strikethroughThickness:R,string:null,stroke:null,strokeDashArray:Ve,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:R,strokeOpacity:R,strokeWidth:null,style:null,surfaceScale:R,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:Ve,tabIndex:R,tableValues:null,target:null,targetX:R,targetY:R,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:Ve,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:R,underlineThickness:R,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:R,values:null,vAlphabetic:R,vMathematical:R,vectorEffect:null,vHanging:R,vIdeographic:R,version:null,vertAdvY:R,vertOriginX:R,vertOriginY:R,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:R,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:Zf}),tp=or({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),np=or({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:ep}),rp=or({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),jg={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Cg=/[A-Z]/g,Gu=/-[a-z]/g,Eg=/^data[-\w.:]+$/i;function Ng(e,t){const n=pa(t);let r=t,i=Ue;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&Eg.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(Gu,zg);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!Gu.test(l)){let o=l.replace(Cg,_g);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=xs}return new i(r,t)}function _g(e){return"-"+e.toLowerCase()}function zg(e){return e.charAt(1).toUpperCase()}const Tg=Gf([Jf,Sg,tp,np,rp],"html"),ks=Gf([Jf,bg,tp,np,rp],"svg");function Lg(e){return e.join(" ").trim()}var ws={},Ju=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Pg=/\n/g,Ig=/^\s*/,Dg=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Mg=/^:\s*/,Ag=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Rg=/^[;\s]*/,Og=/^\s+|\s+$/g,Fg=`
`,Zu="/",ec="*",cn="",Bg="comment",Ug="declaration";function $g(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(k){var S=k.match(Pg);S&&(n+=S.length);var C=k.lastIndexOf(Fg);r=~C?k.length-C:r+k.length}function l(){var k={line:n,column:r};return function(S){return S.position=new o(k),c(),S}}function o(k){this.start=k,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(k){var S=new Error(t.source+":"+n+":"+r+": "+k);if(S.reason=k,S.filename=t.source,S.line=n,S.column=r,S.source=e,!t.silent)throw S}function s(k){var S=k.exec(e);if(S){var C=S[0];return i(C),e=e.slice(C.length),S}}function c(){s(Ig)}function d(k){var S;for(k=k||[];S=f();)S!==!1&&k.push(S);return k}function f(){var k=l();if(!(Zu!=e.charAt(0)||ec!=e.charAt(1))){for(var S=2;cn!=e.charAt(S)&&(ec!=e.charAt(S)||Zu!=e.charAt(S+1));)++S;if(S+=2,cn===e.charAt(S-1))return a("End of comment missing");var C=e.slice(2,S-2);return r+=2,i(C),e=e.slice(S),r+=2,k({type:Bg,comment:C})}}function h(){var k=l(),S=s(Dg);if(S){if(f(),!s(Mg))return a("property missing ':'");var C=s(Ag),g=k({type:Ug,property:tc(S[0].replace(Ju,cn)),value:C?tc(C[0].replace(Ju,cn)):cn});return s(Rg),g}}function p(){var k=[];d(k);for(var S;S=h();)S!==!1&&(k.push(S),d(k));return k}return c(),p()}function tc(e){return e?e.replace(Og,cn):cn}var Hg=$g,Vg=Hi&&Hi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(ws,"__esModule",{value:!0});ws.default=Qg;const Wg=Vg(Hg);function Qg(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Wg.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Pl={};Object.defineProperty(Pl,"__esModule",{value:!0});Pl.camelCase=void 0;var Kg=/^--[a-zA-Z0-9_-]+$/,qg=/-([a-z])/g,Yg=/^[^-]+$/,Xg=/^-(webkit|moz|ms|o|khtml)-/,Gg=/^-(ms)-/,Jg=function(e){return!e||Yg.test(e)||Kg.test(e)},Zg=function(e,t){return t.toUpperCase()},nc=function(e,t){return"".concat(t,"-")},ev=function(e,t){return t===void 0&&(t={}),Jg(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(Gg,nc):e=e.replace(Xg,nc),e.replace(qg,Zg))};Pl.camelCase=ev;var tv=Hi&&Hi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},nv=tv(ws),rv=Pl;function ga(e,t){var n={};return!e||typeof e!="string"||(0,nv.default)(e,function(r,i){r&&i&&(n[(0,rv.camelCase)(r,t)]=i)}),n}ga.default=ga;var iv=ga;const lv=ja(iv),ip=lp("end"),Ss=lp("start");function lp(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function ov(e){const t=Ss(e),n=ip(e);if(t&&n)return{start:t,end:n}}function Ir(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?rc(e.position):"start"in e||"end"in e?rc(e):"line"in e||"column"in e?va(e):""}function va(e){return ic(e&&e.line)+":"+ic(e&&e.column)}function rc(e){return va(e&&e.start)+"-"+va(e&&e.end)}function ic(e){return e&&typeof e=="number"?e:1}class ze extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Ir(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}ze.prototype.file="";ze.prototype.name="";ze.prototype.reason="";ze.prototype.message="";ze.prototype.stack="";ze.prototype.column=void 0;ze.prototype.line=void 0;ze.prototype.ancestors=void 0;ze.prototype.cause=void 0;ze.prototype.fatal=void 0;ze.prototype.place=void 0;ze.prototype.ruleId=void 0;ze.prototype.source=void 0;const bs={}.hasOwnProperty,av=new Map,sv=/[A-Z]/g,uv=new Set(["table","tbody","thead","tfoot","tr"]),cv=new Set(["td","th"]),op="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function dv(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=xv(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=yv(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?ks:Tg,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=ap(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function ap(e,t,n){if(t.type==="element")return fv(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return pv(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return mv(e,t,n);if(t.type==="mdxjsEsm")return hv(e,t);if(t.type==="root")return gv(e,t,n);if(t.type==="text")return vv(e,t)}function fv(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=ks,e.schema=i),e.ancestors.push(t);const l=up(e,t.tagName,!1),o=kv(e,t);let a=Cs(e,t);return uv.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!kg(s):!0})),sp(e,o,l,t),js(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function pv(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}Zr(e,t.position)}function hv(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);Zr(e,t.position)}function mv(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=ks,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:up(e,t.name,!0),o=wv(e,t),a=Cs(e,t);return sp(e,o,l,t),js(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function gv(e,t,n){const r={};return js(r,Cs(e,t)),e.create(t,e.Fragment,r,n)}function vv(e,t){return t.value}function sp(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function js(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function yv(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function xv(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=Ss(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function kv(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&bs.call(t.properties,i)){const l=Sv(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&cv.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function wv(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else Zr(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else Zr(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Cs(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:av;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const c=i.get(s)||0;o=s+"-"+c,i.set(s,c+1)}}const a=ap(e,l,o);a!==void 0&&n.push(a)}return n}function Sv(e,t,n){const r=Ng(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?mg(n):Lg(n)),r.property==="style"){let i=typeof n=="object"?n:bv(e,String(n));return e.stylePropertyNameCase==="css"&&(i=jv(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?jg[r.property]||r.property:r.attribute,n]}}function bv(e,t){try{return lv(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new ze("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=op+"#cannot-parse-style-attribute",i}}function up(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=qu(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=qu(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return bs.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);Zr(e)}function Zr(e,t){const n=new ze("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=op+"#cannot-handle-mdx-estrees-without-createevaluater",n}function jv(e){const t={};let n;for(n in e)bs.call(e,n)&&(t[Cv(n)]=e[n]);return t}function Cv(e){let t=e.replace(sv,Ev);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function Ev(e){return"-"+e.toLowerCase()}const uo={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},Nv={};function _v(e,t){const n=Nv,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return cp(e,r,i)}function cp(e,t,n){if(zv(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return lc(e.children,t,n)}return Array.isArray(e)?lc(e,t,n):""}function lc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=cp(e[i],t,n);return r.join("")}function zv(e){return!!(e&&typeof e=="object")}const oc=document.createElement("i");function Es(e){const t="&"+e+";";oc.innerHTML=t;const n=oc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function bt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function et(e,t){return e.length>0?(bt(e,e.length,0,t),e):t}const ac={}.hasOwnProperty;function Tv(e){const t={};let n=-1;for(;++n<e.length;)Lv(t,e[n]);return t}function Lv(e,t){let n;for(n in t){const i=(ac.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){ac.call(i,o)||(i[o]=[]);const a=l[o];Pv(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Pv(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);bt(e,0,0,r)}function dp(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function Yn(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const kt=on(/[A-Za-z]/),Ke=on(/[\dA-Za-z]/),Iv=on(/[#-'*+\--9=?A-Z^-~]/);function ya(e){return e!==null&&(e<32||e===127)}const xa=on(/\d/),Dv=on(/[\dA-Fa-f]/),Mv=on(/[!-/:-@[-`{-~]/);function V(e){return e!==null&&e<-2}function Be(e){return e!==null&&(e<0||e===32)}function Z(e){return e===-2||e===-1||e===32}const Av=on(new RegExp("\\p{P}|\\p{S}","u")),Rv=on(/\s/);function on(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function ar(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&Ke(e.charCodeAt(n+1))&&Ke(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function ie(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return Z(s)?(e.enter(n),a(s)):t(s)}function a(s){return Z(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const Ov={tokenize:Fv};function Fv(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),ie(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return V(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const Bv={tokenize:Uv},sc={tokenize:$v};function Uv(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const j=n[r];return t.containerState=j[1],e.attempt(j[0].continuation,s,c)(y)}return c(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&m();const j=t.events.length;let z=j,w;for(;z--;)if(t.events[z][0]==="exit"&&t.events[z][1].type==="chunkFlow"){w=t.events[z][1].end;break}g(r);let E=j;for(;E<t.events.length;)t.events[E][1].end={...w},E++;return bt(t.events,z+1,0,t.events.slice(j)),t.events.length=E,c(y)}return a(y)}function c(y){if(r===n.length){if(!i)return h(y);if(i.currentConstruct&&i.currentConstruct.concrete)return k(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(sc,d,f)(y)}function d(y){return i&&m(),g(r),h(y)}function f(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,k(y)}function h(y){return t.containerState={},e.attempt(sc,p,k)(y)}function p(y){return r++,n.push([t.currentConstruct,t.containerState]),h(y)}function k(y){if(y===null){i&&m(),g(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),S(y)}function S(y){if(y===null){C(e.exit("chunkFlow"),!0),g(0),e.consume(y);return}return V(y)?(e.consume(y),C(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),S)}function C(y,j){const z=t.sliceStream(y);if(j&&z.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(z),t.parser.lazy[y.start.line]){let w=i.events.length;for(;w--;)if(i.events[w][1].start.offset<o&&(!i.events[w][1].end||i.events[w][1].end.offset>o))return;const E=t.events.length;let b=E,N,A;for(;b--;)if(t.events[b][0]==="exit"&&t.events[b][1].type==="chunkFlow"){if(N){A=t.events[b][1].end;break}N=!0}for(g(r),w=E;w<t.events.length;)t.events[w][1].end={...A},w++;bt(t.events,b+1,0,t.events.slice(E)),t.events.length=w}}function g(y){let j=n.length;for(;j-- >y;){const z=n[j];t.containerState=z[1],z[0].exit.call(t,e)}n.length=y}function m(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function $v(e,t,n){return ie(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function uc(e){if(e===null||Be(e)||Rv(e))return 1;if(Av(e))return 2}function Ns(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const ka={name:"attention",resolveAll:Hv,tokenize:Vv};function Hv(e,t){let n=-1,r,i,l,o,a,s,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const f={...e[r][1].end},h={...e[n][1].start};cc(f,-s),cc(h,s),o={type:s>1?"strongSequence":"emphasisSequence",start:f,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:h},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=et(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=et(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=et(c,Ns(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=et(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=et(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,bt(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Vv(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=uc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const c=e.exit("attentionSequence"),d=uc(s),f=!d||d===2&&i||n.includes(s),h=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?f:f&&(i||!h)),c._close=!!(l===42?h:h&&(d||!f)),t(s)}}function cc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Wv={name:"autolink",tokenize:Qv};function Qv(e,t,n){let r=0;return i;function i(p){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(p){return kt(p)?(e.consume(p),o):p===64?n(p):c(p)}function o(p){return p===43||p===45||p===46||Ke(p)?(r=1,a(p)):c(p)}function a(p){return p===58?(e.consume(p),r=0,s):(p===43||p===45||p===46||Ke(p))&&r++<32?(e.consume(p),a):(r=0,c(p))}function s(p){return p===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):p===null||p===32||p===60||ya(p)?n(p):(e.consume(p),s)}function c(p){return p===64?(e.consume(p),d):Iv(p)?(e.consume(p),c):n(p)}function d(p){return Ke(p)?f(p):n(p)}function f(p){return p===46?(e.consume(p),r=0,d):p===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):h(p)}function h(p){if((p===45||Ke(p))&&r++<63){const k=p===45?h:f;return e.consume(p),k}return n(p)}}const Il={partial:!0,tokenize:Kv};function Kv(e,t,n){return r;function r(l){return Z(l)?ie(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||V(l)?t(l):n(l)}}const fp={continuation:{tokenize:Yv},exit:Xv,name:"blockQuote",tokenize:qv};function qv(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return Z(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function Yv(e,t,n){const r=this;return i;function i(o){return Z(o)?ie(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(fp,t,n)(o)}}function Xv(e){e.exit("blockQuote")}const pp={name:"characterEscape",tokenize:Gv};function Gv(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return Mv(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const hp={name:"characterReference",tokenize:Jv};function Jv(e,t,n){const r=this;let i=0,l,o;return a;function a(f){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),s}function s(f){return f===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(f),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=Ke,d(f))}function c(f){return f===88||f===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(f),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=Dv,d):(e.enter("characterReferenceValue"),l=7,o=xa,d(f))}function d(f){if(f===59&&i){const h=e.exit("characterReferenceValue");return o===Ke&&!Es(r.sliceSerialize(h))?n(f):(e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(f)&&i++<l?(e.consume(f),d):n(f)}}const dc={partial:!0,tokenize:ey},fc={concrete:!0,name:"codeFenced",tokenize:Zv};function Zv(e,t,n){const r=this,i={partial:!0,tokenize:z};let l=0,o=0,a;return s;function s(w){return c(w)}function c(w){const E=r.events[r.events.length-1];return l=E&&E[1].type==="linePrefix"?E[2].sliceSerialize(E[1],!0).length:0,a=w,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(w)}function d(w){return w===a?(o++,e.consume(w),d):o<3?n(w):(e.exit("codeFencedFenceSequence"),Z(w)?ie(e,f,"whitespace")(w):f(w))}function f(w){return w===null||V(w)?(e.exit("codeFencedFence"),r.interrupt?t(w):e.check(dc,S,j)(w)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),h(w))}function h(w){return w===null||V(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),f(w)):Z(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),ie(e,p,"whitespace")(w)):w===96&&w===a?n(w):(e.consume(w),h)}function p(w){return w===null||V(w)?f(w):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),k(w))}function k(w){return w===null||V(w)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),f(w)):w===96&&w===a?n(w):(e.consume(w),k)}function S(w){return e.attempt(i,j,C)(w)}function C(w){return e.enter("lineEnding"),e.consume(w),e.exit("lineEnding"),g}function g(w){return l>0&&Z(w)?ie(e,m,"linePrefix",l+1)(w):m(w)}function m(w){return w===null||V(w)?e.check(dc,S,j)(w):(e.enter("codeFlowValue"),y(w))}function y(w){return w===null||V(w)?(e.exit("codeFlowValue"),m(w)):(e.consume(w),y)}function j(w){return e.exit("codeFenced"),t(w)}function z(w,E,b){let N=0;return A;function A(L){return w.enter("lineEnding"),w.consume(L),w.exit("lineEnding"),_}function _(L){return w.enter("codeFencedFence"),Z(L)?ie(w,I,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(L):I(L)}function I(L){return L===a?(w.enter("codeFencedFenceSequence"),H(L)):b(L)}function H(L){return L===a?(N++,w.consume(L),H):N>=o?(w.exit("codeFencedFenceSequence"),Z(L)?ie(w,K,"whitespace")(L):K(L)):b(L)}function K(L){return L===null||V(L)?(w.exit("codeFencedFence"),E(L)):b(L)}}}function ey(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const co={name:"codeIndented",tokenize:ny},ty={partial:!0,tokenize:ry};function ny(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),ie(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?s(c):V(c)?e.attempt(ty,o,s)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||V(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function s(c){return e.exit("codeIndented"),t(c)}}function ry(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):V(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):ie(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):V(o)?i(o):n(o)}}const iy={name:"codeText",previous:oy,resolve:ly,tokenize:ay};function ly(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function oy(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function ay(e,t,n){let r=0,i,l;return o;function o(f){return e.enter("codeText"),e.enter("codeTextSequence"),a(f)}function a(f){return f===96?(e.consume(f),r++,a):(e.exit("codeTextSequence"),s(f))}function s(f){return f===null?n(f):f===32?(e.enter("space"),e.consume(f),e.exit("space"),s):f===96?(l=e.enter("codeTextSequence"),i=0,d(f)):V(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),s):(e.enter("codeTextData"),c(f))}function c(f){return f===null||f===32||f===96||V(f)?(e.exit("codeTextData"),s(f)):(e.consume(f),c)}function d(f){return f===96?(e.consume(f),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(f)):(l.type="codeTextData",c(f))}}class sy{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&yr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),yr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),yr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);yr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);yr(this.left,n.reverse())}}}function yr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function mp(e){const t={};let n=-1,r,i,l,o,a,s,c;const d=new sy(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,uy(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return bt(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function uy(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],c={};let d,f,h=-1,p=n,k=0,S=0;const C=[S];for(;p;){for(;e.get(++i)[1]!==p;);l.push(i),p._tokenizer||(d=r.sliceStream(p),p.next||d.push(null),f&&o.defineSkip(p.start),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),f=p,p=p.next}for(p=n;++h<a.length;)a[h][0]==="exit"&&a[h-1][0]==="enter"&&a[h][1].type===a[h-1][1].type&&a[h][1].start.line!==a[h][1].end.line&&(S=h+1,C.push(S),p._tokenizer=void 0,p.previous=void 0,p=p.next);for(o.events=[],p?(p._tokenizer=void 0,p.previous=void 0):C.pop(),h=C.length;h--;){const g=a.slice(C[h],C[h+1]),m=l.pop();s.push([m,m+g.length-1]),e.splice(m,2,g)}for(s.reverse(),h=-1;++h<s.length;)c[k+s[h][0]]=k+s[h][1],k+=s[h][1]-s[h][0]-1;return c}const cy={resolve:fy,tokenize:py},dy={partial:!0,tokenize:hy};function fy(e){return mp(e),e}function py(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):V(a)?e.check(dy,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function hy(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),ie(e,l,"linePrefix")}function l(o){if(o===null||V(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function gp(e,t,n,r,i,l,o,a,s){const c=s||Number.POSITIVE_INFINITY;let d=0;return f;function f(g){return g===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(g),e.exit(l),h):g===null||g===32||g===41||ya(g)?n(g):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),S(g))}function h(g){return g===62?(e.enter(l),e.consume(g),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),p(g))}function p(g){return g===62?(e.exit("chunkString"),e.exit(a),h(g)):g===null||g===60||V(g)?n(g):(e.consume(g),g===92?k:p)}function k(g){return g===60||g===62||g===92?(e.consume(g),p):p(g)}function S(g){return!d&&(g===null||g===41||Be(g))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(g)):d<c&&g===40?(e.consume(g),d++,S):g===41?(e.consume(g),d--,S):g===null||g===32||g===40||ya(g)?n(g):(e.consume(g),g===92?C:S)}function C(g){return g===40||g===41||g===92?(e.consume(g),S):S(g)}}function vp(e,t,n,r,i,l){const o=this;let a=0,s;return c;function c(p){return e.enter(r),e.enter(i),e.consume(p),e.exit(i),e.enter(l),d}function d(p){return a>999||p===null||p===91||p===93&&!s||p===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(p):p===93?(e.exit(l),e.enter(i),e.consume(p),e.exit(i),e.exit(r),t):V(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),f(p))}function f(p){return p===null||p===91||p===93||V(p)||a++>999?(e.exit("chunkString"),d(p)):(e.consume(p),s||(s=!Z(p)),p===92?h:f)}function h(p){return p===91||p===92||p===93?(e.consume(p),a++,f):f(p)}}function yp(e,t,n,r,i,l){let o;return a;function a(h){return h===34||h===39||h===40?(e.enter(r),e.enter(i),e.consume(h),e.exit(i),o=h===40?41:h,s):n(h)}function s(h){return h===o?(e.enter(i),e.consume(h),e.exit(i),e.exit(r),t):(e.enter(l),c(h))}function c(h){return h===o?(e.exit(l),s(o)):h===null?n(h):V(h)?(e.enter("lineEnding"),e.consume(h),e.exit("lineEnding"),ie(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(h))}function d(h){return h===o||h===null||V(h)?(e.exit("chunkString"),c(h)):(e.consume(h),h===92?f:d)}function f(h){return h===o||h===92?(e.consume(h),d):d(h)}}function Dr(e,t){let n;return r;function r(i){return V(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):Z(i)?ie(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const my={name:"definition",tokenize:vy},gy={partial:!0,tokenize:yy};function vy(e,t,n){const r=this;let i;return l;function l(p){return e.enter("definition"),o(p)}function o(p){return vp.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(p)}function a(p){return i=Yn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),p===58?(e.enter("definitionMarker"),e.consume(p),e.exit("definitionMarker"),s):n(p)}function s(p){return Be(p)?Dr(e,c)(p):c(p)}function c(p){return gp(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(p)}function d(p){return e.attempt(gy,f,f)(p)}function f(p){return Z(p)?ie(e,h,"whitespace")(p):h(p)}function h(p){return p===null||V(p)?(e.exit("definition"),r.parser.defined.push(i),t(p)):n(p)}}function yy(e,t,n){return r;function r(a){return Be(a)?Dr(e,i)(a):n(a)}function i(a){return yp(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return Z(a)?ie(e,o,"whitespace")(a):o(a)}function o(a){return a===null||V(a)?t(a):n(a)}}const xy={name:"hardBreakEscape",tokenize:ky};function ky(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return V(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const wy={name:"headingAtx",resolve:Sy,tokenize:by};function Sy(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},bt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function by(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||Be(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||V(d)?(e.exit("atxHeading"),t(d)):Z(d)?ie(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||Be(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const jy=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],pc=["pre","script","style","textarea"],Cy={concrete:!0,name:"htmlFlow",resolveTo:_y,tokenize:zy},Ey={partial:!0,tokenize:Ly},Ny={partial:!0,tokenize:Ty};function _y(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function zy(e,t,n){const r=this;let i,l,o,a,s;return c;function c(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),f}function f(x){return x===33?(e.consume(x),h):x===47?(e.consume(x),l=!0,S):x===63?(e.consume(x),i=3,r.interrupt?t:v):kt(x)?(e.consume(x),o=String.fromCharCode(x),C):n(x)}function h(x){return x===45?(e.consume(x),i=2,p):x===91?(e.consume(x),i=5,a=0,k):kt(x)?(e.consume(x),i=4,r.interrupt?t:v):n(x)}function p(x){return x===45?(e.consume(x),r.interrupt?t:v):n(x)}function k(x){const ve="CDATA[";return x===ve.charCodeAt(a++)?(e.consume(x),a===ve.length?r.interrupt?t:I:k):n(x)}function S(x){return kt(x)?(e.consume(x),o=String.fromCharCode(x),C):n(x)}function C(x){if(x===null||x===47||x===62||Be(x)){const ve=x===47,lt=o.toLowerCase();return!ve&&!l&&pc.includes(lt)?(i=1,r.interrupt?t(x):I(x)):jy.includes(o.toLowerCase())?(i=6,ve?(e.consume(x),g):r.interrupt?t(x):I(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?m(x):y(x))}return x===45||Ke(x)?(e.consume(x),o+=String.fromCharCode(x),C):n(x)}function g(x){return x===62?(e.consume(x),r.interrupt?t:I):n(x)}function m(x){return Z(x)?(e.consume(x),m):A(x)}function y(x){return x===47?(e.consume(x),A):x===58||x===95||kt(x)?(e.consume(x),j):Z(x)?(e.consume(x),y):A(x)}function j(x){return x===45||x===46||x===58||x===95||Ke(x)?(e.consume(x),j):z(x)}function z(x){return x===61?(e.consume(x),w):Z(x)?(e.consume(x),z):y(x)}function w(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,E):Z(x)?(e.consume(x),w):b(x)}function E(x){return x===s?(e.consume(x),s=null,N):x===null||V(x)?n(x):(e.consume(x),E)}function b(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||Be(x)?z(x):(e.consume(x),b)}function N(x){return x===47||x===62||Z(x)?y(x):n(x)}function A(x){return x===62?(e.consume(x),_):n(x)}function _(x){return x===null||V(x)?I(x):Z(x)?(e.consume(x),_):n(x)}function I(x){return x===45&&i===2?(e.consume(x),D):x===60&&i===1?(e.consume(x),F):x===62&&i===4?(e.consume(x),q):x===63&&i===3?(e.consume(x),v):x===93&&i===5?(e.consume(x),B):V(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(Ey,J,H)(x)):x===null||V(x)?(e.exit("htmlFlowData"),H(x)):(e.consume(x),I)}function H(x){return e.check(Ny,K,J)(x)}function K(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),L}function L(x){return x===null||V(x)?H(x):(e.enter("htmlFlowData"),I(x))}function D(x){return x===45?(e.consume(x),v):I(x)}function F(x){return x===47?(e.consume(x),o="",P):I(x)}function P(x){if(x===62){const ve=o.toLowerCase();return pc.includes(ve)?(e.consume(x),q):I(x)}return kt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),P):I(x)}function B(x){return x===93?(e.consume(x),v):I(x)}function v(x){return x===62?(e.consume(x),q):x===45&&i===2?(e.consume(x),v):I(x)}function q(x){return x===null||V(x)?(e.exit("htmlFlowData"),J(x)):(e.consume(x),q)}function J(x){return e.exit("htmlFlow"),t(x)}}function Ty(e,t,n){const r=this;return i;function i(o){return V(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Ly(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Il,t,n)}}const Py={name:"htmlText",tokenize:Iy};function Iy(e,t,n){const r=this;let i,l,o;return a;function a(v){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(v),s}function s(v){return v===33?(e.consume(v),c):v===47?(e.consume(v),z):v===63?(e.consume(v),y):kt(v)?(e.consume(v),b):n(v)}function c(v){return v===45?(e.consume(v),d):v===91?(e.consume(v),l=0,k):kt(v)?(e.consume(v),m):n(v)}function d(v){return v===45?(e.consume(v),p):n(v)}function f(v){return v===null?n(v):v===45?(e.consume(v),h):V(v)?(o=f,F(v)):(e.consume(v),f)}function h(v){return v===45?(e.consume(v),p):f(v)}function p(v){return v===62?D(v):v===45?h(v):f(v)}function k(v){const q="CDATA[";return v===q.charCodeAt(l++)?(e.consume(v),l===q.length?S:k):n(v)}function S(v){return v===null?n(v):v===93?(e.consume(v),C):V(v)?(o=S,F(v)):(e.consume(v),S)}function C(v){return v===93?(e.consume(v),g):S(v)}function g(v){return v===62?D(v):v===93?(e.consume(v),g):S(v)}function m(v){return v===null||v===62?D(v):V(v)?(o=m,F(v)):(e.consume(v),m)}function y(v){return v===null?n(v):v===63?(e.consume(v),j):V(v)?(o=y,F(v)):(e.consume(v),y)}function j(v){return v===62?D(v):y(v)}function z(v){return kt(v)?(e.consume(v),w):n(v)}function w(v){return v===45||Ke(v)?(e.consume(v),w):E(v)}function E(v){return V(v)?(o=E,F(v)):Z(v)?(e.consume(v),E):D(v)}function b(v){return v===45||Ke(v)?(e.consume(v),b):v===47||v===62||Be(v)?N(v):n(v)}function N(v){return v===47?(e.consume(v),D):v===58||v===95||kt(v)?(e.consume(v),A):V(v)?(o=N,F(v)):Z(v)?(e.consume(v),N):D(v)}function A(v){return v===45||v===46||v===58||v===95||Ke(v)?(e.consume(v),A):_(v)}function _(v){return v===61?(e.consume(v),I):V(v)?(o=_,F(v)):Z(v)?(e.consume(v),_):N(v)}function I(v){return v===null||v===60||v===61||v===62||v===96?n(v):v===34||v===39?(e.consume(v),i=v,H):V(v)?(o=I,F(v)):Z(v)?(e.consume(v),I):(e.consume(v),K)}function H(v){return v===i?(e.consume(v),i=void 0,L):v===null?n(v):V(v)?(o=H,F(v)):(e.consume(v),H)}function K(v){return v===null||v===34||v===39||v===60||v===61||v===96?n(v):v===47||v===62||Be(v)?N(v):(e.consume(v),K)}function L(v){return v===47||v===62||Be(v)?N(v):n(v)}function D(v){return v===62?(e.consume(v),e.exit("htmlTextData"),e.exit("htmlText"),t):n(v)}function F(v){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(v),e.exit("lineEnding"),P}function P(v){return Z(v)?ie(e,B,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(v):B(v)}function B(v){return e.enter("htmlTextData"),o(v)}}const _s={name:"labelEnd",resolveAll:Ry,resolveTo:Oy,tokenize:Fy},Dy={tokenize:By},My={tokenize:Uy},Ay={tokenize:$y};function Ry(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&bt(e,0,e.length,n),e}function Oy(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",c,t]],a=et(a,e.slice(l+1,l+r+3)),a=et(a,[["enter",d,t]]),a=et(a,Ns(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=et(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=et(a,e.slice(o+1)),a=et(a,[["exit",s,t]]),bt(e,l,e.length,a),e}function Fy(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(h){return l?l._inactive?f(h):(o=r.parser.defined.includes(Yn(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(h),e.exit("labelMarker"),e.exit("labelEnd"),s):n(h)}function s(h){return h===40?e.attempt(Dy,d,o?d:f)(h):h===91?e.attempt(My,d,o?c:f)(h):o?d(h):f(h)}function c(h){return e.attempt(Ay,d,f)(h)}function d(h){return t(h)}function f(h){return l._balanced=!0,n(h)}}function By(e,t,n){return r;function r(f){return e.enter("resource"),e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),i}function i(f){return Be(f)?Dr(e,l)(f):l(f)}function l(f){return f===41?d(f):gp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(f)}function o(f){return Be(f)?Dr(e,s)(f):d(f)}function a(f){return n(f)}function s(f){return f===34||f===39||f===40?yp(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(f):d(f)}function c(f){return Be(f)?Dr(e,d)(f):d(f)}function d(f){return f===41?(e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),e.exit("resource"),t):n(f)}}function Uy(e,t,n){const r=this;return i;function i(a){return vp.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(Yn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function $y(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Hy={name:"labelStartImage",resolveAll:_s.resolveAll,tokenize:Vy};function Vy(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Wy={name:"labelStartLink",resolveAll:_s.resolveAll,tokenize:Qy};function Qy(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const fo={name:"lineEnding",tokenize:Ky};function Ky(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),ie(e,t,"linePrefix")}}const Ui={name:"thematicBreak",tokenize:qy};function qy(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),s(c)):r>=3&&(c===null||V(c))?(e.exit("thematicBreak"),t(c)):n(c)}function s(c){return c===i?(e.consume(c),r++,s):(e.exit("thematicBreakSequence"),Z(c)?ie(e,a,"whitespace")(c):a(c))}}const De={continuation:{tokenize:Jy},exit:ex,name:"list",tokenize:Gy},Yy={partial:!0,tokenize:tx},Xy={partial:!0,tokenize:Zy};function Gy(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(p){const k=r.containerState.type||(p===42||p===43||p===45?"listUnordered":"listOrdered");if(k==="listUnordered"?!r.containerState.marker||p===r.containerState.marker:xa(p)){if(r.containerState.type||(r.containerState.type=k,e.enter(k,{_container:!0})),k==="listUnordered")return e.enter("listItemPrefix"),p===42||p===45?e.check(Ui,n,c)(p):c(p);if(!r.interrupt||p===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(p)}return n(p)}function s(p){return xa(p)&&++o<10?(e.consume(p),s):(!r.interrupt||o<2)&&(r.containerState.marker?p===r.containerState.marker:p===41||p===46)?(e.exit("listItemValue"),c(p)):n(p)}function c(p){return e.enter("listItemMarker"),e.consume(p),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||p,e.check(Il,r.interrupt?n:d,e.attempt(Yy,h,f))}function d(p){return r.containerState.initialBlankLine=!0,l++,h(p)}function f(p){return Z(p)?(e.enter("listItemPrefixWhitespace"),e.consume(p),e.exit("listItemPrefixWhitespace"),h):n(p)}function h(p){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(p)}}function Jy(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Il,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,ie(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!Z(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Xy,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,ie(e,e.attempt(De,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function Zy(e,t,n){const r=this;return ie(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function ex(e){e.exit(this.containerState.type)}function tx(e,t,n){const r=this;return ie(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!Z(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const hc={name:"setextUnderline",resolveTo:nx,tokenize:rx};function nx(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function rx(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,f;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){f=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||f)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),Z(c)?ie(e,s,"lineSuffix")(c):s(c))}function s(c){return c===null||V(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const ix={tokenize:lx};function lx(e){const t=this,n=e.attempt(Il,r,e.attempt(this.parser.constructs.flowInitial,i,ie(e,e.attempt(this.parser.constructs.flow,i,e.attempt(cy,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const ox={resolveAll:kp()},ax=xp("string"),sx=xp("text");function xp(e){return{resolveAll:kp(e==="text"?ux:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function c(d){if(d===null)return!0;const f=i[d];let h=-1;if(f)for(;++h<f.length;){const p=f[h];if(!p.previous||p.previous.call(r,r.previous))return!0}return!1}}}function kp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function ux(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)s=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const cx={42:De,43:De,45:De,48:De,49:De,50:De,51:De,52:De,53:De,54:De,55:De,56:De,57:De,62:fp},dx={91:my},fx={[-2]:co,[-1]:co,32:co},px={35:wy,42:Ui,45:[hc,Ui],60:Cy,61:hc,95:Ui,96:fc,126:fc},hx={38:hp,92:pp},mx={[-5]:fo,[-4]:fo,[-3]:fo,33:Hy,38:hp,42:ka,60:[Wv,Py],91:Wy,92:[xy,pp],93:_s,95:ka,96:iy},gx={null:[ka,ox]},vx={null:[42,95]},yx={null:[]},xx=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:vx,contentInitial:dx,disable:yx,document:cx,flow:px,flowInitial:fx,insideSpan:gx,string:hx,text:mx},Symbol.toStringTag,{value:"Module"}));function kx(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:E(z),check:E(w),consume:m,enter:y,exit:j,interrupt:E(w,{interrupt:!0})},c={code:null,containerState:{},defineSkip:S,events:[],now:k,parser:e,previous:null,sliceSerialize:h,sliceStream:p,write:f};let d=t.tokenize.call(c,s);return t.resolveAll&&l.push(t),c;function f(_){return o=et(o,_),C(),o[o.length-1]!==null?[]:(b(t,0),c.events=Ns(l,c.events,c),c.events)}function h(_,I){return Sx(p(_),I)}function p(_){return wx(o,_)}function k(){const{_bufferIndex:_,_index:I,line:H,column:K,offset:L}=r;return{_bufferIndex:_,_index:I,line:H,column:K,offset:L}}function S(_){i[_.line]=_.column,A()}function C(){let _;for(;r._index<o.length;){const I=o[r._index];if(typeof I=="string")for(_=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===_&&r._bufferIndex<I.length;)g(I.charCodeAt(r._bufferIndex));else g(I)}}function g(_){d=d(_)}function m(_){V(_)?(r.line++,r.column=1,r.offset+=_===-3?2:1,A()):_!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=_}function y(_,I){const H=I||{};return H.type=_,H.start=k(),c.events.push(["enter",H,c]),a.push(H),H}function j(_){const I=a.pop();return I.end=k(),c.events.push(["exit",I,c]),I}function z(_,I){b(_,I.from)}function w(_,I){I.restore()}function E(_,I){return H;function H(K,L,D){let F,P,B,v;return Array.isArray(K)?J(K):"tokenize"in K?J([K]):q(K);function q(le){return ht;function ht(Rt){const Cn=Rt!==null&&le[Rt],En=Rt!==null&&le.null,ai=[...Array.isArray(Cn)?Cn:Cn?[Cn]:[],...Array.isArray(En)?En:En?[En]:[]];return J(ai)(Rt)}}function J(le){return F=le,P=0,le.length===0?D:x(le[P])}function x(le){return ht;function ht(Rt){return v=N(),B=le,le.partial||(c.currentConstruct=le),le.name&&c.parser.constructs.disable.null.includes(le.name)?lt():le.tokenize.call(I?Object.assign(Object.create(c),I):c,s,ve,lt)(Rt)}}function ve(le){return _(B,v),L}function lt(le){return v.restore(),++P<F.length?x(F[P]):D}}}function b(_,I){_.resolveAll&&!l.includes(_)&&l.push(_),_.resolve&&bt(c.events,I,c.events.length-I,_.resolve(c.events.slice(I),c)),_.resolveTo&&(c.events=_.resolveTo(c.events,c))}function N(){const _=k(),I=c.previous,H=c.currentConstruct,K=c.events.length,L=Array.from(a);return{from:K,restore:D};function D(){r=_,c.previous=I,c.currentConstruct=H,c.events.length=K,a=L,A()}}function A(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function wx(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function Sx(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function bx(e){const r={constructs:Tv([xx,...(e||{}).extensions||[]]),content:i(Ov),defined:[],document:i(Bv),flow:i(ix),lazy:{},string:i(ax),text:i(sx)};return r;function i(l){return o;function o(a){return kx(r,l,a)}}}function jx(e){for(;!mp(e););return e}const mc=/[\0\t\n\r]/g;function Cx(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let c,d,f,h,p;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),f=0,t="",n&&(l.charCodeAt(0)===65279&&f++,n=void 0);f<l.length;){if(mc.lastIndex=f,c=mc.exec(l),h=c&&c.index!==void 0?c.index:l.length,p=l.charCodeAt(h),!c){t=l.slice(f);break}if(p===10&&f===h&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),f<h&&(s.push(l.slice(f,h)),e+=h-f),p){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}f=h+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const Ex=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function Nx(e){return e.replace(Ex,_x)}function _x(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return dp(n.slice(l?2:1),l?16:10)}return Es(n)||e}const wp={}.hasOwnProperty;function zx(e,t,n){return typeof t!="string"&&(n=t,t=void 0),Tx(n)(jx(bx(n).document().write(Cx()(e,t,!0))))}function Tx(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Rs),autolinkProtocol:N,autolinkEmail:N,atxHeading:l(Ds),blockQuote:l(En),characterEscape:N,characterReference:N,codeFenced:l(ai),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(ai,o),codeText:l(Ip,o),codeTextData:N,data:N,codeFlowValue:N,definition:l(Dp),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Mp),hardBreakEscape:l(Ms),hardBreakTrailing:l(Ms),htmlFlow:l(As,o),htmlFlowData:N,htmlText:l(As,o),htmlTextData:N,image:l(Ap),label:o,link:l(Rs),listItem:l(Rp),listItemValue:h,listOrdered:l(Os,f),listUnordered:l(Os),paragraph:l(Op),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Ds),strong:l(Fp),thematicBreak:l(Up)},exit:{atxHeading:s(),atxHeadingSequence:z,autolink:s(),autolinkEmail:Cn,autolinkProtocol:Rt,blockQuote:s(),characterEscapeValue:A,characterReferenceMarkerHexadecimal:lt,characterReferenceMarkerNumeric:lt,characterReferenceValue:le,characterReference:ht,codeFenced:s(C),codeFencedFence:S,codeFencedFenceInfo:p,codeFencedFenceMeta:k,codeFlowValue:A,codeIndented:s(g),codeText:s(L),codeTextData:A,data:A,definition:s(),definitionDestinationString:j,definitionLabelString:m,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(I),hardBreakTrailing:s(I),htmlFlow:s(H),htmlFlowData:A,htmlText:s(K),htmlTextData:A,image:s(F),label:B,labelText:P,lineEnding:_,link:s(D),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:ve,resourceDestinationString:v,resourceTitleString:q,resource:J,setextHeading:s(b),setextHeadingLineSequence:E,setextHeadingText:w,strong:s(),thematicBreak:s()}};Sp(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(T){let O={type:"root",children:[]};const W={stack:[O],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},X=[];let te=-1;for(;++te<T.length;)if(T[te][1].type==="listOrdered"||T[te][1].type==="listUnordered")if(T[te][0]==="enter")X.push(te);else{const ot=X.pop();te=i(T,ot,te)}for(te=-1;++te<T.length;){const ot=t[T[te][0]];wp.call(ot,T[te][1].type)&&ot[T[te][1].type].call(Object.assign({sliceSerialize:T[te][2].sliceSerialize},W),T[te][1])}if(W.tokenStack.length>0){const ot=W.tokenStack[W.tokenStack.length-1];(ot[1]||gc).call(W,void 0,ot[0])}for(O.position={start:Ft(T.length>0?T[0][1].start:{line:1,column:1,offset:0}),end:Ft(T.length>0?T[T.length-2][1].end:{line:1,column:1,offset:0})},te=-1;++te<t.transforms.length;)O=t.transforms[te](O)||O;return O}function i(T,O,W){let X=O-1,te=-1,ot=!1,an,jt,sr,ur;for(;++X<=W;){const $e=T[X];switch($e[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{$e[0]==="enter"?te++:te--,ur=void 0;break}case"lineEndingBlank":{$e[0]==="enter"&&(an&&!ur&&!te&&!sr&&(sr=X),ur=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:ur=void 0}if(!te&&$e[0]==="enter"&&$e[1].type==="listItemPrefix"||te===-1&&$e[0]==="exit"&&($e[1].type==="listUnordered"||$e[1].type==="listOrdered")){if(an){let Nn=X;for(jt=void 0;Nn--;){const Ct=T[Nn];if(Ct[1].type==="lineEnding"||Ct[1].type==="lineEndingBlank"){if(Ct[0]==="exit")continue;jt&&(T[jt][1].type="lineEndingBlank",ot=!0),Ct[1].type="lineEnding",jt=Nn}else if(!(Ct[1].type==="linePrefix"||Ct[1].type==="blockQuotePrefix"||Ct[1].type==="blockQuotePrefixWhitespace"||Ct[1].type==="blockQuoteMarker"||Ct[1].type==="listItemIndent"))break}sr&&(!jt||sr<jt)&&(an._spread=!0),an.end=Object.assign({},jt?T[jt][1].start:$e[1].end),T.splice(jt||X,0,["exit",an,$e[2]]),X++,W++}if($e[1].type==="listItemPrefix"){const Nn={type:"listItem",_spread:!1,start:Object.assign({},$e[1].start),end:void 0};an=Nn,T.splice(X,0,["enter",Nn,$e[2]]),X++,W++,sr=void 0,ur=!0}}}return T[O][1]._spread=ot,W}function l(T,O){return W;function W(X){a.call(this,T(X),X),O&&O.call(this,X)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(T,O,W){this.stack[this.stack.length-1].children.push(T),this.stack.push(T),this.tokenStack.push([O,W||void 0]),T.position={start:Ft(O.start),end:void 0}}function s(T){return O;function O(W){T&&T.call(this,W),c.call(this,W)}}function c(T,O){const W=this.stack.pop(),X=this.tokenStack.pop();if(X)X[0].type!==T.type&&(O?O.call(this,T,X[0]):(X[1]||gc).call(this,T,X[0]));else throw new Error("Cannot close `"+T.type+"` ("+Ir({start:T.start,end:T.end})+"): it’s not open");W.position.end=Ft(T.end)}function d(){return _v(this.stack.pop())}function f(){this.data.expectingFirstListItemValue=!0}function h(T){if(this.data.expectingFirstListItemValue){const O=this.stack[this.stack.length-2];O.start=Number.parseInt(this.sliceSerialize(T),10),this.data.expectingFirstListItemValue=void 0}}function p(){const T=this.resume(),O=this.stack[this.stack.length-1];O.lang=T}function k(){const T=this.resume(),O=this.stack[this.stack.length-1];O.meta=T}function S(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function C(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function g(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T.replace(/(\r?\n|\r)$/g,"")}function m(T){const O=this.resume(),W=this.stack[this.stack.length-1];W.label=O,W.identifier=Yn(this.sliceSerialize(T)).toLowerCase()}function y(){const T=this.resume(),O=this.stack[this.stack.length-1];O.title=T}function j(){const T=this.resume(),O=this.stack[this.stack.length-1];O.url=T}function z(T){const O=this.stack[this.stack.length-1];if(!O.depth){const W=this.sliceSerialize(T).length;O.depth=W}}function w(){this.data.setextHeadingSlurpLineEnding=!0}function E(T){const O=this.stack[this.stack.length-1];O.depth=this.sliceSerialize(T).codePointAt(0)===61?1:2}function b(){this.data.setextHeadingSlurpLineEnding=void 0}function N(T){const W=this.stack[this.stack.length-1].children;let X=W[W.length-1];(!X||X.type!=="text")&&(X=Bp(),X.position={start:Ft(T.start),end:void 0},W.push(X)),this.stack.push(X)}function A(T){const O=this.stack.pop();O.value+=this.sliceSerialize(T),O.position.end=Ft(T.end)}function _(T){const O=this.stack[this.stack.length-1];if(this.data.atHardBreak){const W=O.children[O.children.length-1];W.position.end=Ft(T.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(O.type)&&(N.call(this,T),A.call(this,T))}function I(){this.data.atHardBreak=!0}function H(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function K(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function L(){const T=this.resume(),O=this.stack[this.stack.length-1];O.value=T}function D(){const T=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";T.type+="Reference",T.referenceType=O,delete T.url,delete T.title}else delete T.identifier,delete T.label;this.data.referenceType=void 0}function F(){const T=this.stack[this.stack.length-1];if(this.data.inReference){const O=this.data.referenceType||"shortcut";T.type+="Reference",T.referenceType=O,delete T.url,delete T.title}else delete T.identifier,delete T.label;this.data.referenceType=void 0}function P(T){const O=this.sliceSerialize(T),W=this.stack[this.stack.length-2];W.label=Nx(O),W.identifier=Yn(O).toLowerCase()}function B(){const T=this.stack[this.stack.length-1],O=this.resume(),W=this.stack[this.stack.length-1];if(this.data.inReference=!0,W.type==="link"){const X=T.children;W.children=X}else W.alt=O}function v(){const T=this.resume(),O=this.stack[this.stack.length-1];O.url=T}function q(){const T=this.resume(),O=this.stack[this.stack.length-1];O.title=T}function J(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function ve(T){const O=this.resume(),W=this.stack[this.stack.length-1];W.label=O,W.identifier=Yn(this.sliceSerialize(T)).toLowerCase(),this.data.referenceType="full"}function lt(T){this.data.characterReferenceType=T.type}function le(T){const O=this.sliceSerialize(T),W=this.data.characterReferenceType;let X;W?(X=dp(O,W==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):X=Es(O);const te=this.stack[this.stack.length-1];te.value+=X}function ht(T){const O=this.stack.pop();O.position.end=Ft(T.end)}function Rt(T){A.call(this,T);const O=this.stack[this.stack.length-1];O.url=this.sliceSerialize(T)}function Cn(T){A.call(this,T);const O=this.stack[this.stack.length-1];O.url="mailto:"+this.sliceSerialize(T)}function En(){return{type:"blockquote",children:[]}}function ai(){return{type:"code",lang:null,meta:null,value:""}}function Ip(){return{type:"inlineCode",value:""}}function Dp(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Mp(){return{type:"emphasis",children:[]}}function Ds(){return{type:"heading",depth:0,children:[]}}function Ms(){return{type:"break"}}function As(){return{type:"html",value:""}}function Ap(){return{type:"image",title:null,url:"",alt:null}}function Rs(){return{type:"link",title:null,url:"",children:[]}}function Os(T){return{type:"list",ordered:T.type==="listOrdered",start:null,spread:T._spread,children:[]}}function Rp(T){return{type:"listItem",spread:T._spread,checked:null,children:[]}}function Op(){return{type:"paragraph",children:[]}}function Fp(){return{type:"strong",children:[]}}function Bp(){return{type:"text",value:""}}function Up(){return{type:"thematicBreak"}}}function Ft(e){return{line:e.line,column:e.column,offset:e.offset}}function Sp(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Sp(e,r):Lx(e,r)}}function Lx(e,t){let n;for(n in t)if(wp.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function gc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Ir({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Ir({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Ir({start:t.start,end:t.end})+") is still open")}function Px(e){const t=this;t.parser=n;function n(r){return zx(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function Ix(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function Dx(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function Mx(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function Ax(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Rx(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Ox(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=ar(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const c={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,c),e.applyData(t,c)}function Fx(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Bx(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function bp(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function Ux(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return bp(e,t);const i={src:ar(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function $x(e,t){const n={src:ar(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function Hx(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function Vx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return bp(e,t);const i={href:ar(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function Wx(e,t){const n={href:ar(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function Qx(e,t,n){const r=e.all(t),i=n?Kx(n):jp(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let f;d&&d.type==="element"&&d.tagName==="p"?f=d:(f={type:"element",tagName:"p",properties:{},children:[]},r.unshift(f)),f.children.length>0&&f.children.unshift({type:"text",value:" "}),f.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function Kx(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=jp(n[r])}return t}function jp(e){const t=e.spread;return t??e.children.length>1}function qx(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function Yx(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Xx(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function Gx(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Jx(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=Ss(t.children[1]),s=ip(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function Zx(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const c=[];for(;++s<a;){const f=t.children[s],h={},p=o?o[s]:void 0;p&&(h.align=p);let k={type:"element",tagName:l,properties:h,children:[]};f&&(k.children=e.all(f),e.patch(f,k),k=e.applyData(f,k)),c.push(k)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function e1(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const vc=9,yc=32;function t1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(xc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(xc(t.slice(i),i>0,!1)),l.join("")}function xc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===vc||l===yc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===vc||l===yc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function n1(e,t){const n={type:"text",value:t1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function r1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const i1={blockquote:Ix,break:Dx,code:Mx,delete:Ax,emphasis:Rx,footnoteReference:Ox,heading:Fx,html:Bx,imageReference:Ux,image:$x,inlineCode:Hx,linkReference:Vx,link:Wx,listItem:Qx,list:qx,paragraph:Yx,root:Xx,strong:Gx,table:Jx,tableCell:e1,tableRow:Zx,text:n1,thematicBreak:r1,toml:Ci,yaml:Ci,definition:Ci,footnoteDefinition:Ci};function Ci(){}const Cp=-1,Dl=0,Mr=1,hl=2,zs=3,Ts=4,Ls=5,Ps=6,Ep=7,Np=8,kc=typeof self=="object"?self:globalThis,l1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Dl:case Cp:return n(o,i);case Mr:{const a=n([],i);for(const s of o)a.push(r(s));return a}case hl:{const a=n({},i);for(const[s,c]of o)a[r(s)]=r(c);return a}case zs:return n(new Date(o),i);case Ts:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Ls:{const a=n(new Map,i);for(const[s,c]of o)a.set(r(s),r(c));return a}case Ps:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Ep:{const{name:a,message:s}=o;return n(new kc[a](s),i)}case Np:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new kc[l](o),i)};return r},wc=e=>l1(new Map,e)(0),zn="",{toString:o1}={},{keys:a1}=Object,xr=e=>{const t=typeof e;if(t!=="object"||!e)return[Dl,t];const n=o1.call(e).slice(8,-1);switch(n){case"Array":return[Mr,zn];case"Object":return[hl,zn];case"Date":return[zs,zn];case"RegExp":return[Ts,zn];case"Map":return[Ls,zn];case"Set":return[Ps,zn];case"DataView":return[Mr,n]}return n.includes("Array")?[Mr,n]:n.includes("Error")?[Ep,n]:[hl,n]},Ei=([e,t])=>e===Dl&&(t==="function"||t==="symbol"),s1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=xr(o);switch(a){case Dl:{let d=o;switch(s){case"bigint":a=Np,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([Cp],o)}return i([a,d],o)}case Mr:{if(s){let h=o;return s==="DataView"?h=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(h=new Uint8Array(o)),i([s,[...h]],o)}const d=[],f=i([a,d],o);for(const h of o)d.push(l(h));return f}case hl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],f=i([a,d],o);for(const h of a1(o))(e||!Ei(xr(o[h])))&&d.push([l(h),l(o[h])]);return f}case zs:return i([a,o.toISOString()],o);case Ts:{const{source:d,flags:f}=o;return i([a,{source:d,flags:f}],o)}case Ls:{const d=[],f=i([a,d],o);for(const[h,p]of o)(e||!(Ei(xr(h))||Ei(xr(p))))&&d.push([l(h),l(p)]);return f}case Ps:{const d=[],f=i([a,d],o);for(const h of o)(e||!Ei(xr(h)))&&d.push(l(h));return f}}const{message:c}=o;return i([a,{name:s,message:c}],o)};return l},Sc=(e,{json:t,lossy:n}={})=>{const r=[];return s1(!(t||n),!!t,new Map,r)(e),r},ml=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?wc(Sc(e,t)):structuredClone(e):(e,t)=>wc(Sc(e,t));function u1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function c1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function d1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||u1,r=e.options.footnoteBackLabel||c1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[s]);if(!c)continue;const d=e.all(c),f=String(c.identifier).toUpperCase(),h=ar(f.toLowerCase());let p=0;const k=[],S=e.footnoteCounts.get(f);for(;S!==void 0&&++p<=S;){k.length>0&&k.push({type:"text",value:" "});let m=typeof n=="string"?n:n(s,p);typeof m=="string"&&(m={type:"text",value:m}),k.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+h+(p>1?"-"+p:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,p),className:["data-footnote-backref"]},children:Array.isArray(m)?m:[m]})}const C=d[d.length-1];if(C&&C.type==="element"&&C.tagName==="p"){const m=C.children[C.children.length-1];m&&m.type==="text"?m.value+=" ":C.children.push({type:"text",value:" "}),C.children.push(...k)}else d.push(...k);const g={type:"element",tagName:"li",properties:{id:t+"fn-"+h},children:e.wrap(d,!0)};e.patch(c,g),a.push(g)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...ml(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const _p=function(e){if(e==null)return m1;if(typeof e=="function")return Ml(e);if(typeof e=="object")return Array.isArray(e)?f1(e):p1(e);if(typeof e=="string")return h1(e);throw new Error("Expected function, string, or object as test")};function f1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=_p(e[n]);return Ml(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function p1(e){const t=e;return Ml(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function h1(e){return Ml(t);function t(n){return n&&n.type===e}}function Ml(e){return t;function t(n,r,i){return!!(g1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function m1(){return!0}function g1(e){return e!==null&&typeof e=="object"&&"type"in e}const zp=[],v1=!0,bc=!1,y1="skip";function x1(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=_p(i),o=r?-1:1;a(e,void 0,[])();function a(s,c,d){const f=s&&typeof s=="object"?s:{};if(typeof f.type=="string"){const p=typeof f.tagName=="string"?f.tagName:typeof f.name=="string"?f.name:void 0;Object.defineProperty(h,"name",{value:"node ("+(s.type+(p?"<"+p+">":""))+")"})}return h;function h(){let p=zp,k,S,C;if((!t||l(s,c,d[d.length-1]||void 0))&&(p=k1(n(s,d)),p[0]===bc))return p;if("children"in s&&s.children){const g=s;if(g.children&&p[0]!==y1)for(S=(r?g.children.length:-1)+o,C=d.concat(g);S>-1&&S<g.children.length;){const m=g.children[S];if(k=a(m,S,C)(),k[0]===bc)return k;S=typeof k[1]=="number"?k[1]:S+o}}return p}}}function k1(e){return Array.isArray(e)?e:typeof e=="number"?[v1,e]:e==null?zp:[e]}function Tp(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),x1(e,l,a,i);function a(s,c){const d=c[c.length-1],f=d?d.children.indexOf(s):void 0;return o(s,f,d)}}const wa={}.hasOwnProperty,w1={};function S1(e,t){const n=t||w1,r=new Map,i=new Map,l=new Map,o={...i1,...n.handlers},a={all:c,applyData:j1,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:b1,wrap:E1};return Tp(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const f=d.type==="definition"?r:i,h=String(d.identifier).toUpperCase();f.has(h)||f.set(h,d)}}),a;function s(d,f){const h=d.type,p=a.handlers[h];if(wa.call(a.handlers,h)&&p)return p(a,d,f);if(a.options.passThrough&&a.options.passThrough.includes(h)){if("children"in d){const{children:S,...C}=d,g=ml(C);return g.children=a.all(d),g}return ml(d)}return(a.options.unknownHandler||C1)(a,d,f)}function c(d){const f=[];if("children"in d){const h=d.children;let p=-1;for(;++p<h.length;){const k=a.one(h[p],d);if(k){if(p&&h[p-1].type==="break"&&(!Array.isArray(k)&&k.type==="text"&&(k.value=jc(k.value)),!Array.isArray(k)&&k.type==="element")){const S=k.children[0];S&&S.type==="text"&&(S.value=jc(S.value))}Array.isArray(k)?f.push(...k):f.push(k)}}}return f}}function b1(e,t){e.position&&(t.position=ov(e))}function j1(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,ml(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function C1(e,t){const n=t.data||{},r="value"in t&&!(wa.call(n,"hProperties")||wa.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function E1(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function jc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Cc(e,t){const n=S1(e,t),r=n.one(e,void 0),i=d1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function N1(e,t){return e&&"run"in e?async function(n,r){const i=Cc(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Cc(n,{file:r,...e||t})}}function Ec(e){if(e)throw e}var $i=Object.prototype.hasOwnProperty,Lp=Object.prototype.toString,Nc=Object.defineProperty,_c=Object.getOwnPropertyDescriptor,zc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Lp.call(t)==="[object Array]"},Tc=function(t){if(!t||Lp.call(t)!=="[object Object]")return!1;var n=$i.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&$i.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||$i.call(t,i)},Lc=function(t,n){Nc&&n.name==="__proto__"?Nc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Pc=function(t,n){if(n==="__proto__")if($i.call(t,n)){if(_c)return _c(t,n).value}else return;return t[n]},_1=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<c;++s)if(t=arguments[s],t!=null)for(n in t)r=Pc(a,n),i=Pc(t,n),a!==i&&(d&&i&&(Tc(i)||(l=zc(i)))?(l?(l=!1,o=r&&zc(r)?r:[]):o=r&&Tc(r)?r:{},Lc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Lc(a,{name:n,newValue:i}));return a};const po=ja(_1);function Sa(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function z1(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...c){const d=e[++l];let f=-1;if(s){o(s);return}for(;++f<i.length;)(c[f]===null||c[f]===void 0)&&(c[f]=i[f]);i=c,d?T1(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function T1(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const vt={basename:L1,dirname:P1,extname:I1,join:D1,sep:"/"};function L1(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');oi(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function P1(e){if(oi(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function I1(e){oi(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function D1(...e){let t=-1,n;for(;++t<e.length;)oi(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":M1(n)}function M1(e){oi(e);const t=e.codePointAt(0)===47;let n=A1(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function A1(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function oi(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const R1={cwd:O1};function O1(){return"/"}function ba(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function F1(e){if(typeof e=="string")e=new URL(e);else if(!ba(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return B1(e)}function B1(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const ho=["history","path","basename","stem","extname","dirname"];class Pp{constructor(t){let n;t?ba(t)?n={path:t}:typeof t=="string"||U1(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":R1.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<ho.length;){const l=ho[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)ho.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?vt.basename(this.path):void 0}set basename(t){go(t,"basename"),mo(t,"basename"),this.path=vt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?vt.dirname(this.path):void 0}set dirname(t){Ic(this.basename,"dirname"),this.path=vt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?vt.extname(this.path):void 0}set extname(t){if(mo(t,"extname"),Ic(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=vt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){ba(t)&&(t=F1(t)),go(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?vt.basename(this.path,this.extname):void 0}set stem(t){go(t,"stem"),mo(t,"stem"),this.path=vt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new ze(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function mo(e,t){if(e&&e.includes(vt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+vt.sep+"`")}function go(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Ic(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function U1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const $1=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},H1={}.hasOwnProperty;class Is extends $1{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=z1()}copy(){const t=new Is;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(po(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(xo("data",this.frozen),this.namespace[t]=n,this):H1.call(this.namespace,t)&&this.namespace[t]||void 0:t?(xo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Ni(t),r=this.parser||this.Parser;return vo("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),vo("process",this.parser||this.Parser),yo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Ni(t),s=r.parse(a);r.run(s,a,function(d,f,h){if(d||!f||!h)return c(d);const p=f,k=r.stringify(p,h);Q1(k)?h.value=k:h.result=k,c(d,h)});function c(d,f){d||!f?o(d):l?l(f):n(void 0,f)}}}processSync(t){let n=!1,r;return this.freeze(),vo("processSync",this.parser||this.Parser),yo("processSync",this.compiler||this.Compiler),this.process(t,i),Mc("processSync","process",n),r;function i(l,o){n=!0,Ec(l),r=o}}run(t,n,r){Dc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Ni(n);i.run(t,s,c);function c(d,f,h){const p=f||t;d?a(d):o?o(p):r(void 0,p,h)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Mc("runSync","run",r),i;function l(o,a){Ec(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Ni(n),i=this.compiler||this.Compiler;return yo("stringify",i),Dc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(xo("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")s(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...f]=c;s(d,f)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=po(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const f=c[d];l(f)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function s(c,d){let f=-1,h=-1;for(;++f<r.length;)if(r[f][0]===c){h=f;break}if(h===-1)r.push([c,...d]);else if(d.length>0){let[p,...k]=d;const S=r[h][1];Sa(S)&&Sa(p)&&(p=po(!0,S,p)),r[h]=[c,p,...k]}}}}const V1=new Is().freeze();function vo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function yo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function xo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Dc(e){if(!Sa(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Mc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Ni(e){return W1(e)?e:new Pp(e)}function W1(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function Q1(e){return typeof e=="string"||K1(e)}function K1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const q1="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Ac=[],Rc={allowDangerousHtml:!0},Y1=/^(https?|ircs?|mailto|xmpp)$/i,X1=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function G1(e){const t=J1(e),n=Z1(e);return e0(t.runSync(t.parse(n),n),e)}function J1(e){const t=e.rehypePlugins||Ac,n=e.remarkPlugins||Ac,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Rc}:Rc;return V1().use(Px).use(n).use(N1,r).use(t)}function Z1(e){const t=e.children||"",n=new Pp;return typeof t=="string"&&(n.value=t),n}function e0(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||t0;for(const d of X1)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+q1+d.id,void 0);return Tp(e,c),dv(e,{Fragment:u.Fragment,components:i,ignoreInvalidStyle:!0,jsx:u.jsx,jsxs:u.jsxs,passKeys:!0,passNode:!0});function c(d,f,h){if(d.type==="raw"&&h&&typeof f=="number")return o?h.children.splice(f,1):h.children[f]={type:"text",value:d.value},f;if(d.type==="element"){let p;for(p in uo)if(Object.hasOwn(uo,p)&&Object.hasOwn(d.properties,p)){const k=d.properties[p],S=uo[p];(S===null||S.includes(d.tagName))&&(d.properties[p]=s(String(k||""),p,d))}}if(d.type==="element"){let p=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!p&&r&&typeof f=="number"&&(p=!r(d,f,h)),p&&h&&typeof f=="number")return a&&d.children?h.children.splice(f,1,...d.children):h.children.splice(f,1),f}}}function t0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||Y1.test(e.slice(0,t))?e:""}const yt={send:u.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),u.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),u.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),u.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]})},n0=e=>{switch(e){case"directive":return yt.directive;case"question":return yt.question;case"status":return yt.status;case"result":return yt.result;case"approval_request":return yt.lock;default:return yt.directive}},r0=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r})=>{const i=$.useRef(null),[l,o]=dn.useState(""),[a,s]=dn.useState("directive"),[c,d]=dn.useState(""),[f,h]=dn.useState(!1);$.useEffect(()=>{e!=null&&e.workspace?d(e.workspace):d("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),$.useEffect(()=>{var m;(m=i.current)==null||m.scrollIntoView({behavior:"smooth"})},[t]);const p=m=>{d(m),r&&r(m)},k=()=>{l.trim()&&(n(l,a,c||void 0),o(""))},S=m=>{m.key==="Enter"&&!m.shiftKey&&(m.preventDefault(),k())},C=m=>new Date(m).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),g=m=>m.length>12?`${m.slice(0,8)}...`:m;return e?u.jsxs("div",{className:"conversation-view",children:[u.jsxs("div",{className:"conversation-header",children:[u.jsxs("div",{className:"header-info",children:[u.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&u.jsxs("span",{className:"thread-agent-badge",children:[yt.bot,e.target_agent]})]}),u.jsxs("div",{className:"header-stats",children:[u.jsxs("span",{className:"message-count",children:[t.length," messages"]}),u.jsx("span",{className:"thread-id",title:e.id,children:g(e.id)})]})]}),u.jsxs("div",{className:"messages-container",children:[t.length===0?u.jsxs("div",{className:"empty-messages",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("p",{children:"No messages yet"}),u.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((m,y)=>{const j=m.from_type==="human",z=y===0||t[y-1].from_type!==m.from_type;return u.jsxs("div",{className:`message ${j?"human":"agent"}`,children:[u.jsx("div",{className:`message-avatar ${z?"visible":""}`,children:z&&(j?yt.user:yt.bot)}),u.jsxs("div",{className:"message-body",children:[z&&u.jsxs("div",{className:"message-meta",children:[u.jsx("span",{className:"sender-name",children:m.from_id}),u.jsxs("span",{className:"kind-badge",children:[n0(m.kind)," ",m.kind]}),u.jsx("span",{className:"message-time",children:C(m.created_at)})]}),u.jsx("div",{className:"message-content",children:m.kind==="result"||!j?u.jsx(G1,{components:{a:({href:w,children:E})=>{let b=w;return w&&w.startsWith("/")&&!w.startsWith("//")&&(b=`file://${w}`),u.jsx("a",{href:b,target:"_blank",rel:"noopener noreferrer",children:E})},code:({className:w,children:E,...b})=>!w?u.jsx("code",{className:"inline-code",...b,children:E}):u.jsx("code",{className:w,...b,children:E})},children:m.content}):m.content}),u.jsxs("div",{className:"message-footer",children:[u.jsxs("span",{className:"message-seq",children:["#",m.message_seq]}),m.delivery_state!=="acked"&&u.jsx("span",{className:`delivery-status ${m.delivery_state}`,children:m.delivery_state==="pending"?"sending...":"delivered"})]})]})]},m.id)}),u.jsx("div",{ref:i})]}),u.jsxs("div",{className:"input-area",children:[f&&u.jsxs("div",{className:"workspace-input-row",children:[u.jsx("input",{type:"text",value:c,onChange:m=>p(m.target.value),onBlur:()=>{r&&r(c)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),u.jsx("button",{onClick:async()=>{try{const y=await(await fetch("/api/select-folder")).json();!y.cancelled&&y.path&&p(y.path)}catch(m){console.error("Failed to open folder picker:",m)}},className:"workspace-browse",title:"Browse for folder",children:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),u.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),u.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),c&&u.jsx("button",{onClick:()=>{p(""),h(!1)},className:"workspace-clear",children:"Clear"})]}),u.jsxs("div",{className:"input-wrapper",children:[u.jsx("button",{onClick:()=>h(!f),className:`workspace-toggle ${c?"has-workspace":""}`,title:c||"Set working directory",children:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),u.jsxs("select",{value:a,onChange:m=>s(m.target.value),className:"kind-selector",children:[u.jsx("option",{value:"directive",children:"Directive"}),u.jsx("option",{value:"question",children:"Question"})]}),u.jsx("textarea",{value:l,onChange:m=>o(m.target.value),onKeyPress:S,placeholder:c?`Message (workspace: ${c.split("/").pop()})`:"Type a message...",rows:1}),u.jsx("button",{onClick:k,className:"send-btn",disabled:!l.trim(),children:yt.send})]}),u.jsxs("div",{className:"input-hint",children:["Press ",u.jsx("kbd",{children:"Enter"})," to send, ",u.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),u.jsx("style",{children:`
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
      `})]}):null},i0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,reconnectInterval:l=5e3})=>{const o=$.useRef(null),[a,s]=$.useState(!1),[c,d]=$.useState(null),f=$.useRef(null),h=$.useRef(new Map),p=$.useCallback(()=>{try{const j=`${e}?instance_id=${t}`;o.current=new WebSocket(j),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),d(null),h.current.forEach((z,w)=>{C(w,z)})},o.current.onmessage=z=>{try{const w=JSON.parse(z.data);k(w)}catch(w){console.error("Failed to parse WebSocket message:",w)}},o.current.onerror=z=>{console.error("WebSocket error:",z),d("Connection error")},o.current.onclose=()=>{console.log("WebSocket disconnected"),s(!1),f.current=setTimeout(()=>{console.log("Attempting to reconnect..."),p()},l)}}catch(j){console.error("Failed to connect to WebSocket:",j),d("Failed to connect")}},[e,t,l]),k=$.useCallback(j=>{switch(j.type){case"message":n&&j.data&&n(j.data);break;case"batch":if(r&&j.data){const z=j.data;r(z),n&&z.messages.forEach(w=>n(w))}break;case"error":i&&j.data&&i(j.data),console.error("WebSocket error event:",j.data);break;case"pong":break;default:console.log("Unknown event type:",j.type)}},[n,r,i]),S=$.useCallback(j=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(j)):console.warn("WebSocket not connected, cannot send event")},[]),C=$.useCallback((j,z=0)=>{h.current.set(j,z);const w={type:"subscribe",timestamp:Date.now(),data:{thread_id:j,from_seq:z}};S(w)},[S]),g=$.useCallback((j,z)=>{const w=h.current.get(j)||0;z>w&&h.current.set(j,z);const E={type:"ack",timestamp:Date.now(),data:{thread_id:j,ack_seq:z}};S(E)},[S]),m=$.useCallback(()=>{const j={type:"ping",timestamp:Date.now()};S(j)},[S]),y=$.useCallback(j=>{h.current.delete(j)},[]);return $.useEffect(()=>(p(),()=>{f.current&&clearTimeout(f.current),o.current&&o.current.close()}),[p]),$.useEffect(()=>{if(!a)return;const j=setInterval(()=>{m()},3e4);return()=>clearInterval(j)},[a,m]),{isConnected:a,connectionError:c,subscribe:C,unsubscribe:y,acknowledge:g,ping:m}},l0=({connected:e})=>u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?u.jsxs(u.Fragment,{children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):u.jsxs(u.Fragment,{children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),u.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),o0=({websocketUrl:e,instanceId:t})=>{const[n,r]=$.useState([]),[i,l]=$.useState(null),[o,a]=$.useState(new Map),[s,c]=$.useState(new Map),[d,f]=$.useState([]),[h,p]=$.useState(!1),[k,S]=$.useState(""),{isConnected:C,subscribe:g,acknowledge:m}=i0({url:e,instanceId:t,onMessage:y,onBatch:j});function y(L){const D={id:L.id,thread_id:L.thread_id,message_seq:L.message_seq,created_at:L.created_at,from_type:L.from_type,from_id:L.from_id,to_type:L.to_type,to_id:L.to_id,kind:L.kind,subject:L.subject,content:L.content,metadata_json:L.metadata_json,delivery_state:"visible",business_state:"open"};a(F=>{const P=F.get(D.thread_id)||[];return P.find(B=>B.id===D.id)?F:new Map(F).set(D.thread_id,[...P,D].sort((B,v)=>B.message_seq-v.message_seq))}),D.thread_id!==i&&c(F=>{const P=F.get(D.thread_id)||0;return new Map(F).set(D.thread_id,P+1)}),m(D.thread_id,D.message_seq)}function j(L){L.messages.forEach(D=>{y(D)})}const z=$.useCallback(L=>{if(l(L),c(D=>{const F=new Map(D);return F.delete(L),F}),C){const D=o.get(L)||[],F=D.length>0?Math.max(...D.map(P=>P.message_seq)):0;g(L,F)}},[C,g,o]),w=$.useCallback(async(L,D,F)=>{if(!i)return;const P=F?JSON.stringify({workspace:F}):void 0;try{const B=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:i,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:D,content:L,metadata_json:P})});if(!B.ok){console.error("Failed to send message:",await B.text());return}const v=await B.json();a(q=>{const J=q.get(i)||[];return J.find(x=>x.id===v.id)?q:new Map(q).set(i,[...J,v])})}catch(B){console.error("Error sending message:",B)}},[i,t]);$.useEffect(()=>{(async()=>{try{const D=await fetch("/api/threads");if(!D.ok){console.error("Failed to fetch threads:",await D.text());return}const F=await D.json();r(F),F.length>0&&!i&&l(F[0].id)}catch(D){console.error("Error fetching threads:",D)}})()},[]);const E=$.useCallback(async L=>{try{const D=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:L,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!D.ok){console.error("Failed to create thread:",await D.text());return}const F=await D.json();r(P=>[F,...P]),l(F.id)}catch(D){console.error("Error creating thread:",D)}},[t]),b=$.useCallback(async()=>{try{const L=await fetch("/api/agents");if(!L.ok){console.error("Failed to fetch agents:",await L.text());return}const D=await L.json();f(D.running||[])}catch(L){console.error("Error fetching agents:",L)}},[]);$.useEffect(()=>{b();const L=setInterval(b,5e3);return()=>clearInterval(L)},[b]);const N=$.useCallback(async()=>{if(k.trim())try{const L=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:k.trim()})});if(!L.ok){const F=await L.text();console.error("Failed to launch agent:",F),alert(`Failed to launch agent: ${F}`);return}const D=await L.json();f(F=>[...F,D]),S(""),p(!1)}catch(L){console.error("Error launching agent:",L)}},[k]),A=$.useCallback(async L=>{try{const D=await fetch(`/api/agents/${L}`,{method:"DELETE"});if(!D.ok){console.error("Failed to stop agent:",await D.text());return}f(F=>F.filter(P=>P.instance_id!==L))}catch(D){console.error("Error stopping agent:",D)}},[]),_=$.useCallback(async L=>{if(i)try{const D=await fetch(`/api/threads/${i}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:L})});if(!D.ok){console.error("Failed to update workspace:",await D.text());return}const F=await D.json();r(P=>P.map(B=>B.id===i?F:B))}catch(D){console.error("Error updating workspace:",D)}},[i]),I=$.useCallback(async L=>{try{const D=await fetch(`/api/threads/${L}`,{method:"DELETE"});if(!D.ok){console.error("Failed to delete thread:",await D.text());return}r(F=>F.filter(P=>P.id!==L)),a(F=>{const P=new Map(F);return P.delete(L),P}),c(F=>{const P=new Map(F);return P.delete(L),P}),i===L&&l(null)}catch(D){console.error("Error deleting thread:",D)}},[i]),H=$.useCallback(async(L,D)=>{try{const F=await fetch(`/api/threads/${L}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:D})});if(!F.ok){console.error("Failed to rename thread:",await F.text());return}const P=await F.json();r(B=>B.map(v=>v.id===L?P:v))}catch(F){console.error("Error renaming thread:",F)}},[]),K=i?o.get(i)||[]:[];return u.jsxs("div",{className:"message-center",children:[u.jsxs("div",{className:"status-bar",children:[u.jsxs("div",{className:`status-indicator ${C?"connected":"disconnected"}`,children:[u.jsx(l0,{connected:C}),u.jsx("span",{children:C?"Connected":"Disconnected"})]}),u.jsxs("div",{className:"status-meta",children:[u.jsxs("span",{className:"thread-count",children:[n.length," threads"]}),u.jsxs("span",{className:"agent-count",children:[d.length," agents"]}),u.jsx("button",{className:"launch-agent-btn",onClick:()=>p(!0),children:"+ Agent"})]})]}),d.length>0&&u.jsx("div",{className:"agents-bar",children:d.map(L=>u.jsxs("div",{className:"agent-chip",children:[u.jsx("span",{className:"agent-pulse"}),u.jsx("span",{className:"agent-name",children:L.instance_id}),u.jsxs("span",{className:"agent-pid",children:["PID ",L.pid]}),u.jsx("button",{className:"agent-stop-btn",onClick:()=>A(L.instance_id),title:"Stop agent",children:"×"})]},L.instance_id))}),h&&u.jsx("div",{className:"modal-overlay",onClick:()=>p(!1),children:u.jsxs("div",{className:"modal-content",onClick:L=>L.stopPropagation(),children:[u.jsx("h3",{children:"Launch New Agent"}),u.jsx("input",{type:"text",value:k,onChange:L=>S(L.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:L=>{L.key==="Enter"&&N(),L.key==="Escape"&&p(!1)}}),u.jsxs("div",{className:"modal-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>p(!1),children:"Cancel"}),u.jsx("button",{className:"launch-btn",onClick:N,children:"Launch"})]})]})}),u.jsxs("div",{className:"center-layout",children:[u.jsx("aside",{className:"threads-panel",children:u.jsx(hg,{threads:n,selectedThreadId:i,onSelectThread:z,onCreateThread:E,onDeleteThread:I,onRenameThread:H,unreadCounts:s})}),u.jsx("main",{className:"conversation-panel",children:i?u.jsx(r0,{thread:n.find(L=>L.id===i),messages:K,onSendMessage:w,onWorkspaceChange:_}):u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("h3",{children:"Select a conversation"}),u.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),u.jsx("style",{children:`
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
      `})]})},He={check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),dollar:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),sparkles:u.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),u.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),u.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},a0=({approvals:e,history:t=[],onApprove:n,onReject:r})=>{const[i,l]=$.useState(!0),[o,a]=$.useState(null),[s,c]=$.useState(new Map),d=C=>{try{return JSON.parse(C)}catch{return null}},f=C=>new Date(C).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),h=C=>{const g=s.get(C)||"";n(C,g),c(new Map(s.set(C,"")))},p=C=>{const g=s.get(C)||"";if(!g.trim()){alert("Please provide a reason for rejection");return}r(C,g),c(new Map(s.set(C,"")))},k=(C,g)=>{c(new Map(s.set(C,g)))},S=e.filter(C=>C.status==="pending");return u.jsxs("div",{className:"approval-queue",children:[u.jsx("div",{className:"queue-header",children:u.jsxs("div",{className:"header-title",children:[u.jsx("h2",{children:"Approval Queue"}),u.jsxs("span",{className:"pending-count",children:[S.length," pending"]})]})}),u.jsxs("div",{className:"approvals-container",children:[S.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:He.sparkles}),u.jsx("h3",{children:"All caught up!"}),u.jsx("p",{children:"No pending approvals to review"})]}):u.jsx("div",{className:"approvals-list",children:S.map(C=>{const g=d(C.effect_delta_json),m=o===C.id;return u.jsxs("div",{className:`approval-card impact-${C.impact}`,children:[u.jsxs("div",{className:"card-header",onClick:()=>a(m?null:C.id),children:[u.jsxs("div",{className:"header-left",children:[u.jsx("div",{className:`impact-indicator ${C.impact}`}),u.jsxs("div",{className:"proposal-info",children:[u.jsx("span",{className:"proposal-text",children:C.proposal}),u.jsxs("div",{className:"proposal-meta",children:[u.jsxs("span",{className:"meta-item",children:[He.bot,C.instance_id]}),u.jsxs("span",{className:"meta-item",children:[He.clock,f(C.created_at)]})]})]})]}),u.jsxs("div",{className:"header-right",children:[u.jsxs("span",{className:"cost-badge",children:[He.dollar,"$",C.estimated_cost.toFixed(2)]}),u.jsx("span",{className:`impact-badge ${C.impact}`,children:C.impact}),u.jsx("button",{className:"expand-btn",children:m?He.chevronUp:He.chevronDown})]})]}),m&&u.jsxs("div",{className:"card-details",children:[g&&u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Effect Details"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Capability"}),u.jsx("span",{className:"detail-value code",children:g.cap_type})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Budget Delta"}),u.jsxs("span",{className:"detail-value",children:["$",g.budget_delta.toFixed(2)]})]}),g.paths.length>0&&u.jsxs("div",{className:"detail-item full-width",children:[u.jsx("span",{className:"detail-label",children:"Paths"}),u.jsx("div",{className:"paths-list",children:g.paths.map((y,j)=>u.jsxs("span",{className:"path-tag",children:[He.folder,y]},j))})]})]})]}),u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Request Info"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Thread"}),u.jsx("span",{className:"detail-value code",children:C.thread_id})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Impact Level"}),u.jsx("span",{className:`detail-value impact-text ${C.impact}`,children:C.impact.toUpperCase()})]})]})]}),u.jsxs("div",{className:"review-section",children:[u.jsx("h4",{children:"Review Notes"}),u.jsx("textarea",{value:s.get(C.id)||"",onChange:y=>k(C.id,y.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),u.jsxs("div",{className:"action-buttons",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>p(C.id),children:[He.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>h(C.id),children:[He.check,"Approve"]})]})]})]})]},C.id)})}),t.length>0&&u.jsxs("div",{className:"history-section",children:[u.jsxs("div",{className:"history-header",onClick:()=>l(!i),children:[u.jsxs("h3",{children:[i?He.chevronDown:He.chevronUp,"Review History"]}),u.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),i&&u.jsx("div",{className:"history-list",children:t.map(C=>{const g=o===`history-${C.id}`;return u.jsxs("div",{className:`history-card ${C.status}`,onClick:()=>a(g?null:`history-${C.id}`),children:[u.jsxs("div",{className:"history-card-header",children:[u.jsxs("div",{className:"history-status",children:[u.jsx("span",{className:`status-icon ${C.status}`,children:C.status==="approved"?He.check:He.x}),u.jsx("span",{className:"history-proposal",children:C.proposal})]}),u.jsxs("div",{className:"history-meta",children:[u.jsx("span",{className:"history-agent",children:C.instance_id}),u.jsx("span",{className:`history-badge ${C.status}`,children:C.status}),u.jsx("span",{className:"history-time",children:C.reviewed_at?f(C.reviewed_at):f(C.created_at)})]})]}),g&&u.jsxs("div",{className:"history-details",children:[u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Reviewed by"}),u.jsx("span",{className:"detail-value",children:C.reviewed_by||"Unknown"})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Cost"}),u.jsxs("span",{className:"detail-value",children:["$",C.estimated_cost.toFixed(2)]})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Impact"}),u.jsx("span",{className:`detail-value impact-text ${C.impact}`,children:C.impact.toUpperCase()})]}),C.review_notes&&u.jsxs("div",{className:"detail-row full-width",children:[u.jsx("span",{className:"detail-label",children:"Notes"}),u.jsx("span",{className:"detail-value notes",children:C.review_notes})]})]})]},`history-${C.id}`)})})]})]}),u.jsx("style",{children:`
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
          align-items: center;
          gap: var(--space-2);
          flex: 1;
          min-width: 0;
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
      `})]})},he={cpu:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"4",y:"4",width:"16",height:"16",rx:"2"}),u.jsx("rect",{x:"9",y:"9",width:"6",height:"6"}),u.jsx("line",{x1:"9",y1:"1",x2:"9",y2:"4"}),u.jsx("line",{x1:"15",y1:"1",x2:"15",y2:"4"}),u.jsx("line",{x1:"9",y1:"20",x2:"9",y2:"23"}),u.jsx("line",{x1:"15",y1:"20",x2:"15",y2:"23"}),u.jsx("line",{x1:"20",y1:"9",x2:"23",y2:"9"}),u.jsx("line",{x1:"20",y1:"14",x2:"23",y2:"14"}),u.jsx("line",{x1:"1",y1:"9",x2:"4",y2:"9"}),u.jsx("line",{x1:"1",y1:"14",x2:"4",y2:"14"})]}),memory:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"2",y:"6",width:"20",height:"12",rx:"2"}),u.jsx("line",{x1:"6",y1:"10",x2:"6",y2:"14"}),u.jsx("line",{x1:"10",y1:"10",x2:"10",y2:"14"}),u.jsx("line",{x1:"14",y1:"10",x2:"14",y2:"14"}),u.jsx("line",{x1:"18",y1:"10",x2:"18",y2:"14"})]}),clock:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),dollar:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),activity:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"22 12 18 12 15 21 9 3 6 12 2 12"})}),tokens:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"}),u.jsx("line",{x1:"10",y1:"9",x2:"8",y2:"9"})]}),message:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),stop:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("rect",{x:"3",y:"3",width:"18",height:"18",rx:"2"})}),warning:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"}),u.jsx("line",{x1:"12",y1:"9",x2:"12",y2:"13"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),check:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})})},s0=()=>{const[e,t]=$.useState(null),[n,r]=$.useState(null),[i,l]=$.useState(null),[o,a]=$.useState(new Map),[s,c]=$.useState({prevCost:0,prevTokensIn:0,prevTokensOut:0,timestamp:0}),d=$.useCallback(async()=>{try{const b=await fetch("/api/monitor");if(!b.ok)throw new Error(`Failed to fetch: ${b.statusText}`);const N=await b.json();t(N),l(new Date),r(null)}catch(b){r(b instanceof Error?b.message:"Unknown error")}},[]);$.useEffect(()=>{d();const b=setInterval(d,2e3);return()=>clearInterval(b)},[d]),$.useEffect(()=>{const N=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;let A=null,_=null;const I=()=>{A=new WebSocket(N),A.onmessage=H=>{try{const K=JSON.parse(H.data);if(K.type==="telemetry"){const L=K.data;a(D=>{const F=new Map(D);return F.set(L.instance_id,L),F})}}catch{}},A.onclose=()=>{_=setTimeout(I,3e3)},A.onerror=()=>{A==null||A.close()}};return I(),()=>{_&&clearTimeout(_),A==null||A.close()}},[]);const f=async b=>{try{await fetch(`/api/agents/${b}`,{method:"DELETE"}),d()}catch(N){console.error("Failed to stop process:",N)}},h=b=>{if(b<0)return"Unknown";if(b<60)return`${b}s`;if(b<3600){const _=Math.floor(b/60),I=b%60;return`${_}m ${I}s`}const N=Math.floor(b/3600),A=Math.floor(b%3600/60);return`${N}h ${A}m`},p=b=>b===0?"$0.00":b<.01?`$${b.toFixed(4)}`:`$${b.toFixed(2)}`,k=b=>{switch(b){case"running":return"var(--color-success)";case"completed":return"var(--color-primary)";case"failed":return"var(--color-danger)";case"orphan":return"var(--color-warning)";default:return"var(--text-tertiary)"}},S=b=>b.cpu_percent>80||b.duration_sec>300,C=b=>{const N=o.get(b.instance_id);return N?{...b,turns:N.turns,tokens_in:N.tokens_in,tokens_out:N.tokens_out,cost:N.cost,hasLiveTelemetry:!0}:{...b,hasLiveTelemetry:!1}},g=Array.from(o.values()).reduce((b,N)=>({tokens_in:b.tokens_in+N.tokens_in,tokens_out:b.tokens_out+N.tokens_out,cost:b.cost+N.cost,turns:b.turns+N.turns}),{tokens_in:0,tokens_out:0,cost:0,turns:0}),m=g.cost>0?g.cost:(e==null?void 0:e.summary.total_cost)||0,y={in:g.tokens_in,out:g.tokens_out},j=o.size>0;$.useEffect(()=>{if(j){const b=Date.now();b-s.timestamp>2e3&&c({prevCost:m,prevTokensIn:y.in,prevTokensOut:y.out,timestamp:b})}},[m,y.in,y.out,j,s.timestamp]);const z=m>s.prevCost&&s.prevCost>0?"up":null,w=y.in+y.out>s.prevTokensIn+s.prevTokensOut&&s.prevTokensIn+s.prevTokensOut>0?"up":null,E=b=>b>=1e6?`${(b/1e6).toFixed(1)}M`:b>=1e3?`${(b/1e3).toFixed(1)}K`:b.toString();return u.jsxs("div",{className:"monitor",children:[u.jsxs("div",{className:"monitor-summary",children:[u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:he.activity}),u.jsx("span",{className:"summary-value",children:(e==null?void 0:e.summary.total_processes)||0}),u.jsx("span",{className:"summary-label",children:"Running"})]}),u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:he.cpu}),u.jsxs("span",{className:"summary-value",children:[(e==null?void 0:e.summary.total_cpu_percent.toFixed(1))||"0.0","%"]}),u.jsx("span",{className:"summary-label",children:"CPU"})]}),u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:he.memory}),u.jsxs("span",{className:"summary-value",children:[(e==null?void 0:e.summary.total_memory_mb.toFixed(0))||"0"," MB"]}),u.jsx("span",{className:"summary-label",children:"Memory"})]}),u.jsxs("div",{className:`summary-item ${j?"live":""}`,children:[u.jsx("span",{className:"summary-icon",children:he.dollar}),u.jsxs("span",{className:"summary-value",children:[p(m),z==="up"&&u.jsx("span",{className:"trend-up",children:"▲"})]}),u.jsxs("span",{className:"summary-label",children:["Cost ",j&&u.jsx("span",{className:"live-indicator",children:"●"})]})]}),u.jsxs("div",{className:`summary-item ${j?"live":""}`,children:[u.jsx("span",{className:"summary-icon",children:he.tokens}),u.jsxs("span",{className:"summary-value",children:[E(y.in),"↓ / ",E(y.out),"↑",w==="up"&&u.jsx("span",{className:"trend-up",children:"▲"})]}),u.jsxs("span",{className:"summary-label",children:["Tokens ",j&&u.jsx("span",{className:"live-indicator",children:"●"})]})]}),g.turns>0&&u.jsxs("div",{className:"summary-item live",children:[u.jsx("span",{className:"summary-icon",children:he.message}),u.jsx("span",{className:"summary-value",children:g.turns}),u.jsxs("span",{className:"summary-label",children:["Turns ",u.jsx("span",{className:"live-indicator",children:"●"})]})]}),((e==null?void 0:e.summary.warning_count)||0)>0&&u.jsxs("div",{className:"summary-item warning",children:[u.jsx("span",{className:"summary-icon",children:he.warning}),u.jsx("span",{className:"summary-value",children:e==null?void 0:e.summary.warning_count}),u.jsx("span",{className:"summary-label",children:"Warnings"})]}),u.jsx("div",{className:"summary-spacer"}),u.jsxs("div",{className:"summary-update",children:[j&&u.jsx("span",{className:"live-badge-summary",children:"LIVE"}),"Last update: ",i?i.toLocaleTimeString():"Never"]})]}),u.jsxs("div",{className:"process-grid",children:[n&&u.jsxs("div",{className:"error-card",children:[u.jsx("span",{className:"error-icon",children:he.warning}),u.jsx("span",{children:n})]}),(!(e!=null&&e.processes)||e.processes.length===0)&&!n&&u.jsxs("div",{className:"empty-state",children:[u.jsx("span",{className:"empty-icon",children:he.activity}),u.jsx("h3",{children:"No Active Processes"}),u.jsx("p",{children:"Spawn an agent from the Messages tab to see it here."})]}),e==null?void 0:e.processes.map(b=>{const N=C(b);return u.jsxs("div",{className:`process-card ${S(N)?"warning":""} ${N.hasLiveTelemetry?"live":""}`,children:[u.jsxs("div",{className:"process-header",children:[u.jsxs("div",{className:"process-status",children:[u.jsx("span",{className:"status-dot",style:{background:k(N.status)}}),u.jsx("span",{className:"process-name",children:N.instance_id}),N.hasLiveTelemetry&&u.jsx("span",{className:"live-badge",children:"LIVE"})]}),N.status==="running"&&u.jsx("button",{className:"stop-btn",onClick:()=>f(N.instance_id),title:"Stop process",children:he.stop}),N.status==="completed"&&u.jsxs("span",{className:"status-badge completed",children:[he.check," Done"]})]}),u.jsxs("div",{className:"process-metrics",children:[u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:he.cpu}),u.jsxs("span",{className:`metric-value ${N.cpu_percent>80?"high":""}`,children:[N.cpu_percent.toFixed(1),"%"]}),u.jsx("span",{className:"metric-label",children:"CPU"})]}),u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:he.memory}),u.jsxs("span",{className:"metric-value",children:[N.memory_mb.toFixed(0)," MB"]}),u.jsx("span",{className:"metric-label",children:"Memory"})]}),u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:he.clock}),u.jsx("span",{className:`metric-value ${N.duration_sec>300?"high":""}`,children:h(N.duration_sec)}),u.jsx("span",{className:"metric-label",children:"Duration"})]})]}),N.hasLiveTelemetry&&u.jsxs("div",{className:"process-telemetry",children:[u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:he.message}),u.jsx("span",{className:"telemetry-value",children:N.turns||0}),u.jsx("span",{className:"telemetry-label",children:"Turns"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:he.tokens}),u.jsx("span",{className:"telemetry-value",children:E(N.tokens_in||0)}),u.jsx("span",{className:"telemetry-label",children:"In"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:he.tokens}),u.jsx("span",{className:"telemetry-value",children:E(N.tokens_out||0)}),u.jsx("span",{className:"telemetry-label",children:"Out"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:he.dollar}),u.jsx("span",{className:"telemetry-value cost",children:p(N.cost||0)}),u.jsx("span",{className:"telemetry-label",children:"Cost"})]})]}),u.jsxs("div",{className:"process-footer",children:[u.jsxs("span",{className:"process-pid",children:["PID: ",N.pid]}),N.source&&u.jsx("span",{className:`source-badge ${N.source}`,children:N.source}),N.command&&u.jsx("span",{className:"process-command",title:N.full_cmd,children:N.command}),!N.hasLiveTelemetry&&N.turns&&u.jsxs("span",{className:"process-turns",children:[N.turns," turns"]}),!N.hasLiveTelemetry&&N.cost!==void 0&&N.cost>0&&u.jsx("span",{className:"process-cost",children:p(N.cost)})]})]},N.instance_id)}),(e==null?void 0:e.history)&&e.history.length>0&&u.jsxs(u.Fragment,{children:[u.jsx("div",{className:"history-divider",children:u.jsx("span",{children:"Recent History"})}),e.history.map(b=>u.jsxs("div",{className:`process-card history ${b.status==="failed"?"failed":""}`,children:[u.jsx("div",{className:"process-header",children:u.jsxs("div",{className:"process-status",children:[u.jsx("span",{className:"status-dot",style:{background:k(b.status)}}),u.jsx("span",{className:"process-name",children:b.instance_id}),u.jsxs("span",{className:`status-badge ${b.status}`,children:[b.status==="completed"?he.check:he.warning,b.status]})]})}),u.jsxs("div",{className:"process-footer",children:[u.jsxs("span",{className:"process-pid",children:["PID: ",b.pid]}),b.source&&u.jsx("span",{className:`source-badge ${b.source}`,children:b.source}),b.command&&u.jsx("span",{className:"process-command",title:b.full_cmd,children:b.command}),u.jsx("span",{className:"process-duration",children:h(b.duration_sec)}),b.cost!==void 0&&b.cost>0&&u.jsx("span",{className:"process-cost",children:p(b.cost)})]})]},`history-${b.instance_id}-${b.stopped_at}`))]})]}),u.jsx("style",{children:`
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
      `})]})},_i={messages:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),shield:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"})}),activity:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"22 12 18 12 15 21 9 3 6 12 2 12"})}),logo:u.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M12 6v12M6 12h12"}),u.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]})},u0=()=>{const[e,t]=$.useState("messages"),[n,r]=$.useState([]),[i,l]=$.useState([]),[o,a]=$.useState("my-agent"),[s,c]=$.useState([]),[d,f]=$.useState(""),[h,p]=$.useState(!1),S=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;dn.useEffect(()=>{const E=async()=>{try{const N=await fetch("/api/agents");if(N.ok){const A=await N.json();c(A),A.length>0&&!o&&a(A[0].id)}}catch(N){console.error("Error fetching agents:",N)}};E();const b=setInterval(E,1e4);return()=>clearInterval(b)},[]);const C=E=>{const b=E.target.value;b==="__custom__"?p(!0):(a(b),p(!1))},g=()=>{d.trim()&&(a(d.trim()),p(!1),f(""))},m=E=>E.last_active?Date.now()-E.last_active<3e4:!1,y=E=>m(E)?"●":"○",j=async(E,b)=>{try{const N=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:b})});if(!N.ok){console.error("Failed to approve:",await N.text());return}const A=n.find(_=>_.id===E);if(A){const _={...A,status:"approved",reviewed_by:"user",review_notes:b,reviewed_at:Date.now()};l(I=>[_,...I])}r(_=>_.filter(I=>I.id!==E))}catch(N){console.error("Error approving:",N)}},z=async(E,b)=>{try{const N=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:b})});if(!N.ok){console.error("Failed to reject:",await N.text());return}const A=n.find(_=>_.id===E);if(A){const _={...A,status:"rejected",reviewed_by:"user",review_notes:b,reviewed_at:Date.now()};l(I=>[_,...I])}r(_=>_.filter(I=>I.id!==E))}catch(N){console.error("Error rejecting:",N)}};dn.useEffect(()=>{const E=async()=>{try{const N=await fetch("/api/approvals?status=pending");if(N.ok){const H=await N.json();r(H)}const[A,_]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),I=[];if(A.ok){const H=await A.json();I.push(...H)}if(_.ok){const H=await _.json();I.push(...H)}I.sort((H,K)=>{const L=H.reviewed_at?new Date(H.reviewed_at).getTime():0;return(K.reviewed_at?new Date(K.reviewed_at).getTime():0)-L}),l(I)}catch(N){console.error("Error fetching approvals:",N)}};E();const b=setInterval(E,5e3);return()=>clearInterval(b)},[]);const w=(n==null?void 0:n.filter(E=>E.status==="pending").length)||0;return u.jsxs("div",{className:"app",children:[u.jsxs("header",{className:"app-header",children:[u.jsxs("div",{className:"header-brand",children:[u.jsx("div",{className:"brand-logo",children:_i.logo}),u.jsxs("div",{className:"brand-text",children:[u.jsx("h1",{children:"AILANG"}),u.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),u.jsxs("nav",{className:"header-nav",children:[u.jsxs("button",{className:`nav-tab ${e==="messages"?"active":""}`,onClick:()=>t("messages"),children:[u.jsx("span",{className:"nav-icon",children:_i.messages}),u.jsx("span",{className:"nav-label",children:"Messages"})]}),u.jsxs("button",{className:`nav-tab ${e==="approvals"?"active":""}`,onClick:()=>t("approvals"),children:[u.jsx("span",{className:"nav-icon",children:_i.shield}),u.jsx("span",{className:"nav-label",children:"Approvals"}),w>0&&u.jsx("span",{className:"nav-badge",children:w})]}),u.jsxs("button",{className:`nav-tab ${e==="monitor"?"active":""}`,onClick:()=>t("monitor"),children:[u.jsx("span",{className:"nav-icon",children:_i.activity}),u.jsx("span",{className:"nav-label",children:"Monitor"})]})]}),u.jsxs("div",{className:"header-meta",children:[u.jsxs("div",{className:"agent-selector",children:[u.jsx("label",{className:"agent-label",children:"Target:"}),h?u.jsxs("div",{className:"custom-agent-input",children:[u.jsx("input",{type:"text",value:d,onChange:E=>f(E.target.value),onKeyDown:E=>E.key==="Enter"&&g(),className:"agent-input",placeholder:"agent-id",autoFocus:!0}),u.jsx("button",{onClick:g,className:"agent-apply",children:"Add"}),u.jsx("button",{onClick:()=>p(!1),className:"agent-cancel",children:"Cancel"})]}):u.jsxs(u.Fragment,{children:[u.jsxs("select",{value:o,onChange:C,className:"agent-select",children:[s.map(E=>u.jsxs("option",{value:E.id,children:[y(E)," ",E.id]},E.id)),!s.find(E=>E.id===o)&&o&&u.jsxs("option",{value:o,children:["○ ",o]}),u.jsx("option",{value:"__custom__",children:"+ Add custom..."})]}),s.find(E=>E.id===o)&&u.jsx("span",{className:`agent-status ${m(s.find(E=>E.id===o))?"active":"inactive"}`,children:m(s.find(E=>E.id===o))?"Online":"Offline"})]})]}),u.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),u.jsxs("main",{className:"app-content",children:[e==="messages"&&u.jsx(o0,{websocketUrl:S,instanceId:o}),e==="approvals"&&u.jsx(a0,{approvals:n,history:i,onApprove:j,onReject:z}),e==="monitor"&&u.jsx(s0,{})]}),u.jsx("style",{children:`
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
      `})]})};ko.createRoot(document.getElementById("root")).render(u.jsx(dn.StrictMode,{children:u.jsx(u0,{})}));
