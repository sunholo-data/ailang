(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var qi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Ta(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Qc={exports:{}},kl={},qc={exports:{}},Y={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var li=Symbol.for("react.element"),Kp=Symbol.for("react.portal"),Yp=Symbol.for("react.fragment"),Xp=Symbol.for("react.strict_mode"),Gp=Symbol.for("react.profiler"),Jp=Symbol.for("react.provider"),Zp=Symbol.for("react.context"),eh=Symbol.for("react.forward_ref"),th=Symbol.for("react.suspense"),nh=Symbol.for("react.memo"),rh=Symbol.for("react.lazy"),Ws=Symbol.iterator;function ih(e){return e===null||typeof e!="object"?null:(e=Ws&&e[Ws]||e["@@iterator"],typeof e=="function"?e:null)}var Kc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Yc=Object.assign,Xc={};function sr(e,t,n){this.props=e,this.context=t,this.refs=Xc,this.updater=n||Kc}sr.prototype.isReactComponent={};sr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};sr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function Gc(){}Gc.prototype=sr.prototype;function La(e,t,n){this.props=e,this.context=t,this.refs=Xc,this.updater=n||Kc}var Pa=La.prototype=new Gc;Pa.constructor=La;Yc(Pa,sr.prototype);Pa.isPureReactComponent=!0;var Qs=Array.isArray,Jc=Object.prototype.hasOwnProperty,Ia={current:null},Zc={key:!0,ref:!0,__self:!0,__source:!0};function ed(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)Jc.call(t,r)&&!Zc.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),c=0;c<a;c++)s[c]=arguments[c+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:li,type:e,key:l,ref:o,props:i,_owner:Ia.current}}function lh(e,t){return{$$typeof:li,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function Aa(e){return typeof e=="object"&&e!==null&&e.$$typeof===li}function oh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var qs=/\/+/g;function Bl(e,t){return typeof e=="object"&&e!==null&&e.key!=null?oh(""+e.key):t.toString(36)}function Ii(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case li:case Kp:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Bl(o,0):r,Qs(i)?(n="",e!=null&&(n=e.replace(qs,"$&/")+"/"),Ii(i,t,n,"",function(c){return c})):i!=null&&(Aa(i)&&(i=lh(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(qs,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Qs(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Bl(l,a);o+=Ii(l,t,n,s,i)}else if(s=ih(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Bl(l,a++),o+=Ii(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function fi(e,t,n){if(e==null)return e;var r=[],i=0;return Ii(e,r,"","",function(l){return t.call(n,l,i++)}),r}function ah(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Fe={current:null},Ai={transition:null},sh={ReactCurrentDispatcher:Fe,ReactCurrentBatchConfig:Ai,ReactCurrentOwner:Ia};function td(){throw Error("act(...) is not supported in production builds of React.")}Y.Children={map:fi,forEach:function(e,t,n){fi(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return fi(e,function(){t++}),t},toArray:function(e){return fi(e,function(t){return t})||[]},only:function(e){if(!Aa(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};Y.Component=sr;Y.Fragment=Yp;Y.Profiler=Gp;Y.PureComponent=La;Y.StrictMode=Xp;Y.Suspense=th;Y.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=sh;Y.act=td;Y.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Yc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Ia.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)Jc.call(t,s)&&!Zc.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var c=0;c<s;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:li,type:e.type,key:i,ref:l,props:r,_owner:o}};Y.createContext=function(e){return e={$$typeof:Zp,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:Jp,_context:e},e.Consumer=e};Y.createElement=ed;Y.createFactory=function(e){var t=ed.bind(null,e);return t.type=e,t};Y.createRef=function(){return{current:null}};Y.forwardRef=function(e){return{$$typeof:eh,render:e}};Y.isValidElement=Aa;Y.lazy=function(e){return{$$typeof:rh,_payload:{_status:-1,_result:e},_init:ah}};Y.memo=function(e,t){return{$$typeof:nh,type:e,compare:t===void 0?null:t}};Y.startTransition=function(e){var t=Ai.transition;Ai.transition={};try{e()}finally{Ai.transition=t}};Y.unstable_act=td;Y.useCallback=function(e,t){return Fe.current.useCallback(e,t)};Y.useContext=function(e){return Fe.current.useContext(e)};Y.useDebugValue=function(){};Y.useDeferredValue=function(e){return Fe.current.useDeferredValue(e)};Y.useEffect=function(e,t){return Fe.current.useEffect(e,t)};Y.useId=function(){return Fe.current.useId()};Y.useImperativeHandle=function(e,t,n){return Fe.current.useImperativeHandle(e,t,n)};Y.useInsertionEffect=function(e,t){return Fe.current.useInsertionEffect(e,t)};Y.useLayoutEffect=function(e,t){return Fe.current.useLayoutEffect(e,t)};Y.useMemo=function(e,t){return Fe.current.useMemo(e,t)};Y.useReducer=function(e,t,n){return Fe.current.useReducer(e,t,n)};Y.useRef=function(e){return Fe.current.useRef(e)};Y.useState=function(e){return Fe.current.useState(e)};Y.useSyncExternalStore=function(e,t,n){return Fe.current.useSyncExternalStore(e,t,n)};Y.useTransition=function(){return Fe.current.useTransition()};Y.version="18.3.1";qc.exports=Y;var O=qc.exports;const Qt=Ta(O);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var uh=O,ch=Symbol.for("react.element"),dh=Symbol.for("react.fragment"),fh=Object.prototype.hasOwnProperty,ph=uh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,hh={key:!0,ref:!0,__self:!0,__source:!0};function nd(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)fh.call(t,r)&&!hh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:ch,type:e,key:l,ref:o,props:i,_owner:ph.current}}kl.Fragment=dh;kl.jsx=nd;kl.jsxs=nd;Qc.exports=kl;var u=Qc.exports,Eo={},rd={exports:{}},rt={},id={exports:{}},ld={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(C,E){var g=C.length;C.push(E);e:for(;0<g;){var L=g-1>>>1,$=C[L];if(0<i($,E))C[L]=E,C[g]=$,g=L;else break e}}function n(C){return C.length===0?null:C[0]}function r(C){if(C.length===0)return null;var E=C[0],g=C.pop();if(g!==E){C[0]=g;e:for(var L=0,$=C.length,x=$>>>1;L<x;){var ne=2*(L+1)-1,be=C[ne],te=ne+1,Ae=C[te];if(0>i(be,g))te<$&&0>i(Ae,be)?(C[L]=Ae,C[te]=g,L=te):(C[L]=be,C[ne]=g,L=ne);else if(te<$&&0>i(Ae,g))C[L]=Ae,C[te]=g,L=te;else break e}}return E}function i(C,E){var g=C.sortIndex-E.sortIndex;return g!==0?g:C.id-E.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],c=[],d=1,f=null,h=3,p=!1,w=!1,S=!1,I=typeof setTimeout=="function"?setTimeout:null,m=typeof clearTimeout=="function"?clearTimeout:null,v=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(C){for(var E=n(c);E!==null;){if(E.callback===null)r(c);else if(E.startTime<=C)r(c),E.sortIndex=E.expirationTime,t(s,E);else break;E=n(c)}}function b(C){if(S=!1,y(C),!w)if(n(s)!==null)w=!0,Q(j);else{var E=n(c);E!==null&&ie(b,E.startTime-C)}}function j(C,E){w=!1,S&&(S=!1,m(T),T=-1),p=!0;var g=h;try{for(y(E),f=n(s);f!==null&&(!(f.expirationTime>E)||C&&!_());){var L=f.callback;if(typeof L=="function"){f.callback=null,h=f.priorityLevel;var $=L(f.expirationTime<=E);E=e.unstable_now(),typeof $=="function"?f.callback=$:f===n(s)&&r(s),y(E)}else r(s);f=n(s)}if(f!==null)var x=!0;else{var ne=n(c);ne!==null&&ie(b,ne.startTime-E),x=!1}return x}finally{f=null,h=g,p=!1}}var k=!1,N=null,T=-1,R=5,P=-1;function _(){return!(e.unstable_now()-P<R)}function D(){if(N!==null){var C=e.unstable_now();P=C;var E=!0;try{E=N(!0,C)}finally{E?W():(k=!1,N=null)}}else k=!1}var W;if(typeof v=="function")W=function(){v(D)};else if(typeof MessageChannel<"u"){var X=new MessageChannel,U=X.port2;X.port1.onmessage=D,W=function(){U.postMessage(null)}}else W=function(){I(D,0)};function Q(C){N=C,k||(k=!0,W())}function ie(C,E){T=I(function(){C(e.unstable_now())},E)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(C){C.callback=null},e.unstable_continueExecution=function(){w||p||(w=!0,Q(j))},e.unstable_forceFrameRate=function(C){0>C||125<C?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):R=0<C?Math.floor(1e3/C):5},e.unstable_getCurrentPriorityLevel=function(){return h},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(C){switch(h){case 1:case 2:case 3:var E=3;break;default:E=h}var g=h;h=E;try{return C()}finally{h=g}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(C,E){switch(C){case 1:case 2:case 3:case 4:case 5:break;default:C=3}var g=h;h=C;try{return E()}finally{h=g}},e.unstable_scheduleCallback=function(C,E,g){var L=e.unstable_now();switch(typeof g=="object"&&g!==null?(g=g.delay,g=typeof g=="number"&&0<g?L+g:L):g=L,C){case 1:var $=-1;break;case 2:$=250;break;case 5:$=1073741823;break;case 4:$=1e4;break;default:$=5e3}return $=g+$,C={id:d++,callback:E,priorityLevel:C,startTime:g,expirationTime:$,sortIndex:-1},g>L?(C.sortIndex=g,t(c,C),n(s)===null&&C===n(c)&&(S?(m(T),T=-1):S=!0,ie(b,g-L))):(C.sortIndex=$,t(s,C),w||p||(w=!0,Q(j))),C},e.unstable_shouldYield=_,e.unstable_wrapCallback=function(C){var E=h;return function(){var g=h;h=E;try{return C.apply(this,arguments)}finally{h=g}}}})(ld);id.exports=ld;var mh=id.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var gh=O,nt=mh;function A(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var od=new Set,$r={};function _n(e,t){tr(e,t),tr(e+"Capture",t)}function tr(e,t){for($r[e]=t,e=0;e<t.length;e++)od.add(t[e])}var Rt=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),No=Object.prototype.hasOwnProperty,vh=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,Ks={},Ys={};function yh(e){return No.call(Ys,e)?!0:No.call(Ks,e)?!1:vh.test(e)?Ys[e]=!0:(Ks[e]=!0,!1)}function xh(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function kh(e,t,n,r){if(t===null||typeof t>"u"||xh(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Oe(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ne={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ne[e]=new Oe(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ne[t]=new Oe(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ne[e]=new Oe(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ne[e]=new Oe(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ne[e]=new Oe(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ne[e]=new Oe(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ne[e]=new Oe(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ne[e]=new Oe(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ne[e]=new Oe(e,5,!1,e.toLowerCase(),null,!1,!1)});var Ma=/[\-:]([a-z])/g;function Da(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(Ma,Da);Ne[t]=new Oe(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(Ma,Da);Ne[t]=new Oe(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(Ma,Da);Ne[t]=new Oe(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ne[e]=new Oe(e,1,!1,e.toLowerCase(),null,!1,!1)});Ne.xlinkHref=new Oe("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ne[e]=new Oe(e,1,!1,e.toLowerCase(),null,!0,!0)});function Ra(e,t,n,r){var i=Ne.hasOwnProperty(t)?Ne[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(kh(t,n,i,r)&&(n=null),r||i===null?yh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var $t=gh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,pi=Symbol.for("react.element"),Mn=Symbol.for("react.portal"),Dn=Symbol.for("react.fragment"),Fa=Symbol.for("react.strict_mode"),_o=Symbol.for("react.profiler"),ad=Symbol.for("react.provider"),sd=Symbol.for("react.context"),Oa=Symbol.for("react.forward_ref"),zo=Symbol.for("react.suspense"),To=Symbol.for("react.suspense_list"),Ba=Symbol.for("react.memo"),qt=Symbol.for("react.lazy"),ud=Symbol.for("react.offscreen"),Xs=Symbol.iterator;function mr(e){return e===null||typeof e!="object"?null:(e=Xs&&e[Xs]||e["@@iterator"],typeof e=="function"?e:null)}var he=Object.assign,$l;function jr(e){if($l===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);$l=t&&t[1]||""}return`
`+$l+e}var Ul=!1;function Hl(e,t){if(!e||Ul)return"";Ul=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Ul=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?jr(e):""}function wh(e){switch(e.tag){case 5:return jr(e.type);case 16:return jr("Lazy");case 13:return jr("Suspense");case 19:return jr("SuspenseList");case 0:case 2:case 15:return e=Hl(e.type,!1),e;case 11:return e=Hl(e.type.render,!1),e;case 1:return e=Hl(e.type,!0),e;default:return""}}function Lo(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Dn:return"Fragment";case Mn:return"Portal";case _o:return"Profiler";case Fa:return"StrictMode";case zo:return"Suspense";case To:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case sd:return(e.displayName||"Context")+".Consumer";case ad:return(e._context.displayName||"Context")+".Provider";case Oa:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Ba:return t=e.displayName||null,t!==null?t:Lo(e.type)||"Memo";case qt:t=e._payload,e=e._init;try{return Lo(e(t))}catch{}}return null}function Sh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Lo(t);case 8:return t===Fa?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function sn(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function cd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function bh(e){var t=cd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function hi(e){e._valueTracker||(e._valueTracker=bh(e))}function dd(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=cd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Ki(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function Po(e,t){var n=t.checked;return he({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function Gs(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=sn(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function fd(e,t){t=t.checked,t!=null&&Ra(e,"checked",t,!1)}function Io(e,t){fd(e,t);var n=sn(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?Ao(e,t.type,n):t.hasOwnProperty("defaultValue")&&Ao(e,t.type,sn(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function Js(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function Ao(e,t,n){(t!=="number"||Ki(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var Er=Array.isArray;function qn(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+sn(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function Mo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(A(91));return he({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function Zs(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(A(92));if(Er(n)){if(1<n.length)throw Error(A(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:sn(n)}}function pd(e,t){var n=sn(t.value),r=sn(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function eu(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function hd(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Do(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?hd(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var mi,md=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(mi=mi||document.createElement("div"),mi.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=mi.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Ur(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var zr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},Ch=["Webkit","ms","Moz","O"];Object.keys(zr).forEach(function(e){Ch.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),zr[t]=zr[e]})});function gd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||zr.hasOwnProperty(e)&&zr[e]?(""+t).trim():t+"px"}function vd(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=gd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var jh=he({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Ro(e,t){if(t){if(jh[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(A(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(A(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(A(61))}if(t.style!=null&&typeof t.style!="object")throw Error(A(62))}}function Fo(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Oo=null;function $a(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Bo=null,Kn=null,Yn=null;function tu(e){if(e=si(e)){if(typeof Bo!="function")throw Error(A(280));var t=e.stateNode;t&&(t=jl(t),Bo(e.stateNode,e.type,t))}}function yd(e){Kn?Yn?Yn.push(e):Yn=[e]:Kn=e}function xd(){if(Kn){var e=Kn,t=Yn;if(Yn=Kn=null,tu(e),t)for(e=0;e<t.length;e++)tu(t[e])}}function kd(e,t){return e(t)}function wd(){}var Vl=!1;function Sd(e,t,n){if(Vl)return e(t,n);Vl=!0;try{return kd(e,t,n)}finally{Vl=!1,(Kn!==null||Yn!==null)&&(wd(),xd())}}function Hr(e,t){var n=e.stateNode;if(n===null)return null;var r=jl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(A(231,t,typeof n));return n}var $o=!1;if(Rt)try{var gr={};Object.defineProperty(gr,"passive",{get:function(){$o=!0}}),window.addEventListener("test",gr,gr),window.removeEventListener("test",gr,gr)}catch{$o=!1}function Eh(e,t,n,r,i,l,o,a,s){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Tr=!1,Yi=null,Xi=!1,Uo=null,Nh={onError:function(e){Tr=!0,Yi=e}};function _h(e,t,n,r,i,l,o,a,s){Tr=!1,Yi=null,Eh.apply(Nh,arguments)}function zh(e,t,n,r,i,l,o,a,s){if(_h.apply(this,arguments),Tr){if(Tr){var c=Yi;Tr=!1,Yi=null}else throw Error(A(198));Xi||(Xi=!0,Uo=c)}}function zn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function bd(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function nu(e){if(zn(e)!==e)throw Error(A(188))}function Th(e){var t=e.alternate;if(!t){if(t=zn(e),t===null)throw Error(A(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return nu(i),e;if(l===r)return nu(i),t;l=l.sibling}throw Error(A(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(A(189))}}if(n.alternate!==r)throw Error(A(190))}if(n.tag!==3)throw Error(A(188));return n.stateNode.current===n?e:t}function Cd(e){return e=Th(e),e!==null?jd(e):null}function jd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=jd(e);if(t!==null)return t;e=e.sibling}return null}var Ed=nt.unstable_scheduleCallback,ru=nt.unstable_cancelCallback,Lh=nt.unstable_shouldYield,Ph=nt.unstable_requestPaint,ge=nt.unstable_now,Ih=nt.unstable_getCurrentPriorityLevel,Ua=nt.unstable_ImmediatePriority,Nd=nt.unstable_UserBlockingPriority,Gi=nt.unstable_NormalPriority,Ah=nt.unstable_LowPriority,_d=nt.unstable_IdlePriority,wl=null,Et=null;function Mh(e){if(Et&&typeof Et.onCommitFiberRoot=="function")try{Et.onCommitFiberRoot(wl,e,void 0,(e.current.flags&128)===128)}catch{}}var yt=Math.clz32?Math.clz32:Fh,Dh=Math.log,Rh=Math.LN2;function Fh(e){return e>>>=0,e===0?32:31-(Dh(e)/Rh|0)|0}var gi=64,vi=4194304;function Nr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function Ji(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=Nr(a):(l&=o,l!==0&&(r=Nr(l)))}else o=n&~i,o!==0?r=Nr(o):l!==0&&(r=Nr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-yt(t),i=1<<n,r|=e[n],t&=~i;return r}function Oh(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Bh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-yt(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Oh(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Ho(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function zd(){var e=gi;return gi<<=1,!(gi&4194240)&&(gi=64),e}function Wl(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function oi(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-yt(t),e[t]=n}function $h(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-yt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Ha(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-yt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var re=0;function Td(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Ld,Va,Pd,Id,Ad,Vo=!1,yi=[],Zt=null,en=null,tn=null,Vr=new Map,Wr=new Map,Yt=[],Uh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function iu(e,t){switch(e){case"focusin":case"focusout":Zt=null;break;case"dragenter":case"dragleave":en=null;break;case"mouseover":case"mouseout":tn=null;break;case"pointerover":case"pointerout":Vr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Wr.delete(t.pointerId)}}function vr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=si(t),t!==null&&Va(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Hh(e,t,n,r,i){switch(t){case"focusin":return Zt=vr(Zt,e,t,n,r,i),!0;case"dragenter":return en=vr(en,e,t,n,r,i),!0;case"mouseover":return tn=vr(tn,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Vr.set(l,vr(Vr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Wr.set(l,vr(Wr.get(l)||null,e,t,n,r,i)),!0}return!1}function Md(e){var t=yn(e.target);if(t!==null){var n=zn(t);if(n!==null){if(t=n.tag,t===13){if(t=bd(n),t!==null){e.blockedOn=t,Ad(e.priority,function(){Pd(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Mi(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Wo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Oo=r,n.target.dispatchEvent(r),Oo=null}else return t=si(n),t!==null&&Va(t),e.blockedOn=n,!1;t.shift()}return!0}function lu(e,t,n){Mi(e)&&n.delete(t)}function Vh(){Vo=!1,Zt!==null&&Mi(Zt)&&(Zt=null),en!==null&&Mi(en)&&(en=null),tn!==null&&Mi(tn)&&(tn=null),Vr.forEach(lu),Wr.forEach(lu)}function yr(e,t){e.blockedOn===t&&(e.blockedOn=null,Vo||(Vo=!0,nt.unstable_scheduleCallback(nt.unstable_NormalPriority,Vh)))}function Qr(e){function t(i){return yr(i,e)}if(0<yi.length){yr(yi[0],e);for(var n=1;n<yi.length;n++){var r=yi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(Zt!==null&&yr(Zt,e),en!==null&&yr(en,e),tn!==null&&yr(tn,e),Vr.forEach(t),Wr.forEach(t),n=0;n<Yt.length;n++)r=Yt[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Yt.length&&(n=Yt[0],n.blockedOn===null);)Md(n),n.blockedOn===null&&Yt.shift()}var Xn=$t.ReactCurrentBatchConfig,Zi=!0;function Wh(e,t,n,r){var i=re,l=Xn.transition;Xn.transition=null;try{re=1,Wa(e,t,n,r)}finally{re=i,Xn.transition=l}}function Qh(e,t,n,r){var i=re,l=Xn.transition;Xn.transition=null;try{re=4,Wa(e,t,n,r)}finally{re=i,Xn.transition=l}}function Wa(e,t,n,r){if(Zi){var i=Wo(e,t,n,r);if(i===null)to(e,t,r,el,n),iu(e,r);else if(Hh(i,e,t,n,r))r.stopPropagation();else if(iu(e,r),t&4&&-1<Uh.indexOf(e)){for(;i!==null;){var l=si(i);if(l!==null&&Ld(l),l=Wo(e,t,n,r),l===null&&to(e,t,r,el,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else to(e,t,r,null,n)}}var el=null;function Wo(e,t,n,r){if(el=null,e=$a(r),e=yn(e),e!==null)if(t=zn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=bd(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return el=e,null}function Dd(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(Ih()){case Ua:return 1;case Nd:return 4;case Gi:case Ah:return 16;case _d:return 536870912;default:return 16}default:return 16}}var Gt=null,Qa=null,Di=null;function Rd(){if(Di)return Di;var e,t=Qa,n=t.length,r,i="value"in Gt?Gt.value:Gt.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Di=i.slice(e,1<r?1-r:void 0)}function Ri(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function xi(){return!0}function ou(){return!1}function it(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?xi:ou,this.isPropagationStopped=ou,this}return he(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=xi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=xi)},persist:function(){},isPersistent:xi}),t}var ur={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},qa=it(ur),ai=he({},ur,{view:0,detail:0}),qh=it(ai),Ql,ql,xr,Sl=he({},ai,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:Ka,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==xr&&(xr&&e.type==="mousemove"?(Ql=e.screenX-xr.screenX,ql=e.screenY-xr.screenY):ql=Ql=0,xr=e),Ql)},movementY:function(e){return"movementY"in e?e.movementY:ql}}),au=it(Sl),Kh=he({},Sl,{dataTransfer:0}),Yh=it(Kh),Xh=he({},ai,{relatedTarget:0}),Kl=it(Xh),Gh=he({},ur,{animationName:0,elapsedTime:0,pseudoElement:0}),Jh=it(Gh),Zh=he({},ur,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),em=it(Zh),tm=he({},ur,{data:0}),su=it(tm),nm={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},rm={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},im={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function lm(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=im[e])?!!t[e]:!1}function Ka(){return lm}var om=he({},ai,{key:function(e){if(e.key){var t=nm[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Ri(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?rm[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:Ka,charCode:function(e){return e.type==="keypress"?Ri(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Ri(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),am=it(om),sm=he({},Sl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),uu=it(sm),um=he({},ai,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:Ka}),cm=it(um),dm=he({},ur,{propertyName:0,elapsedTime:0,pseudoElement:0}),fm=it(dm),pm=he({},Sl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),hm=it(pm),mm=[9,13,27,32],Ya=Rt&&"CompositionEvent"in window,Lr=null;Rt&&"documentMode"in document&&(Lr=document.documentMode);var gm=Rt&&"TextEvent"in window&&!Lr,Fd=Rt&&(!Ya||Lr&&8<Lr&&11>=Lr),cu=" ",du=!1;function Od(e,t){switch(e){case"keyup":return mm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Bd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Rn=!1;function vm(e,t){switch(e){case"compositionend":return Bd(t);case"keypress":return t.which!==32?null:(du=!0,cu);case"textInput":return e=t.data,e===cu&&du?null:e;default:return null}}function ym(e,t){if(Rn)return e==="compositionend"||!Ya&&Od(e,t)?(e=Rd(),Di=Qa=Gt=null,Rn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Fd&&t.locale!=="ko"?null:t.data;default:return null}}var xm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function fu(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!xm[e.type]:t==="textarea"}function $d(e,t,n,r){yd(r),t=tl(t,"onChange"),0<t.length&&(n=new qa("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Pr=null,qr=null;function km(e){Jd(e,0)}function bl(e){var t=Bn(e);if(dd(t))return e}function wm(e,t){if(e==="change")return t}var Ud=!1;if(Rt){var Yl;if(Rt){var Xl="oninput"in document;if(!Xl){var pu=document.createElement("div");pu.setAttribute("oninput","return;"),Xl=typeof pu.oninput=="function"}Yl=Xl}else Yl=!1;Ud=Yl&&(!document.documentMode||9<document.documentMode)}function hu(){Pr&&(Pr.detachEvent("onpropertychange",Hd),qr=Pr=null)}function Hd(e){if(e.propertyName==="value"&&bl(qr)){var t=[];$d(t,qr,e,$a(e)),Sd(km,t)}}function Sm(e,t,n){e==="focusin"?(hu(),Pr=t,qr=n,Pr.attachEvent("onpropertychange",Hd)):e==="focusout"&&hu()}function bm(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return bl(qr)}function Cm(e,t){if(e==="click")return bl(t)}function jm(e,t){if(e==="input"||e==="change")return bl(t)}function Em(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var kt=typeof Object.is=="function"?Object.is:Em;function Kr(e,t){if(kt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!No.call(t,i)||!kt(e[i],t[i]))return!1}return!0}function mu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function gu(e,t){var n=mu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=mu(n)}}function Vd(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Vd(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Wd(){for(var e=window,t=Ki();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Ki(e.document)}return t}function Xa(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function Nm(e){var t=Wd(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Vd(n.ownerDocument.documentElement,n)){if(r!==null&&Xa(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=gu(n,l);var o=gu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var _m=Rt&&"documentMode"in document&&11>=document.documentMode,Fn=null,Qo=null,Ir=null,qo=!1;function vu(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;qo||Fn==null||Fn!==Ki(r)||(r=Fn,"selectionStart"in r&&Xa(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Ir&&Kr(Ir,r)||(Ir=r,r=tl(Qo,"onSelect"),0<r.length&&(t=new qa("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Fn)))}function ki(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var On={animationend:ki("Animation","AnimationEnd"),animationiteration:ki("Animation","AnimationIteration"),animationstart:ki("Animation","AnimationStart"),transitionend:ki("Transition","TransitionEnd")},Gl={},Qd={};Rt&&(Qd=document.createElement("div").style,"AnimationEvent"in window||(delete On.animationend.animation,delete On.animationiteration.animation,delete On.animationstart.animation),"TransitionEvent"in window||delete On.transitionend.transition);function Cl(e){if(Gl[e])return Gl[e];if(!On[e])return e;var t=On[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Qd)return Gl[e]=t[n];return e}var qd=Cl("animationend"),Kd=Cl("animationiteration"),Yd=Cl("animationstart"),Xd=Cl("transitionend"),Gd=new Map,yu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function cn(e,t){Gd.set(e,t),_n(t,[e])}for(var Jl=0;Jl<yu.length;Jl++){var Zl=yu[Jl],zm=Zl.toLowerCase(),Tm=Zl[0].toUpperCase()+Zl.slice(1);cn(zm,"on"+Tm)}cn(qd,"onAnimationEnd");cn(Kd,"onAnimationIteration");cn(Yd,"onAnimationStart");cn("dblclick","onDoubleClick");cn("focusin","onFocus");cn("focusout","onBlur");cn(Xd,"onTransitionEnd");tr("onMouseEnter",["mouseout","mouseover"]);tr("onMouseLeave",["mouseout","mouseover"]);tr("onPointerEnter",["pointerout","pointerover"]);tr("onPointerLeave",["pointerout","pointerover"]);_n("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));_n("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));_n("onBeforeInput",["compositionend","keypress","textInput","paste"]);_n("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));_n("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));_n("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var _r="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Lm=new Set("cancel close invalid load scroll toggle".split(" ").concat(_r));function xu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,zh(r,t,void 0,e),e.currentTarget=null}function Jd(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,c=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;xu(i,a,c),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,c=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;xu(i,a,c),l=s}}}if(Xi)throw e=Uo,Xi=!1,Uo=null,e}function ue(e,t){var n=t[Jo];n===void 0&&(n=t[Jo]=new Set);var r=e+"__bubble";n.has(r)||(Zd(t,e,2,!1),n.add(r))}function eo(e,t,n){var r=0;t&&(r|=4),Zd(n,e,r,t)}var wi="_reactListening"+Math.random().toString(36).slice(2);function Yr(e){if(!e[wi]){e[wi]=!0,od.forEach(function(n){n!=="selectionchange"&&(Lm.has(n)||eo(n,!1,e),eo(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[wi]||(t[wi]=!0,eo("selectionchange",!1,t))}}function Zd(e,t,n,r){switch(Dd(t)){case 1:var i=Wh;break;case 4:i=Qh;break;default:i=Wa}n=i.bind(null,t,n,e),i=void 0,!$o||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function to(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=yn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}Sd(function(){var c=l,d=$a(n),f=[];e:{var h=Gd.get(e);if(h!==void 0){var p=qa,w=e;switch(e){case"keypress":if(Ri(n)===0)break e;case"keydown":case"keyup":p=am;break;case"focusin":w="focus",p=Kl;break;case"focusout":w="blur",p=Kl;break;case"beforeblur":case"afterblur":p=Kl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":p=au;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":p=Yh;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":p=cm;break;case qd:case Kd:case Yd:p=Jh;break;case Xd:p=fm;break;case"scroll":p=qh;break;case"wheel":p=hm;break;case"copy":case"cut":case"paste":p=em;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":p=uu}var S=(t&4)!==0,I=!S&&e==="scroll",m=S?h!==null?h+"Capture":null:h;S=[];for(var v=c,y;v!==null;){y=v;var b=y.stateNode;if(y.tag===5&&b!==null&&(y=b,m!==null&&(b=Hr(v,m),b!=null&&S.push(Xr(v,b,y)))),I)break;v=v.return}0<S.length&&(h=new p(h,w,null,n,d),f.push({event:h,listeners:S}))}}if(!(t&7)){e:{if(h=e==="mouseover"||e==="pointerover",p=e==="mouseout"||e==="pointerout",h&&n!==Oo&&(w=n.relatedTarget||n.fromElement)&&(yn(w)||w[Ft]))break e;if((p||h)&&(h=d.window===d?d:(h=d.ownerDocument)?h.defaultView||h.parentWindow:window,p?(w=n.relatedTarget||n.toElement,p=c,w=w?yn(w):null,w!==null&&(I=zn(w),w!==I||w.tag!==5&&w.tag!==6)&&(w=null)):(p=null,w=c),p!==w)){if(S=au,b="onMouseLeave",m="onMouseEnter",v="mouse",(e==="pointerout"||e==="pointerover")&&(S=uu,b="onPointerLeave",m="onPointerEnter",v="pointer"),I=p==null?h:Bn(p),y=w==null?h:Bn(w),h=new S(b,v+"leave",p,n,d),h.target=I,h.relatedTarget=y,b=null,yn(d)===c&&(S=new S(m,v+"enter",w,n,d),S.target=y,S.relatedTarget=I,b=S),I=b,p&&w)t:{for(S=p,m=w,v=0,y=S;y;y=Pn(y))v++;for(y=0,b=m;b;b=Pn(b))y++;for(;0<v-y;)S=Pn(S),v--;for(;0<y-v;)m=Pn(m),y--;for(;v--;){if(S===m||m!==null&&S===m.alternate)break t;S=Pn(S),m=Pn(m)}S=null}else S=null;p!==null&&ku(f,h,p,S,!1),w!==null&&I!==null&&ku(f,I,w,S,!0)}}e:{if(h=c?Bn(c):window,p=h.nodeName&&h.nodeName.toLowerCase(),p==="select"||p==="input"&&h.type==="file")var j=wm;else if(fu(h))if(Ud)j=jm;else{j=bm;var k=Sm}else(p=h.nodeName)&&p.toLowerCase()==="input"&&(h.type==="checkbox"||h.type==="radio")&&(j=Cm);if(j&&(j=j(e,c))){$d(f,j,n,d);break e}k&&k(e,h,c),e==="focusout"&&(k=h._wrapperState)&&k.controlled&&h.type==="number"&&Ao(h,"number",h.value)}switch(k=c?Bn(c):window,e){case"focusin":(fu(k)||k.contentEditable==="true")&&(Fn=k,Qo=c,Ir=null);break;case"focusout":Ir=Qo=Fn=null;break;case"mousedown":qo=!0;break;case"contextmenu":case"mouseup":case"dragend":qo=!1,vu(f,n,d);break;case"selectionchange":if(_m)break;case"keydown":case"keyup":vu(f,n,d)}var N;if(Ya)e:{switch(e){case"compositionstart":var T="onCompositionStart";break e;case"compositionend":T="onCompositionEnd";break e;case"compositionupdate":T="onCompositionUpdate";break e}T=void 0}else Rn?Od(e,n)&&(T="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(T="onCompositionStart");T&&(Fd&&n.locale!=="ko"&&(Rn||T!=="onCompositionStart"?T==="onCompositionEnd"&&Rn&&(N=Rd()):(Gt=d,Qa="value"in Gt?Gt.value:Gt.textContent,Rn=!0)),k=tl(c,T),0<k.length&&(T=new su(T,e,null,n,d),f.push({event:T,listeners:k}),N?T.data=N:(N=Bd(n),N!==null&&(T.data=N)))),(N=gm?vm(e,n):ym(e,n))&&(c=tl(c,"onBeforeInput"),0<c.length&&(d=new su("onBeforeInput","beforeinput",null,n,d),f.push({event:d,listeners:c}),d.data=N))}Jd(f,t)})}function Xr(e,t,n){return{instance:e,listener:t,currentTarget:n}}function tl(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Hr(e,n),l!=null&&r.unshift(Xr(e,l,i)),l=Hr(e,t),l!=null&&r.push(Xr(e,l,i))),e=e.return}return r}function Pn(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function ku(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,c=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&c!==null&&(a=c,i?(s=Hr(n,l),s!=null&&o.unshift(Xr(n,s,a))):i||(s=Hr(n,l),s!=null&&o.push(Xr(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Pm=/\r\n?/g,Im=/\u0000|\uFFFD/g;function wu(e){return(typeof e=="string"?e:""+e).replace(Pm,`
`).replace(Im,"")}function Si(e,t,n){if(t=wu(t),wu(e)!==t&&n)throw Error(A(425))}function nl(){}var Ko=null,Yo=null;function Xo(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var Go=typeof setTimeout=="function"?setTimeout:void 0,Am=typeof clearTimeout=="function"?clearTimeout:void 0,Su=typeof Promise=="function"?Promise:void 0,Mm=typeof queueMicrotask=="function"?queueMicrotask:typeof Su<"u"?function(e){return Su.resolve(null).then(e).catch(Dm)}:Go;function Dm(e){setTimeout(function(){throw e})}function no(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Qr(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Qr(t)}function nn(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function bu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var cr=Math.random().toString(36).slice(2),Ct="__reactFiber$"+cr,Gr="__reactProps$"+cr,Ft="__reactContainer$"+cr,Jo="__reactEvents$"+cr,Rm="__reactListeners$"+cr,Fm="__reactHandles$"+cr;function yn(e){var t=e[Ct];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Ft]||n[Ct]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=bu(e);e!==null;){if(n=e[Ct])return n;e=bu(e)}return t}e=n,n=e.parentNode}return null}function si(e){return e=e[Ct]||e[Ft],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function Bn(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(A(33))}function jl(e){return e[Gr]||null}var Zo=[],$n=-1;function dn(e){return{current:e}}function ce(e){0>$n||(e.current=Zo[$n],Zo[$n]=null,$n--)}function ae(e,t){$n++,Zo[$n]=e.current,e.current=t}var un={},Pe=dn(un),Ve=dn(!1),bn=un;function nr(e,t){var n=e.type.contextTypes;if(!n)return un;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function We(e){return e=e.childContextTypes,e!=null}function rl(){ce(Ve),ce(Pe)}function Cu(e,t,n){if(Pe.current!==un)throw Error(A(168));ae(Pe,t),ae(Ve,n)}function ef(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(A(108,Sh(e)||"Unknown",i));return he({},n,r)}function il(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||un,bn=Pe.current,ae(Pe,e),ae(Ve,Ve.current),!0}function ju(e,t,n){var r=e.stateNode;if(!r)throw Error(A(169));n?(e=ef(e,t,bn),r.__reactInternalMemoizedMergedChildContext=e,ce(Ve),ce(Pe),ae(Pe,e)):ce(Ve),ae(Ve,n)}var It=null,El=!1,ro=!1;function tf(e){It===null?It=[e]:It.push(e)}function Om(e){El=!0,tf(e)}function fn(){if(!ro&&It!==null){ro=!0;var e=0,t=re;try{var n=It;for(re=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}It=null,El=!1}catch(i){throw It!==null&&(It=It.slice(e+1)),Ed(Ua,fn),i}finally{re=t,ro=!1}}return null}var Un=[],Hn=0,ll=null,ol=0,ot=[],at=0,Cn=null,At=1,Mt="";function mn(e,t){Un[Hn++]=ol,Un[Hn++]=ll,ll=e,ol=t}function nf(e,t,n){ot[at++]=At,ot[at++]=Mt,ot[at++]=Cn,Cn=e;var r=At;e=Mt;var i=32-yt(r)-1;r&=~(1<<i),n+=1;var l=32-yt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,At=1<<32-yt(t)+i|n<<i|r,Mt=l+e}else At=1<<l|n<<i|r,Mt=e}function Ga(e){e.return!==null&&(mn(e,1),nf(e,1,0))}function Ja(e){for(;e===ll;)ll=Un[--Hn],Un[Hn]=null,ol=Un[--Hn],Un[Hn]=null;for(;e===Cn;)Cn=ot[--at],ot[at]=null,Mt=ot[--at],ot[at]=null,At=ot[--at],ot[at]=null}var tt=null,Ze=null,de=!1,vt=null;function rf(e,t){var n=ut(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function Eu(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,tt=e,Ze=nn(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,tt=e,Ze=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=Cn!==null?{id:At,overflow:Mt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=ut(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,tt=e,Ze=null,!0):!1;default:return!1}}function ea(e){return(e.mode&1)!==0&&(e.flags&128)===0}function ta(e){if(de){var t=Ze;if(t){var n=t;if(!Eu(e,t)){if(ea(e))throw Error(A(418));t=nn(n.nextSibling);var r=tt;t&&Eu(e,t)?rf(r,n):(e.flags=e.flags&-4097|2,de=!1,tt=e)}}else{if(ea(e))throw Error(A(418));e.flags=e.flags&-4097|2,de=!1,tt=e}}}function Nu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;tt=e}function bi(e){if(e!==tt)return!1;if(!de)return Nu(e),de=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!Xo(e.type,e.memoizedProps)),t&&(t=Ze)){if(ea(e))throw lf(),Error(A(418));for(;t;)rf(e,t),t=nn(t.nextSibling)}if(Nu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(A(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){Ze=nn(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}Ze=null}}else Ze=tt?nn(e.stateNode.nextSibling):null;return!0}function lf(){for(var e=Ze;e;)e=nn(e.nextSibling)}function rr(){Ze=tt=null,de=!1}function Za(e){vt===null?vt=[e]:vt.push(e)}var Bm=$t.ReactCurrentBatchConfig;function kr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(A(309));var r=n.stateNode}if(!r)throw Error(A(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(A(284));if(!n._owner)throw Error(A(290,e))}return e}function Ci(e,t){throw e=Object.prototype.toString.call(t),Error(A(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function _u(e){var t=e._init;return t(e._payload)}function of(e){function t(m,v){if(e){var y=m.deletions;y===null?(m.deletions=[v],m.flags|=16):y.push(v)}}function n(m,v){if(!e)return null;for(;v!==null;)t(m,v),v=v.sibling;return null}function r(m,v){for(m=new Map;v!==null;)v.key!==null?m.set(v.key,v):m.set(v.index,v),v=v.sibling;return m}function i(m,v){return m=an(m,v),m.index=0,m.sibling=null,m}function l(m,v,y){return m.index=y,e?(y=m.alternate,y!==null?(y=y.index,y<v?(m.flags|=2,v):y):(m.flags|=2,v)):(m.flags|=1048576,v)}function o(m){return e&&m.alternate===null&&(m.flags|=2),m}function a(m,v,y,b){return v===null||v.tag!==6?(v=co(y,m.mode,b),v.return=m,v):(v=i(v,y),v.return=m,v)}function s(m,v,y,b){var j=y.type;return j===Dn?d(m,v,y.props.children,b,y.key):v!==null&&(v.elementType===j||typeof j=="object"&&j!==null&&j.$$typeof===qt&&_u(j)===v.type)?(b=i(v,y.props),b.ref=kr(m,v,y),b.return=m,b):(b=Vi(y.type,y.key,y.props,null,m.mode,b),b.ref=kr(m,v,y),b.return=m,b)}function c(m,v,y,b){return v===null||v.tag!==4||v.stateNode.containerInfo!==y.containerInfo||v.stateNode.implementation!==y.implementation?(v=fo(y,m.mode,b),v.return=m,v):(v=i(v,y.children||[]),v.return=m,v)}function d(m,v,y,b,j){return v===null||v.tag!==7?(v=Sn(y,m.mode,b,j),v.return=m,v):(v=i(v,y),v.return=m,v)}function f(m,v,y){if(typeof v=="string"&&v!==""||typeof v=="number")return v=co(""+v,m.mode,y),v.return=m,v;if(typeof v=="object"&&v!==null){switch(v.$$typeof){case pi:return y=Vi(v.type,v.key,v.props,null,m.mode,y),y.ref=kr(m,null,v),y.return=m,y;case Mn:return v=fo(v,m.mode,y),v.return=m,v;case qt:var b=v._init;return f(m,b(v._payload),y)}if(Er(v)||mr(v))return v=Sn(v,m.mode,y,null),v.return=m,v;Ci(m,v)}return null}function h(m,v,y,b){var j=v!==null?v.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return j!==null?null:a(m,v,""+y,b);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case pi:return y.key===j?s(m,v,y,b):null;case Mn:return y.key===j?c(m,v,y,b):null;case qt:return j=y._init,h(m,v,j(y._payload),b)}if(Er(y)||mr(y))return j!==null?null:d(m,v,y,b,null);Ci(m,y)}return null}function p(m,v,y,b,j){if(typeof b=="string"&&b!==""||typeof b=="number")return m=m.get(y)||null,a(v,m,""+b,j);if(typeof b=="object"&&b!==null){switch(b.$$typeof){case pi:return m=m.get(b.key===null?y:b.key)||null,s(v,m,b,j);case Mn:return m=m.get(b.key===null?y:b.key)||null,c(v,m,b,j);case qt:var k=b._init;return p(m,v,y,k(b._payload),j)}if(Er(b)||mr(b))return m=m.get(y)||null,d(v,m,b,j,null);Ci(v,b)}return null}function w(m,v,y,b){for(var j=null,k=null,N=v,T=v=0,R=null;N!==null&&T<y.length;T++){N.index>T?(R=N,N=null):R=N.sibling;var P=h(m,N,y[T],b);if(P===null){N===null&&(N=R);break}e&&N&&P.alternate===null&&t(m,N),v=l(P,v,T),k===null?j=P:k.sibling=P,k=P,N=R}if(T===y.length)return n(m,N),de&&mn(m,T),j;if(N===null){for(;T<y.length;T++)N=f(m,y[T],b),N!==null&&(v=l(N,v,T),k===null?j=N:k.sibling=N,k=N);return de&&mn(m,T),j}for(N=r(m,N);T<y.length;T++)R=p(N,m,T,y[T],b),R!==null&&(e&&R.alternate!==null&&N.delete(R.key===null?T:R.key),v=l(R,v,T),k===null?j=R:k.sibling=R,k=R);return e&&N.forEach(function(_){return t(m,_)}),de&&mn(m,T),j}function S(m,v,y,b){var j=mr(y);if(typeof j!="function")throw Error(A(150));if(y=j.call(y),y==null)throw Error(A(151));for(var k=j=null,N=v,T=v=0,R=null,P=y.next();N!==null&&!P.done;T++,P=y.next()){N.index>T?(R=N,N=null):R=N.sibling;var _=h(m,N,P.value,b);if(_===null){N===null&&(N=R);break}e&&N&&_.alternate===null&&t(m,N),v=l(_,v,T),k===null?j=_:k.sibling=_,k=_,N=R}if(P.done)return n(m,N),de&&mn(m,T),j;if(N===null){for(;!P.done;T++,P=y.next())P=f(m,P.value,b),P!==null&&(v=l(P,v,T),k===null?j=P:k.sibling=P,k=P);return de&&mn(m,T),j}for(N=r(m,N);!P.done;T++,P=y.next())P=p(N,m,T,P.value,b),P!==null&&(e&&P.alternate!==null&&N.delete(P.key===null?T:P.key),v=l(P,v,T),k===null?j=P:k.sibling=P,k=P);return e&&N.forEach(function(D){return t(m,D)}),de&&mn(m,T),j}function I(m,v,y,b){if(typeof y=="object"&&y!==null&&y.type===Dn&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case pi:e:{for(var j=y.key,k=v;k!==null;){if(k.key===j){if(j=y.type,j===Dn){if(k.tag===7){n(m,k.sibling),v=i(k,y.props.children),v.return=m,m=v;break e}}else if(k.elementType===j||typeof j=="object"&&j!==null&&j.$$typeof===qt&&_u(j)===k.type){n(m,k.sibling),v=i(k,y.props),v.ref=kr(m,k,y),v.return=m,m=v;break e}n(m,k);break}else t(m,k);k=k.sibling}y.type===Dn?(v=Sn(y.props.children,m.mode,b,y.key),v.return=m,m=v):(b=Vi(y.type,y.key,y.props,null,m.mode,b),b.ref=kr(m,v,y),b.return=m,m=b)}return o(m);case Mn:e:{for(k=y.key;v!==null;){if(v.key===k)if(v.tag===4&&v.stateNode.containerInfo===y.containerInfo&&v.stateNode.implementation===y.implementation){n(m,v.sibling),v=i(v,y.children||[]),v.return=m,m=v;break e}else{n(m,v);break}else t(m,v);v=v.sibling}v=fo(y,m.mode,b),v.return=m,m=v}return o(m);case qt:return k=y._init,I(m,v,k(y._payload),b)}if(Er(y))return w(m,v,y,b);if(mr(y))return S(m,v,y,b);Ci(m,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,v!==null&&v.tag===6?(n(m,v.sibling),v=i(v,y),v.return=m,m=v):(n(m,v),v=co(y,m.mode,b),v.return=m,m=v),o(m)):n(m,v)}return I}var ir=of(!0),af=of(!1),al=dn(null),sl=null,Vn=null,es=null;function ts(){es=Vn=sl=null}function ns(e){var t=al.current;ce(al),e._currentValue=t}function na(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Gn(e,t){sl=e,es=Vn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(He=!0),e.firstContext=null)}function dt(e){var t=e._currentValue;if(es!==e)if(e={context:e,memoizedValue:t,next:null},Vn===null){if(sl===null)throw Error(A(308));Vn=e,sl.dependencies={lanes:0,firstContext:e}}else Vn=Vn.next=e;return t}var xn=null;function rs(e){xn===null?xn=[e]:xn.push(e)}function sf(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,rs(t)):(n.next=i.next,i.next=n),t.interleaved=n,Ot(e,r)}function Ot(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Kt=!1;function is(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function uf(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Dt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function rn(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,Z&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Ot(e,n)}return i=r.interleaved,i===null?(t.next=t,rs(r)):(t.next=i.next,i.next=t),r.interleaved=t,Ot(e,n)}function Fi(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ha(e,n)}}function zu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function ul(e,t,n,r){var i=e.updateQueue;Kt=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,c=s.next;s.next=null,o===null?l=c:o.next=c,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=s))}if(l!==null){var f=i.baseState;o=0,d=c=s=null,a=l;do{var h=a.lane,p=a.eventTime;if((r&h)===h){d!==null&&(d=d.next={eventTime:p,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var w=e,S=a;switch(h=t,p=n,S.tag){case 1:if(w=S.payload,typeof w=="function"){f=w.call(p,f,h);break e}f=w;break e;case 3:w.flags=w.flags&-65537|128;case 0:if(w=S.payload,h=typeof w=="function"?w.call(p,f,h):w,h==null)break e;f=he({},f,h);break e;case 2:Kt=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,h=i.effects,h===null?i.effects=[a]:h.push(a))}else p={eventTime:p,lane:h,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=p,s=f):d=d.next=p,o|=h;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;h=a,a=h.next,h.next=null,i.lastBaseUpdate=h,i.shared.pending=null}}while(!0);if(d===null&&(s=f),i.baseState=s,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);En|=o,e.lanes=o,e.memoizedState=f}}function Tu(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(A(191,i));i.call(r)}}}var ui={},Nt=dn(ui),Jr=dn(ui),Zr=dn(ui);function kn(e){if(e===ui)throw Error(A(174));return e}function ls(e,t){switch(ae(Zr,t),ae(Jr,e),ae(Nt,ui),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Do(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Do(t,e)}ce(Nt),ae(Nt,t)}function lr(){ce(Nt),ce(Jr),ce(Zr)}function cf(e){kn(Zr.current);var t=kn(Nt.current),n=Do(t,e.type);t!==n&&(ae(Jr,e),ae(Nt,n))}function os(e){Jr.current===e&&(ce(Nt),ce(Jr))}var fe=dn(0);function cl(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var io=[];function as(){for(var e=0;e<io.length;e++)io[e]._workInProgressVersionPrimary=null;io.length=0}var Oi=$t.ReactCurrentDispatcher,lo=$t.ReactCurrentBatchConfig,jn=0,pe=null,xe=null,we=null,dl=!1,Ar=!1,ei=0,$m=0;function _e(){throw Error(A(321))}function ss(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!kt(e[n],t[n]))return!1;return!0}function us(e,t,n,r,i,l){if(jn=l,pe=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Oi.current=e===null||e.memoizedState===null?Wm:Qm,e=n(r,i),Ar){l=0;do{if(Ar=!1,ei=0,25<=l)throw Error(A(301));l+=1,we=xe=null,t.updateQueue=null,Oi.current=qm,e=n(r,i)}while(Ar)}if(Oi.current=fl,t=xe!==null&&xe.next!==null,jn=0,we=xe=pe=null,dl=!1,t)throw Error(A(300));return e}function cs(){var e=ei!==0;return ei=0,e}function St(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return we===null?pe.memoizedState=we=e:we=we.next=e,we}function ft(){if(xe===null){var e=pe.alternate;e=e!==null?e.memoizedState:null}else e=xe.next;var t=we===null?pe.memoizedState:we.next;if(t!==null)we=t,xe=e;else{if(e===null)throw Error(A(310));xe=e,e={memoizedState:xe.memoizedState,baseState:xe.baseState,baseQueue:xe.baseQueue,queue:xe.queue,next:null},we===null?pe.memoizedState=we=e:we=we.next=e}return we}function ti(e,t){return typeof t=="function"?t(e):t}function oo(e){var t=ft(),n=t.queue;if(n===null)throw Error(A(311));n.lastRenderedReducer=e;var r=xe,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,c=l;do{var d=c.lane;if((jn&d)===d)s!==null&&(s=s.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var f={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};s===null?(a=s=f,o=r):s=s.next=f,pe.lanes|=d,En|=d}c=c.next}while(c!==null&&c!==l);s===null?o=r:s.next=a,kt(r,t.memoizedState)||(He=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,pe.lanes|=l,En|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function ao(e){var t=ft(),n=t.queue;if(n===null)throw Error(A(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);kt(l,t.memoizedState)||(He=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function df(){}function ff(e,t){var n=pe,r=ft(),i=t(),l=!kt(r.memoizedState,i);if(l&&(r.memoizedState=i,He=!0),r=r.queue,ds(mf.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||we!==null&&we.memoizedState.tag&1){if(n.flags|=2048,ni(9,hf.bind(null,n,r,i,t),void 0,null),Se===null)throw Error(A(349));jn&30||pf(n,t,i)}return i}function pf(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=pe.updateQueue,t===null?(t={lastEffect:null,stores:null},pe.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function hf(e,t,n,r){t.value=n,t.getSnapshot=r,gf(t)&&vf(e)}function mf(e,t,n){return n(function(){gf(t)&&vf(e)})}function gf(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!kt(e,n)}catch{return!0}}function vf(e){var t=Ot(e,1);t!==null&&xt(t,e,1,-1)}function Lu(e){var t=St();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:ti,lastRenderedState:e},t.queue=e,e=e.dispatch=Vm.bind(null,pe,e),[t.memoizedState,e]}function ni(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=pe.updateQueue,t===null?(t={lastEffect:null,stores:null},pe.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function yf(){return ft().memoizedState}function Bi(e,t,n,r){var i=St();pe.flags|=e,i.memoizedState=ni(1|t,n,void 0,r===void 0?null:r)}function Nl(e,t,n,r){var i=ft();r=r===void 0?null:r;var l=void 0;if(xe!==null){var o=xe.memoizedState;if(l=o.destroy,r!==null&&ss(r,o.deps)){i.memoizedState=ni(t,n,l,r);return}}pe.flags|=e,i.memoizedState=ni(1|t,n,l,r)}function Pu(e,t){return Bi(8390656,8,e,t)}function ds(e,t){return Nl(2048,8,e,t)}function xf(e,t){return Nl(4,2,e,t)}function kf(e,t){return Nl(4,4,e,t)}function wf(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function Sf(e,t,n){return n=n!=null?n.concat([e]):null,Nl(4,4,wf.bind(null,t,e),n)}function fs(){}function bf(e,t){var n=ft();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ss(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function Cf(e,t){var n=ft();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ss(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function jf(e,t,n){return jn&21?(kt(n,t)||(n=zd(),pe.lanes|=n,En|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,He=!0),e.memoizedState=n)}function Um(e,t){var n=re;re=n!==0&&4>n?n:4,e(!0);var r=lo.transition;lo.transition={};try{e(!1),t()}finally{re=n,lo.transition=r}}function Ef(){return ft().memoizedState}function Hm(e,t,n){var r=on(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},Nf(e))_f(t,n);else if(n=sf(e,t,n,r),n!==null){var i=Re();xt(n,e,r,i),zf(n,t,r)}}function Vm(e,t,n){var r=on(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(Nf(e))_f(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,kt(a,o)){var s=t.interleaved;s===null?(i.next=i,rs(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=sf(e,t,i,r),n!==null&&(i=Re(),xt(n,e,r,i),zf(n,t,r))}}function Nf(e){var t=e.alternate;return e===pe||t!==null&&t===pe}function _f(e,t){Ar=dl=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function zf(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Ha(e,n)}}var fl={readContext:dt,useCallback:_e,useContext:_e,useEffect:_e,useImperativeHandle:_e,useInsertionEffect:_e,useLayoutEffect:_e,useMemo:_e,useReducer:_e,useRef:_e,useState:_e,useDebugValue:_e,useDeferredValue:_e,useTransition:_e,useMutableSource:_e,useSyncExternalStore:_e,useId:_e,unstable_isNewReconciler:!1},Wm={readContext:dt,useCallback:function(e,t){return St().memoizedState=[e,t===void 0?null:t],e},useContext:dt,useEffect:Pu,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Bi(4194308,4,wf.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Bi(4194308,4,e,t)},useInsertionEffect:function(e,t){return Bi(4,2,e,t)},useMemo:function(e,t){var n=St();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=St();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Hm.bind(null,pe,e),[r.memoizedState,e]},useRef:function(e){var t=St();return e={current:e},t.memoizedState=e},useState:Lu,useDebugValue:fs,useDeferredValue:function(e){return St().memoizedState=e},useTransition:function(){var e=Lu(!1),t=e[0];return e=Um.bind(null,e[1]),St().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=pe,i=St();if(de){if(n===void 0)throw Error(A(407));n=n()}else{if(n=t(),Se===null)throw Error(A(349));jn&30||pf(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Pu(mf.bind(null,r,l,e),[e]),r.flags|=2048,ni(9,hf.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=St(),t=Se.identifierPrefix;if(de){var n=Mt,r=At;n=(r&~(1<<32-yt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=ei++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=$m++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},Qm={readContext:dt,useCallback:bf,useContext:dt,useEffect:ds,useImperativeHandle:Sf,useInsertionEffect:xf,useLayoutEffect:kf,useMemo:Cf,useReducer:oo,useRef:yf,useState:function(){return oo(ti)},useDebugValue:fs,useDeferredValue:function(e){var t=ft();return jf(t,xe.memoizedState,e)},useTransition:function(){var e=oo(ti)[0],t=ft().memoizedState;return[e,t]},useMutableSource:df,useSyncExternalStore:ff,useId:Ef,unstable_isNewReconciler:!1},qm={readContext:dt,useCallback:bf,useContext:dt,useEffect:ds,useImperativeHandle:Sf,useInsertionEffect:xf,useLayoutEffect:kf,useMemo:Cf,useReducer:ao,useRef:yf,useState:function(){return ao(ti)},useDebugValue:fs,useDeferredValue:function(e){var t=ft();return xe===null?t.memoizedState=e:jf(t,xe.memoizedState,e)},useTransition:function(){var e=ao(ti)[0],t=ft().memoizedState;return[e,t]},useMutableSource:df,useSyncExternalStore:ff,useId:Ef,unstable_isNewReconciler:!1};function mt(e,t){if(e&&e.defaultProps){t=he({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function ra(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:he({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var _l={isMounted:function(e){return(e=e._reactInternals)?zn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Re(),i=on(e),l=Dt(r,i);l.payload=t,n!=null&&(l.callback=n),t=rn(e,l,i),t!==null&&(xt(t,e,i,r),Fi(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Re(),i=on(e),l=Dt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=rn(e,l,i),t!==null&&(xt(t,e,i,r),Fi(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Re(),r=on(e),i=Dt(n,r);i.tag=2,t!=null&&(i.callback=t),t=rn(e,i,r),t!==null&&(xt(t,e,r,n),Fi(t,e,r))}};function Iu(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Kr(n,r)||!Kr(i,l):!0}function Tf(e,t,n){var r=!1,i=un,l=t.contextType;return typeof l=="object"&&l!==null?l=dt(l):(i=We(t)?bn:Pe.current,r=t.contextTypes,l=(r=r!=null)?nr(e,i):un),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=_l,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function Au(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&_l.enqueueReplaceState(t,t.state,null)}function ia(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},is(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=dt(l):(l=We(t)?bn:Pe.current,i.context=nr(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(ra(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&_l.enqueueReplaceState(i,i.state,null),ul(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function or(e,t){try{var n="",r=t;do n+=wh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function so(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function la(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var Km=typeof WeakMap=="function"?WeakMap:Map;function Lf(e,t,n){n=Dt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){hl||(hl=!0,ma=r),la(e,t)},n}function Pf(e,t,n){n=Dt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){la(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){la(e,t),typeof r!="function"&&(ln===null?ln=new Set([this]):ln.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function Mu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new Km;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=sg.bind(null,e,t,n),t.then(e,e))}function Du(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Ru(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Dt(-1,1),t.tag=2,rn(n,t,1))),n.lanes|=1),e)}var Ym=$t.ReactCurrentOwner,He=!1;function De(e,t,n,r){t.child=e===null?af(t,null,n,r):ir(t,e.child,n,r)}function Fu(e,t,n,r,i){n=n.render;var l=t.ref;return Gn(t,i),r=us(e,t,n,r,l,i),n=cs(),e!==null&&!He?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Bt(e,t,i)):(de&&n&&Ga(t),t.flags|=1,De(e,t,r,i),t.child)}function Ou(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!ks(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,If(e,t,l,r,i)):(e=Vi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Kr,n(o,r)&&e.ref===t.ref)return Bt(e,t,i)}return t.flags|=1,e=an(l,r),e.ref=t.ref,e.return=t,t.child=e}function If(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Kr(l,r)&&e.ref===t.ref)if(He=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(He=!0);else return t.lanes=e.lanes,Bt(e,t,i)}return oa(e,t,n,r,i)}function Af(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},ae(Qn,Je),Je|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,ae(Qn,Je),Je|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,ae(Qn,Je),Je|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,ae(Qn,Je),Je|=r;return De(e,t,i,n),t.child}function Mf(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function oa(e,t,n,r,i){var l=We(n)?bn:Pe.current;return l=nr(t,l),Gn(t,i),n=us(e,t,n,r,l,i),r=cs(),e!==null&&!He?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Bt(e,t,i)):(de&&r&&Ga(t),t.flags|=1,De(e,t,n,i),t.child)}function Bu(e,t,n,r,i){if(We(n)){var l=!0;il(t)}else l=!1;if(Gn(t,i),t.stateNode===null)$i(e,t),Tf(t,n,r),ia(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=dt(c):(c=We(n)?bn:Pe.current,c=nr(t,c));var d=n.getDerivedStateFromProps,f=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";f||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==c)&&Au(t,o,r,c),Kt=!1;var h=t.memoizedState;o.state=h,ul(t,r,o,i),s=t.memoizedState,a!==r||h!==s||Ve.current||Kt?(typeof d=="function"&&(ra(t,n,d,r),s=t.memoizedState),(a=Kt||Iu(t,n,a,r,h,s,c))?(f||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,uf(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:mt(t.type,a),o.props=c,f=t.pendingProps,h=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=dt(s):(s=We(n)?bn:Pe.current,s=nr(t,s));var p=n.getDerivedStateFromProps;(d=typeof p=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==f||h!==s)&&Au(t,o,r,s),Kt=!1,h=t.memoizedState,o.state=h,ul(t,r,o,i);var w=t.memoizedState;a!==f||h!==w||Ve.current||Kt?(typeof p=="function"&&(ra(t,n,p,r),w=t.memoizedState),(c=Kt||Iu(t,n,c,r,h,w,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,w,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,w,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&h===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&h===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=w),o.props=r,o.state=w,o.context=s,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&h===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&h===e.memoizedState||(t.flags|=1024),r=!1)}return aa(e,t,n,r,l,i)}function aa(e,t,n,r,i,l){Mf(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&ju(t,n,!1),Bt(e,t,l);r=t.stateNode,Ym.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=ir(t,e.child,null,l),t.child=ir(t,null,a,l)):De(e,t,a,l),t.memoizedState=r.state,i&&ju(t,n,!0),t.child}function Df(e){var t=e.stateNode;t.pendingContext?Cu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&Cu(e,t.context,!1),ls(e,t.containerInfo)}function $u(e,t,n,r,i){return rr(),Za(i),t.flags|=256,De(e,t,n,r),t.child}var sa={dehydrated:null,treeContext:null,retryLane:0};function ua(e){return{baseLanes:e,cachePool:null,transitions:null}}function Rf(e,t,n){var r=t.pendingProps,i=fe.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),ae(fe,i&1),e===null)return ta(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Ll(o,r,0,null),e=Sn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=ua(n),t.memoizedState=sa,e):ps(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return Xm(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=an(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=an(a,l):(l=Sn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?ua(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=sa,r}return l=e.child,e=l.sibling,r=an(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function ps(e,t){return t=Ll({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function ji(e,t,n,r){return r!==null&&Za(r),ir(t,e.child,null,n),e=ps(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function Xm(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=so(Error(A(422))),ji(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Ll({mode:"visible",children:r.children},i,0,null),l=Sn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&ir(t,e.child,null,o),t.child.memoizedState=ua(o),t.memoizedState=sa,l);if(!(t.mode&1))return ji(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(A(419)),r=so(l,r,void 0),ji(e,t,o,r)}if(a=(o&e.childLanes)!==0,He||a){if(r=Se,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Ot(e,i),xt(r,e,i,-1))}return xs(),r=so(Error(A(421))),ji(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=ug.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,Ze=nn(i.nextSibling),tt=t,de=!0,vt=null,e!==null&&(ot[at++]=At,ot[at++]=Mt,ot[at++]=Cn,At=e.id,Mt=e.overflow,Cn=t),t=ps(t,r.children),t.flags|=4096,t)}function Uu(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),na(e.return,t,n)}function uo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Ff(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(De(e,t,r.children,n),r=fe.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Uu(e,n,t);else if(e.tag===19)Uu(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(ae(fe,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&cl(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),uo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&cl(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}uo(t,!0,n,null,l);break;case"together":uo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function $i(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Bt(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),En|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(A(153));if(t.child!==null){for(e=t.child,n=an(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=an(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function Gm(e,t,n){switch(t.tag){case 3:Df(t),rr();break;case 5:cf(t);break;case 1:We(t.type)&&il(t);break;case 4:ls(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;ae(al,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(ae(fe,fe.current&1),t.flags|=128,null):n&t.child.childLanes?Rf(e,t,n):(ae(fe,fe.current&1),e=Bt(e,t,n),e!==null?e.sibling:null);ae(fe,fe.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Ff(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),ae(fe,fe.current),r)break;return null;case 22:case 23:return t.lanes=0,Af(e,t,n)}return Bt(e,t,n)}var Of,ca,Bf,$f;Of=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};ca=function(){};Bf=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,kn(Nt.current);var l=null;switch(n){case"input":i=Po(e,i),r=Po(e,r),l=[];break;case"select":i=he({},i,{value:void 0}),r=he({},r,{value:void 0}),l=[];break;case"textarea":i=Mo(e,i),r=Mo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=nl)}Ro(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&($r.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var s=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&s!==a&&(s!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(c,n)),n=s;else c==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(c,s)):c==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(c,""+s):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&($r.hasOwnProperty(c)?(s!=null&&c==="onScroll"&&ue("scroll",e),l||a===s||(l=[])):(l=l||[]).push(c,s))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};$f=function(e,t,n,r){n!==r&&(t.flags|=4)};function wr(e,t){if(!de)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function ze(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function Jm(e,t,n){var r=t.pendingProps;switch(Ja(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return ze(t),null;case 1:return We(t.type)&&rl(),ze(t),null;case 3:return r=t.stateNode,lr(),ce(Ve),ce(Pe),as(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(bi(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,vt!==null&&(ya(vt),vt=null))),ca(e,t),ze(t),null;case 5:os(t);var i=kn(Zr.current);if(n=t.type,e!==null&&t.stateNode!=null)Bf(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(A(166));return ze(t),null}if(e=kn(Nt.current),bi(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[Ct]=t,r[Gr]=l,e=(t.mode&1)!==0,n){case"dialog":ue("cancel",r),ue("close",r);break;case"iframe":case"object":case"embed":ue("load",r);break;case"video":case"audio":for(i=0;i<_r.length;i++)ue(_r[i],r);break;case"source":ue("error",r);break;case"img":case"image":case"link":ue("error",r),ue("load",r);break;case"details":ue("toggle",r);break;case"input":Gs(r,l),ue("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},ue("invalid",r);break;case"textarea":Zs(r,l),ue("invalid",r)}Ro(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&Si(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&Si(r.textContent,a,e),i=["children",""+a]):$r.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&ue("scroll",r)}switch(n){case"input":hi(r),Js(r,l,!0);break;case"textarea":hi(r),eu(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=nl)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=hd(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[Ct]=t,e[Gr]=r,Of(e,t,!1,!1),t.stateNode=e;e:{switch(o=Fo(n,r),n){case"dialog":ue("cancel",e),ue("close",e),i=r;break;case"iframe":case"object":case"embed":ue("load",e),i=r;break;case"video":case"audio":for(i=0;i<_r.length;i++)ue(_r[i],e);i=r;break;case"source":ue("error",e),i=r;break;case"img":case"image":case"link":ue("error",e),ue("load",e),i=r;break;case"details":ue("toggle",e),i=r;break;case"input":Gs(e,r),i=Po(e,r),ue("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=he({},r,{value:void 0}),ue("invalid",e);break;case"textarea":Zs(e,r),i=Mo(e,r),ue("invalid",e);break;default:i=r}Ro(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?vd(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&md(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Ur(e,s):typeof s=="number"&&Ur(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&($r.hasOwnProperty(l)?s!=null&&l==="onScroll"&&ue("scroll",e):s!=null&&Ra(e,l,s,o))}switch(n){case"input":hi(e),Js(e,r,!1);break;case"textarea":hi(e),eu(e);break;case"option":r.value!=null&&e.setAttribute("value",""+sn(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?qn(e,!!r.multiple,l,!1):r.defaultValue!=null&&qn(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=nl)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return ze(t),null;case 6:if(e&&t.stateNode!=null)$f(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(A(166));if(n=kn(Zr.current),kn(Nt.current),bi(t)){if(r=t.stateNode,n=t.memoizedProps,r[Ct]=t,(l=r.nodeValue!==n)&&(e=tt,e!==null))switch(e.tag){case 3:Si(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&Si(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[Ct]=t,t.stateNode=r}return ze(t),null;case 13:if(ce(fe),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(de&&Ze!==null&&t.mode&1&&!(t.flags&128))lf(),rr(),t.flags|=98560,l=!1;else if(l=bi(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(A(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(A(317));l[Ct]=t}else rr(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;ze(t),l=!1}else vt!==null&&(ya(vt),vt=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||fe.current&1?ke===0&&(ke=3):xs())),t.updateQueue!==null&&(t.flags|=4),ze(t),null);case 4:return lr(),ca(e,t),e===null&&Yr(t.stateNode.containerInfo),ze(t),null;case 10:return ns(t.type._context),ze(t),null;case 17:return We(t.type)&&rl(),ze(t),null;case 19:if(ce(fe),l=t.memoizedState,l===null)return ze(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)wr(l,!1);else{if(ke!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=cl(e),o!==null){for(t.flags|=128,wr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return ae(fe,fe.current&1|2),t.child}e=e.sibling}l.tail!==null&&ge()>ar&&(t.flags|=128,r=!0,wr(l,!1),t.lanes=4194304)}else{if(!r)if(e=cl(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),wr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!de)return ze(t),null}else 2*ge()-l.renderingStartTime>ar&&n!==1073741824&&(t.flags|=128,r=!0,wr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=ge(),t.sibling=null,n=fe.current,ae(fe,r?n&1|2:n&1),t):(ze(t),null);case 22:case 23:return ys(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?Je&1073741824&&(ze(t),t.subtreeFlags&6&&(t.flags|=8192)):ze(t),null;case 24:return null;case 25:return null}throw Error(A(156,t.tag))}function Zm(e,t){switch(Ja(t),t.tag){case 1:return We(t.type)&&rl(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return lr(),ce(Ve),ce(Pe),as(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return os(t),null;case 13:if(ce(fe),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(A(340));rr()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return ce(fe),null;case 4:return lr(),null;case 10:return ns(t.type._context),null;case 22:case 23:return ys(),null;case 24:return null;default:return null}}var Ei=!1,Le=!1,eg=typeof WeakSet=="function"?WeakSet:Set,B=null;function Wn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){me(e,t,r)}else n.current=null}function da(e,t,n){try{n()}catch(r){me(e,t,r)}}var Hu=!1;function tg(e,t){if(Ko=Zi,e=Wd(),Xa(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,c=0,d=0,f=e,h=null;t:for(;;){for(var p;f!==n||i!==0&&f.nodeType!==3||(a=o+i),f!==l||r!==0&&f.nodeType!==3||(s=o+r),f.nodeType===3&&(o+=f.nodeValue.length),(p=f.firstChild)!==null;)h=f,f=p;for(;;){if(f===e)break t;if(h===n&&++c===i&&(a=o),h===l&&++d===r&&(s=o),(p=f.nextSibling)!==null)break;f=h,h=f.parentNode}f=p}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Yo={focusedElem:e,selectionRange:n},Zi=!1,B=t;B!==null;)if(t=B,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,B=e;else for(;B!==null;){t=B;try{var w=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(w!==null){var S=w.memoizedProps,I=w.memoizedState,m=t.stateNode,v=m.getSnapshotBeforeUpdate(t.elementType===t.type?S:mt(t.type,S),I);m.__reactInternalSnapshotBeforeUpdate=v}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(A(163))}}catch(b){me(t,t.return,b)}if(e=t.sibling,e!==null){e.return=t.return,B=e;break}B=t.return}return w=Hu,Hu=!1,w}function Mr(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&da(t,n,l)}i=i.next}while(i!==r)}}function zl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function fa(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Uf(e){var t=e.alternate;t!==null&&(e.alternate=null,Uf(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[Ct],delete t[Gr],delete t[Jo],delete t[Rm],delete t[Fm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Hf(e){return e.tag===5||e.tag===3||e.tag===4}function Vu(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Hf(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function pa(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=nl));else if(r!==4&&(e=e.child,e!==null))for(pa(e,t,n),e=e.sibling;e!==null;)pa(e,t,n),e=e.sibling}function ha(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(ha(e,t,n),e=e.sibling;e!==null;)ha(e,t,n),e=e.sibling}var je=null,gt=!1;function Vt(e,t,n){for(n=n.child;n!==null;)Vf(e,t,n),n=n.sibling}function Vf(e,t,n){if(Et&&typeof Et.onCommitFiberUnmount=="function")try{Et.onCommitFiberUnmount(wl,n)}catch{}switch(n.tag){case 5:Le||Wn(n,t);case 6:var r=je,i=gt;je=null,Vt(e,t,n),je=r,gt=i,je!==null&&(gt?(e=je,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):je.removeChild(n.stateNode));break;case 18:je!==null&&(gt?(e=je,n=n.stateNode,e.nodeType===8?no(e.parentNode,n):e.nodeType===1&&no(e,n),Qr(e)):no(je,n.stateNode));break;case 4:r=je,i=gt,je=n.stateNode.containerInfo,gt=!0,Vt(e,t,n),je=r,gt=i;break;case 0:case 11:case 14:case 15:if(!Le&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&da(n,t,o),i=i.next}while(i!==r)}Vt(e,t,n);break;case 1:if(!Le&&(Wn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){me(n,t,a)}Vt(e,t,n);break;case 21:Vt(e,t,n);break;case 22:n.mode&1?(Le=(r=Le)||n.memoizedState!==null,Vt(e,t,n),Le=r):Vt(e,t,n);break;default:Vt(e,t,n)}}function Wu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new eg),t.forEach(function(r){var i=cg.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function ht(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:je=a.stateNode,gt=!1;break e;case 3:je=a.stateNode.containerInfo,gt=!0;break e;case 4:je=a.stateNode.containerInfo,gt=!0;break e}a=a.return}if(je===null)throw Error(A(160));Vf(l,o,i),je=null,gt=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(c){me(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Wf(t,e),t=t.sibling}function Wf(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(ht(t,e),wt(e),r&4){try{Mr(3,e,e.return),zl(3,e)}catch(S){me(e,e.return,S)}try{Mr(5,e,e.return)}catch(S){me(e,e.return,S)}}break;case 1:ht(t,e),wt(e),r&512&&n!==null&&Wn(n,n.return);break;case 5:if(ht(t,e),wt(e),r&512&&n!==null&&Wn(n,n.return),e.flags&32){var i=e.stateNode;try{Ur(i,"")}catch(S){me(e,e.return,S)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&fd(i,l),Fo(a,o);var c=Fo(a,l);for(o=0;o<s.length;o+=2){var d=s[o],f=s[o+1];d==="style"?vd(i,f):d==="dangerouslySetInnerHTML"?md(i,f):d==="children"?Ur(i,f):Ra(i,d,f,c)}switch(a){case"input":Io(i,l);break;case"textarea":pd(i,l);break;case"select":var h=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var p=l.value;p!=null?qn(i,!!l.multiple,p,!1):h!==!!l.multiple&&(l.defaultValue!=null?qn(i,!!l.multiple,l.defaultValue,!0):qn(i,!!l.multiple,l.multiple?[]:"",!1))}i[Gr]=l}catch(S){me(e,e.return,S)}}break;case 6:if(ht(t,e),wt(e),r&4){if(e.stateNode===null)throw Error(A(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(S){me(e,e.return,S)}}break;case 3:if(ht(t,e),wt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Qr(t.containerInfo)}catch(S){me(e,e.return,S)}break;case 4:ht(t,e),wt(e);break;case 13:ht(t,e),wt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(gs=ge())),r&4&&Wu(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Le=(c=Le)||d,ht(t,e),Le=c):ht(t,e),wt(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for(B=e,d=e.child;d!==null;){for(f=B=d;B!==null;){switch(h=B,p=h.child,h.tag){case 0:case 11:case 14:case 15:Mr(4,h,h.return);break;case 1:Wn(h,h.return);var w=h.stateNode;if(typeof w.componentWillUnmount=="function"){r=h,n=h.return;try{t=r,w.props=t.memoizedProps,w.state=t.memoizedState,w.componentWillUnmount()}catch(S){me(r,n,S)}}break;case 5:Wn(h,h.return);break;case 22:if(h.memoizedState!==null){qu(f);continue}}p!==null?(p.return=h,B=p):qu(f)}d=d.sibling}e:for(d=null,f=e;;){if(f.tag===5){if(d===null){d=f;try{i=f.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=f.stateNode,s=f.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=gd("display",o))}catch(S){me(e,e.return,S)}}}else if(f.tag===6){if(d===null)try{f.stateNode.nodeValue=c?"":f.memoizedProps}catch(S){me(e,e.return,S)}}else if((f.tag!==22&&f.tag!==23||f.memoizedState===null||f===e)&&f.child!==null){f.child.return=f,f=f.child;continue}if(f===e)break e;for(;f.sibling===null;){if(f.return===null||f.return===e)break e;d===f&&(d=null),f=f.return}d===f&&(d=null),f.sibling.return=f.return,f=f.sibling}}break;case 19:ht(t,e),wt(e),r&4&&Wu(e);break;case 21:break;default:ht(t,e),wt(e)}}function wt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Hf(n)){var r=n;break e}n=n.return}throw Error(A(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Ur(i,""),r.flags&=-33);var l=Vu(e);ha(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Vu(e);pa(e,a,o);break;default:throw Error(A(161))}}catch(s){me(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function ng(e,t,n){B=e,Qf(e)}function Qf(e,t,n){for(var r=(e.mode&1)!==0;B!==null;){var i=B,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||Ei;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||Le;a=Ei;var c=Le;if(Ei=o,(Le=s)&&!c)for(B=i;B!==null;)o=B,s=o.child,o.tag===22&&o.memoizedState!==null?Ku(i):s!==null?(s.return=o,B=s):Ku(i);for(;l!==null;)B=l,Qf(l),l=l.sibling;B=i,Ei=a,Le=c}Qu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,B=l):Qu(e)}}function Qu(e){for(;B!==null;){var t=B;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Le||zl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Le)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:mt(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Tu(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Tu(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var f=d.dehydrated;f!==null&&Qr(f)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(A(163))}Le||t.flags&512&&fa(t)}catch(h){me(t,t.return,h)}}if(t===e){B=null;break}if(n=t.sibling,n!==null){n.return=t.return,B=n;break}B=t.return}}function qu(e){for(;B!==null;){var t=B;if(t===e){B=null;break}var n=t.sibling;if(n!==null){n.return=t.return,B=n;break}B=t.return}}function Ku(e){for(;B!==null;){var t=B;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{zl(4,t)}catch(s){me(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){me(t,i,s)}}var l=t.return;try{fa(t)}catch(s){me(t,l,s)}break;case 5:var o=t.return;try{fa(t)}catch(s){me(t,o,s)}}}catch(s){me(t,t.return,s)}if(t===e){B=null;break}var a=t.sibling;if(a!==null){a.return=t.return,B=a;break}B=t.return}}var rg=Math.ceil,pl=$t.ReactCurrentDispatcher,hs=$t.ReactCurrentOwner,ct=$t.ReactCurrentBatchConfig,Z=0,Se=null,ye=null,Ee=0,Je=0,Qn=dn(0),ke=0,ri=null,En=0,Tl=0,ms=0,Dr=null,Ue=null,gs=0,ar=1/0,Pt=null,hl=!1,ma=null,ln=null,Ni=!1,Jt=null,ml=0,Rr=0,ga=null,Ui=-1,Hi=0;function Re(){return Z&6?ge():Ui!==-1?Ui:Ui=ge()}function on(e){return e.mode&1?Z&2&&Ee!==0?Ee&-Ee:Bm.transition!==null?(Hi===0&&(Hi=zd()),Hi):(e=re,e!==0||(e=window.event,e=e===void 0?16:Dd(e.type)),e):1}function xt(e,t,n,r){if(50<Rr)throw Rr=0,ga=null,Error(A(185));oi(e,n,r),(!(Z&2)||e!==Se)&&(e===Se&&(!(Z&2)&&(Tl|=n),ke===4&&Xt(e,Ee)),Qe(e,r),n===1&&Z===0&&!(t.mode&1)&&(ar=ge()+500,El&&fn()))}function Qe(e,t){var n=e.callbackNode;Bh(e,t);var r=Ji(e,e===Se?Ee:0);if(r===0)n!==null&&ru(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&ru(n),t===1)e.tag===0?Om(Yu.bind(null,e)):tf(Yu.bind(null,e)),Mm(function(){!(Z&6)&&fn()}),n=null;else{switch(Td(r)){case 1:n=Ua;break;case 4:n=Nd;break;case 16:n=Gi;break;case 536870912:n=_d;break;default:n=Gi}n=ep(n,qf.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function qf(e,t){if(Ui=-1,Hi=0,Z&6)throw Error(A(327));var n=e.callbackNode;if(Jn()&&e.callbackNode!==n)return null;var r=Ji(e,e===Se?Ee:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=gl(e,r);else{t=r;var i=Z;Z|=2;var l=Yf();(Se!==e||Ee!==t)&&(Pt=null,ar=ge()+500,wn(e,t));do try{og();break}catch(a){Kf(e,a)}while(!0);ts(),pl.current=l,Z=i,ye!==null?t=0:(Se=null,Ee=0,t=ke)}if(t!==0){if(t===2&&(i=Ho(e),i!==0&&(r=i,t=va(e,i))),t===1)throw n=ri,wn(e,0),Xt(e,r),Qe(e,ge()),n;if(t===6)Xt(e,r);else{if(i=e.current.alternate,!(r&30)&&!ig(i)&&(t=gl(e,r),t===2&&(l=Ho(e),l!==0&&(r=l,t=va(e,l))),t===1))throw n=ri,wn(e,0),Xt(e,r),Qe(e,ge()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(A(345));case 2:gn(e,Ue,Pt);break;case 3:if(Xt(e,r),(r&130023424)===r&&(t=gs+500-ge(),10<t)){if(Ji(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Re(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=Go(gn.bind(null,e,Ue,Pt),t);break}gn(e,Ue,Pt);break;case 4:if(Xt(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-yt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=ge()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*rg(r/1960))-r,10<r){e.timeoutHandle=Go(gn.bind(null,e,Ue,Pt),r);break}gn(e,Ue,Pt);break;case 5:gn(e,Ue,Pt);break;default:throw Error(A(329))}}}return Qe(e,ge()),e.callbackNode===n?qf.bind(null,e):null}function va(e,t){var n=Dr;return e.current.memoizedState.isDehydrated&&(wn(e,t).flags|=256),e=gl(e,t),e!==2&&(t=Ue,Ue=n,t!==null&&ya(t)),e}function ya(e){Ue===null?Ue=e:Ue.push.apply(Ue,e)}function ig(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!kt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Xt(e,t){for(t&=~ms,t&=~Tl,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-yt(t),r=1<<n;e[n]=-1,t&=~r}}function Yu(e){if(Z&6)throw Error(A(327));Jn();var t=Ji(e,0);if(!(t&1))return Qe(e,ge()),null;var n=gl(e,t);if(e.tag!==0&&n===2){var r=Ho(e);r!==0&&(t=r,n=va(e,r))}if(n===1)throw n=ri,wn(e,0),Xt(e,t),Qe(e,ge()),n;if(n===6)throw Error(A(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,gn(e,Ue,Pt),Qe(e,ge()),null}function vs(e,t){var n=Z;Z|=1;try{return e(t)}finally{Z=n,Z===0&&(ar=ge()+500,El&&fn())}}function Nn(e){Jt!==null&&Jt.tag===0&&!(Z&6)&&Jn();var t=Z;Z|=1;var n=ct.transition,r=re;try{if(ct.transition=null,re=1,e)return e()}finally{re=r,ct.transition=n,Z=t,!(Z&6)&&fn()}}function ys(){Je=Qn.current,ce(Qn)}function wn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,Am(n)),ye!==null)for(n=ye.return;n!==null;){var r=n;switch(Ja(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&rl();break;case 3:lr(),ce(Ve),ce(Pe),as();break;case 5:os(r);break;case 4:lr();break;case 13:ce(fe);break;case 19:ce(fe);break;case 10:ns(r.type._context);break;case 22:case 23:ys()}n=n.return}if(Se=e,ye=e=an(e.current,null),Ee=Je=t,ke=0,ri=null,ms=Tl=En=0,Ue=Dr=null,xn!==null){for(t=0;t<xn.length;t++)if(n=xn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}xn=null}return e}function Kf(e,t){do{var n=ye;try{if(ts(),Oi.current=fl,dl){for(var r=pe.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}dl=!1}if(jn=0,we=xe=pe=null,Ar=!1,ei=0,hs.current=null,n===null||n.return===null){ke=1,ri=t,ye=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=Ee,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var c=s,d=a,f=d.tag;if(!(d.mode&1)&&(f===0||f===11||f===15)){var h=d.alternate;h?(d.updateQueue=h.updateQueue,d.memoizedState=h.memoizedState,d.lanes=h.lanes):(d.updateQueue=null,d.memoizedState=null)}var p=Du(o);if(p!==null){p.flags&=-257,Ru(p,o,a,l,t),p.mode&1&&Mu(l,c,t),t=p,s=c;var w=t.updateQueue;if(w===null){var S=new Set;S.add(s),t.updateQueue=S}else w.add(s);break e}else{if(!(t&1)){Mu(l,c,t),xs();break e}s=Error(A(426))}}else if(de&&a.mode&1){var I=Du(o);if(I!==null){!(I.flags&65536)&&(I.flags|=256),Ru(I,o,a,l,t),Za(or(s,a));break e}}l=s=or(s,a),ke!==4&&(ke=2),Dr===null?Dr=[l]:Dr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var m=Lf(l,s,t);zu(l,m);break e;case 1:a=s;var v=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof v.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(ln===null||!ln.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var b=Pf(l,a,t);zu(l,b);break e}}l=l.return}while(l!==null)}Gf(n)}catch(j){t=j,ye===n&&n!==null&&(ye=n=n.return);continue}break}while(!0)}function Yf(){var e=pl.current;return pl.current=fl,e===null?fl:e}function xs(){(ke===0||ke===3||ke===2)&&(ke=4),Se===null||!(En&268435455)&&!(Tl&268435455)||Xt(Se,Ee)}function gl(e,t){var n=Z;Z|=2;var r=Yf();(Se!==e||Ee!==t)&&(Pt=null,wn(e,t));do try{lg();break}catch(i){Kf(e,i)}while(!0);if(ts(),Z=n,pl.current=r,ye!==null)throw Error(A(261));return Se=null,Ee=0,ke}function lg(){for(;ye!==null;)Xf(ye)}function og(){for(;ye!==null&&!Lh();)Xf(ye)}function Xf(e){var t=Zf(e.alternate,e,Je);e.memoizedProps=e.pendingProps,t===null?Gf(e):ye=t,hs.current=null}function Gf(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=Zm(n,t),n!==null){n.flags&=32767,ye=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{ke=6,ye=null;return}}else if(n=Jm(n,t,Je),n!==null){ye=n;return}if(t=t.sibling,t!==null){ye=t;return}ye=t=e}while(t!==null);ke===0&&(ke=5)}function gn(e,t,n){var r=re,i=ct.transition;try{ct.transition=null,re=1,ag(e,t,n,r)}finally{ct.transition=i,re=r}return null}function ag(e,t,n,r){do Jn();while(Jt!==null);if(Z&6)throw Error(A(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(A(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if($h(e,l),e===Se&&(ye=Se=null,Ee=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||Ni||(Ni=!0,ep(Gi,function(){return Jn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=ct.transition,ct.transition=null;var o=re;re=1;var a=Z;Z|=4,hs.current=null,tg(e,n),Wf(n,e),Nm(Yo),Zi=!!Ko,Yo=Ko=null,e.current=n,ng(n),Ph(),Z=a,re=o,ct.transition=l}else e.current=n;if(Ni&&(Ni=!1,Jt=e,ml=i),l=e.pendingLanes,l===0&&(ln=null),Mh(n.stateNode),Qe(e,ge()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(hl)throw hl=!1,e=ma,ma=null,e;return ml&1&&e.tag!==0&&Jn(),l=e.pendingLanes,l&1?e===ga?Rr++:(Rr=0,ga=e):Rr=0,fn(),null}function Jn(){if(Jt!==null){var e=Td(ml),t=ct.transition,n=re;try{if(ct.transition=null,re=16>e?16:e,Jt===null)var r=!1;else{if(e=Jt,Jt=null,ml=0,Z&6)throw Error(A(331));var i=Z;for(Z|=4,B=e.current;B!==null;){var l=B,o=l.child;if(B.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var c=a[s];for(B=c;B!==null;){var d=B;switch(d.tag){case 0:case 11:case 15:Mr(8,d,l)}var f=d.child;if(f!==null)f.return=d,B=f;else for(;B!==null;){d=B;var h=d.sibling,p=d.return;if(Uf(d),d===c){B=null;break}if(h!==null){h.return=p,B=h;break}B=p}}}var w=l.alternate;if(w!==null){var S=w.child;if(S!==null){w.child=null;do{var I=S.sibling;S.sibling=null,S=I}while(S!==null)}}B=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,B=o;else e:for(;B!==null;){if(l=B,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Mr(9,l,l.return)}var m=l.sibling;if(m!==null){m.return=l.return,B=m;break e}B=l.return}}var v=e.current;for(B=v;B!==null;){o=B;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,B=y;else e:for(o=v;B!==null;){if(a=B,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:zl(9,a)}}catch(j){me(a,a.return,j)}if(a===o){B=null;break e}var b=a.sibling;if(b!==null){b.return=a.return,B=b;break e}B=a.return}}if(Z=i,fn(),Et&&typeof Et.onPostCommitFiberRoot=="function")try{Et.onPostCommitFiberRoot(wl,e)}catch{}r=!0}return r}finally{re=n,ct.transition=t}}return!1}function Xu(e,t,n){t=or(n,t),t=Lf(e,t,1),e=rn(e,t,1),t=Re(),e!==null&&(oi(e,1,t),Qe(e,t))}function me(e,t,n){if(e.tag===3)Xu(e,e,n);else for(;t!==null;){if(t.tag===3){Xu(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(ln===null||!ln.has(r))){e=or(n,e),e=Pf(t,e,1),t=rn(t,e,1),e=Re(),t!==null&&(oi(t,1,e),Qe(t,e));break}}t=t.return}}function sg(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Re(),e.pingedLanes|=e.suspendedLanes&n,Se===e&&(Ee&n)===n&&(ke===4||ke===3&&(Ee&130023424)===Ee&&500>ge()-gs?wn(e,0):ms|=n),Qe(e,t)}function Jf(e,t){t===0&&(e.mode&1?(t=vi,vi<<=1,!(vi&130023424)&&(vi=4194304)):t=1);var n=Re();e=Ot(e,t),e!==null&&(oi(e,t,n),Qe(e,n))}function ug(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),Jf(e,n)}function cg(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(A(314))}r!==null&&r.delete(t),Jf(e,n)}var Zf;Zf=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Ve.current)He=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return He=!1,Gm(e,t,n);He=!!(e.flags&131072)}else He=!1,de&&t.flags&1048576&&nf(t,ol,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;$i(e,t),e=t.pendingProps;var i=nr(t,Pe.current);Gn(t,n),i=us(null,t,r,e,i,n);var l=cs();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,We(r)?(l=!0,il(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,is(t),i.updater=_l,t.stateNode=i,i._reactInternals=t,ia(t,r,e,n),t=aa(null,t,r,!0,l,n)):(t.tag=0,de&&l&&Ga(t),De(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch($i(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=fg(r),e=mt(r,e),i){case 0:t=oa(null,t,r,e,n);break e;case 1:t=Bu(null,t,r,e,n);break e;case 11:t=Fu(null,t,r,e,n);break e;case 14:t=Ou(null,t,r,mt(r.type,e),n);break e}throw Error(A(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:mt(r,i),oa(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:mt(r,i),Bu(e,t,r,i,n);case 3:e:{if(Df(t),e===null)throw Error(A(387));r=t.pendingProps,l=t.memoizedState,i=l.element,uf(e,t),ul(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=or(Error(A(423)),t),t=$u(e,t,r,n,i);break e}else if(r!==i){i=or(Error(A(424)),t),t=$u(e,t,r,n,i);break e}else for(Ze=nn(t.stateNode.containerInfo.firstChild),tt=t,de=!0,vt=null,n=af(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(rr(),r===i){t=Bt(e,t,n);break e}De(e,t,r,n)}t=t.child}return t;case 5:return cf(t),e===null&&ta(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,Xo(r,i)?o=null:l!==null&&Xo(r,l)&&(t.flags|=32),Mf(e,t),De(e,t,o,n),t.child;case 6:return e===null&&ta(t),null;case 13:return Rf(e,t,n);case 4:return ls(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=ir(t,null,r,n):De(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:mt(r,i),Fu(e,t,r,i,n);case 7:return De(e,t,t.pendingProps,n),t.child;case 8:return De(e,t,t.pendingProps.children,n),t.child;case 12:return De(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,ae(al,r._currentValue),r._currentValue=o,l!==null)if(kt(l.value,o)){if(l.children===i.children&&!Ve.current){t=Bt(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Dt(-1,n&-n),s.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?s.next=s:(s.next=d.next,d.next=s),c.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),na(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(A(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),na(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}De(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Gn(t,n),i=dt(i),r=r(i),t.flags|=1,De(e,t,r,n),t.child;case 14:return r=t.type,i=mt(r,t.pendingProps),i=mt(r.type,i),Ou(e,t,r,i,n);case 15:return If(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:mt(r,i),$i(e,t),t.tag=1,We(r)?(e=!0,il(t)):e=!1,Gn(t,n),Tf(t,r,i),ia(t,r,i,n),aa(null,t,r,!0,e,n);case 19:return Ff(e,t,n);case 22:return Af(e,t,n)}throw Error(A(156,t.tag))};function ep(e,t){return Ed(e,t)}function dg(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function ut(e,t,n,r){return new dg(e,t,n,r)}function ks(e){return e=e.prototype,!(!e||!e.isReactComponent)}function fg(e){if(typeof e=="function")return ks(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Oa)return 11;if(e===Ba)return 14}return 2}function an(e,t){var n=e.alternate;return n===null?(n=ut(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Vi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")ks(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Dn:return Sn(n.children,i,l,t);case Fa:o=8,i|=8;break;case _o:return e=ut(12,n,t,i|2),e.elementType=_o,e.lanes=l,e;case zo:return e=ut(13,n,t,i),e.elementType=zo,e.lanes=l,e;case To:return e=ut(19,n,t,i),e.elementType=To,e.lanes=l,e;case ud:return Ll(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case ad:o=10;break e;case sd:o=9;break e;case Oa:o=11;break e;case Ba:o=14;break e;case qt:o=16,r=null;break e}throw Error(A(130,e==null?e:typeof e,""))}return t=ut(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function Sn(e,t,n,r){return e=ut(7,e,r,t),e.lanes=n,e}function Ll(e,t,n,r){return e=ut(22,e,r,t),e.elementType=ud,e.lanes=n,e.stateNode={isHidden:!1},e}function co(e,t,n){return e=ut(6,e,null,t),e.lanes=n,e}function fo(e,t,n){return t=ut(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function pg(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Wl(0),this.expirationTimes=Wl(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Wl(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function ws(e,t,n,r,i,l,o,a,s){return e=new pg(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=ut(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},is(l),e}function hg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:Mn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function tp(e){if(!e)return un;e=e._reactInternals;e:{if(zn(e)!==e||e.tag!==1)throw Error(A(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(We(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(A(171))}if(e.tag===1){var n=e.type;if(We(n))return ef(e,n,t)}return t}function np(e,t,n,r,i,l,o,a,s){return e=ws(n,r,!0,e,i,l,o,a,s),e.context=tp(null),n=e.current,r=Re(),i=on(n),l=Dt(r,i),l.callback=t??null,rn(n,l,i),e.current.lanes=i,oi(e,i,r),Qe(e,r),e}function Pl(e,t,n,r){var i=t.current,l=Re(),o=on(i);return n=tp(n),t.context===null?t.context=n:t.pendingContext=n,t=Dt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=rn(i,t,o),e!==null&&(xt(e,i,o,l),Fi(e,i,o)),o}function vl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function Gu(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function Ss(e,t){Gu(e,t),(e=e.alternate)&&Gu(e,t)}function mg(){return null}var rp=typeof reportError=="function"?reportError:function(e){console.error(e)};function bs(e){this._internalRoot=e}Il.prototype.render=bs.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(A(409));Pl(e,t,null,null)};Il.prototype.unmount=bs.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;Nn(function(){Pl(null,e,null,null)}),t[Ft]=null}};function Il(e){this._internalRoot=e}Il.prototype.unstable_scheduleHydration=function(e){if(e){var t=Id();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Yt.length&&t!==0&&t<Yt[n].priority;n++);Yt.splice(n,0,e),n===0&&Md(e)}};function Cs(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Al(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function Ju(){}function gg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=vl(o);l.call(c)}}var o=np(t,r,e,0,null,!1,!1,"",Ju);return e._reactRootContainer=o,e[Ft]=o.current,Yr(e.nodeType===8?e.parentNode:e),Nn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=vl(s);a.call(c)}}var s=ws(e,0,!1,null,null,!1,!1,"",Ju);return e._reactRootContainer=s,e[Ft]=s.current,Yr(e.nodeType===8?e.parentNode:e),Nn(function(){Pl(t,s,n,r)}),s}function Ml(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=vl(o);a.call(s)}}Pl(t,o,e,i)}else o=gg(n,t,e,i,r);return vl(o)}Ld=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=Nr(t.pendingLanes);n!==0&&(Ha(t,n|1),Qe(t,ge()),!(Z&6)&&(ar=ge()+500,fn()))}break;case 13:Nn(function(){var r=Ot(e,1);if(r!==null){var i=Re();xt(r,e,1,i)}}),Ss(e,1)}};Va=function(e){if(e.tag===13){var t=Ot(e,134217728);if(t!==null){var n=Re();xt(t,e,134217728,n)}Ss(e,134217728)}};Pd=function(e){if(e.tag===13){var t=on(e),n=Ot(e,t);if(n!==null){var r=Re();xt(n,e,t,r)}Ss(e,t)}};Id=function(){return re};Ad=function(e,t){var n=re;try{return re=e,t()}finally{re=n}};Bo=function(e,t,n){switch(t){case"input":if(Io(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=jl(r);if(!i)throw Error(A(90));dd(r),Io(r,i)}}}break;case"textarea":pd(e,n);break;case"select":t=n.value,t!=null&&qn(e,!!n.multiple,t,!1)}};kd=vs;wd=Nn;var vg={usingClientEntryPoint:!1,Events:[si,Bn,jl,yd,xd,vs]},Sr={findFiberByHostInstance:yn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},yg={bundleType:Sr.bundleType,version:Sr.version,rendererPackageName:Sr.rendererPackageName,rendererConfig:Sr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:$t.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=Cd(e),e===null?null:e.stateNode},findFiberByHostInstance:Sr.findFiberByHostInstance||mg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var _i=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!_i.isDisabled&&_i.supportsFiber)try{wl=_i.inject(yg),Et=_i}catch{}}rt.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=vg;rt.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!Cs(t))throw Error(A(200));return hg(e,t,null,n)};rt.createRoot=function(e,t){if(!Cs(e))throw Error(A(299));var n=!1,r="",i=rp;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=ws(e,1,!1,null,null,n,!1,r,i),e[Ft]=t.current,Yr(e.nodeType===8?e.parentNode:e),new bs(t)};rt.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(A(188)):(e=Object.keys(e).join(","),Error(A(268,e)));return e=Cd(t),e=e===null?null:e.stateNode,e};rt.flushSync=function(e){return Nn(e)};rt.hydrate=function(e,t,n){if(!Al(t))throw Error(A(200));return Ml(null,e,t,!0,n)};rt.hydrateRoot=function(e,t,n){if(!Cs(e))throw Error(A(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=rp;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=np(t,null,e,1,n??null,i,!1,l,o),e[Ft]=t.current,Yr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new Il(t)};rt.render=function(e,t,n){if(!Al(t))throw Error(A(200));return Ml(null,e,t,!1,n)};rt.unmountComponentAtNode=function(e){if(!Al(e))throw Error(A(40));return e._reactRootContainer?(Nn(function(){Ml(null,null,e,!1,function(){e._reactRootContainer=null,e[Ft]=null})}),!0):!1};rt.unstable_batchedUpdates=vs;rt.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Al(n))throw Error(A(200));if(e==null||e._reactInternals===void 0)throw Error(A(38));return Ml(e,t,n,!1,r)};rt.version="18.3.1-next-f1338f8080-20240426";function ip(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(ip)}catch(e){console.error(e)}}ip(),rd.exports=rt;var xg=rd.exports,Zu=xg;Eo.createRoot=Zu.createRoot,Eo.hydrateRoot=Zu.hydrateRoot;const kg="",wg=({selection:e,onSelect:t,onRefresh:n})=>{const[r,i]=O.useState(null),[l,o]=O.useState(new Set(["all"])),[a,s]=O.useState(!0),[c,d]=O.useState(null),f=async()=>{try{const v=await fetch(`${kg}/api/hierarchy`);if(!v.ok)throw new Error("Failed to fetch hierarchy");const y=await v.json();i(y),d(null)}catch(v){d(v instanceof Error?v.message:"Unknown error")}finally{s(!1)}};O.useEffect(()=>{f();const v=setInterval(f,5e3);return()=>clearInterval(v)},[]);const h=v=>{o(y=>{const b=new Set(y);return b.has(v)?b.delete(v):b.add(v),b})},p=v=>{var y;if(v.type==="root")t({type:"overview"});else if(v.type==="agent")t({type:"agent",agentId:v.id});else if(v.type==="thread"){const b=(y=r==null?void 0:r.root.children)==null?void 0:y.find(j=>{var k;return(k=j.children)==null?void 0:k.some(N=>N.id===v.id)});t({type:"thread",agentId:b==null?void 0:b.id,threadId:v.id})}},w=v=>v.type==="root"&&e.type==="overview"||v.type==="agent"&&e.type==="agent"&&e.agentId===v.id||v.type==="thread"&&e.threadId===v.id,S=v=>!v||v.length===0?null:u.jsx("span",{className:"badges",children:v.map((y,b)=>u.jsxs("span",{className:`badge badge-${y.type}`,title:`${y.count} ${y.type}`,children:[y.type==="pending"&&"⏳",y.type==="unread"&&"📬",y.type==="running"&&"▶️",y.count]},b))}),I=v=>{if(!v)return null;const y={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsx("span",{className:"status-indicator",style:{backgroundColor:y[v]||y.idle},title:v})},m=(v,y=0)=>{const b=l.has(v.id),j=v.children&&v.children.length>0,k=w(v);return u.jsxs("div",{className:"tree-node",children:[u.jsxs("div",{className:`tree-node-content ${k?"selected":""} ${v.type}`,style:{paddingLeft:`${y*16+8}px`},onClick:()=>p(v),children:[j&&u.jsx("span",{className:`expand-icon ${b?"expanded":""}`,onClick:N=>{N.stopPropagation(),h(v.id)},children:b?"▼":"▶"}),!j&&u.jsx("span",{className:"expand-icon-placeholder"}),v.type==="agent"&&I(v.status),u.jsx("span",{className:"node-label",children:v.label}),S(v.badges)]}),j&&b&&u.jsx("div",{className:"tree-children",children:v.children.map(N=>m(N,y+1))})]},v.id)};return a&&!r?u.jsx("div",{className:"hierarchy-tree loading",children:"Loading..."}):c?u.jsxs("div",{className:"hierarchy-tree error",children:[u.jsxs("p",{children:["Error: ",c]}),u.jsx("button",{onClick:f,children:"Retry"})]}):u.jsxs("div",{className:"hierarchy-tree",children:[u.jsxs("div",{className:"tree-header",children:[u.jsx("h3",{children:"Agents"}),u.jsx("button",{className:"refresh-btn",onClick:()=>{f(),n==null||n()},title:"Refresh",children:"\\u21BB"})]}),u.jsx("div",{className:"tree-content",children:r&&m(r.root)}),r&&u.jsx("div",{className:"tree-footer",children:u.jsxs("div",{className:"aggregate-stats",children:[u.jsxs("span",{title:"Total agents",children:[r.aggregate.total_agents," agents"]}),u.jsxs("span",{title:"Active",children:[r.aggregate.active_agents," active"]}),r.aggregate.pending_approvals>0&&u.jsxs("span",{className:"pending",title:"Pending approvals",children:[r.aggregate.pending_approvals," pending"]})]})})]})},Sg="_card_1d3of_1",bg="_compact_1d3of_9",Cg="_title_1d3of_13",jg="_metricsGrid_1d3of_20",Eg="_metricItem_1d3of_26",Ng="_metricLabel_1d3of_32",_g="_metricValue_1d3of_39",zg="_cost_1d3of_46",Tg="_averages_1d3of_50",Lg="_averagesLabel_1d3of_61",Pg="_avgItem_1d3of_65",Ig="_compactRow_1d3of_72",Ag="_compactLabel_1d3of_80",Mg="_compactValue_1d3of_84",Dg="_loading_1d3of_91",Rg="_error_1d3of_97",Fg="_errorText_1d3of_101",K={card:Sg,compact:bg,title:Cg,metricsGrid:jg,metricItem:Eg,metricLabel:Ng,metricValue:_g,cost:zg,averages:Tg,averagesLabel:Lg,avgItem:Pg,compactRow:Ig,compactLabel:Ag,compactValue:Mg,loading:Dg,error:Rg,errorText:Fg};function ec(e){return e<1e3?`${e}ms`:e<6e4?`${(e/1e3).toFixed(1)}s`:e<36e5?`${(e/6e4).toFixed(1)}m`:`${(e/36e5).toFixed(1)}h`}function In(e){return e.toLocaleString()}function po(e){return e===0?"$0.00":e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`}function Og({scopeType:e,scopeId:t="",title:n,compact:r=!1}){const[i,l]=O.useState(null),[o,a]=O.useState(!0),[s,c]=O.useState(null),d=O.useCallback(async()=>{try{let h="/api/metrics";e!=="global"&&(h=`/api/metrics/${e}/${t}`);const p=await fetch(h);if(!p.ok)throw new Error(`Failed to fetch metrics: ${p.status}`);const w=await p.json();l(w),c(null)}catch(h){c(h instanceof Error?h.message:"Failed to load metrics")}finally{a(!1)}},[e,t]);if(O.useEffect(()=>{d();const h=setInterval(d,3e4);return()=>clearInterval(h)},[d]),o)return u.jsx("div",{className:`${K.card} ${r?K.compact:""}`,children:u.jsx("div",{className:K.loading,children:"Loading metrics..."})});if(s)return u.jsx("div",{className:`${K.card} ${r?K.compact:""} ${K.error}`,children:u.jsx("div",{className:K.errorText,children:s})});if(!i)return null;const f=n||(e==="global"?"Global Metrics":e==="agent"?`Agent: ${t}`:`Thread: ${t.slice(0,12)}...`);return r?u.jsx("div",{className:`${K.card} ${K.compact}`,children:u.jsxs("div",{className:K.compactRow,children:[u.jsx("span",{className:K.compactLabel,children:"Runs:"}),u.jsx("span",{className:K.compactValue,children:In(i.total_runs)}),u.jsx("span",{className:K.compactLabel,children:"Tokens:"}),u.jsx("span",{className:K.compactValue,children:In(i.total_tokens)}),u.jsx("span",{className:K.compactLabel,children:"Cost:"}),u.jsx("span",{className:K.compactValue,children:po(i.total_cost)})]})}):u.jsxs("div",{className:K.card,children:[u.jsx("h3",{className:K.title,children:f}),u.jsxs("div",{className:K.metricsGrid,children:[u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Runs"}),u.jsx("span",{className:K.metricValue,children:In(i.total_runs)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Tokens"}),u.jsx("span",{className:K.metricValue,children:In(i.total_tokens)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Cost"}),u.jsx("span",{className:`${K.metricValue} ${K.cost}`,children:po(i.total_cost)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Duration"}),u.jsx("span",{className:K.metricValue,children:ec(i.total_duration_ms)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Files Modified"}),u.jsx("span",{className:K.metricValue,children:In(i.total_files_modified)})]})]}),i.total_runs>0&&u.jsxs("div",{className:K.averages,children:[u.jsx("span",{className:K.averagesLabel,children:"Averages per run:"}),u.jsxs("span",{className:K.avgItem,children:[In(Math.round(i.avg_tokens_per_run))," tokens"]}),u.jsx("span",{className:K.avgItem,children:po(i.avg_cost_per_run)}),u.jsx("span",{className:K.avgItem,children:ec(Math.round(i.avg_duration_per_run))})]})]})}const Xe=({title:e,value:t,color:n="default",small:r})=>u.jsxs("div",{className:`stat-card stat-${n} ${r?"stat-small":""}`,children:[u.jsx("div",{className:"stat-value",children:t}),u.jsx("div",{className:"stat-title",children:e})]}),Bg=e=>{if(e<1e3)return`${e}ms`;const t=e/1e3;if(t<60)return`${t.toFixed(1)}s`;const n=Math.floor(t/60),r=(t%60).toFixed(0);return`${n}m ${r}s`},$g=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,zi=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}k`:e.toString(),Ug=({agent:e,onClick:t})=>{var o,a,s,c,d;const n=((o=e.children)==null?void 0:o.length)||0,r=((s=(a=e.badges)==null?void 0:a.find(f=>f.type==="pending"))==null?void 0:s.count)||0,i=((d=(c=e.badges)==null?void 0:c.find(f=>f.type==="running"))==null?void 0:d.count)||0,l={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsxs("div",{className:"agent-card",onClick:t,children:[u.jsxs("div",{className:"agent-card-header",children:[u.jsx("span",{className:"agent-status-dot",style:{backgroundColor:l[e.status||"idle"]}}),u.jsx("span",{className:"agent-name",children:e.label})]}),u.jsxs("div",{className:"agent-card-stats",children:[u.jsxs("span",{className:"agent-stat",children:[u.jsx("span",{className:"agent-stat-value",children:n}),u.jsx("span",{className:"agent-stat-label",children:"threads"})]}),r>0&&u.jsxs("span",{className:"agent-stat pending",children:[u.jsx("span",{className:"agent-stat-value",children:r}),u.jsx("span",{className:"agent-stat-label",children:"pending"})]}),i>0&&u.jsxs("span",{className:"agent-stat running",children:[u.jsx("span",{className:"agent-stat-value",children:i}),u.jsx("span",{className:"agent-stat-label",children:"running"})]})]})]})},Hg=({aggregate:e,agents:t,onSelectAgent:n})=>{const r=e.execution,i=r&&r.total_executions>0,l=i?Math.round(r.successful_executions/r.total_executions*100):0;return u.jsxs("div",{className:"all-agents-overview",children:[u.jsx("div",{className:"overview-header",children:u.jsx("h2",{children:"All Agents Overview"})}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Xe,{title:"Total Agents",value:e.total_agents}),u.jsx(Xe,{title:"Active",value:e.active_agents,color:"green"}),u.jsx(Xe,{title:"Pending Approvals",value:e.pending_approvals,color:"orange"}),u.jsx(Xe,{title:"Total Threads",value:e.total_threads,color:"blue"})]}),u.jsxs("div",{className:"metrics-section",children:[u.jsx("h3",{children:"Usage Metrics (Today)"}),u.jsx(Og,{scopeType:"global",title:"Global Metrics"})]}),i&&u.jsxs("div",{className:"execution-stats-section",children:[u.jsx("h3",{children:"Execution Statistics"}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Xe,{title:"Total Executions",value:r.total_executions,color:"purple"}),u.jsx(Xe,{title:"Success Rate",value:`${l}%`,color:"green"}),u.jsx(Xe,{title:"Total Duration",value:Bg(r.total_duration_ms)}),u.jsx(Xe,{title:"Total Cost",value:$g(r.total_cost),color:"orange"})]}),u.jsxs("div",{className:"stats-row token-stats",children:[u.jsx(Xe,{title:"Input Tokens",value:zi(r.total_input_tokens),small:!0}),u.jsx(Xe,{title:"Output Tokens",value:zi(r.total_output_tokens),small:!0}),u.jsx(Xe,{title:"Cache Read",value:zi(r.total_cache_read_tokens),small:!0}),u.jsx(Xe,{title:"Cache Created",value:zi(r.total_cache_create_tokens),small:!0}),u.jsx(Xe,{title:"Files Created",value:r.total_files_created,small:!0})]})]}),u.jsxs("div",{className:"agents-section",children:[u.jsx("h3",{children:"Agents"}),u.jsxs("div",{className:"agent-cards-grid",children:[t.map(o=>u.jsx(Ug,{agent:o,onClick:()=>n(o.id)},o.id)),t.length===0&&u.jsx("div",{className:"no-agents",children:"No agents found. Start an agent to see it here."})]})]})]})},Vg=({items:e})=>u.jsx("nav",{className:"breadcrumb",children:e.map((t,n)=>u.jsxs(Qt.Fragment,{children:[n>0&&u.jsx("span",{className:"breadcrumb-separator",children:"/"}),t.onClick?u.jsx("button",{className:"breadcrumb-link",onClick:t.onClick,children:t.label}):u.jsx("span",{className:"breadcrumb-current",children:t.label})]},n))}),Lt={plus:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),u.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"}),u.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),u.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),u.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),u.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),u.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),u.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("polyline",{points:"3 6 5 6 21 6"}),u.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:u.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},Wg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=O.useState(!1),[c,d]=O.useState(""),[f,h]=O.useState(null),[p,w]=O.useState(""),[S,I]=O.useState(null),m=()=>{c.trim()&&(r(c.trim()),d(""),s(!1))},v=_=>{_.key==="Enter"&&!_.shiftKey?(_.preventDefault(),m()):_.key==="Escape"&&(s(!1),d(""))},y=(_,D)=>{D.stopPropagation(),h(_.id),w(_.title)},b=_=>{var D;p.trim()&&p.trim()!==((D=e.find(W=>W.id===_))==null?void 0:D.title)&&l(_,p.trim()),h(null),w("")},j=()=>{h(null),w("")},k=(_,D)=>{_.key==="Enter"?(_.preventDefault(),b(D)):_.key==="Escape"&&j()},N=(_,D)=>{D.stopPropagation(),I(_)},T=(_,D)=>{D.stopPropagation(),i(_),I(null)},R=_=>{_.stopPropagation(),I(null)},P=_=>{const D=new Date(_),X=new Date().getTime()-D.getTime(),U=Math.floor(X/6e4),Q=Math.floor(X/36e5),ie=Math.floor(X/864e5);return U<1?"now":U<60?`${U}m`:Q<24?`${Q}h`:ie<7?`${ie}d`:D.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return u.jsxs("div",{className:"thread-list",children:[u.jsxs("div",{className:"list-header",children:[u.jsx("h2",{children:"Conversations"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:Lt.plus})]}),a&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:c,onChange:_=>d(_.target.value),onKeyDown:v,placeholder:"Conversation title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:m,children:"Create"})]})]}),u.jsx("div",{className:"thread-items",children:e.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Lt.hash}),u.jsx("p",{children:"No conversations yet"}),u.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(_=>{const D=o.get(_.id)||0,W=_.id===t,X=f===_.id,U=S===_.id;return u.jsxs("div",{className:`thread-item ${W?"selected":""} ${D>0?"has-unread":""}`,onClick:()=>!X&&n(_.id),children:[u.jsx("div",{className:`status-dot ${_.status}`}),u.jsxs("div",{className:"thread-content",children:[u.jsx("div",{className:"thread-title-row",children:X?u.jsxs("div",{className:"edit-title-form",onClick:Q=>Q.stopPropagation(),children:[u.jsx("input",{type:"text",value:p,onChange:Q=>w(Q.target.value),onKeyDown:Q=>k(Q,_.id),autoFocus:!0}),u.jsx("button",{className:"edit-action save",onClick:()=>b(_.id),title:"Save",children:Lt.check}),u.jsx("button",{className:"edit-action cancel",onClick:j,title:"Cancel",children:Lt.x})]}):u.jsxs(u.Fragment,{children:[u.jsx("span",{className:"thread-title",children:_.title}),u.jsx("span",{className:"thread-time",children:P(_.updated_at)})]})}),u.jsxs("div",{className:"thread-meta",children:[_.target_agent&&u.jsxs("span",{className:"thread-agent",title:`Target: ${_.target_agent}`,children:[Lt.bot,_.target_agent]}),u.jsxs("span",{className:"thread-seq",children:["#",_.last_seq]})]})]}),!X&&!U&&u.jsxs("div",{className:"thread-actions",children:[u.jsx("button",{className:"action-btn edit",onClick:Q=>y(_,Q),title:"Rename",children:Lt.edit}),u.jsx("button",{className:"action-btn delete",onClick:Q=>N(_.id,Q),title:"Delete",children:Lt.trash})]}),U&&u.jsxs("div",{className:"delete-confirm",onClick:Q=>Q.stopPropagation(),children:[u.jsx("span",{className:"confirm-text",children:"Delete?"}),u.jsx("button",{className:"confirm-btn yes",onClick:Q=>T(_.id,Q),title:"Confirm delete",children:Lt.check}),u.jsx("button",{className:"confirm-btn no",onClick:R,title:"Cancel",children:Lt.x})]}),D>0&&!U&&u.jsx("span",{className:"unread-badge",children:D})]},_.id)})}),u.jsx("style",{children:`
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
      `})]})};function Qg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const qg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,Kg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,Yg={};function tc(e,t){return(Yg.jsx?Kg:qg).test(e)}const Xg=/[ \t\n\f\r]/g;function Gg(e){return typeof e=="object"?e.type==="text"?nc(e.value):!1:nc(e)}function nc(e){return e.replace(Xg,"")===""}class ci{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}ci.prototype.normal={};ci.prototype.property={};ci.prototype.space=void 0;function lp(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new ci(n,r,t)}function xa(e){return e.toLowerCase()}class Ke{constructor(t,n){this.attribute=n,this.property=t}}Ke.prototype.attribute="";Ke.prototype.booleanish=!1;Ke.prototype.boolean=!1;Ke.prototype.commaOrSpaceSeparated=!1;Ke.prototype.commaSeparated=!1;Ke.prototype.defined=!1;Ke.prototype.mustUseProperty=!1;Ke.prototype.number=!1;Ke.prototype.overloadedBoolean=!1;Ke.prototype.property="";Ke.prototype.spaceSeparated=!1;Ke.prototype.space=void 0;let Jg=0;const q=Tn(),ve=Tn(),ka=Tn(),M=Tn(),oe=Tn(),Zn=Tn(),Ge=Tn();function Tn(){return 2**++Jg}const wa=Object.freeze(Object.defineProperty({__proto__:null,boolean:q,booleanish:ve,commaOrSpaceSeparated:Ge,commaSeparated:Zn,number:M,overloadedBoolean:ka,spaceSeparated:oe},Symbol.toStringTag,{value:"Module"})),ho=Object.keys(wa);class js extends Ke{constructor(t,n,r,i){let l=-1;if(super(t,n),rc(this,"space",i),typeof r=="number")for(;++l<ho.length;){const o=ho[l];rc(this,ho[l],(r&wa[o])===wa[o])}}}js.prototype.defined=!0;function rc(e,t,n){n&&(e[t]=n)}function dr(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new js(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[xa(r)]=r,n[xa(l.attribute)]=r}return new ci(t,n,e.space)}const op=dr({properties:{ariaActiveDescendant:null,ariaAtomic:ve,ariaAutoComplete:null,ariaBusy:ve,ariaChecked:ve,ariaColCount:M,ariaColIndex:M,ariaColSpan:M,ariaControls:oe,ariaCurrent:null,ariaDescribedBy:oe,ariaDetails:null,ariaDisabled:ve,ariaDropEffect:oe,ariaErrorMessage:null,ariaExpanded:ve,ariaFlowTo:oe,ariaGrabbed:ve,ariaHasPopup:null,ariaHidden:ve,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:oe,ariaLevel:M,ariaLive:null,ariaModal:ve,ariaMultiLine:ve,ariaMultiSelectable:ve,ariaOrientation:null,ariaOwns:oe,ariaPlaceholder:null,ariaPosInSet:M,ariaPressed:ve,ariaReadOnly:ve,ariaRelevant:null,ariaRequired:ve,ariaRoleDescription:oe,ariaRowCount:M,ariaRowIndex:M,ariaRowSpan:M,ariaSelected:ve,ariaSetSize:M,ariaSort:null,ariaValueMax:M,ariaValueMin:M,ariaValueNow:M,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function ap(e,t){return t in e?e[t]:t}function sp(e,t){return ap(e,t.toLowerCase())}const Zg=dr({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:Zn,acceptCharset:oe,accessKey:oe,action:null,allow:null,allowFullScreen:q,allowPaymentRequest:q,allowUserMedia:q,alt:null,as:null,async:q,autoCapitalize:null,autoComplete:oe,autoFocus:q,autoPlay:q,blocking:oe,capture:null,charSet:null,checked:q,cite:null,className:oe,cols:M,colSpan:null,content:null,contentEditable:ve,controls:q,controlsList:oe,coords:M|Zn,crossOrigin:null,data:null,dateTime:null,decoding:null,default:q,defer:q,dir:null,dirName:null,disabled:q,download:ka,draggable:ve,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:q,formTarget:null,headers:oe,height:M,hidden:ka,high:M,href:null,hrefLang:null,htmlFor:oe,httpEquiv:oe,id:null,imageSizes:null,imageSrcSet:null,inert:q,inputMode:null,integrity:null,is:null,isMap:q,itemId:null,itemProp:oe,itemRef:oe,itemScope:q,itemType:oe,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:q,low:M,manifest:null,max:null,maxLength:M,media:null,method:null,min:null,minLength:M,multiple:q,muted:q,name:null,nonce:null,noModule:q,noValidate:q,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:q,optimum:M,pattern:null,ping:oe,placeholder:null,playsInline:q,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:q,referrerPolicy:null,rel:oe,required:q,reversed:q,rows:M,rowSpan:M,sandbox:oe,scope:null,scoped:q,seamless:q,selected:q,shadowRootClonable:q,shadowRootDelegatesFocus:q,shadowRootMode:null,shape:null,size:M,sizes:null,slot:null,span:M,spellCheck:ve,src:null,srcDoc:null,srcLang:null,srcSet:null,start:M,step:null,style:null,tabIndex:M,target:null,title:null,translate:null,type:null,typeMustMatch:q,useMap:null,value:ve,width:M,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:oe,axis:null,background:null,bgColor:null,border:M,borderColor:null,bottomMargin:M,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:q,declare:q,event:null,face:null,frame:null,frameBorder:null,hSpace:M,leftMargin:M,link:null,longDesc:null,lowSrc:null,marginHeight:M,marginWidth:M,noResize:q,noHref:q,noShade:q,noWrap:q,object:null,profile:null,prompt:null,rev:null,rightMargin:M,rules:null,scheme:null,scrolling:ve,standby:null,summary:null,text:null,topMargin:M,valueType:null,version:null,vAlign:null,vLink:null,vSpace:M,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:q,disableRemotePlayback:q,prefix:null,property:null,results:M,security:null,unselectable:null},space:"html",transform:sp}),ev=dr({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:Ge,accentHeight:M,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:M,amplitude:M,arabicForm:null,ascent:M,attributeName:null,attributeType:null,azimuth:M,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:M,by:null,calcMode:null,capHeight:M,className:oe,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:M,diffuseConstant:M,direction:null,display:null,dur:null,divisor:M,dominantBaseline:null,download:q,dx:null,dy:null,edgeMode:null,editable:null,elevation:M,enableBackground:null,end:null,event:null,exponent:M,externalResourcesRequired:null,fill:null,fillOpacity:M,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:Zn,g2:Zn,glyphName:Zn,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:M,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:M,horizOriginX:M,horizOriginY:M,id:null,ideographic:M,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:M,k:M,k1:M,k2:M,k3:M,k4:M,kernelMatrix:Ge,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:M,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:M,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:M,overlineThickness:M,paintOrder:null,panose1:null,path:null,pathLength:M,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:oe,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:M,pointsAtY:M,pointsAtZ:M,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:Ge,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:Ge,rev:Ge,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:Ge,requiredFeatures:Ge,requiredFonts:Ge,requiredFormats:Ge,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:M,specularExponent:M,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:M,strikethroughThickness:M,string:null,stroke:null,strokeDashArray:Ge,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:M,strokeOpacity:M,strokeWidth:null,style:null,surfaceScale:M,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:Ge,tabIndex:M,tableValues:null,target:null,targetX:M,targetY:M,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:Ge,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:M,underlineThickness:M,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:M,values:null,vAlphabetic:M,vMathematical:M,vectorEffect:null,vHanging:M,vIdeographic:M,version:null,vertAdvY:M,vertOriginX:M,vertOriginY:M,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:M,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:ap}),up=dr({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),cp=dr({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:sp}),dp=dr({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),tv={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},nv=/[A-Z]/g,ic=/-[a-z]/g,rv=/^data[-\w.:]+$/i;function iv(e,t){const n=xa(t);let r=t,i=Ke;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&rv.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(ic,ov);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!ic.test(l)){let o=l.replace(nv,lv);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=js}return new i(r,t)}function lv(e){return"-"+e.toLowerCase()}function ov(e){return e.charAt(1).toUpperCase()}const av=lp([op,Zg,up,cp,dp],"html"),Es=lp([op,ev,up,cp,dp],"svg");function sv(e){return e.join(" ").trim()}var Ns={},lc=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,uv=/\n/g,cv=/^\s*/,dv=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,fv=/^:\s*/,pv=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,hv=/^[;\s]*/,mv=/^\s+|\s+$/g,gv=`
`,oc="/",ac="*",vn="",vv="comment",yv="declaration";function xv(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(w){var S=w.match(uv);S&&(n+=S.length);var I=w.lastIndexOf(gv);r=~I?w.length-I:r+w.length}function l(){var w={line:n,column:r};return function(S){return S.position=new o(w),c(),S}}function o(w){this.start=w,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(w){var S=new Error(t.source+":"+n+":"+r+": "+w);if(S.reason=w,S.filename=t.source,S.line=n,S.column=r,S.source=e,!t.silent)throw S}function s(w){var S=w.exec(e);if(S){var I=S[0];return i(I),e=e.slice(I.length),S}}function c(){s(cv)}function d(w){var S;for(w=w||[];S=f();)S!==!1&&w.push(S);return w}function f(){var w=l();if(!(oc!=e.charAt(0)||ac!=e.charAt(1))){for(var S=2;vn!=e.charAt(S)&&(ac!=e.charAt(S)||oc!=e.charAt(S+1));)++S;if(S+=2,vn===e.charAt(S-1))return a("End of comment missing");var I=e.slice(2,S-2);return r+=2,i(I),e=e.slice(S),r+=2,w({type:vv,comment:I})}}function h(){var w=l(),S=s(dv);if(S){if(f(),!s(fv))return a("property missing ':'");var I=s(pv),m=w({type:yv,property:sc(S[0].replace(lc,vn)),value:I?sc(I[0].replace(lc,vn)):vn});return s(hv),m}}function p(){var w=[];d(w);for(var S;S=h();)S!==!1&&(w.push(S),d(w));return w}return c(),p()}function sc(e){return e?e.replace(mv,vn):vn}var kv=xv,wv=qi&&qi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(Ns,"__esModule",{value:!0});Ns.default=bv;const Sv=wv(kv);function bv(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Sv.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Dl={};Object.defineProperty(Dl,"__esModule",{value:!0});Dl.camelCase=void 0;var Cv=/^--[a-zA-Z0-9_-]+$/,jv=/-([a-z])/g,Ev=/^[^-]+$/,Nv=/^-(webkit|moz|ms|o|khtml)-/,_v=/^-(ms)-/,zv=function(e){return!e||Ev.test(e)||Cv.test(e)},Tv=function(e,t){return t.toUpperCase()},uc=function(e,t){return"".concat(t,"-")},Lv=function(e,t){return t===void 0&&(t={}),zv(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(_v,uc):e=e.replace(Nv,uc),e.replace(jv,Tv))};Dl.camelCase=Lv;var Pv=qi&&qi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},Iv=Pv(Ns),Av=Dl;function Sa(e,t){var n={};return!e||typeof e!="string"||(0,Iv.default)(e,function(r,i){r&&i&&(n[(0,Av.camelCase)(r,t)]=i)}),n}Sa.default=Sa;var Mv=Sa;const Dv=Ta(Mv),fp=pp("end"),_s=pp("start");function pp(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function Rv(e){const t=_s(e),n=fp(e);if(t&&n)return{start:t,end:n}}function Fr(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?cc(e.position):"start"in e||"end"in e?cc(e):"line"in e||"column"in e?ba(e):""}function ba(e){return dc(e&&e.line)+":"+dc(e&&e.column)}function cc(e){return ba(e&&e.start)+"-"+ba(e&&e.end)}function dc(e){return e&&typeof e=="number"?e:1}class Ie extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Fr(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}Ie.prototype.file="";Ie.prototype.name="";Ie.prototype.reason="";Ie.prototype.message="";Ie.prototype.stack="";Ie.prototype.column=void 0;Ie.prototype.line=void 0;Ie.prototype.ancestors=void 0;Ie.prototype.cause=void 0;Ie.prototype.fatal=void 0;Ie.prototype.place=void 0;Ie.prototype.ruleId=void 0;Ie.prototype.source=void 0;const zs={}.hasOwnProperty,Fv=new Map,Ov=/[A-Z]/g,Bv=new Set(["table","tbody","thead","tfoot","tr"]),$v=new Set(["td","th"]),hp="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function Uv(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=Xv(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=Yv(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?Es:av,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=mp(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function mp(e,t,n){if(t.type==="element")return Hv(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return Vv(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return Qv(e,t,n);if(t.type==="mdxjsEsm")return Wv(e,t);if(t.type==="root")return qv(e,t,n);if(t.type==="text")return Kv(e,t)}function Hv(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=Es,e.schema=i),e.ancestors.push(t);const l=vp(e,t.tagName,!1),o=Gv(e,t);let a=Ls(e,t);return Bv.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!Gg(s):!0})),gp(e,o,l,t),Ts(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function Vv(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}ii(e,t.position)}function Wv(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);ii(e,t.position)}function Qv(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=Es,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:vp(e,t.name,!0),o=Jv(e,t),a=Ls(e,t);return gp(e,o,l,t),Ts(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function qv(e,t,n){const r={};return Ts(r,Ls(e,t)),e.create(t,e.Fragment,r,n)}function Kv(e,t){return t.value}function gp(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Ts(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function Yv(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function Xv(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=_s(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function Gv(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&zs.call(t.properties,i)){const l=Zv(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&$v.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function Jv(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else ii(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else ii(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Ls(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:Fv;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const c=i.get(s)||0;o=s+"-"+c,i.set(s,c+1)}}const a=mp(e,l,o);a!==void 0&&n.push(a)}return n}function Zv(e,t,n){const r=iv(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?Qg(n):sv(n)),r.property==="style"){let i=typeof n=="object"?n:ey(e,String(n));return e.stylePropertyNameCase==="css"&&(i=ty(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?tv[r.property]||r.property:r.attribute,n]}}function ey(e,t){try{return Dv(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new Ie("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=hp+"#cannot-parse-style-attribute",i}}function vp(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=tc(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=tc(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return zs.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);ii(e)}function ii(e,t){const n=new Ie("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=hp+"#cannot-handle-mdx-estrees-without-createevaluater",n}function ty(e){const t={};let n;for(n in e)zs.call(e,n)&&(t[ny(n)]=e[n]);return t}function ny(e){let t=e.replace(Ov,ry);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function ry(e){return"-"+e.toLowerCase()}const mo={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},iy={};function ly(e,t){const n=iy,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return yp(e,r,i)}function yp(e,t,n){if(oy(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return fc(e.children,t,n)}return Array.isArray(e)?fc(e,t,n):""}function fc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=yp(e[i],t,n);return r.join("")}function oy(e){return!!(e&&typeof e=="object")}const pc=document.createElement("i");function Ps(e){const t="&"+e+";";pc.innerHTML=t;const n=pc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function _t(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function st(e,t){return e.length>0?(_t(e,e.length,0,t),e):t}const hc={}.hasOwnProperty;function ay(e){const t={};let n=-1;for(;++n<e.length;)sy(t,e[n]);return t}function sy(e,t){let n;for(n in t){const i=(hc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){hc.call(i,o)||(i[o]=[]);const a=l[o];uy(i[o],Array.isArray(a)?a:a?[a]:[])}}}function uy(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);_t(e,0,0,r)}function xp(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function er(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const jt=pn(/[A-Za-z]/),et=pn(/[\dA-Za-z]/),cy=pn(/[#-'*+\--9=?A-Z^-~]/);function Ca(e){return e!==null&&(e<32||e===127)}const ja=pn(/\d/),dy=pn(/[\dA-Fa-f]/),fy=pn(/[!-/:-@[-`{-~]/);function H(e){return e!==null&&e<-2}function qe(e){return e!==null&&(e<0||e===32)}function ee(e){return e===-2||e===-1||e===32}const py=pn(new RegExp("\\p{P}|\\p{S}","u")),hy=pn(/\s/);function pn(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function fr(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&et(e.charCodeAt(n+1))&&et(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function se(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return ee(s)?(e.enter(n),a(s)):t(s)}function a(s){return ee(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const my={tokenize:gy};function gy(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),se(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return H(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const vy={tokenize:yy},mc={tokenize:xy};function yy(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const b=n[r];return t.containerState=b[1],e.attempt(b[0].continuation,s,c)(y)}return c(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&v();const b=t.events.length;let j=b,k;for(;j--;)if(t.events[j][0]==="exit"&&t.events[j][1].type==="chunkFlow"){k=t.events[j][1].end;break}m(r);let N=b;for(;N<t.events.length;)t.events[N][1].end={...k},N++;return _t(t.events,j+1,0,t.events.slice(b)),t.events.length=N,c(y)}return a(y)}function c(y){if(r===n.length){if(!i)return h(y);if(i.currentConstruct&&i.currentConstruct.concrete)return w(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(mc,d,f)(y)}function d(y){return i&&v(),m(r),h(y)}function f(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,w(y)}function h(y){return t.containerState={},e.attempt(mc,p,w)(y)}function p(y){return r++,n.push([t.currentConstruct,t.containerState]),h(y)}function w(y){if(y===null){i&&v(),m(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),S(y)}function S(y){if(y===null){I(e.exit("chunkFlow"),!0),m(0),e.consume(y);return}return H(y)?(e.consume(y),I(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),S)}function I(y,b){const j=t.sliceStream(y);if(b&&j.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(j),t.parser.lazy[y.start.line]){let k=i.events.length;for(;k--;)if(i.events[k][1].start.offset<o&&(!i.events[k][1].end||i.events[k][1].end.offset>o))return;const N=t.events.length;let T=N,R,P;for(;T--;)if(t.events[T][0]==="exit"&&t.events[T][1].type==="chunkFlow"){if(R){P=t.events[T][1].end;break}R=!0}for(m(r),k=N;k<t.events.length;)t.events[k][1].end={...P},k++;_t(t.events,T+1,0,t.events.slice(N)),t.events.length=k}}function m(y){let b=n.length;for(;b-- >y;){const j=n[b];t.containerState=j[1],j[0].exit.call(t,e)}n.length=y}function v(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function xy(e,t,n){return se(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function gc(e){if(e===null||qe(e)||hy(e))return 1;if(py(e))return 2}function Is(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const Ea={name:"attention",resolveAll:ky,tokenize:wy};function ky(e,t){let n=-1,r,i,l,o,a,s,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const f={...e[r][1].end},h={...e[n][1].start};vc(f,-s),vc(h,s),o={type:s>1?"strongSequence":"emphasisSequence",start:f,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:h},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=st(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=st(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=st(c,Is(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=st(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=st(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,_t(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function wy(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=gc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const c=e.exit("attentionSequence"),d=gc(s),f=!d||d===2&&i||n.includes(s),h=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?f:f&&(i||!h)),c._close=!!(l===42?h:h&&(d||!f)),t(s)}}function vc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Sy={name:"autolink",tokenize:by};function by(e,t,n){let r=0;return i;function i(p){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(p){return jt(p)?(e.consume(p),o):p===64?n(p):c(p)}function o(p){return p===43||p===45||p===46||et(p)?(r=1,a(p)):c(p)}function a(p){return p===58?(e.consume(p),r=0,s):(p===43||p===45||p===46||et(p))&&r++<32?(e.consume(p),a):(r=0,c(p))}function s(p){return p===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):p===null||p===32||p===60||Ca(p)?n(p):(e.consume(p),s)}function c(p){return p===64?(e.consume(p),d):cy(p)?(e.consume(p),c):n(p)}function d(p){return et(p)?f(p):n(p)}function f(p){return p===46?(e.consume(p),r=0,d):p===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):h(p)}function h(p){if((p===45||et(p))&&r++<63){const w=p===45?h:f;return e.consume(p),w}return n(p)}}const Rl={partial:!0,tokenize:Cy};function Cy(e,t,n){return r;function r(l){return ee(l)?se(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||H(l)?t(l):n(l)}}const kp={continuation:{tokenize:Ey},exit:Ny,name:"blockQuote",tokenize:jy};function jy(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return ee(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function Ey(e,t,n){const r=this;return i;function i(o){return ee(o)?se(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(kp,t,n)(o)}}function Ny(e){e.exit("blockQuote")}const wp={name:"characterEscape",tokenize:_y};function _y(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return fy(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const Sp={name:"characterReference",tokenize:zy};function zy(e,t,n){const r=this;let i=0,l,o;return a;function a(f){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),s}function s(f){return f===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(f),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=et,d(f))}function c(f){return f===88||f===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(f),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=dy,d):(e.enter("characterReferenceValue"),l=7,o=ja,d(f))}function d(f){if(f===59&&i){const h=e.exit("characterReferenceValue");return o===et&&!Ps(r.sliceSerialize(h))?n(f):(e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(f)&&i++<l?(e.consume(f),d):n(f)}}const yc={partial:!0,tokenize:Ly},xc={concrete:!0,name:"codeFenced",tokenize:Ty};function Ty(e,t,n){const r=this,i={partial:!0,tokenize:j};let l=0,o=0,a;return s;function s(k){return c(k)}function c(k){const N=r.events[r.events.length-1];return l=N&&N[1].type==="linePrefix"?N[2].sliceSerialize(N[1],!0).length:0,a=k,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(k)}function d(k){return k===a?(o++,e.consume(k),d):o<3?n(k):(e.exit("codeFencedFenceSequence"),ee(k)?se(e,f,"whitespace")(k):f(k))}function f(k){return k===null||H(k)?(e.exit("codeFencedFence"),r.interrupt?t(k):e.check(yc,S,b)(k)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),h(k))}function h(k){return k===null||H(k)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),f(k)):ee(k)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),se(e,p,"whitespace")(k)):k===96&&k===a?n(k):(e.consume(k),h)}function p(k){return k===null||H(k)?f(k):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),w(k))}function w(k){return k===null||H(k)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),f(k)):k===96&&k===a?n(k):(e.consume(k),w)}function S(k){return e.attempt(i,b,I)(k)}function I(k){return e.enter("lineEnding"),e.consume(k),e.exit("lineEnding"),m}function m(k){return l>0&&ee(k)?se(e,v,"linePrefix",l+1)(k):v(k)}function v(k){return k===null||H(k)?e.check(yc,S,b)(k):(e.enter("codeFlowValue"),y(k))}function y(k){return k===null||H(k)?(e.exit("codeFlowValue"),v(k)):(e.consume(k),y)}function b(k){return e.exit("codeFenced"),t(k)}function j(k,N,T){let R=0;return P;function P(U){return k.enter("lineEnding"),k.consume(U),k.exit("lineEnding"),_}function _(U){return k.enter("codeFencedFence"),ee(U)?se(k,D,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(U):D(U)}function D(U){return U===a?(k.enter("codeFencedFenceSequence"),W(U)):T(U)}function W(U){return U===a?(R++,k.consume(U),W):R>=o?(k.exit("codeFencedFenceSequence"),ee(U)?se(k,X,"whitespace")(U):X(U)):T(U)}function X(U){return U===null||H(U)?(k.exit("codeFencedFence"),N(U)):T(U)}}}function Ly(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const go={name:"codeIndented",tokenize:Iy},Py={partial:!0,tokenize:Ay};function Iy(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),se(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?s(c):H(c)?e.attempt(Py,o,s)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||H(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function s(c){return e.exit("codeIndented"),t(c)}}function Ay(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):H(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):se(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):H(o)?i(o):n(o)}}const My={name:"codeText",previous:Ry,resolve:Dy,tokenize:Fy};function Dy(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function Ry(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function Fy(e,t,n){let r=0,i,l;return o;function o(f){return e.enter("codeText"),e.enter("codeTextSequence"),a(f)}function a(f){return f===96?(e.consume(f),r++,a):(e.exit("codeTextSequence"),s(f))}function s(f){return f===null?n(f):f===32?(e.enter("space"),e.consume(f),e.exit("space"),s):f===96?(l=e.enter("codeTextSequence"),i=0,d(f)):H(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),s):(e.enter("codeTextData"),c(f))}function c(f){return f===null||f===32||f===96||H(f)?(e.exit("codeTextData"),s(f)):(e.consume(f),c)}function d(f){return f===96?(e.consume(f),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(f)):(l.type="codeTextData",c(f))}}class Oy{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&br(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),br(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),br(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);br(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);br(this.left,n.reverse())}}}function br(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function bp(e){const t={};let n=-1,r,i,l,o,a,s,c;const d=new Oy(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,By(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return _t(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function By(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],c={};let d,f,h=-1,p=n,w=0,S=0;const I=[S];for(;p;){for(;e.get(++i)[1]!==p;);l.push(i),p._tokenizer||(d=r.sliceStream(p),p.next||d.push(null),f&&o.defineSkip(p.start),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),f=p,p=p.next}for(p=n;++h<a.length;)a[h][0]==="exit"&&a[h-1][0]==="enter"&&a[h][1].type===a[h-1][1].type&&a[h][1].start.line!==a[h][1].end.line&&(S=h+1,I.push(S),p._tokenizer=void 0,p.previous=void 0,p=p.next);for(o.events=[],p?(p._tokenizer=void 0,p.previous=void 0):I.pop(),h=I.length;h--;){const m=a.slice(I[h],I[h+1]),v=l.pop();s.push([v,v+m.length-1]),e.splice(v,2,m)}for(s.reverse(),h=-1;++h<s.length;)c[w+s[h][0]]=w+s[h][1],w+=s[h][1]-s[h][0]-1;return c}const $y={resolve:Hy,tokenize:Vy},Uy={partial:!0,tokenize:Wy};function Hy(e){return bp(e),e}function Vy(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):H(a)?e.check(Uy,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function Wy(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),se(e,l,"linePrefix")}function l(o){if(o===null||H(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function Cp(e,t,n,r,i,l,o,a,s){const c=s||Number.POSITIVE_INFINITY;let d=0;return f;function f(m){return m===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(m),e.exit(l),h):m===null||m===32||m===41||Ca(m)?n(m):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),S(m))}function h(m){return m===62?(e.enter(l),e.consume(m),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),p(m))}function p(m){return m===62?(e.exit("chunkString"),e.exit(a),h(m)):m===null||m===60||H(m)?n(m):(e.consume(m),m===92?w:p)}function w(m){return m===60||m===62||m===92?(e.consume(m),p):p(m)}function S(m){return!d&&(m===null||m===41||qe(m))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(m)):d<c&&m===40?(e.consume(m),d++,S):m===41?(e.consume(m),d--,S):m===null||m===32||m===40||Ca(m)?n(m):(e.consume(m),m===92?I:S)}function I(m){return m===40||m===41||m===92?(e.consume(m),S):S(m)}}function jp(e,t,n,r,i,l){const o=this;let a=0,s;return c;function c(p){return e.enter(r),e.enter(i),e.consume(p),e.exit(i),e.enter(l),d}function d(p){return a>999||p===null||p===91||p===93&&!s||p===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(p):p===93?(e.exit(l),e.enter(i),e.consume(p),e.exit(i),e.exit(r),t):H(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),f(p))}function f(p){return p===null||p===91||p===93||H(p)||a++>999?(e.exit("chunkString"),d(p)):(e.consume(p),s||(s=!ee(p)),p===92?h:f)}function h(p){return p===91||p===92||p===93?(e.consume(p),a++,f):f(p)}}function Ep(e,t,n,r,i,l){let o;return a;function a(h){return h===34||h===39||h===40?(e.enter(r),e.enter(i),e.consume(h),e.exit(i),o=h===40?41:h,s):n(h)}function s(h){return h===o?(e.enter(i),e.consume(h),e.exit(i),e.exit(r),t):(e.enter(l),c(h))}function c(h){return h===o?(e.exit(l),s(o)):h===null?n(h):H(h)?(e.enter("lineEnding"),e.consume(h),e.exit("lineEnding"),se(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(h))}function d(h){return h===o||h===null||H(h)?(e.exit("chunkString"),c(h)):(e.consume(h),h===92?f:d)}function f(h){return h===o||h===92?(e.consume(h),d):d(h)}}function Or(e,t){let n;return r;function r(i){return H(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):ee(i)?se(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const Qy={name:"definition",tokenize:Ky},qy={partial:!0,tokenize:Yy};function Ky(e,t,n){const r=this;let i;return l;function l(p){return e.enter("definition"),o(p)}function o(p){return jp.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(p)}function a(p){return i=er(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),p===58?(e.enter("definitionMarker"),e.consume(p),e.exit("definitionMarker"),s):n(p)}function s(p){return qe(p)?Or(e,c)(p):c(p)}function c(p){return Cp(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(p)}function d(p){return e.attempt(qy,f,f)(p)}function f(p){return ee(p)?se(e,h,"whitespace")(p):h(p)}function h(p){return p===null||H(p)?(e.exit("definition"),r.parser.defined.push(i),t(p)):n(p)}}function Yy(e,t,n){return r;function r(a){return qe(a)?Or(e,i)(a):n(a)}function i(a){return Ep(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return ee(a)?se(e,o,"whitespace")(a):o(a)}function o(a){return a===null||H(a)?t(a):n(a)}}const Xy={name:"hardBreakEscape",tokenize:Gy};function Gy(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return H(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const Jy={name:"headingAtx",resolve:Zy,tokenize:ex};function Zy(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},_t(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function ex(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||qe(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||H(d)?(e.exit("atxHeading"),t(d)):ee(d)?se(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||qe(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const tx=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],kc=["pre","script","style","textarea"],nx={concrete:!0,name:"htmlFlow",resolveTo:lx,tokenize:ox},rx={partial:!0,tokenize:sx},ix={partial:!0,tokenize:ax};function lx(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function ox(e,t,n){const r=this;let i,l,o,a,s;return c;function c(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),f}function f(x){return x===33?(e.consume(x),h):x===47?(e.consume(x),l=!0,S):x===63?(e.consume(x),i=3,r.interrupt?t:g):jt(x)?(e.consume(x),o=String.fromCharCode(x),I):n(x)}function h(x){return x===45?(e.consume(x),i=2,p):x===91?(e.consume(x),i=5,a=0,w):jt(x)?(e.consume(x),i=4,r.interrupt?t:g):n(x)}function p(x){return x===45?(e.consume(x),r.interrupt?t:g):n(x)}function w(x){const ne="CDATA[";return x===ne.charCodeAt(a++)?(e.consume(x),a===ne.length?r.interrupt?t:D:w):n(x)}function S(x){return jt(x)?(e.consume(x),o=String.fromCharCode(x),I):n(x)}function I(x){if(x===null||x===47||x===62||qe(x)){const ne=x===47,be=o.toLowerCase();return!ne&&!l&&kc.includes(be)?(i=1,r.interrupt?t(x):D(x)):tx.includes(o.toLowerCase())?(i=6,ne?(e.consume(x),m):r.interrupt?t(x):D(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?v(x):y(x))}return x===45||et(x)?(e.consume(x),o+=String.fromCharCode(x),I):n(x)}function m(x){return x===62?(e.consume(x),r.interrupt?t:D):n(x)}function v(x){return ee(x)?(e.consume(x),v):P(x)}function y(x){return x===47?(e.consume(x),P):x===58||x===95||jt(x)?(e.consume(x),b):ee(x)?(e.consume(x),y):P(x)}function b(x){return x===45||x===46||x===58||x===95||et(x)?(e.consume(x),b):j(x)}function j(x){return x===61?(e.consume(x),k):ee(x)?(e.consume(x),j):y(x)}function k(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,N):ee(x)?(e.consume(x),k):T(x)}function N(x){return x===s?(e.consume(x),s=null,R):x===null||H(x)?n(x):(e.consume(x),N)}function T(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||qe(x)?j(x):(e.consume(x),T)}function R(x){return x===47||x===62||ee(x)?y(x):n(x)}function P(x){return x===62?(e.consume(x),_):n(x)}function _(x){return x===null||H(x)?D(x):ee(x)?(e.consume(x),_):n(x)}function D(x){return x===45&&i===2?(e.consume(x),Q):x===60&&i===1?(e.consume(x),ie):x===62&&i===4?(e.consume(x),L):x===63&&i===3?(e.consume(x),g):x===93&&i===5?(e.consume(x),E):H(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(rx,$,W)(x)):x===null||H(x)?(e.exit("htmlFlowData"),W(x)):(e.consume(x),D)}function W(x){return e.check(ix,X,$)(x)}function X(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),U}function U(x){return x===null||H(x)?W(x):(e.enter("htmlFlowData"),D(x))}function Q(x){return x===45?(e.consume(x),g):D(x)}function ie(x){return x===47?(e.consume(x),o="",C):D(x)}function C(x){if(x===62){const ne=o.toLowerCase();return kc.includes(ne)?(e.consume(x),L):D(x)}return jt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),C):D(x)}function E(x){return x===93?(e.consume(x),g):D(x)}function g(x){return x===62?(e.consume(x),L):x===45&&i===2?(e.consume(x),g):D(x)}function L(x){return x===null||H(x)?(e.exit("htmlFlowData"),$(x)):(e.consume(x),L)}function $(x){return e.exit("htmlFlow"),t(x)}}function ax(e,t,n){const r=this;return i;function i(o){return H(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function sx(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Rl,t,n)}}const ux={name:"htmlText",tokenize:cx};function cx(e,t,n){const r=this;let i,l,o;return a;function a(g){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(g),s}function s(g){return g===33?(e.consume(g),c):g===47?(e.consume(g),j):g===63?(e.consume(g),y):jt(g)?(e.consume(g),T):n(g)}function c(g){return g===45?(e.consume(g),d):g===91?(e.consume(g),l=0,w):jt(g)?(e.consume(g),v):n(g)}function d(g){return g===45?(e.consume(g),p):n(g)}function f(g){return g===null?n(g):g===45?(e.consume(g),h):H(g)?(o=f,ie(g)):(e.consume(g),f)}function h(g){return g===45?(e.consume(g),p):f(g)}function p(g){return g===62?Q(g):g===45?h(g):f(g)}function w(g){const L="CDATA[";return g===L.charCodeAt(l++)?(e.consume(g),l===L.length?S:w):n(g)}function S(g){return g===null?n(g):g===93?(e.consume(g),I):H(g)?(o=S,ie(g)):(e.consume(g),S)}function I(g){return g===93?(e.consume(g),m):S(g)}function m(g){return g===62?Q(g):g===93?(e.consume(g),m):S(g)}function v(g){return g===null||g===62?Q(g):H(g)?(o=v,ie(g)):(e.consume(g),v)}function y(g){return g===null?n(g):g===63?(e.consume(g),b):H(g)?(o=y,ie(g)):(e.consume(g),y)}function b(g){return g===62?Q(g):y(g)}function j(g){return jt(g)?(e.consume(g),k):n(g)}function k(g){return g===45||et(g)?(e.consume(g),k):N(g)}function N(g){return H(g)?(o=N,ie(g)):ee(g)?(e.consume(g),N):Q(g)}function T(g){return g===45||et(g)?(e.consume(g),T):g===47||g===62||qe(g)?R(g):n(g)}function R(g){return g===47?(e.consume(g),Q):g===58||g===95||jt(g)?(e.consume(g),P):H(g)?(o=R,ie(g)):ee(g)?(e.consume(g),R):Q(g)}function P(g){return g===45||g===46||g===58||g===95||et(g)?(e.consume(g),P):_(g)}function _(g){return g===61?(e.consume(g),D):H(g)?(o=_,ie(g)):ee(g)?(e.consume(g),_):R(g)}function D(g){return g===null||g===60||g===61||g===62||g===96?n(g):g===34||g===39?(e.consume(g),i=g,W):H(g)?(o=D,ie(g)):ee(g)?(e.consume(g),D):(e.consume(g),X)}function W(g){return g===i?(e.consume(g),i=void 0,U):g===null?n(g):H(g)?(o=W,ie(g)):(e.consume(g),W)}function X(g){return g===null||g===34||g===39||g===60||g===61||g===96?n(g):g===47||g===62||qe(g)?R(g):(e.consume(g),X)}function U(g){return g===47||g===62||qe(g)?R(g):n(g)}function Q(g){return g===62?(e.consume(g),e.exit("htmlTextData"),e.exit("htmlText"),t):n(g)}function ie(g){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),C}function C(g){return ee(g)?se(e,E,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(g):E(g)}function E(g){return e.enter("htmlTextData"),o(g)}}const As={name:"labelEnd",resolveAll:hx,resolveTo:mx,tokenize:gx},dx={tokenize:vx},fx={tokenize:yx},px={tokenize:xx};function hx(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&_t(e,0,e.length,n),e}function mx(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",c,t]],a=st(a,e.slice(l+1,l+r+3)),a=st(a,[["enter",d,t]]),a=st(a,Is(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=st(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=st(a,e.slice(o+1)),a=st(a,[["exit",s,t]]),_t(e,l,e.length,a),e}function gx(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(h){return l?l._inactive?f(h):(o=r.parser.defined.includes(er(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(h),e.exit("labelMarker"),e.exit("labelEnd"),s):n(h)}function s(h){return h===40?e.attempt(dx,d,o?d:f)(h):h===91?e.attempt(fx,d,o?c:f)(h):o?d(h):f(h)}function c(h){return e.attempt(px,d,f)(h)}function d(h){return t(h)}function f(h){return l._balanced=!0,n(h)}}function vx(e,t,n){return r;function r(f){return e.enter("resource"),e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),i}function i(f){return qe(f)?Or(e,l)(f):l(f)}function l(f){return f===41?d(f):Cp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(f)}function o(f){return qe(f)?Or(e,s)(f):d(f)}function a(f){return n(f)}function s(f){return f===34||f===39||f===40?Ep(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(f):d(f)}function c(f){return qe(f)?Or(e,d)(f):d(f)}function d(f){return f===41?(e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),e.exit("resource"),t):n(f)}}function yx(e,t,n){const r=this;return i;function i(a){return jp.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(er(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function xx(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const kx={name:"labelStartImage",resolveAll:As.resolveAll,tokenize:wx};function wx(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Sx={name:"labelStartLink",resolveAll:As.resolveAll,tokenize:bx};function bx(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const vo={name:"lineEnding",tokenize:Cx};function Cx(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),se(e,t,"linePrefix")}}const Wi={name:"thematicBreak",tokenize:jx};function jx(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),s(c)):r>=3&&(c===null||H(c))?(e.exit("thematicBreak"),t(c)):n(c)}function s(c){return c===i?(e.consume(c),r++,s):(e.exit("thematicBreakSequence"),ee(c)?se(e,a,"whitespace")(c):a(c))}}const $e={continuation:{tokenize:zx},exit:Lx,name:"list",tokenize:_x},Ex={partial:!0,tokenize:Px},Nx={partial:!0,tokenize:Tx};function _x(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(p){const w=r.containerState.type||(p===42||p===43||p===45?"listUnordered":"listOrdered");if(w==="listUnordered"?!r.containerState.marker||p===r.containerState.marker:ja(p)){if(r.containerState.type||(r.containerState.type=w,e.enter(w,{_container:!0})),w==="listUnordered")return e.enter("listItemPrefix"),p===42||p===45?e.check(Wi,n,c)(p):c(p);if(!r.interrupt||p===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(p)}return n(p)}function s(p){return ja(p)&&++o<10?(e.consume(p),s):(!r.interrupt||o<2)&&(r.containerState.marker?p===r.containerState.marker:p===41||p===46)?(e.exit("listItemValue"),c(p)):n(p)}function c(p){return e.enter("listItemMarker"),e.consume(p),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||p,e.check(Rl,r.interrupt?n:d,e.attempt(Ex,h,f))}function d(p){return r.containerState.initialBlankLine=!0,l++,h(p)}function f(p){return ee(p)?(e.enter("listItemPrefixWhitespace"),e.consume(p),e.exit("listItemPrefixWhitespace"),h):n(p)}function h(p){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(p)}}function zx(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Rl,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,se(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!ee(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Nx,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,se(e,e.attempt($e,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function Tx(e,t,n){const r=this;return se(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function Lx(e){e.exit(this.containerState.type)}function Px(e,t,n){const r=this;return se(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!ee(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const wc={name:"setextUnderline",resolveTo:Ix,tokenize:Ax};function Ix(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function Ax(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,f;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){f=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||f)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),ee(c)?se(e,s,"lineSuffix")(c):s(c))}function s(c){return c===null||H(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const Mx={tokenize:Dx};function Dx(e){const t=this,n=e.attempt(Rl,r,e.attempt(this.parser.constructs.flowInitial,i,se(e,e.attempt(this.parser.constructs.flow,i,e.attempt($y,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const Rx={resolveAll:_p()},Fx=Np("string"),Ox=Np("text");function Np(e){return{resolveAll:_p(e==="text"?Bx:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function c(d){if(d===null)return!0;const f=i[d];let h=-1;if(f)for(;++h<f.length;){const p=f[h];if(!p.previous||p.previous.call(r,r.previous))return!0}return!1}}}function _p(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function Bx(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)s=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const $x={42:$e,43:$e,45:$e,48:$e,49:$e,50:$e,51:$e,52:$e,53:$e,54:$e,55:$e,56:$e,57:$e,62:kp},Ux={91:Qy},Hx={[-2]:go,[-1]:go,32:go},Vx={35:Jy,42:Wi,45:[wc,Wi],60:nx,61:wc,95:Wi,96:xc,126:xc},Wx={38:Sp,92:wp},Qx={[-5]:vo,[-4]:vo,[-3]:vo,33:kx,38:Sp,42:Ea,60:[Sy,ux],91:Sx,92:[Xy,wp],93:As,95:Ea,96:My},qx={null:[Ea,Rx]},Kx={null:[42,95]},Yx={null:[]},Xx=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:Kx,contentInitial:Ux,disable:Yx,document:$x,flow:Vx,flowInitial:Hx,insideSpan:qx,string:Wx,text:Qx},Symbol.toStringTag,{value:"Module"}));function Gx(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:N(j),check:N(k),consume:v,enter:y,exit:b,interrupt:N(k,{interrupt:!0})},c={code:null,containerState:{},defineSkip:S,events:[],now:w,parser:e,previous:null,sliceSerialize:h,sliceStream:p,write:f};let d=t.tokenize.call(c,s);return t.resolveAll&&l.push(t),c;function f(_){return o=st(o,_),I(),o[o.length-1]!==null?[]:(T(t,0),c.events=Is(l,c.events,c),c.events)}function h(_,D){return Zx(p(_),D)}function p(_){return Jx(o,_)}function w(){const{_bufferIndex:_,_index:D,line:W,column:X,offset:U}=r;return{_bufferIndex:_,_index:D,line:W,column:X,offset:U}}function S(_){i[_.line]=_.column,P()}function I(){let _;for(;r._index<o.length;){const D=o[r._index];if(typeof D=="string")for(_=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===_&&r._bufferIndex<D.length;)m(D.charCodeAt(r._bufferIndex));else m(D)}}function m(_){d=d(_)}function v(_){H(_)?(r.line++,r.column=1,r.offset+=_===-3?2:1,P()):_!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=_}function y(_,D){const W=D||{};return W.type=_,W.start=w(),c.events.push(["enter",W,c]),a.push(W),W}function b(_){const D=a.pop();return D.end=w(),c.events.push(["exit",D,c]),D}function j(_,D){T(_,D.from)}function k(_,D){D.restore()}function N(_,D){return W;function W(X,U,Q){let ie,C,E,g;return Array.isArray(X)?$(X):"tokenize"in X?$([X]):L(X);function L(te){return Ae;function Ae(lt){const J=lt!==null&&te[lt],Ce=lt!==null&&te.null,Be=[...Array.isArray(J)?J:J?[J]:[],...Array.isArray(Ce)?Ce:Ce?[Ce]:[]];return $(Be)(lt)}}function $(te){return ie=te,C=0,te.length===0?Q:x(te[C])}function x(te){return Ae;function Ae(lt){return g=R(),E=te,te.partial||(c.currentConstruct=te),te.name&&c.parser.constructs.disable.null.includes(te.name)?be():te.tokenize.call(D?Object.assign(Object.create(c),D):c,s,ne,be)(lt)}}function ne(te){return _(E,g),U}function be(te){return g.restore(),++C<ie.length?x(ie[C]):Q}}}function T(_,D){_.resolveAll&&!l.includes(_)&&l.push(_),_.resolve&&_t(c.events,D,c.events.length-D,_.resolve(c.events.slice(D),c)),_.resolveTo&&(c.events=_.resolveTo(c.events,c))}function R(){const _=w(),D=c.previous,W=c.currentConstruct,X=c.events.length,U=Array.from(a);return{from:X,restore:Q};function Q(){r=_,c.previous=D,c.currentConstruct=W,c.events.length=X,a=U,P()}}function P(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function Jx(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function Zx(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function e1(e){const r={constructs:ay([Xx,...(e||{}).extensions||[]]),content:i(my),defined:[],document:i(vy),flow:i(Mx),lazy:{},string:i(Fx),text:i(Ox)};return r;function i(l){return o;function o(a){return Gx(r,l,a)}}}function t1(e){for(;!bp(e););return e}const Sc=/[\0\t\n\r]/g;function n1(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let c,d,f,h,p;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),f=0,t="",n&&(l.charCodeAt(0)===65279&&f++,n=void 0);f<l.length;){if(Sc.lastIndex=f,c=Sc.exec(l),h=c&&c.index!==void 0?c.index:l.length,p=l.charCodeAt(h),!c){t=l.slice(f);break}if(p===10&&f===h&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),f<h&&(s.push(l.slice(f,h)),e+=h-f),p){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}f=h+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const r1=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function i1(e){return e.replace(r1,l1)}function l1(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return xp(n.slice(l?2:1),l?16:10)}return Ps(n)||e}const zp={}.hasOwnProperty;function o1(e,t,n){return typeof t!="string"&&(n=t,t=void 0),a1(n)(t1(e1(n).document().write(n1()(e,t,!0))))}function a1(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Hs),autolinkProtocol:R,autolinkEmail:R,atxHeading:l(Bs),blockQuote:l(Ce),characterEscape:R,characterReference:R,codeFenced:l(Be),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(Be,o),codeText:l(Ut,o),codeTextData:R,data:R,codeFlowValue:R,definition:l(Ht),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l($p),hardBreakEscape:l($s),hardBreakTrailing:l($s),htmlFlow:l(Us,o),htmlFlowData:R,htmlText:l(Us,o),htmlTextData:R,image:l(Up),label:o,link:l(Hs),listItem:l(Hp),listItemValue:h,listOrdered:l(Vs,f),listUnordered:l(Vs),paragraph:l(Vp),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Bs),strong:l(Wp),thematicBreak:l(qp)},exit:{atxHeading:s(),atxHeadingSequence:j,autolink:s(),autolinkEmail:J,autolinkProtocol:lt,blockQuote:s(),characterEscapeValue:P,characterReferenceMarkerHexadecimal:be,characterReferenceMarkerNumeric:be,characterReferenceValue:te,characterReference:Ae,codeFenced:s(I),codeFencedFence:S,codeFencedFenceInfo:p,codeFencedFenceMeta:w,codeFlowValue:P,codeIndented:s(m),codeText:s(U),codeTextData:P,data:P,definition:s(),definitionDestinationString:b,definitionLabelString:v,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(D),hardBreakTrailing:s(D),htmlFlow:s(W),htmlFlowData:P,htmlText:s(X),htmlTextData:P,image:s(ie),label:E,labelText:C,lineEnding:_,link:s(Q),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:ne,resourceDestinationString:g,resourceTitleString:L,resource:$,setextHeading:s(T),setextHeadingLineSequence:N,setextHeadingText:k,strong:s(),thematicBreak:s()}};Tp(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(z){let F={type:"root",children:[]};const V={stack:[F],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},G=[];let le=-1;for(;++le<z.length;)if(z[le][1].type==="listOrdered"||z[le][1].type==="listUnordered")if(z[le][0]==="enter")G.push(le);else{const pt=G.pop();le=i(z,pt,le)}for(le=-1;++le<z.length;){const pt=t[z[le][0]];zp.call(pt,z[le][1].type)&&pt[z[le][1].type].call(Object.assign({sliceSerialize:z[le][2].sliceSerialize},V),z[le][1])}if(V.tokenStack.length>0){const pt=V.tokenStack[V.tokenStack.length-1];(pt[1]||bc).call(V,void 0,pt[0])}for(F.position={start:Wt(z.length>0?z[0][1].start:{line:1,column:1,offset:0}),end:Wt(z.length>0?z[z.length-2][1].end:{line:1,column:1,offset:0})},le=-1;++le<t.transforms.length;)F=t.transforms[le](F)||F;return F}function i(z,F,V){let G=F-1,le=-1,pt=!1,hn,zt,pr,hr;for(;++G<=V;){const Ye=z[G];switch(Ye[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Ye[0]==="enter"?le++:le--,hr=void 0;break}case"lineEndingBlank":{Ye[0]==="enter"&&(hn&&!hr&&!le&&!pr&&(pr=G),hr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:hr=void 0}if(!le&&Ye[0]==="enter"&&Ye[1].type==="listItemPrefix"||le===-1&&Ye[0]==="exit"&&(Ye[1].type==="listUnordered"||Ye[1].type==="listOrdered")){if(hn){let Ln=G;for(zt=void 0;Ln--;){const Tt=z[Ln];if(Tt[1].type==="lineEnding"||Tt[1].type==="lineEndingBlank"){if(Tt[0]==="exit")continue;zt&&(z[zt][1].type="lineEndingBlank",pt=!0),Tt[1].type="lineEnding",zt=Ln}else if(!(Tt[1].type==="linePrefix"||Tt[1].type==="blockQuotePrefix"||Tt[1].type==="blockQuotePrefixWhitespace"||Tt[1].type==="blockQuoteMarker"||Tt[1].type==="listItemIndent"))break}pr&&(!zt||pr<zt)&&(hn._spread=!0),hn.end=Object.assign({},zt?z[zt][1].start:Ye[1].end),z.splice(zt||G,0,["exit",hn,Ye[2]]),G++,V++}if(Ye[1].type==="listItemPrefix"){const Ln={type:"listItem",_spread:!1,start:Object.assign({},Ye[1].start),end:void 0};hn=Ln,z.splice(G,0,["enter",Ln,Ye[2]]),G++,V++,pr=void 0,hr=!0}}}return z[F][1]._spread=pt,V}function l(z,F){return V;function V(G){a.call(this,z(G),G),F&&F.call(this,G)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(z,F,V){this.stack[this.stack.length-1].children.push(z),this.stack.push(z),this.tokenStack.push([F,V||void 0]),z.position={start:Wt(F.start),end:void 0}}function s(z){return F;function F(V){z&&z.call(this,V),c.call(this,V)}}function c(z,F){const V=this.stack.pop(),G=this.tokenStack.pop();if(G)G[0].type!==z.type&&(F?F.call(this,z,G[0]):(G[1]||bc).call(this,z,G[0]));else throw new Error("Cannot close `"+z.type+"` ("+Fr({start:z.start,end:z.end})+"): it’s not open");V.position.end=Wt(z.end)}function d(){return ly(this.stack.pop())}function f(){this.data.expectingFirstListItemValue=!0}function h(z){if(this.data.expectingFirstListItemValue){const F=this.stack[this.stack.length-2];F.start=Number.parseInt(this.sliceSerialize(z),10),this.data.expectingFirstListItemValue=void 0}}function p(){const z=this.resume(),F=this.stack[this.stack.length-1];F.lang=z}function w(){const z=this.resume(),F=this.stack[this.stack.length-1];F.meta=z}function S(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function I(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function m(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z.replace(/(\r?\n|\r)$/g,"")}function v(z){const F=this.resume(),V=this.stack[this.stack.length-1];V.label=F,V.identifier=er(this.sliceSerialize(z)).toLowerCase()}function y(){const z=this.resume(),F=this.stack[this.stack.length-1];F.title=z}function b(){const z=this.resume(),F=this.stack[this.stack.length-1];F.url=z}function j(z){const F=this.stack[this.stack.length-1];if(!F.depth){const V=this.sliceSerialize(z).length;F.depth=V}}function k(){this.data.setextHeadingSlurpLineEnding=!0}function N(z){const F=this.stack[this.stack.length-1];F.depth=this.sliceSerialize(z).codePointAt(0)===61?1:2}function T(){this.data.setextHeadingSlurpLineEnding=void 0}function R(z){const V=this.stack[this.stack.length-1].children;let G=V[V.length-1];(!G||G.type!=="text")&&(G=Qp(),G.position={start:Wt(z.start),end:void 0},V.push(G)),this.stack.push(G)}function P(z){const F=this.stack.pop();F.value+=this.sliceSerialize(z),F.position.end=Wt(z.end)}function _(z){const F=this.stack[this.stack.length-1];if(this.data.atHardBreak){const V=F.children[F.children.length-1];V.position.end=Wt(z.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(F.type)&&(R.call(this,z),P.call(this,z))}function D(){this.data.atHardBreak=!0}function W(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function X(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function U(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function Q(){const z=this.stack[this.stack.length-1];if(this.data.inReference){const F=this.data.referenceType||"shortcut";z.type+="Reference",z.referenceType=F,delete z.url,delete z.title}else delete z.identifier,delete z.label;this.data.referenceType=void 0}function ie(){const z=this.stack[this.stack.length-1];if(this.data.inReference){const F=this.data.referenceType||"shortcut";z.type+="Reference",z.referenceType=F,delete z.url,delete z.title}else delete z.identifier,delete z.label;this.data.referenceType=void 0}function C(z){const F=this.sliceSerialize(z),V=this.stack[this.stack.length-2];V.label=i1(F),V.identifier=er(F).toLowerCase()}function E(){const z=this.stack[this.stack.length-1],F=this.resume(),V=this.stack[this.stack.length-1];if(this.data.inReference=!0,V.type==="link"){const G=z.children;V.children=G}else V.alt=F}function g(){const z=this.resume(),F=this.stack[this.stack.length-1];F.url=z}function L(){const z=this.resume(),F=this.stack[this.stack.length-1];F.title=z}function $(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function ne(z){const F=this.resume(),V=this.stack[this.stack.length-1];V.label=F,V.identifier=er(this.sliceSerialize(z)).toLowerCase(),this.data.referenceType="full"}function be(z){this.data.characterReferenceType=z.type}function te(z){const F=this.sliceSerialize(z),V=this.data.characterReferenceType;let G;V?(G=xp(F,V==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):G=Ps(F);const le=this.stack[this.stack.length-1];le.value+=G}function Ae(z){const F=this.stack.pop();F.position.end=Wt(z.end)}function lt(z){P.call(this,z);const F=this.stack[this.stack.length-1];F.url=this.sliceSerialize(z)}function J(z){P.call(this,z);const F=this.stack[this.stack.length-1];F.url="mailto:"+this.sliceSerialize(z)}function Ce(){return{type:"blockquote",children:[]}}function Be(){return{type:"code",lang:null,meta:null,value:""}}function Ut(){return{type:"inlineCode",value:""}}function Ht(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function $p(){return{type:"emphasis",children:[]}}function Bs(){return{type:"heading",depth:0,children:[]}}function $s(){return{type:"break"}}function Us(){return{type:"html",value:""}}function Up(){return{type:"image",title:null,url:"",alt:null}}function Hs(){return{type:"link",title:null,url:"",children:[]}}function Vs(z){return{type:"list",ordered:z.type==="listOrdered",start:null,spread:z._spread,children:[]}}function Hp(z){return{type:"listItem",spread:z._spread,checked:null,children:[]}}function Vp(){return{type:"paragraph",children:[]}}function Wp(){return{type:"strong",children:[]}}function Qp(){return{type:"text",value:""}}function qp(){return{type:"thematicBreak"}}}function Wt(e){return{line:e.line,column:e.column,offset:e.offset}}function Tp(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Tp(e,r):s1(e,r)}}function s1(e,t){let n;for(n in t)if(zp.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function bc(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Fr({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Fr({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Fr({start:t.start,end:t.end})+") is still open")}function u1(e){const t=this;t.parser=n;function n(r){return o1(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function c1(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function d1(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function f1(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function p1(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function h1(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function m1(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=fr(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const c={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,c),e.applyData(t,c)}function g1(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function v1(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Lp(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function y1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Lp(e,t);const i={src:fr(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function x1(e,t){const n={src:fr(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function k1(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function w1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Lp(e,t);const i={href:fr(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function S1(e,t){const n={href:fr(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function b1(e,t,n){const r=e.all(t),i=n?C1(n):Pp(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let f;d&&d.type==="element"&&d.tagName==="p"?f=d:(f={type:"element",tagName:"p",properties:{},children:[]},r.unshift(f)),f.children.length>0&&f.children.unshift({type:"text",value:" "}),f.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function C1(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Pp(n[r])}return t}function Pp(e){const t=e.spread;return t??e.children.length>1}function j1(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function E1(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function N1(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function _1(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function z1(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=_s(t.children[1]),s=fp(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function T1(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const c=[];for(;++s<a;){const f=t.children[s],h={},p=o?o[s]:void 0;p&&(h.align=p);let w={type:"element",tagName:l,properties:h,children:[]};f&&(w.children=e.all(f),e.patch(f,w),w=e.applyData(f,w)),c.push(w)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function L1(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const Cc=9,jc=32;function P1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(Ec(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(Ec(t.slice(i),i>0,!1)),l.join("")}function Ec(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===Cc||l===jc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===Cc||l===jc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function I1(e,t){const n={type:"text",value:P1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function A1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const M1={blockquote:c1,break:d1,code:f1,delete:p1,emphasis:h1,footnoteReference:m1,heading:g1,html:v1,imageReference:y1,image:x1,inlineCode:k1,linkReference:w1,link:S1,listItem:b1,list:j1,paragraph:E1,root:N1,strong:_1,table:z1,tableCell:L1,tableRow:T1,text:I1,thematicBreak:A1,toml:Ti,yaml:Ti,definition:Ti,footnoteDefinition:Ti};function Ti(){}const Ip=-1,Fl=0,Br=1,yl=2,Ms=3,Ds=4,Rs=5,Fs=6,Ap=7,Mp=8,Nc=typeof self=="object"?self:globalThis,D1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Fl:case Ip:return n(o,i);case Br:{const a=n([],i);for(const s of o)a.push(r(s));return a}case yl:{const a=n({},i);for(const[s,c]of o)a[r(s)]=r(c);return a}case Ms:return n(new Date(o),i);case Ds:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Rs:{const a=n(new Map,i);for(const[s,c]of o)a.set(r(s),r(c));return a}case Fs:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Ap:{const{name:a,message:s}=o;return n(new Nc[a](s),i)}case Mp:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new Nc[l](o),i)};return r},_c=e=>D1(new Map,e)(0),An="",{toString:R1}={},{keys:F1}=Object,Cr=e=>{const t=typeof e;if(t!=="object"||!e)return[Fl,t];const n=R1.call(e).slice(8,-1);switch(n){case"Array":return[Br,An];case"Object":return[yl,An];case"Date":return[Ms,An];case"RegExp":return[Ds,An];case"Map":return[Rs,An];case"Set":return[Fs,An];case"DataView":return[Br,n]}return n.includes("Array")?[Br,n]:n.includes("Error")?[Ap,n]:[yl,n]},Li=([e,t])=>e===Fl&&(t==="function"||t==="symbol"),O1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=Cr(o);switch(a){case Fl:{let d=o;switch(s){case"bigint":a=Mp,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([Ip],o)}return i([a,d],o)}case Br:{if(s){let h=o;return s==="DataView"?h=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(h=new Uint8Array(o)),i([s,[...h]],o)}const d=[],f=i([a,d],o);for(const h of o)d.push(l(h));return f}case yl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],f=i([a,d],o);for(const h of F1(o))(e||!Li(Cr(o[h])))&&d.push([l(h),l(o[h])]);return f}case Ms:return i([a,o.toISOString()],o);case Ds:{const{source:d,flags:f}=o;return i([a,{source:d,flags:f}],o)}case Rs:{const d=[],f=i([a,d],o);for(const[h,p]of o)(e||!(Li(Cr(h))||Li(Cr(p))))&&d.push([l(h),l(p)]);return f}case Fs:{const d=[],f=i([a,d],o);for(const h of o)(e||!Li(Cr(h)))&&d.push(l(h));return f}}const{message:c}=o;return i([a,{name:s,message:c}],o)};return l},zc=(e,{json:t,lossy:n}={})=>{const r=[];return O1(!(t||n),!!t,new Map,r)(e),r},xl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?_c(zc(e,t)):structuredClone(e):(e,t)=>_c(zc(e,t));function B1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function $1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function U1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||B1,r=e.options.footnoteBackLabel||$1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[s]);if(!c)continue;const d=e.all(c),f=String(c.identifier).toUpperCase(),h=fr(f.toLowerCase());let p=0;const w=[],S=e.footnoteCounts.get(f);for(;S!==void 0&&++p<=S;){w.length>0&&w.push({type:"text",value:" "});let v=typeof n=="string"?n:n(s,p);typeof v=="string"&&(v={type:"text",value:v}),w.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+h+(p>1?"-"+p:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,p),className:["data-footnote-backref"]},children:Array.isArray(v)?v:[v]})}const I=d[d.length-1];if(I&&I.type==="element"&&I.tagName==="p"){const v=I.children[I.children.length-1];v&&v.type==="text"?v.value+=" ":I.children.push({type:"text",value:" "}),I.children.push(...w)}else d.push(...w);const m={type:"element",tagName:"li",properties:{id:t+"fn-"+h},children:e.wrap(d,!0)};e.patch(c,m),a.push(m)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...xl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const Dp=function(e){if(e==null)return Q1;if(typeof e=="function")return Ol(e);if(typeof e=="object")return Array.isArray(e)?H1(e):V1(e);if(typeof e=="string")return W1(e);throw new Error("Expected function, string, or object as test")};function H1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=Dp(e[n]);return Ol(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function V1(e){const t=e;return Ol(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function W1(e){return Ol(t);function t(n){return n&&n.type===e}}function Ol(e){return t;function t(n,r,i){return!!(q1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function Q1(){return!0}function q1(e){return e!==null&&typeof e=="object"&&"type"in e}const Rp=[],K1=!0,Tc=!1,Y1="skip";function X1(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=Dp(i),o=r?-1:1;a(e,void 0,[])();function a(s,c,d){const f=s&&typeof s=="object"?s:{};if(typeof f.type=="string"){const p=typeof f.tagName=="string"?f.tagName:typeof f.name=="string"?f.name:void 0;Object.defineProperty(h,"name",{value:"node ("+(s.type+(p?"<"+p+">":""))+")"})}return h;function h(){let p=Rp,w,S,I;if((!t||l(s,c,d[d.length-1]||void 0))&&(p=G1(n(s,d)),p[0]===Tc))return p;if("children"in s&&s.children){const m=s;if(m.children&&p[0]!==Y1)for(S=(r?m.children.length:-1)+o,I=d.concat(m);S>-1&&S<m.children.length;){const v=m.children[S];if(w=a(v,S,I)(),w[0]===Tc)return w;S=typeof w[1]=="number"?w[1]:S+o}}return p}}}function G1(e){return Array.isArray(e)?e:typeof e=="number"?[K1,e]:e==null?Rp:[e]}function Fp(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),X1(e,l,a,i);function a(s,c){const d=c[c.length-1],f=d?d.children.indexOf(s):void 0;return o(s,f,d)}}const Na={}.hasOwnProperty,J1={};function Z1(e,t){const n=t||J1,r=new Map,i=new Map,l=new Map,o={...M1,...n.handlers},a={all:c,applyData:t0,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:e0,wrap:r0};return Fp(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const f=d.type==="definition"?r:i,h=String(d.identifier).toUpperCase();f.has(h)||f.set(h,d)}}),a;function s(d,f){const h=d.type,p=a.handlers[h];if(Na.call(a.handlers,h)&&p)return p(a,d,f);if(a.options.passThrough&&a.options.passThrough.includes(h)){if("children"in d){const{children:S,...I}=d,m=xl(I);return m.children=a.all(d),m}return xl(d)}return(a.options.unknownHandler||n0)(a,d,f)}function c(d){const f=[];if("children"in d){const h=d.children;let p=-1;for(;++p<h.length;){const w=a.one(h[p],d);if(w){if(p&&h[p-1].type==="break"&&(!Array.isArray(w)&&w.type==="text"&&(w.value=Lc(w.value)),!Array.isArray(w)&&w.type==="element")){const S=w.children[0];S&&S.type==="text"&&(S.value=Lc(S.value))}Array.isArray(w)?f.push(...w):f.push(w)}}}return f}}function e0(e,t){e.position&&(t.position=Rv(e))}function t0(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,xl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function n0(e,t){const n=t.data||{},r="value"in t&&!(Na.call(n,"hProperties")||Na.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function r0(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Lc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Pc(e,t){const n=Z1(e,t),r=n.one(e,void 0),i=U1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function i0(e,t){return e&&"run"in e?async function(n,r){const i=Pc(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Pc(n,{file:r,...e||t})}}function Ic(e){if(e)throw e}var Qi=Object.prototype.hasOwnProperty,Op=Object.prototype.toString,Ac=Object.defineProperty,Mc=Object.getOwnPropertyDescriptor,Dc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Op.call(t)==="[object Array]"},Rc=function(t){if(!t||Op.call(t)!=="[object Object]")return!1;var n=Qi.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Qi.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Qi.call(t,i)},Fc=function(t,n){Ac&&n.name==="__proto__"?Ac(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Oc=function(t,n){if(n==="__proto__")if(Qi.call(t,n)){if(Mc)return Mc(t,n).value}else return;return t[n]},l0=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<c;++s)if(t=arguments[s],t!=null)for(n in t)r=Oc(a,n),i=Oc(t,n),a!==i&&(d&&i&&(Rc(i)||(l=Dc(i)))?(l?(l=!1,o=r&&Dc(r)?r:[]):o=r&&Rc(r)?r:{},Fc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Fc(a,{name:n,newValue:i}));return a};const yo=Ta(l0);function _a(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function o0(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...c){const d=e[++l];let f=-1;if(s){o(s);return}for(;++f<i.length;)(c[f]===null||c[f]===void 0)&&(c[f]=i[f]);i=c,d?a0(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function a0(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const bt={basename:s0,dirname:u0,extname:c0,join:d0,sep:"/"};function s0(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');di(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function u0(e){if(di(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function c0(e){di(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function d0(...e){let t=-1,n;for(;++t<e.length;)di(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":f0(n)}function f0(e){di(e);const t=e.codePointAt(0)===47;let n=p0(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function p0(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function di(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const h0={cwd:m0};function m0(){return"/"}function za(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function g0(e){if(typeof e=="string")e=new URL(e);else if(!za(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return v0(e)}function v0(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const xo=["history","path","basename","stem","extname","dirname"];class Bp{constructor(t){let n;t?za(t)?n={path:t}:typeof t=="string"||y0(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":h0.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<xo.length;){const l=xo[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)xo.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?bt.basename(this.path):void 0}set basename(t){wo(t,"basename"),ko(t,"basename"),this.path=bt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?bt.dirname(this.path):void 0}set dirname(t){Bc(this.basename,"dirname"),this.path=bt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?bt.extname(this.path):void 0}set extname(t){if(ko(t,"extname"),Bc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=bt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){za(t)&&(t=g0(t)),wo(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?bt.basename(this.path,this.extname):void 0}set stem(t){wo(t,"stem"),ko(t,"stem"),this.path=bt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new Ie(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function ko(e,t){if(e&&e.includes(bt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+bt.sep+"`")}function wo(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Bc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function y0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const x0=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},k0={}.hasOwnProperty;class Os extends x0{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=o0()}copy(){const t=new Os;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(yo(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(Co("data",this.frozen),this.namespace[t]=n,this):k0.call(this.namespace,t)&&this.namespace[t]||void 0:t?(Co("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Pi(t),r=this.parser||this.Parser;return So("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),So("process",this.parser||this.Parser),bo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Pi(t),s=r.parse(a);r.run(s,a,function(d,f,h){if(d||!f||!h)return c(d);const p=f,w=r.stringify(p,h);b0(w)?h.value=w:h.result=w,c(d,h)});function c(d,f){d||!f?o(d):l?l(f):n(void 0,f)}}}processSync(t){let n=!1,r;return this.freeze(),So("processSync",this.parser||this.Parser),bo("processSync",this.compiler||this.Compiler),this.process(t,i),Uc("processSync","process",n),r;function i(l,o){n=!0,Ic(l),r=o}}run(t,n,r){$c(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Pi(n);i.run(t,s,c);function c(d,f,h){const p=f||t;d?a(d):o?o(p):r(void 0,p,h)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Uc("runSync","run",r),i;function l(o,a){Ic(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Pi(n),i=this.compiler||this.Compiler;return bo("stringify",i),$c(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(Co("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")s(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...f]=c;s(d,f)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=yo(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const f=c[d];l(f)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function s(c,d){let f=-1,h=-1;for(;++f<r.length;)if(r[f][0]===c){h=f;break}if(h===-1)r.push([c,...d]);else if(d.length>0){let[p,...w]=d;const S=r[h][1];_a(S)&&_a(p)&&(p=yo(!0,S,p)),r[h]=[c,p,...w]}}}}const w0=new Os().freeze();function So(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function bo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function Co(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function $c(e){if(!_a(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Uc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Pi(e){return S0(e)?e:new Bp(e)}function S0(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function b0(e){return typeof e=="string"||C0(e)}function C0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const j0="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Hc=[],Vc={allowDangerousHtml:!0},E0=/^(https?|ircs?|mailto|xmpp)$/i,N0=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function _0(e){const t=z0(e),n=T0(e);return L0(t.runSync(t.parse(n),n),e)}function z0(e){const t=e.rehypePlugins||Hc,n=e.remarkPlugins||Hc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Vc}:Vc;return w0().use(u1).use(n).use(i0,r).use(t)}function T0(e){const t=e.children||"",n=new Bp;return typeof t=="string"&&(n.value=t),n}function L0(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||P0;for(const d of N0)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+j0+d.id,void 0);return Fp(e,c),Uv(e,{Fragment:u.Fragment,components:i,ignoreInvalidStyle:!0,jsx:u.jsx,jsxs:u.jsxs,passKeys:!0,passNode:!0});function c(d,f,h){if(d.type==="raw"&&h&&typeof f=="number")return o?h.children.splice(f,1):h.children[f]={type:"text",value:d.value},f;if(d.type==="element"){let p;for(p in mo)if(Object.hasOwn(mo,p)&&Object.hasOwn(d.properties,p)){const w=d.properties[p],S=mo[p];(S===null||S.includes(d.tagName))&&(d.properties[p]=s(String(w||""),p,d))}}if(d.type==="element"){let p=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!p&&r&&typeof f=="number"&&(p=!r(d,f,h)),p&&h&&typeof f=="number")return a&&d.children?h.children.splice(f,1,...d.children):h.children.splice(f,1),f}}}function P0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||E0.test(e.slice(0,t))?e:""}const I0=e=>{if(!e)return null;try{return JSON.parse(e).execution_stats||null}catch{return null}},A0=e=>{if(e.kind!=="status")return!1;const t=e.content.toLowerCase();return t.includes("running")||t.includes("thinking")||t.includes("executing")||t.includes("processing")},Wc=10*1024,jo=200,Te={send:u.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),u.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),u.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),u.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),file:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),spinner:u.jsx("svg",{className:"spinner-icon",width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 12a9 9 0 1 1-6.219-8.56"})})},M0=e=>{switch(e){case"directive":return Te.directive;case"question":return Te.question;case"status":return Te.status;case"result":return Te.result;case"approval_request":return Te.lock;default:return Te.directive}},D0=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=O.useRef(null),[a,s]=Qt.useState(""),[c,d]=Qt.useState("directive"),[f,h]=Qt.useState(""),[p,w]=Qt.useState(!1),[S,I]=Qt.useState(new Map),[m,v]=Qt.useState(new Set),[y,b]=O.useState(new Set),[j,k]=O.useState(new Set),N=C=>{const E=(C.match(/\n/g)||[]).length+1;if(!(C.length>Wc||E>jo))return{needsTruncation:!1,truncated:C,fullLength:C.length,lineCount:E};let L=C.slice(0,Wc);const $=L.split(`
`);$.length>jo&&(L=$.slice(0,jo).join(`
`));const x=L.lastIndexOf(`
`);return x>L.length*.8&&(L=L.slice(0,x)),{needsTruncation:!0,truncated:L,fullLength:C.length,lineCount:E}},T=C=>{b(E=>{const g=new Set(E);return g.has(C)?g.delete(C):g.add(C),g})};O.useEffect(()=>{e!=null&&e.workspace?h(e.workspace):h("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),O.useEffect(()=>{var C;(C=o.current)==null||C.scrollIntoView({behavior:"smooth"})},[t]);const R=C=>{h(C),r&&r(C)},P=()=>{a.trim()&&(n(a,c,f||void 0),s(""))},_=C=>{C.key==="Enter"&&!C.shiftKey&&(C.preventDefault(),P())},D=C=>new Date(C).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),W=C=>C.length>12?`${C.slice(0,8)}...`:C,X=C=>{if(!C.metadata_json)return null;try{return JSON.parse(C.metadata_json).approval_id||null}catch{return null}},U=C=>{const E=S.get(C)||"";i&&(i(C,E),v(g=>new Set(g).add(C)),I(g=>{const L=new Map(g);return L.delete(C),L}))},Q=C=>{const E=S.get(C)||"";if(!E.trim()){alert("Please provide a reason for rejection");return}l&&(l(C,E),v(g=>new Set(g).add(C)),I(g=>{const L=new Map(g);return L.delete(C),L}))},ie=(C,E)=>{I(g=>new Map(g).set(C,E))};return e?u.jsxs("div",{className:"conversation-view",children:[u.jsxs("div",{className:"conversation-header",children:[u.jsxs("div",{className:"header-info",children:[u.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&u.jsxs("span",{className:"thread-agent-badge",children:[Te.bot,e.target_agent]})]}),u.jsxs("div",{className:"header-stats",children:[u.jsxs("span",{className:"message-count",children:[t.length," messages"]}),u.jsx("span",{className:"thread-id",title:e.id,children:W(e.id)})]})]}),u.jsxs("div",{className:"messages-container",children:[t.length===0?u.jsxs("div",{className:"empty-messages",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("p",{children:"No messages yet"}),u.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((C,E)=>{const g=C.from_type==="human",L=E===0||t[E-1].from_type!==C.from_type,$=y.has(C.id),{needsTruncation:x,truncated:ne,fullLength:be,lineCount:te}=N(C.content),Ae=$?C.content:ne,lt=A0(C);return u.jsxs("div",{className:`message ${g?"human":"agent"}${lt?" running-status":""}`,children:[u.jsx("div",{className:`message-avatar ${L?"visible":""}`,children:L&&(g?Te.user:Te.bot)}),u.jsxs("div",{className:"message-body",children:[L&&u.jsxs("div",{className:"message-meta",children:[u.jsx("span",{className:"sender-name",children:C.from_id}),u.jsxs("span",{className:`kind-badge${lt?" running":""}`,children:[lt?Te.spinner:M0(C.kind)," ",C.kind]}),u.jsx("span",{className:"message-time",children:D(C.created_at)})]}),u.jsxs("div",{className:"message-content",children:[C.kind==="result"||!g?u.jsx(_0,{components:{a:({href:J,children:Ce})=>{let Be=J;return J&&J.startsWith("/")&&!J.startsWith("//")&&(Be=`file://${J}`),u.jsx("a",{href:Be,target:"_blank",rel:"noopener noreferrer",children:Ce})},code:({className:J,children:Ce,...Be})=>!J?u.jsx("code",{className:"inline-code",...Be,children:Ce}):u.jsx("code",{className:J,...Be,children:Ce})},children:Ae}):Ae,x&&u.jsx("div",{className:"truncation-notice",children:u.jsx("button",{className:"expand-btn",onClick:()=>T(C.id),children:$?u.jsx(u.Fragment,{children:"Show less"}):u.jsxs(u.Fragment,{children:["Show more (",Math.round(be/1024),"KB, ",te," lines)"]})})}),C.kind==="approval_request"&&(()=>{const J=X(C),Ce=J&&m.has(J);return J?u.jsx("div",{className:"inline-approval",children:Ce?u.jsxs("div",{className:"approval-handled",children:[Te.check,u.jsx("span",{children:"Action taken"})]}):u.jsxs(u.Fragment,{children:[u.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:S.get(J)||"",onChange:Be=>ie(J,Be.target.value)}),u.jsxs("div",{className:"approval-actions",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>Q(J),title:"Reject",children:[Te.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>U(J),title:"Approve",children:[Te.check,"Approve"]})]})]})}):null})(),C.kind==="result"&&(()=>{const J=I0(C.metadata_json);if(!J||!J.files_created||J.files_created.length===0)return null;const Ce=j.has(C.id),Be=()=>{k(Ut=>{const Ht=new Set(Ut);return Ht.has(C.id)?Ht.delete(C.id):Ht.add(C.id),Ht})};return u.jsxs("div",{className:"files-created-section",children:[u.jsxs("button",{className:`files-toggle-btn ${Ce?"expanded":""}`,onClick:Be,children:[Te.file,u.jsxs("span",{children:["Files Created (",J.files_created.length,")"]}),J.workspace&&u.jsxs("span",{className:"workspace-badge",title:J.workspace,children:[Te.folder,J.workspace.split("/").pop()]}),u.jsx("span",{className:"toggle-chevron",children:Ce?"▼":"▶"})]}),Ce&&u.jsx("ul",{className:"files-list",children:J.files_created.map((Ut,Ht)=>u.jsx("li",{className:"file-item",children:u.jsx("a",{href:`file://${J.workspace?J.workspace+"/":""}${Ut}`,target:"_blank",rel:"noopener noreferrer",title:Ut,children:Ut})},Ht))})]})})()]}),u.jsxs("div",{className:"message-footer",children:[u.jsxs("span",{className:"message-seq",children:["#",C.message_seq]}),C.delivery_state!=="acked"&&u.jsx("span",{className:`delivery-status ${C.delivery_state}`,children:C.delivery_state==="pending"?"sending...":"delivered"})]})]})]},C.id)}),u.jsx("div",{ref:o})]}),u.jsxs("div",{className:"input-area",children:[p&&u.jsxs("div",{className:"workspace-input-row",children:[u.jsx("input",{type:"text",value:f,onChange:C=>R(C.target.value),onBlur:()=>{r&&r(f)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),u.jsx("button",{onClick:async()=>{try{const E=await(await fetch("/api/select-folder")).json();!E.cancelled&&E.path&&R(E.path)}catch(C){console.error("Failed to open folder picker:",C)}},className:"workspace-browse",title:"Browse for folder",children:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),u.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),u.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),f&&u.jsx("button",{onClick:()=>{R(""),w(!1)},className:"workspace-clear",children:"Clear"})]}),u.jsxs("div",{className:"input-wrapper",children:[u.jsx("button",{onClick:()=>w(!p),className:`workspace-toggle ${f?"has-workspace":""}`,title:f||"Set working directory",children:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),u.jsxs("select",{value:c,onChange:C=>d(C.target.value),className:"kind-selector",children:[u.jsx("option",{value:"directive",children:"Directive"}),u.jsx("option",{value:"question",children:"Question"})]}),u.jsx("textarea",{value:a,onChange:C=>s(C.target.value),onKeyPress:_,placeholder:f?`Message (workspace: ${f.split("/").pop()})`:"Type a message...",rows:1}),u.jsx("button",{onClick:P,className:"send-btn",disabled:!a.trim(),children:Te.send})]}),u.jsxs("div",{className:"input-hint",children:["Press ",u.jsx("kbd",{children:"Enter"})," to send, ",u.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),u.jsx("style",{children:`
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
      `})]}):null},R0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,reconnectInterval:l=5e3})=>{const o=O.useRef(null),[a,s]=O.useState(!1),[c,d]=O.useState(null),f=O.useRef(null),h=O.useRef(new Map),p=O.useCallback(()=>{try{const b=`${e}?instance_id=${t}`;o.current=new WebSocket(b),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),d(null),h.current.forEach((j,k)=>{I(k,j)})},o.current.onmessage=j=>{try{const k=JSON.parse(j.data);w(k)}catch(k){console.error("Failed to parse WebSocket message:",k)}},o.current.onerror=j=>{console.error("WebSocket error:",j),d("Connection error")},o.current.onclose=()=>{console.log("WebSocket disconnected"),s(!1),f.current=setTimeout(()=>{console.log("Attempting to reconnect..."),p()},l)}}catch(b){console.error("Failed to connect to WebSocket:",b),d("Failed to connect")}},[e,t,l]),w=O.useCallback(b=>{switch(b.type){case"message":n&&b.data&&n(b.data);break;case"batch":if(r&&b.data){const j=b.data;r(j),n&&j.messages.forEach(k=>n(k))}break;case"error":i&&b.data&&i(b.data),console.error("WebSocket error event:",b.data);break;case"pong":break;default:console.log("Unknown event type:",b.type)}},[n,r,i]),S=O.useCallback(b=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(b)):console.warn("WebSocket not connected, cannot send event")},[]),I=O.useCallback((b,j=0)=>{h.current.set(b,j);const k={type:"subscribe",timestamp:Date.now(),data:{thread_id:b,from_seq:j}};S(k)},[S]),m=O.useCallback((b,j)=>{const k=h.current.get(b)||0;j>k&&h.current.set(b,j);const N={type:"ack",timestamp:Date.now(),data:{thread_id:b,ack_seq:j}};S(N)},[S]),v=O.useCallback(()=>{const b={type:"ping",timestamp:Date.now()};S(b)},[S]),y=O.useCallback(b=>{h.current.delete(b)},[]);return O.useEffect(()=>(p(),()=>{f.current&&clearTimeout(f.current),o.current&&o.current.close()}),[p]),O.useEffect(()=>{if(!a)return;const b=setInterval(()=>{v()},3e4);return()=>clearInterval(b)},[a,v]),{isConnected:a,connectionError:c,subscribe:I,unsubscribe:y,acknowledge:m,ping:v}},F0=({connected:e})=>u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?u.jsxs(u.Fragment,{children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):u.jsxs(u.Fragment,{children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),u.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),O0=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=O.useState([]),[o,a]=O.useState(null),[s,c]=O.useState(new Map),[d,f]=O.useState(new Map),[h,p]=O.useState([]),[w,S]=O.useState(!1),[I,m]=O.useState(""),{isConnected:v,subscribe:y,acknowledge:b}=R0({url:e,instanceId:t,onMessage:j,onBatch:k});function j(E){const g={id:E.id,thread_id:E.thread_id,message_seq:E.message_seq,created_at:E.created_at,from_type:E.from_type,from_id:E.from_id,to_type:E.to_type,to_id:E.to_id,kind:E.kind,subject:E.subject,content:E.content,metadata_json:E.metadata_json,delivery_state:"visible",business_state:"open"};c(L=>{const $=L.get(g.thread_id)||[];return $.find(x=>x.id===g.id)?L:new Map(L).set(g.thread_id,[...$,g].sort((x,ne)=>x.message_seq-ne.message_seq))}),g.thread_id!==o&&f(L=>{const $=L.get(g.thread_id)||0;return new Map(L).set(g.thread_id,$+1)}),b(g.thread_id,g.message_seq)}function k(E){E.messages.forEach(g=>{j(g)})}const N=O.useCallback(E=>{if(a(E),f(g=>{const L=new Map(g);return L.delete(E),L}),v){const g=s.get(E)||[],L=g.length>0?Math.max(...g.map($=>$.message_seq)):0;y(E,L)}},[v,y,s]),T=O.useCallback(async(E,g,L)=>{if(!o)return;const $=L?JSON.stringify({workspace:L}):void 0;try{const x=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:g,content:E,metadata_json:$})});if(!x.ok){console.error("Failed to send message:",await x.text());return}const ne=await x.json();c(be=>{const te=be.get(o)||[];return te.find(Ae=>Ae.id===ne.id)?be:new Map(be).set(o,[...te,ne])})}catch(x){console.error("Error sending message:",x)}},[o,t]);O.useEffect(()=>{(async()=>{try{const g=await fetch("/api/threads");if(!g.ok){console.error("Failed to fetch threads:",await g.text());return}const L=await g.json();l(L),L.length>0&&!o&&a(L[0].id)}catch(g){console.error("Error fetching threads:",g)}})()},[]),O.useEffect(()=>{n&&i.length>0&&(i.some(g=>g.id===n)&&(a(n),f(g=>{const L=new Map(g);return L.delete(n),L})),r&&r())},[n,i,r]);const R=O.useCallback(async E=>{try{const g=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:E,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!g.ok){console.error("Failed to create thread:",await g.text());return}const L=await g.json();l($=>[L,...$]),a(L.id)}catch(g){console.error("Error creating thread:",g)}},[t]),P=O.useCallback(async()=>{try{const E=await fetch("/api/agents");if(!E.ok){console.error("Failed to fetch agents:",await E.text());return}const g=await E.json();p(g.running||[])}catch(E){console.error("Error fetching agents:",E)}},[]);O.useEffect(()=>{P();const E=setInterval(P,5e3);return()=>clearInterval(E)},[P]);const _=O.useCallback(async()=>{if(I.trim())try{const E=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:I.trim()})});if(!E.ok){const L=await E.text();console.error("Failed to launch agent:",L),alert(`Failed to launch agent: ${L}`);return}const g=await E.json();p(L=>[...L,g]),m(""),S(!1)}catch(E){console.error("Error launching agent:",E)}},[I]),D=O.useCallback(async E=>{try{const g=await fetch(`/api/agents/${E}`,{method:"DELETE"});if(!g.ok){console.error("Failed to stop agent:",await g.text());return}p(L=>L.filter($=>$.instance_id!==E))}catch(g){console.error("Error stopping agent:",g)}},[]),W=O.useCallback(async E=>{if(o)try{const g=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:E})});if(!g.ok){console.error("Failed to update workspace:",await g.text());return}const L=await g.json();l($=>$.map(x=>x.id===o?L:x))}catch(g){console.error("Error updating workspace:",g)}},[o]),X=O.useCallback(async E=>{try{const g=await fetch(`/api/threads/${E}`,{method:"DELETE"});if(!g.ok){console.error("Failed to delete thread:",await g.text());return}l(L=>L.filter($=>$.id!==E)),c(L=>{const $=new Map(L);return $.delete(E),$}),f(L=>{const $=new Map(L);return $.delete(E),$}),o===E&&a(null)}catch(g){console.error("Error deleting thread:",g)}},[o]),U=O.useCallback(async(E,g)=>{try{const L=await fetch(`/api/threads/${E}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:g})});if(!L.ok){console.error("Failed to rename thread:",await L.text());return}const $=await L.json();l(x=>x.map(ne=>ne.id===E?$:ne))}catch(L){console.error("Error renaming thread:",L)}},[]),Q=O.useCallback(async(E,g)=>{try{const L=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!L.ok){const $=await L.text();console.error("Failed to approve request:",$),alert(`Failed to approve: ${$}`);return}console.log("Approval approved successfully")}catch(L){console.error("Error approving request:",L)}},[]),ie=O.useCallback(async(E,g)=>{try{const L=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!L.ok){const $=await L.text();console.error("Failed to reject request:",$),alert(`Failed to reject: ${$}`);return}console.log("Approval rejected successfully")}catch(L){console.error("Error rejecting request:",L)}},[]),C=o?s.get(o)||[]:[];return u.jsxs("div",{className:"message-center",children:[u.jsxs("div",{className:"status-bar",children:[u.jsxs("div",{className:`status-indicator ${v?"connected":"disconnected"}`,children:[u.jsx(F0,{connected:v}),u.jsx("span",{children:v?"Connected":"Disconnected"})]}),u.jsxs("div",{className:"status-meta",children:[u.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),u.jsxs("span",{className:"agent-count",children:[h.length," agents"]}),u.jsx("button",{className:"launch-agent-btn",onClick:()=>S(!0),children:"+ Agent"})]})]}),h.length>0&&u.jsx("div",{className:"agents-bar",children:h.map(E=>u.jsxs("div",{className:"agent-chip",children:[u.jsx("span",{className:"agent-pulse"}),u.jsx("span",{className:"agent-name",children:E.instance_id}),u.jsxs("span",{className:"agent-pid",children:["PID ",E.pid]}),u.jsx("button",{className:"agent-stop-btn",onClick:()=>D(E.instance_id),title:"Stop agent",children:"×"})]},E.instance_id))}),w&&u.jsx("div",{className:"modal-overlay",onClick:()=>S(!1),children:u.jsxs("div",{className:"modal-content",onClick:E=>E.stopPropagation(),children:[u.jsx("h3",{children:"Launch New Agent"}),u.jsx("input",{type:"text",value:I,onChange:E=>m(E.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:E=>{E.key==="Enter"&&_(),E.key==="Escape"&&S(!1)}}),u.jsxs("div",{className:"modal-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>S(!1),children:"Cancel"}),u.jsx("button",{className:"launch-btn",onClick:_,children:"Launch"})]})]})}),u.jsxs("div",{className:"center-layout",children:[u.jsx("aside",{className:"threads-panel",children:u.jsx(Wg,{threads:i,selectedThreadId:o,onSelectThread:N,onCreateThread:R,onDeleteThread:X,onRenameThread:U,unreadCounts:d})}),u.jsx("main",{className:"conversation-panel",children:o?u.jsx(D0,{thread:i.find(E=>E.id===o),messages:C,onSendMessage:T,onWorkspaceChange:W,onApproveRequest:Q,onRejectRequest:ie}):u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("h3",{children:"Select a conversation"}),u.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),u.jsx("style",{children:`
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
      `})]})},Me={check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),dollar:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:u.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),u.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),u.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},B0=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=O.useState(!0),[a,s]=O.useState(null),[c,d]=O.useState(new Map),f=m=>{try{return JSON.parse(m)}catch{return null}},h=m=>new Date(m).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),p=m=>{const v=c.get(m)||"";n(m,v),d(new Map(c.set(m,"")))},w=m=>{const v=c.get(m)||"";if(!v.trim()){alert("Please provide a reason for rejection");return}r(m,v),d(new Map(c.set(m,"")))},S=(m,v)=>{d(new Map(c.set(m,v)))},I=e.filter(m=>m.status==="pending");return u.jsxs("div",{className:"approval-queue",children:[u.jsx("div",{className:"queue-header",children:u.jsxs("div",{className:"header-title",children:[u.jsx("h2",{children:"Approval Queue"}),u.jsxs("span",{className:"pending-count",children:[I.length," pending"]})]})}),u.jsxs("div",{className:"approvals-container",children:[I.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Me.sparkles}),u.jsx("h3",{children:"All caught up!"}),u.jsx("p",{children:"No pending approvals to review"})]}):u.jsx("div",{className:"approvals-list",children:I.map(m=>{const v=f(m.effect_delta_json),y=a===m.id;return u.jsxs("div",{className:`approval-card impact-${m.impact}`,children:[u.jsxs("div",{className:"card-header",onClick:()=>s(y?null:m.id),children:[u.jsxs("div",{className:"header-left",children:[u.jsx("div",{className:`impact-indicator ${m.impact}`}),u.jsxs("div",{className:"proposal-info",children:[u.jsx("span",{className:"proposal-text",children:m.proposal}),u.jsxs("div",{className:"proposal-meta",children:[m.thread_title&&u.jsxs("span",{className:"meta-item thread-link",onClick:b=>{b.stopPropagation(),i==null||i(m.thread_id)},title:"Go to thread",children:[Me.message,m.thread_title]}),u.jsxs("span",{className:"meta-item",children:[Me.bot,m.instance_id]}),u.jsxs("span",{className:"meta-item",children:[Me.clock,h(m.created_at)]})]})]})]}),u.jsxs("div",{className:"header-right",children:[u.jsxs("span",{className:"cost-badge",children:[Me.dollar,"$",m.estimated_cost.toFixed(2)]}),u.jsx("span",{className:`impact-badge ${m.impact}`,children:m.impact}),u.jsx("button",{className:"expand-btn",children:y?Me.chevronUp:Me.chevronDown})]})]}),y&&u.jsxs("div",{className:"card-details",children:[v&&u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Effect Details"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Capability"}),u.jsx("span",{className:"detail-value code",children:v.cap_type})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Budget Delta"}),u.jsxs("span",{className:"detail-value",children:["$",v.budget_delta.toFixed(2)]})]}),v.paths.length>0&&u.jsxs("div",{className:"detail-item full-width",children:[u.jsx("span",{className:"detail-label",children:"Paths"}),u.jsx("div",{className:"paths-list",children:v.paths.map((b,j)=>u.jsxs("span",{className:"path-tag",children:[Me.folder,b]},j))})]})]})]}),u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Request Info"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Thread"}),u.jsx("span",{className:"detail-value code",children:m.thread_id})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Impact Level"}),u.jsx("span",{className:`detail-value impact-text ${m.impact}`,children:m.impact.toUpperCase()})]})]})]}),u.jsxs("div",{className:"review-section",children:[u.jsx("h4",{children:"Review Notes"}),u.jsx("textarea",{value:c.get(m.id)||"",onChange:b=>S(m.id,b.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),u.jsxs("div",{className:"action-buttons",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>w(m.id),children:[Me.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>p(m.id),children:[Me.check,"Approve"]})]})]})]})]},m.id)})}),t.length>0&&u.jsxs("div",{className:"history-section",children:[u.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[u.jsxs("h3",{children:[l?Me.chevronDown:Me.chevronUp,"Review History"]}),u.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&u.jsx("div",{className:"history-list",children:t.map(m=>{const v=a===`history-${m.id}`;return u.jsxs("div",{className:`history-card ${m.status}`,onClick:()=>s(v?null:`history-${m.id}`),children:[u.jsxs("div",{className:"history-card-header",children:[u.jsxs("div",{className:"history-status",children:[u.jsx("span",{className:`status-icon ${m.status}`,children:m.status==="approved"?Me.check:Me.x}),u.jsxs("div",{className:"history-info",children:[u.jsx("span",{className:"history-proposal",children:m.proposal}),m.thread_title&&u.jsxs("span",{className:"history-thread",onClick:y=>{y.stopPropagation(),i==null||i(m.thread_id)},title:"Go to thread",children:[Me.message,m.thread_title]})]})]}),u.jsxs("div",{className:"history-meta",children:[u.jsx("span",{className:"history-agent",children:m.instance_id}),u.jsx("span",{className:`history-badge ${m.status}`,children:m.status}),u.jsx("span",{className:"history-time",children:m.reviewed_at?h(m.reviewed_at):h(m.created_at)})]})]}),v&&u.jsxs("div",{className:"history-details",children:[u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Reviewed by"}),u.jsx("span",{className:"detail-value",children:m.reviewed_by||"Unknown"})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Cost"}),u.jsxs("span",{className:"detail-value",children:["$",m.estimated_cost.toFixed(2)]})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Impact"}),u.jsx("span",{className:`detail-value impact-text ${m.impact}`,children:m.impact.toUpperCase()})]}),m.review_notes&&u.jsxs("div",{className:"detail-row full-width",children:[u.jsx("span",{className:"detail-label",children:"Notes"}),u.jsx("span",{className:"detail-value notes",children:m.review_notes})]})]})]},`history-${m.id}`)})})]})]}),u.jsx("style",{children:`
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
      `})]})},$0=u.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M12 6v12M6 12h12"}),u.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]}),U0=()=>{const[e,t]=O.useState({type:"overview"}),[n,r]=O.useState(null),[i,l]=O.useState([]),[o,a]=O.useState([]),[s,c]=O.useState(!1),[d,f]=O.useState(""),p=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;O.useEffect(()=>{const j=async()=>{try{const N=await fetch("/api/hierarchy");if(N.ok){const T=await N.json();r(T)}}catch(N){console.error("Error fetching hierarchy:",N)}};j();const k=setInterval(j,5e3);return()=>clearInterval(k)},[]),O.useEffect(()=>{const j=async()=>{try{const N=await fetch("/api/approvals?status=pending");if(N.ok){const _=await N.json();l(_)}const[T,R]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),P=[];if(T.ok){const _=await T.json();P.push(..._)}if(R.ok){const _=await R.json();P.push(..._)}P.sort((_,D)=>{const W=_.reviewed_at?new Date(_.reviewed_at).getTime():0;return(D.reviewed_at?new Date(D.reviewed_at).getTime():0)-W}),a(P)}catch(N){console.error("Error fetching approvals:",N)}};j();const k=setInterval(j,5e3);return()=>clearInterval(k)},[]);const w=async(j,k)=>{try{const N=await fetch(`/api/approvals/${j}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:k})});if(!N.ok){console.error("Failed to approve:",await N.text());return}const T=i.find(R=>R.id===j);if(T){const R={...T,status:"approved",reviewed_by:"user",review_notes:k,reviewed_at:Date.now()};a(P=>[R,...P])}l(R=>R.filter(P=>P.id!==j))}catch(N){console.error("Error approving:",N)}},S=async(j,k)=>{try{const N=await fetch(`/api/approvals/${j}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:k})});if(!N.ok){console.error("Failed to reject:",await N.text());return}const T=i.find(R=>R.id===j);if(T){const R={...T,status:"rejected",reviewed_by:"user",review_notes:k,reviewed_at:Date.now()};a(P=>[R,...P])}l(R=>R.filter(P=>P.id!==j))}catch(N){console.error("Error rejecting:",N)}},I=()=>{var k,N;const j=[{label:"All Agents",onClick:()=>t({type:"overview"})}];if(e.type==="agent"&&e.agentId&&j.push({label:e.agentId}),e.type==="thread"&&e.threadId){e.agentId&&j.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})});const T=(k=n==null?void 0:n.root.children)==null?void 0:k.find(P=>P.id===e.agentId),R=(N=T==null?void 0:T.children)==null?void 0:N.find(P=>P.id===e.threadId);j.push({label:(R==null?void 0:R.label)||"Thread"})}return j},m=j=>{var N;const k=(N=n==null?void 0:n.root.children)==null?void 0:N.find(T=>{var R;return(R=T.children)==null?void 0:R.some(P=>P.id===j)});t({type:"thread",agentId:k==null?void 0:k.id,threadId:j})},v=async j=>{if(d.trim())try{const k=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:d.trim(),created_by_type:"human",created_by_id:"user",target_agent:j})});if(!k.ok){console.error("Failed to create thread:",await k.text());return}const N=await k.json();f(""),c(!1),t({type:"thread",agentId:j,threadId:N.id})}catch(k){console.error("Error creating thread:",k)}},y=()=>{var j,k,N;if(e.type==="overview"&&n)return u.jsx(Hg,{aggregate:n.aggregate,agents:n.root.children||[],onSelectAgent:T=>t({type:"agent",agentId:T})});if(e.type==="agent"&&e.agentId){const T=(j=n==null?void 0:n.root.children)==null?void 0:j.find(P=>P.id===e.agentId),R=i.filter(P=>{var _;return(_=T==null?void 0:T.children)==null?void 0:_.some(D=>D.id===P.thread_id)});return u.jsxs("div",{className:"agent-view",children:[u.jsxs("div",{className:"agent-view-header",children:[u.jsx("h2",{children:e.agentId}),u.jsxs("span",{className:"agent-thread-count",children:[((k=T==null?void 0:T.children)==null?void 0:k.length)||0," threads"]})]}),u.jsxs("div",{className:"agent-view-content",children:[u.jsxs("div",{className:"agent-threads",children:[u.jsxs("div",{className:"threads-header",children:[u.jsx("h3",{children:"Threads"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>c(!0),title:"New thread",children:"+ New Thread"})]}),s&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:d,onChange:P=>f(P.target.value),onKeyDown:P=>{P.key==="Enter"&&v(e.agentId),P.key==="Escape"&&(c(!1),f(""))},placeholder:"Thread title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{onClick:()=>{c(!1),f("")},children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:()=>v(e.agentId),children:"Create"})]})]}),(N=T==null?void 0:T.children)==null?void 0:N.map(P=>u.jsxs("div",{className:"thread-card",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:P.id}),children:[u.jsx("span",{className:"thread-title",children:P.label}),P.badges&&P.badges.length>0&&u.jsx("span",{className:"thread-badges",children:P.badges.map((_,D)=>u.jsx("span",{className:`badge badge-${_.type}`,children:_.count},D))})]},P.id)),(!(T!=null&&T.children)||T.children.length===0)&&!s&&u.jsxs("div",{className:"no-threads",children:["No threads yet",u.jsx("button",{className:"start-thread-btn",onClick:()=>c(!0),children:"Start a conversation"})]})]}),R.length>0&&u.jsxs("div",{className:"agent-approvals",children:[u.jsx("h3",{children:"Pending Approvals"}),u.jsx(B0,{approvals:R,history:[],onApprove:w,onReject:S,onNavigateToThread:m})]})]})]})}return e.type==="thread"&&e.threadId?u.jsx(O0,{websocketUrl:p,instanceId:e.agentId||"default",initialThreadId:e.threadId,onThreadNavigated:()=>{}}):u.jsx("div",{className:"empty-state",children:u.jsx("p",{children:"Select an agent or thread from the sidebar"})})},b=(i==null?void 0:i.filter(j=>j.status==="pending").length)||0;return u.jsxs("div",{className:"app",children:[u.jsxs("header",{className:"app-header",children:[u.jsxs("div",{className:"header-brand",children:[u.jsx("div",{className:"brand-logo",children:$0}),u.jsxs("div",{className:"brand-text",children:[u.jsx("h1",{children:"AILANG"}),u.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),u.jsxs("div",{className:"header-meta",children:[b>0&&u.jsxs("span",{className:"pending-badge",title:`${b} pending approvals`,children:[b," pending"]}),u.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),u.jsxs("div",{className:"app-body",children:[u.jsx("aside",{className:"app-sidebar",children:u.jsx(wg,{selection:e,onSelect:t})}),u.jsxs("main",{className:"app-main",children:[e.type!=="overview"&&u.jsx(Vg,{items:I()}),u.jsx("div",{className:"main-content",children:y()})]})]}),u.jsx("style",{children:`
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
      `})]})};Eo.createRoot(document.getElementById("root")).render(u.jsx(Qt.StrictMode,{children:u.jsx(U0,{})}));
