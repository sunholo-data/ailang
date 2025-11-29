(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Ui=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Ca(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Dc={exports:{}},hl={},Rc={exports:{}},V={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var Jr=Symbol.for("react.element"),Up=Symbol.for("react.portal"),Hp=Symbol.for("react.fragment"),Vp=Symbol.for("react.strict_mode"),$p=Symbol.for("react.profiler"),Wp=Symbol.for("react.provider"),Qp=Symbol.for("react.context"),Kp=Symbol.for("react.forward_ref"),qp=Symbol.for("react.suspense"),Yp=Symbol.for("react.memo"),Xp=Symbol.for("react.lazy"),Rs=Symbol.iterator;function Gp(e){return e===null||typeof e!="object"?null:(e=Rs&&e[Rs]||e["@@iterator"],typeof e=="function"?e:null)}var Fc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Bc=Object.assign,Uc={};function tr(e,t,n){this.props=e,this.context=t,this.refs=Uc,this.updater=n||Fc}tr.prototype.isReactComponent={};tr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};tr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function Hc(){}Hc.prototype=tr.prototype;function Ea(e,t,n){this.props=e,this.context=t,this.refs=Uc,this.updater=n||Fc}var Na=Ea.prototype=new Hc;Na.constructor=Ea;Bc(Na,tr.prototype);Na.isPureReactComponent=!0;var Fs=Array.isArray,Vc=Object.prototype.hasOwnProperty,_a={current:null},$c={key:!0,ref:!0,__self:!0,__source:!0};function Wc(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)Vc.call(t,r)&&!$c.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),u=0;u<a;u++)s[u]=arguments[u+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:Jr,type:e,key:l,ref:o,props:i,_owner:_a.current}}function Jp(e,t){return{$$typeof:Jr,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function ja(e){return typeof e=="object"&&e!==null&&e.$$typeof===Jr}function Zp(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Bs=/\/+/g;function Ml(e,t){return typeof e=="object"&&e!==null&&e.key!=null?Zp(""+e.key):t.toString(36)}function ji(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case Jr:case Up:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Ml(o,0):r,Fs(i)?(n="",e!=null&&(n=e.replace(Bs,"$&/")+"/"),ji(i,t,n,"",function(u){return u})):i!=null&&(ja(i)&&(i=Jp(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Bs,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Fs(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Ml(l,a);o+=ji(l,t,n,s,i)}else if(s=Gp(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Ml(l,a++),o+=ji(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function oi(e,t,n){if(e==null)return e;var r=[],i=0;return ji(e,r,"","",function(l){return t.call(n,l,i++)}),r}function eh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Te={current:null},bi={transition:null},th={ReactCurrentDispatcher:Te,ReactCurrentBatchConfig:bi,ReactCurrentOwner:_a};function Qc(){throw Error("act(...) is not supported in production builds of React.")}V.Children={map:oi,forEach:function(e,t,n){oi(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return oi(e,function(){t++}),t},toArray:function(e){return oi(e,function(t){return t})||[]},only:function(e){if(!ja(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};V.Component=tr;V.Fragment=Hp;V.Profiler=$p;V.PureComponent=Ea;V.StrictMode=Vp;V.Suspense=qp;V.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=th;V.act=Qc;V.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Bc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=_a.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)Vc.call(t,s)&&!$c.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var u=0;u<s;u++)a[u]=arguments[u+2];r.children=a}return{$$typeof:Jr,type:e.type,key:i,ref:l,props:r,_owner:o}};V.createContext=function(e){return e={$$typeof:Qp,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:Wp,_context:e},e.Consumer=e};V.createElement=Wc;V.createFactory=function(e){var t=Wc.bind(null,e);return t.type=e,t};V.createRef=function(){return{current:null}};V.forwardRef=function(e){return{$$typeof:Kp,render:e}};V.isValidElement=ja;V.lazy=function(e){return{$$typeof:Xp,_payload:{_status:-1,_result:e},_init:eh}};V.memo=function(e,t){return{$$typeof:Yp,type:e,compare:t===void 0?null:t}};V.startTransition=function(e){var t=bi.transition;bi.transition={};try{e()}finally{bi.transition=t}};V.unstable_act=Qc;V.useCallback=function(e,t){return Te.current.useCallback(e,t)};V.useContext=function(e){return Te.current.useContext(e)};V.useDebugValue=function(){};V.useDeferredValue=function(e){return Te.current.useDeferredValue(e)};V.useEffect=function(e,t){return Te.current.useEffect(e,t)};V.useId=function(){return Te.current.useId()};V.useImperativeHandle=function(e,t,n){return Te.current.useImperativeHandle(e,t,n)};V.useInsertionEffect=function(e,t){return Te.current.useInsertionEffect(e,t)};V.useLayoutEffect=function(e,t){return Te.current.useLayoutEffect(e,t)};V.useMemo=function(e,t){return Te.current.useMemo(e,t)};V.useReducer=function(e,t,n){return Te.current.useReducer(e,t,n)};V.useRef=function(e){return Te.current.useRef(e)};V.useState=function(e){return Te.current.useState(e)};V.useSyncExternalStore=function(e,t,n){return Te.current.useSyncExternalStore(e,t,n)};V.useTransition=function(){return Te.current.useTransition()};V.version="18.3.1";Rc.exports=V;var H=Rc.exports;const un=Ca(H);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var nh=H,rh=Symbol.for("react.element"),ih=Symbol.for("react.fragment"),lh=Object.prototype.hasOwnProperty,oh=nh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,ah={key:!0,ref:!0,__self:!0,__source:!0};function Kc(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)lh.call(t,r)&&!ah.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:rh,type:e,key:l,ref:o,props:i,_owner:oh.current}}hl.Fragment=ih;hl.jsx=Kc;hl.jsxs=Kc;Dc.exports=hl;var h=Dc.exports,xo={},qc={exports:{}},qe={},Yc={exports:{}},Xc={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(L,R){var v=L.length;L.push(R);e:for(;0<v;){var Q=v-1>>>1,G=L[Q];if(0<i(G,R))L[Q]=R,L[v]=G,v=Q;else break e}}function n(L){return L.length===0?null:L[0]}function r(L){if(L.length===0)return null;var R=L[0],v=L.pop();if(v!==R){L[0]=v;e:for(var Q=0,G=L.length,x=G>>>1;Q<x;){var ge=2*(Q+1)-1,rt=L[ge],ne=ge+1,dt=L[ne];if(0>i(rt,v))ne<G&&0>i(dt,rt)?(L[Q]=dt,L[ne]=v,Q=ne):(L[Q]=rt,L[ge]=v,Q=ge);else if(ne<G&&0>i(dt,v))L[Q]=dt,L[ne]=v,Q=ne;else break e}}return R}function i(L,R){var v=L.sortIndex-R.sortIndex;return v!==0?v:L.id-R.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],u=[],c=1,d=null,p=3,f=!1,k=!1,C=!1,N=typeof setTimeout=="function"?setTimeout:null,m=typeof clearTimeout=="function"?clearTimeout:null,y=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function g(L){for(var R=n(u);R!==null;){if(R.callback===null)r(u);else if(R.startTime<=L)r(u),R.sortIndex=R.expirationTime,t(s,R);else break;R=n(u)}}function S(L){if(C=!1,g(L),!k)if(n(s)!==null)k=!0,pe(E);else{var R=n(u);R!==null&&fe(S,R.startTime-L)}}function E(L,R){k=!1,C&&(C=!1,m(P),P=-1),f=!0;var v=p;try{for(g(R),d=n(s);d!==null&&(!(d.expirationTime>R)||L&&!A());){var Q=d.callback;if(typeof Q=="function"){d.callback=null,p=d.priorityLevel;var G=Q(d.expirationTime<=R);R=e.unstable_now(),typeof G=="function"?d.callback=G:d===n(s)&&r(s),g(R)}else r(s);d=n(s)}if(d!==null)var x=!0;else{var ge=n(u);ge!==null&&fe(S,ge.startTime-R),x=!1}return x}finally{d=null,p=v,f=!1}}var w=!1,_=null,P=-1,O=5,M=-1;function A(){return!(e.unstable_now()-M<O)}function D(){if(_!==null){var L=e.unstable_now();M=L;var R=!0;try{R=_(!0,L)}finally{R?Y():(w=!1,_=null)}}else w=!1}var Y;if(typeof y=="function")Y=function(){y(D)};else if(typeof MessageChannel<"u"){var oe=new MessageChannel,$=oe.port2;oe.port1.onmessage=D,Y=function(){$.postMessage(null)}}else Y=function(){N(D,0)};function pe(L){_=L,w||(w=!0,Y())}function fe(L,R){P=N(function(){L(e.unstable_now())},R)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(L){L.callback=null},e.unstable_continueExecution=function(){k||f||(k=!0,pe(E))},e.unstable_forceFrameRate=function(L){0>L||125<L?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):O=0<L?Math.floor(1e3/L):5},e.unstable_getCurrentPriorityLevel=function(){return p},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(L){switch(p){case 1:case 2:case 3:var R=3;break;default:R=p}var v=p;p=R;try{return L()}finally{p=v}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(L,R){switch(L){case 1:case 2:case 3:case 4:case 5:break;default:L=3}var v=p;p=L;try{return R()}finally{p=v}},e.unstable_scheduleCallback=function(L,R,v){var Q=e.unstable_now();switch(typeof v=="object"&&v!==null?(v=v.delay,v=typeof v=="number"&&0<v?Q+v:Q):v=Q,L){case 1:var G=-1;break;case 2:G=250;break;case 5:G=1073741823;break;case 4:G=1e4;break;default:G=5e3}return G=v+G,L={id:c++,callback:R,priorityLevel:L,startTime:v,expirationTime:G,sortIndex:-1},v>Q?(L.sortIndex=v,t(u,L),n(s)===null&&L===n(u)&&(C?(m(P),P=-1):C=!0,fe(S,v-Q))):(L.sortIndex=G,t(s,L),k||f||(k=!0,pe(E))),L},e.unstable_shouldYield=A,e.unstable_wrapCallback=function(L){var R=p;return function(){var v=p;p=R;try{return L.apply(this,arguments)}finally{p=v}}}})(Xc);Yc.exports=Xc;var sh=Yc.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var uh=H,Ke=sh;function b(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var Gc=new Set,Mr={};function kn(e,t){qn(e,t),qn(e+"Capture",t)}function qn(e,t){for(Mr[e]=t,e=0;e<t.length;e++)Gc.add(t[e])}var Pt=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),ko=Object.prototype.hasOwnProperty,ch=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,Us={},Hs={};function fh(e){return ko.call(Hs,e)?!0:ko.call(Us,e)?!1:ch.test(e)?Hs[e]=!0:(Us[e]=!0,!1)}function dh(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function ph(e,t,n,r){if(t===null||typeof t>"u"||dh(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Le(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ce={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ce[e]=new Le(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ce[t]=new Le(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ce[e]=new Le(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ce[e]=new Le(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ce[e]=new Le(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ce[e]=new Le(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ce[e]=new Le(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ce[e]=new Le(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ce[e]=new Le(e,5,!1,e.toLowerCase(),null,!1,!1)});var ba=/[\-:]([a-z])/g;function za(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(ba,za);Ce[t]=new Le(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(ba,za);Ce[t]=new Le(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(ba,za);Ce[t]=new Le(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ce[e]=new Le(e,1,!1,e.toLowerCase(),null,!1,!1)});Ce.xlinkHref=new Le("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ce[e]=new Le(e,1,!1,e.toLowerCase(),null,!0,!0)});function Pa(e,t,n,r){var i=Ce.hasOwnProperty(t)?Ce[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(ph(t,n,i,r)&&(n=null),r||i===null?fh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Mt=uh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,ai=Symbol.for("react.element"),bn=Symbol.for("react.portal"),zn=Symbol.for("react.fragment"),Ta=Symbol.for("react.strict_mode"),wo=Symbol.for("react.profiler"),Jc=Symbol.for("react.provider"),Zc=Symbol.for("react.context"),La=Symbol.for("react.forward_ref"),So=Symbol.for("react.suspense"),Co=Symbol.for("react.suspense_list"),Ia=Symbol.for("react.memo"),Rt=Symbol.for("react.lazy"),ef=Symbol.for("react.offscreen"),Vs=Symbol.iterator;function sr(e){return e===null||typeof e!="object"?null:(e=Vs&&e[Vs]||e["@@iterator"],typeof e=="function"?e:null)}var ue=Object.assign,Al;function vr(e){if(Al===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Al=t&&t[1]||""}return`
`+Al+e}var Ol=!1;function Dl(e,t){if(!e||Ol)return"";Ol=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(u){var r=u}Reflect.construct(e,[],t)}else{try{t.call()}catch(u){r=u}e.call(t.prototype)}else{try{throw Error()}catch(u){r=u}e()}}catch(u){if(u&&r&&typeof u.stack=="string"){for(var i=u.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Ol=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?vr(e):""}function hh(e){switch(e.tag){case 5:return vr(e.type);case 16:return vr("Lazy");case 13:return vr("Suspense");case 19:return vr("SuspenseList");case 0:case 2:case 15:return e=Dl(e.type,!1),e;case 11:return e=Dl(e.type.render,!1),e;case 1:return e=Dl(e.type,!0),e;default:return""}}function Eo(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case zn:return"Fragment";case bn:return"Portal";case wo:return"Profiler";case Ta:return"StrictMode";case So:return"Suspense";case Co:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case Zc:return(e.displayName||"Context")+".Consumer";case Jc:return(e._context.displayName||"Context")+".Provider";case La:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Ia:return t=e.displayName||null,t!==null?t:Eo(e.type)||"Memo";case Rt:t=e._payload,e=e._init;try{return Eo(e(t))}catch{}}return null}function mh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Eo(t);case 8:return t===Ta?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function Jt(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function tf(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function gh(e){var t=tf(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function si(e){e._valueTracker||(e._valueTracker=gh(e))}function nf(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=tf(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Hi(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function No(e,t){var n=t.checked;return ue({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function $s(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=Jt(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function rf(e,t){t=t.checked,t!=null&&Pa(e,"checked",t,!1)}function _o(e,t){rf(e,t);var n=Jt(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?jo(e,t.type,n):t.hasOwnProperty("defaultValue")&&jo(e,t.type,Jt(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function Ws(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function jo(e,t,n){(t!=="number"||Hi(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var xr=Array.isArray;function Bn(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+Jt(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function bo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(b(91));return ue({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function Qs(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(b(92));if(xr(n)){if(1<n.length)throw Error(b(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:Jt(n)}}function lf(e,t){var n=Jt(t.value),r=Jt(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function Ks(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function of(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function zo(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?of(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var ui,af=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(ui=ui||document.createElement("div"),ui.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=ui.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Ar(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Sr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},yh=["Webkit","ms","Moz","O"];Object.keys(Sr).forEach(function(e){yh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Sr[t]=Sr[e]})});function sf(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Sr.hasOwnProperty(e)&&Sr[e]?(""+t).trim():t+"px"}function uf(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=sf(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var vh=ue({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Po(e,t){if(t){if(vh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(b(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(b(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(b(61))}if(t.style!=null&&typeof t.style!="object")throw Error(b(62))}}function To(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Lo=null;function Ma(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Io=null,Un=null,Hn=null;function qs(e){if(e=ti(e)){if(typeof Io!="function")throw Error(b(280));var t=e.stateNode;t&&(t=xl(t),Io(e.stateNode,e.type,t))}}function cf(e){Un?Hn?Hn.push(e):Hn=[e]:Un=e}function ff(){if(Un){var e=Un,t=Hn;if(Hn=Un=null,qs(e),t)for(e=0;e<t.length;e++)qs(t[e])}}function df(e,t){return e(t)}function pf(){}var Rl=!1;function hf(e,t,n){if(Rl)return e(t,n);Rl=!0;try{return df(e,t,n)}finally{Rl=!1,(Un!==null||Hn!==null)&&(pf(),ff())}}function Or(e,t){var n=e.stateNode;if(n===null)return null;var r=xl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(b(231,t,typeof n));return n}var Mo=!1;if(Pt)try{var ur={};Object.defineProperty(ur,"passive",{get:function(){Mo=!0}}),window.addEventListener("test",ur,ur),window.removeEventListener("test",ur,ur)}catch{Mo=!1}function xh(e,t,n,r,i,l,o,a,s){var u=Array.prototype.slice.call(arguments,3);try{t.apply(n,u)}catch(c){this.onError(c)}}var Cr=!1,Vi=null,$i=!1,Ao=null,kh={onError:function(e){Cr=!0,Vi=e}};function wh(e,t,n,r,i,l,o,a,s){Cr=!1,Vi=null,xh.apply(kh,arguments)}function Sh(e,t,n,r,i,l,o,a,s){if(wh.apply(this,arguments),Cr){if(Cr){var u=Vi;Cr=!1,Vi=null}else throw Error(b(198));$i||($i=!0,Ao=u)}}function wn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function mf(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function Ys(e){if(wn(e)!==e)throw Error(b(188))}function Ch(e){var t=e.alternate;if(!t){if(t=wn(e),t===null)throw Error(b(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return Ys(i),e;if(l===r)return Ys(i),t;l=l.sibling}throw Error(b(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(b(189))}}if(n.alternate!==r)throw Error(b(190))}if(n.tag!==3)throw Error(b(188));return n.stateNode.current===n?e:t}function gf(e){return e=Ch(e),e!==null?yf(e):null}function yf(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=yf(e);if(t!==null)return t;e=e.sibling}return null}var vf=Ke.unstable_scheduleCallback,Xs=Ke.unstable_cancelCallback,Eh=Ke.unstable_shouldYield,Nh=Ke.unstable_requestPaint,de=Ke.unstable_now,_h=Ke.unstable_getCurrentPriorityLevel,Aa=Ke.unstable_ImmediatePriority,xf=Ke.unstable_UserBlockingPriority,Wi=Ke.unstable_NormalPriority,jh=Ke.unstable_LowPriority,kf=Ke.unstable_IdlePriority,ml=null,vt=null;function bh(e){if(vt&&typeof vt.onCommitFiberRoot=="function")try{vt.onCommitFiberRoot(ml,e,void 0,(e.current.flags&128)===128)}catch{}}var ut=Math.clz32?Math.clz32:Th,zh=Math.log,Ph=Math.LN2;function Th(e){return e>>>=0,e===0?32:31-(zh(e)/Ph|0)|0}var ci=64,fi=4194304;function kr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function Qi(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=kr(a):(l&=o,l!==0&&(r=kr(l)))}else o=n&~i,o!==0?r=kr(o):l!==0&&(r=kr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-ut(t),i=1<<n,r|=e[n],t&=~i;return r}function Lh(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Ih(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-ut(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Lh(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Oo(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function wf(){var e=ci;return ci<<=1,!(ci&4194240)&&(ci=64),e}function Fl(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function Zr(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-ut(t),e[t]=n}function Mh(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-ut(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Oa(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-ut(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var X=0;function Sf(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Cf,Da,Ef,Nf,_f,Do=!1,di=[],$t=null,Wt=null,Qt=null,Dr=new Map,Rr=new Map,Bt=[],Ah="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function Gs(e,t){switch(e){case"focusin":case"focusout":$t=null;break;case"dragenter":case"dragleave":Wt=null;break;case"mouseover":case"mouseout":Qt=null;break;case"pointerover":case"pointerout":Dr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Rr.delete(t.pointerId)}}function cr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=ti(t),t!==null&&Da(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Oh(e,t,n,r,i){switch(t){case"focusin":return $t=cr($t,e,t,n,r,i),!0;case"dragenter":return Wt=cr(Wt,e,t,n,r,i),!0;case"mouseover":return Qt=cr(Qt,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Dr.set(l,cr(Dr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Rr.set(l,cr(Rr.get(l)||null,e,t,n,r,i)),!0}return!1}function jf(e){var t=cn(e.target);if(t!==null){var n=wn(t);if(n!==null){if(t=n.tag,t===13){if(t=mf(n),t!==null){e.blockedOn=t,_f(e.priority,function(){Ef(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function zi(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Ro(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Lo=r,n.target.dispatchEvent(r),Lo=null}else return t=ti(n),t!==null&&Da(t),e.blockedOn=n,!1;t.shift()}return!0}function Js(e,t,n){zi(e)&&n.delete(t)}function Dh(){Do=!1,$t!==null&&zi($t)&&($t=null),Wt!==null&&zi(Wt)&&(Wt=null),Qt!==null&&zi(Qt)&&(Qt=null),Dr.forEach(Js),Rr.forEach(Js)}function fr(e,t){e.blockedOn===t&&(e.blockedOn=null,Do||(Do=!0,Ke.unstable_scheduleCallback(Ke.unstable_NormalPriority,Dh)))}function Fr(e){function t(i){return fr(i,e)}if(0<di.length){fr(di[0],e);for(var n=1;n<di.length;n++){var r=di[n];r.blockedOn===e&&(r.blockedOn=null)}}for($t!==null&&fr($t,e),Wt!==null&&fr(Wt,e),Qt!==null&&fr(Qt,e),Dr.forEach(t),Rr.forEach(t),n=0;n<Bt.length;n++)r=Bt[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Bt.length&&(n=Bt[0],n.blockedOn===null);)jf(n),n.blockedOn===null&&Bt.shift()}var Vn=Mt.ReactCurrentBatchConfig,Ki=!0;function Rh(e,t,n,r){var i=X,l=Vn.transition;Vn.transition=null;try{X=1,Ra(e,t,n,r)}finally{X=i,Vn.transition=l}}function Fh(e,t,n,r){var i=X,l=Vn.transition;Vn.transition=null;try{X=4,Ra(e,t,n,r)}finally{X=i,Vn.transition=l}}function Ra(e,t,n,r){if(Ki){var i=Ro(e,t,n,r);if(i===null)Yl(e,t,r,qi,n),Gs(e,r);else if(Oh(i,e,t,n,r))r.stopPropagation();else if(Gs(e,r),t&4&&-1<Ah.indexOf(e)){for(;i!==null;){var l=ti(i);if(l!==null&&Cf(l),l=Ro(e,t,n,r),l===null&&Yl(e,t,r,qi,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else Yl(e,t,r,null,n)}}var qi=null;function Ro(e,t,n,r){if(qi=null,e=Ma(r),e=cn(e),e!==null)if(t=wn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=mf(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return qi=e,null}function bf(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(_h()){case Aa:return 1;case xf:return 4;case Wi:case jh:return 16;case kf:return 536870912;default:return 16}default:return 16}}var Ht=null,Fa=null,Pi=null;function zf(){if(Pi)return Pi;var e,t=Fa,n=t.length,r,i="value"in Ht?Ht.value:Ht.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Pi=i.slice(e,1<r?1-r:void 0)}function Ti(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function pi(){return!0}function Zs(){return!1}function Ye(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?pi:Zs,this.isPropagationStopped=Zs,this}return ue(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=pi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=pi)},persist:function(){},isPersistent:pi}),t}var nr={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Ba=Ye(nr),ei=ue({},nr,{view:0,detail:0}),Bh=Ye(ei),Bl,Ul,dr,gl=ue({},ei,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:Ua,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==dr&&(dr&&e.type==="mousemove"?(Bl=e.screenX-dr.screenX,Ul=e.screenY-dr.screenY):Ul=Bl=0,dr=e),Bl)},movementY:function(e){return"movementY"in e?e.movementY:Ul}}),eu=Ye(gl),Uh=ue({},gl,{dataTransfer:0}),Hh=Ye(Uh),Vh=ue({},ei,{relatedTarget:0}),Hl=Ye(Vh),$h=ue({},nr,{animationName:0,elapsedTime:0,pseudoElement:0}),Wh=Ye($h),Qh=ue({},nr,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),Kh=Ye(Qh),qh=ue({},nr,{data:0}),tu=Ye(qh),Yh={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},Xh={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},Gh={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function Jh(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=Gh[e])?!!t[e]:!1}function Ua(){return Jh}var Zh=ue({},ei,{key:function(e){if(e.key){var t=Yh[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Ti(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?Xh[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:Ua,charCode:function(e){return e.type==="keypress"?Ti(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Ti(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),em=Ye(Zh),tm=ue({},gl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),nu=Ye(tm),nm=ue({},ei,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:Ua}),rm=Ye(nm),im=ue({},nr,{propertyName:0,elapsedTime:0,pseudoElement:0}),lm=Ye(im),om=ue({},gl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),am=Ye(om),sm=[9,13,27,32],Ha=Pt&&"CompositionEvent"in window,Er=null;Pt&&"documentMode"in document&&(Er=document.documentMode);var um=Pt&&"TextEvent"in window&&!Er,Pf=Pt&&(!Ha||Er&&8<Er&&11>=Er),ru=" ",iu=!1;function Tf(e,t){switch(e){case"keyup":return sm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Lf(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Pn=!1;function cm(e,t){switch(e){case"compositionend":return Lf(t);case"keypress":return t.which!==32?null:(iu=!0,ru);case"textInput":return e=t.data,e===ru&&iu?null:e;default:return null}}function fm(e,t){if(Pn)return e==="compositionend"||!Ha&&Tf(e,t)?(e=zf(),Pi=Fa=Ht=null,Pn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Pf&&t.locale!=="ko"?null:t.data;default:return null}}var dm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function lu(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!dm[e.type]:t==="textarea"}function If(e,t,n,r){cf(r),t=Yi(t,"onChange"),0<t.length&&(n=new Ba("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Nr=null,Br=null;function pm(e){$f(e,0)}function yl(e){var t=In(e);if(nf(t))return e}function hm(e,t){if(e==="change")return t}var Mf=!1;if(Pt){var Vl;if(Pt){var $l="oninput"in document;if(!$l){var ou=document.createElement("div");ou.setAttribute("oninput","return;"),$l=typeof ou.oninput=="function"}Vl=$l}else Vl=!1;Mf=Vl&&(!document.documentMode||9<document.documentMode)}function au(){Nr&&(Nr.detachEvent("onpropertychange",Af),Br=Nr=null)}function Af(e){if(e.propertyName==="value"&&yl(Br)){var t=[];If(t,Br,e,Ma(e)),hf(pm,t)}}function mm(e,t,n){e==="focusin"?(au(),Nr=t,Br=n,Nr.attachEvent("onpropertychange",Af)):e==="focusout"&&au()}function gm(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return yl(Br)}function ym(e,t){if(e==="click")return yl(t)}function vm(e,t){if(e==="input"||e==="change")return yl(t)}function xm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var ft=typeof Object.is=="function"?Object.is:xm;function Ur(e,t){if(ft(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!ko.call(t,i)||!ft(e[i],t[i]))return!1}return!0}function su(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function uu(e,t){var n=su(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=su(n)}}function Of(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Of(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Df(){for(var e=window,t=Hi();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Hi(e.document)}return t}function Va(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function km(e){var t=Df(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Of(n.ownerDocument.documentElement,n)){if(r!==null&&Va(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=uu(n,l);var o=uu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var wm=Pt&&"documentMode"in document&&11>=document.documentMode,Tn=null,Fo=null,_r=null,Bo=!1;function cu(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Bo||Tn==null||Tn!==Hi(r)||(r=Tn,"selectionStart"in r&&Va(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),_r&&Ur(_r,r)||(_r=r,r=Yi(Fo,"onSelect"),0<r.length&&(t=new Ba("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Tn)))}function hi(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Ln={animationend:hi("Animation","AnimationEnd"),animationiteration:hi("Animation","AnimationIteration"),animationstart:hi("Animation","AnimationStart"),transitionend:hi("Transition","TransitionEnd")},Wl={},Rf={};Pt&&(Rf=document.createElement("div").style,"AnimationEvent"in window||(delete Ln.animationend.animation,delete Ln.animationiteration.animation,delete Ln.animationstart.animation),"TransitionEvent"in window||delete Ln.transitionend.transition);function vl(e){if(Wl[e])return Wl[e];if(!Ln[e])return e;var t=Ln[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Rf)return Wl[e]=t[n];return e}var Ff=vl("animationend"),Bf=vl("animationiteration"),Uf=vl("animationstart"),Hf=vl("transitionend"),Vf=new Map,fu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function en(e,t){Vf.set(e,t),kn(t,[e])}for(var Ql=0;Ql<fu.length;Ql++){var Kl=fu[Ql],Sm=Kl.toLowerCase(),Cm=Kl[0].toUpperCase()+Kl.slice(1);en(Sm,"on"+Cm)}en(Ff,"onAnimationEnd");en(Bf,"onAnimationIteration");en(Uf,"onAnimationStart");en("dblclick","onDoubleClick");en("focusin","onFocus");en("focusout","onBlur");en(Hf,"onTransitionEnd");qn("onMouseEnter",["mouseout","mouseover"]);qn("onMouseLeave",["mouseout","mouseover"]);qn("onPointerEnter",["pointerout","pointerover"]);qn("onPointerLeave",["pointerout","pointerover"]);kn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));kn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));kn("onBeforeInput",["compositionend","keypress","textInput","paste"]);kn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));kn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));kn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var wr="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Em=new Set("cancel close invalid load scroll toggle".split(" ").concat(wr));function du(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,Sh(r,t,void 0,e),e.currentTarget=null}function $f(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,u=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;du(i,a,u),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,u=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;du(i,a,u),l=s}}}if($i)throw e=Ao,$i=!1,Ao=null,e}function re(e,t){var n=t[Wo];n===void 0&&(n=t[Wo]=new Set);var r=e+"__bubble";n.has(r)||(Wf(t,e,2,!1),n.add(r))}function ql(e,t,n){var r=0;t&&(r|=4),Wf(n,e,r,t)}var mi="_reactListening"+Math.random().toString(36).slice(2);function Hr(e){if(!e[mi]){e[mi]=!0,Gc.forEach(function(n){n!=="selectionchange"&&(Em.has(n)||ql(n,!1,e),ql(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[mi]||(t[mi]=!0,ql("selectionchange",!1,t))}}function Wf(e,t,n,r){switch(bf(t)){case 1:var i=Rh;break;case 4:i=Fh;break;default:i=Ra}n=i.bind(null,t,n,e),i=void 0,!Mo||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function Yl(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=cn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}hf(function(){var u=l,c=Ma(n),d=[];e:{var p=Vf.get(e);if(p!==void 0){var f=Ba,k=e;switch(e){case"keypress":if(Ti(n)===0)break e;case"keydown":case"keyup":f=em;break;case"focusin":k="focus",f=Hl;break;case"focusout":k="blur",f=Hl;break;case"beforeblur":case"afterblur":f=Hl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":f=eu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":f=Hh;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":f=rm;break;case Ff:case Bf:case Uf:f=Wh;break;case Hf:f=lm;break;case"scroll":f=Bh;break;case"wheel":f=am;break;case"copy":case"cut":case"paste":f=Kh;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":f=nu}var C=(t&4)!==0,N=!C&&e==="scroll",m=C?p!==null?p+"Capture":null:p;C=[];for(var y=u,g;y!==null;){g=y;var S=g.stateNode;if(g.tag===5&&S!==null&&(g=S,m!==null&&(S=Or(y,m),S!=null&&C.push(Vr(y,S,g)))),N)break;y=y.return}0<C.length&&(p=new f(p,k,null,n,c),d.push({event:p,listeners:C}))}}if(!(t&7)){e:{if(p=e==="mouseover"||e==="pointerover",f=e==="mouseout"||e==="pointerout",p&&n!==Lo&&(k=n.relatedTarget||n.fromElement)&&(cn(k)||k[Tt]))break e;if((f||p)&&(p=c.window===c?c:(p=c.ownerDocument)?p.defaultView||p.parentWindow:window,f?(k=n.relatedTarget||n.toElement,f=u,k=k?cn(k):null,k!==null&&(N=wn(k),k!==N||k.tag!==5&&k.tag!==6)&&(k=null)):(f=null,k=u),f!==k)){if(C=eu,S="onMouseLeave",m="onMouseEnter",y="mouse",(e==="pointerout"||e==="pointerover")&&(C=nu,S="onPointerLeave",m="onPointerEnter",y="pointer"),N=f==null?p:In(f),g=k==null?p:In(k),p=new C(S,y+"leave",f,n,c),p.target=N,p.relatedTarget=g,S=null,cn(c)===u&&(C=new C(m,y+"enter",k,n,c),C.target=g,C.relatedTarget=N,S=C),N=S,f&&k)t:{for(C=f,m=k,y=0,g=C;g;g=_n(g))y++;for(g=0,S=m;S;S=_n(S))g++;for(;0<y-g;)C=_n(C),y--;for(;0<g-y;)m=_n(m),g--;for(;y--;){if(C===m||m!==null&&C===m.alternate)break t;C=_n(C),m=_n(m)}C=null}else C=null;f!==null&&pu(d,p,f,C,!1),k!==null&&N!==null&&pu(d,N,k,C,!0)}}e:{if(p=u?In(u):window,f=p.nodeName&&p.nodeName.toLowerCase(),f==="select"||f==="input"&&p.type==="file")var E=hm;else if(lu(p))if(Mf)E=vm;else{E=gm;var w=mm}else(f=p.nodeName)&&f.toLowerCase()==="input"&&(p.type==="checkbox"||p.type==="radio")&&(E=ym);if(E&&(E=E(e,u))){If(d,E,n,c);break e}w&&w(e,p,u),e==="focusout"&&(w=p._wrapperState)&&w.controlled&&p.type==="number"&&jo(p,"number",p.value)}switch(w=u?In(u):window,e){case"focusin":(lu(w)||w.contentEditable==="true")&&(Tn=w,Fo=u,_r=null);break;case"focusout":_r=Fo=Tn=null;break;case"mousedown":Bo=!0;break;case"contextmenu":case"mouseup":case"dragend":Bo=!1,cu(d,n,c);break;case"selectionchange":if(wm)break;case"keydown":case"keyup":cu(d,n,c)}var _;if(Ha)e:{switch(e){case"compositionstart":var P="onCompositionStart";break e;case"compositionend":P="onCompositionEnd";break e;case"compositionupdate":P="onCompositionUpdate";break e}P=void 0}else Pn?Tf(e,n)&&(P="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(P="onCompositionStart");P&&(Pf&&n.locale!=="ko"&&(Pn||P!=="onCompositionStart"?P==="onCompositionEnd"&&Pn&&(_=zf()):(Ht=c,Fa="value"in Ht?Ht.value:Ht.textContent,Pn=!0)),w=Yi(u,P),0<w.length&&(P=new tu(P,e,null,n,c),d.push({event:P,listeners:w}),_?P.data=_:(_=Lf(n),_!==null&&(P.data=_)))),(_=um?cm(e,n):fm(e,n))&&(u=Yi(u,"onBeforeInput"),0<u.length&&(c=new tu("onBeforeInput","beforeinput",null,n,c),d.push({event:c,listeners:u}),c.data=_))}$f(d,t)})}function Vr(e,t,n){return{instance:e,listener:t,currentTarget:n}}function Yi(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Or(e,n),l!=null&&r.unshift(Vr(e,l,i)),l=Or(e,t),l!=null&&r.push(Vr(e,l,i))),e=e.return}return r}function _n(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function pu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,u=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&u!==null&&(a=u,i?(s=Or(n,l),s!=null&&o.unshift(Vr(n,s,a))):i||(s=Or(n,l),s!=null&&o.push(Vr(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Nm=/\r\n?/g,_m=/\u0000|\uFFFD/g;function hu(e){return(typeof e=="string"?e:""+e).replace(Nm,`
`).replace(_m,"")}function gi(e,t,n){if(t=hu(t),hu(e)!==t&&n)throw Error(b(425))}function Xi(){}var Uo=null,Ho=null;function Vo(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var $o=typeof setTimeout=="function"?setTimeout:void 0,jm=typeof clearTimeout=="function"?clearTimeout:void 0,mu=typeof Promise=="function"?Promise:void 0,bm=typeof queueMicrotask=="function"?queueMicrotask:typeof mu<"u"?function(e){return mu.resolve(null).then(e).catch(zm)}:$o;function zm(e){setTimeout(function(){throw e})}function Xl(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Fr(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Fr(t)}function Kt(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function gu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var rr=Math.random().toString(36).slice(2),gt="__reactFiber$"+rr,$r="__reactProps$"+rr,Tt="__reactContainer$"+rr,Wo="__reactEvents$"+rr,Pm="__reactListeners$"+rr,Tm="__reactHandles$"+rr;function cn(e){var t=e[gt];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Tt]||n[gt]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=gu(e);e!==null;){if(n=e[gt])return n;e=gu(e)}return t}e=n,n=e.parentNode}return null}function ti(e){return e=e[gt]||e[Tt],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function In(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(b(33))}function xl(e){return e[$r]||null}var Qo=[],Mn=-1;function tn(e){return{current:e}}function ie(e){0>Mn||(e.current=Qo[Mn],Qo[Mn]=null,Mn--)}function ee(e,t){Mn++,Qo[Mn]=e.current,e.current=t}var Zt={},je=tn(Zt),Oe=tn(!1),mn=Zt;function Yn(e,t){var n=e.type.contextTypes;if(!n)return Zt;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function De(e){return e=e.childContextTypes,e!=null}function Gi(){ie(Oe),ie(je)}function yu(e,t,n){if(je.current!==Zt)throw Error(b(168));ee(je,t),ee(Oe,n)}function Qf(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(b(108,mh(e)||"Unknown",i));return ue({},n,r)}function Ji(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||Zt,mn=je.current,ee(je,e),ee(Oe,Oe.current),!0}function vu(e,t,n){var r=e.stateNode;if(!r)throw Error(b(169));n?(e=Qf(e,t,mn),r.__reactInternalMemoizedMergedChildContext=e,ie(Oe),ie(je),ee(je,e)):ie(Oe),ee(Oe,n)}var Nt=null,kl=!1,Gl=!1;function Kf(e){Nt===null?Nt=[e]:Nt.push(e)}function Lm(e){kl=!0,Kf(e)}function nn(){if(!Gl&&Nt!==null){Gl=!0;var e=0,t=X;try{var n=Nt;for(X=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}Nt=null,kl=!1}catch(i){throw Nt!==null&&(Nt=Nt.slice(e+1)),vf(Aa,nn),i}finally{X=t,Gl=!1}}return null}var An=[],On=0,Zi=null,el=0,Xe=[],Ge=0,gn=null,jt=1,bt="";function on(e,t){An[On++]=el,An[On++]=Zi,Zi=e,el=t}function qf(e,t,n){Xe[Ge++]=jt,Xe[Ge++]=bt,Xe[Ge++]=gn,gn=e;var r=jt;e=bt;var i=32-ut(r)-1;r&=~(1<<i),n+=1;var l=32-ut(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,jt=1<<32-ut(t)+i|n<<i|r,bt=l+e}else jt=1<<l|n<<i|r,bt=e}function $a(e){e.return!==null&&(on(e,1),qf(e,1,0))}function Wa(e){for(;e===Zi;)Zi=An[--On],An[On]=null,el=An[--On],An[On]=null;for(;e===gn;)gn=Xe[--Ge],Xe[Ge]=null,bt=Xe[--Ge],Xe[Ge]=null,jt=Xe[--Ge],Xe[Ge]=null}var Qe=null,$e=null,le=!1,st=null;function Yf(e,t){var n=Ze(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function xu(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,Qe=e,$e=Kt(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,Qe=e,$e=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=gn!==null?{id:jt,overflow:bt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=Ze(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,Qe=e,$e=null,!0):!1;default:return!1}}function Ko(e){return(e.mode&1)!==0&&(e.flags&128)===0}function qo(e){if(le){var t=$e;if(t){var n=t;if(!xu(e,t)){if(Ko(e))throw Error(b(418));t=Kt(n.nextSibling);var r=Qe;t&&xu(e,t)?Yf(r,n):(e.flags=e.flags&-4097|2,le=!1,Qe=e)}}else{if(Ko(e))throw Error(b(418));e.flags=e.flags&-4097|2,le=!1,Qe=e}}}function ku(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;Qe=e}function yi(e){if(e!==Qe)return!1;if(!le)return ku(e),le=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!Vo(e.type,e.memoizedProps)),t&&(t=$e)){if(Ko(e))throw Xf(),Error(b(418));for(;t;)Yf(e,t),t=Kt(t.nextSibling)}if(ku(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(b(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){$e=Kt(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}$e=null}}else $e=Qe?Kt(e.stateNode.nextSibling):null;return!0}function Xf(){for(var e=$e;e;)e=Kt(e.nextSibling)}function Xn(){$e=Qe=null,le=!1}function Qa(e){st===null?st=[e]:st.push(e)}var Im=Mt.ReactCurrentBatchConfig;function pr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(b(309));var r=n.stateNode}if(!r)throw Error(b(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(b(284));if(!n._owner)throw Error(b(290,e))}return e}function vi(e,t){throw e=Object.prototype.toString.call(t),Error(b(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function wu(e){var t=e._init;return t(e._payload)}function Gf(e){function t(m,y){if(e){var g=m.deletions;g===null?(m.deletions=[y],m.flags|=16):g.push(y)}}function n(m,y){if(!e)return null;for(;y!==null;)t(m,y),y=y.sibling;return null}function r(m,y){for(m=new Map;y!==null;)y.key!==null?m.set(y.key,y):m.set(y.index,y),y=y.sibling;return m}function i(m,y){return m=Gt(m,y),m.index=0,m.sibling=null,m}function l(m,y,g){return m.index=g,e?(g=m.alternate,g!==null?(g=g.index,g<y?(m.flags|=2,y):g):(m.flags|=2,y)):(m.flags|=1048576,y)}function o(m){return e&&m.alternate===null&&(m.flags|=2),m}function a(m,y,g,S){return y===null||y.tag!==6?(y=io(g,m.mode,S),y.return=m,y):(y=i(y,g),y.return=m,y)}function s(m,y,g,S){var E=g.type;return E===zn?c(m,y,g.props.children,S,g.key):y!==null&&(y.elementType===E||typeof E=="object"&&E!==null&&E.$$typeof===Rt&&wu(E)===y.type)?(S=i(y,g.props),S.ref=pr(m,y,g),S.return=m,S):(S=Ri(g.type,g.key,g.props,null,m.mode,S),S.ref=pr(m,y,g),S.return=m,S)}function u(m,y,g,S){return y===null||y.tag!==4||y.stateNode.containerInfo!==g.containerInfo||y.stateNode.implementation!==g.implementation?(y=lo(g,m.mode,S),y.return=m,y):(y=i(y,g.children||[]),y.return=m,y)}function c(m,y,g,S,E){return y===null||y.tag!==7?(y=hn(g,m.mode,S,E),y.return=m,y):(y=i(y,g),y.return=m,y)}function d(m,y,g){if(typeof y=="string"&&y!==""||typeof y=="number")return y=io(""+y,m.mode,g),y.return=m,y;if(typeof y=="object"&&y!==null){switch(y.$$typeof){case ai:return g=Ri(y.type,y.key,y.props,null,m.mode,g),g.ref=pr(m,null,y),g.return=m,g;case bn:return y=lo(y,m.mode,g),y.return=m,y;case Rt:var S=y._init;return d(m,S(y._payload),g)}if(xr(y)||sr(y))return y=hn(y,m.mode,g,null),y.return=m,y;vi(m,y)}return null}function p(m,y,g,S){var E=y!==null?y.key:null;if(typeof g=="string"&&g!==""||typeof g=="number")return E!==null?null:a(m,y,""+g,S);if(typeof g=="object"&&g!==null){switch(g.$$typeof){case ai:return g.key===E?s(m,y,g,S):null;case bn:return g.key===E?u(m,y,g,S):null;case Rt:return E=g._init,p(m,y,E(g._payload),S)}if(xr(g)||sr(g))return E!==null?null:c(m,y,g,S,null);vi(m,g)}return null}function f(m,y,g,S,E){if(typeof S=="string"&&S!==""||typeof S=="number")return m=m.get(g)||null,a(y,m,""+S,E);if(typeof S=="object"&&S!==null){switch(S.$$typeof){case ai:return m=m.get(S.key===null?g:S.key)||null,s(y,m,S,E);case bn:return m=m.get(S.key===null?g:S.key)||null,u(y,m,S,E);case Rt:var w=S._init;return f(m,y,g,w(S._payload),E)}if(xr(S)||sr(S))return m=m.get(g)||null,c(y,m,S,E,null);vi(y,S)}return null}function k(m,y,g,S){for(var E=null,w=null,_=y,P=y=0,O=null;_!==null&&P<g.length;P++){_.index>P?(O=_,_=null):O=_.sibling;var M=p(m,_,g[P],S);if(M===null){_===null&&(_=O);break}e&&_&&M.alternate===null&&t(m,_),y=l(M,y,P),w===null?E=M:w.sibling=M,w=M,_=O}if(P===g.length)return n(m,_),le&&on(m,P),E;if(_===null){for(;P<g.length;P++)_=d(m,g[P],S),_!==null&&(y=l(_,y,P),w===null?E=_:w.sibling=_,w=_);return le&&on(m,P),E}for(_=r(m,_);P<g.length;P++)O=f(_,m,P,g[P],S),O!==null&&(e&&O.alternate!==null&&_.delete(O.key===null?P:O.key),y=l(O,y,P),w===null?E=O:w.sibling=O,w=O);return e&&_.forEach(function(A){return t(m,A)}),le&&on(m,P),E}function C(m,y,g,S){var E=sr(g);if(typeof E!="function")throw Error(b(150));if(g=E.call(g),g==null)throw Error(b(151));for(var w=E=null,_=y,P=y=0,O=null,M=g.next();_!==null&&!M.done;P++,M=g.next()){_.index>P?(O=_,_=null):O=_.sibling;var A=p(m,_,M.value,S);if(A===null){_===null&&(_=O);break}e&&_&&A.alternate===null&&t(m,_),y=l(A,y,P),w===null?E=A:w.sibling=A,w=A,_=O}if(M.done)return n(m,_),le&&on(m,P),E;if(_===null){for(;!M.done;P++,M=g.next())M=d(m,M.value,S),M!==null&&(y=l(M,y,P),w===null?E=M:w.sibling=M,w=M);return le&&on(m,P),E}for(_=r(m,_);!M.done;P++,M=g.next())M=f(_,m,P,M.value,S),M!==null&&(e&&M.alternate!==null&&_.delete(M.key===null?P:M.key),y=l(M,y,P),w===null?E=M:w.sibling=M,w=M);return e&&_.forEach(function(D){return t(m,D)}),le&&on(m,P),E}function N(m,y,g,S){if(typeof g=="object"&&g!==null&&g.type===zn&&g.key===null&&(g=g.props.children),typeof g=="object"&&g!==null){switch(g.$$typeof){case ai:e:{for(var E=g.key,w=y;w!==null;){if(w.key===E){if(E=g.type,E===zn){if(w.tag===7){n(m,w.sibling),y=i(w,g.props.children),y.return=m,m=y;break e}}else if(w.elementType===E||typeof E=="object"&&E!==null&&E.$$typeof===Rt&&wu(E)===w.type){n(m,w.sibling),y=i(w,g.props),y.ref=pr(m,w,g),y.return=m,m=y;break e}n(m,w);break}else t(m,w);w=w.sibling}g.type===zn?(y=hn(g.props.children,m.mode,S,g.key),y.return=m,m=y):(S=Ri(g.type,g.key,g.props,null,m.mode,S),S.ref=pr(m,y,g),S.return=m,m=S)}return o(m);case bn:e:{for(w=g.key;y!==null;){if(y.key===w)if(y.tag===4&&y.stateNode.containerInfo===g.containerInfo&&y.stateNode.implementation===g.implementation){n(m,y.sibling),y=i(y,g.children||[]),y.return=m,m=y;break e}else{n(m,y);break}else t(m,y);y=y.sibling}y=lo(g,m.mode,S),y.return=m,m=y}return o(m);case Rt:return w=g._init,N(m,y,w(g._payload),S)}if(xr(g))return k(m,y,g,S);if(sr(g))return C(m,y,g,S);vi(m,g)}return typeof g=="string"&&g!==""||typeof g=="number"?(g=""+g,y!==null&&y.tag===6?(n(m,y.sibling),y=i(y,g),y.return=m,m=y):(n(m,y),y=io(g,m.mode,S),y.return=m,m=y),o(m)):n(m,y)}return N}var Gn=Gf(!0),Jf=Gf(!1),tl=tn(null),nl=null,Dn=null,Ka=null;function qa(){Ka=Dn=nl=null}function Ya(e){var t=tl.current;ie(tl),e._currentValue=t}function Yo(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function $n(e,t){nl=e,Ka=Dn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(Ae=!0),e.firstContext=null)}function tt(e){var t=e._currentValue;if(Ka!==e)if(e={context:e,memoizedValue:t,next:null},Dn===null){if(nl===null)throw Error(b(308));Dn=e,nl.dependencies={lanes:0,firstContext:e}}else Dn=Dn.next=e;return t}var fn=null;function Xa(e){fn===null?fn=[e]:fn.push(e)}function Zf(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,Xa(t)):(n.next=i.next,i.next=n),t.interleaved=n,Lt(e,r)}function Lt(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Ft=!1;function Ga(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function ed(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function zt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function qt(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,K&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Lt(e,n)}return i=r.interleaved,i===null?(t.next=t,Xa(r)):(t.next=i.next,i.next=t),r.interleaved=t,Lt(e,n)}function Li(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Oa(e,n)}}function Su(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function rl(e,t,n,r){var i=e.updateQueue;Ft=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,u=s.next;s.next=null,o===null?l=u:o.next=u,o=s;var c=e.alternate;c!==null&&(c=c.updateQueue,a=c.lastBaseUpdate,a!==o&&(a===null?c.firstBaseUpdate=u:a.next=u,c.lastBaseUpdate=s))}if(l!==null){var d=i.baseState;o=0,c=u=s=null,a=l;do{var p=a.lane,f=a.eventTime;if((r&p)===p){c!==null&&(c=c.next={eventTime:f,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var k=e,C=a;switch(p=t,f=n,C.tag){case 1:if(k=C.payload,typeof k=="function"){d=k.call(f,d,p);break e}d=k;break e;case 3:k.flags=k.flags&-65537|128;case 0:if(k=C.payload,p=typeof k=="function"?k.call(f,d,p):k,p==null)break e;d=ue({},d,p);break e;case 2:Ft=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,p=i.effects,p===null?i.effects=[a]:p.push(a))}else f={eventTime:f,lane:p,tag:a.tag,payload:a.payload,callback:a.callback,next:null},c===null?(u=c=f,s=d):c=c.next=f,o|=p;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;p=a,a=p.next,p.next=null,i.lastBaseUpdate=p,i.shared.pending=null}}while(!0);if(c===null&&(s=d),i.baseState=s,i.firstBaseUpdate=u,i.lastBaseUpdate=c,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);vn|=o,e.lanes=o,e.memoizedState=d}}function Cu(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(b(191,i));i.call(r)}}}var ni={},xt=tn(ni),Wr=tn(ni),Qr=tn(ni);function dn(e){if(e===ni)throw Error(b(174));return e}function Ja(e,t){switch(ee(Qr,t),ee(Wr,e),ee(xt,ni),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:zo(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=zo(t,e)}ie(xt),ee(xt,t)}function Jn(){ie(xt),ie(Wr),ie(Qr)}function td(e){dn(Qr.current);var t=dn(xt.current),n=zo(t,e.type);t!==n&&(ee(Wr,e),ee(xt,n))}function Za(e){Wr.current===e&&(ie(xt),ie(Wr))}var ae=tn(0);function il(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var Jl=[];function es(){for(var e=0;e<Jl.length;e++)Jl[e]._workInProgressVersionPrimary=null;Jl.length=0}var Ii=Mt.ReactCurrentDispatcher,Zl=Mt.ReactCurrentBatchConfig,yn=0,se=null,ye=null,xe=null,ll=!1,jr=!1,Kr=0,Mm=0;function Ee(){throw Error(b(321))}function ts(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!ft(e[n],t[n]))return!1;return!0}function ns(e,t,n,r,i,l){if(yn=l,se=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Ii.current=e===null||e.memoizedState===null?Rm:Fm,e=n(r,i),jr){l=0;do{if(jr=!1,Kr=0,25<=l)throw Error(b(301));l+=1,xe=ye=null,t.updateQueue=null,Ii.current=Bm,e=n(r,i)}while(jr)}if(Ii.current=ol,t=ye!==null&&ye.next!==null,yn=0,xe=ye=se=null,ll=!1,t)throw Error(b(300));return e}function rs(){var e=Kr!==0;return Kr=0,e}function ht(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return xe===null?se.memoizedState=xe=e:xe=xe.next=e,xe}function nt(){if(ye===null){var e=se.alternate;e=e!==null?e.memoizedState:null}else e=ye.next;var t=xe===null?se.memoizedState:xe.next;if(t!==null)xe=t,ye=e;else{if(e===null)throw Error(b(310));ye=e,e={memoizedState:ye.memoizedState,baseState:ye.baseState,baseQueue:ye.baseQueue,queue:ye.queue,next:null},xe===null?se.memoizedState=xe=e:xe=xe.next=e}return xe}function qr(e,t){return typeof t=="function"?t(e):t}function eo(e){var t=nt(),n=t.queue;if(n===null)throw Error(b(311));n.lastRenderedReducer=e;var r=ye,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,u=l;do{var c=u.lane;if((yn&c)===c)s!==null&&(s=s.next={lane:0,action:u.action,hasEagerState:u.hasEagerState,eagerState:u.eagerState,next:null}),r=u.hasEagerState?u.eagerState:e(r,u.action);else{var d={lane:c,action:u.action,hasEagerState:u.hasEagerState,eagerState:u.eagerState,next:null};s===null?(a=s=d,o=r):s=s.next=d,se.lanes|=c,vn|=c}u=u.next}while(u!==null&&u!==l);s===null?o=r:s.next=a,ft(r,t.memoizedState)||(Ae=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,se.lanes|=l,vn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function to(e){var t=nt(),n=t.queue;if(n===null)throw Error(b(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);ft(l,t.memoizedState)||(Ae=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function nd(){}function rd(e,t){var n=se,r=nt(),i=t(),l=!ft(r.memoizedState,i);if(l&&(r.memoizedState=i,Ae=!0),r=r.queue,is(od.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||xe!==null&&xe.memoizedState.tag&1){if(n.flags|=2048,Yr(9,ld.bind(null,n,r,i,t),void 0,null),ke===null)throw Error(b(349));yn&30||id(n,t,i)}return i}function id(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=se.updateQueue,t===null?(t={lastEffect:null,stores:null},se.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function ld(e,t,n,r){t.value=n,t.getSnapshot=r,ad(t)&&sd(e)}function od(e,t,n){return n(function(){ad(t)&&sd(e)})}function ad(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!ft(e,n)}catch{return!0}}function sd(e){var t=Lt(e,1);t!==null&&ct(t,e,1,-1)}function Eu(e){var t=ht();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:qr,lastRenderedState:e},t.queue=e,e=e.dispatch=Dm.bind(null,se,e),[t.memoizedState,e]}function Yr(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=se.updateQueue,t===null?(t={lastEffect:null,stores:null},se.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function ud(){return nt().memoizedState}function Mi(e,t,n,r){var i=ht();se.flags|=e,i.memoizedState=Yr(1|t,n,void 0,r===void 0?null:r)}function wl(e,t,n,r){var i=nt();r=r===void 0?null:r;var l=void 0;if(ye!==null){var o=ye.memoizedState;if(l=o.destroy,r!==null&&ts(r,o.deps)){i.memoizedState=Yr(t,n,l,r);return}}se.flags|=e,i.memoizedState=Yr(1|t,n,l,r)}function Nu(e,t){return Mi(8390656,8,e,t)}function is(e,t){return wl(2048,8,e,t)}function cd(e,t){return wl(4,2,e,t)}function fd(e,t){return wl(4,4,e,t)}function dd(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function pd(e,t,n){return n=n!=null?n.concat([e]):null,wl(4,4,dd.bind(null,t,e),n)}function ls(){}function hd(e,t){var n=nt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ts(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function md(e,t){var n=nt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ts(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function gd(e,t,n){return yn&21?(ft(n,t)||(n=wf(),se.lanes|=n,vn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,Ae=!0),e.memoizedState=n)}function Am(e,t){var n=X;X=n!==0&&4>n?n:4,e(!0);var r=Zl.transition;Zl.transition={};try{e(!1),t()}finally{X=n,Zl.transition=r}}function yd(){return nt().memoizedState}function Om(e,t,n){var r=Xt(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},vd(e))xd(t,n);else if(n=Zf(e,t,n,r),n!==null){var i=Pe();ct(n,e,r,i),kd(n,t,r)}}function Dm(e,t,n){var r=Xt(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(vd(e))xd(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,ft(a,o)){var s=t.interleaved;s===null?(i.next=i,Xa(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=Zf(e,t,i,r),n!==null&&(i=Pe(),ct(n,e,r,i),kd(n,t,r))}}function vd(e){var t=e.alternate;return e===se||t!==null&&t===se}function xd(e,t){jr=ll=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function kd(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Oa(e,n)}}var ol={readContext:tt,useCallback:Ee,useContext:Ee,useEffect:Ee,useImperativeHandle:Ee,useInsertionEffect:Ee,useLayoutEffect:Ee,useMemo:Ee,useReducer:Ee,useRef:Ee,useState:Ee,useDebugValue:Ee,useDeferredValue:Ee,useTransition:Ee,useMutableSource:Ee,useSyncExternalStore:Ee,useId:Ee,unstable_isNewReconciler:!1},Rm={readContext:tt,useCallback:function(e,t){return ht().memoizedState=[e,t===void 0?null:t],e},useContext:tt,useEffect:Nu,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Mi(4194308,4,dd.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Mi(4194308,4,e,t)},useInsertionEffect:function(e,t){return Mi(4,2,e,t)},useMemo:function(e,t){var n=ht();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=ht();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Om.bind(null,se,e),[r.memoizedState,e]},useRef:function(e){var t=ht();return e={current:e},t.memoizedState=e},useState:Eu,useDebugValue:ls,useDeferredValue:function(e){return ht().memoizedState=e},useTransition:function(){var e=Eu(!1),t=e[0];return e=Am.bind(null,e[1]),ht().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=se,i=ht();if(le){if(n===void 0)throw Error(b(407));n=n()}else{if(n=t(),ke===null)throw Error(b(349));yn&30||id(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Nu(od.bind(null,r,l,e),[e]),r.flags|=2048,Yr(9,ld.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=ht(),t=ke.identifierPrefix;if(le){var n=bt,r=jt;n=(r&~(1<<32-ut(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=Kr++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Mm++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},Fm={readContext:tt,useCallback:hd,useContext:tt,useEffect:is,useImperativeHandle:pd,useInsertionEffect:cd,useLayoutEffect:fd,useMemo:md,useReducer:eo,useRef:ud,useState:function(){return eo(qr)},useDebugValue:ls,useDeferredValue:function(e){var t=nt();return gd(t,ye.memoizedState,e)},useTransition:function(){var e=eo(qr)[0],t=nt().memoizedState;return[e,t]},useMutableSource:nd,useSyncExternalStore:rd,useId:yd,unstable_isNewReconciler:!1},Bm={readContext:tt,useCallback:hd,useContext:tt,useEffect:is,useImperativeHandle:pd,useInsertionEffect:cd,useLayoutEffect:fd,useMemo:md,useReducer:to,useRef:ud,useState:function(){return to(qr)},useDebugValue:ls,useDeferredValue:function(e){var t=nt();return ye===null?t.memoizedState=e:gd(t,ye.memoizedState,e)},useTransition:function(){var e=to(qr)[0],t=nt().memoizedState;return[e,t]},useMutableSource:nd,useSyncExternalStore:rd,useId:yd,unstable_isNewReconciler:!1};function ot(e,t){if(e&&e.defaultProps){t=ue({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function Xo(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:ue({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var Sl={isMounted:function(e){return(e=e._reactInternals)?wn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Pe(),i=Xt(e),l=zt(r,i);l.payload=t,n!=null&&(l.callback=n),t=qt(e,l,i),t!==null&&(ct(t,e,i,r),Li(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Pe(),i=Xt(e),l=zt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=qt(e,l,i),t!==null&&(ct(t,e,i,r),Li(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Pe(),r=Xt(e),i=zt(n,r);i.tag=2,t!=null&&(i.callback=t),t=qt(e,i,r),t!==null&&(ct(t,e,r,n),Li(t,e,r))}};function _u(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Ur(n,r)||!Ur(i,l):!0}function wd(e,t,n){var r=!1,i=Zt,l=t.contextType;return typeof l=="object"&&l!==null?l=tt(l):(i=De(t)?mn:je.current,r=t.contextTypes,l=(r=r!=null)?Yn(e,i):Zt),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=Sl,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function ju(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&Sl.enqueueReplaceState(t,t.state,null)}function Go(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},Ga(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=tt(l):(l=De(t)?mn:je.current,i.context=Yn(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(Xo(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&Sl.enqueueReplaceState(i,i.state,null),rl(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function Zn(e,t){try{var n="",r=t;do n+=hh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function no(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function Jo(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var Um=typeof WeakMap=="function"?WeakMap:Map;function Sd(e,t,n){n=zt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){sl||(sl=!0,sa=r),Jo(e,t)},n}function Cd(e,t,n){n=zt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){Jo(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){Jo(e,t),typeof r!="function"&&(Yt===null?Yt=new Set([this]):Yt.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function bu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new Um;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=tg.bind(null,e,t,n),t.then(e,e))}function zu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Pu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=zt(-1,1),t.tag=2,qt(n,t,1))),n.lanes|=1),e)}var Hm=Mt.ReactCurrentOwner,Ae=!1;function ze(e,t,n,r){t.child=e===null?Jf(t,null,n,r):Gn(t,e.child,n,r)}function Tu(e,t,n,r,i){n=n.render;var l=t.ref;return $n(t,i),r=ns(e,t,n,r,l,i),n=rs(),e!==null&&!Ae?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,It(e,t,i)):(le&&n&&$a(t),t.flags|=1,ze(e,t,r,i),t.child)}function Lu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!ps(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,Ed(e,t,l,r,i)):(e=Ri(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Ur,n(o,r)&&e.ref===t.ref)return It(e,t,i)}return t.flags|=1,e=Gt(l,r),e.ref=t.ref,e.return=t,t.child=e}function Ed(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Ur(l,r)&&e.ref===t.ref)if(Ae=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(Ae=!0);else return t.lanes=e.lanes,It(e,t,i)}return Zo(e,t,n,r,i)}function Nd(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},ee(Fn,Ve),Ve|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,ee(Fn,Ve),Ve|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,ee(Fn,Ve),Ve|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,ee(Fn,Ve),Ve|=r;return ze(e,t,i,n),t.child}function _d(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function Zo(e,t,n,r,i){var l=De(n)?mn:je.current;return l=Yn(t,l),$n(t,i),n=ns(e,t,n,r,l,i),r=rs(),e!==null&&!Ae?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,It(e,t,i)):(le&&r&&$a(t),t.flags|=1,ze(e,t,n,i),t.child)}function Iu(e,t,n,r,i){if(De(n)){var l=!0;Ji(t)}else l=!1;if($n(t,i),t.stateNode===null)Ai(e,t),wd(t,n,r),Go(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,u=n.contextType;typeof u=="object"&&u!==null?u=tt(u):(u=De(n)?mn:je.current,u=Yn(t,u));var c=n.getDerivedStateFromProps,d=typeof c=="function"||typeof o.getSnapshotBeforeUpdate=="function";d||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==u)&&ju(t,o,r,u),Ft=!1;var p=t.memoizedState;o.state=p,rl(t,r,o,i),s=t.memoizedState,a!==r||p!==s||Oe.current||Ft?(typeof c=="function"&&(Xo(t,n,c,r),s=t.memoizedState),(a=Ft||_u(t,n,a,r,p,s,u))?(d||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=u,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,ed(e,t),a=t.memoizedProps,u=t.type===t.elementType?a:ot(t.type,a),o.props=u,d=t.pendingProps,p=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=tt(s):(s=De(n)?mn:je.current,s=Yn(t,s));var f=n.getDerivedStateFromProps;(c=typeof f=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==d||p!==s)&&ju(t,o,r,s),Ft=!1,p=t.memoizedState,o.state=p,rl(t,r,o,i);var k=t.memoizedState;a!==d||p!==k||Oe.current||Ft?(typeof f=="function"&&(Xo(t,n,f,r),k=t.memoizedState),(u=Ft||_u(t,n,u,r,p,k,s)||!1)?(c||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,k,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,k,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&p===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&p===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=k),o.props=r,o.state=k,o.context=s,r=u):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&p===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&p===e.memoizedState||(t.flags|=1024),r=!1)}return ea(e,t,n,r,l,i)}function ea(e,t,n,r,i,l){_d(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&vu(t,n,!1),It(e,t,l);r=t.stateNode,Hm.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=Gn(t,e.child,null,l),t.child=Gn(t,null,a,l)):ze(e,t,a,l),t.memoizedState=r.state,i&&vu(t,n,!0),t.child}function jd(e){var t=e.stateNode;t.pendingContext?yu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&yu(e,t.context,!1),Ja(e,t.containerInfo)}function Mu(e,t,n,r,i){return Xn(),Qa(i),t.flags|=256,ze(e,t,n,r),t.child}var ta={dehydrated:null,treeContext:null,retryLane:0};function na(e){return{baseLanes:e,cachePool:null,transitions:null}}function bd(e,t,n){var r=t.pendingProps,i=ae.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),ee(ae,i&1),e===null)return qo(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Nl(o,r,0,null),e=hn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=na(n),t.memoizedState=ta,e):os(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return Vm(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=Gt(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=Gt(a,l):(l=hn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?na(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=ta,r}return l=e.child,e=l.sibling,r=Gt(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function os(e,t){return t=Nl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function xi(e,t,n,r){return r!==null&&Qa(r),Gn(t,e.child,null,n),e=os(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function Vm(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=no(Error(b(422))),xi(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Nl({mode:"visible",children:r.children},i,0,null),l=hn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&Gn(t,e.child,null,o),t.child.memoizedState=na(o),t.memoizedState=ta,l);if(!(t.mode&1))return xi(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(b(419)),r=no(l,r,void 0),xi(e,t,o,r)}if(a=(o&e.childLanes)!==0,Ae||a){if(r=ke,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Lt(e,i),ct(r,e,i,-1))}return ds(),r=no(Error(b(421))),xi(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=ng.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,$e=Kt(i.nextSibling),Qe=t,le=!0,st=null,e!==null&&(Xe[Ge++]=jt,Xe[Ge++]=bt,Xe[Ge++]=gn,jt=e.id,bt=e.overflow,gn=t),t=os(t,r.children),t.flags|=4096,t)}function Au(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),Yo(e.return,t,n)}function ro(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function zd(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(ze(e,t,r.children,n),r=ae.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Au(e,n,t);else if(e.tag===19)Au(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(ee(ae,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&il(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),ro(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&il(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}ro(t,!0,n,null,l);break;case"together":ro(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Ai(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function It(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),vn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(b(153));if(t.child!==null){for(e=t.child,n=Gt(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=Gt(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function $m(e,t,n){switch(t.tag){case 3:jd(t),Xn();break;case 5:td(t);break;case 1:De(t.type)&&Ji(t);break;case 4:Ja(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;ee(tl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(ee(ae,ae.current&1),t.flags|=128,null):n&t.child.childLanes?bd(e,t,n):(ee(ae,ae.current&1),e=It(e,t,n),e!==null?e.sibling:null);ee(ae,ae.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return zd(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),ee(ae,ae.current),r)break;return null;case 22:case 23:return t.lanes=0,Nd(e,t,n)}return It(e,t,n)}var Pd,ra,Td,Ld;Pd=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};ra=function(){};Td=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,dn(xt.current);var l=null;switch(n){case"input":i=No(e,i),r=No(e,r),l=[];break;case"select":i=ue({},i,{value:void 0}),r=ue({},r,{value:void 0}),l=[];break;case"textarea":i=bo(e,i),r=bo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=Xi)}Po(n,r);var o;n=null;for(u in i)if(!r.hasOwnProperty(u)&&i.hasOwnProperty(u)&&i[u]!=null)if(u==="style"){var a=i[u];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else u!=="dangerouslySetInnerHTML"&&u!=="children"&&u!=="suppressContentEditableWarning"&&u!=="suppressHydrationWarning"&&u!=="autoFocus"&&(Mr.hasOwnProperty(u)?l||(l=[]):(l=l||[]).push(u,null));for(u in r){var s=r[u];if(a=i!=null?i[u]:void 0,r.hasOwnProperty(u)&&s!==a&&(s!=null||a!=null))if(u==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(u,n)),n=s;else u==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(u,s)):u==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(u,""+s):u!=="suppressContentEditableWarning"&&u!=="suppressHydrationWarning"&&(Mr.hasOwnProperty(u)?(s!=null&&u==="onScroll"&&re("scroll",e),l||a===s||(l=[])):(l=l||[]).push(u,s))}n&&(l=l||[]).push("style",n);var u=l;(t.updateQueue=u)&&(t.flags|=4)}};Ld=function(e,t,n,r){n!==r&&(t.flags|=4)};function hr(e,t){if(!le)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function Ne(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function Wm(e,t,n){var r=t.pendingProps;switch(Wa(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return Ne(t),null;case 1:return De(t.type)&&Gi(),Ne(t),null;case 3:return r=t.stateNode,Jn(),ie(Oe),ie(je),es(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(yi(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,st!==null&&(fa(st),st=null))),ra(e,t),Ne(t),null;case 5:Za(t);var i=dn(Qr.current);if(n=t.type,e!==null&&t.stateNode!=null)Td(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(b(166));return Ne(t),null}if(e=dn(xt.current),yi(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[gt]=t,r[$r]=l,e=(t.mode&1)!==0,n){case"dialog":re("cancel",r),re("close",r);break;case"iframe":case"object":case"embed":re("load",r);break;case"video":case"audio":for(i=0;i<wr.length;i++)re(wr[i],r);break;case"source":re("error",r);break;case"img":case"image":case"link":re("error",r),re("load",r);break;case"details":re("toggle",r);break;case"input":$s(r,l),re("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},re("invalid",r);break;case"textarea":Qs(r,l),re("invalid",r)}Po(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&gi(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&gi(r.textContent,a,e),i=["children",""+a]):Mr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&re("scroll",r)}switch(n){case"input":si(r),Ws(r,l,!0);break;case"textarea":si(r),Ks(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=Xi)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=of(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[gt]=t,e[$r]=r,Pd(e,t,!1,!1),t.stateNode=e;e:{switch(o=To(n,r),n){case"dialog":re("cancel",e),re("close",e),i=r;break;case"iframe":case"object":case"embed":re("load",e),i=r;break;case"video":case"audio":for(i=0;i<wr.length;i++)re(wr[i],e);i=r;break;case"source":re("error",e),i=r;break;case"img":case"image":case"link":re("error",e),re("load",e),i=r;break;case"details":re("toggle",e),i=r;break;case"input":$s(e,r),i=No(e,r),re("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=ue({},r,{value:void 0}),re("invalid",e);break;case"textarea":Qs(e,r),i=bo(e,r),re("invalid",e);break;default:i=r}Po(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?uf(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&af(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Ar(e,s):typeof s=="number"&&Ar(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Mr.hasOwnProperty(l)?s!=null&&l==="onScroll"&&re("scroll",e):s!=null&&Pa(e,l,s,o))}switch(n){case"input":si(e),Ws(e,r,!1);break;case"textarea":si(e),Ks(e);break;case"option":r.value!=null&&e.setAttribute("value",""+Jt(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Bn(e,!!r.multiple,l,!1):r.defaultValue!=null&&Bn(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=Xi)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return Ne(t),null;case 6:if(e&&t.stateNode!=null)Ld(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(b(166));if(n=dn(Qr.current),dn(xt.current),yi(t)){if(r=t.stateNode,n=t.memoizedProps,r[gt]=t,(l=r.nodeValue!==n)&&(e=Qe,e!==null))switch(e.tag){case 3:gi(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&gi(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[gt]=t,t.stateNode=r}return Ne(t),null;case 13:if(ie(ae),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(le&&$e!==null&&t.mode&1&&!(t.flags&128))Xf(),Xn(),t.flags|=98560,l=!1;else if(l=yi(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(b(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(b(317));l[gt]=t}else Xn(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;Ne(t),l=!1}else st!==null&&(fa(st),st=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||ae.current&1?ve===0&&(ve=3):ds())),t.updateQueue!==null&&(t.flags|=4),Ne(t),null);case 4:return Jn(),ra(e,t),e===null&&Hr(t.stateNode.containerInfo),Ne(t),null;case 10:return Ya(t.type._context),Ne(t),null;case 17:return De(t.type)&&Gi(),Ne(t),null;case 19:if(ie(ae),l=t.memoizedState,l===null)return Ne(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)hr(l,!1);else{if(ve!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=il(e),o!==null){for(t.flags|=128,hr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return ee(ae,ae.current&1|2),t.child}e=e.sibling}l.tail!==null&&de()>er&&(t.flags|=128,r=!0,hr(l,!1),t.lanes=4194304)}else{if(!r)if(e=il(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),hr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!le)return Ne(t),null}else 2*de()-l.renderingStartTime>er&&n!==1073741824&&(t.flags|=128,r=!0,hr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=de(),t.sibling=null,n=ae.current,ee(ae,r?n&1|2:n&1),t):(Ne(t),null);case 22:case 23:return fs(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?Ve&1073741824&&(Ne(t),t.subtreeFlags&6&&(t.flags|=8192)):Ne(t),null;case 24:return null;case 25:return null}throw Error(b(156,t.tag))}function Qm(e,t){switch(Wa(t),t.tag){case 1:return De(t.type)&&Gi(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return Jn(),ie(Oe),ie(je),es(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return Za(t),null;case 13:if(ie(ae),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(b(340));Xn()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return ie(ae),null;case 4:return Jn(),null;case 10:return Ya(t.type._context),null;case 22:case 23:return fs(),null;case 24:return null;default:return null}}var ki=!1,_e=!1,Km=typeof WeakSet=="function"?WeakSet:Set,I=null;function Rn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){ce(e,t,r)}else n.current=null}function ia(e,t,n){try{n()}catch(r){ce(e,t,r)}}var Ou=!1;function qm(e,t){if(Uo=Ki,e=Df(),Va(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,u=0,c=0,d=e,p=null;t:for(;;){for(var f;d!==n||i!==0&&d.nodeType!==3||(a=o+i),d!==l||r!==0&&d.nodeType!==3||(s=o+r),d.nodeType===3&&(o+=d.nodeValue.length),(f=d.firstChild)!==null;)p=d,d=f;for(;;){if(d===e)break t;if(p===n&&++u===i&&(a=o),p===l&&++c===r&&(s=o),(f=d.nextSibling)!==null)break;d=p,p=d.parentNode}d=f}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Ho={focusedElem:e,selectionRange:n},Ki=!1,I=t;I!==null;)if(t=I,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,I=e;else for(;I!==null;){t=I;try{var k=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(k!==null){var C=k.memoizedProps,N=k.memoizedState,m=t.stateNode,y=m.getSnapshotBeforeUpdate(t.elementType===t.type?C:ot(t.type,C),N);m.__reactInternalSnapshotBeforeUpdate=y}break;case 3:var g=t.stateNode.containerInfo;g.nodeType===1?g.textContent="":g.nodeType===9&&g.documentElement&&g.removeChild(g.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(b(163))}}catch(S){ce(t,t.return,S)}if(e=t.sibling,e!==null){e.return=t.return,I=e;break}I=t.return}return k=Ou,Ou=!1,k}function br(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&ia(t,n,l)}i=i.next}while(i!==r)}}function Cl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function la(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Id(e){var t=e.alternate;t!==null&&(e.alternate=null,Id(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[gt],delete t[$r],delete t[Wo],delete t[Pm],delete t[Tm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Md(e){return e.tag===5||e.tag===3||e.tag===4}function Du(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Md(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function oa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=Xi));else if(r!==4&&(e=e.child,e!==null))for(oa(e,t,n),e=e.sibling;e!==null;)oa(e,t,n),e=e.sibling}function aa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(aa(e,t,n),e=e.sibling;e!==null;)aa(e,t,n),e=e.sibling}var we=null,at=!1;function Ot(e,t,n){for(n=n.child;n!==null;)Ad(e,t,n),n=n.sibling}function Ad(e,t,n){if(vt&&typeof vt.onCommitFiberUnmount=="function")try{vt.onCommitFiberUnmount(ml,n)}catch{}switch(n.tag){case 5:_e||Rn(n,t);case 6:var r=we,i=at;we=null,Ot(e,t,n),we=r,at=i,we!==null&&(at?(e=we,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):we.removeChild(n.stateNode));break;case 18:we!==null&&(at?(e=we,n=n.stateNode,e.nodeType===8?Xl(e.parentNode,n):e.nodeType===1&&Xl(e,n),Fr(e)):Xl(we,n.stateNode));break;case 4:r=we,i=at,we=n.stateNode.containerInfo,at=!0,Ot(e,t,n),we=r,at=i;break;case 0:case 11:case 14:case 15:if(!_e&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&ia(n,t,o),i=i.next}while(i!==r)}Ot(e,t,n);break;case 1:if(!_e&&(Rn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){ce(n,t,a)}Ot(e,t,n);break;case 21:Ot(e,t,n);break;case 22:n.mode&1?(_e=(r=_e)||n.memoizedState!==null,Ot(e,t,n),_e=r):Ot(e,t,n);break;default:Ot(e,t,n)}}function Ru(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new Km),t.forEach(function(r){var i=rg.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function lt(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:we=a.stateNode,at=!1;break e;case 3:we=a.stateNode.containerInfo,at=!0;break e;case 4:we=a.stateNode.containerInfo,at=!0;break e}a=a.return}if(we===null)throw Error(b(160));Ad(l,o,i),we=null,at=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(u){ce(i,t,u)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Od(t,e),t=t.sibling}function Od(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(lt(t,e),pt(e),r&4){try{br(3,e,e.return),Cl(3,e)}catch(C){ce(e,e.return,C)}try{br(5,e,e.return)}catch(C){ce(e,e.return,C)}}break;case 1:lt(t,e),pt(e),r&512&&n!==null&&Rn(n,n.return);break;case 5:if(lt(t,e),pt(e),r&512&&n!==null&&Rn(n,n.return),e.flags&32){var i=e.stateNode;try{Ar(i,"")}catch(C){ce(e,e.return,C)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&rf(i,l),To(a,o);var u=To(a,l);for(o=0;o<s.length;o+=2){var c=s[o],d=s[o+1];c==="style"?uf(i,d):c==="dangerouslySetInnerHTML"?af(i,d):c==="children"?Ar(i,d):Pa(i,c,d,u)}switch(a){case"input":_o(i,l);break;case"textarea":lf(i,l);break;case"select":var p=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var f=l.value;f!=null?Bn(i,!!l.multiple,f,!1):p!==!!l.multiple&&(l.defaultValue!=null?Bn(i,!!l.multiple,l.defaultValue,!0):Bn(i,!!l.multiple,l.multiple?[]:"",!1))}i[$r]=l}catch(C){ce(e,e.return,C)}}break;case 6:if(lt(t,e),pt(e),r&4){if(e.stateNode===null)throw Error(b(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(C){ce(e,e.return,C)}}break;case 3:if(lt(t,e),pt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Fr(t.containerInfo)}catch(C){ce(e,e.return,C)}break;case 4:lt(t,e),pt(e);break;case 13:lt(t,e),pt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(us=de())),r&4&&Ru(e);break;case 22:if(c=n!==null&&n.memoizedState!==null,e.mode&1?(_e=(u=_e)||c,lt(t,e),_e=u):lt(t,e),pt(e),r&8192){if(u=e.memoizedState!==null,(e.stateNode.isHidden=u)&&!c&&e.mode&1)for(I=e,c=e.child;c!==null;){for(d=I=c;I!==null;){switch(p=I,f=p.child,p.tag){case 0:case 11:case 14:case 15:br(4,p,p.return);break;case 1:Rn(p,p.return);var k=p.stateNode;if(typeof k.componentWillUnmount=="function"){r=p,n=p.return;try{t=r,k.props=t.memoizedProps,k.state=t.memoizedState,k.componentWillUnmount()}catch(C){ce(r,n,C)}}break;case 5:Rn(p,p.return);break;case 22:if(p.memoizedState!==null){Bu(d);continue}}f!==null?(f.return=p,I=f):Bu(d)}c=c.sibling}e:for(c=null,d=e;;){if(d.tag===5){if(c===null){c=d;try{i=d.stateNode,u?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=d.stateNode,s=d.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=sf("display",o))}catch(C){ce(e,e.return,C)}}}else if(d.tag===6){if(c===null)try{d.stateNode.nodeValue=u?"":d.memoizedProps}catch(C){ce(e,e.return,C)}}else if((d.tag!==22&&d.tag!==23||d.memoizedState===null||d===e)&&d.child!==null){d.child.return=d,d=d.child;continue}if(d===e)break e;for(;d.sibling===null;){if(d.return===null||d.return===e)break e;c===d&&(c=null),d=d.return}c===d&&(c=null),d.sibling.return=d.return,d=d.sibling}}break;case 19:lt(t,e),pt(e),r&4&&Ru(e);break;case 21:break;default:lt(t,e),pt(e)}}function pt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Md(n)){var r=n;break e}n=n.return}throw Error(b(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Ar(i,""),r.flags&=-33);var l=Du(e);aa(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Du(e);oa(e,a,o);break;default:throw Error(b(161))}}catch(s){ce(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function Ym(e,t,n){I=e,Dd(e)}function Dd(e,t,n){for(var r=(e.mode&1)!==0;I!==null;){var i=I,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||ki;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||_e;a=ki;var u=_e;if(ki=o,(_e=s)&&!u)for(I=i;I!==null;)o=I,s=o.child,o.tag===22&&o.memoizedState!==null?Uu(i):s!==null?(s.return=o,I=s):Uu(i);for(;l!==null;)I=l,Dd(l),l=l.sibling;I=i,ki=a,_e=u}Fu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,I=l):Fu(e)}}function Fu(e){for(;I!==null;){var t=I;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:_e||Cl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!_e)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:ot(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Cu(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Cu(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var u=t.alternate;if(u!==null){var c=u.memoizedState;if(c!==null){var d=c.dehydrated;d!==null&&Fr(d)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(b(163))}_e||t.flags&512&&la(t)}catch(p){ce(t,t.return,p)}}if(t===e){I=null;break}if(n=t.sibling,n!==null){n.return=t.return,I=n;break}I=t.return}}function Bu(e){for(;I!==null;){var t=I;if(t===e){I=null;break}var n=t.sibling;if(n!==null){n.return=t.return,I=n;break}I=t.return}}function Uu(e){for(;I!==null;){var t=I;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Cl(4,t)}catch(s){ce(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){ce(t,i,s)}}var l=t.return;try{la(t)}catch(s){ce(t,l,s)}break;case 5:var o=t.return;try{la(t)}catch(s){ce(t,o,s)}}}catch(s){ce(t,t.return,s)}if(t===e){I=null;break}var a=t.sibling;if(a!==null){a.return=t.return,I=a;break}I=t.return}}var Xm=Math.ceil,al=Mt.ReactCurrentDispatcher,as=Mt.ReactCurrentOwner,et=Mt.ReactCurrentBatchConfig,K=0,ke=null,me=null,Se=0,Ve=0,Fn=tn(0),ve=0,Xr=null,vn=0,El=0,ss=0,zr=null,Me=null,us=0,er=1/0,Et=null,sl=!1,sa=null,Yt=null,wi=!1,Vt=null,ul=0,Pr=0,ua=null,Oi=-1,Di=0;function Pe(){return K&6?de():Oi!==-1?Oi:Oi=de()}function Xt(e){return e.mode&1?K&2&&Se!==0?Se&-Se:Im.transition!==null?(Di===0&&(Di=wf()),Di):(e=X,e!==0||(e=window.event,e=e===void 0?16:bf(e.type)),e):1}function ct(e,t,n,r){if(50<Pr)throw Pr=0,ua=null,Error(b(185));Zr(e,n,r),(!(K&2)||e!==ke)&&(e===ke&&(!(K&2)&&(El|=n),ve===4&&Ut(e,Se)),Re(e,r),n===1&&K===0&&!(t.mode&1)&&(er=de()+500,kl&&nn()))}function Re(e,t){var n=e.callbackNode;Ih(e,t);var r=Qi(e,e===ke?Se:0);if(r===0)n!==null&&Xs(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&Xs(n),t===1)e.tag===0?Lm(Hu.bind(null,e)):Kf(Hu.bind(null,e)),bm(function(){!(K&6)&&nn()}),n=null;else{switch(Sf(r)){case 1:n=Aa;break;case 4:n=xf;break;case 16:n=Wi;break;case 536870912:n=kf;break;default:n=Wi}n=Wd(n,Rd.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function Rd(e,t){if(Oi=-1,Di=0,K&6)throw Error(b(327));var n=e.callbackNode;if(Wn()&&e.callbackNode!==n)return null;var r=Qi(e,e===ke?Se:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=cl(e,r);else{t=r;var i=K;K|=2;var l=Bd();(ke!==e||Se!==t)&&(Et=null,er=de()+500,pn(e,t));do try{Zm();break}catch(a){Fd(e,a)}while(!0);qa(),al.current=l,K=i,me!==null?t=0:(ke=null,Se=0,t=ve)}if(t!==0){if(t===2&&(i=Oo(e),i!==0&&(r=i,t=ca(e,i))),t===1)throw n=Xr,pn(e,0),Ut(e,r),Re(e,de()),n;if(t===6)Ut(e,r);else{if(i=e.current.alternate,!(r&30)&&!Gm(i)&&(t=cl(e,r),t===2&&(l=Oo(e),l!==0&&(r=l,t=ca(e,l))),t===1))throw n=Xr,pn(e,0),Ut(e,r),Re(e,de()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(b(345));case 2:an(e,Me,Et);break;case 3:if(Ut(e,r),(r&130023424)===r&&(t=us+500-de(),10<t)){if(Qi(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Pe(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=$o(an.bind(null,e,Me,Et),t);break}an(e,Me,Et);break;case 4:if(Ut(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-ut(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=de()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*Xm(r/1960))-r,10<r){e.timeoutHandle=$o(an.bind(null,e,Me,Et),r);break}an(e,Me,Et);break;case 5:an(e,Me,Et);break;default:throw Error(b(329))}}}return Re(e,de()),e.callbackNode===n?Rd.bind(null,e):null}function ca(e,t){var n=zr;return e.current.memoizedState.isDehydrated&&(pn(e,t).flags|=256),e=cl(e,t),e!==2&&(t=Me,Me=n,t!==null&&fa(t)),e}function fa(e){Me===null?Me=e:Me.push.apply(Me,e)}function Gm(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!ft(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Ut(e,t){for(t&=~ss,t&=~El,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-ut(t),r=1<<n;e[n]=-1,t&=~r}}function Hu(e){if(K&6)throw Error(b(327));Wn();var t=Qi(e,0);if(!(t&1))return Re(e,de()),null;var n=cl(e,t);if(e.tag!==0&&n===2){var r=Oo(e);r!==0&&(t=r,n=ca(e,r))}if(n===1)throw n=Xr,pn(e,0),Ut(e,t),Re(e,de()),n;if(n===6)throw Error(b(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,an(e,Me,Et),Re(e,de()),null}function cs(e,t){var n=K;K|=1;try{return e(t)}finally{K=n,K===0&&(er=de()+500,kl&&nn())}}function xn(e){Vt!==null&&Vt.tag===0&&!(K&6)&&Wn();var t=K;K|=1;var n=et.transition,r=X;try{if(et.transition=null,X=1,e)return e()}finally{X=r,et.transition=n,K=t,!(K&6)&&nn()}}function fs(){Ve=Fn.current,ie(Fn)}function pn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,jm(n)),me!==null)for(n=me.return;n!==null;){var r=n;switch(Wa(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&Gi();break;case 3:Jn(),ie(Oe),ie(je),es();break;case 5:Za(r);break;case 4:Jn();break;case 13:ie(ae);break;case 19:ie(ae);break;case 10:Ya(r.type._context);break;case 22:case 23:fs()}n=n.return}if(ke=e,me=e=Gt(e.current,null),Se=Ve=t,ve=0,Xr=null,ss=El=vn=0,Me=zr=null,fn!==null){for(t=0;t<fn.length;t++)if(n=fn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}fn=null}return e}function Fd(e,t){do{var n=me;try{if(qa(),Ii.current=ol,ll){for(var r=se.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}ll=!1}if(yn=0,xe=ye=se=null,jr=!1,Kr=0,as.current=null,n===null||n.return===null){ve=1,Xr=t,me=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=Se,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var u=s,c=a,d=c.tag;if(!(c.mode&1)&&(d===0||d===11||d===15)){var p=c.alternate;p?(c.updateQueue=p.updateQueue,c.memoizedState=p.memoizedState,c.lanes=p.lanes):(c.updateQueue=null,c.memoizedState=null)}var f=zu(o);if(f!==null){f.flags&=-257,Pu(f,o,a,l,t),f.mode&1&&bu(l,u,t),t=f,s=u;var k=t.updateQueue;if(k===null){var C=new Set;C.add(s),t.updateQueue=C}else k.add(s);break e}else{if(!(t&1)){bu(l,u,t),ds();break e}s=Error(b(426))}}else if(le&&a.mode&1){var N=zu(o);if(N!==null){!(N.flags&65536)&&(N.flags|=256),Pu(N,o,a,l,t),Qa(Zn(s,a));break e}}l=s=Zn(s,a),ve!==4&&(ve=2),zr===null?zr=[l]:zr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var m=Sd(l,s,t);Su(l,m);break e;case 1:a=s;var y=l.type,g=l.stateNode;if(!(l.flags&128)&&(typeof y.getDerivedStateFromError=="function"||g!==null&&typeof g.componentDidCatch=="function"&&(Yt===null||!Yt.has(g)))){l.flags|=65536,t&=-t,l.lanes|=t;var S=Cd(l,a,t);Su(l,S);break e}}l=l.return}while(l!==null)}Hd(n)}catch(E){t=E,me===n&&n!==null&&(me=n=n.return);continue}break}while(!0)}function Bd(){var e=al.current;return al.current=ol,e===null?ol:e}function ds(){(ve===0||ve===3||ve===2)&&(ve=4),ke===null||!(vn&268435455)&&!(El&268435455)||Ut(ke,Se)}function cl(e,t){var n=K;K|=2;var r=Bd();(ke!==e||Se!==t)&&(Et=null,pn(e,t));do try{Jm();break}catch(i){Fd(e,i)}while(!0);if(qa(),K=n,al.current=r,me!==null)throw Error(b(261));return ke=null,Se=0,ve}function Jm(){for(;me!==null;)Ud(me)}function Zm(){for(;me!==null&&!Eh();)Ud(me)}function Ud(e){var t=$d(e.alternate,e,Ve);e.memoizedProps=e.pendingProps,t===null?Hd(e):me=t,as.current=null}function Hd(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=Qm(n,t),n!==null){n.flags&=32767,me=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{ve=6,me=null;return}}else if(n=Wm(n,t,Ve),n!==null){me=n;return}if(t=t.sibling,t!==null){me=t;return}me=t=e}while(t!==null);ve===0&&(ve=5)}function an(e,t,n){var r=X,i=et.transition;try{et.transition=null,X=1,eg(e,t,n,r)}finally{et.transition=i,X=r}return null}function eg(e,t,n,r){do Wn();while(Vt!==null);if(K&6)throw Error(b(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(b(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Mh(e,l),e===ke&&(me=ke=null,Se=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||wi||(wi=!0,Wd(Wi,function(){return Wn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=et.transition,et.transition=null;var o=X;X=1;var a=K;K|=4,as.current=null,qm(e,n),Od(n,e),km(Ho),Ki=!!Uo,Ho=Uo=null,e.current=n,Ym(n),Nh(),K=a,X=o,et.transition=l}else e.current=n;if(wi&&(wi=!1,Vt=e,ul=i),l=e.pendingLanes,l===0&&(Yt=null),bh(n.stateNode),Re(e,de()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(sl)throw sl=!1,e=sa,sa=null,e;return ul&1&&e.tag!==0&&Wn(),l=e.pendingLanes,l&1?e===ua?Pr++:(Pr=0,ua=e):Pr=0,nn(),null}function Wn(){if(Vt!==null){var e=Sf(ul),t=et.transition,n=X;try{if(et.transition=null,X=16>e?16:e,Vt===null)var r=!1;else{if(e=Vt,Vt=null,ul=0,K&6)throw Error(b(331));var i=K;for(K|=4,I=e.current;I!==null;){var l=I,o=l.child;if(I.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var u=a[s];for(I=u;I!==null;){var c=I;switch(c.tag){case 0:case 11:case 15:br(8,c,l)}var d=c.child;if(d!==null)d.return=c,I=d;else for(;I!==null;){c=I;var p=c.sibling,f=c.return;if(Id(c),c===u){I=null;break}if(p!==null){p.return=f,I=p;break}I=f}}}var k=l.alternate;if(k!==null){var C=k.child;if(C!==null){k.child=null;do{var N=C.sibling;C.sibling=null,C=N}while(C!==null)}}I=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,I=o;else e:for(;I!==null;){if(l=I,l.flags&2048)switch(l.tag){case 0:case 11:case 15:br(9,l,l.return)}var m=l.sibling;if(m!==null){m.return=l.return,I=m;break e}I=l.return}}var y=e.current;for(I=y;I!==null;){o=I;var g=o.child;if(o.subtreeFlags&2064&&g!==null)g.return=o,I=g;else e:for(o=y;I!==null;){if(a=I,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Cl(9,a)}}catch(E){ce(a,a.return,E)}if(a===o){I=null;break e}var S=a.sibling;if(S!==null){S.return=a.return,I=S;break e}I=a.return}}if(K=i,nn(),vt&&typeof vt.onPostCommitFiberRoot=="function")try{vt.onPostCommitFiberRoot(ml,e)}catch{}r=!0}return r}finally{X=n,et.transition=t}}return!1}function Vu(e,t,n){t=Zn(n,t),t=Sd(e,t,1),e=qt(e,t,1),t=Pe(),e!==null&&(Zr(e,1,t),Re(e,t))}function ce(e,t,n){if(e.tag===3)Vu(e,e,n);else for(;t!==null;){if(t.tag===3){Vu(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(Yt===null||!Yt.has(r))){e=Zn(n,e),e=Cd(t,e,1),t=qt(t,e,1),e=Pe(),t!==null&&(Zr(t,1,e),Re(t,e));break}}t=t.return}}function tg(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Pe(),e.pingedLanes|=e.suspendedLanes&n,ke===e&&(Se&n)===n&&(ve===4||ve===3&&(Se&130023424)===Se&&500>de()-us?pn(e,0):ss|=n),Re(e,t)}function Vd(e,t){t===0&&(e.mode&1?(t=fi,fi<<=1,!(fi&130023424)&&(fi=4194304)):t=1);var n=Pe();e=Lt(e,t),e!==null&&(Zr(e,t,n),Re(e,n))}function ng(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),Vd(e,n)}function rg(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(b(314))}r!==null&&r.delete(t),Vd(e,n)}var $d;$d=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Oe.current)Ae=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return Ae=!1,$m(e,t,n);Ae=!!(e.flags&131072)}else Ae=!1,le&&t.flags&1048576&&qf(t,el,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Ai(e,t),e=t.pendingProps;var i=Yn(t,je.current);$n(t,n),i=ns(null,t,r,e,i,n);var l=rs();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,De(r)?(l=!0,Ji(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,Ga(t),i.updater=Sl,t.stateNode=i,i._reactInternals=t,Go(t,r,e,n),t=ea(null,t,r,!0,l,n)):(t.tag=0,le&&l&&$a(t),ze(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Ai(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=lg(r),e=ot(r,e),i){case 0:t=Zo(null,t,r,e,n);break e;case 1:t=Iu(null,t,r,e,n);break e;case 11:t=Tu(null,t,r,e,n);break e;case 14:t=Lu(null,t,r,ot(r.type,e),n);break e}throw Error(b(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ot(r,i),Zo(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ot(r,i),Iu(e,t,r,i,n);case 3:e:{if(jd(t),e===null)throw Error(b(387));r=t.pendingProps,l=t.memoizedState,i=l.element,ed(e,t),rl(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=Zn(Error(b(423)),t),t=Mu(e,t,r,n,i);break e}else if(r!==i){i=Zn(Error(b(424)),t),t=Mu(e,t,r,n,i);break e}else for($e=Kt(t.stateNode.containerInfo.firstChild),Qe=t,le=!0,st=null,n=Jf(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(Xn(),r===i){t=It(e,t,n);break e}ze(e,t,r,n)}t=t.child}return t;case 5:return td(t),e===null&&qo(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,Vo(r,i)?o=null:l!==null&&Vo(r,l)&&(t.flags|=32),_d(e,t),ze(e,t,o,n),t.child;case 6:return e===null&&qo(t),null;case 13:return bd(e,t,n);case 4:return Ja(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=Gn(t,null,r,n):ze(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ot(r,i),Tu(e,t,r,i,n);case 7:return ze(e,t,t.pendingProps,n),t.child;case 8:return ze(e,t,t.pendingProps.children,n),t.child;case 12:return ze(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,ee(tl,r._currentValue),r._currentValue=o,l!==null)if(ft(l.value,o)){if(l.children===i.children&&!Oe.current){t=It(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=zt(-1,n&-n),s.tag=2;var u=l.updateQueue;if(u!==null){u=u.shared;var c=u.pending;c===null?s.next=s:(s.next=c.next,c.next=s),u.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),Yo(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(b(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),Yo(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}ze(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,$n(t,n),i=tt(i),r=r(i),t.flags|=1,ze(e,t,r,n),t.child;case 14:return r=t.type,i=ot(r,t.pendingProps),i=ot(r.type,i),Lu(e,t,r,i,n);case 15:return Ed(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:ot(r,i),Ai(e,t),t.tag=1,De(r)?(e=!0,Ji(t)):e=!1,$n(t,n),wd(t,r,i),Go(t,r,i,n),ea(null,t,r,!0,e,n);case 19:return zd(e,t,n);case 22:return Nd(e,t,n)}throw Error(b(156,t.tag))};function Wd(e,t){return vf(e,t)}function ig(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function Ze(e,t,n,r){return new ig(e,t,n,r)}function ps(e){return e=e.prototype,!(!e||!e.isReactComponent)}function lg(e){if(typeof e=="function")return ps(e)?1:0;if(e!=null){if(e=e.$$typeof,e===La)return 11;if(e===Ia)return 14}return 2}function Gt(e,t){var n=e.alternate;return n===null?(n=Ze(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Ri(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")ps(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case zn:return hn(n.children,i,l,t);case Ta:o=8,i|=8;break;case wo:return e=Ze(12,n,t,i|2),e.elementType=wo,e.lanes=l,e;case So:return e=Ze(13,n,t,i),e.elementType=So,e.lanes=l,e;case Co:return e=Ze(19,n,t,i),e.elementType=Co,e.lanes=l,e;case ef:return Nl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case Jc:o=10;break e;case Zc:o=9;break e;case La:o=11;break e;case Ia:o=14;break e;case Rt:o=16,r=null;break e}throw Error(b(130,e==null?e:typeof e,""))}return t=Ze(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function hn(e,t,n,r){return e=Ze(7,e,r,t),e.lanes=n,e}function Nl(e,t,n,r){return e=Ze(22,e,r,t),e.elementType=ef,e.lanes=n,e.stateNode={isHidden:!1},e}function io(e,t,n){return e=Ze(6,e,null,t),e.lanes=n,e}function lo(e,t,n){return t=Ze(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function og(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Fl(0),this.expirationTimes=Fl(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Fl(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function hs(e,t,n,r,i,l,o,a,s){return e=new og(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=Ze(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},Ga(l),e}function ag(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:bn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function Qd(e){if(!e)return Zt;e=e._reactInternals;e:{if(wn(e)!==e||e.tag!==1)throw Error(b(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(De(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(b(171))}if(e.tag===1){var n=e.type;if(De(n))return Qf(e,n,t)}return t}function Kd(e,t,n,r,i,l,o,a,s){return e=hs(n,r,!0,e,i,l,o,a,s),e.context=Qd(null),n=e.current,r=Pe(),i=Xt(n),l=zt(r,i),l.callback=t??null,qt(n,l,i),e.current.lanes=i,Zr(e,i,r),Re(e,r),e}function _l(e,t,n,r){var i=t.current,l=Pe(),o=Xt(i);return n=Qd(n),t.context===null?t.context=n:t.pendingContext=n,t=zt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=qt(i,t,o),e!==null&&(ct(e,i,o,l),Li(e,i,o)),o}function fl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function $u(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function ms(e,t){$u(e,t),(e=e.alternate)&&$u(e,t)}function sg(){return null}var qd=typeof reportError=="function"?reportError:function(e){console.error(e)};function gs(e){this._internalRoot=e}jl.prototype.render=gs.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(b(409));_l(e,t,null,null)};jl.prototype.unmount=gs.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;xn(function(){_l(null,e,null,null)}),t[Tt]=null}};function jl(e){this._internalRoot=e}jl.prototype.unstable_scheduleHydration=function(e){if(e){var t=Nf();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Bt.length&&t!==0&&t<Bt[n].priority;n++);Bt.splice(n,0,e),n===0&&jf(e)}};function ys(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function bl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function Wu(){}function ug(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var u=fl(o);l.call(u)}}var o=Kd(t,r,e,0,null,!1,!1,"",Wu);return e._reactRootContainer=o,e[Tt]=o.current,Hr(e.nodeType===8?e.parentNode:e),xn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var u=fl(s);a.call(u)}}var s=hs(e,0,!1,null,null,!1,!1,"",Wu);return e._reactRootContainer=s,e[Tt]=s.current,Hr(e.nodeType===8?e.parentNode:e),xn(function(){_l(t,s,n,r)}),s}function zl(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=fl(o);a.call(s)}}_l(t,o,e,i)}else o=ug(n,t,e,i,r);return fl(o)}Cf=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=kr(t.pendingLanes);n!==0&&(Oa(t,n|1),Re(t,de()),!(K&6)&&(er=de()+500,nn()))}break;case 13:xn(function(){var r=Lt(e,1);if(r!==null){var i=Pe();ct(r,e,1,i)}}),ms(e,1)}};Da=function(e){if(e.tag===13){var t=Lt(e,134217728);if(t!==null){var n=Pe();ct(t,e,134217728,n)}ms(e,134217728)}};Ef=function(e){if(e.tag===13){var t=Xt(e),n=Lt(e,t);if(n!==null){var r=Pe();ct(n,e,t,r)}ms(e,t)}};Nf=function(){return X};_f=function(e,t){var n=X;try{return X=e,t()}finally{X=n}};Io=function(e,t,n){switch(t){case"input":if(_o(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=xl(r);if(!i)throw Error(b(90));nf(r),_o(r,i)}}}break;case"textarea":lf(e,n);break;case"select":t=n.value,t!=null&&Bn(e,!!n.multiple,t,!1)}};df=cs;pf=xn;var cg={usingClientEntryPoint:!1,Events:[ti,In,xl,cf,ff,cs]},mr={findFiberByHostInstance:cn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},fg={bundleType:mr.bundleType,version:mr.version,rendererPackageName:mr.rendererPackageName,rendererConfig:mr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Mt.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=gf(e),e===null?null:e.stateNode},findFiberByHostInstance:mr.findFiberByHostInstance||sg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var Si=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!Si.isDisabled&&Si.supportsFiber)try{ml=Si.inject(fg),vt=Si}catch{}}qe.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=cg;qe.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!ys(t))throw Error(b(200));return ag(e,t,null,n)};qe.createRoot=function(e,t){if(!ys(e))throw Error(b(299));var n=!1,r="",i=qd;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=hs(e,1,!1,null,null,n,!1,r,i),e[Tt]=t.current,Hr(e.nodeType===8?e.parentNode:e),new gs(t)};qe.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(b(188)):(e=Object.keys(e).join(","),Error(b(268,e)));return e=gf(t),e=e===null?null:e.stateNode,e};qe.flushSync=function(e){return xn(e)};qe.hydrate=function(e,t,n){if(!bl(t))throw Error(b(200));return zl(null,e,t,!0,n)};qe.hydrateRoot=function(e,t,n){if(!ys(e))throw Error(b(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=qd;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=Kd(t,null,e,1,n??null,i,!1,l,o),e[Tt]=t.current,Hr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new jl(t)};qe.render=function(e,t,n){if(!bl(t))throw Error(b(200));return zl(null,e,t,!1,n)};qe.unmountComponentAtNode=function(e){if(!bl(e))throw Error(b(40));return e._reactRootContainer?(xn(function(){zl(null,null,e,!1,function(){e._reactRootContainer=null,e[Tt]=null})}),!0):!1};qe.unstable_batchedUpdates=cs;qe.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!bl(n))throw Error(b(200));if(e==null||e._reactInternals===void 0)throw Error(b(38));return zl(e,t,n,!1,r)};qe.version="18.3.1-next-f1338f8080-20240426";function Yd(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Yd)}catch(e){console.error(e)}}Yd(),qc.exports=qe;var dg=qc.exports,Qu=dg;xo.createRoot=Qu.createRoot,xo.hydrateRoot=Qu.hydrateRoot;const Ci={plus:h.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),h.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),user:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),h.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),h.jsx("circle",{cx:"12",cy:"5",r:"2"}),h.jsx("path",{d:"M12 7v4"}),h.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),h.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),h.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),h.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),h.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]})},pg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,unreadCounts:i})=>{const[l,o]=H.useState(!1),[a,s]=H.useState(""),u=()=>{a.trim()&&(r(a.trim()),s(""),o(!1))},c=p=>{p.key==="Enter"&&!p.shiftKey?(p.preventDefault(),u()):p.key==="Escape"&&(o(!1),s(""))},d=p=>{const f=new Date(p),C=new Date().getTime()-f.getTime(),N=Math.floor(C/6e4),m=Math.floor(C/36e5),y=Math.floor(C/864e5);return N<1?"now":N<60?`${N}m`:m<24?`${m}h`:y<7?`${y}d`:f.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return h.jsxs("div",{className:"thread-list",children:[h.jsxs("div",{className:"list-header",children:[h.jsx("h2",{children:"Conversations"}),h.jsx("button",{className:"new-thread-btn",onClick:()=>o(!0),title:"New conversation",children:Ci.plus})]}),l&&h.jsxs("div",{className:"new-thread-form",children:[h.jsx("input",{type:"text",value:a,onChange:p=>s(p.target.value),onKeyDown:c,placeholder:"Conversation title...",autoFocus:!0}),h.jsxs("div",{className:"form-actions",children:[h.jsx("button",{className:"cancel-btn",onClick:()=>o(!1),children:"Cancel"}),h.jsx("button",{className:"create-btn",onClick:u,children:"Create"})]})]}),h.jsx("div",{className:"thread-items",children:e.length===0?h.jsxs("div",{className:"empty-state",children:[h.jsx("div",{className:"empty-icon",children:Ci.hash}),h.jsx("p",{children:"No conversations yet"}),h.jsx("button",{className:"start-btn",onClick:()=>o(!0),children:"Start a conversation"})]}):e.map(p=>{const f=i.get(p.id)||0,k=p.id===t;return h.jsxs("div",{className:`thread-item ${k?"selected":""} ${f>0?"has-unread":""}`,onClick:()=>n(p.id),children:[h.jsx("div",{className:`status-dot ${p.status}`}),h.jsxs("div",{className:"thread-content",children:[h.jsxs("div",{className:"thread-title-row",children:[h.jsx("span",{className:"thread-title",children:p.title}),h.jsx("span",{className:"thread-time",children:d(p.updated_at)})]}),h.jsxs("div",{className:"thread-meta",children:[h.jsxs("span",{className:"thread-creator",children:[p.created_by_type==="human"?Ci.user:Ci.bot,p.created_by_id]}),h.jsxs("span",{className:"thread-seq",children:["#",p.last_seq]})]})]}),f>0&&h.jsx("span",{className:"unread-badge",children:f})]},p.id)})}),h.jsx("style",{children:`
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
      `})]})};function hg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const mg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,gg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,yg={};function Ku(e,t){return(yg.jsx?gg:mg).test(e)}const vg=/[ \t\n\f\r]/g;function xg(e){return typeof e=="object"?e.type==="text"?qu(e.value):!1:qu(e)}function qu(e){return e.replace(vg,"")===""}class ri{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}ri.prototype.normal={};ri.prototype.property={};ri.prototype.space=void 0;function Xd(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new ri(n,r,t)}function da(e){return e.toLowerCase()}class Be{constructor(t,n){this.attribute=n,this.property=t}}Be.prototype.attribute="";Be.prototype.booleanish=!1;Be.prototype.boolean=!1;Be.prototype.commaOrSpaceSeparated=!1;Be.prototype.commaSeparated=!1;Be.prototype.defined=!1;Be.prototype.mustUseProperty=!1;Be.prototype.number=!1;Be.prototype.overloadedBoolean=!1;Be.prototype.property="";Be.prototype.spaceSeparated=!1;Be.prototype.space=void 0;let kg=0;const U=Sn(),he=Sn(),pa=Sn(),z=Sn(),Z=Sn(),Qn=Sn(),He=Sn();function Sn(){return 2**++kg}const ha=Object.freeze(Object.defineProperty({__proto__:null,boolean:U,booleanish:he,commaOrSpaceSeparated:He,commaSeparated:Qn,number:z,overloadedBoolean:pa,spaceSeparated:Z},Symbol.toStringTag,{value:"Module"})),oo=Object.keys(ha);class vs extends Be{constructor(t,n,r,i){let l=-1;if(super(t,n),Yu(this,"space",i),typeof r=="number")for(;++l<oo.length;){const o=oo[l];Yu(this,oo[l],(r&ha[o])===ha[o])}}}vs.prototype.defined=!0;function Yu(e,t,n){n&&(e[t]=n)}function ir(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new vs(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[da(r)]=r,n[da(l.attribute)]=r}return new ri(t,n,e.space)}const Gd=ir({properties:{ariaActiveDescendant:null,ariaAtomic:he,ariaAutoComplete:null,ariaBusy:he,ariaChecked:he,ariaColCount:z,ariaColIndex:z,ariaColSpan:z,ariaControls:Z,ariaCurrent:null,ariaDescribedBy:Z,ariaDetails:null,ariaDisabled:he,ariaDropEffect:Z,ariaErrorMessage:null,ariaExpanded:he,ariaFlowTo:Z,ariaGrabbed:he,ariaHasPopup:null,ariaHidden:he,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:Z,ariaLevel:z,ariaLive:null,ariaModal:he,ariaMultiLine:he,ariaMultiSelectable:he,ariaOrientation:null,ariaOwns:Z,ariaPlaceholder:null,ariaPosInSet:z,ariaPressed:he,ariaReadOnly:he,ariaRelevant:null,ariaRequired:he,ariaRoleDescription:Z,ariaRowCount:z,ariaRowIndex:z,ariaRowSpan:z,ariaSelected:he,ariaSetSize:z,ariaSort:null,ariaValueMax:z,ariaValueMin:z,ariaValueNow:z,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function Jd(e,t){return t in e?e[t]:t}function Zd(e,t){return Jd(e,t.toLowerCase())}const wg=ir({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:Qn,acceptCharset:Z,accessKey:Z,action:null,allow:null,allowFullScreen:U,allowPaymentRequest:U,allowUserMedia:U,alt:null,as:null,async:U,autoCapitalize:null,autoComplete:Z,autoFocus:U,autoPlay:U,blocking:Z,capture:null,charSet:null,checked:U,cite:null,className:Z,cols:z,colSpan:null,content:null,contentEditable:he,controls:U,controlsList:Z,coords:z|Qn,crossOrigin:null,data:null,dateTime:null,decoding:null,default:U,defer:U,dir:null,dirName:null,disabled:U,download:pa,draggable:he,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:U,formTarget:null,headers:Z,height:z,hidden:pa,high:z,href:null,hrefLang:null,htmlFor:Z,httpEquiv:Z,id:null,imageSizes:null,imageSrcSet:null,inert:U,inputMode:null,integrity:null,is:null,isMap:U,itemId:null,itemProp:Z,itemRef:Z,itemScope:U,itemType:Z,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:U,low:z,manifest:null,max:null,maxLength:z,media:null,method:null,min:null,minLength:z,multiple:U,muted:U,name:null,nonce:null,noModule:U,noValidate:U,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:U,optimum:z,pattern:null,ping:Z,placeholder:null,playsInline:U,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:U,referrerPolicy:null,rel:Z,required:U,reversed:U,rows:z,rowSpan:z,sandbox:Z,scope:null,scoped:U,seamless:U,selected:U,shadowRootClonable:U,shadowRootDelegatesFocus:U,shadowRootMode:null,shape:null,size:z,sizes:null,slot:null,span:z,spellCheck:he,src:null,srcDoc:null,srcLang:null,srcSet:null,start:z,step:null,style:null,tabIndex:z,target:null,title:null,translate:null,type:null,typeMustMatch:U,useMap:null,value:he,width:z,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:Z,axis:null,background:null,bgColor:null,border:z,borderColor:null,bottomMargin:z,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:U,declare:U,event:null,face:null,frame:null,frameBorder:null,hSpace:z,leftMargin:z,link:null,longDesc:null,lowSrc:null,marginHeight:z,marginWidth:z,noResize:U,noHref:U,noShade:U,noWrap:U,object:null,profile:null,prompt:null,rev:null,rightMargin:z,rules:null,scheme:null,scrolling:he,standby:null,summary:null,text:null,topMargin:z,valueType:null,version:null,vAlign:null,vLink:null,vSpace:z,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:U,disableRemotePlayback:U,prefix:null,property:null,results:z,security:null,unselectable:null},space:"html",transform:Zd}),Sg=ir({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:He,accentHeight:z,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:z,amplitude:z,arabicForm:null,ascent:z,attributeName:null,attributeType:null,azimuth:z,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:z,by:null,calcMode:null,capHeight:z,className:Z,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:z,diffuseConstant:z,direction:null,display:null,dur:null,divisor:z,dominantBaseline:null,download:U,dx:null,dy:null,edgeMode:null,editable:null,elevation:z,enableBackground:null,end:null,event:null,exponent:z,externalResourcesRequired:null,fill:null,fillOpacity:z,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:Qn,g2:Qn,glyphName:Qn,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:z,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:z,horizOriginX:z,horizOriginY:z,id:null,ideographic:z,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:z,k:z,k1:z,k2:z,k3:z,k4:z,kernelMatrix:He,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:z,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:z,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:z,overlineThickness:z,paintOrder:null,panose1:null,path:null,pathLength:z,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:Z,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:z,pointsAtY:z,pointsAtZ:z,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:He,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:He,rev:He,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:He,requiredFeatures:He,requiredFonts:He,requiredFormats:He,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:z,specularExponent:z,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:z,strikethroughThickness:z,string:null,stroke:null,strokeDashArray:He,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:z,strokeOpacity:z,strokeWidth:null,style:null,surfaceScale:z,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:He,tabIndex:z,tableValues:null,target:null,targetX:z,targetY:z,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:He,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:z,underlineThickness:z,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:z,values:null,vAlphabetic:z,vMathematical:z,vectorEffect:null,vHanging:z,vIdeographic:z,version:null,vertAdvY:z,vertOriginX:z,vertOriginY:z,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:z,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:Jd}),ep=ir({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),tp=ir({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:Zd}),np=ir({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),Cg={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Eg=/[A-Z]/g,Xu=/-[a-z]/g,Ng=/^data[-\w.:]+$/i;function _g(e,t){const n=da(t);let r=t,i=Be;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&Ng.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(Xu,bg);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!Xu.test(l)){let o=l.replace(Eg,jg);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=vs}return new i(r,t)}function jg(e){return"-"+e.toLowerCase()}function bg(e){return e.charAt(1).toUpperCase()}const zg=Xd([Gd,wg,ep,tp,np],"html"),xs=Xd([Gd,Sg,ep,tp,np],"svg");function Pg(e){return e.join(" ").trim()}var ks={},Gu=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Tg=/\n/g,Lg=/^\s*/,Ig=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Mg=/^:\s*/,Ag=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Og=/^[;\s]*/,Dg=/^\s+|\s+$/g,Rg=`
`,Ju="/",Zu="*",sn="",Fg="comment",Bg="declaration";function Ug(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(k){var C=k.match(Tg);C&&(n+=C.length);var N=k.lastIndexOf(Rg);r=~N?k.length-N:r+k.length}function l(){var k={line:n,column:r};return function(C){return C.position=new o(k),u(),C}}function o(k){this.start=k,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(k){var C=new Error(t.source+":"+n+":"+r+": "+k);if(C.reason=k,C.filename=t.source,C.line=n,C.column=r,C.source=e,!t.silent)throw C}function s(k){var C=k.exec(e);if(C){var N=C[0];return i(N),e=e.slice(N.length),C}}function u(){s(Lg)}function c(k){var C;for(k=k||[];C=d();)C!==!1&&k.push(C);return k}function d(){var k=l();if(!(Ju!=e.charAt(0)||Zu!=e.charAt(1))){for(var C=2;sn!=e.charAt(C)&&(Zu!=e.charAt(C)||Ju!=e.charAt(C+1));)++C;if(C+=2,sn===e.charAt(C-1))return a("End of comment missing");var N=e.slice(2,C-2);return r+=2,i(N),e=e.slice(C),r+=2,k({type:Fg,comment:N})}}function p(){var k=l(),C=s(Ig);if(C){if(d(),!s(Mg))return a("property missing ':'");var N=s(Ag),m=k({type:Bg,property:ec(C[0].replace(Gu,sn)),value:N?ec(N[0].replace(Gu,sn)):sn});return s(Og),m}}function f(){var k=[];c(k);for(var C;C=p();)C!==!1&&(k.push(C),c(k));return k}return u(),f()}function ec(e){return e?e.replace(Dg,sn):sn}var Hg=Ug,Vg=Ui&&Ui.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(ks,"__esModule",{value:!0});ks.default=Wg;const $g=Vg(Hg);function Wg(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,$g.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Pl={};Object.defineProperty(Pl,"__esModule",{value:!0});Pl.camelCase=void 0;var Qg=/^--[a-zA-Z0-9_-]+$/,Kg=/-([a-z])/g,qg=/^[^-]+$/,Yg=/^-(webkit|moz|ms|o|khtml)-/,Xg=/^-(ms)-/,Gg=function(e){return!e||qg.test(e)||Qg.test(e)},Jg=function(e,t){return t.toUpperCase()},tc=function(e,t){return"".concat(t,"-")},Zg=function(e,t){return t===void 0&&(t={}),Gg(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(Xg,tc):e=e.replace(Yg,tc),e.replace(Kg,Jg))};Pl.camelCase=Zg;var ey=Ui&&Ui.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},ty=ey(ks),ny=Pl;function ma(e,t){var n={};return!e||typeof e!="string"||(0,ty.default)(e,function(r,i){r&&i&&(n[(0,ny.camelCase)(r,t)]=i)}),n}ma.default=ma;var ry=ma;const iy=Ca(ry),rp=ip("end"),ws=ip("start");function ip(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function ly(e){const t=ws(e),n=rp(e);if(t&&n)return{start:t,end:n}}function Tr(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?nc(e.position):"start"in e||"end"in e?nc(e):"line"in e||"column"in e?ga(e):""}function ga(e){return rc(e&&e.line)+":"+rc(e&&e.column)}function nc(e){return ga(e&&e.start)+"-"+ga(e&&e.end)}function rc(e){return e&&typeof e=="number"?e:1}class be extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Tr(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}be.prototype.file="";be.prototype.name="";be.prototype.reason="";be.prototype.message="";be.prototype.stack="";be.prototype.column=void 0;be.prototype.line=void 0;be.prototype.ancestors=void 0;be.prototype.cause=void 0;be.prototype.fatal=void 0;be.prototype.place=void 0;be.prototype.ruleId=void 0;be.prototype.source=void 0;const Ss={}.hasOwnProperty,oy=new Map,ay=/[A-Z]/g,sy=new Set(["table","tbody","thead","tfoot","tr"]),uy=new Set(["td","th"]),lp="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function cy(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=vy(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=yy(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?xs:zg,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=op(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function op(e,t,n){if(t.type==="element")return fy(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return dy(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return hy(e,t,n);if(t.type==="mdxjsEsm")return py(e,t);if(t.type==="root")return my(e,t,n);if(t.type==="text")return gy(e,t)}function fy(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=xs,e.schema=i),e.ancestors.push(t);const l=sp(e,t.tagName,!1),o=xy(e,t);let a=Es(e,t);return sy.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!xg(s):!0})),ap(e,o,l,t),Cs(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function dy(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}Gr(e,t.position)}function py(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);Gr(e,t.position)}function hy(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=xs,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:sp(e,t.name,!0),o=ky(e,t),a=Es(e,t);return ap(e,o,l,t),Cs(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function my(e,t,n){const r={};return Cs(r,Es(e,t)),e.create(t,e.Fragment,r,n)}function gy(e,t){return t.value}function ap(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Cs(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function yy(e,t,n){return r;function r(i,l,o,a){const u=Array.isArray(o.children)?n:t;return a?u(l,o,a):u(l,o)}}function vy(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=ws(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function xy(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Ss.call(t.properties,i)){const l=wy(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&uy.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function ky(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else Gr(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else Gr(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Es(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:oy;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const u=i.get(s)||0;o=s+"-"+u,i.set(s,u+1)}}const a=op(e,l,o);a!==void 0&&n.push(a)}return n}function wy(e,t,n){const r=_g(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?hg(n):Pg(n)),r.property==="style"){let i=typeof n=="object"?n:Sy(e,String(n));return e.stylePropertyNameCase==="css"&&(i=Cy(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?Cg[r.property]||r.property:r.attribute,n]}}function Sy(e,t){try{return iy(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new be("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=lp+"#cannot-parse-style-attribute",i}}function sp(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=Ku(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=Ku(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Ss.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);Gr(e)}function Gr(e,t){const n=new be("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=lp+"#cannot-handle-mdx-estrees-without-createevaluater",n}function Cy(e){const t={};let n;for(n in e)Ss.call(e,n)&&(t[Ey(n)]=e[n]);return t}function Ey(e){let t=e.replace(ay,Ny);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function Ny(e){return"-"+e.toLowerCase()}const ao={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},_y={};function jy(e,t){const n=_y,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return up(e,r,i)}function up(e,t,n){if(by(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return ic(e.children,t,n)}return Array.isArray(e)?ic(e,t,n):""}function ic(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=up(e[i],t,n);return r.join("")}function by(e){return!!(e&&typeof e=="object")}const lc=document.createElement("i");function Ns(e){const t="&"+e+";";lc.innerHTML=t;const n=lc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function kt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function Je(e,t){return e.length>0?(kt(e,e.length,0,t),e):t}const oc={}.hasOwnProperty;function zy(e){const t={};let n=-1;for(;++n<e.length;)Py(t,e[n]);return t}function Py(e,t){let n;for(n in t){const i=(oc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){oc.call(i,o)||(i[o]=[]);const a=l[o];Ty(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Ty(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);kt(e,0,0,r)}function cp(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function Kn(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const yt=rn(/[A-Za-z]/),We=rn(/[\dA-Za-z]/),Ly=rn(/[#-'*+\--9=?A-Z^-~]/);function ya(e){return e!==null&&(e<32||e===127)}const va=rn(/\d/),Iy=rn(/[\dA-Fa-f]/),My=rn(/[!-/:-@[-`{-~]/);function F(e){return e!==null&&e<-2}function Fe(e){return e!==null&&(e<0||e===32)}function q(e){return e===-2||e===-1||e===32}const Ay=rn(new RegExp("\\p{P}|\\p{S}","u")),Oy=rn(/\s/);function rn(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function lr(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&We(e.charCodeAt(n+1))&&We(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function te(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return q(s)?(e.enter(n),a(s)):t(s)}function a(s){return q(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const Dy={tokenize:Ry};function Ry(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),te(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return F(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const Fy={tokenize:By},ac={tokenize:Uy};function By(e){const t=this,n=[];let r=0,i,l,o;return a;function a(g){if(r<n.length){const S=n[r];return t.containerState=S[1],e.attempt(S[0].continuation,s,u)(g)}return u(g)}function s(g){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&y();const S=t.events.length;let E=S,w;for(;E--;)if(t.events[E][0]==="exit"&&t.events[E][1].type==="chunkFlow"){w=t.events[E][1].end;break}m(r);let _=S;for(;_<t.events.length;)t.events[_][1].end={...w},_++;return kt(t.events,E+1,0,t.events.slice(S)),t.events.length=_,u(g)}return a(g)}function u(g){if(r===n.length){if(!i)return p(g);if(i.currentConstruct&&i.currentConstruct.concrete)return k(g);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(ac,c,d)(g)}function c(g){return i&&y(),m(r),p(g)}function d(g){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,k(g)}function p(g){return t.containerState={},e.attempt(ac,f,k)(g)}function f(g){return r++,n.push([t.currentConstruct,t.containerState]),p(g)}function k(g){if(g===null){i&&y(),m(0),e.consume(g);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),C(g)}function C(g){if(g===null){N(e.exit("chunkFlow"),!0),m(0),e.consume(g);return}return F(g)?(e.consume(g),N(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(g),C)}function N(g,S){const E=t.sliceStream(g);if(S&&E.push(null),g.previous=l,l&&(l.next=g),l=g,i.defineSkip(g.start),i.write(E),t.parser.lazy[g.start.line]){let w=i.events.length;for(;w--;)if(i.events[w][1].start.offset<o&&(!i.events[w][1].end||i.events[w][1].end.offset>o))return;const _=t.events.length;let P=_,O,M;for(;P--;)if(t.events[P][0]==="exit"&&t.events[P][1].type==="chunkFlow"){if(O){M=t.events[P][1].end;break}O=!0}for(m(r),w=_;w<t.events.length;)t.events[w][1].end={...M},w++;kt(t.events,P+1,0,t.events.slice(_)),t.events.length=w}}function m(g){let S=n.length;for(;S-- >g;){const E=n[S];t.containerState=E[1],E[0].exit.call(t,e)}n.length=g}function y(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Uy(e,t,n){return te(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function sc(e){if(e===null||Fe(e)||Oy(e))return 1;if(Ay(e))return 2}function _s(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const xa={name:"attention",resolveAll:Hy,tokenize:Vy};function Hy(e,t){let n=-1,r,i,l,o,a,s,u,c;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const d={...e[r][1].end},p={...e[n][1].start};uc(d,-s),uc(p,s),o={type:s>1?"strongSequence":"emphasisSequence",start:d,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:p},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},u=[],e[r][1].end.offset-e[r][1].start.offset&&(u=Je(u,[["enter",e[r][1],t],["exit",e[r][1],t]])),u=Je(u,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),u=Je(u,_s(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),u=Je(u,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(c=2,u=Je(u,[["enter",e[n][1],t],["exit",e[n][1],t]])):c=0,kt(e,r-1,n-r+3,u),n=r+u.length-c-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Vy(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=sc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const u=e.exit("attentionSequence"),c=sc(s),d=!c||c===2&&i||n.includes(s),p=!i||i===2&&c||n.includes(r);return u._open=!!(l===42?d:d&&(i||!p)),u._close=!!(l===42?p:p&&(c||!d)),t(s)}}function uc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const $y={name:"autolink",tokenize:Wy};function Wy(e,t,n){let r=0;return i;function i(f){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(f){return yt(f)?(e.consume(f),o):f===64?n(f):u(f)}function o(f){return f===43||f===45||f===46||We(f)?(r=1,a(f)):u(f)}function a(f){return f===58?(e.consume(f),r=0,s):(f===43||f===45||f===46||We(f))&&r++<32?(e.consume(f),a):(r=0,u(f))}function s(f){return f===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.exit("autolink"),t):f===null||f===32||f===60||ya(f)?n(f):(e.consume(f),s)}function u(f){return f===64?(e.consume(f),c):Ly(f)?(e.consume(f),u):n(f)}function c(f){return We(f)?d(f):n(f)}function d(f){return f===46?(e.consume(f),r=0,c):f===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.exit("autolink"),t):p(f)}function p(f){if((f===45||We(f))&&r++<63){const k=f===45?p:d;return e.consume(f),k}return n(f)}}const Tl={partial:!0,tokenize:Qy};function Qy(e,t,n){return r;function r(l){return q(l)?te(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||F(l)?t(l):n(l)}}const fp={continuation:{tokenize:qy},exit:Yy,name:"blockQuote",tokenize:Ky};function Ky(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return q(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function qy(e,t,n){const r=this;return i;function i(o){return q(o)?te(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(fp,t,n)(o)}}function Yy(e){e.exit("blockQuote")}const dp={name:"characterEscape",tokenize:Xy};function Xy(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return My(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const pp={name:"characterReference",tokenize:Gy};function Gy(e,t,n){const r=this;let i=0,l,o;return a;function a(d){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(d),e.exit("characterReferenceMarker"),s}function s(d){return d===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(d),e.exit("characterReferenceMarkerNumeric"),u):(e.enter("characterReferenceValue"),l=31,o=We,c(d))}function u(d){return d===88||d===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(d),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=Iy,c):(e.enter("characterReferenceValue"),l=7,o=va,c(d))}function c(d){if(d===59&&i){const p=e.exit("characterReferenceValue");return o===We&&!Ns(r.sliceSerialize(p))?n(d):(e.enter("characterReferenceMarker"),e.consume(d),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(d)&&i++<l?(e.consume(d),c):n(d)}}const cc={partial:!0,tokenize:Zy},fc={concrete:!0,name:"codeFenced",tokenize:Jy};function Jy(e,t,n){const r=this,i={partial:!0,tokenize:E};let l=0,o=0,a;return s;function s(w){return u(w)}function u(w){const _=r.events[r.events.length-1];return l=_&&_[1].type==="linePrefix"?_[2].sliceSerialize(_[1],!0).length:0,a=w,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),c(w)}function c(w){return w===a?(o++,e.consume(w),c):o<3?n(w):(e.exit("codeFencedFenceSequence"),q(w)?te(e,d,"whitespace")(w):d(w))}function d(w){return w===null||F(w)?(e.exit("codeFencedFence"),r.interrupt?t(w):e.check(cc,C,S)(w)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),p(w))}function p(w){return w===null||F(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),d(w)):q(w)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),te(e,f,"whitespace")(w)):w===96&&w===a?n(w):(e.consume(w),p)}function f(w){return w===null||F(w)?d(w):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),k(w))}function k(w){return w===null||F(w)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),d(w)):w===96&&w===a?n(w):(e.consume(w),k)}function C(w){return e.attempt(i,S,N)(w)}function N(w){return e.enter("lineEnding"),e.consume(w),e.exit("lineEnding"),m}function m(w){return l>0&&q(w)?te(e,y,"linePrefix",l+1)(w):y(w)}function y(w){return w===null||F(w)?e.check(cc,C,S)(w):(e.enter("codeFlowValue"),g(w))}function g(w){return w===null||F(w)?(e.exit("codeFlowValue"),y(w)):(e.consume(w),g)}function S(w){return e.exit("codeFenced"),t(w)}function E(w,_,P){let O=0;return M;function M($){return w.enter("lineEnding"),w.consume($),w.exit("lineEnding"),A}function A($){return w.enter("codeFencedFence"),q($)?te(w,D,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)($):D($)}function D($){return $===a?(w.enter("codeFencedFenceSequence"),Y($)):P($)}function Y($){return $===a?(O++,w.consume($),Y):O>=o?(w.exit("codeFencedFenceSequence"),q($)?te(w,oe,"whitespace")($):oe($)):P($)}function oe($){return $===null||F($)?(w.exit("codeFencedFence"),_($)):P($)}}}function Zy(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const so={name:"codeIndented",tokenize:tv},ev={partial:!0,tokenize:nv};function tv(e,t,n){const r=this;return i;function i(u){return e.enter("codeIndented"),te(e,l,"linePrefix",5)(u)}function l(u){const c=r.events[r.events.length-1];return c&&c[1].type==="linePrefix"&&c[2].sliceSerialize(c[1],!0).length>=4?o(u):n(u)}function o(u){return u===null?s(u):F(u)?e.attempt(ev,o,s)(u):(e.enter("codeFlowValue"),a(u))}function a(u){return u===null||F(u)?(e.exit("codeFlowValue"),o(u)):(e.consume(u),a)}function s(u){return e.exit("codeIndented"),t(u)}}function nv(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):F(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):te(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):F(o)?i(o):n(o)}}const rv={name:"codeText",previous:lv,resolve:iv,tokenize:ov};function iv(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function lv(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function ov(e,t,n){let r=0,i,l;return o;function o(d){return e.enter("codeText"),e.enter("codeTextSequence"),a(d)}function a(d){return d===96?(e.consume(d),r++,a):(e.exit("codeTextSequence"),s(d))}function s(d){return d===null?n(d):d===32?(e.enter("space"),e.consume(d),e.exit("space"),s):d===96?(l=e.enter("codeTextSequence"),i=0,c(d)):F(d)?(e.enter("lineEnding"),e.consume(d),e.exit("lineEnding"),s):(e.enter("codeTextData"),u(d))}function u(d){return d===null||d===32||d===96||F(d)?(e.exit("codeTextData"),s(d)):(e.consume(d),u)}function c(d){return d===96?(e.consume(d),i++,c):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(d)):(l.type="codeTextData",u(d))}}class av{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&gr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),gr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),gr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);gr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);gr(this.left,n.reverse())}}}function gr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function hp(e){const t={};let n=-1,r,i,l,o,a,s,u;const c=new av(e);for(;++n<c.length;){for(;n in t;)n=t[n];if(r=c.get(n),n&&r[1].type==="chunkFlow"&&c.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,sv(c,n)),n=t[n],u=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=c.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(c.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...c.get(i)[1].start},a=c.slice(i,n),a.unshift(r),c.splice(i,n-i+1,a))}}return kt(e,0,Number.POSITIVE_INFINITY,c.slice(0)),!u}function sv(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],u={};let c,d,p=-1,f=n,k=0,C=0;const N=[C];for(;f;){for(;e.get(++i)[1]!==f;);l.push(i),f._tokenizer||(c=r.sliceStream(f),f.next||c.push(null),d&&o.defineSkip(f.start),f._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(c),f._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),d=f,f=f.next}for(f=n;++p<a.length;)a[p][0]==="exit"&&a[p-1][0]==="enter"&&a[p][1].type===a[p-1][1].type&&a[p][1].start.line!==a[p][1].end.line&&(C=p+1,N.push(C),f._tokenizer=void 0,f.previous=void 0,f=f.next);for(o.events=[],f?(f._tokenizer=void 0,f.previous=void 0):N.pop(),p=N.length;p--;){const m=a.slice(N[p],N[p+1]),y=l.pop();s.push([y,y+m.length-1]),e.splice(y,2,m)}for(s.reverse(),p=-1;++p<s.length;)u[k+s[p][0]]=k+s[p][1],k+=s[p][1]-s[p][0]-1;return u}const uv={resolve:fv,tokenize:dv},cv={partial:!0,tokenize:pv};function fv(e){return hp(e),e}function dv(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):F(a)?e.check(cv,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function pv(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),te(e,l,"linePrefix")}function l(o){if(o===null||F(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function mp(e,t,n,r,i,l,o,a,s){const u=s||Number.POSITIVE_INFINITY;let c=0;return d;function d(m){return m===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(m),e.exit(l),p):m===null||m===32||m===41||ya(m)?n(m):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),C(m))}function p(m){return m===62?(e.enter(l),e.consume(m),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),f(m))}function f(m){return m===62?(e.exit("chunkString"),e.exit(a),p(m)):m===null||m===60||F(m)?n(m):(e.consume(m),m===92?k:f)}function k(m){return m===60||m===62||m===92?(e.consume(m),f):f(m)}function C(m){return!c&&(m===null||m===41||Fe(m))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(m)):c<u&&m===40?(e.consume(m),c++,C):m===41?(e.consume(m),c--,C):m===null||m===32||m===40||ya(m)?n(m):(e.consume(m),m===92?N:C)}function N(m){return m===40||m===41||m===92?(e.consume(m),C):C(m)}}function gp(e,t,n,r,i,l){const o=this;let a=0,s;return u;function u(f){return e.enter(r),e.enter(i),e.consume(f),e.exit(i),e.enter(l),c}function c(f){return a>999||f===null||f===91||f===93&&!s||f===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(f):f===93?(e.exit(l),e.enter(i),e.consume(f),e.exit(i),e.exit(r),t):F(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),c):(e.enter("chunkString",{contentType:"string"}),d(f))}function d(f){return f===null||f===91||f===93||F(f)||a++>999?(e.exit("chunkString"),c(f)):(e.consume(f),s||(s=!q(f)),f===92?p:d)}function p(f){return f===91||f===92||f===93?(e.consume(f),a++,d):d(f)}}function yp(e,t,n,r,i,l){let o;return a;function a(p){return p===34||p===39||p===40?(e.enter(r),e.enter(i),e.consume(p),e.exit(i),o=p===40?41:p,s):n(p)}function s(p){return p===o?(e.enter(i),e.consume(p),e.exit(i),e.exit(r),t):(e.enter(l),u(p))}function u(p){return p===o?(e.exit(l),s(o)):p===null?n(p):F(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),te(e,u,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),c(p))}function c(p){return p===o||p===null||F(p)?(e.exit("chunkString"),u(p)):(e.consume(p),p===92?d:c)}function d(p){return p===o||p===92?(e.consume(p),c):c(p)}}function Lr(e,t){let n;return r;function r(i){return F(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):q(i)?te(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const hv={name:"definition",tokenize:gv},mv={partial:!0,tokenize:yv};function gv(e,t,n){const r=this;let i;return l;function l(f){return e.enter("definition"),o(f)}function o(f){return gp.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(f)}function a(f){return i=Kn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),f===58?(e.enter("definitionMarker"),e.consume(f),e.exit("definitionMarker"),s):n(f)}function s(f){return Fe(f)?Lr(e,u)(f):u(f)}function u(f){return mp(e,c,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(f)}function c(f){return e.attempt(mv,d,d)(f)}function d(f){return q(f)?te(e,p,"whitespace")(f):p(f)}function p(f){return f===null||F(f)?(e.exit("definition"),r.parser.defined.push(i),t(f)):n(f)}}function yv(e,t,n){return r;function r(a){return Fe(a)?Lr(e,i)(a):n(a)}function i(a){return yp(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return q(a)?te(e,o,"whitespace")(a):o(a)}function o(a){return a===null||F(a)?t(a):n(a)}}const vv={name:"hardBreakEscape",tokenize:xv};function xv(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return F(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const kv={name:"headingAtx",resolve:wv,tokenize:Sv};function wv(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},kt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function Sv(e,t,n){let r=0;return i;function i(c){return e.enter("atxHeading"),l(c)}function l(c){return e.enter("atxHeadingSequence"),o(c)}function o(c){return c===35&&r++<6?(e.consume(c),o):c===null||Fe(c)?(e.exit("atxHeadingSequence"),a(c)):n(c)}function a(c){return c===35?(e.enter("atxHeadingSequence"),s(c)):c===null||F(c)?(e.exit("atxHeading"),t(c)):q(c)?te(e,a,"whitespace")(c):(e.enter("atxHeadingText"),u(c))}function s(c){return c===35?(e.consume(c),s):(e.exit("atxHeadingSequence"),a(c))}function u(c){return c===null||c===35||Fe(c)?(e.exit("atxHeadingText"),a(c)):(e.consume(c),u)}}const Cv=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],dc=["pre","script","style","textarea"],Ev={concrete:!0,name:"htmlFlow",resolveTo:jv,tokenize:bv},Nv={partial:!0,tokenize:Pv},_v={partial:!0,tokenize:zv};function jv(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function bv(e,t,n){const r=this;let i,l,o,a,s;return u;function u(x){return c(x)}function c(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),d}function d(x){return x===33?(e.consume(x),p):x===47?(e.consume(x),l=!0,C):x===63?(e.consume(x),i=3,r.interrupt?t:v):yt(x)?(e.consume(x),o=String.fromCharCode(x),N):n(x)}function p(x){return x===45?(e.consume(x),i=2,f):x===91?(e.consume(x),i=5,a=0,k):yt(x)?(e.consume(x),i=4,r.interrupt?t:v):n(x)}function f(x){return x===45?(e.consume(x),r.interrupt?t:v):n(x)}function k(x){const ge="CDATA[";return x===ge.charCodeAt(a++)?(e.consume(x),a===ge.length?r.interrupt?t:D:k):n(x)}function C(x){return yt(x)?(e.consume(x),o=String.fromCharCode(x),N):n(x)}function N(x){if(x===null||x===47||x===62||Fe(x)){const ge=x===47,rt=o.toLowerCase();return!ge&&!l&&dc.includes(rt)?(i=1,r.interrupt?t(x):D(x)):Cv.includes(o.toLowerCase())?(i=6,ge?(e.consume(x),m):r.interrupt?t(x):D(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?y(x):g(x))}return x===45||We(x)?(e.consume(x),o+=String.fromCharCode(x),N):n(x)}function m(x){return x===62?(e.consume(x),r.interrupt?t:D):n(x)}function y(x){return q(x)?(e.consume(x),y):M(x)}function g(x){return x===47?(e.consume(x),M):x===58||x===95||yt(x)?(e.consume(x),S):q(x)?(e.consume(x),g):M(x)}function S(x){return x===45||x===46||x===58||x===95||We(x)?(e.consume(x),S):E(x)}function E(x){return x===61?(e.consume(x),w):q(x)?(e.consume(x),E):g(x)}function w(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,_):q(x)?(e.consume(x),w):P(x)}function _(x){return x===s?(e.consume(x),s=null,O):x===null||F(x)?n(x):(e.consume(x),_)}function P(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||Fe(x)?E(x):(e.consume(x),P)}function O(x){return x===47||x===62||q(x)?g(x):n(x)}function M(x){return x===62?(e.consume(x),A):n(x)}function A(x){return x===null||F(x)?D(x):q(x)?(e.consume(x),A):n(x)}function D(x){return x===45&&i===2?(e.consume(x),pe):x===60&&i===1?(e.consume(x),fe):x===62&&i===4?(e.consume(x),Q):x===63&&i===3?(e.consume(x),v):x===93&&i===5?(e.consume(x),R):F(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(Nv,G,Y)(x)):x===null||F(x)?(e.exit("htmlFlowData"),Y(x)):(e.consume(x),D)}function Y(x){return e.check(_v,oe,G)(x)}function oe(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),$}function $(x){return x===null||F(x)?Y(x):(e.enter("htmlFlowData"),D(x))}function pe(x){return x===45?(e.consume(x),v):D(x)}function fe(x){return x===47?(e.consume(x),o="",L):D(x)}function L(x){if(x===62){const ge=o.toLowerCase();return dc.includes(ge)?(e.consume(x),Q):D(x)}return yt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),L):D(x)}function R(x){return x===93?(e.consume(x),v):D(x)}function v(x){return x===62?(e.consume(x),Q):x===45&&i===2?(e.consume(x),v):D(x)}function Q(x){return x===null||F(x)?(e.exit("htmlFlowData"),G(x)):(e.consume(x),Q)}function G(x){return e.exit("htmlFlow"),t(x)}}function zv(e,t,n){const r=this;return i;function i(o){return F(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Pv(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Tl,t,n)}}const Tv={name:"htmlText",tokenize:Lv};function Lv(e,t,n){const r=this;let i,l,o;return a;function a(v){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(v),s}function s(v){return v===33?(e.consume(v),u):v===47?(e.consume(v),E):v===63?(e.consume(v),g):yt(v)?(e.consume(v),P):n(v)}function u(v){return v===45?(e.consume(v),c):v===91?(e.consume(v),l=0,k):yt(v)?(e.consume(v),y):n(v)}function c(v){return v===45?(e.consume(v),f):n(v)}function d(v){return v===null?n(v):v===45?(e.consume(v),p):F(v)?(o=d,fe(v)):(e.consume(v),d)}function p(v){return v===45?(e.consume(v),f):d(v)}function f(v){return v===62?pe(v):v===45?p(v):d(v)}function k(v){const Q="CDATA[";return v===Q.charCodeAt(l++)?(e.consume(v),l===Q.length?C:k):n(v)}function C(v){return v===null?n(v):v===93?(e.consume(v),N):F(v)?(o=C,fe(v)):(e.consume(v),C)}function N(v){return v===93?(e.consume(v),m):C(v)}function m(v){return v===62?pe(v):v===93?(e.consume(v),m):C(v)}function y(v){return v===null||v===62?pe(v):F(v)?(o=y,fe(v)):(e.consume(v),y)}function g(v){return v===null?n(v):v===63?(e.consume(v),S):F(v)?(o=g,fe(v)):(e.consume(v),g)}function S(v){return v===62?pe(v):g(v)}function E(v){return yt(v)?(e.consume(v),w):n(v)}function w(v){return v===45||We(v)?(e.consume(v),w):_(v)}function _(v){return F(v)?(o=_,fe(v)):q(v)?(e.consume(v),_):pe(v)}function P(v){return v===45||We(v)?(e.consume(v),P):v===47||v===62||Fe(v)?O(v):n(v)}function O(v){return v===47?(e.consume(v),pe):v===58||v===95||yt(v)?(e.consume(v),M):F(v)?(o=O,fe(v)):q(v)?(e.consume(v),O):pe(v)}function M(v){return v===45||v===46||v===58||v===95||We(v)?(e.consume(v),M):A(v)}function A(v){return v===61?(e.consume(v),D):F(v)?(o=A,fe(v)):q(v)?(e.consume(v),A):O(v)}function D(v){return v===null||v===60||v===61||v===62||v===96?n(v):v===34||v===39?(e.consume(v),i=v,Y):F(v)?(o=D,fe(v)):q(v)?(e.consume(v),D):(e.consume(v),oe)}function Y(v){return v===i?(e.consume(v),i=void 0,$):v===null?n(v):F(v)?(o=Y,fe(v)):(e.consume(v),Y)}function oe(v){return v===null||v===34||v===39||v===60||v===61||v===96?n(v):v===47||v===62||Fe(v)?O(v):(e.consume(v),oe)}function $(v){return v===47||v===62||Fe(v)?O(v):n(v)}function pe(v){return v===62?(e.consume(v),e.exit("htmlTextData"),e.exit("htmlText"),t):n(v)}function fe(v){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(v),e.exit("lineEnding"),L}function L(v){return q(v)?te(e,R,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(v):R(v)}function R(v){return e.enter("htmlTextData"),o(v)}}const js={name:"labelEnd",resolveAll:Ov,resolveTo:Dv,tokenize:Rv},Iv={tokenize:Fv},Mv={tokenize:Bv},Av={tokenize:Uv};function Ov(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&kt(e,0,e.length,n),e}function Dv(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},u={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},c={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",u,t]],a=Je(a,e.slice(l+1,l+r+3)),a=Je(a,[["enter",c,t]]),a=Je(a,_s(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=Je(a,[["exit",c,t],e[o-2],e[o-1],["exit",u,t]]),a=Je(a,e.slice(o+1)),a=Je(a,[["exit",s,t]]),kt(e,l,e.length,a),e}function Rv(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(p){return l?l._inactive?d(p):(o=r.parser.defined.includes(Kn(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(p),e.exit("labelMarker"),e.exit("labelEnd"),s):n(p)}function s(p){return p===40?e.attempt(Iv,c,o?c:d)(p):p===91?e.attempt(Mv,c,o?u:d)(p):o?c(p):d(p)}function u(p){return e.attempt(Av,c,d)(p)}function c(p){return t(p)}function d(p){return l._balanced=!0,n(p)}}function Fv(e,t,n){return r;function r(d){return e.enter("resource"),e.enter("resourceMarker"),e.consume(d),e.exit("resourceMarker"),i}function i(d){return Fe(d)?Lr(e,l)(d):l(d)}function l(d){return d===41?c(d):mp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(d)}function o(d){return Fe(d)?Lr(e,s)(d):c(d)}function a(d){return n(d)}function s(d){return d===34||d===39||d===40?yp(e,u,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(d):c(d)}function u(d){return Fe(d)?Lr(e,c)(d):c(d)}function c(d){return d===41?(e.enter("resourceMarker"),e.consume(d),e.exit("resourceMarker"),e.exit("resource"),t):n(d)}}function Bv(e,t,n){const r=this;return i;function i(a){return gp.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(Kn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Uv(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Hv={name:"labelStartImage",resolveAll:js.resolveAll,tokenize:Vv};function Vv(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const $v={name:"labelStartLink",resolveAll:js.resolveAll,tokenize:Wv};function Wv(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const uo={name:"lineEnding",tokenize:Qv};function Qv(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),te(e,t,"linePrefix")}}const Fi={name:"thematicBreak",tokenize:Kv};function Kv(e,t,n){let r=0,i;return l;function l(u){return e.enter("thematicBreak"),o(u)}function o(u){return i=u,a(u)}function a(u){return u===i?(e.enter("thematicBreakSequence"),s(u)):r>=3&&(u===null||F(u))?(e.exit("thematicBreak"),t(u)):n(u)}function s(u){return u===i?(e.consume(u),r++,s):(e.exit("thematicBreakSequence"),q(u)?te(e,a,"whitespace")(u):a(u))}}const Ie={continuation:{tokenize:Gv},exit:Zv,name:"list",tokenize:Xv},qv={partial:!0,tokenize:ex},Yv={partial:!0,tokenize:Jv};function Xv(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(f){const k=r.containerState.type||(f===42||f===43||f===45?"listUnordered":"listOrdered");if(k==="listUnordered"?!r.containerState.marker||f===r.containerState.marker:va(f)){if(r.containerState.type||(r.containerState.type=k,e.enter(k,{_container:!0})),k==="listUnordered")return e.enter("listItemPrefix"),f===42||f===45?e.check(Fi,n,u)(f):u(f);if(!r.interrupt||f===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(f)}return n(f)}function s(f){return va(f)&&++o<10?(e.consume(f),s):(!r.interrupt||o<2)&&(r.containerState.marker?f===r.containerState.marker:f===41||f===46)?(e.exit("listItemValue"),u(f)):n(f)}function u(f){return e.enter("listItemMarker"),e.consume(f),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||f,e.check(Tl,r.interrupt?n:c,e.attempt(qv,p,d))}function c(f){return r.containerState.initialBlankLine=!0,l++,p(f)}function d(f){return q(f)?(e.enter("listItemPrefixWhitespace"),e.consume(f),e.exit("listItemPrefixWhitespace"),p):n(f)}function p(f){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(f)}}function Gv(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Tl,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,te(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!q(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Yv,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,te(e,e.attempt(Ie,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function Jv(e,t,n){const r=this;return te(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function Zv(e){e.exit(this.containerState.type)}function ex(e,t,n){const r=this;return te(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!q(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const pc={name:"setextUnderline",resolveTo:tx,tokenize:nx};function tx(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function nx(e,t,n){const r=this;let i;return l;function l(u){let c=r.events.length,d;for(;c--;)if(r.events[c][1].type!=="lineEnding"&&r.events[c][1].type!=="linePrefix"&&r.events[c][1].type!=="content"){d=r.events[c][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||d)?(e.enter("setextHeadingLine"),i=u,o(u)):n(u)}function o(u){return e.enter("setextHeadingLineSequence"),a(u)}function a(u){return u===i?(e.consume(u),a):(e.exit("setextHeadingLineSequence"),q(u)?te(e,s,"lineSuffix")(u):s(u))}function s(u){return u===null||F(u)?(e.exit("setextHeadingLine"),t(u)):n(u)}}const rx={tokenize:ix};function ix(e){const t=this,n=e.attempt(Tl,r,e.attempt(this.parser.constructs.flowInitial,i,te(e,e.attempt(this.parser.constructs.flow,i,e.attempt(uv,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const lx={resolveAll:xp()},ox=vp("string"),ax=vp("text");function vp(e){return{resolveAll:xp(e==="text"?sx:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(c){return u(c)?l(c):a(c)}function a(c){if(c===null){n.consume(c);return}return n.enter("data"),n.consume(c),s}function s(c){return u(c)?(n.exit("data"),l(c)):(n.consume(c),s)}function u(c){if(c===null)return!0;const d=i[c];let p=-1;if(d)for(;++p<d.length;){const f=d[p];if(!f.previous||f.previous.call(r,r.previous))return!0}return!1}}}function xp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function sx(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const u=i[l];if(typeof u=="string"){for(o=u.length;u.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(u===-2)s=!0,a++;else if(u!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const u={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...u.start},r.start.offset===r.end.offset?Object.assign(r,u):(e.splice(n,0,["enter",u,t],["exit",u,t]),n+=2)}n++}return e}const ux={42:Ie,43:Ie,45:Ie,48:Ie,49:Ie,50:Ie,51:Ie,52:Ie,53:Ie,54:Ie,55:Ie,56:Ie,57:Ie,62:fp},cx={91:hv},fx={[-2]:so,[-1]:so,32:so},dx={35:kv,42:Fi,45:[pc,Fi],60:Ev,61:pc,95:Fi,96:fc,126:fc},px={38:pp,92:dp},hx={[-5]:uo,[-4]:uo,[-3]:uo,33:Hv,38:pp,42:xa,60:[$y,Tv],91:$v,92:[vv,dp],93:js,95:xa,96:rv},mx={null:[xa,lx]},gx={null:[42,95]},yx={null:[]},vx=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:gx,contentInitial:cx,disable:yx,document:ux,flow:dx,flowInitial:fx,insideSpan:mx,string:px,text:hx},Symbol.toStringTag,{value:"Module"}));function xx(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:_(E),check:_(w),consume:y,enter:g,exit:S,interrupt:_(w,{interrupt:!0})},u={code:null,containerState:{},defineSkip:C,events:[],now:k,parser:e,previous:null,sliceSerialize:p,sliceStream:f,write:d};let c=t.tokenize.call(u,s);return t.resolveAll&&l.push(t),u;function d(A){return o=Je(o,A),N(),o[o.length-1]!==null?[]:(P(t,0),u.events=_s(l,u.events,u),u.events)}function p(A,D){return wx(f(A),D)}function f(A){return kx(o,A)}function k(){const{_bufferIndex:A,_index:D,line:Y,column:oe,offset:$}=r;return{_bufferIndex:A,_index:D,line:Y,column:oe,offset:$}}function C(A){i[A.line]=A.column,M()}function N(){let A;for(;r._index<o.length;){const D=o[r._index];if(typeof D=="string")for(A=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===A&&r._bufferIndex<D.length;)m(D.charCodeAt(r._bufferIndex));else m(D)}}function m(A){c=c(A)}function y(A){F(A)?(r.line++,r.column=1,r.offset+=A===-3?2:1,M()):A!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),u.previous=A}function g(A,D){const Y=D||{};return Y.type=A,Y.start=k(),u.events.push(["enter",Y,u]),a.push(Y),Y}function S(A){const D=a.pop();return D.end=k(),u.events.push(["exit",D,u]),D}function E(A,D){P(A,D.from)}function w(A,D){D.restore()}function _(A,D){return Y;function Y(oe,$,pe){let fe,L,R,v;return Array.isArray(oe)?G(oe):"tokenize"in oe?G([oe]):Q(oe);function Q(ne){return dt;function dt(At){const Cn=At!==null&&ne[At],En=At!==null&&ne.null,li=[...Array.isArray(Cn)?Cn:Cn?[Cn]:[],...Array.isArray(En)?En:En?[En]:[]];return G(li)(At)}}function G(ne){return fe=ne,L=0,ne.length===0?pe:x(ne[L])}function x(ne){return dt;function dt(At){return v=O(),R=ne,ne.partial||(u.currentConstruct=ne),ne.name&&u.parser.constructs.disable.null.includes(ne.name)?rt():ne.tokenize.call(D?Object.assign(Object.create(u),D):u,s,ge,rt)(At)}}function ge(ne){return A(R,v),$}function rt(ne){return v.restore(),++L<fe.length?x(fe[L]):pe}}}function P(A,D){A.resolveAll&&!l.includes(A)&&l.push(A),A.resolve&&kt(u.events,D,u.events.length-D,A.resolve(u.events.slice(D),u)),A.resolveTo&&(u.events=A.resolveTo(u.events,u))}function O(){const A=k(),D=u.previous,Y=u.currentConstruct,oe=u.events.length,$=Array.from(a);return{from:oe,restore:pe};function pe(){r=A,u.previous=D,u.currentConstruct=Y,u.events.length=oe,a=$,M()}}function M(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function kx(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function wx(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function Sx(e){const r={constructs:zy([vx,...(e||{}).extensions||[]]),content:i(Dy),defined:[],document:i(Fy),flow:i(rx),lazy:{},string:i(ox),text:i(ax)};return r;function i(l){return o;function o(a){return xx(r,l,a)}}}function Cx(e){for(;!hp(e););return e}const hc=/[\0\t\n\r]/g;function Ex(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let u,c,d,p,f;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),d=0,t="",n&&(l.charCodeAt(0)===65279&&d++,n=void 0);d<l.length;){if(hc.lastIndex=d,u=hc.exec(l),p=u&&u.index!==void 0?u.index:l.length,f=l.charCodeAt(p),!u){t=l.slice(d);break}if(f===10&&d===p&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),d<p&&(s.push(l.slice(d,p)),e+=p-d),f){case 0:{s.push(65533),e++;break}case 9:{for(c=Math.ceil(e/4)*4,s.push(-2);e++<c;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}d=p+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const Nx=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function _x(e){return e.replace(Nx,jx)}function jx(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return cp(n.slice(l?2:1),l?16:10)}return Ns(n)||e}const kp={}.hasOwnProperty;function bx(e,t,n){return typeof t!="string"&&(n=t,t=void 0),zx(n)(Cx(Sx(n).document().write(Ex()(e,t,!0))))}function zx(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Os),autolinkProtocol:O,autolinkEmail:O,atxHeading:l(Is),blockQuote:l(En),characterEscape:O,characterReference:O,codeFenced:l(li),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(li,o),codeText:l(Lp,o),codeTextData:O,data:O,codeFlowValue:O,definition:l(Ip),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Mp),hardBreakEscape:l(Ms),hardBreakTrailing:l(Ms),htmlFlow:l(As,o),htmlFlowData:O,htmlText:l(As,o),htmlTextData:O,image:l(Ap),label:o,link:l(Os),listItem:l(Op),listItemValue:p,listOrdered:l(Ds,d),listUnordered:l(Ds),paragraph:l(Dp),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Is),strong:l(Rp),thematicBreak:l(Bp)},exit:{atxHeading:s(),atxHeadingSequence:E,autolink:s(),autolinkEmail:Cn,autolinkProtocol:At,blockQuote:s(),characterEscapeValue:M,characterReferenceMarkerHexadecimal:rt,characterReferenceMarkerNumeric:rt,characterReferenceValue:ne,characterReference:dt,codeFenced:s(N),codeFencedFence:C,codeFencedFenceInfo:f,codeFencedFenceMeta:k,codeFlowValue:M,codeIndented:s(m),codeText:s($),codeTextData:M,data:M,definition:s(),definitionDestinationString:S,definitionLabelString:y,definitionTitleString:g,emphasis:s(),hardBreakEscape:s(D),hardBreakTrailing:s(D),htmlFlow:s(Y),htmlFlowData:M,htmlText:s(oe),htmlTextData:M,image:s(fe),label:R,labelText:L,lineEnding:A,link:s(pe),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:ge,resourceDestinationString:v,resourceTitleString:Q,resource:G,setextHeading:s(P),setextHeadingLineSequence:_,setextHeadingText:w,strong:s(),thematicBreak:s()}};wp(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(j){let T={type:"root",children:[]};const B={stack:[T],tokenStack:[],config:t,enter:a,exit:u,buffer:o,resume:c,data:n},W=[];let J=-1;for(;++J<j.length;)if(j[J][1].type==="listOrdered"||j[J][1].type==="listUnordered")if(j[J][0]==="enter")W.push(J);else{const it=W.pop();J=i(j,it,J)}for(J=-1;++J<j.length;){const it=t[j[J][0]];kp.call(it,j[J][1].type)&&it[j[J][1].type].call(Object.assign({sliceSerialize:j[J][2].sliceSerialize},B),j[J][1])}if(B.tokenStack.length>0){const it=B.tokenStack[B.tokenStack.length-1];(it[1]||mc).call(B,void 0,it[0])}for(T.position={start:Dt(j.length>0?j[0][1].start:{line:1,column:1,offset:0}),end:Dt(j.length>0?j[j.length-2][1].end:{line:1,column:1,offset:0})},J=-1;++J<t.transforms.length;)T=t.transforms[J](T)||T;return T}function i(j,T,B){let W=T-1,J=-1,it=!1,ln,wt,or,ar;for(;++W<=B;){const Ue=j[W];switch(Ue[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Ue[0]==="enter"?J++:J--,ar=void 0;break}case"lineEndingBlank":{Ue[0]==="enter"&&(ln&&!ar&&!J&&!or&&(or=W),ar=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:ar=void 0}if(!J&&Ue[0]==="enter"&&Ue[1].type==="listItemPrefix"||J===-1&&Ue[0]==="exit"&&(Ue[1].type==="listUnordered"||Ue[1].type==="listOrdered")){if(ln){let Nn=W;for(wt=void 0;Nn--;){const St=j[Nn];if(St[1].type==="lineEnding"||St[1].type==="lineEndingBlank"){if(St[0]==="exit")continue;wt&&(j[wt][1].type="lineEndingBlank",it=!0),St[1].type="lineEnding",wt=Nn}else if(!(St[1].type==="linePrefix"||St[1].type==="blockQuotePrefix"||St[1].type==="blockQuotePrefixWhitespace"||St[1].type==="blockQuoteMarker"||St[1].type==="listItemIndent"))break}or&&(!wt||or<wt)&&(ln._spread=!0),ln.end=Object.assign({},wt?j[wt][1].start:Ue[1].end),j.splice(wt||W,0,["exit",ln,Ue[2]]),W++,B++}if(Ue[1].type==="listItemPrefix"){const Nn={type:"listItem",_spread:!1,start:Object.assign({},Ue[1].start),end:void 0};ln=Nn,j.splice(W,0,["enter",Nn,Ue[2]]),W++,B++,or=void 0,ar=!0}}}return j[T][1]._spread=it,B}function l(j,T){return B;function B(W){a.call(this,j(W),W),T&&T.call(this,W)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(j,T,B){this.stack[this.stack.length-1].children.push(j),this.stack.push(j),this.tokenStack.push([T,B||void 0]),j.position={start:Dt(T.start),end:void 0}}function s(j){return T;function T(B){j&&j.call(this,B),u.call(this,B)}}function u(j,T){const B=this.stack.pop(),W=this.tokenStack.pop();if(W)W[0].type!==j.type&&(T?T.call(this,j,W[0]):(W[1]||mc).call(this,j,W[0]));else throw new Error("Cannot close `"+j.type+"` ("+Tr({start:j.start,end:j.end})+"): it’s not open");B.position.end=Dt(j.end)}function c(){return jy(this.stack.pop())}function d(){this.data.expectingFirstListItemValue=!0}function p(j){if(this.data.expectingFirstListItemValue){const T=this.stack[this.stack.length-2];T.start=Number.parseInt(this.sliceSerialize(j),10),this.data.expectingFirstListItemValue=void 0}}function f(){const j=this.resume(),T=this.stack[this.stack.length-1];T.lang=j}function k(){const j=this.resume(),T=this.stack[this.stack.length-1];T.meta=j}function C(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function N(){const j=this.resume(),T=this.stack[this.stack.length-1];T.value=j.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function m(){const j=this.resume(),T=this.stack[this.stack.length-1];T.value=j.replace(/(\r?\n|\r)$/g,"")}function y(j){const T=this.resume(),B=this.stack[this.stack.length-1];B.label=T,B.identifier=Kn(this.sliceSerialize(j)).toLowerCase()}function g(){const j=this.resume(),T=this.stack[this.stack.length-1];T.title=j}function S(){const j=this.resume(),T=this.stack[this.stack.length-1];T.url=j}function E(j){const T=this.stack[this.stack.length-1];if(!T.depth){const B=this.sliceSerialize(j).length;T.depth=B}}function w(){this.data.setextHeadingSlurpLineEnding=!0}function _(j){const T=this.stack[this.stack.length-1];T.depth=this.sliceSerialize(j).codePointAt(0)===61?1:2}function P(){this.data.setextHeadingSlurpLineEnding=void 0}function O(j){const B=this.stack[this.stack.length-1].children;let W=B[B.length-1];(!W||W.type!=="text")&&(W=Fp(),W.position={start:Dt(j.start),end:void 0},B.push(W)),this.stack.push(W)}function M(j){const T=this.stack.pop();T.value+=this.sliceSerialize(j),T.position.end=Dt(j.end)}function A(j){const T=this.stack[this.stack.length-1];if(this.data.atHardBreak){const B=T.children[T.children.length-1];B.position.end=Dt(j.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(T.type)&&(O.call(this,j),M.call(this,j))}function D(){this.data.atHardBreak=!0}function Y(){const j=this.resume(),T=this.stack[this.stack.length-1];T.value=j}function oe(){const j=this.resume(),T=this.stack[this.stack.length-1];T.value=j}function $(){const j=this.resume(),T=this.stack[this.stack.length-1];T.value=j}function pe(){const j=this.stack[this.stack.length-1];if(this.data.inReference){const T=this.data.referenceType||"shortcut";j.type+="Reference",j.referenceType=T,delete j.url,delete j.title}else delete j.identifier,delete j.label;this.data.referenceType=void 0}function fe(){const j=this.stack[this.stack.length-1];if(this.data.inReference){const T=this.data.referenceType||"shortcut";j.type+="Reference",j.referenceType=T,delete j.url,delete j.title}else delete j.identifier,delete j.label;this.data.referenceType=void 0}function L(j){const T=this.sliceSerialize(j),B=this.stack[this.stack.length-2];B.label=_x(T),B.identifier=Kn(T).toLowerCase()}function R(){const j=this.stack[this.stack.length-1],T=this.resume(),B=this.stack[this.stack.length-1];if(this.data.inReference=!0,B.type==="link"){const W=j.children;B.children=W}else B.alt=T}function v(){const j=this.resume(),T=this.stack[this.stack.length-1];T.url=j}function Q(){const j=this.resume(),T=this.stack[this.stack.length-1];T.title=j}function G(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function ge(j){const T=this.resume(),B=this.stack[this.stack.length-1];B.label=T,B.identifier=Kn(this.sliceSerialize(j)).toLowerCase(),this.data.referenceType="full"}function rt(j){this.data.characterReferenceType=j.type}function ne(j){const T=this.sliceSerialize(j),B=this.data.characterReferenceType;let W;B?(W=cp(T,B==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):W=Ns(T);const J=this.stack[this.stack.length-1];J.value+=W}function dt(j){const T=this.stack.pop();T.position.end=Dt(j.end)}function At(j){M.call(this,j);const T=this.stack[this.stack.length-1];T.url=this.sliceSerialize(j)}function Cn(j){M.call(this,j);const T=this.stack[this.stack.length-1];T.url="mailto:"+this.sliceSerialize(j)}function En(){return{type:"blockquote",children:[]}}function li(){return{type:"code",lang:null,meta:null,value:""}}function Lp(){return{type:"inlineCode",value:""}}function Ip(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Mp(){return{type:"emphasis",children:[]}}function Is(){return{type:"heading",depth:0,children:[]}}function Ms(){return{type:"break"}}function As(){return{type:"html",value:""}}function Ap(){return{type:"image",title:null,url:"",alt:null}}function Os(){return{type:"link",title:null,url:"",children:[]}}function Ds(j){return{type:"list",ordered:j.type==="listOrdered",start:null,spread:j._spread,children:[]}}function Op(j){return{type:"listItem",spread:j._spread,checked:null,children:[]}}function Dp(){return{type:"paragraph",children:[]}}function Rp(){return{type:"strong",children:[]}}function Fp(){return{type:"text",value:""}}function Bp(){return{type:"thematicBreak"}}}function Dt(e){return{line:e.line,column:e.column,offset:e.offset}}function wp(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?wp(e,r):Px(e,r)}}function Px(e,t){let n;for(n in t)if(kp.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function mc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Tr({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Tr({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Tr({start:t.start,end:t.end})+") is still open")}function Tx(e){const t=this;t.parser=n;function n(r){return bx(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function Lx(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function Ix(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function Mx(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function Ax(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Ox(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Dx(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=lr(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const u={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,u),e.applyData(t,u)}function Rx(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Fx(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Sp(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function Bx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Sp(e,t);const i={src:lr(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function Ux(e,t){const n={src:lr(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function Hx(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function Vx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Sp(e,t);const i={href:lr(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function $x(e,t){const n={href:lr(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function Wx(e,t,n){const r=e.all(t),i=n?Qx(n):Cp(t),l={},o=[];if(typeof t.checked=="boolean"){const c=r[0];let d;c&&c.type==="element"&&c.tagName==="p"?d=c:(d={type:"element",tagName:"p",properties:{},children:[]},r.unshift(d)),d.children.length>0&&d.children.unshift({type:"text",value:" "}),d.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const c=r[a];(i||a!==0||c.type!=="element"||c.tagName!=="p")&&o.push({type:"text",value:`
`}),c.type==="element"&&c.tagName==="p"&&!i?o.push(...c.children):o.push(c)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const u={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,u),e.applyData(t,u)}function Qx(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Cp(n[r])}return t}function Cp(e){const t=e.spread;return t??e.children.length>1}function Kx(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function qx(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Yx(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function Xx(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Gx(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=ws(t.children[1]),s=rp(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function Jx(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const u=[];for(;++s<a;){const d=t.children[s],p={},f=o?o[s]:void 0;f&&(p.align=f);let k={type:"element",tagName:l,properties:p,children:[]};d&&(k.children=e.all(d),e.patch(d,k),k=e.applyData(d,k)),u.push(k)}const c={type:"element",tagName:"tr",properties:{},children:e.wrap(u,!0)};return e.patch(t,c),e.applyData(t,c)}function Zx(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const gc=9,yc=32;function e1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(vc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(vc(t.slice(i),i>0,!1)),l.join("")}function vc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===gc||l===yc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===gc||l===yc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function t1(e,t){const n={type:"text",value:e1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function n1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const r1={blockquote:Lx,break:Ix,code:Mx,delete:Ax,emphasis:Ox,footnoteReference:Dx,heading:Rx,html:Fx,imageReference:Bx,image:Ux,inlineCode:Hx,linkReference:Vx,link:$x,listItem:Wx,list:Kx,paragraph:qx,root:Yx,strong:Xx,table:Gx,tableCell:Zx,tableRow:Jx,text:t1,thematicBreak:n1,toml:Ei,yaml:Ei,definition:Ei,footnoteDefinition:Ei};function Ei(){}const Ep=-1,Ll=0,Ir=1,dl=2,bs=3,zs=4,Ps=5,Ts=6,Np=7,_p=8,xc=typeof self=="object"?self:globalThis,i1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Ll:case Ep:return n(o,i);case Ir:{const a=n([],i);for(const s of o)a.push(r(s));return a}case dl:{const a=n({},i);for(const[s,u]of o)a[r(s)]=r(u);return a}case bs:return n(new Date(o),i);case zs:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Ps:{const a=n(new Map,i);for(const[s,u]of o)a.set(r(s),r(u));return a}case Ts:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Np:{const{name:a,message:s}=o;return n(new xc[a](s),i)}case _p:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new xc[l](o),i)};return r},kc=e=>i1(new Map,e)(0),jn="",{toString:l1}={},{keys:o1}=Object,yr=e=>{const t=typeof e;if(t!=="object"||!e)return[Ll,t];const n=l1.call(e).slice(8,-1);switch(n){case"Array":return[Ir,jn];case"Object":return[dl,jn];case"Date":return[bs,jn];case"RegExp":return[zs,jn];case"Map":return[Ps,jn];case"Set":return[Ts,jn];case"DataView":return[Ir,n]}return n.includes("Array")?[Ir,n]:n.includes("Error")?[Np,n]:[dl,n]},Ni=([e,t])=>e===Ll&&(t==="function"||t==="symbol"),a1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=yr(o);switch(a){case Ll:{let c=o;switch(s){case"bigint":a=_p,c=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);c=null;break;case"undefined":return i([Ep],o)}return i([a,c],o)}case Ir:{if(s){let p=o;return s==="DataView"?p=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(p=new Uint8Array(o)),i([s,[...p]],o)}const c=[],d=i([a,c],o);for(const p of o)c.push(l(p));return d}case dl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const c=[],d=i([a,c],o);for(const p of o1(o))(e||!Ni(yr(o[p])))&&c.push([l(p),l(o[p])]);return d}case bs:return i([a,o.toISOString()],o);case zs:{const{source:c,flags:d}=o;return i([a,{source:c,flags:d}],o)}case Ps:{const c=[],d=i([a,c],o);for(const[p,f]of o)(e||!(Ni(yr(p))||Ni(yr(f))))&&c.push([l(p),l(f)]);return d}case Ts:{const c=[],d=i([a,c],o);for(const p of o)(e||!Ni(yr(p)))&&c.push(l(p));return d}}const{message:u}=o;return i([a,{name:s,message:u}],o)};return l},wc=(e,{json:t,lossy:n}={})=>{const r=[];return a1(!(t||n),!!t,new Map,r)(e),r},pl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?kc(wc(e,t)):structuredClone(e):(e,t)=>kc(wc(e,t));function s1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function u1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function c1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||s1,r=e.options.footnoteBackLabel||u1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const u=e.footnoteById.get(e.footnoteOrder[s]);if(!u)continue;const c=e.all(u),d=String(u.identifier).toUpperCase(),p=lr(d.toLowerCase());let f=0;const k=[],C=e.footnoteCounts.get(d);for(;C!==void 0&&++f<=C;){k.length>0&&k.push({type:"text",value:" "});let y=typeof n=="string"?n:n(s,f);typeof y=="string"&&(y={type:"text",value:y}),k.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+p+(f>1?"-"+f:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,f),className:["data-footnote-backref"]},children:Array.isArray(y)?y:[y]})}const N=c[c.length-1];if(N&&N.type==="element"&&N.tagName==="p"){const y=N.children[N.children.length-1];y&&y.type==="text"?y.value+=" ":N.children.push({type:"text",value:" "}),N.children.push(...k)}else c.push(...k);const m={type:"element",tagName:"li",properties:{id:t+"fn-"+p},children:e.wrap(c,!0)};e.patch(u,m),a.push(m)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...pl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const jp=function(e){if(e==null)return h1;if(typeof e=="function")return Il(e);if(typeof e=="object")return Array.isArray(e)?f1(e):d1(e);if(typeof e=="string")return p1(e);throw new Error("Expected function, string, or object as test")};function f1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=jp(e[n]);return Il(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function d1(e){const t=e;return Il(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function p1(e){return Il(t);function t(n){return n&&n.type===e}}function Il(e){return t;function t(n,r,i){return!!(m1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function h1(){return!0}function m1(e){return e!==null&&typeof e=="object"&&"type"in e}const bp=[],g1=!0,Sc=!1,y1="skip";function v1(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=jp(i),o=r?-1:1;a(e,void 0,[])();function a(s,u,c){const d=s&&typeof s=="object"?s:{};if(typeof d.type=="string"){const f=typeof d.tagName=="string"?d.tagName:typeof d.name=="string"?d.name:void 0;Object.defineProperty(p,"name",{value:"node ("+(s.type+(f?"<"+f+">":""))+")"})}return p;function p(){let f=bp,k,C,N;if((!t||l(s,u,c[c.length-1]||void 0))&&(f=x1(n(s,c)),f[0]===Sc))return f;if("children"in s&&s.children){const m=s;if(m.children&&f[0]!==y1)for(C=(r?m.children.length:-1)+o,N=c.concat(m);C>-1&&C<m.children.length;){const y=m.children[C];if(k=a(y,C,N)(),k[0]===Sc)return k;C=typeof k[1]=="number"?k[1]:C+o}}return f}}}function x1(e){return Array.isArray(e)?e:typeof e=="number"?[g1,e]:e==null?bp:[e]}function zp(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),v1(e,l,a,i);function a(s,u){const c=u[u.length-1],d=c?c.children.indexOf(s):void 0;return o(s,d,c)}}const ka={}.hasOwnProperty,k1={};function w1(e,t){const n=t||k1,r=new Map,i=new Map,l=new Map,o={...r1,...n.handlers},a={all:u,applyData:C1,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:S1,wrap:N1};return zp(e,function(c){if(c.type==="definition"||c.type==="footnoteDefinition"){const d=c.type==="definition"?r:i,p=String(c.identifier).toUpperCase();d.has(p)||d.set(p,c)}}),a;function s(c,d){const p=c.type,f=a.handlers[p];if(ka.call(a.handlers,p)&&f)return f(a,c,d);if(a.options.passThrough&&a.options.passThrough.includes(p)){if("children"in c){const{children:C,...N}=c,m=pl(N);return m.children=a.all(c),m}return pl(c)}return(a.options.unknownHandler||E1)(a,c,d)}function u(c){const d=[];if("children"in c){const p=c.children;let f=-1;for(;++f<p.length;){const k=a.one(p[f],c);if(k){if(f&&p[f-1].type==="break"&&(!Array.isArray(k)&&k.type==="text"&&(k.value=Cc(k.value)),!Array.isArray(k)&&k.type==="element")){const C=k.children[0];C&&C.type==="text"&&(C.value=Cc(C.value))}Array.isArray(k)?d.push(...k):d.push(k)}}}return d}}function S1(e,t){e.position&&(t.position=ly(e))}function C1(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,pl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function E1(e,t){const n=t.data||{},r="value"in t&&!(ka.call(n,"hProperties")||ka.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function N1(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Cc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Ec(e,t){const n=w1(e,t),r=n.one(e,void 0),i=c1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function _1(e,t){return e&&"run"in e?async function(n,r){const i=Ec(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Ec(n,{file:r,...e||t})}}function Nc(e){if(e)throw e}var Bi=Object.prototype.hasOwnProperty,Pp=Object.prototype.toString,_c=Object.defineProperty,jc=Object.getOwnPropertyDescriptor,bc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Pp.call(t)==="[object Array]"},zc=function(t){if(!t||Pp.call(t)!=="[object Object]")return!1;var n=Bi.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Bi.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Bi.call(t,i)},Pc=function(t,n){_c&&n.name==="__proto__"?_c(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Tc=function(t,n){if(n==="__proto__")if(Bi.call(t,n)){if(jc)return jc(t,n).value}else return;return t[n]},j1=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,u=arguments.length,c=!1;for(typeof a=="boolean"&&(c=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<u;++s)if(t=arguments[s],t!=null)for(n in t)r=Tc(a,n),i=Tc(t,n),a!==i&&(c&&i&&(zc(i)||(l=bc(i)))?(l?(l=!1,o=r&&bc(r)?r:[]):o=r&&zc(r)?r:{},Pc(a,{name:n,newValue:e(c,o,i)})):typeof i<"u"&&Pc(a,{name:n,newValue:i}));return a};const co=Ca(j1);function wa(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function b1(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...u){const c=e[++l];let d=-1;if(s){o(s);return}for(;++d<i.length;)(u[d]===null||u[d]===void 0)&&(u[d]=i[d]);i=u,c?z1(c,a)(...u):o(null,...u)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function z1(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(u){const c=u;if(a&&n)throw c;return i(c)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const mt={basename:P1,dirname:T1,extname:L1,join:I1,sep:"/"};function P1(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');ii(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function T1(e){if(ii(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function L1(e){ii(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function I1(...e){let t=-1,n;for(;++t<e.length;)ii(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":M1(n)}function M1(e){ii(e);const t=e.codePointAt(0)===47;let n=A1(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function A1(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function ii(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const O1={cwd:D1};function D1(){return"/"}function Sa(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function R1(e){if(typeof e=="string")e=new URL(e);else if(!Sa(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return F1(e)}function F1(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const fo=["history","path","basename","stem","extname","dirname"];class Tp{constructor(t){let n;t?Sa(t)?n={path:t}:typeof t=="string"||B1(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":O1.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<fo.length;){const l=fo[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)fo.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?mt.basename(this.path):void 0}set basename(t){ho(t,"basename"),po(t,"basename"),this.path=mt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?mt.dirname(this.path):void 0}set dirname(t){Lc(this.basename,"dirname"),this.path=mt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?mt.extname(this.path):void 0}set extname(t){if(po(t,"extname"),Lc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=mt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){Sa(t)&&(t=R1(t)),ho(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?mt.basename(this.path,this.extname):void 0}set stem(t){ho(t,"stem"),po(t,"stem"),this.path=mt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new be(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function po(e,t){if(e&&e.includes(mt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+mt.sep+"`")}function ho(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Lc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function B1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const U1=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},H1={}.hasOwnProperty;class Ls extends U1{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=b1()}copy(){const t=new Ls;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(co(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(yo("data",this.frozen),this.namespace[t]=n,this):H1.call(this.namespace,t)&&this.namespace[t]||void 0:t?(yo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=_i(t),r=this.parser||this.Parser;return mo("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),mo("process",this.parser||this.Parser),go("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=_i(t),s=r.parse(a);r.run(s,a,function(c,d,p){if(c||!d||!p)return u(c);const f=d,k=r.stringify(f,p);W1(k)?p.value=k:p.result=k,u(c,p)});function u(c,d){c||!d?o(c):l?l(d):n(void 0,d)}}}processSync(t){let n=!1,r;return this.freeze(),mo("processSync",this.parser||this.Parser),go("processSync",this.compiler||this.Compiler),this.process(t,i),Mc("processSync","process",n),r;function i(l,o){n=!0,Nc(l),r=o}}run(t,n,r){Ic(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=_i(n);i.run(t,s,u);function u(c,d,p){const f=d||t;c?a(c):o?o(f):r(void 0,f,p)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Mc("runSync","run",r),i;function l(o,a){Nc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=_i(n),i=this.compiler||this.Compiler;return go("stringify",i),Ic(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(yo("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(u){if(typeof u=="function")s(u,[]);else if(typeof u=="object")if(Array.isArray(u)){const[c,...d]=u;s(c,d)}else o(u);else throw new TypeError("Expected usable value, not `"+u+"`")}function o(u){if(!("plugins"in u)&&!("settings"in u))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(u.plugins),u.settings&&(i.settings=co(!0,i.settings,u.settings))}function a(u){let c=-1;if(u!=null)if(Array.isArray(u))for(;++c<u.length;){const d=u[c];l(d)}else throw new TypeError("Expected a list of plugins, not `"+u+"`")}function s(u,c){let d=-1,p=-1;for(;++d<r.length;)if(r[d][0]===u){p=d;break}if(p===-1)r.push([u,...c]);else if(c.length>0){let[f,...k]=c;const C=r[p][1];wa(C)&&wa(f)&&(f=co(!0,C,f)),r[p]=[u,f,...k]}}}}const V1=new Ls().freeze();function mo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function go(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function yo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Ic(e){if(!wa(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Mc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function _i(e){return $1(e)?e:new Tp(e)}function $1(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function W1(e){return typeof e=="string"||Q1(e)}function Q1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const K1="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Ac=[],Oc={allowDangerousHtml:!0},q1=/^(https?|ircs?|mailto|xmpp)$/i,Y1=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function X1(e){const t=G1(e),n=J1(e);return Z1(t.runSync(t.parse(n),n),e)}function G1(e){const t=e.rehypePlugins||Ac,n=e.remarkPlugins||Ac,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Oc}:Oc;return V1().use(Tx).use(n).use(_1,r).use(t)}function J1(e){const t=e.children||"",n=new Tp;return typeof t=="string"&&(n.value=t),n}function Z1(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||e0;for(const c of Y1)Object.hasOwn(t,c.from)&&(""+c.from+(c.to?"use `"+c.to+"` instead":"remove it")+K1+c.id,void 0);return zp(e,u),cy(e,{Fragment:h.Fragment,components:i,ignoreInvalidStyle:!0,jsx:h.jsx,jsxs:h.jsxs,passKeys:!0,passNode:!0});function u(c,d,p){if(c.type==="raw"&&p&&typeof d=="number")return o?p.children.splice(d,1):p.children[d]={type:"text",value:c.value},d;if(c.type==="element"){let f;for(f in ao)if(Object.hasOwn(ao,f)&&Object.hasOwn(c.properties,f)){const k=c.properties[f],C=ao[f];(C===null||C.includes(c.tagName))&&(c.properties[f]=s(String(k||""),f,c))}}if(c.type==="element"){let f=n?!n.includes(c.tagName):l?l.includes(c.tagName):!1;if(!f&&r&&typeof d=="number"&&(f=!r(c,d,p)),f&&p&&typeof d=="number")return a&&c.children?p.children.splice(d,1,...c.children):p.children.splice(d,1),d}}}function e0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||q1.test(e.slice(0,t))?e:""}const _t={send:h.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),h.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),h.jsx("polyline",{points:"14 2 14 8 20 8"}),h.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),h.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("circle",{cx:"12",cy:"12",r:"10"}),h.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),h.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:h.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:h.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),h.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),h.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:h.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),h.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:h.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),h.jsx("circle",{cx:"12",cy:"5",r:"2"}),h.jsx("path",{d:"M12 7v4"})]})},t0=e=>{switch(e){case"directive":return _t.directive;case"question":return _t.question;case"status":return _t.status;case"result":return _t.result;case"approval_request":return _t.lock;default:return _t.directive}},n0=({threadId:e,messages:t,onSendMessage:n})=>{const r=H.useRef(null),[i,l]=un.useState(""),[o,a]=un.useState("directive"),[s,u]=un.useState(""),[c,d]=un.useState(!1);H.useEffect(()=>{var N;(N=r.current)==null||N.scrollIntoView({behavior:"smooth"})},[t]);const p=()=>{i.trim()&&(n(i,o,s||void 0),l(""))},f=N=>{N.key==="Enter"&&!N.shiftKey&&(N.preventDefault(),p())},k=N=>new Date(N).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),C=N=>N.length>12?`${N.slice(0,8)}...`:N;return h.jsxs("div",{className:"conversation-view",children:[h.jsxs("div",{className:"conversation-header",children:[h.jsxs("div",{className:"header-info",children:[h.jsx("span",{className:"thread-label",children:"Thread"}),h.jsx("span",{className:"thread-id",title:e,children:C(e)})]}),h.jsx("div",{className:"header-stats",children:h.jsxs("span",{className:"message-count",children:[t.length," messages"]})})]}),h.jsxs("div",{className:"messages-container",children:[t.length===0?h.jsxs("div",{className:"empty-messages",children:[h.jsx("div",{className:"empty-icon",children:h.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:h.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),h.jsx("p",{children:"No messages yet"}),h.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((N,m)=>{const y=N.from_type==="human",g=m===0||t[m-1].from_type!==N.from_type;return h.jsxs("div",{className:`message ${y?"human":"agent"}`,children:[h.jsx("div",{className:`message-avatar ${g?"visible":""}`,children:g&&(y?_t.user:_t.bot)}),h.jsxs("div",{className:"message-body",children:[g&&h.jsxs("div",{className:"message-meta",children:[h.jsx("span",{className:"sender-name",children:N.from_id}),h.jsxs("span",{className:"kind-badge",children:[t0(N.kind)," ",N.kind]}),h.jsx("span",{className:"message-time",children:k(N.created_at)})]}),h.jsx("div",{className:"message-content",children:N.kind==="result"||!y?h.jsx(X1,{components:{a:({href:S,children:E})=>h.jsx("a",{href:S,target:"_blank",rel:"noopener noreferrer",children:E}),code:({className:S,children:E,...w})=>!S?h.jsx("code",{className:"inline-code",...w,children:E}):h.jsx("code",{className:S,...w,children:E})},children:N.content}):N.content}),h.jsxs("div",{className:"message-footer",children:[h.jsxs("span",{className:"message-seq",children:["#",N.message_seq]}),N.delivery_state!=="acked"&&h.jsx("span",{className:`delivery-status ${N.delivery_state}`,children:N.delivery_state==="pending"?"sending...":"delivered"})]})]})]},N.id)}),h.jsx("div",{ref:r})]}),h.jsxs("div",{className:"input-area",children:[h.jsxs("div",{className:"workspace-row",children:[h.jsxs("button",{onClick:()=>d(!c),className:`workspace-toggle ${s?"has-workspace":""}`,title:s||"Set working directory",children:[h.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:h.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),h.jsx("span",{children:s?"Workspace set":"Set workspace"})]}),s&&h.jsx("span",{className:"workspace-path",title:s,children:s.length>40?`...${s.slice(-37)}`:s})]}),c&&h.jsxs("div",{className:"workspace-input-row",children:[h.jsx("input",{type:"text",value:s,onChange:N=>u(N.target.value),placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),s&&h.jsx("button",{onClick:()=>{u(""),d(!1)},className:"workspace-clear",children:"Clear"})]}),h.jsxs("div",{className:"input-wrapper",children:[h.jsxs("select",{value:o,onChange:N=>a(N.target.value),className:"kind-selector",children:[h.jsx("option",{value:"directive",children:"Directive"}),h.jsx("option",{value:"question",children:"Question"})]}),h.jsx("textarea",{value:i,onChange:N=>l(N.target.value),onKeyPress:f,placeholder:"Type a message...",rows:1}),h.jsx("button",{onClick:p,className:"send-btn",disabled:!i.trim(),children:_t.send})]}),h.jsxs("div",{className:"input-hint",children:["Press ",h.jsx("kbd",{children:"Enter"})," to send, ",h.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),h.jsx("style",{children:`
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
      `})]})},r0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,reconnectInterval:l=5e3})=>{const o=H.useRef(null),[a,s]=H.useState(!1),[u,c]=H.useState(null),d=H.useRef(null),p=H.useRef(new Map),f=H.useCallback(()=>{try{const S=`${e}?instance_id=${t}`;o.current=new WebSocket(S),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),c(null),p.current.forEach((E,w)=>{N(w,E)})},o.current.onmessage=E=>{try{const w=JSON.parse(E.data);k(w)}catch(w){console.error("Failed to parse WebSocket message:",w)}},o.current.onerror=E=>{console.error("WebSocket error:",E),c("Connection error")},o.current.onclose=()=>{console.log("WebSocket disconnected"),s(!1),d.current=setTimeout(()=>{console.log("Attempting to reconnect..."),f()},l)}}catch(S){console.error("Failed to connect to WebSocket:",S),c("Failed to connect")}},[e,t,l]),k=H.useCallback(S=>{switch(S.type){case"message":n&&S.data&&n(S.data);break;case"batch":if(r&&S.data){const E=S.data;r(E),n&&E.messages.forEach(w=>n(w))}break;case"error":i&&S.data&&i(S.data),console.error("WebSocket error event:",S.data);break;case"pong":break;default:console.log("Unknown event type:",S.type)}},[n,r,i]),C=H.useCallback(S=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(S)):console.warn("WebSocket not connected, cannot send event")},[]),N=H.useCallback((S,E=0)=>{p.current.set(S,E);const w={type:"subscribe",timestamp:Date.now(),data:{thread_id:S,from_seq:E}};C(w)},[C]),m=H.useCallback((S,E)=>{const w=p.current.get(S)||0;E>w&&p.current.set(S,E);const _={type:"ack",timestamp:Date.now(),data:{thread_id:S,ack_seq:E}};C(_)},[C]),y=H.useCallback(()=>{const S={type:"ping",timestamp:Date.now()};C(S)},[C]),g=H.useCallback(S=>{p.current.delete(S)},[]);return H.useEffect(()=>(f(),()=>{d.current&&clearTimeout(d.current),o.current&&o.current.close()}),[f]),H.useEffect(()=>{if(!a)return;const S=setInterval(()=>{y()},3e4);return()=>clearInterval(S)},[a,y]),{isConnected:a,connectionError:u,subscribe:N,unsubscribe:g,acknowledge:m,ping:y}},i0=({connected:e})=>h.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?h.jsxs(h.Fragment,{children:[h.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),h.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):h.jsxs(h.Fragment,{children:[h.jsx("circle",{cx:"12",cy:"12",r:"10"}),h.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),h.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),l0=({websocketUrl:e,instanceId:t})=>{const[n,r]=H.useState([]),[i,l]=H.useState(null),[o,a]=H.useState(new Map),[s,u]=H.useState(new Map),{isConnected:c,subscribe:d,acknowledge:p}=r0({url:e,instanceId:t,onMessage:f,onBatch:k});function f(g){const S={id:g.id,thread_id:g.thread_id,message_seq:g.message_seq,created_at:g.created_at,from_type:g.from_type,from_id:g.from_id,to_type:g.to_type,to_id:g.to_id,kind:g.kind,subject:g.subject,content:g.content,metadata_json:g.metadata_json,delivery_state:"visible",business_state:"open"};a(E=>{const w=E.get(S.thread_id)||[];return w.find(_=>_.id===S.id)?E:new Map(E).set(S.thread_id,[...w,S].sort((_,P)=>_.message_seq-P.message_seq))}),S.thread_id!==i&&u(E=>{const w=E.get(S.thread_id)||0;return new Map(E).set(S.thread_id,w+1)}),p(S.thread_id,S.message_seq)}function k(g){g.messages.forEach(S=>{f(S)})}const C=H.useCallback(g=>{if(l(g),u(S=>{const E=new Map(S);return E.delete(g),E}),c){const S=o.get(g)||[],E=S.length>0?Math.max(...S.map(w=>w.message_seq)):0;d(g,E)}},[c,d,o]),N=H.useCallback(async(g,S,E)=>{if(!i)return;const w=E?JSON.stringify({workspace:E}):void 0;try{const _=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:i,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:S,content:g,metadata_json:w})});if(!_.ok){console.error("Failed to send message:",await _.text());return}const P=await _.json();a(O=>{const M=O.get(i)||[];return M.find(A=>A.id===P.id)?O:new Map(O).set(i,[...M,P])})}catch(_){console.error("Error sending message:",_)}},[i,t]);H.useEffect(()=>{(async()=>{try{const S=await fetch("/api/threads");if(!S.ok){console.error("Failed to fetch threads:",await S.text());return}const E=await S.json();r(E)}catch(S){console.error("Error fetching threads:",S)}})()},[]);const m=H.useCallback(async g=>{try{const S=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:g,created_by_type:"human",created_by_id:"user"})});if(!S.ok){console.error("Failed to create thread:",await S.text());return}const E=await S.json();r(w=>[E,...w]),l(E.id)}catch(S){console.error("Error creating thread:",S)}},[]),y=i?o.get(i)||[]:[];return h.jsxs("div",{className:"message-center",children:[h.jsxs("div",{className:"status-bar",children:[h.jsxs("div",{className:`status-indicator ${c?"connected":"disconnected"}`,children:[h.jsx(i0,{connected:c}),h.jsx("span",{children:c?"Connected":"Disconnected"})]}),h.jsx("div",{className:"status-meta",children:h.jsxs("span",{className:"thread-count",children:[n.length," threads"]})})]}),h.jsxs("div",{className:"center-layout",children:[h.jsx("aside",{className:"threads-panel",children:h.jsx(pg,{threads:n,selectedThreadId:i,onSelectThread:C,onCreateThread:m,unreadCounts:s})}),h.jsx("main",{className:"conversation-panel",children:i?h.jsx(n0,{threadId:i,messages:y,onSendMessage:N}):h.jsxs("div",{className:"empty-state",children:[h.jsx("div",{className:"empty-icon",children:h.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:h.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),h.jsx("h3",{children:"Select a conversation"}),h.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),h.jsx("style",{children:`
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

        .thread-count {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
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
      `})]})},Ct={check:h.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:h.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:h.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),h.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:h.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:h.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:h.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:h.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),h.jsx("circle",{cx:"12",cy:"5",r:"2"}),h.jsx("path",{d:"M12 7v4"})]}),dollar:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),h.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:h.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:h.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:h.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("circle",{cx:"12",cy:"12",r:"10"}),h.jsx("polyline",{points:"12 6 12 12 16 14"})]}),sparkles:h.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),h.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),h.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},o0=({approvals:e,onApprove:t,onReject:n})=>{const[r,i]=H.useState(null),[l,o]=H.useState(new Map),a=f=>{try{return JSON.parse(f)}catch{return null}},s=f=>new Date(f).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),u=f=>{const k=l.get(f)||"";t(f,k),o(new Map(l.set(f,"")))},c=f=>{const k=l.get(f)||"";if(!k.trim()){alert("Please provide a reason for rejection");return}n(f,k),o(new Map(l.set(f,"")))},d=(f,k)=>{o(new Map(l.set(f,k)))},p=e.filter(f=>f.status==="pending");return h.jsxs("div",{className:"approval-queue",children:[h.jsx("div",{className:"queue-header",children:h.jsxs("div",{className:"header-title",children:[h.jsx("h2",{children:"Approval Queue"}),h.jsxs("span",{className:"pending-count",children:[p.length," pending"]})]})}),h.jsx("div",{className:"approvals-container",children:p.length===0?h.jsxs("div",{className:"empty-state",children:[h.jsx("div",{className:"empty-icon",children:Ct.sparkles}),h.jsx("h3",{children:"All caught up!"}),h.jsx("p",{children:"No pending approvals to review"})]}):h.jsx("div",{className:"approvals-list",children:p.map(f=>{const k=a(f.effect_delta_json),C=r===f.id;return h.jsxs("div",{className:`approval-card impact-${f.impact}`,children:[h.jsxs("div",{className:"card-header",onClick:()=>i(C?null:f.id),children:[h.jsxs("div",{className:"header-left",children:[h.jsx("div",{className:`impact-indicator ${f.impact}`}),h.jsxs("div",{className:"proposal-info",children:[h.jsx("span",{className:"proposal-text",children:f.proposal}),h.jsxs("div",{className:"proposal-meta",children:[h.jsxs("span",{className:"meta-item",children:[Ct.bot,f.instance_id]}),h.jsxs("span",{className:"meta-item",children:[Ct.clock,s(f.created_at)]})]})]})]}),h.jsxs("div",{className:"header-right",children:[h.jsxs("span",{className:"cost-badge",children:[Ct.dollar,"$",f.estimated_cost.toFixed(2)]}),h.jsx("span",{className:`impact-badge ${f.impact}`,children:f.impact}),h.jsx("button",{className:"expand-btn",children:C?Ct.chevronUp:Ct.chevronDown})]})]}),C&&h.jsxs("div",{className:"card-details",children:[k&&h.jsxs("div",{className:"detail-section",children:[h.jsx("h4",{children:"Effect Details"}),h.jsxs("div",{className:"detail-grid",children:[h.jsxs("div",{className:"detail-item",children:[h.jsx("span",{className:"detail-label",children:"Capability"}),h.jsx("span",{className:"detail-value code",children:k.cap_type})]}),h.jsxs("div",{className:"detail-item",children:[h.jsx("span",{className:"detail-label",children:"Budget Delta"}),h.jsxs("span",{className:"detail-value",children:["$",k.budget_delta.toFixed(2)]})]}),k.paths.length>0&&h.jsxs("div",{className:"detail-item full-width",children:[h.jsx("span",{className:"detail-label",children:"Paths"}),h.jsx("div",{className:"paths-list",children:k.paths.map((N,m)=>h.jsxs("span",{className:"path-tag",children:[Ct.folder,N]},m))})]})]})]}),h.jsxs("div",{className:"detail-section",children:[h.jsx("h4",{children:"Request Info"}),h.jsxs("div",{className:"detail-grid",children:[h.jsxs("div",{className:"detail-item",children:[h.jsx("span",{className:"detail-label",children:"Thread"}),h.jsx("span",{className:"detail-value code",children:f.thread_id})]}),h.jsxs("div",{className:"detail-item",children:[h.jsx("span",{className:"detail-label",children:"Impact Level"}),h.jsx("span",{className:`detail-value impact-text ${f.impact}`,children:f.impact.toUpperCase()})]})]})]}),h.jsxs("div",{className:"review-section",children:[h.jsx("h4",{children:"Review Notes"}),h.jsx("textarea",{value:l.get(f.id)||"",onChange:N=>d(f.id,N.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),h.jsxs("div",{className:"action-buttons",children:[h.jsxs("button",{className:"reject-btn",onClick:()=>c(f.id),children:[Ct.x,"Reject"]}),h.jsxs("button",{className:"approve-btn",onClick:()=>u(f.id),children:[Ct.check,"Approve"]})]})]})]})]},f.id)})})}),h.jsx("style",{children:`
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
      `})]})},vo={messages:h.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:h.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),shield:h.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:h.jsx("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"})}),logo:h.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[h.jsx("circle",{cx:"12",cy:"12",r:"10"}),h.jsx("path",{d:"M12 6v12M6 12h12"}),h.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]})},a0=()=>{const[e,t]=H.useState("messages"),[n,r]=H.useState([]),[i,l]=H.useState("my-agent"),[o,a]=H.useState([]),[s,u]=H.useState(""),[c,d]=H.useState(!1),f=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;un.useEffect(()=>{const E=async()=>{try{const _=await fetch("/api/agents");if(_.ok){const P=await _.json();a(P),P.length>0&&!i&&l(P[0].id)}}catch(_){console.error("Error fetching agents:",_)}};E();const w=setInterval(E,1e4);return()=>clearInterval(w)},[]);const k=E=>{const w=E.target.value;w==="__custom__"?d(!0):(l(w),d(!1))},C=()=>{s.trim()&&(l(s.trim()),d(!1),u(""))},N=E=>E.last_active?Date.now()-E.last_active<3e4:!1,m=E=>N(E)?"●":"○",y=async(E,w)=>{try{const _=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:w})});if(!_.ok){console.error("Failed to approve:",await _.text());return}r(P=>P.map(O=>O.id===E?{...O,status:"approved",reviewed_by:"user",review_notes:w}:O))}catch(_){console.error("Error approving:",_)}},g=async(E,w)=>{try{const _=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:w})});if(!_.ok){console.error("Failed to reject:",await _.text());return}r(P=>P.map(O=>O.id===E?{...O,status:"rejected",reviewed_by:"user",review_notes:w}:O))}catch(_){console.error("Error rejecting:",_)}};un.useEffect(()=>{const E=async()=>{try{const _=await fetch("/api/approvals?status=pending");if(!_.ok){console.error("Failed to fetch approvals:",await _.text());return}const P=await _.json();r(P)}catch(_){console.error("Error fetching approvals:",_)}};E();const w=setInterval(E,5e3);return()=>clearInterval(w)},[]);const S=(n==null?void 0:n.filter(E=>E.status==="pending").length)||0;return h.jsxs("div",{className:"app",children:[h.jsxs("header",{className:"app-header",children:[h.jsxs("div",{className:"header-brand",children:[h.jsx("div",{className:"brand-logo",children:vo.logo}),h.jsxs("div",{className:"brand-text",children:[h.jsx("h1",{children:"AILANG"}),h.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),h.jsxs("nav",{className:"header-nav",children:[h.jsxs("button",{className:`nav-tab ${e==="messages"?"active":""}`,onClick:()=>t("messages"),children:[h.jsx("span",{className:"nav-icon",children:vo.messages}),h.jsx("span",{className:"nav-label",children:"Messages"})]}),h.jsxs("button",{className:`nav-tab ${e==="approvals"?"active":""}`,onClick:()=>t("approvals"),children:[h.jsx("span",{className:"nav-icon",children:vo.shield}),h.jsx("span",{className:"nav-label",children:"Approvals"}),S>0&&h.jsx("span",{className:"nav-badge",children:S})]})]}),h.jsxs("div",{className:"header-meta",children:[h.jsxs("div",{className:"agent-selector",children:[h.jsx("label",{className:"agent-label",children:"Target:"}),c?h.jsxs("div",{className:"custom-agent-input",children:[h.jsx("input",{type:"text",value:s,onChange:E=>u(E.target.value),onKeyDown:E=>E.key==="Enter"&&C(),className:"agent-input",placeholder:"agent-id",autoFocus:!0}),h.jsx("button",{onClick:C,className:"agent-apply",children:"Add"}),h.jsx("button",{onClick:()=>d(!1),className:"agent-cancel",children:"Cancel"})]}):h.jsxs(h.Fragment,{children:[h.jsxs("select",{value:i,onChange:k,className:"agent-select",children:[o.map(E=>h.jsxs("option",{value:E.id,children:[m(E)," ",E.id]},E.id)),!o.find(E=>E.id===i)&&i&&h.jsxs("option",{value:i,children:["○ ",i]}),h.jsx("option",{value:"__custom__",children:"+ Add custom..."})]}),o.find(E=>E.id===i)&&h.jsx("span",{className:`agent-status ${N(o.find(E=>E.id===i))?"active":"inactive"}`,children:N(o.find(E=>E.id===i))?"Online":"Offline"})]})]}),h.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),h.jsx("main",{className:"app-content",children:e==="messages"?h.jsx(l0,{websocketUrl:f,instanceId:i}):h.jsx(o0,{approvals:n,onApprove:y,onReject:g})}),h.jsx("style",{children:`
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
      `})]})};xo.createRoot(document.getElementById("root")).render(h.jsx(un.StrictMode,{children:h.jsx(a0,{})}));
