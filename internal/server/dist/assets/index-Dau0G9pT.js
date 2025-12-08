(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Xi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Ma(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Gc={exports:{}},bl={},Jc={exports:{}},Y={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var ai=Symbol.for("react.element"),nh=Symbol.for("react.portal"),rh=Symbol.for("react.fragment"),ih=Symbol.for("react.strict_mode"),lh=Symbol.for("react.profiler"),oh=Symbol.for("react.provider"),ah=Symbol.for("react.context"),sh=Symbol.for("react.forward_ref"),uh=Symbol.for("react.suspense"),ch=Symbol.for("react.memo"),dh=Symbol.for("react.lazy"),Ys=Symbol.iterator;function fh(e){return e===null||typeof e!="object"?null:(e=Ys&&e[Ys]||e["@@iterator"],typeof e=="function"?e:null)}var Zc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},ed=Object.assign,td={};function cr(e,t,n){this.props=e,this.context=t,this.refs=td,this.updater=n||Zc}cr.prototype.isReactComponent={};cr.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};cr.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function nd(){}nd.prototype=cr.prototype;function Aa(e,t,n){this.props=e,this.context=t,this.refs=td,this.updater=n||Zc}var Da=Aa.prototype=new nd;Da.constructor=Aa;ed(Da,cr.prototype);Da.isPureReactComponent=!0;var Xs=Array.isArray,rd=Object.prototype.hasOwnProperty,Ra={current:null},id={key:!0,ref:!0,__self:!0,__source:!0};function ld(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)rd.call(t,r)&&!id.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),c=0;c<a;c++)s[c]=arguments[c+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:ai,type:e,key:l,ref:o,props:i,_owner:Ra.current}}function ph(e,t){return{$$typeof:ai,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function Fa(e){return typeof e=="object"&&e!==null&&e.$$typeof===ai}function hh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Gs=/\/+/g;function Vl(e,t){return typeof e=="object"&&e!==null&&e.key!=null?hh(""+e.key):t.toString(36)}function Di(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case ai:case nh:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Vl(o,0):r,Xs(i)?(n="",e!=null&&(n=e.replace(Gs,"$&/")+"/"),Di(i,t,n,"",function(c){return c})):i!=null&&(Fa(i)&&(i=ph(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Gs,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Xs(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Vl(l,a);o+=Di(l,t,n,s,i)}else if(s=fh(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Vl(l,a++),o+=Di(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function hi(e,t,n){if(e==null)return e;var r=[],i=0;return Di(e,r,"","",function(l){return t.call(n,l,i++)}),r}function mh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Oe={current:null},Ri={transition:null},gh={ReactCurrentDispatcher:Oe,ReactCurrentBatchConfig:Ri,ReactCurrentOwner:Ra};function od(){throw Error("act(...) is not supported in production builds of React.")}Y.Children={map:hi,forEach:function(e,t,n){hi(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return hi(e,function(){t++}),t},toArray:function(e){return hi(e,function(t){return t})||[]},only:function(e){if(!Fa(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};Y.Component=cr;Y.Fragment=rh;Y.Profiler=lh;Y.PureComponent=Aa;Y.StrictMode=ih;Y.Suspense=uh;Y.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=gh;Y.act=od;Y.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=ed({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Ra.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)rd.call(t,s)&&!id.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var c=0;c<s;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:ai,type:e.type,key:i,ref:l,props:r,_owner:o}};Y.createContext=function(e){return e={$$typeof:ah,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:oh,_context:e},e.Consumer=e};Y.createElement=ld;Y.createFactory=function(e){var t=ld.bind(null,e);return t.type=e,t};Y.createRef=function(){return{current:null}};Y.forwardRef=function(e){return{$$typeof:sh,render:e}};Y.isValidElement=Fa;Y.lazy=function(e){return{$$typeof:dh,_payload:{_status:-1,_result:e},_init:mh}};Y.memo=function(e,t){return{$$typeof:ch,type:e,compare:t===void 0?null:t}};Y.startTransition=function(e){var t=Ri.transition;Ri.transition={};try{e()}finally{Ri.transition=t}};Y.unstable_act=od;Y.useCallback=function(e,t){return Oe.current.useCallback(e,t)};Y.useContext=function(e){return Oe.current.useContext(e)};Y.useDebugValue=function(){};Y.useDeferredValue=function(e){return Oe.current.useDeferredValue(e)};Y.useEffect=function(e,t){return Oe.current.useEffect(e,t)};Y.useId=function(){return Oe.current.useId()};Y.useImperativeHandle=function(e,t,n){return Oe.current.useImperativeHandle(e,t,n)};Y.useInsertionEffect=function(e,t){return Oe.current.useInsertionEffect(e,t)};Y.useLayoutEffect=function(e,t){return Oe.current.useLayoutEffect(e,t)};Y.useMemo=function(e,t){return Oe.current.useMemo(e,t)};Y.useReducer=function(e,t,n){return Oe.current.useReducer(e,t,n)};Y.useRef=function(e){return Oe.current.useRef(e)};Y.useState=function(e){return Oe.current.useState(e)};Y.useSyncExternalStore=function(e,t,n){return Oe.current.useSyncExternalStore(e,t,n)};Y.useTransition=function(){return Oe.current.useTransition()};Y.version="18.3.1";Jc.exports=Y;var O=Jc.exports;const Kt=Ma(O);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var vh=O,yh=Symbol.for("react.element"),xh=Symbol.for("react.fragment"),kh=Object.prototype.hasOwnProperty,wh=vh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,Sh={key:!0,ref:!0,__self:!0,__source:!0};function ad(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)kh.call(t,r)&&!Sh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:yh,type:e,key:l,ref:o,props:i,_owner:wh.current}}bl.Fragment=xh;bl.jsx=ad;bl.jsxs=ad;Gc.exports=bl;var u=Gc.exports,To={},sd={exports:{}},it={},ud={exports:{}},cd={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(j,N){var g=j.length;j.push(N);e:for(;0<g;){var L=g-1>>>1,$=j[L];if(0<i($,N))j[L]=N,j[g]=$,g=L;else break e}}function n(j){return j.length===0?null:j[0]}function r(j){if(j.length===0)return null;var N=j[0],g=j.pop();if(g!==N){j[0]=g;e:for(var L=0,$=j.length,x=$>>>1;L<x;){var ne=2*(L+1)-1,be=j[ne],te=ne+1,Ae=j[te];if(0>i(be,g))te<$&&0>i(Ae,be)?(j[L]=Ae,j[te]=g,L=te):(j[L]=be,j[ne]=g,L=ne);else if(te<$&&0>i(Ae,g))j[L]=Ae,j[te]=g,L=te;else break e}}return N}function i(j,N){var g=j.sortIndex-N.sortIndex;return g!==0?g:j.id-N.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],c=[],d=1,f=null,m=3,p=!1,w=!1,S=!1,I=typeof setTimeout=="function"?setTimeout:null,h=typeof clearTimeout=="function"?clearTimeout:null,v=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(j){for(var N=n(c);N!==null;){if(N.callback===null)r(c);else if(N.startTime<=j)r(c),N.sortIndex=N.expirationTime,t(s,N);else break;N=n(c)}}function b(j){if(S=!1,y(j),!w)if(n(s)!==null)w=!0,Q(E);else{var N=n(c);N!==null&&ie(b,N.startTime-j)}}function E(j,N){w=!1,S&&(S=!1,h(_),_=-1),p=!0;var g=m;try{for(y(N),f=n(s);f!==null&&(!(f.expirationTime>N)||j&&!T());){var L=f.callback;if(typeof L=="function"){f.callback=null,m=f.priorityLevel;var $=L(f.expirationTime<=N);N=e.unstable_now(),typeof $=="function"?f.callback=$:f===n(s)&&r(s),y(N)}else r(s);f=n(s)}if(f!==null)var x=!0;else{var ne=n(c);ne!==null&&ie(b,ne.startTime-N),x=!1}return x}finally{f=null,m=g,p=!1}}var k=!1,C=null,_=-1,R=5,P=-1;function T(){return!(e.unstable_now()-P<R)}function D(){if(C!==null){var j=e.unstable_now();P=j;var N=!0;try{N=C(!0,j)}finally{N?W():(k=!1,C=null)}}else k=!1}var W;if(typeof v=="function")W=function(){v(D)};else if(typeof MessageChannel<"u"){var X=new MessageChannel,U=X.port2;X.port1.onmessage=D,W=function(){U.postMessage(null)}}else W=function(){I(D,0)};function Q(j){C=j,k||(k=!0,W())}function ie(j,N){_=I(function(){j(e.unstable_now())},N)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(j){j.callback=null},e.unstable_continueExecution=function(){w||p||(w=!0,Q(E))},e.unstable_forceFrameRate=function(j){0>j||125<j?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):R=0<j?Math.floor(1e3/j):5},e.unstable_getCurrentPriorityLevel=function(){return m},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(j){switch(m){case 1:case 2:case 3:var N=3;break;default:N=m}var g=m;m=N;try{return j()}finally{m=g}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(j,N){switch(j){case 1:case 2:case 3:case 4:case 5:break;default:j=3}var g=m;m=j;try{return N()}finally{m=g}},e.unstable_scheduleCallback=function(j,N,g){var L=e.unstable_now();switch(typeof g=="object"&&g!==null?(g=g.delay,g=typeof g=="number"&&0<g?L+g:L):g=L,j){case 1:var $=-1;break;case 2:$=250;break;case 5:$=1073741823;break;case 4:$=1e4;break;default:$=5e3}return $=g+$,j={id:d++,callback:N,priorityLevel:j,startTime:g,expirationTime:$,sortIndex:-1},g>L?(j.sortIndex=g,t(c,j),n(s)===null&&j===n(c)&&(S?(h(_),_=-1):S=!0,ie(b,g-L))):(j.sortIndex=$,t(s,j),w||p||(w=!0,Q(E))),j},e.unstable_shouldYield=T,e.unstable_wrapCallback=function(j){var N=m;return function(){var g=m;m=N;try{return j.apply(this,arguments)}finally{m=g}}}})(cd);ud.exports=cd;var bh=ud.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var jh=O,rt=bh;function M(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var dd=new Set,Vr={};function zn(e,t){rr(e,t),rr(e+"Capture",t)}function rr(e,t){for(Vr[e]=t,e=0;e<t.length;e++)dd.add(t[e])}var Ot=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),zo=Object.prototype.hasOwnProperty,Ch=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,Js={},Zs={};function Eh(e){return zo.call(Zs,e)?!0:zo.call(Js,e)?!1:Ch.test(e)?Zs[e]=!0:(Js[e]=!0,!1)}function Nh(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function _h(e,t,n,r){if(t===null||typeof t>"u"||Nh(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Be(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var _e={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){_e[e]=new Be(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];_e[t]=new Be(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){_e[e]=new Be(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){_e[e]=new Be(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){_e[e]=new Be(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){_e[e]=new Be(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){_e[e]=new Be(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){_e[e]=new Be(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){_e[e]=new Be(e,5,!1,e.toLowerCase(),null,!1,!1)});var Oa=/[\-:]([a-z])/g;function Ba(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(Oa,Ba);_e[t]=new Be(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(Oa,Ba);_e[t]=new Be(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(Oa,Ba);_e[t]=new Be(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){_e[e]=new Be(e,1,!1,e.toLowerCase(),null,!1,!1)});_e.xlinkHref=new Be("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){_e[e]=new Be(e,1,!1,e.toLowerCase(),null,!0,!0)});function $a(e,t,n,r){var i=_e.hasOwnProperty(t)?_e[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(_h(t,n,i,r)&&(n=null),r||i===null?Eh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Vt=jh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,mi=Symbol.for("react.element"),Rn=Symbol.for("react.portal"),Fn=Symbol.for("react.fragment"),Ua=Symbol.for("react.strict_mode"),Lo=Symbol.for("react.profiler"),fd=Symbol.for("react.provider"),pd=Symbol.for("react.context"),Va=Symbol.for("react.forward_ref"),Po=Symbol.for("react.suspense"),Io=Symbol.for("react.suspense_list"),Ha=Symbol.for("react.memo"),Yt=Symbol.for("react.lazy"),hd=Symbol.for("react.offscreen"),eu=Symbol.iterator;function vr(e){return e===null||typeof e!="object"?null:(e=eu&&e[eu]||e["@@iterator"],typeof e=="function"?e:null)}var he=Object.assign,Hl;function Nr(e){if(Hl===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Hl=t&&t[1]||""}return`
`+Hl+e}var Wl=!1;function Ql(e,t){if(!e||Wl)return"";Wl=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Wl=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?Nr(e):""}function Th(e){switch(e.tag){case 5:return Nr(e.type);case 16:return Nr("Lazy");case 13:return Nr("Suspense");case 19:return Nr("SuspenseList");case 0:case 2:case 15:return e=Ql(e.type,!1),e;case 11:return e=Ql(e.type.render,!1),e;case 1:return e=Ql(e.type,!0),e;default:return""}}function Mo(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Fn:return"Fragment";case Rn:return"Portal";case Lo:return"Profiler";case Ua:return"StrictMode";case Po:return"Suspense";case Io:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case pd:return(e.displayName||"Context")+".Consumer";case fd:return(e._context.displayName||"Context")+".Provider";case Va:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Ha:return t=e.displayName||null,t!==null?t:Mo(e.type)||"Memo";case Yt:t=e._payload,e=e._init;try{return Mo(e(t))}catch{}}return null}function zh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Mo(t);case 8:return t===Ua?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function cn(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function md(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function Lh(e){var t=md(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function gi(e){e._valueTracker||(e._valueTracker=Lh(e))}function gd(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=md(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Gi(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function Ao(e,t){var n=t.checked;return he({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function tu(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=cn(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function vd(e,t){t=t.checked,t!=null&&$a(e,"checked",t,!1)}function Do(e,t){vd(e,t);var n=cn(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?Ro(e,t.type,n):t.hasOwnProperty("defaultValue")&&Ro(e,t.type,cn(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function nu(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function Ro(e,t,n){(t!=="number"||Gi(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var _r=Array.isArray;function Yn(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+cn(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function Fo(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(M(91));return he({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function ru(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(M(92));if(_r(n)){if(1<n.length)throw Error(M(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:cn(n)}}function yd(e,t){var n=cn(t.value),r=cn(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function iu(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function xd(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Oo(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?xd(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var vi,kd=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(vi=vi||document.createElement("div"),vi.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=vi.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Hr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Lr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},Ph=["Webkit","ms","Moz","O"];Object.keys(Lr).forEach(function(e){Ph.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Lr[t]=Lr[e]})});function wd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Lr.hasOwnProperty(e)&&Lr[e]?(""+t).trim():t+"px"}function Sd(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=wd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var Ih=he({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Bo(e,t){if(t){if(Ih[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(M(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(M(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(M(61))}if(t.style!=null&&typeof t.style!="object")throw Error(M(62))}}function $o(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var Uo=null;function Wa(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Vo=null,Xn=null,Gn=null;function lu(e){if(e=ci(e)){if(typeof Vo!="function")throw Error(M(280));var t=e.stateNode;t&&(t=_l(t),Vo(e.stateNode,e.type,t))}}function bd(e){Xn?Gn?Gn.push(e):Gn=[e]:Xn=e}function jd(){if(Xn){var e=Xn,t=Gn;if(Gn=Xn=null,lu(e),t)for(e=0;e<t.length;e++)lu(t[e])}}function Cd(e,t){return e(t)}function Ed(){}var ql=!1;function Nd(e,t,n){if(ql)return e(t,n);ql=!0;try{return Cd(e,t,n)}finally{ql=!1,(Xn!==null||Gn!==null)&&(Ed(),jd())}}function Wr(e,t){var n=e.stateNode;if(n===null)return null;var r=_l(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(M(231,t,typeof n));return n}var Ho=!1;if(Ot)try{var yr={};Object.defineProperty(yr,"passive",{get:function(){Ho=!0}}),window.addEventListener("test",yr,yr),window.removeEventListener("test",yr,yr)}catch{Ho=!1}function Mh(e,t,n,r,i,l,o,a,s){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Pr=!1,Ji=null,Zi=!1,Wo=null,Ah={onError:function(e){Pr=!0,Ji=e}};function Dh(e,t,n,r,i,l,o,a,s){Pr=!1,Ji=null,Mh.apply(Ah,arguments)}function Rh(e,t,n,r,i,l,o,a,s){if(Dh.apply(this,arguments),Pr){if(Pr){var c=Ji;Pr=!1,Ji=null}else throw Error(M(198));Zi||(Zi=!0,Wo=c)}}function Ln(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function _d(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function ou(e){if(Ln(e)!==e)throw Error(M(188))}function Fh(e){var t=e.alternate;if(!t){if(t=Ln(e),t===null)throw Error(M(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return ou(i),e;if(l===r)return ou(i),t;l=l.sibling}throw Error(M(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(M(189))}}if(n.alternate!==r)throw Error(M(190))}if(n.tag!==3)throw Error(M(188));return n.stateNode.current===n?e:t}function Td(e){return e=Fh(e),e!==null?zd(e):null}function zd(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=zd(e);if(t!==null)return t;e=e.sibling}return null}var Ld=rt.unstable_scheduleCallback,au=rt.unstable_cancelCallback,Oh=rt.unstable_shouldYield,Bh=rt.unstable_requestPaint,ge=rt.unstable_now,$h=rt.unstable_getCurrentPriorityLevel,Qa=rt.unstable_ImmediatePriority,Pd=rt.unstable_UserBlockingPriority,el=rt.unstable_NormalPriority,Uh=rt.unstable_LowPriority,Id=rt.unstable_IdlePriority,jl=null,Nt=null;function Vh(e){if(Nt&&typeof Nt.onCommitFiberRoot=="function")try{Nt.onCommitFiberRoot(jl,e,void 0,(e.current.flags&128)===128)}catch{}}var xt=Math.clz32?Math.clz32:Qh,Hh=Math.log,Wh=Math.LN2;function Qh(e){return e>>>=0,e===0?32:31-(Hh(e)/Wh|0)|0}var yi=64,xi=4194304;function Tr(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function tl(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=Tr(a):(l&=o,l!==0&&(r=Tr(l)))}else o=n&~i,o!==0?r=Tr(o):l!==0&&(r=Tr(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-xt(t),i=1<<n,r|=e[n],t&=~i;return r}function qh(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Kh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-xt(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=qh(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Qo(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function Md(){var e=yi;return yi<<=1,!(yi&4194240)&&(yi=64),e}function Kl(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function si(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-xt(t),e[t]=n}function Yh(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-xt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function qa(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-xt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var re=0;function Ad(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Dd,Ka,Rd,Fd,Od,qo=!1,ki=[],tn=null,nn=null,rn=null,Qr=new Map,qr=new Map,Gt=[],Xh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function su(e,t){switch(e){case"focusin":case"focusout":tn=null;break;case"dragenter":case"dragleave":nn=null;break;case"mouseover":case"mouseout":rn=null;break;case"pointerover":case"pointerout":Qr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":qr.delete(t.pointerId)}}function xr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=ci(t),t!==null&&Ka(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Gh(e,t,n,r,i){switch(t){case"focusin":return tn=xr(tn,e,t,n,r,i),!0;case"dragenter":return nn=xr(nn,e,t,n,r,i),!0;case"mouseover":return rn=xr(rn,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Qr.set(l,xr(Qr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,qr.set(l,xr(qr.get(l)||null,e,t,n,r,i)),!0}return!1}function Bd(e){var t=kn(e.target);if(t!==null){var n=Ln(t);if(n!==null){if(t=n.tag,t===13){if(t=_d(n),t!==null){e.blockedOn=t,Od(e.priority,function(){Rd(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Fi(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=Ko(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);Uo=r,n.target.dispatchEvent(r),Uo=null}else return t=ci(n),t!==null&&Ka(t),e.blockedOn=n,!1;t.shift()}return!0}function uu(e,t,n){Fi(e)&&n.delete(t)}function Jh(){qo=!1,tn!==null&&Fi(tn)&&(tn=null),nn!==null&&Fi(nn)&&(nn=null),rn!==null&&Fi(rn)&&(rn=null),Qr.forEach(uu),qr.forEach(uu)}function kr(e,t){e.blockedOn===t&&(e.blockedOn=null,qo||(qo=!0,rt.unstable_scheduleCallback(rt.unstable_NormalPriority,Jh)))}function Kr(e){function t(i){return kr(i,e)}if(0<ki.length){kr(ki[0],e);for(var n=1;n<ki.length;n++){var r=ki[n];r.blockedOn===e&&(r.blockedOn=null)}}for(tn!==null&&kr(tn,e),nn!==null&&kr(nn,e),rn!==null&&kr(rn,e),Qr.forEach(t),qr.forEach(t),n=0;n<Gt.length;n++)r=Gt[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Gt.length&&(n=Gt[0],n.blockedOn===null);)Bd(n),n.blockedOn===null&&Gt.shift()}var Jn=Vt.ReactCurrentBatchConfig,nl=!0;function Zh(e,t,n,r){var i=re,l=Jn.transition;Jn.transition=null;try{re=1,Ya(e,t,n,r)}finally{re=i,Jn.transition=l}}function em(e,t,n,r){var i=re,l=Jn.transition;Jn.transition=null;try{re=4,Ya(e,t,n,r)}finally{re=i,Jn.transition=l}}function Ya(e,t,n,r){if(nl){var i=Ko(e,t,n,r);if(i===null)io(e,t,r,rl,n),su(e,r);else if(Gh(i,e,t,n,r))r.stopPropagation();else if(su(e,r),t&4&&-1<Xh.indexOf(e)){for(;i!==null;){var l=ci(i);if(l!==null&&Dd(l),l=Ko(e,t,n,r),l===null&&io(e,t,r,rl,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else io(e,t,r,null,n)}}var rl=null;function Ko(e,t,n,r){if(rl=null,e=Wa(r),e=kn(e),e!==null)if(t=Ln(e),t===null)e=null;else if(n=t.tag,n===13){if(e=_d(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return rl=e,null}function $d(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch($h()){case Qa:return 1;case Pd:return 4;case el:case Uh:return 16;case Id:return 536870912;default:return 16}default:return 16}}var Zt=null,Xa=null,Oi=null;function Ud(){if(Oi)return Oi;var e,t=Xa,n=t.length,r,i="value"in Zt?Zt.value:Zt.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Oi=i.slice(e,1<r?1-r:void 0)}function Bi(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function wi(){return!0}function cu(){return!1}function lt(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?wi:cu,this.isPropagationStopped=cu,this}return he(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=wi)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=wi)},persist:function(){},isPersistent:wi}),t}var dr={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Ga=lt(dr),ui=he({},dr,{view:0,detail:0}),tm=lt(ui),Yl,Xl,wr,Cl=he({},ui,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:Ja,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==wr&&(wr&&e.type==="mousemove"?(Yl=e.screenX-wr.screenX,Xl=e.screenY-wr.screenY):Xl=Yl=0,wr=e),Yl)},movementY:function(e){return"movementY"in e?e.movementY:Xl}}),du=lt(Cl),nm=he({},Cl,{dataTransfer:0}),rm=lt(nm),im=he({},ui,{relatedTarget:0}),Gl=lt(im),lm=he({},dr,{animationName:0,elapsedTime:0,pseudoElement:0}),om=lt(lm),am=he({},dr,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),sm=lt(am),um=he({},dr,{data:0}),fu=lt(um),cm={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},dm={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},fm={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function pm(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=fm[e])?!!t[e]:!1}function Ja(){return pm}var hm=he({},ui,{key:function(e){if(e.key){var t=cm[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Bi(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?dm[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:Ja,charCode:function(e){return e.type==="keypress"?Bi(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Bi(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),mm=lt(hm),gm=he({},Cl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),pu=lt(gm),vm=he({},ui,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:Ja}),ym=lt(vm),xm=he({},dr,{propertyName:0,elapsedTime:0,pseudoElement:0}),km=lt(xm),wm=he({},Cl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),Sm=lt(wm),bm=[9,13,27,32],Za=Ot&&"CompositionEvent"in window,Ir=null;Ot&&"documentMode"in document&&(Ir=document.documentMode);var jm=Ot&&"TextEvent"in window&&!Ir,Vd=Ot&&(!Za||Ir&&8<Ir&&11>=Ir),hu=" ",mu=!1;function Hd(e,t){switch(e){case"keyup":return bm.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Wd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var On=!1;function Cm(e,t){switch(e){case"compositionend":return Wd(t);case"keypress":return t.which!==32?null:(mu=!0,hu);case"textInput":return e=t.data,e===hu&&mu?null:e;default:return null}}function Em(e,t){if(On)return e==="compositionend"||!Za&&Hd(e,t)?(e=Ud(),Oi=Xa=Zt=null,On=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return Vd&&t.locale!=="ko"?null:t.data;default:return null}}var Nm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function gu(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!Nm[e.type]:t==="textarea"}function Qd(e,t,n,r){bd(r),t=il(t,"onChange"),0<t.length&&(n=new Ga("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Mr=null,Yr=null;function _m(e){rf(e,0)}function El(e){var t=Un(e);if(gd(t))return e}function Tm(e,t){if(e==="change")return t}var qd=!1;if(Ot){var Jl;if(Ot){var Zl="oninput"in document;if(!Zl){var vu=document.createElement("div");vu.setAttribute("oninput","return;"),Zl=typeof vu.oninput=="function"}Jl=Zl}else Jl=!1;qd=Jl&&(!document.documentMode||9<document.documentMode)}function yu(){Mr&&(Mr.detachEvent("onpropertychange",Kd),Yr=Mr=null)}function Kd(e){if(e.propertyName==="value"&&El(Yr)){var t=[];Qd(t,Yr,e,Wa(e)),Nd(_m,t)}}function zm(e,t,n){e==="focusin"?(yu(),Mr=t,Yr=n,Mr.attachEvent("onpropertychange",Kd)):e==="focusout"&&yu()}function Lm(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return El(Yr)}function Pm(e,t){if(e==="click")return El(t)}function Im(e,t){if(e==="input"||e==="change")return El(t)}function Mm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var wt=typeof Object.is=="function"?Object.is:Mm;function Xr(e,t){if(wt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!zo.call(t,i)||!wt(e[i],t[i]))return!1}return!0}function xu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function ku(e,t){var n=xu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=xu(n)}}function Yd(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?Yd(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Xd(){for(var e=window,t=Gi();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Gi(e.document)}return t}function es(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function Am(e){var t=Xd(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&Yd(n.ownerDocument.documentElement,n)){if(r!==null&&es(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=ku(n,l);var o=ku(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Dm=Ot&&"documentMode"in document&&11>=document.documentMode,Bn=null,Yo=null,Ar=null,Xo=!1;function wu(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Xo||Bn==null||Bn!==Gi(r)||(r=Bn,"selectionStart"in r&&es(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Ar&&Xr(Ar,r)||(Ar=r,r=il(Yo,"onSelect"),0<r.length&&(t=new Ga("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=Bn)))}function Si(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var $n={animationend:Si("Animation","AnimationEnd"),animationiteration:Si("Animation","AnimationIteration"),animationstart:Si("Animation","AnimationStart"),transitionend:Si("Transition","TransitionEnd")},eo={},Gd={};Ot&&(Gd=document.createElement("div").style,"AnimationEvent"in window||(delete $n.animationend.animation,delete $n.animationiteration.animation,delete $n.animationstart.animation),"TransitionEvent"in window||delete $n.transitionend.transition);function Nl(e){if(eo[e])return eo[e];if(!$n[e])return e;var t=$n[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Gd)return eo[e]=t[n];return e}var Jd=Nl("animationend"),Zd=Nl("animationiteration"),ef=Nl("animationstart"),tf=Nl("transitionend"),nf=new Map,Su="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function fn(e,t){nf.set(e,t),zn(t,[e])}for(var to=0;to<Su.length;to++){var no=Su[to],Rm=no.toLowerCase(),Fm=no[0].toUpperCase()+no.slice(1);fn(Rm,"on"+Fm)}fn(Jd,"onAnimationEnd");fn(Zd,"onAnimationIteration");fn(ef,"onAnimationStart");fn("dblclick","onDoubleClick");fn("focusin","onFocus");fn("focusout","onBlur");fn(tf,"onTransitionEnd");rr("onMouseEnter",["mouseout","mouseover"]);rr("onMouseLeave",["mouseout","mouseover"]);rr("onPointerEnter",["pointerout","pointerover"]);rr("onPointerLeave",["pointerout","pointerover"]);zn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));zn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));zn("onBeforeInput",["compositionend","keypress","textInput","paste"]);zn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));zn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));zn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var zr="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Om=new Set("cancel close invalid load scroll toggle".split(" ").concat(zr));function bu(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,Rh(r,t,void 0,e),e.currentTarget=null}function rf(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,c=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;bu(i,a,c),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,c=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;bu(i,a,c),l=s}}}if(Zi)throw e=Wo,Zi=!1,Wo=null,e}function ue(e,t){var n=t[ta];n===void 0&&(n=t[ta]=new Set);var r=e+"__bubble";n.has(r)||(lf(t,e,2,!1),n.add(r))}function ro(e,t,n){var r=0;t&&(r|=4),lf(n,e,r,t)}var bi="_reactListening"+Math.random().toString(36).slice(2);function Gr(e){if(!e[bi]){e[bi]=!0,dd.forEach(function(n){n!=="selectionchange"&&(Om.has(n)||ro(n,!1,e),ro(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[bi]||(t[bi]=!0,ro("selectionchange",!1,t))}}function lf(e,t,n,r){switch($d(t)){case 1:var i=Zh;break;case 4:i=em;break;default:i=Ya}n=i.bind(null,t,n,e),i=void 0,!Ho||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function io(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=kn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}Nd(function(){var c=l,d=Wa(n),f=[];e:{var m=nf.get(e);if(m!==void 0){var p=Ga,w=e;switch(e){case"keypress":if(Bi(n)===0)break e;case"keydown":case"keyup":p=mm;break;case"focusin":w="focus",p=Gl;break;case"focusout":w="blur",p=Gl;break;case"beforeblur":case"afterblur":p=Gl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":p=du;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":p=rm;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":p=ym;break;case Jd:case Zd:case ef:p=om;break;case tf:p=km;break;case"scroll":p=tm;break;case"wheel":p=Sm;break;case"copy":case"cut":case"paste":p=sm;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":p=pu}var S=(t&4)!==0,I=!S&&e==="scroll",h=S?m!==null?m+"Capture":null:m;S=[];for(var v=c,y;v!==null;){y=v;var b=y.stateNode;if(y.tag===5&&b!==null&&(y=b,h!==null&&(b=Wr(v,h),b!=null&&S.push(Jr(v,b,y)))),I)break;v=v.return}0<S.length&&(m=new p(m,w,null,n,d),f.push({event:m,listeners:S}))}}if(!(t&7)){e:{if(m=e==="mouseover"||e==="pointerover",p=e==="mouseout"||e==="pointerout",m&&n!==Uo&&(w=n.relatedTarget||n.fromElement)&&(kn(w)||w[Bt]))break e;if((p||m)&&(m=d.window===d?d:(m=d.ownerDocument)?m.defaultView||m.parentWindow:window,p?(w=n.relatedTarget||n.toElement,p=c,w=w?kn(w):null,w!==null&&(I=Ln(w),w!==I||w.tag!==5&&w.tag!==6)&&(w=null)):(p=null,w=c),p!==w)){if(S=du,b="onMouseLeave",h="onMouseEnter",v="mouse",(e==="pointerout"||e==="pointerover")&&(S=pu,b="onPointerLeave",h="onPointerEnter",v="pointer"),I=p==null?m:Un(p),y=w==null?m:Un(w),m=new S(b,v+"leave",p,n,d),m.target=I,m.relatedTarget=y,b=null,kn(d)===c&&(S=new S(h,v+"enter",w,n,d),S.target=y,S.relatedTarget=I,b=S),I=b,p&&w)t:{for(S=p,h=w,v=0,y=S;y;y=Mn(y))v++;for(y=0,b=h;b;b=Mn(b))y++;for(;0<v-y;)S=Mn(S),v--;for(;0<y-v;)h=Mn(h),y--;for(;v--;){if(S===h||h!==null&&S===h.alternate)break t;S=Mn(S),h=Mn(h)}S=null}else S=null;p!==null&&ju(f,m,p,S,!1),w!==null&&I!==null&&ju(f,I,w,S,!0)}}e:{if(m=c?Un(c):window,p=m.nodeName&&m.nodeName.toLowerCase(),p==="select"||p==="input"&&m.type==="file")var E=Tm;else if(gu(m))if(qd)E=Im;else{E=Lm;var k=zm}else(p=m.nodeName)&&p.toLowerCase()==="input"&&(m.type==="checkbox"||m.type==="radio")&&(E=Pm);if(E&&(E=E(e,c))){Qd(f,E,n,d);break e}k&&k(e,m,c),e==="focusout"&&(k=m._wrapperState)&&k.controlled&&m.type==="number"&&Ro(m,"number",m.value)}switch(k=c?Un(c):window,e){case"focusin":(gu(k)||k.contentEditable==="true")&&(Bn=k,Yo=c,Ar=null);break;case"focusout":Ar=Yo=Bn=null;break;case"mousedown":Xo=!0;break;case"contextmenu":case"mouseup":case"dragend":Xo=!1,wu(f,n,d);break;case"selectionchange":if(Dm)break;case"keydown":case"keyup":wu(f,n,d)}var C;if(Za)e:{switch(e){case"compositionstart":var _="onCompositionStart";break e;case"compositionend":_="onCompositionEnd";break e;case"compositionupdate":_="onCompositionUpdate";break e}_=void 0}else On?Hd(e,n)&&(_="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(_="onCompositionStart");_&&(Vd&&n.locale!=="ko"&&(On||_!=="onCompositionStart"?_==="onCompositionEnd"&&On&&(C=Ud()):(Zt=d,Xa="value"in Zt?Zt.value:Zt.textContent,On=!0)),k=il(c,_),0<k.length&&(_=new fu(_,e,null,n,d),f.push({event:_,listeners:k}),C?_.data=C:(C=Wd(n),C!==null&&(_.data=C)))),(C=jm?Cm(e,n):Em(e,n))&&(c=il(c,"onBeforeInput"),0<c.length&&(d=new fu("onBeforeInput","beforeinput",null,n,d),f.push({event:d,listeners:c}),d.data=C))}rf(f,t)})}function Jr(e,t,n){return{instance:e,listener:t,currentTarget:n}}function il(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Wr(e,n),l!=null&&r.unshift(Jr(e,l,i)),l=Wr(e,t),l!=null&&r.push(Jr(e,l,i))),e=e.return}return r}function Mn(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function ju(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,c=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&c!==null&&(a=c,i?(s=Wr(n,l),s!=null&&o.unshift(Jr(n,s,a))):i||(s=Wr(n,l),s!=null&&o.push(Jr(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Bm=/\r\n?/g,$m=/\u0000|\uFFFD/g;function Cu(e){return(typeof e=="string"?e:""+e).replace(Bm,`
`).replace($m,"")}function ji(e,t,n){if(t=Cu(t),Cu(e)!==t&&n)throw Error(M(425))}function ll(){}var Go=null,Jo=null;function Zo(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var ea=typeof setTimeout=="function"?setTimeout:void 0,Um=typeof clearTimeout=="function"?clearTimeout:void 0,Eu=typeof Promise=="function"?Promise:void 0,Vm=typeof queueMicrotask=="function"?queueMicrotask:typeof Eu<"u"?function(e){return Eu.resolve(null).then(e).catch(Hm)}:ea;function Hm(e){setTimeout(function(){throw e})}function lo(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),Kr(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);Kr(t)}function ln(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function Nu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var fr=Math.random().toString(36).slice(2),Ct="__reactFiber$"+fr,Zr="__reactProps$"+fr,Bt="__reactContainer$"+fr,ta="__reactEvents$"+fr,Wm="__reactListeners$"+fr,Qm="__reactHandles$"+fr;function kn(e){var t=e[Ct];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Bt]||n[Ct]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=Nu(e);e!==null;){if(n=e[Ct])return n;e=Nu(e)}return t}e=n,n=e.parentNode}return null}function ci(e){return e=e[Ct]||e[Bt],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function Un(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(M(33))}function _l(e){return e[Zr]||null}var na=[],Vn=-1;function pn(e){return{current:e}}function ce(e){0>Vn||(e.current=na[Vn],na[Vn]=null,Vn--)}function ae(e,t){Vn++,na[Vn]=e.current,e.current=t}var dn={},Ie=pn(dn),We=pn(!1),Cn=dn;function ir(e,t){var n=e.type.contextTypes;if(!n)return dn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function Qe(e){return e=e.childContextTypes,e!=null}function ol(){ce(We),ce(Ie)}function _u(e,t,n){if(Ie.current!==dn)throw Error(M(168));ae(Ie,t),ae(We,n)}function of(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(M(108,zh(e)||"Unknown",i));return he({},n,r)}function al(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||dn,Cn=Ie.current,ae(Ie,e),ae(We,We.current),!0}function Tu(e,t,n){var r=e.stateNode;if(!r)throw Error(M(169));n?(e=of(e,t,Cn),r.__reactInternalMemoizedMergedChildContext=e,ce(We),ce(Ie),ae(Ie,e)):ce(We),ae(We,n)}var At=null,Tl=!1,oo=!1;function af(e){At===null?At=[e]:At.push(e)}function qm(e){Tl=!0,af(e)}function hn(){if(!oo&&At!==null){oo=!0;var e=0,t=re;try{var n=At;for(re=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}At=null,Tl=!1}catch(i){throw At!==null&&(At=At.slice(e+1)),Ld(Qa,hn),i}finally{re=t,oo=!1}}return null}var Hn=[],Wn=0,sl=null,ul=0,at=[],st=0,En=null,Dt=1,Rt="";function vn(e,t){Hn[Wn++]=ul,Hn[Wn++]=sl,sl=e,ul=t}function sf(e,t,n){at[st++]=Dt,at[st++]=Rt,at[st++]=En,En=e;var r=Dt;e=Rt;var i=32-xt(r)-1;r&=~(1<<i),n+=1;var l=32-xt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,Dt=1<<32-xt(t)+i|n<<i|r,Rt=l+e}else Dt=1<<l|n<<i|r,Rt=e}function ts(e){e.return!==null&&(vn(e,1),sf(e,1,0))}function ns(e){for(;e===sl;)sl=Hn[--Wn],Hn[Wn]=null,ul=Hn[--Wn],Hn[Wn]=null;for(;e===En;)En=at[--st],at[st]=null,Rt=at[--st],at[st]=null,Dt=at[--st],at[st]=null}var nt=null,et=null,de=!1,yt=null;function uf(e,t){var n=ct(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function zu(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,nt=e,et=ln(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,nt=e,et=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=En!==null?{id:Dt,overflow:Rt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=ct(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,nt=e,et=null,!0):!1;default:return!1}}function ra(e){return(e.mode&1)!==0&&(e.flags&128)===0}function ia(e){if(de){var t=et;if(t){var n=t;if(!zu(e,t)){if(ra(e))throw Error(M(418));t=ln(n.nextSibling);var r=nt;t&&zu(e,t)?uf(r,n):(e.flags=e.flags&-4097|2,de=!1,nt=e)}}else{if(ra(e))throw Error(M(418));e.flags=e.flags&-4097|2,de=!1,nt=e}}}function Lu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;nt=e}function Ci(e){if(e!==nt)return!1;if(!de)return Lu(e),de=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!Zo(e.type,e.memoizedProps)),t&&(t=et)){if(ra(e))throw cf(),Error(M(418));for(;t;)uf(e,t),t=ln(t.nextSibling)}if(Lu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(M(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){et=ln(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}et=null}}else et=nt?ln(e.stateNode.nextSibling):null;return!0}function cf(){for(var e=et;e;)e=ln(e.nextSibling)}function lr(){et=nt=null,de=!1}function rs(e){yt===null?yt=[e]:yt.push(e)}var Km=Vt.ReactCurrentBatchConfig;function Sr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(M(309));var r=n.stateNode}if(!r)throw Error(M(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(M(284));if(!n._owner)throw Error(M(290,e))}return e}function Ei(e,t){throw e=Object.prototype.toString.call(t),Error(M(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Pu(e){var t=e._init;return t(e._payload)}function df(e){function t(h,v){if(e){var y=h.deletions;y===null?(h.deletions=[v],h.flags|=16):y.push(v)}}function n(h,v){if(!e)return null;for(;v!==null;)t(h,v),v=v.sibling;return null}function r(h,v){for(h=new Map;v!==null;)v.key!==null?h.set(v.key,v):h.set(v.index,v),v=v.sibling;return h}function i(h,v){return h=un(h,v),h.index=0,h.sibling=null,h}function l(h,v,y){return h.index=y,e?(y=h.alternate,y!==null?(y=y.index,y<v?(h.flags|=2,v):y):(h.flags|=2,v)):(h.flags|=1048576,v)}function o(h){return e&&h.alternate===null&&(h.flags|=2),h}function a(h,v,y,b){return v===null||v.tag!==6?(v=ho(y,h.mode,b),v.return=h,v):(v=i(v,y),v.return=h,v)}function s(h,v,y,b){var E=y.type;return E===Fn?d(h,v,y.props.children,b,y.key):v!==null&&(v.elementType===E||typeof E=="object"&&E!==null&&E.$$typeof===Yt&&Pu(E)===v.type)?(b=i(v,y.props),b.ref=Sr(h,v,y),b.return=h,b):(b=qi(y.type,y.key,y.props,null,h.mode,b),b.ref=Sr(h,v,y),b.return=h,b)}function c(h,v,y,b){return v===null||v.tag!==4||v.stateNode.containerInfo!==y.containerInfo||v.stateNode.implementation!==y.implementation?(v=mo(y,h.mode,b),v.return=h,v):(v=i(v,y.children||[]),v.return=h,v)}function d(h,v,y,b,E){return v===null||v.tag!==7?(v=jn(y,h.mode,b,E),v.return=h,v):(v=i(v,y),v.return=h,v)}function f(h,v,y){if(typeof v=="string"&&v!==""||typeof v=="number")return v=ho(""+v,h.mode,y),v.return=h,v;if(typeof v=="object"&&v!==null){switch(v.$$typeof){case mi:return y=qi(v.type,v.key,v.props,null,h.mode,y),y.ref=Sr(h,null,v),y.return=h,y;case Rn:return v=mo(v,h.mode,y),v.return=h,v;case Yt:var b=v._init;return f(h,b(v._payload),y)}if(_r(v)||vr(v))return v=jn(v,h.mode,y,null),v.return=h,v;Ei(h,v)}return null}function m(h,v,y,b){var E=v!==null?v.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return E!==null?null:a(h,v,""+y,b);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case mi:return y.key===E?s(h,v,y,b):null;case Rn:return y.key===E?c(h,v,y,b):null;case Yt:return E=y._init,m(h,v,E(y._payload),b)}if(_r(y)||vr(y))return E!==null?null:d(h,v,y,b,null);Ei(h,y)}return null}function p(h,v,y,b,E){if(typeof b=="string"&&b!==""||typeof b=="number")return h=h.get(y)||null,a(v,h,""+b,E);if(typeof b=="object"&&b!==null){switch(b.$$typeof){case mi:return h=h.get(b.key===null?y:b.key)||null,s(v,h,b,E);case Rn:return h=h.get(b.key===null?y:b.key)||null,c(v,h,b,E);case Yt:var k=b._init;return p(h,v,y,k(b._payload),E)}if(_r(b)||vr(b))return h=h.get(y)||null,d(v,h,b,E,null);Ei(v,b)}return null}function w(h,v,y,b){for(var E=null,k=null,C=v,_=v=0,R=null;C!==null&&_<y.length;_++){C.index>_?(R=C,C=null):R=C.sibling;var P=m(h,C,y[_],b);if(P===null){C===null&&(C=R);break}e&&C&&P.alternate===null&&t(h,C),v=l(P,v,_),k===null?E=P:k.sibling=P,k=P,C=R}if(_===y.length)return n(h,C),de&&vn(h,_),E;if(C===null){for(;_<y.length;_++)C=f(h,y[_],b),C!==null&&(v=l(C,v,_),k===null?E=C:k.sibling=C,k=C);return de&&vn(h,_),E}for(C=r(h,C);_<y.length;_++)R=p(C,h,_,y[_],b),R!==null&&(e&&R.alternate!==null&&C.delete(R.key===null?_:R.key),v=l(R,v,_),k===null?E=R:k.sibling=R,k=R);return e&&C.forEach(function(T){return t(h,T)}),de&&vn(h,_),E}function S(h,v,y,b){var E=vr(y);if(typeof E!="function")throw Error(M(150));if(y=E.call(y),y==null)throw Error(M(151));for(var k=E=null,C=v,_=v=0,R=null,P=y.next();C!==null&&!P.done;_++,P=y.next()){C.index>_?(R=C,C=null):R=C.sibling;var T=m(h,C,P.value,b);if(T===null){C===null&&(C=R);break}e&&C&&T.alternate===null&&t(h,C),v=l(T,v,_),k===null?E=T:k.sibling=T,k=T,C=R}if(P.done)return n(h,C),de&&vn(h,_),E;if(C===null){for(;!P.done;_++,P=y.next())P=f(h,P.value,b),P!==null&&(v=l(P,v,_),k===null?E=P:k.sibling=P,k=P);return de&&vn(h,_),E}for(C=r(h,C);!P.done;_++,P=y.next())P=p(C,h,_,P.value,b),P!==null&&(e&&P.alternate!==null&&C.delete(P.key===null?_:P.key),v=l(P,v,_),k===null?E=P:k.sibling=P,k=P);return e&&C.forEach(function(D){return t(h,D)}),de&&vn(h,_),E}function I(h,v,y,b){if(typeof y=="object"&&y!==null&&y.type===Fn&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case mi:e:{for(var E=y.key,k=v;k!==null;){if(k.key===E){if(E=y.type,E===Fn){if(k.tag===7){n(h,k.sibling),v=i(k,y.props.children),v.return=h,h=v;break e}}else if(k.elementType===E||typeof E=="object"&&E!==null&&E.$$typeof===Yt&&Pu(E)===k.type){n(h,k.sibling),v=i(k,y.props),v.ref=Sr(h,k,y),v.return=h,h=v;break e}n(h,k);break}else t(h,k);k=k.sibling}y.type===Fn?(v=jn(y.props.children,h.mode,b,y.key),v.return=h,h=v):(b=qi(y.type,y.key,y.props,null,h.mode,b),b.ref=Sr(h,v,y),b.return=h,h=b)}return o(h);case Rn:e:{for(k=y.key;v!==null;){if(v.key===k)if(v.tag===4&&v.stateNode.containerInfo===y.containerInfo&&v.stateNode.implementation===y.implementation){n(h,v.sibling),v=i(v,y.children||[]),v.return=h,h=v;break e}else{n(h,v);break}else t(h,v);v=v.sibling}v=mo(y,h.mode,b),v.return=h,h=v}return o(h);case Yt:return k=y._init,I(h,v,k(y._payload),b)}if(_r(y))return w(h,v,y,b);if(vr(y))return S(h,v,y,b);Ei(h,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,v!==null&&v.tag===6?(n(h,v.sibling),v=i(v,y),v.return=h,h=v):(n(h,v),v=ho(y,h.mode,b),v.return=h,h=v),o(h)):n(h,v)}return I}var or=df(!0),ff=df(!1),cl=pn(null),dl=null,Qn=null,is=null;function ls(){is=Qn=dl=null}function os(e){var t=cl.current;ce(cl),e._currentValue=t}function la(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Zn(e,t){dl=e,is=Qn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(He=!0),e.firstContext=null)}function ft(e){var t=e._currentValue;if(is!==e)if(e={context:e,memoizedValue:t,next:null},Qn===null){if(dl===null)throw Error(M(308));Qn=e,dl.dependencies={lanes:0,firstContext:e}}else Qn=Qn.next=e;return t}var wn=null;function as(e){wn===null?wn=[e]:wn.push(e)}function pf(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,as(t)):(n.next=i.next,i.next=n),t.interleaved=n,$t(e,r)}function $t(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Xt=!1;function ss(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function hf(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Ft(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function on(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,Z&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,$t(e,n)}return i=r.interleaved,i===null?(t.next=t,as(r)):(t.next=i.next,i.next=t),r.interleaved=t,$t(e,n)}function $i(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,qa(e,n)}}function Iu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function fl(e,t,n,r){var i=e.updateQueue;Xt=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,c=s.next;s.next=null,o===null?l=c:o.next=c,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=s))}if(l!==null){var f=i.baseState;o=0,d=c=s=null,a=l;do{var m=a.lane,p=a.eventTime;if((r&m)===m){d!==null&&(d=d.next={eventTime:p,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var w=e,S=a;switch(m=t,p=n,S.tag){case 1:if(w=S.payload,typeof w=="function"){f=w.call(p,f,m);break e}f=w;break e;case 3:w.flags=w.flags&-65537|128;case 0:if(w=S.payload,m=typeof w=="function"?w.call(p,f,m):w,m==null)break e;f=he({},f,m);break e;case 2:Xt=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,m=i.effects,m===null?i.effects=[a]:m.push(a))}else p={eventTime:p,lane:m,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=p,s=f):d=d.next=p,o|=m;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;m=a,a=m.next,m.next=null,i.lastBaseUpdate=m,i.shared.pending=null}}while(!0);if(d===null&&(s=f),i.baseState=s,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);_n|=o,e.lanes=o,e.memoizedState=f}}function Mu(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(M(191,i));i.call(r)}}}var di={},_t=pn(di),ei=pn(di),ti=pn(di);function Sn(e){if(e===di)throw Error(M(174));return e}function us(e,t){switch(ae(ti,t),ae(ei,e),ae(_t,di),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Oo(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Oo(t,e)}ce(_t),ae(_t,t)}function ar(){ce(_t),ce(ei),ce(ti)}function mf(e){Sn(ti.current);var t=Sn(_t.current),n=Oo(t,e.type);t!==n&&(ae(ei,e),ae(_t,n))}function cs(e){ei.current===e&&(ce(_t),ce(ei))}var fe=pn(0);function pl(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var ao=[];function ds(){for(var e=0;e<ao.length;e++)ao[e]._workInProgressVersionPrimary=null;ao.length=0}var Ui=Vt.ReactCurrentDispatcher,so=Vt.ReactCurrentBatchConfig,Nn=0,pe=null,xe=null,we=null,hl=!1,Dr=!1,ni=0,Ym=0;function Te(){throw Error(M(321))}function fs(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!wt(e[n],t[n]))return!1;return!0}function ps(e,t,n,r,i,l){if(Nn=l,pe=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,Ui.current=e===null||e.memoizedState===null?Zm:eg,e=n(r,i),Dr){l=0;do{if(Dr=!1,ni=0,25<=l)throw Error(M(301));l+=1,we=xe=null,t.updateQueue=null,Ui.current=tg,e=n(r,i)}while(Dr)}if(Ui.current=ml,t=xe!==null&&xe.next!==null,Nn=0,we=xe=pe=null,hl=!1,t)throw Error(M(300));return e}function hs(){var e=ni!==0;return ni=0,e}function bt(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return we===null?pe.memoizedState=we=e:we=we.next=e,we}function pt(){if(xe===null){var e=pe.alternate;e=e!==null?e.memoizedState:null}else e=xe.next;var t=we===null?pe.memoizedState:we.next;if(t!==null)we=t,xe=e;else{if(e===null)throw Error(M(310));xe=e,e={memoizedState:xe.memoizedState,baseState:xe.baseState,baseQueue:xe.baseQueue,queue:xe.queue,next:null},we===null?pe.memoizedState=we=e:we=we.next=e}return we}function ri(e,t){return typeof t=="function"?t(e):t}function uo(e){var t=pt(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=xe,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,c=l;do{var d=c.lane;if((Nn&d)===d)s!==null&&(s=s.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var f={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};s===null?(a=s=f,o=r):s=s.next=f,pe.lanes|=d,_n|=d}c=c.next}while(c!==null&&c!==l);s===null?o=r:s.next=a,wt(r,t.memoizedState)||(He=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,pe.lanes|=l,_n|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function co(e){var t=pt(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);wt(l,t.memoizedState)||(He=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function gf(){}function vf(e,t){var n=pe,r=pt(),i=t(),l=!wt(r.memoizedState,i);if(l&&(r.memoizedState=i,He=!0),r=r.queue,ms(kf.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||we!==null&&we.memoizedState.tag&1){if(n.flags|=2048,ii(9,xf.bind(null,n,r,i,t),void 0,null),Se===null)throw Error(M(349));Nn&30||yf(n,t,i)}return i}function yf(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=pe.updateQueue,t===null?(t={lastEffect:null,stores:null},pe.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function xf(e,t,n,r){t.value=n,t.getSnapshot=r,wf(t)&&Sf(e)}function kf(e,t,n){return n(function(){wf(t)&&Sf(e)})}function wf(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!wt(e,n)}catch{return!0}}function Sf(e){var t=$t(e,1);t!==null&&kt(t,e,1,-1)}function Au(e){var t=bt();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:ri,lastRenderedState:e},t.queue=e,e=e.dispatch=Jm.bind(null,pe,e),[t.memoizedState,e]}function ii(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=pe.updateQueue,t===null?(t={lastEffect:null,stores:null},pe.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function bf(){return pt().memoizedState}function Vi(e,t,n,r){var i=bt();pe.flags|=e,i.memoizedState=ii(1|t,n,void 0,r===void 0?null:r)}function zl(e,t,n,r){var i=pt();r=r===void 0?null:r;var l=void 0;if(xe!==null){var o=xe.memoizedState;if(l=o.destroy,r!==null&&fs(r,o.deps)){i.memoizedState=ii(t,n,l,r);return}}pe.flags|=e,i.memoizedState=ii(1|t,n,l,r)}function Du(e,t){return Vi(8390656,8,e,t)}function ms(e,t){return zl(2048,8,e,t)}function jf(e,t){return zl(4,2,e,t)}function Cf(e,t){return zl(4,4,e,t)}function Ef(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function Nf(e,t,n){return n=n!=null?n.concat([e]):null,zl(4,4,Ef.bind(null,t,e),n)}function gs(){}function _f(e,t){var n=pt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&fs(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function Tf(e,t){var n=pt();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&fs(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function zf(e,t,n){return Nn&21?(wt(n,t)||(n=Md(),pe.lanes|=n,_n|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,He=!0),e.memoizedState=n)}function Xm(e,t){var n=re;re=n!==0&&4>n?n:4,e(!0);var r=so.transition;so.transition={};try{e(!1),t()}finally{re=n,so.transition=r}}function Lf(){return pt().memoizedState}function Gm(e,t,n){var r=sn(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},Pf(e))If(t,n);else if(n=pf(e,t,n,r),n!==null){var i=Fe();kt(n,e,r,i),Mf(n,t,r)}}function Jm(e,t,n){var r=sn(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(Pf(e))If(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,wt(a,o)){var s=t.interleaved;s===null?(i.next=i,as(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=pf(e,t,i,r),n!==null&&(i=Fe(),kt(n,e,r,i),Mf(n,t,r))}}function Pf(e){var t=e.alternate;return e===pe||t!==null&&t===pe}function If(e,t){Dr=hl=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function Mf(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,qa(e,n)}}var ml={readContext:ft,useCallback:Te,useContext:Te,useEffect:Te,useImperativeHandle:Te,useInsertionEffect:Te,useLayoutEffect:Te,useMemo:Te,useReducer:Te,useRef:Te,useState:Te,useDebugValue:Te,useDeferredValue:Te,useTransition:Te,useMutableSource:Te,useSyncExternalStore:Te,useId:Te,unstable_isNewReconciler:!1},Zm={readContext:ft,useCallback:function(e,t){return bt().memoizedState=[e,t===void 0?null:t],e},useContext:ft,useEffect:Du,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Vi(4194308,4,Ef.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Vi(4194308,4,e,t)},useInsertionEffect:function(e,t){return Vi(4,2,e,t)},useMemo:function(e,t){var n=bt();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=bt();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Gm.bind(null,pe,e),[r.memoizedState,e]},useRef:function(e){var t=bt();return e={current:e},t.memoizedState=e},useState:Au,useDebugValue:gs,useDeferredValue:function(e){return bt().memoizedState=e},useTransition:function(){var e=Au(!1),t=e[0];return e=Xm.bind(null,e[1]),bt().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=pe,i=bt();if(de){if(n===void 0)throw Error(M(407));n=n()}else{if(n=t(),Se===null)throw Error(M(349));Nn&30||yf(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Du(kf.bind(null,r,l,e),[e]),r.flags|=2048,ii(9,xf.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=bt(),t=Se.identifierPrefix;if(de){var n=Rt,r=Dt;n=(r&~(1<<32-xt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=ni++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Ym++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},eg={readContext:ft,useCallback:_f,useContext:ft,useEffect:ms,useImperativeHandle:Nf,useInsertionEffect:jf,useLayoutEffect:Cf,useMemo:Tf,useReducer:uo,useRef:bf,useState:function(){return uo(ri)},useDebugValue:gs,useDeferredValue:function(e){var t=pt();return zf(t,xe.memoizedState,e)},useTransition:function(){var e=uo(ri)[0],t=pt().memoizedState;return[e,t]},useMutableSource:gf,useSyncExternalStore:vf,useId:Lf,unstable_isNewReconciler:!1},tg={readContext:ft,useCallback:_f,useContext:ft,useEffect:ms,useImperativeHandle:Nf,useInsertionEffect:jf,useLayoutEffect:Cf,useMemo:Tf,useReducer:co,useRef:bf,useState:function(){return co(ri)},useDebugValue:gs,useDeferredValue:function(e){var t=pt();return xe===null?t.memoizedState=e:zf(t,xe.memoizedState,e)},useTransition:function(){var e=co(ri)[0],t=pt().memoizedState;return[e,t]},useMutableSource:gf,useSyncExternalStore:vf,useId:Lf,unstable_isNewReconciler:!1};function gt(e,t){if(e&&e.defaultProps){t=he({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function oa(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:he({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var Ll={isMounted:function(e){return(e=e._reactInternals)?Ln(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Fe(),i=sn(e),l=Ft(r,i);l.payload=t,n!=null&&(l.callback=n),t=on(e,l,i),t!==null&&(kt(t,e,i,r),$i(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Fe(),i=sn(e),l=Ft(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=on(e,l,i),t!==null&&(kt(t,e,i,r),$i(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Fe(),r=sn(e),i=Ft(n,r);i.tag=2,t!=null&&(i.callback=t),t=on(e,i,r),t!==null&&(kt(t,e,r,n),$i(t,e,r))}};function Ru(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Xr(n,r)||!Xr(i,l):!0}function Af(e,t,n){var r=!1,i=dn,l=t.contextType;return typeof l=="object"&&l!==null?l=ft(l):(i=Qe(t)?Cn:Ie.current,r=t.contextTypes,l=(r=r!=null)?ir(e,i):dn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=Ll,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function Fu(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&Ll.enqueueReplaceState(t,t.state,null)}function aa(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},ss(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=ft(l):(l=Qe(t)?Cn:Ie.current,i.context=ir(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(oa(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&Ll.enqueueReplaceState(i,i.state,null),fl(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function sr(e,t){try{var n="",r=t;do n+=Th(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function fo(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function sa(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var ng=typeof WeakMap=="function"?WeakMap:Map;function Df(e,t,n){n=Ft(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){vl||(vl=!0,ya=r),sa(e,t)},n}function Rf(e,t,n){n=Ft(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){sa(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){sa(e,t),typeof r!="function"&&(an===null?an=new Set([this]):an.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function Ou(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new ng;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=gg.bind(null,e,t,n),t.then(e,e))}function Bu(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function $u(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Ft(-1,1),t.tag=2,on(n,t,1))),n.lanes|=1),e)}var rg=Vt.ReactCurrentOwner,He=!1;function Re(e,t,n,r){t.child=e===null?ff(t,null,n,r):or(t,e.child,n,r)}function Uu(e,t,n,r,i){n=n.render;var l=t.ref;return Zn(t,i),r=ps(e,t,n,r,l,i),n=hs(),e!==null&&!He?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Ut(e,t,i)):(de&&n&&ts(t),t.flags|=1,Re(e,t,r,i),t.child)}function Vu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!js(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,Ff(e,t,l,r,i)):(e=qi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Xr,n(o,r)&&e.ref===t.ref)return Ut(e,t,i)}return t.flags|=1,e=un(l,r),e.ref=t.ref,e.return=t,t.child=e}function Ff(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Xr(l,r)&&e.ref===t.ref)if(He=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(He=!0);else return t.lanes=e.lanes,Ut(e,t,i)}return ua(e,t,n,r,i)}function Of(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},ae(Kn,Ze),Ze|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,ae(Kn,Ze),Ze|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,ae(Kn,Ze),Ze|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,ae(Kn,Ze),Ze|=r;return Re(e,t,i,n),t.child}function Bf(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function ua(e,t,n,r,i){var l=Qe(n)?Cn:Ie.current;return l=ir(t,l),Zn(t,i),n=ps(e,t,n,r,l,i),r=hs(),e!==null&&!He?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,Ut(e,t,i)):(de&&r&&ts(t),t.flags|=1,Re(e,t,n,i),t.child)}function Hu(e,t,n,r,i){if(Qe(n)){var l=!0;al(t)}else l=!1;if(Zn(t,i),t.stateNode===null)Hi(e,t),Af(t,n,r),aa(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=ft(c):(c=Qe(n)?Cn:Ie.current,c=ir(t,c));var d=n.getDerivedStateFromProps,f=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";f||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==c)&&Fu(t,o,r,c),Xt=!1;var m=t.memoizedState;o.state=m,fl(t,r,o,i),s=t.memoizedState,a!==r||m!==s||We.current||Xt?(typeof d=="function"&&(oa(t,n,d,r),s=t.memoizedState),(a=Xt||Ru(t,n,a,r,m,s,c))?(f||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,hf(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:gt(t.type,a),o.props=c,f=t.pendingProps,m=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=ft(s):(s=Qe(n)?Cn:Ie.current,s=ir(t,s));var p=n.getDerivedStateFromProps;(d=typeof p=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==f||m!==s)&&Fu(t,o,r,s),Xt=!1,m=t.memoizedState,o.state=m,fl(t,r,o,i);var w=t.memoizedState;a!==f||m!==w||We.current||Xt?(typeof p=="function"&&(oa(t,n,p,r),w=t.memoizedState),(c=Xt||Ru(t,n,c,r,m,w,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,w,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,w,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=w),o.props=r,o.state=w,o.context=s,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=1024),r=!1)}return ca(e,t,n,r,l,i)}function ca(e,t,n,r,i,l){Bf(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&Tu(t,n,!1),Ut(e,t,l);r=t.stateNode,rg.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=or(t,e.child,null,l),t.child=or(t,null,a,l)):Re(e,t,a,l),t.memoizedState=r.state,i&&Tu(t,n,!0),t.child}function $f(e){var t=e.stateNode;t.pendingContext?_u(e,t.pendingContext,t.pendingContext!==t.context):t.context&&_u(e,t.context,!1),us(e,t.containerInfo)}function Wu(e,t,n,r,i){return lr(),rs(i),t.flags|=256,Re(e,t,n,r),t.child}var da={dehydrated:null,treeContext:null,retryLane:0};function fa(e){return{baseLanes:e,cachePool:null,transitions:null}}function Uf(e,t,n){var r=t.pendingProps,i=fe.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),ae(fe,i&1),e===null)return ia(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Ml(o,r,0,null),e=jn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=fa(n),t.memoizedState=da,e):vs(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return ig(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=un(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=un(a,l):(l=jn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?fa(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=da,r}return l=e.child,e=l.sibling,r=un(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function vs(e,t){return t=Ml({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function Ni(e,t,n,r){return r!==null&&rs(r),or(t,e.child,null,n),e=vs(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function ig(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=fo(Error(M(422))),Ni(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Ml({mode:"visible",children:r.children},i,0,null),l=jn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&or(t,e.child,null,o),t.child.memoizedState=fa(o),t.memoizedState=da,l);if(!(t.mode&1))return Ni(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(M(419)),r=fo(l,r,void 0),Ni(e,t,o,r)}if(a=(o&e.childLanes)!==0,He||a){if(r=Se,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,$t(e,i),kt(r,e,i,-1))}return bs(),r=fo(Error(M(421))),Ni(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=vg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,et=ln(i.nextSibling),nt=t,de=!0,yt=null,e!==null&&(at[st++]=Dt,at[st++]=Rt,at[st++]=En,Dt=e.id,Rt=e.overflow,En=t),t=vs(t,r.children),t.flags|=4096,t)}function Qu(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),la(e.return,t,n)}function po(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function Vf(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(Re(e,t,r.children,n),r=fe.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Qu(e,n,t);else if(e.tag===19)Qu(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(ae(fe,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&pl(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),po(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&pl(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}po(t,!0,n,null,l);break;case"together":po(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Hi(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function Ut(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),_n|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(M(153));if(t.child!==null){for(e=t.child,n=un(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=un(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function lg(e,t,n){switch(t.tag){case 3:$f(t),lr();break;case 5:mf(t);break;case 1:Qe(t.type)&&al(t);break;case 4:us(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;ae(cl,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(ae(fe,fe.current&1),t.flags|=128,null):n&t.child.childLanes?Uf(e,t,n):(ae(fe,fe.current&1),e=Ut(e,t,n),e!==null?e.sibling:null);ae(fe,fe.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return Vf(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),ae(fe,fe.current),r)break;return null;case 22:case 23:return t.lanes=0,Of(e,t,n)}return Ut(e,t,n)}var Hf,pa,Wf,Qf;Hf=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};pa=function(){};Wf=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,Sn(_t.current);var l=null;switch(n){case"input":i=Ao(e,i),r=Ao(e,r),l=[];break;case"select":i=he({},i,{value:void 0}),r=he({},r,{value:void 0}),l=[];break;case"textarea":i=Fo(e,i),r=Fo(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=ll)}Bo(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(Vr.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var s=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&s!==a&&(s!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(c,n)),n=s;else c==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(c,s)):c==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(c,""+s):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(Vr.hasOwnProperty(c)?(s!=null&&c==="onScroll"&&ue("scroll",e),l||a===s||(l=[])):(l=l||[]).push(c,s))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};Qf=function(e,t,n,r){n!==r&&(t.flags|=4)};function br(e,t){if(!de)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function ze(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function og(e,t,n){var r=t.pendingProps;switch(ns(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return ze(t),null;case 1:return Qe(t.type)&&ol(),ze(t),null;case 3:return r=t.stateNode,ar(),ce(We),ce(Ie),ds(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(Ci(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,yt!==null&&(wa(yt),yt=null))),pa(e,t),ze(t),null;case 5:cs(t);var i=Sn(ti.current);if(n=t.type,e!==null&&t.stateNode!=null)Wf(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(M(166));return ze(t),null}if(e=Sn(_t.current),Ci(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[Ct]=t,r[Zr]=l,e=(t.mode&1)!==0,n){case"dialog":ue("cancel",r),ue("close",r);break;case"iframe":case"object":case"embed":ue("load",r);break;case"video":case"audio":for(i=0;i<zr.length;i++)ue(zr[i],r);break;case"source":ue("error",r);break;case"img":case"image":case"link":ue("error",r),ue("load",r);break;case"details":ue("toggle",r);break;case"input":tu(r,l),ue("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},ue("invalid",r);break;case"textarea":ru(r,l),ue("invalid",r)}Bo(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&ji(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&ji(r.textContent,a,e),i=["children",""+a]):Vr.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&ue("scroll",r)}switch(n){case"input":gi(r),nu(r,l,!0);break;case"textarea":gi(r),iu(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=ll)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=xd(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[Ct]=t,e[Zr]=r,Hf(e,t,!1,!1),t.stateNode=e;e:{switch(o=$o(n,r),n){case"dialog":ue("cancel",e),ue("close",e),i=r;break;case"iframe":case"object":case"embed":ue("load",e),i=r;break;case"video":case"audio":for(i=0;i<zr.length;i++)ue(zr[i],e);i=r;break;case"source":ue("error",e),i=r;break;case"img":case"image":case"link":ue("error",e),ue("load",e),i=r;break;case"details":ue("toggle",e),i=r;break;case"input":tu(e,r),i=Ao(e,r),ue("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=he({},r,{value:void 0}),ue("invalid",e);break;case"textarea":ru(e,r),i=Fo(e,r),ue("invalid",e);break;default:i=r}Bo(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?Sd(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&kd(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Hr(e,s):typeof s=="number"&&Hr(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Vr.hasOwnProperty(l)?s!=null&&l==="onScroll"&&ue("scroll",e):s!=null&&$a(e,l,s,o))}switch(n){case"input":gi(e),nu(e,r,!1);break;case"textarea":gi(e),iu(e);break;case"option":r.value!=null&&e.setAttribute("value",""+cn(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Yn(e,!!r.multiple,l,!1):r.defaultValue!=null&&Yn(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=ll)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return ze(t),null;case 6:if(e&&t.stateNode!=null)Qf(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(M(166));if(n=Sn(ti.current),Sn(_t.current),Ci(t)){if(r=t.stateNode,n=t.memoizedProps,r[Ct]=t,(l=r.nodeValue!==n)&&(e=nt,e!==null))switch(e.tag){case 3:ji(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&ji(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[Ct]=t,t.stateNode=r}return ze(t),null;case 13:if(ce(fe),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(de&&et!==null&&t.mode&1&&!(t.flags&128))cf(),lr(),t.flags|=98560,l=!1;else if(l=Ci(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(M(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(M(317));l[Ct]=t}else lr(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;ze(t),l=!1}else yt!==null&&(wa(yt),yt=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||fe.current&1?ke===0&&(ke=3):bs())),t.updateQueue!==null&&(t.flags|=4),ze(t),null);case 4:return ar(),pa(e,t),e===null&&Gr(t.stateNode.containerInfo),ze(t),null;case 10:return os(t.type._context),ze(t),null;case 17:return Qe(t.type)&&ol(),ze(t),null;case 19:if(ce(fe),l=t.memoizedState,l===null)return ze(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)br(l,!1);else{if(ke!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=pl(e),o!==null){for(t.flags|=128,br(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return ae(fe,fe.current&1|2),t.child}e=e.sibling}l.tail!==null&&ge()>ur&&(t.flags|=128,r=!0,br(l,!1),t.lanes=4194304)}else{if(!r)if(e=pl(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),br(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!de)return ze(t),null}else 2*ge()-l.renderingStartTime>ur&&n!==1073741824&&(t.flags|=128,r=!0,br(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=ge(),t.sibling=null,n=fe.current,ae(fe,r?n&1|2:n&1),t):(ze(t),null);case 22:case 23:return Ss(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?Ze&1073741824&&(ze(t),t.subtreeFlags&6&&(t.flags|=8192)):ze(t),null;case 24:return null;case 25:return null}throw Error(M(156,t.tag))}function ag(e,t){switch(ns(t),t.tag){case 1:return Qe(t.type)&&ol(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return ar(),ce(We),ce(Ie),ds(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return cs(t),null;case 13:if(ce(fe),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(M(340));lr()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return ce(fe),null;case 4:return ar(),null;case 10:return os(t.type._context),null;case 22:case 23:return Ss(),null;case 24:return null;default:return null}}var _i=!1,Pe=!1,sg=typeof WeakSet=="function"?WeakSet:Set,B=null;function qn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){me(e,t,r)}else n.current=null}function ha(e,t,n){try{n()}catch(r){me(e,t,r)}}var qu=!1;function ug(e,t){if(Go=nl,e=Xd(),es(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,c=0,d=0,f=e,m=null;t:for(;;){for(var p;f!==n||i!==0&&f.nodeType!==3||(a=o+i),f!==l||r!==0&&f.nodeType!==3||(s=o+r),f.nodeType===3&&(o+=f.nodeValue.length),(p=f.firstChild)!==null;)m=f,f=p;for(;;){if(f===e)break t;if(m===n&&++c===i&&(a=o),m===l&&++d===r&&(s=o),(p=f.nextSibling)!==null)break;f=m,m=f.parentNode}f=p}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Jo={focusedElem:e,selectionRange:n},nl=!1,B=t;B!==null;)if(t=B,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,B=e;else for(;B!==null;){t=B;try{var w=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(w!==null){var S=w.memoizedProps,I=w.memoizedState,h=t.stateNode,v=h.getSnapshotBeforeUpdate(t.elementType===t.type?S:gt(t.type,S),I);h.__reactInternalSnapshotBeforeUpdate=v}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(M(163))}}catch(b){me(t,t.return,b)}if(e=t.sibling,e!==null){e.return=t.return,B=e;break}B=t.return}return w=qu,qu=!1,w}function Rr(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&ha(t,n,l)}i=i.next}while(i!==r)}}function Pl(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function ma(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function qf(e){var t=e.alternate;t!==null&&(e.alternate=null,qf(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[Ct],delete t[Zr],delete t[ta],delete t[Wm],delete t[Qm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Kf(e){return e.tag===5||e.tag===3||e.tag===4}function Ku(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Kf(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function ga(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=ll));else if(r!==4&&(e=e.child,e!==null))for(ga(e,t,n),e=e.sibling;e!==null;)ga(e,t,n),e=e.sibling}function va(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(va(e,t,n),e=e.sibling;e!==null;)va(e,t,n),e=e.sibling}var Ee=null,vt=!1;function Qt(e,t,n){for(n=n.child;n!==null;)Yf(e,t,n),n=n.sibling}function Yf(e,t,n){if(Nt&&typeof Nt.onCommitFiberUnmount=="function")try{Nt.onCommitFiberUnmount(jl,n)}catch{}switch(n.tag){case 5:Pe||qn(n,t);case 6:var r=Ee,i=vt;Ee=null,Qt(e,t,n),Ee=r,vt=i,Ee!==null&&(vt?(e=Ee,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):Ee.removeChild(n.stateNode));break;case 18:Ee!==null&&(vt?(e=Ee,n=n.stateNode,e.nodeType===8?lo(e.parentNode,n):e.nodeType===1&&lo(e,n),Kr(e)):lo(Ee,n.stateNode));break;case 4:r=Ee,i=vt,Ee=n.stateNode.containerInfo,vt=!0,Qt(e,t,n),Ee=r,vt=i;break;case 0:case 11:case 14:case 15:if(!Pe&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&ha(n,t,o),i=i.next}while(i!==r)}Qt(e,t,n);break;case 1:if(!Pe&&(qn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){me(n,t,a)}Qt(e,t,n);break;case 21:Qt(e,t,n);break;case 22:n.mode&1?(Pe=(r=Pe)||n.memoizedState!==null,Qt(e,t,n),Pe=r):Qt(e,t,n);break;default:Qt(e,t,n)}}function Yu(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new sg),t.forEach(function(r){var i=yg.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function mt(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:Ee=a.stateNode,vt=!1;break e;case 3:Ee=a.stateNode.containerInfo,vt=!0;break e;case 4:Ee=a.stateNode.containerInfo,vt=!0;break e}a=a.return}if(Ee===null)throw Error(M(160));Yf(l,o,i),Ee=null,vt=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(c){me(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Xf(t,e),t=t.sibling}function Xf(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(mt(t,e),St(e),r&4){try{Rr(3,e,e.return),Pl(3,e)}catch(S){me(e,e.return,S)}try{Rr(5,e,e.return)}catch(S){me(e,e.return,S)}}break;case 1:mt(t,e),St(e),r&512&&n!==null&&qn(n,n.return);break;case 5:if(mt(t,e),St(e),r&512&&n!==null&&qn(n,n.return),e.flags&32){var i=e.stateNode;try{Hr(i,"")}catch(S){me(e,e.return,S)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&vd(i,l),$o(a,o);var c=$o(a,l);for(o=0;o<s.length;o+=2){var d=s[o],f=s[o+1];d==="style"?Sd(i,f):d==="dangerouslySetInnerHTML"?kd(i,f):d==="children"?Hr(i,f):$a(i,d,f,c)}switch(a){case"input":Do(i,l);break;case"textarea":yd(i,l);break;case"select":var m=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var p=l.value;p!=null?Yn(i,!!l.multiple,p,!1):m!==!!l.multiple&&(l.defaultValue!=null?Yn(i,!!l.multiple,l.defaultValue,!0):Yn(i,!!l.multiple,l.multiple?[]:"",!1))}i[Zr]=l}catch(S){me(e,e.return,S)}}break;case 6:if(mt(t,e),St(e),r&4){if(e.stateNode===null)throw Error(M(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(S){me(e,e.return,S)}}break;case 3:if(mt(t,e),St(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{Kr(t.containerInfo)}catch(S){me(e,e.return,S)}break;case 4:mt(t,e),St(e);break;case 13:mt(t,e),St(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(ks=ge())),r&4&&Yu(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Pe=(c=Pe)||d,mt(t,e),Pe=c):mt(t,e),St(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for(B=e,d=e.child;d!==null;){for(f=B=d;B!==null;){switch(m=B,p=m.child,m.tag){case 0:case 11:case 14:case 15:Rr(4,m,m.return);break;case 1:qn(m,m.return);var w=m.stateNode;if(typeof w.componentWillUnmount=="function"){r=m,n=m.return;try{t=r,w.props=t.memoizedProps,w.state=t.memoizedState,w.componentWillUnmount()}catch(S){me(r,n,S)}}break;case 5:qn(m,m.return);break;case 22:if(m.memoizedState!==null){Gu(f);continue}}p!==null?(p.return=m,B=p):Gu(f)}d=d.sibling}e:for(d=null,f=e;;){if(f.tag===5){if(d===null){d=f;try{i=f.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=f.stateNode,s=f.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=wd("display",o))}catch(S){me(e,e.return,S)}}}else if(f.tag===6){if(d===null)try{f.stateNode.nodeValue=c?"":f.memoizedProps}catch(S){me(e,e.return,S)}}else if((f.tag!==22&&f.tag!==23||f.memoizedState===null||f===e)&&f.child!==null){f.child.return=f,f=f.child;continue}if(f===e)break e;for(;f.sibling===null;){if(f.return===null||f.return===e)break e;d===f&&(d=null),f=f.return}d===f&&(d=null),f.sibling.return=f.return,f=f.sibling}}break;case 19:mt(t,e),St(e),r&4&&Yu(e);break;case 21:break;default:mt(t,e),St(e)}}function St(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Kf(n)){var r=n;break e}n=n.return}throw Error(M(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Hr(i,""),r.flags&=-33);var l=Ku(e);va(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=Ku(e);ga(e,a,o);break;default:throw Error(M(161))}}catch(s){me(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function cg(e,t,n){B=e,Gf(e)}function Gf(e,t,n){for(var r=(e.mode&1)!==0;B!==null;){var i=B,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||_i;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||Pe;a=_i;var c=Pe;if(_i=o,(Pe=s)&&!c)for(B=i;B!==null;)o=B,s=o.child,o.tag===22&&o.memoizedState!==null?Ju(i):s!==null?(s.return=o,B=s):Ju(i);for(;l!==null;)B=l,Gf(l),l=l.sibling;B=i,_i=a,Pe=c}Xu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,B=l):Xu(e)}}function Xu(e){for(;B!==null;){var t=B;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Pe||Pl(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Pe)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:gt(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Mu(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Mu(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var f=d.dehydrated;f!==null&&Kr(f)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(M(163))}Pe||t.flags&512&&ma(t)}catch(m){me(t,t.return,m)}}if(t===e){B=null;break}if(n=t.sibling,n!==null){n.return=t.return,B=n;break}B=t.return}}function Gu(e){for(;B!==null;){var t=B;if(t===e){B=null;break}var n=t.sibling;if(n!==null){n.return=t.return,B=n;break}B=t.return}}function Ju(e){for(;B!==null;){var t=B;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Pl(4,t)}catch(s){me(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){me(t,i,s)}}var l=t.return;try{ma(t)}catch(s){me(t,l,s)}break;case 5:var o=t.return;try{ma(t)}catch(s){me(t,o,s)}}}catch(s){me(t,t.return,s)}if(t===e){B=null;break}var a=t.sibling;if(a!==null){a.return=t.return,B=a;break}B=t.return}}var dg=Math.ceil,gl=Vt.ReactCurrentDispatcher,ys=Vt.ReactCurrentOwner,dt=Vt.ReactCurrentBatchConfig,Z=0,Se=null,ye=null,Ne=0,Ze=0,Kn=pn(0),ke=0,li=null,_n=0,Il=0,xs=0,Fr=null,Ve=null,ks=0,ur=1/0,Mt=null,vl=!1,ya=null,an=null,Ti=!1,en=null,yl=0,Or=0,xa=null,Wi=-1,Qi=0;function Fe(){return Z&6?ge():Wi!==-1?Wi:Wi=ge()}function sn(e){return e.mode&1?Z&2&&Ne!==0?Ne&-Ne:Km.transition!==null?(Qi===0&&(Qi=Md()),Qi):(e=re,e!==0||(e=window.event,e=e===void 0?16:$d(e.type)),e):1}function kt(e,t,n,r){if(50<Or)throw Or=0,xa=null,Error(M(185));si(e,n,r),(!(Z&2)||e!==Se)&&(e===Se&&(!(Z&2)&&(Il|=n),ke===4&&Jt(e,Ne)),qe(e,r),n===1&&Z===0&&!(t.mode&1)&&(ur=ge()+500,Tl&&hn()))}function qe(e,t){var n=e.callbackNode;Kh(e,t);var r=tl(e,e===Se?Ne:0);if(r===0)n!==null&&au(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&au(n),t===1)e.tag===0?qm(Zu.bind(null,e)):af(Zu.bind(null,e)),Vm(function(){!(Z&6)&&hn()}),n=null;else{switch(Ad(r)){case 1:n=Qa;break;case 4:n=Pd;break;case 16:n=el;break;case 536870912:n=Id;break;default:n=el}n=lp(n,Jf.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function Jf(e,t){if(Wi=-1,Qi=0,Z&6)throw Error(M(327));var n=e.callbackNode;if(er()&&e.callbackNode!==n)return null;var r=tl(e,e===Se?Ne:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=xl(e,r);else{t=r;var i=Z;Z|=2;var l=ep();(Se!==e||Ne!==t)&&(Mt=null,ur=ge()+500,bn(e,t));do try{hg();break}catch(a){Zf(e,a)}while(!0);ls(),gl.current=l,Z=i,ye!==null?t=0:(Se=null,Ne=0,t=ke)}if(t!==0){if(t===2&&(i=Qo(e),i!==0&&(r=i,t=ka(e,i))),t===1)throw n=li,bn(e,0),Jt(e,r),qe(e,ge()),n;if(t===6)Jt(e,r);else{if(i=e.current.alternate,!(r&30)&&!fg(i)&&(t=xl(e,r),t===2&&(l=Qo(e),l!==0&&(r=l,t=ka(e,l))),t===1))throw n=li,bn(e,0),Jt(e,r),qe(e,ge()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(M(345));case 2:yn(e,Ve,Mt);break;case 3:if(Jt(e,r),(r&130023424)===r&&(t=ks+500-ge(),10<t)){if(tl(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Fe(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=ea(yn.bind(null,e,Ve,Mt),t);break}yn(e,Ve,Mt);break;case 4:if(Jt(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-xt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=ge()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*dg(r/1960))-r,10<r){e.timeoutHandle=ea(yn.bind(null,e,Ve,Mt),r);break}yn(e,Ve,Mt);break;case 5:yn(e,Ve,Mt);break;default:throw Error(M(329))}}}return qe(e,ge()),e.callbackNode===n?Jf.bind(null,e):null}function ka(e,t){var n=Fr;return e.current.memoizedState.isDehydrated&&(bn(e,t).flags|=256),e=xl(e,t),e!==2&&(t=Ve,Ve=n,t!==null&&wa(t)),e}function wa(e){Ve===null?Ve=e:Ve.push.apply(Ve,e)}function fg(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!wt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Jt(e,t){for(t&=~xs,t&=~Il,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-xt(t),r=1<<n;e[n]=-1,t&=~r}}function Zu(e){if(Z&6)throw Error(M(327));er();var t=tl(e,0);if(!(t&1))return qe(e,ge()),null;var n=xl(e,t);if(e.tag!==0&&n===2){var r=Qo(e);r!==0&&(t=r,n=ka(e,r))}if(n===1)throw n=li,bn(e,0),Jt(e,t),qe(e,ge()),n;if(n===6)throw Error(M(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,yn(e,Ve,Mt),qe(e,ge()),null}function ws(e,t){var n=Z;Z|=1;try{return e(t)}finally{Z=n,Z===0&&(ur=ge()+500,Tl&&hn())}}function Tn(e){en!==null&&en.tag===0&&!(Z&6)&&er();var t=Z;Z|=1;var n=dt.transition,r=re;try{if(dt.transition=null,re=1,e)return e()}finally{re=r,dt.transition=n,Z=t,!(Z&6)&&hn()}}function Ss(){Ze=Kn.current,ce(Kn)}function bn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,Um(n)),ye!==null)for(n=ye.return;n!==null;){var r=n;switch(ns(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&ol();break;case 3:ar(),ce(We),ce(Ie),ds();break;case 5:cs(r);break;case 4:ar();break;case 13:ce(fe);break;case 19:ce(fe);break;case 10:os(r.type._context);break;case 22:case 23:Ss()}n=n.return}if(Se=e,ye=e=un(e.current,null),Ne=Ze=t,ke=0,li=null,xs=Il=_n=0,Ve=Fr=null,wn!==null){for(t=0;t<wn.length;t++)if(n=wn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}wn=null}return e}function Zf(e,t){do{var n=ye;try{if(ls(),Ui.current=ml,hl){for(var r=pe.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}hl=!1}if(Nn=0,we=xe=pe=null,Dr=!1,ni=0,ys.current=null,n===null||n.return===null){ke=1,li=t,ye=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=Ne,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var c=s,d=a,f=d.tag;if(!(d.mode&1)&&(f===0||f===11||f===15)){var m=d.alternate;m?(d.updateQueue=m.updateQueue,d.memoizedState=m.memoizedState,d.lanes=m.lanes):(d.updateQueue=null,d.memoizedState=null)}var p=Bu(o);if(p!==null){p.flags&=-257,$u(p,o,a,l,t),p.mode&1&&Ou(l,c,t),t=p,s=c;var w=t.updateQueue;if(w===null){var S=new Set;S.add(s),t.updateQueue=S}else w.add(s);break e}else{if(!(t&1)){Ou(l,c,t),bs();break e}s=Error(M(426))}}else if(de&&a.mode&1){var I=Bu(o);if(I!==null){!(I.flags&65536)&&(I.flags|=256),$u(I,o,a,l,t),rs(sr(s,a));break e}}l=s=sr(s,a),ke!==4&&(ke=2),Fr===null?Fr=[l]:Fr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var h=Df(l,s,t);Iu(l,h);break e;case 1:a=s;var v=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof v.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(an===null||!an.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var b=Rf(l,a,t);Iu(l,b);break e}}l=l.return}while(l!==null)}np(n)}catch(E){t=E,ye===n&&n!==null&&(ye=n=n.return);continue}break}while(!0)}function ep(){var e=gl.current;return gl.current=ml,e===null?ml:e}function bs(){(ke===0||ke===3||ke===2)&&(ke=4),Se===null||!(_n&268435455)&&!(Il&268435455)||Jt(Se,Ne)}function xl(e,t){var n=Z;Z|=2;var r=ep();(Se!==e||Ne!==t)&&(Mt=null,bn(e,t));do try{pg();break}catch(i){Zf(e,i)}while(!0);if(ls(),Z=n,gl.current=r,ye!==null)throw Error(M(261));return Se=null,Ne=0,ke}function pg(){for(;ye!==null;)tp(ye)}function hg(){for(;ye!==null&&!Oh();)tp(ye)}function tp(e){var t=ip(e.alternate,e,Ze);e.memoizedProps=e.pendingProps,t===null?np(e):ye=t,ys.current=null}function np(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=ag(n,t),n!==null){n.flags&=32767,ye=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{ke=6,ye=null;return}}else if(n=og(n,t,Ze),n!==null){ye=n;return}if(t=t.sibling,t!==null){ye=t;return}ye=t=e}while(t!==null);ke===0&&(ke=5)}function yn(e,t,n){var r=re,i=dt.transition;try{dt.transition=null,re=1,mg(e,t,n,r)}finally{dt.transition=i,re=r}return null}function mg(e,t,n,r){do er();while(en!==null);if(Z&6)throw Error(M(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(M(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Yh(e,l),e===Se&&(ye=Se=null,Ne=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||Ti||(Ti=!0,lp(el,function(){return er(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=dt.transition,dt.transition=null;var o=re;re=1;var a=Z;Z|=4,ys.current=null,ug(e,n),Xf(n,e),Am(Jo),nl=!!Go,Jo=Go=null,e.current=n,cg(n),Bh(),Z=a,re=o,dt.transition=l}else e.current=n;if(Ti&&(Ti=!1,en=e,yl=i),l=e.pendingLanes,l===0&&(an=null),Vh(n.stateNode),qe(e,ge()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(vl)throw vl=!1,e=ya,ya=null,e;return yl&1&&e.tag!==0&&er(),l=e.pendingLanes,l&1?e===xa?Or++:(Or=0,xa=e):Or=0,hn(),null}function er(){if(en!==null){var e=Ad(yl),t=dt.transition,n=re;try{if(dt.transition=null,re=16>e?16:e,en===null)var r=!1;else{if(e=en,en=null,yl=0,Z&6)throw Error(M(331));var i=Z;for(Z|=4,B=e.current;B!==null;){var l=B,o=l.child;if(B.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var c=a[s];for(B=c;B!==null;){var d=B;switch(d.tag){case 0:case 11:case 15:Rr(8,d,l)}var f=d.child;if(f!==null)f.return=d,B=f;else for(;B!==null;){d=B;var m=d.sibling,p=d.return;if(qf(d),d===c){B=null;break}if(m!==null){m.return=p,B=m;break}B=p}}}var w=l.alternate;if(w!==null){var S=w.child;if(S!==null){w.child=null;do{var I=S.sibling;S.sibling=null,S=I}while(S!==null)}}B=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,B=o;else e:for(;B!==null;){if(l=B,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Rr(9,l,l.return)}var h=l.sibling;if(h!==null){h.return=l.return,B=h;break e}B=l.return}}var v=e.current;for(B=v;B!==null;){o=B;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,B=y;else e:for(o=v;B!==null;){if(a=B,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Pl(9,a)}}catch(E){me(a,a.return,E)}if(a===o){B=null;break e}var b=a.sibling;if(b!==null){b.return=a.return,B=b;break e}B=a.return}}if(Z=i,hn(),Nt&&typeof Nt.onPostCommitFiberRoot=="function")try{Nt.onPostCommitFiberRoot(jl,e)}catch{}r=!0}return r}finally{re=n,dt.transition=t}}return!1}function ec(e,t,n){t=sr(n,t),t=Df(e,t,1),e=on(e,t,1),t=Fe(),e!==null&&(si(e,1,t),qe(e,t))}function me(e,t,n){if(e.tag===3)ec(e,e,n);else for(;t!==null;){if(t.tag===3){ec(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(an===null||!an.has(r))){e=sr(n,e),e=Rf(t,e,1),t=on(t,e,1),e=Fe(),t!==null&&(si(t,1,e),qe(t,e));break}}t=t.return}}function gg(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Fe(),e.pingedLanes|=e.suspendedLanes&n,Se===e&&(Ne&n)===n&&(ke===4||ke===3&&(Ne&130023424)===Ne&&500>ge()-ks?bn(e,0):xs|=n),qe(e,t)}function rp(e,t){t===0&&(e.mode&1?(t=xi,xi<<=1,!(xi&130023424)&&(xi=4194304)):t=1);var n=Fe();e=$t(e,t),e!==null&&(si(e,t,n),qe(e,n))}function vg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),rp(e,n)}function yg(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(M(314))}r!==null&&r.delete(t),rp(e,n)}var ip;ip=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||We.current)He=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return He=!1,lg(e,t,n);He=!!(e.flags&131072)}else He=!1,de&&t.flags&1048576&&sf(t,ul,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Hi(e,t),e=t.pendingProps;var i=ir(t,Ie.current);Zn(t,n),i=ps(null,t,r,e,i,n);var l=hs();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,Qe(r)?(l=!0,al(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,ss(t),i.updater=Ll,t.stateNode=i,i._reactInternals=t,aa(t,r,e,n),t=ca(null,t,r,!0,l,n)):(t.tag=0,de&&l&&ts(t),Re(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Hi(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=kg(r),e=gt(r,e),i){case 0:t=ua(null,t,r,e,n);break e;case 1:t=Hu(null,t,r,e,n);break e;case 11:t=Uu(null,t,r,e,n);break e;case 14:t=Vu(null,t,r,gt(r.type,e),n);break e}throw Error(M(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:gt(r,i),ua(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:gt(r,i),Hu(e,t,r,i,n);case 3:e:{if($f(t),e===null)throw Error(M(387));r=t.pendingProps,l=t.memoizedState,i=l.element,hf(e,t),fl(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=sr(Error(M(423)),t),t=Wu(e,t,r,n,i);break e}else if(r!==i){i=sr(Error(M(424)),t),t=Wu(e,t,r,n,i);break e}else for(et=ln(t.stateNode.containerInfo.firstChild),nt=t,de=!0,yt=null,n=ff(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(lr(),r===i){t=Ut(e,t,n);break e}Re(e,t,r,n)}t=t.child}return t;case 5:return mf(t),e===null&&ia(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,Zo(r,i)?o=null:l!==null&&Zo(r,l)&&(t.flags|=32),Bf(e,t),Re(e,t,o,n),t.child;case 6:return e===null&&ia(t),null;case 13:return Uf(e,t,n);case 4:return us(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=or(t,null,r,n):Re(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:gt(r,i),Uu(e,t,r,i,n);case 7:return Re(e,t,t.pendingProps,n),t.child;case 8:return Re(e,t,t.pendingProps.children,n),t.child;case 12:return Re(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,ae(cl,r._currentValue),r._currentValue=o,l!==null)if(wt(l.value,o)){if(l.children===i.children&&!We.current){t=Ut(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Ft(-1,n&-n),s.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?s.next=s:(s.next=d.next,d.next=s),c.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),la(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(M(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),la(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}Re(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Zn(t,n),i=ft(i),r=r(i),t.flags|=1,Re(e,t,r,n),t.child;case 14:return r=t.type,i=gt(r,t.pendingProps),i=gt(r.type,i),Vu(e,t,r,i,n);case 15:return Ff(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:gt(r,i),Hi(e,t),t.tag=1,Qe(r)?(e=!0,al(t)):e=!1,Zn(t,n),Af(t,r,i),aa(t,r,i,n),ca(null,t,r,!0,e,n);case 19:return Vf(e,t,n);case 22:return Of(e,t,n)}throw Error(M(156,t.tag))};function lp(e,t){return Ld(e,t)}function xg(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function ct(e,t,n,r){return new xg(e,t,n,r)}function js(e){return e=e.prototype,!(!e||!e.isReactComponent)}function kg(e){if(typeof e=="function")return js(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Va)return 11;if(e===Ha)return 14}return 2}function un(e,t){var n=e.alternate;return n===null?(n=ct(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function qi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")js(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Fn:return jn(n.children,i,l,t);case Ua:o=8,i|=8;break;case Lo:return e=ct(12,n,t,i|2),e.elementType=Lo,e.lanes=l,e;case Po:return e=ct(13,n,t,i),e.elementType=Po,e.lanes=l,e;case Io:return e=ct(19,n,t,i),e.elementType=Io,e.lanes=l,e;case hd:return Ml(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case fd:o=10;break e;case pd:o=9;break e;case Va:o=11;break e;case Ha:o=14;break e;case Yt:o=16,r=null;break e}throw Error(M(130,e==null?e:typeof e,""))}return t=ct(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function jn(e,t,n,r){return e=ct(7,e,r,t),e.lanes=n,e}function Ml(e,t,n,r){return e=ct(22,e,r,t),e.elementType=hd,e.lanes=n,e.stateNode={isHidden:!1},e}function ho(e,t,n){return e=ct(6,e,null,t),e.lanes=n,e}function mo(e,t,n){return t=ct(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function wg(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=Kl(0),this.expirationTimes=Kl(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=Kl(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function Cs(e,t,n,r,i,l,o,a,s){return e=new wg(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=ct(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},ss(l),e}function Sg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:Rn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function op(e){if(!e)return dn;e=e._reactInternals;e:{if(Ln(e)!==e||e.tag!==1)throw Error(M(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(Qe(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(M(171))}if(e.tag===1){var n=e.type;if(Qe(n))return of(e,n,t)}return t}function ap(e,t,n,r,i,l,o,a,s){return e=Cs(n,r,!0,e,i,l,o,a,s),e.context=op(null),n=e.current,r=Fe(),i=sn(n),l=Ft(r,i),l.callback=t??null,on(n,l,i),e.current.lanes=i,si(e,i,r),qe(e,r),e}function Al(e,t,n,r){var i=t.current,l=Fe(),o=sn(i);return n=op(n),t.context===null?t.context=n:t.pendingContext=n,t=Ft(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=on(i,t,o),e!==null&&(kt(e,i,o,l),$i(e,i,o)),o}function kl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function tc(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function Es(e,t){tc(e,t),(e=e.alternate)&&tc(e,t)}function bg(){return null}var sp=typeof reportError=="function"?reportError:function(e){console.error(e)};function Ns(e){this._internalRoot=e}Dl.prototype.render=Ns.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(M(409));Al(e,t,null,null)};Dl.prototype.unmount=Ns.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;Tn(function(){Al(null,e,null,null)}),t[Bt]=null}};function Dl(e){this._internalRoot=e}Dl.prototype.unstable_scheduleHydration=function(e){if(e){var t=Fd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Gt.length&&t!==0&&t<Gt[n].priority;n++);Gt.splice(n,0,e),n===0&&Bd(e)}};function _s(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Rl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function nc(){}function jg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=kl(o);l.call(c)}}var o=ap(t,r,e,0,null,!1,!1,"",nc);return e._reactRootContainer=o,e[Bt]=o.current,Gr(e.nodeType===8?e.parentNode:e),Tn(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=kl(s);a.call(c)}}var s=Cs(e,0,!1,null,null,!1,!1,"",nc);return e._reactRootContainer=s,e[Bt]=s.current,Gr(e.nodeType===8?e.parentNode:e),Tn(function(){Al(t,s,n,r)}),s}function Fl(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=kl(o);a.call(s)}}Al(t,o,e,i)}else o=jg(n,t,e,i,r);return kl(o)}Dd=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=Tr(t.pendingLanes);n!==0&&(qa(t,n|1),qe(t,ge()),!(Z&6)&&(ur=ge()+500,hn()))}break;case 13:Tn(function(){var r=$t(e,1);if(r!==null){var i=Fe();kt(r,e,1,i)}}),Es(e,1)}};Ka=function(e){if(e.tag===13){var t=$t(e,134217728);if(t!==null){var n=Fe();kt(t,e,134217728,n)}Es(e,134217728)}};Rd=function(e){if(e.tag===13){var t=sn(e),n=$t(e,t);if(n!==null){var r=Fe();kt(n,e,t,r)}Es(e,t)}};Fd=function(){return re};Od=function(e,t){var n=re;try{return re=e,t()}finally{re=n}};Vo=function(e,t,n){switch(t){case"input":if(Do(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=_l(r);if(!i)throw Error(M(90));gd(r),Do(r,i)}}}break;case"textarea":yd(e,n);break;case"select":t=n.value,t!=null&&Yn(e,!!n.multiple,t,!1)}};Cd=ws;Ed=Tn;var Cg={usingClientEntryPoint:!1,Events:[ci,Un,_l,bd,jd,ws]},jr={findFiberByHostInstance:kn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},Eg={bundleType:jr.bundleType,version:jr.version,rendererPackageName:jr.rendererPackageName,rendererConfig:jr.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Vt.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=Td(e),e===null?null:e.stateNode},findFiberByHostInstance:jr.findFiberByHostInstance||bg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var zi=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!zi.isDisabled&&zi.supportsFiber)try{jl=zi.inject(Eg),Nt=zi}catch{}}it.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=Cg;it.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!_s(t))throw Error(M(200));return Sg(e,t,null,n)};it.createRoot=function(e,t){if(!_s(e))throw Error(M(299));var n=!1,r="",i=sp;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=Cs(e,1,!1,null,null,n,!1,r,i),e[Bt]=t.current,Gr(e.nodeType===8?e.parentNode:e),new Ns(t)};it.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(M(188)):(e=Object.keys(e).join(","),Error(M(268,e)));return e=Td(t),e=e===null?null:e.stateNode,e};it.flushSync=function(e){return Tn(e)};it.hydrate=function(e,t,n){if(!Rl(t))throw Error(M(200));return Fl(null,e,t,!0,n)};it.hydrateRoot=function(e,t,n){if(!_s(e))throw Error(M(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=sp;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=ap(t,null,e,1,n??null,i,!1,l,o),e[Bt]=t.current,Gr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new Dl(t)};it.render=function(e,t,n){if(!Rl(t))throw Error(M(200));return Fl(null,e,t,!1,n)};it.unmountComponentAtNode=function(e){if(!Rl(e))throw Error(M(40));return e._reactRootContainer?(Tn(function(){Fl(null,null,e,!1,function(){e._reactRootContainer=null,e[Bt]=null})}),!0):!1};it.unstable_batchedUpdates=ws;it.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Rl(n))throw Error(M(200));if(e==null||e._reactInternals===void 0)throw Error(M(38));return Fl(e,t,n,!1,r)};it.version="18.3.1-next-f1338f8080-20240426";function up(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(up)}catch(e){console.error(e)}}up(),sd.exports=it;var Ng=sd.exports,rc=Ng;To.createRoot=rc.createRoot,To.hydrateRoot=rc.hydrateRoot;const _g=new Set(["user","human"]);function Tg(e){return e?_g.has(e.toLowerCase()):!1}function cp(e){return Tg(e)?"You (Human)":e}const zg="",Lg=({selection:e,onSelect:t,onRefresh:n})=>{const[r,i]=O.useState(null),[l,o]=O.useState(new Set(["all"])),[a,s]=O.useState(!0),[c,d]=O.useState(null),f=async()=>{try{const v=await fetch(`${zg}/api/hierarchy`);if(!v.ok)throw new Error("Failed to fetch hierarchy");const y=await v.json();i(y),d(null)}catch(v){d(v instanceof Error?v.message:"Unknown error")}finally{s(!1)}};O.useEffect(()=>{f();const v=setInterval(f,5e3);return()=>clearInterval(v)},[]);const m=v=>{o(y=>{const b=new Set(y);return b.has(v)?b.delete(v):b.add(v),b})},p=v=>{var y;if(v.type==="root")t({type:"overview"});else if(v.type==="agent")t({type:"agent",agentId:v.id});else if(v.type==="thread"){const b=(y=r==null?void 0:r.root.children)==null?void 0:y.find(E=>{var k;return(k=E.children)==null?void 0:k.some(C=>C.id===v.id)});t({type:"thread",agentId:b==null?void 0:b.id,threadId:v.id})}},w=v=>v.type==="root"&&e.type==="overview"||v.type==="agent"&&e.type==="agent"&&e.agentId===v.id||v.type==="thread"&&e.threadId===v.id,S=v=>!v||v.length===0?null:u.jsx("span",{className:"badges",children:v.map((y,b)=>u.jsxs("span",{className:`badge badge-${y.type}`,title:`${y.count} ${y.type}`,children:[y.type==="pending"&&"⏳",y.type==="unread"&&"📬",y.type==="running"&&"▶️",y.count]},b))}),I=v=>{if(!v)return null;const y={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsx("span",{className:"status-indicator",style:{backgroundColor:y[v]||y.idle},title:v})},h=(v,y=0)=>{const b=l.has(v.id),E=v.children&&v.children.length>0,k=w(v);return u.jsxs("div",{className:"tree-node",children:[u.jsxs("div",{className:`tree-node-content ${k?"selected":""} ${v.type}`,style:{paddingLeft:`${y*16+8}px`},onClick:()=>p(v),children:[E&&u.jsx("span",{className:`expand-icon ${b?"expanded":""}`,onClick:C=>{C.stopPropagation(),m(v.id)},children:b?"▼":"▶"}),!E&&u.jsx("span",{className:"expand-icon-placeholder"}),v.type==="agent"&&I(v.status),u.jsx("span",{className:"node-label",children:v.type==="agent"?cp(v.id):v.label}),S(v.badges)]}),E&&b&&u.jsx("div",{className:"tree-children",children:v.children.map(C=>h(C,y+1))})]},v.id)};return a&&!r?u.jsx("div",{className:"hierarchy-tree loading",children:"Loading..."}):c?u.jsxs("div",{className:"hierarchy-tree error",children:[u.jsxs("p",{children:["Error: ",c]}),u.jsx("button",{onClick:f,children:"Retry"})]}):u.jsxs("div",{className:"hierarchy-tree",children:[u.jsxs("div",{className:"tree-header",children:[u.jsx("h3",{children:"Agents"}),u.jsx("button",{className:"refresh-btn",onClick:()=>{f(),n==null||n()},title:"Refresh",children:"\\u21BB"})]}),u.jsx("div",{className:"tree-content",children:r&&h(r.root)}),r&&u.jsx("div",{className:"tree-footer",children:u.jsxs("div",{className:"aggregate-stats",children:[u.jsxs("span",{title:"Total agents",children:[r.aggregate.total_agents," agents"]}),u.jsxs("span",{title:"Active",children:[r.aggregate.active_agents," active"]}),r.aggregate.pending_approvals>0&&u.jsxs("span",{className:"pending",title:"Pending approvals",children:[r.aggregate.pending_approvals," pending"]})]})})]})},Pg="_card_1d3of_1",Ig="_compact_1d3of_9",Mg="_title_1d3of_13",Ag="_metricsGrid_1d3of_20",Dg="_metricItem_1d3of_26",Rg="_metricLabel_1d3of_32",Fg="_metricValue_1d3of_39",Og="_cost_1d3of_46",Bg="_averages_1d3of_50",$g="_averagesLabel_1d3of_61",Ug="_avgItem_1d3of_65",Vg="_compactRow_1d3of_72",Hg="_compactLabel_1d3of_80",Wg="_compactValue_1d3of_84",Qg="_loading_1d3of_91",qg="_error_1d3of_97",Kg="_errorText_1d3of_101",K={card:Pg,compact:Ig,title:Mg,metricsGrid:Ag,metricItem:Dg,metricLabel:Rg,metricValue:Fg,cost:Og,averages:Bg,averagesLabel:$g,avgItem:Ug,compactRow:Vg,compactLabel:Hg,compactValue:Wg,loading:Qg,error:qg,errorText:Kg};function ic(e){return e<1e3?`${e}ms`:e<6e4?`${(e/1e3).toFixed(1)}s`:e<36e5?`${(e/6e4).toFixed(1)}m`:`${(e/36e5).toFixed(1)}h`}function An(e){return e.toLocaleString()}function go(e){return e===0?"$0.00":e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`}function Yg({scopeType:e,scopeId:t="",title:n,compact:r=!1}){const[i,l]=O.useState(null),[o,a]=O.useState(!0),[s,c]=O.useState(null),d=O.useCallback(async()=>{try{let m="/api/metrics";e!=="global"&&(m=`/api/metrics/${e}/${t}`);const p=await fetch(m);if(!p.ok)throw new Error(`Failed to fetch metrics: ${p.status}`);const w=await p.json();l(w),c(null)}catch(m){c(m instanceof Error?m.message:"Failed to load metrics")}finally{a(!1)}},[e,t]);if(O.useEffect(()=>{d();const m=setInterval(d,3e4);return()=>clearInterval(m)},[d]),o)return u.jsx("div",{className:`${K.card} ${r?K.compact:""}`,children:u.jsx("div",{className:K.loading,children:"Loading metrics..."})});if(s)return u.jsx("div",{className:`${K.card} ${r?K.compact:""} ${K.error}`,children:u.jsx("div",{className:K.errorText,children:s})});if(!i)return null;const f=n||(e==="global"?"Global Metrics":e==="agent"?`Agent: ${t}`:`Thread: ${t.slice(0,12)}...`);return r?u.jsx("div",{className:`${K.card} ${K.compact}`,children:u.jsxs("div",{className:K.compactRow,children:[u.jsx("span",{className:K.compactLabel,children:"Runs:"}),u.jsx("span",{className:K.compactValue,children:An(i.total_runs)}),u.jsx("span",{className:K.compactLabel,children:"Tokens:"}),u.jsx("span",{className:K.compactValue,children:An(i.total_tokens)}),u.jsx("span",{className:K.compactLabel,children:"Cost:"}),u.jsx("span",{className:K.compactValue,children:go(i.total_cost)})]})}):u.jsxs("div",{className:K.card,children:[u.jsx("h3",{className:K.title,children:f}),u.jsxs("div",{className:K.metricsGrid,children:[u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Runs"}),u.jsx("span",{className:K.metricValue,children:An(i.total_runs)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Tokens"}),u.jsx("span",{className:K.metricValue,children:An(i.total_tokens)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Cost"}),u.jsx("span",{className:`${K.metricValue} ${K.cost}`,children:go(i.total_cost)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Duration"}),u.jsx("span",{className:K.metricValue,children:ic(i.total_duration_ms)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Files Modified"}),u.jsx("span",{className:K.metricValue,children:An(i.total_files_modified)})]})]}),i.total_runs>0&&u.jsxs("div",{className:K.averages,children:[u.jsx("span",{className:K.averagesLabel,children:"Averages per run:"}),u.jsxs("span",{className:K.avgItem,children:[An(Math.round(i.avg_tokens_per_run))," tokens"]}),u.jsx("span",{className:K.avgItem,children:go(i.avg_cost_per_run)}),u.jsx("span",{className:K.avgItem,children:ic(Math.round(i.avg_duration_per_run))})]})]})}const Xg="_container_1q26w_1",Gg="_title_1q26w_9",Jg="_header_1q26w_16",Zg="_metricLabel_1q26w_25",ev="_total_1q26w_31",tv="_chart_1q26w_37",nv="_barContainer_1q26w_46",rv="_barWrapper_1q26w_55",iv="_bar_1q26w_46",lv="_barValue_1q26w_80",ov="_label_1q26w_89",av="_loading_1q26w_98",sv="_error_1q26w_99",uv="_empty_1q26w_100",Ce={container:Xg,title:Gg,header:Jg,metricLabel:Zg,total:ev,chart:tv,barContainer:nv,barWrapper:rv,bar:iv,barValue:lv,label:ov,loading:av,error:sv,empty:uv};function lc({scopeType:e,scopeId:t,period:n="hour",limit:r=24,metric:i="cost",title:l}){const[o,a]=O.useState([]),[s,c]=O.useState(!0),[d,f]=O.useState(null);O.useEffect(()=>{const y=async()=>{try{c(!0);const E=await fetch(`/api/metrics/trends/${e}/${t}?period=${n}&limit=${r}`);if(!E.ok)throw new Error("Failed to fetch trends");const k=await E.json();a(k||[]),f(null)}catch(E){f(E instanceof Error?E.message:"Unknown error")}finally{c(!1)}};y();const b=setInterval(y,6e4);return()=>clearInterval(b)},[e,t,n,r]);const m=y=>{switch(i){case"cost":return y.cost;case"tokens":return y.tokens;case"duration":return y.duration_ms/1e3;case"runs":return y.runs;default:return y.cost}},p=y=>{switch(i){case"cost":return`$${y.toFixed(2)}`;case"tokens":return y>=1e3?`${(y/1e3).toFixed(1)}k`:y.toString();case"duration":return`${y.toFixed(1)}s`;case"runs":return y.toString();default:return y.toFixed(2)}},w=y=>{const b=new Date(y);return n==="minute"?b.toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}):n==="hour"?b.toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}):b.toLocaleDateString([],{month:"short",day:"numeric"})},S=()=>{switch(i){case"cost":return"Cost ($)";case"tokens":return"Tokens";case"duration":return"Duration (s)";case"runs":return"Runs";default:return""}};if(s&&o.length===0)return u.jsx("div",{className:Ce.container,children:u.jsx("div",{className:Ce.loading,children:"Loading trends..."})});if(d)return u.jsx("div",{className:Ce.container,children:u.jsx("div",{className:Ce.error,children:d})});if(o.length===0)return u.jsx("div",{className:Ce.container,children:u.jsx("div",{className:Ce.empty,children:"No data available"})});const I=o.map(m),h=Math.max(...I,1),v=I.reduce((y,b)=>y+b,0);return u.jsxs("div",{className:Ce.container,children:[l&&u.jsx("div",{className:Ce.title,children:l}),u.jsxs("div",{className:Ce.header,children:[u.jsx("span",{className:Ce.metricLabel,children:S()}),u.jsxs("span",{className:Ce.total,children:["Total: ",p(v)]})]}),u.jsx("div",{className:Ce.chart,children:o.map((y,b)=>{const E=m(y),k=E/h*100;return u.jsxs("div",{className:Ce.barContainer,children:[u.jsx("div",{className:Ce.barWrapper,children:u.jsx("div",{className:Ce.bar,style:{height:`${Math.max(k,2)}%`},title:`${w(y.period_start)}: ${p(E)}`,children:k>30&&u.jsx("span",{className:Ce.barValue,children:p(E)})})}),b%Math.ceil(o.length/6)===0&&u.jsx("span",{className:Ce.label,children:w(y.period_start)})]},y.period_start)})})]})}const Ge=({title:e,value:t,color:n="default",small:r})=>u.jsxs("div",{className:`stat-card stat-${n} ${r?"stat-small":""}`,children:[u.jsx("div",{className:"stat-value",children:t}),u.jsx("div",{className:"stat-title",children:e})]}),cv=e=>{if(e<1e3)return`${e}ms`;const t=e/1e3;if(t<60)return`${t.toFixed(1)}s`;const n=Math.floor(t/60),r=(t%60).toFixed(0);return`${n}m ${r}s`},dv=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,Li=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}k`:e.toString(),fv=({agent:e,onClick:t})=>{var o,a,s,c,d;const n=((o=e.children)==null?void 0:o.length)||0,r=((s=(a=e.badges)==null?void 0:a.find(f=>f.type==="pending"))==null?void 0:s.count)||0,i=((d=(c=e.badges)==null?void 0:c.find(f=>f.type==="running"))==null?void 0:d.count)||0,l={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsxs("div",{className:"agent-card",onClick:t,children:[u.jsxs("div",{className:"agent-card-header",children:[u.jsx("span",{className:"agent-status-dot",style:{backgroundColor:l[e.status||"idle"]}}),u.jsx("span",{className:"agent-name",children:cp(e.id)})]}),u.jsxs("div",{className:"agent-card-stats",children:[u.jsxs("span",{className:"agent-stat",children:[u.jsx("span",{className:"agent-stat-value",children:n}),u.jsx("span",{className:"agent-stat-label",children:"threads"})]}),r>0&&u.jsxs("span",{className:"agent-stat pending",children:[u.jsx("span",{className:"agent-stat-value",children:r}),u.jsx("span",{className:"agent-stat-label",children:"pending"})]}),i>0&&u.jsxs("span",{className:"agent-stat running",children:[u.jsx("span",{className:"agent-stat-value",children:i}),u.jsx("span",{className:"agent-stat-label",children:"running"})]})]})]})},pv=({aggregate:e,agents:t,onSelectAgent:n})=>{const r=e.execution,i=r&&r.total_executions>0,l=i?Math.round(r.successful_executions/r.total_executions*100):0;return u.jsxs("div",{className:"all-agents-overview",children:[u.jsx("div",{className:"overview-header",children:u.jsx("h2",{children:"All Agents Overview"})}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Ge,{title:"Total Agents",value:e.total_agents}),u.jsx(Ge,{title:"Active",value:e.active_agents,color:"green"}),u.jsx(Ge,{title:"Pending Approvals",value:e.pending_approvals,color:"orange"}),u.jsx(Ge,{title:"Total Threads",value:e.total_threads,color:"blue"})]}),u.jsxs("div",{className:"metrics-section",children:[u.jsx("h3",{children:"Usage Metrics (Today)"}),u.jsx(Yg,{scopeType:"global",title:"Global Metrics"})]}),u.jsxs("div",{className:"trends-section",children:[u.jsx("h3",{children:"Usage Trends (Last 24 Hours)"}),u.jsxs("div",{className:"trends-grid",children:[u.jsx(lc,{scopeType:"global",scopeId:"",period:"hour",limit:24,metric:"cost",title:"Cost"}),u.jsx(lc,{scopeType:"global",scopeId:"",period:"hour",limit:24,metric:"tokens",title:"Tokens"})]})]}),i&&u.jsxs("div",{className:"execution-stats-section",children:[u.jsx("h3",{children:"Execution Statistics"}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Ge,{title:"Total Executions",value:r.total_executions,color:"purple"}),u.jsx(Ge,{title:"Success Rate",value:`${l}%`,color:"green"}),u.jsx(Ge,{title:"Total Duration",value:cv(r.total_duration_ms)}),u.jsx(Ge,{title:"Total Cost",value:dv(r.total_cost),color:"orange"})]}),u.jsxs("div",{className:"stats-row token-stats",children:[u.jsx(Ge,{title:"Input Tokens",value:Li(r.total_input_tokens),small:!0}),u.jsx(Ge,{title:"Output Tokens",value:Li(r.total_output_tokens),small:!0}),u.jsx(Ge,{title:"Cache Read",value:Li(r.total_cache_read_tokens),small:!0}),u.jsx(Ge,{title:"Cache Created",value:Li(r.total_cache_create_tokens),small:!0}),u.jsx(Ge,{title:"Files Created",value:r.total_files_created,small:!0})]})]}),u.jsxs("div",{className:"agents-section",children:[u.jsx("h3",{children:"Agents"}),u.jsxs("div",{className:"agent-cards-grid",children:[t.map(o=>u.jsx(fv,{agent:o,onClick:()=>n(o.id)},o.id)),t.length===0&&u.jsx("div",{className:"no-agents",children:"No agents found. Start an agent to see it here."})]})]})]})},hv=({items:e})=>u.jsx("nav",{className:"breadcrumb",children:e.map((t,n)=>u.jsxs(Kt.Fragment,{children:[n>0&&u.jsx("span",{className:"breadcrumb-separator",children:"/"}),t.onClick?u.jsx("button",{className:"breadcrumb-link",onClick:t.onClick,children:t.label}):u.jsx("span",{className:"breadcrumb-current",children:t.label})]},n))}),Pt={plus:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),u.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"}),u.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),u.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),u.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),u.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),u.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),u.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("polyline",{points:"3 6 5 6 21 6"}),u.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:u.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},mv=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=O.useState(!1),[c,d]=O.useState(""),[f,m]=O.useState(null),[p,w]=O.useState(""),[S,I]=O.useState(null),h=()=>{c.trim()&&(r(c.trim()),d(""),s(!1))},v=T=>{T.key==="Enter"&&!T.shiftKey?(T.preventDefault(),h()):T.key==="Escape"&&(s(!1),d(""))},y=(T,D)=>{D.stopPropagation(),m(T.id),w(T.title)},b=T=>{var D;p.trim()&&p.trim()!==((D=e.find(W=>W.id===T))==null?void 0:D.title)&&l(T,p.trim()),m(null),w("")},E=()=>{m(null),w("")},k=(T,D)=>{T.key==="Enter"?(T.preventDefault(),b(D)):T.key==="Escape"&&E()},C=(T,D)=>{D.stopPropagation(),I(T)},_=(T,D)=>{D.stopPropagation(),i(T),I(null)},R=T=>{T.stopPropagation(),I(null)},P=T=>{const D=new Date(T),X=new Date().getTime()-D.getTime(),U=Math.floor(X/6e4),Q=Math.floor(X/36e5),ie=Math.floor(X/864e5);return U<1?"now":U<60?`${U}m`:Q<24?`${Q}h`:ie<7?`${ie}d`:D.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return u.jsxs("div",{className:"thread-list",children:[u.jsxs("div",{className:"list-header",children:[u.jsx("h2",{children:"Conversations"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:Pt.plus})]}),a&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:c,onChange:T=>d(T.target.value),onKeyDown:v,placeholder:"Conversation title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:h,children:"Create"})]})]}),u.jsx("div",{className:"thread-items",children:e.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Pt.hash}),u.jsx("p",{children:"No conversations yet"}),u.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(T=>{const D=o.get(T.id)||0,W=T.id===t,X=f===T.id,U=S===T.id;return u.jsxs("div",{className:`thread-item ${W?"selected":""} ${D>0?"has-unread":""}`,onClick:()=>!X&&n(T.id),children:[u.jsx("div",{className:`status-dot ${T.status}`}),u.jsxs("div",{className:"thread-content",children:[u.jsx("div",{className:"thread-title-row",children:X?u.jsxs("div",{className:"edit-title-form",onClick:Q=>Q.stopPropagation(),children:[u.jsx("input",{type:"text",value:p,onChange:Q=>w(Q.target.value),onKeyDown:Q=>k(Q,T.id),autoFocus:!0}),u.jsx("button",{className:"edit-action save",onClick:()=>b(T.id),title:"Save",children:Pt.check}),u.jsx("button",{className:"edit-action cancel",onClick:E,title:"Cancel",children:Pt.x})]}):u.jsxs(u.Fragment,{children:[u.jsx("span",{className:"thread-title",children:T.title}),u.jsx("span",{className:"thread-time",children:P(T.updated_at)})]})}),u.jsxs("div",{className:"thread-meta",children:[T.target_agent&&u.jsxs("span",{className:"thread-agent",title:`Target: ${T.target_agent}`,children:[Pt.bot,T.target_agent]}),u.jsxs("span",{className:"thread-seq",children:["#",T.last_seq]})]})]}),!X&&!U&&u.jsxs("div",{className:"thread-actions",children:[u.jsx("button",{className:"action-btn edit",onClick:Q=>y(T,Q),title:"Rename",children:Pt.edit}),u.jsx("button",{className:"action-btn delete",onClick:Q=>C(T.id,Q),title:"Delete",children:Pt.trash})]}),U&&u.jsxs("div",{className:"delete-confirm",onClick:Q=>Q.stopPropagation(),children:[u.jsx("span",{className:"confirm-text",children:"Delete?"}),u.jsx("button",{className:"confirm-btn yes",onClick:Q=>_(T.id,Q),title:"Confirm delete",children:Pt.check}),u.jsx("button",{className:"confirm-btn no",onClick:R,title:"Cancel",children:Pt.x})]}),D>0&&!U&&u.jsx("span",{className:"unread-badge",children:D})]},T.id)})}),u.jsx("style",{children:`
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
      `})]})};function gv(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const vv=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,yv=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,xv={};function oc(e,t){return(xv.jsx?yv:vv).test(e)}const kv=/[ \t\n\f\r]/g;function wv(e){return typeof e=="object"?e.type==="text"?ac(e.value):!1:ac(e)}function ac(e){return e.replace(kv,"")===""}class fi{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}fi.prototype.normal={};fi.prototype.property={};fi.prototype.space=void 0;function dp(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new fi(n,r,t)}function Sa(e){return e.toLowerCase()}class Ye{constructor(t,n){this.attribute=n,this.property=t}}Ye.prototype.attribute="";Ye.prototype.booleanish=!1;Ye.prototype.boolean=!1;Ye.prototype.commaOrSpaceSeparated=!1;Ye.prototype.commaSeparated=!1;Ye.prototype.defined=!1;Ye.prototype.mustUseProperty=!1;Ye.prototype.number=!1;Ye.prototype.overloadedBoolean=!1;Ye.prototype.property="";Ye.prototype.spaceSeparated=!1;Ye.prototype.space=void 0;let Sv=0;const q=Pn(),ve=Pn(),ba=Pn(),A=Pn(),oe=Pn(),tr=Pn(),Je=Pn();function Pn(){return 2**++Sv}const ja=Object.freeze(Object.defineProperty({__proto__:null,boolean:q,booleanish:ve,commaOrSpaceSeparated:Je,commaSeparated:tr,number:A,overloadedBoolean:ba,spaceSeparated:oe},Symbol.toStringTag,{value:"Module"})),vo=Object.keys(ja);class Ts extends Ye{constructor(t,n,r,i){let l=-1;if(super(t,n),sc(this,"space",i),typeof r=="number")for(;++l<vo.length;){const o=vo[l];sc(this,vo[l],(r&ja[o])===ja[o])}}}Ts.prototype.defined=!0;function sc(e,t,n){n&&(e[t]=n)}function pr(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new Ts(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[Sa(r)]=r,n[Sa(l.attribute)]=r}return new fi(t,n,e.space)}const fp=pr({properties:{ariaActiveDescendant:null,ariaAtomic:ve,ariaAutoComplete:null,ariaBusy:ve,ariaChecked:ve,ariaColCount:A,ariaColIndex:A,ariaColSpan:A,ariaControls:oe,ariaCurrent:null,ariaDescribedBy:oe,ariaDetails:null,ariaDisabled:ve,ariaDropEffect:oe,ariaErrorMessage:null,ariaExpanded:ve,ariaFlowTo:oe,ariaGrabbed:ve,ariaHasPopup:null,ariaHidden:ve,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:oe,ariaLevel:A,ariaLive:null,ariaModal:ve,ariaMultiLine:ve,ariaMultiSelectable:ve,ariaOrientation:null,ariaOwns:oe,ariaPlaceholder:null,ariaPosInSet:A,ariaPressed:ve,ariaReadOnly:ve,ariaRelevant:null,ariaRequired:ve,ariaRoleDescription:oe,ariaRowCount:A,ariaRowIndex:A,ariaRowSpan:A,ariaSelected:ve,ariaSetSize:A,ariaSort:null,ariaValueMax:A,ariaValueMin:A,ariaValueNow:A,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function pp(e,t){return t in e?e[t]:t}function hp(e,t){return pp(e,t.toLowerCase())}const bv=pr({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:tr,acceptCharset:oe,accessKey:oe,action:null,allow:null,allowFullScreen:q,allowPaymentRequest:q,allowUserMedia:q,alt:null,as:null,async:q,autoCapitalize:null,autoComplete:oe,autoFocus:q,autoPlay:q,blocking:oe,capture:null,charSet:null,checked:q,cite:null,className:oe,cols:A,colSpan:null,content:null,contentEditable:ve,controls:q,controlsList:oe,coords:A|tr,crossOrigin:null,data:null,dateTime:null,decoding:null,default:q,defer:q,dir:null,dirName:null,disabled:q,download:ba,draggable:ve,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:q,formTarget:null,headers:oe,height:A,hidden:ba,high:A,href:null,hrefLang:null,htmlFor:oe,httpEquiv:oe,id:null,imageSizes:null,imageSrcSet:null,inert:q,inputMode:null,integrity:null,is:null,isMap:q,itemId:null,itemProp:oe,itemRef:oe,itemScope:q,itemType:oe,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:q,low:A,manifest:null,max:null,maxLength:A,media:null,method:null,min:null,minLength:A,multiple:q,muted:q,name:null,nonce:null,noModule:q,noValidate:q,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:q,optimum:A,pattern:null,ping:oe,placeholder:null,playsInline:q,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:q,referrerPolicy:null,rel:oe,required:q,reversed:q,rows:A,rowSpan:A,sandbox:oe,scope:null,scoped:q,seamless:q,selected:q,shadowRootClonable:q,shadowRootDelegatesFocus:q,shadowRootMode:null,shape:null,size:A,sizes:null,slot:null,span:A,spellCheck:ve,src:null,srcDoc:null,srcLang:null,srcSet:null,start:A,step:null,style:null,tabIndex:A,target:null,title:null,translate:null,type:null,typeMustMatch:q,useMap:null,value:ve,width:A,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:oe,axis:null,background:null,bgColor:null,border:A,borderColor:null,bottomMargin:A,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:q,declare:q,event:null,face:null,frame:null,frameBorder:null,hSpace:A,leftMargin:A,link:null,longDesc:null,lowSrc:null,marginHeight:A,marginWidth:A,noResize:q,noHref:q,noShade:q,noWrap:q,object:null,profile:null,prompt:null,rev:null,rightMargin:A,rules:null,scheme:null,scrolling:ve,standby:null,summary:null,text:null,topMargin:A,valueType:null,version:null,vAlign:null,vLink:null,vSpace:A,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:q,disableRemotePlayback:q,prefix:null,property:null,results:A,security:null,unselectable:null},space:"html",transform:hp}),jv=pr({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:Je,accentHeight:A,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:A,amplitude:A,arabicForm:null,ascent:A,attributeName:null,attributeType:null,azimuth:A,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:A,by:null,calcMode:null,capHeight:A,className:oe,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:A,diffuseConstant:A,direction:null,display:null,dur:null,divisor:A,dominantBaseline:null,download:q,dx:null,dy:null,edgeMode:null,editable:null,elevation:A,enableBackground:null,end:null,event:null,exponent:A,externalResourcesRequired:null,fill:null,fillOpacity:A,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:tr,g2:tr,glyphName:tr,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:A,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:A,horizOriginX:A,horizOriginY:A,id:null,ideographic:A,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:A,k:A,k1:A,k2:A,k3:A,k4:A,kernelMatrix:Je,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:A,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:A,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:A,overlineThickness:A,paintOrder:null,panose1:null,path:null,pathLength:A,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:oe,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:A,pointsAtY:A,pointsAtZ:A,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:Je,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:Je,rev:Je,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:Je,requiredFeatures:Je,requiredFonts:Je,requiredFormats:Je,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:A,specularExponent:A,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:A,strikethroughThickness:A,string:null,stroke:null,strokeDashArray:Je,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:A,strokeOpacity:A,strokeWidth:null,style:null,surfaceScale:A,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:Je,tabIndex:A,tableValues:null,target:null,targetX:A,targetY:A,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:Je,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:A,underlineThickness:A,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:A,values:null,vAlphabetic:A,vMathematical:A,vectorEffect:null,vHanging:A,vIdeographic:A,version:null,vertAdvY:A,vertOriginX:A,vertOriginY:A,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:A,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:pp}),mp=pr({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),gp=pr({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:hp}),vp=pr({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),Cv={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},Ev=/[A-Z]/g,uc=/-[a-z]/g,Nv=/^data[-\w.:]+$/i;function _v(e,t){const n=Sa(t);let r=t,i=Ye;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&Nv.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(uc,zv);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!uc.test(l)){let o=l.replace(Ev,Tv);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=Ts}return new i(r,t)}function Tv(e){return"-"+e.toLowerCase()}function zv(e){return e.charAt(1).toUpperCase()}const Lv=dp([fp,bv,mp,gp,vp],"html"),zs=dp([fp,jv,mp,gp,vp],"svg");function Pv(e){return e.join(" ").trim()}var Ls={},cc=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,Iv=/\n/g,Mv=/^\s*/,Av=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,Dv=/^:\s*/,Rv=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,Fv=/^[;\s]*/,Ov=/^\s+|\s+$/g,Bv=`
`,dc="/",fc="*",xn="",$v="comment",Uv="declaration";function Vv(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(w){var S=w.match(Iv);S&&(n+=S.length);var I=w.lastIndexOf(Bv);r=~I?w.length-I:r+w.length}function l(){var w={line:n,column:r};return function(S){return S.position=new o(w),c(),S}}function o(w){this.start=w,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(w){var S=new Error(t.source+":"+n+":"+r+": "+w);if(S.reason=w,S.filename=t.source,S.line=n,S.column=r,S.source=e,!t.silent)throw S}function s(w){var S=w.exec(e);if(S){var I=S[0];return i(I),e=e.slice(I.length),S}}function c(){s(Mv)}function d(w){var S;for(w=w||[];S=f();)S!==!1&&w.push(S);return w}function f(){var w=l();if(!(dc!=e.charAt(0)||fc!=e.charAt(1))){for(var S=2;xn!=e.charAt(S)&&(fc!=e.charAt(S)||dc!=e.charAt(S+1));)++S;if(S+=2,xn===e.charAt(S-1))return a("End of comment missing");var I=e.slice(2,S-2);return r+=2,i(I),e=e.slice(S),r+=2,w({type:$v,comment:I})}}function m(){var w=l(),S=s(Av);if(S){if(f(),!s(Dv))return a("property missing ':'");var I=s(Rv),h=w({type:Uv,property:pc(S[0].replace(cc,xn)),value:I?pc(I[0].replace(cc,xn)):xn});return s(Fv),h}}function p(){var w=[];d(w);for(var S;S=m();)S!==!1&&(w.push(S),d(w));return w}return c(),p()}function pc(e){return e?e.replace(Ov,xn):xn}var Hv=Vv,Wv=Xi&&Xi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(Ls,"__esModule",{value:!0});Ls.default=qv;const Qv=Wv(Hv);function qv(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Qv.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Ol={};Object.defineProperty(Ol,"__esModule",{value:!0});Ol.camelCase=void 0;var Kv=/^--[a-zA-Z0-9_-]+$/,Yv=/-([a-z])/g,Xv=/^[^-]+$/,Gv=/^-(webkit|moz|ms|o|khtml)-/,Jv=/^-(ms)-/,Zv=function(e){return!e||Xv.test(e)||Kv.test(e)},ey=function(e,t){return t.toUpperCase()},hc=function(e,t){return"".concat(t,"-")},ty=function(e,t){return t===void 0&&(t={}),Zv(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(Jv,hc):e=e.replace(Gv,hc),e.replace(Yv,ey))};Ol.camelCase=ty;var ny=Xi&&Xi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},ry=ny(Ls),iy=Ol;function Ca(e,t){var n={};return!e||typeof e!="string"||(0,ry.default)(e,function(r,i){r&&i&&(n[(0,iy.camelCase)(r,t)]=i)}),n}Ca.default=Ca;var ly=Ca;const oy=Ma(ly),yp=xp("end"),Ps=xp("start");function xp(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function ay(e){const t=Ps(e),n=yp(e);if(t&&n)return{start:t,end:n}}function Br(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?mc(e.position):"start"in e||"end"in e?mc(e):"line"in e||"column"in e?Ea(e):""}function Ea(e){return gc(e&&e.line)+":"+gc(e&&e.column)}function mc(e){return Ea(e&&e.start)+"-"+Ea(e&&e.end)}function gc(e){return e&&typeof e=="number"?e:1}class Me extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Br(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}Me.prototype.file="";Me.prototype.name="";Me.prototype.reason="";Me.prototype.message="";Me.prototype.stack="";Me.prototype.column=void 0;Me.prototype.line=void 0;Me.prototype.ancestors=void 0;Me.prototype.cause=void 0;Me.prototype.fatal=void 0;Me.prototype.place=void 0;Me.prototype.ruleId=void 0;Me.prototype.source=void 0;const Is={}.hasOwnProperty,sy=new Map,uy=/[A-Z]/g,cy=new Set(["table","tbody","thead","tfoot","tr"]),dy=new Set(["td","th"]),kp="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function fy(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=ky(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=xy(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?zs:Lv,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=wp(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function wp(e,t,n){if(t.type==="element")return py(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return hy(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return gy(e,t,n);if(t.type==="mdxjsEsm")return my(e,t);if(t.type==="root")return vy(e,t,n);if(t.type==="text")return yy(e,t)}function py(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=zs,e.schema=i),e.ancestors.push(t);const l=bp(e,t.tagName,!1),o=wy(e,t);let a=As(e,t);return cy.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!wv(s):!0})),Sp(e,o,l,t),Ms(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function hy(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}oi(e,t.position)}function my(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);oi(e,t.position)}function gy(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=zs,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:bp(e,t.name,!0),o=Sy(e,t),a=As(e,t);return Sp(e,o,l,t),Ms(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function vy(e,t,n){const r={};return Ms(r,As(e,t)),e.create(t,e.Fragment,r,n)}function yy(e,t){return t.value}function Sp(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Ms(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function xy(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function ky(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=Ps(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function wy(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Is.call(t.properties,i)){const l=by(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&dy.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function Sy(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else oi(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else oi(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function As(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:sy;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const c=i.get(s)||0;o=s+"-"+c,i.set(s,c+1)}}const a=wp(e,l,o);a!==void 0&&n.push(a)}return n}function by(e,t,n){const r=_v(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?gv(n):Pv(n)),r.property==="style"){let i=typeof n=="object"?n:jy(e,String(n));return e.stylePropertyNameCase==="css"&&(i=Cy(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?Cv[r.property]||r.property:r.attribute,n]}}function jy(e,t){try{return oy(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new Me("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=kp+"#cannot-parse-style-attribute",i}}function bp(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=oc(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=oc(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Is.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);oi(e)}function oi(e,t){const n=new Me("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=kp+"#cannot-handle-mdx-estrees-without-createevaluater",n}function Cy(e){const t={};let n;for(n in e)Is.call(e,n)&&(t[Ey(n)]=e[n]);return t}function Ey(e){let t=e.replace(uy,Ny);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function Ny(e){return"-"+e.toLowerCase()}const yo={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},_y={};function Ty(e,t){const n=_y,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return jp(e,r,i)}function jp(e,t,n){if(zy(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return vc(e.children,t,n)}return Array.isArray(e)?vc(e,t,n):""}function vc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=jp(e[i],t,n);return r.join("")}function zy(e){return!!(e&&typeof e=="object")}const yc=document.createElement("i");function Ds(e){const t="&"+e+";";yc.innerHTML=t;const n=yc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function Tt(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function ut(e,t){return e.length>0?(Tt(e,e.length,0,t),e):t}const xc={}.hasOwnProperty;function Ly(e){const t={};let n=-1;for(;++n<e.length;)Py(t,e[n]);return t}function Py(e,t){let n;for(n in t){const i=(xc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){xc.call(i,o)||(i[o]=[]);const a=l[o];Iy(i[o],Array.isArray(a)?a:a?[a]:[])}}}function Iy(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);Tt(e,0,0,r)}function Cp(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function nr(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const Et=mn(/[A-Za-z]/),tt=mn(/[\dA-Za-z]/),My=mn(/[#-'*+\--9=?A-Z^-~]/);function Na(e){return e!==null&&(e<32||e===127)}const _a=mn(/\d/),Ay=mn(/[\dA-Fa-f]/),Dy=mn(/[!-/:-@[-`{-~]/);function V(e){return e!==null&&e<-2}function Ke(e){return e!==null&&(e<0||e===32)}function ee(e){return e===-2||e===-1||e===32}const Ry=mn(new RegExp("\\p{P}|\\p{S}","u")),Fy=mn(/\s/);function mn(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function hr(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&tt(e.charCodeAt(n+1))&&tt(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function se(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return ee(s)?(e.enter(n),a(s)):t(s)}function a(s){return ee(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const Oy={tokenize:By};function By(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),se(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return V(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const $y={tokenize:Uy},kc={tokenize:Vy};function Uy(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const b=n[r];return t.containerState=b[1],e.attempt(b[0].continuation,s,c)(y)}return c(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&v();const b=t.events.length;let E=b,k;for(;E--;)if(t.events[E][0]==="exit"&&t.events[E][1].type==="chunkFlow"){k=t.events[E][1].end;break}h(r);let C=b;for(;C<t.events.length;)t.events[C][1].end={...k},C++;return Tt(t.events,E+1,0,t.events.slice(b)),t.events.length=C,c(y)}return a(y)}function c(y){if(r===n.length){if(!i)return m(y);if(i.currentConstruct&&i.currentConstruct.concrete)return w(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(kc,d,f)(y)}function d(y){return i&&v(),h(r),m(y)}function f(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,w(y)}function m(y){return t.containerState={},e.attempt(kc,p,w)(y)}function p(y){return r++,n.push([t.currentConstruct,t.containerState]),m(y)}function w(y){if(y===null){i&&v(),h(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),S(y)}function S(y){if(y===null){I(e.exit("chunkFlow"),!0),h(0),e.consume(y);return}return V(y)?(e.consume(y),I(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),S)}function I(y,b){const E=t.sliceStream(y);if(b&&E.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(E),t.parser.lazy[y.start.line]){let k=i.events.length;for(;k--;)if(i.events[k][1].start.offset<o&&(!i.events[k][1].end||i.events[k][1].end.offset>o))return;const C=t.events.length;let _=C,R,P;for(;_--;)if(t.events[_][0]==="exit"&&t.events[_][1].type==="chunkFlow"){if(R){P=t.events[_][1].end;break}R=!0}for(h(r),k=C;k<t.events.length;)t.events[k][1].end={...P},k++;Tt(t.events,_+1,0,t.events.slice(C)),t.events.length=k}}function h(y){let b=n.length;for(;b-- >y;){const E=n[b];t.containerState=E[1],E[0].exit.call(t,e)}n.length=y}function v(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Vy(e,t,n){return se(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function wc(e){if(e===null||Ke(e)||Fy(e))return 1;if(Ry(e))return 2}function Rs(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const Ta={name:"attention",resolveAll:Hy,tokenize:Wy};function Hy(e,t){let n=-1,r,i,l,o,a,s,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const f={...e[r][1].end},m={...e[n][1].start};Sc(f,-s),Sc(m,s),o={type:s>1?"strongSequence":"emphasisSequence",start:f,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:m},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=ut(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=ut(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=ut(c,Rs(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=ut(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=ut(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,Tt(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Wy(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=wc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const c=e.exit("attentionSequence"),d=wc(s),f=!d||d===2&&i||n.includes(s),m=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?f:f&&(i||!m)),c._close=!!(l===42?m:m&&(d||!f)),t(s)}}function Sc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Qy={name:"autolink",tokenize:qy};function qy(e,t,n){let r=0;return i;function i(p){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(p){return Et(p)?(e.consume(p),o):p===64?n(p):c(p)}function o(p){return p===43||p===45||p===46||tt(p)?(r=1,a(p)):c(p)}function a(p){return p===58?(e.consume(p),r=0,s):(p===43||p===45||p===46||tt(p))&&r++<32?(e.consume(p),a):(r=0,c(p))}function s(p){return p===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):p===null||p===32||p===60||Na(p)?n(p):(e.consume(p),s)}function c(p){return p===64?(e.consume(p),d):My(p)?(e.consume(p),c):n(p)}function d(p){return tt(p)?f(p):n(p)}function f(p){return p===46?(e.consume(p),r=0,d):p===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):m(p)}function m(p){if((p===45||tt(p))&&r++<63){const w=p===45?m:f;return e.consume(p),w}return n(p)}}const Bl={partial:!0,tokenize:Ky};function Ky(e,t,n){return r;function r(l){return ee(l)?se(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||V(l)?t(l):n(l)}}const Ep={continuation:{tokenize:Xy},exit:Gy,name:"blockQuote",tokenize:Yy};function Yy(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return ee(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function Xy(e,t,n){const r=this;return i;function i(o){return ee(o)?se(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(Ep,t,n)(o)}}function Gy(e){e.exit("blockQuote")}const Np={name:"characterEscape",tokenize:Jy};function Jy(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return Dy(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const _p={name:"characterReference",tokenize:Zy};function Zy(e,t,n){const r=this;let i=0,l,o;return a;function a(f){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),s}function s(f){return f===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(f),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=tt,d(f))}function c(f){return f===88||f===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(f),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=Ay,d):(e.enter("characterReferenceValue"),l=7,o=_a,d(f))}function d(f){if(f===59&&i){const m=e.exit("characterReferenceValue");return o===tt&&!Ds(r.sliceSerialize(m))?n(f):(e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(f)&&i++<l?(e.consume(f),d):n(f)}}const bc={partial:!0,tokenize:tx},jc={concrete:!0,name:"codeFenced",tokenize:ex};function ex(e,t,n){const r=this,i={partial:!0,tokenize:E};let l=0,o=0,a;return s;function s(k){return c(k)}function c(k){const C=r.events[r.events.length-1];return l=C&&C[1].type==="linePrefix"?C[2].sliceSerialize(C[1],!0).length:0,a=k,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(k)}function d(k){return k===a?(o++,e.consume(k),d):o<3?n(k):(e.exit("codeFencedFenceSequence"),ee(k)?se(e,f,"whitespace")(k):f(k))}function f(k){return k===null||V(k)?(e.exit("codeFencedFence"),r.interrupt?t(k):e.check(bc,S,b)(k)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),m(k))}function m(k){return k===null||V(k)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),f(k)):ee(k)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),se(e,p,"whitespace")(k)):k===96&&k===a?n(k):(e.consume(k),m)}function p(k){return k===null||V(k)?f(k):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),w(k))}function w(k){return k===null||V(k)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),f(k)):k===96&&k===a?n(k):(e.consume(k),w)}function S(k){return e.attempt(i,b,I)(k)}function I(k){return e.enter("lineEnding"),e.consume(k),e.exit("lineEnding"),h}function h(k){return l>0&&ee(k)?se(e,v,"linePrefix",l+1)(k):v(k)}function v(k){return k===null||V(k)?e.check(bc,S,b)(k):(e.enter("codeFlowValue"),y(k))}function y(k){return k===null||V(k)?(e.exit("codeFlowValue"),v(k)):(e.consume(k),y)}function b(k){return e.exit("codeFenced"),t(k)}function E(k,C,_){let R=0;return P;function P(U){return k.enter("lineEnding"),k.consume(U),k.exit("lineEnding"),T}function T(U){return k.enter("codeFencedFence"),ee(U)?se(k,D,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(U):D(U)}function D(U){return U===a?(k.enter("codeFencedFenceSequence"),W(U)):_(U)}function W(U){return U===a?(R++,k.consume(U),W):R>=o?(k.exit("codeFencedFenceSequence"),ee(U)?se(k,X,"whitespace")(U):X(U)):_(U)}function X(U){return U===null||V(U)?(k.exit("codeFencedFence"),C(U)):_(U)}}}function tx(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const xo={name:"codeIndented",tokenize:rx},nx={partial:!0,tokenize:ix};function rx(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),se(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?s(c):V(c)?e.attempt(nx,o,s)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||V(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function s(c){return e.exit("codeIndented"),t(c)}}function ix(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):V(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):se(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):V(o)?i(o):n(o)}}const lx={name:"codeText",previous:ax,resolve:ox,tokenize:sx};function ox(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function ax(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function sx(e,t,n){let r=0,i,l;return o;function o(f){return e.enter("codeText"),e.enter("codeTextSequence"),a(f)}function a(f){return f===96?(e.consume(f),r++,a):(e.exit("codeTextSequence"),s(f))}function s(f){return f===null?n(f):f===32?(e.enter("space"),e.consume(f),e.exit("space"),s):f===96?(l=e.enter("codeTextSequence"),i=0,d(f)):V(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),s):(e.enter("codeTextData"),c(f))}function c(f){return f===null||f===32||f===96||V(f)?(e.exit("codeTextData"),s(f)):(e.consume(f),c)}function d(f){return f===96?(e.consume(f),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(f)):(l.type="codeTextData",c(f))}}class ux{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&Cr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),Cr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),Cr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);Cr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);Cr(this.left,n.reverse())}}}function Cr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function Tp(e){const t={};let n=-1,r,i,l,o,a,s,c;const d=new ux(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,cx(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return Tt(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function cx(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],c={};let d,f,m=-1,p=n,w=0,S=0;const I=[S];for(;p;){for(;e.get(++i)[1]!==p;);l.push(i),p._tokenizer||(d=r.sliceStream(p),p.next||d.push(null),f&&o.defineSkip(p.start),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),f=p,p=p.next}for(p=n;++m<a.length;)a[m][0]==="exit"&&a[m-1][0]==="enter"&&a[m][1].type===a[m-1][1].type&&a[m][1].start.line!==a[m][1].end.line&&(S=m+1,I.push(S),p._tokenizer=void 0,p.previous=void 0,p=p.next);for(o.events=[],p?(p._tokenizer=void 0,p.previous=void 0):I.pop(),m=I.length;m--;){const h=a.slice(I[m],I[m+1]),v=l.pop();s.push([v,v+h.length-1]),e.splice(v,2,h)}for(s.reverse(),m=-1;++m<s.length;)c[w+s[m][0]]=w+s[m][1],w+=s[m][1]-s[m][0]-1;return c}const dx={resolve:px,tokenize:hx},fx={partial:!0,tokenize:mx};function px(e){return Tp(e),e}function hx(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):V(a)?e.check(fx,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function mx(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),se(e,l,"linePrefix")}function l(o){if(o===null||V(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function zp(e,t,n,r,i,l,o,a,s){const c=s||Number.POSITIVE_INFINITY;let d=0;return f;function f(h){return h===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(h),e.exit(l),m):h===null||h===32||h===41||Na(h)?n(h):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),S(h))}function m(h){return h===62?(e.enter(l),e.consume(h),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),p(h))}function p(h){return h===62?(e.exit("chunkString"),e.exit(a),m(h)):h===null||h===60||V(h)?n(h):(e.consume(h),h===92?w:p)}function w(h){return h===60||h===62||h===92?(e.consume(h),p):p(h)}function S(h){return!d&&(h===null||h===41||Ke(h))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(h)):d<c&&h===40?(e.consume(h),d++,S):h===41?(e.consume(h),d--,S):h===null||h===32||h===40||Na(h)?n(h):(e.consume(h),h===92?I:S)}function I(h){return h===40||h===41||h===92?(e.consume(h),S):S(h)}}function Lp(e,t,n,r,i,l){const o=this;let a=0,s;return c;function c(p){return e.enter(r),e.enter(i),e.consume(p),e.exit(i),e.enter(l),d}function d(p){return a>999||p===null||p===91||p===93&&!s||p===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(p):p===93?(e.exit(l),e.enter(i),e.consume(p),e.exit(i),e.exit(r),t):V(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),f(p))}function f(p){return p===null||p===91||p===93||V(p)||a++>999?(e.exit("chunkString"),d(p)):(e.consume(p),s||(s=!ee(p)),p===92?m:f)}function m(p){return p===91||p===92||p===93?(e.consume(p),a++,f):f(p)}}function Pp(e,t,n,r,i,l){let o;return a;function a(m){return m===34||m===39||m===40?(e.enter(r),e.enter(i),e.consume(m),e.exit(i),o=m===40?41:m,s):n(m)}function s(m){return m===o?(e.enter(i),e.consume(m),e.exit(i),e.exit(r),t):(e.enter(l),c(m))}function c(m){return m===o?(e.exit(l),s(o)):m===null?n(m):V(m)?(e.enter("lineEnding"),e.consume(m),e.exit("lineEnding"),se(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(m))}function d(m){return m===o||m===null||V(m)?(e.exit("chunkString"),c(m)):(e.consume(m),m===92?f:d)}function f(m){return m===o||m===92?(e.consume(m),d):d(m)}}function $r(e,t){let n;return r;function r(i){return V(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):ee(i)?se(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const gx={name:"definition",tokenize:yx},vx={partial:!0,tokenize:xx};function yx(e,t,n){const r=this;let i;return l;function l(p){return e.enter("definition"),o(p)}function o(p){return Lp.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(p)}function a(p){return i=nr(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),p===58?(e.enter("definitionMarker"),e.consume(p),e.exit("definitionMarker"),s):n(p)}function s(p){return Ke(p)?$r(e,c)(p):c(p)}function c(p){return zp(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(p)}function d(p){return e.attempt(vx,f,f)(p)}function f(p){return ee(p)?se(e,m,"whitespace")(p):m(p)}function m(p){return p===null||V(p)?(e.exit("definition"),r.parser.defined.push(i),t(p)):n(p)}}function xx(e,t,n){return r;function r(a){return Ke(a)?$r(e,i)(a):n(a)}function i(a){return Pp(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return ee(a)?se(e,o,"whitespace")(a):o(a)}function o(a){return a===null||V(a)?t(a):n(a)}}const kx={name:"hardBreakEscape",tokenize:wx};function wx(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return V(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const Sx={name:"headingAtx",resolve:bx,tokenize:jx};function bx(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},Tt(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function jx(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||Ke(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||V(d)?(e.exit("atxHeading"),t(d)):ee(d)?se(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||Ke(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const Cx=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],Cc=["pre","script","style","textarea"],Ex={concrete:!0,name:"htmlFlow",resolveTo:Tx,tokenize:zx},Nx={partial:!0,tokenize:Px},_x={partial:!0,tokenize:Lx};function Tx(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function zx(e,t,n){const r=this;let i,l,o,a,s;return c;function c(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),f}function f(x){return x===33?(e.consume(x),m):x===47?(e.consume(x),l=!0,S):x===63?(e.consume(x),i=3,r.interrupt?t:g):Et(x)?(e.consume(x),o=String.fromCharCode(x),I):n(x)}function m(x){return x===45?(e.consume(x),i=2,p):x===91?(e.consume(x),i=5,a=0,w):Et(x)?(e.consume(x),i=4,r.interrupt?t:g):n(x)}function p(x){return x===45?(e.consume(x),r.interrupt?t:g):n(x)}function w(x){const ne="CDATA[";return x===ne.charCodeAt(a++)?(e.consume(x),a===ne.length?r.interrupt?t:D:w):n(x)}function S(x){return Et(x)?(e.consume(x),o=String.fromCharCode(x),I):n(x)}function I(x){if(x===null||x===47||x===62||Ke(x)){const ne=x===47,be=o.toLowerCase();return!ne&&!l&&Cc.includes(be)?(i=1,r.interrupt?t(x):D(x)):Cx.includes(o.toLowerCase())?(i=6,ne?(e.consume(x),h):r.interrupt?t(x):D(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?v(x):y(x))}return x===45||tt(x)?(e.consume(x),o+=String.fromCharCode(x),I):n(x)}function h(x){return x===62?(e.consume(x),r.interrupt?t:D):n(x)}function v(x){return ee(x)?(e.consume(x),v):P(x)}function y(x){return x===47?(e.consume(x),P):x===58||x===95||Et(x)?(e.consume(x),b):ee(x)?(e.consume(x),y):P(x)}function b(x){return x===45||x===46||x===58||x===95||tt(x)?(e.consume(x),b):E(x)}function E(x){return x===61?(e.consume(x),k):ee(x)?(e.consume(x),E):y(x)}function k(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,C):ee(x)?(e.consume(x),k):_(x)}function C(x){return x===s?(e.consume(x),s=null,R):x===null||V(x)?n(x):(e.consume(x),C)}function _(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||Ke(x)?E(x):(e.consume(x),_)}function R(x){return x===47||x===62||ee(x)?y(x):n(x)}function P(x){return x===62?(e.consume(x),T):n(x)}function T(x){return x===null||V(x)?D(x):ee(x)?(e.consume(x),T):n(x)}function D(x){return x===45&&i===2?(e.consume(x),Q):x===60&&i===1?(e.consume(x),ie):x===62&&i===4?(e.consume(x),L):x===63&&i===3?(e.consume(x),g):x===93&&i===5?(e.consume(x),N):V(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(Nx,$,W)(x)):x===null||V(x)?(e.exit("htmlFlowData"),W(x)):(e.consume(x),D)}function W(x){return e.check(_x,X,$)(x)}function X(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),U}function U(x){return x===null||V(x)?W(x):(e.enter("htmlFlowData"),D(x))}function Q(x){return x===45?(e.consume(x),g):D(x)}function ie(x){return x===47?(e.consume(x),o="",j):D(x)}function j(x){if(x===62){const ne=o.toLowerCase();return Cc.includes(ne)?(e.consume(x),L):D(x)}return Et(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),j):D(x)}function N(x){return x===93?(e.consume(x),g):D(x)}function g(x){return x===62?(e.consume(x),L):x===45&&i===2?(e.consume(x),g):D(x)}function L(x){return x===null||V(x)?(e.exit("htmlFlowData"),$(x)):(e.consume(x),L)}function $(x){return e.exit("htmlFlow"),t(x)}}function Lx(e,t,n){const r=this;return i;function i(o){return V(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function Px(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Bl,t,n)}}const Ix={name:"htmlText",tokenize:Mx};function Mx(e,t,n){const r=this;let i,l,o;return a;function a(g){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(g),s}function s(g){return g===33?(e.consume(g),c):g===47?(e.consume(g),E):g===63?(e.consume(g),y):Et(g)?(e.consume(g),_):n(g)}function c(g){return g===45?(e.consume(g),d):g===91?(e.consume(g),l=0,w):Et(g)?(e.consume(g),v):n(g)}function d(g){return g===45?(e.consume(g),p):n(g)}function f(g){return g===null?n(g):g===45?(e.consume(g),m):V(g)?(o=f,ie(g)):(e.consume(g),f)}function m(g){return g===45?(e.consume(g),p):f(g)}function p(g){return g===62?Q(g):g===45?m(g):f(g)}function w(g){const L="CDATA[";return g===L.charCodeAt(l++)?(e.consume(g),l===L.length?S:w):n(g)}function S(g){return g===null?n(g):g===93?(e.consume(g),I):V(g)?(o=S,ie(g)):(e.consume(g),S)}function I(g){return g===93?(e.consume(g),h):S(g)}function h(g){return g===62?Q(g):g===93?(e.consume(g),h):S(g)}function v(g){return g===null||g===62?Q(g):V(g)?(o=v,ie(g)):(e.consume(g),v)}function y(g){return g===null?n(g):g===63?(e.consume(g),b):V(g)?(o=y,ie(g)):(e.consume(g),y)}function b(g){return g===62?Q(g):y(g)}function E(g){return Et(g)?(e.consume(g),k):n(g)}function k(g){return g===45||tt(g)?(e.consume(g),k):C(g)}function C(g){return V(g)?(o=C,ie(g)):ee(g)?(e.consume(g),C):Q(g)}function _(g){return g===45||tt(g)?(e.consume(g),_):g===47||g===62||Ke(g)?R(g):n(g)}function R(g){return g===47?(e.consume(g),Q):g===58||g===95||Et(g)?(e.consume(g),P):V(g)?(o=R,ie(g)):ee(g)?(e.consume(g),R):Q(g)}function P(g){return g===45||g===46||g===58||g===95||tt(g)?(e.consume(g),P):T(g)}function T(g){return g===61?(e.consume(g),D):V(g)?(o=T,ie(g)):ee(g)?(e.consume(g),T):R(g)}function D(g){return g===null||g===60||g===61||g===62||g===96?n(g):g===34||g===39?(e.consume(g),i=g,W):V(g)?(o=D,ie(g)):ee(g)?(e.consume(g),D):(e.consume(g),X)}function W(g){return g===i?(e.consume(g),i=void 0,U):g===null?n(g):V(g)?(o=W,ie(g)):(e.consume(g),W)}function X(g){return g===null||g===34||g===39||g===60||g===61||g===96?n(g):g===47||g===62||Ke(g)?R(g):(e.consume(g),X)}function U(g){return g===47||g===62||Ke(g)?R(g):n(g)}function Q(g){return g===62?(e.consume(g),e.exit("htmlTextData"),e.exit("htmlText"),t):n(g)}function ie(g){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),j}function j(g){return ee(g)?se(e,N,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(g):N(g)}function N(g){return e.enter("htmlTextData"),o(g)}}const Fs={name:"labelEnd",resolveAll:Fx,resolveTo:Ox,tokenize:Bx},Ax={tokenize:$x},Dx={tokenize:Ux},Rx={tokenize:Vx};function Fx(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&Tt(e,0,e.length,n),e}function Ox(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",c,t]],a=ut(a,e.slice(l+1,l+r+3)),a=ut(a,[["enter",d,t]]),a=ut(a,Rs(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=ut(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=ut(a,e.slice(o+1)),a=ut(a,[["exit",s,t]]),Tt(e,l,e.length,a),e}function Bx(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(m){return l?l._inactive?f(m):(o=r.parser.defined.includes(nr(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(m),e.exit("labelMarker"),e.exit("labelEnd"),s):n(m)}function s(m){return m===40?e.attempt(Ax,d,o?d:f)(m):m===91?e.attempt(Dx,d,o?c:f)(m):o?d(m):f(m)}function c(m){return e.attempt(Rx,d,f)(m)}function d(m){return t(m)}function f(m){return l._balanced=!0,n(m)}}function $x(e,t,n){return r;function r(f){return e.enter("resource"),e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),i}function i(f){return Ke(f)?$r(e,l)(f):l(f)}function l(f){return f===41?d(f):zp(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(f)}function o(f){return Ke(f)?$r(e,s)(f):d(f)}function a(f){return n(f)}function s(f){return f===34||f===39||f===40?Pp(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(f):d(f)}function c(f){return Ke(f)?$r(e,d)(f):d(f)}function d(f){return f===41?(e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),e.exit("resource"),t):n(f)}}function Ux(e,t,n){const r=this;return i;function i(a){return Lp.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(nr(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Vx(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const Hx={name:"labelStartImage",resolveAll:Fs.resolveAll,tokenize:Wx};function Wx(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Qx={name:"labelStartLink",resolveAll:Fs.resolveAll,tokenize:qx};function qx(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const ko={name:"lineEnding",tokenize:Kx};function Kx(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),se(e,t,"linePrefix")}}const Ki={name:"thematicBreak",tokenize:Yx};function Yx(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),s(c)):r>=3&&(c===null||V(c))?(e.exit("thematicBreak"),t(c)):n(c)}function s(c){return c===i?(e.consume(c),r++,s):(e.exit("thematicBreakSequence"),ee(c)?se(e,a,"whitespace")(c):a(c))}}const Ue={continuation:{tokenize:Zx},exit:t1,name:"list",tokenize:Jx},Xx={partial:!0,tokenize:n1},Gx={partial:!0,tokenize:e1};function Jx(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(p){const w=r.containerState.type||(p===42||p===43||p===45?"listUnordered":"listOrdered");if(w==="listUnordered"?!r.containerState.marker||p===r.containerState.marker:_a(p)){if(r.containerState.type||(r.containerState.type=w,e.enter(w,{_container:!0})),w==="listUnordered")return e.enter("listItemPrefix"),p===42||p===45?e.check(Ki,n,c)(p):c(p);if(!r.interrupt||p===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(p)}return n(p)}function s(p){return _a(p)&&++o<10?(e.consume(p),s):(!r.interrupt||o<2)&&(r.containerState.marker?p===r.containerState.marker:p===41||p===46)?(e.exit("listItemValue"),c(p)):n(p)}function c(p){return e.enter("listItemMarker"),e.consume(p),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||p,e.check(Bl,r.interrupt?n:d,e.attempt(Xx,m,f))}function d(p){return r.containerState.initialBlankLine=!0,l++,m(p)}function f(p){return ee(p)?(e.enter("listItemPrefixWhitespace"),e.consume(p),e.exit("listItemPrefixWhitespace"),m):n(p)}function m(p){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(p)}}function Zx(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Bl,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,se(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!ee(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Gx,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,se(e,e.attempt(Ue,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function e1(e,t,n){const r=this;return se(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function t1(e){e.exit(this.containerState.type)}function n1(e,t,n){const r=this;return se(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!ee(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const Ec={name:"setextUnderline",resolveTo:r1,tokenize:i1};function r1(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function i1(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,f;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){f=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||f)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),ee(c)?se(e,s,"lineSuffix")(c):s(c))}function s(c){return c===null||V(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const l1={tokenize:o1};function o1(e){const t=this,n=e.attempt(Bl,r,e.attempt(this.parser.constructs.flowInitial,i,se(e,e.attempt(this.parser.constructs.flow,i,e.attempt(dx,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const a1={resolveAll:Mp()},s1=Ip("string"),u1=Ip("text");function Ip(e){return{resolveAll:Mp(e==="text"?c1:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function c(d){if(d===null)return!0;const f=i[d];let m=-1;if(f)for(;++m<f.length;){const p=f[m];if(!p.previous||p.previous.call(r,r.previous))return!0}return!1}}}function Mp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function c1(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)s=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const d1={42:Ue,43:Ue,45:Ue,48:Ue,49:Ue,50:Ue,51:Ue,52:Ue,53:Ue,54:Ue,55:Ue,56:Ue,57:Ue,62:Ep},f1={91:gx},p1={[-2]:xo,[-1]:xo,32:xo},h1={35:Sx,42:Ki,45:[Ec,Ki],60:Ex,61:Ec,95:Ki,96:jc,126:jc},m1={38:_p,92:Np},g1={[-5]:ko,[-4]:ko,[-3]:ko,33:Hx,38:_p,42:Ta,60:[Qy,Ix],91:Qx,92:[kx,Np],93:Fs,95:Ta,96:lx},v1={null:[Ta,a1]},y1={null:[42,95]},x1={null:[]},k1=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:y1,contentInitial:f1,disable:x1,document:d1,flow:h1,flowInitial:p1,insideSpan:v1,string:m1,text:g1},Symbol.toStringTag,{value:"Module"}));function w1(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:C(E),check:C(k),consume:v,enter:y,exit:b,interrupt:C(k,{interrupt:!0})},c={code:null,containerState:{},defineSkip:S,events:[],now:w,parser:e,previous:null,sliceSerialize:m,sliceStream:p,write:f};let d=t.tokenize.call(c,s);return t.resolveAll&&l.push(t),c;function f(T){return o=ut(o,T),I(),o[o.length-1]!==null?[]:(_(t,0),c.events=Rs(l,c.events,c),c.events)}function m(T,D){return b1(p(T),D)}function p(T){return S1(o,T)}function w(){const{_bufferIndex:T,_index:D,line:W,column:X,offset:U}=r;return{_bufferIndex:T,_index:D,line:W,column:X,offset:U}}function S(T){i[T.line]=T.column,P()}function I(){let T;for(;r._index<o.length;){const D=o[r._index];if(typeof D=="string")for(T=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===T&&r._bufferIndex<D.length;)h(D.charCodeAt(r._bufferIndex));else h(D)}}function h(T){d=d(T)}function v(T){V(T)?(r.line++,r.column=1,r.offset+=T===-3?2:1,P()):T!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=T}function y(T,D){const W=D||{};return W.type=T,W.start=w(),c.events.push(["enter",W,c]),a.push(W),W}function b(T){const D=a.pop();return D.end=w(),c.events.push(["exit",D,c]),D}function E(T,D){_(T,D.from)}function k(T,D){D.restore()}function C(T,D){return W;function W(X,U,Q){let ie,j,N,g;return Array.isArray(X)?$(X):"tokenize"in X?$([X]):L(X);function L(te){return Ae;function Ae(ot){const J=ot!==null&&te[ot],je=ot!==null&&te.null,$e=[...Array.isArray(J)?J:J?[J]:[],...Array.isArray(je)?je:je?[je]:[]];return $($e)(ot)}}function $(te){return ie=te,j=0,te.length===0?Q:x(te[j])}function x(te){return Ae;function Ae(ot){return g=R(),N=te,te.partial||(c.currentConstruct=te),te.name&&c.parser.constructs.disable.null.includes(te.name)?be():te.tokenize.call(D?Object.assign(Object.create(c),D):c,s,ne,be)(ot)}}function ne(te){return T(N,g),U}function be(te){return g.restore(),++j<ie.length?x(ie[j]):Q}}}function _(T,D){T.resolveAll&&!l.includes(T)&&l.push(T),T.resolve&&Tt(c.events,D,c.events.length-D,T.resolve(c.events.slice(D),c)),T.resolveTo&&(c.events=T.resolveTo(c.events,c))}function R(){const T=w(),D=c.previous,W=c.currentConstruct,X=c.events.length,U=Array.from(a);return{from:X,restore:Q};function Q(){r=T,c.previous=D,c.currentConstruct=W,c.events.length=X,a=U,P()}}function P(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function S1(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function b1(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function j1(e){const r={constructs:Ly([k1,...(e||{}).extensions||[]]),content:i(Oy),defined:[],document:i($y),flow:i(l1),lazy:{},string:i(s1),text:i(u1)};return r;function i(l){return o;function o(a){return w1(r,l,a)}}}function C1(e){for(;!Tp(e););return e}const Nc=/[\0\t\n\r]/g;function E1(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let c,d,f,m,p;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),f=0,t="",n&&(l.charCodeAt(0)===65279&&f++,n=void 0);f<l.length;){if(Nc.lastIndex=f,c=Nc.exec(l),m=c&&c.index!==void 0?c.index:l.length,p=l.charCodeAt(m),!c){t=l.slice(f);break}if(p===10&&f===m&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),f<m&&(s.push(l.slice(f,m)),e+=m-f),p){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}f=m+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const N1=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function _1(e){return e.replace(N1,T1)}function T1(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return Cp(n.slice(l?2:1),l?16:10)}return Ds(n)||e}const Ap={}.hasOwnProperty;function z1(e,t,n){return typeof t!="string"&&(n=t,t=void 0),L1(n)(C1(j1(n).document().write(E1()(e,t,!0))))}function L1(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(qs),autolinkProtocol:R,autolinkEmail:R,atxHeading:l(Hs),blockQuote:l(je),characterEscape:R,characterReference:R,codeFenced:l($e),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l($e,o),codeText:l(Ht,o),codeTextData:R,data:R,codeFlowValue:R,definition:l(Wt),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Yp),hardBreakEscape:l(Ws),hardBreakTrailing:l(Ws),htmlFlow:l(Qs,o),htmlFlowData:R,htmlText:l(Qs,o),htmlTextData:R,image:l(Xp),label:o,link:l(qs),listItem:l(Gp),listItemValue:m,listOrdered:l(Ks,f),listUnordered:l(Ks),paragraph:l(Jp),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Hs),strong:l(Zp),thematicBreak:l(th)},exit:{atxHeading:s(),atxHeadingSequence:E,autolink:s(),autolinkEmail:J,autolinkProtocol:ot,blockQuote:s(),characterEscapeValue:P,characterReferenceMarkerHexadecimal:be,characterReferenceMarkerNumeric:be,characterReferenceValue:te,characterReference:Ae,codeFenced:s(I),codeFencedFence:S,codeFencedFenceInfo:p,codeFencedFenceMeta:w,codeFlowValue:P,codeIndented:s(h),codeText:s(U),codeTextData:P,data:P,definition:s(),definitionDestinationString:b,definitionLabelString:v,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(D),hardBreakTrailing:s(D),htmlFlow:s(W),htmlFlowData:P,htmlText:s(X),htmlTextData:P,image:s(ie),label:N,labelText:j,lineEnding:T,link:s(Q),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:ne,resourceDestinationString:g,resourceTitleString:L,resource:$,setextHeading:s(_),setextHeadingLineSequence:C,setextHeadingText:k,strong:s(),thematicBreak:s()}};Dp(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(z){let F={type:"root",children:[]};const H={stack:[F],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},G=[];let le=-1;for(;++le<z.length;)if(z[le][1].type==="listOrdered"||z[le][1].type==="listUnordered")if(z[le][0]==="enter")G.push(le);else{const ht=G.pop();le=i(z,ht,le)}for(le=-1;++le<z.length;){const ht=t[z[le][0]];Ap.call(ht,z[le][1].type)&&ht[z[le][1].type].call(Object.assign({sliceSerialize:z[le][2].sliceSerialize},H),z[le][1])}if(H.tokenStack.length>0){const ht=H.tokenStack[H.tokenStack.length-1];(ht[1]||_c).call(H,void 0,ht[0])}for(F.position={start:qt(z.length>0?z[0][1].start:{line:1,column:1,offset:0}),end:qt(z.length>0?z[z.length-2][1].end:{line:1,column:1,offset:0})},le=-1;++le<t.transforms.length;)F=t.transforms[le](F)||F;return F}function i(z,F,H){let G=F-1,le=-1,ht=!1,gn,zt,mr,gr;for(;++G<=H;){const Xe=z[G];switch(Xe[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Xe[0]==="enter"?le++:le--,gr=void 0;break}case"lineEndingBlank":{Xe[0]==="enter"&&(gn&&!gr&&!le&&!mr&&(mr=G),gr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:gr=void 0}if(!le&&Xe[0]==="enter"&&Xe[1].type==="listItemPrefix"||le===-1&&Xe[0]==="exit"&&(Xe[1].type==="listUnordered"||Xe[1].type==="listOrdered")){if(gn){let In=G;for(zt=void 0;In--;){const Lt=z[In];if(Lt[1].type==="lineEnding"||Lt[1].type==="lineEndingBlank"){if(Lt[0]==="exit")continue;zt&&(z[zt][1].type="lineEndingBlank",ht=!0),Lt[1].type="lineEnding",zt=In}else if(!(Lt[1].type==="linePrefix"||Lt[1].type==="blockQuotePrefix"||Lt[1].type==="blockQuotePrefixWhitespace"||Lt[1].type==="blockQuoteMarker"||Lt[1].type==="listItemIndent"))break}mr&&(!zt||mr<zt)&&(gn._spread=!0),gn.end=Object.assign({},zt?z[zt][1].start:Xe[1].end),z.splice(zt||G,0,["exit",gn,Xe[2]]),G++,H++}if(Xe[1].type==="listItemPrefix"){const In={type:"listItem",_spread:!1,start:Object.assign({},Xe[1].start),end:void 0};gn=In,z.splice(G,0,["enter",In,Xe[2]]),G++,H++,mr=void 0,gr=!0}}}return z[F][1]._spread=ht,H}function l(z,F){return H;function H(G){a.call(this,z(G),G),F&&F.call(this,G)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(z,F,H){this.stack[this.stack.length-1].children.push(z),this.stack.push(z),this.tokenStack.push([F,H||void 0]),z.position={start:qt(F.start),end:void 0}}function s(z){return F;function F(H){z&&z.call(this,H),c.call(this,H)}}function c(z,F){const H=this.stack.pop(),G=this.tokenStack.pop();if(G)G[0].type!==z.type&&(F?F.call(this,z,G[0]):(G[1]||_c).call(this,z,G[0]));else throw new Error("Cannot close `"+z.type+"` ("+Br({start:z.start,end:z.end})+"): it’s not open");H.position.end=qt(z.end)}function d(){return Ty(this.stack.pop())}function f(){this.data.expectingFirstListItemValue=!0}function m(z){if(this.data.expectingFirstListItemValue){const F=this.stack[this.stack.length-2];F.start=Number.parseInt(this.sliceSerialize(z),10),this.data.expectingFirstListItemValue=void 0}}function p(){const z=this.resume(),F=this.stack[this.stack.length-1];F.lang=z}function w(){const z=this.resume(),F=this.stack[this.stack.length-1];F.meta=z}function S(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function I(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function h(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z.replace(/(\r?\n|\r)$/g,"")}function v(z){const F=this.resume(),H=this.stack[this.stack.length-1];H.label=F,H.identifier=nr(this.sliceSerialize(z)).toLowerCase()}function y(){const z=this.resume(),F=this.stack[this.stack.length-1];F.title=z}function b(){const z=this.resume(),F=this.stack[this.stack.length-1];F.url=z}function E(z){const F=this.stack[this.stack.length-1];if(!F.depth){const H=this.sliceSerialize(z).length;F.depth=H}}function k(){this.data.setextHeadingSlurpLineEnding=!0}function C(z){const F=this.stack[this.stack.length-1];F.depth=this.sliceSerialize(z).codePointAt(0)===61?1:2}function _(){this.data.setextHeadingSlurpLineEnding=void 0}function R(z){const H=this.stack[this.stack.length-1].children;let G=H[H.length-1];(!G||G.type!=="text")&&(G=eh(),G.position={start:qt(z.start),end:void 0},H.push(G)),this.stack.push(G)}function P(z){const F=this.stack.pop();F.value+=this.sliceSerialize(z),F.position.end=qt(z.end)}function T(z){const F=this.stack[this.stack.length-1];if(this.data.atHardBreak){const H=F.children[F.children.length-1];H.position.end=qt(z.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(F.type)&&(R.call(this,z),P.call(this,z))}function D(){this.data.atHardBreak=!0}function W(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function X(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function U(){const z=this.resume(),F=this.stack[this.stack.length-1];F.value=z}function Q(){const z=this.stack[this.stack.length-1];if(this.data.inReference){const F=this.data.referenceType||"shortcut";z.type+="Reference",z.referenceType=F,delete z.url,delete z.title}else delete z.identifier,delete z.label;this.data.referenceType=void 0}function ie(){const z=this.stack[this.stack.length-1];if(this.data.inReference){const F=this.data.referenceType||"shortcut";z.type+="Reference",z.referenceType=F,delete z.url,delete z.title}else delete z.identifier,delete z.label;this.data.referenceType=void 0}function j(z){const F=this.sliceSerialize(z),H=this.stack[this.stack.length-2];H.label=_1(F),H.identifier=nr(F).toLowerCase()}function N(){const z=this.stack[this.stack.length-1],F=this.resume(),H=this.stack[this.stack.length-1];if(this.data.inReference=!0,H.type==="link"){const G=z.children;H.children=G}else H.alt=F}function g(){const z=this.resume(),F=this.stack[this.stack.length-1];F.url=z}function L(){const z=this.resume(),F=this.stack[this.stack.length-1];F.title=z}function $(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function ne(z){const F=this.resume(),H=this.stack[this.stack.length-1];H.label=F,H.identifier=nr(this.sliceSerialize(z)).toLowerCase(),this.data.referenceType="full"}function be(z){this.data.characterReferenceType=z.type}function te(z){const F=this.sliceSerialize(z),H=this.data.characterReferenceType;let G;H?(G=Cp(F,H==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):G=Ds(F);const le=this.stack[this.stack.length-1];le.value+=G}function Ae(z){const F=this.stack.pop();F.position.end=qt(z.end)}function ot(z){P.call(this,z);const F=this.stack[this.stack.length-1];F.url=this.sliceSerialize(z)}function J(z){P.call(this,z);const F=this.stack[this.stack.length-1];F.url="mailto:"+this.sliceSerialize(z)}function je(){return{type:"blockquote",children:[]}}function $e(){return{type:"code",lang:null,meta:null,value:""}}function Ht(){return{type:"inlineCode",value:""}}function Wt(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Yp(){return{type:"emphasis",children:[]}}function Hs(){return{type:"heading",depth:0,children:[]}}function Ws(){return{type:"break"}}function Qs(){return{type:"html",value:""}}function Xp(){return{type:"image",title:null,url:"",alt:null}}function qs(){return{type:"link",title:null,url:"",children:[]}}function Ks(z){return{type:"list",ordered:z.type==="listOrdered",start:null,spread:z._spread,children:[]}}function Gp(z){return{type:"listItem",spread:z._spread,checked:null,children:[]}}function Jp(){return{type:"paragraph",children:[]}}function Zp(){return{type:"strong",children:[]}}function eh(){return{type:"text",value:""}}function th(){return{type:"thematicBreak"}}}function qt(e){return{line:e.line,column:e.column,offset:e.offset}}function Dp(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Dp(e,r):P1(e,r)}}function P1(e,t){let n;for(n in t)if(Ap.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function _c(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Br({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Br({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Br({start:t.start,end:t.end})+") is still open")}function I1(e){const t=this;t.parser=n;function n(r){return z1(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function M1(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function A1(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function D1(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function R1(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function F1(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function O1(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=hr(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const c={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,c),e.applyData(t,c)}function B1(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function $1(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Rp(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function U1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Rp(e,t);const i={src:hr(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function V1(e,t){const n={src:hr(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function H1(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function W1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Rp(e,t);const i={href:hr(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function Q1(e,t){const n={href:hr(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function q1(e,t,n){const r=e.all(t),i=n?K1(n):Fp(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let f;d&&d.type==="element"&&d.tagName==="p"?f=d:(f={type:"element",tagName:"p",properties:{},children:[]},r.unshift(f)),f.children.length>0&&f.children.unshift({type:"text",value:" "}),f.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function K1(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Fp(n[r])}return t}function Fp(e){const t=e.spread;return t??e.children.length>1}function Y1(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function X1(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function G1(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function J1(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function Z1(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=Ps(t.children[1]),s=yp(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function e0(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const c=[];for(;++s<a;){const f=t.children[s],m={},p=o?o[s]:void 0;p&&(m.align=p);let w={type:"element",tagName:l,properties:m,children:[]};f&&(w.children=e.all(f),e.patch(f,w),w=e.applyData(f,w)),c.push(w)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function t0(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const Tc=9,zc=32;function n0(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(Lc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(Lc(t.slice(i),i>0,!1)),l.join("")}function Lc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===Tc||l===zc;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===Tc||l===zc;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function r0(e,t){const n={type:"text",value:n0(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function i0(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const l0={blockquote:M1,break:A1,code:D1,delete:R1,emphasis:F1,footnoteReference:O1,heading:B1,html:$1,imageReference:U1,image:V1,inlineCode:H1,linkReference:W1,link:Q1,listItem:q1,list:Y1,paragraph:X1,root:G1,strong:J1,table:Z1,tableCell:t0,tableRow:e0,text:r0,thematicBreak:i0,toml:Pi,yaml:Pi,definition:Pi,footnoteDefinition:Pi};function Pi(){}const Op=-1,$l=0,Ur=1,wl=2,Os=3,Bs=4,$s=5,Us=6,Bp=7,$p=8,Pc=typeof self=="object"?self:globalThis,o0=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case $l:case Op:return n(o,i);case Ur:{const a=n([],i);for(const s of o)a.push(r(s));return a}case wl:{const a=n({},i);for(const[s,c]of o)a[r(s)]=r(c);return a}case Os:return n(new Date(o),i);case Bs:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case $s:{const a=n(new Map,i);for(const[s,c]of o)a.set(r(s),r(c));return a}case Us:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Bp:{const{name:a,message:s}=o;return n(new Pc[a](s),i)}case $p:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new Pc[l](o),i)};return r},Ic=e=>o0(new Map,e)(0),Dn="",{toString:a0}={},{keys:s0}=Object,Er=e=>{const t=typeof e;if(t!=="object"||!e)return[$l,t];const n=a0.call(e).slice(8,-1);switch(n){case"Array":return[Ur,Dn];case"Object":return[wl,Dn];case"Date":return[Os,Dn];case"RegExp":return[Bs,Dn];case"Map":return[$s,Dn];case"Set":return[Us,Dn];case"DataView":return[Ur,n]}return n.includes("Array")?[Ur,n]:n.includes("Error")?[Bp,n]:[wl,n]},Ii=([e,t])=>e===$l&&(t==="function"||t==="symbol"),u0=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=Er(o);switch(a){case $l:{let d=o;switch(s){case"bigint":a=$p,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([Op],o)}return i([a,d],o)}case Ur:{if(s){let m=o;return s==="DataView"?m=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(m=new Uint8Array(o)),i([s,[...m]],o)}const d=[],f=i([a,d],o);for(const m of o)d.push(l(m));return f}case wl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],f=i([a,d],o);for(const m of s0(o))(e||!Ii(Er(o[m])))&&d.push([l(m),l(o[m])]);return f}case Os:return i([a,o.toISOString()],o);case Bs:{const{source:d,flags:f}=o;return i([a,{source:d,flags:f}],o)}case $s:{const d=[],f=i([a,d],o);for(const[m,p]of o)(e||!(Ii(Er(m))||Ii(Er(p))))&&d.push([l(m),l(p)]);return f}case Us:{const d=[],f=i([a,d],o);for(const m of o)(e||!Ii(Er(m)))&&d.push(l(m));return f}}const{message:c}=o;return i([a,{name:s,message:c}],o)};return l},Mc=(e,{json:t,lossy:n}={})=>{const r=[];return u0(!(t||n),!!t,new Map,r)(e),r},Sl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?Ic(Mc(e,t)):structuredClone(e):(e,t)=>Ic(Mc(e,t));function c0(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function d0(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function f0(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||c0,r=e.options.footnoteBackLabel||d0,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[s]);if(!c)continue;const d=e.all(c),f=String(c.identifier).toUpperCase(),m=hr(f.toLowerCase());let p=0;const w=[],S=e.footnoteCounts.get(f);for(;S!==void 0&&++p<=S;){w.length>0&&w.push({type:"text",value:" "});let v=typeof n=="string"?n:n(s,p);typeof v=="string"&&(v={type:"text",value:v}),w.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+m+(p>1?"-"+p:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,p),className:["data-footnote-backref"]},children:Array.isArray(v)?v:[v]})}const I=d[d.length-1];if(I&&I.type==="element"&&I.tagName==="p"){const v=I.children[I.children.length-1];v&&v.type==="text"?v.value+=" ":I.children.push({type:"text",value:" "}),I.children.push(...w)}else d.push(...w);const h={type:"element",tagName:"li",properties:{id:t+"fn-"+m},children:e.wrap(d,!0)};e.patch(c,h),a.push(h)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...Sl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const Up=function(e){if(e==null)return g0;if(typeof e=="function")return Ul(e);if(typeof e=="object")return Array.isArray(e)?p0(e):h0(e);if(typeof e=="string")return m0(e);throw new Error("Expected function, string, or object as test")};function p0(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=Up(e[n]);return Ul(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function h0(e){const t=e;return Ul(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function m0(e){return Ul(t);function t(n){return n&&n.type===e}}function Ul(e){return t;function t(n,r,i){return!!(v0(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function g0(){return!0}function v0(e){return e!==null&&typeof e=="object"&&"type"in e}const Vp=[],y0=!0,Ac=!1,x0="skip";function k0(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=Up(i),o=r?-1:1;a(e,void 0,[])();function a(s,c,d){const f=s&&typeof s=="object"?s:{};if(typeof f.type=="string"){const p=typeof f.tagName=="string"?f.tagName:typeof f.name=="string"?f.name:void 0;Object.defineProperty(m,"name",{value:"node ("+(s.type+(p?"<"+p+">":""))+")"})}return m;function m(){let p=Vp,w,S,I;if((!t||l(s,c,d[d.length-1]||void 0))&&(p=w0(n(s,d)),p[0]===Ac))return p;if("children"in s&&s.children){const h=s;if(h.children&&p[0]!==x0)for(S=(r?h.children.length:-1)+o,I=d.concat(h);S>-1&&S<h.children.length;){const v=h.children[S];if(w=a(v,S,I)(),w[0]===Ac)return w;S=typeof w[1]=="number"?w[1]:S+o}}return p}}}function w0(e){return Array.isArray(e)?e:typeof e=="number"?[y0,e]:e==null?Vp:[e]}function Hp(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),k0(e,l,a,i);function a(s,c){const d=c[c.length-1],f=d?d.children.indexOf(s):void 0;return o(s,f,d)}}const za={}.hasOwnProperty,S0={};function b0(e,t){const n=t||S0,r=new Map,i=new Map,l=new Map,o={...l0,...n.handlers},a={all:c,applyData:C0,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:j0,wrap:N0};return Hp(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const f=d.type==="definition"?r:i,m=String(d.identifier).toUpperCase();f.has(m)||f.set(m,d)}}),a;function s(d,f){const m=d.type,p=a.handlers[m];if(za.call(a.handlers,m)&&p)return p(a,d,f);if(a.options.passThrough&&a.options.passThrough.includes(m)){if("children"in d){const{children:S,...I}=d,h=Sl(I);return h.children=a.all(d),h}return Sl(d)}return(a.options.unknownHandler||E0)(a,d,f)}function c(d){const f=[];if("children"in d){const m=d.children;let p=-1;for(;++p<m.length;){const w=a.one(m[p],d);if(w){if(p&&m[p-1].type==="break"&&(!Array.isArray(w)&&w.type==="text"&&(w.value=Dc(w.value)),!Array.isArray(w)&&w.type==="element")){const S=w.children[0];S&&S.type==="text"&&(S.value=Dc(S.value))}Array.isArray(w)?f.push(...w):f.push(w)}}}return f}}function j0(e,t){e.position&&(t.position=ay(e))}function C0(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,Sl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function E0(e,t){const n=t.data||{},r="value"in t&&!(za.call(n,"hProperties")||za.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function N0(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Dc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Rc(e,t){const n=b0(e,t),r=n.one(e,void 0),i=f0(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function _0(e,t){return e&&"run"in e?async function(n,r){const i=Rc(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Rc(n,{file:r,...e||t})}}function Fc(e){if(e)throw e}var Yi=Object.prototype.hasOwnProperty,Wp=Object.prototype.toString,Oc=Object.defineProperty,Bc=Object.getOwnPropertyDescriptor,$c=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Wp.call(t)==="[object Array]"},Uc=function(t){if(!t||Wp.call(t)!=="[object Object]")return!1;var n=Yi.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Yi.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Yi.call(t,i)},Vc=function(t,n){Oc&&n.name==="__proto__"?Oc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Hc=function(t,n){if(n==="__proto__")if(Yi.call(t,n)){if(Bc)return Bc(t,n).value}else return;return t[n]},T0=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<c;++s)if(t=arguments[s],t!=null)for(n in t)r=Hc(a,n),i=Hc(t,n),a!==i&&(d&&i&&(Uc(i)||(l=$c(i)))?(l?(l=!1,o=r&&$c(r)?r:[]):o=r&&Uc(r)?r:{},Vc(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&Vc(a,{name:n,newValue:i}));return a};const wo=Ma(T0);function La(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function z0(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...c){const d=e[++l];let f=-1;if(s){o(s);return}for(;++f<i.length;)(c[f]===null||c[f]===void 0)&&(c[f]=i[f]);i=c,d?L0(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function L0(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const jt={basename:P0,dirname:I0,extname:M0,join:A0,sep:"/"};function P0(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');pi(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function I0(e){if(pi(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function M0(e){pi(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function A0(...e){let t=-1,n;for(;++t<e.length;)pi(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":D0(n)}function D0(e){pi(e);const t=e.codePointAt(0)===47;let n=R0(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function R0(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function pi(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const F0={cwd:O0};function O0(){return"/"}function Pa(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function B0(e){if(typeof e=="string")e=new URL(e);else if(!Pa(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return $0(e)}function $0(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const So=["history","path","basename","stem","extname","dirname"];class Qp{constructor(t){let n;t?Pa(t)?n={path:t}:typeof t=="string"||U0(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":F0.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<So.length;){const l=So[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)So.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?jt.basename(this.path):void 0}set basename(t){jo(t,"basename"),bo(t,"basename"),this.path=jt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?jt.dirname(this.path):void 0}set dirname(t){Wc(this.basename,"dirname"),this.path=jt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?jt.extname(this.path):void 0}set extname(t){if(bo(t,"extname"),Wc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=jt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){Pa(t)&&(t=B0(t)),jo(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?jt.basename(this.path,this.extname):void 0}set stem(t){jo(t,"stem"),bo(t,"stem"),this.path=jt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new Me(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function bo(e,t){if(e&&e.includes(jt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+jt.sep+"`")}function jo(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Wc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function U0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const V0=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},H0={}.hasOwnProperty;class Vs extends V0{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=z0()}copy(){const t=new Vs;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(wo(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(No("data",this.frozen),this.namespace[t]=n,this):H0.call(this.namespace,t)&&this.namespace[t]||void 0:t?(No("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Mi(t),r=this.parser||this.Parser;return Co("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),Co("process",this.parser||this.Parser),Eo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Mi(t),s=r.parse(a);r.run(s,a,function(d,f,m){if(d||!f||!m)return c(d);const p=f,w=r.stringify(p,m);q0(w)?m.value=w:m.result=w,c(d,m)});function c(d,f){d||!f?o(d):l?l(f):n(void 0,f)}}}processSync(t){let n=!1,r;return this.freeze(),Co("processSync",this.parser||this.Parser),Eo("processSync",this.compiler||this.Compiler),this.process(t,i),qc("processSync","process",n),r;function i(l,o){n=!0,Fc(l),r=o}}run(t,n,r){Qc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Mi(n);i.run(t,s,c);function c(d,f,m){const p=f||t;d?a(d):o?o(p):r(void 0,p,m)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),qc("runSync","run",r),i;function l(o,a){Fc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Mi(n),i=this.compiler||this.Compiler;return Eo("stringify",i),Qc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(No("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")s(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...f]=c;s(d,f)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=wo(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const f=c[d];l(f)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function s(c,d){let f=-1,m=-1;for(;++f<r.length;)if(r[f][0]===c){m=f;break}if(m===-1)r.push([c,...d]);else if(d.length>0){let[p,...w]=d;const S=r[m][1];La(S)&&La(p)&&(p=wo(!0,S,p)),r[m]=[c,p,...w]}}}}const W0=new Vs().freeze();function Co(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function Eo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function No(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Qc(e){if(!La(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function qc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Mi(e){return Q0(e)?e:new Qp(e)}function Q0(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function q0(e){return typeof e=="string"||K0(e)}function K0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const Y0="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Kc=[],Yc={allowDangerousHtml:!0},X0=/^(https?|ircs?|mailto|xmpp)$/i,G0=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function J0(e){const t=Z0(e),n=ek(e);return tk(t.runSync(t.parse(n),n),e)}function Z0(e){const t=e.rehypePlugins||Kc,n=e.remarkPlugins||Kc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...Yc}:Yc;return W0().use(I1).use(n).use(_0,r).use(t)}function ek(e){const t=e.children||"",n=new Qp;return typeof t=="string"&&(n.value=t),n}function tk(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||nk;for(const d of G0)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+Y0+d.id,void 0);return Hp(e,c),fy(e,{Fragment:u.Fragment,components:i,ignoreInvalidStyle:!0,jsx:u.jsx,jsxs:u.jsxs,passKeys:!0,passNode:!0});function c(d,f,m){if(d.type==="raw"&&m&&typeof f=="number")return o?m.children.splice(f,1):m.children[f]={type:"text",value:d.value},f;if(d.type==="element"){let p;for(p in yo)if(Object.hasOwn(yo,p)&&Object.hasOwn(d.properties,p)){const w=d.properties[p],S=yo[p];(S===null||S.includes(d.tagName))&&(d.properties[p]=s(String(w||""),p,d))}}if(d.type==="element"){let p=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!p&&r&&typeof f=="number"&&(p=!r(d,f,m)),p&&m&&typeof f=="number")return a&&d.children?m.children.splice(f,1,...d.children):m.children.splice(f,1),f}}}function nk(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||X0.test(e.slice(0,t))?e:""}const rk=e=>{if(!e)return null;try{return JSON.parse(e).execution_stats||null}catch{return null}},ik=e=>{if(e.kind!=="status")return!1;const t=e.content.toLowerCase();return t.includes("running")||t.includes("thinking")||t.includes("executing")||t.includes("processing")},Xc=10*1024,_o=200,Le={send:u.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),u.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),u.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),u.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),file:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),spinner:u.jsx("svg",{className:"spinner-icon",width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 12a9 9 0 1 1-6.219-8.56"})})},lk=e=>{switch(e){case"directive":return Le.directive;case"question":return Le.question;case"status":return Le.status;case"result":return Le.result;case"approval_request":return Le.lock;default:return Le.directive}},ok=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=O.useRef(null),[a,s]=Kt.useState(""),[c,d]=Kt.useState("directive"),[f,m]=Kt.useState(""),[p,w]=Kt.useState(!1),[S,I]=Kt.useState(new Map),[h,v]=Kt.useState(new Set),[y,b]=O.useState(new Set),[E,k]=O.useState(new Set),C=j=>{const N=(j.match(/\n/g)||[]).length+1;if(!(j.length>Xc||N>_o))return{needsTruncation:!1,truncated:j,fullLength:j.length,lineCount:N};let L=j.slice(0,Xc);const $=L.split(`
`);$.length>_o&&(L=$.slice(0,_o).join(`
`));const x=L.lastIndexOf(`
`);return x>L.length*.8&&(L=L.slice(0,x)),{needsTruncation:!0,truncated:L,fullLength:j.length,lineCount:N}},_=j=>{b(N=>{const g=new Set(N);return g.has(j)?g.delete(j):g.add(j),g})};O.useEffect(()=>{e!=null&&e.workspace?m(e.workspace):m("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),O.useEffect(()=>{var j;(j=o.current)==null||j.scrollIntoView({behavior:"smooth"})},[t]);const R=j=>{m(j),r&&r(j)},P=()=>{a.trim()&&(n(a,c,f||void 0),s(""))},T=j=>{j.key==="Enter"&&!j.shiftKey&&(j.preventDefault(),P())},D=j=>new Date(j).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),W=j=>j.length>12?`${j.slice(0,8)}...`:j,X=j=>{if(!j.metadata_json)return null;try{return JSON.parse(j.metadata_json).approval_id||null}catch{return null}},U=j=>{const N=S.get(j)||"";i&&(i(j,N),v(g=>new Set(g).add(j)),I(g=>{const L=new Map(g);return L.delete(j),L}))},Q=j=>{const N=S.get(j)||"";if(!N.trim()){alert("Please provide a reason for rejection");return}l&&(l(j,N),v(g=>new Set(g).add(j)),I(g=>{const L=new Map(g);return L.delete(j),L}))},ie=(j,N)=>{I(g=>new Map(g).set(j,N))};return e?u.jsxs("div",{className:"conversation-view",children:[u.jsxs("div",{className:"conversation-header",children:[u.jsxs("div",{className:"header-info",children:[u.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&u.jsxs("span",{className:"thread-agent-badge",children:[Le.bot,e.target_agent]})]}),u.jsxs("div",{className:"header-stats",children:[u.jsxs("span",{className:"message-count",children:[t.length," messages"]}),u.jsx("span",{className:"thread-id",title:e.id,children:W(e.id)})]})]}),u.jsxs("div",{className:"messages-container",children:[t.length===0?u.jsxs("div",{className:"empty-messages",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("p",{children:"No messages yet"}),u.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((j,N)=>{const g=j.from_type==="human",L=N===0||t[N-1].from_type!==j.from_type,$=y.has(j.id),{needsTruncation:x,truncated:ne,fullLength:be,lineCount:te}=C(j.content),Ae=$?j.content:ne,ot=ik(j);return u.jsxs("div",{className:`message ${g?"human":"agent"}${ot?" running-status":""}`,children:[u.jsx("div",{className:`message-avatar ${L?"visible":""}`,children:L&&(g?Le.user:Le.bot)}),u.jsxs("div",{className:"message-body",children:[L&&u.jsxs("div",{className:"message-meta",children:[u.jsx("span",{className:"sender-name",children:j.from_id}),u.jsxs("span",{className:`kind-badge${ot?" running":""}`,children:[ot?Le.spinner:lk(j.kind)," ",j.kind]}),u.jsx("span",{className:"message-time",children:D(j.created_at)})]}),u.jsxs("div",{className:"message-content",children:[j.kind==="result"||!g?u.jsx(J0,{components:{a:({href:J,children:je})=>{let $e=J;return J&&J.startsWith("/")&&!J.startsWith("//")&&($e=`file://${J}`),u.jsx("a",{href:$e,target:"_blank",rel:"noopener noreferrer",children:je})},code:({className:J,children:je,...$e})=>!J?u.jsx("code",{className:"inline-code",...$e,children:je}):u.jsx("code",{className:J,...$e,children:je})},children:Ae}):Ae,x&&u.jsx("div",{className:"truncation-notice",children:u.jsx("button",{className:"expand-btn",onClick:()=>_(j.id),children:$?u.jsx(u.Fragment,{children:"Show less"}):u.jsxs(u.Fragment,{children:["Show more (",Math.round(be/1024),"KB, ",te," lines)"]})})}),j.kind==="approval_request"&&(()=>{const J=X(j),je=J&&h.has(J);return J?u.jsx("div",{className:"inline-approval",children:je?u.jsxs("div",{className:"approval-handled",children:[Le.check,u.jsx("span",{children:"Action taken"})]}):u.jsxs(u.Fragment,{children:[u.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:S.get(J)||"",onChange:$e=>ie(J,$e.target.value)}),u.jsxs("div",{className:"approval-actions",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>Q(J),title:"Reject",children:[Le.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>U(J),title:"Approve",children:[Le.check,"Approve"]})]})]})}):null})(),j.kind==="result"&&(()=>{const J=rk(j.metadata_json);if(!J||!J.files_created||J.files_created.length===0)return null;const je=E.has(j.id),$e=()=>{k(Ht=>{const Wt=new Set(Ht);return Wt.has(j.id)?Wt.delete(j.id):Wt.add(j.id),Wt})};return u.jsxs("div",{className:"files-created-section",children:[u.jsxs("button",{className:`files-toggle-btn ${je?"expanded":""}`,onClick:$e,children:[Le.file,u.jsxs("span",{children:["Files Created (",J.files_created.length,")"]}),J.workspace&&u.jsxs("span",{className:"workspace-badge",title:J.workspace,children:[Le.folder,J.workspace.split("/").pop()]}),u.jsx("span",{className:"toggle-chevron",children:je?"▼":"▶"})]}),je&&u.jsx("ul",{className:"files-list",children:J.files_created.map((Ht,Wt)=>u.jsx("li",{className:"file-item",children:u.jsx("a",{href:`file://${J.workspace?J.workspace+"/":""}${Ht}`,target:"_blank",rel:"noopener noreferrer",title:Ht,children:Ht})},Wt))})]})})()]}),u.jsxs("div",{className:"message-footer",children:[u.jsxs("span",{className:"message-seq",children:["#",j.message_seq]}),j.delivery_state!=="acked"&&u.jsx("span",{className:`delivery-status ${j.delivery_state}`,children:j.delivery_state==="pending"?"sending...":"delivered"})]})]})]},j.id)}),u.jsx("div",{ref:o})]}),u.jsxs("div",{className:"input-area",children:[p&&u.jsxs("div",{className:"workspace-input-row",children:[u.jsx("input",{type:"text",value:f,onChange:j=>R(j.target.value),onBlur:()=>{r&&r(f)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),u.jsx("button",{onClick:async()=>{try{const N=await(await fetch("/api/select-folder")).json();!N.cancelled&&N.path&&R(N.path)}catch(j){console.error("Failed to open folder picker:",j)}},className:"workspace-browse",title:"Browse for folder",children:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),u.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),u.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),f&&u.jsx("button",{onClick:()=>{R(""),w(!1)},className:"workspace-clear",children:"Clear"})]}),u.jsxs("div",{className:"input-wrapper",children:[u.jsx("button",{onClick:()=>w(!p),className:`workspace-toggle ${f?"has-workspace":""}`,title:f||"Set working directory",children:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),u.jsxs("select",{value:c,onChange:j=>d(j.target.value),className:"kind-selector",children:[u.jsx("option",{value:"directive",children:"Directive"}),u.jsx("option",{value:"question",children:"Question"})]}),u.jsx("textarea",{value:a,onChange:j=>s(j.target.value),onKeyPress:T,placeholder:f?`Message (workspace: ${f.split("/").pop()})`:"Type a message...",rows:1}),u.jsx("button",{onClick:P,className:"send-btn",disabled:!a.trim(),children:Le.send})]}),u.jsxs("div",{className:"input-hint",children:["Press ",u.jsx("kbd",{children:"Enter"})," to send, ",u.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),u.jsx("style",{children:`
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
      `})]}):null};let qp="disconnected",Kp=0;const Ia=new Set;function Ai(e,t){qp=e,Kp=t,Ia.forEach(n=>n(e,t))}function ak(e){return Ia.add(e),e(qp,Kp),()=>Ia.delete(e)}function sk(e,t=1e3,n=3e4){const r=Math.min(t*Math.pow(2,e),n),i=r*Math.random()*.3;return Math.round(r+i)}const uk=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,maxReconnectAttempts:l=10})=>{const o=O.useRef(null),[a,s]=O.useState(!1),[c,d]=O.useState(null),[f,m]=O.useState(0),p=O.useRef(null),w=O.useRef(new Map),S=O.useCallback(()=>{try{const k=`${e}?instance_id=${t}`;o.current=new WebSocket(k),Ai(f>0?"reconnecting":"connecting",f),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),d(null),m(0),Ai("connected",0),w.current.forEach((C,_)=>{v(_,C)})},o.current.onmessage=C=>{try{const _=JSON.parse(C.data);I(_)}catch(_){console.error("Failed to parse WebSocket message:",_)}},o.current.onerror=C=>{console.error("WebSocket error:",C),d("Connection error")},o.current.onclose=()=>{if(console.log("WebSocket disconnected"),s(!1),Ai("disconnected",f),f<l){const C=sk(f);console.log(`WebSocket reconnecting in ${C}ms (attempt ${f+1}/${l})`),p.current=setTimeout(()=>{m(_=>_+1),S()},C)}else console.error("Max reconnection attempts reached"),d("Connection lost. Please refresh the page.")}}catch(k){console.error("Failed to connect to WebSocket:",k),d("Failed to connect"),Ai("disconnected",f)}},[e,t,f,l]),I=O.useCallback(k=>{switch(k.type){case"message":n&&k.data&&n(k.data);break;case"batch":if(r&&k.data){const C=k.data;r(C),n&&C.messages.forEach(_=>n(_))}break;case"error":i&&k.data&&i(k.data),console.error("WebSocket error event:",k.data);break;case"pong":break;default:console.log("Unknown event type:",k.type)}},[n,r,i]),h=O.useCallback(k=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(k)):console.warn("WebSocket not connected, cannot send event")},[]),v=O.useCallback((k,C=0)=>{w.current.set(k,C);const _={type:"subscribe",timestamp:Date.now(),data:{thread_id:k,from_seq:C}};h(_)},[h]),y=O.useCallback((k,C)=>{const _=w.current.get(k)||0;C>_&&w.current.set(k,C);const R={type:"ack",timestamp:Date.now(),data:{thread_id:k,ack_seq:C}};h(R)},[h]),b=O.useCallback(()=>{const k={type:"ping",timestamp:Date.now()};h(k)},[h]),E=O.useCallback(k=>{w.current.delete(k)},[]);return O.useEffect(()=>(S(),()=>{p.current&&clearTimeout(p.current),o.current&&o.current.close()}),[S]),O.useEffect(()=>{if(!a)return;const k=setInterval(()=>{b()},3e4);return()=>clearInterval(k)},[a,b]),{isConnected:a,connectionError:c,subscribe:v,unsubscribe:E,acknowledge:y,ping:b}},ck=({connected:e})=>u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?u.jsxs(u.Fragment,{children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):u.jsxs(u.Fragment,{children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),u.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),dk=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=O.useState([]),[o,a]=O.useState(null),[s,c]=O.useState(new Map),[d,f]=O.useState(new Map),[m,p]=O.useState([]),[w,S]=O.useState(!1),[I,h]=O.useState(""),{isConnected:v,subscribe:y,acknowledge:b}=uk({url:e,instanceId:t,onMessage:E,onBatch:k});function E(N){const g={id:N.id,thread_id:N.thread_id,message_seq:N.message_seq,created_at:N.created_at,from_type:N.from_type,from_id:N.from_id,to_type:N.to_type,to_id:N.to_id,kind:N.kind,subject:N.subject,content:N.content,metadata_json:N.metadata_json,delivery_state:"visible",business_state:"open"};c(L=>{const $=L.get(g.thread_id)||[];return $.find(x=>x.id===g.id)?L:new Map(L).set(g.thread_id,[...$,g].sort((x,ne)=>x.message_seq-ne.message_seq))}),g.thread_id!==o&&f(L=>{const $=L.get(g.thread_id)||0;return new Map(L).set(g.thread_id,$+1)}),b(g.thread_id,g.message_seq)}function k(N){N.messages.forEach(g=>{E(g)})}const C=O.useCallback(N=>{if(a(N),f(g=>{const L=new Map(g);return L.delete(N),L}),v){const g=s.get(N)||[],L=g.length>0?Math.max(...g.map($=>$.message_seq)):0;y(N,L)}},[v,y,s]),_=O.useCallback(async(N,g,L)=>{if(!o)return;const $=L?JSON.stringify({workspace:L}):void 0;try{const x=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:g,content:N,metadata_json:$})});if(!x.ok){console.error("Failed to send message:",await x.text());return}const ne=await x.json();c(be=>{const te=be.get(o)||[];return te.find(Ae=>Ae.id===ne.id)?be:new Map(be).set(o,[...te,ne])})}catch(x){console.error("Error sending message:",x)}},[o,t]);O.useEffect(()=>{(async()=>{try{const g=await fetch("/api/threads");if(!g.ok){console.error("Failed to fetch threads:",await g.text());return}const L=await g.json();l(L),L.length>0&&!o&&a(L[0].id)}catch(g){console.error("Error fetching threads:",g)}})()},[]),O.useEffect(()=>{n&&i.length>0&&(i.some(g=>g.id===n)&&(a(n),f(g=>{const L=new Map(g);return L.delete(n),L})),r&&r())},[n,i,r]);const R=O.useCallback(async N=>{try{const g=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:N,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!g.ok){console.error("Failed to create thread:",await g.text());return}const L=await g.json();l($=>[L,...$]),a(L.id)}catch(g){console.error("Error creating thread:",g)}},[t]),P=O.useCallback(async()=>{try{const N=await fetch("/api/agents");if(!N.ok){console.error("Failed to fetch agents:",await N.text());return}const g=await N.json();p(g.running||[])}catch(N){console.error("Error fetching agents:",N)}},[]);O.useEffect(()=>{P();const N=setInterval(P,5e3);return()=>clearInterval(N)},[P]);const T=O.useCallback(async()=>{if(I.trim())try{const N=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:I.trim()})});if(!N.ok){const L=await N.text();console.error("Failed to launch agent:",L),alert(`Failed to launch agent: ${L}`);return}const g=await N.json();p(L=>[...L,g]),h(""),S(!1)}catch(N){console.error("Error launching agent:",N)}},[I]),D=O.useCallback(async N=>{try{const g=await fetch(`/api/agents/${N}`,{method:"DELETE"});if(!g.ok){console.error("Failed to stop agent:",await g.text());return}p(L=>L.filter($=>$.instance_id!==N))}catch(g){console.error("Error stopping agent:",g)}},[]),W=O.useCallback(async N=>{if(o)try{const g=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:N})});if(!g.ok){console.error("Failed to update workspace:",await g.text());return}const L=await g.json();l($=>$.map(x=>x.id===o?L:x))}catch(g){console.error("Error updating workspace:",g)}},[o]),X=O.useCallback(async N=>{try{const g=await fetch(`/api/threads/${N}`,{method:"DELETE"});if(!g.ok){console.error("Failed to delete thread:",await g.text());return}l(L=>L.filter($=>$.id!==N)),c(L=>{const $=new Map(L);return $.delete(N),$}),f(L=>{const $=new Map(L);return $.delete(N),$}),o===N&&a(null)}catch(g){console.error("Error deleting thread:",g)}},[o]),U=O.useCallback(async(N,g)=>{try{const L=await fetch(`/api/threads/${N}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:g})});if(!L.ok){console.error("Failed to rename thread:",await L.text());return}const $=await L.json();l(x=>x.map(ne=>ne.id===N?$:ne))}catch(L){console.error("Error renaming thread:",L)}},[]),Q=O.useCallback(async(N,g)=>{try{const L=await fetch(`/api/approvals/${N}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!L.ok){const $=await L.text();console.error("Failed to approve request:",$),alert(`Failed to approve: ${$}`);return}console.log("Approval approved successfully")}catch(L){console.error("Error approving request:",L)}},[]),ie=O.useCallback(async(N,g)=>{try{const L=await fetch(`/api/approvals/${N}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!L.ok){const $=await L.text();console.error("Failed to reject request:",$),alert(`Failed to reject: ${$}`);return}console.log("Approval rejected successfully")}catch(L){console.error("Error rejecting request:",L)}},[]),j=o?s.get(o)||[]:[];return u.jsxs("div",{className:"message-center",children:[u.jsxs("div",{className:"status-bar",children:[u.jsxs("div",{className:`status-indicator ${v?"connected":"disconnected"}`,children:[u.jsx(ck,{connected:v}),u.jsx("span",{children:v?"Connected":"Disconnected"})]}),u.jsxs("div",{className:"status-meta",children:[u.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),u.jsxs("span",{className:"agent-count",children:[m.length," agents"]}),u.jsx("button",{className:"launch-agent-btn",onClick:()=>S(!0),children:"+ Agent"})]})]}),m.length>0&&u.jsx("div",{className:"agents-bar",children:m.map(N=>u.jsxs("div",{className:"agent-chip",children:[u.jsx("span",{className:"agent-pulse"}),u.jsx("span",{className:"agent-name",children:N.instance_id}),u.jsxs("span",{className:"agent-pid",children:["PID ",N.pid]}),u.jsx("button",{className:"agent-stop-btn",onClick:()=>D(N.instance_id),title:"Stop agent",children:"×"})]},N.instance_id))}),w&&u.jsx("div",{className:"modal-overlay",onClick:()=>S(!1),children:u.jsxs("div",{className:"modal-content",onClick:N=>N.stopPropagation(),children:[u.jsx("h3",{children:"Launch New Agent"}),u.jsx("input",{type:"text",value:I,onChange:N=>h(N.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:N=>{N.key==="Enter"&&T(),N.key==="Escape"&&S(!1)}}),u.jsxs("div",{className:"modal-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>S(!1),children:"Cancel"}),u.jsx("button",{className:"launch-btn",onClick:T,children:"Launch"})]})]})}),u.jsxs("div",{className:"center-layout",children:[u.jsx("aside",{className:"threads-panel",children:u.jsx(mv,{threads:i,selectedThreadId:o,onSelectThread:C,onCreateThread:R,onDeleteThread:X,onRenameThread:U,unreadCounts:d})}),u.jsx("main",{className:"conversation-panel",children:o?u.jsx(ok,{thread:i.find(N=>N.id===o),messages:j,onSendMessage:_,onWorkspaceChange:W,onApproveRequest:Q,onRejectRequest:ie}):u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("h3",{children:"Select a conversation"}),u.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),u.jsx("style",{children:`
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
      `})]})},De={check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),dollar:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:u.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),u.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),u.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},fk=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=O.useState(!0),[a,s]=O.useState(null),[c,d]=O.useState(new Map),f=h=>{try{return JSON.parse(h)}catch{return null}},m=h=>new Date(h).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),p=h=>{const v=c.get(h)||"";n(h,v),d(new Map(c.set(h,"")))},w=h=>{const v=c.get(h)||"";if(!v.trim()){alert("Please provide a reason for rejection");return}r(h,v),d(new Map(c.set(h,"")))},S=(h,v)=>{d(new Map(c.set(h,v)))},I=e.filter(h=>h.status==="pending");return u.jsxs("div",{className:"approval-queue",children:[u.jsx("div",{className:"queue-header",children:u.jsxs("div",{className:"header-title",children:[u.jsx("h2",{children:"Approval Queue"}),u.jsxs("span",{className:"pending-count",children:[I.length," pending"]})]})}),u.jsxs("div",{className:"approvals-container",children:[I.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:De.sparkles}),u.jsx("h3",{children:"All caught up!"}),u.jsx("p",{children:"No pending approvals to review"})]}):u.jsx("div",{className:"approvals-list",children:I.map(h=>{const v=f(h.effect_delta_json),y=a===h.id;return u.jsxs("div",{className:`approval-card impact-${h.impact}`,children:[u.jsxs("div",{className:"card-header",onClick:()=>s(y?null:h.id),children:[u.jsxs("div",{className:"header-left",children:[u.jsx("div",{className:`impact-indicator ${h.impact}`}),u.jsxs("div",{className:"proposal-info",children:[u.jsx("span",{className:"proposal-text",children:h.proposal}),u.jsxs("div",{className:"proposal-meta",children:[h.thread_title&&u.jsxs("span",{className:"meta-item thread-link",onClick:b=>{b.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[De.message,h.thread_title]}),u.jsxs("span",{className:"meta-item",children:[De.bot,h.instance_id]}),u.jsxs("span",{className:"meta-item",children:[De.clock,m(h.created_at)]})]})]})]}),u.jsxs("div",{className:"header-right",children:[u.jsxs("span",{className:"cost-badge",children:[De.dollar,"$",h.estimated_cost.toFixed(2)]}),u.jsx("span",{className:`impact-badge ${h.impact}`,children:h.impact}),u.jsx("button",{className:"expand-btn",children:y?De.chevronUp:De.chevronDown})]})]}),y&&u.jsxs("div",{className:"card-details",children:[v&&u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Effect Details"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Capability"}),u.jsx("span",{className:"detail-value code",children:v.cap_type})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Budget Delta"}),u.jsxs("span",{className:"detail-value",children:["$",v.budget_delta.toFixed(2)]})]}),v.paths.length>0&&u.jsxs("div",{className:"detail-item full-width",children:[u.jsx("span",{className:"detail-label",children:"Paths"}),u.jsx("div",{className:"paths-list",children:v.paths.map((b,E)=>u.jsxs("span",{className:"path-tag",children:[De.folder,b]},E))})]})]})]}),u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Request Info"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Thread"}),u.jsx("span",{className:"detail-value code",children:h.thread_id})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Impact Level"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]})]})]}),u.jsxs("div",{className:"review-section",children:[u.jsx("h4",{children:"Review Notes"}),u.jsx("textarea",{value:c.get(h.id)||"",onChange:b=>S(h.id,b.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),u.jsxs("div",{className:"action-buttons",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>w(h.id),children:[De.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>p(h.id),children:[De.check,"Approve"]})]})]})]})]},h.id)})}),t.length>0&&u.jsxs("div",{className:"history-section",children:[u.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[u.jsxs("h3",{children:[l?De.chevronDown:De.chevronUp,"Review History"]}),u.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&u.jsx("div",{className:"history-list",children:t.map(h=>{const v=a===`history-${h.id}`;return u.jsxs("div",{className:`history-card ${h.status}`,onClick:()=>s(v?null:`history-${h.id}`),children:[u.jsxs("div",{className:"history-card-header",children:[u.jsxs("div",{className:"history-status",children:[u.jsx("span",{className:`status-icon ${h.status}`,children:h.status==="approved"?De.check:De.x}),u.jsxs("div",{className:"history-info",children:[u.jsx("span",{className:"history-proposal",children:h.proposal}),h.thread_title&&u.jsxs("span",{className:"history-thread",onClick:y=>{y.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[De.message,h.thread_title]})]})]}),u.jsxs("div",{className:"history-meta",children:[u.jsx("span",{className:"history-agent",children:h.instance_id}),u.jsx("span",{className:`history-badge ${h.status}`,children:h.status}),u.jsx("span",{className:"history-time",children:h.reviewed_at?m(h.reviewed_at):m(h.created_at)})]})]}),v&&u.jsxs("div",{className:"history-details",children:[u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Reviewed by"}),u.jsx("span",{className:"detail-value",children:h.reviewed_by||"Unknown"})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Cost"}),u.jsxs("span",{className:"detail-value",children:["$",h.estimated_cost.toFixed(2)]})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Impact"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]}),h.review_notes&&u.jsxs("div",{className:"detail-row full-width",children:[u.jsx("span",{className:"detail-label",children:"Notes"}),u.jsx("span",{className:"detail-value notes",children:h.review_notes})]})]})]},`history-${h.id}`)})})]})]}),u.jsx("style",{children:`
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
      `})]})},pk="_indicator_1ctaf_1",hk="_dot_1ctaf_12",mk="_connected_1ctaf_19",gk="_connecting_1ctaf_28",vk="_disconnected_1ctaf_37",yk="_pulsing_1ctaf_46",xk="_text_1ctaf_61",It={indicator:pk,dot:hk,connected:mk,connecting:gk,disconnected:vk,pulsing:yk,text:xk};function kk(){const[e,t]=O.useState("disconnected"),[n,r]=O.useState(0);if(O.useEffect(()=>ak((o,a)=>{t(o),r(a)}),[]),e==="connected")return u.jsx("div",{className:`${It.indicator} ${It.connected}`,title:"Connected",children:u.jsx("span",{className:It.dot})});const i=()=>{switch(e){case"connecting":return"Connecting...";case"reconnecting":return`Reconnecting... (${n})`;case"disconnected":return n>0?"Disconnected":"Offline";default:return"Unknown"}},l=()=>{switch(e){case"connecting":case"reconnecting":return It.connecting;case"disconnected":return It.disconnected;default:return""}};return u.jsxs("div",{className:`${It.indicator} ${l()}`,title:i(),children:[u.jsx("span",{className:`${It.dot} ${e==="connecting"||e==="reconnecting"?It.pulsing:""}`}),u.jsx("span",{className:It.text,children:i()})]})}const wk=u.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M12 6v12M6 12h12"}),u.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]}),Sk=()=>{const[e,t]=O.useState({type:"overview"}),[n,r]=O.useState(null),[i,l]=O.useState([]),[o,a]=O.useState([]),[s,c]=O.useState(!1),[d,f]=O.useState(""),p=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;O.useEffect(()=>{const E=async()=>{try{const C=await fetch("/api/hierarchy");if(C.ok){const _=await C.json();r(_)}}catch(C){console.error("Error fetching hierarchy:",C)}};E();const k=setInterval(E,5e3);return()=>clearInterval(k)},[]),O.useEffect(()=>{const E=async()=>{try{const C=await fetch("/api/approvals?status=pending");if(C.ok){const T=await C.json();l(T)}const[_,R]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),P=[];if(_.ok){const T=await _.json();P.push(...T)}if(R.ok){const T=await R.json();P.push(...T)}P.sort((T,D)=>{const W=T.reviewed_at?new Date(T.reviewed_at).getTime():0;return(D.reviewed_at?new Date(D.reviewed_at).getTime():0)-W}),a(P)}catch(C){console.error("Error fetching approvals:",C)}};E();const k=setInterval(E,5e3);return()=>clearInterval(k)},[]);const w=async(E,k)=>{try{const C=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:k})});if(!C.ok){console.error("Failed to approve:",await C.text());return}const _=i.find(R=>R.id===E);if(_){const R={..._,status:"approved",reviewed_by:"user",review_notes:k,reviewed_at:Date.now()};a(P=>[R,...P])}l(R=>R.filter(P=>P.id!==E))}catch(C){console.error("Error approving:",C)}},S=async(E,k)=>{try{const C=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:k})});if(!C.ok){console.error("Failed to reject:",await C.text());return}const _=i.find(R=>R.id===E);if(_){const R={..._,status:"rejected",reviewed_by:"user",review_notes:k,reviewed_at:Date.now()};a(P=>[R,...P])}l(R=>R.filter(P=>P.id!==E))}catch(C){console.error("Error rejecting:",C)}},I=()=>{var k,C;const E=[{label:"All Agents",onClick:()=>t({type:"overview"})}];if(e.type==="agent"&&e.agentId&&E.push({label:e.agentId}),e.type==="thread"&&e.threadId){e.agentId&&E.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})});const _=(k=n==null?void 0:n.root.children)==null?void 0:k.find(P=>P.id===e.agentId),R=(C=_==null?void 0:_.children)==null?void 0:C.find(P=>P.id===e.threadId);E.push({label:(R==null?void 0:R.label)||"Thread"})}return E},h=E=>{var C;const k=(C=n==null?void 0:n.root.children)==null?void 0:C.find(_=>{var R;return(R=_.children)==null?void 0:R.some(P=>P.id===E)});t({type:"thread",agentId:k==null?void 0:k.id,threadId:E})},v=async E=>{if(d.trim())try{const k=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:d.trim(),created_by_type:"human",created_by_id:"user",target_agent:E})});if(!k.ok){console.error("Failed to create thread:",await k.text());return}const C=await k.json();f(""),c(!1),t({type:"thread",agentId:E,threadId:C.id})}catch(k){console.error("Error creating thread:",k)}},y=()=>{var E,k,C;if(e.type==="overview"&&n)return u.jsx(pv,{aggregate:n.aggregate,agents:n.root.children||[],onSelectAgent:_=>t({type:"agent",agentId:_})});if(e.type==="agent"&&e.agentId){const _=(E=n==null?void 0:n.root.children)==null?void 0:E.find(P=>P.id===e.agentId),R=i.filter(P=>{var T;return(T=_==null?void 0:_.children)==null?void 0:T.some(D=>D.id===P.thread_id)});return u.jsxs("div",{className:"agent-view",children:[u.jsxs("div",{className:"agent-view-header",children:[u.jsx("h2",{children:e.agentId}),u.jsxs("span",{className:"agent-thread-count",children:[((k=_==null?void 0:_.children)==null?void 0:k.length)||0," threads"]})]}),u.jsxs("div",{className:"agent-view-content",children:[u.jsxs("div",{className:"agent-threads",children:[u.jsxs("div",{className:"threads-header",children:[u.jsx("h3",{children:"Threads"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>c(!0),title:"New thread",children:"+ New Thread"})]}),s&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:d,onChange:P=>f(P.target.value),onKeyDown:P=>{P.key==="Enter"&&v(e.agentId),P.key==="Escape"&&(c(!1),f(""))},placeholder:"Thread title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{onClick:()=>{c(!1),f("")},children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:()=>v(e.agentId),children:"Create"})]})]}),(C=_==null?void 0:_.children)==null?void 0:C.map(P=>u.jsxs("div",{className:"thread-card",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:P.id}),children:[u.jsx("span",{className:"thread-title",children:P.label}),P.badges&&P.badges.length>0&&u.jsx("span",{className:"thread-badges",children:P.badges.map((T,D)=>u.jsx("span",{className:`badge badge-${T.type}`,children:T.count},D))})]},P.id)),(!(_!=null&&_.children)||_.children.length===0)&&!s&&u.jsxs("div",{className:"no-threads",children:["No threads yet",u.jsx("button",{className:"start-thread-btn",onClick:()=>c(!0),children:"Start a conversation"})]})]}),R.length>0&&u.jsxs("div",{className:"agent-approvals",children:[u.jsx("h3",{children:"Pending Approvals"}),u.jsx(fk,{approvals:R,history:[],onApprove:w,onReject:S,onNavigateToThread:h})]})]})]})}return e.type==="thread"&&e.threadId?u.jsx(dk,{websocketUrl:p,instanceId:e.agentId||"default",initialThreadId:e.threadId,onThreadNavigated:()=>{}}):u.jsx("div",{className:"empty-state",children:u.jsx("p",{children:"Select an agent or thread from the sidebar"})})},b=(i==null?void 0:i.filter(E=>E.status==="pending").length)||0;return u.jsxs("div",{className:"app",children:[u.jsxs("header",{className:"app-header",children:[u.jsxs("div",{className:"header-brand",children:[u.jsx("div",{className:"brand-logo",children:wk}),u.jsxs("div",{className:"brand-text",children:[u.jsx("h1",{children:"AILANG"}),u.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),u.jsxs("div",{className:"header-meta",children:[u.jsx(kk,{}),b>0&&u.jsxs("span",{className:"pending-badge",title:`${b} pending approvals`,children:[b," pending"]}),u.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),u.jsxs("div",{className:"app-body",children:[u.jsx("aside",{className:"app-sidebar",children:u.jsx(Lg,{selection:e,onSelect:t})}),u.jsxs("main",{className:"app-main",children:[e.type!=="overview"&&u.jsx(hv,{items:I()}),u.jsx("div",{className:"main-content",children:y()})]})]}),u.jsx("style",{children:`
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
      `})]})};To.createRoot(document.getElementById("root")).render(u.jsx(Kt.StrictMode,{children:u.jsx(Sk,{})}));
