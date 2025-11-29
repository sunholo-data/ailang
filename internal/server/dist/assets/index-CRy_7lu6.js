(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Hi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Ca(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Bc={exports:{}},gl={},Uc={exports:{}},G={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ei=Symbol.for("react.element"),Vf=Symbol.for("react.portal"),Wf=Symbol.for("react.fragment"),Qf=Symbol.for("react.strict_mode"),Kf=Symbol.for("react.profiler"),qf=Symbol.for("react.provider"),Yf=Symbol.for("react.context"),Xf=Symbol.for("react.forward_ref"),Gf=Symbol.for("react.suspense"),Jf=Symbol.for("react.memo"),Zf=Symbol.for("react.lazy"),Bs=Symbol.iterator;function eh(e){return e===null||typeof e!="object"?null:(e=Bs&&e[Bs]||e["@@iterator"],typeof e=="function"?e:null)}var $c={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Hc=Object.assign,Vc={};function rr(e,t,n){this.props=e,this.context=t,this.refs=Vc,this.updater=n||$c}rr.prototype.isReactComponent={};rr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};rr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function Wc(){}Wc.prototype=rr.prototype;function Ea(e,t,n){this.props=e,this.context=t,this.refs=Vc,this.updater=n||$c}var Na=Ea.prototype=new Wc;Na.constructor=Ea;Hc(Na,rr.prototype);Na.isPureReactComponent=!0;var Us=Array.isArray,Qc=Object.prototype.hasOwnProperty,_a={current:null},Kc={key:!0,ref:!0,__self:!0,__source:!0};function qc(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)Qc.call(t,r)&&!Kc.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),c=0;c<a;c++)s[c]=arguments[c+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:ei,type:e,key:l,ref:o,props:i,_owner:_a.current}}function th(e,t){return{$$typeof:ei,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function za(e){return typeof e=="object"&&e!==null&&e.$$typeof===ei}function nh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var $s=/\/+/g;function Dl(e,t){return typeof e=="object"&&e!==null&&e.key!=null?nh(""+e.key):t.toString(36)}function zi(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case ei:case Vf:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Dl(o,0):r,Us(i)?(n="",e!=null&&(n=e.replace($s,"$&/")+"/"),zi(i,t,n,"",function(c){return c})):i!=null&&(za(i)&&(i=th(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace($s,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Us(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Dl(l,a);o+=zi(l,t,n,s,i)}else if(s=eh(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Dl(l,a++),o+=zi(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function si(e,t,n){if(e==null)return e;var r=[],i=0;return zi(e,r,"","",function(l){return t.call(n,l,i++)}),r}function rh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var De={current:null},Ti={transition:null},ih={ReactCurrentDispatcher:De,ReactCurrentBatchConfig:Ti,ReactCurrentOwner:_a};function Yc(){throw Error("act(...) is not supported in production builds of React.")}G.Children={map:si,forEach:function(e,t,n){si(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return si(e,function(){t++}),t},toArray:function(e){return si(e,function(t){return t})||[]},only:function(e){if(!za(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};G.Component=rr;G.Fragment=Wf;G.Profiler=Kf;G.PureComponent=Ea;G.StrictMode=Qf;G.Suspense=Gf;G.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=ih;G.act=Yc;G.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Hc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=_a.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)Qc.call(t,s)&&!Kc.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var c=0;c<s;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:ei,type:e.type,key:i,ref:l,props:r,_owner:o}};G.createContext=function(e){return e={$$typeof:Yf,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:qf,_context:e},e.Consumer=e};G.createElement=qc;G.createFactory=function(e){var t=qc.bind(null,e);return t.type=e,t};G.createRef=function(){return{current:null}};G.forwardRef=function(e){return{$$typeof:Xf,render:e}};G.isValidElement=za;G.lazy=function(e){return{$$typeof:Zf,_payload:{_status:-1,_result:e},_init:rh}};G.memo=function(e,t){return{$$typeof:Jf,type:e,compare:t===void 0?null:t}};G.startTransition=function(e){var t=Ti.transition;Ti.transition={};try{e()}finally{Ti.transition=t}};G.unstable_act=Yc;G.useCallback=function(e,t){return De.current.useCallback(e,t)};G.useContext=function(e){return De.current.useContext(e)};G.useDebugValue=function(){};G.useDeferredValue=function(e){return De.current.useDeferredValue(e)};G.useEffect=function(e,t){return De.current.useEffect(e,t)};G.useId=function(){return De.current.useId()};G.useImperativeHandle=function(e,t,n){return De.current.useImperativeHandle(e,t,n)};G.useInsertionEffect=function(e,t){return De.current.useInsertionEffect(e,t)};G.useLayoutEffect=function(e,t){return De.current.useLayoutEffect(e,t)};G.useMemo=function(e,t){return De.current.useMemo(e,t)};G.useReducer=function(e,t,n){return De.current.useReducer(e,t,n)};G.useRef=function(e){return De.current.useRef(e)};G.useState=function(e){return De.current.useState(e)};G.useSyncExternalStore=function(e,t,n){return De.current.useSyncExternalStore(e,t,n)};G.useTransition=function(){return De.current.useTransition()};G.version="18.3.1";Uc.exports=G;var B=Uc.exports;const zt=Ca(B);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var lh=B,oh=Symbol.for("react.element"),ah=Symbol.for("react.fragment"),sh=Object.prototype.hasOwnProperty,uh=lh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,ch={key:!0,ref:!0,__self:!0,__source:!0};function Xc(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)sh.call(t,r)&&!ch.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:oh,type:e,key:l,ref:o,props:i,_owner:uh.current}}gl.Fragment=ah;gl.jsx=Xc;gl.jsxs=Xc;Bc.exports=gl;var u=Bc.exports,wo={},Gc={exports:{}},et={},Jc={exports:{}},Zc={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(P,E){var g=P.length;P.push(E);e:for(;0<g;){var A=g-1>>>1,V=P[A];if(0<i(V,E))P[A]=E,P[g]=V,g=A;else break e}}function n(P){return P.length===0?null:P[0]}function r(P){if(P.length===0)return null;var E=P[0],g=P.pop();if(g!==E){P[0]=g;e:for(var A=0,V=P.length,x=V>>>1;A<x;){var te=2*(A+1)-1,we=P[te],q=te+1,ye=P[q];if(0>i(we,g))q<V&&0>i(ye,we)?(P[A]=ye,P[q]=g,A=q):(P[A]=we,P[te]=g,A=te);else if(q<V&&0>i(ye,g))P[A]=ye,P[q]=g,A=q;else break e}}return E}function i(P,E){var g=P.sortIndex-E.sortIndex;return g!==0?g:P.id-E.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],c=[],d=1,p=null,m=3,f=!1,k=!1,w=!1,M=typeof setTimeout=="function"?setTimeout:null,h=typeof clearTimeout=="function"?clearTimeout:null,v=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(P){for(var E=n(c);E!==null;){if(E.callback===null)r(c);else if(E.startTime<=P)r(c),E.sortIndex=E.expirationTime,t(s,E);else break;E=n(c)}}function b(P){if(w=!1,y(P),!k)if(n(s)!==null)k=!0,N(_);else{var E=n(c);E!==null&&W(b,E.startTime-P)}}function _(P,E){k=!1,w&&(w=!1,h(C),C=-1),f=!0;var g=m;try{for(y(E),p=n(s);p!==null&&(!(p.expirationTime>E)||P&&!j());){var A=p.callback;if(typeof A=="function"){p.callback=null,m=p.priorityLevel;var V=A(p.expirationTime<=E);E=e.unstable_now(),typeof V=="function"?p.callback=V:p===n(s)&&r(s),y(E)}else r(s);p=n(s)}if(p!==null)var x=!0;else{var te=n(c);te!==null&&W(b,te.startTime-E),x=!1}return x}finally{p=null,m=g,f=!1}}var S=!1,L=null,C=-1,T=5,O=-1;function j(){return!(e.unstable_now()-O<T)}function I(){if(L!==null){var P=e.unstable_now();O=P;var E=!0;try{E=L(!0,P)}finally{E?H():(S=!1,L=null)}}else S=!1}var H;if(typeof v=="function")H=function(){v(I)};else if(typeof MessageChannel<"u"){var Q=new MessageChannel,$=Q.port2;Q.port1.onmessage=I,H=function(){$.postMessage(null)}}else H=function(){M(I,0)};function N(P){L=P,S||(S=!0,H())}function W(P,E){C=M(function(){P(e.unstable_now())},E)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(P){P.callback=null},e.unstable_continueExecution=function(){k||f||(k=!0,N(_))},e.unstable_forceFrameRate=function(P){0>P||125<P?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):T=0<P?Math.floor(1e3/P):5},e.unstable_getCurrentPriorityLevel=function(){return m},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(P){switch(m){case 1:case 2:case 3:var E=3;break;default:E=m}var g=m;m=E;try{return P()}finally{m=g}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(P,E){switch(P){case 1:case 2:case 3:case 4:case 5:break;default:P=3}var g=m;m=P;try{return E()}finally{m=g}},e.unstable_scheduleCallback=function(P,E,g){var A=e.unstable_now();switch(typeof g=="object"&&g!==null?(g=g.delay,g=typeof g=="number"&&0<g?A+g:A):g=A,P){case 1:var V=-1;break;case 2:V=250;break;case 5:V=1073741823;break;case 4:V=1e4;break;default:V=5e3}return V=g+V,P={id:d++,callback:E,priorityLevel:P,startTime:g,expirationTime:V,sortIndex:-1},g>A?(P.sortIndex=g,t(c,P),n(s)===null&&P===n(c)&&(w?(h(C),C=-1):w=!0,W(b,g-A))):(P.sortIndex=V,t(s,P),k||f||(k=!0,N(_))),P},e.unstable_shouldYield=j,e.unstable_wrapCallback=function(P){var E=m;return function(){var g=m;m=E;try{return P.apply(this,arguments)}finally{m=g}}}})(Zc);Jc.exports=Zc;var dh=Jc.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ph=B,Ze=dh;function D(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var ed=new Set,Dr={};function bn(e,t){Xn(e,t),Xn(e+"Capture",t)}function Xn(e,t){for(Dr[e]=t,e=0;e<t.length;e++)ed.add(t[e])}var Mt=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),So=Object.prototype.hasOwnProperty,fh=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,Hs={},Vs={};function hh(e){return So.call(Vs,e)?!0:So.call(Hs,e)?!1:fh.test(e)?Vs[e]=!0:(Hs[e]=!0,!1)}function mh(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function gh(e,t,n,r){if(t===null||typeof t>"u"||mh(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Re(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ee={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ee[e]=new Re(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ee[t]=new Re(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ee[e]=new Re(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ee[e]=new Re(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ee[e]=new Re(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ee[e]=new Re(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ee[e]=new Re(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ee[e]=new Re(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ee[e]=new Re(e,5,!1,e.toLowerCase(),null,!1,!1)});var Ta=/[\-:]([a-z])/g;function La(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(Ta,La);Ee[t]=new Re(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(Ta,La);Ee[t]=new Re(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(Ta,La);Ee[t]=new Re(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ee[e]=new Re(e,1,!1,e.toLowerCase(),null,!1,!1)});Ee.xlinkHref=new Re("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ee[e]=new Re(e,1,!1,e.toLowerCase(),null,!0,!0)});function Pa(e,t,n,r){var i=Ee.hasOwnProperty(t)?Ee[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(gh(t,n,i,r)&&(n=null),r||i===null?hh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Ot=ph.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,ui=Symbol.for("react.element"),Tn=Symbol.for("react.portal"),Ln=Symbol.for("react.fragment"),Ia=Symbol.for("react.strict_mode"),bo=Symbol.for("react.profiler"),td=Symbol.for("react.provider"),nd=Symbol.for("react.context"),Ma=Symbol.for("react.forward_ref"),jo=Symbol.for("react.suspense"),Co=Symbol.for("react.suspense_list"),Aa=Symbol.for("react.memo"),Ut=Symbol.for("react.lazy"),rd=Symbol.for("react.offscreen"),Ws=Symbol.iterator;function cr(e){return e===null||typeof e!="object"?null:(e=Ws&&e[Ws]||e["@@iterator"],typeof e=="function"?e:null)}var pe=Object.assign,Rl;function kr(e){if(Rl===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Rl=t&&t[1]||""}return`
`+Rl+e}var Ol=!1;function Fl(e,t){if(!e||Ol)return"";Ol=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Ol=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?kr(e):""}function vh(e){switch(e.tag){case 5:return kr(e.type);case 16:return kr("Lazy");case 13:return kr("Suspense");case 19:return kr("SuspenseList");case 0:case 2:case 15:return e=Fl(e.type,!1),e;case 11:return e=Fl(e.type.render,!1),e;case 1:return e=Fl(e.type,!0),e;default:return""}}function Eo(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Ln:return"Fragment";case Tn:return"Portal";case bo:return"Profiler";case Ia:return"StrictMode";case jo:return"Suspense";case Co:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case nd:return(e.displayName||"Context")+".Consumer";case td:return(e._context.displayName||"Context")+".Provider";case Ma:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Aa:return t=e.displayName||null,t!==null?t:Eo(e.type)||"Memo";case Ut:t=e._payload,e=e._init;try{return Eo(e(t))}catch{}}return null}function yh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Eo(t);case 8:return t===Ia?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function tn(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function id(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function xh(e){var t=id(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function ci(e){e._valueTracker||(e._valueTracker=xh(e))}function ld(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=id(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Vi(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function No(e,t){var n=t.checked;return pe({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function Qs(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=tn(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function od(e,t){t=t.checked,t!=null&&Pa(e,"checked",t,!1)}function _o(e,t){od(e,t);var n=tn(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?zo(e,t.type,n):t.hasOwnProperty("defaultValue")&&zo(e,t.type,tn(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function Ks(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function zo(e,t,n){(t!=="number"||Vi(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var wr=Array.isArray;function $n(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+tn(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function To(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(D(91));return pe({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function qs(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(D(92));if(wr(n)){if(1<n.length)throw Error(D(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:tn(n)}}function ad(e,t){var n=tn(t.value),r=tn(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function Ys(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function sd(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Lo(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?sd(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var di,ud=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(di=di||document.createElement("div"),di.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=di.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Rr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var jr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},kh=["Webkit","ms","Moz","O"];Object.keys(jr).forEach(function(e){kh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),jr[t]=jr[e]})});function cd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||jr.hasOwnProperty(e)&&jr[e]?(""+t).trim():t+"px"}function dd(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=cd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var wh=pe({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Po(e,t){if(t){if(wh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(D(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(D(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(D(61))}if(t.style!=null&&typeof t.style!="object")throw Error(D(62))}}function Io(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Mo=null;function Da(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Ao=null,Hn=null,Vn=null;function Xs(e){if(e=ri(e)){if(typeof Ao!="function")throw Error(D(280));var t=e.stateNode;t&&(t=wl(t),Ao(e.stateNode,e.type,t))}}function pd(e){Hn?Vn?Vn.push(e):Vn=[e]:Hn=e}function fd(){if(Hn){var e=Hn,t=Vn;if(Vn=Hn=null,Xs(e),t)for(e=0;e<t.length;e++)Xs(t[e])}}function hd(e,t){return e(t)}function md(){}var Bl=!1;function gd(e,t,n){if(Bl)return e(t,n);Bl=!0;try{return hd(e,t,n)}finally{Bl=!1,(Hn!==null||Vn!==null)&&(md(),fd())}}function Or(e,t){var n=e.stateNode;if(n===null)return null;var r=wl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(D(231,t,typeof n));return n}var Do=!1;if(Mt)try{var dr={};Object.defineProperty(dr,"passive",{get:function(){Do=!0}}),window.addEventListener("test",dr,dr),window.removeEventListener("test",dr,dr)}catch{Do=!1}function Sh(e,t,n,r,i,l,o,a,s){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Cr=!1,Wi=null,Qi=!1,Ro=null,bh={onError:function(e){Cr=!0,Wi=e}};function jh(e,t,n,r,i,l,o,a,s){Cr=!1,Wi=null,Sh.apply(bh,arguments)}function Ch(e,t,n,r,i,l,o,a,s){if(jh.apply(this,arguments),Cr){if(Cr){var c=Wi;Cr=!1,Wi=null}else throw Error(D(198));Qi||(Qi=!0,Ro=c)}}function jn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function vd(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function Gs(e){if(jn(e)!==e)throw Error(D(188))}function Eh(e){var t=e.alternate;if(!t){if(t=jn(e),t===null)throw Error(D(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return Gs(i),e;if(l===r)return Gs(i),t;l=l.sibling}throw Error(D(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(D(189))}}if(n.alternate!==r)throw Error(D(190))}if(n.tag!==3)throw Error(D(188));return n.stateNode.current===n?e:t}function yd(e){return e=Eh(e),e!==null?xd(e):null}function xd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=xd(e);if(t!==null)return t;e=e.sibling}return null}var kd=Ze.unstable_scheduleCallback,Js=Ze.unstable_cancelCallback,Nh=Ze.unstable_shouldYield,_h=Ze.unstable_requestPaint,he=Ze.unstable_now,zh=Ze.unstable_getCurrentPriorityLevel,Ra=Ze.unstable_ImmediatePriority,wd=Ze.unstable_UserBlockingPriority,Ki=Ze.unstable_NormalPriority,Th=Ze.unstable_LowPriority,Sd=Ze.unstable_IdlePriority,vl=null,St=null;function Lh(e){if(St&&typeof St.onCommitFiberRoot=="function")try{St.onCommitFiberRoot(vl,e,void 0,(e.current.flags&128)===128)}catch{}}var ht=Math.clz32?Math.clz32:Mh,Ph=Math.log,Ih=Math.LN2;function Mh(e){return e>>>=0,e===0?32:31-(Ph(e)/Ih|0)|0}var pi=64,fi=4194304;function Sr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function qi(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=Sr(a):(l&=o,l!==0&&(r=Sr(l)))}else o=n&~i,o!==0?r=Sr(o):l!==0&&(r=Sr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-ht(t),i=1<<n,r|=e[n],t&=~i;return r}function Ah(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Dh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-ht(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Ah(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Oo(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function bd(){var e=pi;return pi<<=1,!(pi&4194240)&&(pi=64),e}function Ul(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function ti(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-ht(t),e[t]=n}function Rh(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-ht(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Oa(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-ht(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var ne=0;function jd(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Cd,Fa,Ed,Nd,_d,Fo=!1,hi=[],Kt=null,qt=null,Yt=null,Fr=new Map,Br=new Map,Ht=[],Oh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function Zs(e,t){switch(e){case"focusin":case"focusout":Kt=null;break;case"dragenter":case"dragleave":qt=null;break;case"mouseover":case"mouseout":Yt=null;break;case"pointerover":case"pointerout":Fr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Br.delete(t.pointerId)}}function pr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=ri(t),t!==null&&Fa(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Fh(e,t,n,r,i){switch(t){case"focusin":return Kt=pr(Kt,e,t,n,r,i),!0;case"dragenter":return qt=pr(qt,e,t,n,r,i),!0;case"mouseover":return Yt=pr(Yt,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Fr.set(l,pr(Fr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Br.set(l,pr(Br.get(l)||null,e,t,n,r,i)),!0}return!1}function zd(e){var t=fn(e.target);if(t!==null){var n=jn(t);if(n!==null){if(t=n.tag,t===13){if(t=vd(n),t!==null){e.blockedOn=t,_d(e.priority,function(){Ed(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Li(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Bo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Mo=r,n.target.dispatchEvent(r),Mo=null}else return t=ri(n),t!==null&&Fa(t),e.blockedOn=n,!1;t.shift()}return!0}function eu(e,t,n){Li(e)&&n.delete(t)}function Bh(){Fo=!1,Kt!==null&&Li(Kt)&&(Kt=null),qt!==null&&Li(qt)&&(qt=null),Yt!==null&&Li(Yt)&&(Yt=null),Fr.forEach(eu),Br.forEach(eu)}function fr(e,t){e.blockedOn===t&&(e.blockedOn=null,Fo||(Fo=!0,Ze.unstable_scheduleCallback(Ze.unstable_NormalPriority,Bh)))}function Ur(e){function t(i){return fr(i,e)}if(0<hi.length){fr(hi[0],e);for(var n=1;n<hi.length;n++){var r=hi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(Kt!==null&&fr(Kt,e),qt!==null&&fr(qt,e),Yt!==null&&fr(Yt,e),Fr.forEach(t),Br.forEach(t),n=0;n<Ht.length;n++)r=Ht[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Ht.length&&(n=Ht[0],n.blockedOn===null);)zd(n),n.blockedOn===null&&Ht.shift()}var Wn=Ot.ReactCurrentBatchConfig,Yi=!0;function Uh(e,t,n,r){var i=ne,l=Wn.transition;Wn.transition=null;try{ne=1,Ba(e,t,n,r)}finally{ne=i,Wn.transition=l}}function $h(e,t,n,r){var i=ne,l=Wn.transition;Wn.transition=null;try{ne=4,Ba(e,t,n,r)}finally{ne=i,Wn.transition=l}}function Ba(e,t,n,r){if(Yi){var i=Bo(e,t,n,r);if(i===null)Gl(e,t,r,Xi,n),Zs(e,r);else if(Fh(i,e,t,n,r))r.stopPropagation();else if(Zs(e,r),t&4&&-1<Oh.indexOf(e)){for(;i!==null;){var l=ri(i);if(l!==null&&Cd(l),l=Bo(e,t,n,r),l===null&&Gl(e,t,r,Xi,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else Gl(e,t,r,null,n)}}var Xi=null;function Bo(e,t,n,r){if(Xi=null,e=Da(r),e=fn(e),e!==null)if(t=jn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=vd(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return Xi=e,null}function Td(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(zh()){case Ra:return 1;case wd:return 4;case Ki:case Th:return 16;case Sd:return 536870912;default:return 16}default:return 16}}var Wt=null,Ua=null,Pi=null;function Ld(){if(Pi)return Pi;var e,t=Ua,n=t.length,r,i="value"in Wt?Wt.value:Wt.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Pi=i.slice(e,1<r?1-r:void 0)}function Ii(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function mi(){return!0}function tu(){return!1}function tt(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?mi:tu,this.isPropagationStopped=tu,this}return pe(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=mi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=mi)},persist:function(){},isPersistent:mi}),t}var ir={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},$a=tt(ir),ni=pe({},ir,{view:0,detail:0}),Hh=tt(ni),$l,Hl,hr,yl=pe({},ni,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:Ha,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==hr&&(hr&&e.type==="mousemove"?($l=e.screenX-hr.screenX,Hl=e.screenY-hr.screenY):Hl=$l=0,hr=e),$l)},movementY:function(e){return"movementY"in e?e.movementY:Hl}}),nu=tt(yl),Vh=pe({},yl,{dataTransfer:0}),Wh=tt(Vh),Qh=pe({},ni,{relatedTarget:0}),Vl=tt(Qh),Kh=pe({},ir,{animationName:0,elapsedTime:0,pseudoElement:0}),qh=tt(Kh),Yh=pe({},ir,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),Xh=tt(Yh),Gh=pe({},ir,{data:0}),ru=tt(Gh),Jh={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},Zh={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},em={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function tm(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=em[e])?!!t[e]:!1}function Ha(){return tm}var nm=pe({},ni,{key:function(e){if(e.key){var t=Jh[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Ii(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?Zh[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:Ha,charCode:function(e){return e.type==="keypress"?Ii(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Ii(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),rm=tt(nm),im=pe({},yl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),iu=tt(im),lm=pe({},ni,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:Ha}),om=tt(lm),am=pe({},ir,{propertyName:0,elapsedTime:0,pseudoElement:0}),sm=tt(am),um=pe({},yl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),cm=tt(um),dm=[9,13,27,32],Va=Mt&&"CompositionEvent"in window,Er=null;Mt&&"documentMode"in document&&(Er=document.documentMode);var pm=Mt&&"TextEvent"in window&&!Er,Pd=Mt&&(!Va||Er&&8<Er&&11>=Er),lu=" ",ou=!1;function Id(e,t){switch(e){case"keyup":return dm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Md(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Pn=!1;function fm(e,t){switch(e){case"compositionend":return Md(t);case"keypress":return t.which!==32?null:(ou=!0,lu);case"textInput":return e=t.data,e===lu&&ou?null:e;default:return null}}function hm(e,t){if(Pn)return e==="compositionend"||!Va&&Id(e,t)?(e=Ld(),Pi=Ua=Wt=null,Pn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Pd&&t.locale!=="ko"?null:t.data;default:return null}}var mm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function au(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!mm[e.type]:t==="textarea"}function Ad(e,t,n,r){pd(r),t=Gi(t,"onChange"),0<t.length&&(n=new $a("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Nr=null,$r=null;function gm(e){Qd(e,0)}function xl(e){var t=An(e);if(ld(t))return e}function vm(e,t){if(e==="change")return t}var Dd=!1;if(Mt){var Wl;if(Mt){var Ql="oninput"in document;if(!Ql){var su=document.createElement("div");su.setAttribute("oninput","return;"),Ql=typeof su.oninput=="function"}Wl=Ql}else Wl=!1;Dd=Wl&&(!document.documentMode||9<document.documentMode)}function uu(){Nr&&(Nr.detachEvent("onpropertychange",Rd),$r=Nr=null)}function Rd(e){if(e.propertyName==="value"&&xl($r)){var t=[];Ad(t,$r,e,Da(e)),gd(gm,t)}}function ym(e,t,n){e==="focusin"?(uu(),Nr=t,$r=n,Nr.attachEvent("onpropertychange",Rd)):e==="focusout"&&uu()}function xm(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return xl($r)}function km(e,t){if(e==="click")return xl(t)}function wm(e,t){if(e==="input"||e==="change")return xl(t)}function Sm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var gt=typeof Object.is=="function"?Object.is:Sm;function Hr(e,t){if(gt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!So.call(t,i)||!gt(e[i],t[i]))return!1}return!0}function cu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function du(e,t){var n=cu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=cu(n)}}function Od(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Od(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Fd(){for(var e=window,t=Vi();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Vi(e.document)}return t}function Wa(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function bm(e){var t=Fd(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Od(n.ownerDocument.documentElement,n)){if(r!==null&&Wa(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=du(n,l);var o=du(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var jm=Mt&&"documentMode"in document&&11>=document.documentMode,In=null,Uo=null,_r=null,$o=!1;function pu(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;$o||In==null||In!==Vi(r)||(r=In,"selectionStart"in r&&Wa(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),_r&&Hr(_r,r)||(_r=r,r=Gi(Uo,"onSelect"),0<r.length&&(t=new $a("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=In)))}function gi(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Mn={animationend:gi("Animation","AnimationEnd"),animationiteration:gi("Animation","AnimationIteration"),animationstart:gi("Animation","AnimationStart"),transitionend:gi("Transition","TransitionEnd")},Kl={},Bd={};Mt&&(Bd=document.createElement("div").style,"AnimationEvent"in window||(delete Mn.animationend.animation,delete Mn.animationiteration.animation,delete Mn.animationstart.animation),"TransitionEvent"in window||delete Mn.transitionend.transition);function kl(e){if(Kl[e])return Kl[e];if(!Mn[e])return e;var t=Mn[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Bd)return Kl[e]=t[n];return e}var Ud=kl("animationend"),$d=kl("animationiteration"),Hd=kl("animationstart"),Vd=kl("transitionend"),Wd=new Map,fu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function rn(e,t){Wd.set(e,t),bn(t,[e])}for(var ql=0;ql<fu.length;ql++){var Yl=fu[ql],Cm=Yl.toLowerCase(),Em=Yl[0].toUpperCase()+Yl.slice(1);rn(Cm,"on"+Em)}rn(Ud,"onAnimationEnd");rn($d,"onAnimationIteration");rn(Hd,"onAnimationStart");rn("dblclick","onDoubleClick");rn("focusin","onFocus");rn("focusout","onBlur");rn(Vd,"onTransitionEnd");Xn("onMouseEnter",["mouseout","mouseover"]);Xn("onMouseLeave",["mouseout","mouseover"]);Xn("onPointerEnter",["pointerout","pointerover"]);Xn("onPointerLeave",["pointerout","pointerover"]);bn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));bn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));bn("onBeforeInput",["compositionend","keypress","textInput","paste"]);bn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));bn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));bn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var br="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Nm=new Set("cancel close invalid load scroll toggle".split(" ").concat(br));function hu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,Ch(r,t,void 0,e),e.currentTarget=null}function Qd(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,c=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;hu(i,a,c),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,c=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;hu(i,a,c),l=s}}}if(Qi)throw e=Ro,Qi=!1,Ro=null,e}function ae(e,t){var n=t[Ko];n===void 0&&(n=t[Ko]=new Set);var r=e+"__bubble";n.has(r)||(Kd(t,e,2,!1),n.add(r))}function Xl(e,t,n){var r=0;t&&(r|=4),Kd(n,e,r,t)}var vi="_reactListening"+Math.random().toString(36).slice(2);function Vr(e){if(!e[vi]){e[vi]=!0,ed.forEach(function(n){n!=="selectionchange"&&(Nm.has(n)||Xl(n,!1,e),Xl(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[vi]||(t[vi]=!0,Xl("selectionchange",!1,t))}}function Kd(e,t,n,r){switch(Td(t)){case 1:var i=Uh;break;case 4:i=$h;break;default:i=Ba}n=i.bind(null,t,n,e),i=void 0,!Do||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function Gl(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=fn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}gd(function(){var c=l,d=Da(n),p=[];e:{var m=Wd.get(e);if(m!==void 0){var f=$a,k=e;switch(e){case"keypress":if(Ii(n)===0)break e;case"keydown":case"keyup":f=rm;break;case"focusin":k="focus",f=Vl;break;case"focusout":k="blur",f=Vl;break;case"beforeblur":case"afterblur":f=Vl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":f=nu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":f=Wh;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":f=om;break;case Ud:case $d:case Hd:f=qh;break;case Vd:f=sm;break;case"scroll":f=Hh;break;case"wheel":f=cm;break;case"copy":case"cut":case"paste":f=Xh;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":f=iu}var w=(t&4)!==0,M=!w&&e==="scroll",h=w?m!==null?m+"Capture":null:m;w=[];for(var v=c,y;v!==null;){y=v;var b=y.stateNode;if(y.tag===5&&b!==null&&(y=b,h!==null&&(b=Or(v,h),b!=null&&w.push(Wr(v,b,y)))),M)break;v=v.return}0<w.length&&(m=new f(m,k,null,n,d),p.push({event:m,listeners:w}))}}if(!(t&7)){e:{if(m=e==="mouseover"||e==="pointerover",f=e==="mouseout"||e==="pointerout",m&&n!==Mo&&(k=n.relatedTarget||n.fromElement)&&(fn(k)||k[At]))break e;if((f||m)&&(m=d.window===d?d:(m=d.ownerDocument)?m.defaultView||m.parentWindow:window,f?(k=n.relatedTarget||n.toElement,f=c,k=k?fn(k):null,k!==null&&(M=jn(k),k!==M||k.tag!==5&&k.tag!==6)&&(k=null)):(f=null,k=c),f!==k)){if(w=nu,b="onMouseLeave",h="onMouseEnter",v="mouse",(e==="pointerout"||e==="pointerover")&&(w=iu,b="onPointerLeave",h="onPointerEnter",v="pointer"),M=f==null?m:An(f),y=k==null?m:An(k),m=new w(b,v+"leave",f,n,d),m.target=M,m.relatedTarget=y,b=null,fn(d)===c&&(w=new w(h,v+"enter",k,n,d),w.target=y,w.relatedTarget=M,b=w),M=b,f&&k)t:{for(w=f,h=k,v=0,y=w;y;y=_n(y))v++;for(y=0,b=h;b;b=_n(b))y++;for(;0<v-y;)w=_n(w),v--;for(;0<y-v;)h=_n(h),y--;for(;v--;){if(w===h||h!==null&&w===h.alternate)break t;w=_n(w),h=_n(h)}w=null}else w=null;f!==null&&mu(p,m,f,w,!1),k!==null&&M!==null&&mu(p,M,k,w,!0)}}e:{if(m=c?An(c):window,f=m.nodeName&&m.nodeName.toLowerCase(),f==="select"||f==="input"&&m.type==="file")var _=vm;else if(au(m))if(Dd)_=wm;else{_=xm;var S=ym}else(f=m.nodeName)&&f.toLowerCase()==="input"&&(m.type==="checkbox"||m.type==="radio")&&(_=km);if(_&&(_=_(e,c))){Ad(p,_,n,d);break e}S&&S(e,m,c),e==="focusout"&&(S=m._wrapperState)&&S.controlled&&m.type==="number"&&zo(m,"number",m.value)}switch(S=c?An(c):window,e){case"focusin":(au(S)||S.contentEditable==="true")&&(In=S,Uo=c,_r=null);break;case"focusout":_r=Uo=In=null;break;case"mousedown":$o=!0;break;case"contextmenu":case"mouseup":case"dragend":$o=!1,pu(p,n,d);break;case"selectionchange":if(jm)break;case"keydown":case"keyup":pu(p,n,d)}var L;if(Va)e:{switch(e){case"compositionstart":var C="onCompositionStart";break e;case"compositionend":C="onCompositionEnd";break e;case"compositionupdate":C="onCompositionUpdate";break e}C=void 0}else Pn?Id(e,n)&&(C="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(C="onCompositionStart");C&&(Pd&&n.locale!=="ko"&&(Pn||C!=="onCompositionStart"?C==="onCompositionEnd"&&Pn&&(L=Ld()):(Wt=d,Ua="value"in Wt?Wt.value:Wt.textContent,Pn=!0)),S=Gi(c,C),0<S.length&&(C=new ru(C,e,null,n,d),p.push({event:C,listeners:S}),L?C.data=L:(L=Md(n),L!==null&&(C.data=L)))),(L=pm?fm(e,n):hm(e,n))&&(c=Gi(c,"onBeforeInput"),0<c.length&&(d=new ru("onBeforeInput","beforeinput",null,n,d),p.push({event:d,listeners:c}),d.data=L))}Qd(p,t)})}function Wr(e,t,n){return{instance:e,listener:t,currentTarget:n}}function Gi(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Or(e,n),l!=null&&r.unshift(Wr(e,l,i)),l=Or(e,t),l!=null&&r.push(Wr(e,l,i))),e=e.return}return r}function _n(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function mu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,c=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&c!==null&&(a=c,i?(s=Or(n,l),s!=null&&o.unshift(Wr(n,s,a))):i||(s=Or(n,l),s!=null&&o.push(Wr(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var _m=/\r\n?/g,zm=/\u0000|\uFFFD/g;function gu(e){return(typeof e=="string"?e:""+e).replace(_m,`
`).replace(zm,"")}function yi(e,t,n){if(t=gu(t),gu(e)!==t&&n)throw Error(D(425))}function Ji(){}var Ho=null,Vo=null;function Wo(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var Qo=typeof setTimeout=="function"?setTimeout:void 0,Tm=typeof clearTimeout=="function"?clearTimeout:void 0,vu=typeof Promise=="function"?Promise:void 0,Lm=typeof queueMicrotask=="function"?queueMicrotask:typeof vu<"u"?function(e){return vu.resolve(null).then(e).catch(Pm)}:Qo;function Pm(e){setTimeout(function(){throw e})}function Jl(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Ur(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Ur(t)}function Xt(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function yu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var lr=Math.random().toString(36).slice(2),kt="__reactFiber$"+lr,Qr="__reactProps$"+lr,At="__reactContainer$"+lr,Ko="__reactEvents$"+lr,Im="__reactListeners$"+lr,Mm="__reactHandles$"+lr;function fn(e){var t=e[kt];if(t)return t;for(var n=e.parentNode;n;){if(t=n[At]||n[kt]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=yu(e);e!==null;){if(n=e[kt])return n;e=yu(e)}return t}e=n,n=e.parentNode}return null}function ri(e){return e=e[kt]||e[At],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function An(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(D(33))}function wl(e){return e[Qr]||null}var qo=[],Dn=-1;function ln(e){return{current:e}}function se(e){0>Dn||(e.current=qo[Dn],qo[Dn]=null,Dn--)}function le(e,t){Dn++,qo[Dn]=e.current,e.current=t}var nn={},Le=ln(nn),Ue=ln(!1),yn=nn;function Gn(e,t){var n=e.type.contextTypes;if(!n)return nn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function $e(e){return e=e.childContextTypes,e!=null}function Zi(){se(Ue),se(Le)}function xu(e,t,n){if(Le.current!==nn)throw Error(D(168));le(Le,t),le(Ue,n)}function qd(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(D(108,yh(e)||"Unknown",i));return pe({},n,r)}function el(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||nn,yn=Le.current,le(Le,e),le(Ue,Ue.current),!0}function ku(e,t,n){var r=e.stateNode;if(!r)throw Error(D(169));n?(e=qd(e,t,yn),r.__reactInternalMemoizedMergedChildContext=e,se(Ue),se(Le),le(Le,e)):se(Ue),le(Ue,n)}var Tt=null,Sl=!1,Zl=!1;function Yd(e){Tt===null?Tt=[e]:Tt.push(e)}function Am(e){Sl=!0,Yd(e)}function on(){if(!Zl&&Tt!==null){Zl=!0;var e=0,t=ne;try{var n=Tt;for(ne=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}Tt=null,Sl=!1}catch(i){throw Tt!==null&&(Tt=Tt.slice(e+1)),kd(Ra,on),i}finally{ne=t,Zl=!1}}return null}var Rn=[],On=0,tl=null,nl=0,nt=[],rt=0,xn=null,Lt=1,Pt="";function cn(e,t){Rn[On++]=nl,Rn[On++]=tl,tl=e,nl=t}function Xd(e,t,n){nt[rt++]=Lt,nt[rt++]=Pt,nt[rt++]=xn,xn=e;var r=Lt;e=Pt;var i=32-ht(r)-1;r&=~(1<<i),n+=1;var l=32-ht(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,Lt=1<<32-ht(t)+i|n<<i|r,Pt=l+e}else Lt=1<<l|n<<i|r,Pt=e}function Qa(e){e.return!==null&&(cn(e,1),Xd(e,1,0))}function Ka(e){for(;e===tl;)tl=Rn[--On],Rn[On]=null,nl=Rn[--On],Rn[On]=null;for(;e===xn;)xn=nt[--rt],nt[rt]=null,Pt=nt[--rt],nt[rt]=null,Lt=nt[--rt],nt[rt]=null}var Je=null,Xe=null,ue=!1,ft=null;function Gd(e,t){var n=lt(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function wu(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,Je=e,Xe=Xt(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,Je=e,Xe=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=xn!==null?{id:Lt,overflow:Pt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=lt(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,Je=e,Xe=null,!0):!1;default:return!1}}function Yo(e){return(e.mode&1)!==0&&(e.flags&128)===0}function Xo(e){if(ue){var t=Xe;if(t){var n=t;if(!wu(e,t)){if(Yo(e))throw Error(D(418));t=Xt(n.nextSibling);var r=Je;t&&wu(e,t)?Gd(r,n):(e.flags=e.flags&-4097|2,ue=!1,Je=e)}}else{if(Yo(e))throw Error(D(418));e.flags=e.flags&-4097|2,ue=!1,Je=e}}}function Su(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;Je=e}function xi(e){if(e!==Je)return!1;if(!ue)return Su(e),ue=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!Wo(e.type,e.memoizedProps)),t&&(t=Xe)){if(Yo(e))throw Jd(),Error(D(418));for(;t;)Gd(e,t),t=Xt(t.nextSibling)}if(Su(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(D(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){Xe=Xt(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}Xe=null}}else Xe=Je?Xt(e.stateNode.nextSibling):null;return!0}function Jd(){for(var e=Xe;e;)e=Xt(e.nextSibling)}function Jn(){Xe=Je=null,ue=!1}function qa(e){ft===null?ft=[e]:ft.push(e)}var Dm=Ot.ReactCurrentBatchConfig;function mr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(D(309));var r=n.stateNode}if(!r)throw Error(D(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(D(284));if(!n._owner)throw Error(D(290,e))}return e}function ki(e,t){throw e=Object.prototype.toString.call(t),Error(D(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function bu(e){var t=e._init;return t(e._payload)}function Zd(e){function t(h,v){if(e){var y=h.deletions;y===null?(h.deletions=[v],h.flags|=16):y.push(v)}}function n(h,v){if(!e)return null;for(;v!==null;)t(h,v),v=v.sibling;return null}function r(h,v){for(h=new Map;v!==null;)v.key!==null?h.set(v.key,v):h.set(v.index,v),v=v.sibling;return h}function i(h,v){return h=en(h,v),h.index=0,h.sibling=null,h}function l(h,v,y){return h.index=y,e?(y=h.alternate,y!==null?(y=y.index,y<v?(h.flags|=2,v):y):(h.flags|=2,v)):(h.flags|=1048576,v)}function o(h){return e&&h.alternate===null&&(h.flags|=2),h}function a(h,v,y,b){return v===null||v.tag!==6?(v=oo(y,h.mode,b),v.return=h,v):(v=i(v,y),v.return=h,v)}function s(h,v,y,b){var _=y.type;return _===Ln?d(h,v,y.props.children,b,y.key):v!==null&&(v.elementType===_||typeof _=="object"&&_!==null&&_.$$typeof===Ut&&bu(_)===v.type)?(b=i(v,y.props),b.ref=mr(h,v,y),b.return=h,b):(b=Bi(y.type,y.key,y.props,null,h.mode,b),b.ref=mr(h,v,y),b.return=h,b)}function c(h,v,y,b){return v===null||v.tag!==4||v.stateNode.containerInfo!==y.containerInfo||v.stateNode.implementation!==y.implementation?(v=ao(y,h.mode,b),v.return=h,v):(v=i(v,y.children||[]),v.return=h,v)}function d(h,v,y,b,_){return v===null||v.tag!==7?(v=vn(y,h.mode,b,_),v.return=h,v):(v=i(v,y),v.return=h,v)}function p(h,v,y){if(typeof v=="string"&&v!==""||typeof v=="number")return v=oo(""+v,h.mode,y),v.return=h,v;if(typeof v=="object"&&v!==null){switch(v.$$typeof){case ui:return y=Bi(v.type,v.key,v.props,null,h.mode,y),y.ref=mr(h,null,v),y.return=h,y;case Tn:return v=ao(v,h.mode,y),v.return=h,v;case Ut:var b=v._init;return p(h,b(v._payload),y)}if(wr(v)||cr(v))return v=vn(v,h.mode,y,null),v.return=h,v;ki(h,v)}return null}function m(h,v,y,b){var _=v!==null?v.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return _!==null?null:a(h,v,""+y,b);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case ui:return y.key===_?s(h,v,y,b):null;case Tn:return y.key===_?c(h,v,y,b):null;case Ut:return _=y._init,m(h,v,_(y._payload),b)}if(wr(y)||cr(y))return _!==null?null:d(h,v,y,b,null);ki(h,y)}return null}function f(h,v,y,b,_){if(typeof b=="string"&&b!==""||typeof b=="number")return h=h.get(y)||null,a(v,h,""+b,_);if(typeof b=="object"&&b!==null){switch(b.$$typeof){case ui:return h=h.get(b.key===null?y:b.key)||null,s(v,h,b,_);case Tn:return h=h.get(b.key===null?y:b.key)||null,c(v,h,b,_);case Ut:var S=b._init;return f(h,v,y,S(b._payload),_)}if(wr(b)||cr(b))return h=h.get(y)||null,d(v,h,b,_,null);ki(v,b)}return null}function k(h,v,y,b){for(var _=null,S=null,L=v,C=v=0,T=null;L!==null&&C<y.length;C++){L.index>C?(T=L,L=null):T=L.sibling;var O=m(h,L,y[C],b);if(O===null){L===null&&(L=T);break}e&&L&&O.alternate===null&&t(h,L),v=l(O,v,C),S===null?_=O:S.sibling=O,S=O,L=T}if(C===y.length)return n(h,L),ue&&cn(h,C),_;if(L===null){for(;C<y.length;C++)L=p(h,y[C],b),L!==null&&(v=l(L,v,C),S===null?_=L:S.sibling=L,S=L);return ue&&cn(h,C),_}for(L=r(h,L);C<y.length;C++)T=f(L,h,C,y[C],b),T!==null&&(e&&T.alternate!==null&&L.delete(T.key===null?C:T.key),v=l(T,v,C),S===null?_=T:S.sibling=T,S=T);return e&&L.forEach(function(j){return t(h,j)}),ue&&cn(h,C),_}function w(h,v,y,b){var _=cr(y);if(typeof _!="function")throw Error(D(150));if(y=_.call(y),y==null)throw Error(D(151));for(var S=_=null,L=v,C=v=0,T=null,O=y.next();L!==null&&!O.done;C++,O=y.next()){L.index>C?(T=L,L=null):T=L.sibling;var j=m(h,L,O.value,b);if(j===null){L===null&&(L=T);break}e&&L&&j.alternate===null&&t(h,L),v=l(j,v,C),S===null?_=j:S.sibling=j,S=j,L=T}if(O.done)return n(h,L),ue&&cn(h,C),_;if(L===null){for(;!O.done;C++,O=y.next())O=p(h,O.value,b),O!==null&&(v=l(O,v,C),S===null?_=O:S.sibling=O,S=O);return ue&&cn(h,C),_}for(L=r(h,L);!O.done;C++,O=y.next())O=f(L,h,C,O.value,b),O!==null&&(e&&O.alternate!==null&&L.delete(O.key===null?C:O.key),v=l(O,v,C),S===null?_=O:S.sibling=O,S=O);return e&&L.forEach(function(I){return t(h,I)}),ue&&cn(h,C),_}function M(h,v,y,b){if(typeof y=="object"&&y!==null&&y.type===Ln&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case ui:e:{for(var _=y.key,S=v;S!==null;){if(S.key===_){if(_=y.type,_===Ln){if(S.tag===7){n(h,S.sibling),v=i(S,y.props.children),v.return=h,h=v;break e}}else if(S.elementType===_||typeof _=="object"&&_!==null&&_.$$typeof===Ut&&bu(_)===S.type){n(h,S.sibling),v=i(S,y.props),v.ref=mr(h,S,y),v.return=h,h=v;break e}n(h,S);break}else t(h,S);S=S.sibling}y.type===Ln?(v=vn(y.props.children,h.mode,b,y.key),v.return=h,h=v):(b=Bi(y.type,y.key,y.props,null,h.mode,b),b.ref=mr(h,v,y),b.return=h,h=b)}return o(h);case Tn:e:{for(S=y.key;v!==null;){if(v.key===S)if(v.tag===4&&v.stateNode.containerInfo===y.containerInfo&&v.stateNode.implementation===y.implementation){n(h,v.sibling),v=i(v,y.children||[]),v.return=h,h=v;break e}else{n(h,v);break}else t(h,v);v=v.sibling}v=ao(y,h.mode,b),v.return=h,h=v}return o(h);case Ut:return S=y._init,M(h,v,S(y._payload),b)}if(wr(y))return k(h,v,y,b);if(cr(y))return w(h,v,y,b);ki(h,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,v!==null&&v.tag===6?(n(h,v.sibling),v=i(v,y),v.return=h,h=v):(n(h,v),v=oo(y,h.mode,b),v.return=h,h=v),o(h)):n(h,v)}return M}var Zn=Zd(!0),ep=Zd(!1),rl=ln(null),il=null,Fn=null,Ya=null;function Xa(){Ya=Fn=il=null}function Ga(e){var t=rl.current;se(rl),e._currentValue=t}function Go(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Qn(e,t){il=e,Ya=Fn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(Be=!0),e.firstContext=null)}function at(e){var t=e._currentValue;if(Ya!==e)if(e={context:e,memoizedValue:t,next:null},Fn===null){if(il===null)throw Error(D(308));Fn=e,il.dependencies={lanes:0,firstContext:e}}else Fn=Fn.next=e;return t}var hn=null;function Ja(e){hn===null?hn=[e]:hn.push(e)}function tp(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,Ja(t)):(n.next=i.next,i.next=n),t.interleaved=n,Dt(e,r)}function Dt(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var $t=!1;function Za(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function np(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function It(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function Gt(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,Z&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Dt(e,n)}return i=r.interleaved,i===null?(t.next=t,Ja(r)):(t.next=i.next,i.next=t),r.interleaved=t,Dt(e,n)}function Mi(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Oa(e,n)}}function ju(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function ll(e,t,n,r){var i=e.updateQueue;$t=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,c=s.next;s.next=null,o===null?l=c:o.next=c,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=s))}if(l!==null){var p=i.baseState;o=0,d=c=s=null,a=l;do{var m=a.lane,f=a.eventTime;if((r&m)===m){d!==null&&(d=d.next={eventTime:f,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var k=e,w=a;switch(m=t,f=n,w.tag){case 1:if(k=w.payload,typeof k=="function"){p=k.call(f,p,m);break e}p=k;break e;case 3:k.flags=k.flags&-65537|128;case 0:if(k=w.payload,m=typeof k=="function"?k.call(f,p,m):k,m==null)break e;p=pe({},p,m);break e;case 2:$t=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,m=i.effects,m===null?i.effects=[a]:m.push(a))}else f={eventTime:f,lane:m,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=f,s=p):d=d.next=f,o|=m;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;m=a,a=m.next,m.next=null,i.lastBaseUpdate=m,i.shared.pending=null}}while(!0);if(d===null&&(s=p),i.baseState=s,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);wn|=o,e.lanes=o,e.memoizedState=p}}function Cu(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(D(191,i));i.call(r)}}}var ii={},bt=ln(ii),Kr=ln(ii),qr=ln(ii);function mn(e){if(e===ii)throw Error(D(174));return e}function es(e,t){switch(le(qr,t),le(Kr,e),le(bt,ii),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Lo(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Lo(t,e)}se(bt),le(bt,t)}function er(){se(bt),se(Kr),se(qr)}function rp(e){mn(qr.current);var t=mn(bt.current),n=Lo(t,e.type);t!==n&&(le(Kr,e),le(bt,n))}function ts(e){Kr.current===e&&(se(bt),se(Kr))}var ce=ln(0);function ol(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var eo=[];function ns(){for(var e=0;e<eo.length;e++)eo[e]._workInProgressVersionPrimary=null;eo.length=0}var Ai=Ot.ReactCurrentDispatcher,to=Ot.ReactCurrentBatchConfig,kn=0,de=null,xe=null,Se=null,al=!1,zr=!1,Yr=0,Rm=0;function _e(){throw Error(D(321))}function rs(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!gt(e[n],t[n]))return!1;return!0}function is(e,t,n,r,i,l){if(kn=l,de=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Ai.current=e===null||e.memoizedState===null?Um:$m,e=n(r,i),zr){l=0;do{if(zr=!1,Yr=0,25<=l)throw Error(D(301));l+=1,Se=xe=null,t.updateQueue=null,Ai.current=Hm,e=n(r,i)}while(zr)}if(Ai.current=sl,t=xe!==null&&xe.next!==null,kn=0,Se=xe=de=null,al=!1,t)throw Error(D(300));return e}function ls(){var e=Yr!==0;return Yr=0,e}function yt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return Se===null?de.memoizedState=Se=e:Se=Se.next=e,Se}function st(){if(xe===null){var e=de.alternate;e=e!==null?e.memoizedState:null}else e=xe.next;var t=Se===null?de.memoizedState:Se.next;if(t!==null)Se=t,xe=e;else{if(e===null)throw Error(D(310));xe=e,e={memoizedState:xe.memoizedState,baseState:xe.baseState,baseQueue:xe.baseQueue,queue:xe.queue,next:null},Se===null?de.memoizedState=Se=e:Se=Se.next=e}return Se}function Xr(e,t){return typeof t=="function"?t(e):t}function no(e){var t=st(),n=t.queue;if(n===null)throw Error(D(311));n.lastRenderedReducer=e;var r=xe,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,c=l;do{var d=c.lane;if((kn&d)===d)s!==null&&(s=s.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var p={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};s===null?(a=s=p,o=r):s=s.next=p,de.lanes|=d,wn|=d}c=c.next}while(c!==null&&c!==l);s===null?o=r:s.next=a,gt(r,t.memoizedState)||(Be=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,de.lanes|=l,wn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function ro(e){var t=st(),n=t.queue;if(n===null)throw Error(D(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);gt(l,t.memoizedState)||(Be=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function ip(){}function lp(e,t){var n=de,r=st(),i=t(),l=!gt(r.memoizedState,i);if(l&&(r.memoizedState=i,Be=!0),r=r.queue,os(sp.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||Se!==null&&Se.memoizedState.tag&1){if(n.flags|=2048,Gr(9,ap.bind(null,n,r,i,t),void 0,null),be===null)throw Error(D(349));kn&30||op(n,t,i)}return i}function op(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=de.updateQueue,t===null?(t={lastEffect:null,stores:null},de.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function ap(e,t,n,r){t.value=n,t.getSnapshot=r,up(t)&&cp(e)}function sp(e,t,n){return n(function(){up(t)&&cp(e)})}function up(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!gt(e,n)}catch{return!0}}function cp(e){var t=Dt(e,1);t!==null&&mt(t,e,1,-1)}function Eu(e){var t=yt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:Xr,lastRenderedState:e},t.queue=e,e=e.dispatch=Bm.bind(null,de,e),[t.memoizedState,e]}function Gr(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=de.updateQueue,t===null?(t={lastEffect:null,stores:null},de.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function dp(){return st().memoizedState}function Di(e,t,n,r){var i=yt();de.flags|=e,i.memoizedState=Gr(1|t,n,void 0,r===void 0?null:r)}function bl(e,t,n,r){var i=st();r=r===void 0?null:r;var l=void 0;if(xe!==null){var o=xe.memoizedState;if(l=o.destroy,r!==null&&rs(r,o.deps)){i.memoizedState=Gr(t,n,l,r);return}}de.flags|=e,i.memoizedState=Gr(1|t,n,l,r)}function Nu(e,t){return Di(8390656,8,e,t)}function os(e,t){return bl(2048,8,e,t)}function pp(e,t){return bl(4,2,e,t)}function fp(e,t){return bl(4,4,e,t)}function hp(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function mp(e,t,n){return n=n!=null?n.concat([e]):null,bl(4,4,hp.bind(null,t,e),n)}function as(){}function gp(e,t){var n=st();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&rs(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function vp(e,t){var n=st();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&rs(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function yp(e,t,n){return kn&21?(gt(n,t)||(n=bd(),de.lanes|=n,wn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,Be=!0),e.memoizedState=n)}function Om(e,t){var n=ne;ne=n!==0&&4>n?n:4,e(!0);var r=to.transition;to.transition={};try{e(!1),t()}finally{ne=n,to.transition=r}}function xp(){return st().memoizedState}function Fm(e,t,n){var r=Zt(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},kp(e))wp(t,n);else if(n=tp(e,t,n,r),n!==null){var i=Ae();mt(n,e,r,i),Sp(n,t,r)}}function Bm(e,t,n){var r=Zt(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(kp(e))wp(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,gt(a,o)){var s=t.interleaved;s===null?(i.next=i,Ja(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=tp(e,t,i,r),n!==null&&(i=Ae(),mt(n,e,r,i),Sp(n,t,r))}}function kp(e){var t=e.alternate;return e===de||t!==null&&t===de}function wp(e,t){zr=al=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function Sp(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Oa(e,n)}}var sl={readContext:at,useCallback:_e,useContext:_e,useEffect:_e,useImperativeHandle:_e,useInsertionEffect:_e,useLayoutEffect:_e,useMemo:_e,useReducer:_e,useRef:_e,useState:_e,useDebugValue:_e,useDeferredValue:_e,useTransition:_e,useMutableSource:_e,useSyncExternalStore:_e,useId:_e,unstable_isNewReconciler:!1},Um={readContext:at,useCallback:function(e,t){return yt().memoizedState=[e,t===void 0?null:t],e},useContext:at,useEffect:Nu,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Di(4194308,4,hp.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Di(4194308,4,e,t)},useInsertionEffect:function(e,t){return Di(4,2,e,t)},useMemo:function(e,t){var n=yt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=yt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Fm.bind(null,de,e),[r.memoizedState,e]},useRef:function(e){var t=yt();return e={current:e},t.memoizedState=e},useState:Eu,useDebugValue:as,useDeferredValue:function(e){return yt().memoizedState=e},useTransition:function(){var e=Eu(!1),t=e[0];return e=Om.bind(null,e[1]),yt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=de,i=yt();if(ue){if(n===void 0)throw Error(D(407));n=n()}else{if(n=t(),be===null)throw Error(D(349));kn&30||op(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Nu(sp.bind(null,r,l,e),[e]),r.flags|=2048,Gr(9,ap.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=yt(),t=be.identifierPrefix;if(ue){var n=Pt,r=Lt;n=(r&~(1<<32-ht(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=Yr++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Rm++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},$m={readContext:at,useCallback:gp,useContext:at,useEffect:os,useImperativeHandle:mp,useInsertionEffect:pp,useLayoutEffect:fp,useMemo:vp,useReducer:no,useRef:dp,useState:function(){return no(Xr)},useDebugValue:as,useDeferredValue:function(e){var t=st();return yp(t,xe.memoizedState,e)},useTransition:function(){var e=no(Xr)[0],t=st().memoizedState;return[e,t]},useMutableSource:ip,useSyncExternalStore:lp,useId:xp,unstable_isNewReconciler:!1},Hm={readContext:at,useCallback:gp,useContext:at,useEffect:os,useImperativeHandle:mp,useInsertionEffect:pp,useLayoutEffect:fp,useMemo:vp,useReducer:ro,useRef:dp,useState:function(){return ro(Xr)},useDebugValue:as,useDeferredValue:function(e){var t=st();return xe===null?t.memoizedState=e:yp(t,xe.memoizedState,e)},useTransition:function(){var e=ro(Xr)[0],t=st().memoizedState;return[e,t]},useMutableSource:ip,useSyncExternalStore:lp,useId:xp,unstable_isNewReconciler:!1};function dt(e,t){if(e&&e.defaultProps){t=pe({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function Jo(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:pe({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var jl={isMounted:function(e){return(e=e._reactInternals)?jn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Ae(),i=Zt(e),l=It(r,i);l.payload=t,n!=null&&(l.callback=n),t=Gt(e,l,i),t!==null&&(mt(t,e,i,r),Mi(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Ae(),i=Zt(e),l=It(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=Gt(e,l,i),t!==null&&(mt(t,e,i,r),Mi(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Ae(),r=Zt(e),i=It(n,r);i.tag=2,t!=null&&(i.callback=t),t=Gt(e,i,r),t!==null&&(mt(t,e,r,n),Mi(t,e,r))}};function _u(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Hr(n,r)||!Hr(i,l):!0}function bp(e,t,n){var r=!1,i=nn,l=t.contextType;return typeof l=="object"&&l!==null?l=at(l):(i=$e(t)?yn:Le.current,r=t.contextTypes,l=(r=r!=null)?Gn(e,i):nn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=jl,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function zu(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&jl.enqueueReplaceState(t,t.state,null)}function Zo(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},Za(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=at(l):(l=$e(t)?yn:Le.current,i.context=Gn(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(Jo(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&jl.enqueueReplaceState(i,i.state,null),ll(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function tr(e,t){try{var n="",r=t;do n+=vh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function io(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function ea(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var Vm=typeof WeakMap=="function"?WeakMap:Map;function jp(e,t,n){n=It(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){cl||(cl=!0,ca=r),ea(e,t)},n}function Cp(e,t,n){n=It(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){ea(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){ea(e,t),typeof r!="function"&&(Jt===null?Jt=new Set([this]):Jt.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function Tu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new Vm;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=ig.bind(null,e,t,n),t.then(e,e))}function Lu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Pu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=It(-1,1),t.tag=2,Gt(n,t,1))),n.lanes|=1),e)}var Wm=Ot.ReactCurrentOwner,Be=!1;function Me(e,t,n,r){t.child=e===null?ep(t,null,n,r):Zn(t,e.child,n,r)}function Iu(e,t,n,r,i){n=n.render;var l=t.ref;return Qn(t,i),r=is(e,t,n,r,l,i),n=ls(),e!==null&&!Be?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Rt(e,t,i)):(ue&&n&&Qa(t),t.flags|=1,Me(e,t,r,i),t.child)}function Mu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!ms(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,Ep(e,t,l,r,i)):(e=Bi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Hr,n(o,r)&&e.ref===t.ref)return Rt(e,t,i)}return t.flags|=1,e=en(l,r),e.ref=t.ref,e.return=t,t.child=e}function Ep(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Hr(l,r)&&e.ref===t.ref)if(Be=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(Be=!0);else return t.lanes=e.lanes,Rt(e,t,i)}return ta(e,t,n,r,i)}function Np(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},le(Un,qe),qe|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,le(Un,qe),qe|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,le(Un,qe),qe|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,le(Un,qe),qe|=r;return Me(e,t,i,n),t.child}function _p(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ta(e,t,n,r,i){var l=$e(n)?yn:Le.current;return l=Gn(t,l),Qn(t,i),n=is(e,t,n,r,l,i),r=ls(),e!==null&&!Be?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Rt(e,t,i)):(ue&&r&&Qa(t),t.flags|=1,Me(e,t,n,i),t.child)}function Au(e,t,n,r,i){if($e(n)){var l=!0;el(t)}else l=!1;if(Qn(t,i),t.stateNode===null)Ri(e,t),bp(t,n,r),Zo(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=at(c):(c=$e(n)?yn:Le.current,c=Gn(t,c));var d=n.getDerivedStateFromProps,p=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";p||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==c)&&zu(t,o,r,c),$t=!1;var m=t.memoizedState;o.state=m,ll(t,r,o,i),s=t.memoizedState,a!==r||m!==s||Ue.current||$t?(typeof d=="function"&&(Jo(t,n,d,r),s=t.memoizedState),(a=$t||_u(t,n,a,r,m,s,c))?(p||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,np(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:dt(t.type,a),o.props=c,p=t.pendingProps,m=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=at(s):(s=$e(n)?yn:Le.current,s=Gn(t,s));var f=n.getDerivedStateFromProps;(d=typeof f=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==p||m!==s)&&zu(t,o,r,s),$t=!1,m=t.memoizedState,o.state=m,ll(t,r,o,i);var k=t.memoizedState;a!==p||m!==k||Ue.current||$t?(typeof f=="function"&&(Jo(t,n,f,r),k=t.memoizedState),(c=$t||_u(t,n,c,r,m,k,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,k,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,k,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=k),o.props=r,o.state=k,o.context=s,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=1024),r=!1)}return na(e,t,n,r,l,i)}function na(e,t,n,r,i,l){_p(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&ku(t,n,!1),Rt(e,t,l);r=t.stateNode,Wm.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=Zn(t,e.child,null,l),t.child=Zn(t,null,a,l)):Me(e,t,a,l),t.memoizedState=r.state,i&&ku(t,n,!0),t.child}function zp(e){var t=e.stateNode;t.pendingContext?xu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&xu(e,t.context,!1),es(e,t.containerInfo)}function Du(e,t,n,r,i){return Jn(),qa(i),t.flags|=256,Me(e,t,n,r),t.child}var ra={dehydrated:null,treeContext:null,retryLane:0};function ia(e){return{baseLanes:e,cachePool:null,transitions:null}}function Tp(e,t,n){var r=t.pendingProps,i=ce.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),le(ce,i&1),e===null)return Xo(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Nl(o,r,0,null),e=vn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=ia(n),t.memoizedState=ra,e):ss(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return Qm(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=en(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=en(a,l):(l=vn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?ia(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=ra,r}return l=e.child,e=l.sibling,r=en(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function ss(e,t){return t=Nl({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function wi(e,t,n,r){return r!==null&&qa(r),Zn(t,e.child,null,n),e=ss(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function Qm(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=io(Error(D(422))),wi(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Nl({mode:"visible",children:r.children},i,0,null),l=vn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&Zn(t,e.child,null,o),t.child.memoizedState=ia(o),t.memoizedState=ra,l);if(!(t.mode&1))return wi(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(D(419)),r=io(l,r,void 0),wi(e,t,o,r)}if(a=(o&e.childLanes)!==0,Be||a){if(r=be,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Dt(e,i),mt(r,e,i,-1))}return hs(),r=io(Error(D(421))),wi(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=lg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,Xe=Xt(i.nextSibling),Je=t,ue=!0,ft=null,e!==null&&(nt[rt++]=Lt,nt[rt++]=Pt,nt[rt++]=xn,Lt=e.id,Pt=e.overflow,xn=t),t=ss(t,r.children),t.flags|=4096,t)}function Ru(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),Go(e.return,t,n)}function lo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Lp(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(Me(e,t,r.children,n),r=ce.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Ru(e,n,t);else if(e.tag===19)Ru(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(le(ce,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&ol(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),lo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&ol(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}lo(t,!0,n,null,l);break;case"together":lo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Ri(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Rt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),wn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(D(153));if(t.child!==null){for(e=t.child,n=en(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=en(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function Km(e,t,n){switch(t.tag){case 3:zp(t),Jn();break;case 5:rp(t);break;case 1:$e(t.type)&&el(t);break;case 4:es(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;le(rl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(le(ce,ce.current&1),t.flags|=128,null):n&t.child.childLanes?Tp(e,t,n):(le(ce,ce.current&1),e=Rt(e,t,n),e!==null?e.sibling:null);le(ce,ce.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Lp(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),le(ce,ce.current),r)break;return null;case 22:case 23:return t.lanes=0,Np(e,t,n)}return Rt(e,t,n)}var Pp,la,Ip,Mp;Pp=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};la=function(){};Ip=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,mn(bt.current);var l=null;switch(n){case"input":i=No(e,i),r=No(e,r),l=[];break;case"select":i=pe({},i,{value:void 0}),r=pe({},r,{value:void 0}),l=[];break;case"textarea":i=To(e,i),r=To(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=Ji)}Po(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(Dr.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var s=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&s!==a&&(s!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(c,n)),n=s;else c==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(c,s)):c==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(c,""+s):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(Dr.hasOwnProperty(c)?(s!=null&&c==="onScroll"&&ae("scroll",e),l||a===s||(l=[])):(l=l||[]).push(c,s))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};Mp=function(e,t,n,r){n!==r&&(t.flags|=4)};function gr(e,t){if(!ue)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function ze(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function qm(e,t,n){var r=t.pendingProps;switch(Ka(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return ze(t),null;case 1:return $e(t.type)&&Zi(),ze(t),null;case 3:return r=t.stateNode,er(),se(Ue),se(Le),ns(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(xi(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,ft!==null&&(fa(ft),ft=null))),la(e,t),ze(t),null;case 5:ts(t);var i=mn(qr.current);if(n=t.type,e!==null&&t.stateNode!=null)Ip(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(D(166));return ze(t),null}if(e=mn(bt.current),xi(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[kt]=t,r[Qr]=l,e=(t.mode&1)!==0,n){case"dialog":ae("cancel",r),ae("close",r);break;case"iframe":case"object":case"embed":ae("load",r);break;case"video":case"audio":for(i=0;i<br.length;i++)ae(br[i],r);break;case"source":ae("error",r);break;case"img":case"image":case"link":ae("error",r),ae("load",r);break;case"details":ae("toggle",r);break;case"input":Qs(r,l),ae("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},ae("invalid",r);break;case"textarea":qs(r,l),ae("invalid",r)}Po(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&yi(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&yi(r.textContent,a,e),i=["children",""+a]):Dr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&ae("scroll",r)}switch(n){case"input":ci(r),Ks(r,l,!0);break;case"textarea":ci(r),Ys(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=Ji)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=sd(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[kt]=t,e[Qr]=r,Pp(e,t,!1,!1),t.stateNode=e;e:{switch(o=Io(n,r),n){case"dialog":ae("cancel",e),ae("close",e),i=r;break;case"iframe":case"object":case"embed":ae("load",e),i=r;break;case"video":case"audio":for(i=0;i<br.length;i++)ae(br[i],e);i=r;break;case"source":ae("error",e),i=r;break;case"img":case"image":case"link":ae("error",e),ae("load",e),i=r;break;case"details":ae("toggle",e),i=r;break;case"input":Qs(e,r),i=No(e,r),ae("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=pe({},r,{value:void 0}),ae("invalid",e);break;case"textarea":qs(e,r),i=To(e,r),ae("invalid",e);break;default:i=r}Po(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?dd(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&ud(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Rr(e,s):typeof s=="number"&&Rr(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Dr.hasOwnProperty(l)?s!=null&&l==="onScroll"&&ae("scroll",e):s!=null&&Pa(e,l,s,o))}switch(n){case"input":ci(e),Ks(e,r,!1);break;case"textarea":ci(e),Ys(e);break;case"option":r.value!=null&&e.setAttribute("value",""+tn(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?$n(e,!!r.multiple,l,!1):r.defaultValue!=null&&$n(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=Ji)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return ze(t),null;case 6:if(e&&t.stateNode!=null)Mp(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(D(166));if(n=mn(qr.current),mn(bt.current),xi(t)){if(r=t.stateNode,n=t.memoizedProps,r[kt]=t,(l=r.nodeValue!==n)&&(e=Je,e!==null))switch(e.tag){case 3:yi(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&yi(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[kt]=t,t.stateNode=r}return ze(t),null;case 13:if(se(ce),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(ue&&Xe!==null&&t.mode&1&&!(t.flags&128))Jd(),Jn(),t.flags|=98560,l=!1;else if(l=xi(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(D(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(D(317));l[kt]=t}else Jn(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;ze(t),l=!1}else ft!==null&&(fa(ft),ft=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||ce.current&1?ke===0&&(ke=3):hs())),t.updateQueue!==null&&(t.flags|=4),ze(t),null);case 4:return er(),la(e,t),e===null&&Vr(t.stateNode.containerInfo),ze(t),null;case 10:return Ga(t.type._context),ze(t),null;case 17:return $e(t.type)&&Zi(),ze(t),null;case 19:if(se(ce),l=t.memoizedState,l===null)return ze(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)gr(l,!1);else{if(ke!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=ol(e),o!==null){for(t.flags|=128,gr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return le(ce,ce.current&1|2),t.child}e=e.sibling}l.tail!==null&&he()>nr&&(t.flags|=128,r=!0,gr(l,!1),t.lanes=4194304)}else{if(!r)if(e=ol(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),gr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!ue)return ze(t),null}else 2*he()-l.renderingStartTime>nr&&n!==1073741824&&(t.flags|=128,r=!0,gr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=he(),t.sibling=null,n=ce.current,le(ce,r?n&1|2:n&1),t):(ze(t),null);case 22:case 23:return fs(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?qe&1073741824&&(ze(t),t.subtreeFlags&6&&(t.flags|=8192)):ze(t),null;case 24:return null;case 25:return null}throw Error(D(156,t.tag))}function Ym(e,t){switch(Ka(t),t.tag){case 1:return $e(t.type)&&Zi(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return er(),se(Ue),se(Le),ns(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return ts(t),null;case 13:if(se(ce),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(D(340));Jn()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return se(ce),null;case 4:return er(),null;case 10:return Ga(t.type._context),null;case 22:case 23:return fs(),null;case 24:return null;default:return null}}var Si=!1,Te=!1,Xm=typeof WeakSet=="function"?WeakSet:Set,U=null;function Bn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){fe(e,t,r)}else n.current=null}function oa(e,t,n){try{n()}catch(r){fe(e,t,r)}}var Ou=!1;function Gm(e,t){if(Ho=Yi,e=Fd(),Wa(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,c=0,d=0,p=e,m=null;t:for(;;){for(var f;p!==n||i!==0&&p.nodeType!==3||(a=o+i),p!==l||r!==0&&p.nodeType!==3||(s=o+r),p.nodeType===3&&(o+=p.nodeValue.length),(f=p.firstChild)!==null;)m=p,p=f;for(;;){if(p===e)break t;if(m===n&&++c===i&&(a=o),m===l&&++d===r&&(s=o),(f=p.nextSibling)!==null)break;p=m,m=p.parentNode}p=f}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Vo={focusedElem:e,selectionRange:n},Yi=!1,U=t;U!==null;)if(t=U,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,U=e;else for(;U!==null;){t=U;try{var k=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(k!==null){var w=k.memoizedProps,M=k.memoizedState,h=t.stateNode,v=h.getSnapshotBeforeUpdate(t.elementType===t.type?w:dt(t.type,w),M);h.__reactInternalSnapshotBeforeUpdate=v}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(D(163))}}catch(b){fe(t,t.return,b)}if(e=t.sibling,e!==null){e.return=t.return,U=e;break}U=t.return}return k=Ou,Ou=!1,k}function Tr(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&oa(t,n,l)}i=i.next}while(i!==r)}}function Cl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function aa(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Ap(e){var t=e.alternate;t!==null&&(e.alternate=null,Ap(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[kt],delete t[Qr],delete t[Ko],delete t[Im],delete t[Mm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Dp(e){return e.tag===5||e.tag===3||e.tag===4}function Fu(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Dp(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function sa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=Ji));else if(r!==4&&(e=e.child,e!==null))for(sa(e,t,n),e=e.sibling;e!==null;)sa(e,t,n),e=e.sibling}function ua(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(ua(e,t,n),e=e.sibling;e!==null;)ua(e,t,n),e=e.sibling}var je=null,pt=!1;function Ft(e,t,n){for(n=n.child;n!==null;)Rp(e,t,n),n=n.sibling}function Rp(e,t,n){if(St&&typeof St.onCommitFiberUnmount=="function")try{St.onCommitFiberUnmount(vl,n)}catch{}switch(n.tag){case 5:Te||Bn(n,t);case 6:var r=je,i=pt;je=null,Ft(e,t,n),je=r,pt=i,je!==null&&(pt?(e=je,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):je.removeChild(n.stateNode));break;case 18:je!==null&&(pt?(e=je,n=n.stateNode,e.nodeType===8?Jl(e.parentNode,n):e.nodeType===1&&Jl(e,n),Ur(e)):Jl(je,n.stateNode));break;case 4:r=je,i=pt,je=n.stateNode.containerInfo,pt=!0,Ft(e,t,n),je=r,pt=i;break;case 0:case 11:case 14:case 15:if(!Te&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&oa(n,t,o),i=i.next}while(i!==r)}Ft(e,t,n);break;case 1:if(!Te&&(Bn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){fe(n,t,a)}Ft(e,t,n);break;case 21:Ft(e,t,n);break;case 22:n.mode&1?(Te=(r=Te)||n.memoizedState!==null,Ft(e,t,n),Te=r):Ft(e,t,n);break;default:Ft(e,t,n)}}function Bu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new Xm),t.forEach(function(r){var i=og.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function ct(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:je=a.stateNode,pt=!1;break e;case 3:je=a.stateNode.containerInfo,pt=!0;break e;case 4:je=a.stateNode.containerInfo,pt=!0;break e}a=a.return}if(je===null)throw Error(D(160));Rp(l,o,i),je=null,pt=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(c){fe(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Op(t,e),t=t.sibling}function Op(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(ct(t,e),vt(e),r&4){try{Tr(3,e,e.return),Cl(3,e)}catch(w){fe(e,e.return,w)}try{Tr(5,e,e.return)}catch(w){fe(e,e.return,w)}}break;case 1:ct(t,e),vt(e),r&512&&n!==null&&Bn(n,n.return);break;case 5:if(ct(t,e),vt(e),r&512&&n!==null&&Bn(n,n.return),e.flags&32){var i=e.stateNode;try{Rr(i,"")}catch(w){fe(e,e.return,w)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&od(i,l),Io(a,o);var c=Io(a,l);for(o=0;o<s.length;o+=2){var d=s[o],p=s[o+1];d==="style"?dd(i,p):d==="dangerouslySetInnerHTML"?ud(i,p):d==="children"?Rr(i,p):Pa(i,d,p,c)}switch(a){case"input":_o(i,l);break;case"textarea":ad(i,l);break;case"select":var m=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var f=l.value;f!=null?$n(i,!!l.multiple,f,!1):m!==!!l.multiple&&(l.defaultValue!=null?$n(i,!!l.multiple,l.defaultValue,!0):$n(i,!!l.multiple,l.multiple?[]:"",!1))}i[Qr]=l}catch(w){fe(e,e.return,w)}}break;case 6:if(ct(t,e),vt(e),r&4){if(e.stateNode===null)throw Error(D(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(w){fe(e,e.return,w)}}break;case 3:if(ct(t,e),vt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Ur(t.containerInfo)}catch(w){fe(e,e.return,w)}break;case 4:ct(t,e),vt(e);break;case 13:ct(t,e),vt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(ds=he())),r&4&&Bu(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Te=(c=Te)||d,ct(t,e),Te=c):ct(t,e),vt(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for(U=e,d=e.child;d!==null;){for(p=U=d;U!==null;){switch(m=U,f=m.child,m.tag){case 0:case 11:case 14:case 15:Tr(4,m,m.return);break;case 1:Bn(m,m.return);var k=m.stateNode;if(typeof k.componentWillUnmount=="function"){r=m,n=m.return;try{t=r,k.props=t.memoizedProps,k.state=t.memoizedState,k.componentWillUnmount()}catch(w){fe(r,n,w)}}break;case 5:Bn(m,m.return);break;case 22:if(m.memoizedState!==null){$u(p);continue}}f!==null?(f.return=m,U=f):$u(p)}d=d.sibling}e:for(d=null,p=e;;){if(p.tag===5){if(d===null){d=p;try{i=p.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=p.stateNode,s=p.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=cd("display",o))}catch(w){fe(e,e.return,w)}}}else if(p.tag===6){if(d===null)try{p.stateNode.nodeValue=c?"":p.memoizedProps}catch(w){fe(e,e.return,w)}}else if((p.tag!==22&&p.tag!==23||p.memoizedState===null||p===e)&&p.child!==null){p.child.return=p,p=p.child;continue}if(p===e)break e;for(;p.sibling===null;){if(p.return===null||p.return===e)break e;d===p&&(d=null),p=p.return}d===p&&(d=null),p.sibling.return=p.return,p=p.sibling}}break;case 19:ct(t,e),vt(e),r&4&&Bu(e);break;case 21:break;default:ct(t,e),vt(e)}}function vt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Dp(n)){var r=n;break e}n=n.return}throw Error(D(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Rr(i,""),r.flags&=-33);var l=Fu(e);ua(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Fu(e);sa(e,a,o);break;default:throw Error(D(161))}}catch(s){fe(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function Jm(e,t,n){U=e,Fp(e)}function Fp(e,t,n){for(var r=(e.mode&1)!==0;U!==null;){var i=U,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||Si;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||Te;a=Si;var c=Te;if(Si=o,(Te=s)&&!c)for(U=i;U!==null;)o=U,s=o.child,o.tag===22&&o.memoizedState!==null?Hu(i):s!==null?(s.return=o,U=s):Hu(i);for(;l!==null;)U=l,Fp(l),l=l.sibling;U=i,Si=a,Te=c}Uu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,U=l):Uu(e)}}function Uu(e){for(;U!==null;){var t=U;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Te||Cl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Te)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:dt(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Cu(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Cu(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var p=d.dehydrated;p!==null&&Ur(p)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(D(163))}Te||t.flags&512&&aa(t)}catch(m){fe(t,t.return,m)}}if(t===e){U=null;break}if(n=t.sibling,n!==null){n.return=t.return,U=n;break}U=t.return}}function $u(e){for(;U!==null;){var t=U;if(t===e){U=null;break}var n=t.sibling;if(n!==null){n.return=t.return,U=n;break}U=t.return}}function Hu(e){for(;U!==null;){var t=U;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Cl(4,t)}catch(s){fe(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){fe(t,i,s)}}var l=t.return;try{aa(t)}catch(s){fe(t,l,s)}break;case 5:var o=t.return;try{aa(t)}catch(s){fe(t,o,s)}}}catch(s){fe(t,t.return,s)}if(t===e){U=null;break}var a=t.sibling;if(a!==null){a.return=t.return,U=a;break}U=t.return}}var Zm=Math.ceil,ul=Ot.ReactCurrentDispatcher,us=Ot.ReactCurrentOwner,ot=Ot.ReactCurrentBatchConfig,Z=0,be=null,ve=null,Ce=0,qe=0,Un=ln(0),ke=0,Jr=null,wn=0,El=0,cs=0,Lr=null,Fe=null,ds=0,nr=1/0,_t=null,cl=!1,ca=null,Jt=null,bi=!1,Qt=null,dl=0,Pr=0,da=null,Oi=-1,Fi=0;function Ae(){return Z&6?he():Oi!==-1?Oi:Oi=he()}function Zt(e){return e.mode&1?Z&2&&Ce!==0?Ce&-Ce:Dm.transition!==null?(Fi===0&&(Fi=bd()),Fi):(e=ne,e!==0||(e=window.event,e=e===void 0?16:Td(e.type)),e):1}function mt(e,t,n,r){if(50<Pr)throw Pr=0,da=null,Error(D(185));ti(e,n,r),(!(Z&2)||e!==be)&&(e===be&&(!(Z&2)&&(El|=n),ke===4&&Vt(e,Ce)),He(e,r),n===1&&Z===0&&!(t.mode&1)&&(nr=he()+500,Sl&&on()))}function He(e,t){var n=e.callbackNode;Dh(e,t);var r=qi(e,e===be?Ce:0);if(r===0)n!==null&&Js(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&Js(n),t===1)e.tag===0?Am(Vu.bind(null,e)):Yd(Vu.bind(null,e)),Lm(function(){!(Z&6)&&on()}),n=null;else{switch(jd(r)){case 1:n=Ra;break;case 4:n=wd;break;case 16:n=Ki;break;case 536870912:n=Sd;break;default:n=Ki}n=Kp(n,Bp.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function Bp(e,t){if(Oi=-1,Fi=0,Z&6)throw Error(D(327));var n=e.callbackNode;if(Kn()&&e.callbackNode!==n)return null;var r=qi(e,e===be?Ce:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=pl(e,r);else{t=r;var i=Z;Z|=2;var l=$p();(be!==e||Ce!==t)&&(_t=null,nr=he()+500,gn(e,t));do try{ng();break}catch(a){Up(e,a)}while(!0);Xa(),ul.current=l,Z=i,ve!==null?t=0:(be=null,Ce=0,t=ke)}if(t!==0){if(t===2&&(i=Oo(e),i!==0&&(r=i,t=pa(e,i))),t===1)throw n=Jr,gn(e,0),Vt(e,r),He(e,he()),n;if(t===6)Vt(e,r);else{if(i=e.current.alternate,!(r&30)&&!eg(i)&&(t=pl(e,r),t===2&&(l=Oo(e),l!==0&&(r=l,t=pa(e,l))),t===1))throw n=Jr,gn(e,0),Vt(e,r),He(e,he()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(D(345));case 2:dn(e,Fe,_t);break;case 3:if(Vt(e,r),(r&130023424)===r&&(t=ds+500-he(),10<t)){if(qi(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Ae(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=Qo(dn.bind(null,e,Fe,_t),t);break}dn(e,Fe,_t);break;case 4:if(Vt(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-ht(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=he()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*Zm(r/1960))-r,10<r){e.timeoutHandle=Qo(dn.bind(null,e,Fe,_t),r);break}dn(e,Fe,_t);break;case 5:dn(e,Fe,_t);break;default:throw Error(D(329))}}}return He(e,he()),e.callbackNode===n?Bp.bind(null,e):null}function pa(e,t){var n=Lr;return e.current.memoizedState.isDehydrated&&(gn(e,t).flags|=256),e=pl(e,t),e!==2&&(t=Fe,Fe=n,t!==null&&fa(t)),e}function fa(e){Fe===null?Fe=e:Fe.push.apply(Fe,e)}function eg(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!gt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Vt(e,t){for(t&=~cs,t&=~El,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-ht(t),r=1<<n;e[n]=-1,t&=~r}}function Vu(e){if(Z&6)throw Error(D(327));Kn();var t=qi(e,0);if(!(t&1))return He(e,he()),null;var n=pl(e,t);if(e.tag!==0&&n===2){var r=Oo(e);r!==0&&(t=r,n=pa(e,r))}if(n===1)throw n=Jr,gn(e,0),Vt(e,t),He(e,he()),n;if(n===6)throw Error(D(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,dn(e,Fe,_t),He(e,he()),null}function ps(e,t){var n=Z;Z|=1;try{return e(t)}finally{Z=n,Z===0&&(nr=he()+500,Sl&&on())}}function Sn(e){Qt!==null&&Qt.tag===0&&!(Z&6)&&Kn();var t=Z;Z|=1;var n=ot.transition,r=ne;try{if(ot.transition=null,ne=1,e)return e()}finally{ne=r,ot.transition=n,Z=t,!(Z&6)&&on()}}function fs(){qe=Un.current,se(Un)}function gn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,Tm(n)),ve!==null)for(n=ve.return;n!==null;){var r=n;switch(Ka(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&Zi();break;case 3:er(),se(Ue),se(Le),ns();break;case 5:ts(r);break;case 4:er();break;case 13:se(ce);break;case 19:se(ce);break;case 10:Ga(r.type._context);break;case 22:case 23:fs()}n=n.return}if(be=e,ve=e=en(e.current,null),Ce=qe=t,ke=0,Jr=null,cs=El=wn=0,Fe=Lr=null,hn!==null){for(t=0;t<hn.length;t++)if(n=hn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}hn=null}return e}function Up(e,t){do{var n=ve;try{if(Xa(),Ai.current=sl,al){for(var r=de.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}al=!1}if(kn=0,Se=xe=de=null,zr=!1,Yr=0,us.current=null,n===null||n.return===null){ke=1,Jr=t,ve=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=Ce,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var c=s,d=a,p=d.tag;if(!(d.mode&1)&&(p===0||p===11||p===15)){var m=d.alternate;m?(d.updateQueue=m.updateQueue,d.memoizedState=m.memoizedState,d.lanes=m.lanes):(d.updateQueue=null,d.memoizedState=null)}var f=Lu(o);if(f!==null){f.flags&=-257,Pu(f,o,a,l,t),f.mode&1&&Tu(l,c,t),t=f,s=c;var k=t.updateQueue;if(k===null){var w=new Set;w.add(s),t.updateQueue=w}else k.add(s);break e}else{if(!(t&1)){Tu(l,c,t),hs();break e}s=Error(D(426))}}else if(ue&&a.mode&1){var M=Lu(o);if(M!==null){!(M.flags&65536)&&(M.flags|=256),Pu(M,o,a,l,t),qa(tr(s,a));break e}}l=s=tr(s,a),ke!==4&&(ke=2),Lr===null?Lr=[l]:Lr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var h=jp(l,s,t);ju(l,h);break e;case 1:a=s;var v=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof v.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(Jt===null||!Jt.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var b=Cp(l,a,t);ju(l,b);break e}}l=l.return}while(l!==null)}Vp(n)}catch(_){t=_,ve===n&&n!==null&&(ve=n=n.return);continue}break}while(!0)}function $p(){var e=ul.current;return ul.current=sl,e===null?sl:e}function hs(){(ke===0||ke===3||ke===2)&&(ke=4),be===null||!(wn&268435455)&&!(El&268435455)||Vt(be,Ce)}function pl(e,t){var n=Z;Z|=2;var r=$p();(be!==e||Ce!==t)&&(_t=null,gn(e,t));do try{tg();break}catch(i){Up(e,i)}while(!0);if(Xa(),Z=n,ul.current=r,ve!==null)throw Error(D(261));return be=null,Ce=0,ke}function tg(){for(;ve!==null;)Hp(ve)}function ng(){for(;ve!==null&&!Nh();)Hp(ve)}function Hp(e){var t=Qp(e.alternate,e,qe);e.memoizedProps=e.pendingProps,t===null?Vp(e):ve=t,us.current=null}function Vp(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=Ym(n,t),n!==null){n.flags&=32767,ve=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{ke=6,ve=null;return}}else if(n=qm(n,t,qe),n!==null){ve=n;return}if(t=t.sibling,t!==null){ve=t;return}ve=t=e}while(t!==null);ke===0&&(ke=5)}function dn(e,t,n){var r=ne,i=ot.transition;try{ot.transition=null,ne=1,rg(e,t,n,r)}finally{ot.transition=i,ne=r}return null}function rg(e,t,n,r){do Kn();while(Qt!==null);if(Z&6)throw Error(D(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(D(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Rh(e,l),e===be&&(ve=be=null,Ce=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||bi||(bi=!0,Kp(Ki,function(){return Kn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=ot.transition,ot.transition=null;var o=ne;ne=1;var a=Z;Z|=4,us.current=null,Gm(e,n),Op(n,e),bm(Vo),Yi=!!Ho,Vo=Ho=null,e.current=n,Jm(n),_h(),Z=a,ne=o,ot.transition=l}else e.current=n;if(bi&&(bi=!1,Qt=e,dl=i),l=e.pendingLanes,l===0&&(Jt=null),Lh(n.stateNode),He(e,he()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(cl)throw cl=!1,e=ca,ca=null,e;return dl&1&&e.tag!==0&&Kn(),l=e.pendingLanes,l&1?e===da?Pr++:(Pr=0,da=e):Pr=0,on(),null}function Kn(){if(Qt!==null){var e=jd(dl),t=ot.transition,n=ne;try{if(ot.transition=null,ne=16>e?16:e,Qt===null)var r=!1;else{if(e=Qt,Qt=null,dl=0,Z&6)throw Error(D(331));var i=Z;for(Z|=4,U=e.current;U!==null;){var l=U,o=l.child;if(U.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var c=a[s];for(U=c;U!==null;){var d=U;switch(d.tag){case 0:case 11:case 15:Tr(8,d,l)}var p=d.child;if(p!==null)p.return=d,U=p;else for(;U!==null;){d=U;var m=d.sibling,f=d.return;if(Ap(d),d===c){U=null;break}if(m!==null){m.return=f,U=m;break}U=f}}}var k=l.alternate;if(k!==null){var w=k.child;if(w!==null){k.child=null;do{var M=w.sibling;w.sibling=null,w=M}while(w!==null)}}U=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,U=o;else e:for(;U!==null;){if(l=U,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Tr(9,l,l.return)}var h=l.sibling;if(h!==null){h.return=l.return,U=h;break e}U=l.return}}var v=e.current;for(U=v;U!==null;){o=U;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,U=y;else e:for(o=v;U!==null;){if(a=U,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Cl(9,a)}}catch(_){fe(a,a.return,_)}if(a===o){U=null;break e}var b=a.sibling;if(b!==null){b.return=a.return,U=b;break e}U=a.return}}if(Z=i,on(),St&&typeof St.onPostCommitFiberRoot=="function")try{St.onPostCommitFiberRoot(vl,e)}catch{}r=!0}return r}finally{ne=n,ot.transition=t}}return!1}function Wu(e,t,n){t=tr(n,t),t=jp(e,t,1),e=Gt(e,t,1),t=Ae(),e!==null&&(ti(e,1,t),He(e,t))}function fe(e,t,n){if(e.tag===3)Wu(e,e,n);else for(;t!==null;){if(t.tag===3){Wu(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(Jt===null||!Jt.has(r))){e=tr(n,e),e=Cp(t,e,1),t=Gt(t,e,1),e=Ae(),t!==null&&(ti(t,1,e),He(t,e));break}}t=t.return}}function ig(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Ae(),e.pingedLanes|=e.suspendedLanes&n,be===e&&(Ce&n)===n&&(ke===4||ke===3&&(Ce&130023424)===Ce&&500>he()-ds?gn(e,0):cs|=n),He(e,t)}function Wp(e,t){t===0&&(e.mode&1?(t=fi,fi<<=1,!(fi&130023424)&&(fi=4194304)):t=1);var n=Ae();e=Dt(e,t),e!==null&&(ti(e,t,n),He(e,n))}function lg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),Wp(e,n)}function og(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(D(314))}r!==null&&r.delete(t),Wp(e,n)}var Qp;Qp=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Ue.current)Be=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return Be=!1,Km(e,t,n);Be=!!(e.flags&131072)}else Be=!1,ue&&t.flags&1048576&&Xd(t,nl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Ri(e,t),e=t.pendingProps;var i=Gn(t,Le.current);Qn(t,n),i=is(null,t,r,e,i,n);var l=ls();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,$e(r)?(l=!0,el(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,Za(t),i.updater=jl,t.stateNode=i,i._reactInternals=t,Zo(t,r,e,n),t=na(null,t,r,!0,l,n)):(t.tag=0,ue&&l&&Qa(t),Me(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Ri(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=sg(r),e=dt(r,e),i){case 0:t=ta(null,t,r,e,n);break e;case 1:t=Au(null,t,r,e,n);break e;case 11:t=Iu(null,t,r,e,n);break e;case 14:t=Mu(null,t,r,dt(r.type,e),n);break e}throw Error(D(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:dt(r,i),ta(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:dt(r,i),Au(e,t,r,i,n);case 3:e:{if(zp(t),e===null)throw Error(D(387));r=t.pendingProps,l=t.memoizedState,i=l.element,np(e,t),ll(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=tr(Error(D(423)),t),t=Du(e,t,r,n,i);break e}else if(r!==i){i=tr(Error(D(424)),t),t=Du(e,t,r,n,i);break e}else for(Xe=Xt(t.stateNode.containerInfo.firstChild),Je=t,ue=!0,ft=null,n=ep(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(Jn(),r===i){t=Rt(e,t,n);break e}Me(e,t,r,n)}t=t.child}return t;case 5:return rp(t),e===null&&Xo(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,Wo(r,i)?o=null:l!==null&&Wo(r,l)&&(t.flags|=32),_p(e,t),Me(e,t,o,n),t.child;case 6:return e===null&&Xo(t),null;case 13:return Tp(e,t,n);case 4:return es(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=Zn(t,null,r,n):Me(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:dt(r,i),Iu(e,t,r,i,n);case 7:return Me(e,t,t.pendingProps,n),t.child;case 8:return Me(e,t,t.pendingProps.children,n),t.child;case 12:return Me(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,le(rl,r._currentValue),r._currentValue=o,l!==null)if(gt(l.value,o)){if(l.children===i.children&&!Ue.current){t=Rt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=It(-1,n&-n),s.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?s.next=s:(s.next=d.next,d.next=s),c.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),Go(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(D(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),Go(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}Me(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Qn(t,n),i=at(i),r=r(i),t.flags|=1,Me(e,t,r,n),t.child;case 14:return r=t.type,i=dt(r,t.pendingProps),i=dt(r.type,i),Mu(e,t,r,i,n);case 15:return Ep(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:dt(r,i),Ri(e,t),t.tag=1,$e(r)?(e=!0,el(t)):e=!1,Qn(t,n),bp(t,r,i),Zo(t,r,i,n),na(null,t,r,!0,e,n);case 19:return Lp(e,t,n);case 22:return Np(e,t,n)}throw Error(D(156,t.tag))};function Kp(e,t){return kd(e,t)}function ag(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function lt(e,t,n,r){return new ag(e,t,n,r)}function ms(e){return e=e.prototype,!(!e||!e.isReactComponent)}function sg(e){if(typeof e=="function")return ms(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Ma)return 11;if(e===Aa)return 14}return 2}function en(e,t){var n=e.alternate;return n===null?(n=lt(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Bi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")ms(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Ln:return vn(n.children,i,l,t);case Ia:o=8,i|=8;break;case bo:return e=lt(12,n,t,i|2),e.elementType=bo,e.lanes=l,e;case jo:return e=lt(13,n,t,i),e.elementType=jo,e.lanes=l,e;case Co:return e=lt(19,n,t,i),e.elementType=Co,e.lanes=l,e;case rd:return Nl(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case td:o=10;break e;case nd:o=9;break e;case Ma:o=11;break e;case Aa:o=14;break e;case Ut:o=16,r=null;break e}throw Error(D(130,e==null?e:typeof e,""))}return t=lt(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function vn(e,t,n,r){return e=lt(7,e,r,t),e.lanes=n,e}function Nl(e,t,n,r){return e=lt(22,e,r,t),e.elementType=rd,e.lanes=n,e.stateNode={isHidden:!1},e}function oo(e,t,n){return e=lt(6,e,null,t),e.lanes=n,e}function ao(e,t,n){return t=lt(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function ug(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Ul(0),this.expirationTimes=Ul(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Ul(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function gs(e,t,n,r,i,l,o,a,s){return e=new ug(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=lt(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},Za(l),e}function cg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:Tn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function qp(e){if(!e)return nn;e=e._reactInternals;e:{if(jn(e)!==e||e.tag!==1)throw Error(D(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if($e(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(D(171))}if(e.tag===1){var n=e.type;if($e(n))return qd(e,n,t)}return t}function Yp(e,t,n,r,i,l,o,a,s){return e=gs(n,r,!0,e,i,l,o,a,s),e.context=qp(null),n=e.current,r=Ae(),i=Zt(n),l=It(r,i),l.callback=t??null,Gt(n,l,i),e.current.lanes=i,ti(e,i,r),He(e,r),e}function _l(e,t,n,r){var i=t.current,l=Ae(),o=Zt(i);return n=qp(n),t.context===null?t.context=n:t.pendingContext=n,t=It(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=Gt(i,t,o),e!==null&&(mt(e,i,o,l),Mi(e,i,o)),o}function fl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function Qu(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function vs(e,t){Qu(e,t),(e=e.alternate)&&Qu(e,t)}function dg(){return null}var Xp=typeof reportError=="function"?reportError:function(e){console.error(e)};function ys(e){this._internalRoot=e}zl.prototype.render=ys.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(D(409));_l(e,t,null,null)};zl.prototype.unmount=ys.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;Sn(function(){_l(null,e,null,null)}),t[At]=null}};function zl(e){this._internalRoot=e}zl.prototype.unstable_scheduleHydration=function(e){if(e){var t=Nd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Ht.length&&t!==0&&t<Ht[n].priority;n++);Ht.splice(n,0,e),n===0&&zd(e)}};function xs(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Tl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function Ku(){}function pg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=fl(o);l.call(c)}}var o=Yp(t,r,e,0,null,!1,!1,"",Ku);return e._reactRootContainer=o,e[At]=o.current,Vr(e.nodeType===8?e.parentNode:e),Sn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=fl(s);a.call(c)}}var s=gs(e,0,!1,null,null,!1,!1,"",Ku);return e._reactRootContainer=s,e[At]=s.current,Vr(e.nodeType===8?e.parentNode:e),Sn(function(){_l(t,s,n,r)}),s}function Ll(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=fl(o);a.call(s)}}_l(t,o,e,i)}else o=pg(n,t,e,i,r);return fl(o)}Cd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=Sr(t.pendingLanes);n!==0&&(Oa(t,n|1),He(t,he()),!(Z&6)&&(nr=he()+500,on()))}break;case 13:Sn(function(){var r=Dt(e,1);if(r!==null){var i=Ae();mt(r,e,1,i)}}),vs(e,1)}};Fa=function(e){if(e.tag===13){var t=Dt(e,134217728);if(t!==null){var n=Ae();mt(t,e,134217728,n)}vs(e,134217728)}};Ed=function(e){if(e.tag===13){var t=Zt(e),n=Dt(e,t);if(n!==null){var r=Ae();mt(n,e,t,r)}vs(e,t)}};Nd=function(){return ne};_d=function(e,t){var n=ne;try{return ne=e,t()}finally{ne=n}};Ao=function(e,t,n){switch(t){case"input":if(_o(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=wl(r);if(!i)throw Error(D(90));ld(r),_o(r,i)}}}break;case"textarea":ad(e,n);break;case"select":t=n.value,t!=null&&$n(e,!!n.multiple,t,!1)}};hd=ps;md=Sn;var fg={usingClientEntryPoint:!1,Events:[ri,An,wl,pd,fd,ps]},vr={findFiberByHostInstance:fn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},hg={bundleType:vr.bundleType,version:vr.version,rendererPackageName:vr.rendererPackageName,rendererConfig:vr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Ot.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=yd(e),e===null?null:e.stateNode},findFiberByHostInstance:vr.findFiberByHostInstance||dg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var ji=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!ji.isDisabled&&ji.supportsFiber)try{vl=ji.inject(hg),St=ji}catch{}}et.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=fg;et.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!xs(t))throw Error(D(200));return cg(e,t,null,n)};et.createRoot=function(e,t){if(!xs(e))throw Error(D(299));var n=!1,r="",i=Xp;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=gs(e,1,!1,null,null,n,!1,r,i),e[At]=t.current,Vr(e.nodeType===8?e.parentNode:e),new ys(t)};et.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(D(188)):(e=Object.keys(e).join(","),Error(D(268,e)));return e=yd(t),e=e===null?null:e.stateNode,e};et.flushSync=function(e){return Sn(e)};et.hydrate=function(e,t,n){if(!Tl(t))throw Error(D(200));return Ll(null,e,t,!0,n)};et.hydrateRoot=function(e,t,n){if(!xs(e))throw Error(D(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=Xp;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=Yp(t,null,e,1,n??null,i,!1,l,o),e[At]=t.current,Vr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new zl(t)};et.render=function(e,t,n){if(!Tl(t))throw Error(D(200));return Ll(null,e,t,!1,n)};et.unmountComponentAtNode=function(e){if(!Tl(e))throw Error(D(40));return e._reactRootContainer?(Sn(function(){Ll(null,null,e,!1,function(){e._reactRootContainer=null,e[At]=null})}),!0):!1};et.unstable_batchedUpdates=ps;et.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Tl(n))throw Error(D(200));if(e==null||e._reactInternals===void 0)throw Error(D(38));return Ll(e,t,n,!1,r)};et.version="18.3.1-next-f1338f8080-20240426";function Gp(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Gp)}catch(e){console.error(e)}}Gp(),Gc.exports=et;var mg=Gc.exports,qu=mg;wo.createRoot=qu.createRoot,wo.hydrateRoot=qu.hydrateRoot;const Nt={plus:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),u.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"}),u.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),u.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),u.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),u.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),u.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),u.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("polyline",{points:"3 6 5 6 21 6"}),u.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:u.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},gg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=B.useState(!1),[c,d]=B.useState(""),[p,m]=B.useState(null),[f,k]=B.useState(""),[w,M]=B.useState(null),h=()=>{c.trim()&&(r(c.trim()),d(""),s(!1))},v=j=>{j.key==="Enter"&&!j.shiftKey?(j.preventDefault(),h()):j.key==="Escape"&&(s(!1),d(""))},y=(j,I)=>{I.stopPropagation(),m(j.id),k(j.title)},b=j=>{var I;f.trim()&&f.trim()!==((I=e.find(H=>H.id===j))==null?void 0:I.title)&&l(j,f.trim()),m(null),k("")},_=()=>{m(null),k("")},S=(j,I)=>{j.key==="Enter"?(j.preventDefault(),b(I)):j.key==="Escape"&&_()},L=(j,I)=>{I.stopPropagation(),M(j)},C=(j,I)=>{I.stopPropagation(),i(j),M(null)},T=j=>{j.stopPropagation(),M(null)},O=j=>{const I=new Date(j),Q=new Date().getTime()-I.getTime(),$=Math.floor(Q/6e4),N=Math.floor(Q/36e5),W=Math.floor(Q/864e5);return $<1?"now":$<60?`${$}m`:N<24?`${N}h`:W<7?`${W}d`:I.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return u.jsxs("div",{className:"thread-list",children:[u.jsxs("div",{className:"list-header",children:[u.jsx("h2",{children:"Conversations"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:Nt.plus})]}),a&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:c,onChange:j=>d(j.target.value),onKeyDown:v,placeholder:"Conversation title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:h,children:"Create"})]})]}),u.jsx("div",{className:"thread-items",children:e.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Nt.hash}),u.jsx("p",{children:"No conversations yet"}),u.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(j=>{const I=o.get(j.id)||0,H=j.id===t,Q=p===j.id,$=w===j.id;return u.jsxs("div",{className:`thread-item ${H?"selected":""} ${I>0?"has-unread":""}`,onClick:()=>!Q&&n(j.id),children:[u.jsx("div",{className:`status-dot ${j.status}`}),u.jsxs("div",{className:"thread-content",children:[u.jsx("div",{className:"thread-title-row",children:Q?u.jsxs("div",{className:"edit-title-form",onClick:N=>N.stopPropagation(),children:[u.jsx("input",{type:"text",value:f,onChange:N=>k(N.target.value),onKeyDown:N=>S(N,j.id),autoFocus:!0}),u.jsx("button",{className:"edit-action save",onClick:()=>b(j.id),title:"Save",children:Nt.check}),u.jsx("button",{className:"edit-action cancel",onClick:_,title:"Cancel",children:Nt.x})]}):u.jsxs(u.Fragment,{children:[u.jsx("span",{className:"thread-title",children:j.title}),u.jsx("span",{className:"thread-time",children:O(j.updated_at)})]})}),u.jsxs("div",{className:"thread-meta",children:[j.target_agent&&u.jsxs("span",{className:"thread-agent",title:`Target: ${j.target_agent}`,children:[Nt.bot,j.target_agent]}),u.jsxs("span",{className:"thread-seq",children:["#",j.last_seq]})]})]}),!Q&&!$&&u.jsxs("div",{className:"thread-actions",children:[u.jsx("button",{className:"action-btn edit",onClick:N=>y(j,N),title:"Rename",children:Nt.edit}),u.jsx("button",{className:"action-btn delete",onClick:N=>L(j.id,N),title:"Delete",children:Nt.trash})]}),$&&u.jsxs("div",{className:"delete-confirm",onClick:N=>N.stopPropagation(),children:[u.jsx("span",{className:"confirm-text",children:"Delete?"}),u.jsx("button",{className:"confirm-btn yes",onClick:N=>C(j.id,N),title:"Confirm delete",children:Nt.check}),u.jsx("button",{className:"confirm-btn no",onClick:T,title:"Cancel",children:Nt.x})]}),I>0&&!$&&u.jsx("span",{className:"unread-badge",children:I})]},j.id)})}),u.jsx("style",{children:`
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
      `})]})};function vg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const yg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,xg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,kg={};function Yu(e,t){return(kg.jsx?xg:yg).test(e)}const wg=/[ \t\n\f\r]/g;function Sg(e){return typeof e=="object"?e.type==="text"?Xu(e.value):!1:Xu(e)}function Xu(e){return e.replace(wg,"")===""}class li{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}li.prototype.normal={};li.prototype.property={};li.prototype.space=void 0;function Jp(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new li(n,r,t)}function ha(e){return e.toLowerCase()}class We{constructor(t,n){this.attribute=n,this.property=t}}We.prototype.attribute="";We.prototype.booleanish=!1;We.prototype.boolean=!1;We.prototype.commaOrSpaceSeparated=!1;We.prototype.commaSeparated=!1;We.prototype.defined=!1;We.prototype.mustUseProperty=!1;We.prototype.number=!1;We.prototype.overloadedBoolean=!1;We.prototype.property="";We.prototype.spaceSeparated=!1;We.prototype.space=void 0;let bg=0;const X=Cn(),ge=Cn(),ma=Cn(),R=Cn(),ie=Cn(),qn=Cn(),Ke=Cn();function Cn(){return 2**++bg}const ga=Object.freeze(Object.defineProperty({__proto__:null,boolean:X,booleanish:ge,commaOrSpaceSeparated:Ke,commaSeparated:qn,number:R,overloadedBoolean:ma,spaceSeparated:ie},Symbol.toStringTag,{value:"Module"})),so=Object.keys(ga);class ks extends We{constructor(t,n,r,i){let l=-1;if(super(t,n),Gu(this,"space",i),typeof r=="number")for(;++l<so.length;){const o=so[l];Gu(this,so[l],(r&ga[o])===ga[o])}}}ks.prototype.defined=!0;function Gu(e,t,n){n&&(e[t]=n)}function or(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new ks(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[ha(r)]=r,n[ha(l.attribute)]=r}return new li(t,n,e.space)}const Zp=or({properties:{ariaActiveDescendant:null,ariaAtomic:ge,ariaAutoComplete:null,ariaBusy:ge,ariaChecked:ge,ariaColCount:R,ariaColIndex:R,ariaColSpan:R,ariaControls:ie,ariaCurrent:null,ariaDescribedBy:ie,ariaDetails:null,ariaDisabled:ge,ariaDropEffect:ie,ariaErrorMessage:null,ariaExpanded:ge,ariaFlowTo:ie,ariaGrabbed:ge,ariaHasPopup:null,ariaHidden:ge,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:ie,ariaLevel:R,ariaLive:null,ariaModal:ge,ariaMultiLine:ge,ariaMultiSelectable:ge,ariaOrientation:null,ariaOwns:ie,ariaPlaceholder:null,ariaPosInSet:R,ariaPressed:ge,ariaReadOnly:ge,ariaRelevant:null,ariaRequired:ge,ariaRoleDescription:ie,ariaRowCount:R,ariaRowIndex:R,ariaRowSpan:R,ariaSelected:ge,ariaSetSize:R,ariaSort:null,ariaValueMax:R,ariaValueMin:R,ariaValueNow:R,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function ef(e,t){return t in e?e[t]:t}function tf(e,t){return ef(e,t.toLowerCase())}const jg=or({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:qn,acceptCharset:ie,accessKey:ie,action:null,allow:null,allowFullScreen:X,allowPaymentRequest:X,allowUserMedia:X,alt:null,as:null,async:X,autoCapitalize:null,autoComplete:ie,autoFocus:X,autoPlay:X,blocking:ie,capture:null,charSet:null,checked:X,cite:null,className:ie,cols:R,colSpan:null,content:null,contentEditable:ge,controls:X,controlsList:ie,coords:R|qn,crossOrigin:null,data:null,dateTime:null,decoding:null,default:X,defer:X,dir:null,dirName:null,disabled:X,download:ma,draggable:ge,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:X,formTarget:null,headers:ie,height:R,hidden:ma,high:R,href:null,hrefLang:null,htmlFor:ie,httpEquiv:ie,id:null,imageSizes:null,imageSrcSet:null,inert:X,inputMode:null,integrity:null,is:null,isMap:X,itemId:null,itemProp:ie,itemRef:ie,itemScope:X,itemType:ie,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:X,low:R,manifest:null,max:null,maxLength:R,media:null,method:null,min:null,minLength:R,multiple:X,muted:X,name:null,nonce:null,noModule:X,noValidate:X,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:X,optimum:R,pattern:null,ping:ie,placeholder:null,playsInline:X,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:X,referrerPolicy:null,rel:ie,required:X,reversed:X,rows:R,rowSpan:R,sandbox:ie,scope:null,scoped:X,seamless:X,selected:X,shadowRootClonable:X,shadowRootDelegatesFocus:X,shadowRootMode:null,shape:null,size:R,sizes:null,slot:null,span:R,spellCheck:ge,src:null,srcDoc:null,srcLang:null,srcSet:null,start:R,step:null,style:null,tabIndex:R,target:null,title:null,translate:null,type:null,typeMustMatch:X,useMap:null,value:ge,width:R,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:ie,axis:null,background:null,bgColor:null,border:R,borderColor:null,bottomMargin:R,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:X,declare:X,event:null,face:null,frame:null,frameBorder:null,hSpace:R,leftMargin:R,link:null,longDesc:null,lowSrc:null,marginHeight:R,marginWidth:R,noResize:X,noHref:X,noShade:X,noWrap:X,object:null,profile:null,prompt:null,rev:null,rightMargin:R,rules:null,scheme:null,scrolling:ge,standby:null,summary:null,text:null,topMargin:R,valueType:null,version:null,vAlign:null,vLink:null,vSpace:R,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:X,disableRemotePlayback:X,prefix:null,property:null,results:R,security:null,unselectable:null},space:"html",transform:tf}),Cg=or({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:Ke,accentHeight:R,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:R,amplitude:R,arabicForm:null,ascent:R,attributeName:null,attributeType:null,azimuth:R,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:R,by:null,calcMode:null,capHeight:R,className:ie,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:R,diffuseConstant:R,direction:null,display:null,dur:null,divisor:R,dominantBaseline:null,download:X,dx:null,dy:null,edgeMode:null,editable:null,elevation:R,enableBackground:null,end:null,event:null,exponent:R,externalResourcesRequired:null,fill:null,fillOpacity:R,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:qn,g2:qn,glyphName:qn,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:R,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:R,horizOriginX:R,horizOriginY:R,id:null,ideographic:R,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:R,k:R,k1:R,k2:R,k3:R,k4:R,kernelMatrix:Ke,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:R,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:R,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:R,overlineThickness:R,paintOrder:null,panose1:null,path:null,pathLength:R,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:ie,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:R,pointsAtY:R,pointsAtZ:R,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:Ke,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:Ke,rev:Ke,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:Ke,requiredFeatures:Ke,requiredFonts:Ke,requiredFormats:Ke,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:R,specularExponent:R,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:R,strikethroughThickness:R,string:null,stroke:null,strokeDashArray:Ke,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:R,strokeOpacity:R,strokeWidth:null,style:null,surfaceScale:R,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:Ke,tabIndex:R,tableValues:null,target:null,targetX:R,targetY:R,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:Ke,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:R,underlineThickness:R,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:R,values:null,vAlphabetic:R,vMathematical:R,vectorEffect:null,vHanging:R,vIdeographic:R,version:null,vertAdvY:R,vertOriginX:R,vertOriginY:R,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:R,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:ef}),nf=or({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),rf=or({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:tf}),lf=or({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),Eg={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Ng=/[A-Z]/g,Ju=/-[a-z]/g,_g=/^data[-\w.:]+$/i;function zg(e,t){const n=ha(t);let r=t,i=We;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&_g.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(Ju,Lg);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!Ju.test(l)){let o=l.replace(Ng,Tg);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=ks}return new i(r,t)}function Tg(e){return"-"+e.toLowerCase()}function Lg(e){return e.charAt(1).toUpperCase()}const Pg=Jp([Zp,jg,nf,rf,lf],"html"),ws=Jp([Zp,Cg,nf,rf,lf],"svg");function Ig(e){return e.join(" ").trim()}var Ss={},Zu=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Mg=/\n/g,Ag=/^\s*/,Dg=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Rg=/^:\s*/,Og=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Fg=/^[;\s]*/,Bg=/^\s+|\s+$/g,Ug=`
`,ec="/",tc="*",pn="",$g="comment",Hg="declaration";function Vg(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(k){var w=k.match(Mg);w&&(n+=w.length);var M=k.lastIndexOf(Ug);r=~M?k.length-M:r+k.length}function l(){var k={line:n,column:r};return function(w){return w.position=new o(k),c(),w}}function o(k){this.start=k,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(k){var w=new Error(t.source+":"+n+":"+r+": "+k);if(w.reason=k,w.filename=t.source,w.line=n,w.column=r,w.source=e,!t.silent)throw w}function s(k){var w=k.exec(e);if(w){var M=w[0];return i(M),e=e.slice(M.length),w}}function c(){s(Ag)}function d(k){var w;for(k=k||[];w=p();)w!==!1&&k.push(w);return k}function p(){var k=l();if(!(ec!=e.charAt(0)||tc!=e.charAt(1))){for(var w=2;pn!=e.charAt(w)&&(tc!=e.charAt(w)||ec!=e.charAt(w+1));)++w;if(w+=2,pn===e.charAt(w-1))return a("End of comment missing");var M=e.slice(2,w-2);return r+=2,i(M),e=e.slice(w),r+=2,k({type:$g,comment:M})}}function m(){var k=l(),w=s(Dg);if(w){if(p(),!s(Rg))return a("property missing ':'");var M=s(Og),h=k({type:Hg,property:nc(w[0].replace(Zu,pn)),value:M?nc(M[0].replace(Zu,pn)):pn});return s(Fg),h}}function f(){var k=[];d(k);for(var w;w=m();)w!==!1&&(k.push(w),d(k));return k}return c(),f()}function nc(e){return e?e.replace(Bg,pn):pn}var Wg=Vg,Qg=Hi&&Hi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(Ss,"__esModule",{value:!0});Ss.default=qg;const Kg=Qg(Wg);function qg(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Kg.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Pl={};Object.defineProperty(Pl,"__esModule",{value:!0});Pl.camelCase=void 0;var Yg=/^--[a-zA-Z0-9_-]+$/,Xg=/-([a-z])/g,Gg=/^[^-]+$/,Jg=/^-(webkit|moz|ms|o|khtml)-/,Zg=/^-(ms)-/,ev=function(e){return!e||Gg.test(e)||Yg.test(e)},tv=function(e,t){return t.toUpperCase()},rc=function(e,t){return"".concat(t,"-")},nv=function(e,t){return t===void 0&&(t={}),ev(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(Zg,rc):e=e.replace(Jg,rc),e.replace(Xg,tv))};Pl.camelCase=nv;var rv=Hi&&Hi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},iv=rv(Ss),lv=Pl;function va(e,t){var n={};return!e||typeof e!="string"||(0,iv.default)(e,function(r,i){r&&i&&(n[(0,lv.camelCase)(r,t)]=i)}),n}va.default=va;var ov=va;const av=Ca(ov),of=af("end"),bs=af("start");function af(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function sv(e){const t=bs(e),n=of(e);if(t&&n)return{start:t,end:n}}function Ir(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?ic(e.position):"start"in e||"end"in e?ic(e):"line"in e||"column"in e?ya(e):""}function ya(e){return lc(e&&e.line)+":"+lc(e&&e.column)}function ic(e){return ya(e&&e.start)+"-"+ya(e&&e.end)}function lc(e){return e&&typeof e=="number"?e:1}class Pe extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Ir(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}Pe.prototype.file="";Pe.prototype.name="";Pe.prototype.reason="";Pe.prototype.message="";Pe.prototype.stack="";Pe.prototype.column=void 0;Pe.prototype.line=void 0;Pe.prototype.ancestors=void 0;Pe.prototype.cause=void 0;Pe.prototype.fatal=void 0;Pe.prototype.place=void 0;Pe.prototype.ruleId=void 0;Pe.prototype.source=void 0;const js={}.hasOwnProperty,uv=new Map,cv=/[A-Z]/g,dv=new Set(["table","tbody","thead","tfoot","tr"]),pv=new Set(["td","th"]),sf="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function fv(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=wv(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=kv(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?ws:Pg,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=uf(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function uf(e,t,n){if(t.type==="element")return hv(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return mv(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return vv(e,t,n);if(t.type==="mdxjsEsm")return gv(e,t);if(t.type==="root")return yv(e,t,n);if(t.type==="text")return xv(e,t)}function hv(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=ws,e.schema=i),e.ancestors.push(t);const l=df(e,t.tagName,!1),o=Sv(e,t);let a=Es(e,t);return dv.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!Sg(s):!0})),cf(e,o,l,t),Cs(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function mv(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}Zr(e,t.position)}function gv(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);Zr(e,t.position)}function vv(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=ws,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:df(e,t.name,!0),o=bv(e,t),a=Es(e,t);return cf(e,o,l,t),Cs(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function yv(e,t,n){const r={};return Cs(r,Es(e,t)),e.create(t,e.Fragment,r,n)}function xv(e,t){return t.value}function cf(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Cs(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function kv(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function wv(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=bs(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function Sv(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&js.call(t.properties,i)){const l=jv(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&pv.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function bv(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else Zr(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else Zr(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Es(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:uv;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const c=i.get(s)||0;o=s+"-"+c,i.set(s,c+1)}}const a=uf(e,l,o);a!==void 0&&n.push(a)}return n}function jv(e,t,n){const r=zg(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?vg(n):Ig(n)),r.property==="style"){let i=typeof n=="object"?n:Cv(e,String(n));return e.stylePropertyNameCase==="css"&&(i=Ev(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?Eg[r.property]||r.property:r.attribute,n]}}function Cv(e,t){try{return av(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new Pe("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=sf+"#cannot-parse-style-attribute",i}}function df(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=Yu(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=Yu(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return js.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);Zr(e)}function Zr(e,t){const n=new Pe("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=sf+"#cannot-handle-mdx-estrees-without-createevaluater",n}function Ev(e){const t={};let n;for(n in e)js.call(e,n)&&(t[Nv(n)]=e[n]);return t}function Nv(e){let t=e.replace(cv,_v);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function _v(e){return"-"+e.toLowerCase()}const uo={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},zv={};function Tv(e,t){const n=zv,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return pf(e,r,i)}function pf(e,t,n){if(Lv(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return oc(e.children,t,n)}return Array.isArray(e)?oc(e,t,n):""}function oc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=pf(e[i],t,n);return r.join("")}function Lv(e){return!!(e&&typeof e=="object")}const ac=document.createElement("i");function Ns(e){const t="&"+e+";";ac.innerHTML=t;const n=ac.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function jt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function it(e,t){return e.length>0?(jt(e,e.length,0,t),e):t}const sc={}.hasOwnProperty;function Pv(e){const t={};let n=-1;for(;++n<e.length;)Iv(t,e[n]);return t}function Iv(e,t){let n;for(n in t){const i=(sc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){sc.call(i,o)||(i[o]=[]);const a=l[o];Mv(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Mv(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);jt(e,0,0,r)}function ff(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function Yn(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const wt=an(/[A-Za-z]/),Ge=an(/[\dA-Za-z]/),Av=an(/[#-'*+\--9=?A-Z^-~]/);function xa(e){return e!==null&&(e<32||e===127)}const ka=an(/\d/),Dv=an(/[\dA-Fa-f]/),Rv=an(/[!-/:-@[-`{-~]/);function K(e){return e!==null&&e<-2}function Ve(e){return e!==null&&(e<0||e===32)}function ee(e){return e===-2||e===-1||e===32}const Ov=an(new RegExp("\\p{P}|\\p{S}","u")),Fv=an(/\s/);function an(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function ar(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&Ge(e.charCodeAt(n+1))&&Ge(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function oe(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return ee(s)?(e.enter(n),a(s)):t(s)}function a(s){return ee(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const Bv={tokenize:Uv};function Uv(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),oe(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return K(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const $v={tokenize:Hv},uc={tokenize:Vv};function Hv(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const b=n[r];return t.containerState=b[1],e.attempt(b[0].continuation,s,c)(y)}return c(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&v();const b=t.events.length;let _=b,S;for(;_--;)if(t.events[_][0]==="exit"&&t.events[_][1].type==="chunkFlow"){S=t.events[_][1].end;break}h(r);let L=b;for(;L<t.events.length;)t.events[L][1].end={...S},L++;return jt(t.events,_+1,0,t.events.slice(b)),t.events.length=L,c(y)}return a(y)}function c(y){if(r===n.length){if(!i)return m(y);if(i.currentConstruct&&i.currentConstruct.concrete)return k(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(uc,d,p)(y)}function d(y){return i&&v(),h(r),m(y)}function p(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,k(y)}function m(y){return t.containerState={},e.attempt(uc,f,k)(y)}function f(y){return r++,n.push([t.currentConstruct,t.containerState]),m(y)}function k(y){if(y===null){i&&v(),h(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),w(y)}function w(y){if(y===null){M(e.exit("chunkFlow"),!0),h(0),e.consume(y);return}return K(y)?(e.consume(y),M(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),w)}function M(y,b){const _=t.sliceStream(y);if(b&&_.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(_),t.parser.lazy[y.start.line]){let S=i.events.length;for(;S--;)if(i.events[S][1].start.offset<o&&(!i.events[S][1].end||i.events[S][1].end.offset>o))return;const L=t.events.length;let C=L,T,O;for(;C--;)if(t.events[C][0]==="exit"&&t.events[C][1].type==="chunkFlow"){if(T){O=t.events[C][1].end;break}T=!0}for(h(r),S=L;S<t.events.length;)t.events[S][1].end={...O},S++;jt(t.events,C+1,0,t.events.slice(L)),t.events.length=S}}function h(y){let b=n.length;for(;b-- >y;){const _=n[b];t.containerState=_[1],_[0].exit.call(t,e)}n.length=y}function v(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Vv(e,t,n){return oe(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function cc(e){if(e===null||Ve(e)||Fv(e))return 1;if(Ov(e))return 2}function _s(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const wa={name:"attention",resolveAll:Wv,tokenize:Qv};function Wv(e,t){let n=-1,r,i,l,o,a,s,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const p={...e[r][1].end},m={...e[n][1].start};dc(p,-s),dc(m,s),o={type:s>1?"strongSequence":"emphasisSequence",start:p,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:m},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=it(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=it(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=it(c,_s(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=it(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=it(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,jt(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Qv(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=cc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const c=e.exit("attentionSequence"),d=cc(s),p=!d||d===2&&i||n.includes(s),m=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?p:p&&(i||!m)),c._close=!!(l===42?m:m&&(d||!p)),t(s)}}function dc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Kv={name:"autolink",tokenize:qv};function qv(e,t,n){let r=0;return i;function i(f){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(f){return wt(f)?(e.consume(f),o):f===64?n(f):c(f)}function o(f){return f===43||f===45||f===46||Ge(f)?(r=1,a(f)):c(f)}function a(f){return f===58?(e.consume(f),r=0,s):(f===43||f===45||f===46||Ge(f))&&r++<32?(e.consume(f),a):(r=0,c(f))}function s(f){return f===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.exit("autolink"),t):f===null||f===32||f===60||xa(f)?n(f):(e.consume(f),s)}function c(f){return f===64?(e.consume(f),d):Av(f)?(e.consume(f),c):n(f)}function d(f){return Ge(f)?p(f):n(f)}function p(f){return f===46?(e.consume(f),r=0,d):f===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(f),e.exit("autolinkMarker"),e.exit("autolink"),t):m(f)}function m(f){if((f===45||Ge(f))&&r++<63){const k=f===45?m:p;return e.consume(f),k}return n(f)}}const Il={partial:!0,tokenize:Yv};function Yv(e,t,n){return r;function r(l){return ee(l)?oe(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||K(l)?t(l):n(l)}}const hf={continuation:{tokenize:Gv},exit:Jv,name:"blockQuote",tokenize:Xv};function Xv(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return ee(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function Gv(e,t,n){const r=this;return i;function i(o){return ee(o)?oe(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(hf,t,n)(o)}}function Jv(e){e.exit("blockQuote")}const mf={name:"characterEscape",tokenize:Zv};function Zv(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return Rv(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const gf={name:"characterReference",tokenize:ey};function ey(e,t,n){const r=this;let i=0,l,o;return a;function a(p){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(p),e.exit("characterReferenceMarker"),s}function s(p){return p===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(p),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=Ge,d(p))}function c(p){return p===88||p===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(p),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=Dv,d):(e.enter("characterReferenceValue"),l=7,o=ka,d(p))}function d(p){if(p===59&&i){const m=e.exit("characterReferenceValue");return o===Ge&&!Ns(r.sliceSerialize(m))?n(p):(e.enter("characterReferenceMarker"),e.consume(p),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(p)&&i++<l?(e.consume(p),d):n(p)}}const pc={partial:!0,tokenize:ny},fc={concrete:!0,name:"codeFenced",tokenize:ty};function ty(e,t,n){const r=this,i={partial:!0,tokenize:_};let l=0,o=0,a;return s;function s(S){return c(S)}function c(S){const L=r.events[r.events.length-1];return l=L&&L[1].type==="linePrefix"?L[2].sliceSerialize(L[1],!0).length:0,a=S,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(S)}function d(S){return S===a?(o++,e.consume(S),d):o<3?n(S):(e.exit("codeFencedFenceSequence"),ee(S)?oe(e,p,"whitespace")(S):p(S))}function p(S){return S===null||K(S)?(e.exit("codeFencedFence"),r.interrupt?t(S):e.check(pc,w,b)(S)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),m(S))}function m(S){return S===null||K(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),p(S)):ee(S)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),oe(e,f,"whitespace")(S)):S===96&&S===a?n(S):(e.consume(S),m)}function f(S){return S===null||K(S)?p(S):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),k(S))}function k(S){return S===null||K(S)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),p(S)):S===96&&S===a?n(S):(e.consume(S),k)}function w(S){return e.attempt(i,b,M)(S)}function M(S){return e.enter("lineEnding"),e.consume(S),e.exit("lineEnding"),h}function h(S){return l>0&&ee(S)?oe(e,v,"linePrefix",l+1)(S):v(S)}function v(S){return S===null||K(S)?e.check(pc,w,b)(S):(e.enter("codeFlowValue"),y(S))}function y(S){return S===null||K(S)?(e.exit("codeFlowValue"),v(S)):(e.consume(S),y)}function b(S){return e.exit("codeFenced"),t(S)}function _(S,L,C){let T=0;return O;function O($){return S.enter("lineEnding"),S.consume($),S.exit("lineEnding"),j}function j($){return S.enter("codeFencedFence"),ee($)?oe(S,I,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)($):I($)}function I($){return $===a?(S.enter("codeFencedFenceSequence"),H($)):C($)}function H($){return $===a?(T++,S.consume($),H):T>=o?(S.exit("codeFencedFenceSequence"),ee($)?oe(S,Q,"whitespace")($):Q($)):C($)}function Q($){return $===null||K($)?(S.exit("codeFencedFence"),L($)):C($)}}}function ny(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const co={name:"codeIndented",tokenize:iy},ry={partial:!0,tokenize:ly};function iy(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),oe(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?s(c):K(c)?e.attempt(ry,o,s)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||K(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function s(c){return e.exit("codeIndented"),t(c)}}function ly(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):K(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):oe(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):K(o)?i(o):n(o)}}const oy={name:"codeText",previous:sy,resolve:ay,tokenize:uy};function ay(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function sy(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function uy(e,t,n){let r=0,i,l;return o;function o(p){return e.enter("codeText"),e.enter("codeTextSequence"),a(p)}function a(p){return p===96?(e.consume(p),r++,a):(e.exit("codeTextSequence"),s(p))}function s(p){return p===null?n(p):p===32?(e.enter("space"),e.consume(p),e.exit("space"),s):p===96?(l=e.enter("codeTextSequence"),i=0,d(p)):K(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),s):(e.enter("codeTextData"),c(p))}function c(p){return p===null||p===32||p===96||K(p)?(e.exit("codeTextData"),s(p)):(e.consume(p),c)}function d(p){return p===96?(e.consume(p),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(p)):(l.type="codeTextData",c(p))}}class cy{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&yr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),yr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),yr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);yr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);yr(this.left,n.reverse())}}}function yr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function vf(e){const t={};let n=-1,r,i,l,o,a,s,c;const d=new cy(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,dy(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return jt(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function dy(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],c={};let d,p,m=-1,f=n,k=0,w=0;const M=[w];for(;f;){for(;e.get(++i)[1]!==f;);l.push(i),f._tokenizer||(d=r.sliceStream(f),f.next||d.push(null),p&&o.defineSkip(f.start),f._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),f._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),p=f,f=f.next}for(f=n;++m<a.length;)a[m][0]==="exit"&&a[m-1][0]==="enter"&&a[m][1].type===a[m-1][1].type&&a[m][1].start.line!==a[m][1].end.line&&(w=m+1,M.push(w),f._tokenizer=void 0,f.previous=void 0,f=f.next);for(o.events=[],f?(f._tokenizer=void 0,f.previous=void 0):M.pop(),m=M.length;m--;){const h=a.slice(M[m],M[m+1]),v=l.pop();s.push([v,v+h.length-1]),e.splice(v,2,h)}for(s.reverse(),m=-1;++m<s.length;)c[k+s[m][0]]=k+s[m][1],k+=s[m][1]-s[m][0]-1;return c}const py={resolve:hy,tokenize:my},fy={partial:!0,tokenize:gy};function hy(e){return vf(e),e}function my(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):K(a)?e.check(fy,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function gy(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),oe(e,l,"linePrefix")}function l(o){if(o===null||K(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function yf(e,t,n,r,i,l,o,a,s){const c=s||Number.POSITIVE_INFINITY;let d=0;return p;function p(h){return h===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(h),e.exit(l),m):h===null||h===32||h===41||xa(h)?n(h):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),w(h))}function m(h){return h===62?(e.enter(l),e.consume(h),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),f(h))}function f(h){return h===62?(e.exit("chunkString"),e.exit(a),m(h)):h===null||h===60||K(h)?n(h):(e.consume(h),h===92?k:f)}function k(h){return h===60||h===62||h===92?(e.consume(h),f):f(h)}function w(h){return!d&&(h===null||h===41||Ve(h))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(h)):d<c&&h===40?(e.consume(h),d++,w):h===41?(e.consume(h),d--,w):h===null||h===32||h===40||xa(h)?n(h):(e.consume(h),h===92?M:w)}function M(h){return h===40||h===41||h===92?(e.consume(h),w):w(h)}}function xf(e,t,n,r,i,l){const o=this;let a=0,s;return c;function c(f){return e.enter(r),e.enter(i),e.consume(f),e.exit(i),e.enter(l),d}function d(f){return a>999||f===null||f===91||f===93&&!s||f===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(f):f===93?(e.exit(l),e.enter(i),e.consume(f),e.exit(i),e.exit(r),t):K(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),p(f))}function p(f){return f===null||f===91||f===93||K(f)||a++>999?(e.exit("chunkString"),d(f)):(e.consume(f),s||(s=!ee(f)),f===92?m:p)}function m(f){return f===91||f===92||f===93?(e.consume(f),a++,p):p(f)}}function kf(e,t,n,r,i,l){let o;return a;function a(m){return m===34||m===39||m===40?(e.enter(r),e.enter(i),e.consume(m),e.exit(i),o=m===40?41:m,s):n(m)}function s(m){return m===o?(e.enter(i),e.consume(m),e.exit(i),e.exit(r),t):(e.enter(l),c(m))}function c(m){return m===o?(e.exit(l),s(o)):m===null?n(m):K(m)?(e.enter("lineEnding"),e.consume(m),e.exit("lineEnding"),oe(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(m))}function d(m){return m===o||m===null||K(m)?(e.exit("chunkString"),c(m)):(e.consume(m),m===92?p:d)}function p(m){return m===o||m===92?(e.consume(m),d):d(m)}}function Mr(e,t){let n;return r;function r(i){return K(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):ee(i)?oe(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const vy={name:"definition",tokenize:xy},yy={partial:!0,tokenize:ky};function xy(e,t,n){const r=this;let i;return l;function l(f){return e.enter("definition"),o(f)}function o(f){return xf.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(f)}function a(f){return i=Yn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),f===58?(e.enter("definitionMarker"),e.consume(f),e.exit("definitionMarker"),s):n(f)}function s(f){return Ve(f)?Mr(e,c)(f):c(f)}function c(f){return yf(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(f)}function d(f){return e.attempt(yy,p,p)(f)}function p(f){return ee(f)?oe(e,m,"whitespace")(f):m(f)}function m(f){return f===null||K(f)?(e.exit("definition"),r.parser.defined.push(i),t(f)):n(f)}}function ky(e,t,n){return r;function r(a){return Ve(a)?Mr(e,i)(a):n(a)}function i(a){return kf(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return ee(a)?oe(e,o,"whitespace")(a):o(a)}function o(a){return a===null||K(a)?t(a):n(a)}}const wy={name:"hardBreakEscape",tokenize:Sy};function Sy(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return K(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const by={name:"headingAtx",resolve:jy,tokenize:Cy};function jy(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},jt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function Cy(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||Ve(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||K(d)?(e.exit("atxHeading"),t(d)):ee(d)?oe(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||Ve(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const Ey=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],hc=["pre","script","style","textarea"],Ny={concrete:!0,name:"htmlFlow",resolveTo:Ty,tokenize:Ly},_y={partial:!0,tokenize:Iy},zy={partial:!0,tokenize:Py};function Ty(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function Ly(e,t,n){const r=this;let i,l,o,a,s;return c;function c(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),p}function p(x){return x===33?(e.consume(x),m):x===47?(e.consume(x),l=!0,w):x===63?(e.consume(x),i=3,r.interrupt?t:g):wt(x)?(e.consume(x),o=String.fromCharCode(x),M):n(x)}function m(x){return x===45?(e.consume(x),i=2,f):x===91?(e.consume(x),i=5,a=0,k):wt(x)?(e.consume(x),i=4,r.interrupt?t:g):n(x)}function f(x){return x===45?(e.consume(x),r.interrupt?t:g):n(x)}function k(x){const te="CDATA[";return x===te.charCodeAt(a++)?(e.consume(x),a===te.length?r.interrupt?t:I:k):n(x)}function w(x){return wt(x)?(e.consume(x),o=String.fromCharCode(x),M):n(x)}function M(x){if(x===null||x===47||x===62||Ve(x)){const te=x===47,we=o.toLowerCase();return!te&&!l&&hc.includes(we)?(i=1,r.interrupt?t(x):I(x)):Ey.includes(o.toLowerCase())?(i=6,te?(e.consume(x),h):r.interrupt?t(x):I(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?v(x):y(x))}return x===45||Ge(x)?(e.consume(x),o+=String.fromCharCode(x),M):n(x)}function h(x){return x===62?(e.consume(x),r.interrupt?t:I):n(x)}function v(x){return ee(x)?(e.consume(x),v):O(x)}function y(x){return x===47?(e.consume(x),O):x===58||x===95||wt(x)?(e.consume(x),b):ee(x)?(e.consume(x),y):O(x)}function b(x){return x===45||x===46||x===58||x===95||Ge(x)?(e.consume(x),b):_(x)}function _(x){return x===61?(e.consume(x),S):ee(x)?(e.consume(x),_):y(x)}function S(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,L):ee(x)?(e.consume(x),S):C(x)}function L(x){return x===s?(e.consume(x),s=null,T):x===null||K(x)?n(x):(e.consume(x),L)}function C(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||Ve(x)?_(x):(e.consume(x),C)}function T(x){return x===47||x===62||ee(x)?y(x):n(x)}function O(x){return x===62?(e.consume(x),j):n(x)}function j(x){return x===null||K(x)?I(x):ee(x)?(e.consume(x),j):n(x)}function I(x){return x===45&&i===2?(e.consume(x),N):x===60&&i===1?(e.consume(x),W):x===62&&i===4?(e.consume(x),A):x===63&&i===3?(e.consume(x),g):x===93&&i===5?(e.consume(x),E):K(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(_y,V,H)(x)):x===null||K(x)?(e.exit("htmlFlowData"),H(x)):(e.consume(x),I)}function H(x){return e.check(zy,Q,V)(x)}function Q(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),$}function $(x){return x===null||K(x)?H(x):(e.enter("htmlFlowData"),I(x))}function N(x){return x===45?(e.consume(x),g):I(x)}function W(x){return x===47?(e.consume(x),o="",P):I(x)}function P(x){if(x===62){const te=o.toLowerCase();return hc.includes(te)?(e.consume(x),A):I(x)}return wt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),P):I(x)}function E(x){return x===93?(e.consume(x),g):I(x)}function g(x){return x===62?(e.consume(x),A):x===45&&i===2?(e.consume(x),g):I(x)}function A(x){return x===null||K(x)?(e.exit("htmlFlowData"),V(x)):(e.consume(x),A)}function V(x){return e.exit("htmlFlow"),t(x)}}function Py(e,t,n){const r=this;return i;function i(o){return K(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Iy(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Il,t,n)}}const My={name:"htmlText",tokenize:Ay};function Ay(e,t,n){const r=this;let i,l,o;return a;function a(g){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(g),s}function s(g){return g===33?(e.consume(g),c):g===47?(e.consume(g),_):g===63?(e.consume(g),y):wt(g)?(e.consume(g),C):n(g)}function c(g){return g===45?(e.consume(g),d):g===91?(e.consume(g),l=0,k):wt(g)?(e.consume(g),v):n(g)}function d(g){return g===45?(e.consume(g),f):n(g)}function p(g){return g===null?n(g):g===45?(e.consume(g),m):K(g)?(o=p,W(g)):(e.consume(g),p)}function m(g){return g===45?(e.consume(g),f):p(g)}function f(g){return g===62?N(g):g===45?m(g):p(g)}function k(g){const A="CDATA[";return g===A.charCodeAt(l++)?(e.consume(g),l===A.length?w:k):n(g)}function w(g){return g===null?n(g):g===93?(e.consume(g),M):K(g)?(o=w,W(g)):(e.consume(g),w)}function M(g){return g===93?(e.consume(g),h):w(g)}function h(g){return g===62?N(g):g===93?(e.consume(g),h):w(g)}function v(g){return g===null||g===62?N(g):K(g)?(o=v,W(g)):(e.consume(g),v)}function y(g){return g===null?n(g):g===63?(e.consume(g),b):K(g)?(o=y,W(g)):(e.consume(g),y)}function b(g){return g===62?N(g):y(g)}function _(g){return wt(g)?(e.consume(g),S):n(g)}function S(g){return g===45||Ge(g)?(e.consume(g),S):L(g)}function L(g){return K(g)?(o=L,W(g)):ee(g)?(e.consume(g),L):N(g)}function C(g){return g===45||Ge(g)?(e.consume(g),C):g===47||g===62||Ve(g)?T(g):n(g)}function T(g){return g===47?(e.consume(g),N):g===58||g===95||wt(g)?(e.consume(g),O):K(g)?(o=T,W(g)):ee(g)?(e.consume(g),T):N(g)}function O(g){return g===45||g===46||g===58||g===95||Ge(g)?(e.consume(g),O):j(g)}function j(g){return g===61?(e.consume(g),I):K(g)?(o=j,W(g)):ee(g)?(e.consume(g),j):T(g)}function I(g){return g===null||g===60||g===61||g===62||g===96?n(g):g===34||g===39?(e.consume(g),i=g,H):K(g)?(o=I,W(g)):ee(g)?(e.consume(g),I):(e.consume(g),Q)}function H(g){return g===i?(e.consume(g),i=void 0,$):g===null?n(g):K(g)?(o=H,W(g)):(e.consume(g),H)}function Q(g){return g===null||g===34||g===39||g===60||g===61||g===96?n(g):g===47||g===62||Ve(g)?T(g):(e.consume(g),Q)}function $(g){return g===47||g===62||Ve(g)?T(g):n(g)}function N(g){return g===62?(e.consume(g),e.exit("htmlTextData"),e.exit("htmlText"),t):n(g)}function W(g){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),P}function P(g){return ee(g)?oe(e,E,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(g):E(g)}function E(g){return e.enter("htmlTextData"),o(g)}}const zs={name:"labelEnd",resolveAll:Fy,resolveTo:By,tokenize:Uy},Dy={tokenize:$y},Ry={tokenize:Hy},Oy={tokenize:Vy};function Fy(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&jt(e,0,e.length,n),e}function By(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",c,t]],a=it(a,e.slice(l+1,l+r+3)),a=it(a,[["enter",d,t]]),a=it(a,_s(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=it(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=it(a,e.slice(o+1)),a=it(a,[["exit",s,t]]),jt(e,l,e.length,a),e}function Uy(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(m){return l?l._inactive?p(m):(o=r.parser.defined.includes(Yn(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(m),e.exit("labelMarker"),e.exit("labelEnd"),s):n(m)}function s(m){return m===40?e.attempt(Dy,d,o?d:p)(m):m===91?e.attempt(Ry,d,o?c:p)(m):o?d(m):p(m)}function c(m){return e.attempt(Oy,d,p)(m)}function d(m){return t(m)}function p(m){return l._balanced=!0,n(m)}}function $y(e,t,n){return r;function r(p){return e.enter("resource"),e.enter("resourceMarker"),e.consume(p),e.exit("resourceMarker"),i}function i(p){return Ve(p)?Mr(e,l)(p):l(p)}function l(p){return p===41?d(p):yf(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(p)}function o(p){return Ve(p)?Mr(e,s)(p):d(p)}function a(p){return n(p)}function s(p){return p===34||p===39||p===40?kf(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(p):d(p)}function c(p){return Ve(p)?Mr(e,d)(p):d(p)}function d(p){return p===41?(e.enter("resourceMarker"),e.consume(p),e.exit("resourceMarker"),e.exit("resource"),t):n(p)}}function Hy(e,t,n){const r=this;return i;function i(a){return xf.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(Yn(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Vy(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Wy={name:"labelStartImage",resolveAll:zs.resolveAll,tokenize:Qy};function Qy(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Ky={name:"labelStartLink",resolveAll:zs.resolveAll,tokenize:qy};function qy(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const po={name:"lineEnding",tokenize:Yy};function Yy(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),oe(e,t,"linePrefix")}}const Ui={name:"thematicBreak",tokenize:Xy};function Xy(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),s(c)):r>=3&&(c===null||K(c))?(e.exit("thematicBreak"),t(c)):n(c)}function s(c){return c===i?(e.consume(c),r++,s):(e.exit("thematicBreakSequence"),ee(c)?oe(e,a,"whitespace")(c):a(c))}}const Oe={continuation:{tokenize:ex},exit:nx,name:"list",tokenize:Zy},Gy={partial:!0,tokenize:rx},Jy={partial:!0,tokenize:tx};function Zy(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(f){const k=r.containerState.type||(f===42||f===43||f===45?"listUnordered":"listOrdered");if(k==="listUnordered"?!r.containerState.marker||f===r.containerState.marker:ka(f)){if(r.containerState.type||(r.containerState.type=k,e.enter(k,{_container:!0})),k==="listUnordered")return e.enter("listItemPrefix"),f===42||f===45?e.check(Ui,n,c)(f):c(f);if(!r.interrupt||f===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(f)}return n(f)}function s(f){return ka(f)&&++o<10?(e.consume(f),s):(!r.interrupt||o<2)&&(r.containerState.marker?f===r.containerState.marker:f===41||f===46)?(e.exit("listItemValue"),c(f)):n(f)}function c(f){return e.enter("listItemMarker"),e.consume(f),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||f,e.check(Il,r.interrupt?n:d,e.attempt(Gy,m,p))}function d(f){return r.containerState.initialBlankLine=!0,l++,m(f)}function p(f){return ee(f)?(e.enter("listItemPrefixWhitespace"),e.consume(f),e.exit("listItemPrefixWhitespace"),m):n(f)}function m(f){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(f)}}function ex(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Il,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,oe(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!ee(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Jy,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,oe(e,e.attempt(Oe,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function tx(e,t,n){const r=this;return oe(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function nx(e){e.exit(this.containerState.type)}function rx(e,t,n){const r=this;return oe(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!ee(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const mc={name:"setextUnderline",resolveTo:ix,tokenize:lx};function ix(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function lx(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,p;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){p=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||p)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),ee(c)?oe(e,s,"lineSuffix")(c):s(c))}function s(c){return c===null||K(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const ox={tokenize:ax};function ax(e){const t=this,n=e.attempt(Il,r,e.attempt(this.parser.constructs.flowInitial,i,oe(e,e.attempt(this.parser.constructs.flow,i,e.attempt(py,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const sx={resolveAll:Sf()},ux=wf("string"),cx=wf("text");function wf(e){return{resolveAll:Sf(e==="text"?dx:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function c(d){if(d===null)return!0;const p=i[d];let m=-1;if(p)for(;++m<p.length;){const f=p[m];if(!f.previous||f.previous.call(r,r.previous))return!0}return!1}}}function Sf(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function dx(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)s=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const px={42:Oe,43:Oe,45:Oe,48:Oe,49:Oe,50:Oe,51:Oe,52:Oe,53:Oe,54:Oe,55:Oe,56:Oe,57:Oe,62:hf},fx={91:vy},hx={[-2]:co,[-1]:co,32:co},mx={35:by,42:Ui,45:[mc,Ui],60:Ny,61:mc,95:Ui,96:fc,126:fc},gx={38:gf,92:mf},vx={[-5]:po,[-4]:po,[-3]:po,33:Wy,38:gf,42:wa,60:[Kv,My],91:Ky,92:[wy,mf],93:zs,95:wa,96:oy},yx={null:[wa,sx]},xx={null:[42,95]},kx={null:[]},wx=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:xx,contentInitial:fx,disable:kx,document:px,flow:mx,flowInitial:hx,insideSpan:yx,string:gx,text:vx},Symbol.toStringTag,{value:"Module"}));function Sx(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:L(_),check:L(S),consume:v,enter:y,exit:b,interrupt:L(S,{interrupt:!0})},c={code:null,containerState:{},defineSkip:w,events:[],now:k,parser:e,previous:null,sliceSerialize:m,sliceStream:f,write:p};let d=t.tokenize.call(c,s);return t.resolveAll&&l.push(t),c;function p(j){return o=it(o,j),M(),o[o.length-1]!==null?[]:(C(t,0),c.events=_s(l,c.events,c),c.events)}function m(j,I){return jx(f(j),I)}function f(j){return bx(o,j)}function k(){const{_bufferIndex:j,_index:I,line:H,column:Q,offset:$}=r;return{_bufferIndex:j,_index:I,line:H,column:Q,offset:$}}function w(j){i[j.line]=j.column,O()}function M(){let j;for(;r._index<o.length;){const I=o[r._index];if(typeof I=="string")for(j=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===j&&r._bufferIndex<I.length;)h(I.charCodeAt(r._bufferIndex));else h(I)}}function h(j){d=d(j)}function v(j){K(j)?(r.line++,r.column=1,r.offset+=j===-3?2:1,O()):j!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=j}function y(j,I){const H=I||{};return H.type=j,H.start=k(),c.events.push(["enter",H,c]),a.push(H),H}function b(j){const I=a.pop();return I.end=k(),c.events.push(["exit",I,c]),I}function _(j,I){C(j,I.from)}function S(j,I){I.restore()}function L(j,I){return H;function H(Q,$,N){let W,P,E,g;return Array.isArray(Q)?V(Q):"tokenize"in Q?V([Q]):A(Q);function A(q){return ye;function ye(Ne){const sn=Ne!==null&&q[Ne],En=Ne!==null&&q.null,ai=[...Array.isArray(sn)?sn:sn?[sn]:[],...Array.isArray(En)?En:En?[En]:[]];return V(ai)(Ne)}}function V(q){return W=q,P=0,q.length===0?N:x(q[P])}function x(q){return ye;function ye(Ne){return g=T(),E=q,q.partial||(c.currentConstruct=q),q.name&&c.parser.constructs.disable.null.includes(q.name)?we():q.tokenize.call(I?Object.assign(Object.create(c),I):c,s,te,we)(Ne)}}function te(q){return j(E,g),$}function we(q){return g.restore(),++P<W.length?x(W[P]):N}}}function C(j,I){j.resolveAll&&!l.includes(j)&&l.push(j),j.resolve&&jt(c.events,I,c.events.length-I,j.resolve(c.events.slice(I),c)),j.resolveTo&&(c.events=j.resolveTo(c.events,c))}function T(){const j=k(),I=c.previous,H=c.currentConstruct,Q=c.events.length,$=Array.from(a);return{from:Q,restore:N};function N(){r=j,c.previous=I,c.currentConstruct=H,c.events.length=Q,a=$,O()}}function O(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function bx(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function jx(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function Cx(e){const r={constructs:Pv([wx,...(e||{}).extensions||[]]),content:i(Bv),defined:[],document:i($v),flow:i(ox),lazy:{},string:i(ux),text:i(cx)};return r;function i(l){return o;function o(a){return Sx(r,l,a)}}}function Ex(e){for(;!vf(e););return e}const gc=/[\0\t\n\r]/g;function Nx(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let c,d,p,m,f;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),p=0,t="",n&&(l.charCodeAt(0)===65279&&p++,n=void 0);p<l.length;){if(gc.lastIndex=p,c=gc.exec(l),m=c&&c.index!==void 0?c.index:l.length,f=l.charCodeAt(m),!c){t=l.slice(p);break}if(f===10&&p===m&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),p<m&&(s.push(l.slice(p,m)),e+=m-p),f){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}p=m+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const _x=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function zx(e){return e.replace(_x,Tx)}function Tx(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return ff(n.slice(l?2:1),l?16:10)}return Ns(n)||e}const bf={}.hasOwnProperty;function Lx(e,t,n){return typeof t!="string"&&(n=t,t=void 0),Px(n)(Ex(Cx(n).document().write(Nx()(e,t,!0))))}function Px(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Os),autolinkProtocol:T,autolinkEmail:T,atxHeading:l(As),blockQuote:l(En),characterEscape:T,characterReference:T,codeFenced:l(ai),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(ai,o),codeText:l(Af,o),codeTextData:T,data:T,codeFlowValue:T,definition:l(Df),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Rf),hardBreakEscape:l(Ds),hardBreakTrailing:l(Ds),htmlFlow:l(Rs,o),htmlFlowData:T,htmlText:l(Rs,o),htmlTextData:T,image:l(Of),label:o,link:l(Os),listItem:l(Ff),listItemValue:m,listOrdered:l(Fs,p),listUnordered:l(Fs),paragraph:l(Bf),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(As),strong:l(Uf),thematicBreak:l(Hf)},exit:{atxHeading:s(),atxHeadingSequence:_,autolink:s(),autolinkEmail:sn,autolinkProtocol:Ne,blockQuote:s(),characterEscapeValue:O,characterReferenceMarkerHexadecimal:we,characterReferenceMarkerNumeric:we,characterReferenceValue:q,characterReference:ye,codeFenced:s(M),codeFencedFence:w,codeFencedFenceInfo:f,codeFencedFenceMeta:k,codeFlowValue:O,codeIndented:s(h),codeText:s($),codeTextData:O,data:O,definition:s(),definitionDestinationString:b,definitionLabelString:v,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(I),hardBreakTrailing:s(I),htmlFlow:s(H),htmlFlowData:O,htmlText:s(Q),htmlTextData:O,image:s(W),label:E,labelText:P,lineEnding:j,link:s(N),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:te,resourceDestinationString:g,resourceTitleString:A,resource:V,setextHeading:s(C),setextHeadingLineSequence:L,setextHeadingText:S,strong:s(),thematicBreak:s()}};jf(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(z){let F={type:"root",children:[]};const Y={stack:[F],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},J=[];let re=-1;for(;++re<z.length;)if(z[re][1].type==="listOrdered"||z[re][1].type==="listUnordered")if(z[re][0]==="enter")J.push(re);else{const ut=J.pop();re=i(z,ut,re)}for(re=-1;++re<z.length;){const ut=t[z[re][0]];bf.call(ut,z[re][1].type)&&ut[z[re][1].type].call(Object.assign({sliceSerialize:z[re][2].sliceSerialize},Y),z[re][1])}if(Y.tokenStack.length>0){const ut=Y.tokenStack[Y.tokenStack.length-1];(ut[1]||vc).call(Y,void 0,ut[0])}for(F.position={start:Bt(z.length>0?z[0][1].start:{line:1,column:1,offset:0}),end:Bt(z.length>0?z[z.length-2][1].end:{line:1,column:1,offset:0})},re=-1;++re<t.transforms.length;)F=t.transforms[re](F)||F;return F}function i(z,F,Y){let J=F-1,re=-1,ut=!1,un,Ct,sr,ur;for(;++J<=Y;){const Qe=z[J];switch(Qe[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Qe[0]==="enter"?re++:re--,ur=void 0;break}case"lineEndingBlank":{Qe[0]==="enter"&&(un&&!ur&&!re&&!sr&&(sr=J),ur=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:ur=void 0}if(!re&&Qe[0]==="enter"&&Qe[1].type==="listItemPrefix"||re===-1&&Qe[0]==="exit"&&(Qe[1].type==="listUnordered"||Qe[1].type==="listOrdered")){if(un){let Nn=J;for(Ct=void 0;Nn--;){const Et=z[Nn];if(Et[1].type==="lineEnding"||Et[1].type==="lineEndingBlank"){if(Et[0]==="exit")continue;Ct&&(z[Ct][1].type="lineEndingBlank",ut=!0),Et[1].type="lineEnding",Ct=Nn}else if(!(Et[1].type==="linePrefix"||Et[1].type==="blockQuotePrefix"||Et[1].type==="blockQuotePrefixWhitespace"||Et[1].type==="blockQuoteMarker"||Et[1].type==="listItemIndent"))break}sr&&(!Ct||sr<Ct)&&(un._spread=!0),un.end=Object.assign({},Ct?z[Ct][1].start:Qe[1].end),z.splice(Ct||J,0,["exit",un,Qe[2]]),J++,Y++}if(Qe[1].type==="listItemPrefix"){const Nn={type:"listItem",_spread:!1,start:Object.assign({},Qe[1].start),end:void 0};un=Nn,z.splice(J,0,["enter",Nn,Qe[2]]),J++,Y++,sr=void 0,ur=!0}}}return z[F][1]._spread=ut,Y}function l(z,F){return Y;function Y(J){a.call(this,z(J),J),F&&F.call(this,J)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(z,F,Y){this.stack[this.stack.length-1].children.push(z),this.stack.push(z),this.tokenStack.push([F,Y||void 0]),z.position={start:Bt(F.start),end:void 0}}function s(z){return F;function F(Y){z&&z.call(this,Y),c.call(this,Y)}}function c(z,F){const Y=this.stack.pop(),J=this.tokenStack.pop();if(J)J[0].type!==z.type&&(F?F.call(this,z,J[0]):(J[1]||vc).call(this,z,J[0]));else throw new Error("Cannot close `"+z.type+"` ("+Ir({start:z.start,end:z.end})+"): it’s not open");Y.position.end=Bt(z.end)}function d(){return Tv(this.stack.pop())}function p(){this.data.expectingFirstListItemValue=!0}function m(z){if(this.data.expectingFirstListItemValue){const F=this.stack[this.stack.length-2];F.start=Number.parseInt(this.sliceSerialize(z),10),this.data.expectingFirstListItemValue=void 0}}function f(){const z=this.resume(),F=this.stack[this.stack.length-1];F.lang=z}function k(){const z=this.resume(),F=this.stack[this.stack.length-1];F.meta=z}function w(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function M(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function h(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z.replace(/(\r?\n|\r)$/g,"")}function v(z){const F=this.resume(),Y=this.stack[this.stack.length-1];Y.label=F,Y.identifier=Yn(this.sliceSerialize(z)).toLowerCase()}function y(){const z=this.resume(),F=this.stack[this.stack.length-1];F.title=z}function b(){const z=this.resume(),F=this.stack[this.stack.length-1];F.url=z}function _(z){const F=this.stack[this.stack.length-1];if(!F.depth){const Y=this.sliceSerialize(z).length;F.depth=Y}}function S(){this.data.setextHeadingSlurpLineEnding=!0}function L(z){const F=this.stack[this.stack.length-1];F.depth=this.sliceSerialize(z).codePointAt(0)===61?1:2}function C(){this.data.setextHeadingSlurpLineEnding=void 0}function T(z){const Y=this.stack[this.stack.length-1].children;let J=Y[Y.length-1];(!J||J.type!=="text")&&(J=$f(),J.position={start:Bt(z.start),end:void 0},Y.push(J)),this.stack.push(J)}function O(z){const F=this.stack.pop();F.value+=this.sliceSerialize(z),F.position.end=Bt(z.end)}function j(z){const F=this.stack[this.stack.length-1];if(this.data.atHardBreak){const Y=F.children[F.children.length-1];Y.position.end=Bt(z.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(F.type)&&(T.call(this,z),O.call(this,z))}function I(){this.data.atHardBreak=!0}function H(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function Q(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function $(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function N(){const z=this.stack[this.stack.length-1];if(this.data.inReference){const F=this.data.referenceType||"shortcut";z.type+="Reference",z.referenceType=F,delete z.url,delete z.title}else delete z.identifier,delete z.label;this.data.referenceType=void 0}function W(){const z=this.stack[this.stack.length-1];if(this.data.inReference){const F=this.data.referenceType||"shortcut";z.type+="Reference",z.referenceType=F,delete z.url,delete z.title}else delete z.identifier,delete z.label;this.data.referenceType=void 0}function P(z){const F=this.sliceSerialize(z),Y=this.stack[this.stack.length-2];Y.label=zx(F),Y.identifier=Yn(F).toLowerCase()}function E(){const z=this.stack[this.stack.length-1],F=this.resume(),Y=this.stack[this.stack.length-1];if(this.data.inReference=!0,Y.type==="link"){const J=z.children;Y.children=J}else Y.alt=F}function g(){const z=this.resume(),F=this.stack[this.stack.length-1];F.url=z}function A(){const z=this.resume(),F=this.stack[this.stack.length-1];F.title=z}function V(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function te(z){const F=this.resume(),Y=this.stack[this.stack.length-1];Y.label=F,Y.identifier=Yn(this.sliceSerialize(z)).toLowerCase(),this.data.referenceType="full"}function we(z){this.data.characterReferenceType=z.type}function q(z){const F=this.sliceSerialize(z),Y=this.data.characterReferenceType;let J;Y?(J=ff(F,Y==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):J=Ns(F);const re=this.stack[this.stack.length-1];re.value+=J}function ye(z){const F=this.stack.pop();F.position.end=Bt(z.end)}function Ne(z){O.call(this,z);const F=this.stack[this.stack.length-1];F.url=this.sliceSerialize(z)}function sn(z){O.call(this,z);const F=this.stack[this.stack.length-1];F.url="mailto:"+this.sliceSerialize(z)}function En(){return{type:"blockquote",children:[]}}function ai(){return{type:"code",lang:null,meta:null,value:""}}function Af(){return{type:"inlineCode",value:""}}function Df(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Rf(){return{type:"emphasis",children:[]}}function As(){return{type:"heading",depth:0,children:[]}}function Ds(){return{type:"break"}}function Rs(){return{type:"html",value:""}}function Of(){return{type:"image",title:null,url:"",alt:null}}function Os(){return{type:"link",title:null,url:"",children:[]}}function Fs(z){return{type:"list",ordered:z.type==="listOrdered",start:null,spread:z._spread,children:[]}}function Ff(z){return{type:"listItem",spread:z._spread,checked:null,children:[]}}function Bf(){return{type:"paragraph",children:[]}}function Uf(){return{type:"strong",children:[]}}function $f(){return{type:"text",value:""}}function Hf(){return{type:"thematicBreak"}}}function Bt(e){return{line:e.line,column:e.column,offset:e.offset}}function jf(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?jf(e,r):Ix(e,r)}}function Ix(e,t){let n;for(n in t)if(bf.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function vc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Ir({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Ir({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Ir({start:t.start,end:t.end})+") is still open")}function Mx(e){const t=this;t.parser=n;function n(r){return Lx(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function Ax(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function Dx(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function Rx(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function Ox(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Fx(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Bx(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=ar(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const c={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,c),e.applyData(t,c)}function Ux(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function $x(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Cf(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function Hx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Cf(e,t);const i={src:ar(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function Vx(e,t){const n={src:ar(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function Wx(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function Qx(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Cf(e,t);const i={href:ar(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function Kx(e,t){const n={href:ar(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function qx(e,t,n){const r=e.all(t),i=n?Yx(n):Ef(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let p;d&&d.type==="element"&&d.tagName==="p"?p=d:(p={type:"element",tagName:"p",properties:{},children:[]},r.unshift(p)),p.children.length>0&&p.children.unshift({type:"text",value:" "}),p.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function Yx(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Ef(n[r])}return t}function Ef(e){const t=e.spread;return t??e.children.length>1}function Xx(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function Gx(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Jx(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function Zx(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function e1(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=bs(t.children[1]),s=of(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function t1(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const c=[];for(;++s<a;){const p=t.children[s],m={},f=o?o[s]:void 0;f&&(m.align=f);let k={type:"element",tagName:l,properties:m,children:[]};p&&(k.children=e.all(p),e.patch(p,k),k=e.applyData(p,k)),c.push(k)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function n1(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const yc=9,xc=32;function r1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(kc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(kc(t.slice(i),i>0,!1)),l.join("")}function kc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===yc||l===xc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===yc||l===xc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function i1(e,t){const n={type:"text",value:r1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function l1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const o1={blockquote:Ax,break:Dx,code:Rx,delete:Ox,emphasis:Fx,footnoteReference:Bx,heading:Ux,html:$x,imageReference:Hx,image:Vx,inlineCode:Wx,linkReference:Qx,link:Kx,listItem:qx,list:Xx,paragraph:Gx,root:Jx,strong:Zx,table:e1,tableCell:n1,tableRow:t1,text:i1,thematicBreak:l1,toml:Ci,yaml:Ci,definition:Ci,footnoteDefinition:Ci};function Ci(){}const Nf=-1,Ml=0,Ar=1,hl=2,Ts=3,Ls=4,Ps=5,Is=6,_f=7,zf=8,wc=typeof self=="object"?self:globalThis,a1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Ml:case Nf:return n(o,i);case Ar:{const a=n([],i);for(const s of o)a.push(r(s));return a}case hl:{const a=n({},i);for(const[s,c]of o)a[r(s)]=r(c);return a}case Ts:return n(new Date(o),i);case Ls:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Ps:{const a=n(new Map,i);for(const[s,c]of o)a.set(r(s),r(c));return a}case Is:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case _f:{const{name:a,message:s}=o;return n(new wc[a](s),i)}case zf:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new wc[l](o),i)};return r},Sc=e=>a1(new Map,e)(0),zn="",{toString:s1}={},{keys:u1}=Object,xr=e=>{const t=typeof e;if(t!=="object"||!e)return[Ml,t];const n=s1.call(e).slice(8,-1);switch(n){case"Array":return[Ar,zn];case"Object":return[hl,zn];case"Date":return[Ts,zn];case"RegExp":return[Ls,zn];case"Map":return[Ps,zn];case"Set":return[Is,zn];case"DataView":return[Ar,n]}return n.includes("Array")?[Ar,n]:n.includes("Error")?[_f,n]:[hl,n]},Ei=([e,t])=>e===Ml&&(t==="function"||t==="symbol"),c1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=xr(o);switch(a){case Ml:{let d=o;switch(s){case"bigint":a=zf,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([Nf],o)}return i([a,d],o)}case Ar:{if(s){let m=o;return s==="DataView"?m=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(m=new Uint8Array(o)),i([s,[...m]],o)}const d=[],p=i([a,d],o);for(const m of o)d.push(l(m));return p}case hl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],p=i([a,d],o);for(const m of u1(o))(e||!Ei(xr(o[m])))&&d.push([l(m),l(o[m])]);return p}case Ts:return i([a,o.toISOString()],o);case Ls:{const{source:d,flags:p}=o;return i([a,{source:d,flags:p}],o)}case Ps:{const d=[],p=i([a,d],o);for(const[m,f]of o)(e||!(Ei(xr(m))||Ei(xr(f))))&&d.push([l(m),l(f)]);return p}case Is:{const d=[],p=i([a,d],o);for(const m of o)(e||!Ei(xr(m)))&&d.push(l(m));return p}}const{message:c}=o;return i([a,{name:s,message:c}],o)};return l},bc=(e,{json:t,lossy:n}={})=>{const r=[];return c1(!(t||n),!!t,new Map,r)(e),r},ml=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?Sc(bc(e,t)):structuredClone(e):(e,t)=>Sc(bc(e,t));function d1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function p1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function f1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||d1,r=e.options.footnoteBackLabel||p1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[s]);if(!c)continue;const d=e.all(c),p=String(c.identifier).toUpperCase(),m=ar(p.toLowerCase());let f=0;const k=[],w=e.footnoteCounts.get(p);for(;w!==void 0&&++f<=w;){k.length>0&&k.push({type:"text",value:" "});let v=typeof n=="string"?n:n(s,f);typeof v=="string"&&(v={type:"text",value:v}),k.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+m+(f>1?"-"+f:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,f),className:["data-footnote-backref"]},children:Array.isArray(v)?v:[v]})}const M=d[d.length-1];if(M&&M.type==="element"&&M.tagName==="p"){const v=M.children[M.children.length-1];v&&v.type==="text"?v.value+=" ":M.children.push({type:"text",value:" "}),M.children.push(...k)}else d.push(...k);const h={type:"element",tagName:"li",properties:{id:t+"fn-"+m},children:e.wrap(d,!0)};e.patch(c,h),a.push(h)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...ml(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const Tf=function(e){if(e==null)return v1;if(typeof e=="function")return Al(e);if(typeof e=="object")return Array.isArray(e)?h1(e):m1(e);if(typeof e=="string")return g1(e);throw new Error("Expected function, string, or object as test")};function h1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=Tf(e[n]);return Al(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function m1(e){const t=e;return Al(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function g1(e){return Al(t);function t(n){return n&&n.type===e}}function Al(e){return t;function t(n,r,i){return!!(y1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function v1(){return!0}function y1(e){return e!==null&&typeof e=="object"&&"type"in e}const Lf=[],x1=!0,jc=!1,k1="skip";function w1(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=Tf(i),o=r?-1:1;a(e,void 0,[])();function a(s,c,d){const p=s&&typeof s=="object"?s:{};if(typeof p.type=="string"){const f=typeof p.tagName=="string"?p.tagName:typeof p.name=="string"?p.name:void 0;Object.defineProperty(m,"name",{value:"node ("+(s.type+(f?"<"+f+">":""))+")"})}return m;function m(){let f=Lf,k,w,M;if((!t||l(s,c,d[d.length-1]||void 0))&&(f=S1(n(s,d)),f[0]===jc))return f;if("children"in s&&s.children){const h=s;if(h.children&&f[0]!==k1)for(w=(r?h.children.length:-1)+o,M=d.concat(h);w>-1&&w<h.children.length;){const v=h.children[w];if(k=a(v,w,M)(),k[0]===jc)return k;w=typeof k[1]=="number"?k[1]:w+o}}return f}}}function S1(e){return Array.isArray(e)?e:typeof e=="number"?[x1,e]:e==null?Lf:[e]}function Pf(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),w1(e,l,a,i);function a(s,c){const d=c[c.length-1],p=d?d.children.indexOf(s):void 0;return o(s,p,d)}}const Sa={}.hasOwnProperty,b1={};function j1(e,t){const n=t||b1,r=new Map,i=new Map,l=new Map,o={...o1,...n.handlers},a={all:c,applyData:E1,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:C1,wrap:_1};return Pf(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const p=d.type==="definition"?r:i,m=String(d.identifier).toUpperCase();p.has(m)||p.set(m,d)}}),a;function s(d,p){const m=d.type,f=a.handlers[m];if(Sa.call(a.handlers,m)&&f)return f(a,d,p);if(a.options.passThrough&&a.options.passThrough.includes(m)){if("children"in d){const{children:w,...M}=d,h=ml(M);return h.children=a.all(d),h}return ml(d)}return(a.options.unknownHandler||N1)(a,d,p)}function c(d){const p=[];if("children"in d){const m=d.children;let f=-1;for(;++f<m.length;){const k=a.one(m[f],d);if(k){if(f&&m[f-1].type==="break"&&(!Array.isArray(k)&&k.type==="text"&&(k.value=Cc(k.value)),!Array.isArray(k)&&k.type==="element")){const w=k.children[0];w&&w.type==="text"&&(w.value=Cc(w.value))}Array.isArray(k)?p.push(...k):p.push(k)}}}return p}}function C1(e,t){e.position&&(t.position=sv(e))}function E1(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,ml(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function N1(e,t){const n=t.data||{},r="value"in t&&!(Sa.call(n,"hProperties")||Sa.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function _1(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Cc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Ec(e,t){const n=j1(e,t),r=n.one(e,void 0),i=f1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function z1(e,t){return e&&"run"in e?async function(n,r){const i=Ec(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Ec(n,{file:r,...e||t})}}function Nc(e){if(e)throw e}var $i=Object.prototype.hasOwnProperty,If=Object.prototype.toString,_c=Object.defineProperty,zc=Object.getOwnPropertyDescriptor,Tc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):If.call(t)==="[object Array]"},Lc=function(t){if(!t||If.call(t)!=="[object Object]")return!1;var n=$i.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&$i.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||$i.call(t,i)},Pc=function(t,n){_c&&n.name==="__proto__"?_c(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Ic=function(t,n){if(n==="__proto__")if($i.call(t,n)){if(zc)return zc(t,n).value}else return;return t[n]},T1=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<c;++s)if(t=arguments[s],t!=null)for(n in t)r=Ic(a,n),i=Ic(t,n),a!==i&&(d&&i&&(Lc(i)||(l=Tc(i)))?(l?(l=!1,o=r&&Tc(r)?r:[]):o=r&&Lc(r)?r:{},Pc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Pc(a,{name:n,newValue:i}));return a};const fo=Ca(T1);function ba(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function L1(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...c){const d=e[++l];let p=-1;if(s){o(s);return}for(;++p<i.length;)(c[p]===null||c[p]===void 0)&&(c[p]=i[p]);i=c,d?P1(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function P1(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const xt={basename:I1,dirname:M1,extname:A1,join:D1,sep:"/"};function I1(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');oi(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function M1(e){if(oi(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function A1(e){oi(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function D1(...e){let t=-1,n;for(;++t<e.length;)oi(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":R1(n)}function R1(e){oi(e);const t=e.codePointAt(0)===47;let n=O1(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function O1(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function oi(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const F1={cwd:B1};function B1(){return"/"}function ja(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function U1(e){if(typeof e=="string")e=new URL(e);else if(!ja(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return $1(e)}function $1(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const ho=["history","path","basename","stem","extname","dirname"];class Mf{constructor(t){let n;t?ja(t)?n={path:t}:typeof t=="string"||H1(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":F1.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<ho.length;){const l=ho[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)ho.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?xt.basename(this.path):void 0}set basename(t){go(t,"basename"),mo(t,"basename"),this.path=xt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?xt.dirname(this.path):void 0}set dirname(t){Mc(this.basename,"dirname"),this.path=xt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?xt.extname(this.path):void 0}set extname(t){if(mo(t,"extname"),Mc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=xt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){ja(t)&&(t=U1(t)),go(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?xt.basename(this.path,this.extname):void 0}set stem(t){go(t,"stem"),mo(t,"stem"),this.path=xt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new Pe(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function mo(e,t){if(e&&e.includes(xt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+xt.sep+"`")}function go(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Mc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function H1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const V1=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},W1={}.hasOwnProperty;class Ms extends V1{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=L1()}copy(){const t=new Ms;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(fo(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(xo("data",this.frozen),this.namespace[t]=n,this):W1.call(this.namespace,t)&&this.namespace[t]||void 0:t?(xo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Ni(t),r=this.parser||this.Parser;return vo("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),vo("process",this.parser||this.Parser),yo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Ni(t),s=r.parse(a);r.run(s,a,function(d,p,m){if(d||!p||!m)return c(d);const f=p,k=r.stringify(f,m);q1(k)?m.value=k:m.result=k,c(d,m)});function c(d,p){d||!p?o(d):l?l(p):n(void 0,p)}}}processSync(t){let n=!1,r;return this.freeze(),vo("processSync",this.parser||this.Parser),yo("processSync",this.compiler||this.Compiler),this.process(t,i),Dc("processSync","process",n),r;function i(l,o){n=!0,Nc(l),r=o}}run(t,n,r){Ac(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Ni(n);i.run(t,s,c);function c(d,p,m){const f=p||t;d?a(d):o?o(f):r(void 0,f,m)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Dc("runSync","run",r),i;function l(o,a){Nc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Ni(n),i=this.compiler||this.Compiler;return yo("stringify",i),Ac(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(xo("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")s(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...p]=c;s(d,p)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=fo(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const p=c[d];l(p)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function s(c,d){let p=-1,m=-1;for(;++p<r.length;)if(r[p][0]===c){m=p;break}if(m===-1)r.push([c,...d]);else if(d.length>0){let[f,...k]=d;const w=r[m][1];ba(w)&&ba(f)&&(f=fo(!0,w,f)),r[m]=[c,f,...k]}}}}const Q1=new Ms().freeze();function vo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function yo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function xo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Ac(e){if(!ba(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Dc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Ni(e){return K1(e)?e:new Mf(e)}function K1(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function q1(e){return typeof e=="string"||Y1(e)}function Y1(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const X1="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Rc=[],Oc={allowDangerousHtml:!0},G1=/^(https?|ircs?|mailto|xmpp)$/i,J1=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function Z1(e){const t=e0(e),n=t0(e);return n0(t.runSync(t.parse(n),n),e)}function e0(e){const t=e.rehypePlugins||Rc,n=e.remarkPlugins||Rc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Oc}:Oc;return Q1().use(Mx).use(n).use(z1,r).use(t)}function t0(e){const t=e.children||"",n=new Mf;return typeof t=="string"&&(n.value=t),n}function n0(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||r0;for(const d of J1)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+X1+d.id,void 0);return Pf(e,c),fv(e,{Fragment:u.Fragment,components:i,ignoreInvalidStyle:!0,jsx:u.jsx,jsxs:u.jsxs,passKeys:!0,passNode:!0});function c(d,p,m){if(d.type==="raw"&&m&&typeof p=="number")return o?m.children.splice(p,1):m.children[p]={type:"text",value:d.value},p;if(d.type==="element"){let f;for(f in uo)if(Object.hasOwn(uo,f)&&Object.hasOwn(d.properties,f)){const k=d.properties[f],w=uo[f];(w===null||w.includes(d.tagName))&&(d.properties[f]=s(String(k||""),f,d))}}if(d.type==="element"){let f=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!f&&r&&typeof p=="number"&&(f=!r(d,p,m)),f&&m&&typeof p=="number")return a&&d.children?m.children.splice(p,1,...d.children):m.children.splice(p,1),p}}}function r0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||G1.test(e.slice(0,t))?e:""}const Fc=10*1024,ko=200,Ye={send:u.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),u.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),u.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),u.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},i0=e=>{switch(e){case"directive":return Ye.directive;case"question":return Ye.question;case"status":return Ye.status;case"result":return Ye.result;case"approval_request":return Ye.lock;default:return Ye.directive}},l0=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=B.useRef(null),[a,s]=zt.useState(""),[c,d]=zt.useState("directive"),[p,m]=zt.useState(""),[f,k]=zt.useState(!1),[w,M]=zt.useState(new Map),[h,v]=zt.useState(new Set),[y,b]=B.useState(new Set),_=N=>{const W=(N.match(/\n/g)||[]).length+1;if(!(N.length>Fc||W>ko))return{needsTruncation:!1,truncated:N,fullLength:N.length,lineCount:W};let E=N.slice(0,Fc);const g=E.split(`
`);g.length>ko&&(E=g.slice(0,ko).join(`
`));const A=E.lastIndexOf(`
`);return A>E.length*.8&&(E=E.slice(0,A)),{needsTruncation:!0,truncated:E,fullLength:N.length,lineCount:W}},S=N=>{b(W=>{const P=new Set(W);return P.has(N)?P.delete(N):P.add(N),P})};B.useEffect(()=>{e!=null&&e.workspace?m(e.workspace):m("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),B.useEffect(()=>{var N;(N=o.current)==null||N.scrollIntoView({behavior:"smooth"})},[t]);const L=N=>{m(N),r&&r(N)},C=()=>{a.trim()&&(n(a,c,p||void 0),s(""))},T=N=>{N.key==="Enter"&&!N.shiftKey&&(N.preventDefault(),C())},O=N=>new Date(N).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),j=N=>N.length>12?`${N.slice(0,8)}...`:N,I=N=>{if(!N.metadata_json)return null;try{return JSON.parse(N.metadata_json).approval_id||null}catch{return null}},H=N=>{const W=w.get(N)||"";i&&(i(N,W),v(P=>new Set(P).add(N)),M(P=>{const E=new Map(P);return E.delete(N),E}))},Q=N=>{const W=w.get(N)||"";if(!W.trim()){alert("Please provide a reason for rejection");return}l&&(l(N,W),v(P=>new Set(P).add(N)),M(P=>{const E=new Map(P);return E.delete(N),E}))},$=(N,W)=>{M(P=>new Map(P).set(N,W))};return e?u.jsxs("div",{className:"conversation-view",children:[u.jsxs("div",{className:"conversation-header",children:[u.jsxs("div",{className:"header-info",children:[u.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&u.jsxs("span",{className:"thread-agent-badge",children:[Ye.bot,e.target_agent]})]}),u.jsxs("div",{className:"header-stats",children:[u.jsxs("span",{className:"message-count",children:[t.length," messages"]}),u.jsx("span",{className:"thread-id",title:e.id,children:j(e.id)})]})]}),u.jsxs("div",{className:"messages-container",children:[t.length===0?u.jsxs("div",{className:"empty-messages",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("p",{children:"No messages yet"}),u.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((N,W)=>{const P=N.from_type==="human",E=W===0||t[W-1].from_type!==N.from_type,g=y.has(N.id),{needsTruncation:A,truncated:V,fullLength:x,lineCount:te}=_(N.content),we=g?N.content:V;return u.jsxs("div",{className:`message ${P?"human":"agent"}`,children:[u.jsx("div",{className:`message-avatar ${E?"visible":""}`,children:E&&(P?Ye.user:Ye.bot)}),u.jsxs("div",{className:"message-body",children:[E&&u.jsxs("div",{className:"message-meta",children:[u.jsx("span",{className:"sender-name",children:N.from_id}),u.jsxs("span",{className:"kind-badge",children:[i0(N.kind)," ",N.kind]}),u.jsx("span",{className:"message-time",children:O(N.created_at)})]}),u.jsxs("div",{className:"message-content",children:[N.kind==="result"||!P?u.jsx(Z1,{components:{a:({href:q,children:ye})=>{let Ne=q;return q&&q.startsWith("/")&&!q.startsWith("//")&&(Ne=`file://${q}`),u.jsx("a",{href:Ne,target:"_blank",rel:"noopener noreferrer",children:ye})},code:({className:q,children:ye,...Ne})=>!q?u.jsx("code",{className:"inline-code",...Ne,children:ye}):u.jsx("code",{className:q,...Ne,children:ye})},children:we}):we,A&&u.jsx("div",{className:"truncation-notice",children:u.jsx("button",{className:"expand-btn",onClick:()=>S(N.id),children:g?u.jsx(u.Fragment,{children:"Show less"}):u.jsxs(u.Fragment,{children:["Show more (",Math.round(x/1024),"KB, ",te," lines)"]})})}),N.kind==="approval_request"&&(()=>{const q=I(N),ye=q&&h.has(q);return q?u.jsx("div",{className:"inline-approval",children:ye?u.jsxs("div",{className:"approval-handled",children:[Ye.check,u.jsx("span",{children:"Action taken"})]}):u.jsxs(u.Fragment,{children:[u.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:w.get(q)||"",onChange:Ne=>$(q,Ne.target.value)}),u.jsxs("div",{className:"approval-actions",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>Q(q),title:"Reject",children:[Ye.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>H(q),title:"Approve",children:[Ye.check,"Approve"]})]})]})}):null})()]}),u.jsxs("div",{className:"message-footer",children:[u.jsxs("span",{className:"message-seq",children:["#",N.message_seq]}),N.delivery_state!=="acked"&&u.jsx("span",{className:`delivery-status ${N.delivery_state}`,children:N.delivery_state==="pending"?"sending...":"delivered"})]})]})]},N.id)}),u.jsx("div",{ref:o})]}),u.jsxs("div",{className:"input-area",children:[f&&u.jsxs("div",{className:"workspace-input-row",children:[u.jsx("input",{type:"text",value:p,onChange:N=>L(N.target.value),onBlur:()=>{r&&r(p)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),u.jsx("button",{onClick:async()=>{try{const W=await(await fetch("/api/select-folder")).json();!W.cancelled&&W.path&&L(W.path)}catch(N){console.error("Failed to open folder picker:",N)}},className:"workspace-browse",title:"Browse for folder",children:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),u.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),u.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),p&&u.jsx("button",{onClick:()=>{L(""),k(!1)},className:"workspace-clear",children:"Clear"})]}),u.jsxs("div",{className:"input-wrapper",children:[u.jsx("button",{onClick:()=>k(!f),className:`workspace-toggle ${p?"has-workspace":""}`,title:p||"Set working directory",children:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),u.jsxs("select",{value:c,onChange:N=>d(N.target.value),className:"kind-selector",children:[u.jsx("option",{value:"directive",children:"Directive"}),u.jsx("option",{value:"question",children:"Question"})]}),u.jsx("textarea",{value:a,onChange:N=>s(N.target.value),onKeyPress:T,placeholder:p?`Message (workspace: ${p.split("/").pop()})`:"Type a message...",rows:1}),u.jsx("button",{onClick:C,className:"send-btn",disabled:!a.trim(),children:Ye.send})]}),u.jsxs("div",{className:"input-hint",children:["Press ",u.jsx("kbd",{children:"Enter"})," to send, ",u.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),u.jsx("style",{children:`
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
      `})]}):null},o0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,reconnectInterval:l=5e3})=>{const o=B.useRef(null),[a,s]=B.useState(!1),[c,d]=B.useState(null),p=B.useRef(null),m=B.useRef(new Map),f=B.useCallback(()=>{try{const b=`${e}?instance_id=${t}`;o.current=new WebSocket(b),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),d(null),m.current.forEach((_,S)=>{M(S,_)})},o.current.onmessage=_=>{try{const S=JSON.parse(_.data);k(S)}catch(S){console.error("Failed to parse WebSocket message:",S)}},o.current.onerror=_=>{console.error("WebSocket error:",_),d("Connection error")},o.current.onclose=()=>{console.log("WebSocket disconnected"),s(!1),p.current=setTimeout(()=>{console.log("Attempting to reconnect..."),f()},l)}}catch(b){console.error("Failed to connect to WebSocket:",b),d("Failed to connect")}},[e,t,l]),k=B.useCallback(b=>{switch(b.type){case"message":n&&b.data&&n(b.data);break;case"batch":if(r&&b.data){const _=b.data;r(_),n&&_.messages.forEach(S=>n(S))}break;case"error":i&&b.data&&i(b.data),console.error("WebSocket error event:",b.data);break;case"pong":break;default:console.log("Unknown event type:",b.type)}},[n,r,i]),w=B.useCallback(b=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(b)):console.warn("WebSocket not connected, cannot send event")},[]),M=B.useCallback((b,_=0)=>{m.current.set(b,_);const S={type:"subscribe",timestamp:Date.now(),data:{thread_id:b,from_seq:_}};w(S)},[w]),h=B.useCallback((b,_)=>{const S=m.current.get(b)||0;_>S&&m.current.set(b,_);const L={type:"ack",timestamp:Date.now(),data:{thread_id:b,ack_seq:_}};w(L)},[w]),v=B.useCallback(()=>{const b={type:"ping",timestamp:Date.now()};w(b)},[w]),y=B.useCallback(b=>{m.current.delete(b)},[]);return B.useEffect(()=>(f(),()=>{p.current&&clearTimeout(p.current),o.current&&o.current.close()}),[f]),B.useEffect(()=>{if(!a)return;const b=setInterval(()=>{v()},3e4);return()=>clearInterval(b)},[a,v]),{isConnected:a,connectionError:c,subscribe:M,unsubscribe:y,acknowledge:h,ping:v}},a0=({connected:e})=>u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?u.jsxs(u.Fragment,{children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):u.jsxs(u.Fragment,{children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),u.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),s0=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=B.useState([]),[o,a]=B.useState(null),[s,c]=B.useState(new Map),[d,p]=B.useState(new Map),[m,f]=B.useState([]),[k,w]=B.useState(!1),[M,h]=B.useState(""),{isConnected:v,subscribe:y,acknowledge:b}=o0({url:e,instanceId:t,onMessage:_,onBatch:S});function _(E){const g={id:E.id,thread_id:E.thread_id,message_seq:E.message_seq,created_at:E.created_at,from_type:E.from_type,from_id:E.from_id,to_type:E.to_type,to_id:E.to_id,kind:E.kind,subject:E.subject,content:E.content,metadata_json:E.metadata_json,delivery_state:"visible",business_state:"open"};c(A=>{const V=A.get(g.thread_id)||[];return V.find(x=>x.id===g.id)?A:new Map(A).set(g.thread_id,[...V,g].sort((x,te)=>x.message_seq-te.message_seq))}),g.thread_id!==o&&p(A=>{const V=A.get(g.thread_id)||0;return new Map(A).set(g.thread_id,V+1)}),b(g.thread_id,g.message_seq)}function S(E){E.messages.forEach(g=>{_(g)})}const L=B.useCallback(E=>{if(a(E),p(g=>{const A=new Map(g);return A.delete(E),A}),v){const g=s.get(E)||[],A=g.length>0?Math.max(...g.map(V=>V.message_seq)):0;y(E,A)}},[v,y,s]),C=B.useCallback(async(E,g,A)=>{if(!o)return;const V=A?JSON.stringify({workspace:A}):void 0;try{const x=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:g,content:E,metadata_json:V})});if(!x.ok){console.error("Failed to send message:",await x.text());return}const te=await x.json();c(we=>{const q=we.get(o)||[];return q.find(ye=>ye.id===te.id)?we:new Map(we).set(o,[...q,te])})}catch(x){console.error("Error sending message:",x)}},[o,t]);B.useEffect(()=>{(async()=>{try{const g=await fetch("/api/threads");if(!g.ok){console.error("Failed to fetch threads:",await g.text());return}const A=await g.json();l(A),A.length>0&&!o&&a(A[0].id)}catch(g){console.error("Error fetching threads:",g)}})()},[]),B.useEffect(()=>{n&&i.length>0&&(i.some(g=>g.id===n)&&(a(n),p(g=>{const A=new Map(g);return A.delete(n),A})),r&&r())},[n,i,r]);const T=B.useCallback(async E=>{try{const g=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:E,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!g.ok){console.error("Failed to create thread:",await g.text());return}const A=await g.json();l(V=>[A,...V]),a(A.id)}catch(g){console.error("Error creating thread:",g)}},[t]),O=B.useCallback(async()=>{try{const E=await fetch("/api/agents");if(!E.ok){console.error("Failed to fetch agents:",await E.text());return}const g=await E.json();f(g.running||[])}catch(E){console.error("Error fetching agents:",E)}},[]);B.useEffect(()=>{O();const E=setInterval(O,5e3);return()=>clearInterval(E)},[O]);const j=B.useCallback(async()=>{if(M.trim())try{const E=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:M.trim()})});if(!E.ok){const A=await E.text();console.error("Failed to launch agent:",A),alert(`Failed to launch agent: ${A}`);return}const g=await E.json();f(A=>[...A,g]),h(""),w(!1)}catch(E){console.error("Error launching agent:",E)}},[M]),I=B.useCallback(async E=>{try{const g=await fetch(`/api/agents/${E}`,{method:"DELETE"});if(!g.ok){console.error("Failed to stop agent:",await g.text());return}f(A=>A.filter(V=>V.instance_id!==E))}catch(g){console.error("Error stopping agent:",g)}},[]),H=B.useCallback(async E=>{if(o)try{const g=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:E})});if(!g.ok){console.error("Failed to update workspace:",await g.text());return}const A=await g.json();l(V=>V.map(x=>x.id===o?A:x))}catch(g){console.error("Error updating workspace:",g)}},[o]),Q=B.useCallback(async E=>{try{const g=await fetch(`/api/threads/${E}`,{method:"DELETE"});if(!g.ok){console.error("Failed to delete thread:",await g.text());return}l(A=>A.filter(V=>V.id!==E)),c(A=>{const V=new Map(A);return V.delete(E),V}),p(A=>{const V=new Map(A);return V.delete(E),V}),o===E&&a(null)}catch(g){console.error("Error deleting thread:",g)}},[o]),$=B.useCallback(async(E,g)=>{try{const A=await fetch(`/api/threads/${E}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:g})});if(!A.ok){console.error("Failed to rename thread:",await A.text());return}const V=await A.json();l(x=>x.map(te=>te.id===E?V:te))}catch(A){console.error("Error renaming thread:",A)}},[]),N=B.useCallback(async(E,g)=>{try{const A=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!A.ok){const V=await A.text();console.error("Failed to approve request:",V),alert(`Failed to approve: ${V}`);return}console.log("Approval approved successfully")}catch(A){console.error("Error approving request:",A)}},[]),W=B.useCallback(async(E,g)=>{try{const A=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!A.ok){const V=await A.text();console.error("Failed to reject request:",V),alert(`Failed to reject: ${V}`);return}console.log("Approval rejected successfully")}catch(A){console.error("Error rejecting request:",A)}},[]),P=o?s.get(o)||[]:[];return u.jsxs("div",{className:"message-center",children:[u.jsxs("div",{className:"status-bar",children:[u.jsxs("div",{className:`status-indicator ${v?"connected":"disconnected"}`,children:[u.jsx(a0,{connected:v}),u.jsx("span",{children:v?"Connected":"Disconnected"})]}),u.jsxs("div",{className:"status-meta",children:[u.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),u.jsxs("span",{className:"agent-count",children:[m.length," agents"]}),u.jsx("button",{className:"launch-agent-btn",onClick:()=>w(!0),children:"+ Agent"})]})]}),m.length>0&&u.jsx("div",{className:"agents-bar",children:m.map(E=>u.jsxs("div",{className:"agent-chip",children:[u.jsx("span",{className:"agent-pulse"}),u.jsx("span",{className:"agent-name",children:E.instance_id}),u.jsxs("span",{className:"agent-pid",children:["PID ",E.pid]}),u.jsx("button",{className:"agent-stop-btn",onClick:()=>I(E.instance_id),title:"Stop agent",children:"×"})]},E.instance_id))}),k&&u.jsx("div",{className:"modal-overlay",onClick:()=>w(!1),children:u.jsxs("div",{className:"modal-content",onClick:E=>E.stopPropagation(),children:[u.jsx("h3",{children:"Launch New Agent"}),u.jsx("input",{type:"text",value:M,onChange:E=>h(E.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:E=>{E.key==="Enter"&&j(),E.key==="Escape"&&w(!1)}}),u.jsxs("div",{className:"modal-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>w(!1),children:"Cancel"}),u.jsx("button",{className:"launch-btn",onClick:j,children:"Launch"})]})]})}),u.jsxs("div",{className:"center-layout",children:[u.jsx("aside",{className:"threads-panel",children:u.jsx(gg,{threads:i,selectedThreadId:o,onSelectThread:L,onCreateThread:T,onDeleteThread:Q,onRenameThread:$,unreadCounts:d})}),u.jsx("main",{className:"conversation-panel",children:o?u.jsx(l0,{thread:i.find(E=>E.id===o),messages:P,onSendMessage:C,onWorkspaceChange:H,onApproveRequest:N,onRejectRequest:W}):u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("h3",{children:"Select a conversation"}),u.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),u.jsx("style",{children:`
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
      `})]})},Ie={check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),dollar:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:u.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),u.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),u.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},u0=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=B.useState(!0),[a,s]=B.useState(null),[c,d]=B.useState(new Map),p=h=>{try{return JSON.parse(h)}catch{return null}},m=h=>new Date(h).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),f=h=>{const v=c.get(h)||"";n(h,v),d(new Map(c.set(h,"")))},k=h=>{const v=c.get(h)||"";if(!v.trim()){alert("Please provide a reason for rejection");return}r(h,v),d(new Map(c.set(h,"")))},w=(h,v)=>{d(new Map(c.set(h,v)))},M=e.filter(h=>h.status==="pending");return u.jsxs("div",{className:"approval-queue",children:[u.jsx("div",{className:"queue-header",children:u.jsxs("div",{className:"header-title",children:[u.jsx("h2",{children:"Approval Queue"}),u.jsxs("span",{className:"pending-count",children:[M.length," pending"]})]})}),u.jsxs("div",{className:"approvals-container",children:[M.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Ie.sparkles}),u.jsx("h3",{children:"All caught up!"}),u.jsx("p",{children:"No pending approvals to review"})]}):u.jsx("div",{className:"approvals-list",children:M.map(h=>{const v=p(h.effect_delta_json),y=a===h.id;return u.jsxs("div",{className:`approval-card impact-${h.impact}`,children:[u.jsxs("div",{className:"card-header",onClick:()=>s(y?null:h.id),children:[u.jsxs("div",{className:"header-left",children:[u.jsx("div",{className:`impact-indicator ${h.impact}`}),u.jsxs("div",{className:"proposal-info",children:[u.jsx("span",{className:"proposal-text",children:h.proposal}),u.jsxs("div",{className:"proposal-meta",children:[h.thread_title&&u.jsxs("span",{className:"meta-item thread-link",onClick:b=>{b.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Ie.message,h.thread_title]}),u.jsxs("span",{className:"meta-item",children:[Ie.bot,h.instance_id]}),u.jsxs("span",{className:"meta-item",children:[Ie.clock,m(h.created_at)]})]})]})]}),u.jsxs("div",{className:"header-right",children:[u.jsxs("span",{className:"cost-badge",children:[Ie.dollar,"$",h.estimated_cost.toFixed(2)]}),u.jsx("span",{className:`impact-badge ${h.impact}`,children:h.impact}),u.jsx("button",{className:"expand-btn",children:y?Ie.chevronUp:Ie.chevronDown})]})]}),y&&u.jsxs("div",{className:"card-details",children:[v&&u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Effect Details"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Capability"}),u.jsx("span",{className:"detail-value code",children:v.cap_type})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Budget Delta"}),u.jsxs("span",{className:"detail-value",children:["$",v.budget_delta.toFixed(2)]})]}),v.paths.length>0&&u.jsxs("div",{className:"detail-item full-width",children:[u.jsx("span",{className:"detail-label",children:"Paths"}),u.jsx("div",{className:"paths-list",children:v.paths.map((b,_)=>u.jsxs("span",{className:"path-tag",children:[Ie.folder,b]},_))})]})]})]}),u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Request Info"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Thread"}),u.jsx("span",{className:"detail-value code",children:h.thread_id})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Impact Level"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]})]})]}),u.jsxs("div",{className:"review-section",children:[u.jsx("h4",{children:"Review Notes"}),u.jsx("textarea",{value:c.get(h.id)||"",onChange:b=>w(h.id,b.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),u.jsxs("div",{className:"action-buttons",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>k(h.id),children:[Ie.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>f(h.id),children:[Ie.check,"Approve"]})]})]})]})]},h.id)})}),t.length>0&&u.jsxs("div",{className:"history-section",children:[u.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[u.jsxs("h3",{children:[l?Ie.chevronDown:Ie.chevronUp,"Review History"]}),u.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&u.jsx("div",{className:"history-list",children:t.map(h=>{const v=a===`history-${h.id}`;return u.jsxs("div",{className:`history-card ${h.status}`,onClick:()=>s(v?null:`history-${h.id}`),children:[u.jsxs("div",{className:"history-card-header",children:[u.jsxs("div",{className:"history-status",children:[u.jsx("span",{className:`status-icon ${h.status}`,children:h.status==="approved"?Ie.check:Ie.x}),u.jsxs("div",{className:"history-info",children:[u.jsx("span",{className:"history-proposal",children:h.proposal}),h.thread_title&&u.jsxs("span",{className:"history-thread",onClick:y=>{y.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Ie.message,h.thread_title]})]})]}),u.jsxs("div",{className:"history-meta",children:[u.jsx("span",{className:"history-agent",children:h.instance_id}),u.jsx("span",{className:`history-badge ${h.status}`,children:h.status}),u.jsx("span",{className:"history-time",children:h.reviewed_at?m(h.reviewed_at):m(h.created_at)})]})]}),v&&u.jsxs("div",{className:"history-details",children:[u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Reviewed by"}),u.jsx("span",{className:"detail-value",children:h.reviewed_by||"Unknown"})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Cost"}),u.jsxs("span",{className:"detail-value",children:["$",h.estimated_cost.toFixed(2)]})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Impact"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]}),h.review_notes&&u.jsxs("div",{className:"detail-row full-width",children:[u.jsx("span",{className:"detail-label",children:"Notes"}),u.jsx("span",{className:"detail-value notes",children:h.review_notes})]})]})]},`history-${h.id}`)})})]})]}),u.jsx("style",{children:`
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
      `})]})},me={cpu:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"4",y:"4",width:"16",height:"16",rx:"2"}),u.jsx("rect",{x:"9",y:"9",width:"6",height:"6"}),u.jsx("line",{x1:"9",y1:"1",x2:"9",y2:"4"}),u.jsx("line",{x1:"15",y1:"1",x2:"15",y2:"4"}),u.jsx("line",{x1:"9",y1:"20",x2:"9",y2:"23"}),u.jsx("line",{x1:"15",y1:"20",x2:"15",y2:"23"}),u.jsx("line",{x1:"20",y1:"9",x2:"23",y2:"9"}),u.jsx("line",{x1:"20",y1:"14",x2:"23",y2:"14"}),u.jsx("line",{x1:"1",y1:"9",x2:"4",y2:"9"}),u.jsx("line",{x1:"1",y1:"14",x2:"4",y2:"14"})]}),memory:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"2",y:"6",width:"20",height:"12",rx:"2"}),u.jsx("line",{x1:"6",y1:"10",x2:"6",y2:"14"}),u.jsx("line",{x1:"10",y1:"10",x2:"10",y2:"14"}),u.jsx("line",{x1:"14",y1:"10",x2:"14",y2:"14"}),u.jsx("line",{x1:"18",y1:"10",x2:"18",y2:"14"})]}),clock:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),dollar:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),activity:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"22 12 18 12 15 21 9 3 6 12 2 12"})}),tokens:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"}),u.jsx("line",{x1:"10",y1:"9",x2:"8",y2:"9"})]}),message:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),stop:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("rect",{x:"3",y:"3",width:"18",height:"18",rx:"2"})}),warning:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"}),u.jsx("line",{x1:"12",y1:"9",x2:"12",y2:"13"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),check:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})})},c0=()=>{const[e,t]=B.useState(null),[n,r]=B.useState(null),[i,l]=B.useState(null),[o,a]=B.useState(new Map),[s,c]=B.useState({prevCost:0,prevTokensIn:0,prevTokensOut:0,timestamp:0}),d=B.useCallback(async()=>{try{const C=await fetch("/api/monitor");if(!C.ok)throw new Error(`Failed to fetch: ${C.statusText}`);const T=await C.json();t(T),l(new Date),r(null)}catch(C){r(C instanceof Error?C.message:"Unknown error")}},[]);B.useEffect(()=>{d();const C=setInterval(d,2e3);return()=>clearInterval(C)},[d]),B.useEffect(()=>{const T=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;let O=null,j=null;const I=()=>{O=new WebSocket(T),O.onmessage=H=>{try{const Q=JSON.parse(H.data);if(Q.type==="telemetry"){const $=Q.data;a(N=>{const W=new Map(N);return W.set($.instance_id,$),W})}}catch{}},O.onclose=()=>{j=setTimeout(I,3e3)},O.onerror=()=>{O==null||O.close()}};return I(),()=>{j&&clearTimeout(j),O==null||O.close()}},[]);const p=async C=>{try{await fetch(`/api/agents/${C}`,{method:"DELETE"}),d()}catch(T){console.error("Failed to stop process:",T)}},m=C=>{if(C<0)return"Unknown";if(C<60)return`${C}s`;if(C<3600){const j=Math.floor(C/60),I=C%60;return`${j}m ${I}s`}const T=Math.floor(C/3600),O=Math.floor(C%3600/60);return`${T}h ${O}m`},f=C=>C===0?"$0.00":C<.01?`$${C.toFixed(4)}`:`$${C.toFixed(2)}`,k=C=>{switch(C){case"running":return"var(--color-success)";case"completed":return"var(--color-primary)";case"failed":return"var(--color-danger)";case"orphan":return"var(--color-warning)";default:return"var(--text-tertiary)"}},w=C=>C.cpu_percent>80||C.duration_sec>300,M=C=>{const T=o.get(C.instance_id);return T?{...C,turns:T.turns,tokens_in:T.tokens_in,tokens_out:T.tokens_out,cost:T.cost,hasLiveTelemetry:!0}:{...C,hasLiveTelemetry:!1}},h=Array.from(o.values()).reduce((C,T)=>({tokens_in:C.tokens_in+T.tokens_in,tokens_out:C.tokens_out+T.tokens_out,cost:C.cost+T.cost,turns:C.turns+T.turns}),{tokens_in:0,tokens_out:0,cost:0,turns:0}),v=h.cost>0?h.cost:(e==null?void 0:e.summary.total_cost)||0,y={in:h.tokens_in,out:h.tokens_out},b=o.size>0;B.useEffect(()=>{if(b){const C=Date.now();C-s.timestamp>2e3&&c({prevCost:v,prevTokensIn:y.in,prevTokensOut:y.out,timestamp:C})}},[v,y.in,y.out,b,s.timestamp]);const _=v>s.prevCost&&s.prevCost>0?"up":null,S=y.in+y.out>s.prevTokensIn+s.prevTokensOut&&s.prevTokensIn+s.prevTokensOut>0?"up":null,L=C=>C>=1e6?`${(C/1e6).toFixed(1)}M`:C>=1e3?`${(C/1e3).toFixed(1)}K`:C.toString();return u.jsxs("div",{className:"monitor",children:[u.jsxs("div",{className:"monitor-summary",children:[u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:me.activity}),u.jsx("span",{className:"summary-value",children:(e==null?void 0:e.summary.total_processes)||0}),u.jsx("span",{className:"summary-label",children:"Running"})]}),u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:me.cpu}),u.jsxs("span",{className:"summary-value",children:[(e==null?void 0:e.summary.total_cpu_percent.toFixed(1))||"0.0","%"]}),u.jsx("span",{className:"summary-label",children:"CPU"})]}),u.jsxs("div",{className:"summary-item",children:[u.jsx("span",{className:"summary-icon",children:me.memory}),u.jsxs("span",{className:"summary-value",children:[(e==null?void 0:e.summary.total_memory_mb.toFixed(0))||"0"," MB"]}),u.jsx("span",{className:"summary-label",children:"Memory"})]}),u.jsxs("div",{className:`summary-item ${b?"live":""}`,children:[u.jsx("span",{className:"summary-icon",children:me.dollar}),u.jsxs("span",{className:"summary-value",children:[f(v),_==="up"&&u.jsx("span",{className:"trend-up",children:"▲"})]}),u.jsxs("span",{className:"summary-label",children:["Cost ",b&&u.jsx("span",{className:"live-indicator",children:"●"})]})]}),u.jsxs("div",{className:`summary-item ${b?"live":""}`,children:[u.jsx("span",{className:"summary-icon",children:me.tokens}),u.jsxs("span",{className:"summary-value",children:[L(y.in),"↓ / ",L(y.out),"↑",S==="up"&&u.jsx("span",{className:"trend-up",children:"▲"})]}),u.jsxs("span",{className:"summary-label",children:["Tokens ",b&&u.jsx("span",{className:"live-indicator",children:"●"})]})]}),h.turns>0&&u.jsxs("div",{className:"summary-item live",children:[u.jsx("span",{className:"summary-icon",children:me.message}),u.jsx("span",{className:"summary-value",children:h.turns}),u.jsxs("span",{className:"summary-label",children:["Turns ",u.jsx("span",{className:"live-indicator",children:"●"})]})]}),((e==null?void 0:e.summary.warning_count)||0)>0&&u.jsxs("div",{className:"summary-item warning",children:[u.jsx("span",{className:"summary-icon",children:me.warning}),u.jsx("span",{className:"summary-value",children:e==null?void 0:e.summary.warning_count}),u.jsx("span",{className:"summary-label",children:"Warnings"})]}),u.jsx("div",{className:"summary-spacer"}),u.jsxs("div",{className:"summary-update",children:[b&&u.jsx("span",{className:"live-badge-summary",children:"LIVE"}),"Last update: ",i?i.toLocaleTimeString():"Never"]})]}),u.jsxs("div",{className:"process-grid",children:[n&&u.jsxs("div",{className:"error-card",children:[u.jsx("span",{className:"error-icon",children:me.warning}),u.jsx("span",{children:n})]}),(!(e!=null&&e.processes)||e.processes.length===0)&&!n&&u.jsxs("div",{className:"empty-state",children:[u.jsx("span",{className:"empty-icon",children:me.activity}),u.jsx("h3",{children:"No Active Processes"}),u.jsx("p",{children:"Spawn an agent from the Messages tab to see it here."})]}),e==null?void 0:e.processes.map(C=>{const T=M(C);return u.jsxs("div",{className:`process-card ${w(T)?"warning":""} ${T.hasLiveTelemetry?"live":""}`,children:[u.jsxs("div",{className:"process-header",children:[u.jsxs("div",{className:"process-status",children:[u.jsx("span",{className:"status-dot",style:{background:k(T.status)}}),u.jsx("span",{className:"process-name",children:T.instance_id}),T.hasLiveTelemetry&&u.jsx("span",{className:"live-badge",children:"LIVE"})]}),T.status==="running"&&u.jsx("button",{className:"stop-btn",onClick:()=>p(T.instance_id),title:"Stop process",children:me.stop}),T.status==="completed"&&u.jsxs("span",{className:"status-badge completed",children:[me.check," Done"]})]}),u.jsxs("div",{className:"process-metrics",children:[u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:me.cpu}),u.jsxs("span",{className:`metric-value ${T.cpu_percent>80?"high":""}`,children:[T.cpu_percent.toFixed(1),"%"]}),u.jsx("span",{className:"metric-label",children:"CPU"})]}),u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:me.memory}),u.jsxs("span",{className:"metric-value",children:[T.memory_mb.toFixed(0)," MB"]}),u.jsx("span",{className:"metric-label",children:"Memory"})]}),u.jsxs("div",{className:"metric",children:[u.jsx("span",{className:"metric-icon",children:me.clock}),u.jsx("span",{className:`metric-value ${T.duration_sec>300?"high":""}`,children:m(T.duration_sec)}),u.jsx("span",{className:"metric-label",children:"Duration"})]})]}),T.hasLiveTelemetry&&u.jsxs("div",{className:"process-telemetry",children:[u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:me.message}),u.jsx("span",{className:"telemetry-value",children:T.turns||0}),u.jsx("span",{className:"telemetry-label",children:"Turns"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:me.tokens}),u.jsx("span",{className:"telemetry-value",children:L(T.tokens_in||0)}),u.jsx("span",{className:"telemetry-label",children:"In"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:me.tokens}),u.jsx("span",{className:"telemetry-value",children:L(T.tokens_out||0)}),u.jsx("span",{className:"telemetry-label",children:"Out"})]}),u.jsxs("div",{className:"telemetry-item",children:[u.jsx("span",{className:"telemetry-icon",children:me.dollar}),u.jsx("span",{className:"telemetry-value cost",children:f(T.cost||0)}),u.jsx("span",{className:"telemetry-label",children:"Cost"})]})]}),u.jsxs("div",{className:"process-footer",children:[u.jsxs("span",{className:"process-pid",children:["PID: ",T.pid]}),T.source&&u.jsx("span",{className:`source-badge ${T.source}`,children:T.source}),T.command&&u.jsx("span",{className:"process-command",title:T.full_cmd,children:T.command}),!T.hasLiveTelemetry&&T.turns&&u.jsxs("span",{className:"process-turns",children:[T.turns," turns"]}),!T.hasLiveTelemetry&&T.cost!==void 0&&T.cost>0&&u.jsx("span",{className:"process-cost",children:f(T.cost)})]})]},T.instance_id)}),(e==null?void 0:e.history)&&e.history.length>0&&u.jsxs(u.Fragment,{children:[u.jsx("div",{className:"history-divider",children:u.jsx("span",{children:"Recent History"})}),e.history.map(C=>u.jsxs("div",{className:`process-card history ${C.status==="failed"?"failed":""}`,children:[u.jsx("div",{className:"process-header",children:u.jsxs("div",{className:"process-status",children:[u.jsx("span",{className:"status-dot",style:{background:k(C.status)}}),u.jsx("span",{className:"process-name",children:C.instance_id}),u.jsxs("span",{className:`status-badge ${C.status}`,children:[C.status==="completed"?me.check:me.warning,C.status]})]})}),u.jsxs("div",{className:"process-footer",children:[u.jsxs("span",{className:"process-pid",children:["PID: ",C.pid]}),C.source&&u.jsx("span",{className:`source-badge ${C.source}`,children:C.source}),C.command&&u.jsx("span",{className:"process-command",title:C.full_cmd,children:C.command}),u.jsx("span",{className:"process-duration",children:m(C.duration_sec)}),C.cost!==void 0&&C.cost>0&&u.jsx("span",{className:"process-cost",children:f(C.cost)})]})]},`history-${C.instance_id}-${C.stopped_at}`))]})]}),u.jsx("style",{children:`
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
      `})]})},_i={messages:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),shield:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"})}),activity:u.jsx("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"22 12 18 12 15 21 9 3 6 12 2 12"})}),logo:u.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M12 6v12M6 12h12"}),u.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]})},d0=()=>{const[e,t]=B.useState("messages"),[n,r]=B.useState([]),[i,l]=B.useState([]),[o,a]=B.useState("my-agent"),[s,c]=B.useState([]),[d,p]=B.useState(""),[m,f]=B.useState(!1),[k,w]=B.useState(null),h=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;zt.useEffect(()=>{const j=async()=>{try{const H=await fetch("/api/agents");if(H.ok){const Q=await H.json();c(Q),Q.length>0&&!o&&a(Q[0].id)}}catch(H){console.error("Error fetching agents:",H)}};j();const I=setInterval(j,1e4);return()=>clearInterval(I)},[]);const v=j=>{const I=j.target.value;I==="__custom__"?f(!0):(a(I),f(!1))},y=()=>{d.trim()&&(a(d.trim()),f(!1),p(""))},b=j=>j.last_active?Date.now()-j.last_active<3e4:!1,_=j=>b(j)?"●":"○",S=async(j,I)=>{try{const H=await fetch(`/api/approvals/${j}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:I})});if(!H.ok){console.error("Failed to approve:",await H.text());return}const Q=n.find($=>$.id===j);if(Q){const $={...Q,status:"approved",reviewed_by:"user",review_notes:I,reviewed_at:Date.now()};l(N=>[$,...N])}r($=>$.filter(N=>N.id!==j))}catch(H){console.error("Error approving:",H)}},L=async(j,I)=>{try{const H=await fetch(`/api/approvals/${j}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:I})});if(!H.ok){console.error("Failed to reject:",await H.text());return}const Q=n.find($=>$.id===j);if(Q){const $={...Q,status:"rejected",reviewed_by:"user",review_notes:I,reviewed_at:Date.now()};l(N=>[$,...N])}r($=>$.filter(N=>N.id!==j))}catch(H){console.error("Error rejecting:",H)}};zt.useEffect(()=>{const j=async()=>{try{const H=await fetch("/api/approvals?status=pending");if(H.ok){const W=await H.json();r(W)}const[Q,$]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),N=[];if(Q.ok){const W=await Q.json();N.push(...W)}if($.ok){const W=await $.json();N.push(...W)}N.sort((W,P)=>{const E=W.reviewed_at?new Date(W.reviewed_at).getTime():0;return(P.reviewed_at?new Date(P.reviewed_at).getTime():0)-E}),l(N)}catch(H){console.error("Error fetching approvals:",H)}};j();const I=setInterval(j,5e3);return()=>clearInterval(I)},[]);const C=(n==null?void 0:n.filter(j=>j.status==="pending").length)||0,T=j=>{w(j),t("messages")},O=()=>{w(null)};return u.jsxs("div",{className:"app",children:[u.jsxs("header",{className:"app-header",children:[u.jsxs("div",{className:"header-brand",children:[u.jsx("div",{className:"brand-logo",children:_i.logo}),u.jsxs("div",{className:"brand-text",children:[u.jsx("h1",{children:"AILANG"}),u.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),u.jsxs("nav",{className:"header-nav",children:[u.jsxs("button",{className:`nav-tab ${e==="messages"?"active":""}`,onClick:()=>t("messages"),children:[u.jsx("span",{className:"nav-icon",children:_i.messages}),u.jsx("span",{className:"nav-label",children:"Messages"})]}),u.jsxs("button",{className:`nav-tab ${e==="approvals"?"active":""}`,onClick:()=>t("approvals"),children:[u.jsx("span",{className:"nav-icon",children:_i.shield}),u.jsx("span",{className:"nav-label",children:"Approvals"}),C>0&&u.jsx("span",{className:"nav-badge",children:C})]}),u.jsxs("button",{className:`nav-tab ${e==="monitor"?"active":""}`,onClick:()=>t("monitor"),children:[u.jsx("span",{className:"nav-icon",children:_i.activity}),u.jsx("span",{className:"nav-label",children:"Monitor"})]})]}),u.jsxs("div",{className:"header-meta",children:[u.jsxs("div",{className:"agent-selector",children:[u.jsx("label",{className:"agent-label",children:"Target:"}),m?u.jsxs("div",{className:"custom-agent-input",children:[u.jsx("input",{type:"text",value:d,onChange:j=>p(j.target.value),onKeyDown:j=>j.key==="Enter"&&y(),className:"agent-input",placeholder:"agent-id",autoFocus:!0}),u.jsx("button",{onClick:y,className:"agent-apply",children:"Add"}),u.jsx("button",{onClick:()=>f(!1),className:"agent-cancel",children:"Cancel"})]}):u.jsxs(u.Fragment,{children:[u.jsxs("select",{value:o,onChange:v,className:"agent-select",children:[s.map(j=>u.jsxs("option",{value:j.id,children:[_(j)," ",j.id]},j.id)),!s.find(j=>j.id===o)&&o&&u.jsxs("option",{value:o,children:["○ ",o]}),u.jsx("option",{value:"__custom__",children:"+ Add custom..."})]}),s.find(j=>j.id===o)&&u.jsx("span",{className:`agent-status ${b(s.find(j=>j.id===o))?"active":"inactive"}`,children:b(s.find(j=>j.id===o))?"Online":"Offline"})]})]}),u.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),u.jsxs("main",{className:"app-content",children:[e==="messages"&&u.jsx(s0,{websocketUrl:h,instanceId:o,initialThreadId:k,onThreadNavigated:O}),e==="approvals"&&u.jsx(u0,{approvals:n,history:i,onApprove:S,onReject:L,onNavigateToThread:T}),e==="monitor"&&u.jsx(c0,{})]}),u.jsx("style",{children:`
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
      `})]})};wo.createRoot(document.getElementById("root")).render(u.jsx(zt.StrictMode,{children:u.jsx(d0,{})}));
