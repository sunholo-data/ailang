(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Hi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function ba(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Fc={exports:{}},ml={},Bc={exports:{}},X={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var Zr=Symbol.for("react.element"),Vp=Symbol.for("react.portal"),$p=Symbol.for("react.fragment"),Wp=Symbol.for("react.strict_mode"),Qp=Symbol.for("react.profiler"),qp=Symbol.for("react.provider"),Kp=Symbol.for("react.context"),Yp=Symbol.for("react.forward_ref"),Xp=Symbol.for("react.suspense"),Gp=Symbol.for("react.memo"),Jp=Symbol.for("react.lazy"),Fs=Symbol.iterator;function Zp(e){return e===null||typeof e!="object"?null:(e=Fs&&e[Fs]||e["@@iterator"],typeof e=="function"?e:null)}var Uc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Hc=Object.assign,Vc={};function nr(e,t,n){this.props=e,this.context=t,this.refs=Vc,this.updater=n||Uc}nr.prototype.isReactComponent={};nr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};nr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function $c(){}$c.prototype=nr.prototype;function Ea(e,t,n){this.props=e,this.context=t,this.refs=Vc,this.updater=n||Uc}var ja=Ea.prototype=new $c;ja.constructor=Ea;Hc(ja,nr.prototype);ja.isPureReactComponent=!0;var Bs=Array.isArray,Wc=Object.prototype.hasOwnProperty,Na={current:null},Qc={key:!0,ref:!0,__self:!0,__source:!0};function qc(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)Wc.call(t,r)&&!Qc.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),u=0;u<a;u++)s[u]=arguments[u+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:Zr,type:e,key:l,ref:o,props:i,_owner:Na.current}}function eh(e,t){return{$$typeof:Zr,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function _a(e){return typeof e=="object"&&e!==null&&e.$$typeof===Zr}function th(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Us=/\/+/g;function Dl(e,t){return typeof e=="object"&&e!==null&&e.key!=null?th(""+e.key):t.toString(36)}function _i(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case Zr:case Vp:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Dl(o,0):r,Bs(i)?(n="",e!=null&&(n=e.replace(Us,"$&/")+"/"),_i(i,t,n,"",function(u){return u})):i!=null&&(_a(i)&&(i=eh(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Us,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Bs(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Dl(l,a);o+=_i(l,t,n,s,i)}else if(s=Zp(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Dl(l,a++),o+=_i(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function ai(e,t,n){if(e==null)return e;var r=[],i=0;return _i(e,r,"","",function(l){return t.call(n,l,i++)}),r}function nh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var De={current:null},zi={transition:null},rh={ReactCurrentDispatcher:De,ReactCurrentBatchConfig:zi,ReactCurrentOwner:Na};function Kc(){throw Error("act(...) is not supported in production builds of React.")}X.Children={map:ai,forEach:function(e,t,n){ai(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return ai(e,function(){t++}),t},toArray:function(e){return ai(e,function(t){return t})||[]},only:function(e){if(!_a(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};X.Component=nr;X.Fragment=$p;X.Profiler=Qp;X.PureComponent=Ea;X.StrictMode=Wp;X.Suspense=Xp;X.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=rh;X.act=Kc;X.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Hc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Na.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)Wc.call(t,s)&&!Qc.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var u=0;u<s;u++)a[u]=arguments[u+2];r.children=a}return{$$typeof:Zr,type:e.type,key:i,ref:l,props:r,_owner:o}};X.createContext=function(e){return e={$$typeof:Kp,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:qp,_context:e},e.Consumer=e};X.createElement=qc;X.createFactory=function(e){var t=qc.bind(null,e);return t.type=e,t};X.createRef=function(){return{current:null}};X.forwardRef=function(e){return{$$typeof:Yp,render:e}};X.isValidElement=_a;X.lazy=function(e){return{$$typeof:Jp,_payload:{_status:-1,_result:e},_init:nh}};X.memo=function(e,t){return{$$typeof:Gp,type:e,compare:t===void 0?null:t}};X.startTransition=function(e){var t=zi.transition;zi.transition={};try{e()}finally{zi.transition=t}};X.unstable_act=Kc;X.useCallback=function(e,t){return De.current.useCallback(e,t)};X.useContext=function(e){return De.current.useContext(e)};X.useDebugValue=function(){};X.useDeferredValue=function(e){return De.current.useDeferredValue(e)};X.useEffect=function(e,t){return De.current.useEffect(e,t)};X.useId=function(){return De.current.useId()};X.useImperativeHandle=function(e,t,n){return De.current.useImperativeHandle(e,t,n)};X.useInsertionEffect=function(e,t){return De.current.useInsertionEffect(e,t)};X.useLayoutEffect=function(e,t){return De.current.useLayoutEffect(e,t)};X.useMemo=function(e,t){return De.current.useMemo(e,t)};X.useReducer=function(e,t,n){return De.current.useReducer(e,t,n)};X.useRef=function(e){return De.current.useRef(e)};X.useState=function(e){return De.current.useState(e)};X.useSyncExternalStore=function(e,t,n){return De.current.useSyncExternalStore(e,t,n)};X.useTransition=function(){return De.current.useTransition()};X.version="18.3.1";Bc.exports=X;var U=Bc.exports;const Ft=ba(U);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ih=U,lh=Symbol.for("react.element"),oh=Symbol.for("react.fragment"),ah=Object.prototype.hasOwnProperty,sh=ih.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,uh={key:!0,ref:!0,__self:!0,__source:!0};function Yc(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)ah.call(t,r)&&!uh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:lh,type:e,key:l,ref:o,props:i,_owner:sh.current}}ml.Fragment=oh;ml.jsx=Yc;ml.jsxs=Yc;Fc.exports=ml;var c=Fc.exports,ko={},Xc={exports:{}},Ze={},Gc={exports:{}},Jc={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(T,E){var v=T.length;T.push(E);e:for(;0<v;){var L=v-1>>>1,B=T[L];if(0<i(B,E))T[L]=E,T[v]=B,v=L;else break e}}function n(T){return T.length===0?null:T[0]}function r(T){if(T.length===0)return null;var E=T[0],v=T.pop();if(v!==E){T[0]=v;e:for(var L=0,B=T.length,x=B>>>1;L<x;){var te=2*(L+1)-1,ke=T[te],Q=te+1,ve=T[Q];if(0>i(ke,v))Q<B&&0>i(ve,ke)?(T[L]=ve,T[Q]=v,L=Q):(T[L]=ke,T[te]=v,L=te);else if(Q<B&&0>i(ve,v))T[L]=ve,T[Q]=v,L=Q;else break e}}return E}function i(T,E){var v=T.sortIndex-E.sortIndex;return v!==0?v:T.id-E.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],u=[],d=1,f=null,g=3,m=!1,S=!1,C=!1,j=typeof setTimeout=="function"?setTimeout:null,p=typeof clearTimeout=="function"?clearTimeout:null,h=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(T){for(var E=n(u);E!==null;){if(E.callback===null)r(u);else if(E.startTime<=T)r(u),E.sortIndex=E.expirationTime,t(s,E);else break;E=n(u)}}function k(T){if(C=!1,y(T),!S)if(n(s)!==null)S=!0,P(b);else{var E=n(u);E!==null&&V(k,E.startTime-T)}}function b(T,E){S=!1,C&&(C=!1,p(D),D=-1),m=!0;var v=g;try{for(y(E),f=n(s);f!==null&&(!(f.expirationTime>E)||T&&!_());){var L=f.callback;if(typeof L=="function"){f.callback=null,g=f.priorityLevel;var B=L(f.expirationTime<=E);E=e.unstable_now(),typeof B=="function"?f.callback=B:f===n(s)&&r(s),y(E)}else r(s);f=n(s)}if(f!==null)var x=!0;else{var te=n(u);te!==null&&V(k,te.startTime-E),x=!1}return x}finally{f=null,g=v,m=!1}}var w=!1,z=null,D=-1,H=5,O=-1;function _(){return!(e.unstable_now()-O<H)}function M(){if(z!==null){var T=e.unstable_now();O=T;var E=!0;try{E=z(!0,T)}finally{E?Y():(w=!1,z=null)}}else w=!1}var Y;if(typeof h=="function")Y=function(){h(M)};else if(typeof MessageChannel<"u"){var G=new MessageChannel,$=G.port2;G.port1.onmessage=M,Y=function(){$.postMessage(null)}}else Y=function(){j(M,0)};function P(T){z=T,w||(w=!0,Y())}function V(T,E){D=j(function(){T(e.unstable_now())},E)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(T){T.callback=null},e.unstable_continueExecution=function(){S||m||(S=!0,P(b))},e.unstable_forceFrameRate=function(T){0>T||125<T?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):H=0<T?Math.floor(1e3/T):5},e.unstable_getCurrentPriorityLevel=function(){return g},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(T){switch(g){case 1:case 2:case 3:var E=3;break;default:E=g}var v=g;g=E;try{return T()}finally{g=v}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(T,E){switch(T){case 1:case 2:case 3:case 4:case 5:break;default:T=3}var v=g;g=T;try{return E()}finally{g=v}},e.unstable_scheduleCallback=function(T,E,v){var L=e.unstable_now();switch(typeof v=="object"&&v!==null?(v=v.delay,v=typeof v=="number"&&0<v?L+v:L):v=L,T){case 1:var B=-1;break;case 2:B=250;break;case 5:B=1073741823;break;case 4:B=1e4;break;default:B=5e3}return B=v+B,T={id:d++,callback:E,priorityLevel:T,startTime:v,expirationTime:B,sortIndex:-1},v>L?(T.sortIndex=v,t(u,T),n(s)===null&&T===n(u)&&(C?(p(D),D=-1):C=!0,V(k,v-L))):(T.sortIndex=B,t(s,T),S||m||(S=!0,P(b))),T},e.unstable_shouldYield=_,e.unstable_wrapCallback=function(T){var E=g;return function(){var v=g;g=E;try{return T.apply(this,arguments)}finally{g=v}}}})(Jc);Gc.exports=Jc;var ch=Gc.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var dh=U,Je=ch;function I(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var Zc=new Set,Dr={};function Sn(e,t){Yn(e,t),Yn(e+"Capture",t)}function Yn(e,t){for(Dr[e]=t,e=0;e<t.length;e++)Zc.add(t[e])}var Lt=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),wo=Object.prototype.hasOwnProperty,fh=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,Hs={},Vs={};function ph(e){return wo.call(Vs,e)?!0:wo.call(Hs,e)?!1:fh.test(e)?Vs[e]=!0:(Hs[e]=!0,!1)}function hh(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function mh(e,t,n,r){if(t===null||typeof t>"u"||hh(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Me(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ee={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ee[e]=new Me(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ee[t]=new Me(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ee[e]=new Me(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ee[e]=new Me(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ee[e]=new Me(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ee[e]=new Me(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ee[e]=new Me(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ee[e]=new Me(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ee[e]=new Me(e,5,!1,e.toLowerCase(),null,!1,!1)});var za=/[\-:]([a-z])/g;function Pa(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(za,Pa);Ee[t]=new Me(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(za,Pa);Ee[t]=new Me(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(za,Pa);Ee[t]=new Me(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ee[e]=new Me(e,1,!1,e.toLowerCase(),null,!1,!1)});Ee.xlinkHref=new Me("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ee[e]=new Me(e,1,!1,e.toLowerCase(),null,!0,!0)});function Ta(e,t,n,r){var i=Ee.hasOwnProperty(t)?Ee[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(mh(t,n,i,r)&&(n=null),r||i===null?ph(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Mt=dh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,si=Symbol.for("react.element"),zn=Symbol.for("react.portal"),Pn=Symbol.for("react.fragment"),La=Symbol.for("react.strict_mode"),So=Symbol.for("react.profiler"),ed=Symbol.for("react.provider"),td=Symbol.for("react.context"),Ia=Symbol.for("react.forward_ref"),Co=Symbol.for("react.suspense"),bo=Symbol.for("react.suspense_list"),Aa=Symbol.for("react.memo"),Bt=Symbol.for("react.lazy"),nd=Symbol.for("react.offscreen"),$s=Symbol.iterator;function ur(e){return e===null||typeof e!="object"?null:(e=$s&&e[$s]||e["@@iterator"],typeof e=="function"?e:null)}var fe=Object.assign,Ml;function xr(e){if(Ml===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Ml=t&&t[1]||""}return`
`+Ml+e}var Rl=!1;function Ol(e,t){if(!e||Rl)return"";Rl=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(u){var r=u}Reflect.construct(e,[],t)}else{try{t.call()}catch(u){r=u}e.call(t.prototype)}else{try{throw Error()}catch(u){r=u}e()}}catch(u){if(u&&r&&typeof u.stack=="string"){for(var i=u.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Rl=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?xr(e):""}function gh(e){switch(e.tag){case 5:return xr(e.type);case 16:return xr("Lazy");case 13:return xr("Suspense");case 19:return xr("SuspenseList");case 0:case 2:case 15:return e=Ol(e.type,!1),e;case 11:return e=Ol(e.type.render,!1),e;case 1:return e=Ol(e.type,!0),e;default:return""}}function Eo(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Pn:return"Fragment";case zn:return"Portal";case So:return"Profiler";case La:return"StrictMode";case Co:return"Suspense";case bo:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case td:return(e.displayName||"Context")+".Consumer";case ed:return(e._context.displayName||"Context")+".Provider";case Ia:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Aa:return t=e.displayName||null,t!==null?t:Eo(e.type)||"Memo";case Bt:t=e._payload,e=e._init;try{return Eo(e(t))}catch{}}return null}function vh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Eo(t);case 8:return t===La?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function en(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function rd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function yh(e){var t=rd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function ui(e){e._valueTracker||(e._valueTracker=yh(e))}function id(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=rd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Vi(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function jo(e,t){var n=t.checked;return fe({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function Ws(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=en(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function ld(e,t){t=t.checked,t!=null&&Ta(e,"checked",t,!1)}function No(e,t){ld(e,t);var n=en(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?_o(e,t.type,n):t.hasOwnProperty("defaultValue")&&_o(e,t.type,en(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function Qs(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function _o(e,t,n){(t!=="number"||Vi(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var kr=Array.isArray;function Un(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+en(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function zo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(I(91));return fe({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function qs(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(I(92));if(kr(n)){if(1<n.length)throw Error(I(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:en(n)}}function od(e,t){var n=en(t.value),r=en(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function Ks(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function ad(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Po(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?ad(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var ci,sd=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(ci=ci||document.createElement("div"),ci.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=ci.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Mr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Cr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},xh=["Webkit","ms","Moz","O"];Object.keys(Cr).forEach(function(e){xh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Cr[t]=Cr[e]})});function ud(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Cr.hasOwnProperty(e)&&Cr[e]?(""+t).trim():t+"px"}function cd(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=ud(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var kh=fe({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function To(e,t){if(t){if(kh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(I(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(I(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(I(61))}if(t.style!=null&&typeof t.style!="object")throw Error(I(62))}}function Lo(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Io=null;function Da(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Ao=null,Hn=null,Vn=null;function Ys(e){if(e=ni(e)){if(typeof Ao!="function")throw Error(I(280));var t=e.stateNode;t&&(t=kl(t),Ao(e.stateNode,e.type,t))}}function dd(e){Hn?Vn?Vn.push(e):Vn=[e]:Hn=e}function fd(){if(Hn){var e=Hn,t=Vn;if(Vn=Hn=null,Ys(e),t)for(e=0;e<t.length;e++)Ys(t[e])}}function pd(e,t){return e(t)}function hd(){}var Fl=!1;function md(e,t,n){if(Fl)return e(t,n);Fl=!0;try{return pd(e,t,n)}finally{Fl=!1,(Hn!==null||Vn!==null)&&(hd(),fd())}}function Rr(e,t){var n=e.stateNode;if(n===null)return null;var r=kl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(I(231,t,typeof n));return n}var Do=!1;if(Lt)try{var cr={};Object.defineProperty(cr,"passive",{get:function(){Do=!0}}),window.addEventListener("test",cr,cr),window.removeEventListener("test",cr,cr)}catch{Do=!1}function wh(e,t,n,r,i,l,o,a,s){var u=Array.prototype.slice.call(arguments,3);try{t.apply(n,u)}catch(d){this.onError(d)}}var br=!1,$i=null,Wi=!1,Mo=null,Sh={onError:function(e){br=!0,$i=e}};function Ch(e,t,n,r,i,l,o,a,s){br=!1,$i=null,wh.apply(Sh,arguments)}function bh(e,t,n,r,i,l,o,a,s){if(Ch.apply(this,arguments),br){if(br){var u=$i;br=!1,$i=null}else throw Error(I(198));Wi||(Wi=!0,Mo=u)}}function Cn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function gd(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function Xs(e){if(Cn(e)!==e)throw Error(I(188))}function Eh(e){var t=e.alternate;if(!t){if(t=Cn(e),t===null)throw Error(I(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return Xs(i),e;if(l===r)return Xs(i),t;l=l.sibling}throw Error(I(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(I(189))}}if(n.alternate!==r)throw Error(I(190))}if(n.tag!==3)throw Error(I(188));return n.stateNode.current===n?e:t}function vd(e){return e=Eh(e),e!==null?yd(e):null}function yd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=yd(e);if(t!==null)return t;e=e.sibling}return null}var xd=Je.unstable_scheduleCallback,Gs=Je.unstable_cancelCallback,jh=Je.unstable_shouldYield,Nh=Je.unstable_requestPaint,he=Je.unstable_now,_h=Je.unstable_getCurrentPriorityLevel,Ma=Je.unstable_ImmediatePriority,kd=Je.unstable_UserBlockingPriority,Qi=Je.unstable_NormalPriority,zh=Je.unstable_LowPriority,wd=Je.unstable_IdlePriority,gl=null,wt=null;function Ph(e){if(wt&&typeof wt.onCommitFiberRoot=="function")try{wt.onCommitFiberRoot(gl,e,void 0,(e.current.flags&128)===128)}catch{}}var pt=Math.clz32?Math.clz32:Ih,Th=Math.log,Lh=Math.LN2;function Ih(e){return e>>>=0,e===0?32:31-(Th(e)/Lh|0)|0}var di=64,fi=4194304;function wr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function qi(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=wr(a):(l&=o,l!==0&&(r=wr(l)))}else o=n&~i,o!==0?r=wr(o):l!==0&&(r=wr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-pt(t),i=1<<n,r|=e[n],t&=~i;return r}function Ah(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Dh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-pt(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Ah(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Ro(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function Sd(){var e=di;return di<<=1,!(di&4194240)&&(di=64),e}function Bl(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function ei(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-pt(t),e[t]=n}function Mh(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-pt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Ra(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-pt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var ne=0;function Cd(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var bd,Oa,Ed,jd,Nd,Oo=!1,pi=[],Qt=null,qt=null,Kt=null,Or=new Map,Fr=new Map,Ht=[],Rh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function Js(e,t){switch(e){case"focusin":case"focusout":Qt=null;break;case"dragenter":case"dragleave":qt=null;break;case"mouseover":case"mouseout":Kt=null;break;case"pointerover":case"pointerout":Or.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Fr.delete(t.pointerId)}}function dr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=ni(t),t!==null&&Oa(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Oh(e,t,n,r,i){switch(t){case"focusin":return Qt=dr(Qt,e,t,n,r,i),!0;case"dragenter":return qt=dr(qt,e,t,n,r,i),!0;case"mouseover":return Kt=dr(Kt,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Or.set(l,dr(Or.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Fr.set(l,dr(Fr.get(l)||null,e,t,n,r,i)),!0}return!1}function _d(e){var t=fn(e.target);if(t!==null){var n=Cn(t);if(n!==null){if(t=n.tag,t===13){if(t=gd(n),t!==null){e.blockedOn=t,Nd(e.priority,function(){Ed(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Pi(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Fo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Io=r,n.target.dispatchEvent(r),Io=null}else return t=ni(n),t!==null&&Oa(t),e.blockedOn=n,!1;t.shift()}return!0}function Zs(e,t,n){Pi(e)&&n.delete(t)}function Fh(){Oo=!1,Qt!==null&&Pi(Qt)&&(Qt=null),qt!==null&&Pi(qt)&&(qt=null),Kt!==null&&Pi(Kt)&&(Kt=null),Or.forEach(Zs),Fr.forEach(Zs)}function fr(e,t){e.blockedOn===t&&(e.blockedOn=null,Oo||(Oo=!0,Je.unstable_scheduleCallback(Je.unstable_NormalPriority,Fh)))}function Br(e){function t(i){return fr(i,e)}if(0<pi.length){fr(pi[0],e);for(var n=1;n<pi.length;n++){var r=pi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(Qt!==null&&fr(Qt,e),qt!==null&&fr(qt,e),Kt!==null&&fr(Kt,e),Or.forEach(t),Fr.forEach(t),n=0;n<Ht.length;n++)r=Ht[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Ht.length&&(n=Ht[0],n.blockedOn===null);)_d(n),n.blockedOn===null&&Ht.shift()}var $n=Mt.ReactCurrentBatchConfig,Ki=!0;function Bh(e,t,n,r){var i=ne,l=$n.transition;$n.transition=null;try{ne=1,Fa(e,t,n,r)}finally{ne=i,$n.transition=l}}function Uh(e,t,n,r){var i=ne,l=$n.transition;$n.transition=null;try{ne=4,Fa(e,t,n,r)}finally{ne=i,$n.transition=l}}function Fa(e,t,n,r){if(Ki){var i=Fo(e,t,n,r);if(i===null)Xl(e,t,r,Yi,n),Js(e,r);else if(Oh(i,e,t,n,r))r.stopPropagation();else if(Js(e,r),t&4&&-1<Rh.indexOf(e)){for(;i!==null;){var l=ni(i);if(l!==null&&bd(l),l=Fo(e,t,n,r),l===null&&Xl(e,t,r,Yi,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else Xl(e,t,r,null,n)}}var Yi=null;function Fo(e,t,n,r){if(Yi=null,e=Da(r),e=fn(e),e!==null)if(t=Cn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=gd(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return Yi=e,null}function zd(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(_h()){case Ma:return 1;case kd:return 4;case Qi:case zh:return 16;case wd:return 536870912;default:return 16}default:return 16}}var $t=null,Ba=null,Ti=null;function Pd(){if(Ti)return Ti;var e,t=Ba,n=t.length,r,i="value"in $t?$t.value:$t.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Ti=i.slice(e,1<r?1-r:void 0)}function Li(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function hi(){return!0}function eu(){return!1}function et(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?hi:eu,this.isPropagationStopped=eu,this}return fe(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=hi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=hi)},persist:function(){},isPersistent:hi}),t}var rr={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Ua=et(rr),ti=fe({},rr,{view:0,detail:0}),Hh=et(ti),Ul,Hl,pr,vl=fe({},ti,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:Ha,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==pr&&(pr&&e.type==="mousemove"?(Ul=e.screenX-pr.screenX,Hl=e.screenY-pr.screenY):Hl=Ul=0,pr=e),Ul)},movementY:function(e){return"movementY"in e?e.movementY:Hl}}),tu=et(vl),Vh=fe({},vl,{dataTransfer:0}),$h=et(Vh),Wh=fe({},ti,{relatedTarget:0}),Vl=et(Wh),Qh=fe({},rr,{animationName:0,elapsedTime:0,pseudoElement:0}),qh=et(Qh),Kh=fe({},rr,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),Yh=et(Kh),Xh=fe({},rr,{data:0}),nu=et(Xh),Gh={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},Jh={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},Zh={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function em(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=Zh[e])?!!t[e]:!1}function Ha(){return em}var tm=fe({},ti,{key:function(e){if(e.key){var t=Gh[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Li(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?Jh[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:Ha,charCode:function(e){return e.type==="keypress"?Li(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Li(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),nm=et(tm),rm=fe({},vl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),ru=et(rm),im=fe({},ti,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:Ha}),lm=et(im),om=fe({},rr,{propertyName:0,elapsedTime:0,pseudoElement:0}),am=et(om),sm=fe({},vl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),um=et(sm),cm=[9,13,27,32],Va=Lt&&"CompositionEvent"in window,Er=null;Lt&&"documentMode"in document&&(Er=document.documentMode);var dm=Lt&&"TextEvent"in window&&!Er,Td=Lt&&(!Va||Er&&8<Er&&11>=Er),iu=" ",lu=!1;function Ld(e,t){switch(e){case"keyup":return cm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Id(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Tn=!1;function fm(e,t){switch(e){case"compositionend":return Id(t);case"keypress":return t.which!==32?null:(lu=!0,iu);case"textInput":return e=t.data,e===iu&&lu?null:e;default:return null}}function pm(e,t){if(Tn)return e==="compositionend"||!Va&&Ld(e,t)?(e=Pd(),Ti=Ba=$t=null,Tn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Td&&t.locale!=="ko"?null:t.data;default:return null}}var hm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function ou(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!hm[e.type]:t==="textarea"}function Ad(e,t,n,r){dd(r),t=Xi(t,"onChange"),0<t.length&&(n=new Ua("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var jr=null,Ur=null;function mm(e){Wd(e,0)}function yl(e){var t=An(e);if(id(t))return e}function gm(e,t){if(e==="change")return t}var Dd=!1;if(Lt){var $l;if(Lt){var Wl="oninput"in document;if(!Wl){var au=document.createElement("div");au.setAttribute("oninput","return;"),Wl=typeof au.oninput=="function"}$l=Wl}else $l=!1;Dd=$l&&(!document.documentMode||9<document.documentMode)}function su(){jr&&(jr.detachEvent("onpropertychange",Md),Ur=jr=null)}function Md(e){if(e.propertyName==="value"&&yl(Ur)){var t=[];Ad(t,Ur,e,Da(e)),md(mm,t)}}function vm(e,t,n){e==="focusin"?(su(),jr=t,Ur=n,jr.attachEvent("onpropertychange",Md)):e==="focusout"&&su()}function ym(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return yl(Ur)}function xm(e,t){if(e==="click")return yl(t)}function km(e,t){if(e==="input"||e==="change")return yl(t)}function wm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var mt=typeof Object.is=="function"?Object.is:wm;function Hr(e,t){if(mt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!wo.call(t,i)||!mt(e[i],t[i]))return!1}return!0}function uu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function cu(e,t){var n=uu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=uu(n)}}function Rd(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Rd(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Od(){for(var e=window,t=Vi();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Vi(e.document)}return t}function $a(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function Sm(e){var t=Od(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Rd(n.ownerDocument.documentElement,n)){if(r!==null&&$a(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=cu(n,l);var o=cu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Cm=Lt&&"documentMode"in document&&11>=document.documentMode,Ln=null,Bo=null,Nr=null,Uo=!1;function du(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Uo||Ln==null||Ln!==Vi(r)||(r=Ln,"selectionStart"in r&&$a(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Nr&&Hr(Nr,r)||(Nr=r,r=Xi(Bo,"onSelect"),0<r.length&&(t=new Ua("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Ln)))}function mi(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var In={animationend:mi("Animation","AnimationEnd"),animationiteration:mi("Animation","AnimationIteration"),animationstart:mi("Animation","AnimationStart"),transitionend:mi("Transition","TransitionEnd")},Ql={},Fd={};Lt&&(Fd=document.createElement("div").style,"AnimationEvent"in window||(delete In.animationend.animation,delete In.animationiteration.animation,delete In.animationstart.animation),"TransitionEvent"in window||delete In.transitionend.transition);function xl(e){if(Ql[e])return Ql[e];if(!In[e])return e;var t=In[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Fd)return Ql[e]=t[n];return e}var Bd=xl("animationend"),Ud=xl("animationiteration"),Hd=xl("animationstart"),Vd=xl("transitionend"),$d=new Map,fu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function nn(e,t){$d.set(e,t),Sn(t,[e])}for(var ql=0;ql<fu.length;ql++){var Kl=fu[ql],bm=Kl.toLowerCase(),Em=Kl[0].toUpperCase()+Kl.slice(1);nn(bm,"on"+Em)}nn(Bd,"onAnimationEnd");nn(Ud,"onAnimationIteration");nn(Hd,"onAnimationStart");nn("dblclick","onDoubleClick");nn("focusin","onFocus");nn("focusout","onBlur");nn(Vd,"onTransitionEnd");Yn("onMouseEnter",["mouseout","mouseover"]);Yn("onMouseLeave",["mouseout","mouseover"]);Yn("onPointerEnter",["pointerout","pointerover"]);Yn("onPointerLeave",["pointerout","pointerover"]);Sn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));Sn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));Sn("onBeforeInput",["compositionend","keypress","textInput","paste"]);Sn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));Sn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));Sn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var Sr="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),jm=new Set("cancel close invalid load scroll toggle".split(" ").concat(Sr));function pu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,bh(r,t,void 0,e),e.currentTarget=null}function Wd(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,u=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;pu(i,a,u),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,u=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;pu(i,a,u),l=s}}}if(Wi)throw e=Mo,Wi=!1,Mo=null,e}function ae(e,t){var n=t[Qo];n===void 0&&(n=t[Qo]=new Set);var r=e+"__bubble";n.has(r)||(Qd(t,e,2,!1),n.add(r))}function Yl(e,t,n){var r=0;t&&(r|=4),Qd(n,e,r,t)}var gi="_reactListening"+Math.random().toString(36).slice(2);function Vr(e){if(!e[gi]){e[gi]=!0,Zc.forEach(function(n){n!=="selectionchange"&&(jm.has(n)||Yl(n,!1,e),Yl(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[gi]||(t[gi]=!0,Yl("selectionchange",!1,t))}}function Qd(e,t,n,r){switch(zd(t)){case 1:var i=Bh;break;case 4:i=Uh;break;default:i=Fa}n=i.bind(null,t,n,e),i=void 0,!Do||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function Xl(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=fn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}md(function(){var u=l,d=Da(n),f=[];e:{var g=$d.get(e);if(g!==void 0){var m=Ua,S=e;switch(e){case"keypress":if(Li(n)===0)break e;case"keydown":case"keyup":m=nm;break;case"focusin":S="focus",m=Vl;break;case"focusout":S="blur",m=Vl;break;case"beforeblur":case"afterblur":m=Vl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":m=tu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":m=$h;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":m=lm;break;case Bd:case Ud:case Hd:m=qh;break;case Vd:m=am;break;case"scroll":m=Hh;break;case"wheel":m=um;break;case"copy":case"cut":case"paste":m=Yh;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":m=ru}var C=(t&4)!==0,j=!C&&e==="scroll",p=C?g!==null?g+"Capture":null:g;C=[];for(var h=u,y;h!==null;){y=h;var k=y.stateNode;if(y.tag===5&&k!==null&&(y=k,p!==null&&(k=Rr(h,p),k!=null&&C.push($r(h,k,y)))),j)break;h=h.return}0<C.length&&(g=new m(g,S,null,n,d),f.push({event:g,listeners:C}))}}if(!(t&7)){e:{if(g=e==="mouseover"||e==="pointerover",m=e==="mouseout"||e==="pointerout",g&&n!==Io&&(S=n.relatedTarget||n.fromElement)&&(fn(S)||S[It]))break e;if((m||g)&&(g=d.window===d?d:(g=d.ownerDocument)?g.defaultView||g.parentWindow:window,m?(S=n.relatedTarget||n.toElement,m=u,S=S?fn(S):null,S!==null&&(j=Cn(S),S!==j||S.tag!==5&&S.tag!==6)&&(S=null)):(m=null,S=u),m!==S)){if(C=tu,k="onMouseLeave",p="onMouseEnter",h="mouse",(e==="pointerout"||e==="pointerover")&&(C=ru,k="onPointerLeave",p="onPointerEnter",h="pointer"),j=m==null?g:An(m),y=S==null?g:An(S),g=new C(k,h+"leave",m,n,d),g.target=j,g.relatedTarget=y,k=null,fn(d)===u&&(C=new C(p,h+"enter",S,n,d),C.target=y,C.relatedTarget=j,k=C),j=k,m&&S)t:{for(C=m,p=S,h=0,y=C;y;y=Nn(y))h++;for(y=0,k=p;k;k=Nn(k))y++;for(;0<h-y;)C=Nn(C),h--;for(;0<y-h;)p=Nn(p),y--;for(;h--;){if(C===p||p!==null&&C===p.alternate)break t;C=Nn(C),p=Nn(p)}C=null}else C=null;m!==null&&hu(f,g,m,C,!1),S!==null&&j!==null&&hu(f,j,S,C,!0)}}e:{if(g=u?An(u):window,m=g.nodeName&&g.nodeName.toLowerCase(),m==="select"||m==="input"&&g.type==="file")var b=gm;else if(ou(g))if(Dd)b=km;else{b=ym;var w=vm}else(m=g.nodeName)&&m.toLowerCase()==="input"&&(g.type==="checkbox"||g.type==="radio")&&(b=xm);if(b&&(b=b(e,u))){Ad(f,b,n,d);break e}w&&w(e,g,u),e==="focusout"&&(w=g._wrapperState)&&w.controlled&&g.type==="number"&&_o(g,"number",g.value)}switch(w=u?An(u):window,e){case"focusin":(ou(w)||w.contentEditable==="true")&&(Ln=w,Bo=u,Nr=null);break;case"focusout":Nr=Bo=Ln=null;break;case"mousedown":Uo=!0;break;case"contextmenu":case"mouseup":case"dragend":Uo=!1,du(f,n,d);break;case"selectionchange":if(Cm)break;case"keydown":case"keyup":du(f,n,d)}var z;if(Va)e:{switch(e){case"compositionstart":var D="onCompositionStart";break e;case"compositionend":D="onCompositionEnd";break e;case"compositionupdate":D="onCompositionUpdate";break e}D=void 0}else Tn?Ld(e,n)&&(D="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(D="onCompositionStart");D&&(Td&&n.locale!=="ko"&&(Tn||D!=="onCompositionStart"?D==="onCompositionEnd"&&Tn&&(z=Pd()):($t=d,Ba="value"in $t?$t.value:$t.textContent,Tn=!0)),w=Xi(u,D),0<w.length&&(D=new nu(D,e,null,n,d),f.push({event:D,listeners:w}),z?D.data=z:(z=Id(n),z!==null&&(D.data=z)))),(z=dm?fm(e,n):pm(e,n))&&(u=Xi(u,"onBeforeInput"),0<u.length&&(d=new nu("onBeforeInput","beforeinput",null,n,d),f.push({event:d,listeners:u}),d.data=z))}Wd(f,t)})}function $r(e,t,n){return{instance:e,listener:t,currentTarget:n}}function Xi(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Rr(e,n),l!=null&&r.unshift($r(e,l,i)),l=Rr(e,t),l!=null&&r.push($r(e,l,i))),e=e.return}return r}function Nn(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function hu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,u=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&u!==null&&(a=u,i?(s=Rr(n,l),s!=null&&o.unshift($r(n,s,a))):i||(s=Rr(n,l),s!=null&&o.push($r(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Nm=/\r\n?/g,_m=/\u0000|\uFFFD/g;function mu(e){return(typeof e=="string"?e:""+e).replace(Nm,`
`).replace(_m,"")}function vi(e,t,n){if(t=mu(t),mu(e)!==t&&n)throw Error(I(425))}function Gi(){}var Ho=null,Vo=null;function $o(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var Wo=typeof setTimeout=="function"?setTimeout:void 0,zm=typeof clearTimeout=="function"?clearTimeout:void 0,gu=typeof Promise=="function"?Promise:void 0,Pm=typeof queueMicrotask=="function"?queueMicrotask:typeof gu<"u"?function(e){return gu.resolve(null).then(e).catch(Tm)}:Wo;function Tm(e){setTimeout(function(){throw e})}function Gl(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Br(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Br(t)}function Yt(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function vu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var ir=Math.random().toString(36).slice(2),xt="__reactFiber$"+ir,Wr="__reactProps$"+ir,It="__reactContainer$"+ir,Qo="__reactEvents$"+ir,Lm="__reactListeners$"+ir,Im="__reactHandles$"+ir;function fn(e){var t=e[xt];if(t)return t;for(var n=e.parentNode;n;){if(t=n[It]||n[xt]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=vu(e);e!==null;){if(n=e[xt])return n;e=vu(e)}return t}e=n,n=e.parentNode}return null}function ni(e){return e=e[xt]||e[It],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function An(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(I(33))}function kl(e){return e[Wr]||null}var qo=[],Dn=-1;function rn(e){return{current:e}}function se(e){0>Dn||(e.current=qo[Dn],qo[Dn]=null,Dn--)}function le(e,t){Dn++,qo[Dn]=e.current,e.current=t}var tn={},Pe=rn(tn),Be=rn(!1),vn=tn;function Xn(e,t){var n=e.type.contextTypes;if(!n)return tn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Ue(e){return e=e.childContextTypes,e!=null}function Ji(){se(Be),se(Pe)}function yu(e,t,n){if(Pe.current!==tn)throw Error(I(168));le(Pe,t),le(Be,n)}function qd(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(I(108,vh(e)||"Unknown",i));return fe({},n,r)}function Zi(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||tn,vn=Pe.current,le(Pe,e),le(Be,Be.current),!0}function xu(e,t,n){var r=e.stateNode;if(!r)throw Error(I(169));n?(e=qd(e,t,vn),r.__reactInternalMemoizedMergedChildContext=e,se(Be),se(Pe),le(Pe,e)):se(Be),le(Be,n)}var _t=null,wl=!1,Jl=!1;function Kd(e){_t===null?_t=[e]:_t.push(e)}function Am(e){wl=!0,Kd(e)}function ln(){if(!Jl&&_t!==null){Jl=!0;var e=0,t=ne;try{var n=_t;for(ne=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}_t=null,wl=!1}catch(i){throw _t!==null&&(_t=_t.slice(e+1)),xd(Ma,ln),i}finally{ne=t,Jl=!1}}return null}var Mn=[],Rn=0,el=null,tl=0,tt=[],nt=0,yn=null,zt=1,Pt="";function un(e,t){Mn[Rn++]=tl,Mn[Rn++]=el,el=e,tl=t}function Yd(e,t,n){tt[nt++]=zt,tt[nt++]=Pt,tt[nt++]=yn,yn=e;var r=zt;e=Pt;var i=32-pt(r)-1;r&=~(1<<i),n+=1;var l=32-pt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,zt=1<<32-pt(t)+i|n<<i|r,Pt=l+e}else zt=1<<l|n<<i|r,Pt=e}function Wa(e){e.return!==null&&(un(e,1),Yd(e,1,0))}function Qa(e){for(;e===el;)el=Mn[--Rn],Mn[Rn]=null,tl=Mn[--Rn],Mn[Rn]=null;for(;e===yn;)yn=tt[--nt],tt[nt]=null,Pt=tt[--nt],tt[nt]=null,zt=tt[--nt],tt[nt]=null}var Ge=null,Ye=null,ue=!1,ft=null;function Xd(e,t){var n=it(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function ku(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,Ge=e,Ye=Yt(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,Ge=e,Ye=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=yn!==null?{id:zt,overflow:Pt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=it(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,Ge=e,Ye=null,!0):!1;default:return!1}}function Ko(e){return(e.mode&1)!==0&&(e.flags&128)===0}function Yo(e){if(ue){var t=Ye;if(t){var n=t;if(!ku(e,t)){if(Ko(e))throw Error(I(418));t=Yt(n.nextSibling);var r=Ge;t&&ku(e,t)?Xd(r,n):(e.flags=e.flags&-4097|2,ue=!1,Ge=e)}}else{if(Ko(e))throw Error(I(418));e.flags=e.flags&-4097|2,ue=!1,Ge=e}}}function wu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;Ge=e}function yi(e){if(e!==Ge)return!1;if(!ue)return wu(e),ue=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!$o(e.type,e.memoizedProps)),t&&(t=Ye)){if(Ko(e))throw Gd(),Error(I(418));for(;t;)Xd(e,t),t=Yt(t.nextSibling)}if(wu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(I(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){Ye=Yt(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}Ye=null}}else Ye=Ge?Yt(e.stateNode.nextSibling):null;return!0}function Gd(){for(var e=Ye;e;)e=Yt(e.nextSibling)}function Gn(){Ye=Ge=null,ue=!1}function qa(e){ft===null?ft=[e]:ft.push(e)}var Dm=Mt.ReactCurrentBatchConfig;function hr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(I(309));var r=n.stateNode}if(!r)throw Error(I(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(I(284));if(!n._owner)throw Error(I(290,e))}return e}function xi(e,t){throw e=Object.prototype.toString.call(t),Error(I(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Su(e){var t=e._init;return t(e._payload)}function Jd(e){function t(p,h){if(e){var y=p.deletions;y===null?(p.deletions=[h],p.flags|=16):y.push(h)}}function n(p,h){if(!e)return null;for(;h!==null;)t(p,h),h=h.sibling;return null}function r(p,h){for(p=new Map;h!==null;)h.key!==null?p.set(h.key,h):p.set(h.index,h),h=h.sibling;return p}function i(p,h){return p=Zt(p,h),p.index=0,p.sibling=null,p}function l(p,h,y){return p.index=y,e?(y=p.alternate,y!==null?(y=y.index,y<h?(p.flags|=2,h):y):(p.flags|=2,h)):(p.flags|=1048576,h)}function o(p){return e&&p.alternate===null&&(p.flags|=2),p}function a(p,h,y,k){return h===null||h.tag!==6?(h=lo(y,p.mode,k),h.return=p,h):(h=i(h,y),h.return=p,h)}function s(p,h,y,k){var b=y.type;return b===Pn?d(p,h,y.props.children,k,y.key):h!==null&&(h.elementType===b||typeof b=="object"&&b!==null&&b.$$typeof===Bt&&Su(b)===h.type)?(k=i(h,y.props),k.ref=hr(p,h,y),k.return=p,k):(k=Fi(y.type,y.key,y.props,null,p.mode,k),k.ref=hr(p,h,y),k.return=p,k)}function u(p,h,y,k){return h===null||h.tag!==4||h.stateNode.containerInfo!==y.containerInfo||h.stateNode.implementation!==y.implementation?(h=oo(y,p.mode,k),h.return=p,h):(h=i(h,y.children||[]),h.return=p,h)}function d(p,h,y,k,b){return h===null||h.tag!==7?(h=gn(y,p.mode,k,b),h.return=p,h):(h=i(h,y),h.return=p,h)}function f(p,h,y){if(typeof h=="string"&&h!==""||typeof h=="number")return h=lo(""+h,p.mode,y),h.return=p,h;if(typeof h=="object"&&h!==null){switch(h.$$typeof){case si:return y=Fi(h.type,h.key,h.props,null,p.mode,y),y.ref=hr(p,null,h),y.return=p,y;case zn:return h=oo(h,p.mode,y),h.return=p,h;case Bt:var k=h._init;return f(p,k(h._payload),y)}if(kr(h)||ur(h))return h=gn(h,p.mode,y,null),h.return=p,h;xi(p,h)}return null}function g(p,h,y,k){var b=h!==null?h.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return b!==null?null:a(p,h,""+y,k);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case si:return y.key===b?s(p,h,y,k):null;case zn:return y.key===b?u(p,h,y,k):null;case Bt:return b=y._init,g(p,h,b(y._payload),k)}if(kr(y)||ur(y))return b!==null?null:d(p,h,y,k,null);xi(p,y)}return null}function m(p,h,y,k,b){if(typeof k=="string"&&k!==""||typeof k=="number")return p=p.get(y)||null,a(h,p,""+k,b);if(typeof k=="object"&&k!==null){switch(k.$$typeof){case si:return p=p.get(k.key===null?y:k.key)||null,s(h,p,k,b);case zn:return p=p.get(k.key===null?y:k.key)||null,u(h,p,k,b);case Bt:var w=k._init;return m(p,h,y,w(k._payload),b)}if(kr(k)||ur(k))return p=p.get(y)||null,d(h,p,k,b,null);xi(h,k)}return null}function S(p,h,y,k){for(var b=null,w=null,z=h,D=h=0,H=null;z!==null&&D<y.length;D++){z.index>D?(H=z,z=null):H=z.sibling;var O=g(p,z,y[D],k);if(O===null){z===null&&(z=H);break}e&&z&&O.alternate===null&&t(p,z),h=l(O,h,D),w===null?b=O:w.sibling=O,w=O,z=H}if(D===y.length)return n(p,z),ue&&un(p,D),b;if(z===null){for(;D<y.length;D++)z=f(p,y[D],k),z!==null&&(h=l(z,h,D),w===null?b=z:w.sibling=z,w=z);return ue&&un(p,D),b}for(z=r(p,z);D<y.length;D++)H=m(z,p,D,y[D],k),H!==null&&(e&&H.alternate!==null&&z.delete(H.key===null?D:H.key),h=l(H,h,D),w===null?b=H:w.sibling=H,w=H);return e&&z.forEach(function(_){return t(p,_)}),ue&&un(p,D),b}function C(p,h,y,k){var b=ur(y);if(typeof b!="function")throw Error(I(150));if(y=b.call(y),y==null)throw Error(I(151));for(var w=b=null,z=h,D=h=0,H=null,O=y.next();z!==null&&!O.done;D++,O=y.next()){z.index>D?(H=z,z=null):H=z.sibling;var _=g(p,z,O.value,k);if(_===null){z===null&&(z=H);break}e&&z&&_.alternate===null&&t(p,z),h=l(_,h,D),w===null?b=_:w.sibling=_,w=_,z=H}if(O.done)return n(p,z),ue&&un(p,D),b;if(z===null){for(;!O.done;D++,O=y.next())O=f(p,O.value,k),O!==null&&(h=l(O,h,D),w===null?b=O:w.sibling=O,w=O);return ue&&un(p,D),b}for(z=r(p,z);!O.done;D++,O=y.next())O=m(z,p,D,O.value,k),O!==null&&(e&&O.alternate!==null&&z.delete(O.key===null?D:O.key),h=l(O,h,D),w===null?b=O:w.sibling=O,w=O);return e&&z.forEach(function(M){return t(p,M)}),ue&&un(p,D),b}function j(p,h,y,k){if(typeof y=="object"&&y!==null&&y.type===Pn&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case si:e:{for(var b=y.key,w=h;w!==null;){if(w.key===b){if(b=y.type,b===Pn){if(w.tag===7){n(p,w.sibling),h=i(w,y.props.children),h.return=p,p=h;break e}}else if(w.elementType===b||typeof b=="object"&&b!==null&&b.$$typeof===Bt&&Su(b)===w.type){n(p,w.sibling),h=i(w,y.props),h.ref=hr(p,w,y),h.return=p,p=h;break e}n(p,w);break}else t(p,w);w=w.sibling}y.type===Pn?(h=gn(y.props.children,p.mode,k,y.key),h.return=p,p=h):(k=Fi(y.type,y.key,y.props,null,p.mode,k),k.ref=hr(p,h,y),k.return=p,p=k)}return o(p);case zn:e:{for(w=y.key;h!==null;){if(h.key===w)if(h.tag===4&&h.stateNode.containerInfo===y.containerInfo&&h.stateNode.implementation===y.implementation){n(p,h.sibling),h=i(h,y.children||[]),h.return=p,p=h;break e}else{n(p,h);break}else t(p,h);h=h.sibling}h=oo(y,p.mode,k),h.return=p,p=h}return o(p);case Bt:return w=y._init,j(p,h,w(y._payload),k)}if(kr(y))return S(p,h,y,k);if(ur(y))return C(p,h,y,k);xi(p,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,h!==null&&h.tag===6?(n(p,h.sibling),h=i(h,y),h.return=p,p=h):(n(p,h),h=lo(y,p.mode,k),h.return=p,p=h),o(p)):n(p,h)}return j}var Jn=Jd(!0),Zd=Jd(!1),nl=rn(null),rl=null,On=null,Ka=null;function Ya(){Ka=On=rl=null}function Xa(e){var t=nl.current;se(nl),e._currentValue=t}function Xo(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Wn(e,t){rl=e,Ka=On=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(Fe=!0),e.firstContext=null)}function ot(e){var t=e._currentValue;if(Ka!==e)if(e={context:e,memoizedValue:t,next:null},On===null){if(rl===null)throw Error(I(308));On=e,rl.dependencies={lanes:0,firstContext:e}}else On=On.next=e;return t}var pn=null;function Ga(e){pn===null?pn=[e]:pn.push(e)}function ef(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,Ga(t)):(n.next=i.next,i.next=n),t.interleaved=n,At(e,r)}function At(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Ut=!1;function Ja(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function tf(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Tt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function Xt(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,Z&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,At(e,n)}return i=r.interleaved,i===null?(t.next=t,Ga(r)):(t.next=i.next,i.next=t),r.interleaved=t,At(e,n)}function Ii(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ra(e,n)}}function Cu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function il(e,t,n,r){var i=e.updateQueue;Ut=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,u=s.next;s.next=null,o===null?l=u:o.next=u,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=u:a.next=u,d.lastBaseUpdate=s))}if(l!==null){var f=i.baseState;o=0,d=u=s=null,a=l;do{var g=a.lane,m=a.eventTime;if((r&g)===g){d!==null&&(d=d.next={eventTime:m,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var S=e,C=a;switch(g=t,m=n,C.tag){case 1:if(S=C.payload,typeof S=="function"){f=S.call(m,f,g);break e}f=S;break e;case 3:S.flags=S.flags&-65537|128;case 0:if(S=C.payload,g=typeof S=="function"?S.call(m,f,g):S,g==null)break e;f=fe({},f,g);break e;case 2:Ut=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,g=i.effects,g===null?i.effects=[a]:g.push(a))}else m={eventTime:m,lane:g,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(u=d=m,s=f):d=d.next=m,o|=g;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;g=a,a=g.next,g.next=null,i.lastBaseUpdate=g,i.shared.pending=null}}while(!0);if(d===null&&(s=f),i.baseState=s,i.firstBaseUpdate=u,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);kn|=o,e.lanes=o,e.memoizedState=f}}function bu(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(I(191,i));i.call(r)}}}var ri={},St=rn(ri),Qr=rn(ri),qr=rn(ri);function hn(e){if(e===ri)throw Error(I(174));return e}function Za(e,t){switch(le(qr,t),le(Qr,e),le(St,ri),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Po(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Po(t,e)}se(St),le(St,t)}function Zn(){se(St),se(Qr),se(qr)}function nf(e){hn(qr.current);var t=hn(St.current),n=Po(t,e.type);t!==n&&(le(Qr,e),le(St,n))}function es(e){Qr.current===e&&(se(St),se(Qr))}var ce=rn(0);function ll(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var Zl=[];function ts(){for(var e=0;e<Zl.length;e++)Zl[e]._workInProgressVersionPrimary=null;Zl.length=0}var Ai=Mt.ReactCurrentDispatcher,eo=Mt.ReactCurrentBatchConfig,xn=0,de=null,ye=null,we=null,ol=!1,_r=!1,Kr=0,Mm=0;function Ne(){throw Error(I(321))}function ns(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!mt(e[n],t[n]))return!1;return!0}function rs(e,t,n,r,i,l){if(xn=l,de=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Ai.current=e===null||e.memoizedState===null?Bm:Um,e=n(r,i),_r){l=0;do{if(_r=!1,Kr=0,25<=l)throw Error(I(301));l+=1,we=ye=null,t.updateQueue=null,Ai.current=Hm,e=n(r,i)}while(_r)}if(Ai.current=al,t=ye!==null&&ye.next!==null,xn=0,we=ye=de=null,ol=!1,t)throw Error(I(300));return e}function is(){var e=Kr!==0;return Kr=0,e}function vt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return we===null?de.memoizedState=we=e:we=we.next=e,we}function at(){if(ye===null){var e=de.alternate;e=e!==null?e.memoizedState:null}else e=ye.next;var t=we===null?de.memoizedState:we.next;if(t!==null)we=t,ye=e;else{if(e===null)throw Error(I(310));ye=e,e={memoizedState:ye.memoizedState,baseState:ye.baseState,baseQueue:ye.baseQueue,queue:ye.queue,next:null},we===null?de.memoizedState=we=e:we=we.next=e}return we}function Yr(e,t){return typeof t=="function"?t(e):t}function to(e){var t=at(),n=t.queue;if(n===null)throw Error(I(311));n.lastRenderedReducer=e;var r=ye,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,u=l;do{var d=u.lane;if((xn&d)===d)s!==null&&(s=s.next={lane:0,action:u.action,hasEagerState:u.hasEagerState,eagerState:u.eagerState,next:null}),r=u.hasEagerState?u.eagerState:e(r,u.action);else{var f={lane:d,action:u.action,hasEagerState:u.hasEagerState,eagerState:u.eagerState,next:null};s===null?(a=s=f,o=r):s=s.next=f,de.lanes|=d,kn|=d}u=u.next}while(u!==null&&u!==l);s===null?o=r:s.next=a,mt(r,t.memoizedState)||(Fe=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,de.lanes|=l,kn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function no(e){var t=at(),n=t.queue;if(n===null)throw Error(I(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);mt(l,t.memoizedState)||(Fe=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function rf(){}function lf(e,t){var n=de,r=at(),i=t(),l=!mt(r.memoizedState,i);if(l&&(r.memoizedState=i,Fe=!0),r=r.queue,ls(sf.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||we!==null&&we.memoizedState.tag&1){if(n.flags|=2048,Xr(9,af.bind(null,n,r,i,t),void 0,null),Se===null)throw Error(I(349));xn&30||of(n,t,i)}return i}function of(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=de.updateQueue,t===null?(t={lastEffect:null,stores:null},de.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function af(e,t,n,r){t.value=n,t.getSnapshot=r,uf(t)&&cf(e)}function sf(e,t,n){return n(function(){uf(t)&&cf(e)})}function uf(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!mt(e,n)}catch{return!0}}function cf(e){var t=At(e,1);t!==null&&ht(t,e,1,-1)}function Eu(e){var t=vt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:Yr,lastRenderedState:e},t.queue=e,e=e.dispatch=Fm.bind(null,de,e),[t.memoizedState,e]}function Xr(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=de.updateQueue,t===null?(t={lastEffect:null,stores:null},de.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function df(){return at().memoizedState}function Di(e,t,n,r){var i=vt();de.flags|=e,i.memoizedState=Xr(1|t,n,void 0,r===void 0?null:r)}function Sl(e,t,n,r){var i=at();r=r===void 0?null:r;var l=void 0;if(ye!==null){var o=ye.memoizedState;if(l=o.destroy,r!==null&&ns(r,o.deps)){i.memoizedState=Xr(t,n,l,r);return}}de.flags|=e,i.memoizedState=Xr(1|t,n,l,r)}function ju(e,t){return Di(8390656,8,e,t)}function ls(e,t){return Sl(2048,8,e,t)}function ff(e,t){return Sl(4,2,e,t)}function pf(e,t){return Sl(4,4,e,t)}function hf(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function mf(e,t,n){return n=n!=null?n.concat([e]):null,Sl(4,4,hf.bind(null,t,e),n)}function os(){}function gf(e,t){var n=at();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ns(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function vf(e,t){var n=at();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ns(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function yf(e,t,n){return xn&21?(mt(n,t)||(n=Sd(),de.lanes|=n,kn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,Fe=!0),e.memoizedState=n)}function Rm(e,t){var n=ne;ne=n!==0&&4>n?n:4,e(!0);var r=eo.transition;eo.transition={};try{e(!1),t()}finally{ne=n,eo.transition=r}}function xf(){return at().memoizedState}function Om(e,t,n){var r=Jt(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},kf(e))wf(t,n);else if(n=ef(e,t,n,r),n!==null){var i=Ae();ht(n,e,r,i),Sf(n,t,r)}}function Fm(e,t,n){var r=Jt(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(kf(e))wf(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,mt(a,o)){var s=t.interleaved;s===null?(i.next=i,Ga(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=ef(e,t,i,r),n!==null&&(i=Ae(),ht(n,e,r,i),Sf(n,t,r))}}function kf(e){var t=e.alternate;return e===de||t!==null&&t===de}function wf(e,t){_r=ol=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function Sf(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ra(e,n)}}var al={readContext:ot,useCallback:Ne,useContext:Ne,useEffect:Ne,useImperativeHandle:Ne,useInsertionEffect:Ne,useLayoutEffect:Ne,useMemo:Ne,useReducer:Ne,useRef:Ne,useState:Ne,useDebugValue:Ne,useDeferredValue:Ne,useTransition:Ne,useMutableSource:Ne,useSyncExternalStore:Ne,useId:Ne,unstable_isNewReconciler:!1},Bm={readContext:ot,useCallback:function(e,t){return vt().memoizedState=[e,t===void 0?null:t],e},useContext:ot,useEffect:ju,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Di(4194308,4,hf.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Di(4194308,4,e,t)},useInsertionEffect:function(e,t){return Di(4,2,e,t)},useMemo:function(e,t){var n=vt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=vt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Om.bind(null,de,e),[r.memoizedState,e]},useRef:function(e){var t=vt();return e={current:e},t.memoizedState=e},useState:Eu,useDebugValue:os,useDeferredValue:function(e){return vt().memoizedState=e},useTransition:function(){var e=Eu(!1),t=e[0];return e=Rm.bind(null,e[1]),vt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=de,i=vt();if(ue){if(n===void 0)throw Error(I(407));n=n()}else{if(n=t(),Se===null)throw Error(I(349));xn&30||of(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,ju(sf.bind(null,r,l,e),[e]),r.flags|=2048,Xr(9,af.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=vt(),t=Se.identifierPrefix;if(ue){var n=Pt,r=zt;n=(r&~(1<<32-pt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=Kr++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Mm++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},Um={readContext:ot,useCallback:gf,useContext:ot,useEffect:ls,useImperativeHandle:mf,useInsertionEffect:ff,useLayoutEffect:pf,useMemo:vf,useReducer:to,useRef:df,useState:function(){return to(Yr)},useDebugValue:os,useDeferredValue:function(e){var t=at();return yf(t,ye.memoizedState,e)},useTransition:function(){var e=to(Yr)[0],t=at().memoizedState;return[e,t]},useMutableSource:rf,useSyncExternalStore:lf,useId:xf,unstable_isNewReconciler:!1},Hm={readContext:ot,useCallback:gf,useContext:ot,useEffect:ls,useImperativeHandle:mf,useInsertionEffect:ff,useLayoutEffect:pf,useMemo:vf,useReducer:no,useRef:df,useState:function(){return no(Yr)},useDebugValue:os,useDeferredValue:function(e){var t=at();return ye===null?t.memoizedState=e:yf(t,ye.memoizedState,e)},useTransition:function(){var e=no(Yr)[0],t=at().memoizedState;return[e,t]},useMutableSource:rf,useSyncExternalStore:lf,useId:xf,unstable_isNewReconciler:!1};function ct(e,t){if(e&&e.defaultProps){t=fe({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function Go(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:fe({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var Cl={isMounted:function(e){return(e=e._reactInternals)?Cn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Ae(),i=Jt(e),l=Tt(r,i);l.payload=t,n!=null&&(l.callback=n),t=Xt(e,l,i),t!==null&&(ht(t,e,i,r),Ii(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Ae(),i=Jt(e),l=Tt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=Xt(e,l,i),t!==null&&(ht(t,e,i,r),Ii(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Ae(),r=Jt(e),i=Tt(n,r);i.tag=2,t!=null&&(i.callback=t),t=Xt(e,i,r),t!==null&&(ht(t,e,r,n),Ii(t,e,r))}};function Nu(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Hr(n,r)||!Hr(i,l):!0}function Cf(e,t,n){var r=!1,i=tn,l=t.contextType;return typeof l=="object"&&l!==null?l=ot(l):(i=Ue(t)?vn:Pe.current,r=t.contextTypes,l=(r=r!=null)?Xn(e,i):tn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=Cl,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function _u(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&Cl.enqueueReplaceState(t,t.state,null)}function Jo(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},Ja(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=ot(l):(l=Ue(t)?vn:Pe.current,i.context=Xn(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(Go(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&Cl.enqueueReplaceState(i,i.state,null),il(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function er(e,t){try{var n="",r=t;do n+=gh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function ro(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function Zo(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var Vm=typeof WeakMap=="function"?WeakMap:Map;function bf(e,t,n){n=Tt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){ul||(ul=!0,ua=r),Zo(e,t)},n}function Ef(e,t,n){n=Tt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){Zo(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){Zo(e,t),typeof r!="function"&&(Gt===null?Gt=new Set([this]):Gt.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function zu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new Vm;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=rg.bind(null,e,t,n),t.then(e,e))}function Pu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Tu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Tt(-1,1),t.tag=2,Xt(n,t,1))),n.lanes|=1),e)}var $m=Mt.ReactCurrentOwner,Fe=!1;function Ie(e,t,n,r){t.child=e===null?Zd(t,null,n,r):Jn(t,e.child,n,r)}function Lu(e,t,n,r,i){n=n.render;var l=t.ref;return Wn(t,i),r=rs(e,t,n,r,l,i),n=is(),e!==null&&!Fe?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Dt(e,t,i)):(ue&&n&&Wa(t),t.flags|=1,Ie(e,t,r,i),t.child)}function Iu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!hs(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,jf(e,t,l,r,i)):(e=Fi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Hr,n(o,r)&&e.ref===t.ref)return Dt(e,t,i)}return t.flags|=1,e=Zt(l,r),e.ref=t.ref,e.return=t,t.child=e}function jf(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Hr(l,r)&&e.ref===t.ref)if(Fe=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(Fe=!0);else return t.lanes=e.lanes,Dt(e,t,i)}return ea(e,t,n,r,i)}function Nf(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},le(Bn,qe),qe|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,le(Bn,qe),qe|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,le(Bn,qe),qe|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,le(Bn,qe),qe|=r;return Ie(e,t,i,n),t.child}function _f(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ea(e,t,n,r,i){var l=Ue(n)?vn:Pe.current;return l=Xn(t,l),Wn(t,i),n=rs(e,t,n,r,l,i),r=is(),e!==null&&!Fe?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Dt(e,t,i)):(ue&&r&&Wa(t),t.flags|=1,Ie(e,t,n,i),t.child)}function Au(e,t,n,r,i){if(Ue(n)){var l=!0;Zi(t)}else l=!1;if(Wn(t,i),t.stateNode===null)Mi(e,t),Cf(t,n,r),Jo(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,u=n.contextType;typeof u=="object"&&u!==null?u=ot(u):(u=Ue(n)?vn:Pe.current,u=Xn(t,u));var d=n.getDerivedStateFromProps,f=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";f||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==u)&&_u(t,o,r,u),Ut=!1;var g=t.memoizedState;o.state=g,il(t,r,o,i),s=t.memoizedState,a!==r||g!==s||Be.current||Ut?(typeof d=="function"&&(Go(t,n,d,r),s=t.memoizedState),(a=Ut||Nu(t,n,a,r,g,s,u))?(f||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=u,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,tf(e,t),a=t.memoizedProps,u=t.type===t.elementType?a:ct(t.type,a),o.props=u,f=t.pendingProps,g=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=ot(s):(s=Ue(n)?vn:Pe.current,s=Xn(t,s));var m=n.getDerivedStateFromProps;(d=typeof m=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==f||g!==s)&&_u(t,o,r,s),Ut=!1,g=t.memoizedState,o.state=g,il(t,r,o,i);var S=t.memoizedState;a!==f||g!==S||Be.current||Ut?(typeof m=="function"&&(Go(t,n,m,r),S=t.memoizedState),(u=Ut||Nu(t,n,u,r,g,S,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,S,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,S,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=S),o.props=r,o.state=S,o.context=s,r=u):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&g===e.memoizedState||(t.flags|=1024),r=!1)}return ta(e,t,n,r,l,i)}function ta(e,t,n,r,i,l){_f(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&xu(t,n,!1),Dt(e,t,l);r=t.stateNode,$m.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=Jn(t,e.child,null,l),t.child=Jn(t,null,a,l)):Ie(e,t,a,l),t.memoizedState=r.state,i&&xu(t,n,!0),t.child}function zf(e){var t=e.stateNode;t.pendingContext?yu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&yu(e,t.context,!1),Za(e,t.containerInfo)}function Du(e,t,n,r,i){return Gn(),qa(i),t.flags|=256,Ie(e,t,n,r),t.child}var na={dehydrated:null,treeContext:null,retryLane:0};function ra(e){return{baseLanes:e,cachePool:null,transitions:null}}function Pf(e,t,n){var r=t.pendingProps,i=ce.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),le(ce,i&1),e===null)return Yo(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=jl(o,r,0,null),e=gn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=ra(n),t.memoizedState=na,e):as(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return Wm(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=Zt(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=Zt(a,l):(l=gn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?ra(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=na,r}return l=e.child,e=l.sibling,r=Zt(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function as(e,t){return t=jl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function ki(e,t,n,r){return r!==null&&qa(r),Jn(t,e.child,null,n),e=as(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function Wm(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=ro(Error(I(422))),ki(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=jl({mode:"visible",children:r.children},i,0,null),l=gn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&Jn(t,e.child,null,o),t.child.memoizedState=ra(o),t.memoizedState=na,l);if(!(t.mode&1))return ki(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(I(419)),r=ro(l,r,void 0),ki(e,t,o,r)}if(a=(o&e.childLanes)!==0,Fe||a){if(r=Se,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,At(e,i),ht(r,e,i,-1))}return ps(),r=ro(Error(I(421))),ki(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=ig.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,Ye=Yt(i.nextSibling),Ge=t,ue=!0,ft=null,e!==null&&(tt[nt++]=zt,tt[nt++]=Pt,tt[nt++]=yn,zt=e.id,Pt=e.overflow,yn=t),t=as(t,r.children),t.flags|=4096,t)}function Mu(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),Xo(e.return,t,n)}function io(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Tf(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(Ie(e,t,r.children,n),r=ce.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Mu(e,n,t);else if(e.tag===19)Mu(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(le(ce,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&ll(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),io(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&ll(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}io(t,!0,n,null,l);break;case"together":io(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Mi(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Dt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),kn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(I(153));if(t.child!==null){for(e=t.child,n=Zt(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=Zt(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function Qm(e,t,n){switch(t.tag){case 3:zf(t),Gn();break;case 5:nf(t);break;case 1:Ue(t.type)&&Zi(t);break;case 4:Za(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;le(nl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(le(ce,ce.current&1),t.flags|=128,null):n&t.child.childLanes?Pf(e,t,n):(le(ce,ce.current&1),e=Dt(e,t,n),e!==null?e.sibling:null);le(ce,ce.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Tf(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),le(ce,ce.current),r)break;return null;case 22:case 23:return t.lanes=0,Nf(e,t,n)}return Dt(e,t,n)}var Lf,ia,If,Af;Lf=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};ia=function(){};If=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,hn(St.current);var l=null;switch(n){case"input":i=jo(e,i),r=jo(e,r),l=[];break;case"select":i=fe({},i,{value:void 0}),r=fe({},r,{value:void 0}),l=[];break;case"textarea":i=zo(e,i),r=zo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=Gi)}To(n,r);var o;n=null;for(u in i)if(!r.hasOwnProperty(u)&&i.hasOwnProperty(u)&&i[u]!=null)if(u==="style"){var a=i[u];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else u!=="dangerouslySetInnerHTML"&&u!=="children"&&u!=="suppressContentEditableWarning"&&u!=="suppressHydrationWarning"&&u!=="autoFocus"&&(Dr.hasOwnProperty(u)?l||(l=[]):(l=l||[]).push(u,null));for(u in r){var s=r[u];if(a=i!=null?i[u]:void 0,r.hasOwnProperty(u)&&s!==a&&(s!=null||a!=null))if(u==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(u,n)),n=s;else u==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(u,s)):u==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(u,""+s):u!=="suppressContentEditableWarning"&&u!=="suppressHydrationWarning"&&(Dr.hasOwnProperty(u)?(s!=null&&u==="onScroll"&&ae("scroll",e),l||a===s||(l=[])):(l=l||[]).push(u,s))}n&&(l=l||[]).push("style",n);var u=l;(t.updateQueue=u)&&(t.flags|=4)}};Af=function(e,t,n,r){n!==r&&(t.flags|=4)};function mr(e,t){if(!ue)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function _e(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function qm(e,t,n){var r=t.pendingProps;switch(Qa(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return _e(t),null;case 1:return Ue(t.type)&&Ji(),_e(t),null;case 3:return r=t.stateNode,Zn(),se(Be),se(Pe),ts(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(yi(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,ft!==null&&(fa(ft),ft=null))),ia(e,t),_e(t),null;case 5:es(t);var i=hn(qr.current);if(n=t.type,e!==null&&t.stateNode!=null)If(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(I(166));return _e(t),null}if(e=hn(St.current),yi(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[xt]=t,r[Wr]=l,e=(t.mode&1)!==0,n){case"dialog":ae("cancel",r),ae("close",r);break;case"iframe":case"object":case"embed":ae("load",r);break;case"video":case"audio":for(i=0;i<Sr.length;i++)ae(Sr[i],r);break;case"source":ae("error",r);break;case"img":case"image":case"link":ae("error",r),ae("load",r);break;case"details":ae("toggle",r);break;case"input":Ws(r,l),ae("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},ae("invalid",r);break;case"textarea":qs(r,l),ae("invalid",r)}To(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&vi(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&vi(r.textContent,a,e),i=["children",""+a]):Dr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&ae("scroll",r)}switch(n){case"input":ui(r),Qs(r,l,!0);break;case"textarea":ui(r),Ks(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=Gi)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=ad(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[xt]=t,e[Wr]=r,Lf(e,t,!1,!1),t.stateNode=e;e:{switch(o=Lo(n,r),n){case"dialog":ae("cancel",e),ae("close",e),i=r;break;case"iframe":case"object":case"embed":ae("load",e),i=r;break;case"video":case"audio":for(i=0;i<Sr.length;i++)ae(Sr[i],e);i=r;break;case"source":ae("error",e),i=r;break;case"img":case"image":case"link":ae("error",e),ae("load",e),i=r;break;case"details":ae("toggle",e),i=r;break;case"input":Ws(e,r),i=jo(e,r),ae("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=fe({},r,{value:void 0}),ae("invalid",e);break;case"textarea":qs(e,r),i=zo(e,r),ae("invalid",e);break;default:i=r}To(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?cd(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&sd(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Mr(e,s):typeof s=="number"&&Mr(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Dr.hasOwnProperty(l)?s!=null&&l==="onScroll"&&ae("scroll",e):s!=null&&Ta(e,l,s,o))}switch(n){case"input":ui(e),Qs(e,r,!1);break;case"textarea":ui(e),Ks(e);break;case"option":r.value!=null&&e.setAttribute("value",""+en(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Un(e,!!r.multiple,l,!1):r.defaultValue!=null&&Un(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=Gi)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return _e(t),null;case 6:if(e&&t.stateNode!=null)Af(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(I(166));if(n=hn(qr.current),hn(St.current),yi(t)){if(r=t.stateNode,n=t.memoizedProps,r[xt]=t,(l=r.nodeValue!==n)&&(e=Ge,e!==null))switch(e.tag){case 3:vi(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&vi(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[xt]=t,t.stateNode=r}return _e(t),null;case 13:if(se(ce),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(ue&&Ye!==null&&t.mode&1&&!(t.flags&128))Gd(),Gn(),t.flags|=98560,l=!1;else if(l=yi(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(I(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(I(317));l[xt]=t}else Gn(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;_e(t),l=!1}else ft!==null&&(fa(ft),ft=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||ce.current&1?xe===0&&(xe=3):ps())),t.updateQueue!==null&&(t.flags|=4),_e(t),null);case 4:return Zn(),ia(e,t),e===null&&Vr(t.stateNode.containerInfo),_e(t),null;case 10:return Xa(t.type._context),_e(t),null;case 17:return Ue(t.type)&&Ji(),_e(t),null;case 19:if(se(ce),l=t.memoizedState,l===null)return _e(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)mr(l,!1);else{if(xe!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=ll(e),o!==null){for(t.flags|=128,mr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return le(ce,ce.current&1|2),t.child}e=e.sibling}l.tail!==null&&he()>tr&&(t.flags|=128,r=!0,mr(l,!1),t.lanes=4194304)}else{if(!r)if(e=ll(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),mr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!ue)return _e(t),null}else 2*he()-l.renderingStartTime>tr&&n!==1073741824&&(t.flags|=128,r=!0,mr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=he(),t.sibling=null,n=ce.current,le(ce,r?n&1|2:n&1),t):(_e(t),null);case 22:case 23:return fs(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?qe&1073741824&&(_e(t),t.subtreeFlags&6&&(t.flags|=8192)):_e(t),null;case 24:return null;case 25:return null}throw Error(I(156,t.tag))}function Km(e,t){switch(Qa(t),t.tag){case 1:return Ue(t.type)&&Ji(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return Zn(),se(Be),se(Pe),ts(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return es(t),null;case 13:if(se(ce),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(I(340));Gn()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return se(ce),null;case 4:return Zn(),null;case 10:return Xa(t.type._context),null;case 22:case 23:return fs(),null;case 24:return null;default:return null}}var wi=!1,ze=!1,Ym=typeof WeakSet=="function"?WeakSet:Set,F=null;function Fn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){pe(e,t,r)}else n.current=null}function la(e,t,n){try{n()}catch(r){pe(e,t,r)}}var Ru=!1;function Xm(e,t){if(Ho=Ki,e=Od(),$a(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,u=0,d=0,f=e,g=null;t:for(;;){for(var m;f!==n||i!==0&&f.nodeType!==3||(a=o+i),f!==l||r!==0&&f.nodeType!==3||(s=o+r),f.nodeType===3&&(o+=f.nodeValue.length),(m=f.firstChild)!==null;)g=f,f=m;for(;;){if(f===e)break t;if(g===n&&++u===i&&(a=o),g===l&&++d===r&&(s=o),(m=f.nextSibling)!==null)break;f=g,g=f.parentNode}f=m}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Vo={focusedElem:e,selectionRange:n},Ki=!1,F=t;F!==null;)if(t=F,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,F=e;else for(;F!==null;){t=F;try{var S=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(S!==null){var C=S.memoizedProps,j=S.memoizedState,p=t.stateNode,h=p.getSnapshotBeforeUpdate(t.elementType===t.type?C:ct(t.type,C),j);p.__reactInternalSnapshotBeforeUpdate=h}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(I(163))}}catch(k){pe(t,t.return,k)}if(e=t.sibling,e!==null){e.return=t.return,F=e;break}F=t.return}return S=Ru,Ru=!1,S}function zr(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&la(t,n,l)}i=i.next}while(i!==r)}}function bl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function oa(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Df(e){var t=e.alternate;t!==null&&(e.alternate=null,Df(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[xt],delete t[Wr],delete t[Qo],delete t[Lm],delete t[Im])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Mf(e){return e.tag===5||e.tag===3||e.tag===4}function Ou(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Mf(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function aa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=Gi));else if(r!==4&&(e=e.child,e!==null))for(aa(e,t,n),e=e.sibling;e!==null;)aa(e,t,n),e=e.sibling}function sa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(sa(e,t,n),e=e.sibling;e!==null;)sa(e,t,n),e=e.sibling}var Ce=null,dt=!1;function Rt(e,t,n){for(n=n.child;n!==null;)Rf(e,t,n),n=n.sibling}function Rf(e,t,n){if(wt&&typeof wt.onCommitFiberUnmount=="function")try{wt.onCommitFiberUnmount(gl,n)}catch{}switch(n.tag){case 5:ze||Fn(n,t);case 6:var r=Ce,i=dt;Ce=null,Rt(e,t,n),Ce=r,dt=i,Ce!==null&&(dt?(e=Ce,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):Ce.removeChild(n.stateNode));break;case 18:Ce!==null&&(dt?(e=Ce,n=n.stateNode,e.nodeType===8?Gl(e.parentNode,n):e.nodeType===1&&Gl(e,n),Br(e)):Gl(Ce,n.stateNode));break;case 4:r=Ce,i=dt,Ce=n.stateNode.containerInfo,dt=!0,Rt(e,t,n),Ce=r,dt=i;break;case 0:case 11:case 14:case 15:if(!ze&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&la(n,t,o),i=i.next}while(i!==r)}Rt(e,t,n);break;case 1:if(!ze&&(Fn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){pe(n,t,a)}Rt(e,t,n);break;case 21:Rt(e,t,n);break;case 22:n.mode&1?(ze=(r=ze)||n.memoizedState!==null,Rt(e,t,n),ze=r):Rt(e,t,n);break;default:Rt(e,t,n)}}function Fu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new Ym),t.forEach(function(r){var i=lg.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function ut(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:Ce=a.stateNode,dt=!1;break e;case 3:Ce=a.stateNode.containerInfo,dt=!0;break e;case 4:Ce=a.stateNode.containerInfo,dt=!0;break e}a=a.return}if(Ce===null)throw Error(I(160));Rf(l,o,i),Ce=null,dt=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(u){pe(i,t,u)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Of(t,e),t=t.sibling}function Of(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(ut(t,e),gt(e),r&4){try{zr(3,e,e.return),bl(3,e)}catch(C){pe(e,e.return,C)}try{zr(5,e,e.return)}catch(C){pe(e,e.return,C)}}break;case 1:ut(t,e),gt(e),r&512&&n!==null&&Fn(n,n.return);break;case 5:if(ut(t,e),gt(e),r&512&&n!==null&&Fn(n,n.return),e.flags&32){var i=e.stateNode;try{Mr(i,"")}catch(C){pe(e,e.return,C)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&ld(i,l),Lo(a,o);var u=Lo(a,l);for(o=0;o<s.length;o+=2){var d=s[o],f=s[o+1];d==="style"?cd(i,f):d==="dangerouslySetInnerHTML"?sd(i,f):d==="children"?Mr(i,f):Ta(i,d,f,u)}switch(a){case"input":No(i,l);break;case"textarea":od(i,l);break;case"select":var g=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var m=l.value;m!=null?Un(i,!!l.multiple,m,!1):g!==!!l.multiple&&(l.defaultValue!=null?Un(i,!!l.multiple,l.defaultValue,!0):Un(i,!!l.multiple,l.multiple?[]:"",!1))}i[Wr]=l}catch(C){pe(e,e.return,C)}}break;case 6:if(ut(t,e),gt(e),r&4){if(e.stateNode===null)throw Error(I(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(C){pe(e,e.return,C)}}break;case 3:if(ut(t,e),gt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Br(t.containerInfo)}catch(C){pe(e,e.return,C)}break;case 4:ut(t,e),gt(e);break;case 13:ut(t,e),gt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(cs=he())),r&4&&Fu(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(ze=(u=ze)||d,ut(t,e),ze=u):ut(t,e),gt(e),r&8192){if(u=e.memoizedState!==null,(e.stateNode.isHidden=u)&&!d&&e.mode&1)for(F=e,d=e.child;d!==null;){for(f=F=d;F!==null;){switch(g=F,m=g.child,g.tag){case 0:case 11:case 14:case 15:zr(4,g,g.return);break;case 1:Fn(g,g.return);var S=g.stateNode;if(typeof S.componentWillUnmount=="function"){r=g,n=g.return;try{t=r,S.props=t.memoizedProps,S.state=t.memoizedState,S.componentWillUnmount()}catch(C){pe(r,n,C)}}break;case 5:Fn(g,g.return);break;case 22:if(g.memoizedState!==null){Uu(f);continue}}m!==null?(m.return=g,F=m):Uu(f)}d=d.sibling}e:for(d=null,f=e;;){if(f.tag===5){if(d===null){d=f;try{i=f.stateNode,u?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=f.stateNode,s=f.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=ud("display",o))}catch(C){pe(e,e.return,C)}}}else if(f.tag===6){if(d===null)try{f.stateNode.nodeValue=u?"":f.memoizedProps}catch(C){pe(e,e.return,C)}}else if((f.tag!==22&&f.tag!==23||f.memoizedState===null||f===e)&&f.child!==null){f.child.return=f,f=f.child;continue}if(f===e)break e;for(;f.sibling===null;){if(f.return===null||f.return===e)break e;d===f&&(d=null),f=f.return}d===f&&(d=null),f.sibling.return=f.return,f=f.sibling}}break;case 19:ut(t,e),gt(e),r&4&&Fu(e);break;case 21:break;default:ut(t,e),gt(e)}}function gt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Mf(n)){var r=n;break e}n=n.return}throw Error(I(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Mr(i,""),r.flags&=-33);var l=Ou(e);sa(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Ou(e);aa(e,a,o);break;default:throw Error(I(161))}}catch(s){pe(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function Gm(e,t,n){F=e,Ff(e)}function Ff(e,t,n){for(var r=(e.mode&1)!==0;F!==null;){var i=F,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||wi;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||ze;a=wi;var u=ze;if(wi=o,(ze=s)&&!u)for(F=i;F!==null;)o=F,s=o.child,o.tag===22&&o.memoizedState!==null?Hu(i):s!==null?(s.return=o,F=s):Hu(i);for(;l!==null;)F=l,Ff(l),l=l.sibling;F=i,wi=a,ze=u}Bu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,F=l):Bu(e)}}function Bu(e){for(;F!==null;){var t=F;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:ze||bl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!ze)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:ct(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&bu(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}bu(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var u=t.alternate;if(u!==null){var d=u.memoizedState;if(d!==null){var f=d.dehydrated;f!==null&&Br(f)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(I(163))}ze||t.flags&512&&oa(t)}catch(g){pe(t,t.return,g)}}if(t===e){F=null;break}if(n=t.sibling,n!==null){n.return=t.return,F=n;break}F=t.return}}function Uu(e){for(;F!==null;){var t=F;if(t===e){F=null;break}var n=t.sibling;if(n!==null){n.return=t.return,F=n;break}F=t.return}}function Hu(e){for(;F!==null;){var t=F;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{bl(4,t)}catch(s){pe(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){pe(t,i,s)}}var l=t.return;try{oa(t)}catch(s){pe(t,l,s)}break;case 5:var o=t.return;try{oa(t)}catch(s){pe(t,o,s)}}}catch(s){pe(t,t.return,s)}if(t===e){F=null;break}var a=t.sibling;if(a!==null){a.return=t.return,F=a;break}F=t.return}}var Jm=Math.ceil,sl=Mt.ReactCurrentDispatcher,ss=Mt.ReactCurrentOwner,lt=Mt.ReactCurrentBatchConfig,Z=0,Se=null,ge=null,be=0,qe=0,Bn=rn(0),xe=0,Gr=null,kn=0,El=0,us=0,Pr=null,Oe=null,cs=0,tr=1/0,Nt=null,ul=!1,ua=null,Gt=null,Si=!1,Wt=null,cl=0,Tr=0,ca=null,Ri=-1,Oi=0;function Ae(){return Z&6?he():Ri!==-1?Ri:Ri=he()}function Jt(e){return e.mode&1?Z&2&&be!==0?be&-be:Dm.transition!==null?(Oi===0&&(Oi=Sd()),Oi):(e=ne,e!==0||(e=window.event,e=e===void 0?16:zd(e.type)),e):1}function ht(e,t,n,r){if(50<Tr)throw Tr=0,ca=null,Error(I(185));ei(e,n,r),(!(Z&2)||e!==Se)&&(e===Se&&(!(Z&2)&&(El|=n),xe===4&&Vt(e,be)),He(e,r),n===1&&Z===0&&!(t.mode&1)&&(tr=he()+500,wl&&ln()))}function He(e,t){var n=e.callbackNode;Dh(e,t);var r=qi(e,e===Se?be:0);if(r===0)n!==null&&Gs(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&Gs(n),t===1)e.tag===0?Am(Vu.bind(null,e)):Kd(Vu.bind(null,e)),Pm(function(){!(Z&6)&&ln()}),n=null;else{switch(Cd(r)){case 1:n=Ma;break;case 4:n=kd;break;case 16:n=Qi;break;case 536870912:n=wd;break;default:n=Qi}n=qf(n,Bf.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function Bf(e,t){if(Ri=-1,Oi=0,Z&6)throw Error(I(327));var n=e.callbackNode;if(Qn()&&e.callbackNode!==n)return null;var r=qi(e,e===Se?be:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=dl(e,r);else{t=r;var i=Z;Z|=2;var l=Hf();(Se!==e||be!==t)&&(Nt=null,tr=he()+500,mn(e,t));do try{tg();break}catch(a){Uf(e,a)}while(!0);Ya(),sl.current=l,Z=i,ge!==null?t=0:(Se=null,be=0,t=xe)}if(t!==0){if(t===2&&(i=Ro(e),i!==0&&(r=i,t=da(e,i))),t===1)throw n=Gr,mn(e,0),Vt(e,r),He(e,he()),n;if(t===6)Vt(e,r);else{if(i=e.current.alternate,!(r&30)&&!Zm(i)&&(t=dl(e,r),t===2&&(l=Ro(e),l!==0&&(r=l,t=da(e,l))),t===1))throw n=Gr,mn(e,0),Vt(e,r),He(e,he()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(I(345));case 2:cn(e,Oe,Nt);break;case 3:if(Vt(e,r),(r&130023424)===r&&(t=cs+500-he(),10<t)){if(qi(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Ae(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=Wo(cn.bind(null,e,Oe,Nt),t);break}cn(e,Oe,Nt);break;case 4:if(Vt(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-pt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=he()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*Jm(r/1960))-r,10<r){e.timeoutHandle=Wo(cn.bind(null,e,Oe,Nt),r);break}cn(e,Oe,Nt);break;case 5:cn(e,Oe,Nt);break;default:throw Error(I(329))}}}return He(e,he()),e.callbackNode===n?Bf.bind(null,e):null}function da(e,t){var n=Pr;return e.current.memoizedState.isDehydrated&&(mn(e,t).flags|=256),e=dl(e,t),e!==2&&(t=Oe,Oe=n,t!==null&&fa(t)),e}function fa(e){Oe===null?Oe=e:Oe.push.apply(Oe,e)}function Zm(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!mt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Vt(e,t){for(t&=~us,t&=~El,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-pt(t),r=1<<n;e[n]=-1,t&=~r}}function Vu(e){if(Z&6)throw Error(I(327));Qn();var t=qi(e,0);if(!(t&1))return He(e,he()),null;var n=dl(e,t);if(e.tag!==0&&n===2){var r=Ro(e);r!==0&&(t=r,n=da(e,r))}if(n===1)throw n=Gr,mn(e,0),Vt(e,t),He(e,he()),n;if(n===6)throw Error(I(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,cn(e,Oe,Nt),He(e,he()),null}function ds(e,t){var n=Z;Z|=1;try{return e(t)}finally{Z=n,Z===0&&(tr=he()+500,wl&&ln())}}function wn(e){Wt!==null&&Wt.tag===0&&!(Z&6)&&Qn();var t=Z;Z|=1;var n=lt.transition,r=ne;try{if(lt.transition=null,ne=1,e)return e()}finally{ne=r,lt.transition=n,Z=t,!(Z&6)&&ln()}}function fs(){qe=Bn.current,se(Bn)}function mn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,zm(n)),ge!==null)for(n=ge.return;n!==null;){var r=n;switch(Qa(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&Ji();break;case 3:Zn(),se(Be),se(Pe),ts();break;case 5:es(r);break;case 4:Zn();break;case 13:se(ce);break;case 19:se(ce);break;case 10:Xa(r.type._context);break;case 22:case 23:fs()}n=n.return}if(Se=e,ge=e=Zt(e.current,null),be=qe=t,xe=0,Gr=null,us=El=kn=0,Oe=Pr=null,pn!==null){for(t=0;t<pn.length;t++)if(n=pn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}pn=null}return e}function Uf(e,t){do{var n=ge;try{if(Ya(),Ai.current=al,ol){for(var r=de.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}ol=!1}if(xn=0,we=ye=de=null,_r=!1,Kr=0,ss.current=null,n===null||n.return===null){xe=1,Gr=t,ge=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=be,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var u=s,d=a,f=d.tag;if(!(d.mode&1)&&(f===0||f===11||f===15)){var g=d.alternate;g?(d.updateQueue=g.updateQueue,d.memoizedState=g.memoizedState,d.lanes=g.lanes):(d.updateQueue=null,d.memoizedState=null)}var m=Pu(o);if(m!==null){m.flags&=-257,Tu(m,o,a,l,t),m.mode&1&&zu(l,u,t),t=m,s=u;var S=t.updateQueue;if(S===null){var C=new Set;C.add(s),t.updateQueue=C}else S.add(s);break e}else{if(!(t&1)){zu(l,u,t),ps();break e}s=Error(I(426))}}else if(ue&&a.mode&1){var j=Pu(o);if(j!==null){!(j.flags&65536)&&(j.flags|=256),Tu(j,o,a,l,t),qa(er(s,a));break e}}l=s=er(s,a),xe!==4&&(xe=2),Pr===null?Pr=[l]:Pr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var p=bf(l,s,t);Cu(l,p);break e;case 1:a=s;var h=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof h.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(Gt===null||!Gt.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var k=Ef(l,a,t);Cu(l,k);break e}}l=l.return}while(l!==null)}$f(n)}catch(b){t=b,ge===n&&n!==null&&(ge=n=n.return);continue}break}while(!0)}function Hf(){var e=sl.current;return sl.current=al,e===null?al:e}function ps(){(xe===0||xe===3||xe===2)&&(xe=4),Se===null||!(kn&268435455)&&!(El&268435455)||Vt(Se,be)}function dl(e,t){var n=Z;Z|=2;var r=Hf();(Se!==e||be!==t)&&(Nt=null,mn(e,t));do try{eg();break}catch(i){Uf(e,i)}while(!0);if(Ya(),Z=n,sl.current=r,ge!==null)throw Error(I(261));return Se=null,be=0,xe}function eg(){for(;ge!==null;)Vf(ge)}function tg(){for(;ge!==null&&!jh();)Vf(ge)}function Vf(e){var t=Qf(e.alternate,e,qe);e.memoizedProps=e.pendingProps,t===null?$f(e):ge=t,ss.current=null}function $f(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=Km(n,t),n!==null){n.flags&=32767,ge=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{xe=6,ge=null;return}}else if(n=qm(n,t,qe),n!==null){ge=n;return}if(t=t.sibling,t!==null){ge=t;return}ge=t=e}while(t!==null);xe===0&&(xe=5)}function cn(e,t,n){var r=ne,i=lt.transition;try{lt.transition=null,ne=1,ng(e,t,n,r)}finally{lt.transition=i,ne=r}return null}function ng(e,t,n,r){do Qn();while(Wt!==null);if(Z&6)throw Error(I(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(I(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Mh(e,l),e===Se&&(ge=Se=null,be=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||Si||(Si=!0,qf(Qi,function(){return Qn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=lt.transition,lt.transition=null;var o=ne;ne=1;var a=Z;Z|=4,ss.current=null,Xm(e,n),Of(n,e),Sm(Vo),Ki=!!Ho,Vo=Ho=null,e.current=n,Gm(n),Nh(),Z=a,ne=o,lt.transition=l}else e.current=n;if(Si&&(Si=!1,Wt=e,cl=i),l=e.pendingLanes,l===0&&(Gt=null),Ph(n.stateNode),He(e,he()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(ul)throw ul=!1,e=ua,ua=null,e;return cl&1&&e.tag!==0&&Qn(),l=e.pendingLanes,l&1?e===ca?Tr++:(Tr=0,ca=e):Tr=0,ln(),null}function Qn(){if(Wt!==null){var e=Cd(cl),t=lt.transition,n=ne;try{if(lt.transition=null,ne=16>e?16:e,Wt===null)var r=!1;else{if(e=Wt,Wt=null,cl=0,Z&6)throw Error(I(331));var i=Z;for(Z|=4,F=e.current;F!==null;){var l=F,o=l.child;if(F.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var u=a[s];for(F=u;F!==null;){var d=F;switch(d.tag){case 0:case 11:case 15:zr(8,d,l)}var f=d.child;if(f!==null)f.return=d,F=f;else for(;F!==null;){d=F;var g=d.sibling,m=d.return;if(Df(d),d===u){F=null;break}if(g!==null){g.return=m,F=g;break}F=m}}}var S=l.alternate;if(S!==null){var C=S.child;if(C!==null){S.child=null;do{var j=C.sibling;C.sibling=null,C=j}while(C!==null)}}F=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,F=o;else e:for(;F!==null;){if(l=F,l.flags&2048)switch(l.tag){case 0:case 11:case 15:zr(9,l,l.return)}var p=l.sibling;if(p!==null){p.return=l.return,F=p;break e}F=l.return}}var h=e.current;for(F=h;F!==null;){o=F;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,F=y;else e:for(o=h;F!==null;){if(a=F,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:bl(9,a)}}catch(b){pe(a,a.return,b)}if(a===o){F=null;break e}var k=a.sibling;if(k!==null){k.return=a.return,F=k;break e}F=a.return}}if(Z=i,ln(),wt&&typeof wt.onPostCommitFiberRoot=="function")try{wt.onPostCommitFiberRoot(gl,e)}catch{}r=!0}return r}finally{ne=n,lt.transition=t}}return!1}function $u(e,t,n){t=er(n,t),t=bf(e,t,1),e=Xt(e,t,1),t=Ae(),e!==null&&(ei(e,1,t),He(e,t))}function pe(e,t,n){if(e.tag===3)$u(e,e,n);else for(;t!==null;){if(t.tag===3){$u(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(Gt===null||!Gt.has(r))){e=er(n,e),e=Ef(t,e,1),t=Xt(t,e,1),e=Ae(),t!==null&&(ei(t,1,e),He(t,e));break}}t=t.return}}function rg(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Ae(),e.pingedLanes|=e.suspendedLanes&n,Se===e&&(be&n)===n&&(xe===4||xe===3&&(be&130023424)===be&&500>he()-cs?mn(e,0):us|=n),He(e,t)}function Wf(e,t){t===0&&(e.mode&1?(t=fi,fi<<=1,!(fi&130023424)&&(fi=4194304)):t=1);var n=Ae();e=At(e,t),e!==null&&(ei(e,t,n),He(e,n))}function ig(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),Wf(e,n)}function lg(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(I(314))}r!==null&&r.delete(t),Wf(e,n)}var Qf;Qf=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Be.current)Fe=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return Fe=!1,Qm(e,t,n);Fe=!!(e.flags&131072)}else Fe=!1,ue&&t.flags&1048576&&Yd(t,tl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Mi(e,t),e=t.pendingProps;var i=Xn(t,Pe.current);Wn(t,n),i=rs(null,t,r,e,i,n);var l=is();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Ue(r)?(l=!0,Zi(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,Ja(t),i.updater=Cl,t.stateNode=i,i._reactInternals=t,Jo(t,r,e,n),t=ta(null,t,r,!0,l,n)):(t.tag=0,ue&&l&&Wa(t),Ie(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Mi(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=ag(r),e=ct(r,e),i){case 0:t=ea(null,t,r,e,n);break e;case 1:t=Au(null,t,r,e,n);break e;case 11:t=Lu(null,t,r,e,n);break e;case 14:t=Iu(null,t,r,ct(r.type,e),n);break e}throw Error(I(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ct(r,i),ea(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ct(r,i),Au(e,t,r,i,n);case 3:e:{if(zf(t),e===null)throw Error(I(387));r=t.pendingProps,l=t.memoizedState,i=l.element,tf(e,t),il(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=er(Error(I(423)),t),t=Du(e,t,r,n,i);break e}else if(r!==i){i=er(Error(I(424)),t),t=Du(e,t,r,n,i);break e}else for(Ye=Yt(t.stateNode.containerInfo.firstChild),Ge=t,ue=!0,ft=null,n=Zd(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(Gn(),r===i){t=Dt(e,t,n);break e}Ie(e,t,r,n)}t=t.child}return t;case 5:return nf(t),e===null&&Yo(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,$o(r,i)?o=null:l!==null&&$o(r,l)&&(t.flags|=32),_f(e,t),Ie(e,t,o,n),t.child;case 6:return e===null&&Yo(t),null;case 13:return Pf(e,t,n);case 4:return Za(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=Jn(t,null,r,n):Ie(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ct(r,i),Lu(e,t,r,i,n);case 7:return Ie(e,t,t.pendingProps,n),t.child;case 8:return Ie(e,t,t.pendingProps.children,n),t.child;case 12:return Ie(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,le(nl,r._currentValue),r._currentValue=o,l!==null)if(mt(l.value,o)){if(l.children===i.children&&!Be.current){t=Dt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Tt(-1,n&-n),s.tag=2;var u=l.updateQueue;if(u!==null){u=u.shared;var d=u.pending;d===null?s.next=s:(s.next=d.next,d.next=s),u.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),Xo(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(I(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),Xo(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}Ie(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Wn(t,n),i=ot(i),r=r(i),t.flags|=1,Ie(e,t,r,n),t.child;case 14:return r=t.type,i=ct(r,t.pendingProps),i=ct(r.type,i),Iu(e,t,r,i,n);case 15:return jf(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ct(r,i),Mi(e,t),t.tag=1,Ue(r)?(e=!0,Zi(t)):e=!1,Wn(t,n),Cf(t,r,i),Jo(t,r,i,n),ta(null,t,r,!0,e,n);case 19:return Tf(e,t,n);case 22:return Nf(e,t,n)}throw Error(I(156,t.tag))};function qf(e,t){return xd(e,t)}function og(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function it(e,t,n,r){return new og(e,t,n,r)}function hs(e){return e=e.prototype,!(!e||!e.isReactComponent)}function ag(e){if(typeof e=="function")return hs(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Ia)return 11;if(e===Aa)return 14}return 2}function Zt(e,t){var n=e.alternate;return n===null?(n=it(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Fi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")hs(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Pn:return gn(n.children,i,l,t);case La:o=8,i|=8;break;case So:return e=it(12,n,t,i|2),e.elementType=So,e.lanes=l,e;case Co:return e=it(13,n,t,i),e.elementType=Co,e.lanes=l,e;case bo:return e=it(19,n,t,i),e.elementType=bo,e.lanes=l,e;case nd:return jl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case ed:o=10;break e;case td:o=9;break e;case Ia:o=11;break e;case Aa:o=14;break e;case Bt:o=16,r=null;break e}throw Error(I(130,e==null?e:typeof e,""))}return t=it(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function gn(e,t,n,r){return e=it(7,e,r,t),e.lanes=n,e}function jl(e,t,n,r){return e=it(22,e,r,t),e.elementType=nd,e.lanes=n,e.stateNode={isHidden:!1},e}function lo(e,t,n){return e=it(6,e,null,t),e.lanes=n,e}function oo(e,t,n){return t=it(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function sg(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Bl(0),this.expirationTimes=Bl(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Bl(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function ms(e,t,n,r,i,l,o,a,s){return e=new sg(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=it(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},Ja(l),e}function ug(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:zn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function Kf(e){if(!e)return tn;e=e._reactInternals;e:{if(Cn(e)!==e||e.tag!==1)throw Error(I(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Ue(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(I(171))}if(e.tag===1){var n=e.type;if(Ue(n))return qd(e,n,t)}return t}function Yf(e,t,n,r,i,l,o,a,s){return e=ms(n,r,!0,e,i,l,o,a,s),e.context=Kf(null),n=e.current,r=Ae(),i=Jt(n),l=Tt(r,i),l.callback=t??null,Xt(n,l,i),e.current.lanes=i,ei(e,i,r),He(e,r),e}function Nl(e,t,n,r){var i=t.current,l=Ae(),o=Jt(i);return n=Kf(n),t.context===null?t.context=n:t.pendingContext=n,t=Tt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=Xt(i,t,o),e!==null&&(ht(e,i,o,l),Ii(e,i,o)),o}function fl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function Wu(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function gs(e,t){Wu(e,t),(e=e.alternate)&&Wu(e,t)}function cg(){return null}var Xf=typeof reportError=="function"?reportError:function(e){console.error(e)};function vs(e){this._internalRoot=e}_l.prototype.render=vs.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(I(409));Nl(e,t,null,null)};_l.prototype.unmount=vs.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;wn(function(){Nl(null,e,null,null)}),t[It]=null}};function _l(e){this._internalRoot=e}_l.prototype.unstable_scheduleHydration=function(e){if(e){var t=jd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Ht.length&&t!==0&&t<Ht[n].priority;n++);Ht.splice(n,0,e),n===0&&_d(e)}};function ys(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function zl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function Qu(){}function dg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var u=fl(o);l.call(u)}}var o=Yf(t,r,e,0,null,!1,!1,"",Qu);return e._reactRootContainer=o,e[It]=o.current,Vr(e.nodeType===8?e.parentNode:e),wn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var u=fl(s);a.call(u)}}var s=ms(e,0,!1,null,null,!1,!1,"",Qu);return e._reactRootContainer=s,e[It]=s.current,Vr(e.nodeType===8?e.parentNode:e),wn(function(){Nl(t,s,n,r)}),s}function Pl(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=fl(o);a.call(s)}}Nl(t,o,e,i)}else o=dg(n,t,e,i,r);return fl(o)}bd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=wr(t.pendingLanes);n!==0&&(Ra(t,n|1),He(t,he()),!(Z&6)&&(tr=he()+500,ln()))}break;case 13:wn(function(){var r=At(e,1);if(r!==null){var i=Ae();ht(r,e,1,i)}}),gs(e,1)}};Oa=function(e){if(e.tag===13){var t=At(e,134217728);if(t!==null){var n=Ae();ht(t,e,134217728,n)}gs(e,134217728)}};Ed=function(e){if(e.tag===13){var t=Jt(e),n=At(e,t);if(n!==null){var r=Ae();ht(n,e,t,r)}gs(e,t)}};jd=function(){return ne};Nd=function(e,t){var n=ne;try{return ne=e,t()}finally{ne=n}};Ao=function(e,t,n){switch(t){case"input":if(No(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=kl(r);if(!i)throw Error(I(90));id(r),No(r,i)}}}break;case"textarea":od(e,n);break;case"select":t=n.value,t!=null&&Un(e,!!n.multiple,t,!1)}};pd=ds;hd=wn;var fg={usingClientEntryPoint:!1,Events:[ni,An,kl,dd,fd,ds]},gr={findFiberByHostInstance:fn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},pg={bundleType:gr.bundleType,version:gr.version,rendererPackageName:gr.rendererPackageName,rendererConfig:gr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Mt.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=vd(e),e===null?null:e.stateNode},findFiberByHostInstance:gr.findFiberByHostInstance||cg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var Ci=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!Ci.isDisabled&&Ci.supportsFiber)try{gl=Ci.inject(pg),wt=Ci}catch{}}Ze.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=fg;Ze.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!ys(t))throw Error(I(200));return ug(e,t,null,n)};Ze.createRoot=function(e,t){if(!ys(e))throw Error(I(299));var n=!1,r="",i=Xf;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=ms(e,1,!1,null,null,n,!1,r,i),e[It]=t.current,Vr(e.nodeType===8?e.parentNode:e),new vs(t)};Ze.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(I(188)):(e=Object.keys(e).join(","),Error(I(268,e)));return e=vd(t),e=e===null?null:e.stateNode,e};Ze.flushSync=function(e){return wn(e)};Ze.hydrate=function(e,t,n){if(!zl(t))throw Error(I(200));return Pl(null,e,t,!0,n)};Ze.hydrateRoot=function(e,t,n){if(!ys(e))throw Error(I(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=Xf;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=Yf(t,null,e,1,n??null,i,!1,l,o),e[It]=t.current,Vr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new _l(t)};Ze.render=function(e,t,n){if(!zl(t))throw Error(I(200));return Pl(null,e,t,!1,n)};Ze.unmountComponentAtNode=function(e){if(!zl(e))throw Error(I(40));return e._reactRootContainer?(wn(function(){Pl(null,null,e,!1,function(){e._reactRootContainer=null,e[It]=null})}),!0):!1};Ze.unstable_batchedUpdates=ds;Ze.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!zl(n))throw Error(I(200));if(e==null||e._reactInternals===void 0)throw Error(I(38));return Pl(e,t,n,!1,r)};Ze.version="18.3.1-next-f1338f8080-20240426";function Gf(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Gf)}catch(e){console.error(e)}}Gf(),Xc.exports=Ze;var hg=Xc.exports,qu=hg;ko.createRoot=qu.createRoot,ko.hydrateRoot=qu.hydrateRoot;const mg="",gg=({selection:e,onSelect:t,onRefresh:n})=>{const[r,i]=U.useState(null),[l,o]=U.useState(new Set(["all"])),[a,s]=U.useState(!0),[u,d]=U.useState(null),f=async()=>{try{const h=await fetch(`${mg}/api/hierarchy`);if(!h.ok)throw new Error("Failed to fetch hierarchy");const y=await h.json();i(y),d(null)}catch(h){d(h instanceof Error?h.message:"Unknown error")}finally{s(!1)}};U.useEffect(()=>{f();const h=setInterval(f,5e3);return()=>clearInterval(h)},[]);const g=h=>{o(y=>{const k=new Set(y);return k.has(h)?k.delete(h):k.add(h),k})},m=h=>{var y;if(h.type==="root")t({type:"overview"});else if(h.type==="agent")t({type:"agent",agentId:h.id});else if(h.type==="thread"){const k=(y=r==null?void 0:r.root.children)==null?void 0:y.find(b=>{var w;return(w=b.children)==null?void 0:w.some(z=>z.id===h.id)});t({type:"thread",agentId:k==null?void 0:k.id,threadId:h.id})}},S=h=>h.type==="root"&&e.type==="overview"||h.type==="agent"&&e.type==="agent"&&e.agentId===h.id||h.type==="thread"&&e.threadId===h.id,C=h=>!h||h.length===0?null:c.jsx("span",{className:"badges",children:h.map((y,k)=>c.jsxs("span",{className:`badge badge-${y.type}`,title:`${y.count} ${y.type}`,children:[y.type==="pending"&&"⏳",y.type==="unread"&&"📬",y.type==="running"&&"▶️",y.count]},k))}),j=h=>{if(!h)return null;const y={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return c.jsx("span",{className:"status-indicator",style:{backgroundColor:y[h]||y.idle},title:h})},p=(h,y=0)=>{const k=l.has(h.id),b=h.children&&h.children.length>0,w=S(h);return c.jsxs("div",{className:"tree-node",children:[c.jsxs("div",{className:`tree-node-content ${w?"selected":""} ${h.type}`,style:{paddingLeft:`${y*16+8}px`},onClick:()=>m(h),children:[b&&c.jsx("span",{className:`expand-icon ${k?"expanded":""}`,onClick:z=>{z.stopPropagation(),g(h.id)},children:k?"▼":"▶"}),!b&&c.jsx("span",{className:"expand-icon-placeholder"}),h.type==="agent"&&j(h.status),c.jsx("span",{className:"node-label",children:h.label}),C(h.badges)]}),b&&k&&c.jsx("div",{className:"tree-children",children:h.children.map(z=>p(z,y+1))})]},h.id)};return a&&!r?c.jsx("div",{className:"hierarchy-tree loading",children:"Loading..."}):u?c.jsxs("div",{className:"hierarchy-tree error",children:[c.jsxs("p",{children:["Error: ",u]}),c.jsx("button",{onClick:f,children:"Retry"})]}):c.jsxs("div",{className:"hierarchy-tree",children:[c.jsxs("div",{className:"tree-header",children:[c.jsx("h3",{children:"Agents"}),c.jsx("button",{className:"refresh-btn",onClick:()=>{f(),n==null||n()},title:"Refresh",children:"\\u21BB"})]}),c.jsx("div",{className:"tree-content",children:r&&p(r.root)}),r&&c.jsx("div",{className:"tree-footer",children:c.jsxs("div",{className:"aggregate-stats",children:[c.jsxs("span",{title:"Total agents",children:[r.aggregate.total_agents," agents"]}),c.jsxs("span",{title:"Active",children:[r.aggregate.active_agents," active"]}),r.aggregate.pending_approvals>0&&c.jsxs("span",{className:"pending",title:"Pending approvals",children:[r.aggregate.pending_approvals," pending"]})]})})]})},bi=({title:e,value:t,color:n="default"})=>c.jsxs("div",{className:`stat-card stat-${n}`,children:[c.jsx("div",{className:"stat-value",children:t}),c.jsx("div",{className:"stat-title",children:e})]}),vg=({agent:e,onClick:t})=>{var o,a,s,u,d;const n=((o=e.children)==null?void 0:o.length)||0,r=((s=(a=e.badges)==null?void 0:a.find(f=>f.type==="pending"))==null?void 0:s.count)||0,i=((d=(u=e.badges)==null?void 0:u.find(f=>f.type==="running"))==null?void 0:d.count)||0,l={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return c.jsxs("div",{className:"agent-card",onClick:t,children:[c.jsxs("div",{className:"agent-card-header",children:[c.jsx("span",{className:"agent-status-dot",style:{backgroundColor:l[e.status||"idle"]}}),c.jsx("span",{className:"agent-name",children:e.label})]}),c.jsxs("div",{className:"agent-card-stats",children:[c.jsxs("span",{className:"agent-stat",children:[c.jsx("span",{className:"agent-stat-value",children:n}),c.jsx("span",{className:"agent-stat-label",children:"threads"})]}),r>0&&c.jsxs("span",{className:"agent-stat pending",children:[c.jsx("span",{className:"agent-stat-value",children:r}),c.jsx("span",{className:"agent-stat-label",children:"pending"})]}),i>0&&c.jsxs("span",{className:"agent-stat running",children:[c.jsx("span",{className:"agent-stat-value",children:i}),c.jsx("span",{className:"agent-stat-label",children:"running"})]})]})]})},yg=({aggregate:e,agents:t,onSelectAgent:n})=>c.jsxs("div",{className:"all-agents-overview",children:[c.jsx("div",{className:"overview-header",children:c.jsx("h2",{children:"All Agents Overview"})}),c.jsxs("div",{className:"stats-row",children:[c.jsx(bi,{title:"Total Agents",value:e.total_agents}),c.jsx(bi,{title:"Active",value:e.active_agents,color:"green"}),c.jsx(bi,{title:"Pending Approvals",value:e.pending_approvals,color:"orange"}),c.jsx(bi,{title:"Total Threads",value:e.total_threads,color:"blue"})]}),c.jsxs("div",{className:"agents-section",children:[c.jsx("h3",{children:"Agents"}),c.jsxs("div",{className:"agent-cards-grid",children:[t.map(r=>c.jsx(vg,{agent:r,onClick:()=>n(r.id)},r.id)),t.length===0&&c.jsx("div",{className:"no-agents",children:"No agents found. Start an agent to see it here."})]})]})]}),xg=({items:e})=>c.jsx("nav",{className:"breadcrumb",children:e.map((t,n)=>c.jsxs(Ft.Fragment,{children:[n>0&&c.jsx("span",{className:"breadcrumb-separator",children:"/"}),t.onClick?c.jsx("button",{className:"breadcrumb-link",onClick:t.onClick,children:t.label}):c.jsx("span",{className:"breadcrumb-current",children:t.label})]},n))}),jt={plus:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),c.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),c.jsx("circle",{cx:"12",cy:"5",r:"2"}),c.jsx("path",{d:"M12 7v4"}),c.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),c.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),c.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),c.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),c.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:c.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),c.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:c.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("polyline",{points:"3 6 5 6 21 6"}),c.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:c.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:c.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),c.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},kg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=U.useState(!1),[u,d]=U.useState(""),[f,g]=U.useState(null),[m,S]=U.useState(""),[C,j]=U.useState(null),p=()=>{u.trim()&&(r(u.trim()),d(""),s(!1))},h=_=>{_.key==="Enter"&&!_.shiftKey?(_.preventDefault(),p()):_.key==="Escape"&&(s(!1),d(""))},y=(_,M)=>{M.stopPropagation(),g(_.id),S(_.title)},k=_=>{var M;m.trim()&&m.trim()!==((M=e.find(Y=>Y.id===_))==null?void 0:M.title)&&l(_,m.trim()),g(null),S("")},b=()=>{g(null),S("")},w=(_,M)=>{_.key==="Enter"?(_.preventDefault(),k(M)):_.key==="Escape"&&b()},z=(_,M)=>{M.stopPropagation(),j(_)},D=(_,M)=>{M.stopPropagation(),i(_),j(null)},H=_=>{_.stopPropagation(),j(null)},O=_=>{const M=new Date(_),G=new Date().getTime()-M.getTime(),$=Math.floor(G/6e4),P=Math.floor(G/36e5),V=Math.floor(G/864e5);return $<1?"now":$<60?`${$}m`:P<24?`${P}h`:V<7?`${V}d`:M.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return c.jsxs("div",{className:"thread-list",children:[c.jsxs("div",{className:"list-header",children:[c.jsx("h2",{children:"Conversations"}),c.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:jt.plus})]}),a&&c.jsxs("div",{className:"new-thread-form",children:[c.jsx("input",{type:"text",value:u,onChange:_=>d(_.target.value),onKeyDown:h,placeholder:"Conversation title...",autoFocus:!0}),c.jsxs("div",{className:"form-actions",children:[c.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),c.jsx("button",{className:"create-btn",onClick:p,children:"Create"})]})]}),c.jsx("div",{className:"thread-items",children:e.length===0?c.jsxs("div",{className:"empty-state",children:[c.jsx("div",{className:"empty-icon",children:jt.hash}),c.jsx("p",{children:"No conversations yet"}),c.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(_=>{const M=o.get(_.id)||0,Y=_.id===t,G=f===_.id,$=C===_.id;return c.jsxs("div",{className:`thread-item ${Y?"selected":""} ${M>0?"has-unread":""}`,onClick:()=>!G&&n(_.id),children:[c.jsx("div",{className:`status-dot ${_.status}`}),c.jsxs("div",{className:"thread-content",children:[c.jsx("div",{className:"thread-title-row",children:G?c.jsxs("div",{className:"edit-title-form",onClick:P=>P.stopPropagation(),children:[c.jsx("input",{type:"text",value:m,onChange:P=>S(P.target.value),onKeyDown:P=>w(P,_.id),autoFocus:!0}),c.jsx("button",{className:"edit-action save",onClick:()=>k(_.id),title:"Save",children:jt.check}),c.jsx("button",{className:"edit-action cancel",onClick:b,title:"Cancel",children:jt.x})]}):c.jsxs(c.Fragment,{children:[c.jsx("span",{className:"thread-title",children:_.title}),c.jsx("span",{className:"thread-time",children:O(_.updated_at)})]})}),c.jsxs("div",{className:"thread-meta",children:[_.target_agent&&c.jsxs("span",{className:"thread-agent",title:`Target: ${_.target_agent}`,children:[jt.bot,_.target_agent]}),c.jsxs("span",{className:"thread-seq",children:["#",_.last_seq]})]})]}),!G&&!$&&c.jsxs("div",{className:"thread-actions",children:[c.jsx("button",{className:"action-btn edit",onClick:P=>y(_,P),title:"Rename",children:jt.edit}),c.jsx("button",{className:"action-btn delete",onClick:P=>z(_.id,P),title:"Delete",children:jt.trash})]}),$&&c.jsxs("div",{className:"delete-confirm",onClick:P=>P.stopPropagation(),children:[c.jsx("span",{className:"confirm-text",children:"Delete?"}),c.jsx("button",{className:"confirm-btn yes",onClick:P=>D(_.id,P),title:"Confirm delete",children:jt.check}),c.jsx("button",{className:"confirm-btn no",onClick:H,title:"Cancel",children:jt.x})]}),M>0&&!$&&c.jsx("span",{className:"unread-badge",children:M})]},_.id)})}),c.jsx("style",{children:`
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
      `})]})};function wg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const Sg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,Cg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,bg={};function Ku(e,t){return(bg.jsx?Cg:Sg).test(e)}const Eg=/[ \t\n\f\r]/g;function jg(e){return typeof e=="object"?e.type==="text"?Yu(e.value):!1:Yu(e)}function Yu(e){return e.replace(Eg,"")===""}class ii{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}ii.prototype.normal={};ii.prototype.property={};ii.prototype.space=void 0;function Jf(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new ii(n,r,t)}function pa(e){return e.toLowerCase()}class $e{constructor(t,n){this.attribute=n,this.property=t}}$e.prototype.attribute="";$e.prototype.booleanish=!1;$e.prototype.boolean=!1;$e.prototype.commaOrSpaceSeparated=!1;$e.prototype.commaSeparated=!1;$e.prototype.defined=!1;$e.prototype.mustUseProperty=!1;$e.prototype.number=!1;$e.prototype.overloadedBoolean=!1;$e.prototype.property="";$e.prototype.spaceSeparated=!1;$e.prototype.space=void 0;let Ng=0;const K=bn(),me=bn(),ha=bn(),A=bn(),ie=bn(),qn=bn(),Qe=bn();function bn(){return 2**++Ng}const ma=Object.freeze(Object.defineProperty({__proto__:null,boolean:K,booleanish:me,commaOrSpaceSeparated:Qe,commaSeparated:qn,number:A,overloadedBoolean:ha,spaceSeparated:ie},Symbol.toStringTag,{value:"Module"})),ao=Object.keys(ma);class xs extends $e{constructor(t,n,r,i){let l=-1;if(super(t,n),Xu(this,"space",i),typeof r=="number")for(;++l<ao.length;){const o=ao[l];Xu(this,ao[l],(r&ma[o])===ma[o])}}}xs.prototype.defined=!0;function Xu(e,t,n){n&&(e[t]=n)}function lr(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new xs(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[pa(r)]=r,n[pa(l.attribute)]=r}return new ii(t,n,e.space)}const Zf=lr({properties:{ariaActiveDescendant:null,ariaAtomic:me,ariaAutoComplete:null,ariaBusy:me,ariaChecked:me,ariaColCount:A,ariaColIndex:A,ariaColSpan:A,ariaControls:ie,ariaCurrent:null,ariaDescribedBy:ie,ariaDetails:null,ariaDisabled:me,ariaDropEffect:ie,ariaErrorMessage:null,ariaExpanded:me,ariaFlowTo:ie,ariaGrabbed:me,ariaHasPopup:null,ariaHidden:me,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:ie,ariaLevel:A,ariaLive:null,ariaModal:me,ariaMultiLine:me,ariaMultiSelectable:me,ariaOrientation:null,ariaOwns:ie,ariaPlaceholder:null,ariaPosInSet:A,ariaPressed:me,ariaReadOnly:me,ariaRelevant:null,ariaRequired:me,ariaRoleDescription:ie,ariaRowCount:A,ariaRowIndex:A,ariaRowSpan:A,ariaSelected:me,ariaSetSize:A,ariaSort:null,ariaValueMax:A,ariaValueMin:A,ariaValueNow:A,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function ep(e,t){return t in e?e[t]:t}function tp(e,t){return ep(e,t.toLowerCase())}const _g=lr({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:qn,acceptCharset:ie,accessKey:ie,action:null,allow:null,allowFullScreen:K,allowPaymentRequest:K,allowUserMedia:K,alt:null,as:null,async:K,autoCapitalize:null,autoComplete:ie,autoFocus:K,autoPlay:K,blocking:ie,capture:null,charSet:null,checked:K,cite:null,className:ie,cols:A,colSpan:null,content:null,contentEditable:me,controls:K,controlsList:ie,coords:A|qn,crossOrigin:null,data:null,dateTime:null,decoding:null,default:K,defer:K,dir:null,dirName:null,disabled:K,download:ha,draggable:me,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:K,formTarget:null,headers:ie,height:A,hidden:ha,high:A,href:null,hrefLang:null,htmlFor:ie,httpEquiv:ie,id:null,imageSizes:null,imageSrcSet:null,inert:K,inputMode:null,integrity:null,is:null,isMap:K,itemId:null,itemProp:ie,itemRef:ie,itemScope:K,itemType:ie,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:K,low:A,manifest:null,max:null,maxLength:A,media:null,method:null,min:null,minLength:A,multiple:K,muted:K,name:null,nonce:null,noModule:K,noValidate:K,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:K,optimum:A,pattern:null,ping:ie,placeholder:null,playsInline:K,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:K,referrerPolicy:null,rel:ie,required:K,reversed:K,rows:A,rowSpan:A,sandbox:ie,scope:null,scoped:K,seamless:K,selected:K,shadowRootClonable:K,shadowRootDelegatesFocus:K,shadowRootMode:null,shape:null,size:A,sizes:null,slot:null,span:A,spellCheck:me,src:null,srcDoc:null,srcLang:null,srcSet:null,start:A,step:null,style:null,tabIndex:A,target:null,title:null,translate:null,type:null,typeMustMatch:K,useMap:null,value:me,width:A,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:ie,axis:null,background:null,bgColor:null,border:A,borderColor:null,bottomMargin:A,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:K,declare:K,event:null,face:null,frame:null,frameBorder:null,hSpace:A,leftMargin:A,link:null,longDesc:null,lowSrc:null,marginHeight:A,marginWidth:A,noResize:K,noHref:K,noShade:K,noWrap:K,object:null,profile:null,prompt:null,rev:null,rightMargin:A,rules:null,scheme:null,scrolling:me,standby:null,summary:null,text:null,topMargin:A,valueType:null,version:null,vAlign:null,vLink:null,vSpace:A,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:K,disableRemotePlayback:K,prefix:null,property:null,results:A,security:null,unselectable:null},space:"html",transform:tp}),zg=lr({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:Qe,accentHeight:A,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:A,amplitude:A,arabicForm:null,ascent:A,attributeName:null,attributeType:null,azimuth:A,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:A,by:null,calcMode:null,capHeight:A,className:ie,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:A,diffuseConstant:A,direction:null,display:null,dur:null,divisor:A,dominantBaseline:null,download:K,dx:null,dy:null,edgeMode:null,editable:null,elevation:A,enableBackground:null,end:null,event:null,exponent:A,externalResourcesRequired:null,fill:null,fillOpacity:A,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:qn,g2:qn,glyphName:qn,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:A,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:A,horizOriginX:A,horizOriginY:A,id:null,ideographic:A,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:A,k:A,k1:A,k2:A,k3:A,k4:A,kernelMatrix:Qe,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:A,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:A,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:A,overlineThickness:A,paintOrder:null,panose1:null,path:null,pathLength:A,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:ie,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:A,pointsAtY:A,pointsAtZ:A,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:Qe,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:Qe,rev:Qe,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:Qe,requiredFeatures:Qe,requiredFonts:Qe,requiredFormats:Qe,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:A,specularExponent:A,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:A,strikethroughThickness:A,string:null,stroke:null,strokeDashArray:Qe,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:A,strokeOpacity:A,strokeWidth:null,style:null,surfaceScale:A,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:Qe,tabIndex:A,tableValues:null,target:null,targetX:A,targetY:A,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:Qe,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:A,underlineThickness:A,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:A,values:null,vAlphabetic:A,vMathematical:A,vectorEffect:null,vHanging:A,vIdeographic:A,version:null,vertAdvY:A,vertOriginX:A,vertOriginY:A,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:A,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:ep}),np=lr({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),rp=lr({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:tp}),ip=lr({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),Pg={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Tg=/[A-Z]/g,Gu=/-[a-z]/g,Lg=/^data[-\w.:]+$/i;function Ig(e,t){const n=pa(t);let r=t,i=$e;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&Lg.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(Gu,Dg);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!Gu.test(l)){let o=l.replace(Tg,Ag);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=xs}return new i(r,t)}function Ag(e){return"-"+e.toLowerCase()}function Dg(e){return e.charAt(1).toUpperCase()}const Mg=Jf([Zf,_g,np,rp,ip],"html"),ks=Jf([Zf,zg,np,rp,ip],"svg");function Rg(e){return e.join(" ").trim()}var ws={},Ju=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Og=/\n/g,Fg=/^\s*/,Bg=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Ug=/^:\s*/,Hg=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Vg=/^[;\s]*/,$g=/^\s+|\s+$/g,Wg=`
`,Zu="/",ec="*",dn="",Qg="comment",qg="declaration";function Kg(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(S){var C=S.match(Og);C&&(n+=C.length);var j=S.lastIndexOf(Wg);r=~j?S.length-j:r+S.length}function l(){var S={line:n,column:r};return function(C){return C.position=new o(S),u(),C}}function o(S){this.start=S,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(S){var C=new Error(t.source+":"+n+":"+r+": "+S);if(C.reason=S,C.filename=t.source,C.line=n,C.column=r,C.source=e,!t.silent)throw C}function s(S){var C=S.exec(e);if(C){var j=C[0];return i(j),e=e.slice(j.length),C}}function u(){s(Fg)}function d(S){var C;for(S=S||[];C=f();)C!==!1&&S.push(C);return S}function f(){var S=l();if(!(Zu!=e.charAt(0)||ec!=e.charAt(1))){for(var C=2;dn!=e.charAt(C)&&(ec!=e.charAt(C)||Zu!=e.charAt(C+1));)++C;if(C+=2,dn===e.charAt(C-1))return a("End of comment missing");var j=e.slice(2,C-2);return r+=2,i(j),e=e.slice(C),r+=2,S({type:Qg,comment:j})}}function g(){var S=l(),C=s(Bg);if(C){if(f(),!s(Ug))return a("property missing ':'");var j=s(Hg),p=S({type:qg,property:tc(C[0].replace(Ju,dn)),value:j?tc(j[0].replace(Ju,dn)):dn});return s(Vg),p}}function m(){var S=[];d(S);for(var C;C=g();)C!==!1&&(S.push(C),d(S));return S}return u(),m()}function tc(e){return e?e.replace($g,dn):dn}var Yg=Kg,Xg=Hi&&Hi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(ws,"__esModule",{value:!0});ws.default=Jg;const Gg=Xg(Yg);function Jg(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Gg.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Tl={};Object.defineProperty(Tl,"__esModule",{value:!0});Tl.camelCase=void 0;var Zg=/^--[a-zA-Z0-9_-]+$/,ev=/-([a-z])/g,tv=/^[^-]+$/,nv=/^-(webkit|moz|ms|o|khtml)-/,rv=/^-(ms)-/,iv=function(e){return!e||tv.test(e)||Zg.test(e)},lv=function(e,t){return t.toUpperCase()},nc=function(e,t){return"".concat(t,"-")},ov=function(e,t){return t===void 0&&(t={}),iv(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(rv,nc):e=e.replace(nv,nc),e.replace(ev,lv))};Tl.camelCase=ov;var av=Hi&&Hi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},sv=av(ws),uv=Tl;function ga(e,t){var n={};return!e||typeof e!="string"||(0,sv.default)(e,function(r,i){r&&i&&(n[(0,uv.camelCase)(r,t)]=i)}),n}ga.default=ga;var cv=ga;const dv=ba(cv),lp=op("end"),Ss=op("start");function op(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function fv(e){const t=Ss(e),n=lp(e);if(t&&n)return{start:t,end:n}}function Lr(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?rc(e.position):"start"in e||"end"in e?rc(e):"line"in e||"column"in e?va(e):""}function va(e){return ic(e&&e.line)+":"+ic(e&&e.column)}function rc(e){return va(e&&e.start)+"-"+va(e&&e.end)}function ic(e){return e&&typeof e=="number"?e:1}class Te extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Lr(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}Te.prototype.file="";Te.prototype.name="";Te.prototype.reason="";Te.prototype.message="";Te.prototype.stack="";Te.prototype.column=void 0;Te.prototype.line=void 0;Te.prototype.ancestors=void 0;Te.prototype.cause=void 0;Te.prototype.fatal=void 0;Te.prototype.place=void 0;Te.prototype.ruleId=void 0;Te.prototype.source=void 0;const Cs={}.hasOwnProperty,pv=new Map,hv=/[A-Z]/g,mv=new Set(["table","tbody","thead","tfoot","tr"]),gv=new Set(["td","th"]),ap="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function vv(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=Ev(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=bv(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?ks:Mg,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=sp(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function sp(e,t,n){if(t.type==="element")return yv(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return xv(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return wv(e,t,n);if(t.type==="mdxjsEsm")return kv(e,t);if(t.type==="root")return Sv(e,t,n);if(t.type==="text")return Cv(e,t)}function yv(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=ks,e.schema=i),e.ancestors.push(t);const l=cp(e,t.tagName,!1),o=jv(e,t);let a=Es(e,t);return mv.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!jg(s):!0})),up(e,o,l,t),bs(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function xv(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}Jr(e,t.position)}function kv(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);Jr(e,t.position)}function wv(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=ks,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:cp(e,t.name,!0),o=Nv(e,t),a=Es(e,t);return up(e,o,l,t),bs(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function Sv(e,t,n){const r={};return bs(r,Es(e,t)),e.create(t,e.Fragment,r,n)}function Cv(e,t){return t.value}function up(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function bs(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function bv(e,t,n){return r;function r(i,l,o,a){const u=Array.isArray(o.children)?n:t;return a?u(l,o,a):u(l,o)}}function Ev(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=Ss(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function jv(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Cs.call(t.properties,i)){const l=_v(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&gv.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function Nv(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else Jr(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else Jr(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Es(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:pv;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const u=i.get(s)||0;o=s+"-"+u,i.set(s,u+1)}}const a=sp(e,l,o);a!==void 0&&n.push(a)}return n}function _v(e,t,n){const r=Ig(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?wg(n):Rg(n)),r.property==="style"){let i=typeof n=="object"?n:zv(e,String(n));return e.stylePropertyNameCase==="css"&&(i=Pv(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?Pg[r.property]||r.property:r.attribute,n]}}function zv(e,t){try{return dv(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new Te("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=ap+"#cannot-parse-style-attribute",i}}function cp(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=Ku(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=Ku(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Cs.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);Jr(e)}function Jr(e,t){const n=new Te("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=ap+"#cannot-handle-mdx-estrees-without-createevaluater",n}function Pv(e){const t={};let n;for(n in e)Cs.call(e,n)&&(t[Tv(n)]=e[n]);return t}function Tv(e){let t=e.replace(hv,Lv);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function Lv(e){return"-"+e.toLowerCase()}const so={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},Iv={};function Av(e,t){const n=Iv,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return dp(e,r,i)}function dp(e,t,n){if(Dv(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return lc(e.children,t,n)}return Array.isArray(e)?lc(e,t,n):""}function lc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=dp(e[i],t,n);return r.join("")}function Dv(e){return!!(e&&typeof e=="object")}const oc=document.createElement("i");function js(e){const t="&"+e+";";oc.innerHTML=t;const n=oc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function Ct(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function rt(e,t){return e.length>0?(Ct(e,e.length,0,t),e):t}const ac={}.hasOwnProperty;function Mv(e){const t={};let n=-1;for(;++n<e.length;)Rv(t,e[n]);return t}function Rv(e,t){let n;for(n in t){const i=(ac.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){ac.call(i,o)||(i[o]=[]);const a=l[o];Ov(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Ov(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);Ct(e,0,0,r)}function fp(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function Kn(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const kt=on(/[A-Za-z]/),Xe=on(/[\dA-Za-z]/),Fv=on(/[#-'*+\--9=?A-Z^-~]/);function ya(e){return e!==null&&(e<32||e===127)}const xa=on(/\d/),Bv=on(/[\dA-Fa-f]/),Uv=on(/[!-/:-@[-`{-~]/);function W(e){return e!==null&&e<-2}function Ve(e){return e!==null&&(e<0||e===32)}function ee(e){return e===-2||e===-1||e===32}const Hv=on(new RegExp("\\p{P}|\\p{S}","u")),Vv=on(/\s/);function on(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function or(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&Xe(e.charCodeAt(n+1))&&Xe(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function oe(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return ee(s)?(e.enter(n),a(s)):t(s)}function a(s){return ee(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const $v={tokenize:Wv};function Wv(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),oe(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return W(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const Qv={tokenize:qv},sc={tokenize:Kv};function qv(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const k=n[r];return t.containerState=k[1],e.attempt(k[0].continuation,s,u)(y)}return u(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&h();const k=t.events.length;let b=k,w;for(;b--;)if(t.events[b][0]==="exit"&&t.events[b][1].type==="chunkFlow"){w=t.events[b][1].end;break}p(r);let z=k;for(;z<t.events.length;)t.events[z][1].end={...w},z++;return Ct(t.events,b+1,0,t.events.slice(k)),t.events.length=z,u(y)}return a(y)}function u(y){if(r===n.length){if(!i)return g(y);if(i.currentConstruct&&i.currentConstruct.concrete)return S(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(sc,d,f)(y)}function d(y){return i&&h(),p(r),g(y)}function f(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,S(y)}function g(y){return t.containerState={},e.attempt(sc,m,S)(y)}function m(y){return r++,n.push([t.currentConstruct,t.containerState]),g(y)}function S(y){if(y===null){i&&h(),p(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),C(y)}function C(y){if(y===null){j(e.exit("chunkFlow"),!0),p(0),e.consume(y);return}return W(y)?(e.consume(y),j(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),C)}function j(y,k){const b=t.sliceStream(y);if(k&&b.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(b),t.parser.lazy[y.start.line]){let w=i.events.length;for(;w--;)if(i.events[w][1].start.offset<o&&(!i.events[w][1].end||i.events[w][1].end.offset>o))return;const z=t.events.length;let D=z,H,O;for(;D--;)if(t.events[D][0]==="exit"&&t.events[D][1].type==="chunkFlow"){if(H){O=t.events[D][1].end;break}H=!0}for(p(r),w=z;w<t.events.length;)t.events[w][1].end={...O},w++;Ct(t.events,D+1,0,t.events.slice(z)),t.events.length=w}}function p(y){let k=n.length;for(;k-- >y;){const b=n[k];t.containerState=b[1],b[0].exit.call(t,e)}n.length=y}function h(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Kv(e,t,n){return oe(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function uc(e){if(e===null||Ve(e)||Vv(e))return 1;if(Hv(e))return 2}function Ns(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const ka={name:"attention",resolveAll:Yv,tokenize:Xv};function Yv(e,t){let n=-1,r,i,l,o,a,s,u,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const f={...e[r][1].end},g={...e[n][1].start};cc(f,-s),cc(g,s),o={type:s>1?"strongSequence":"emphasisSequence",start:f,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:g},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},u=[],e[r][1].end.offset-e[r][1].start.offset&&(u=rt(u,[["enter",e[r][1],t],["exit",e[r][1],t]])),u=rt(u,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),u=rt(u,Ns(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),u=rt(u,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,u=rt(u,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,Ct(e,r-1,n-r+3,u),n=r+u.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Xv(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=uc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const u=e.exit("attentionSequence"),d=uc(s),f=!d||d===2&&i||n.includes(s),g=!i||i===2&&d||n.includes(r);return u._open=!!(l===42?f:f&&(i||!g)),u._close=!!(l===42?g:g&&(d||!f)),t(s)}}function cc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Gv={name:"autolink",tokenize:Jv};function Jv(e,t,n){let r=0;return i;function i(m){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(m),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(m){return kt(m)?(e.consume(m),o):m===64?n(m):u(m)}function o(m){return m===43||m===45||m===46||Xe(m)?(r=1,a(m)):u(m)}function a(m){return m===58?(e.consume(m),r=0,s):(m===43||m===45||m===46||Xe(m))&&r++<32?(e.consume(m),a):(r=0,u(m))}function s(m){return m===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(m),e.exit("autolinkMarker"),e.exit("autolink"),t):m===null||m===32||m===60||ya(m)?n(m):(e.consume(m),s)}function u(m){return m===64?(e.consume(m),d):Fv(m)?(e.consume(m),u):n(m)}function d(m){return Xe(m)?f(m):n(m)}function f(m){return m===46?(e.consume(m),r=0,d):m===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(m),e.exit("autolinkMarker"),e.exit("autolink"),t):g(m)}function g(m){if((m===45||Xe(m))&&r++<63){const S=m===45?g:f;return e.consume(m),S}return n(m)}}const Ll={partial:!0,tokenize:Zv};function Zv(e,t,n){return r;function r(l){return ee(l)?oe(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||W(l)?t(l):n(l)}}const pp={continuation:{tokenize:ty},exit:ny,name:"blockQuote",tokenize:ey};function ey(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return ee(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function ty(e,t,n){const r=this;return i;function i(o){return ee(o)?oe(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(pp,t,n)(o)}}function ny(e){e.exit("blockQuote")}const hp={name:"characterEscape",tokenize:ry};function ry(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return Uv(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const mp={name:"characterReference",tokenize:iy};function iy(e,t,n){const r=this;let i=0,l,o;return a;function a(f){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),s}function s(f){return f===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(f),e.exit("characterReferenceMarkerNumeric"),u):(e.enter("characterReferenceValue"),l=31,o=Xe,d(f))}function u(f){return f===88||f===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(f),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=Bv,d):(e.enter("characterReferenceValue"),l=7,o=xa,d(f))}function d(f){if(f===59&&i){const g=e.exit("characterReferenceValue");return o===Xe&&!js(r.sliceSerialize(g))?n(f):(e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(f)&&i++<l?(e.consume(f),d):n(f)}}const dc={partial:!0,tokenize:oy},fc={concrete:!0,name:"codeFenced",tokenize:ly};function ly(e,t,n){const r=this,i={partial:!0,tokenize:b};let l=0,o=0,a;return s;function s(w){return u(w)}function u(w){const z=r.events[r.events.length-1];return l=z&&z[1].type==="linePrefix"?z[2].sliceSerialize(z[1],!0).length:0,a=w,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(w)}function d(w){return w===a?(o++,e.consume(w),d):o<3?n(w):(e.exit("codeFencedFenceSequence"),ee(w)?oe(e,f,"whitespace")(w):f(w))}function f(w){return w===null||W(w)?(e.exit("codeFencedFence"),r.interrupt?t(w):e.check(dc,C,k)(w)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),g(w))}function g(w){return w===null||W(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),f(w)):ee(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),oe(e,m,"whitespace")(w)):w===96&&w===a?n(w):(e.consume(w),g)}function m(w){return w===null||W(w)?f(w):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),S(w))}function S(w){return w===null||W(w)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),f(w)):w===96&&w===a?n(w):(e.consume(w),S)}function C(w){return e.attempt(i,k,j)(w)}function j(w){return e.enter("lineEnding"),e.consume(w),e.exit("lineEnding"),p}function p(w){return l>0&&ee(w)?oe(e,h,"linePrefix",l+1)(w):h(w)}function h(w){return w===null||W(w)?e.check(dc,C,k)(w):(e.enter("codeFlowValue"),y(w))}function y(w){return w===null||W(w)?(e.exit("codeFlowValue"),h(w)):(e.consume(w),y)}function k(w){return e.exit("codeFenced"),t(w)}function b(w,z,D){let H=0;return O;function O($){return w.enter("lineEnding"),w.consume($),w.exit("lineEnding"),_}function _($){return w.enter("codeFencedFence"),ee($)?oe(w,M,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)($):M($)}function M($){return $===a?(w.enter("codeFencedFenceSequence"),Y($)):D($)}function Y($){return $===a?(H++,w.consume($),Y):H>=o?(w.exit("codeFencedFenceSequence"),ee($)?oe(w,G,"whitespace")($):G($)):D($)}function G($){return $===null||W($)?(w.exit("codeFencedFence"),z($)):D($)}}}function oy(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const uo={name:"codeIndented",tokenize:sy},ay={partial:!0,tokenize:uy};function sy(e,t,n){const r=this;return i;function i(u){return e.enter("codeIndented"),oe(e,l,"linePrefix",5)(u)}function l(u){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(u):n(u)}function o(u){return u===null?s(u):W(u)?e.attempt(ay,o,s)(u):(e.enter("codeFlowValue"),a(u))}function a(u){return u===null||W(u)?(e.exit("codeFlowValue"),o(u)):(e.consume(u),a)}function s(u){return e.exit("codeIndented"),t(u)}}function uy(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):W(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):oe(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):W(o)?i(o):n(o)}}const cy={name:"codeText",previous:fy,resolve:dy,tokenize:py};function dy(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function fy(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function py(e,t,n){let r=0,i,l;return o;function o(f){return e.enter("codeText"),e.enter("codeTextSequence"),a(f)}function a(f){return f===96?(e.consume(f),r++,a):(e.exit("codeTextSequence"),s(f))}function s(f){return f===null?n(f):f===32?(e.enter("space"),e.consume(f),e.exit("space"),s):f===96?(l=e.enter("codeTextSequence"),i=0,d(f)):W(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),s):(e.enter("codeTextData"),u(f))}function u(f){return f===null||f===32||f===96||W(f)?(e.exit("codeTextData"),s(f)):(e.consume(f),u)}function d(f){return f===96?(e.consume(f),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(f)):(l.type="codeTextData",u(f))}}class hy{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&vr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),vr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),vr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);vr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);vr(this.left,n.reverse())}}}function vr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function gp(e){const t={};let n=-1,r,i,l,o,a,s,u;const d=new hy(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,my(d,n)),n=t[n],u=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return Ct(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!u}function my(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],u={};let d,f,g=-1,m=n,S=0,C=0;const j=[C];for(;m;){for(;e.get(++i)[1]!==m;);l.push(i),m._tokenizer||(d=r.sliceStream(m),m.next||d.push(null),f&&o.defineSkip(m.start),m._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),m._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),f=m,m=m.next}for(m=n;++g<a.length;)a[g][0]==="exit"&&a[g-1][0]==="enter"&&a[g][1].type===a[g-1][1].type&&a[g][1].start.line!==a[g][1].end.line&&(C=g+1,j.push(C),m._tokenizer=void 0,m.previous=void 0,m=m.next);for(o.events=[],m?(m._tokenizer=void 0,m.previous=void 0):j.pop(),g=j.length;g--;){const p=a.slice(j[g],j[g+1]),h=l.pop();s.push([h,h+p.length-1]),e.splice(h,2,p)}for(s.reverse(),g=-1;++g<s.length;)u[S+s[g][0]]=S+s[g][1],S+=s[g][1]-s[g][0]-1;return u}const gy={resolve:yy,tokenize:xy},vy={partial:!0,tokenize:ky};function yy(e){return gp(e),e}function xy(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):W(a)?e.check(vy,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function ky(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),oe(e,l,"linePrefix")}function l(o){if(o===null||W(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function vp(e,t,n,r,i,l,o,a,s){const u=s||Number.POSITIVE_INFINITY;let d=0;return f;function f(p){return p===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(p),e.exit(l),g):p===null||p===32||p===41||ya(p)?n(p):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),C(p))}function g(p){return p===62?(e.enter(l),e.consume(p),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),m(p))}function m(p){return p===62?(e.exit("chunkString"),e.exit(a),g(p)):p===null||p===60||W(p)?n(p):(e.consume(p),p===92?S:m)}function S(p){return p===60||p===62||p===92?(e.consume(p),m):m(p)}function C(p){return!d&&(p===null||p===41||Ve(p))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(p)):d<u&&p===40?(e.consume(p),d++,C):p===41?(e.consume(p),d--,C):p===null||p===32||p===40||ya(p)?n(p):(e.consume(p),p===92?j:C)}function j(p){return p===40||p===41||p===92?(e.consume(p),C):C(p)}}function yp(e,t,n,r,i,l){const o=this;let a=0,s;return u;function u(m){return e.enter(r),e.enter(i),e.consume(m),e.exit(i),e.enter(l),d}function d(m){return a>999||m===null||m===91||m===93&&!s||m===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(m):m===93?(e.exit(l),e.enter(i),e.consume(m),e.exit(i),e.exit(r),t):W(m)?(e.enter("lineEnding"),e.consume(m),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),f(m))}function f(m){return m===null||m===91||m===93||W(m)||a++>999?(e.exit("chunkString"),d(m)):(e.consume(m),s||(s=!ee(m)),m===92?g:f)}function g(m){return m===91||m===92||m===93?(e.consume(m),a++,f):f(m)}}function xp(e,t,n,r,i,l){let o;return a;function a(g){return g===34||g===39||g===40?(e.enter(r),e.enter(i),e.consume(g),e.exit(i),o=g===40?41:g,s):n(g)}function s(g){return g===o?(e.enter(i),e.consume(g),e.exit(i),e.exit(r),t):(e.enter(l),u(g))}function u(g){return g===o?(e.exit(l),s(o)):g===null?n(g):W(g)?(e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),oe(e,u,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(g))}function d(g){return g===o||g===null||W(g)?(e.exit("chunkString"),u(g)):(e.consume(g),g===92?f:d)}function f(g){return g===o||g===92?(e.consume(g),d):d(g)}}function Ir(e,t){let n;return r;function r(i){return W(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):ee(i)?oe(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const wy={name:"definition",tokenize:Cy},Sy={partial:!0,tokenize:by};function Cy(e,t,n){const r=this;let i;return l;function l(m){return e.enter("definition"),o(m)}function o(m){return yp.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(m)}function a(m){return i=Kn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),m===58?(e.enter("definitionMarker"),e.consume(m),e.exit("definitionMarker"),s):n(m)}function s(m){return Ve(m)?Ir(e,u)(m):u(m)}function u(m){return vp(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(m)}function d(m){return e.attempt(Sy,f,f)(m)}function f(m){return ee(m)?oe(e,g,"whitespace")(m):g(m)}function g(m){return m===null||W(m)?(e.exit("definition"),r.parser.defined.push(i),t(m)):n(m)}}function by(e,t,n){return r;function r(a){return Ve(a)?Ir(e,i)(a):n(a)}function i(a){return xp(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return ee(a)?oe(e,o,"whitespace")(a):o(a)}function o(a){return a===null||W(a)?t(a):n(a)}}const Ey={name:"hardBreakEscape",tokenize:jy};function jy(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return W(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const Ny={name:"headingAtx",resolve:_y,tokenize:zy};function _y(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},Ct(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function zy(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||Ve(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||W(d)?(e.exit("atxHeading"),t(d)):ee(d)?oe(e,a,"whitespace")(d):(e.enter("atxHeadingText"),u(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function u(d){return d===null||d===35||Ve(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),u)}}const Py=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],pc=["pre","script","style","textarea"],Ty={concrete:!0,name:"htmlFlow",resolveTo:Ay,tokenize:Dy},Ly={partial:!0,tokenize:Ry},Iy={partial:!0,tokenize:My};function Ay(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function Dy(e,t,n){const r=this;let i,l,o,a,s;return u;function u(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),f}function f(x){return x===33?(e.consume(x),g):x===47?(e.consume(x),l=!0,C):x===63?(e.consume(x),i=3,r.interrupt?t:v):kt(x)?(e.consume(x),o=String.fromCharCode(x),j):n(x)}function g(x){return x===45?(e.consume(x),i=2,m):x===91?(e.consume(x),i=5,a=0,S):kt(x)?(e.consume(x),i=4,r.interrupt?t:v):n(x)}function m(x){return x===45?(e.consume(x),r.interrupt?t:v):n(x)}function S(x){const te="CDATA[";return x===te.charCodeAt(a++)?(e.consume(x),a===te.length?r.interrupt?t:M:S):n(x)}function C(x){return kt(x)?(e.consume(x),o=String.fromCharCode(x),j):n(x)}function j(x){if(x===null||x===47||x===62||Ve(x)){const te=x===47,ke=o.toLowerCase();return!te&&!l&&pc.includes(ke)?(i=1,r.interrupt?t(x):M(x)):Py.includes(o.toLowerCase())?(i=6,te?(e.consume(x),p):r.interrupt?t(x):M(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?h(x):y(x))}return x===45||Xe(x)?(e.consume(x),o+=String.fromCharCode(x),j):n(x)}function p(x){return x===62?(e.consume(x),r.interrupt?t:M):n(x)}function h(x){return ee(x)?(e.consume(x),h):O(x)}function y(x){return x===47?(e.consume(x),O):x===58||x===95||kt(x)?(e.consume(x),k):ee(x)?(e.consume(x),y):O(x)}function k(x){return x===45||x===46||x===58||x===95||Xe(x)?(e.consume(x),k):b(x)}function b(x){return x===61?(e.consume(x),w):ee(x)?(e.consume(x),b):y(x)}function w(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,z):ee(x)?(e.consume(x),w):D(x)}function z(x){return x===s?(e.consume(x),s=null,H):x===null||W(x)?n(x):(e.consume(x),z)}function D(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||Ve(x)?b(x):(e.consume(x),D)}function H(x){return x===47||x===62||ee(x)?y(x):n(x)}function O(x){return x===62?(e.consume(x),_):n(x)}function _(x){return x===null||W(x)?M(x):ee(x)?(e.consume(x),_):n(x)}function M(x){return x===45&&i===2?(e.consume(x),P):x===60&&i===1?(e.consume(x),V):x===62&&i===4?(e.consume(x),L):x===63&&i===3?(e.consume(x),v):x===93&&i===5?(e.consume(x),E):W(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(Ly,B,Y)(x)):x===null||W(x)?(e.exit("htmlFlowData"),Y(x)):(e.consume(x),M)}function Y(x){return e.check(Iy,G,B)(x)}function G(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),$}function $(x){return x===null||W(x)?Y(x):(e.enter("htmlFlowData"),M(x))}function P(x){return x===45?(e.consume(x),v):M(x)}function V(x){return x===47?(e.consume(x),o="",T):M(x)}function T(x){if(x===62){const te=o.toLowerCase();return pc.includes(te)?(e.consume(x),L):M(x)}return kt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),T):M(x)}function E(x){return x===93?(e.consume(x),v):M(x)}function v(x){return x===62?(e.consume(x),L):x===45&&i===2?(e.consume(x),v):M(x)}function L(x){return x===null||W(x)?(e.exit("htmlFlowData"),B(x)):(e.consume(x),L)}function B(x){return e.exit("htmlFlow"),t(x)}}function My(e,t,n){const r=this;return i;function i(o){return W(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Ry(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Ll,t,n)}}const Oy={name:"htmlText",tokenize:Fy};function Fy(e,t,n){const r=this;let i,l,o;return a;function a(v){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(v),s}function s(v){return v===33?(e.consume(v),u):v===47?(e.consume(v),b):v===63?(e.consume(v),y):kt(v)?(e.consume(v),D):n(v)}function u(v){return v===45?(e.consume(v),d):v===91?(e.consume(v),l=0,S):kt(v)?(e.consume(v),h):n(v)}function d(v){return v===45?(e.consume(v),m):n(v)}function f(v){return v===null?n(v):v===45?(e.consume(v),g):W(v)?(o=f,V(v)):(e.consume(v),f)}function g(v){return v===45?(e.consume(v),m):f(v)}function m(v){return v===62?P(v):v===45?g(v):f(v)}function S(v){const L="CDATA[";return v===L.charCodeAt(l++)?(e.consume(v),l===L.length?C:S):n(v)}function C(v){return v===null?n(v):v===93?(e.consume(v),j):W(v)?(o=C,V(v)):(e.consume(v),C)}function j(v){return v===93?(e.consume(v),p):C(v)}function p(v){return v===62?P(v):v===93?(e.consume(v),p):C(v)}function h(v){return v===null||v===62?P(v):W(v)?(o=h,V(v)):(e.consume(v),h)}function y(v){return v===null?n(v):v===63?(e.consume(v),k):W(v)?(o=y,V(v)):(e.consume(v),y)}function k(v){return v===62?P(v):y(v)}function b(v){return kt(v)?(e.consume(v),w):n(v)}function w(v){return v===45||Xe(v)?(e.consume(v),w):z(v)}function z(v){return W(v)?(o=z,V(v)):ee(v)?(e.consume(v),z):P(v)}function D(v){return v===45||Xe(v)?(e.consume(v),D):v===47||v===62||Ve(v)?H(v):n(v)}function H(v){return v===47?(e.consume(v),P):v===58||v===95||kt(v)?(e.consume(v),O):W(v)?(o=H,V(v)):ee(v)?(e.consume(v),H):P(v)}function O(v){return v===45||v===46||v===58||v===95||Xe(v)?(e.consume(v),O):_(v)}function _(v){return v===61?(e.consume(v),M):W(v)?(o=_,V(v)):ee(v)?(e.consume(v),_):H(v)}function M(v){return v===null||v===60||v===61||v===62||v===96?n(v):v===34||v===39?(e.consume(v),i=v,Y):W(v)?(o=M,V(v)):ee(v)?(e.consume(v),M):(e.consume(v),G)}function Y(v){return v===i?(e.consume(v),i=void 0,$):v===null?n(v):W(v)?(o=Y,V(v)):(e.consume(v),Y)}function G(v){return v===null||v===34||v===39||v===60||v===61||v===96?n(v):v===47||v===62||Ve(v)?H(v):(e.consume(v),G)}function $(v){return v===47||v===62||Ve(v)?H(v):n(v)}function P(v){return v===62?(e.consume(v),e.exit("htmlTextData"),e.exit("htmlText"),t):n(v)}function V(v){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(v),e.exit("lineEnding"),T}function T(v){return ee(v)?oe(e,E,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(v):E(v)}function E(v){return e.enter("htmlTextData"),o(v)}}const _s={name:"labelEnd",resolveAll:Vy,resolveTo:$y,tokenize:Wy},By={tokenize:Qy},Uy={tokenize:qy},Hy={tokenize:Ky};function Vy(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&Ct(e,0,e.length,n),e}function $y(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},u={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",u,t]],a=rt(a,e.slice(l+1,l+r+3)),a=rt(a,[["enter",d,t]]),a=rt(a,Ns(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=rt(a,[["exit",d,t],e[o-2],e[o-1],["exit",u,t]]),a=rt(a,e.slice(o+1)),a=rt(a,[["exit",s,t]]),Ct(e,l,e.length,a),e}function Wy(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(g){return l?l._inactive?f(g):(o=r.parser.defined.includes(Kn(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(g),e.exit("labelMarker"),e.exit("labelEnd"),s):n(g)}function s(g){return g===40?e.attempt(By,d,o?d:f)(g):g===91?e.attempt(Uy,d,o?u:f)(g):o?d(g):f(g)}function u(g){return e.attempt(Hy,d,f)(g)}function d(g){return t(g)}function f(g){return l._balanced=!0,n(g)}}function Qy(e,t,n){return r;function r(f){return e.enter("resource"),e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),i}function i(f){return Ve(f)?Ir(e,l)(f):l(f)}function l(f){return f===41?d(f):vp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(f)}function o(f){return Ve(f)?Ir(e,s)(f):d(f)}function a(f){return n(f)}function s(f){return f===34||f===39||f===40?xp(e,u,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(f):d(f)}function u(f){return Ve(f)?Ir(e,d)(f):d(f)}function d(f){return f===41?(e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),e.exit("resource"),t):n(f)}}function qy(e,t,n){const r=this;return i;function i(a){return yp.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(Kn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Ky(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Yy={name:"labelStartImage",resolveAll:_s.resolveAll,tokenize:Xy};function Xy(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Gy={name:"labelStartLink",resolveAll:_s.resolveAll,tokenize:Jy};function Jy(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const co={name:"lineEnding",tokenize:Zy};function Zy(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),oe(e,t,"linePrefix")}}const Bi={name:"thematicBreak",tokenize:ex};function ex(e,t,n){let r=0,i;return l;function l(u){return e.enter("thematicBreak"),o(u)}function o(u){return i=u,a(u)}function a(u){return u===i?(e.enter("thematicBreakSequence"),s(u)):r>=3&&(u===null||W(u))?(e.exit("thematicBreak"),t(u)):n(u)}function s(u){return u===i?(e.consume(u),r++,s):(e.exit("thematicBreakSequence"),ee(u)?oe(e,a,"whitespace")(u):a(u))}}const Re={continuation:{tokenize:ix},exit:ox,name:"list",tokenize:rx},tx={partial:!0,tokenize:ax},nx={partial:!0,tokenize:lx};function rx(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(m){const S=r.containerState.type||(m===42||m===43||m===45?"listUnordered":"listOrdered");if(S==="listUnordered"?!r.containerState.marker||m===r.containerState.marker:xa(m)){if(r.containerState.type||(r.containerState.type=S,e.enter(S,{_container:!0})),S==="listUnordered")return e.enter("listItemPrefix"),m===42||m===45?e.check(Bi,n,u)(m):u(m);if(!r.interrupt||m===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(m)}return n(m)}function s(m){return xa(m)&&++o<10?(e.consume(m),s):(!r.interrupt||o<2)&&(r.containerState.marker?m===r.containerState.marker:m===41||m===46)?(e.exit("listItemValue"),u(m)):n(m)}function u(m){return e.enter("listItemMarker"),e.consume(m),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||m,e.check(Ll,r.interrupt?n:d,e.attempt(tx,g,f))}function d(m){return r.containerState.initialBlankLine=!0,l++,g(m)}function f(m){return ee(m)?(e.enter("listItemPrefixWhitespace"),e.consume(m),e.exit("listItemPrefixWhitespace"),g):n(m)}function g(m){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(m)}}function ix(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Ll,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,oe(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!ee(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(nx,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,oe(e,e.attempt(Re,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function lx(e,t,n){const r=this;return oe(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function ox(e){e.exit(this.containerState.type)}function ax(e,t,n){const r=this;return oe(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!ee(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const hc={name:"setextUnderline",resolveTo:sx,tokenize:ux};function sx(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function ux(e,t,n){const r=this;let i;return l;function l(u){let d=r.events.length,f;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){f=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||f)?(e.enter("setextHeadingLine"),i=u,o(u)):n(u)}function o(u){return e.enter("setextHeadingLineSequence"),a(u)}function a(u){return u===i?(e.consume(u),a):(e.exit("setextHeadingLineSequence"),ee(u)?oe(e,s,"lineSuffix")(u):s(u))}function s(u){return u===null||W(u)?(e.exit("setextHeadingLine"),t(u)):n(u)}}const cx={tokenize:dx};function dx(e){const t=this,n=e.attempt(Ll,r,e.attempt(this.parser.constructs.flowInitial,i,oe(e,e.attempt(this.parser.constructs.flow,i,e.attempt(gy,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const fx={resolveAll:wp()},px=kp("string"),hx=kp("text");function kp(e){return{resolveAll:wp(e==="text"?mx:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return u(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return u(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function u(d){if(d===null)return!0;const f=i[d];let g=-1;if(f)for(;++g<f.length;){const m=f[g];if(!m.previous||m.previous.call(r,r.previous))return!0}return!1}}}function wp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function mx(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const u=i[l];if(typeof u=="string"){for(o=u.length;u.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(u===-2)s=!0,a++;else if(u!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const u={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...u.start},r.start.offset===r.end.offset?Object.assign(r,u):(e.splice(n,0,["enter",u,t],["exit",u,t]),n+=2)}n++}return e}const gx={42:Re,43:Re,45:Re,48:Re,49:Re,50:Re,51:Re,52:Re,53:Re,54:Re,55:Re,56:Re,57:Re,62:pp},vx={91:wy},yx={[-2]:uo,[-1]:uo,32:uo},xx={35:Ny,42:Bi,45:[hc,Bi],60:Ty,61:hc,95:Bi,96:fc,126:fc},kx={38:mp,92:hp},wx={[-5]:co,[-4]:co,[-3]:co,33:Yy,38:mp,42:ka,60:[Gv,Oy],91:Gy,92:[Ey,hp],93:_s,95:ka,96:cy},Sx={null:[ka,fx]},Cx={null:[42,95]},bx={null:[]},Ex=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:Cx,contentInitial:vx,disable:bx,document:gx,flow:xx,flowInitial:yx,insideSpan:Sx,string:kx,text:wx},Symbol.toStringTag,{value:"Module"}));function jx(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:z(b),check:z(w),consume:h,enter:y,exit:k,interrupt:z(w,{interrupt:!0})},u={code:null,containerState:{},defineSkip:C,events:[],now:S,parser:e,previous:null,sliceSerialize:g,sliceStream:m,write:f};let d=t.tokenize.call(u,s);return t.resolveAll&&l.push(t),u;function f(_){return o=rt(o,_),j(),o[o.length-1]!==null?[]:(D(t,0),u.events=Ns(l,u.events,u),u.events)}function g(_,M){return _x(m(_),M)}function m(_){return Nx(o,_)}function S(){const{_bufferIndex:_,_index:M,line:Y,column:G,offset:$}=r;return{_bufferIndex:_,_index:M,line:Y,column:G,offset:$}}function C(_){i[_.line]=_.column,O()}function j(){let _;for(;r._index<o.length;){const M=o[r._index];if(typeof M=="string")for(_=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===_&&r._bufferIndex<M.length;)p(M.charCodeAt(r._bufferIndex));else p(M)}}function p(_){d=d(_)}function h(_){W(_)?(r.line++,r.column=1,r.offset+=_===-3?2:1,O()):_!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),u.previous=_}function y(_,M){const Y=M||{};return Y.type=_,Y.start=S(),u.events.push(["enter",Y,u]),a.push(Y),Y}function k(_){const M=a.pop();return M.end=S(),u.events.push(["exit",M,u]),M}function b(_,M){D(_,M.from)}function w(_,M){M.restore()}function z(_,M){return Y;function Y(G,$,P){let V,T,E,v;return Array.isArray(G)?B(G):"tokenize"in G?B([G]):L(G);function L(Q){return ve;function ve(je){const an=je!==null&&Q[je],En=je!==null&&Q.null,oi=[...Array.isArray(an)?an:an?[an]:[],...Array.isArray(En)?En:En?[En]:[]];return B(oi)(je)}}function B(Q){return V=Q,T=0,Q.length===0?P:x(Q[T])}function x(Q){return ve;function ve(je){return v=H(),E=Q,Q.partial||(u.currentConstruct=Q),Q.name&&u.parser.constructs.disable.null.includes(Q.name)?ke():Q.tokenize.call(M?Object.assign(Object.create(u),M):u,s,te,ke)(je)}}function te(Q){return _(E,v),$}function ke(Q){return v.restore(),++T<V.length?x(V[T]):P}}}function D(_,M){_.resolveAll&&!l.includes(_)&&l.push(_),_.resolve&&Ct(u.events,M,u.events.length-M,_.resolve(u.events.slice(M),u)),_.resolveTo&&(u.events=_.resolveTo(u.events,u))}function H(){const _=S(),M=u.previous,Y=u.currentConstruct,G=u.events.length,$=Array.from(a);return{from:G,restore:P};function P(){r=_,u.previous=M,u.currentConstruct=Y,u.events.length=G,a=$,O()}}function O(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function Nx(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function _x(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function zx(e){const r={constructs:Mv([Ex,...(e||{}).extensions||[]]),content:i($v),defined:[],document:i(Qv),flow:i(cx),lazy:{},string:i(px),text:i(hx)};return r;function i(l){return o;function o(a){return jx(r,l,a)}}}function Px(e){for(;!gp(e););return e}const mc=/[\0\t\n\r]/g;function Tx(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let u,d,f,g,m;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),f=0,t="",n&&(l.charCodeAt(0)===65279&&f++,n=void 0);f<l.length;){if(mc.lastIndex=f,u=mc.exec(l),g=u&&u.index!==void 0?u.index:l.length,m=l.charCodeAt(g),!u){t=l.slice(f);break}if(m===10&&f===g&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),f<g&&(s.push(l.slice(f,g)),e+=g-f),m){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}f=g+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const Lx=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function Ix(e){return e.replace(Lx,Ax)}function Ax(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return fp(n.slice(l?2:1),l?16:10)}return js(n)||e}const Sp={}.hasOwnProperty;function Dx(e,t,n){return typeof t!="string"&&(n=t,t=void 0),Mx(n)(Px(zx(n).document().write(Tx()(e,t,!0))))}function Mx(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Rs),autolinkProtocol:H,autolinkEmail:H,atxHeading:l(As),blockQuote:l(En),characterEscape:H,characterReference:H,codeFenced:l(oi),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(oi,o),codeText:l(Ap,o),codeTextData:H,data:H,codeFlowValue:H,definition:l(Dp),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Mp),hardBreakEscape:l(Ds),hardBreakTrailing:l(Ds),htmlFlow:l(Ms,o),htmlFlowData:H,htmlText:l(Ms,o),htmlTextData:H,image:l(Rp),label:o,link:l(Rs),listItem:l(Op),listItemValue:g,listOrdered:l(Os,f),listUnordered:l(Os),paragraph:l(Fp),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(As),strong:l(Bp),thematicBreak:l(Hp)},exit:{atxHeading:s(),atxHeadingSequence:b,autolink:s(),autolinkEmail:an,autolinkProtocol:je,blockQuote:s(),characterEscapeValue:O,characterReferenceMarkerHexadecimal:ke,characterReferenceMarkerNumeric:ke,characterReferenceValue:Q,characterReference:ve,codeFenced:s(j),codeFencedFence:C,codeFencedFenceInfo:m,codeFencedFenceMeta:S,codeFlowValue:O,codeIndented:s(p),codeText:s($),codeTextData:O,data:O,definition:s(),definitionDestinationString:k,definitionLabelString:h,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(M),hardBreakTrailing:s(M),htmlFlow:s(Y),htmlFlowData:O,htmlText:s(G),htmlTextData:O,image:s(V),label:E,labelText:T,lineEnding:_,link:s(P),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:te,resourceDestinationString:v,resourceTitleString:L,resource:B,setextHeading:s(D),setextHeadingLineSequence:z,setextHeadingText:w,strong:s(),thematicBreak:s()}};Cp(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(N){let R={type:"root",children:[]};const q={stack:[R],tokenStack:[],config:t,enter:a,exit:u,buffer:o,resume:d,data:n},J=[];let re=-1;for(;++re<N.length;)if(N[re][1].type==="listOrdered"||N[re][1].type==="listUnordered")if(N[re][0]==="enter")J.push(re);else{const st=J.pop();re=i(N,st,re)}for(re=-1;++re<N.length;){const st=t[N[re][0]];Sp.call(st,N[re][1].type)&&st[N[re][1].type].call(Object.assign({sliceSerialize:N[re][2].sliceSerialize},q),N[re][1])}if(q.tokenStack.length>0){const st=q.tokenStack[q.tokenStack.length-1];(st[1]||gc).call(q,void 0,st[0])}for(R.position={start:Ot(N.length>0?N[0][1].start:{line:1,column:1,offset:0}),end:Ot(N.length>0?N[N.length-2][1].end:{line:1,column:1,offset:0})},re=-1;++re<t.transforms.length;)R=t.transforms[re](R)||R;return R}function i(N,R,q){let J=R-1,re=-1,st=!1,sn,bt,ar,sr;for(;++J<=q;){const We=N[J];switch(We[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{We[0]==="enter"?re++:re--,sr=void 0;break}case"lineEndingBlank":{We[0]==="enter"&&(sn&&!sr&&!re&&!ar&&(ar=J),sr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:sr=void 0}if(!re&&We[0]==="enter"&&We[1].type==="listItemPrefix"||re===-1&&We[0]==="exit"&&(We[1].type==="listUnordered"||We[1].type==="listOrdered")){if(sn){let jn=J;for(bt=void 0;jn--;){const Et=N[jn];if(Et[1].type==="lineEnding"||Et[1].type==="lineEndingBlank"){if(Et[0]==="exit")continue;bt&&(N[bt][1].type="lineEndingBlank",st=!0),Et[1].type="lineEnding",bt=jn}else if(!(Et[1].type==="linePrefix"||Et[1].type==="blockQuotePrefix"||Et[1].type==="blockQuotePrefixWhitespace"||Et[1].type==="blockQuoteMarker"||Et[1].type==="listItemIndent"))break}ar&&(!bt||ar<bt)&&(sn._spread=!0),sn.end=Object.assign({},bt?N[bt][1].start:We[1].end),N.splice(bt||J,0,["exit",sn,We[2]]),J++,q++}if(We[1].type==="listItemPrefix"){const jn={type:"listItem",_spread:!1,start:Object.assign({},We[1].start),end:void 0};sn=jn,N.splice(J,0,["enter",jn,We[2]]),J++,q++,ar=void 0,sr=!0}}}return N[R][1]._spread=st,q}function l(N,R){return q;function q(J){a.call(this,N(J),J),R&&R.call(this,J)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(N,R,q){this.stack[this.stack.length-1].children.push(N),this.stack.push(N),this.tokenStack.push([R,q||void 0]),N.position={start:Ot(R.start),end:void 0}}function s(N){return R;function R(q){N&&N.call(this,q),u.call(this,q)}}function u(N,R){const q=this.stack.pop(),J=this.tokenStack.pop();if(J)J[0].type!==N.type&&(R?R.call(this,N,J[0]):(J[1]||gc).call(this,N,J[0]));else throw new Error("Cannot close `"+N.type+"` ("+Lr({start:N.start,end:N.end})+"): it’s not open");q.position.end=Ot(N.end)}function d(){return Av(this.stack.pop())}function f(){this.data.expectingFirstListItemValue=!0}function g(N){if(this.data.expectingFirstListItemValue){const R=this.stack[this.stack.length-2];R.start=Number.parseInt(this.sliceSerialize(N),10),this.data.expectingFirstListItemValue=void 0}}function m(){const N=this.resume(),R=this.stack[this.stack.length-1];R.lang=N}function S(){const N=this.resume(),R=this.stack[this.stack.length-1];R.meta=N}function C(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function j(){const N=this.resume(),R=this.stack[this.stack.length-1];R.value=N.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function p(){const N=this.resume(),R=this.stack[this.stack.length-1];R.value=N.replace(/(\r?\n|\r)$/g,"")}function h(N){const R=this.resume(),q=this.stack[this.stack.length-1];q.label=R,q.identifier=Kn(this.sliceSerialize(N)).toLowerCase()}function y(){const N=this.resume(),R=this.stack[this.stack.length-1];R.title=N}function k(){const N=this.resume(),R=this.stack[this.stack.length-1];R.url=N}function b(N){const R=this.stack[this.stack.length-1];if(!R.depth){const q=this.sliceSerialize(N).length;R.depth=q}}function w(){this.data.setextHeadingSlurpLineEnding=!0}function z(N){const R=this.stack[this.stack.length-1];R.depth=this.sliceSerialize(N).codePointAt(0)===61?1:2}function D(){this.data.setextHeadingSlurpLineEnding=void 0}function H(N){const q=this.stack[this.stack.length-1].children;let J=q[q.length-1];(!J||J.type!=="text")&&(J=Up(),J.position={start:Ot(N.start),end:void 0},q.push(J)),this.stack.push(J)}function O(N){const R=this.stack.pop();R.value+=this.sliceSerialize(N),R.position.end=Ot(N.end)}function _(N){const R=this.stack[this.stack.length-1];if(this.data.atHardBreak){const q=R.children[R.children.length-1];q.position.end=Ot(N.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(R.type)&&(H.call(this,N),O.call(this,N))}function M(){this.data.atHardBreak=!0}function Y(){const N=this.resume(),R=this.stack[this.stack.length-1];R.value=N}function G(){const N=this.resume(),R=this.stack[this.stack.length-1];R.value=N}function $(){const N=this.resume(),R=this.stack[this.stack.length-1];R.value=N}function P(){const N=this.stack[this.stack.length-1];if(this.data.inReference){const R=this.data.referenceType||"shortcut";N.type+="Reference",N.referenceType=R,delete N.url,delete N.title}else delete N.identifier,delete N.label;this.data.referenceType=void 0}function V(){const N=this.stack[this.stack.length-1];if(this.data.inReference){const R=this.data.referenceType||"shortcut";N.type+="Reference",N.referenceType=R,delete N.url,delete N.title}else delete N.identifier,delete N.label;this.data.referenceType=void 0}function T(N){const R=this.sliceSerialize(N),q=this.stack[this.stack.length-2];q.label=Ix(R),q.identifier=Kn(R).toLowerCase()}function E(){const N=this.stack[this.stack.length-1],R=this.resume(),q=this.stack[this.stack.length-1];if(this.data.inReference=!0,q.type==="link"){const J=N.children;q.children=J}else q.alt=R}function v(){const N=this.resume(),R=this.stack[this.stack.length-1];R.url=N}function L(){const N=this.resume(),R=this.stack[this.stack.length-1];R.title=N}function B(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function te(N){const R=this.resume(),q=this.stack[this.stack.length-1];q.label=R,q.identifier=Kn(this.sliceSerialize(N)).toLowerCase(),this.data.referenceType="full"}function ke(N){this.data.characterReferenceType=N.type}function Q(N){const R=this.sliceSerialize(N),q=this.data.characterReferenceType;let J;q?(J=fp(R,q==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):J=js(R);const re=this.stack[this.stack.length-1];re.value+=J}function ve(N){const R=this.stack.pop();R.position.end=Ot(N.end)}function je(N){O.call(this,N);const R=this.stack[this.stack.length-1];R.url=this.sliceSerialize(N)}function an(N){O.call(this,N);const R=this.stack[this.stack.length-1];R.url="mailto:"+this.sliceSerialize(N)}function En(){return{type:"blockquote",children:[]}}function oi(){return{type:"code",lang:null,meta:null,value:""}}function Ap(){return{type:"inlineCode",value:""}}function Dp(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Mp(){return{type:"emphasis",children:[]}}function As(){return{type:"heading",depth:0,children:[]}}function Ds(){return{type:"break"}}function Ms(){return{type:"html",value:""}}function Rp(){return{type:"image",title:null,url:"",alt:null}}function Rs(){return{type:"link",title:null,url:"",children:[]}}function Os(N){return{type:"list",ordered:N.type==="listOrdered",start:null,spread:N._spread,children:[]}}function Op(N){return{type:"listItem",spread:N._spread,checked:null,children:[]}}function Fp(){return{type:"paragraph",children:[]}}function Bp(){return{type:"strong",children:[]}}function Up(){return{type:"text",value:""}}function Hp(){return{type:"thematicBreak"}}}function Ot(e){return{line:e.line,column:e.column,offset:e.offset}}function Cp(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Cp(e,r):Rx(e,r)}}function Rx(e,t){let n;for(n in t)if(Sp.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function gc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Lr({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Lr({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Lr({start:t.start,end:t.end})+") is still open")}function Ox(e){const t=this;t.parser=n;function n(r){return Dx(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function Fx(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function Bx(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function Ux(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function Hx(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Vx(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function $x(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=or(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const u={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,u),e.applyData(t,u)}function Wx(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Qx(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function bp(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function qx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return bp(e,t);const i={src:or(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function Kx(e,t){const n={src:or(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function Yx(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function Xx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return bp(e,t);const i={href:or(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function Gx(e,t){const n={href:or(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function Jx(e,t,n){const r=e.all(t),i=n?Zx(n):Ep(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let f;d&&d.type==="element"&&d.tagName==="p"?f=d:(f={type:"element",tagName:"p",properties:{},children:[]},r.unshift(f)),f.children.length>0&&f.children.unshift({type:"text",value:" "}),f.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const u={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,u),e.applyData(t,u)}function Zx(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Ep(n[r])}return t}function Ep(e){const t=e.spread;return t??e.children.length>1}function e1(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function t1(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function n1(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function r1(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function i1(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=Ss(t.children[1]),s=lp(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function l1(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const u=[];for(;++s<a;){const f=t.children[s],g={},m=o?o[s]:void 0;m&&(g.align=m);let S={type:"element",tagName:l,properties:g,children:[]};f&&(S.children=e.all(f),e.patch(f,S),S=e.applyData(f,S)),u.push(S)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(u,!0)};return e.patch(t,d),e.applyData(t,d)}function o1(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const vc=9,yc=32;function a1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(xc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(xc(t.slice(i),i>0,!1)),l.join("")}function xc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===vc||l===yc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===vc||l===yc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function s1(e,t){const n={type:"text",value:a1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function u1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const c1={blockquote:Fx,break:Bx,code:Ux,delete:Hx,emphasis:Vx,footnoteReference:$x,heading:Wx,html:Qx,imageReference:qx,image:Kx,inlineCode:Yx,linkReference:Xx,link:Gx,listItem:Jx,list:e1,paragraph:t1,root:n1,strong:r1,table:i1,tableCell:o1,tableRow:l1,text:s1,thematicBreak:u1,toml:Ei,yaml:Ei,definition:Ei,footnoteDefinition:Ei};function Ei(){}const jp=-1,Il=0,Ar=1,pl=2,zs=3,Ps=4,Ts=5,Ls=6,Np=7,_p=8,kc=typeof self=="object"?self:globalThis,d1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Il:case jp:return n(o,i);case Ar:{const a=n([],i);for(const s of o)a.push(r(s));return a}case pl:{const a=n({},i);for(const[s,u]of o)a[r(s)]=r(u);return a}case zs:return n(new Date(o),i);case Ps:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Ts:{const a=n(new Map,i);for(const[s,u]of o)a.set(r(s),r(u));return a}case Ls:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Np:{const{name:a,message:s}=o;return n(new kc[a](s),i)}case _p:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new kc[l](o),i)};return r},wc=e=>d1(new Map,e)(0),_n="",{toString:f1}={},{keys:p1}=Object,yr=e=>{const t=typeof e;if(t!=="object"||!e)return[Il,t];const n=f1.call(e).slice(8,-1);switch(n){case"Array":return[Ar,_n];case"Object":return[pl,_n];case"Date":return[zs,_n];case"RegExp":return[Ps,_n];case"Map":return[Ts,_n];case"Set":return[Ls,_n];case"DataView":return[Ar,n]}return n.includes("Array")?[Ar,n]:n.includes("Error")?[Np,n]:[pl,n]},ji=([e,t])=>e===Il&&(t==="function"||t==="symbol"),h1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=yr(o);switch(a){case Il:{let d=o;switch(s){case"bigint":a=_p,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([jp],o)}return i([a,d],o)}case Ar:{if(s){let g=o;return s==="DataView"?g=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(g=new Uint8Array(o)),i([s,[...g]],o)}const d=[],f=i([a,d],o);for(const g of o)d.push(l(g));return f}case pl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],f=i([a,d],o);for(const g of p1(o))(e||!ji(yr(o[g])))&&d.push([l(g),l(o[g])]);return f}case zs:return i([a,o.toISOString()],o);case Ps:{const{source:d,flags:f}=o;return i([a,{source:d,flags:f}],o)}case Ts:{const d=[],f=i([a,d],o);for(const[g,m]of o)(e||!(ji(yr(g))||ji(yr(m))))&&d.push([l(g),l(m)]);return f}case Ls:{const d=[],f=i([a,d],o);for(const g of o)(e||!ji(yr(g)))&&d.push(l(g));return f}}const{message:u}=o;return i([a,{name:s,message:u}],o)};return l},Sc=(e,{json:t,lossy:n}={})=>{const r=[];return h1(!(t||n),!!t,new Map,r)(e),r},hl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?wc(Sc(e,t)):structuredClone(e):(e,t)=>wc(Sc(e,t));function m1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function g1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function v1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||m1,r=e.options.footnoteBackLabel||g1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const u=e.footnoteById.get(e.footnoteOrder[s]);if(!u)continue;const d=e.all(u),f=String(u.identifier).toUpperCase(),g=or(f.toLowerCase());let m=0;const S=[],C=e.footnoteCounts.get(f);for(;C!==void 0&&++m<=C;){S.length>0&&S.push({type:"text",value:" "});let h=typeof n=="string"?n:n(s,m);typeof h=="string"&&(h={type:"text",value:h}),S.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+g+(m>1?"-"+m:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,m),className:["data-footnote-backref"]},children:Array.isArray(h)?h:[h]})}const j=d[d.length-1];if(j&&j.type==="element"&&j.tagName==="p"){const h=j.children[j.children.length-1];h&&h.type==="text"?h.value+=" ":j.children.push({type:"text",value:" "}),j.children.push(...S)}else d.push(...S);const p={type:"element",tagName:"li",properties:{id:t+"fn-"+g},children:e.wrap(d,!0)};e.patch(u,p),a.push(p)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...hl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const zp=function(e){if(e==null)return w1;if(typeof e=="function")return Al(e);if(typeof e=="object")return Array.isArray(e)?y1(e):x1(e);if(typeof e=="string")return k1(e);throw new Error("Expected function, string, or object as test")};function y1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=zp(e[n]);return Al(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function x1(e){const t=e;return Al(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function k1(e){return Al(t);function t(n){return n&&n.type===e}}function Al(e){return t;function t(n,r,i){return!!(S1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function w1(){return!0}function S1(e){return e!==null&&typeof e=="object"&&"type"in e}const Pp=[],C1=!0,Cc=!1,b1="skip";function E1(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=zp(i),o=r?-1:1;a(e,void 0,[])();function a(s,u,d){const f=s&&typeof s=="object"?s:{};if(typeof f.type=="string"){const m=typeof f.tagName=="string"?f.tagName:typeof f.name=="string"?f.name:void 0;Object.defineProperty(g,"name",{value:"node ("+(s.type+(m?"<"+m+">":""))+")"})}return g;function g(){let m=Pp,S,C,j;if((!t||l(s,u,d[d.length-1]||void 0))&&(m=j1(n(s,d)),m[0]===Cc))return m;if("children"in s&&s.children){const p=s;if(p.children&&m[0]!==b1)for(C=(r?p.children.length:-1)+o,j=d.concat(p);C>-1&&C<p.children.length;){const h=p.children[C];if(S=a(h,C,j)(),S[0]===Cc)return S;C=typeof S[1]=="number"?S[1]:C+o}}return m}}}function j1(e){return Array.isArray(e)?e:typeof e=="number"?[C1,e]:e==null?Pp:[e]}function Tp(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),E1(e,l,a,i);function a(s,u){const d=u[u.length-1],f=d?d.children.indexOf(s):void 0;return o(s,f,d)}}const wa={}.hasOwnProperty,N1={};function _1(e,t){const n=t||N1,r=new Map,i=new Map,l=new Map,o={...c1,...n.handlers},a={all:u,applyData:P1,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:z1,wrap:L1};return Tp(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const f=d.type==="definition"?r:i,g=String(d.identifier).toUpperCase();f.has(g)||f.set(g,d)}}),a;function s(d,f){const g=d.type,m=a.handlers[g];if(wa.call(a.handlers,g)&&m)return m(a,d,f);if(a.options.passThrough&&a.options.passThrough.includes(g)){if("children"in d){const{children:C,...j}=d,p=hl(j);return p.children=a.all(d),p}return hl(d)}return(a.options.unknownHandler||T1)(a,d,f)}function u(d){const f=[];if("children"in d){const g=d.children;let m=-1;for(;++m<g.length;){const S=a.one(g[m],d);if(S){if(m&&g[m-1].type==="break"&&(!Array.isArray(S)&&S.type==="text"&&(S.value=bc(S.value)),!Array.isArray(S)&&S.type==="element")){const C=S.children[0];C&&C.type==="text"&&(C.value=bc(C.value))}Array.isArray(S)?f.push(...S):f.push(S)}}}return f}}function z1(e,t){e.position&&(t.position=fv(e))}function P1(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,hl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function T1(e,t){const n=t.data||{},r="value"in t&&!(wa.call(n,"hProperties")||wa.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function L1(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function bc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Ec(e,t){const n=_1(e,t),r=n.one(e,void 0),i=v1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function I1(e,t){return e&&"run"in e?async function(n,r){const i=Ec(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Ec(n,{file:r,...e||t})}}function jc(e){if(e)throw e}var Ui=Object.prototype.hasOwnProperty,Lp=Object.prototype.toString,Nc=Object.defineProperty,_c=Object.getOwnPropertyDescriptor,zc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Lp.call(t)==="[object Array]"},Pc=function(t){if(!t||Lp.call(t)!=="[object Object]")return!1;var n=Ui.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Ui.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Ui.call(t,i)},Tc=function(t,n){Nc&&n.name==="__proto__"?Nc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Lc=function(t,n){if(n==="__proto__")if(Ui.call(t,n)){if(_c)return _c(t,n).value}else return;return t[n]},A1=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,u=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<u;++s)if(t=arguments[s],t!=null)for(n in t)r=Lc(a,n),i=Lc(t,n),a!==i&&(d&&i&&(Pc(i)||(l=zc(i)))?(l?(l=!1,o=r&&zc(r)?r:[]):o=r&&Pc(r)?r:{},Tc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Tc(a,{name:n,newValue:i}));return a};const fo=ba(A1);function Sa(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function D1(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...u){const d=e[++l];let f=-1;if(s){o(s);return}for(;++f<i.length;)(u[f]===null||u[f]===void 0)&&(u[f]=i[f]);i=u,d?M1(d,a)(...u):o(null,...u)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function M1(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(u){const d=u;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const yt={basename:R1,dirname:O1,extname:F1,join:B1,sep:"/"};function R1(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');li(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function O1(e){if(li(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function F1(e){li(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function B1(...e){let t=-1,n;for(;++t<e.length;)li(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":U1(n)}function U1(e){li(e);const t=e.codePointAt(0)===47;let n=H1(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function H1(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function li(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const V1={cwd:$1};function $1(){return"/"}function Ca(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function W1(e){if(typeof e=="string")e=new URL(e);else if(!Ca(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return Q1(e)}function Q1(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const po=["history","path","basename","stem","extname","dirname"];class Ip{constructor(t){let n;t?Ca(t)?n={path:t}:typeof t=="string"||q1(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":V1.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<po.length;){const l=po[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)po.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?yt.basename(this.path):void 0}set basename(t){mo(t,"basename"),ho(t,"basename"),this.path=yt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?yt.dirname(this.path):void 0}set dirname(t){Ic(this.basename,"dirname"),this.path=yt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?yt.extname(this.path):void 0}set extname(t){if(ho(t,"extname"),Ic(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=yt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){Ca(t)&&(t=W1(t)),mo(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?yt.basename(this.path,this.extname):void 0}set stem(t){mo(t,"stem"),ho(t,"stem"),this.path=yt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new Te(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function ho(e,t){if(e&&e.includes(yt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+yt.sep+"`")}function mo(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Ic(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function q1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const K1=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},Y1={}.hasOwnProperty;class Is extends K1{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=D1()}copy(){const t=new Is;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(fo(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(yo("data",this.frozen),this.namespace[t]=n,this):Y1.call(this.namespace,t)&&this.namespace[t]||void 0:t?(yo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Ni(t),r=this.parser||this.Parser;return go("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),go("process",this.parser||this.Parser),vo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Ni(t),s=r.parse(a);r.run(s,a,function(d,f,g){if(d||!f||!g)return u(d);const m=f,S=r.stringify(m,g);J1(S)?g.value=S:g.result=S,u(d,g)});function u(d,f){d||!f?o(d):l?l(f):n(void 0,f)}}}processSync(t){let n=!1,r;return this.freeze(),go("processSync",this.parser||this.Parser),vo("processSync",this.compiler||this.Compiler),this.process(t,i),Dc("processSync","process",n),r;function i(l,o){n=!0,jc(l),r=o}}run(t,n,r){Ac(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Ni(n);i.run(t,s,u);function u(d,f,g){const m=f||t;d?a(d):o?o(m):r(void 0,m,g)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Dc("runSync","run",r),i;function l(o,a){jc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Ni(n),i=this.compiler||this.Compiler;return vo("stringify",i),Ac(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(yo("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(u){if(typeof u=="function")s(u,[]);else if(typeof u=="object")if(Array.isArray(u)){const[d,...f]=u;s(d,f)}else o(u);else throw new TypeError("Expected usable value, not `"+u+"`")}function o(u){if(!("plugins"in u)&&!("settings"in u))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(u.plugins),u.settings&&(i.settings=fo(!0,i.settings,u.settings))}function a(u){let d=-1;if(u!=null)if(Array.isArray(u))for(;++d<u.length;){const f=u[d];l(f)}else throw new TypeError("Expected a list of plugins, not `"+u+"`")}function s(u,d){let f=-1,g=-1;for(;++f<r.length;)if(r[f][0]===u){g=f;break}if(g===-1)r.push([u,...d]);else if(d.length>0){let[m,...S]=d;const C=r[g][1];Sa(C)&&Sa(m)&&(m=fo(!0,C,m)),r[g]=[u,m,...S]}}}}const X1=new Is().freeze();function go(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function vo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function yo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Ac(e){if(!Sa(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Dc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Ni(e){return G1(e)?e:new Ip(e)}function G1(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function J1(e){return typeof e=="string"||Z1(e)}function Z1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const e0="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Mc=[],Rc={allowDangerousHtml:!0},t0=/^(https?|ircs?|mailto|xmpp)$/i,n0=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function r0(e){const t=i0(e),n=l0(e);return o0(t.runSync(t.parse(n),n),e)}function i0(e){const t=e.rehypePlugins||Mc,n=e.remarkPlugins||Mc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Rc}:Rc;return X1().use(Ox).use(n).use(I1,r).use(t)}function l0(e){const t=e.children||"",n=new Ip;return typeof t=="string"&&(n.value=t),n}function o0(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||a0;for(const d of n0)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+e0+d.id,void 0);return Tp(e,u),vv(e,{Fragment:c.Fragment,components:i,ignoreInvalidStyle:!0,jsx:c.jsx,jsxs:c.jsxs,passKeys:!0,passNode:!0});function u(d,f,g){if(d.type==="raw"&&g&&typeof f=="number")return o?g.children.splice(f,1):g.children[f]={type:"text",value:d.value},f;if(d.type==="element"){let m;for(m in so)if(Object.hasOwn(so,m)&&Object.hasOwn(d.properties,m)){const S=d.properties[m],C=so[m];(C===null||C.includes(d.tagName))&&(d.properties[m]=s(String(S||""),m,d))}}if(d.type==="element"){let m=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!m&&r&&typeof f=="number"&&(m=!r(d,f,g)),m&&g&&typeof f=="number")return a&&d.children?g.children.splice(f,1,...d.children):g.children.splice(f,1),f}}}function a0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||t0.test(e.slice(0,t))?e:""}const Oc=10*1024,xo=200,Ke={send:c.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),c.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),c.jsx("polyline",{points:"14 2 14 8 20 8"}),c.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),c.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),c.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),c.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),c.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),c.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),c.jsx("circle",{cx:"12",cy:"5",r:"2"}),c.jsx("path",{d:"M12 7v4"})]}),check:c.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),c.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},s0=e=>{switch(e){case"directive":return Ke.directive;case"question":return Ke.question;case"status":return Ke.status;case"result":return Ke.result;case"approval_request":return Ke.lock;default:return Ke.directive}},u0=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=U.useRef(null),[a,s]=Ft.useState(""),[u,d]=Ft.useState("directive"),[f,g]=Ft.useState(""),[m,S]=Ft.useState(!1),[C,j]=Ft.useState(new Map),[p,h]=Ft.useState(new Set),[y,k]=U.useState(new Set),b=P=>{const V=(P.match(/\n/g)||[]).length+1;if(!(P.length>Oc||V>xo))return{needsTruncation:!1,truncated:P,fullLength:P.length,lineCount:V};let E=P.slice(0,Oc);const v=E.split(`
`);v.length>xo&&(E=v.slice(0,xo).join(`
`));const L=E.lastIndexOf(`
`);return L>E.length*.8&&(E=E.slice(0,L)),{needsTruncation:!0,truncated:E,fullLength:P.length,lineCount:V}},w=P=>{k(V=>{const T=new Set(V);return T.has(P)?T.delete(P):T.add(P),T})};U.useEffect(()=>{e!=null&&e.workspace?g(e.workspace):g("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),U.useEffect(()=>{var P;(P=o.current)==null||P.scrollIntoView({behavior:"smooth"})},[t]);const z=P=>{g(P),r&&r(P)},D=()=>{a.trim()&&(n(a,u,f||void 0),s(""))},H=P=>{P.key==="Enter"&&!P.shiftKey&&(P.preventDefault(),D())},O=P=>new Date(P).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),_=P=>P.length>12?`${P.slice(0,8)}...`:P,M=P=>{if(!P.metadata_json)return null;try{return JSON.parse(P.metadata_json).approval_id||null}catch{return null}},Y=P=>{const V=C.get(P)||"";i&&(i(P,V),h(T=>new Set(T).add(P)),j(T=>{const E=new Map(T);return E.delete(P),E}))},G=P=>{const V=C.get(P)||"";if(!V.trim()){alert("Please provide a reason for rejection");return}l&&(l(P,V),h(T=>new Set(T).add(P)),j(T=>{const E=new Map(T);return E.delete(P),E}))},$=(P,V)=>{j(T=>new Map(T).set(P,V))};return e?c.jsxs("div",{className:"conversation-view",children:[c.jsxs("div",{className:"conversation-header",children:[c.jsxs("div",{className:"header-info",children:[c.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&c.jsxs("span",{className:"thread-agent-badge",children:[Ke.bot,e.target_agent]})]}),c.jsxs("div",{className:"header-stats",children:[c.jsxs("span",{className:"message-count",children:[t.length," messages"]}),c.jsx("span",{className:"thread-id",title:e.id,children:_(e.id)})]})]}),c.jsxs("div",{className:"messages-container",children:[t.length===0?c.jsxs("div",{className:"empty-messages",children:[c.jsx("div",{className:"empty-icon",children:c.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:c.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),c.jsx("p",{children:"No messages yet"}),c.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((P,V)=>{const T=P.from_type==="human",E=V===0||t[V-1].from_type!==P.from_type,v=y.has(P.id),{needsTruncation:L,truncated:B,fullLength:x,lineCount:te}=b(P.content),ke=v?P.content:B;return c.jsxs("div",{className:`message ${T?"human":"agent"}`,children:[c.jsx("div",{className:`message-avatar ${E?"visible":""}`,children:E&&(T?Ke.user:Ke.bot)}),c.jsxs("div",{className:"message-body",children:[E&&c.jsxs("div",{className:"message-meta",children:[c.jsx("span",{className:"sender-name",children:P.from_id}),c.jsxs("span",{className:"kind-badge",children:[s0(P.kind)," ",P.kind]}),c.jsx("span",{className:"message-time",children:O(P.created_at)})]}),c.jsxs("div",{className:"message-content",children:[P.kind==="result"||!T?c.jsx(r0,{components:{a:({href:Q,children:ve})=>{let je=Q;return Q&&Q.startsWith("/")&&!Q.startsWith("//")&&(je=`file://${Q}`),c.jsx("a",{href:je,target:"_blank",rel:"noopener noreferrer",children:ve})},code:({className:Q,children:ve,...je})=>!Q?c.jsx("code",{className:"inline-code",...je,children:ve}):c.jsx("code",{className:Q,...je,children:ve})},children:ke}):ke,L&&c.jsx("div",{className:"truncation-notice",children:c.jsx("button",{className:"expand-btn",onClick:()=>w(P.id),children:v?c.jsx(c.Fragment,{children:"Show less"}):c.jsxs(c.Fragment,{children:["Show more (",Math.round(x/1024),"KB, ",te," lines)"]})})}),P.kind==="approval_request"&&(()=>{const Q=M(P),ve=Q&&p.has(Q);return Q?c.jsx("div",{className:"inline-approval",children:ve?c.jsxs("div",{className:"approval-handled",children:[Ke.check,c.jsx("span",{children:"Action taken"})]}):c.jsxs(c.Fragment,{children:[c.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:C.get(Q)||"",onChange:je=>$(Q,je.target.value)}),c.jsxs("div",{className:"approval-actions",children:[c.jsxs("button",{className:"reject-btn",onClick:()=>G(Q),title:"Reject",children:[Ke.x,"Reject"]}),c.jsxs("button",{className:"approve-btn",onClick:()=>Y(Q),title:"Approve",children:[Ke.check,"Approve"]})]})]})}):null})()]}),c.jsxs("div",{className:"message-footer",children:[c.jsxs("span",{className:"message-seq",children:["#",P.message_seq]}),P.delivery_state!=="acked"&&c.jsx("span",{className:`delivery-status ${P.delivery_state}`,children:P.delivery_state==="pending"?"sending...":"delivered"})]})]})]},P.id)}),c.jsx("div",{ref:o})]}),c.jsxs("div",{className:"input-area",children:[m&&c.jsxs("div",{className:"workspace-input-row",children:[c.jsx("input",{type:"text",value:f,onChange:P=>z(P.target.value),onBlur:()=>{r&&r(f)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),c.jsx("button",{onClick:async()=>{try{const V=await(await fetch("/api/select-folder")).json();!V.cancelled&&V.path&&z(V.path)}catch(P){console.error("Failed to open folder picker:",P)}},className:"workspace-browse",title:"Browse for folder",children:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),c.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),c.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),f&&c.jsx("button",{onClick:()=>{z(""),S(!1)},className:"workspace-clear",children:"Clear"})]}),c.jsxs("div",{className:"input-wrapper",children:[c.jsx("button",{onClick:()=>S(!m),className:`workspace-toggle ${f?"has-workspace":""}`,title:f||"Set working directory",children:c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),c.jsxs("select",{value:u,onChange:P=>d(P.target.value),className:"kind-selector",children:[c.jsx("option",{value:"directive",children:"Directive"}),c.jsx("option",{value:"question",children:"Question"})]}),c.jsx("textarea",{value:a,onChange:P=>s(P.target.value),onKeyPress:H,placeholder:f?`Message (workspace: ${f.split("/").pop()})`:"Type a message...",rows:1}),c.jsx("button",{onClick:D,className:"send-btn",disabled:!a.trim(),children:Ke.send})]}),c.jsxs("div",{className:"input-hint",children:["Press ",c.jsx("kbd",{children:"Enter"})," to send, ",c.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),c.jsx("style",{children:`
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
      `})]}):null},c0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,reconnectInterval:l=5e3})=>{const o=U.useRef(null),[a,s]=U.useState(!1),[u,d]=U.useState(null),f=U.useRef(null),g=U.useRef(new Map),m=U.useCallback(()=>{try{const k=`${e}?instance_id=${t}`;o.current=new WebSocket(k),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),d(null),g.current.forEach((b,w)=>{j(w,b)})},o.current.onmessage=b=>{try{const w=JSON.parse(b.data);S(w)}catch(w){console.error("Failed to parse WebSocket message:",w)}},o.current.onerror=b=>{console.error("WebSocket error:",b),d("Connection error")},o.current.onclose=()=>{console.log("WebSocket disconnected"),s(!1),f.current=setTimeout(()=>{console.log("Attempting to reconnect..."),m()},l)}}catch(k){console.error("Failed to connect to WebSocket:",k),d("Failed to connect")}},[e,t,l]),S=U.useCallback(k=>{switch(k.type){case"message":n&&k.data&&n(k.data);break;case"batch":if(r&&k.data){const b=k.data;r(b),n&&b.messages.forEach(w=>n(w))}break;case"error":i&&k.data&&i(k.data),console.error("WebSocket error event:",k.data);break;case"pong":break;default:console.log("Unknown event type:",k.type)}},[n,r,i]),C=U.useCallback(k=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(k)):console.warn("WebSocket not connected, cannot send event")},[]),j=U.useCallback((k,b=0)=>{g.current.set(k,b);const w={type:"subscribe",timestamp:Date.now(),data:{thread_id:k,from_seq:b}};C(w)},[C]),p=U.useCallback((k,b)=>{const w=g.current.get(k)||0;b>w&&g.current.set(k,b);const z={type:"ack",timestamp:Date.now(),data:{thread_id:k,ack_seq:b}};C(z)},[C]),h=U.useCallback(()=>{const k={type:"ping",timestamp:Date.now()};C(k)},[C]),y=U.useCallback(k=>{g.current.delete(k)},[]);return U.useEffect(()=>(m(),()=>{f.current&&clearTimeout(f.current),o.current&&o.current.close()}),[m]),U.useEffect(()=>{if(!a)return;const k=setInterval(()=>{h()},3e4);return()=>clearInterval(k)},[a,h]),{isConnected:a,connectionError:u,subscribe:j,unsubscribe:y,acknowledge:p,ping:h}},d0=({connected:e})=>c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?c.jsxs(c.Fragment,{children:[c.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),c.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):c.jsxs(c.Fragment,{children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),c.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),f0=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=U.useState([]),[o,a]=U.useState(null),[s,u]=U.useState(new Map),[d,f]=U.useState(new Map),[g,m]=U.useState([]),[S,C]=U.useState(!1),[j,p]=U.useState(""),{isConnected:h,subscribe:y,acknowledge:k}=c0({url:e,instanceId:t,onMessage:b,onBatch:w});function b(E){const v={id:E.id,thread_id:E.thread_id,message_seq:E.message_seq,created_at:E.created_at,from_type:E.from_type,from_id:E.from_id,to_type:E.to_type,to_id:E.to_id,kind:E.kind,subject:E.subject,content:E.content,metadata_json:E.metadata_json,delivery_state:"visible",business_state:"open"};u(L=>{const B=L.get(v.thread_id)||[];return B.find(x=>x.id===v.id)?L:new Map(L).set(v.thread_id,[...B,v].sort((x,te)=>x.message_seq-te.message_seq))}),v.thread_id!==o&&f(L=>{const B=L.get(v.thread_id)||0;return new Map(L).set(v.thread_id,B+1)}),k(v.thread_id,v.message_seq)}function w(E){E.messages.forEach(v=>{b(v)})}const z=U.useCallback(E=>{if(a(E),f(v=>{const L=new Map(v);return L.delete(E),L}),h){const v=s.get(E)||[],L=v.length>0?Math.max(...v.map(B=>B.message_seq)):0;y(E,L)}},[h,y,s]),D=U.useCallback(async(E,v,L)=>{if(!o)return;const B=L?JSON.stringify({workspace:L}):void 0;try{const x=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:v,content:E,metadata_json:B})});if(!x.ok){console.error("Failed to send message:",await x.text());return}const te=await x.json();u(ke=>{const Q=ke.get(o)||[];return Q.find(ve=>ve.id===te.id)?ke:new Map(ke).set(o,[...Q,te])})}catch(x){console.error("Error sending message:",x)}},[o,t]);U.useEffect(()=>{(async()=>{try{const v=await fetch("/api/threads");if(!v.ok){console.error("Failed to fetch threads:",await v.text());return}const L=await v.json();l(L),L.length>0&&!o&&a(L[0].id)}catch(v){console.error("Error fetching threads:",v)}})()},[]),U.useEffect(()=>{n&&i.length>0&&(i.some(v=>v.id===n)&&(a(n),f(v=>{const L=new Map(v);return L.delete(n),L})),r&&r())},[n,i,r]);const H=U.useCallback(async E=>{try{const v=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:E,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!v.ok){console.error("Failed to create thread:",await v.text());return}const L=await v.json();l(B=>[L,...B]),a(L.id)}catch(v){console.error("Error creating thread:",v)}},[t]),O=U.useCallback(async()=>{try{const E=await fetch("/api/agents");if(!E.ok){console.error("Failed to fetch agents:",await E.text());return}const v=await E.json();m(v.running||[])}catch(E){console.error("Error fetching agents:",E)}},[]);U.useEffect(()=>{O();const E=setInterval(O,5e3);return()=>clearInterval(E)},[O]);const _=U.useCallback(async()=>{if(j.trim())try{const E=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:j.trim()})});if(!E.ok){const L=await E.text();console.error("Failed to launch agent:",L),alert(`Failed to launch agent: ${L}`);return}const v=await E.json();m(L=>[...L,v]),p(""),C(!1)}catch(E){console.error("Error launching agent:",E)}},[j]),M=U.useCallback(async E=>{try{const v=await fetch(`/api/agents/${E}`,{method:"DELETE"});if(!v.ok){console.error("Failed to stop agent:",await v.text());return}m(L=>L.filter(B=>B.instance_id!==E))}catch(v){console.error("Error stopping agent:",v)}},[]),Y=U.useCallback(async E=>{if(o)try{const v=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:E})});if(!v.ok){console.error("Failed to update workspace:",await v.text());return}const L=await v.json();l(B=>B.map(x=>x.id===o?L:x))}catch(v){console.error("Error updating workspace:",v)}},[o]),G=U.useCallback(async E=>{try{const v=await fetch(`/api/threads/${E}`,{method:"DELETE"});if(!v.ok){console.error("Failed to delete thread:",await v.text());return}l(L=>L.filter(B=>B.id!==E)),u(L=>{const B=new Map(L);return B.delete(E),B}),f(L=>{const B=new Map(L);return B.delete(E),B}),o===E&&a(null)}catch(v){console.error("Error deleting thread:",v)}},[o]),$=U.useCallback(async(E,v)=>{try{const L=await fetch(`/api/threads/${E}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:v})});if(!L.ok){console.error("Failed to rename thread:",await L.text());return}const B=await L.json();l(x=>x.map(te=>te.id===E?B:te))}catch(L){console.error("Error renaming thread:",L)}},[]),P=U.useCallback(async(E,v)=>{try{const L=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:v})});if(!L.ok){const B=await L.text();console.error("Failed to approve request:",B),alert(`Failed to approve: ${B}`);return}console.log("Approval approved successfully")}catch(L){console.error("Error approving request:",L)}},[]),V=U.useCallback(async(E,v)=>{try{const L=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:v})});if(!L.ok){const B=await L.text();console.error("Failed to reject request:",B),alert(`Failed to reject: ${B}`);return}console.log("Approval rejected successfully")}catch(L){console.error("Error rejecting request:",L)}},[]),T=o?s.get(o)||[]:[];return c.jsxs("div",{className:"message-center",children:[c.jsxs("div",{className:"status-bar",children:[c.jsxs("div",{className:`status-indicator ${h?"connected":"disconnected"}`,children:[c.jsx(d0,{connected:h}),c.jsx("span",{children:h?"Connected":"Disconnected"})]}),c.jsxs("div",{className:"status-meta",children:[c.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),c.jsxs("span",{className:"agent-count",children:[g.length," agents"]}),c.jsx("button",{className:"launch-agent-btn",onClick:()=>C(!0),children:"+ Agent"})]})]}),g.length>0&&c.jsx("div",{className:"agents-bar",children:g.map(E=>c.jsxs("div",{className:"agent-chip",children:[c.jsx("span",{className:"agent-pulse"}),c.jsx("span",{className:"agent-name",children:E.instance_id}),c.jsxs("span",{className:"agent-pid",children:["PID ",E.pid]}),c.jsx("button",{className:"agent-stop-btn",onClick:()=>M(E.instance_id),title:"Stop agent",children:"×"})]},E.instance_id))}),S&&c.jsx("div",{className:"modal-overlay",onClick:()=>C(!1),children:c.jsxs("div",{className:"modal-content",onClick:E=>E.stopPropagation(),children:[c.jsx("h3",{children:"Launch New Agent"}),c.jsx("input",{type:"text",value:j,onChange:E=>p(E.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:E=>{E.key==="Enter"&&_(),E.key==="Escape"&&C(!1)}}),c.jsxs("div",{className:"modal-actions",children:[c.jsx("button",{className:"cancel-btn",onClick:()=>C(!1),children:"Cancel"}),c.jsx("button",{className:"launch-btn",onClick:_,children:"Launch"})]})]})}),c.jsxs("div",{className:"center-layout",children:[c.jsx("aside",{className:"threads-panel",children:c.jsx(kg,{threads:i,selectedThreadId:o,onSelectThread:z,onCreateThread:H,onDeleteThread:G,onRenameThread:$,unreadCounts:d})}),c.jsx("main",{className:"conversation-panel",children:o?c.jsx(u0,{thread:i.find(E=>E.id===o),messages:T,onSendMessage:D,onWorkspaceChange:Y,onApproveRequest:P,onRejectRequest:V}):c.jsxs("div",{className:"empty-state",children:[c.jsx("div",{className:"empty-icon",children:c.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:c.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),c.jsx("h3",{children:"Select a conversation"}),c.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),c.jsx("style",{children:`
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
      `})]})},Le={check:c.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:c.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),c.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:c.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:c.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),c.jsx("circle",{cx:"12",cy:"5",r:"2"}),c.jsx("path",{d:"M12 7v4"})]}),dollar:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),c.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:c.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:c.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:c.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:c.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),c.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),c.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},p0=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=U.useState(!0),[a,s]=U.useState(null),[u,d]=U.useState(new Map),f=p=>{try{return JSON.parse(p)}catch{return null}},g=p=>new Date(p).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),m=p=>{const h=u.get(p)||"";n(p,h),d(new Map(u.set(p,"")))},S=p=>{const h=u.get(p)||"";if(!h.trim()){alert("Please provide a reason for rejection");return}r(p,h),d(new Map(u.set(p,"")))},C=(p,h)=>{d(new Map(u.set(p,h)))},j=e.filter(p=>p.status==="pending");return c.jsxs("div",{className:"approval-queue",children:[c.jsx("div",{className:"queue-header",children:c.jsxs("div",{className:"header-title",children:[c.jsx("h2",{children:"Approval Queue"}),c.jsxs("span",{className:"pending-count",children:[j.length," pending"]})]})}),c.jsxs("div",{className:"approvals-container",children:[j.length===0?c.jsxs("div",{className:"empty-state",children:[c.jsx("div",{className:"empty-icon",children:Le.sparkles}),c.jsx("h3",{children:"All caught up!"}),c.jsx("p",{children:"No pending approvals to review"})]}):c.jsx("div",{className:"approvals-list",children:j.map(p=>{const h=f(p.effect_delta_json),y=a===p.id;return c.jsxs("div",{className:`approval-card impact-${p.impact}`,children:[c.jsxs("div",{className:"card-header",onClick:()=>s(y?null:p.id),children:[c.jsxs("div",{className:"header-left",children:[c.jsx("div",{className:`impact-indicator ${p.impact}`}),c.jsxs("div",{className:"proposal-info",children:[c.jsx("span",{className:"proposal-text",children:p.proposal}),c.jsxs("div",{className:"proposal-meta",children:[p.thread_title&&c.jsxs("span",{className:"meta-item thread-link",onClick:k=>{k.stopPropagation(),i==null||i(p.thread_id)},title:"Go to thread",children:[Le.message,p.thread_title]}),c.jsxs("span",{className:"meta-item",children:[Le.bot,p.instance_id]}),c.jsxs("span",{className:"meta-item",children:[Le.clock,g(p.created_at)]})]})]})]}),c.jsxs("div",{className:"header-right",children:[c.jsxs("span",{className:"cost-badge",children:[Le.dollar,"$",p.estimated_cost.toFixed(2)]}),c.jsx("span",{className:`impact-badge ${p.impact}`,children:p.impact}),c.jsx("button",{className:"expand-btn",children:y?Le.chevronUp:Le.chevronDown})]})]}),y&&c.jsxs("div",{className:"card-details",children:[h&&c.jsxs("div",{className:"detail-section",children:[c.jsx("h4",{children:"Effect Details"}),c.jsxs("div",{className:"detail-grid",children:[c.jsxs("div",{className:"detail-item",children:[c.jsx("span",{className:"detail-label",children:"Capability"}),c.jsx("span",{className:"detail-value code",children:h.cap_type})]}),c.jsxs("div",{className:"detail-item",children:[c.jsx("span",{className:"detail-label",children:"Budget Delta"}),c.jsxs("span",{className:"detail-value",children:["$",h.budget_delta.toFixed(2)]})]}),h.paths.length>0&&c.jsxs("div",{className:"detail-item full-width",children:[c.jsx("span",{className:"detail-label",children:"Paths"}),c.jsx("div",{className:"paths-list",children:h.paths.map((k,b)=>c.jsxs("span",{className:"path-tag",children:[Le.folder,k]},b))})]})]})]}),c.jsxs("div",{className:"detail-section",children:[c.jsx("h4",{children:"Request Info"}),c.jsxs("div",{className:"detail-grid",children:[c.jsxs("div",{className:"detail-item",children:[c.jsx("span",{className:"detail-label",children:"Thread"}),c.jsx("span",{className:"detail-value code",children:p.thread_id})]}),c.jsxs("div",{className:"detail-item",children:[c.jsx("span",{className:"detail-label",children:"Impact Level"}),c.jsx("span",{className:`detail-value impact-text ${p.impact}`,children:p.impact.toUpperCase()})]})]})]}),c.jsxs("div",{className:"review-section",children:[c.jsx("h4",{children:"Review Notes"}),c.jsx("textarea",{value:u.get(p.id)||"",onChange:k=>C(p.id,k.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),c.jsxs("div",{className:"action-buttons",children:[c.jsxs("button",{className:"reject-btn",onClick:()=>S(p.id),children:[Le.x,"Reject"]}),c.jsxs("button",{className:"approve-btn",onClick:()=>m(p.id),children:[Le.check,"Approve"]})]})]})]})]},p.id)})}),t.length>0&&c.jsxs("div",{className:"history-section",children:[c.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[c.jsxs("h3",{children:[l?Le.chevronDown:Le.chevronUp,"Review History"]}),c.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&c.jsx("div",{className:"history-list",children:t.map(p=>{const h=a===`history-${p.id}`;return c.jsxs("div",{className:`history-card ${p.status}`,onClick:()=>s(h?null:`history-${p.id}`),children:[c.jsxs("div",{className:"history-card-header",children:[c.jsxs("div",{className:"history-status",children:[c.jsx("span",{className:`status-icon ${p.status}`,children:p.status==="approved"?Le.check:Le.x}),c.jsxs("div",{className:"history-info",children:[c.jsx("span",{className:"history-proposal",children:p.proposal}),p.thread_title&&c.jsxs("span",{className:"history-thread",onClick:y=>{y.stopPropagation(),i==null||i(p.thread_id)},title:"Go to thread",children:[Le.message,p.thread_title]})]})]}),c.jsxs("div",{className:"history-meta",children:[c.jsx("span",{className:"history-agent",children:p.instance_id}),c.jsx("span",{className:`history-badge ${p.status}`,children:p.status}),c.jsx("span",{className:"history-time",children:p.reviewed_at?g(p.reviewed_at):g(p.created_at)})]})]}),h&&c.jsxs("div",{className:"history-details",children:[c.jsxs("div",{className:"detail-row",children:[c.jsx("span",{className:"detail-label",children:"Reviewed by"}),c.jsx("span",{className:"detail-value",children:p.reviewed_by||"Unknown"})]}),c.jsxs("div",{className:"detail-row",children:[c.jsx("span",{className:"detail-label",children:"Cost"}),c.jsxs("span",{className:"detail-value",children:["$",p.estimated_cost.toFixed(2)]})]}),c.jsxs("div",{className:"detail-row",children:[c.jsx("span",{className:"detail-label",children:"Impact"}),c.jsx("span",{className:`detail-value impact-text ${p.impact}`,children:p.impact.toUpperCase()})]}),p.review_notes&&c.jsxs("div",{className:"detail-row full-width",children:[c.jsx("span",{className:"detail-label",children:"Notes"}),c.jsx("span",{className:"detail-value notes",children:p.review_notes})]})]})]},`history-${p.id}`)})})]})]}),c.jsx("style",{children:`
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
      `})]})},h0=c.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[c.jsx("circle",{cx:"12",cy:"12",r:"10"}),c.jsx("path",{d:"M12 6v12M6 12h12"}),c.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]}),m0=()=>{const[e,t]=U.useState({type:"overview"}),[n,r]=U.useState(null),[i,l]=U.useState([]),[o,a]=U.useState([]),u=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;U.useEffect(()=>{const j=async()=>{try{const h=await fetch("/api/hierarchy");if(h.ok){const y=await h.json();r(y)}}catch(h){console.error("Error fetching hierarchy:",h)}};j();const p=setInterval(j,5e3);return()=>clearInterval(p)},[]),U.useEffect(()=>{const j=async()=>{try{const h=await fetch("/api/approvals?status=pending");if(h.ok){const w=await h.json();l(w)}const[y,k]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),b=[];if(y.ok){const w=await y.json();b.push(...w)}if(k.ok){const w=await k.json();b.push(...w)}b.sort((w,z)=>{const D=w.reviewed_at?new Date(w.reviewed_at).getTime():0;return(z.reviewed_at?new Date(z.reviewed_at).getTime():0)-D}),a(b)}catch(h){console.error("Error fetching approvals:",h)}};j();const p=setInterval(j,5e3);return()=>clearInterval(p)},[]);const d=async(j,p)=>{try{const h=await fetch(`/api/approvals/${j}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:p})});if(!h.ok){console.error("Failed to approve:",await h.text());return}const y=i.find(k=>k.id===j);if(y){const k={...y,status:"approved",reviewed_by:"user",review_notes:p,reviewed_at:Date.now()};a(b=>[k,...b])}l(k=>k.filter(b=>b.id!==j))}catch(h){console.error("Error approving:",h)}},f=async(j,p)=>{try{const h=await fetch(`/api/approvals/${j}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:p})});if(!h.ok){console.error("Failed to reject:",await h.text());return}const y=i.find(k=>k.id===j);if(y){const k={...y,status:"rejected",reviewed_by:"user",review_notes:p,reviewed_at:Date.now()};a(b=>[k,...b])}l(k=>k.filter(b=>b.id!==j))}catch(h){console.error("Error rejecting:",h)}},g=()=>{var p,h;const j=[{label:"All Agents",onClick:()=>t({type:"overview"})}];if(e.type==="agent"&&e.agentId&&j.push({label:e.agentId}),e.type==="thread"&&e.threadId){e.agentId&&j.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})});const y=(p=n==null?void 0:n.root.children)==null?void 0:p.find(b=>b.id===e.agentId),k=(h=y==null?void 0:y.children)==null?void 0:h.find(b=>b.id===e.threadId);j.push({label:(k==null?void 0:k.label)||"Thread"})}return j},m=j=>{var h;const p=(h=n==null?void 0:n.root.children)==null?void 0:h.find(y=>{var k;return(k=y.children)==null?void 0:k.some(b=>b.id===j)});t({type:"thread",agentId:p==null?void 0:p.id,threadId:j})},S=()=>{var j,p,h;if(e.type==="overview"&&n)return c.jsx(yg,{aggregate:n.aggregate,agents:n.root.children||[],onSelectAgent:y=>t({type:"agent",agentId:y})});if(e.type==="agent"&&e.agentId){const y=(j=n==null?void 0:n.root.children)==null?void 0:j.find(b=>b.id===e.agentId),k=i.filter(b=>{var w;return(w=y==null?void 0:y.children)==null?void 0:w.some(z=>z.id===b.thread_id)});return c.jsxs("div",{className:"agent-view",children:[c.jsxs("div",{className:"agent-view-header",children:[c.jsx("h2",{children:e.agentId}),c.jsxs("span",{className:"agent-thread-count",children:[((p=y==null?void 0:y.children)==null?void 0:p.length)||0," threads"]})]}),c.jsxs("div",{className:"agent-view-content",children:[c.jsxs("div",{className:"agent-threads",children:[c.jsx("h3",{children:"Threads"}),(h=y==null?void 0:y.children)==null?void 0:h.map(b=>c.jsxs("div",{className:"thread-card",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:b.id}),children:[c.jsx("span",{className:"thread-title",children:b.label}),b.badges&&b.badges.length>0&&c.jsx("span",{className:"thread-badges",children:b.badges.map((w,z)=>c.jsx("span",{className:`badge badge-${w.type}`,children:w.count},z))})]},b.id)),(!(y!=null&&y.children)||y.children.length===0)&&c.jsx("div",{className:"no-threads",children:"No threads yet"})]}),k.length>0&&c.jsxs("div",{className:"agent-approvals",children:[c.jsx("h3",{children:"Pending Approvals"}),c.jsx(p0,{approvals:k,history:[],onApprove:d,onReject:f,onNavigateToThread:m})]})]})]})}return e.type==="thread"&&e.threadId?c.jsx(f0,{websocketUrl:u,instanceId:e.agentId||"default",initialThreadId:e.threadId,onThreadNavigated:()=>{}}):c.jsx("div",{className:"empty-state",children:c.jsx("p",{children:"Select an agent or thread from the sidebar"})})},C=(i==null?void 0:i.filter(j=>j.status==="pending").length)||0;return c.jsxs("div",{className:"app",children:[c.jsxs("header",{className:"app-header",children:[c.jsxs("div",{className:"header-brand",children:[c.jsx("div",{className:"brand-logo",children:h0}),c.jsxs("div",{className:"brand-text",children:[c.jsx("h1",{children:"AILANG"}),c.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),c.jsxs("div",{className:"header-meta",children:[C>0&&c.jsxs("span",{className:"pending-badge",title:`${C} pending approvals`,children:[C," pending"]}),c.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),c.jsxs("div",{className:"app-body",children:[c.jsx("aside",{className:"app-sidebar",children:c.jsx(gg,{selection:e,onSelect:t})}),c.jsxs("main",{className:"app-main",children:[e.type!=="overview"&&c.jsx(xg,{items:g()}),c.jsx("div",{className:"main-content",children:S()})]})]}),c.jsx("style",{children:`
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
      `})]})};ko.createRoot(document.getElementById("root")).render(c.jsx(Ft.StrictMode,{children:c.jsx(m0,{})}));
