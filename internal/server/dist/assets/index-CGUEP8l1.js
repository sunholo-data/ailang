(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))r(i);new MutationObserver(i=>{for(const l of i)if(l.type==="childList")for(const o of l.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&r(o)}).observe(document,{childList:!0,subtree:!0});function n(i){const l={};return i.integrity&&(l.integrity=i.integrity),i.referrerPolicy&&(l.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?l.credentials="include":i.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function r(i){if(i.ep)return;i.ep=!0;const l=n(i);fetch(i.href,l)}})();var Yi=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Ia(e){return e&&e.__esModule&&Object.prototype.hasOwnProperty.call(e,"default")?e.default:e}var Yc={exports:{}},Sl={},Xc={exports:{}},Y={};/**
 * @license React
 * react.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var oi=Symbol.for("react.element"),Zp=Symbol.for("react.portal"),eh=Symbol.for("react.fragment"),th=Symbol.for("react.strict_mode"),nh=Symbol.for("react.profiler"),rh=Symbol.for("react.provider"),ih=Symbol.for("react.context"),lh=Symbol.for("react.forward_ref"),oh=Symbol.for("react.suspense"),ah=Symbol.for("react.memo"),sh=Symbol.for("react.lazy"),Ks=Symbol.iterator;function uh(e){return e===null||typeof e!="object"?null:(e=Ks&&e[Ks]||e["@@iterator"],typeof e=="function"?e:null)}var Gc={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},Jc=Object.assign,Zc={};function ur(e,t,n){this.props=e,this.context=t,this.refs=Zc,this.updater=n||Gc}ur.prototype.isReactComponent={};ur.prototype.setState=function(e,t){if(typeof e!="object"&&typeof e!="function"&&e!=null)throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,e,t,"setState")};ur.prototype.forceUpdate=function(e){this.updater.enqueueForceUpdate(this,e,"forceUpdate")};function ed(){}ed.prototype=ur.prototype;function Ma(e,t,n){this.props=e,this.context=t,this.refs=Zc,this.updater=n||Gc}var Aa=Ma.prototype=new ed;Aa.constructor=Ma;Jc(Aa,ur.prototype);Aa.isPureReactComponent=!0;var Ys=Array.isArray,td=Object.prototype.hasOwnProperty,Da={current:null},nd={key:!0,ref:!0,__self:!0,__source:!0};function rd(e,t,n){var r,i={},l=null,o=null;if(t!=null)for(r in t.ref!==void 0&&(o=t.ref),t.key!==void 0&&(l=""+t.key),t)td.call(t,r)&&!nd.hasOwnProperty(r)&&(i[r]=t[r]);var a=arguments.length-2;if(a===1)i.children=n;else if(1<a){for(var s=Array(a),c=0;c<a;c++)s[c]=arguments[c+2];i.children=s}if(e&&e.defaultProps)for(r in a=e.defaultProps,a)i[r]===void 0&&(i[r]=a[r]);return{$$typeof:oi,type:e,key:l,ref:o,props:i,_owner:Da.current}}function ch(e,t){return{$$typeof:oi,type:e.type,key:t,ref:e.ref,props:e.props,_owner:e._owner}}function Ra(e){return typeof e=="object"&&e!==null&&e.$$typeof===oi}function dh(e){var t={"=":"=0",":":"=2"};return"$"+e.replace(/[=:]/g,function(n){return t[n]})}var Xs=/\/+/g;function Ul(e,t){return typeof e=="object"&&e!==null&&e.key!=null?dh(""+e.key):t.toString(36)}function Ai(e,t,n,r,i){var l=typeof e;(l==="undefined"||l==="boolean")&&(e=null);var o=!1;if(e===null)o=!0;else switch(l){case"string":case"number":o=!0;break;case"object":switch(e.$$typeof){case oi:case Zp:o=!0}}if(o)return o=e,i=i(o),e=r===""?"."+Ul(o,0):r,Ys(i)?(n="",e!=null&&(n=e.replace(Xs,"$&/")+"/"),Ai(i,t,n,"",function(c){return c})):i!=null&&(Ra(i)&&(i=ch(i,n+(!i.key||o&&o.key===i.key?"":(""+i.key).replace(Xs,"$&/")+"/")+e)),t.push(i)),1;if(o=0,r=r===""?".":r+":",Ys(e))for(var a=0;a<e.length;a++){l=e[a];var s=r+Ul(l,a);o+=Ai(l,t,n,s,i)}else if(s=uh(e),typeof s=="function")for(e=s.call(e),a=0;!(l=e.next()).done;)l=l.value,s=r+Ul(l,a++),o+=Ai(l,t,n,s,i);else if(l==="object")throw t=String(e),Error("Objects are not valid as a React child (found: "+(t==="[object Object]"?"object with keys {"+Object.keys(e).join(", ")+"}":t)+"). If you meant to render a collection of children, use an array instead.");return o}function pi(e,t,n){if(e==null)return e;var r=[],i=0;return Ai(e,r,"","",function(l){return t.call(n,l,i++)}),r}function fh(e){if(e._status===-1){var t=e._result;t=t(),t.then(function(n){(e._status===0||e._status===-1)&&(e._status=1,e._result=n)},function(n){(e._status===0||e._status===-1)&&(e._status=2,e._result=n)}),e._status===-1&&(e._status=0,e._result=t)}if(e._status===1)return e._result.default;throw e._result}var Fe={current:null},Di={transition:null},ph={ReactCurrentDispatcher:Fe,ReactCurrentBatchConfig:Di,ReactCurrentOwner:Da};function id(){throw Error("act(...) is not supported in production builds of React.")}Y.Children={map:pi,forEach:function(e,t,n){pi(e,function(){t.apply(this,arguments)},n)},count:function(e){var t=0;return pi(e,function(){t++}),t},toArray:function(e){return pi(e,function(t){return t})||[]},only:function(e){if(!Ra(e))throw Error("React.Children.only expected to receive a single React element child.");return e}};Y.Component=ur;Y.Fragment=eh;Y.Profiler=nh;Y.PureComponent=Ma;Y.StrictMode=th;Y.Suspense=oh;Y.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=ph;Y.act=id;Y.cloneElement=function(e,t,n){if(e==null)throw Error("React.cloneElement(...): The argument must be a React element, but you passed "+e+".");var r=Jc({},e.props),i=e.key,l=e.ref,o=e._owner;if(t!=null){if(t.ref!==void 0&&(l=t.ref,o=Da.current),t.key!==void 0&&(i=""+t.key),e.type&&e.type.defaultProps)var a=e.type.defaultProps;for(s in t)td.call(t,s)&&!nd.hasOwnProperty(s)&&(r[s]=t[s]===void 0&&a!==void 0?a[s]:t[s])}var s=arguments.length-2;if(s===1)r.children=n;else if(1<s){a=Array(s);for(var c=0;c<s;c++)a[c]=arguments[c+2];r.children=a}return{$$typeof:oi,type:e.type,key:i,ref:l,props:r,_owner:o}};Y.createContext=function(e){return e={$$typeof:ih,_currentValue:e,_currentValue2:e,_threadCount:0,Provider:null,Consumer:null,_defaultValue:null,_globalName:null},e.Provider={$$typeof:rh,_context:e},e.Consumer=e};Y.createElement=rd;Y.createFactory=function(e){var t=rd.bind(null,e);return t.type=e,t};Y.createRef=function(){return{current:null}};Y.forwardRef=function(e){return{$$typeof:lh,render:e}};Y.isValidElement=Ra;Y.lazy=function(e){return{$$typeof:sh,_payload:{_status:-1,_result:e},_init:fh}};Y.memo=function(e,t){return{$$typeof:ah,type:e,compare:t===void 0?null:t}};Y.startTransition=function(e){var t=Di.transition;Di.transition={};try{e()}finally{Di.transition=t}};Y.unstable_act=id;Y.useCallback=function(e,t){return Fe.current.useCallback(e,t)};Y.useContext=function(e){return Fe.current.useContext(e)};Y.useDebugValue=function(){};Y.useDeferredValue=function(e){return Fe.current.useDeferredValue(e)};Y.useEffect=function(e,t){return Fe.current.useEffect(e,t)};Y.useId=function(){return Fe.current.useId()};Y.useImperativeHandle=function(e,t,n){return Fe.current.useImperativeHandle(e,t,n)};Y.useInsertionEffect=function(e,t){return Fe.current.useInsertionEffect(e,t)};Y.useLayoutEffect=function(e,t){return Fe.current.useLayoutEffect(e,t)};Y.useMemo=function(e,t){return Fe.current.useMemo(e,t)};Y.useReducer=function(e,t,n){return Fe.current.useReducer(e,t,n)};Y.useRef=function(e){return Fe.current.useRef(e)};Y.useState=function(e){return Fe.current.useState(e)};Y.useSyncExternalStore=function(e,t,n){return Fe.current.useSyncExternalStore(e,t,n)};Y.useTransition=function(){return Fe.current.useTransition()};Y.version="18.3.1";Xc.exports=Y;var O=Xc.exports;const qt=Ia(O);/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var hh=O,mh=Symbol.for("react.element"),gh=Symbol.for("react.fragment"),vh=Object.prototype.hasOwnProperty,yh=hh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,xh={key:!0,ref:!0,__self:!0,__source:!0};function ld(e,t,n){var r,i={},l=null,o=null;n!==void 0&&(l=""+n),t.key!==void 0&&(l=""+t.key),t.ref!==void 0&&(o=t.ref);for(r in t)vh.call(t,r)&&!xh.hasOwnProperty(r)&&(i[r]=t[r]);if(e&&e.defaultProps)for(r in t=e.defaultProps,t)i[r]===void 0&&(i[r]=t[r]);return{$$typeof:mh,type:e,key:l,ref:o,props:i,_owner:yh.current}}Sl.Fragment=gh;Sl.jsx=ld;Sl.jsxs=ld;Yc.exports=Sl;var u=Yc.exports,_o={},od={exports:{}},rt={},ad={exports:{}},sd={};/**
 * @license React
 * scheduler.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */(function(e){function t(b,E){var g=b.length;b.push(E);e:for(;0<g;){var L=g-1>>>1,$=b[L];if(0<i($,E))b[L]=E,b[g]=$,g=L;else break e}}function n(b){return b.length===0?null:b[0]}function r(b){if(b.length===0)return null;var E=b[0],g=b.pop();if(g!==E){b[0]=g;e:for(var L=0,$=b.length,x=$>>>1;L<x;){var ne=2*(L+1)-1,be=b[ne],te=ne+1,Me=b[te];if(0>i(be,g))te<$&&0>i(Me,be)?(b[L]=Me,b[te]=g,L=te):(b[L]=be,b[ne]=g,L=ne);else if(te<$&&0>i(Me,g))b[L]=Me,b[te]=g,L=te;else break e}}return E}function i(b,E){var g=b.sortIndex-E.sortIndex;return g!==0?g:b.id-E.id}if(typeof performance=="object"&&typeof performance.now=="function"){var l=performance;e.unstable_now=function(){return l.now()}}else{var o=Date,a=o.now();e.unstable_now=function(){return o.now()-a}}var s=[],c=[],d=1,f=null,m=3,p=!1,w=!1,S=!1,I=typeof setTimeout=="function"?setTimeout:null,h=typeof clearTimeout=="function"?clearTimeout:null,v=typeof setImmediate<"u"?setImmediate:null;typeof navigator<"u"&&navigator.scheduling!==void 0&&navigator.scheduling.isInputPending!==void 0&&navigator.scheduling.isInputPending.bind(navigator.scheduling);function y(b){for(var E=n(c);E!==null;){if(E.callback===null)r(c);else if(E.startTime<=b)r(c),E.sortIndex=E.expirationTime,t(s,E);else break;E=n(c)}}function C(b){if(S=!1,y(b),!w)if(n(s)!==null)w=!0,Q(N);else{var E=n(c);E!==null&&ie(C,E.startTime-b)}}function N(b,E){w=!1,S&&(S=!1,h(_),_=-1),p=!0;var g=m;try{for(y(E),f=n(s);f!==null&&(!(f.expirationTime>E)||b&&!z());){var L=f.callback;if(typeof L=="function"){f.callback=null,m=f.priorityLevel;var $=L(f.expirationTime<=E);E=e.unstable_now(),typeof $=="function"?f.callback=$:f===n(s)&&r(s),y(E)}else r(s);f=n(s)}if(f!==null)var x=!0;else{var ne=n(c);ne!==null&&ie(C,ne.startTime-E),x=!1}return x}finally{f=null,m=g,p=!1}}var k=!1,j=null,_=-1,R=5,P=-1;function z(){return!(e.unstable_now()-P<R)}function D(){if(j!==null){var b=e.unstable_now();P=b;var E=!0;try{E=j(!0,b)}finally{E?W():(k=!1,j=null)}}else k=!1}var W;if(typeof v=="function")W=function(){v(D)};else if(typeof MessageChannel<"u"){var X=new MessageChannel,U=X.port2;X.port1.onmessage=D,W=function(){U.postMessage(null)}}else W=function(){I(D,0)};function Q(b){j=b,k||(k=!0,W())}function ie(b,E){_=I(function(){b(e.unstable_now())},E)}e.unstable_IdlePriority=5,e.unstable_ImmediatePriority=1,e.unstable_LowPriority=4,e.unstable_NormalPriority=3,e.unstable_Profiling=null,e.unstable_UserBlockingPriority=2,e.unstable_cancelCallback=function(b){b.callback=null},e.unstable_continueExecution=function(){w||p||(w=!0,Q(N))},e.unstable_forceFrameRate=function(b){0>b||125<b?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):R=0<b?Math.floor(1e3/b):5},e.unstable_getCurrentPriorityLevel=function(){return m},e.unstable_getFirstCallbackNode=function(){return n(s)},e.unstable_next=function(b){switch(m){case 1:case 2:case 3:var E=3;break;default:E=m}var g=m;m=E;try{return b()}finally{m=g}},e.unstable_pauseExecution=function(){},e.unstable_requestPaint=function(){},e.unstable_runWithPriority=function(b,E){switch(b){case 1:case 2:case 3:case 4:case 5:break;default:b=3}var g=m;m=b;try{return E()}finally{m=g}},e.unstable_scheduleCallback=function(b,E,g){var L=e.unstable_now();switch(typeof g=="object"&&g!==null?(g=g.delay,g=typeof g=="number"&&0<g?L+g:L):g=L,b){case 1:var $=-1;break;case 2:$=250;break;case 5:$=1073741823;break;case 4:$=1e4;break;default:$=5e3}return $=g+$,b={id:d++,callback:E,priorityLevel:b,startTime:g,expirationTime:$,sortIndex:-1},g>L?(b.sortIndex=g,t(c,b),n(s)===null&&b===n(c)&&(S?(h(_),_=-1):S=!0,ie(C,g-L))):(b.sortIndex=$,t(s,b),w||p||(w=!0,Q(N))),b},e.unstable_shouldYield=z,e.unstable_wrapCallback=function(b){var E=m;return function(){var g=m;m=E;try{return b.apply(this,arguments)}finally{m=g}}}})(sd);ad.exports=sd;var kh=ad.exports;/**
 * @license React
 * react-dom.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var wh=O,nt=kh;function M(e){for(var t="https://reactjs.org/docs/error-decoder.html?invariant="+e,n=1;n<arguments.length;n++)t+="&args[]="+encodeURIComponent(arguments[n]);return"Minified React error #"+e+"; visit "+t+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}var ud=new Set,Ur={};function zn(e,t){nr(e,t),nr(e+"Capture",t)}function nr(e,t){for(Ur[e]=t,e=0;e<t.length;e++)ud.add(t[e])}var Ft=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),zo=Object.prototype.hasOwnProperty,Sh=/^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/,Gs={},Js={};function bh(e){return zo.call(Js,e)?!0:zo.call(Gs,e)?!1:Sh.test(e)?Js[e]=!0:(Gs[e]=!0,!1)}function Ch(e,t,n,r){if(n!==null&&n.type===0)return!1;switch(typeof t){case"function":case"symbol":return!0;case"boolean":return r?!1:n!==null?!n.acceptsBooleans:(e=e.toLowerCase().slice(0,5),e!=="data-"&&e!=="aria-");default:return!1}}function jh(e,t,n,r){if(t===null||typeof t>"u"||Ch(e,t,n,r))return!0;if(r)return!1;if(n!==null)switch(n.type){case 3:return!t;case 4:return t===!1;case 5:return isNaN(t);case 6:return isNaN(t)||1>t}return!1}function Oe(e,t,n,r,i,l,o){this.acceptsBooleans=t===2||t===3||t===4,this.attributeName=r,this.attributeNamespace=i,this.mustUseProperty=n,this.propertyName=e,this.type=t,this.sanitizeURL=l,this.removeEmptyString=o}var Ne={};"children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(e){Ne[e]=new Oe(e,0,!1,e,null,!1,!1)});[["acceptCharset","accept-charset"],["className","class"],["htmlFor","for"],["httpEquiv","http-equiv"]].forEach(function(e){var t=e[0];Ne[t]=new Oe(t,1,!1,e[1],null,!1,!1)});["contentEditable","draggable","spellCheck","value"].forEach(function(e){Ne[e]=new Oe(e,2,!1,e.toLowerCase(),null,!1,!1)});["autoReverse","externalResourcesRequired","focusable","preserveAlpha"].forEach(function(e){Ne[e]=new Oe(e,2,!1,e,null,!1,!1)});"allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(e){Ne[e]=new Oe(e,3,!1,e.toLowerCase(),null,!1,!1)});["checked","multiple","muted","selected"].forEach(function(e){Ne[e]=new Oe(e,3,!0,e,null,!1,!1)});["capture","download"].forEach(function(e){Ne[e]=new Oe(e,4,!1,e,null,!1,!1)});["cols","rows","size","span"].forEach(function(e){Ne[e]=new Oe(e,6,!1,e,null,!1,!1)});["rowSpan","start"].forEach(function(e){Ne[e]=new Oe(e,5,!1,e.toLowerCase(),null,!1,!1)});var Fa=/[\-:]([a-z])/g;function Oa(e){return e[1].toUpperCase()}"accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(e){var t=e.replace(Fa,Oa);Ne[t]=new Oe(t,1,!1,e,null,!1,!1)});"xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(e){var t=e.replace(Fa,Oa);Ne[t]=new Oe(t,1,!1,e,"http://www.w3.org/1999/xlink",!1,!1)});["xml:base","xml:lang","xml:space"].forEach(function(e){var t=e.replace(Fa,Oa);Ne[t]=new Oe(t,1,!1,e,"http://www.w3.org/XML/1998/namespace",!1,!1)});["tabIndex","crossOrigin"].forEach(function(e){Ne[e]=new Oe(e,1,!1,e.toLowerCase(),null,!1,!1)});Ne.xlinkHref=new Oe("xlinkHref",1,!1,"xlink:href","http://www.w3.org/1999/xlink",!0,!1);["src","href","action","formAction"].forEach(function(e){Ne[e]=new Oe(e,1,!1,e.toLowerCase(),null,!0,!0)});function Ba(e,t,n,r){var i=Ne.hasOwnProperty(t)?Ne[t]:null;(i!==null?i.type!==0:r||!(2<t.length)||t[0]!=="o"&&t[0]!=="O"||t[1]!=="n"&&t[1]!=="N")&&(jh(t,n,i,r)&&(n=null),r||i===null?bh(t)&&(n===null?e.removeAttribute(t):e.setAttribute(t,""+n)):i.mustUseProperty?e[i.propertyName]=n===null?i.type===3?!1:"":n:(t=i.attributeName,r=i.attributeNamespace,n===null?e.removeAttribute(t):(i=i.type,n=i===3||i===4&&n===!0?"":""+n,r?e.setAttributeNS(r,t,n):e.setAttribute(t,n))))}var Ut=wh.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED,hi=Symbol.for("react.element"),Dn=Symbol.for("react.portal"),Rn=Symbol.for("react.fragment"),$a=Symbol.for("react.strict_mode"),To=Symbol.for("react.profiler"),cd=Symbol.for("react.provider"),dd=Symbol.for("react.context"),Ua=Symbol.for("react.forward_ref"),Lo=Symbol.for("react.suspense"),Po=Symbol.for("react.suspense_list"),Ha=Symbol.for("react.memo"),Kt=Symbol.for("react.lazy"),fd=Symbol.for("react.offscreen"),Zs=Symbol.iterator;function gr(e){return e===null||typeof e!="object"?null:(e=Zs&&e[Zs]||e["@@iterator"],typeof e=="function"?e:null)}var he=Object.assign,Hl;function Er(e){if(Hl===void 0)try{throw Error()}catch(n){var t=n.stack.trim().match(/\n( *(at )?)/);Hl=t&&t[1]||""}return`
`+Hl+e}var Vl=!1;function Wl(e,t){if(!e||Vl)return"";Vl=!0;var n=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{if(t)if(t=function(){throw Error()},Object.defineProperty(t.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(t,[])}catch(c){var r=c}Reflect.construct(e,[],t)}else{try{t.call()}catch(c){r=c}e.call(t.prototype)}else{try{throw Error()}catch(c){r=c}e()}}catch(c){if(c&&r&&typeof c.stack=="string"){for(var i=c.stack.split(`
`),l=r.stack.split(`
`),o=i.length-1,a=l.length-1;1<=o&&0<=a&&i[o]!==l[a];)a--;for(;1<=o&&0<=a;o--,a--)if(i[o]!==l[a]){if(o!==1||a!==1)do if(o--,a--,0>a||i[o]!==l[a]){var s=`
`+i[o].replace(" at new "," at ");return e.displayName&&s.includes("<anonymous>")&&(s=s.replace("<anonymous>",e.displayName)),s}while(1<=o&&0<=a);break}}}finally{Vl=!1,Error.prepareStackTrace=n}return(e=e?e.displayName||e.name:"")?Er(e):""}function Eh(e){switch(e.tag){case 5:return Er(e.type);case 16:return Er("Lazy");case 13:return Er("Suspense");case 19:return Er("SuspenseList");case 0:case 2:case 15:return e=Wl(e.type,!1),e;case 11:return e=Wl(e.type.render,!1),e;case 1:return e=Wl(e.type,!0),e;default:return""}}function Io(e){if(e==null)return null;if(typeof e=="function")return e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Rn:return"Fragment";case Dn:return"Portal";case To:return"Profiler";case $a:return"StrictMode";case Lo:return"Suspense";case Po:return"SuspenseList"}if(typeof e=="object")switch(e.$$typeof){case dd:return(e.displayName||"Context")+".Consumer";case cd:return(e._context.displayName||"Context")+".Provider";case Ua:var t=e.render;return e=e.displayName,e||(e=t.displayName||t.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case Ha:return t=e.displayName||null,t!==null?t:Io(e.type)||"Memo";case Kt:t=e._payload,e=e._init;try{return Io(e(t))}catch{}}return null}function Nh(e){var t=e.type;switch(e.tag){case 24:return"Cache";case 9:return(t.displayName||"Context")+".Consumer";case 10:return(t._context.displayName||"Context")+".Provider";case 18:return"DehydratedFragment";case 11:return e=t.render,e=e.displayName||e.name||"",t.displayName||(e!==""?"ForwardRef("+e+")":"ForwardRef");case 7:return"Fragment";case 5:return t;case 4:return"Portal";case 3:return"Root";case 6:return"Text";case 16:return Io(t);case 8:return t===$a?"StrictMode":"Mode";case 22:return"Offscreen";case 12:return"Profiler";case 21:return"Scope";case 13:return"Suspense";case 19:return"SuspenseList";case 25:return"TracingMarker";case 1:case 0:case 17:case 2:case 14:case 15:if(typeof t=="function")return t.displayName||t.name||null;if(typeof t=="string")return t}return null}function un(e){switch(typeof e){case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function pd(e){var t=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(t==="checkbox"||t==="radio")}function _h(e){var t=pd(e)?"checked":"value",n=Object.getOwnPropertyDescriptor(e.constructor.prototype,t),r=""+e[t];if(!e.hasOwnProperty(t)&&typeof n<"u"&&typeof n.get=="function"&&typeof n.set=="function"){var i=n.get,l=n.set;return Object.defineProperty(e,t,{configurable:!0,get:function(){return i.call(this)},set:function(o){r=""+o,l.call(this,o)}}),Object.defineProperty(e,t,{enumerable:n.enumerable}),{getValue:function(){return r},setValue:function(o){r=""+o},stopTracking:function(){e._valueTracker=null,delete e[t]}}}}function mi(e){e._valueTracker||(e._valueTracker=_h(e))}function hd(e){if(!e)return!1;var t=e._valueTracker;if(!t)return!0;var n=t.getValue(),r="";return e&&(r=pd(e)?e.checked?"true":"false":e.value),e=r,e!==n?(t.setValue(e),!0):!1}function Xi(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function Mo(e,t){var n=t.checked;return he({},t,{defaultChecked:void 0,defaultValue:void 0,value:void 0,checked:n??e._wrapperState.initialChecked})}function eu(e,t){var n=t.defaultValue==null?"":t.defaultValue,r=t.checked!=null?t.checked:t.defaultChecked;n=un(t.value!=null?t.value:n),e._wrapperState={initialChecked:r,initialValue:n,controlled:t.type==="checkbox"||t.type==="radio"?t.checked!=null:t.value!=null}}function md(e,t){t=t.checked,t!=null&&Ba(e,"checked",t,!1)}function Ao(e,t){md(e,t);var n=un(t.value),r=t.type;if(n!=null)r==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+n):e.value!==""+n&&(e.value=""+n);else if(r==="submit"||r==="reset"){e.removeAttribute("value");return}t.hasOwnProperty("value")?Do(e,t.type,n):t.hasOwnProperty("defaultValue")&&Do(e,t.type,un(t.defaultValue)),t.checked==null&&t.defaultChecked!=null&&(e.defaultChecked=!!t.defaultChecked)}function tu(e,t,n){if(t.hasOwnProperty("value")||t.hasOwnProperty("defaultValue")){var r=t.type;if(!(r!=="submit"&&r!=="reset"||t.value!==void 0&&t.value!==null))return;t=""+e._wrapperState.initialValue,n||t===e.value||(e.value=t),e.defaultValue=t}n=e.name,n!==""&&(e.name=""),e.defaultChecked=!!e._wrapperState.initialChecked,n!==""&&(e.name=n)}function Do(e,t,n){(t!=="number"||Xi(e.ownerDocument)!==e)&&(n==null?e.defaultValue=""+e._wrapperState.initialValue:e.defaultValue!==""+n&&(e.defaultValue=""+n))}var Nr=Array.isArray;function Kn(e,t,n,r){if(e=e.options,t){t={};for(var i=0;i<n.length;i++)t["$"+n[i]]=!0;for(n=0;n<e.length;n++)i=t.hasOwnProperty("$"+e[n].value),e[n].selected!==i&&(e[n].selected=i),i&&r&&(e[n].defaultSelected=!0)}else{for(n=""+un(n),t=null,i=0;i<e.length;i++){if(e[i].value===n){e[i].selected=!0,r&&(e[i].defaultSelected=!0);return}t!==null||e[i].disabled||(t=e[i])}t!==null&&(t.selected=!0)}}function Ro(e,t){if(t.dangerouslySetInnerHTML!=null)throw Error(M(91));return he({},t,{value:void 0,defaultValue:void 0,children:""+e._wrapperState.initialValue})}function nu(e,t){var n=t.value;if(n==null){if(n=t.children,t=t.defaultValue,n!=null){if(t!=null)throw Error(M(92));if(Nr(n)){if(1<n.length)throw Error(M(93));n=n[0]}t=n}t==null&&(t=""),n=t}e._wrapperState={initialValue:un(n)}}function gd(e,t){var n=un(t.value),r=un(t.defaultValue);n!=null&&(n=""+n,n!==e.value&&(e.value=n),t.defaultValue==null&&e.defaultValue!==n&&(e.defaultValue=n)),r!=null&&(e.defaultValue=""+r)}function ru(e){var t=e.textContent;t===e._wrapperState.initialValue&&t!==""&&t!==null&&(e.value=t)}function vd(e){switch(e){case"svg":return"http://www.w3.org/2000/svg";case"math":return"http://www.w3.org/1998/Math/MathML";default:return"http://www.w3.org/1999/xhtml"}}function Fo(e,t){return e==null||e==="http://www.w3.org/1999/xhtml"?vd(t):e==="http://www.w3.org/2000/svg"&&t==="foreignObject"?"http://www.w3.org/1999/xhtml":e}var gi,yd=function(e){return typeof MSApp<"u"&&MSApp.execUnsafeLocalFunction?function(t,n,r,i){MSApp.execUnsafeLocalFunction(function(){return e(t,n,r,i)})}:e}(function(e,t){if(e.namespaceURI!=="http://www.w3.org/2000/svg"||"innerHTML"in e)e.innerHTML=t;else{for(gi=gi||document.createElement("div"),gi.innerHTML="<svg>"+t.valueOf().toString()+"</svg>",t=gi.firstChild;e.firstChild;)e.removeChild(e.firstChild);for(;t.firstChild;)e.appendChild(t.firstChild)}});function Hr(e,t){if(t){var n=e.firstChild;if(n&&n===e.lastChild&&n.nodeType===3){n.nodeValue=t;return}}e.textContent=t}var Tr={animationIterationCount:!0,aspectRatio:!0,borderImageOutset:!0,borderImageSlice:!0,borderImageWidth:!0,boxFlex:!0,boxFlexGroup:!0,boxOrdinalGroup:!0,columnCount:!0,columns:!0,flex:!0,flexGrow:!0,flexPositive:!0,flexShrink:!0,flexNegative:!0,flexOrder:!0,gridArea:!0,gridRow:!0,gridRowEnd:!0,gridRowSpan:!0,gridRowStart:!0,gridColumn:!0,gridColumnEnd:!0,gridColumnSpan:!0,gridColumnStart:!0,fontWeight:!0,lineClamp:!0,lineHeight:!0,opacity:!0,order:!0,orphans:!0,tabSize:!0,widows:!0,zIndex:!0,zoom:!0,fillOpacity:!0,floodOpacity:!0,stopOpacity:!0,strokeDasharray:!0,strokeDashoffset:!0,strokeMiterlimit:!0,strokeOpacity:!0,strokeWidth:!0},zh=["Webkit","ms","Moz","O"];Object.keys(Tr).forEach(function(e){zh.forEach(function(t){t=t+e.charAt(0).toUpperCase()+e.substring(1),Tr[t]=Tr[e]})});function xd(e,t,n){return t==null||typeof t=="boolean"||t===""?"":n||typeof t!="number"||t===0||Tr.hasOwnProperty(e)&&Tr[e]?(""+t).trim():t+"px"}function kd(e,t){e=e.style;for(var n in t)if(t.hasOwnProperty(n)){var r=n.indexOf("--")===0,i=xd(n,t[n],r);n==="float"&&(n="cssFloat"),r?e.setProperty(n,i):e[n]=i}}var Th=he({menuitem:!0},{area:!0,base:!0,br:!0,col:!0,embed:!0,hr:!0,img:!0,input:!0,keygen:!0,link:!0,meta:!0,param:!0,source:!0,track:!0,wbr:!0});function Oo(e,t){if(t){if(Th[e]&&(t.children!=null||t.dangerouslySetInnerHTML!=null))throw Error(M(137,e));if(t.dangerouslySetInnerHTML!=null){if(t.children!=null)throw Error(M(60));if(typeof t.dangerouslySetInnerHTML!="object"||!("__html"in t.dangerouslySetInnerHTML))throw Error(M(61))}if(t.style!=null&&typeof t.style!="object")throw Error(M(62))}}function Bo(e,t){if(e.indexOf("-")===-1)return typeof t.is=="string";switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var $o=null;function Va(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var Uo=null,Yn=null,Xn=null;function iu(e){if(e=ui(e)){if(typeof Uo!="function")throw Error(M(280));var t=e.stateNode;t&&(t=Nl(t),Uo(e.stateNode,e.type,t))}}function wd(e){Yn?Xn?Xn.push(e):Xn=[e]:Yn=e}function Sd(){if(Yn){var e=Yn,t=Xn;if(Xn=Yn=null,iu(e),t)for(e=0;e<t.length;e++)iu(t[e])}}function bd(e,t){return e(t)}function Cd(){}var Ql=!1;function jd(e,t,n){if(Ql)return e(t,n);Ql=!0;try{return bd(e,t,n)}finally{Ql=!1,(Yn!==null||Xn!==null)&&(Cd(),Sd())}}function Vr(e,t){var n=e.stateNode;if(n===null)return null;var r=Nl(n);if(r===null)return null;n=r[t];e:switch(t){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(r=!r.disabled)||(e=e.type,r=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!r;break e;default:e=!1}if(e)return null;if(n&&typeof n!="function")throw Error(M(231,t,typeof n));return n}var Ho=!1;if(Ft)try{var vr={};Object.defineProperty(vr,"passive",{get:function(){Ho=!0}}),window.addEventListener("test",vr,vr),window.removeEventListener("test",vr,vr)}catch{Ho=!1}function Lh(e,t,n,r,i,l,o,a,s){var c=Array.prototype.slice.call(arguments,3);try{t.apply(n,c)}catch(d){this.onError(d)}}var Lr=!1,Gi=null,Ji=!1,Vo=null,Ph={onError:function(e){Lr=!0,Gi=e}};function Ih(e,t,n,r,i,l,o,a,s){Lr=!1,Gi=null,Lh.apply(Ph,arguments)}function Mh(e,t,n,r,i,l,o,a,s){if(Ih.apply(this,arguments),Lr){if(Lr){var c=Gi;Lr=!1,Gi=null}else throw Error(M(198));Ji||(Ji=!0,Vo=c)}}function Tn(e){var t=e,n=e;if(e.alternate)for(;t.return;)t=t.return;else{e=t;do t=e,t.flags&4098&&(n=t.return),e=t.return;while(e)}return t.tag===3?n:null}function Ed(e){if(e.tag===13){var t=e.memoizedState;if(t===null&&(e=e.alternate,e!==null&&(t=e.memoizedState)),t!==null)return t.dehydrated}return null}function lu(e){if(Tn(e)!==e)throw Error(M(188))}function Ah(e){var t=e.alternate;if(!t){if(t=Tn(e),t===null)throw Error(M(188));return t!==e?null:e}for(var n=e,r=t;;){var i=n.return;if(i===null)break;var l=i.alternate;if(l===null){if(r=i.return,r!==null){n=r;continue}break}if(i.child===l.child){for(l=i.child;l;){if(l===n)return lu(i),e;if(l===r)return lu(i),t;l=l.sibling}throw Error(M(188))}if(n.return!==r.return)n=i,r=l;else{for(var o=!1,a=i.child;a;){if(a===n){o=!0,n=i,r=l;break}if(a===r){o=!0,r=i,n=l;break}a=a.sibling}if(!o){for(a=l.child;a;){if(a===n){o=!0,n=l,r=i;break}if(a===r){o=!0,r=l,n=i;break}a=a.sibling}if(!o)throw Error(M(189))}}if(n.alternate!==r)throw Error(M(190))}if(n.tag!==3)throw Error(M(188));return n.stateNode.current===n?e:t}function Nd(e){return e=Ah(e),e!==null?_d(e):null}function _d(e){if(e.tag===5||e.tag===6)return e;for(e=e.child;e!==null;){var t=_d(e);if(t!==null)return t;e=e.sibling}return null}var zd=nt.unstable_scheduleCallback,ou=nt.unstable_cancelCallback,Dh=nt.unstable_shouldYield,Rh=nt.unstable_requestPaint,ge=nt.unstable_now,Fh=nt.unstable_getCurrentPriorityLevel,Wa=nt.unstable_ImmediatePriority,Td=nt.unstable_UserBlockingPriority,Zi=nt.unstable_NormalPriority,Oh=nt.unstable_LowPriority,Ld=nt.unstable_IdlePriority,bl=null,Et=null;function Bh(e){if(Et&&typeof Et.onCommitFiberRoot=="function")try{Et.onCommitFiberRoot(bl,e,void 0,(e.current.flags&128)===128)}catch{}}var yt=Math.clz32?Math.clz32:Hh,$h=Math.log,Uh=Math.LN2;function Hh(e){return e>>>=0,e===0?32:31-($h(e)/Uh|0)|0}var vi=64,yi=4194304;function _r(e){switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return e&4194240;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return e&130023424;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 1073741824;default:return e}}function el(e,t){var n=e.pendingLanes;if(n===0)return 0;var r=0,i=e.suspendedLanes,l=e.pingedLanes,o=n&268435455;if(o!==0){var a=o&~i;a!==0?r=_r(a):(l&=o,l!==0&&(r=_r(l)))}else o=n&~i,o!==0?r=_r(o):l!==0&&(r=_r(l));if(r===0)return 0;if(t!==0&&t!==r&&!(t&i)&&(i=r&-r,l=t&-t,i>=l||i===16&&(l&4194240)!==0))return t;if(r&4&&(r|=n&16),t=e.entangledLanes,t!==0)for(e=e.entanglements,t&=r;0<t;)n=31-yt(t),i=1<<n,r|=e[n],t&=~i;return r}function Vh(e,t){switch(e){case 1:case 2:case 4:return t+250;case 8:case 16:case 32:case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return t+5e3;case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:return-1;case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function Wh(e,t){for(var n=e.suspendedLanes,r=e.pingedLanes,i=e.expirationTimes,l=e.pendingLanes;0<l;){var o=31-yt(l),a=1<<o,s=i[o];s===-1?(!(a&n)||a&r)&&(i[o]=Vh(a,t)):s<=t&&(e.expiredLanes|=a),l&=~a}}function Wo(e){return e=e.pendingLanes&-1073741825,e!==0?e:e&1073741824?1073741824:0}function Pd(){var e=vi;return vi<<=1,!(vi&4194240)&&(vi=64),e}function ql(e){for(var t=[],n=0;31>n;n++)t.push(e);return t}function ai(e,t,n){e.pendingLanes|=t,t!==536870912&&(e.suspendedLanes=0,e.pingedLanes=0),e=e.eventTimes,t=31-yt(t),e[t]=n}function Qh(e,t){var n=e.pendingLanes&~t;e.pendingLanes=t,e.suspendedLanes=0,e.pingedLanes=0,e.expiredLanes&=t,e.mutableReadLanes&=t,e.entangledLanes&=t,t=e.entanglements;var r=e.eventTimes;for(e=e.expirationTimes;0<n;){var i=31-yt(n),l=1<<i;t[i]=0,r[i]=-1,e[i]=-1,n&=~l}}function Qa(e,t){var n=e.entangledLanes|=t;for(e=e.entanglements;n;){var r=31-yt(n),i=1<<r;i&t|e[r]&t&&(e[r]|=t),n&=~i}}var re=0;function Id(e){return e&=-e,1<e?4<e?e&268435455?16:536870912:4:1}var Md,qa,Ad,Dd,Rd,Qo=!1,xi=[],en=null,tn=null,nn=null,Wr=new Map,Qr=new Map,Xt=[],qh="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");function au(e,t){switch(e){case"focusin":case"focusout":en=null;break;case"dragenter":case"dragleave":tn=null;break;case"mouseover":case"mouseout":nn=null;break;case"pointerover":case"pointerout":Wr.delete(t.pointerId);break;case"gotpointercapture":case"lostpointercapture":Qr.delete(t.pointerId)}}function yr(e,t,n,r,i,l){return e===null||e.nativeEvent!==l?(e={blockedOn:t,domEventName:n,eventSystemFlags:r,nativeEvent:l,targetContainers:[i]},t!==null&&(t=ui(t),t!==null&&qa(t)),e):(e.eventSystemFlags|=r,t=e.targetContainers,i!==null&&t.indexOf(i)===-1&&t.push(i),e)}function Kh(e,t,n,r,i){switch(t){case"focusin":return en=yr(en,e,t,n,r,i),!0;case"dragenter":return tn=yr(tn,e,t,n,r,i),!0;case"mouseover":return nn=yr(nn,e,t,n,r,i),!0;case"pointerover":var l=i.pointerId;return Wr.set(l,yr(Wr.get(l)||null,e,t,n,r,i)),!0;case"gotpointercapture":return l=i.pointerId,Qr.set(l,yr(Qr.get(l)||null,e,t,n,r,i)),!0}return!1}function Fd(e){var t=xn(e.target);if(t!==null){var n=Tn(t);if(n!==null){if(t=n.tag,t===13){if(t=Ed(n),t!==null){e.blockedOn=t,Rd(e.priority,function(){Ad(n)});return}}else if(t===3&&n.stateNode.current.memoizedState.isDehydrated){e.blockedOn=n.tag===3?n.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Ri(e){if(e.blockedOn!==null)return!1;for(var t=e.targetContainers;0<t.length;){var n=qo(e.domEventName,e.eventSystemFlags,t[0],e.nativeEvent);if(n===null){n=e.nativeEvent;var r=new n.constructor(n.type,n);$o=r,n.target.dispatchEvent(r),$o=null}else return t=ui(n),t!==null&&qa(t),e.blockedOn=n,!1;t.shift()}return!0}function su(e,t,n){Ri(e)&&n.delete(t)}function Yh(){Qo=!1,en!==null&&Ri(en)&&(en=null),tn!==null&&Ri(tn)&&(tn=null),nn!==null&&Ri(nn)&&(nn=null),Wr.forEach(su),Qr.forEach(su)}function xr(e,t){e.blockedOn===t&&(e.blockedOn=null,Qo||(Qo=!0,nt.unstable_scheduleCallback(nt.unstable_NormalPriority,Yh)))}function qr(e){function t(i){return xr(i,e)}if(0<xi.length){xr(xi[0],e);for(var n=1;n<xi.length;n++){var r=xi[n];r.blockedOn===e&&(r.blockedOn=null)}}for(en!==null&&xr(en,e),tn!==null&&xr(tn,e),nn!==null&&xr(nn,e),Wr.forEach(t),Qr.forEach(t),n=0;n<Xt.length;n++)r=Xt[n],r.blockedOn===e&&(r.blockedOn=null);for(;0<Xt.length&&(n=Xt[0],n.blockedOn===null);)Fd(n),n.blockedOn===null&&Xt.shift()}var Gn=Ut.ReactCurrentBatchConfig,tl=!0;function Xh(e,t,n,r){var i=re,l=Gn.transition;Gn.transition=null;try{re=1,Ka(e,t,n,r)}finally{re=i,Gn.transition=l}}function Gh(e,t,n,r){var i=re,l=Gn.transition;Gn.transition=null;try{re=4,Ka(e,t,n,r)}finally{re=i,Gn.transition=l}}function Ka(e,t,n,r){if(tl){var i=qo(e,t,n,r);if(i===null)ro(e,t,r,nl,n),au(e,r);else if(Kh(i,e,t,n,r))r.stopPropagation();else if(au(e,r),t&4&&-1<qh.indexOf(e)){for(;i!==null;){var l=ui(i);if(l!==null&&Md(l),l=qo(e,t,n,r),l===null&&ro(e,t,r,nl,n),l===i)break;i=l}i!==null&&r.stopPropagation()}else ro(e,t,r,null,n)}}var nl=null;function qo(e,t,n,r){if(nl=null,e=Va(r),e=xn(e),e!==null)if(t=Tn(e),t===null)e=null;else if(n=t.tag,n===13){if(e=Ed(t),e!==null)return e;e=null}else if(n===3){if(t.stateNode.current.memoizedState.isDehydrated)return t.tag===3?t.stateNode.containerInfo:null;e=null}else t!==e&&(e=null);return nl=e,null}function Od(e){switch(e){case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"resize":case"seeked":case"submit":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 1;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"scroll":case"toggle":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 4;case"message":switch(Fh()){case Wa:return 1;case Td:return 4;case Zi:case Oh:return 16;case Ld:return 536870912;default:return 16}default:return 16}}var Jt=null,Ya=null,Fi=null;function Bd(){if(Fi)return Fi;var e,t=Ya,n=t.length,r,i="value"in Jt?Jt.value:Jt.textContent,l=i.length;for(e=0;e<n&&t[e]===i[e];e++);var o=n-e;for(r=1;r<=o&&t[n-r]===i[l-r];r++);return Fi=i.slice(e,1<r?1-r:void 0)}function Oi(e){var t=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&t===13&&(e=13)):e=t,e===10&&(e=13),32<=e||e===13?e:0}function ki(){return!0}function uu(){return!1}function it(e){function t(n,r,i,l,o){this._reactName=n,this._targetInst=i,this.type=r,this.nativeEvent=l,this.target=o,this.currentTarget=null;for(var a in e)e.hasOwnProperty(a)&&(n=e[a],this[a]=n?n(l):l[a]);return this.isDefaultPrevented=(l.defaultPrevented!=null?l.defaultPrevented:l.returnValue===!1)?ki:uu,this.isPropagationStopped=uu,this}return he(t.prototype,{preventDefault:function(){this.defaultPrevented=!0;var n=this.nativeEvent;n&&(n.preventDefault?n.preventDefault():typeof n.returnValue!="unknown"&&(n.returnValue=!1),this.isDefaultPrevented=ki)},stopPropagation:function(){var n=this.nativeEvent;n&&(n.stopPropagation?n.stopPropagation():typeof n.cancelBubble!="unknown"&&(n.cancelBubble=!0),this.isPropagationStopped=ki)},persist:function(){},isPersistent:ki}),t}var cr={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Xa=it(cr),si=he({},cr,{view:0,detail:0}),Jh=it(si),Kl,Yl,kr,Cl=he({},si,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:Ga,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==kr&&(kr&&e.type==="mousemove"?(Kl=e.screenX-kr.screenX,Yl=e.screenY-kr.screenY):Yl=Kl=0,kr=e),Kl)},movementY:function(e){return"movementY"in e?e.movementY:Yl}}),cu=it(Cl),Zh=he({},Cl,{dataTransfer:0}),em=it(Zh),tm=he({},si,{relatedTarget:0}),Xl=it(tm),nm=he({},cr,{animationName:0,elapsedTime:0,pseudoElement:0}),rm=it(nm),im=he({},cr,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),lm=it(im),om=he({},cr,{data:0}),du=it(om),am={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},sm={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},um={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function cm(e){var t=this.nativeEvent;return t.getModifierState?t.getModifierState(e):(e=um[e])?!!t[e]:!1}function Ga(){return cm}var dm=he({},si,{key:function(e){if(e.key){var t=am[e.key]||e.key;if(t!=="Unidentified")return t}return e.type==="keypress"?(e=Oi(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?sm[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:Ga,charCode:function(e){return e.type==="keypress"?Oi(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Oi(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),fm=it(dm),pm=he({},Cl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),fu=it(pm),hm=he({},si,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:Ga}),mm=it(hm),gm=he({},cr,{propertyName:0,elapsedTime:0,pseudoElement:0}),vm=it(gm),ym=he({},Cl,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),xm=it(ym),km=[9,13,27,32],Ja=Ft&&"CompositionEvent"in window,Pr=null;Ft&&"documentMode"in document&&(Pr=document.documentMode);var wm=Ft&&"TextEvent"in window&&!Pr,$d=Ft&&(!Ja||Pr&&8<Pr&&11>=Pr),pu=" ",hu=!1;function Ud(e,t){switch(e){case"keyup":return km.indexOf(t.keyCode)!==-1;case"keydown":return t.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Hd(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var Fn=!1;function Sm(e,t){switch(e){case"compositionend":return Hd(t);case"keypress":return t.which!==32?null:(hu=!0,pu);case"textInput":return e=t.data,e===pu&&hu?null:e;default:return null}}function bm(e,t){if(Fn)return e==="compositionend"||!Ja&&Ud(e,t)?(e=Bd(),Fi=Ya=Jt=null,Fn=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(t.ctrlKey||t.altKey||t.metaKey)||t.ctrlKey&&t.altKey){if(t.char&&1<t.char.length)return t.char;if(t.which)return String.fromCharCode(t.which)}return null;case"compositionend":return $d&&t.locale!=="ko"?null:t.data;default:return null}}var Cm={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function mu(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t==="input"?!!Cm[e.type]:t==="textarea"}function Vd(e,t,n,r){wd(r),t=rl(t,"onChange"),0<t.length&&(n=new Xa("onChange","change",null,n,r),e.push({event:n,listeners:t}))}var Ir=null,Kr=null;function jm(e){tf(e,0)}function jl(e){var t=$n(e);if(hd(t))return e}function Em(e,t){if(e==="change")return t}var Wd=!1;if(Ft){var Gl;if(Ft){var Jl="oninput"in document;if(!Jl){var gu=document.createElement("div");gu.setAttribute("oninput","return;"),Jl=typeof gu.oninput=="function"}Gl=Jl}else Gl=!1;Wd=Gl&&(!document.documentMode||9<document.documentMode)}function vu(){Ir&&(Ir.detachEvent("onpropertychange",Qd),Kr=Ir=null)}function Qd(e){if(e.propertyName==="value"&&jl(Kr)){var t=[];Vd(t,Kr,e,Va(e)),jd(jm,t)}}function Nm(e,t,n){e==="focusin"?(vu(),Ir=t,Kr=n,Ir.attachEvent("onpropertychange",Qd)):e==="focusout"&&vu()}function _m(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return jl(Kr)}function zm(e,t){if(e==="click")return jl(t)}function Tm(e,t){if(e==="input"||e==="change")return jl(t)}function Lm(e,t){return e===t&&(e!==0||1/e===1/t)||e!==e&&t!==t}var kt=typeof Object.is=="function"?Object.is:Lm;function Yr(e,t){if(kt(e,t))return!0;if(typeof e!="object"||e===null||typeof t!="object"||t===null)return!1;var n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(r=0;r<n.length;r++){var i=n[r];if(!zo.call(t,i)||!kt(e[i],t[i]))return!1}return!0}function yu(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function xu(e,t){var n=yu(e);e=0;for(var r;n;){if(n.nodeType===3){if(r=e+n.textContent.length,e<=t&&r>=t)return{node:n,offset:t-e};e=r}e:{for(;n;){if(n.nextSibling){n=n.nextSibling;break e}n=n.parentNode}n=void 0}n=yu(n)}}function qd(e,t){return e&&t?e===t?!0:e&&e.nodeType===3?!1:t&&t.nodeType===3?qd(e,t.parentNode):"contains"in e?e.contains(t):e.compareDocumentPosition?!!(e.compareDocumentPosition(t)&16):!1:!1}function Kd(){for(var e=window,t=Xi();t instanceof e.HTMLIFrameElement;){try{var n=typeof t.contentWindow.location.href=="string"}catch{n=!1}if(n)e=t.contentWindow;else break;t=Xi(e.document)}return t}function Za(e){var t=e&&e.nodeName&&e.nodeName.toLowerCase();return t&&(t==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||t==="textarea"||e.contentEditable==="true")}function Pm(e){var t=Kd(),n=e.focusedElem,r=e.selectionRange;if(t!==n&&n&&n.ownerDocument&&qd(n.ownerDocument.documentElement,n)){if(r!==null&&Za(n)){if(t=r.start,e=r.end,e===void 0&&(e=t),"selectionStart"in n)n.selectionStart=t,n.selectionEnd=Math.min(e,n.value.length);else if(e=(t=n.ownerDocument||document)&&t.defaultView||window,e.getSelection){e=e.getSelection();var i=n.textContent.length,l=Math.min(r.start,i);r=r.end===void 0?l:Math.min(r.end,i),!e.extend&&l>r&&(i=r,r=l,l=i),i=xu(n,l);var o=xu(n,r);i&&o&&(e.rangeCount!==1||e.anchorNode!==i.node||e.anchorOffset!==i.offset||e.focusNode!==o.node||e.focusOffset!==o.offset)&&(t=t.createRange(),t.setStart(i.node,i.offset),e.removeAllRanges(),l>r?(e.addRange(t),e.extend(o.node,o.offset)):(t.setEnd(o.node,o.offset),e.addRange(t)))}}for(t=[],e=n;e=e.parentNode;)e.nodeType===1&&t.push({element:e,left:e.scrollLeft,top:e.scrollTop});for(typeof n.focus=="function"&&n.focus(),n=0;n<t.length;n++)e=t[n],e.element.scrollLeft=e.left,e.element.scrollTop=e.top}}var Im=Ft&&"documentMode"in document&&11>=document.documentMode,On=null,Ko=null,Mr=null,Yo=!1;function ku(e,t,n){var r=n.window===n?n.document:n.nodeType===9?n:n.ownerDocument;Yo||On==null||On!==Xi(r)||(r=On,"selectionStart"in r&&Za(r)?r={start:r.selectionStart,end:r.selectionEnd}:(r=(r.ownerDocument&&r.ownerDocument.defaultView||window).getSelection(),r={anchorNode:r.anchorNode,anchorOffset:r.anchorOffset,focusNode:r.focusNode,focusOffset:r.focusOffset}),Mr&&Yr(Mr,r)||(Mr=r,r=rl(Ko,"onSelect"),0<r.length&&(t=new Xa("onSelect","select",null,t,n),e.push({event:t,listeners:r}),t.target=On)))}function wi(e,t){var n={};return n[e.toLowerCase()]=t.toLowerCase(),n["Webkit"+e]="webkit"+t,n["Moz"+e]="moz"+t,n}var Bn={animationend:wi("Animation","AnimationEnd"),animationiteration:wi("Animation","AnimationIteration"),animationstart:wi("Animation","AnimationStart"),transitionend:wi("Transition","TransitionEnd")},Zl={},Yd={};Ft&&(Yd=document.createElement("div").style,"AnimationEvent"in window||(delete Bn.animationend.animation,delete Bn.animationiteration.animation,delete Bn.animationstart.animation),"TransitionEvent"in window||delete Bn.transitionend.transition);function El(e){if(Zl[e])return Zl[e];if(!Bn[e])return e;var t=Bn[e],n;for(n in t)if(t.hasOwnProperty(n)&&n in Yd)return Zl[e]=t[n];return e}var Xd=El("animationend"),Gd=El("animationiteration"),Jd=El("animationstart"),Zd=El("transitionend"),ef=new Map,wu="abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");function dn(e,t){ef.set(e,t),zn(t,[e])}for(var eo=0;eo<wu.length;eo++){var to=wu[eo],Mm=to.toLowerCase(),Am=to[0].toUpperCase()+to.slice(1);dn(Mm,"on"+Am)}dn(Xd,"onAnimationEnd");dn(Gd,"onAnimationIteration");dn(Jd,"onAnimationStart");dn("dblclick","onDoubleClick");dn("focusin","onFocus");dn("focusout","onBlur");dn(Zd,"onTransitionEnd");nr("onMouseEnter",["mouseout","mouseover"]);nr("onMouseLeave",["mouseout","mouseover"]);nr("onPointerEnter",["pointerout","pointerover"]);nr("onPointerLeave",["pointerout","pointerover"]);zn("onChange","change click focusin focusout input keydown keyup selectionchange".split(" "));zn("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));zn("onBeforeInput",["compositionend","keypress","textInput","paste"]);zn("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" "));zn("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" "));zn("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var zr="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),Dm=new Set("cancel close invalid load scroll toggle".split(" ").concat(zr));function Su(e,t,n){var r=e.type||"unknown-event";e.currentTarget=n,Mh(r,t,void 0,e),e.currentTarget=null}function tf(e,t){t=(t&4)!==0;for(var n=0;n<e.length;n++){var r=e[n],i=r.event;r=r.listeners;e:{var l=void 0;if(t)for(var o=r.length-1;0<=o;o--){var a=r[o],s=a.instance,c=a.currentTarget;if(a=a.listener,s!==l&&i.isPropagationStopped())break e;Su(i,a,c),l=s}else for(o=0;o<r.length;o++){if(a=r[o],s=a.instance,c=a.currentTarget,a=a.listener,s!==l&&i.isPropagationStopped())break e;Su(i,a,c),l=s}}}if(Ji)throw e=Vo,Ji=!1,Vo=null,e}function ue(e,t){var n=t[ea];n===void 0&&(n=t[ea]=new Set);var r=e+"__bubble";n.has(r)||(nf(t,e,2,!1),n.add(r))}function no(e,t,n){var r=0;t&&(r|=4),nf(n,e,r,t)}var Si="_reactListening"+Math.random().toString(36).slice(2);function Xr(e){if(!e[Si]){e[Si]=!0,ud.forEach(function(n){n!=="selectionchange"&&(Dm.has(n)||no(n,!1,e),no(n,!0,e))});var t=e.nodeType===9?e:e.ownerDocument;t===null||t[Si]||(t[Si]=!0,no("selectionchange",!1,t))}}function nf(e,t,n,r){switch(Od(t)){case 1:var i=Xh;break;case 4:i=Gh;break;default:i=Ka}n=i.bind(null,t,n,e),i=void 0,!Ho||t!=="touchstart"&&t!=="touchmove"&&t!=="wheel"||(i=!0),r?i!==void 0?e.addEventListener(t,n,{capture:!0,passive:i}):e.addEventListener(t,n,!0):i!==void 0?e.addEventListener(t,n,{passive:i}):e.addEventListener(t,n,!1)}function ro(e,t,n,r,i){var l=r;if(!(t&1)&&!(t&2)&&r!==null)e:for(;;){if(r===null)return;var o=r.tag;if(o===3||o===4){var a=r.stateNode.containerInfo;if(a===i||a.nodeType===8&&a.parentNode===i)break;if(o===4)for(o=r.return;o!==null;){var s=o.tag;if((s===3||s===4)&&(s=o.stateNode.containerInfo,s===i||s.nodeType===8&&s.parentNode===i))return;o=o.return}for(;a!==null;){if(o=xn(a),o===null)return;if(s=o.tag,s===5||s===6){r=l=o;continue e}a=a.parentNode}}r=r.return}jd(function(){var c=l,d=Va(n),f=[];e:{var m=ef.get(e);if(m!==void 0){var p=Xa,w=e;switch(e){case"keypress":if(Oi(n)===0)break e;case"keydown":case"keyup":p=fm;break;case"focusin":w="focus",p=Xl;break;case"focusout":w="blur",p=Xl;break;case"beforeblur":case"afterblur":p=Xl;break;case"click":if(n.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":p=cu;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":p=em;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":p=mm;break;case Xd:case Gd:case Jd:p=rm;break;case Zd:p=vm;break;case"scroll":p=Jh;break;case"wheel":p=xm;break;case"copy":case"cut":case"paste":p=lm;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":p=fu}var S=(t&4)!==0,I=!S&&e==="scroll",h=S?m!==null?m+"Capture":null:m;S=[];for(var v=c,y;v!==null;){y=v;var C=y.stateNode;if(y.tag===5&&C!==null&&(y=C,h!==null&&(C=Vr(v,h),C!=null&&S.push(Gr(v,C,y)))),I)break;v=v.return}0<S.length&&(m=new p(m,w,null,n,d),f.push({event:m,listeners:S}))}}if(!(t&7)){e:{if(m=e==="mouseover"||e==="pointerover",p=e==="mouseout"||e==="pointerout",m&&n!==$o&&(w=n.relatedTarget||n.fromElement)&&(xn(w)||w[Ot]))break e;if((p||m)&&(m=d.window===d?d:(m=d.ownerDocument)?m.defaultView||m.parentWindow:window,p?(w=n.relatedTarget||n.toElement,p=c,w=w?xn(w):null,w!==null&&(I=Tn(w),w!==I||w.tag!==5&&w.tag!==6)&&(w=null)):(p=null,w=c),p!==w)){if(S=cu,C="onMouseLeave",h="onMouseEnter",v="mouse",(e==="pointerout"||e==="pointerover")&&(S=fu,C="onPointerLeave",h="onPointerEnter",v="pointer"),I=p==null?m:$n(p),y=w==null?m:$n(w),m=new S(C,v+"leave",p,n,d),m.target=I,m.relatedTarget=y,C=null,xn(d)===c&&(S=new S(h,v+"enter",w,n,d),S.target=y,S.relatedTarget=I,C=S),I=C,p&&w)t:{for(S=p,h=w,v=0,y=S;y;y=In(y))v++;for(y=0,C=h;C;C=In(C))y++;for(;0<v-y;)S=In(S),v--;for(;0<y-v;)h=In(h),y--;for(;v--;){if(S===h||h!==null&&S===h.alternate)break t;S=In(S),h=In(h)}S=null}else S=null;p!==null&&bu(f,m,p,S,!1),w!==null&&I!==null&&bu(f,I,w,S,!0)}}e:{if(m=c?$n(c):window,p=m.nodeName&&m.nodeName.toLowerCase(),p==="select"||p==="input"&&m.type==="file")var N=Em;else if(mu(m))if(Wd)N=Tm;else{N=_m;var k=Nm}else(p=m.nodeName)&&p.toLowerCase()==="input"&&(m.type==="checkbox"||m.type==="radio")&&(N=zm);if(N&&(N=N(e,c))){Vd(f,N,n,d);break e}k&&k(e,m,c),e==="focusout"&&(k=m._wrapperState)&&k.controlled&&m.type==="number"&&Do(m,"number",m.value)}switch(k=c?$n(c):window,e){case"focusin":(mu(k)||k.contentEditable==="true")&&(On=k,Ko=c,Mr=null);break;case"focusout":Mr=Ko=On=null;break;case"mousedown":Yo=!0;break;case"contextmenu":case"mouseup":case"dragend":Yo=!1,ku(f,n,d);break;case"selectionchange":if(Im)break;case"keydown":case"keyup":ku(f,n,d)}var j;if(Ja)e:{switch(e){case"compositionstart":var _="onCompositionStart";break e;case"compositionend":_="onCompositionEnd";break e;case"compositionupdate":_="onCompositionUpdate";break e}_=void 0}else Fn?Ud(e,n)&&(_="onCompositionEnd"):e==="keydown"&&n.keyCode===229&&(_="onCompositionStart");_&&($d&&n.locale!=="ko"&&(Fn||_!=="onCompositionStart"?_==="onCompositionEnd"&&Fn&&(j=Bd()):(Jt=d,Ya="value"in Jt?Jt.value:Jt.textContent,Fn=!0)),k=rl(c,_),0<k.length&&(_=new du(_,e,null,n,d),f.push({event:_,listeners:k}),j?_.data=j:(j=Hd(n),j!==null&&(_.data=j)))),(j=wm?Sm(e,n):bm(e,n))&&(c=rl(c,"onBeforeInput"),0<c.length&&(d=new du("onBeforeInput","beforeinput",null,n,d),f.push({event:d,listeners:c}),d.data=j))}tf(f,t)})}function Gr(e,t,n){return{instance:e,listener:t,currentTarget:n}}function rl(e,t){for(var n=t+"Capture",r=[];e!==null;){var i=e,l=i.stateNode;i.tag===5&&l!==null&&(i=l,l=Vr(e,n),l!=null&&r.unshift(Gr(e,l,i)),l=Vr(e,t),l!=null&&r.push(Gr(e,l,i))),e=e.return}return r}function In(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5);return e||null}function bu(e,t,n,r,i){for(var l=t._reactName,o=[];n!==null&&n!==r;){var a=n,s=a.alternate,c=a.stateNode;if(s!==null&&s===r)break;a.tag===5&&c!==null&&(a=c,i?(s=Vr(n,l),s!=null&&o.unshift(Gr(n,s,a))):i||(s=Vr(n,l),s!=null&&o.push(Gr(n,s,a)))),n=n.return}o.length!==0&&e.push({event:t,listeners:o})}var Rm=/\r\n?/g,Fm=/\u0000|\uFFFD/g;function Cu(e){return(typeof e=="string"?e:""+e).replace(Rm,`
`).replace(Fm,"")}function bi(e,t,n){if(t=Cu(t),Cu(e)!==t&&n)throw Error(M(425))}function il(){}var Xo=null,Go=null;function Jo(e,t){return e==="textarea"||e==="noscript"||typeof t.children=="string"||typeof t.children=="number"||typeof t.dangerouslySetInnerHTML=="object"&&t.dangerouslySetInnerHTML!==null&&t.dangerouslySetInnerHTML.__html!=null}var Zo=typeof setTimeout=="function"?setTimeout:void 0,Om=typeof clearTimeout=="function"?clearTimeout:void 0,ju=typeof Promise=="function"?Promise:void 0,Bm=typeof queueMicrotask=="function"?queueMicrotask:typeof ju<"u"?function(e){return ju.resolve(null).then(e).catch($m)}:Zo;function $m(e){setTimeout(function(){throw e})}function io(e,t){var n=t,r=0;do{var i=n.nextSibling;if(e.removeChild(n),i&&i.nodeType===8)if(n=i.data,n==="/$"){if(r===0){e.removeChild(i),qr(t);return}r--}else n!=="$"&&n!=="$?"&&n!=="$!"||r++;n=i}while(n);qr(t)}function rn(e){for(;e!=null;e=e.nextSibling){var t=e.nodeType;if(t===1||t===3)break;if(t===8){if(t=e.data,t==="$"||t==="$!"||t==="$?")break;if(t==="/$")return null}}return e}function Eu(e){e=e.previousSibling;for(var t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="$"||n==="$!"||n==="$?"){if(t===0)return e;t--}else n==="/$"&&t++}e=e.previousSibling}return null}var dr=Math.random().toString(36).slice(2),Ct="__reactFiber$"+dr,Jr="__reactProps$"+dr,Ot="__reactContainer$"+dr,ea="__reactEvents$"+dr,Um="__reactListeners$"+dr,Hm="__reactHandles$"+dr;function xn(e){var t=e[Ct];if(t)return t;for(var n=e.parentNode;n;){if(t=n[Ot]||n[Ct]){if(n=t.alternate,t.child!==null||n!==null&&n.child!==null)for(e=Eu(e);e!==null;){if(n=e[Ct])return n;e=Eu(e)}return t}e=n,n=e.parentNode}return null}function ui(e){return e=e[Ct]||e[Ot],!e||e.tag!==5&&e.tag!==6&&e.tag!==13&&e.tag!==3?null:e}function $n(e){if(e.tag===5||e.tag===6)return e.stateNode;throw Error(M(33))}function Nl(e){return e[Jr]||null}var ta=[],Un=-1;function fn(e){return{current:e}}function ce(e){0>Un||(e.current=ta[Un],ta[Un]=null,Un--)}function ae(e,t){Un++,ta[Un]=e.current,e.current=t}var cn={},Pe=fn(cn),Ve=fn(!1),Cn=cn;function rr(e,t){var n=e.type.contextTypes;if(!n)return cn;var r=e.stateNode;if(r&&r.__reactInternalMemoizedUnmaskedChildContext===t)return r.__reactInternalMemoizedMaskedChildContext;var i={},l;for(l in n)i[l]=t[l];return r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=t,e.__reactInternalMemoizedMaskedChildContext=i),i}function We(e){return e=e.childContextTypes,e!=null}function ll(){ce(Ve),ce(Pe)}function Nu(e,t,n){if(Pe.current!==cn)throw Error(M(168));ae(Pe,t),ae(Ve,n)}function rf(e,t,n){var r=e.stateNode;if(t=t.childContextTypes,typeof r.getChildContext!="function")return n;r=r.getChildContext();for(var i in r)if(!(i in t))throw Error(M(108,Nh(e)||"Unknown",i));return he({},n,r)}function ol(e){return e=(e=e.stateNode)&&e.__reactInternalMemoizedMergedChildContext||cn,Cn=Pe.current,ae(Pe,e),ae(Ve,Ve.current),!0}function _u(e,t,n){var r=e.stateNode;if(!r)throw Error(M(169));n?(e=rf(e,t,Cn),r.__reactInternalMemoizedMergedChildContext=e,ce(Ve),ce(Pe),ae(Pe,e)):ce(Ve),ae(Ve,n)}var Mt=null,_l=!1,lo=!1;function lf(e){Mt===null?Mt=[e]:Mt.push(e)}function Vm(e){_l=!0,lf(e)}function pn(){if(!lo&&Mt!==null){lo=!0;var e=0,t=re;try{var n=Mt;for(re=1;e<n.length;e++){var r=n[e];do r=r(!0);while(r!==null)}Mt=null,_l=!1}catch(i){throw Mt!==null&&(Mt=Mt.slice(e+1)),zd(Wa,pn),i}finally{re=t,lo=!1}}return null}var Hn=[],Vn=0,al=null,sl=0,ot=[],at=0,jn=null,At=1,Dt="";function gn(e,t){Hn[Vn++]=sl,Hn[Vn++]=al,al=e,sl=t}function of(e,t,n){ot[at++]=At,ot[at++]=Dt,ot[at++]=jn,jn=e;var r=At;e=Dt;var i=32-yt(r)-1;r&=~(1<<i),n+=1;var l=32-yt(t)+i;if(30<l){var o=i-i%5;l=(r&(1<<o)-1).toString(32),r>>=o,i-=o,At=1<<32-yt(t)+i|n<<i|r,Dt=l+e}else At=1<<l|n<<i|r,Dt=e}function es(e){e.return!==null&&(gn(e,1),of(e,1,0))}function ts(e){for(;e===al;)al=Hn[--Vn],Hn[Vn]=null,sl=Hn[--Vn],Hn[Vn]=null;for(;e===jn;)jn=ot[--at],ot[at]=null,Dt=ot[--at],ot[at]=null,At=ot[--at],ot[at]=null}var tt=null,Ze=null,de=!1,vt=null;function af(e,t){var n=ut(5,null,null,0);n.elementType="DELETED",n.stateNode=t,n.return=e,t=e.deletions,t===null?(e.deletions=[n],e.flags|=16):t.push(n)}function zu(e,t){switch(e.tag){case 5:var n=e.type;return t=t.nodeType!==1||n.toLowerCase()!==t.nodeName.toLowerCase()?null:t,t!==null?(e.stateNode=t,tt=e,Ze=rn(t.firstChild),!0):!1;case 6:return t=e.pendingProps===""||t.nodeType!==3?null:t,t!==null?(e.stateNode=t,tt=e,Ze=null,!0):!1;case 13:return t=t.nodeType!==8?null:t,t!==null?(n=jn!==null?{id:At,overflow:Dt}:null,e.memoizedState={dehydrated:t,treeContext:n,retryLane:1073741824},n=ut(18,null,null,0),n.stateNode=t,n.return=e,e.child=n,tt=e,Ze=null,!0):!1;default:return!1}}function na(e){return(e.mode&1)!==0&&(e.flags&128)===0}function ra(e){if(de){var t=Ze;if(t){var n=t;if(!zu(e,t)){if(na(e))throw Error(M(418));t=rn(n.nextSibling);var r=tt;t&&zu(e,t)?af(r,n):(e.flags=e.flags&-4097|2,de=!1,tt=e)}}else{if(na(e))throw Error(M(418));e.flags=e.flags&-4097|2,de=!1,tt=e}}}function Tu(e){for(e=e.return;e!==null&&e.tag!==5&&e.tag!==3&&e.tag!==13;)e=e.return;tt=e}function Ci(e){if(e!==tt)return!1;if(!de)return Tu(e),de=!0,!1;var t;if((t=e.tag!==3)&&!(t=e.tag!==5)&&(t=e.type,t=t!=="head"&&t!=="body"&&!Jo(e.type,e.memoizedProps)),t&&(t=Ze)){if(na(e))throw sf(),Error(M(418));for(;t;)af(e,t),t=rn(t.nextSibling)}if(Tu(e),e.tag===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(M(317));e:{for(e=e.nextSibling,t=0;e;){if(e.nodeType===8){var n=e.data;if(n==="/$"){if(t===0){Ze=rn(e.nextSibling);break e}t--}else n!=="$"&&n!=="$!"&&n!=="$?"||t++}e=e.nextSibling}Ze=null}}else Ze=tt?rn(e.stateNode.nextSibling):null;return!0}function sf(){for(var e=Ze;e;)e=rn(e.nextSibling)}function ir(){Ze=tt=null,de=!1}function ns(e){vt===null?vt=[e]:vt.push(e)}var Wm=Ut.ReactCurrentBatchConfig;function wr(e,t,n){if(e=n.ref,e!==null&&typeof e!="function"&&typeof e!="object"){if(n._owner){if(n=n._owner,n){if(n.tag!==1)throw Error(M(309));var r=n.stateNode}if(!r)throw Error(M(147,e));var i=r,l=""+e;return t!==null&&t.ref!==null&&typeof t.ref=="function"&&t.ref._stringRef===l?t.ref:(t=function(o){var a=i.refs;o===null?delete a[l]:a[l]=o},t._stringRef=l,t)}if(typeof e!="string")throw Error(M(284));if(!n._owner)throw Error(M(290,e))}return e}function ji(e,t){throw e=Object.prototype.toString.call(t),Error(M(31,e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e))}function Lu(e){var t=e._init;return t(e._payload)}function uf(e){function t(h,v){if(e){var y=h.deletions;y===null?(h.deletions=[v],h.flags|=16):y.push(v)}}function n(h,v){if(!e)return null;for(;v!==null;)t(h,v),v=v.sibling;return null}function r(h,v){for(h=new Map;v!==null;)v.key!==null?h.set(v.key,v):h.set(v.index,v),v=v.sibling;return h}function i(h,v){return h=sn(h,v),h.index=0,h.sibling=null,h}function l(h,v,y){return h.index=y,e?(y=h.alternate,y!==null?(y=y.index,y<v?(h.flags|=2,v):y):(h.flags|=2,v)):(h.flags|=1048576,v)}function o(h){return e&&h.alternate===null&&(h.flags|=2),h}function a(h,v,y,C){return v===null||v.tag!==6?(v=po(y,h.mode,C),v.return=h,v):(v=i(v,y),v.return=h,v)}function s(h,v,y,C){var N=y.type;return N===Rn?d(h,v,y.props.children,C,y.key):v!==null&&(v.elementType===N||typeof N=="object"&&N!==null&&N.$$typeof===Kt&&Lu(N)===v.type)?(C=i(v,y.props),C.ref=wr(h,v,y),C.return=h,C):(C=Qi(y.type,y.key,y.props,null,h.mode,C),C.ref=wr(h,v,y),C.return=h,C)}function c(h,v,y,C){return v===null||v.tag!==4||v.stateNode.containerInfo!==y.containerInfo||v.stateNode.implementation!==y.implementation?(v=ho(y,h.mode,C),v.return=h,v):(v=i(v,y.children||[]),v.return=h,v)}function d(h,v,y,C,N){return v===null||v.tag!==7?(v=bn(y,h.mode,C,N),v.return=h,v):(v=i(v,y),v.return=h,v)}function f(h,v,y){if(typeof v=="string"&&v!==""||typeof v=="number")return v=po(""+v,h.mode,y),v.return=h,v;if(typeof v=="object"&&v!==null){switch(v.$$typeof){case hi:return y=Qi(v.type,v.key,v.props,null,h.mode,y),y.ref=wr(h,null,v),y.return=h,y;case Dn:return v=ho(v,h.mode,y),v.return=h,v;case Kt:var C=v._init;return f(h,C(v._payload),y)}if(Nr(v)||gr(v))return v=bn(v,h.mode,y,null),v.return=h,v;ji(h,v)}return null}function m(h,v,y,C){var N=v!==null?v.key:null;if(typeof y=="string"&&y!==""||typeof y=="number")return N!==null?null:a(h,v,""+y,C);if(typeof y=="object"&&y!==null){switch(y.$$typeof){case hi:return y.key===N?s(h,v,y,C):null;case Dn:return y.key===N?c(h,v,y,C):null;case Kt:return N=y._init,m(h,v,N(y._payload),C)}if(Nr(y)||gr(y))return N!==null?null:d(h,v,y,C,null);ji(h,y)}return null}function p(h,v,y,C,N){if(typeof C=="string"&&C!==""||typeof C=="number")return h=h.get(y)||null,a(v,h,""+C,N);if(typeof C=="object"&&C!==null){switch(C.$$typeof){case hi:return h=h.get(C.key===null?y:C.key)||null,s(v,h,C,N);case Dn:return h=h.get(C.key===null?y:C.key)||null,c(v,h,C,N);case Kt:var k=C._init;return p(h,v,y,k(C._payload),N)}if(Nr(C)||gr(C))return h=h.get(y)||null,d(v,h,C,N,null);ji(v,C)}return null}function w(h,v,y,C){for(var N=null,k=null,j=v,_=v=0,R=null;j!==null&&_<y.length;_++){j.index>_?(R=j,j=null):R=j.sibling;var P=m(h,j,y[_],C);if(P===null){j===null&&(j=R);break}e&&j&&P.alternate===null&&t(h,j),v=l(P,v,_),k===null?N=P:k.sibling=P,k=P,j=R}if(_===y.length)return n(h,j),de&&gn(h,_),N;if(j===null){for(;_<y.length;_++)j=f(h,y[_],C),j!==null&&(v=l(j,v,_),k===null?N=j:k.sibling=j,k=j);return de&&gn(h,_),N}for(j=r(h,j);_<y.length;_++)R=p(j,h,_,y[_],C),R!==null&&(e&&R.alternate!==null&&j.delete(R.key===null?_:R.key),v=l(R,v,_),k===null?N=R:k.sibling=R,k=R);return e&&j.forEach(function(z){return t(h,z)}),de&&gn(h,_),N}function S(h,v,y,C){var N=gr(y);if(typeof N!="function")throw Error(M(150));if(y=N.call(y),y==null)throw Error(M(151));for(var k=N=null,j=v,_=v=0,R=null,P=y.next();j!==null&&!P.done;_++,P=y.next()){j.index>_?(R=j,j=null):R=j.sibling;var z=m(h,j,P.value,C);if(z===null){j===null&&(j=R);break}e&&j&&z.alternate===null&&t(h,j),v=l(z,v,_),k===null?N=z:k.sibling=z,k=z,j=R}if(P.done)return n(h,j),de&&gn(h,_),N;if(j===null){for(;!P.done;_++,P=y.next())P=f(h,P.value,C),P!==null&&(v=l(P,v,_),k===null?N=P:k.sibling=P,k=P);return de&&gn(h,_),N}for(j=r(h,j);!P.done;_++,P=y.next())P=p(j,h,_,P.value,C),P!==null&&(e&&P.alternate!==null&&j.delete(P.key===null?_:P.key),v=l(P,v,_),k===null?N=P:k.sibling=P,k=P);return e&&j.forEach(function(D){return t(h,D)}),de&&gn(h,_),N}function I(h,v,y,C){if(typeof y=="object"&&y!==null&&y.type===Rn&&y.key===null&&(y=y.props.children),typeof y=="object"&&y!==null){switch(y.$$typeof){case hi:e:{for(var N=y.key,k=v;k!==null;){if(k.key===N){if(N=y.type,N===Rn){if(k.tag===7){n(h,k.sibling),v=i(k,y.props.children),v.return=h,h=v;break e}}else if(k.elementType===N||typeof N=="object"&&N!==null&&N.$$typeof===Kt&&Lu(N)===k.type){n(h,k.sibling),v=i(k,y.props),v.ref=wr(h,k,y),v.return=h,h=v;break e}n(h,k);break}else t(h,k);k=k.sibling}y.type===Rn?(v=bn(y.props.children,h.mode,C,y.key),v.return=h,h=v):(C=Qi(y.type,y.key,y.props,null,h.mode,C),C.ref=wr(h,v,y),C.return=h,h=C)}return o(h);case Dn:e:{for(k=y.key;v!==null;){if(v.key===k)if(v.tag===4&&v.stateNode.containerInfo===y.containerInfo&&v.stateNode.implementation===y.implementation){n(h,v.sibling),v=i(v,y.children||[]),v.return=h,h=v;break e}else{n(h,v);break}else t(h,v);v=v.sibling}v=ho(y,h.mode,C),v.return=h,h=v}return o(h);case Kt:return k=y._init,I(h,v,k(y._payload),C)}if(Nr(y))return w(h,v,y,C);if(gr(y))return S(h,v,y,C);ji(h,y)}return typeof y=="string"&&y!==""||typeof y=="number"?(y=""+y,v!==null&&v.tag===6?(n(h,v.sibling),v=i(v,y),v.return=h,h=v):(n(h,v),v=po(y,h.mode,C),v.return=h,h=v),o(h)):n(h,v)}return I}var lr=uf(!0),cf=uf(!1),ul=fn(null),cl=null,Wn=null,rs=null;function is(){rs=Wn=cl=null}function ls(e){var t=ul.current;ce(ul),e._currentValue=t}function ia(e,t,n){for(;e!==null;){var r=e.alternate;if((e.childLanes&t)!==t?(e.childLanes|=t,r!==null&&(r.childLanes|=t)):r!==null&&(r.childLanes&t)!==t&&(r.childLanes|=t),e===n)break;e=e.return}}function Jn(e,t){cl=e,rs=Wn=null,e=e.dependencies,e!==null&&e.firstContext!==null&&(e.lanes&t&&(He=!0),e.firstContext=null)}function dt(e){var t=e._currentValue;if(rs!==e)if(e={context:e,memoizedValue:t,next:null},Wn===null){if(cl===null)throw Error(M(308));Wn=e,cl.dependencies={lanes:0,firstContext:e}}else Wn=Wn.next=e;return t}var kn=null;function os(e){kn===null?kn=[e]:kn.push(e)}function df(e,t,n,r){var i=t.interleaved;return i===null?(n.next=n,os(t)):(n.next=i.next,i.next=n),t.interleaved=n,Bt(e,r)}function Bt(e,t){e.lanes|=t;var n=e.alternate;for(n!==null&&(n.lanes|=t),n=e,e=e.return;e!==null;)e.childLanes|=t,n=e.alternate,n!==null&&(n.childLanes|=t),n=e,e=e.return;return n.tag===3?n.stateNode:null}var Yt=!1;function as(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,interleaved:null,lanes:0},effects:null}}function ff(e,t){e=e.updateQueue,t.updateQueue===e&&(t.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,effects:e.effects})}function Rt(e,t){return{eventTime:e,lane:t,tag:0,payload:null,callback:null,next:null}}function ln(e,t,n){var r=e.updateQueue;if(r===null)return null;if(r=r.shared,Z&2){var i=r.pending;return i===null?t.next=t:(t.next=i.next,i.next=t),r.pending=t,Bt(e,n)}return i=r.interleaved,i===null?(t.next=t,os(r)):(t.next=i.next,i.next=t),r.interleaved=t,Bt(e,n)}function Bi(e,t,n){if(t=t.updateQueue,t!==null&&(t=t.shared,(n&4194240)!==0)){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Qa(e,n)}}function Pu(e,t){var n=e.updateQueue,r=e.alternate;if(r!==null&&(r=r.updateQueue,n===r)){var i=null,l=null;if(n=n.firstBaseUpdate,n!==null){do{var o={eventTime:n.eventTime,lane:n.lane,tag:n.tag,payload:n.payload,callback:n.callback,next:null};l===null?i=l=o:l=l.next=o,n=n.next}while(n!==null);l===null?i=l=t:l=l.next=t}else i=l=t;n={baseState:r.baseState,firstBaseUpdate:i,lastBaseUpdate:l,shared:r.shared,effects:r.effects},e.updateQueue=n;return}e=n.lastBaseUpdate,e===null?n.firstBaseUpdate=t:e.next=t,n.lastBaseUpdate=t}function dl(e,t,n,r){var i=e.updateQueue;Yt=!1;var l=i.firstBaseUpdate,o=i.lastBaseUpdate,a=i.shared.pending;if(a!==null){i.shared.pending=null;var s=a,c=s.next;s.next=null,o===null?l=c:o.next=c,o=s;var d=e.alternate;d!==null&&(d=d.updateQueue,a=d.lastBaseUpdate,a!==o&&(a===null?d.firstBaseUpdate=c:a.next=c,d.lastBaseUpdate=s))}if(l!==null){var f=i.baseState;o=0,d=c=s=null,a=l;do{var m=a.lane,p=a.eventTime;if((r&m)===m){d!==null&&(d=d.next={eventTime:p,lane:0,tag:a.tag,payload:a.payload,callback:a.callback,next:null});e:{var w=e,S=a;switch(m=t,p=n,S.tag){case 1:if(w=S.payload,typeof w=="function"){f=w.call(p,f,m);break e}f=w;break e;case 3:w.flags=w.flags&-65537|128;case 0:if(w=S.payload,m=typeof w=="function"?w.call(p,f,m):w,m==null)break e;f=he({},f,m);break e;case 2:Yt=!0}}a.callback!==null&&a.lane!==0&&(e.flags|=64,m=i.effects,m===null?i.effects=[a]:m.push(a))}else p={eventTime:p,lane:m,tag:a.tag,payload:a.payload,callback:a.callback,next:null},d===null?(c=d=p,s=f):d=d.next=p,o|=m;if(a=a.next,a===null){if(a=i.shared.pending,a===null)break;m=a,a=m.next,m.next=null,i.lastBaseUpdate=m,i.shared.pending=null}}while(!0);if(d===null&&(s=f),i.baseState=s,i.firstBaseUpdate=c,i.lastBaseUpdate=d,t=i.shared.interleaved,t!==null){i=t;do o|=i.lane,i=i.next;while(i!==t)}else l===null&&(i.shared.lanes=0);Nn|=o,e.lanes=o,e.memoizedState=f}}function Iu(e,t,n){if(e=t.effects,t.effects=null,e!==null)for(t=0;t<e.length;t++){var r=e[t],i=r.callback;if(i!==null){if(r.callback=null,r=n,typeof i!="function")throw Error(M(191,i));i.call(r)}}}var ci={},Nt=fn(ci),Zr=fn(ci),ei=fn(ci);function wn(e){if(e===ci)throw Error(M(174));return e}function ss(e,t){switch(ae(ei,t),ae(Zr,e),ae(Nt,ci),e=t.nodeType,e){case 9:case 11:t=(t=t.documentElement)?t.namespaceURI:Fo(null,"");break;default:e=e===8?t.parentNode:t,t=e.namespaceURI||null,e=e.tagName,t=Fo(t,e)}ce(Nt),ae(Nt,t)}function or(){ce(Nt),ce(Zr),ce(ei)}function pf(e){wn(ei.current);var t=wn(Nt.current),n=Fo(t,e.type);t!==n&&(ae(Zr,e),ae(Nt,n))}function us(e){Zr.current===e&&(ce(Nt),ce(Zr))}var fe=fn(0);function fl(e){for(var t=e;t!==null;){if(t.tag===13){var n=t.memoizedState;if(n!==null&&(n=n.dehydrated,n===null||n.data==="$?"||n.data==="$!"))return t}else if(t.tag===19&&t.memoizedProps.revealOrder!==void 0){if(t.flags&128)return t}else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return null;t=t.return}t.sibling.return=t.return,t=t.sibling}return null}var oo=[];function cs(){for(var e=0;e<oo.length;e++)oo[e]._workInProgressVersionPrimary=null;oo.length=0}var $i=Ut.ReactCurrentDispatcher,ao=Ut.ReactCurrentBatchConfig,En=0,pe=null,xe=null,we=null,pl=!1,Ar=!1,ti=0,Qm=0;function _e(){throw Error(M(321))}function ds(e,t){if(t===null)return!1;for(var n=0;n<t.length&&n<e.length;n++)if(!kt(e[n],t[n]))return!1;return!0}function fs(e,t,n,r,i,l){if(En=l,pe=t,t.memoizedState=null,t.updateQueue=null,t.lanes=0,$i.current=e===null||e.memoizedState===null?Xm:Gm,e=n(r,i),Ar){l=0;do{if(Ar=!1,ti=0,25<=l)throw Error(M(301));l+=1,we=xe=null,t.updateQueue=null,$i.current=Jm,e=n(r,i)}while(Ar)}if($i.current=hl,t=xe!==null&&xe.next!==null,En=0,we=xe=pe=null,pl=!1,t)throw Error(M(300));return e}function ps(){var e=ti!==0;return ti=0,e}function St(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return we===null?pe.memoizedState=we=e:we=we.next=e,we}function ft(){if(xe===null){var e=pe.alternate;e=e!==null?e.memoizedState:null}else e=xe.next;var t=we===null?pe.memoizedState:we.next;if(t!==null)we=t,xe=e;else{if(e===null)throw Error(M(310));xe=e,e={memoizedState:xe.memoizedState,baseState:xe.baseState,baseQueue:xe.baseQueue,queue:xe.queue,next:null},we===null?pe.memoizedState=we=e:we=we.next=e}return we}function ni(e,t){return typeof t=="function"?t(e):t}function so(e){var t=ft(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=xe,i=r.baseQueue,l=n.pending;if(l!==null){if(i!==null){var o=i.next;i.next=l.next,l.next=o}r.baseQueue=i=l,n.pending=null}if(i!==null){l=i.next,r=r.baseState;var a=o=null,s=null,c=l;do{var d=c.lane;if((En&d)===d)s!==null&&(s=s.next={lane:0,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null}),r=c.hasEagerState?c.eagerState:e(r,c.action);else{var f={lane:d,action:c.action,hasEagerState:c.hasEagerState,eagerState:c.eagerState,next:null};s===null?(a=s=f,o=r):s=s.next=f,pe.lanes|=d,Nn|=d}c=c.next}while(c!==null&&c!==l);s===null?o=r:s.next=a,kt(r,t.memoizedState)||(He=!0),t.memoizedState=r,t.baseState=o,t.baseQueue=s,n.lastRenderedState=r}if(e=n.interleaved,e!==null){i=e;do l=i.lane,pe.lanes|=l,Nn|=l,i=i.next;while(i!==e)}else i===null&&(n.lanes=0);return[t.memoizedState,n.dispatch]}function uo(e){var t=ft(),n=t.queue;if(n===null)throw Error(M(311));n.lastRenderedReducer=e;var r=n.dispatch,i=n.pending,l=t.memoizedState;if(i!==null){n.pending=null;var o=i=i.next;do l=e(l,o.action),o=o.next;while(o!==i);kt(l,t.memoizedState)||(He=!0),t.memoizedState=l,t.baseQueue===null&&(t.baseState=l),n.lastRenderedState=l}return[l,r]}function hf(){}function mf(e,t){var n=pe,r=ft(),i=t(),l=!kt(r.memoizedState,i);if(l&&(r.memoizedState=i,He=!0),r=r.queue,hs(yf.bind(null,n,r,e),[e]),r.getSnapshot!==t||l||we!==null&&we.memoizedState.tag&1){if(n.flags|=2048,ri(9,vf.bind(null,n,r,i,t),void 0,null),Se===null)throw Error(M(349));En&30||gf(n,t,i)}return i}function gf(e,t,n){e.flags|=16384,e={getSnapshot:t,value:n},t=pe.updateQueue,t===null?(t={lastEffect:null,stores:null},pe.updateQueue=t,t.stores=[e]):(n=t.stores,n===null?t.stores=[e]:n.push(e))}function vf(e,t,n,r){t.value=n,t.getSnapshot=r,xf(t)&&kf(e)}function yf(e,t,n){return n(function(){xf(t)&&kf(e)})}function xf(e){var t=e.getSnapshot;e=e.value;try{var n=t();return!kt(e,n)}catch{return!0}}function kf(e){var t=Bt(e,1);t!==null&&xt(t,e,1,-1)}function Mu(e){var t=St();return typeof e=="function"&&(e=e()),t.memoizedState=t.baseState=e,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:ni,lastRenderedState:e},t.queue=e,e=e.dispatch=Ym.bind(null,pe,e),[t.memoizedState,e]}function ri(e,t,n,r){return e={tag:e,create:t,destroy:n,deps:r,next:null},t=pe.updateQueue,t===null?(t={lastEffect:null,stores:null},pe.updateQueue=t,t.lastEffect=e.next=e):(n=t.lastEffect,n===null?t.lastEffect=e.next=e:(r=n.next,n.next=e,e.next=r,t.lastEffect=e)),e}function wf(){return ft().memoizedState}function Ui(e,t,n,r){var i=St();pe.flags|=e,i.memoizedState=ri(1|t,n,void 0,r===void 0?null:r)}function zl(e,t,n,r){var i=ft();r=r===void 0?null:r;var l=void 0;if(xe!==null){var o=xe.memoizedState;if(l=o.destroy,r!==null&&ds(r,o.deps)){i.memoizedState=ri(t,n,l,r);return}}pe.flags|=e,i.memoizedState=ri(1|t,n,l,r)}function Au(e,t){return Ui(8390656,8,e,t)}function hs(e,t){return zl(2048,8,e,t)}function Sf(e,t){return zl(4,2,e,t)}function bf(e,t){return zl(4,4,e,t)}function Cf(e,t){if(typeof t=="function")return e=e(),t(e),function(){t(null)};if(t!=null)return e=e(),t.current=e,function(){t.current=null}}function jf(e,t,n){return n=n!=null?n.concat([e]):null,zl(4,4,Cf.bind(null,t,e),n)}function ms(){}function Ef(e,t){var n=ft();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ds(t,r[1])?r[0]:(n.memoizedState=[e,t],e)}function Nf(e,t){var n=ft();t=t===void 0?null:t;var r=n.memoizedState;return r!==null&&t!==null&&ds(t,r[1])?r[0]:(e=e(),n.memoizedState=[e,t],e)}function _f(e,t,n){return En&21?(kt(n,t)||(n=Pd(),pe.lanes|=n,Nn|=n,e.baseState=!0),t):(e.baseState&&(e.baseState=!1,He=!0),e.memoizedState=n)}function qm(e,t){var n=re;re=n!==0&&4>n?n:4,e(!0);var r=ao.transition;ao.transition={};try{e(!1),t()}finally{re=n,ao.transition=r}}function zf(){return ft().memoizedState}function Km(e,t,n){var r=an(e);if(n={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null},Tf(e))Lf(t,n);else if(n=df(e,t,n,r),n!==null){var i=Re();xt(n,e,r,i),Pf(n,t,r)}}function Ym(e,t,n){var r=an(e),i={lane:r,action:n,hasEagerState:!1,eagerState:null,next:null};if(Tf(e))Lf(t,i);else{var l=e.alternate;if(e.lanes===0&&(l===null||l.lanes===0)&&(l=t.lastRenderedReducer,l!==null))try{var o=t.lastRenderedState,a=l(o,n);if(i.hasEagerState=!0,i.eagerState=a,kt(a,o)){var s=t.interleaved;s===null?(i.next=i,os(t)):(i.next=s.next,s.next=i),t.interleaved=i;return}}catch{}finally{}n=df(e,t,i,r),n!==null&&(i=Re(),xt(n,e,r,i),Pf(n,t,r))}}function Tf(e){var t=e.alternate;return e===pe||t!==null&&t===pe}function Lf(e,t){Ar=pl=!0;var n=e.pending;n===null?t.next=t:(t.next=n.next,n.next=t),e.pending=t}function Pf(e,t,n){if(n&4194240){var r=t.lanes;r&=e.pendingLanes,n|=r,t.lanes=n,Qa(e,n)}}var hl={readContext:dt,useCallback:_e,useContext:_e,useEffect:_e,useImperativeHandle:_e,useInsertionEffect:_e,useLayoutEffect:_e,useMemo:_e,useReducer:_e,useRef:_e,useState:_e,useDebugValue:_e,useDeferredValue:_e,useTransition:_e,useMutableSource:_e,useSyncExternalStore:_e,useId:_e,unstable_isNewReconciler:!1},Xm={readContext:dt,useCallback:function(e,t){return St().memoizedState=[e,t===void 0?null:t],e},useContext:dt,useEffect:Au,useImperativeHandle:function(e,t,n){return n=n!=null?n.concat([e]):null,Ui(4194308,4,Cf.bind(null,t,e),n)},useLayoutEffect:function(e,t){return Ui(4194308,4,e,t)},useInsertionEffect:function(e,t){return Ui(4,2,e,t)},useMemo:function(e,t){var n=St();return t=t===void 0?null:t,e=e(),n.memoizedState=[e,t],e},useReducer:function(e,t,n){var r=St();return t=n!==void 0?n(t):t,r.memoizedState=r.baseState=t,e={pending:null,interleaved:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:t},r.queue=e,e=e.dispatch=Km.bind(null,pe,e),[r.memoizedState,e]},useRef:function(e){var t=St();return e={current:e},t.memoizedState=e},useState:Mu,useDebugValue:ms,useDeferredValue:function(e){return St().memoizedState=e},useTransition:function(){var e=Mu(!1),t=e[0];return e=qm.bind(null,e[1]),St().memoizedState=e,[t,e]},useMutableSource:function(){},useSyncExternalStore:function(e,t,n){var r=pe,i=St();if(de){if(n===void 0)throw Error(M(407));n=n()}else{if(n=t(),Se===null)throw Error(M(349));En&30||gf(r,t,n)}i.memoizedState=n;var l={value:n,getSnapshot:t};return i.queue=l,Au(yf.bind(null,r,l,e),[e]),r.flags|=2048,ri(9,vf.bind(null,r,l,n,t),void 0,null),n},useId:function(){var e=St(),t=Se.identifierPrefix;if(de){var n=Dt,r=At;n=(r&~(1<<32-yt(r)-1)).toString(32)+n,t=":"+t+"R"+n,n=ti++,0<n&&(t+="H"+n.toString(32)),t+=":"}else n=Qm++,t=":"+t+"r"+n.toString(32)+":";return e.memoizedState=t},unstable_isNewReconciler:!1},Gm={readContext:dt,useCallback:Ef,useContext:dt,useEffect:hs,useImperativeHandle:jf,useInsertionEffect:Sf,useLayoutEffect:bf,useMemo:Nf,useReducer:so,useRef:wf,useState:function(){return so(ni)},useDebugValue:ms,useDeferredValue:function(e){var t=ft();return _f(t,xe.memoizedState,e)},useTransition:function(){var e=so(ni)[0],t=ft().memoizedState;return[e,t]},useMutableSource:hf,useSyncExternalStore:mf,useId:zf,unstable_isNewReconciler:!1},Jm={readContext:dt,useCallback:Ef,useContext:dt,useEffect:hs,useImperativeHandle:jf,useInsertionEffect:Sf,useLayoutEffect:bf,useMemo:Nf,useReducer:uo,useRef:wf,useState:function(){return uo(ni)},useDebugValue:ms,useDeferredValue:function(e){var t=ft();return xe===null?t.memoizedState=e:_f(t,xe.memoizedState,e)},useTransition:function(){var e=uo(ni)[0],t=ft().memoizedState;return[e,t]},useMutableSource:hf,useSyncExternalStore:mf,useId:zf,unstable_isNewReconciler:!1};function mt(e,t){if(e&&e.defaultProps){t=he({},t),e=e.defaultProps;for(var n in e)t[n]===void 0&&(t[n]=e[n]);return t}return t}function la(e,t,n,r){t=e.memoizedState,n=n(r,t),n=n==null?t:he({},t,n),e.memoizedState=n,e.lanes===0&&(e.updateQueue.baseState=n)}var Tl={isMounted:function(e){return(e=e._reactInternals)?Tn(e)===e:!1},enqueueSetState:function(e,t,n){e=e._reactInternals;var r=Re(),i=an(e),l=Rt(r,i);l.payload=t,n!=null&&(l.callback=n),t=ln(e,l,i),t!==null&&(xt(t,e,i,r),Bi(t,e,i))},enqueueReplaceState:function(e,t,n){e=e._reactInternals;var r=Re(),i=an(e),l=Rt(r,i);l.tag=1,l.payload=t,n!=null&&(l.callback=n),t=ln(e,l,i),t!==null&&(xt(t,e,i,r),Bi(t,e,i))},enqueueForceUpdate:function(e,t){e=e._reactInternals;var n=Re(),r=an(e),i=Rt(n,r);i.tag=2,t!=null&&(i.callback=t),t=ln(e,i,r),t!==null&&(xt(t,e,r,n),Bi(t,e,r))}};function Du(e,t,n,r,i,l,o){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(r,l,o):t.prototype&&t.prototype.isPureReactComponent?!Yr(n,r)||!Yr(i,l):!0}function If(e,t,n){var r=!1,i=cn,l=t.contextType;return typeof l=="object"&&l!==null?l=dt(l):(i=We(t)?Cn:Pe.current,r=t.contextTypes,l=(r=r!=null)?rr(e,i):cn),t=new t(n,l),e.memoizedState=t.state!==null&&t.state!==void 0?t.state:null,t.updater=Tl,e.stateNode=t,t._reactInternals=e,r&&(e=e.stateNode,e.__reactInternalMemoizedUnmaskedChildContext=i,e.__reactInternalMemoizedMaskedChildContext=l),t}function Ru(e,t,n,r){e=t.state,typeof t.componentWillReceiveProps=="function"&&t.componentWillReceiveProps(n,r),typeof t.UNSAFE_componentWillReceiveProps=="function"&&t.UNSAFE_componentWillReceiveProps(n,r),t.state!==e&&Tl.enqueueReplaceState(t,t.state,null)}function oa(e,t,n,r){var i=e.stateNode;i.props=n,i.state=e.memoizedState,i.refs={},as(e);var l=t.contextType;typeof l=="object"&&l!==null?i.context=dt(l):(l=We(t)?Cn:Pe.current,i.context=rr(e,l)),i.state=e.memoizedState,l=t.getDerivedStateFromProps,typeof l=="function"&&(la(e,t,l,n),i.state=e.memoizedState),typeof t.getDerivedStateFromProps=="function"||typeof i.getSnapshotBeforeUpdate=="function"||typeof i.UNSAFE_componentWillMount!="function"&&typeof i.componentWillMount!="function"||(t=i.state,typeof i.componentWillMount=="function"&&i.componentWillMount(),typeof i.UNSAFE_componentWillMount=="function"&&i.UNSAFE_componentWillMount(),t!==i.state&&Tl.enqueueReplaceState(i,i.state,null),dl(e,n,i,r),i.state=e.memoizedState),typeof i.componentDidMount=="function"&&(e.flags|=4194308)}function ar(e,t){try{var n="",r=t;do n+=Eh(r),r=r.return;while(r);var i=n}catch(l){i=`
Error generating stack: `+l.message+`
`+l.stack}return{value:e,source:t,stack:i,digest:null}}function co(e,t,n){return{value:e,source:null,stack:n??null,digest:t??null}}function aa(e,t){try{console.error(t.value)}catch(n){setTimeout(function(){throw n})}}var Zm=typeof WeakMap=="function"?WeakMap:Map;function Mf(e,t,n){n=Rt(-1,n),n.tag=3,n.payload={element:null};var r=t.value;return n.callback=function(){gl||(gl=!0,va=r),aa(e,t)},n}function Af(e,t,n){n=Rt(-1,n),n.tag=3;var r=e.type.getDerivedStateFromError;if(typeof r=="function"){var i=t.value;n.payload=function(){return r(i)},n.callback=function(){aa(e,t)}}var l=e.stateNode;return l!==null&&typeof l.componentDidCatch=="function"&&(n.callback=function(){aa(e,t),typeof r!="function"&&(on===null?on=new Set([this]):on.add(this));var o=t.stack;this.componentDidCatch(t.value,{componentStack:o!==null?o:""})}),n}function Fu(e,t,n){var r=e.pingCache;if(r===null){r=e.pingCache=new Zm;var i=new Set;r.set(t,i)}else i=r.get(t),i===void 0&&(i=new Set,r.set(t,i));i.has(n)||(i.add(n),e=pg.bind(null,e,t,n),t.then(e,e))}function Ou(e){do{var t;if((t=e.tag===13)&&(t=e.memoizedState,t=t!==null?t.dehydrated!==null:!0),t)return e;e=e.return}while(e!==null);return null}function Bu(e,t,n,r,i){return e.mode&1?(e.flags|=65536,e.lanes=i,e):(e===t?e.flags|=65536:(e.flags|=128,n.flags|=131072,n.flags&=-52805,n.tag===1&&(n.alternate===null?n.tag=17:(t=Rt(-1,1),t.tag=2,ln(n,t,1))),n.lanes|=1),e)}var eg=Ut.ReactCurrentOwner,He=!1;function De(e,t,n,r){t.child=e===null?cf(t,null,n,r):lr(t,e.child,n,r)}function $u(e,t,n,r,i){n=n.render;var l=t.ref;return Jn(t,i),r=fs(e,t,n,r,l,i),n=ps(),e!==null&&!He?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,$t(e,t,i)):(de&&n&&es(t),t.flags|=1,De(e,t,r,i),t.child)}function Uu(e,t,n,r,i){if(e===null){var l=n.type;return typeof l=="function"&&!bs(l)&&l.defaultProps===void 0&&n.compare===null&&n.defaultProps===void 0?(t.tag=15,t.type=l,Df(e,t,l,r,i)):(e=Qi(n.type,null,r,t,t.mode,i),e.ref=t.ref,e.return=t,t.child=e)}if(l=e.child,!(e.lanes&i)){var o=l.memoizedProps;if(n=n.compare,n=n!==null?n:Yr,n(o,r)&&e.ref===t.ref)return $t(e,t,i)}return t.flags|=1,e=sn(l,r),e.ref=t.ref,e.return=t,t.child=e}function Df(e,t,n,r,i){if(e!==null){var l=e.memoizedProps;if(Yr(l,r)&&e.ref===t.ref)if(He=!1,t.pendingProps=r=l,(e.lanes&i)!==0)e.flags&131072&&(He=!0);else return t.lanes=e.lanes,$t(e,t,i)}return sa(e,t,n,r,i)}function Rf(e,t,n){var r=t.pendingProps,i=r.children,l=e!==null?e.memoizedState:null;if(r.mode==="hidden")if(!(t.mode&1))t.memoizedState={baseLanes:0,cachePool:null,transitions:null},ae(qn,Je),Je|=n;else{if(!(n&1073741824))return e=l!==null?l.baseLanes|n:n,t.lanes=t.childLanes=1073741824,t.memoizedState={baseLanes:e,cachePool:null,transitions:null},t.updateQueue=null,ae(qn,Je),Je|=e,null;t.memoizedState={baseLanes:0,cachePool:null,transitions:null},r=l!==null?l.baseLanes:n,ae(qn,Je),Je|=r}else l!==null?(r=l.baseLanes|n,t.memoizedState=null):r=n,ae(qn,Je),Je|=r;return De(e,t,i,n),t.child}function Ff(e,t){var n=t.ref;(e===null&&n!==null||e!==null&&e.ref!==n)&&(t.flags|=512,t.flags|=2097152)}function sa(e,t,n,r,i){var l=We(n)?Cn:Pe.current;return l=rr(t,l),Jn(t,i),n=fs(e,t,n,r,l,i),r=ps(),e!==null&&!He?(t.updateQueue=e.updateQueue,t.flags&=-2053,e.lanes&=~i,$t(e,t,i)):(de&&r&&es(t),t.flags|=1,De(e,t,n,i),t.child)}function Hu(e,t,n,r,i){if(We(n)){var l=!0;ol(t)}else l=!1;if(Jn(t,i),t.stateNode===null)Hi(e,t),If(t,n,r),oa(t,n,r,i),r=!0;else if(e===null){var o=t.stateNode,a=t.memoizedProps;o.props=a;var s=o.context,c=n.contextType;typeof c=="object"&&c!==null?c=dt(c):(c=We(n)?Cn:Pe.current,c=rr(t,c));var d=n.getDerivedStateFromProps,f=typeof d=="function"||typeof o.getSnapshotBeforeUpdate=="function";f||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==r||s!==c)&&Ru(t,o,r,c),Yt=!1;var m=t.memoizedState;o.state=m,dl(t,r,o,i),s=t.memoizedState,a!==r||m!==s||Ve.current||Yt?(typeof d=="function"&&(la(t,n,d,r),s=t.memoizedState),(a=Yt||Du(t,n,a,r,m,s,c))?(f||typeof o.UNSAFE_componentWillMount!="function"&&typeof o.componentWillMount!="function"||(typeof o.componentWillMount=="function"&&o.componentWillMount(),typeof o.UNSAFE_componentWillMount=="function"&&o.UNSAFE_componentWillMount()),typeof o.componentDidMount=="function"&&(t.flags|=4194308)):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),t.memoizedProps=r,t.memoizedState=s),o.props=r,o.state=s,o.context=c,r=a):(typeof o.componentDidMount=="function"&&(t.flags|=4194308),r=!1)}else{o=t.stateNode,ff(e,t),a=t.memoizedProps,c=t.type===t.elementType?a:mt(t.type,a),o.props=c,f=t.pendingProps,m=o.context,s=n.contextType,typeof s=="object"&&s!==null?s=dt(s):(s=We(n)?Cn:Pe.current,s=rr(t,s));var p=n.getDerivedStateFromProps;(d=typeof p=="function"||typeof o.getSnapshotBeforeUpdate=="function")||typeof o.UNSAFE_componentWillReceiveProps!="function"&&typeof o.componentWillReceiveProps!="function"||(a!==f||m!==s)&&Ru(t,o,r,s),Yt=!1,m=t.memoizedState,o.state=m,dl(t,r,o,i);var w=t.memoizedState;a!==f||m!==w||Ve.current||Yt?(typeof p=="function"&&(la(t,n,p,r),w=t.memoizedState),(c=Yt||Du(t,n,c,r,m,w,s)||!1)?(d||typeof o.UNSAFE_componentWillUpdate!="function"&&typeof o.componentWillUpdate!="function"||(typeof o.componentWillUpdate=="function"&&o.componentWillUpdate(r,w,s),typeof o.UNSAFE_componentWillUpdate=="function"&&o.UNSAFE_componentWillUpdate(r,w,s)),typeof o.componentDidUpdate=="function"&&(t.flags|=4),typeof o.getSnapshotBeforeUpdate=="function"&&(t.flags|=1024)):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=1024),t.memoizedProps=r,t.memoizedState=w),o.props=r,o.state=w,o.context=s,r=c):(typeof o.componentDidUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=4),typeof o.getSnapshotBeforeUpdate!="function"||a===e.memoizedProps&&m===e.memoizedState||(t.flags|=1024),r=!1)}return ua(e,t,n,r,l,i)}function ua(e,t,n,r,i,l){Ff(e,t);var o=(t.flags&128)!==0;if(!r&&!o)return i&&_u(t,n,!1),$t(e,t,l);r=t.stateNode,eg.current=t;var a=o&&typeof n.getDerivedStateFromError!="function"?null:r.render();return t.flags|=1,e!==null&&o?(t.child=lr(t,e.child,null,l),t.child=lr(t,null,a,l)):De(e,t,a,l),t.memoizedState=r.state,i&&_u(t,n,!0),t.child}function Of(e){var t=e.stateNode;t.pendingContext?Nu(e,t.pendingContext,t.pendingContext!==t.context):t.context&&Nu(e,t.context,!1),ss(e,t.containerInfo)}function Vu(e,t,n,r,i){return ir(),ns(i),t.flags|=256,De(e,t,n,r),t.child}var ca={dehydrated:null,treeContext:null,retryLane:0};function da(e){return{baseLanes:e,cachePool:null,transitions:null}}function Bf(e,t,n){var r=t.pendingProps,i=fe.current,l=!1,o=(t.flags&128)!==0,a;if((a=o)||(a=e!==null&&e.memoizedState===null?!1:(i&2)!==0),a?(l=!0,t.flags&=-129):(e===null||e.memoizedState!==null)&&(i|=1),ae(fe,i&1),e===null)return ra(t),e=t.memoizedState,e!==null&&(e=e.dehydrated,e!==null)?(t.mode&1?e.data==="$!"?t.lanes=8:t.lanes=1073741824:t.lanes=1,null):(o=r.children,e=r.fallback,l?(r=t.mode,l=t.child,o={mode:"hidden",children:o},!(r&1)&&l!==null?(l.childLanes=0,l.pendingProps=o):l=Il(o,r,0,null),e=bn(e,r,n,null),l.return=t,e.return=t,l.sibling=e,t.child=l,t.child.memoizedState=da(n),t.memoizedState=ca,e):gs(t,o));if(i=e.memoizedState,i!==null&&(a=i.dehydrated,a!==null))return tg(e,t,o,r,a,i,n);if(l){l=r.fallback,o=t.mode,i=e.child,a=i.sibling;var s={mode:"hidden",children:r.children};return!(o&1)&&t.child!==i?(r=t.child,r.childLanes=0,r.pendingProps=s,t.deletions=null):(r=sn(i,s),r.subtreeFlags=i.subtreeFlags&14680064),a!==null?l=sn(a,l):(l=bn(l,o,n,null),l.flags|=2),l.return=t,r.return=t,r.sibling=l,t.child=r,r=l,l=t.child,o=e.child.memoizedState,o=o===null?da(n):{baseLanes:o.baseLanes|n,cachePool:null,transitions:o.transitions},l.memoizedState=o,l.childLanes=e.childLanes&~n,t.memoizedState=ca,r}return l=e.child,e=l.sibling,r=sn(l,{mode:"visible",children:r.children}),!(t.mode&1)&&(r.lanes=n),r.return=t,r.sibling=null,e!==null&&(n=t.deletions,n===null?(t.deletions=[e],t.flags|=16):n.push(e)),t.child=r,t.memoizedState=null,r}function gs(e,t){return t=Il({mode:"visible",children:t},e.mode,0,null),t.return=e,e.child=t}function Ei(e,t,n,r){return r!==null&&ns(r),lr(t,e.child,null,n),e=gs(t,t.pendingProps.children),e.flags|=2,t.memoizedState=null,e}function tg(e,t,n,r,i,l,o){if(n)return t.flags&256?(t.flags&=-257,r=co(Error(M(422))),Ei(e,t,o,r)):t.memoizedState!==null?(t.child=e.child,t.flags|=128,null):(l=r.fallback,i=t.mode,r=Il({mode:"visible",children:r.children},i,0,null),l=bn(l,i,o,null),l.flags|=2,r.return=t,l.return=t,r.sibling=l,t.child=r,t.mode&1&&lr(t,e.child,null,o),t.child.memoizedState=da(o),t.memoizedState=ca,l);if(!(t.mode&1))return Ei(e,t,o,null);if(i.data==="$!"){if(r=i.nextSibling&&i.nextSibling.dataset,r)var a=r.dgst;return r=a,l=Error(M(419)),r=co(l,r,void 0),Ei(e,t,o,r)}if(a=(o&e.childLanes)!==0,He||a){if(r=Se,r!==null){switch(o&-o){case 4:i=2;break;case 16:i=8;break;case 64:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:case 67108864:i=32;break;case 536870912:i=268435456;break;default:i=0}i=i&(r.suspendedLanes|o)?0:i,i!==0&&i!==l.retryLane&&(l.retryLane=i,Bt(e,i),xt(r,e,i,-1))}return Ss(),r=co(Error(M(421))),Ei(e,t,o,r)}return i.data==="$?"?(t.flags|=128,t.child=e.child,t=hg.bind(null,e),i._reactRetry=t,null):(e=l.treeContext,Ze=rn(i.nextSibling),tt=t,de=!0,vt=null,e!==null&&(ot[at++]=At,ot[at++]=Dt,ot[at++]=jn,At=e.id,Dt=e.overflow,jn=t),t=gs(t,r.children),t.flags|=4096,t)}function Wu(e,t,n){e.lanes|=t;var r=e.alternate;r!==null&&(r.lanes|=t),ia(e.return,t,n)}function fo(e,t,n,r,i){var l=e.memoizedState;l===null?e.memoizedState={isBackwards:t,rendering:null,renderingStartTime:0,last:r,tail:n,tailMode:i}:(l.isBackwards=t,l.rendering=null,l.renderingStartTime=0,l.last=r,l.tail=n,l.tailMode=i)}function $f(e,t,n){var r=t.pendingProps,i=r.revealOrder,l=r.tail;if(De(e,t,r.children,n),r=fe.current,r&2)r=r&1|2,t.flags|=128;else{if(e!==null&&e.flags&128)e:for(e=t.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&Wu(e,n,t);else if(e.tag===19)Wu(e,n,t);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===t)break e;for(;e.sibling===null;){if(e.return===null||e.return===t)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}r&=1}if(ae(fe,r),!(t.mode&1))t.memoizedState=null;else switch(i){case"forwards":for(n=t.child,i=null;n!==null;)e=n.alternate,e!==null&&fl(e)===null&&(i=n),n=n.sibling;n=i,n===null?(i=t.child,t.child=null):(i=n.sibling,n.sibling=null),fo(t,!1,i,n,l);break;case"backwards":for(n=null,i=t.child,t.child=null;i!==null;){if(e=i.alternate,e!==null&&fl(e)===null){t.child=i;break}e=i.sibling,i.sibling=n,n=i,i=e}fo(t,!0,n,null,l);break;case"together":fo(t,!1,null,null,void 0);break;default:t.memoizedState=null}return t.child}function Hi(e,t){!(t.mode&1)&&e!==null&&(e.alternate=null,t.alternate=null,t.flags|=2)}function $t(e,t,n){if(e!==null&&(t.dependencies=e.dependencies),Nn|=t.lanes,!(n&t.childLanes))return null;if(e!==null&&t.child!==e.child)throw Error(M(153));if(t.child!==null){for(e=t.child,n=sn(e,e.pendingProps),t.child=n,n.return=t;e.sibling!==null;)e=e.sibling,n=n.sibling=sn(e,e.pendingProps),n.return=t;n.sibling=null}return t.child}function ng(e,t,n){switch(t.tag){case 3:Of(t),ir();break;case 5:pf(t);break;case 1:We(t.type)&&ol(t);break;case 4:ss(t,t.stateNode.containerInfo);break;case 10:var r=t.type._context,i=t.memoizedProps.value;ae(ul,r._currentValue),r._currentValue=i;break;case 13:if(r=t.memoizedState,r!==null)return r.dehydrated!==null?(ae(fe,fe.current&1),t.flags|=128,null):n&t.child.childLanes?Bf(e,t,n):(ae(fe,fe.current&1),e=$t(e,t,n),e!==null?e.sibling:null);ae(fe,fe.current&1);break;case 19:if(r=(n&t.childLanes)!==0,e.flags&128){if(r)return $f(e,t,n);t.flags|=128}if(i=t.memoizedState,i!==null&&(i.rendering=null,i.tail=null,i.lastEffect=null),ae(fe,fe.current),r)break;return null;case 22:case 23:return t.lanes=0,Rf(e,t,n)}return $t(e,t,n)}var Uf,fa,Hf,Vf;Uf=function(e,t){for(var n=t.child;n!==null;){if(n.tag===5||n.tag===6)e.appendChild(n.stateNode);else if(n.tag!==4&&n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return;n=n.return}n.sibling.return=n.return,n=n.sibling}};fa=function(){};Hf=function(e,t,n,r){var i=e.memoizedProps;if(i!==r){e=t.stateNode,wn(Nt.current);var l=null;switch(n){case"input":i=Mo(e,i),r=Mo(e,r),l=[];break;case"select":i=he({},i,{value:void 0}),r=he({},r,{value:void 0}),l=[];break;case"textarea":i=Ro(e,i),r=Ro(e,r),l=[];break;default:typeof i.onClick!="function"&&typeof r.onClick=="function"&&(e.onclick=il)}Oo(n,r);var o;n=null;for(c in i)if(!r.hasOwnProperty(c)&&i.hasOwnProperty(c)&&i[c]!=null)if(c==="style"){var a=i[c];for(o in a)a.hasOwnProperty(o)&&(n||(n={}),n[o]="")}else c!=="dangerouslySetInnerHTML"&&c!=="children"&&c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&c!=="autoFocus"&&(Ur.hasOwnProperty(c)?l||(l=[]):(l=l||[]).push(c,null));for(c in r){var s=r[c];if(a=i!=null?i[c]:void 0,r.hasOwnProperty(c)&&s!==a&&(s!=null||a!=null))if(c==="style")if(a){for(o in a)!a.hasOwnProperty(o)||s&&s.hasOwnProperty(o)||(n||(n={}),n[o]="");for(o in s)s.hasOwnProperty(o)&&a[o]!==s[o]&&(n||(n={}),n[o]=s[o])}else n||(l||(l=[]),l.push(c,n)),n=s;else c==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,a=a?a.__html:void 0,s!=null&&a!==s&&(l=l||[]).push(c,s)):c==="children"?typeof s!="string"&&typeof s!="number"||(l=l||[]).push(c,""+s):c!=="suppressContentEditableWarning"&&c!=="suppressHydrationWarning"&&(Ur.hasOwnProperty(c)?(s!=null&&c==="onScroll"&&ue("scroll",e),l||a===s||(l=[])):(l=l||[]).push(c,s))}n&&(l=l||[]).push("style",n);var c=l;(t.updateQueue=c)&&(t.flags|=4)}};Vf=function(e,t,n,r){n!==r&&(t.flags|=4)};function Sr(e,t){if(!de)switch(e.tailMode){case"hidden":t=e.tail;for(var n=null;t!==null;)t.alternate!==null&&(n=t),t=t.sibling;n===null?e.tail=null:n.sibling=null;break;case"collapsed":n=e.tail;for(var r=null;n!==null;)n.alternate!==null&&(r=n),n=n.sibling;r===null?t||e.tail===null?e.tail=null:e.tail.sibling=null:r.sibling=null}}function ze(e){var t=e.alternate!==null&&e.alternate.child===e.child,n=0,r=0;if(t)for(var i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags&14680064,r|=i.flags&14680064,i.return=e,i=i.sibling;else for(i=e.child;i!==null;)n|=i.lanes|i.childLanes,r|=i.subtreeFlags,r|=i.flags,i.return=e,i=i.sibling;return e.subtreeFlags|=r,e.childLanes=n,t}function rg(e,t,n){var r=t.pendingProps;switch(ts(t),t.tag){case 2:case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return ze(t),null;case 1:return We(t.type)&&ll(),ze(t),null;case 3:return r=t.stateNode,or(),ce(Ve),ce(Pe),cs(),r.pendingContext&&(r.context=r.pendingContext,r.pendingContext=null),(e===null||e.child===null)&&(Ci(t)?t.flags|=4:e===null||e.memoizedState.isDehydrated&&!(t.flags&256)||(t.flags|=1024,vt!==null&&(ka(vt),vt=null))),fa(e,t),ze(t),null;case 5:us(t);var i=wn(ei.current);if(n=t.type,e!==null&&t.stateNode!=null)Hf(e,t,n,r,i),e.ref!==t.ref&&(t.flags|=512,t.flags|=2097152);else{if(!r){if(t.stateNode===null)throw Error(M(166));return ze(t),null}if(e=wn(Nt.current),Ci(t)){r=t.stateNode,n=t.type;var l=t.memoizedProps;switch(r[Ct]=t,r[Jr]=l,e=(t.mode&1)!==0,n){case"dialog":ue("cancel",r),ue("close",r);break;case"iframe":case"object":case"embed":ue("load",r);break;case"video":case"audio":for(i=0;i<zr.length;i++)ue(zr[i],r);break;case"source":ue("error",r);break;case"img":case"image":case"link":ue("error",r),ue("load",r);break;case"details":ue("toggle",r);break;case"input":eu(r,l),ue("invalid",r);break;case"select":r._wrapperState={wasMultiple:!!l.multiple},ue("invalid",r);break;case"textarea":nu(r,l),ue("invalid",r)}Oo(n,l),i=null;for(var o in l)if(l.hasOwnProperty(o)){var a=l[o];o==="children"?typeof a=="string"?r.textContent!==a&&(l.suppressHydrationWarning!==!0&&bi(r.textContent,a,e),i=["children",a]):typeof a=="number"&&r.textContent!==""+a&&(l.suppressHydrationWarning!==!0&&bi(r.textContent,a,e),i=["children",""+a]):Ur.hasOwnProperty(o)&&a!=null&&o==="onScroll"&&ue("scroll",r)}switch(n){case"input":mi(r),tu(r,l,!0);break;case"textarea":mi(r),ru(r);break;case"select":case"option":break;default:typeof l.onClick=="function"&&(r.onclick=il)}r=i,t.updateQueue=r,r!==null&&(t.flags|=4)}else{o=i.nodeType===9?i:i.ownerDocument,e==="http://www.w3.org/1999/xhtml"&&(e=vd(n)),e==="http://www.w3.org/1999/xhtml"?n==="script"?(e=o.createElement("div"),e.innerHTML="<script><\/script>",e=e.removeChild(e.firstChild)):typeof r.is=="string"?e=o.createElement(n,{is:r.is}):(e=o.createElement(n),n==="select"&&(o=e,r.multiple?o.multiple=!0:r.size&&(o.size=r.size))):e=o.createElementNS(e,n),e[Ct]=t,e[Jr]=r,Uf(e,t,!1,!1),t.stateNode=e;e:{switch(o=Bo(n,r),n){case"dialog":ue("cancel",e),ue("close",e),i=r;break;case"iframe":case"object":case"embed":ue("load",e),i=r;break;case"video":case"audio":for(i=0;i<zr.length;i++)ue(zr[i],e);i=r;break;case"source":ue("error",e),i=r;break;case"img":case"image":case"link":ue("error",e),ue("load",e),i=r;break;case"details":ue("toggle",e),i=r;break;case"input":eu(e,r),i=Mo(e,r),ue("invalid",e);break;case"option":i=r;break;case"select":e._wrapperState={wasMultiple:!!r.multiple},i=he({},r,{value:void 0}),ue("invalid",e);break;case"textarea":nu(e,r),i=Ro(e,r),ue("invalid",e);break;default:i=r}Oo(n,i),a=i;for(l in a)if(a.hasOwnProperty(l)){var s=a[l];l==="style"?kd(e,s):l==="dangerouslySetInnerHTML"?(s=s?s.__html:void 0,s!=null&&yd(e,s)):l==="children"?typeof s=="string"?(n!=="textarea"||s!=="")&&Hr(e,s):typeof s=="number"&&Hr(e,""+s):l!=="suppressContentEditableWarning"&&l!=="suppressHydrationWarning"&&l!=="autoFocus"&&(Ur.hasOwnProperty(l)?s!=null&&l==="onScroll"&&ue("scroll",e):s!=null&&Ba(e,l,s,o))}switch(n){case"input":mi(e),tu(e,r,!1);break;case"textarea":mi(e),ru(e);break;case"option":r.value!=null&&e.setAttribute("value",""+un(r.value));break;case"select":e.multiple=!!r.multiple,l=r.value,l!=null?Kn(e,!!r.multiple,l,!1):r.defaultValue!=null&&Kn(e,!!r.multiple,r.defaultValue,!0);break;default:typeof i.onClick=="function"&&(e.onclick=il)}switch(n){case"button":case"input":case"select":case"textarea":r=!!r.autoFocus;break e;case"img":r=!0;break e;default:r=!1}}r&&(t.flags|=4)}t.ref!==null&&(t.flags|=512,t.flags|=2097152)}return ze(t),null;case 6:if(e&&t.stateNode!=null)Vf(e,t,e.memoizedProps,r);else{if(typeof r!="string"&&t.stateNode===null)throw Error(M(166));if(n=wn(ei.current),wn(Nt.current),Ci(t)){if(r=t.stateNode,n=t.memoizedProps,r[Ct]=t,(l=r.nodeValue!==n)&&(e=tt,e!==null))switch(e.tag){case 3:bi(r.nodeValue,n,(e.mode&1)!==0);break;case 5:e.memoizedProps.suppressHydrationWarning!==!0&&bi(r.nodeValue,n,(e.mode&1)!==0)}l&&(t.flags|=4)}else r=(n.nodeType===9?n:n.ownerDocument).createTextNode(r),r[Ct]=t,t.stateNode=r}return ze(t),null;case 13:if(ce(fe),r=t.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(de&&Ze!==null&&t.mode&1&&!(t.flags&128))sf(),ir(),t.flags|=98560,l=!1;else if(l=Ci(t),r!==null&&r.dehydrated!==null){if(e===null){if(!l)throw Error(M(318));if(l=t.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(M(317));l[Ct]=t}else ir(),!(t.flags&128)&&(t.memoizedState=null),t.flags|=4;ze(t),l=!1}else vt!==null&&(ka(vt),vt=null),l=!0;if(!l)return t.flags&65536?t:null}return t.flags&128?(t.lanes=n,t):(r=r!==null,r!==(e!==null&&e.memoizedState!==null)&&r&&(t.child.flags|=8192,t.mode&1&&(e===null||fe.current&1?ke===0&&(ke=3):Ss())),t.updateQueue!==null&&(t.flags|=4),ze(t),null);case 4:return or(),fa(e,t),e===null&&Xr(t.stateNode.containerInfo),ze(t),null;case 10:return ls(t.type._context),ze(t),null;case 17:return We(t.type)&&ll(),ze(t),null;case 19:if(ce(fe),l=t.memoizedState,l===null)return ze(t),null;if(r=(t.flags&128)!==0,o=l.rendering,o===null)if(r)Sr(l,!1);else{if(ke!==0||e!==null&&e.flags&128)for(e=t.child;e!==null;){if(o=fl(e),o!==null){for(t.flags|=128,Sr(l,!1),r=o.updateQueue,r!==null&&(t.updateQueue=r,t.flags|=4),t.subtreeFlags=0,r=n,n=t.child;n!==null;)l=n,e=r,l.flags&=14680066,o=l.alternate,o===null?(l.childLanes=0,l.lanes=e,l.child=null,l.subtreeFlags=0,l.memoizedProps=null,l.memoizedState=null,l.updateQueue=null,l.dependencies=null,l.stateNode=null):(l.childLanes=o.childLanes,l.lanes=o.lanes,l.child=o.child,l.subtreeFlags=0,l.deletions=null,l.memoizedProps=o.memoizedProps,l.memoizedState=o.memoizedState,l.updateQueue=o.updateQueue,l.type=o.type,e=o.dependencies,l.dependencies=e===null?null:{lanes:e.lanes,firstContext:e.firstContext}),n=n.sibling;return ae(fe,fe.current&1|2),t.child}e=e.sibling}l.tail!==null&&ge()>sr&&(t.flags|=128,r=!0,Sr(l,!1),t.lanes=4194304)}else{if(!r)if(e=fl(o),e!==null){if(t.flags|=128,r=!0,n=e.updateQueue,n!==null&&(t.updateQueue=n,t.flags|=4),Sr(l,!0),l.tail===null&&l.tailMode==="hidden"&&!o.alternate&&!de)return ze(t),null}else 2*ge()-l.renderingStartTime>sr&&n!==1073741824&&(t.flags|=128,r=!0,Sr(l,!1),t.lanes=4194304);l.isBackwards?(o.sibling=t.child,t.child=o):(n=l.last,n!==null?n.sibling=o:t.child=o,l.last=o)}return l.tail!==null?(t=l.tail,l.rendering=t,l.tail=t.sibling,l.renderingStartTime=ge(),t.sibling=null,n=fe.current,ae(fe,r?n&1|2:n&1),t):(ze(t),null);case 22:case 23:return ws(),r=t.memoizedState!==null,e!==null&&e.memoizedState!==null!==r&&(t.flags|=8192),r&&t.mode&1?Je&1073741824&&(ze(t),t.subtreeFlags&6&&(t.flags|=8192)):ze(t),null;case 24:return null;case 25:return null}throw Error(M(156,t.tag))}function ig(e,t){switch(ts(t),t.tag){case 1:return We(t.type)&&ll(),e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 3:return or(),ce(Ve),ce(Pe),cs(),e=t.flags,e&65536&&!(e&128)?(t.flags=e&-65537|128,t):null;case 5:return us(t),null;case 13:if(ce(fe),e=t.memoizedState,e!==null&&e.dehydrated!==null){if(t.alternate===null)throw Error(M(340));ir()}return e=t.flags,e&65536?(t.flags=e&-65537|128,t):null;case 19:return ce(fe),null;case 4:return or(),null;case 10:return ls(t.type._context),null;case 22:case 23:return ws(),null;case 24:return null;default:return null}}var Ni=!1,Le=!1,lg=typeof WeakSet=="function"?WeakSet:Set,B=null;function Qn(e,t){var n=e.ref;if(n!==null)if(typeof n=="function")try{n(null)}catch(r){me(e,t,r)}else n.current=null}function pa(e,t,n){try{n()}catch(r){me(e,t,r)}}var Qu=!1;function og(e,t){if(Xo=tl,e=Kd(),Za(e)){if("selectionStart"in e)var n={start:e.selectionStart,end:e.selectionEnd};else e:{n=(n=e.ownerDocument)&&n.defaultView||window;var r=n.getSelection&&n.getSelection();if(r&&r.rangeCount!==0){n=r.anchorNode;var i=r.anchorOffset,l=r.focusNode;r=r.focusOffset;try{n.nodeType,l.nodeType}catch{n=null;break e}var o=0,a=-1,s=-1,c=0,d=0,f=e,m=null;t:for(;;){for(var p;f!==n||i!==0&&f.nodeType!==3||(a=o+i),f!==l||r!==0&&f.nodeType!==3||(s=o+r),f.nodeType===3&&(o+=f.nodeValue.length),(p=f.firstChild)!==null;)m=f,f=p;for(;;){if(f===e)break t;if(m===n&&++c===i&&(a=o),m===l&&++d===r&&(s=o),(p=f.nextSibling)!==null)break;f=m,m=f.parentNode}f=p}n=a===-1||s===-1?null:{start:a,end:s}}else n=null}n=n||{start:0,end:0}}else n=null;for(Go={focusedElem:e,selectionRange:n},tl=!1,B=t;B!==null;)if(t=B,e=t.child,(t.subtreeFlags&1028)!==0&&e!==null)e.return=t,B=e;else for(;B!==null;){t=B;try{var w=t.alternate;if(t.flags&1024)switch(t.tag){case 0:case 11:case 15:break;case 1:if(w!==null){var S=w.memoizedProps,I=w.memoizedState,h=t.stateNode,v=h.getSnapshotBeforeUpdate(t.elementType===t.type?S:mt(t.type,S),I);h.__reactInternalSnapshotBeforeUpdate=v}break;case 3:var y=t.stateNode.containerInfo;y.nodeType===1?y.textContent="":y.nodeType===9&&y.documentElement&&y.removeChild(y.documentElement);break;case 5:case 6:case 4:case 17:break;default:throw Error(M(163))}}catch(C){me(t,t.return,C)}if(e=t.sibling,e!==null){e.return=t.return,B=e;break}B=t.return}return w=Qu,Qu=!1,w}function Dr(e,t,n){var r=t.updateQueue;if(r=r!==null?r.lastEffect:null,r!==null){var i=r=r.next;do{if((i.tag&e)===e){var l=i.destroy;i.destroy=void 0,l!==void 0&&pa(t,n,l)}i=i.next}while(i!==r)}}function Ll(e,t){if(t=t.updateQueue,t=t!==null?t.lastEffect:null,t!==null){var n=t=t.next;do{if((n.tag&e)===e){var r=n.create;n.destroy=r()}n=n.next}while(n!==t)}}function ha(e){var t=e.ref;if(t!==null){var n=e.stateNode;switch(e.tag){case 5:e=n;break;default:e=n}typeof t=="function"?t(e):t.current=e}}function Wf(e){var t=e.alternate;t!==null&&(e.alternate=null,Wf(t)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(t=e.stateNode,t!==null&&(delete t[Ct],delete t[Jr],delete t[ea],delete t[Um],delete t[Hm])),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}function Qf(e){return e.tag===5||e.tag===3||e.tag===4}function qu(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||Qf(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function ma(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.nodeType===8?n.parentNode.insertBefore(e,t):n.insertBefore(e,t):(n.nodeType===8?(t=n.parentNode,t.insertBefore(e,n)):(t=n,t.appendChild(e)),n=n._reactRootContainer,n!=null||t.onclick!==null||(t.onclick=il));else if(r!==4&&(e=e.child,e!==null))for(ma(e,t,n),e=e.sibling;e!==null;)ma(e,t,n),e=e.sibling}function ga(e,t,n){var r=e.tag;if(r===5||r===6)e=e.stateNode,t?n.insertBefore(e,t):n.appendChild(e);else if(r!==4&&(e=e.child,e!==null))for(ga(e,t,n),e=e.sibling;e!==null;)ga(e,t,n),e=e.sibling}var je=null,gt=!1;function Wt(e,t,n){for(n=n.child;n!==null;)qf(e,t,n),n=n.sibling}function qf(e,t,n){if(Et&&typeof Et.onCommitFiberUnmount=="function")try{Et.onCommitFiberUnmount(bl,n)}catch{}switch(n.tag){case 5:Le||Qn(n,t);case 6:var r=je,i=gt;je=null,Wt(e,t,n),je=r,gt=i,je!==null&&(gt?(e=je,n=n.stateNode,e.nodeType===8?e.parentNode.removeChild(n):e.removeChild(n)):je.removeChild(n.stateNode));break;case 18:je!==null&&(gt?(e=je,n=n.stateNode,e.nodeType===8?io(e.parentNode,n):e.nodeType===1&&io(e,n),qr(e)):io(je,n.stateNode));break;case 4:r=je,i=gt,je=n.stateNode.containerInfo,gt=!0,Wt(e,t,n),je=r,gt=i;break;case 0:case 11:case 14:case 15:if(!Le&&(r=n.updateQueue,r!==null&&(r=r.lastEffect,r!==null))){i=r=r.next;do{var l=i,o=l.destroy;l=l.tag,o!==void 0&&(l&2||l&4)&&pa(n,t,o),i=i.next}while(i!==r)}Wt(e,t,n);break;case 1:if(!Le&&(Qn(n,t),r=n.stateNode,typeof r.componentWillUnmount=="function"))try{r.props=n.memoizedProps,r.state=n.memoizedState,r.componentWillUnmount()}catch(a){me(n,t,a)}Wt(e,t,n);break;case 21:Wt(e,t,n);break;case 22:n.mode&1?(Le=(r=Le)||n.memoizedState!==null,Wt(e,t,n),Le=r):Wt(e,t,n);break;default:Wt(e,t,n)}}function Ku(e){var t=e.updateQueue;if(t!==null){e.updateQueue=null;var n=e.stateNode;n===null&&(n=e.stateNode=new lg),t.forEach(function(r){var i=mg.bind(null,e,r);n.has(r)||(n.add(r),r.then(i,i))})}}function ht(e,t){var n=t.deletions;if(n!==null)for(var r=0;r<n.length;r++){var i=n[r];try{var l=e,o=t,a=o;e:for(;a!==null;){switch(a.tag){case 5:je=a.stateNode,gt=!1;break e;case 3:je=a.stateNode.containerInfo,gt=!0;break e;case 4:je=a.stateNode.containerInfo,gt=!0;break e}a=a.return}if(je===null)throw Error(M(160));qf(l,o,i),je=null,gt=!1;var s=i.alternate;s!==null&&(s.return=null),i.return=null}catch(c){me(i,t,c)}}if(t.subtreeFlags&12854)for(t=t.child;t!==null;)Kf(t,e),t=t.sibling}function Kf(e,t){var n=e.alternate,r=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(ht(t,e),wt(e),r&4){try{Dr(3,e,e.return),Ll(3,e)}catch(S){me(e,e.return,S)}try{Dr(5,e,e.return)}catch(S){me(e,e.return,S)}}break;case 1:ht(t,e),wt(e),r&512&&n!==null&&Qn(n,n.return);break;case 5:if(ht(t,e),wt(e),r&512&&n!==null&&Qn(n,n.return),e.flags&32){var i=e.stateNode;try{Hr(i,"")}catch(S){me(e,e.return,S)}}if(r&4&&(i=e.stateNode,i!=null)){var l=e.memoizedProps,o=n!==null?n.memoizedProps:l,a=e.type,s=e.updateQueue;if(e.updateQueue=null,s!==null)try{a==="input"&&l.type==="radio"&&l.name!=null&&md(i,l),Bo(a,o);var c=Bo(a,l);for(o=0;o<s.length;o+=2){var d=s[o],f=s[o+1];d==="style"?kd(i,f):d==="dangerouslySetInnerHTML"?yd(i,f):d==="children"?Hr(i,f):Ba(i,d,f,c)}switch(a){case"input":Ao(i,l);break;case"textarea":gd(i,l);break;case"select":var m=i._wrapperState.wasMultiple;i._wrapperState.wasMultiple=!!l.multiple;var p=l.value;p!=null?Kn(i,!!l.multiple,p,!1):m!==!!l.multiple&&(l.defaultValue!=null?Kn(i,!!l.multiple,l.defaultValue,!0):Kn(i,!!l.multiple,l.multiple?[]:"",!1))}i[Jr]=l}catch(S){me(e,e.return,S)}}break;case 6:if(ht(t,e),wt(e),r&4){if(e.stateNode===null)throw Error(M(162));i=e.stateNode,l=e.memoizedProps;try{i.nodeValue=l}catch(S){me(e,e.return,S)}}break;case 3:if(ht(t,e),wt(e),r&4&&n!==null&&n.memoizedState.isDehydrated)try{qr(t.containerInfo)}catch(S){me(e,e.return,S)}break;case 4:ht(t,e),wt(e);break;case 13:ht(t,e),wt(e),i=e.child,i.flags&8192&&(l=i.memoizedState!==null,i.stateNode.isHidden=l,!l||i.alternate!==null&&i.alternate.memoizedState!==null||(xs=ge())),r&4&&Ku(e);break;case 22:if(d=n!==null&&n.memoizedState!==null,e.mode&1?(Le=(c=Le)||d,ht(t,e),Le=c):ht(t,e),wt(e),r&8192){if(c=e.memoizedState!==null,(e.stateNode.isHidden=c)&&!d&&e.mode&1)for(B=e,d=e.child;d!==null;){for(f=B=d;B!==null;){switch(m=B,p=m.child,m.tag){case 0:case 11:case 14:case 15:Dr(4,m,m.return);break;case 1:Qn(m,m.return);var w=m.stateNode;if(typeof w.componentWillUnmount=="function"){r=m,n=m.return;try{t=r,w.props=t.memoizedProps,w.state=t.memoizedState,w.componentWillUnmount()}catch(S){me(r,n,S)}}break;case 5:Qn(m,m.return);break;case 22:if(m.memoizedState!==null){Xu(f);continue}}p!==null?(p.return=m,B=p):Xu(f)}d=d.sibling}e:for(d=null,f=e;;){if(f.tag===5){if(d===null){d=f;try{i=f.stateNode,c?(l=i.style,typeof l.setProperty=="function"?l.setProperty("display","none","important"):l.display="none"):(a=f.stateNode,s=f.memoizedProps.style,o=s!=null&&s.hasOwnProperty("display")?s.display:null,a.style.display=xd("display",o))}catch(S){me(e,e.return,S)}}}else if(f.tag===6){if(d===null)try{f.stateNode.nodeValue=c?"":f.memoizedProps}catch(S){me(e,e.return,S)}}else if((f.tag!==22&&f.tag!==23||f.memoizedState===null||f===e)&&f.child!==null){f.child.return=f,f=f.child;continue}if(f===e)break e;for(;f.sibling===null;){if(f.return===null||f.return===e)break e;d===f&&(d=null),f=f.return}d===f&&(d=null),f.sibling.return=f.return,f=f.sibling}}break;case 19:ht(t,e),wt(e),r&4&&Ku(e);break;case 21:break;default:ht(t,e),wt(e)}}function wt(e){var t=e.flags;if(t&2){try{e:{for(var n=e.return;n!==null;){if(Qf(n)){var r=n;break e}n=n.return}throw Error(M(160))}switch(r.tag){case 5:var i=r.stateNode;r.flags&32&&(Hr(i,""),r.flags&=-33);var l=qu(e);ga(e,l,i);break;case 3:case 4:var o=r.stateNode.containerInfo,a=qu(e);ma(e,a,o);break;default:throw Error(M(161))}}catch(s){me(e,e.return,s)}e.flags&=-3}t&4096&&(e.flags&=-4097)}function ag(e,t,n){B=e,Yf(e)}function Yf(e,t,n){for(var r=(e.mode&1)!==0;B!==null;){var i=B,l=i.child;if(i.tag===22&&r){var o=i.memoizedState!==null||Ni;if(!o){var a=i.alternate,s=a!==null&&a.memoizedState!==null||Le;a=Ni;var c=Le;if(Ni=o,(Le=s)&&!c)for(B=i;B!==null;)o=B,s=o.child,o.tag===22&&o.memoizedState!==null?Gu(i):s!==null?(s.return=o,B=s):Gu(i);for(;l!==null;)B=l,Yf(l),l=l.sibling;B=i,Ni=a,Le=c}Yu(e)}else i.subtreeFlags&8772&&l!==null?(l.return=i,B=l):Yu(e)}}function Yu(e){for(;B!==null;){var t=B;if(t.flags&8772){var n=t.alternate;try{if(t.flags&8772)switch(t.tag){case 0:case 11:case 15:Le||Ll(5,t);break;case 1:var r=t.stateNode;if(t.flags&4&&!Le)if(n===null)r.componentDidMount();else{var i=t.elementType===t.type?n.memoizedProps:mt(t.type,n.memoizedProps);r.componentDidUpdate(i,n.memoizedState,r.__reactInternalSnapshotBeforeUpdate)}var l=t.updateQueue;l!==null&&Iu(t,l,r);break;case 3:var o=t.updateQueue;if(o!==null){if(n=null,t.child!==null)switch(t.child.tag){case 5:n=t.child.stateNode;break;case 1:n=t.child.stateNode}Iu(t,o,n)}break;case 5:var a=t.stateNode;if(n===null&&t.flags&4){n=a;var s=t.memoizedProps;switch(t.type){case"button":case"input":case"select":case"textarea":s.autoFocus&&n.focus();break;case"img":s.src&&(n.src=s.src)}}break;case 6:break;case 4:break;case 12:break;case 13:if(t.memoizedState===null){var c=t.alternate;if(c!==null){var d=c.memoizedState;if(d!==null){var f=d.dehydrated;f!==null&&qr(f)}}}break;case 19:case 17:case 21:case 22:case 23:case 25:break;default:throw Error(M(163))}Le||t.flags&512&&ha(t)}catch(m){me(t,t.return,m)}}if(t===e){B=null;break}if(n=t.sibling,n!==null){n.return=t.return,B=n;break}B=t.return}}function Xu(e){for(;B!==null;){var t=B;if(t===e){B=null;break}var n=t.sibling;if(n!==null){n.return=t.return,B=n;break}B=t.return}}function Gu(e){for(;B!==null;){var t=B;try{switch(t.tag){case 0:case 11:case 15:var n=t.return;try{Ll(4,t)}catch(s){me(t,n,s)}break;case 1:var r=t.stateNode;if(typeof r.componentDidMount=="function"){var i=t.return;try{r.componentDidMount()}catch(s){me(t,i,s)}}var l=t.return;try{ha(t)}catch(s){me(t,l,s)}break;case 5:var o=t.return;try{ha(t)}catch(s){me(t,o,s)}}}catch(s){me(t,t.return,s)}if(t===e){B=null;break}var a=t.sibling;if(a!==null){a.return=t.return,B=a;break}B=t.return}}var sg=Math.ceil,ml=Ut.ReactCurrentDispatcher,vs=Ut.ReactCurrentOwner,ct=Ut.ReactCurrentBatchConfig,Z=0,Se=null,ye=null,Ee=0,Je=0,qn=fn(0),ke=0,ii=null,Nn=0,Pl=0,ys=0,Rr=null,Ue=null,xs=0,sr=1/0,It=null,gl=!1,va=null,on=null,_i=!1,Zt=null,vl=0,Fr=0,ya=null,Vi=-1,Wi=0;function Re(){return Z&6?ge():Vi!==-1?Vi:Vi=ge()}function an(e){return e.mode&1?Z&2&&Ee!==0?Ee&-Ee:Wm.transition!==null?(Wi===0&&(Wi=Pd()),Wi):(e=re,e!==0||(e=window.event,e=e===void 0?16:Od(e.type)),e):1}function xt(e,t,n,r){if(50<Fr)throw Fr=0,ya=null,Error(M(185));ai(e,n,r),(!(Z&2)||e!==Se)&&(e===Se&&(!(Z&2)&&(Pl|=n),ke===4&&Gt(e,Ee)),Qe(e,r),n===1&&Z===0&&!(t.mode&1)&&(sr=ge()+500,_l&&pn()))}function Qe(e,t){var n=e.callbackNode;Wh(e,t);var r=el(e,e===Se?Ee:0);if(r===0)n!==null&&ou(n),e.callbackNode=null,e.callbackPriority=0;else if(t=r&-r,e.callbackPriority!==t){if(n!=null&&ou(n),t===1)e.tag===0?Vm(Ju.bind(null,e)):lf(Ju.bind(null,e)),Bm(function(){!(Z&6)&&pn()}),n=null;else{switch(Id(r)){case 1:n=Wa;break;case 4:n=Td;break;case 16:n=Zi;break;case 536870912:n=Ld;break;default:n=Zi}n=rp(n,Xf.bind(null,e))}e.callbackPriority=t,e.callbackNode=n}}function Xf(e,t){if(Vi=-1,Wi=0,Z&6)throw Error(M(327));var n=e.callbackNode;if(Zn()&&e.callbackNode!==n)return null;var r=el(e,e===Se?Ee:0);if(r===0)return null;if(r&30||r&e.expiredLanes||t)t=yl(e,r);else{t=r;var i=Z;Z|=2;var l=Jf();(Se!==e||Ee!==t)&&(It=null,sr=ge()+500,Sn(e,t));do try{dg();break}catch(a){Gf(e,a)}while(!0);is(),ml.current=l,Z=i,ye!==null?t=0:(Se=null,Ee=0,t=ke)}if(t!==0){if(t===2&&(i=Wo(e),i!==0&&(r=i,t=xa(e,i))),t===1)throw n=ii,Sn(e,0),Gt(e,r),Qe(e,ge()),n;if(t===6)Gt(e,r);else{if(i=e.current.alternate,!(r&30)&&!ug(i)&&(t=yl(e,r),t===2&&(l=Wo(e),l!==0&&(r=l,t=xa(e,l))),t===1))throw n=ii,Sn(e,0),Gt(e,r),Qe(e,ge()),n;switch(e.finishedWork=i,e.finishedLanes=r,t){case 0:case 1:throw Error(M(345));case 2:vn(e,Ue,It);break;case 3:if(Gt(e,r),(r&130023424)===r&&(t=xs+500-ge(),10<t)){if(el(e,0)!==0)break;if(i=e.suspendedLanes,(i&r)!==r){Re(),e.pingedLanes|=e.suspendedLanes&i;break}e.timeoutHandle=Zo(vn.bind(null,e,Ue,It),t);break}vn(e,Ue,It);break;case 4:if(Gt(e,r),(r&4194240)===r)break;for(t=e.eventTimes,i=-1;0<r;){var o=31-yt(r);l=1<<o,o=t[o],o>i&&(i=o),r&=~l}if(r=i,r=ge()-r,r=(120>r?120:480>r?480:1080>r?1080:1920>r?1920:3e3>r?3e3:4320>r?4320:1960*sg(r/1960))-r,10<r){e.timeoutHandle=Zo(vn.bind(null,e,Ue,It),r);break}vn(e,Ue,It);break;case 5:vn(e,Ue,It);break;default:throw Error(M(329))}}}return Qe(e,ge()),e.callbackNode===n?Xf.bind(null,e):null}function xa(e,t){var n=Rr;return e.current.memoizedState.isDehydrated&&(Sn(e,t).flags|=256),e=yl(e,t),e!==2&&(t=Ue,Ue=n,t!==null&&ka(t)),e}function ka(e){Ue===null?Ue=e:Ue.push.apply(Ue,e)}function ug(e){for(var t=e;;){if(t.flags&16384){var n=t.updateQueue;if(n!==null&&(n=n.stores,n!==null))for(var r=0;r<n.length;r++){var i=n[r],l=i.getSnapshot;i=i.value;try{if(!kt(l(),i))return!1}catch{return!1}}}if(n=t.child,t.subtreeFlags&16384&&n!==null)n.return=t,t=n;else{if(t===e)break;for(;t.sibling===null;){if(t.return===null||t.return===e)return!0;t=t.return}t.sibling.return=t.return,t=t.sibling}}return!0}function Gt(e,t){for(t&=~ys,t&=~Pl,e.suspendedLanes|=t,e.pingedLanes&=~t,e=e.expirationTimes;0<t;){var n=31-yt(t),r=1<<n;e[n]=-1,t&=~r}}function Ju(e){if(Z&6)throw Error(M(327));Zn();var t=el(e,0);if(!(t&1))return Qe(e,ge()),null;var n=yl(e,t);if(e.tag!==0&&n===2){var r=Wo(e);r!==0&&(t=r,n=xa(e,r))}if(n===1)throw n=ii,Sn(e,0),Gt(e,t),Qe(e,ge()),n;if(n===6)throw Error(M(345));return e.finishedWork=e.current.alternate,e.finishedLanes=t,vn(e,Ue,It),Qe(e,ge()),null}function ks(e,t){var n=Z;Z|=1;try{return e(t)}finally{Z=n,Z===0&&(sr=ge()+500,_l&&pn())}}function _n(e){Zt!==null&&Zt.tag===0&&!(Z&6)&&Zn();var t=Z;Z|=1;var n=ct.transition,r=re;try{if(ct.transition=null,re=1,e)return e()}finally{re=r,ct.transition=n,Z=t,!(Z&6)&&pn()}}function ws(){Je=qn.current,ce(qn)}function Sn(e,t){e.finishedWork=null,e.finishedLanes=0;var n=e.timeoutHandle;if(n!==-1&&(e.timeoutHandle=-1,Om(n)),ye!==null)for(n=ye.return;n!==null;){var r=n;switch(ts(r),r.tag){case 1:r=r.type.childContextTypes,r!=null&&ll();break;case 3:or(),ce(Ve),ce(Pe),cs();break;case 5:us(r);break;case 4:or();break;case 13:ce(fe);break;case 19:ce(fe);break;case 10:ls(r.type._context);break;case 22:case 23:ws()}n=n.return}if(Se=e,ye=e=sn(e.current,null),Ee=Je=t,ke=0,ii=null,ys=Pl=Nn=0,Ue=Rr=null,kn!==null){for(t=0;t<kn.length;t++)if(n=kn[t],r=n.interleaved,r!==null){n.interleaved=null;var i=r.next,l=n.pending;if(l!==null){var o=l.next;l.next=i,r.next=o}n.pending=r}kn=null}return e}function Gf(e,t){do{var n=ye;try{if(is(),$i.current=hl,pl){for(var r=pe.memoizedState;r!==null;){var i=r.queue;i!==null&&(i.pending=null),r=r.next}pl=!1}if(En=0,we=xe=pe=null,Ar=!1,ti=0,vs.current=null,n===null||n.return===null){ke=1,ii=t,ye=null;break}e:{var l=e,o=n.return,a=n,s=t;if(t=Ee,a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){var c=s,d=a,f=d.tag;if(!(d.mode&1)&&(f===0||f===11||f===15)){var m=d.alternate;m?(d.updateQueue=m.updateQueue,d.memoizedState=m.memoizedState,d.lanes=m.lanes):(d.updateQueue=null,d.memoizedState=null)}var p=Ou(o);if(p!==null){p.flags&=-257,Bu(p,o,a,l,t),p.mode&1&&Fu(l,c,t),t=p,s=c;var w=t.updateQueue;if(w===null){var S=new Set;S.add(s),t.updateQueue=S}else w.add(s);break e}else{if(!(t&1)){Fu(l,c,t),Ss();break e}s=Error(M(426))}}else if(de&&a.mode&1){var I=Ou(o);if(I!==null){!(I.flags&65536)&&(I.flags|=256),Bu(I,o,a,l,t),ns(ar(s,a));break e}}l=s=ar(s,a),ke!==4&&(ke=2),Rr===null?Rr=[l]:Rr.push(l),l=o;do{switch(l.tag){case 3:l.flags|=65536,t&=-t,l.lanes|=t;var h=Mf(l,s,t);Pu(l,h);break e;case 1:a=s;var v=l.type,y=l.stateNode;if(!(l.flags&128)&&(typeof v.getDerivedStateFromError=="function"||y!==null&&typeof y.componentDidCatch=="function"&&(on===null||!on.has(y)))){l.flags|=65536,t&=-t,l.lanes|=t;var C=Af(l,a,t);Pu(l,C);break e}}l=l.return}while(l!==null)}ep(n)}catch(N){t=N,ye===n&&n!==null&&(ye=n=n.return);continue}break}while(!0)}function Jf(){var e=ml.current;return ml.current=hl,e===null?hl:e}function Ss(){(ke===0||ke===3||ke===2)&&(ke=4),Se===null||!(Nn&268435455)&&!(Pl&268435455)||Gt(Se,Ee)}function yl(e,t){var n=Z;Z|=2;var r=Jf();(Se!==e||Ee!==t)&&(It=null,Sn(e,t));do try{cg();break}catch(i){Gf(e,i)}while(!0);if(is(),Z=n,ml.current=r,ye!==null)throw Error(M(261));return Se=null,Ee=0,ke}function cg(){for(;ye!==null;)Zf(ye)}function dg(){for(;ye!==null&&!Dh();)Zf(ye)}function Zf(e){var t=np(e.alternate,e,Je);e.memoizedProps=e.pendingProps,t===null?ep(e):ye=t,vs.current=null}function ep(e){var t=e;do{var n=t.alternate;if(e=t.return,t.flags&32768){if(n=ig(n,t),n!==null){n.flags&=32767,ye=n;return}if(e!==null)e.flags|=32768,e.subtreeFlags=0,e.deletions=null;else{ke=6,ye=null;return}}else if(n=rg(n,t,Je),n!==null){ye=n;return}if(t=t.sibling,t!==null){ye=t;return}ye=t=e}while(t!==null);ke===0&&(ke=5)}function vn(e,t,n){var r=re,i=ct.transition;try{ct.transition=null,re=1,fg(e,t,n,r)}finally{ct.transition=i,re=r}return null}function fg(e,t,n,r){do Zn();while(Zt!==null);if(Z&6)throw Error(M(327));n=e.finishedWork;var i=e.finishedLanes;if(n===null)return null;if(e.finishedWork=null,e.finishedLanes=0,n===e.current)throw Error(M(177));e.callbackNode=null,e.callbackPriority=0;var l=n.lanes|n.childLanes;if(Qh(e,l),e===Se&&(ye=Se=null,Ee=0),!(n.subtreeFlags&2064)&&!(n.flags&2064)||_i||(_i=!0,rp(Zi,function(){return Zn(),null})),l=(n.flags&15990)!==0,n.subtreeFlags&15990||l){l=ct.transition,ct.transition=null;var o=re;re=1;var a=Z;Z|=4,vs.current=null,og(e,n),Kf(n,e),Pm(Go),tl=!!Xo,Go=Xo=null,e.current=n,ag(n),Rh(),Z=a,re=o,ct.transition=l}else e.current=n;if(_i&&(_i=!1,Zt=e,vl=i),l=e.pendingLanes,l===0&&(on=null),Bh(n.stateNode),Qe(e,ge()),t!==null)for(r=e.onRecoverableError,n=0;n<t.length;n++)i=t[n],r(i.value,{componentStack:i.stack,digest:i.digest});if(gl)throw gl=!1,e=va,va=null,e;return vl&1&&e.tag!==0&&Zn(),l=e.pendingLanes,l&1?e===ya?Fr++:(Fr=0,ya=e):Fr=0,pn(),null}function Zn(){if(Zt!==null){var e=Id(vl),t=ct.transition,n=re;try{if(ct.transition=null,re=16>e?16:e,Zt===null)var r=!1;else{if(e=Zt,Zt=null,vl=0,Z&6)throw Error(M(331));var i=Z;for(Z|=4,B=e.current;B!==null;){var l=B,o=l.child;if(B.flags&16){var a=l.deletions;if(a!==null){for(var s=0;s<a.length;s++){var c=a[s];for(B=c;B!==null;){var d=B;switch(d.tag){case 0:case 11:case 15:Dr(8,d,l)}var f=d.child;if(f!==null)f.return=d,B=f;else for(;B!==null;){d=B;var m=d.sibling,p=d.return;if(Wf(d),d===c){B=null;break}if(m!==null){m.return=p,B=m;break}B=p}}}var w=l.alternate;if(w!==null){var S=w.child;if(S!==null){w.child=null;do{var I=S.sibling;S.sibling=null,S=I}while(S!==null)}}B=l}}if(l.subtreeFlags&2064&&o!==null)o.return=l,B=o;else e:for(;B!==null;){if(l=B,l.flags&2048)switch(l.tag){case 0:case 11:case 15:Dr(9,l,l.return)}var h=l.sibling;if(h!==null){h.return=l.return,B=h;break e}B=l.return}}var v=e.current;for(B=v;B!==null;){o=B;var y=o.child;if(o.subtreeFlags&2064&&y!==null)y.return=o,B=y;else e:for(o=v;B!==null;){if(a=B,a.flags&2048)try{switch(a.tag){case 0:case 11:case 15:Ll(9,a)}}catch(N){me(a,a.return,N)}if(a===o){B=null;break e}var C=a.sibling;if(C!==null){C.return=a.return,B=C;break e}B=a.return}}if(Z=i,pn(),Et&&typeof Et.onPostCommitFiberRoot=="function")try{Et.onPostCommitFiberRoot(bl,e)}catch{}r=!0}return r}finally{re=n,ct.transition=t}}return!1}function Zu(e,t,n){t=ar(n,t),t=Mf(e,t,1),e=ln(e,t,1),t=Re(),e!==null&&(ai(e,1,t),Qe(e,t))}function me(e,t,n){if(e.tag===3)Zu(e,e,n);else for(;t!==null;){if(t.tag===3){Zu(t,e,n);break}else if(t.tag===1){var r=t.stateNode;if(typeof t.type.getDerivedStateFromError=="function"||typeof r.componentDidCatch=="function"&&(on===null||!on.has(r))){e=ar(n,e),e=Af(t,e,1),t=ln(t,e,1),e=Re(),t!==null&&(ai(t,1,e),Qe(t,e));break}}t=t.return}}function pg(e,t,n){var r=e.pingCache;r!==null&&r.delete(t),t=Re(),e.pingedLanes|=e.suspendedLanes&n,Se===e&&(Ee&n)===n&&(ke===4||ke===3&&(Ee&130023424)===Ee&&500>ge()-xs?Sn(e,0):ys|=n),Qe(e,t)}function tp(e,t){t===0&&(e.mode&1?(t=yi,yi<<=1,!(yi&130023424)&&(yi=4194304)):t=1);var n=Re();e=Bt(e,t),e!==null&&(ai(e,t,n),Qe(e,n))}function hg(e){var t=e.memoizedState,n=0;t!==null&&(n=t.retryLane),tp(e,n)}function mg(e,t){var n=0;switch(e.tag){case 13:var r=e.stateNode,i=e.memoizedState;i!==null&&(n=i.retryLane);break;case 19:r=e.stateNode;break;default:throw Error(M(314))}r!==null&&r.delete(t),tp(e,n)}var np;np=function(e,t,n){if(e!==null)if(e.memoizedProps!==t.pendingProps||Ve.current)He=!0;else{if(!(e.lanes&n)&&!(t.flags&128))return He=!1,ng(e,t,n);He=!!(e.flags&131072)}else He=!1,de&&t.flags&1048576&&of(t,sl,t.index);switch(t.lanes=0,t.tag){case 2:var r=t.type;Hi(e,t),e=t.pendingProps;var i=rr(t,Pe.current);Jn(t,n),i=fs(null,t,r,e,i,n);var l=ps();return t.flags|=1,typeof i=="object"&&i!==null&&typeof i.render=="function"&&i.$$typeof===void 0?(t.tag=1,t.memoizedState=null,t.updateQueue=null,We(r)?(l=!0,ol(t)):l=!1,t.memoizedState=i.state!==null&&i.state!==void 0?i.state:null,as(t),i.updater=Tl,t.stateNode=i,i._reactInternals=t,oa(t,r,e,n),t=ua(null,t,r,!0,l,n)):(t.tag=0,de&&l&&es(t),De(null,t,i,n),t=t.child),t;case 16:r=t.elementType;e:{switch(Hi(e,t),e=t.pendingProps,i=r._init,r=i(r._payload),t.type=r,i=t.tag=vg(r),e=mt(r,e),i){case 0:t=sa(null,t,r,e,n);break e;case 1:t=Hu(null,t,r,e,n);break e;case 11:t=$u(null,t,r,e,n);break e;case 14:t=Uu(null,t,r,mt(r.type,e),n);break e}throw Error(M(306,r,""))}return t;case 0:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:mt(r,i),sa(e,t,r,i,n);case 1:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:mt(r,i),Hu(e,t,r,i,n);case 3:e:{if(Of(t),e===null)throw Error(M(387));r=t.pendingProps,l=t.memoizedState,i=l.element,ff(e,t),dl(t,r,null,n);var o=t.memoizedState;if(r=o.element,l.isDehydrated)if(l={element:r,isDehydrated:!1,cache:o.cache,pendingSuspenseBoundaries:o.pendingSuspenseBoundaries,transitions:o.transitions},t.updateQueue.baseState=l,t.memoizedState=l,t.flags&256){i=ar(Error(M(423)),t),t=Vu(e,t,r,n,i);break e}else if(r!==i){i=ar(Error(M(424)),t),t=Vu(e,t,r,n,i);break e}else for(Ze=rn(t.stateNode.containerInfo.firstChild),tt=t,de=!0,vt=null,n=cf(t,null,r,n),t.child=n;n;)n.flags=n.flags&-3|4096,n=n.sibling;else{if(ir(),r===i){t=$t(e,t,n);break e}De(e,t,r,n)}t=t.child}return t;case 5:return pf(t),e===null&&ra(t),r=t.type,i=t.pendingProps,l=e!==null?e.memoizedProps:null,o=i.children,Jo(r,i)?o=null:l!==null&&Jo(r,l)&&(t.flags|=32),Ff(e,t),De(e,t,o,n),t.child;case 6:return e===null&&ra(t),null;case 13:return Bf(e,t,n);case 4:return ss(t,t.stateNode.containerInfo),r=t.pendingProps,e===null?t.child=lr(t,null,r,n):De(e,t,r,n),t.child;case 11:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:mt(r,i),$u(e,t,r,i,n);case 7:return De(e,t,t.pendingProps,n),t.child;case 8:return De(e,t,t.pendingProps.children,n),t.child;case 12:return De(e,t,t.pendingProps.children,n),t.child;case 10:e:{if(r=t.type._context,i=t.pendingProps,l=t.memoizedProps,o=i.value,ae(ul,r._currentValue),r._currentValue=o,l!==null)if(kt(l.value,o)){if(l.children===i.children&&!Ve.current){t=$t(e,t,n);break e}}else for(l=t.child,l!==null&&(l.return=t);l!==null;){var a=l.dependencies;if(a!==null){o=l.child;for(var s=a.firstContext;s!==null;){if(s.context===r){if(l.tag===1){s=Rt(-1,n&-n),s.tag=2;var c=l.updateQueue;if(c!==null){c=c.shared;var d=c.pending;d===null?s.next=s:(s.next=d.next,d.next=s),c.pending=s}}l.lanes|=n,s=l.alternate,s!==null&&(s.lanes|=n),ia(l.return,n,t),a.lanes|=n;break}s=s.next}}else if(l.tag===10)o=l.type===t.type?null:l.child;else if(l.tag===18){if(o=l.return,o===null)throw Error(M(341));o.lanes|=n,a=o.alternate,a!==null&&(a.lanes|=n),ia(o,n,t),o=l.sibling}else o=l.child;if(o!==null)o.return=l;else for(o=l;o!==null;){if(o===t){o=null;break}if(l=o.sibling,l!==null){l.return=o.return,o=l;break}o=o.return}l=o}De(e,t,i.children,n),t=t.child}return t;case 9:return i=t.type,r=t.pendingProps.children,Jn(t,n),i=dt(i),r=r(i),t.flags|=1,De(e,t,r,n),t.child;case 14:return r=t.type,i=mt(r,t.pendingProps),i=mt(r.type,i),Uu(e,t,r,i,n);case 15:return Df(e,t,t.type,t.pendingProps,n);case 17:return r=t.type,i=t.pendingProps,i=t.elementType===r?i:mt(r,i),Hi(e,t),t.tag=1,We(r)?(e=!0,ol(t)):e=!1,Jn(t,n),If(t,r,i),oa(t,r,i,n),ua(null,t,r,!0,e,n);case 19:return $f(e,t,n);case 22:return Rf(e,t,n)}throw Error(M(156,t.tag))};function rp(e,t){return zd(e,t)}function gg(e,t,n,r){this.tag=e,this.key=n,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.ref=null,this.pendingProps=t,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=r,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function ut(e,t,n,r){return new gg(e,t,n,r)}function bs(e){return e=e.prototype,!(!e||!e.isReactComponent)}function vg(e){if(typeof e=="function")return bs(e)?1:0;if(e!=null){if(e=e.$$typeof,e===Ua)return 11;if(e===Ha)return 14}return 2}function sn(e,t){var n=e.alternate;return n===null?(n=ut(e.tag,t,e.key,e.mode),n.elementType=e.elementType,n.type=e.type,n.stateNode=e.stateNode,n.alternate=e,e.alternate=n):(n.pendingProps=t,n.type=e.type,n.flags=0,n.subtreeFlags=0,n.deletions=null),n.flags=e.flags&14680064,n.childLanes=e.childLanes,n.lanes=e.lanes,n.child=e.child,n.memoizedProps=e.memoizedProps,n.memoizedState=e.memoizedState,n.updateQueue=e.updateQueue,t=e.dependencies,n.dependencies=t===null?null:{lanes:t.lanes,firstContext:t.firstContext},n.sibling=e.sibling,n.index=e.index,n.ref=e.ref,n}function Qi(e,t,n,r,i,l){var o=2;if(r=e,typeof e=="function")bs(e)&&(o=1);else if(typeof e=="string")o=5;else e:switch(e){case Rn:return bn(n.children,i,l,t);case $a:o=8,i|=8;break;case To:return e=ut(12,n,t,i|2),e.elementType=To,e.lanes=l,e;case Lo:return e=ut(13,n,t,i),e.elementType=Lo,e.lanes=l,e;case Po:return e=ut(19,n,t,i),e.elementType=Po,e.lanes=l,e;case fd:return Il(n,i,l,t);default:if(typeof e=="object"&&e!==null)switch(e.$$typeof){case cd:o=10;break e;case dd:o=9;break e;case Ua:o=11;break e;case Ha:o=14;break e;case Kt:o=16,r=null;break e}throw Error(M(130,e==null?e:typeof e,""))}return t=ut(o,n,t,i),t.elementType=e,t.type=r,t.lanes=l,t}function bn(e,t,n,r){return e=ut(7,e,r,t),e.lanes=n,e}function Il(e,t,n,r){return e=ut(22,e,r,t),e.elementType=fd,e.lanes=n,e.stateNode={isHidden:!1},e}function po(e,t,n){return e=ut(6,e,null,t),e.lanes=n,e}function ho(e,t,n){return t=ut(4,e.children!==null?e.children:[],e.key,t),t.lanes=n,t.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},t}function yg(e,t,n,r,i){this.tag=t,this.containerInfo=e,this.finishedWork=this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.pendingContext=this.context=null,this.callbackPriority=0,this.eventTimes=ql(0),this.expirationTimes=ql(-1),this.entangledLanes=this.finishedLanes=this.mutableReadLanes=this.expiredLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=ql(0),this.identifierPrefix=r,this.onRecoverableError=i,this.mutableSourceEagerHydrationData=null}function Cs(e,t,n,r,i,l,o,a,s){return e=new yg(e,t,n,a,s),t===1?(t=1,l===!0&&(t|=8)):t=0,l=ut(3,null,null,t),e.current=l,l.stateNode=e,l.memoizedState={element:r,isDehydrated:n,cache:null,transitions:null,pendingSuspenseBoundaries:null},as(l),e}function xg(e,t,n){var r=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:Dn,key:r==null?null:""+r,children:e,containerInfo:t,implementation:n}}function ip(e){if(!e)return cn;e=e._reactInternals;e:{if(Tn(e)!==e||e.tag!==1)throw Error(M(170));var t=e;do{switch(t.tag){case 3:t=t.stateNode.context;break e;case 1:if(We(t.type)){t=t.stateNode.__reactInternalMemoizedMergedChildContext;break e}}t=t.return}while(t!==null);throw Error(M(171))}if(e.tag===1){var n=e.type;if(We(n))return rf(e,n,t)}return t}function lp(e,t,n,r,i,l,o,a,s){return e=Cs(n,r,!0,e,i,l,o,a,s),e.context=ip(null),n=e.current,r=Re(),i=an(n),l=Rt(r,i),l.callback=t??null,ln(n,l,i),e.current.lanes=i,ai(e,i,r),Qe(e,r),e}function Ml(e,t,n,r){var i=t.current,l=Re(),o=an(i);return n=ip(n),t.context===null?t.context=n:t.pendingContext=n,t=Rt(l,o),t.payload={element:e},r=r===void 0?null:r,r!==null&&(t.callback=r),e=ln(i,t,o),e!==null&&(xt(e,i,o,l),Bi(e,i,o)),o}function xl(e){if(e=e.current,!e.child)return null;switch(e.child.tag){case 5:return e.child.stateNode;default:return e.child.stateNode}}function ec(e,t){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var n=e.retryLane;e.retryLane=n!==0&&n<t?n:t}}function js(e,t){ec(e,t),(e=e.alternate)&&ec(e,t)}function kg(){return null}var op=typeof reportError=="function"?reportError:function(e){console.error(e)};function Es(e){this._internalRoot=e}Al.prototype.render=Es.prototype.render=function(e){var t=this._internalRoot;if(t===null)throw Error(M(409));Ml(e,t,null,null)};Al.prototype.unmount=Es.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var t=e.containerInfo;_n(function(){Ml(null,e,null,null)}),t[Ot]=null}};function Al(e){this._internalRoot=e}Al.prototype.unstable_scheduleHydration=function(e){if(e){var t=Dd();e={blockedOn:null,target:e,priority:t};for(var n=0;n<Xt.length&&t!==0&&t<Xt[n].priority;n++);Xt.splice(n,0,e),n===0&&Fd(e)}};function Ns(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function Dl(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11&&(e.nodeType!==8||e.nodeValue!==" react-mount-point-unstable "))}function tc(){}function wg(e,t,n,r,i){if(i){if(typeof r=="function"){var l=r;r=function(){var c=xl(o);l.call(c)}}var o=lp(t,r,e,0,null,!1,!1,"",tc);return e._reactRootContainer=o,e[Ot]=o.current,Xr(e.nodeType===8?e.parentNode:e),_n(),o}for(;i=e.lastChild;)e.removeChild(i);if(typeof r=="function"){var a=r;r=function(){var c=xl(s);a.call(c)}}var s=Cs(e,0,!1,null,null,!1,!1,"",tc);return e._reactRootContainer=s,e[Ot]=s.current,Xr(e.nodeType===8?e.parentNode:e),_n(function(){Ml(t,s,n,r)}),s}function Rl(e,t,n,r,i){var l=n._reactRootContainer;if(l){var o=l;if(typeof i=="function"){var a=i;i=function(){var s=xl(o);a.call(s)}}Ml(t,o,e,i)}else o=wg(n,t,e,i,r);return xl(o)}Md=function(e){switch(e.tag){case 3:var t=e.stateNode;if(t.current.memoizedState.isDehydrated){var n=_r(t.pendingLanes);n!==0&&(Qa(t,n|1),Qe(t,ge()),!(Z&6)&&(sr=ge()+500,pn()))}break;case 13:_n(function(){var r=Bt(e,1);if(r!==null){var i=Re();xt(r,e,1,i)}}),js(e,1)}};qa=function(e){if(e.tag===13){var t=Bt(e,134217728);if(t!==null){var n=Re();xt(t,e,134217728,n)}js(e,134217728)}};Ad=function(e){if(e.tag===13){var t=an(e),n=Bt(e,t);if(n!==null){var r=Re();xt(n,e,t,r)}js(e,t)}};Dd=function(){return re};Rd=function(e,t){var n=re;try{return re=e,t()}finally{re=n}};Uo=function(e,t,n){switch(t){case"input":if(Ao(e,n),t=n.name,n.type==="radio"&&t!=null){for(n=e;n.parentNode;)n=n.parentNode;for(n=n.querySelectorAll("input[name="+JSON.stringify(""+t)+'][type="radio"]'),t=0;t<n.length;t++){var r=n[t];if(r!==e&&r.form===e.form){var i=Nl(r);if(!i)throw Error(M(90));hd(r),Ao(r,i)}}}break;case"textarea":gd(e,n);break;case"select":t=n.value,t!=null&&Kn(e,!!n.multiple,t,!1)}};bd=ks;Cd=_n;var Sg={usingClientEntryPoint:!1,Events:[ui,$n,Nl,wd,Sd,ks]},br={findFiberByHostInstance:xn,bundleType:0,version:"18.3.1",rendererPackageName:"react-dom"},bg={bundleType:br.bundleType,version:br.version,rendererPackageName:br.rendererPackageName,rendererConfig:br.rendererConfig,overrideHookState:null,overrideHookStateDeletePath:null,overrideHookStateRenamePath:null,overrideProps:null,overridePropsDeletePath:null,overridePropsRenamePath:null,setErrorHandler:null,setSuspenseHandler:null,scheduleUpdate:null,currentDispatcherRef:Ut.ReactCurrentDispatcher,findHostInstanceByFiber:function(e){return e=Nd(e),e===null?null:e.stateNode},findFiberByHostInstance:br.findFiberByHostInstance||kg,findHostInstancesForRefresh:null,scheduleRefresh:null,scheduleRoot:null,setRefreshHandler:null,getCurrentFiber:null,reconcilerVersion:"18.3.1-next-f1338f8080-20240426"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var zi=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!zi.isDisabled&&zi.supportsFiber)try{bl=zi.inject(bg),Et=zi}catch{}}rt.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED=Sg;rt.createPortal=function(e,t){var n=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!Ns(t))throw Error(M(200));return xg(e,t,null,n)};rt.createRoot=function(e,t){if(!Ns(e))throw Error(M(299));var n=!1,r="",i=op;return t!=null&&(t.unstable_strictMode===!0&&(n=!0),t.identifierPrefix!==void 0&&(r=t.identifierPrefix),t.onRecoverableError!==void 0&&(i=t.onRecoverableError)),t=Cs(e,1,!1,null,null,n,!1,r,i),e[Ot]=t.current,Xr(e.nodeType===8?e.parentNode:e),new Es(t)};rt.findDOMNode=function(e){if(e==null)return null;if(e.nodeType===1)return e;var t=e._reactInternals;if(t===void 0)throw typeof e.render=="function"?Error(M(188)):(e=Object.keys(e).join(","),Error(M(268,e)));return e=Nd(t),e=e===null?null:e.stateNode,e};rt.flushSync=function(e){return _n(e)};rt.hydrate=function(e,t,n){if(!Dl(t))throw Error(M(200));return Rl(null,e,t,!0,n)};rt.hydrateRoot=function(e,t,n){if(!Ns(e))throw Error(M(405));var r=n!=null&&n.hydratedSources||null,i=!1,l="",o=op;if(n!=null&&(n.unstable_strictMode===!0&&(i=!0),n.identifierPrefix!==void 0&&(l=n.identifierPrefix),n.onRecoverableError!==void 0&&(o=n.onRecoverableError)),t=lp(t,null,e,1,n??null,i,!1,l,o),e[Ot]=t.current,Xr(e),r)for(e=0;e<r.length;e++)n=r[e],i=n._getVersion,i=i(n._source),t.mutableSourceEagerHydrationData==null?t.mutableSourceEagerHydrationData=[n,i]:t.mutableSourceEagerHydrationData.push(n,i);return new Al(t)};rt.render=function(e,t,n){if(!Dl(t))throw Error(M(200));return Rl(null,e,t,!1,n)};rt.unmountComponentAtNode=function(e){if(!Dl(e))throw Error(M(40));return e._reactRootContainer?(_n(function(){Rl(null,null,e,!1,function(){e._reactRootContainer=null,e[Ot]=null})}),!0):!1};rt.unstable_batchedUpdates=ks;rt.unstable_renderSubtreeIntoContainer=function(e,t,n,r){if(!Dl(n))throw Error(M(200));if(e==null||e._reactInternals===void 0)throw Error(M(38));return Rl(e,t,n,!1,r)};rt.version="18.3.1-next-f1338f8080-20240426";function ap(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(ap)}catch(e){console.error(e)}}ap(),od.exports=rt;var Cg=od.exports,nc=Cg;_o.createRoot=nc.createRoot,_o.hydrateRoot=nc.hydrateRoot;const jg="",Eg=({selection:e,onSelect:t,onRefresh:n})=>{const[r,i]=O.useState(null),[l,o]=O.useState(new Set(["all"])),[a,s]=O.useState(!0),[c,d]=O.useState(null),f=async()=>{try{const v=await fetch(`${jg}/api/hierarchy`);if(!v.ok)throw new Error("Failed to fetch hierarchy");const y=await v.json();i(y),d(null)}catch(v){d(v instanceof Error?v.message:"Unknown error")}finally{s(!1)}};O.useEffect(()=>{f();const v=setInterval(f,5e3);return()=>clearInterval(v)},[]);const m=v=>{o(y=>{const C=new Set(y);return C.has(v)?C.delete(v):C.add(v),C})},p=v=>{var y;if(v.type==="root")t({type:"overview"});else if(v.type==="agent")t({type:"agent",agentId:v.id});else if(v.type==="thread"){const C=(y=r==null?void 0:r.root.children)==null?void 0:y.find(N=>{var k;return(k=N.children)==null?void 0:k.some(j=>j.id===v.id)});t({type:"thread",agentId:C==null?void 0:C.id,threadId:v.id})}},w=v=>v.type==="root"&&e.type==="overview"||v.type==="agent"&&e.type==="agent"&&e.agentId===v.id||v.type==="thread"&&e.threadId===v.id,S=v=>!v||v.length===0?null:u.jsx("span",{className:"badges",children:v.map((y,C)=>u.jsxs("span",{className:`badge badge-${y.type}`,title:`${y.count} ${y.type}`,children:[y.type==="pending"&&"⏳",y.type==="unread"&&"📬",y.type==="running"&&"▶️",y.count]},C))}),I=v=>{if(!v)return null;const y={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsx("span",{className:"status-indicator",style:{backgroundColor:y[v]||y.idle},title:v})},h=(v,y=0)=>{const C=l.has(v.id),N=v.children&&v.children.length>0,k=w(v);return u.jsxs("div",{className:"tree-node",children:[u.jsxs("div",{className:`tree-node-content ${k?"selected":""} ${v.type}`,style:{paddingLeft:`${y*16+8}px`},onClick:()=>p(v),children:[N&&u.jsx("span",{className:`expand-icon ${C?"expanded":""}`,onClick:j=>{j.stopPropagation(),m(v.id)},children:C?"▼":"▶"}),!N&&u.jsx("span",{className:"expand-icon-placeholder"}),v.type==="agent"&&I(v.status),u.jsx("span",{className:"node-label",children:v.label}),S(v.badges)]}),N&&C&&u.jsx("div",{className:"tree-children",children:v.children.map(j=>h(j,y+1))})]},v.id)};return a&&!r?u.jsx("div",{className:"hierarchy-tree loading",children:"Loading..."}):c?u.jsxs("div",{className:"hierarchy-tree error",children:[u.jsxs("p",{children:["Error: ",c]}),u.jsx("button",{onClick:f,children:"Retry"})]}):u.jsxs("div",{className:"hierarchy-tree",children:[u.jsxs("div",{className:"tree-header",children:[u.jsx("h3",{children:"Agents"}),u.jsx("button",{className:"refresh-btn",onClick:()=>{f(),n==null||n()},title:"Refresh",children:"\\u21BB"})]}),u.jsx("div",{className:"tree-content",children:r&&h(r.root)}),r&&u.jsx("div",{className:"tree-footer",children:u.jsxs("div",{className:"aggregate-stats",children:[u.jsxs("span",{title:"Total agents",children:[r.aggregate.total_agents," agents"]}),u.jsxs("span",{title:"Active",children:[r.aggregate.active_agents," active"]}),r.aggregate.pending_approvals>0&&u.jsxs("span",{className:"pending",title:"Pending approvals",children:[r.aggregate.pending_approvals," pending"]})]})})]})},Ng="_card_1d3of_1",_g="_compact_1d3of_9",zg="_title_1d3of_13",Tg="_metricsGrid_1d3of_20",Lg="_metricItem_1d3of_26",Pg="_metricLabel_1d3of_32",Ig="_metricValue_1d3of_39",Mg="_cost_1d3of_46",Ag="_averages_1d3of_50",Dg="_averagesLabel_1d3of_61",Rg="_avgItem_1d3of_65",Fg="_compactRow_1d3of_72",Og="_compactLabel_1d3of_80",Bg="_compactValue_1d3of_84",$g="_loading_1d3of_91",Ug="_error_1d3of_97",Hg="_errorText_1d3of_101",K={card:Ng,compact:_g,title:zg,metricsGrid:Tg,metricItem:Lg,metricLabel:Pg,metricValue:Ig,cost:Mg,averages:Ag,averagesLabel:Dg,avgItem:Rg,compactRow:Fg,compactLabel:Og,compactValue:Bg,loading:$g,error:Ug,errorText:Hg};function rc(e){return e<1e3?`${e}ms`:e<6e4?`${(e/1e3).toFixed(1)}s`:e<36e5?`${(e/6e4).toFixed(1)}m`:`${(e/36e5).toFixed(1)}h`}function Mn(e){return e.toLocaleString()}function mo(e){return e===0?"$0.00":e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`}function Vg({scopeType:e,scopeId:t="",title:n,compact:r=!1}){const[i,l]=O.useState(null),[o,a]=O.useState(!0),[s,c]=O.useState(null),d=O.useCallback(async()=>{try{let m="/api/metrics";e!=="global"&&(m=`/api/metrics/${e}/${t}`);const p=await fetch(m);if(!p.ok)throw new Error(`Failed to fetch metrics: ${p.status}`);const w=await p.json();l(w),c(null)}catch(m){c(m instanceof Error?m.message:"Failed to load metrics")}finally{a(!1)}},[e,t]);if(O.useEffect(()=>{d();const m=setInterval(d,3e4);return()=>clearInterval(m)},[d]),o)return u.jsx("div",{className:`${K.card} ${r?K.compact:""}`,children:u.jsx("div",{className:K.loading,children:"Loading metrics..."})});if(s)return u.jsx("div",{className:`${K.card} ${r?K.compact:""} ${K.error}`,children:u.jsx("div",{className:K.errorText,children:s})});if(!i)return null;const f=n||(e==="global"?"Global Metrics":e==="agent"?`Agent: ${t}`:`Thread: ${t.slice(0,12)}...`);return r?u.jsx("div",{className:`${K.card} ${K.compact}`,children:u.jsxs("div",{className:K.compactRow,children:[u.jsx("span",{className:K.compactLabel,children:"Runs:"}),u.jsx("span",{className:K.compactValue,children:Mn(i.total_runs)}),u.jsx("span",{className:K.compactLabel,children:"Tokens:"}),u.jsx("span",{className:K.compactValue,children:Mn(i.total_tokens)}),u.jsx("span",{className:K.compactLabel,children:"Cost:"}),u.jsx("span",{className:K.compactValue,children:mo(i.total_cost)})]})}):u.jsxs("div",{className:K.card,children:[u.jsx("h3",{className:K.title,children:f}),u.jsxs("div",{className:K.metricsGrid,children:[u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Runs"}),u.jsx("span",{className:K.metricValue,children:Mn(i.total_runs)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Tokens"}),u.jsx("span",{className:K.metricValue,children:Mn(i.total_tokens)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Cost"}),u.jsx("span",{className:`${K.metricValue} ${K.cost}`,children:mo(i.total_cost)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Total Duration"}),u.jsx("span",{className:K.metricValue,children:rc(i.total_duration_ms)})]}),u.jsxs("div",{className:K.metricItem,children:[u.jsx("span",{className:K.metricLabel,children:"Files Modified"}),u.jsx("span",{className:K.metricValue,children:Mn(i.total_files_modified)})]})]}),i.total_runs>0&&u.jsxs("div",{className:K.averages,children:[u.jsx("span",{className:K.averagesLabel,children:"Averages per run:"}),u.jsxs("span",{className:K.avgItem,children:[Mn(Math.round(i.avg_tokens_per_run))," tokens"]}),u.jsx("span",{className:K.avgItem,children:mo(i.avg_cost_per_run)}),u.jsx("span",{className:K.avgItem,children:rc(Math.round(i.avg_duration_per_run))})]})]})}const Xe=({title:e,value:t,color:n="default",small:r})=>u.jsxs("div",{className:`stat-card stat-${n} ${r?"stat-small":""}`,children:[u.jsx("div",{className:"stat-value",children:t}),u.jsx("div",{className:"stat-title",children:e})]}),Wg=e=>{if(e<1e3)return`${e}ms`;const t=e/1e3;if(t<60)return`${t.toFixed(1)}s`;const n=Math.floor(t/60),r=(t%60).toFixed(0);return`${n}m ${r}s`},Qg=e=>e<.01?`$${e.toFixed(4)}`:`$${e.toFixed(2)}`,Ti=e=>e>=1e6?`${(e/1e6).toFixed(1)}M`:e>=1e3?`${(e/1e3).toFixed(1)}k`:e.toString(),qg=({agent:e,onClick:t})=>{var o,a,s,c,d;const n=((o=e.children)==null?void 0:o.length)||0,r=((s=(a=e.badges)==null?void 0:a.find(f=>f.type==="pending"))==null?void 0:s.count)||0,i=((d=(c=e.badges)==null?void 0:c.find(f=>f.type==="running"))==null?void 0:d.count)||0,l={active:"#22c55e",pending:"#f59e0b",idle:"#6b7280"};return u.jsxs("div",{className:"agent-card",onClick:t,children:[u.jsxs("div",{className:"agent-card-header",children:[u.jsx("span",{className:"agent-status-dot",style:{backgroundColor:l[e.status||"idle"]}}),u.jsx("span",{className:"agent-name",children:e.label})]}),u.jsxs("div",{className:"agent-card-stats",children:[u.jsxs("span",{className:"agent-stat",children:[u.jsx("span",{className:"agent-stat-value",children:n}),u.jsx("span",{className:"agent-stat-label",children:"threads"})]}),r>0&&u.jsxs("span",{className:"agent-stat pending",children:[u.jsx("span",{className:"agent-stat-value",children:r}),u.jsx("span",{className:"agent-stat-label",children:"pending"})]}),i>0&&u.jsxs("span",{className:"agent-stat running",children:[u.jsx("span",{className:"agent-stat-value",children:i}),u.jsx("span",{className:"agent-stat-label",children:"running"})]})]})]})},Kg=({aggregate:e,agents:t,onSelectAgent:n})=>{const r=e.execution,i=r&&r.total_executions>0,l=i?Math.round(r.successful_executions/r.total_executions*100):0;return u.jsxs("div",{className:"all-agents-overview",children:[u.jsx("div",{className:"overview-header",children:u.jsx("h2",{children:"All Agents Overview"})}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Xe,{title:"Total Agents",value:e.total_agents}),u.jsx(Xe,{title:"Active",value:e.active_agents,color:"green"}),u.jsx(Xe,{title:"Pending Approvals",value:e.pending_approvals,color:"orange"}),u.jsx(Xe,{title:"Total Threads",value:e.total_threads,color:"blue"})]}),u.jsxs("div",{className:"metrics-section",children:[u.jsx("h3",{children:"Usage Metrics (Today)"}),u.jsx(Vg,{scopeType:"global",title:"Global Metrics"})]}),i&&u.jsxs("div",{className:"execution-stats-section",children:[u.jsx("h3",{children:"Execution Statistics"}),u.jsxs("div",{className:"stats-row",children:[u.jsx(Xe,{title:"Total Executions",value:r.total_executions,color:"purple"}),u.jsx(Xe,{title:"Success Rate",value:`${l}%`,color:"green"}),u.jsx(Xe,{title:"Total Duration",value:Wg(r.total_duration_ms)}),u.jsx(Xe,{title:"Total Cost",value:Qg(r.total_cost),color:"orange"})]}),u.jsxs("div",{className:"stats-row token-stats",children:[u.jsx(Xe,{title:"Input Tokens",value:Ti(r.total_input_tokens),small:!0}),u.jsx(Xe,{title:"Output Tokens",value:Ti(r.total_output_tokens),small:!0}),u.jsx(Xe,{title:"Cache Read",value:Ti(r.total_cache_read_tokens),small:!0}),u.jsx(Xe,{title:"Cache Created",value:Ti(r.total_cache_create_tokens),small:!0}),u.jsx(Xe,{title:"Files Created",value:r.total_files_created,small:!0})]})]}),u.jsxs("div",{className:"agents-section",children:[u.jsx("h3",{children:"Agents"}),u.jsxs("div",{className:"agent-cards-grid",children:[t.map(o=>u.jsx(qg,{agent:o,onClick:()=>n(o.id)},o.id)),t.length===0&&u.jsx("div",{className:"no-agents",children:"No agents found. Start an agent to see it here."})]})]})]})},Yg=({items:e})=>u.jsx("nav",{className:"breadcrumb",children:e.map((t,n)=>u.jsxs(qt.Fragment,{children:[n>0&&u.jsx("span",{className:"breadcrumb-separator",children:"/"}),t.onClick?u.jsx("button",{className:"breadcrumb-link",onClick:t.onClick,children:t.label}):u.jsx("span",{className:"breadcrumb-current",children:t.label})]},n))}),Lt={plus:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"5",x2:"12",y2:"19"}),u.jsx("line",{x1:"5",y1:"12",x2:"19",y2:"12"})]}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"}),u.jsx("line",{x1:"8",y1:"16",x2:"8",y2:"16"}),u.jsx("line",{x1:"16",y1:"16",x2:"16",y2:"16"})]}),hash:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"4",y1:"9",x2:"20",y2:"9"}),u.jsx("line",{x1:"4",y1:"15",x2:"20",y2:"15"}),u.jsx("line",{x1:"10",y1:"3",x2:"8",y2:"21"}),u.jsx("line",{x1:"16",y1:"3",x2:"14",y2:"21"})]}),edit:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"}),u.jsx("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"})]}),trash:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("polyline",{points:"3 6 5 6 21 6"}),u.jsx("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"})]}),check:u.jsx("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"12",height:"12",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]})},Xg=({threads:e,selectedThreadId:t,onSelectThread:n,onCreateThread:r,onDeleteThread:i,onRenameThread:l,unreadCounts:o})=>{const[a,s]=O.useState(!1),[c,d]=O.useState(""),[f,m]=O.useState(null),[p,w]=O.useState(""),[S,I]=O.useState(null),h=()=>{c.trim()&&(r(c.trim()),d(""),s(!1))},v=z=>{z.key==="Enter"&&!z.shiftKey?(z.preventDefault(),h()):z.key==="Escape"&&(s(!1),d(""))},y=(z,D)=>{D.stopPropagation(),m(z.id),w(z.title)},C=z=>{var D;p.trim()&&p.trim()!==((D=e.find(W=>W.id===z))==null?void 0:D.title)&&l(z,p.trim()),m(null),w("")},N=()=>{m(null),w("")},k=(z,D)=>{z.key==="Enter"?(z.preventDefault(),C(D)):z.key==="Escape"&&N()},j=(z,D)=>{D.stopPropagation(),I(z)},_=(z,D)=>{D.stopPropagation(),i(z),I(null)},R=z=>{z.stopPropagation(),I(null)},P=z=>{const D=new Date(z),X=new Date().getTime()-D.getTime(),U=Math.floor(X/6e4),Q=Math.floor(X/36e5),ie=Math.floor(X/864e5);return U<1?"now":U<60?`${U}m`:Q<24?`${Q}h`:ie<7?`${ie}d`:D.toLocaleDateString(void 0,{month:"short",day:"numeric"})};return u.jsxs("div",{className:"thread-list",children:[u.jsxs("div",{className:"list-header",children:[u.jsx("h2",{children:"Conversations"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>s(!0),title:"New conversation",children:Lt.plus})]}),a&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:c,onChange:z=>d(z.target.value),onKeyDown:v,placeholder:"Conversation title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>s(!1),children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:h,children:"Create"})]})]}),u.jsx("div",{className:"thread-items",children:e.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Lt.hash}),u.jsx("p",{children:"No conversations yet"}),u.jsx("button",{className:"start-btn",onClick:()=>s(!0),children:"Start a conversation"})]}):e.map(z=>{const D=o.get(z.id)||0,W=z.id===t,X=f===z.id,U=S===z.id;return u.jsxs("div",{className:`thread-item ${W?"selected":""} ${D>0?"has-unread":""}`,onClick:()=>!X&&n(z.id),children:[u.jsx("div",{className:`status-dot ${z.status}`}),u.jsxs("div",{className:"thread-content",children:[u.jsx("div",{className:"thread-title-row",children:X?u.jsxs("div",{className:"edit-title-form",onClick:Q=>Q.stopPropagation(),children:[u.jsx("input",{type:"text",value:p,onChange:Q=>w(Q.target.value),onKeyDown:Q=>k(Q,z.id),autoFocus:!0}),u.jsx("button",{className:"edit-action save",onClick:()=>C(z.id),title:"Save",children:Lt.check}),u.jsx("button",{className:"edit-action cancel",onClick:N,title:"Cancel",children:Lt.x})]}):u.jsxs(u.Fragment,{children:[u.jsx("span",{className:"thread-title",children:z.title}),u.jsx("span",{className:"thread-time",children:P(z.updated_at)})]})}),u.jsxs("div",{className:"thread-meta",children:[z.target_agent&&u.jsxs("span",{className:"thread-agent",title:`Target: ${z.target_agent}`,children:[Lt.bot,z.target_agent]}),u.jsxs("span",{className:"thread-seq",children:["#",z.last_seq]})]})]}),!X&&!U&&u.jsxs("div",{className:"thread-actions",children:[u.jsx("button",{className:"action-btn edit",onClick:Q=>y(z,Q),title:"Rename",children:Lt.edit}),u.jsx("button",{className:"action-btn delete",onClick:Q=>j(z.id,Q),title:"Delete",children:Lt.trash})]}),U&&u.jsxs("div",{className:"delete-confirm",onClick:Q=>Q.stopPropagation(),children:[u.jsx("span",{className:"confirm-text",children:"Delete?"}),u.jsx("button",{className:"confirm-btn yes",onClick:Q=>_(z.id,Q),title:"Confirm delete",children:Lt.check}),u.jsx("button",{className:"confirm-btn no",onClick:R,title:"Cancel",children:Lt.x})]}),D>0&&!U&&u.jsx("span",{className:"unread-badge",children:D})]},z.id)})}),u.jsx("style",{children:`
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
      `})]})};function Gg(e,t){const n={};return(e[e.length-1]===""?[...e,""]:e).join((n.padRight?" ":"")+","+(n.padLeft===!1?"":" ")).trim()}const Jg=/^[$_\p{ID_Start}][$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,Zg=/^[$_\p{ID_Start}][-$_\u{200C}\u{200D}\p{ID_Continue}]*$/u,ev={};function ic(e,t){return(ev.jsx?Zg:Jg).test(e)}const tv=/[ \t\n\f\r]/g;function nv(e){return typeof e=="object"?e.type==="text"?lc(e.value):!1:lc(e)}function lc(e){return e.replace(tv,"")===""}class di{constructor(t,n,r){this.normal=n,this.property=t,r&&(this.space=r)}}di.prototype.normal={};di.prototype.property={};di.prototype.space=void 0;function sp(e,t){const n={},r={};for(const i of e)Object.assign(n,i.property),Object.assign(r,i.normal);return new di(n,r,t)}function wa(e){return e.toLowerCase()}class Ke{constructor(t,n){this.attribute=n,this.property=t}}Ke.prototype.attribute="";Ke.prototype.booleanish=!1;Ke.prototype.boolean=!1;Ke.prototype.commaOrSpaceSeparated=!1;Ke.prototype.commaSeparated=!1;Ke.prototype.defined=!1;Ke.prototype.mustUseProperty=!1;Ke.prototype.number=!1;Ke.prototype.overloadedBoolean=!1;Ke.prototype.property="";Ke.prototype.spaceSeparated=!1;Ke.prototype.space=void 0;let rv=0;const q=Ln(),ve=Ln(),Sa=Ln(),A=Ln(),oe=Ln(),er=Ln(),Ge=Ln();function Ln(){return 2**++rv}const ba=Object.freeze(Object.defineProperty({__proto__:null,boolean:q,booleanish:ve,commaOrSpaceSeparated:Ge,commaSeparated:er,number:A,overloadedBoolean:Sa,spaceSeparated:oe},Symbol.toStringTag,{value:"Module"})),go=Object.keys(ba);class _s extends Ke{constructor(t,n,r,i){let l=-1;if(super(t,n),oc(this,"space",i),typeof r=="number")for(;++l<go.length;){const o=go[l];oc(this,go[l],(r&ba[o])===ba[o])}}}_s.prototype.defined=!0;function oc(e,t,n){n&&(e[t]=n)}function fr(e){const t={},n={};for(const[r,i]of Object.entries(e.properties)){const l=new _s(r,e.transform(e.attributes||{},r),i,e.space);e.mustUseProperty&&e.mustUseProperty.includes(r)&&(l.mustUseProperty=!0),t[r]=l,n[wa(r)]=r,n[wa(l.attribute)]=r}return new di(t,n,e.space)}const up=fr({properties:{ariaActiveDescendant:null,ariaAtomic:ve,ariaAutoComplete:null,ariaBusy:ve,ariaChecked:ve,ariaColCount:A,ariaColIndex:A,ariaColSpan:A,ariaControls:oe,ariaCurrent:null,ariaDescribedBy:oe,ariaDetails:null,ariaDisabled:ve,ariaDropEffect:oe,ariaErrorMessage:null,ariaExpanded:ve,ariaFlowTo:oe,ariaGrabbed:ve,ariaHasPopup:null,ariaHidden:ve,ariaInvalid:null,ariaKeyShortcuts:null,ariaLabel:null,ariaLabelledBy:oe,ariaLevel:A,ariaLive:null,ariaModal:ve,ariaMultiLine:ve,ariaMultiSelectable:ve,ariaOrientation:null,ariaOwns:oe,ariaPlaceholder:null,ariaPosInSet:A,ariaPressed:ve,ariaReadOnly:ve,ariaRelevant:null,ariaRequired:ve,ariaRoleDescription:oe,ariaRowCount:A,ariaRowIndex:A,ariaRowSpan:A,ariaSelected:ve,ariaSetSize:A,ariaSort:null,ariaValueMax:A,ariaValueMin:A,ariaValueNow:A,ariaValueText:null,role:null},transform(e,t){return t==="role"?t:"aria-"+t.slice(4).toLowerCase()}});function cp(e,t){return t in e?e[t]:t}function dp(e,t){return cp(e,t.toLowerCase())}const iv=fr({attributes:{acceptcharset:"accept-charset",classname:"class",htmlfor:"for",httpequiv:"http-equiv"},mustUseProperty:["checked","multiple","muted","selected"],properties:{abbr:null,accept:er,acceptCharset:oe,accessKey:oe,action:null,allow:null,allowFullScreen:q,allowPaymentRequest:q,allowUserMedia:q,alt:null,as:null,async:q,autoCapitalize:null,autoComplete:oe,autoFocus:q,autoPlay:q,blocking:oe,capture:null,charSet:null,checked:q,cite:null,className:oe,cols:A,colSpan:null,content:null,contentEditable:ve,controls:q,controlsList:oe,coords:A|er,crossOrigin:null,data:null,dateTime:null,decoding:null,default:q,defer:q,dir:null,dirName:null,disabled:q,download:Sa,draggable:ve,encType:null,enterKeyHint:null,fetchPriority:null,form:null,formAction:null,formEncType:null,formMethod:null,formNoValidate:q,formTarget:null,headers:oe,height:A,hidden:Sa,high:A,href:null,hrefLang:null,htmlFor:oe,httpEquiv:oe,id:null,imageSizes:null,imageSrcSet:null,inert:q,inputMode:null,integrity:null,is:null,isMap:q,itemId:null,itemProp:oe,itemRef:oe,itemScope:q,itemType:oe,kind:null,label:null,lang:null,language:null,list:null,loading:null,loop:q,low:A,manifest:null,max:null,maxLength:A,media:null,method:null,min:null,minLength:A,multiple:q,muted:q,name:null,nonce:null,noModule:q,noValidate:q,onAbort:null,onAfterPrint:null,onAuxClick:null,onBeforeMatch:null,onBeforePrint:null,onBeforeToggle:null,onBeforeUnload:null,onBlur:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onContextLost:null,onContextMenu:null,onContextRestored:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnded:null,onError:null,onFocus:null,onFormData:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLanguageChange:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadEnd:null,onLoadStart:null,onMessage:null,onMessageError:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRejectionHandled:null,onReset:null,onResize:null,onScroll:null,onScrollEnd:null,onSecurityPolicyViolation:null,onSeeked:null,onSeeking:null,onSelect:null,onSlotChange:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnhandledRejection:null,onUnload:null,onVolumeChange:null,onWaiting:null,onWheel:null,open:q,optimum:A,pattern:null,ping:oe,placeholder:null,playsInline:q,popover:null,popoverTarget:null,popoverTargetAction:null,poster:null,preload:null,readOnly:q,referrerPolicy:null,rel:oe,required:q,reversed:q,rows:A,rowSpan:A,sandbox:oe,scope:null,scoped:q,seamless:q,selected:q,shadowRootClonable:q,shadowRootDelegatesFocus:q,shadowRootMode:null,shape:null,size:A,sizes:null,slot:null,span:A,spellCheck:ve,src:null,srcDoc:null,srcLang:null,srcSet:null,start:A,step:null,style:null,tabIndex:A,target:null,title:null,translate:null,type:null,typeMustMatch:q,useMap:null,value:ve,width:A,wrap:null,writingSuggestions:null,align:null,aLink:null,archive:oe,axis:null,background:null,bgColor:null,border:A,borderColor:null,bottomMargin:A,cellPadding:null,cellSpacing:null,char:null,charOff:null,classId:null,clear:null,code:null,codeBase:null,codeType:null,color:null,compact:q,declare:q,event:null,face:null,frame:null,frameBorder:null,hSpace:A,leftMargin:A,link:null,longDesc:null,lowSrc:null,marginHeight:A,marginWidth:A,noResize:q,noHref:q,noShade:q,noWrap:q,object:null,profile:null,prompt:null,rev:null,rightMargin:A,rules:null,scheme:null,scrolling:ve,standby:null,summary:null,text:null,topMargin:A,valueType:null,version:null,vAlign:null,vLink:null,vSpace:A,allowTransparency:null,autoCorrect:null,autoSave:null,disablePictureInPicture:q,disableRemotePlayback:q,prefix:null,property:null,results:A,security:null,unselectable:null},space:"html",transform:dp}),lv=fr({attributes:{accentHeight:"accent-height",alignmentBaseline:"alignment-baseline",arabicForm:"arabic-form",baselineShift:"baseline-shift",capHeight:"cap-height",className:"class",clipPath:"clip-path",clipRule:"clip-rule",colorInterpolation:"color-interpolation",colorInterpolationFilters:"color-interpolation-filters",colorProfile:"color-profile",colorRendering:"color-rendering",crossOrigin:"crossorigin",dataType:"datatype",dominantBaseline:"dominant-baseline",enableBackground:"enable-background",fillOpacity:"fill-opacity",fillRule:"fill-rule",floodColor:"flood-color",floodOpacity:"flood-opacity",fontFamily:"font-family",fontSize:"font-size",fontSizeAdjust:"font-size-adjust",fontStretch:"font-stretch",fontStyle:"font-style",fontVariant:"font-variant",fontWeight:"font-weight",glyphName:"glyph-name",glyphOrientationHorizontal:"glyph-orientation-horizontal",glyphOrientationVertical:"glyph-orientation-vertical",hrefLang:"hreflang",horizAdvX:"horiz-adv-x",horizOriginX:"horiz-origin-x",horizOriginY:"horiz-origin-y",imageRendering:"image-rendering",letterSpacing:"letter-spacing",lightingColor:"lighting-color",markerEnd:"marker-end",markerMid:"marker-mid",markerStart:"marker-start",navDown:"nav-down",navDownLeft:"nav-down-left",navDownRight:"nav-down-right",navLeft:"nav-left",navNext:"nav-next",navPrev:"nav-prev",navRight:"nav-right",navUp:"nav-up",navUpLeft:"nav-up-left",navUpRight:"nav-up-right",onAbort:"onabort",onActivate:"onactivate",onAfterPrint:"onafterprint",onBeforePrint:"onbeforeprint",onBegin:"onbegin",onCancel:"oncancel",onCanPlay:"oncanplay",onCanPlayThrough:"oncanplaythrough",onChange:"onchange",onClick:"onclick",onClose:"onclose",onCopy:"oncopy",onCueChange:"oncuechange",onCut:"oncut",onDblClick:"ondblclick",onDrag:"ondrag",onDragEnd:"ondragend",onDragEnter:"ondragenter",onDragExit:"ondragexit",onDragLeave:"ondragleave",onDragOver:"ondragover",onDragStart:"ondragstart",onDrop:"ondrop",onDurationChange:"ondurationchange",onEmptied:"onemptied",onEnd:"onend",onEnded:"onended",onError:"onerror",onFocus:"onfocus",onFocusIn:"onfocusin",onFocusOut:"onfocusout",onHashChange:"onhashchange",onInput:"oninput",onInvalid:"oninvalid",onKeyDown:"onkeydown",onKeyPress:"onkeypress",onKeyUp:"onkeyup",onLoad:"onload",onLoadedData:"onloadeddata",onLoadedMetadata:"onloadedmetadata",onLoadStart:"onloadstart",onMessage:"onmessage",onMouseDown:"onmousedown",onMouseEnter:"onmouseenter",onMouseLeave:"onmouseleave",onMouseMove:"onmousemove",onMouseOut:"onmouseout",onMouseOver:"onmouseover",onMouseUp:"onmouseup",onMouseWheel:"onmousewheel",onOffline:"onoffline",onOnline:"ononline",onPageHide:"onpagehide",onPageShow:"onpageshow",onPaste:"onpaste",onPause:"onpause",onPlay:"onplay",onPlaying:"onplaying",onPopState:"onpopstate",onProgress:"onprogress",onRateChange:"onratechange",onRepeat:"onrepeat",onReset:"onreset",onResize:"onresize",onScroll:"onscroll",onSeeked:"onseeked",onSeeking:"onseeking",onSelect:"onselect",onShow:"onshow",onStalled:"onstalled",onStorage:"onstorage",onSubmit:"onsubmit",onSuspend:"onsuspend",onTimeUpdate:"ontimeupdate",onToggle:"ontoggle",onUnload:"onunload",onVolumeChange:"onvolumechange",onWaiting:"onwaiting",onZoom:"onzoom",overlinePosition:"overline-position",overlineThickness:"overline-thickness",paintOrder:"paint-order",panose1:"panose-1",pointerEvents:"pointer-events",referrerPolicy:"referrerpolicy",renderingIntent:"rendering-intent",shapeRendering:"shape-rendering",stopColor:"stop-color",stopOpacity:"stop-opacity",strikethroughPosition:"strikethrough-position",strikethroughThickness:"strikethrough-thickness",strokeDashArray:"stroke-dasharray",strokeDashOffset:"stroke-dashoffset",strokeLineCap:"stroke-linecap",strokeLineJoin:"stroke-linejoin",strokeMiterLimit:"stroke-miterlimit",strokeOpacity:"stroke-opacity",strokeWidth:"stroke-width",tabIndex:"tabindex",textAnchor:"text-anchor",textDecoration:"text-decoration",textRendering:"text-rendering",transformOrigin:"transform-origin",typeOf:"typeof",underlinePosition:"underline-position",underlineThickness:"underline-thickness",unicodeBidi:"unicode-bidi",unicodeRange:"unicode-range",unitsPerEm:"units-per-em",vAlphabetic:"v-alphabetic",vHanging:"v-hanging",vIdeographic:"v-ideographic",vMathematical:"v-mathematical",vectorEffect:"vector-effect",vertAdvY:"vert-adv-y",vertOriginX:"vert-origin-x",vertOriginY:"vert-origin-y",wordSpacing:"word-spacing",writingMode:"writing-mode",xHeight:"x-height",playbackOrder:"playbackorder",timelineBegin:"timelinebegin"},properties:{about:Ge,accentHeight:A,accumulate:null,additive:null,alignmentBaseline:null,alphabetic:A,amplitude:A,arabicForm:null,ascent:A,attributeName:null,attributeType:null,azimuth:A,bandwidth:null,baselineShift:null,baseFrequency:null,baseProfile:null,bbox:null,begin:null,bias:A,by:null,calcMode:null,capHeight:A,className:oe,clip:null,clipPath:null,clipPathUnits:null,clipRule:null,color:null,colorInterpolation:null,colorInterpolationFilters:null,colorProfile:null,colorRendering:null,content:null,contentScriptType:null,contentStyleType:null,crossOrigin:null,cursor:null,cx:null,cy:null,d:null,dataType:null,defaultAction:null,descent:A,diffuseConstant:A,direction:null,display:null,dur:null,divisor:A,dominantBaseline:null,download:q,dx:null,dy:null,edgeMode:null,editable:null,elevation:A,enableBackground:null,end:null,event:null,exponent:A,externalResourcesRequired:null,fill:null,fillOpacity:A,fillRule:null,filter:null,filterRes:null,filterUnits:null,floodColor:null,floodOpacity:null,focusable:null,focusHighlight:null,fontFamily:null,fontSize:null,fontSizeAdjust:null,fontStretch:null,fontStyle:null,fontVariant:null,fontWeight:null,format:null,fr:null,from:null,fx:null,fy:null,g1:er,g2:er,glyphName:er,glyphOrientationHorizontal:null,glyphOrientationVertical:null,glyphRef:null,gradientTransform:null,gradientUnits:null,handler:null,hanging:A,hatchContentUnits:null,hatchUnits:null,height:null,href:null,hrefLang:null,horizAdvX:A,horizOriginX:A,horizOriginY:A,id:null,ideographic:A,imageRendering:null,initialVisibility:null,in:null,in2:null,intercept:A,k:A,k1:A,k2:A,k3:A,k4:A,kernelMatrix:Ge,kernelUnitLength:null,keyPoints:null,keySplines:null,keyTimes:null,kerning:null,lang:null,lengthAdjust:null,letterSpacing:null,lightingColor:null,limitingConeAngle:A,local:null,markerEnd:null,markerMid:null,markerStart:null,markerHeight:null,markerUnits:null,markerWidth:null,mask:null,maskContentUnits:null,maskUnits:null,mathematical:null,max:null,media:null,mediaCharacterEncoding:null,mediaContentEncodings:null,mediaSize:A,mediaTime:null,method:null,min:null,mode:null,name:null,navDown:null,navDownLeft:null,navDownRight:null,navLeft:null,navNext:null,navPrev:null,navRight:null,navUp:null,navUpLeft:null,navUpRight:null,numOctaves:null,observer:null,offset:null,onAbort:null,onActivate:null,onAfterPrint:null,onBeforePrint:null,onBegin:null,onCancel:null,onCanPlay:null,onCanPlayThrough:null,onChange:null,onClick:null,onClose:null,onCopy:null,onCueChange:null,onCut:null,onDblClick:null,onDrag:null,onDragEnd:null,onDragEnter:null,onDragExit:null,onDragLeave:null,onDragOver:null,onDragStart:null,onDrop:null,onDurationChange:null,onEmptied:null,onEnd:null,onEnded:null,onError:null,onFocus:null,onFocusIn:null,onFocusOut:null,onHashChange:null,onInput:null,onInvalid:null,onKeyDown:null,onKeyPress:null,onKeyUp:null,onLoad:null,onLoadedData:null,onLoadedMetadata:null,onLoadStart:null,onMessage:null,onMouseDown:null,onMouseEnter:null,onMouseLeave:null,onMouseMove:null,onMouseOut:null,onMouseOver:null,onMouseUp:null,onMouseWheel:null,onOffline:null,onOnline:null,onPageHide:null,onPageShow:null,onPaste:null,onPause:null,onPlay:null,onPlaying:null,onPopState:null,onProgress:null,onRateChange:null,onRepeat:null,onReset:null,onResize:null,onScroll:null,onSeeked:null,onSeeking:null,onSelect:null,onShow:null,onStalled:null,onStorage:null,onSubmit:null,onSuspend:null,onTimeUpdate:null,onToggle:null,onUnload:null,onVolumeChange:null,onWaiting:null,onZoom:null,opacity:null,operator:null,order:null,orient:null,orientation:null,origin:null,overflow:null,overlay:null,overlinePosition:A,overlineThickness:A,paintOrder:null,panose1:null,path:null,pathLength:A,patternContentUnits:null,patternTransform:null,patternUnits:null,phase:null,ping:oe,pitch:null,playbackOrder:null,pointerEvents:null,points:null,pointsAtX:A,pointsAtY:A,pointsAtZ:A,preserveAlpha:null,preserveAspectRatio:null,primitiveUnits:null,propagate:null,property:Ge,r:null,radius:null,referrerPolicy:null,refX:null,refY:null,rel:Ge,rev:Ge,renderingIntent:null,repeatCount:null,repeatDur:null,requiredExtensions:Ge,requiredFeatures:Ge,requiredFonts:Ge,requiredFormats:Ge,resource:null,restart:null,result:null,rotate:null,rx:null,ry:null,scale:null,seed:null,shapeRendering:null,side:null,slope:null,snapshotTime:null,specularConstant:A,specularExponent:A,spreadMethod:null,spacing:null,startOffset:null,stdDeviation:null,stemh:null,stemv:null,stitchTiles:null,stopColor:null,stopOpacity:null,strikethroughPosition:A,strikethroughThickness:A,string:null,stroke:null,strokeDashArray:Ge,strokeDashOffset:null,strokeLineCap:null,strokeLineJoin:null,strokeMiterLimit:A,strokeOpacity:A,strokeWidth:null,style:null,surfaceScale:A,syncBehavior:null,syncBehaviorDefault:null,syncMaster:null,syncTolerance:null,syncToleranceDefault:null,systemLanguage:Ge,tabIndex:A,tableValues:null,target:null,targetX:A,targetY:A,textAnchor:null,textDecoration:null,textRendering:null,textLength:null,timelineBegin:null,title:null,transformBehavior:null,type:null,typeOf:Ge,to:null,transform:null,transformOrigin:null,u1:null,u2:null,underlinePosition:A,underlineThickness:A,unicode:null,unicodeBidi:null,unicodeRange:null,unitsPerEm:A,values:null,vAlphabetic:A,vMathematical:A,vectorEffect:null,vHanging:A,vIdeographic:A,version:null,vertAdvY:A,vertOriginX:A,vertOriginY:A,viewBox:null,viewTarget:null,visibility:null,width:null,widths:null,wordSpacing:null,writingMode:null,x:null,x1:null,x2:null,xChannelSelector:null,xHeight:A,y:null,y1:null,y2:null,yChannelSelector:null,z:null,zoomAndPan:null},space:"svg",transform:cp}),fp=fr({properties:{xLinkActuate:null,xLinkArcRole:null,xLinkHref:null,xLinkRole:null,xLinkShow:null,xLinkTitle:null,xLinkType:null},space:"xlink",transform(e,t){return"xlink:"+t.slice(5).toLowerCase()}}),pp=fr({attributes:{xmlnsxlink:"xmlns:xlink"},properties:{xmlnsXLink:null,xmlns:null},space:"xmlns",transform:dp}),hp=fr({properties:{xmlBase:null,xmlLang:null,xmlSpace:null},space:"xml",transform(e,t){return"xml:"+t.slice(3).toLowerCase()}}),ov={classId:"classID",dataType:"datatype",itemId:"itemID",strokeDashArray:"strokeDasharray",strokeDashOffset:"strokeDashoffset",strokeLineCap:"strokeLinecap",strokeLineJoin:"strokeLinejoin",strokeMiterLimit:"strokeMiterlimit",typeOf:"typeof",xLinkActuate:"xlinkActuate",xLinkArcRole:"xlinkArcrole",xLinkHref:"xlinkHref",xLinkRole:"xlinkRole",xLinkShow:"xlinkShow",xLinkTitle:"xlinkTitle",xLinkType:"xlinkType",xmlnsXLink:"xmlnsXlink"},av=/[A-Z]/g,ac=/-[a-z]/g,sv=/^data[-\w.:]+$/i;function uv(e,t){const n=wa(t);let r=t,i=Ke;if(n in e.normal)return e.property[e.normal[n]];if(n.length>4&&n.slice(0,4)==="data"&&sv.test(t)){if(t.charAt(4)==="-"){const l=t.slice(5).replace(ac,dv);r="data"+l.charAt(0).toUpperCase()+l.slice(1)}else{const l=t.slice(4);if(!ac.test(l)){let o=l.replace(av,cv);o.charAt(0)!=="-"&&(o="-"+o),t="data"+o}}i=_s}return new i(r,t)}function cv(e){return"-"+e.toLowerCase()}function dv(e){return e.charAt(1).toUpperCase()}const fv=sp([up,iv,fp,pp,hp],"html"),zs=sp([up,lv,fp,pp,hp],"svg");function pv(e){return e.join(" ").trim()}var Ts={},sc=/\/\*[^*]*\*+([^/*][^*]*\*+)*\//g,hv=/\n/g,mv=/^\s*/,gv=/^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/,vv=/^:\s*/,yv=/^((?:'(?:\\'|.)*?'|"(?:\\"|.)*?"|\([^)]*?\)|[^};])+)/,xv=/^[;\s]*/,kv=/^\s+|\s+$/g,wv=`
`,uc="/",cc="*",yn="",Sv="comment",bv="declaration";function Cv(e,t){if(typeof e!="string")throw new TypeError("First argument must be a string");if(!e)return[];t=t||{};var n=1,r=1;function i(w){var S=w.match(hv);S&&(n+=S.length);var I=w.lastIndexOf(wv);r=~I?w.length-I:r+w.length}function l(){var w={line:n,column:r};return function(S){return S.position=new o(w),c(),S}}function o(w){this.start=w,this.end={line:n,column:r},this.source=t.source}o.prototype.content=e;function a(w){var S=new Error(t.source+":"+n+":"+r+": "+w);if(S.reason=w,S.filename=t.source,S.line=n,S.column=r,S.source=e,!t.silent)throw S}function s(w){var S=w.exec(e);if(S){var I=S[0];return i(I),e=e.slice(I.length),S}}function c(){s(mv)}function d(w){var S;for(w=w||[];S=f();)S!==!1&&w.push(S);return w}function f(){var w=l();if(!(uc!=e.charAt(0)||cc!=e.charAt(1))){for(var S=2;yn!=e.charAt(S)&&(cc!=e.charAt(S)||uc!=e.charAt(S+1));)++S;if(S+=2,yn===e.charAt(S-1))return a("End of comment missing");var I=e.slice(2,S-2);return r+=2,i(I),e=e.slice(S),r+=2,w({type:Sv,comment:I})}}function m(){var w=l(),S=s(gv);if(S){if(f(),!s(vv))return a("property missing ':'");var I=s(yv),h=w({type:bv,property:dc(S[0].replace(sc,yn)),value:I?dc(I[0].replace(sc,yn)):yn});return s(xv),h}}function p(){var w=[];d(w);for(var S;S=m();)S!==!1&&(w.push(S),d(w));return w}return c(),p()}function dc(e){return e?e.replace(kv,yn):yn}var jv=Cv,Ev=Yi&&Yi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}};Object.defineProperty(Ts,"__esModule",{value:!0});Ts.default=_v;const Nv=Ev(jv);function _v(e,t){let n=null;if(!e||typeof e!="string")return n;const r=(0,Nv.default)(e),i=typeof t=="function";return r.forEach(l=>{if(l.type!=="declaration")return;const{property:o,value:a}=l;i?t(o,a,l):a&&(n=n||{},n[o]=a)}),n}var Fl={};Object.defineProperty(Fl,"__esModule",{value:!0});Fl.camelCase=void 0;var zv=/^--[a-zA-Z0-9_-]+$/,Tv=/-([a-z])/g,Lv=/^[^-]+$/,Pv=/^-(webkit|moz|ms|o|khtml)-/,Iv=/^-(ms)-/,Mv=function(e){return!e||Lv.test(e)||zv.test(e)},Av=function(e,t){return t.toUpperCase()},fc=function(e,t){return"".concat(t,"-")},Dv=function(e,t){return t===void 0&&(t={}),Mv(e)?e:(e=e.toLowerCase(),t.reactCompat?e=e.replace(Iv,fc):e=e.replace(Pv,fc),e.replace(Tv,Av))};Fl.camelCase=Dv;var Rv=Yi&&Yi.__importDefault||function(e){return e&&e.__esModule?e:{default:e}},Fv=Rv(Ts),Ov=Fl;function Ca(e,t){var n={};return!e||typeof e!="string"||(0,Fv.default)(e,function(r,i){r&&i&&(n[(0,Ov.camelCase)(r,t)]=i)}),n}Ca.default=Ca;var Bv=Ca;const $v=Ia(Bv),mp=gp("end"),Ls=gp("start");function gp(e){return t;function t(n){const r=n&&n.position&&n.position[e]||{};if(typeof r.line=="number"&&r.line>0&&typeof r.column=="number"&&r.column>0)return{line:r.line,column:r.column,offset:typeof r.offset=="number"&&r.offset>-1?r.offset:void 0}}}function Uv(e){const t=Ls(e),n=mp(e);if(t&&n)return{start:t,end:n}}function Or(e){return!e||typeof e!="object"?"":"position"in e||"type"in e?pc(e.position):"start"in e||"end"in e?pc(e):"line"in e||"column"in e?ja(e):""}function ja(e){return hc(e&&e.line)+":"+hc(e&&e.column)}function pc(e){return ja(e&&e.start)+"-"+ja(e&&e.end)}function hc(e){return e&&typeof e=="number"?e:1}class Ie extends Error{constructor(t,n,r){super(),typeof n=="string"&&(r=n,n=void 0);let i="",l={},o=!1;if(n&&("line"in n&&"column"in n?l={place:n}:"start"in n&&"end"in n?l={place:n}:"type"in n?l={ancestors:[n],place:n.position}:l={...n}),typeof t=="string"?i=t:!l.cause&&t&&(o=!0,i=t.message,l.cause=t),!l.ruleId&&!l.source&&typeof r=="string"){const s=r.indexOf(":");s===-1?l.ruleId=r:(l.source=r.slice(0,s),l.ruleId=r.slice(s+1))}if(!l.place&&l.ancestors&&l.ancestors){const s=l.ancestors[l.ancestors.length-1];s&&(l.place=s.position)}const a=l.place&&"start"in l.place?l.place.start:l.place;this.ancestors=l.ancestors||void 0,this.cause=l.cause||void 0,this.column=a?a.column:void 0,this.fatal=void 0,this.file="",this.message=i,this.line=a?a.line:void 0,this.name=Or(l.place)||"1:1",this.place=l.place||void 0,this.reason=this.message,this.ruleId=l.ruleId||void 0,this.source=l.source||void 0,this.stack=o&&l.cause&&typeof l.cause.stack=="string"?l.cause.stack:"",this.actual=void 0,this.expected=void 0,this.note=void 0,this.url=void 0}}Ie.prototype.file="";Ie.prototype.name="";Ie.prototype.reason="";Ie.prototype.message="";Ie.prototype.stack="";Ie.prototype.column=void 0;Ie.prototype.line=void 0;Ie.prototype.ancestors=void 0;Ie.prototype.cause=void 0;Ie.prototype.fatal=void 0;Ie.prototype.place=void 0;Ie.prototype.ruleId=void 0;Ie.prototype.source=void 0;const Ps={}.hasOwnProperty,Hv=new Map,Vv=/[A-Z]/g,Wv=new Set(["table","tbody","thead","tfoot","tr"]),Qv=new Set(["td","th"]),vp="https://github.com/syntax-tree/hast-util-to-jsx-runtime";function qv(e,t){if(!t||t.Fragment===void 0)throw new TypeError("Expected `Fragment` in options");const n=t.filePath||void 0;let r;if(t.development){if(typeof t.jsxDEV!="function")throw new TypeError("Expected `jsxDEV` in options when `development: true`");r=ty(n,t.jsxDEV)}else{if(typeof t.jsx!="function")throw new TypeError("Expected `jsx` in production options");if(typeof t.jsxs!="function")throw new TypeError("Expected `jsxs` in production options");r=ey(n,t.jsx,t.jsxs)}const i={Fragment:t.Fragment,ancestors:[],components:t.components||{},create:r,elementAttributeNameCase:t.elementAttributeNameCase||"react",evaluater:t.createEvaluater?t.createEvaluater():void 0,filePath:n,ignoreInvalidStyle:t.ignoreInvalidStyle||!1,passKeys:t.passKeys!==!1,passNode:t.passNode||!1,schema:t.space==="svg"?zs:fv,stylePropertyNameCase:t.stylePropertyNameCase||"dom",tableCellAlignToStyle:t.tableCellAlignToStyle!==!1},l=yp(i,e,void 0);return l&&typeof l!="string"?l:i.create(e,i.Fragment,{children:l||void 0},void 0)}function yp(e,t,n){if(t.type==="element")return Kv(e,t,n);if(t.type==="mdxFlowExpression"||t.type==="mdxTextExpression")return Yv(e,t);if(t.type==="mdxJsxFlowElement"||t.type==="mdxJsxTextElement")return Gv(e,t,n);if(t.type==="mdxjsEsm")return Xv(e,t);if(t.type==="root")return Jv(e,t,n);if(t.type==="text")return Zv(e,t)}function Kv(e,t,n){const r=e.schema;let i=r;t.tagName.toLowerCase()==="svg"&&r.space==="html"&&(i=zs,e.schema=i),e.ancestors.push(t);const l=kp(e,t.tagName,!1),o=ny(e,t);let a=Ms(e,t);return Wv.has(t.tagName)&&(a=a.filter(function(s){return typeof s=="string"?!nv(s):!0})),xp(e,o,l,t),Is(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function Yv(e,t){if(t.data&&t.data.estree&&e.evaluater){const r=t.data.estree.body[0];return r.type,e.evaluater.evaluateExpression(r.expression)}li(e,t.position)}function Xv(e,t){if(t.data&&t.data.estree&&e.evaluater)return e.evaluater.evaluateProgram(t.data.estree);li(e,t.position)}function Gv(e,t,n){const r=e.schema;let i=r;t.name==="svg"&&r.space==="html"&&(i=zs,e.schema=i),e.ancestors.push(t);const l=t.name===null?e.Fragment:kp(e,t.name,!0),o=ry(e,t),a=Ms(e,t);return xp(e,o,l,t),Is(o,a),e.ancestors.pop(),e.schema=r,e.create(t,l,o,n)}function Jv(e,t,n){const r={};return Is(r,Ms(e,t)),e.create(t,e.Fragment,r,n)}function Zv(e,t){return t.value}function xp(e,t,n,r){typeof n!="string"&&n!==e.Fragment&&e.passNode&&(t.node=r)}function Is(e,t){if(t.length>0){const n=t.length>1?t:t[0];n&&(e.children=n)}}function ey(e,t,n){return r;function r(i,l,o,a){const c=Array.isArray(o.children)?n:t;return a?c(l,o,a):c(l,o)}}function ty(e,t){return n;function n(r,i,l,o){const a=Array.isArray(l.children),s=Ls(r);return t(i,l,o,a,{columnNumber:s?s.column-1:void 0,fileName:e,lineNumber:s?s.line:void 0},void 0)}}function ny(e,t){const n={};let r,i;for(i in t.properties)if(i!=="children"&&Ps.call(t.properties,i)){const l=iy(e,i,t.properties[i]);if(l){const[o,a]=l;e.tableCellAlignToStyle&&o==="align"&&typeof a=="string"&&Qv.has(t.tagName)?r=a:n[o]=a}}if(r){const l=n.style||(n.style={});l[e.stylePropertyNameCase==="css"?"text-align":"textAlign"]=r}return n}function ry(e,t){const n={};for(const r of t.attributes)if(r.type==="mdxJsxExpressionAttribute")if(r.data&&r.data.estree&&e.evaluater){const l=r.data.estree.body[0];l.type;const o=l.expression;o.type;const a=o.properties[0];a.type,Object.assign(n,e.evaluater.evaluateExpression(a.argument))}else li(e,t.position);else{const i=r.name;let l;if(r.value&&typeof r.value=="object")if(r.value.data&&r.value.data.estree&&e.evaluater){const a=r.value.data.estree.body[0];a.type,l=e.evaluater.evaluateExpression(a.expression)}else li(e,t.position);else l=r.value===null?!0:r.value;n[i]=l}return n}function Ms(e,t){const n=[];let r=-1;const i=e.passKeys?new Map:Hv;for(;++r<t.children.length;){const l=t.children[r];let o;if(e.passKeys){const s=l.type==="element"?l.tagName:l.type==="mdxJsxFlowElement"||l.type==="mdxJsxTextElement"?l.name:void 0;if(s){const c=i.get(s)||0;o=s+"-"+c,i.set(s,c+1)}}const a=yp(e,l,o);a!==void 0&&n.push(a)}return n}function iy(e,t,n){const r=uv(e.schema,t);if(!(n==null||typeof n=="number"&&Number.isNaN(n))){if(Array.isArray(n)&&(n=r.commaSeparated?Gg(n):pv(n)),r.property==="style"){let i=typeof n=="object"?n:ly(e,String(n));return e.stylePropertyNameCase==="css"&&(i=oy(i)),["style",i]}return[e.elementAttributeNameCase==="react"&&r.space?ov[r.property]||r.property:r.attribute,n]}}function ly(e,t){try{return $v(t,{reactCompat:!0})}catch(n){if(e.ignoreInvalidStyle)return{};const r=n,i=new Ie("Cannot parse `style` attribute",{ancestors:e.ancestors,cause:r,ruleId:"style",source:"hast-util-to-jsx-runtime"});throw i.file=e.filePath||void 0,i.url=vp+"#cannot-parse-style-attribute",i}}function kp(e,t,n){let r;if(!n)r={type:"Literal",value:t};else if(t.includes(".")){const i=t.split(".");let l=-1,o;for(;++l<i.length;){const a=ic(i[l])?{type:"Identifier",name:i[l]}:{type:"Literal",value:i[l]};o=o?{type:"MemberExpression",object:o,property:a,computed:!!(l&&a.type==="Literal"),optional:!1}:a}r=o}else r=ic(t)&&!/^[a-z]/.test(t)?{type:"Identifier",name:t}:{type:"Literal",value:t};if(r.type==="Literal"){const i=r.value;return Ps.call(e.components,i)?e.components[i]:i}if(e.evaluater)return e.evaluater.evaluateExpression(r);li(e)}function li(e,t){const n=new Ie("Cannot handle MDX estrees without `createEvaluater`",{ancestors:e.ancestors,place:t,ruleId:"mdx-estree",source:"hast-util-to-jsx-runtime"});throw n.file=e.filePath||void 0,n.url=vp+"#cannot-handle-mdx-estrees-without-createevaluater",n}function oy(e){const t={};let n;for(n in e)Ps.call(e,n)&&(t[ay(n)]=e[n]);return t}function ay(e){let t=e.replace(Vv,sy);return t.slice(0,3)==="ms-"&&(t="-"+t),t}function sy(e){return"-"+e.toLowerCase()}const vo={action:["form"],cite:["blockquote","del","ins","q"],data:["object"],formAction:["button","input"],href:["a","area","base","link"],icon:["menuitem"],itemId:null,manifest:["html"],ping:["a","area"],poster:["video"],src:["audio","embed","iframe","img","input","script","source","track","video"]},uy={};function cy(e,t){const n=uy,r=typeof n.includeImageAlt=="boolean"?n.includeImageAlt:!0,i=typeof n.includeHtml=="boolean"?n.includeHtml:!0;return wp(e,r,i)}function wp(e,t,n){if(dy(e)){if("value"in e)return e.type==="html"&&!n?"":e.value;if(t&&"alt"in e&&e.alt)return e.alt;if("children"in e)return mc(e.children,t,n)}return Array.isArray(e)?mc(e,t,n):""}function mc(e,t,n){const r=[];let i=-1;for(;++i<e.length;)r[i]=wp(e[i],t,n);return r.join("")}function dy(e){return!!(e&&typeof e=="object")}const gc=document.createElement("i");function As(e){const t="&"+e+";";gc.innerHTML=t;const n=gc.textContent;return n.charCodeAt(n.length-1)===59&&e!=="semi"||n===t?!1:n}function _t(e,t,n,r){const i=e.length;let l=0,o;if(t<0?t=-t>i?0:i+t:t=t>i?i:t,n=n>0?n:0,r.length<1e4)o=Array.from(r),o.unshift(t,n),e.splice(...o);else for(n&&e.splice(t,n);l<r.length;)o=r.slice(l,l+1e4),o.unshift(t,0),e.splice(...o),l+=1e4,t+=1e4}function st(e,t){return e.length>0?(_t(e,e.length,0,t),e):t}const vc={}.hasOwnProperty;function fy(e){const t={};let n=-1;for(;++n<e.length;)py(t,e[n]);return t}function py(e,t){let n;for(n in t){const i=(vc.call(e,n)?e[n]:void 0)||(e[n]={}),l=t[n];let o;if(l)for(o in l){vc.call(i,o)||(i[o]=[]);const a=l[o];hy(i[o],Array.isArray(a)?a:a?[a]:[])}}}function hy(e,t){let n=-1;const r=[];for(;++n<t.length;)(t[n].add==="after"?e:r).push(t[n]);_t(e,0,0,r)}function Sp(e,t){const n=Number.parseInt(e,t);return n<9||n===11||n>13&&n<32||n>126&&n<160||n>55295&&n<57344||n>64975&&n<65008||(n&65535)===65535||(n&65535)===65534||n>1114111?"�":String.fromCodePoint(n)}function tr(e){return e.replace(/[\t\n\r ]+/g," ").replace(/^ | $/g,"").toLowerCase().toUpperCase()}const jt=hn(/[A-Za-z]/),et=hn(/[\dA-Za-z]/),my=hn(/[#-'*+\--9=?A-Z^-~]/);function Ea(e){return e!==null&&(e<32||e===127)}const Na=hn(/\d/),gy=hn(/[\dA-Fa-f]/),vy=hn(/[!-/:-@[-`{-~]/);function H(e){return e!==null&&e<-2}function qe(e){return e!==null&&(e<0||e===32)}function ee(e){return e===-2||e===-1||e===32}const yy=hn(new RegExp("\\p{P}|\\p{S}","u")),xy=hn(/\s/);function hn(e){return t;function t(n){return n!==null&&n>-1&&e.test(String.fromCharCode(n))}}function pr(e){const t=[];let n=-1,r=0,i=0;for(;++n<e.length;){const l=e.charCodeAt(n);let o="";if(l===37&&et(e.charCodeAt(n+1))&&et(e.charCodeAt(n+2)))i=2;else if(l<128)/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(l))||(o=String.fromCharCode(l));else if(l>55295&&l<57344){const a=e.charCodeAt(n+1);l<56320&&a>56319&&a<57344?(o=String.fromCharCode(l,a),i=1):o="�"}else o=String.fromCharCode(l);o&&(t.push(e.slice(r,n),encodeURIComponent(o)),r=n+i+1,o=""),i&&(n+=i,i=0)}return t.join("")+e.slice(r)}function se(e,t,n,r){const i=r?r-1:Number.POSITIVE_INFINITY;let l=0;return o;function o(s){return ee(s)?(e.enter(n),a(s)):t(s)}function a(s){return ee(s)&&l++<i?(e.consume(s),a):(e.exit(n),t(s))}}const ky={tokenize:wy};function wy(e){const t=e.attempt(this.parser.constructs.contentInitial,r,i);let n;return t;function r(a){if(a===null){e.consume(a);return}return e.enter("lineEnding"),e.consume(a),e.exit("lineEnding"),se(e,t,"linePrefix")}function i(a){return e.enter("paragraph"),l(a)}function l(a){const s=e.enter("chunkText",{contentType:"text",previous:n});return n&&(n.next=s),n=s,o(a)}function o(a){if(a===null){e.exit("chunkText"),e.exit("paragraph"),e.consume(a);return}return H(a)?(e.consume(a),e.exit("chunkText"),l):(e.consume(a),o)}}const Sy={tokenize:by},yc={tokenize:Cy};function by(e){const t=this,n=[];let r=0,i,l,o;return a;function a(y){if(r<n.length){const C=n[r];return t.containerState=C[1],e.attempt(C[0].continuation,s,c)(y)}return c(y)}function s(y){if(r++,t.containerState._closeFlow){t.containerState._closeFlow=void 0,i&&v();const C=t.events.length;let N=C,k;for(;N--;)if(t.events[N][0]==="exit"&&t.events[N][1].type==="chunkFlow"){k=t.events[N][1].end;break}h(r);let j=C;for(;j<t.events.length;)t.events[j][1].end={...k},j++;return _t(t.events,N+1,0,t.events.slice(C)),t.events.length=j,c(y)}return a(y)}function c(y){if(r===n.length){if(!i)return m(y);if(i.currentConstruct&&i.currentConstruct.concrete)return w(y);t.interrupt=!!(i.currentConstruct&&!i._gfmTableDynamicInterruptHack)}return t.containerState={},e.check(yc,d,f)(y)}function d(y){return i&&v(),h(r),m(y)}function f(y){return t.parser.lazy[t.now().line]=r!==n.length,o=t.now().offset,w(y)}function m(y){return t.containerState={},e.attempt(yc,p,w)(y)}function p(y){return r++,n.push([t.currentConstruct,t.containerState]),m(y)}function w(y){if(y===null){i&&v(),h(0),e.consume(y);return}return i=i||t.parser.flow(t.now()),e.enter("chunkFlow",{_tokenizer:i,contentType:"flow",previous:l}),S(y)}function S(y){if(y===null){I(e.exit("chunkFlow"),!0),h(0),e.consume(y);return}return H(y)?(e.consume(y),I(e.exit("chunkFlow")),r=0,t.interrupt=void 0,a):(e.consume(y),S)}function I(y,C){const N=t.sliceStream(y);if(C&&N.push(null),y.previous=l,l&&(l.next=y),l=y,i.defineSkip(y.start),i.write(N),t.parser.lazy[y.start.line]){let k=i.events.length;for(;k--;)if(i.events[k][1].start.offset<o&&(!i.events[k][1].end||i.events[k][1].end.offset>o))return;const j=t.events.length;let _=j,R,P;for(;_--;)if(t.events[_][0]==="exit"&&t.events[_][1].type==="chunkFlow"){if(R){P=t.events[_][1].end;break}R=!0}for(h(r),k=j;k<t.events.length;)t.events[k][1].end={...P},k++;_t(t.events,_+1,0,t.events.slice(j)),t.events.length=k}}function h(y){let C=n.length;for(;C-- >y;){const N=n[C];t.containerState=N[1],N[0].exit.call(t,e)}n.length=y}function v(){i.write([null]),l=void 0,i=void 0,t.containerState._closeFlow=void 0}}function Cy(e,t,n){return se(e,e.attempt(this.parser.constructs.document,t,n),"linePrefix",this.parser.constructs.disable.null.includes("codeIndented")?void 0:4)}function xc(e){if(e===null||qe(e)||xy(e))return 1;if(yy(e))return 2}function Ds(e,t,n){const r=[];let i=-1;for(;++i<e.length;){const l=e[i].resolveAll;l&&!r.includes(l)&&(t=l(t,n),r.push(l))}return t}const _a={name:"attention",resolveAll:jy,tokenize:Ey};function jy(e,t){let n=-1,r,i,l,o,a,s,c,d;for(;++n<e.length;)if(e[n][0]==="enter"&&e[n][1].type==="attentionSequence"&&e[n][1]._close){for(r=n;r--;)if(e[r][0]==="exit"&&e[r][1].type==="attentionSequence"&&e[r][1]._open&&t.sliceSerialize(e[r][1]).charCodeAt(0)===t.sliceSerialize(e[n][1]).charCodeAt(0)){if((e[r][1]._close||e[n][1]._open)&&(e[n][1].end.offset-e[n][1].start.offset)%3&&!((e[r][1].end.offset-e[r][1].start.offset+e[n][1].end.offset-e[n][1].start.offset)%3))continue;s=e[r][1].end.offset-e[r][1].start.offset>1&&e[n][1].end.offset-e[n][1].start.offset>1?2:1;const f={...e[r][1].end},m={...e[n][1].start};kc(f,-s),kc(m,s),o={type:s>1?"strongSequence":"emphasisSequence",start:f,end:{...e[r][1].end}},a={type:s>1?"strongSequence":"emphasisSequence",start:{...e[n][1].start},end:m},l={type:s>1?"strongText":"emphasisText",start:{...e[r][1].end},end:{...e[n][1].start}},i={type:s>1?"strong":"emphasis",start:{...o.start},end:{...a.end}},e[r][1].end={...o.start},e[n][1].start={...a.end},c=[],e[r][1].end.offset-e[r][1].start.offset&&(c=st(c,[["enter",e[r][1],t],["exit",e[r][1],t]])),c=st(c,[["enter",i,t],["enter",o,t],["exit",o,t],["enter",l,t]]),c=st(c,Ds(t.parser.constructs.insideSpan.null,e.slice(r+1,n),t)),c=st(c,[["exit",l,t],["enter",a,t],["exit",a,t],["exit",i,t]]),e[n][1].end.offset-e[n][1].start.offset?(d=2,c=st(c,[["enter",e[n][1],t],["exit",e[n][1],t]])):d=0,_t(e,r-1,n-r+3,c),n=r+c.length-d-2;break}}for(n=-1;++n<e.length;)e[n][1].type==="attentionSequence"&&(e[n][1].type="data");return e}function Ey(e,t){const n=this.parser.constructs.attentionMarkers.null,r=this.previous,i=xc(r);let l;return o;function o(s){return l=s,e.enter("attentionSequence"),a(s)}function a(s){if(s===l)return e.consume(s),a;const c=e.exit("attentionSequence"),d=xc(s),f=!d||d===2&&i||n.includes(s),m=!i||i===2&&d||n.includes(r);return c._open=!!(l===42?f:f&&(i||!m)),c._close=!!(l===42?m:m&&(d||!f)),t(s)}}function kc(e,t){e.column+=t,e.offset+=t,e._bufferIndex+=t}const Ny={name:"autolink",tokenize:_y};function _y(e,t,n){let r=0;return i;function i(p){return e.enter("autolink"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.enter("autolinkProtocol"),l}function l(p){return jt(p)?(e.consume(p),o):p===64?n(p):c(p)}function o(p){return p===43||p===45||p===46||et(p)?(r=1,a(p)):c(p)}function a(p){return p===58?(e.consume(p),r=0,s):(p===43||p===45||p===46||et(p))&&r++<32?(e.consume(p),a):(r=0,c(p))}function s(p){return p===62?(e.exit("autolinkProtocol"),e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):p===null||p===32||p===60||Ea(p)?n(p):(e.consume(p),s)}function c(p){return p===64?(e.consume(p),d):my(p)?(e.consume(p),c):n(p)}function d(p){return et(p)?f(p):n(p)}function f(p){return p===46?(e.consume(p),r=0,d):p===62?(e.exit("autolinkProtocol").type="autolinkEmail",e.enter("autolinkMarker"),e.consume(p),e.exit("autolinkMarker"),e.exit("autolink"),t):m(p)}function m(p){if((p===45||et(p))&&r++<63){const w=p===45?m:f;return e.consume(p),w}return n(p)}}const Ol={partial:!0,tokenize:zy};function zy(e,t,n){return r;function r(l){return ee(l)?se(e,i,"linePrefix")(l):i(l)}function i(l){return l===null||H(l)?t(l):n(l)}}const bp={continuation:{tokenize:Ly},exit:Py,name:"blockQuote",tokenize:Ty};function Ty(e,t,n){const r=this;return i;function i(o){if(o===62){const a=r.containerState;return a.open||(e.enter("blockQuote",{_container:!0}),a.open=!0),e.enter("blockQuotePrefix"),e.enter("blockQuoteMarker"),e.consume(o),e.exit("blockQuoteMarker"),l}return n(o)}function l(o){return ee(o)?(e.enter("blockQuotePrefixWhitespace"),e.consume(o),e.exit("blockQuotePrefixWhitespace"),e.exit("blockQuotePrefix"),t):(e.exit("blockQuotePrefix"),t(o))}}function Ly(e,t,n){const r=this;return i;function i(o){return ee(o)?se(e,l,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(o):l(o)}function l(o){return e.attempt(bp,t,n)(o)}}function Py(e){e.exit("blockQuote")}const Cp={name:"characterEscape",tokenize:Iy};function Iy(e,t,n){return r;function r(l){return e.enter("characterEscape"),e.enter("escapeMarker"),e.consume(l),e.exit("escapeMarker"),i}function i(l){return vy(l)?(e.enter("characterEscapeValue"),e.consume(l),e.exit("characterEscapeValue"),e.exit("characterEscape"),t):n(l)}}const jp={name:"characterReference",tokenize:My};function My(e,t,n){const r=this;let i=0,l,o;return a;function a(f){return e.enter("characterReference"),e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),s}function s(f){return f===35?(e.enter("characterReferenceMarkerNumeric"),e.consume(f),e.exit("characterReferenceMarkerNumeric"),c):(e.enter("characterReferenceValue"),l=31,o=et,d(f))}function c(f){return f===88||f===120?(e.enter("characterReferenceMarkerHexadecimal"),e.consume(f),e.exit("characterReferenceMarkerHexadecimal"),e.enter("characterReferenceValue"),l=6,o=gy,d):(e.enter("characterReferenceValue"),l=7,o=Na,d(f))}function d(f){if(f===59&&i){const m=e.exit("characterReferenceValue");return o===et&&!As(r.sliceSerialize(m))?n(f):(e.enter("characterReferenceMarker"),e.consume(f),e.exit("characterReferenceMarker"),e.exit("characterReference"),t)}return o(f)&&i++<l?(e.consume(f),d):n(f)}}const wc={partial:!0,tokenize:Dy},Sc={concrete:!0,name:"codeFenced",tokenize:Ay};function Ay(e,t,n){const r=this,i={partial:!0,tokenize:N};let l=0,o=0,a;return s;function s(k){return c(k)}function c(k){const j=r.events[r.events.length-1];return l=j&&j[1].type==="linePrefix"?j[2].sliceSerialize(j[1],!0).length:0,a=k,e.enter("codeFenced"),e.enter("codeFencedFence"),e.enter("codeFencedFenceSequence"),d(k)}function d(k){return k===a?(o++,e.consume(k),d):o<3?n(k):(e.exit("codeFencedFenceSequence"),ee(k)?se(e,f,"whitespace")(k):f(k))}function f(k){return k===null||H(k)?(e.exit("codeFencedFence"),r.interrupt?t(k):e.check(wc,S,C)(k)):(e.enter("codeFencedFenceInfo"),e.enter("chunkString",{contentType:"string"}),m(k))}function m(k){return k===null||H(k)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),f(k)):ee(k)?(e.exit("chunkString"),e.exit("codeFencedFenceInfo"),se(e,p,"whitespace")(k)):k===96&&k===a?n(k):(e.consume(k),m)}function p(k){return k===null||H(k)?f(k):(e.enter("codeFencedFenceMeta"),e.enter("chunkString",{contentType:"string"}),w(k))}function w(k){return k===null||H(k)?(e.exit("chunkString"),e.exit("codeFencedFenceMeta"),f(k)):k===96&&k===a?n(k):(e.consume(k),w)}function S(k){return e.attempt(i,C,I)(k)}function I(k){return e.enter("lineEnding"),e.consume(k),e.exit("lineEnding"),h}function h(k){return l>0&&ee(k)?se(e,v,"linePrefix",l+1)(k):v(k)}function v(k){return k===null||H(k)?e.check(wc,S,C)(k):(e.enter("codeFlowValue"),y(k))}function y(k){return k===null||H(k)?(e.exit("codeFlowValue"),v(k)):(e.consume(k),y)}function C(k){return e.exit("codeFenced"),t(k)}function N(k,j,_){let R=0;return P;function P(U){return k.enter("lineEnding"),k.consume(U),k.exit("lineEnding"),z}function z(U){return k.enter("codeFencedFence"),ee(U)?se(k,D,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(U):D(U)}function D(U){return U===a?(k.enter("codeFencedFenceSequence"),W(U)):_(U)}function W(U){return U===a?(R++,k.consume(U),W):R>=o?(k.exit("codeFencedFenceSequence"),ee(U)?se(k,X,"whitespace")(U):X(U)):_(U)}function X(U){return U===null||H(U)?(k.exit("codeFencedFence"),j(U)):_(U)}}}function Dy(e,t,n){const r=this;return i;function i(o){return o===null?n(o):(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}const yo={name:"codeIndented",tokenize:Fy},Ry={partial:!0,tokenize:Oy};function Fy(e,t,n){const r=this;return i;function i(c){return e.enter("codeIndented"),se(e,l,"linePrefix",5)(c)}function l(c){const d=r.events[r.events.length-1];return d&&d[1].type==="linePrefix"&&d[2].sliceSerialize(d[1],!0).length>=4?o(c):n(c)}function o(c){return c===null?s(c):H(c)?e.attempt(Ry,o,s)(c):(e.enter("codeFlowValue"),a(c))}function a(c){return c===null||H(c)?(e.exit("codeFlowValue"),o(c)):(e.consume(c),a)}function s(c){return e.exit("codeIndented"),t(c)}}function Oy(e,t,n){const r=this;return i;function i(o){return r.parser.lazy[r.now().line]?n(o):H(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),i):se(e,l,"linePrefix",5)(o)}function l(o){const a=r.events[r.events.length-1];return a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):H(o)?i(o):n(o)}}const By={name:"codeText",previous:Uy,resolve:$y,tokenize:Hy};function $y(e){let t=e.length-4,n=3,r,i;if((e[n][1].type==="lineEnding"||e[n][1].type==="space")&&(e[t][1].type==="lineEnding"||e[t][1].type==="space")){for(r=n;++r<t;)if(e[r][1].type==="codeTextData"){e[n][1].type="codeTextPadding",e[t][1].type="codeTextPadding",n+=2,t-=2;break}}for(r=n-1,t++;++r<=t;)i===void 0?r!==t&&e[r][1].type!=="lineEnding"&&(i=r):(r===t||e[r][1].type==="lineEnding")&&(e[i][1].type="codeTextData",r!==i+2&&(e[i][1].end=e[r-1][1].end,e.splice(i+2,r-i-2),t-=r-i-2,r=i+2),i=void 0);return e}function Uy(e){return e!==96||this.events[this.events.length-1][1].type==="characterEscape"}function Hy(e,t,n){let r=0,i,l;return o;function o(f){return e.enter("codeText"),e.enter("codeTextSequence"),a(f)}function a(f){return f===96?(e.consume(f),r++,a):(e.exit("codeTextSequence"),s(f))}function s(f){return f===null?n(f):f===32?(e.enter("space"),e.consume(f),e.exit("space"),s):f===96?(l=e.enter("codeTextSequence"),i=0,d(f)):H(f)?(e.enter("lineEnding"),e.consume(f),e.exit("lineEnding"),s):(e.enter("codeTextData"),c(f))}function c(f){return f===null||f===32||f===96||H(f)?(e.exit("codeTextData"),s(f)):(e.consume(f),c)}function d(f){return f===96?(e.consume(f),i++,d):i===r?(e.exit("codeTextSequence"),e.exit("codeText"),t(f)):(l.type="codeTextData",c(f))}}class Vy{constructor(t){this.left=t?[...t]:[],this.right=[]}get(t){if(t<0||t>=this.left.length+this.right.length)throw new RangeError("Cannot access index `"+t+"` in a splice buffer of size `"+(this.left.length+this.right.length)+"`");return t<this.left.length?this.left[t]:this.right[this.right.length-t+this.left.length-1]}get length(){return this.left.length+this.right.length}shift(){return this.setCursor(0),this.right.pop()}slice(t,n){const r=n??Number.POSITIVE_INFINITY;return r<this.left.length?this.left.slice(t,r):t>this.left.length?this.right.slice(this.right.length-r+this.left.length,this.right.length-t+this.left.length).reverse():this.left.slice(t).concat(this.right.slice(this.right.length-r+this.left.length).reverse())}splice(t,n,r){const i=n||0;this.setCursor(Math.trunc(t));const l=this.right.splice(this.right.length-i,Number.POSITIVE_INFINITY);return r&&Cr(this.left,r),l.reverse()}pop(){return this.setCursor(Number.POSITIVE_INFINITY),this.left.pop()}push(t){this.setCursor(Number.POSITIVE_INFINITY),this.left.push(t)}pushMany(t){this.setCursor(Number.POSITIVE_INFINITY),Cr(this.left,t)}unshift(t){this.setCursor(0),this.right.push(t)}unshiftMany(t){this.setCursor(0),Cr(this.right,t.reverse())}setCursor(t){if(!(t===this.left.length||t>this.left.length&&this.right.length===0||t<0&&this.left.length===0))if(t<this.left.length){const n=this.left.splice(t,Number.POSITIVE_INFINITY);Cr(this.right,n.reverse())}else{const n=this.right.splice(this.left.length+this.right.length-t,Number.POSITIVE_INFINITY);Cr(this.left,n.reverse())}}}function Cr(e,t){let n=0;if(t.length<1e4)e.push(...t);else for(;n<t.length;)e.push(...t.slice(n,n+1e4)),n+=1e4}function Ep(e){const t={};let n=-1,r,i,l,o,a,s,c;const d=new Vy(e);for(;++n<d.length;){for(;n in t;)n=t[n];if(r=d.get(n),n&&r[1].type==="chunkFlow"&&d.get(n-1)[1].type==="listItemPrefix"&&(s=r[1]._tokenizer.events,l=0,l<s.length&&s[l][1].type==="lineEndingBlank"&&(l+=2),l<s.length&&s[l][1].type==="content"))for(;++l<s.length&&s[l][1].type!=="content";)s[l][1].type==="chunkText"&&(s[l][1]._isInFirstContentOfListItem=!0,l++);if(r[0]==="enter")r[1].contentType&&(Object.assign(t,Wy(d,n)),n=t[n],c=!0);else if(r[1]._container){for(l=n,i=void 0;l--;)if(o=d.get(l),o[1].type==="lineEnding"||o[1].type==="lineEndingBlank")o[0]==="enter"&&(i&&(d.get(i)[1].type="lineEndingBlank"),o[1].type="lineEnding",i=l);else if(!(o[1].type==="linePrefix"||o[1].type==="listItemIndent"))break;i&&(r[1].end={...d.get(i)[1].start},a=d.slice(i,n),a.unshift(r),d.splice(i,n-i+1,a))}}return _t(e,0,Number.POSITIVE_INFINITY,d.slice(0)),!c}function Wy(e,t){const n=e.get(t)[1],r=e.get(t)[2];let i=t-1;const l=[];let o=n._tokenizer;o||(o=r.parser[n.contentType](n.start),n._contentTypeTextTrailing&&(o._contentTypeTextTrailing=!0));const a=o.events,s=[],c={};let d,f,m=-1,p=n,w=0,S=0;const I=[S];for(;p;){for(;e.get(++i)[1]!==p;);l.push(i),p._tokenizer||(d=r.sliceStream(p),p.next||d.push(null),f&&o.defineSkip(p.start),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=!0),o.write(d),p._isInFirstContentOfListItem&&(o._gfmTasklistFirstContentOfListItem=void 0)),f=p,p=p.next}for(p=n;++m<a.length;)a[m][0]==="exit"&&a[m-1][0]==="enter"&&a[m][1].type===a[m-1][1].type&&a[m][1].start.line!==a[m][1].end.line&&(S=m+1,I.push(S),p._tokenizer=void 0,p.previous=void 0,p=p.next);for(o.events=[],p?(p._tokenizer=void 0,p.previous=void 0):I.pop(),m=I.length;m--;){const h=a.slice(I[m],I[m+1]),v=l.pop();s.push([v,v+h.length-1]),e.splice(v,2,h)}for(s.reverse(),m=-1;++m<s.length;)c[w+s[m][0]]=w+s[m][1],w+=s[m][1]-s[m][0]-1;return c}const Qy={resolve:Ky,tokenize:Yy},qy={partial:!0,tokenize:Xy};function Ky(e){return Ep(e),e}function Yy(e,t){let n;return r;function r(a){return e.enter("content"),n=e.enter("chunkContent",{contentType:"content"}),i(a)}function i(a){return a===null?l(a):H(a)?e.check(qy,o,l)(a):(e.consume(a),i)}function l(a){return e.exit("chunkContent"),e.exit("content"),t(a)}function o(a){return e.consume(a),e.exit("chunkContent"),n.next=e.enter("chunkContent",{contentType:"content",previous:n}),n=n.next,i}}function Xy(e,t,n){const r=this;return i;function i(o){return e.exit("chunkContent"),e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),se(e,l,"linePrefix")}function l(o){if(o===null||H(o))return n(o);const a=r.events[r.events.length-1];return!r.parser.constructs.disable.null.includes("codeIndented")&&a&&a[1].type==="linePrefix"&&a[2].sliceSerialize(a[1],!0).length>=4?t(o):e.interrupt(r.parser.constructs.flow,n,t)(o)}}function Np(e,t,n,r,i,l,o,a,s){const c=s||Number.POSITIVE_INFINITY;let d=0;return f;function f(h){return h===60?(e.enter(r),e.enter(i),e.enter(l),e.consume(h),e.exit(l),m):h===null||h===32||h===41||Ea(h)?n(h):(e.enter(r),e.enter(o),e.enter(a),e.enter("chunkString",{contentType:"string"}),S(h))}function m(h){return h===62?(e.enter(l),e.consume(h),e.exit(l),e.exit(i),e.exit(r),t):(e.enter(a),e.enter("chunkString",{contentType:"string"}),p(h))}function p(h){return h===62?(e.exit("chunkString"),e.exit(a),m(h)):h===null||h===60||H(h)?n(h):(e.consume(h),h===92?w:p)}function w(h){return h===60||h===62||h===92?(e.consume(h),p):p(h)}function S(h){return!d&&(h===null||h===41||qe(h))?(e.exit("chunkString"),e.exit(a),e.exit(o),e.exit(r),t(h)):d<c&&h===40?(e.consume(h),d++,S):h===41?(e.consume(h),d--,S):h===null||h===32||h===40||Ea(h)?n(h):(e.consume(h),h===92?I:S)}function I(h){return h===40||h===41||h===92?(e.consume(h),S):S(h)}}function _p(e,t,n,r,i,l){const o=this;let a=0,s;return c;function c(p){return e.enter(r),e.enter(i),e.consume(p),e.exit(i),e.enter(l),d}function d(p){return a>999||p===null||p===91||p===93&&!s||p===94&&!a&&"_hiddenFootnoteSupport"in o.parser.constructs?n(p):p===93?(e.exit(l),e.enter(i),e.consume(p),e.exit(i),e.exit(r),t):H(p)?(e.enter("lineEnding"),e.consume(p),e.exit("lineEnding"),d):(e.enter("chunkString",{contentType:"string"}),f(p))}function f(p){return p===null||p===91||p===93||H(p)||a++>999?(e.exit("chunkString"),d(p)):(e.consume(p),s||(s=!ee(p)),p===92?m:f)}function m(p){return p===91||p===92||p===93?(e.consume(p),a++,f):f(p)}}function zp(e,t,n,r,i,l){let o;return a;function a(m){return m===34||m===39||m===40?(e.enter(r),e.enter(i),e.consume(m),e.exit(i),o=m===40?41:m,s):n(m)}function s(m){return m===o?(e.enter(i),e.consume(m),e.exit(i),e.exit(r),t):(e.enter(l),c(m))}function c(m){return m===o?(e.exit(l),s(o)):m===null?n(m):H(m)?(e.enter("lineEnding"),e.consume(m),e.exit("lineEnding"),se(e,c,"linePrefix")):(e.enter("chunkString",{contentType:"string"}),d(m))}function d(m){return m===o||m===null||H(m)?(e.exit("chunkString"),c(m)):(e.consume(m),m===92?f:d)}function f(m){return m===o||m===92?(e.consume(m),d):d(m)}}function Br(e,t){let n;return r;function r(i){return H(i)?(e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),n=!0,r):ee(i)?se(e,r,n?"linePrefix":"lineSuffix")(i):t(i)}}const Gy={name:"definition",tokenize:Zy},Jy={partial:!0,tokenize:ex};function Zy(e,t,n){const r=this;let i;return l;function l(p){return e.enter("definition"),o(p)}function o(p){return _p.call(r,e,a,n,"definitionLabel","definitionLabelMarker","definitionLabelString")(p)}function a(p){return i=tr(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)),p===58?(e.enter("definitionMarker"),e.consume(p),e.exit("definitionMarker"),s):n(p)}function s(p){return qe(p)?Br(e,c)(p):c(p)}function c(p){return Np(e,d,n,"definitionDestination","definitionDestinationLiteral","definitionDestinationLiteralMarker","definitionDestinationRaw","definitionDestinationString")(p)}function d(p){return e.attempt(Jy,f,f)(p)}function f(p){return ee(p)?se(e,m,"whitespace")(p):m(p)}function m(p){return p===null||H(p)?(e.exit("definition"),r.parser.defined.push(i),t(p)):n(p)}}function ex(e,t,n){return r;function r(a){return qe(a)?Br(e,i)(a):n(a)}function i(a){return zp(e,l,n,"definitionTitle","definitionTitleMarker","definitionTitleString")(a)}function l(a){return ee(a)?se(e,o,"whitespace")(a):o(a)}function o(a){return a===null||H(a)?t(a):n(a)}}const tx={name:"hardBreakEscape",tokenize:nx};function nx(e,t,n){return r;function r(l){return e.enter("hardBreakEscape"),e.consume(l),i}function i(l){return H(l)?(e.exit("hardBreakEscape"),t(l)):n(l)}}const rx={name:"headingAtx",resolve:ix,tokenize:lx};function ix(e,t){let n=e.length-2,r=3,i,l;return e[r][1].type==="whitespace"&&(r+=2),n-2>r&&e[n][1].type==="whitespace"&&(n-=2),e[n][1].type==="atxHeadingSequence"&&(r===n-1||n-4>r&&e[n-2][1].type==="whitespace")&&(n-=r+1===n?2:4),n>r&&(i={type:"atxHeadingText",start:e[r][1].start,end:e[n][1].end},l={type:"chunkText",start:e[r][1].start,end:e[n][1].end,contentType:"text"},_t(e,r,n-r+1,[["enter",i,t],["enter",l,t],["exit",l,t],["exit",i,t]])),e}function lx(e,t,n){let r=0;return i;function i(d){return e.enter("atxHeading"),l(d)}function l(d){return e.enter("atxHeadingSequence"),o(d)}function o(d){return d===35&&r++<6?(e.consume(d),o):d===null||qe(d)?(e.exit("atxHeadingSequence"),a(d)):n(d)}function a(d){return d===35?(e.enter("atxHeadingSequence"),s(d)):d===null||H(d)?(e.exit("atxHeading"),t(d)):ee(d)?se(e,a,"whitespace")(d):(e.enter("atxHeadingText"),c(d))}function s(d){return d===35?(e.consume(d),s):(e.exit("atxHeadingSequence"),a(d))}function c(d){return d===null||d===35||qe(d)?(e.exit("atxHeadingText"),a(d)):(e.consume(d),c)}}const ox=["address","article","aside","base","basefont","blockquote","body","caption","center","col","colgroup","dd","details","dialog","dir","div","dl","dt","fieldset","figcaption","figure","footer","form","frame","frameset","h1","h2","h3","h4","h5","h6","head","header","hr","html","iframe","legend","li","link","main","menu","menuitem","nav","noframes","ol","optgroup","option","p","param","search","section","summary","table","tbody","td","tfoot","th","thead","title","tr","track","ul"],bc=["pre","script","style","textarea"],ax={concrete:!0,name:"htmlFlow",resolveTo:cx,tokenize:dx},sx={partial:!0,tokenize:px},ux={partial:!0,tokenize:fx};function cx(e){let t=e.length;for(;t--&&!(e[t][0]==="enter"&&e[t][1].type==="htmlFlow"););return t>1&&e[t-2][1].type==="linePrefix"&&(e[t][1].start=e[t-2][1].start,e[t+1][1].start=e[t-2][1].start,e.splice(t-2,2)),e}function dx(e,t,n){const r=this;let i,l,o,a,s;return c;function c(x){return d(x)}function d(x){return e.enter("htmlFlow"),e.enter("htmlFlowData"),e.consume(x),f}function f(x){return x===33?(e.consume(x),m):x===47?(e.consume(x),l=!0,S):x===63?(e.consume(x),i=3,r.interrupt?t:g):jt(x)?(e.consume(x),o=String.fromCharCode(x),I):n(x)}function m(x){return x===45?(e.consume(x),i=2,p):x===91?(e.consume(x),i=5,a=0,w):jt(x)?(e.consume(x),i=4,r.interrupt?t:g):n(x)}function p(x){return x===45?(e.consume(x),r.interrupt?t:g):n(x)}function w(x){const ne="CDATA[";return x===ne.charCodeAt(a++)?(e.consume(x),a===ne.length?r.interrupt?t:D:w):n(x)}function S(x){return jt(x)?(e.consume(x),o=String.fromCharCode(x),I):n(x)}function I(x){if(x===null||x===47||x===62||qe(x)){const ne=x===47,be=o.toLowerCase();return!ne&&!l&&bc.includes(be)?(i=1,r.interrupt?t(x):D(x)):ox.includes(o.toLowerCase())?(i=6,ne?(e.consume(x),h):r.interrupt?t(x):D(x)):(i=7,r.interrupt&&!r.parser.lazy[r.now().line]?n(x):l?v(x):y(x))}return x===45||et(x)?(e.consume(x),o+=String.fromCharCode(x),I):n(x)}function h(x){return x===62?(e.consume(x),r.interrupt?t:D):n(x)}function v(x){return ee(x)?(e.consume(x),v):P(x)}function y(x){return x===47?(e.consume(x),P):x===58||x===95||jt(x)?(e.consume(x),C):ee(x)?(e.consume(x),y):P(x)}function C(x){return x===45||x===46||x===58||x===95||et(x)?(e.consume(x),C):N(x)}function N(x){return x===61?(e.consume(x),k):ee(x)?(e.consume(x),N):y(x)}function k(x){return x===null||x===60||x===61||x===62||x===96?n(x):x===34||x===39?(e.consume(x),s=x,j):ee(x)?(e.consume(x),k):_(x)}function j(x){return x===s?(e.consume(x),s=null,R):x===null||H(x)?n(x):(e.consume(x),j)}function _(x){return x===null||x===34||x===39||x===47||x===60||x===61||x===62||x===96||qe(x)?N(x):(e.consume(x),_)}function R(x){return x===47||x===62||ee(x)?y(x):n(x)}function P(x){return x===62?(e.consume(x),z):n(x)}function z(x){return x===null||H(x)?D(x):ee(x)?(e.consume(x),z):n(x)}function D(x){return x===45&&i===2?(e.consume(x),Q):x===60&&i===1?(e.consume(x),ie):x===62&&i===4?(e.consume(x),L):x===63&&i===3?(e.consume(x),g):x===93&&i===5?(e.consume(x),E):H(x)&&(i===6||i===7)?(e.exit("htmlFlowData"),e.check(sx,$,W)(x)):x===null||H(x)?(e.exit("htmlFlowData"),W(x)):(e.consume(x),D)}function W(x){return e.check(ux,X,$)(x)}function X(x){return e.enter("lineEnding"),e.consume(x),e.exit("lineEnding"),U}function U(x){return x===null||H(x)?W(x):(e.enter("htmlFlowData"),D(x))}function Q(x){return x===45?(e.consume(x),g):D(x)}function ie(x){return x===47?(e.consume(x),o="",b):D(x)}function b(x){if(x===62){const ne=o.toLowerCase();return bc.includes(ne)?(e.consume(x),L):D(x)}return jt(x)&&o.length<8?(e.consume(x),o+=String.fromCharCode(x),b):D(x)}function E(x){return x===93?(e.consume(x),g):D(x)}function g(x){return x===62?(e.consume(x),L):x===45&&i===2?(e.consume(x),g):D(x)}function L(x){return x===null||H(x)?(e.exit("htmlFlowData"),$(x)):(e.consume(x),L)}function $(x){return e.exit("htmlFlow"),t(x)}}function fx(e,t,n){const r=this;return i;function i(o){return H(o)?(e.enter("lineEnding"),e.consume(o),e.exit("lineEnding"),l):n(o)}function l(o){return r.parser.lazy[r.now().line]?n(o):t(o)}}function px(e,t,n){return r;function r(i){return e.enter("lineEnding"),e.consume(i),e.exit("lineEnding"),e.attempt(Ol,t,n)}}const hx={name:"htmlText",tokenize:mx};function mx(e,t,n){const r=this;let i,l,o;return a;function a(g){return e.enter("htmlText"),e.enter("htmlTextData"),e.consume(g),s}function s(g){return g===33?(e.consume(g),c):g===47?(e.consume(g),N):g===63?(e.consume(g),y):jt(g)?(e.consume(g),_):n(g)}function c(g){return g===45?(e.consume(g),d):g===91?(e.consume(g),l=0,w):jt(g)?(e.consume(g),v):n(g)}function d(g){return g===45?(e.consume(g),p):n(g)}function f(g){return g===null?n(g):g===45?(e.consume(g),m):H(g)?(o=f,ie(g)):(e.consume(g),f)}function m(g){return g===45?(e.consume(g),p):f(g)}function p(g){return g===62?Q(g):g===45?m(g):f(g)}function w(g){const L="CDATA[";return g===L.charCodeAt(l++)?(e.consume(g),l===L.length?S:w):n(g)}function S(g){return g===null?n(g):g===93?(e.consume(g),I):H(g)?(o=S,ie(g)):(e.consume(g),S)}function I(g){return g===93?(e.consume(g),h):S(g)}function h(g){return g===62?Q(g):g===93?(e.consume(g),h):S(g)}function v(g){return g===null||g===62?Q(g):H(g)?(o=v,ie(g)):(e.consume(g),v)}function y(g){return g===null?n(g):g===63?(e.consume(g),C):H(g)?(o=y,ie(g)):(e.consume(g),y)}function C(g){return g===62?Q(g):y(g)}function N(g){return jt(g)?(e.consume(g),k):n(g)}function k(g){return g===45||et(g)?(e.consume(g),k):j(g)}function j(g){return H(g)?(o=j,ie(g)):ee(g)?(e.consume(g),j):Q(g)}function _(g){return g===45||et(g)?(e.consume(g),_):g===47||g===62||qe(g)?R(g):n(g)}function R(g){return g===47?(e.consume(g),Q):g===58||g===95||jt(g)?(e.consume(g),P):H(g)?(o=R,ie(g)):ee(g)?(e.consume(g),R):Q(g)}function P(g){return g===45||g===46||g===58||g===95||et(g)?(e.consume(g),P):z(g)}function z(g){return g===61?(e.consume(g),D):H(g)?(o=z,ie(g)):ee(g)?(e.consume(g),z):R(g)}function D(g){return g===null||g===60||g===61||g===62||g===96?n(g):g===34||g===39?(e.consume(g),i=g,W):H(g)?(o=D,ie(g)):ee(g)?(e.consume(g),D):(e.consume(g),X)}function W(g){return g===i?(e.consume(g),i=void 0,U):g===null?n(g):H(g)?(o=W,ie(g)):(e.consume(g),W)}function X(g){return g===null||g===34||g===39||g===60||g===61||g===96?n(g):g===47||g===62||qe(g)?R(g):(e.consume(g),X)}function U(g){return g===47||g===62||qe(g)?R(g):n(g)}function Q(g){return g===62?(e.consume(g),e.exit("htmlTextData"),e.exit("htmlText"),t):n(g)}function ie(g){return e.exit("htmlTextData"),e.enter("lineEnding"),e.consume(g),e.exit("lineEnding"),b}function b(g){return ee(g)?se(e,E,"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(g):E(g)}function E(g){return e.enter("htmlTextData"),o(g)}}const Rs={name:"labelEnd",resolveAll:xx,resolveTo:kx,tokenize:wx},gx={tokenize:Sx},vx={tokenize:bx},yx={tokenize:Cx};function xx(e){let t=-1;const n=[];for(;++t<e.length;){const r=e[t][1];if(n.push(e[t]),r.type==="labelImage"||r.type==="labelLink"||r.type==="labelEnd"){const i=r.type==="labelImage"?4:2;r.type="data",t+=i}}return e.length!==n.length&&_t(e,0,e.length,n),e}function kx(e,t){let n=e.length,r=0,i,l,o,a;for(;n--;)if(i=e[n][1],l){if(i.type==="link"||i.type==="labelLink"&&i._inactive)break;e[n][0]==="enter"&&i.type==="labelLink"&&(i._inactive=!0)}else if(o){if(e[n][0]==="enter"&&(i.type==="labelImage"||i.type==="labelLink")&&!i._balanced&&(l=n,i.type!=="labelLink")){r=2;break}}else i.type==="labelEnd"&&(o=n);const s={type:e[l][1].type==="labelLink"?"link":"image",start:{...e[l][1].start},end:{...e[e.length-1][1].end}},c={type:"label",start:{...e[l][1].start},end:{...e[o][1].end}},d={type:"labelText",start:{...e[l+r+2][1].end},end:{...e[o-2][1].start}};return a=[["enter",s,t],["enter",c,t]],a=st(a,e.slice(l+1,l+r+3)),a=st(a,[["enter",d,t]]),a=st(a,Ds(t.parser.constructs.insideSpan.null,e.slice(l+r+4,o-3),t)),a=st(a,[["exit",d,t],e[o-2],e[o-1],["exit",c,t]]),a=st(a,e.slice(o+1)),a=st(a,[["exit",s,t]]),_t(e,l,e.length,a),e}function wx(e,t,n){const r=this;let i=r.events.length,l,o;for(;i--;)if((r.events[i][1].type==="labelImage"||r.events[i][1].type==="labelLink")&&!r.events[i][1]._balanced){l=r.events[i][1];break}return a;function a(m){return l?l._inactive?f(m):(o=r.parser.defined.includes(tr(r.sliceSerialize({start:l.end,end:r.now()}))),e.enter("labelEnd"),e.enter("labelMarker"),e.consume(m),e.exit("labelMarker"),e.exit("labelEnd"),s):n(m)}function s(m){return m===40?e.attempt(gx,d,o?d:f)(m):m===91?e.attempt(vx,d,o?c:f)(m):o?d(m):f(m)}function c(m){return e.attempt(yx,d,f)(m)}function d(m){return t(m)}function f(m){return l._balanced=!0,n(m)}}function Sx(e,t,n){return r;function r(f){return e.enter("resource"),e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),i}function i(f){return qe(f)?Br(e,l)(f):l(f)}function l(f){return f===41?d(f):Np(e,o,a,"resourceDestination","resourceDestinationLiteral","resourceDestinationLiteralMarker","resourceDestinationRaw","resourceDestinationString",32)(f)}function o(f){return qe(f)?Br(e,s)(f):d(f)}function a(f){return n(f)}function s(f){return f===34||f===39||f===40?zp(e,c,n,"resourceTitle","resourceTitleMarker","resourceTitleString")(f):d(f)}function c(f){return qe(f)?Br(e,d)(f):d(f)}function d(f){return f===41?(e.enter("resourceMarker"),e.consume(f),e.exit("resourceMarker"),e.exit("resource"),t):n(f)}}function bx(e,t,n){const r=this;return i;function i(a){return _p.call(r,e,l,o,"reference","referenceMarker","referenceString")(a)}function l(a){return r.parser.defined.includes(tr(r.sliceSerialize(r.events[r.events.length-1][1]).slice(1,-1)))?t(a):n(a)}function o(a){return n(a)}}function Cx(e,t,n){return r;function r(l){return e.enter("reference"),e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),i}function i(l){return l===93?(e.enter("referenceMarker"),e.consume(l),e.exit("referenceMarker"),e.exit("reference"),t):n(l)}}const jx={name:"labelStartImage",resolveAll:Rs.resolveAll,tokenize:Ex};function Ex(e,t,n){const r=this;return i;function i(a){return e.enter("labelImage"),e.enter("labelImageMarker"),e.consume(a),e.exit("labelImageMarker"),l}function l(a){return a===91?(e.enter("labelMarker"),e.consume(a),e.exit("labelMarker"),e.exit("labelImage"),o):n(a)}function o(a){return a===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(a):t(a)}}const Nx={name:"labelStartLink",resolveAll:Rs.resolveAll,tokenize:_x};function _x(e,t,n){const r=this;return i;function i(o){return e.enter("labelLink"),e.enter("labelMarker"),e.consume(o),e.exit("labelMarker"),e.exit("labelLink"),l}function l(o){return o===94&&"_hiddenFootnoteSupport"in r.parser.constructs?n(o):t(o)}}const xo={name:"lineEnding",tokenize:zx};function zx(e,t){return n;function n(r){return e.enter("lineEnding"),e.consume(r),e.exit("lineEnding"),se(e,t,"linePrefix")}}const qi={name:"thematicBreak",tokenize:Tx};function Tx(e,t,n){let r=0,i;return l;function l(c){return e.enter("thematicBreak"),o(c)}function o(c){return i=c,a(c)}function a(c){return c===i?(e.enter("thematicBreakSequence"),s(c)):r>=3&&(c===null||H(c))?(e.exit("thematicBreak"),t(c)):n(c)}function s(c){return c===i?(e.consume(c),r++,s):(e.exit("thematicBreakSequence"),ee(c)?se(e,a,"whitespace")(c):a(c))}}const $e={continuation:{tokenize:Mx},exit:Dx,name:"list",tokenize:Ix},Lx={partial:!0,tokenize:Rx},Px={partial:!0,tokenize:Ax};function Ix(e,t,n){const r=this,i=r.events[r.events.length-1];let l=i&&i[1].type==="linePrefix"?i[2].sliceSerialize(i[1],!0).length:0,o=0;return a;function a(p){const w=r.containerState.type||(p===42||p===43||p===45?"listUnordered":"listOrdered");if(w==="listUnordered"?!r.containerState.marker||p===r.containerState.marker:Na(p)){if(r.containerState.type||(r.containerState.type=w,e.enter(w,{_container:!0})),w==="listUnordered")return e.enter("listItemPrefix"),p===42||p===45?e.check(qi,n,c)(p):c(p);if(!r.interrupt||p===49)return e.enter("listItemPrefix"),e.enter("listItemValue"),s(p)}return n(p)}function s(p){return Na(p)&&++o<10?(e.consume(p),s):(!r.interrupt||o<2)&&(r.containerState.marker?p===r.containerState.marker:p===41||p===46)?(e.exit("listItemValue"),c(p)):n(p)}function c(p){return e.enter("listItemMarker"),e.consume(p),e.exit("listItemMarker"),r.containerState.marker=r.containerState.marker||p,e.check(Ol,r.interrupt?n:d,e.attempt(Lx,m,f))}function d(p){return r.containerState.initialBlankLine=!0,l++,m(p)}function f(p){return ee(p)?(e.enter("listItemPrefixWhitespace"),e.consume(p),e.exit("listItemPrefixWhitespace"),m):n(p)}function m(p){return r.containerState.size=l+r.sliceSerialize(e.exit("listItemPrefix"),!0).length,t(p)}}function Mx(e,t,n){const r=this;return r.containerState._closeFlow=void 0,e.check(Ol,i,l);function i(a){return r.containerState.furtherBlankLines=r.containerState.furtherBlankLines||r.containerState.initialBlankLine,se(e,t,"listItemIndent",r.containerState.size+1)(a)}function l(a){return r.containerState.furtherBlankLines||!ee(a)?(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,o(a)):(r.containerState.furtherBlankLines=void 0,r.containerState.initialBlankLine=void 0,e.attempt(Px,t,o)(a))}function o(a){return r.containerState._closeFlow=!0,r.interrupt=void 0,se(e,e.attempt($e,t,n),"linePrefix",r.parser.constructs.disable.null.includes("codeIndented")?void 0:4)(a)}}function Ax(e,t,n){const r=this;return se(e,i,"listItemIndent",r.containerState.size+1);function i(l){const o=r.events[r.events.length-1];return o&&o[1].type==="listItemIndent"&&o[2].sliceSerialize(o[1],!0).length===r.containerState.size?t(l):n(l)}}function Dx(e){e.exit(this.containerState.type)}function Rx(e,t,n){const r=this;return se(e,i,"listItemPrefixWhitespace",r.parser.constructs.disable.null.includes("codeIndented")?void 0:5);function i(l){const o=r.events[r.events.length-1];return!ee(l)&&o&&o[1].type==="listItemPrefixWhitespace"?t(l):n(l)}}const Cc={name:"setextUnderline",resolveTo:Fx,tokenize:Ox};function Fx(e,t){let n=e.length,r,i,l;for(;n--;)if(e[n][0]==="enter"){if(e[n][1].type==="content"){r=n;break}e[n][1].type==="paragraph"&&(i=n)}else e[n][1].type==="content"&&e.splice(n,1),!l&&e[n][1].type==="definition"&&(l=n);const o={type:"setextHeading",start:{...e[r][1].start},end:{...e[e.length-1][1].end}};return e[i][1].type="setextHeadingText",l?(e.splice(i,0,["enter",o,t]),e.splice(l+1,0,["exit",e[r][1],t]),e[r][1].end={...e[l][1].end}):e[r][1]=o,e.push(["exit",o,t]),e}function Ox(e,t,n){const r=this;let i;return l;function l(c){let d=r.events.length,f;for(;d--;)if(r.events[d][1].type!=="lineEnding"&&r.events[d][1].type!=="linePrefix"&&r.events[d][1].type!=="content"){f=r.events[d][1].type==="paragraph";break}return!r.parser.lazy[r.now().line]&&(r.interrupt||f)?(e.enter("setextHeadingLine"),i=c,o(c)):n(c)}function o(c){return e.enter("setextHeadingLineSequence"),a(c)}function a(c){return c===i?(e.consume(c),a):(e.exit("setextHeadingLineSequence"),ee(c)?se(e,s,"lineSuffix")(c):s(c))}function s(c){return c===null||H(c)?(e.exit("setextHeadingLine"),t(c)):n(c)}}const Bx={tokenize:$x};function $x(e){const t=this,n=e.attempt(Ol,r,e.attempt(this.parser.constructs.flowInitial,i,se(e,e.attempt(this.parser.constructs.flow,i,e.attempt(Qy,i)),"linePrefix")));return n;function r(l){if(l===null){e.consume(l);return}return e.enter("lineEndingBlank"),e.consume(l),e.exit("lineEndingBlank"),t.currentConstruct=void 0,n}function i(l){if(l===null){e.consume(l);return}return e.enter("lineEnding"),e.consume(l),e.exit("lineEnding"),t.currentConstruct=void 0,n}}const Ux={resolveAll:Lp()},Hx=Tp("string"),Vx=Tp("text");function Tp(e){return{resolveAll:Lp(e==="text"?Wx:void 0),tokenize:t};function t(n){const r=this,i=this.parser.constructs[e],l=n.attempt(i,o,a);return o;function o(d){return c(d)?l(d):a(d)}function a(d){if(d===null){n.consume(d);return}return n.enter("data"),n.consume(d),s}function s(d){return c(d)?(n.exit("data"),l(d)):(n.consume(d),s)}function c(d){if(d===null)return!0;const f=i[d];let m=-1;if(f)for(;++m<f.length;){const p=f[m];if(!p.previous||p.previous.call(r,r.previous))return!0}return!1}}}function Lp(e){return t;function t(n,r){let i=-1,l;for(;++i<=n.length;)l===void 0?n[i]&&n[i][1].type==="data"&&(l=i,i++):(!n[i]||n[i][1].type!=="data")&&(i!==l+2&&(n[l][1].end=n[i-1][1].end,n.splice(l+2,i-l-2),i=l+2),l=void 0);return e?e(n,r):n}}function Wx(e,t){let n=0;for(;++n<=e.length;)if((n===e.length||e[n][1].type==="lineEnding")&&e[n-1][1].type==="data"){const r=e[n-1][1],i=t.sliceStream(r);let l=i.length,o=-1,a=0,s;for(;l--;){const c=i[l];if(typeof c=="string"){for(o=c.length;c.charCodeAt(o-1)===32;)a++,o--;if(o)break;o=-1}else if(c===-2)s=!0,a++;else if(c!==-1){l++;break}}if(t._contentTypeTextTrailing&&n===e.length&&(a=0),a){const c={type:n===e.length||s||a<2?"lineSuffix":"hardBreakTrailing",start:{_bufferIndex:l?o:r.start._bufferIndex+o,_index:r.start._index+l,line:r.end.line,column:r.end.column-a,offset:r.end.offset-a},end:{...r.end}};r.end={...c.start},r.start.offset===r.end.offset?Object.assign(r,c):(e.splice(n,0,["enter",c,t],["exit",c,t]),n+=2)}n++}return e}const Qx={42:$e,43:$e,45:$e,48:$e,49:$e,50:$e,51:$e,52:$e,53:$e,54:$e,55:$e,56:$e,57:$e,62:bp},qx={91:Gy},Kx={[-2]:yo,[-1]:yo,32:yo},Yx={35:rx,42:qi,45:[Cc,qi],60:ax,61:Cc,95:qi,96:Sc,126:Sc},Xx={38:jp,92:Cp},Gx={[-5]:xo,[-4]:xo,[-3]:xo,33:jx,38:jp,42:_a,60:[Ny,hx],91:Nx,92:[tx,Cp],93:Rs,95:_a,96:By},Jx={null:[_a,Ux]},Zx={null:[42,95]},e1={null:[]},t1=Object.freeze(Object.defineProperty({__proto__:null,attentionMarkers:Zx,contentInitial:qx,disable:e1,document:Qx,flow:Yx,flowInitial:Kx,insideSpan:Jx,string:Xx,text:Gx},Symbol.toStringTag,{value:"Module"}));function n1(e,t,n){let r={_bufferIndex:-1,_index:0,line:n&&n.line||1,column:n&&n.column||1,offset:n&&n.offset||0};const i={},l=[];let o=[],a=[];const s={attempt:j(N),check:j(k),consume:v,enter:y,exit:C,interrupt:j(k,{interrupt:!0})},c={code:null,containerState:{},defineSkip:S,events:[],now:w,parser:e,previous:null,sliceSerialize:m,sliceStream:p,write:f};let d=t.tokenize.call(c,s);return t.resolveAll&&l.push(t),c;function f(z){return o=st(o,z),I(),o[o.length-1]!==null?[]:(_(t,0),c.events=Ds(l,c.events,c),c.events)}function m(z,D){return i1(p(z),D)}function p(z){return r1(o,z)}function w(){const{_bufferIndex:z,_index:D,line:W,column:X,offset:U}=r;return{_bufferIndex:z,_index:D,line:W,column:X,offset:U}}function S(z){i[z.line]=z.column,P()}function I(){let z;for(;r._index<o.length;){const D=o[r._index];if(typeof D=="string")for(z=r._index,r._bufferIndex<0&&(r._bufferIndex=0);r._index===z&&r._bufferIndex<D.length;)h(D.charCodeAt(r._bufferIndex));else h(D)}}function h(z){d=d(z)}function v(z){H(z)?(r.line++,r.column=1,r.offset+=z===-3?2:1,P()):z!==-1&&(r.column++,r.offset++),r._bufferIndex<0?r._index++:(r._bufferIndex++,r._bufferIndex===o[r._index].length&&(r._bufferIndex=-1,r._index++)),c.previous=z}function y(z,D){const W=D||{};return W.type=z,W.start=w(),c.events.push(["enter",W,c]),a.push(W),W}function C(z){const D=a.pop();return D.end=w(),c.events.push(["exit",D,c]),D}function N(z,D){_(z,D.from)}function k(z,D){D.restore()}function j(z,D){return W;function W(X,U,Q){let ie,b,E,g;return Array.isArray(X)?$(X):"tokenize"in X?$([X]):L(X);function L(te){return Me;function Me(lt){const J=lt!==null&&te[lt],Ce=lt!==null&&te.null,Be=[...Array.isArray(J)?J:J?[J]:[],...Array.isArray(Ce)?Ce:Ce?[Ce]:[]];return $(Be)(lt)}}function $(te){return ie=te,b=0,te.length===0?Q:x(te[b])}function x(te){return Me;function Me(lt){return g=R(),E=te,te.partial||(c.currentConstruct=te),te.name&&c.parser.constructs.disable.null.includes(te.name)?be():te.tokenize.call(D?Object.assign(Object.create(c),D):c,s,ne,be)(lt)}}function ne(te){return z(E,g),U}function be(te){return g.restore(),++b<ie.length?x(ie[b]):Q}}}function _(z,D){z.resolveAll&&!l.includes(z)&&l.push(z),z.resolve&&_t(c.events,D,c.events.length-D,z.resolve(c.events.slice(D),c)),z.resolveTo&&(c.events=z.resolveTo(c.events,c))}function R(){const z=w(),D=c.previous,W=c.currentConstruct,X=c.events.length,U=Array.from(a);return{from:X,restore:Q};function Q(){r=z,c.previous=D,c.currentConstruct=W,c.events.length=X,a=U,P()}}function P(){r.line in i&&r.column<2&&(r.column=i[r.line],r.offset+=i[r.line]-1)}}function r1(e,t){const n=t.start._index,r=t.start._bufferIndex,i=t.end._index,l=t.end._bufferIndex;let o;if(n===i)o=[e[n].slice(r,l)];else{if(o=e.slice(n,i),r>-1){const a=o[0];typeof a=="string"?o[0]=a.slice(r):o.shift()}l>0&&o.push(e[i].slice(0,l))}return o}function i1(e,t){let n=-1;const r=[];let i;for(;++n<e.length;){const l=e[n];let o;if(typeof l=="string")o=l;else switch(l){case-5:{o="\r";break}case-4:{o=`
`;break}case-3:{o=`\r
`;break}case-2:{o=t?" ":"	";break}case-1:{if(!t&&i)continue;o=" ";break}default:o=String.fromCharCode(l)}i=l===-2,r.push(o)}return r.join("")}function l1(e){const r={constructs:fy([t1,...(e||{}).extensions||[]]),content:i(ky),defined:[],document:i(Sy),flow:i(Bx),lazy:{},string:i(Hx),text:i(Vx)};return r;function i(l){return o;function o(a){return n1(r,l,a)}}}function o1(e){for(;!Ep(e););return e}const jc=/[\0\t\n\r]/g;function a1(){let e=1,t="",n=!0,r;return i;function i(l,o,a){const s=[];let c,d,f,m,p;for(l=t+(typeof l=="string"?l.toString():new TextDecoder(o||void 0).decode(l)),f=0,t="",n&&(l.charCodeAt(0)===65279&&f++,n=void 0);f<l.length;){if(jc.lastIndex=f,c=jc.exec(l),m=c&&c.index!==void 0?c.index:l.length,p=l.charCodeAt(m),!c){t=l.slice(f);break}if(p===10&&f===m&&r)s.push(-3),r=void 0;else switch(r&&(s.push(-5),r=void 0),f<m&&(s.push(l.slice(f,m)),e+=m-f),p){case 0:{s.push(65533),e++;break}case 9:{for(d=Math.ceil(e/4)*4,s.push(-2);e++<d;)s.push(-1);break}case 10:{s.push(-4),e=1;break}default:r=!0,e=1}f=m+1}return a&&(r&&s.push(-5),t&&s.push(t),s.push(null)),s}}const s1=/\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;function u1(e){return e.replace(s1,c1)}function c1(e,t,n){if(t)return t;if(n.charCodeAt(0)===35){const i=n.charCodeAt(1),l=i===120||i===88;return Sp(n.slice(l?2:1),l?16:10)}return As(n)||e}const Pp={}.hasOwnProperty;function d1(e,t,n){return typeof t!="string"&&(n=t,t=void 0),f1(n)(o1(l1(n).document().write(a1()(e,t,!0))))}function f1(e){const t={transforms:[],canContainEols:["emphasis","fragment","heading","paragraph","strong"],enter:{autolink:l(Qs),autolinkProtocol:R,autolinkEmail:R,atxHeading:l(Hs),blockQuote:l(Ce),characterEscape:R,characterReference:R,codeFenced:l(Be),codeFencedFenceInfo:o,codeFencedFenceMeta:o,codeIndented:l(Be,o),codeText:l(Ht,o),codeTextData:R,data:R,codeFlowValue:R,definition:l(Vt),definitionDestinationString:o,definitionLabelString:o,definitionTitleString:o,emphasis:l(Qp),hardBreakEscape:l(Vs),hardBreakTrailing:l(Vs),htmlFlow:l(Ws,o),htmlFlowData:R,htmlText:l(Ws,o),htmlTextData:R,image:l(qp),label:o,link:l(Qs),listItem:l(Kp),listItemValue:m,listOrdered:l(qs,f),listUnordered:l(qs),paragraph:l(Yp),reference:x,referenceString:o,resourceDestinationString:o,resourceTitleString:o,setextHeading:l(Hs),strong:l(Xp),thematicBreak:l(Jp)},exit:{atxHeading:s(),atxHeadingSequence:N,autolink:s(),autolinkEmail:J,autolinkProtocol:lt,blockQuote:s(),characterEscapeValue:P,characterReferenceMarkerHexadecimal:be,characterReferenceMarkerNumeric:be,characterReferenceValue:te,characterReference:Me,codeFenced:s(I),codeFencedFence:S,codeFencedFenceInfo:p,codeFencedFenceMeta:w,codeFlowValue:P,codeIndented:s(h),codeText:s(U),codeTextData:P,data:P,definition:s(),definitionDestinationString:C,definitionLabelString:v,definitionTitleString:y,emphasis:s(),hardBreakEscape:s(D),hardBreakTrailing:s(D),htmlFlow:s(W),htmlFlowData:P,htmlText:s(X),htmlTextData:P,image:s(ie),label:E,labelText:b,lineEnding:z,link:s(Q),listItem:s(),listOrdered:s(),listUnordered:s(),paragraph:s(),referenceString:ne,resourceDestinationString:g,resourceTitleString:L,resource:$,setextHeading:s(_),setextHeadingLineSequence:j,setextHeadingText:k,strong:s(),thematicBreak:s()}};Ip(t,(e||{}).mdastExtensions||[]);const n={};return r;function r(T){let F={type:"root",children:[]};const V={stack:[F],tokenStack:[],config:t,enter:a,exit:c,buffer:o,resume:d,data:n},G=[];let le=-1;for(;++le<T.length;)if(T[le][1].type==="listOrdered"||T[le][1].type==="listUnordered")if(T[le][0]==="enter")G.push(le);else{const pt=G.pop();le=i(T,pt,le)}for(le=-1;++le<T.length;){const pt=t[T[le][0]];Pp.call(pt,T[le][1].type)&&pt[T[le][1].type].call(Object.assign({sliceSerialize:T[le][2].sliceSerialize},V),T[le][1])}if(V.tokenStack.length>0){const pt=V.tokenStack[V.tokenStack.length-1];(pt[1]||Ec).call(V,void 0,pt[0])}for(F.position={start:Qt(T.length>0?T[0][1].start:{line:1,column:1,offset:0}),end:Qt(T.length>0?T[T.length-2][1].end:{line:1,column:1,offset:0})},le=-1;++le<t.transforms.length;)F=t.transforms[le](F)||F;return F}function i(T,F,V){let G=F-1,le=-1,pt=!1,mn,zt,hr,mr;for(;++G<=V;){const Ye=T[G];switch(Ye[1].type){case"listUnordered":case"listOrdered":case"blockQuote":{Ye[0]==="enter"?le++:le--,mr=void 0;break}case"lineEndingBlank":{Ye[0]==="enter"&&(mn&&!mr&&!le&&!hr&&(hr=G),mr=void 0);break}case"linePrefix":case"listItemValue":case"listItemMarker":case"listItemPrefix":case"listItemPrefixWhitespace":break;default:mr=void 0}if(!le&&Ye[0]==="enter"&&Ye[1].type==="listItemPrefix"||le===-1&&Ye[0]==="exit"&&(Ye[1].type==="listUnordered"||Ye[1].type==="listOrdered")){if(mn){let Pn=G;for(zt=void 0;Pn--;){const Tt=T[Pn];if(Tt[1].type==="lineEnding"||Tt[1].type==="lineEndingBlank"){if(Tt[0]==="exit")continue;zt&&(T[zt][1].type="lineEndingBlank",pt=!0),Tt[1].type="lineEnding",zt=Pn}else if(!(Tt[1].type==="linePrefix"||Tt[1].type==="blockQuotePrefix"||Tt[1].type==="blockQuotePrefixWhitespace"||Tt[1].type==="blockQuoteMarker"||Tt[1].type==="listItemIndent"))break}hr&&(!zt||hr<zt)&&(mn._spread=!0),mn.end=Object.assign({},zt?T[zt][1].start:Ye[1].end),T.splice(zt||G,0,["exit",mn,Ye[2]]),G++,V++}if(Ye[1].type==="listItemPrefix"){const Pn={type:"listItem",_spread:!1,start:Object.assign({},Ye[1].start),end:void 0};mn=Pn,T.splice(G,0,["enter",Pn,Ye[2]]),G++,V++,hr=void 0,mr=!0}}}return T[F][1]._spread=pt,V}function l(T,F){return V;function V(G){a.call(this,T(G),G),F&&F.call(this,G)}}function o(){this.stack.push({type:"fragment",children:[]})}function a(T,F,V){this.stack[this.stack.length-1].children.push(T),this.stack.push(T),this.tokenStack.push([F,V||void 0]),T.position={start:Qt(F.start),end:void 0}}function s(T){return F;function F(V){T&&T.call(this,V),c.call(this,V)}}function c(T,F){const V=this.stack.pop(),G=this.tokenStack.pop();if(G)G[0].type!==T.type&&(F?F.call(this,T,G[0]):(G[1]||Ec).call(this,T,G[0]));else throw new Error("Cannot close `"+T.type+"` ("+Or({start:T.start,end:T.end})+"): it’s not open");V.position.end=Qt(T.end)}function d(){return cy(this.stack.pop())}function f(){this.data.expectingFirstListItemValue=!0}function m(T){if(this.data.expectingFirstListItemValue){const F=this.stack[this.stack.length-2];F.start=Number.parseInt(this.sliceSerialize(T),10),this.data.expectingFirstListItemValue=void 0}}function p(){const T=this.resume(),F=this.stack[this.stack.length-1];F.lang=T}function w(){const T=this.resume(),F=this.stack[this.stack.length-1];F.meta=T}function S(){this.data.flowCodeInside||(this.buffer(),this.data.flowCodeInside=!0)}function I(){const T=this.resume(),F=this.stack[this.stack.length-1];F.value=T.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g,""),this.data.flowCodeInside=void 0}function h(){const T=this.resume(),F=this.stack[this.stack.length-1];F.value=T.replace(/(\r?\n|\r)$/g,"")}function v(T){const F=this.resume(),V=this.stack[this.stack.length-1];V.label=F,V.identifier=tr(this.sliceSerialize(T)).toLowerCase()}function y(){const T=this.resume(),F=this.stack[this.stack.length-1];F.title=T}function C(){const T=this.resume(),F=this.stack[this.stack.length-1];F.url=T}function N(T){const F=this.stack[this.stack.length-1];if(!F.depth){const V=this.sliceSerialize(T).length;F.depth=V}}function k(){this.data.setextHeadingSlurpLineEnding=!0}function j(T){const F=this.stack[this.stack.length-1];F.depth=this.sliceSerialize(T).codePointAt(0)===61?1:2}function _(){this.data.setextHeadingSlurpLineEnding=void 0}function R(T){const V=this.stack[this.stack.length-1].children;let G=V[V.length-1];(!G||G.type!=="text")&&(G=Gp(),G.position={start:Qt(T.start),end:void 0},V.push(G)),this.stack.push(G)}function P(T){const F=this.stack.pop();F.value+=this.sliceSerialize(T),F.position.end=Qt(T.end)}function z(T){const F=this.stack[this.stack.length-1];if(this.data.atHardBreak){const V=F.children[F.children.length-1];V.position.end=Qt(T.end),this.data.atHardBreak=void 0;return}!this.data.setextHeadingSlurpLineEnding&&t.canContainEols.includes(F.type)&&(R.call(this,T),P.call(this,T))}function D(){this.data.atHardBreak=!0}function W(){const T=this.resume(),F=this.stack[this.stack.length-1];F.value=T}function X(){const T=this.resume(),F=this.stack[this.stack.length-1];F.value=T}function U(){const T=this.resume(),F=this.stack[this.stack.length-1];F.value=T}function Q(){const T=this.stack[this.stack.length-1];if(this.data.inReference){const F=this.data.referenceType||"shortcut";T.type+="Reference",T.referenceType=F,delete T.url,delete T.title}else delete T.identifier,delete T.label;this.data.referenceType=void 0}function ie(){const T=this.stack[this.stack.length-1];if(this.data.inReference){const F=this.data.referenceType||"shortcut";T.type+="Reference",T.referenceType=F,delete T.url,delete T.title}else delete T.identifier,delete T.label;this.data.referenceType=void 0}function b(T){const F=this.sliceSerialize(T),V=this.stack[this.stack.length-2];V.label=u1(F),V.identifier=tr(F).toLowerCase()}function E(){const T=this.stack[this.stack.length-1],F=this.resume(),V=this.stack[this.stack.length-1];if(this.data.inReference=!0,V.type==="link"){const G=T.children;V.children=G}else V.alt=F}function g(){const T=this.resume(),F=this.stack[this.stack.length-1];F.url=T}function L(){const T=this.resume(),F=this.stack[this.stack.length-1];F.title=T}function $(){this.data.inReference=void 0}function x(){this.data.referenceType="collapsed"}function ne(T){const F=this.resume(),V=this.stack[this.stack.length-1];V.label=F,V.identifier=tr(this.sliceSerialize(T)).toLowerCase(),this.data.referenceType="full"}function be(T){this.data.characterReferenceType=T.type}function te(T){const F=this.sliceSerialize(T),V=this.data.characterReferenceType;let G;V?(G=Sp(F,V==="characterReferenceMarkerNumeric"?10:16),this.data.characterReferenceType=void 0):G=As(F);const le=this.stack[this.stack.length-1];le.value+=G}function Me(T){const F=this.stack.pop();F.position.end=Qt(T.end)}function lt(T){P.call(this,T);const F=this.stack[this.stack.length-1];F.url=this.sliceSerialize(T)}function J(T){P.call(this,T);const F=this.stack[this.stack.length-1];F.url="mailto:"+this.sliceSerialize(T)}function Ce(){return{type:"blockquote",children:[]}}function Be(){return{type:"code",lang:null,meta:null,value:""}}function Ht(){return{type:"inlineCode",value:""}}function Vt(){return{type:"definition",identifier:"",label:null,title:null,url:""}}function Qp(){return{type:"emphasis",children:[]}}function Hs(){return{type:"heading",depth:0,children:[]}}function Vs(){return{type:"break"}}function Ws(){return{type:"html",value:""}}function qp(){return{type:"image",title:null,url:"",alt:null}}function Qs(){return{type:"link",title:null,url:"",children:[]}}function qs(T){return{type:"list",ordered:T.type==="listOrdered",start:null,spread:T._spread,children:[]}}function Kp(T){return{type:"listItem",spread:T._spread,checked:null,children:[]}}function Yp(){return{type:"paragraph",children:[]}}function Xp(){return{type:"strong",children:[]}}function Gp(){return{type:"text",value:""}}function Jp(){return{type:"thematicBreak"}}}function Qt(e){return{line:e.line,column:e.column,offset:e.offset}}function Ip(e,t){let n=-1;for(;++n<t.length;){const r=t[n];Array.isArray(r)?Ip(e,r):p1(e,r)}}function p1(e,t){let n;for(n in t)if(Pp.call(t,n))switch(n){case"canContainEols":{const r=t[n];r&&e[n].push(...r);break}case"transforms":{const r=t[n];r&&e[n].push(...r);break}case"enter":case"exit":{const r=t[n];r&&Object.assign(e[n],r);break}}}function Ec(e,t){throw e?new Error("Cannot close `"+e.type+"` ("+Or({start:e.start,end:e.end})+"): a different token (`"+t.type+"`, "+Or({start:t.start,end:t.end})+") is open"):new Error("Cannot close document, a token (`"+t.type+"`, "+Or({start:t.start,end:t.end})+") is still open")}function h1(e){const t=this;t.parser=n;function n(r){return d1(r,{...t.data("settings"),...e,extensions:t.data("micromarkExtensions")||[],mdastExtensions:t.data("fromMarkdownExtensions")||[]})}}function m1(e,t){const n={type:"element",tagName:"blockquote",properties:{},children:e.wrap(e.all(t),!0)};return e.patch(t,n),e.applyData(t,n)}function g1(e,t){const n={type:"element",tagName:"br",properties:{},children:[]};return e.patch(t,n),[e.applyData(t,n),{type:"text",value:`
`}]}function v1(e,t){const n=t.value?t.value+`
`:"",r={},i=t.lang?t.lang.split(/\s+/):[];i.length>0&&(r.className=["language-"+i[0]]);let l={type:"element",tagName:"code",properties:r,children:[{type:"text",value:n}]};return t.meta&&(l.data={meta:t.meta}),e.patch(t,l),l=e.applyData(t,l),l={type:"element",tagName:"pre",properties:{},children:[l]},e.patch(t,l),l}function y1(e,t){const n={type:"element",tagName:"del",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function x1(e,t){const n={type:"element",tagName:"em",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function k1(e,t){const n=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",r=String(t.identifier).toUpperCase(),i=pr(r.toLowerCase()),l=e.footnoteOrder.indexOf(r);let o,a=e.footnoteCounts.get(r);a===void 0?(a=0,e.footnoteOrder.push(r),o=e.footnoteOrder.length):o=l+1,a+=1,e.footnoteCounts.set(r,a);const s={type:"element",tagName:"a",properties:{href:"#"+n+"fn-"+i,id:n+"fnref-"+i+(a>1?"-"+a:""),dataFootnoteRef:!0,ariaDescribedBy:["footnote-label"]},children:[{type:"text",value:String(o)}]};e.patch(t,s);const c={type:"element",tagName:"sup",properties:{},children:[s]};return e.patch(t,c),e.applyData(t,c)}function w1(e,t){const n={type:"element",tagName:"h"+t.depth,properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function S1(e,t){if(e.options.allowDangerousHtml){const n={type:"raw",value:t.value};return e.patch(t,n),e.applyData(t,n)}}function Mp(e,t){const n=t.referenceType;let r="]";if(n==="collapsed"?r+="[]":n==="full"&&(r+="["+(t.label||t.identifier)+"]"),t.type==="imageReference")return[{type:"text",value:"!["+t.alt+r}];const i=e.all(t),l=i[0];l&&l.type==="text"?l.value="["+l.value:i.unshift({type:"text",value:"["});const o=i[i.length-1];return o&&o.type==="text"?o.value+=r:i.push({type:"text",value:r}),i}function b1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Mp(e,t);const i={src:pr(r.url||""),alt:t.alt};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"img",properties:i,children:[]};return e.patch(t,l),e.applyData(t,l)}function C1(e,t){const n={src:pr(t.url)};t.alt!==null&&t.alt!==void 0&&(n.alt=t.alt),t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"img",properties:n,children:[]};return e.patch(t,r),e.applyData(t,r)}function j1(e,t){const n={type:"text",value:t.value.replace(/\r?\n|\r/g," ")};e.patch(t,n);const r={type:"element",tagName:"code",properties:{},children:[n]};return e.patch(t,r),e.applyData(t,r)}function E1(e,t){const n=String(t.identifier).toUpperCase(),r=e.definitionById.get(n);if(!r)return Mp(e,t);const i={href:pr(r.url||"")};r.title!==null&&r.title!==void 0&&(i.title=r.title);const l={type:"element",tagName:"a",properties:i,children:e.all(t)};return e.patch(t,l),e.applyData(t,l)}function N1(e,t){const n={href:pr(t.url)};t.title!==null&&t.title!==void 0&&(n.title=t.title);const r={type:"element",tagName:"a",properties:n,children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function _1(e,t,n){const r=e.all(t),i=n?z1(n):Ap(t),l={},o=[];if(typeof t.checked=="boolean"){const d=r[0];let f;d&&d.type==="element"&&d.tagName==="p"?f=d:(f={type:"element",tagName:"p",properties:{},children:[]},r.unshift(f)),f.children.length>0&&f.children.unshift({type:"text",value:" "}),f.children.unshift({type:"element",tagName:"input",properties:{type:"checkbox",checked:t.checked,disabled:!0},children:[]}),l.className=["task-list-item"]}let a=-1;for(;++a<r.length;){const d=r[a];(i||a!==0||d.type!=="element"||d.tagName!=="p")&&o.push({type:"text",value:`
`}),d.type==="element"&&d.tagName==="p"&&!i?o.push(...d.children):o.push(d)}const s=r[r.length-1];s&&(i||s.type!=="element"||s.tagName!=="p")&&o.push({type:"text",value:`
`});const c={type:"element",tagName:"li",properties:l,children:o};return e.patch(t,c),e.applyData(t,c)}function z1(e){let t=!1;if(e.type==="list"){t=e.spread||!1;const n=e.children;let r=-1;for(;!t&&++r<n.length;)t=Ap(n[r])}return t}function Ap(e){const t=e.spread;return t??e.children.length>1}function T1(e,t){const n={},r=e.all(t);let i=-1;for(typeof t.start=="number"&&t.start!==1&&(n.start=t.start);++i<r.length;){const o=r[i];if(o.type==="element"&&o.tagName==="li"&&o.properties&&Array.isArray(o.properties.className)&&o.properties.className.includes("task-list-item")){n.className=["contains-task-list"];break}}const l={type:"element",tagName:t.ordered?"ol":"ul",properties:n,children:e.wrap(r,!0)};return e.patch(t,l),e.applyData(t,l)}function L1(e,t){const n={type:"element",tagName:"p",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function P1(e,t){const n={type:"root",children:e.wrap(e.all(t))};return e.patch(t,n),e.applyData(t,n)}function I1(e,t){const n={type:"element",tagName:"strong",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}function M1(e,t){const n=e.all(t),r=n.shift(),i=[];if(r){const o={type:"element",tagName:"thead",properties:{},children:e.wrap([r],!0)};e.patch(t.children[0],o),i.push(o)}if(n.length>0){const o={type:"element",tagName:"tbody",properties:{},children:e.wrap(n,!0)},a=Ls(t.children[1]),s=mp(t.children[t.children.length-1]);a&&s&&(o.position={start:a,end:s}),i.push(o)}const l={type:"element",tagName:"table",properties:{},children:e.wrap(i,!0)};return e.patch(t,l),e.applyData(t,l)}function A1(e,t,n){const r=n?n.children:void 0,l=(r?r.indexOf(t):1)===0?"th":"td",o=n&&n.type==="table"?n.align:void 0,a=o?o.length:t.children.length;let s=-1;const c=[];for(;++s<a;){const f=t.children[s],m={},p=o?o[s]:void 0;p&&(m.align=p);let w={type:"element",tagName:l,properties:m,children:[]};f&&(w.children=e.all(f),e.patch(f,w),w=e.applyData(f,w)),c.push(w)}const d={type:"element",tagName:"tr",properties:{},children:e.wrap(c,!0)};return e.patch(t,d),e.applyData(t,d)}function D1(e,t){const n={type:"element",tagName:"td",properties:{},children:e.all(t)};return e.patch(t,n),e.applyData(t,n)}const Nc=9,_c=32;function R1(e){const t=String(e),n=/\r?\n|\r/g;let r=n.exec(t),i=0;const l=[];for(;r;)l.push(zc(t.slice(i,r.index),i>0,!0),r[0]),i=r.index+r[0].length,r=n.exec(t);return l.push(zc(t.slice(i),i>0,!1)),l.join("")}function zc(e,t,n){let r=0,i=e.length;if(t){let l=e.codePointAt(r);for(;l===Nc||l===_c;)r++,l=e.codePointAt(r)}if(n){let l=e.codePointAt(i-1);for(;l===Nc||l===_c;)i--,l=e.codePointAt(i-1)}return i>r?e.slice(r,i):""}function F1(e,t){const n={type:"text",value:R1(String(t.value))};return e.patch(t,n),e.applyData(t,n)}function O1(e,t){const n={type:"element",tagName:"hr",properties:{},children:[]};return e.patch(t,n),e.applyData(t,n)}const B1={blockquote:m1,break:g1,code:v1,delete:y1,emphasis:x1,footnoteReference:k1,heading:w1,html:S1,imageReference:b1,image:C1,inlineCode:j1,linkReference:E1,link:N1,listItem:_1,list:T1,paragraph:L1,root:P1,strong:I1,table:M1,tableCell:D1,tableRow:A1,text:F1,thematicBreak:O1,toml:Li,yaml:Li,definition:Li,footnoteDefinition:Li};function Li(){}const Dp=-1,Bl=0,$r=1,kl=2,Fs=3,Os=4,Bs=5,$s=6,Rp=7,Fp=8,Tc=typeof self=="object"?self:globalThis,$1=(e,t)=>{const n=(i,l)=>(e.set(l,i),i),r=i=>{if(e.has(i))return e.get(i);const[l,o]=t[i];switch(l){case Bl:case Dp:return n(o,i);case $r:{const a=n([],i);for(const s of o)a.push(r(s));return a}case kl:{const a=n({},i);for(const[s,c]of o)a[r(s)]=r(c);return a}case Fs:return n(new Date(o),i);case Os:{const{source:a,flags:s}=o;return n(new RegExp(a,s),i)}case Bs:{const a=n(new Map,i);for(const[s,c]of o)a.set(r(s),r(c));return a}case $s:{const a=n(new Set,i);for(const s of o)a.add(r(s));return a}case Rp:{const{name:a,message:s}=o;return n(new Tc[a](s),i)}case Fp:return n(BigInt(o),i);case"BigInt":return n(Object(BigInt(o)),i);case"ArrayBuffer":return n(new Uint8Array(o).buffer,o);case"DataView":{const{buffer:a}=new Uint8Array(o);return n(new DataView(a),o)}}return n(new Tc[l](o),i)};return r},Lc=e=>$1(new Map,e)(0),An="",{toString:U1}={},{keys:H1}=Object,jr=e=>{const t=typeof e;if(t!=="object"||!e)return[Bl,t];const n=U1.call(e).slice(8,-1);switch(n){case"Array":return[$r,An];case"Object":return[kl,An];case"Date":return[Fs,An];case"RegExp":return[Os,An];case"Map":return[Bs,An];case"Set":return[$s,An];case"DataView":return[$r,n]}return n.includes("Array")?[$r,n]:n.includes("Error")?[Rp,n]:[kl,n]},Pi=([e,t])=>e===Bl&&(t==="function"||t==="symbol"),V1=(e,t,n,r)=>{const i=(o,a)=>{const s=r.push(o)-1;return n.set(a,s),s},l=o=>{if(n.has(o))return n.get(o);let[a,s]=jr(o);switch(a){case Bl:{let d=o;switch(s){case"bigint":a=Fp,d=o.toString();break;case"function":case"symbol":if(e)throw new TypeError("unable to serialize "+s);d=null;break;case"undefined":return i([Dp],o)}return i([a,d],o)}case $r:{if(s){let m=o;return s==="DataView"?m=new Uint8Array(o.buffer):s==="ArrayBuffer"&&(m=new Uint8Array(o)),i([s,[...m]],o)}const d=[],f=i([a,d],o);for(const m of o)d.push(l(m));return f}case kl:{if(s)switch(s){case"BigInt":return i([s,o.toString()],o);case"Boolean":case"Number":case"String":return i([s,o.valueOf()],o)}if(t&&"toJSON"in o)return l(o.toJSON());const d=[],f=i([a,d],o);for(const m of H1(o))(e||!Pi(jr(o[m])))&&d.push([l(m),l(o[m])]);return f}case Fs:return i([a,o.toISOString()],o);case Os:{const{source:d,flags:f}=o;return i([a,{source:d,flags:f}],o)}case Bs:{const d=[],f=i([a,d],o);for(const[m,p]of o)(e||!(Pi(jr(m))||Pi(jr(p))))&&d.push([l(m),l(p)]);return f}case $s:{const d=[],f=i([a,d],o);for(const m of o)(e||!Pi(jr(m)))&&d.push(l(m));return f}}const{message:c}=o;return i([a,{name:s,message:c}],o)};return l},Pc=(e,{json:t,lossy:n}={})=>{const r=[];return V1(!(t||n),!!t,new Map,r)(e),r},wl=typeof structuredClone=="function"?(e,t)=>t&&("json"in t||"lossy"in t)?Lc(Pc(e,t)):structuredClone(e):(e,t)=>Lc(Pc(e,t));function W1(e,t){const n=[{type:"text",value:"↩"}];return t>1&&n.push({type:"element",tagName:"sup",properties:{},children:[{type:"text",value:String(t)}]}),n}function Q1(e,t){return"Back to reference "+(e+1)+(t>1?"-"+t:"")}function q1(e){const t=typeof e.options.clobberPrefix=="string"?e.options.clobberPrefix:"user-content-",n=e.options.footnoteBackContent||W1,r=e.options.footnoteBackLabel||Q1,i=e.options.footnoteLabel||"Footnotes",l=e.options.footnoteLabelTagName||"h2",o=e.options.footnoteLabelProperties||{className:["sr-only"]},a=[];let s=-1;for(;++s<e.footnoteOrder.length;){const c=e.footnoteById.get(e.footnoteOrder[s]);if(!c)continue;const d=e.all(c),f=String(c.identifier).toUpperCase(),m=pr(f.toLowerCase());let p=0;const w=[],S=e.footnoteCounts.get(f);for(;S!==void 0&&++p<=S;){w.length>0&&w.push({type:"text",value:" "});let v=typeof n=="string"?n:n(s,p);typeof v=="string"&&(v={type:"text",value:v}),w.push({type:"element",tagName:"a",properties:{href:"#"+t+"fnref-"+m+(p>1?"-"+p:""),dataFootnoteBackref:"",ariaLabel:typeof r=="string"?r:r(s,p),className:["data-footnote-backref"]},children:Array.isArray(v)?v:[v]})}const I=d[d.length-1];if(I&&I.type==="element"&&I.tagName==="p"){const v=I.children[I.children.length-1];v&&v.type==="text"?v.value+=" ":I.children.push({type:"text",value:" "}),I.children.push(...w)}else d.push(...w);const h={type:"element",tagName:"li",properties:{id:t+"fn-"+m},children:e.wrap(d,!0)};e.patch(c,h),a.push(h)}if(a.length!==0)return{type:"element",tagName:"section",properties:{dataFootnotes:!0,className:["footnotes"]},children:[{type:"element",tagName:l,properties:{...wl(o),id:"footnote-label"},children:[{type:"text",value:i}]},{type:"text",value:`
`},{type:"element",tagName:"ol",properties:{},children:e.wrap(a,!0)},{type:"text",value:`
`}]}}const Op=function(e){if(e==null)return G1;if(typeof e=="function")return $l(e);if(typeof e=="object")return Array.isArray(e)?K1(e):Y1(e);if(typeof e=="string")return X1(e);throw new Error("Expected function, string, or object as test")};function K1(e){const t=[];let n=-1;for(;++n<e.length;)t[n]=Op(e[n]);return $l(r);function r(...i){let l=-1;for(;++l<t.length;)if(t[l].apply(this,i))return!0;return!1}}function Y1(e){const t=e;return $l(n);function n(r){const i=r;let l;for(l in e)if(i[l]!==t[l])return!1;return!0}}function X1(e){return $l(t);function t(n){return n&&n.type===e}}function $l(e){return t;function t(n,r,i){return!!(J1(n)&&e.call(this,n,typeof r=="number"?r:void 0,i||void 0))}}function G1(){return!0}function J1(e){return e!==null&&typeof e=="object"&&"type"in e}const Bp=[],Z1=!0,Ic=!1,e0="skip";function t0(e,t,n,r){let i;typeof t=="function"&&typeof n!="function"?(r=n,n=t):i=t;const l=Op(i),o=r?-1:1;a(e,void 0,[])();function a(s,c,d){const f=s&&typeof s=="object"?s:{};if(typeof f.type=="string"){const p=typeof f.tagName=="string"?f.tagName:typeof f.name=="string"?f.name:void 0;Object.defineProperty(m,"name",{value:"node ("+(s.type+(p?"<"+p+">":""))+")"})}return m;function m(){let p=Bp,w,S,I;if((!t||l(s,c,d[d.length-1]||void 0))&&(p=n0(n(s,d)),p[0]===Ic))return p;if("children"in s&&s.children){const h=s;if(h.children&&p[0]!==e0)for(S=(r?h.children.length:-1)+o,I=d.concat(h);S>-1&&S<h.children.length;){const v=h.children[S];if(w=a(v,S,I)(),w[0]===Ic)return w;S=typeof w[1]=="number"?w[1]:S+o}}return p}}}function n0(e){return Array.isArray(e)?e:typeof e=="number"?[Z1,e]:e==null?Bp:[e]}function $p(e,t,n,r){let i,l,o;typeof t=="function"&&typeof n!="function"?(l=void 0,o=t,i=n):(l=t,o=n,i=r),t0(e,l,a,i);function a(s,c){const d=c[c.length-1],f=d?d.children.indexOf(s):void 0;return o(s,f,d)}}const za={}.hasOwnProperty,r0={};function i0(e,t){const n=t||r0,r=new Map,i=new Map,l=new Map,o={...B1,...n.handlers},a={all:c,applyData:o0,definitionById:r,footnoteById:i,footnoteCounts:l,footnoteOrder:[],handlers:o,one:s,options:n,patch:l0,wrap:s0};return $p(e,function(d){if(d.type==="definition"||d.type==="footnoteDefinition"){const f=d.type==="definition"?r:i,m=String(d.identifier).toUpperCase();f.has(m)||f.set(m,d)}}),a;function s(d,f){const m=d.type,p=a.handlers[m];if(za.call(a.handlers,m)&&p)return p(a,d,f);if(a.options.passThrough&&a.options.passThrough.includes(m)){if("children"in d){const{children:S,...I}=d,h=wl(I);return h.children=a.all(d),h}return wl(d)}return(a.options.unknownHandler||a0)(a,d,f)}function c(d){const f=[];if("children"in d){const m=d.children;let p=-1;for(;++p<m.length;){const w=a.one(m[p],d);if(w){if(p&&m[p-1].type==="break"&&(!Array.isArray(w)&&w.type==="text"&&(w.value=Mc(w.value)),!Array.isArray(w)&&w.type==="element")){const S=w.children[0];S&&S.type==="text"&&(S.value=Mc(S.value))}Array.isArray(w)?f.push(...w):f.push(w)}}}return f}}function l0(e,t){e.position&&(t.position=Uv(e))}function o0(e,t){let n=t;if(e&&e.data){const r=e.data.hName,i=e.data.hChildren,l=e.data.hProperties;if(typeof r=="string")if(n.type==="element")n.tagName=r;else{const o="children"in n?n.children:[n];n={type:"element",tagName:r,properties:{},children:o}}n.type==="element"&&l&&Object.assign(n.properties,wl(l)),"children"in n&&n.children&&i!==null&&i!==void 0&&(n.children=i)}return n}function a0(e,t){const n=t.data||{},r="value"in t&&!(za.call(n,"hProperties")||za.call(n,"hChildren"))?{type:"text",value:t.value}:{type:"element",tagName:"div",properties:{},children:e.all(t)};return e.patch(t,r),e.applyData(t,r)}function s0(e,t){const n=[];let r=-1;for(t&&n.push({type:"text",value:`
`});++r<e.length;)r&&n.push({type:"text",value:`
`}),n.push(e[r]);return t&&e.length>0&&n.push({type:"text",value:`
`}),n}function Mc(e){let t=0,n=e.charCodeAt(t);for(;n===9||n===32;)t++,n=e.charCodeAt(t);return e.slice(t)}function Ac(e,t){const n=i0(e,t),r=n.one(e,void 0),i=q1(n),l=Array.isArray(r)?{type:"root",children:r}:r||{type:"root",children:[]};return i&&l.children.push({type:"text",value:`
`},i),l}function u0(e,t){return e&&"run"in e?async function(n,r){const i=Ac(n,{file:r,...t});await e.run(i,r)}:function(n,r){return Ac(n,{file:r,...e||t})}}function Dc(e){if(e)throw e}var Ki=Object.prototype.hasOwnProperty,Up=Object.prototype.toString,Rc=Object.defineProperty,Fc=Object.getOwnPropertyDescriptor,Oc=function(t){return typeof Array.isArray=="function"?Array.isArray(t):Up.call(t)==="[object Array]"},Bc=function(t){if(!t||Up.call(t)!=="[object Object]")return!1;var n=Ki.call(t,"constructor"),r=t.constructor&&t.constructor.prototype&&Ki.call(t.constructor.prototype,"isPrototypeOf");if(t.constructor&&!n&&!r)return!1;var i;for(i in t);return typeof i>"u"||Ki.call(t,i)},$c=function(t,n){Rc&&n.name==="__proto__"?Rc(t,n.name,{enumerable:!0,configurable:!0,value:n.newValue,writable:!0}):t[n.name]=n.newValue},Uc=function(t,n){if(n==="__proto__")if(Ki.call(t,n)){if(Fc)return Fc(t,n).value}else return;return t[n]},c0=function e(){var t,n,r,i,l,o,a=arguments[0],s=1,c=arguments.length,d=!1;for(typeof a=="boolean"&&(d=a,a=arguments[1]||{},s=2),(a==null||typeof a!="object"&&typeof a!="function")&&(a={});s<c;++s)if(t=arguments[s],t!=null)for(n in t)r=Uc(a,n),i=Uc(t,n),a!==i&&(d&&i&&(Bc(i)||(l=Oc(i)))?(l?(l=!1,o=r&&Oc(r)?r:[]):o=r&&Bc(r)?r:{},$c(a,{name:n,newValue:e(d,o,i)})):typeof i<"u"&&$c(a,{name:n,newValue:i}));return a};const ko=Ia(c0);function Ta(e){if(typeof e!="object"||e===null)return!1;const t=Object.getPrototypeOf(e);return(t===null||t===Object.prototype||Object.getPrototypeOf(t)===null)&&!(Symbol.toStringTag in e)&&!(Symbol.iterator in e)}function d0(){const e=[],t={run:n,use:r};return t;function n(...i){let l=-1;const o=i.pop();if(typeof o!="function")throw new TypeError("Expected function as last argument, not "+o);a(null,...i);function a(s,...c){const d=e[++l];let f=-1;if(s){o(s);return}for(;++f<i.length;)(c[f]===null||c[f]===void 0)&&(c[f]=i[f]);i=c,d?f0(d,a)(...c):o(null,...c)}}function r(i){if(typeof i!="function")throw new TypeError("Expected `middelware` to be a function, not "+i);return e.push(i),t}}function f0(e,t){let n;return r;function r(...o){const a=e.length>o.length;let s;a&&o.push(i);try{s=e.apply(this,o)}catch(c){const d=c;if(a&&n)throw d;return i(d)}a||(s&&s.then&&typeof s.then=="function"?s.then(l,i):s instanceof Error?i(s):l(s))}function i(o,...a){n||(n=!0,t(o,...a))}function l(o){i(null,o)}}const bt={basename:p0,dirname:h0,extname:m0,join:g0,sep:"/"};function p0(e,t){if(t!==void 0&&typeof t!="string")throw new TypeError('"ext" argument must be a string');fi(e);let n=0,r=-1,i=e.length,l;if(t===void 0||t.length===0||t.length>e.length){for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else r<0&&(l=!0,r=i+1);return r<0?"":e.slice(n,r)}if(t===e)return"";let o=-1,a=t.length-1;for(;i--;)if(e.codePointAt(i)===47){if(l){n=i+1;break}}else o<0&&(l=!0,o=i+1),a>-1&&(e.codePointAt(i)===t.codePointAt(a--)?a<0&&(r=i):(a=-1,r=o));return n===r?r=o:r<0&&(r=e.length),e.slice(n,r)}function h0(e){if(fi(e),e.length===0)return".";let t=-1,n=e.length,r;for(;--n;)if(e.codePointAt(n)===47){if(r){t=n;break}}else r||(r=!0);return t<0?e.codePointAt(0)===47?"/":".":t===1&&e.codePointAt(0)===47?"//":e.slice(0,t)}function m0(e){fi(e);let t=e.length,n=-1,r=0,i=-1,l=0,o;for(;t--;){const a=e.codePointAt(t);if(a===47){if(o){r=t+1;break}continue}n<0&&(o=!0,n=t+1),a===46?i<0?i=t:l!==1&&(l=1):i>-1&&(l=-1)}return i<0||n<0||l===0||l===1&&i===n-1&&i===r+1?"":e.slice(i,n)}function g0(...e){let t=-1,n;for(;++t<e.length;)fi(e[t]),e[t]&&(n=n===void 0?e[t]:n+"/"+e[t]);return n===void 0?".":v0(n)}function v0(e){fi(e);const t=e.codePointAt(0)===47;let n=y0(e,!t);return n.length===0&&!t&&(n="."),n.length>0&&e.codePointAt(e.length-1)===47&&(n+="/"),t?"/"+n:n}function y0(e,t){let n="",r=0,i=-1,l=0,o=-1,a,s;for(;++o<=e.length;){if(o<e.length)a=e.codePointAt(o);else{if(a===47)break;a=47}if(a===47){if(!(i===o-1||l===1))if(i!==o-1&&l===2){if(n.length<2||r!==2||n.codePointAt(n.length-1)!==46||n.codePointAt(n.length-2)!==46){if(n.length>2){if(s=n.lastIndexOf("/"),s!==n.length-1){s<0?(n="",r=0):(n=n.slice(0,s),r=n.length-1-n.lastIndexOf("/")),i=o,l=0;continue}}else if(n.length>0){n="",r=0,i=o,l=0;continue}}t&&(n=n.length>0?n+"/..":"..",r=2)}else n.length>0?n+="/"+e.slice(i+1,o):n=e.slice(i+1,o),r=o-i-1;i=o,l=0}else a===46&&l>-1?l++:l=-1}return n}function fi(e){if(typeof e!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(e))}const x0={cwd:k0};function k0(){return"/"}function La(e){return!!(e!==null&&typeof e=="object"&&"href"in e&&e.href&&"protocol"in e&&e.protocol&&e.auth===void 0)}function w0(e){if(typeof e=="string")e=new URL(e);else if(!La(e)){const t=new TypeError('The "path" argument must be of type string or an instance of URL. Received `'+e+"`");throw t.code="ERR_INVALID_ARG_TYPE",t}if(e.protocol!=="file:"){const t=new TypeError("The URL must be of scheme file");throw t.code="ERR_INVALID_URL_SCHEME",t}return S0(e)}function S0(e){if(e.hostname!==""){const r=new TypeError('File URL host must be "localhost" or empty on darwin');throw r.code="ERR_INVALID_FILE_URL_HOST",r}const t=e.pathname;let n=-1;for(;++n<t.length;)if(t.codePointAt(n)===37&&t.codePointAt(n+1)===50){const r=t.codePointAt(n+2);if(r===70||r===102){const i=new TypeError("File URL path must not include encoded / characters");throw i.code="ERR_INVALID_FILE_URL_PATH",i}}return decodeURIComponent(t)}const wo=["history","path","basename","stem","extname","dirname"];class Hp{constructor(t){let n;t?La(t)?n={path:t}:typeof t=="string"||b0(t)?n={value:t}:n=t:n={},this.cwd="cwd"in n?"":x0.cwd(),this.data={},this.history=[],this.messages=[],this.value,this.map,this.result,this.stored;let r=-1;for(;++r<wo.length;){const l=wo[r];l in n&&n[l]!==void 0&&n[l]!==null&&(this[l]=l==="history"?[...n[l]]:n[l])}let i;for(i in n)wo.includes(i)||(this[i]=n[i])}get basename(){return typeof this.path=="string"?bt.basename(this.path):void 0}set basename(t){bo(t,"basename"),So(t,"basename"),this.path=bt.join(this.dirname||"",t)}get dirname(){return typeof this.path=="string"?bt.dirname(this.path):void 0}set dirname(t){Hc(this.basename,"dirname"),this.path=bt.join(t||"",this.basename)}get extname(){return typeof this.path=="string"?bt.extname(this.path):void 0}set extname(t){if(So(t,"extname"),Hc(this.dirname,"extname"),t){if(t.codePointAt(0)!==46)throw new Error("`extname` must start with `.`");if(t.includes(".",1))throw new Error("`extname` cannot contain multiple dots")}this.path=bt.join(this.dirname,this.stem+(t||""))}get path(){return this.history[this.history.length-1]}set path(t){La(t)&&(t=w0(t)),bo(t,"path"),this.path!==t&&this.history.push(t)}get stem(){return typeof this.path=="string"?bt.basename(this.path,this.extname):void 0}set stem(t){bo(t,"stem"),So(t,"stem"),this.path=bt.join(this.dirname||"",t+(this.extname||""))}fail(t,n,r){const i=this.message(t,n,r);throw i.fatal=!0,i}info(t,n,r){const i=this.message(t,n,r);return i.fatal=void 0,i}message(t,n,r){const i=new Ie(t,n,r);return this.path&&(i.name=this.path+":"+i.name,i.file=this.path),i.fatal=!1,this.messages.push(i),i}toString(t){return this.value===void 0?"":typeof this.value=="string"?this.value:new TextDecoder(t||void 0).decode(this.value)}}function So(e,t){if(e&&e.includes(bt.sep))throw new Error("`"+t+"` cannot be a path: did not expect `"+bt.sep+"`")}function bo(e,t){if(!e)throw new Error("`"+t+"` cannot be empty")}function Hc(e,t){if(!e)throw new Error("Setting `"+t+"` requires `path` to be set too")}function b0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const C0=function(e){const r=this.constructor.prototype,i=r[e],l=function(){return i.apply(l,arguments)};return Object.setPrototypeOf(l,r),l},j0={}.hasOwnProperty;class Us extends C0{constructor(){super("copy"),this.Compiler=void 0,this.Parser=void 0,this.attachers=[],this.compiler=void 0,this.freezeIndex=-1,this.frozen=void 0,this.namespace={},this.parser=void 0,this.transformers=d0()}copy(){const t=new Us;let n=-1;for(;++n<this.attachers.length;){const r=this.attachers[n];t.use(...r)}return t.data(ko(!0,{},this.namespace)),t}data(t,n){return typeof t=="string"?arguments.length===2?(Eo("data",this.frozen),this.namespace[t]=n,this):j0.call(this.namespace,t)&&this.namespace[t]||void 0:t?(Eo("data",this.frozen),this.namespace=t,this):this.namespace}freeze(){if(this.frozen)return this;const t=this;for(;++this.freezeIndex<this.attachers.length;){const[n,...r]=this.attachers[this.freezeIndex];if(r[0]===!1)continue;r[0]===!0&&(r[0]=void 0);const i=n.call(t,...r);typeof i=="function"&&this.transformers.use(i)}return this.frozen=!0,this.freezeIndex=Number.POSITIVE_INFINITY,this}parse(t){this.freeze();const n=Ii(t),r=this.parser||this.Parser;return Co("parse",r),r(String(n),n)}process(t,n){const r=this;return this.freeze(),Co("process",this.parser||this.Parser),jo("process",this.compiler||this.Compiler),n?i(void 0,n):new Promise(i);function i(l,o){const a=Ii(t),s=r.parse(a);r.run(s,a,function(d,f,m){if(d||!f||!m)return c(d);const p=f,w=r.stringify(p,m);_0(w)?m.value=w:m.result=w,c(d,m)});function c(d,f){d||!f?o(d):l?l(f):n(void 0,f)}}}processSync(t){let n=!1,r;return this.freeze(),Co("processSync",this.parser||this.Parser),jo("processSync",this.compiler||this.Compiler),this.process(t,i),Wc("processSync","process",n),r;function i(l,o){n=!0,Dc(l),r=o}}run(t,n,r){Vc(t),this.freeze();const i=this.transformers;return!r&&typeof n=="function"&&(r=n,n=void 0),r?l(void 0,r):new Promise(l);function l(o,a){const s=Ii(n);i.run(t,s,c);function c(d,f,m){const p=f||t;d?a(d):o?o(p):r(void 0,p,m)}}}runSync(t,n){let r=!1,i;return this.run(t,n,l),Wc("runSync","run",r),i;function l(o,a){Dc(o),i=a,r=!0}}stringify(t,n){this.freeze();const r=Ii(n),i=this.compiler||this.Compiler;return jo("stringify",i),Vc(t),i(t,r)}use(t,...n){const r=this.attachers,i=this.namespace;if(Eo("use",this.frozen),t!=null)if(typeof t=="function")s(t,n);else if(typeof t=="object")Array.isArray(t)?a(t):o(t);else throw new TypeError("Expected usable value, not `"+t+"`");return this;function l(c){if(typeof c=="function")s(c,[]);else if(typeof c=="object")if(Array.isArray(c)){const[d,...f]=c;s(d,f)}else o(c);else throw new TypeError("Expected usable value, not `"+c+"`")}function o(c){if(!("plugins"in c)&&!("settings"in c))throw new Error("Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither");a(c.plugins),c.settings&&(i.settings=ko(!0,i.settings,c.settings))}function a(c){let d=-1;if(c!=null)if(Array.isArray(c))for(;++d<c.length;){const f=c[d];l(f)}else throw new TypeError("Expected a list of plugins, not `"+c+"`")}function s(c,d){let f=-1,m=-1;for(;++f<r.length;)if(r[f][0]===c){m=f;break}if(m===-1)r.push([c,...d]);else if(d.length>0){let[p,...w]=d;const S=r[m][1];Ta(S)&&Ta(p)&&(p=ko(!0,S,p)),r[m]=[c,p,...w]}}}}const E0=new Us().freeze();function Co(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `parser`")}function jo(e,t){if(typeof t!="function")throw new TypeError("Cannot `"+e+"` without `compiler`")}function Eo(e,t){if(t)throw new Error("Cannot call `"+e+"` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`.")}function Vc(e){if(!Ta(e)||typeof e.type!="string")throw new TypeError("Expected node, got `"+e+"`")}function Wc(e,t,n){if(!n)throw new Error("`"+e+"` finished async. Use `"+t+"` instead")}function Ii(e){return N0(e)?e:new Hp(e)}function N0(e){return!!(e&&typeof e=="object"&&"message"in e&&"messages"in e)}function _0(e){return typeof e=="string"||z0(e)}function z0(e){return!!(e&&typeof e=="object"&&"byteLength"in e&&"byteOffset"in e)}const T0="https://github.com/remarkjs/react-markdown/blob/main/changelog.md",Qc=[],qc={allowDangerousHtml:!0},L0=/^(https?|ircs?|mailto|xmpp)$/i,P0=[{from:"astPlugins",id:"remove-buggy-html-in-markdown-parser"},{from:"allowDangerousHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"allowNode",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowElement"},{from:"allowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"allowedElements"},{from:"className",id:"remove-classname"},{from:"disallowedTypes",id:"replace-allownode-allowedtypes-and-disallowedtypes",to:"disallowedElements"},{from:"escapeHtml",id:"remove-buggy-html-in-markdown-parser"},{from:"includeElementIndex",id:"#remove-includeelementindex"},{from:"includeNodeIndex",id:"change-includenodeindex-to-includeelementindex"},{from:"linkTarget",id:"remove-linktarget"},{from:"plugins",id:"change-plugins-to-remarkplugins",to:"remarkPlugins"},{from:"rawSourcePos",id:"#remove-rawsourcepos"},{from:"renderers",id:"change-renderers-to-components",to:"components"},{from:"source",id:"change-source-to-children",to:"children"},{from:"sourcePos",id:"#remove-sourcepos"},{from:"transformImageUri",id:"#add-urltransform",to:"urlTransform"},{from:"transformLinkUri",id:"#add-urltransform",to:"urlTransform"}];function I0(e){const t=M0(e),n=A0(e);return D0(t.runSync(t.parse(n),n),e)}function M0(e){const t=e.rehypePlugins||Qc,n=e.remarkPlugins||Qc,r=e.remarkRehypeOptions?{...e.remarkRehypeOptions,...qc}:qc;return E0().use(h1).use(n).use(u0,r).use(t)}function A0(e){const t=e.children||"",n=new Hp;return typeof t=="string"&&(n.value=t),n}function D0(e,t){const n=t.allowedElements,r=t.allowElement,i=t.components,l=t.disallowedElements,o=t.skipHtml,a=t.unwrapDisallowed,s=t.urlTransform||R0;for(const d of P0)Object.hasOwn(t,d.from)&&(""+d.from+(d.to?"use `"+d.to+"` instead":"remove it")+T0+d.id,void 0);return $p(e,c),qv(e,{Fragment:u.Fragment,components:i,ignoreInvalidStyle:!0,jsx:u.jsx,jsxs:u.jsxs,passKeys:!0,passNode:!0});function c(d,f,m){if(d.type==="raw"&&m&&typeof f=="number")return o?m.children.splice(f,1):m.children[f]={type:"text",value:d.value},f;if(d.type==="element"){let p;for(p in vo)if(Object.hasOwn(vo,p)&&Object.hasOwn(d.properties,p)){const w=d.properties[p],S=vo[p];(S===null||S.includes(d.tagName))&&(d.properties[p]=s(String(w||""),p,d))}}if(d.type==="element"){let p=n?!n.includes(d.tagName):l?l.includes(d.tagName):!1;if(!p&&r&&typeof f=="number"&&(p=!r(d,f,m)),p&&m&&typeof f=="number")return a&&d.children?m.children.splice(f,1,...d.children):m.children.splice(f,1),f}}}function R0(e){const t=e.indexOf(":"),n=e.indexOf("?"),r=e.indexOf("#"),i=e.indexOf("/");return t===-1||i!==-1&&t>i||n!==-1&&t>n||r!==-1&&t>r||L0.test(e.slice(0,t))?e:""}const F0=e=>{if(!e)return null;try{return JSON.parse(e).execution_stats||null}catch{return null}},O0=e=>{if(e.kind!=="status")return!1;const t=e.content.toLowerCase();return t.includes("running")||t.includes("thinking")||t.includes("executing")||t.includes("processing")},Kc=10*1024,No=200,Te={send:u.jsxs("svg",{width:"18",height:"18",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"22",y1:"2",x2:"11",y2:"13"}),u.jsx("polygon",{points:"22 2 15 22 11 13 2 9 22 2"})]}),directive:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"}),u.jsx("line",{x1:"16",y1:"13",x2:"8",y2:"13"}),u.jsx("line",{x1:"16",y1:"17",x2:"8",y2:"17"})]}),question:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"}),u.jsx("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"})]}),status:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 12h-4l-3 9L9 3l-3 9H2"})}),result:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}),lock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"11",rx:"2",ry:"2"}),u.jsx("path",{d:"M7 11V7a5 5 0 0 1 10 0v4"})]}),user:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"}),u.jsx("circle",{cx:"12",cy:"7",r:"4"})]}),bot:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),file:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"}),u.jsx("polyline",{points:"14 2 14 8 20 8"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),spinner:u.jsx("svg",{className:"spinner-icon",width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 12a9 9 0 1 1-6.219-8.56"})})},B0=e=>{switch(e){case"directive":return Te.directive;case"question":return Te.question;case"status":return Te.status;case"result":return Te.result;case"approval_request":return Te.lock;default:return Te.directive}},$0=({thread:e,messages:t,onSendMessage:n,onWorkspaceChange:r,onApproveRequest:i,onRejectRequest:l})=>{const o=O.useRef(null),[a,s]=qt.useState(""),[c,d]=qt.useState("directive"),[f,m]=qt.useState(""),[p,w]=qt.useState(!1),[S,I]=qt.useState(new Map),[h,v]=qt.useState(new Set),[y,C]=O.useState(new Set),[N,k]=O.useState(new Set),j=b=>{const E=(b.match(/\n/g)||[]).length+1;if(!(b.length>Kc||E>No))return{needsTruncation:!1,truncated:b,fullLength:b.length,lineCount:E};let L=b.slice(0,Kc);const $=L.split(`
`);$.length>No&&(L=$.slice(0,No).join(`
`));const x=L.lastIndexOf(`
`);return x>L.length*.8&&(L=L.slice(0,x)),{needsTruncation:!0,truncated:L,fullLength:b.length,lineCount:E}},_=b=>{C(E=>{const g=new Set(E);return g.has(b)?g.delete(b):g.add(b),g})};O.useEffect(()=>{e!=null&&e.workspace?m(e.workspace):m("")},[e==null?void 0:e.id,e==null?void 0:e.workspace]),O.useEffect(()=>{var b;(b=o.current)==null||b.scrollIntoView({behavior:"smooth"})},[t]);const R=b=>{m(b),r&&r(b)},P=()=>{a.trim()&&(n(a,c,f||void 0),s(""))},z=b=>{b.key==="Enter"&&!b.shiftKey&&(b.preventDefault(),P())},D=b=>new Date(b).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"}),W=b=>b.length>12?`${b.slice(0,8)}...`:b,X=b=>{if(!b.metadata_json)return null;try{return JSON.parse(b.metadata_json).approval_id||null}catch{return null}},U=b=>{const E=S.get(b)||"";i&&(i(b,E),v(g=>new Set(g).add(b)),I(g=>{const L=new Map(g);return L.delete(b),L}))},Q=b=>{const E=S.get(b)||"";if(!E.trim()){alert("Please provide a reason for rejection");return}l&&(l(b,E),v(g=>new Set(g).add(b)),I(g=>{const L=new Map(g);return L.delete(b),L}))},ie=(b,E)=>{I(g=>new Map(g).set(b,E))};return e?u.jsxs("div",{className:"conversation-view",children:[u.jsxs("div",{className:"conversation-header",children:[u.jsxs("div",{className:"header-info",children:[u.jsx("h2",{className:"thread-title",children:e.title}),e.target_agent&&u.jsxs("span",{className:"thread-agent-badge",children:[Te.bot,e.target_agent]})]}),u.jsxs("div",{className:"header-stats",children:[u.jsxs("span",{className:"message-count",children:[t.length," messages"]}),u.jsx("span",{className:"thread-id",title:e.id,children:W(e.id)})]})]}),u.jsxs("div",{className:"messages-container",children:[t.length===0?u.jsxs("div",{className:"empty-messages",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("p",{children:"No messages yet"}),u.jsx("span",{className:"hint",children:"Send a message to start the conversation"})]}):t.map((b,E)=>{const g=b.from_type==="human",L=E===0||t[E-1].from_type!==b.from_type,$=y.has(b.id),{needsTruncation:x,truncated:ne,fullLength:be,lineCount:te}=j(b.content),Me=$?b.content:ne,lt=O0(b);return u.jsxs("div",{className:`message ${g?"human":"agent"}${lt?" running-status":""}`,children:[u.jsx("div",{className:`message-avatar ${L?"visible":""}`,children:L&&(g?Te.user:Te.bot)}),u.jsxs("div",{className:"message-body",children:[L&&u.jsxs("div",{className:"message-meta",children:[u.jsx("span",{className:"sender-name",children:b.from_id}),u.jsxs("span",{className:`kind-badge${lt?" running":""}`,children:[lt?Te.spinner:B0(b.kind)," ",b.kind]}),u.jsx("span",{className:"message-time",children:D(b.created_at)})]}),u.jsxs("div",{className:"message-content",children:[b.kind==="result"||!g?u.jsx(I0,{components:{a:({href:J,children:Ce})=>{let Be=J;return J&&J.startsWith("/")&&!J.startsWith("//")&&(Be=`file://${J}`),u.jsx("a",{href:Be,target:"_blank",rel:"noopener noreferrer",children:Ce})},code:({className:J,children:Ce,...Be})=>!J?u.jsx("code",{className:"inline-code",...Be,children:Ce}):u.jsx("code",{className:J,...Be,children:Ce})},children:Me}):Me,x&&u.jsx("div",{className:"truncation-notice",children:u.jsx("button",{className:"expand-btn",onClick:()=>_(b.id),children:$?u.jsx(u.Fragment,{children:"Show less"}):u.jsxs(u.Fragment,{children:["Show more (",Math.round(be/1024),"KB, ",te," lines)"]})})}),b.kind==="approval_request"&&(()=>{const J=X(b),Ce=J&&h.has(J);return J?u.jsx("div",{className:"inline-approval",children:Ce?u.jsxs("div",{className:"approval-handled",children:[Te.check,u.jsx("span",{children:"Action taken"})]}):u.jsxs(u.Fragment,{children:[u.jsx("input",{type:"text",className:"approval-notes-input",placeholder:"Notes (required for rejection)...",value:S.get(J)||"",onChange:Be=>ie(J,Be.target.value)}),u.jsxs("div",{className:"approval-actions",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>Q(J),title:"Reject",children:[Te.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>U(J),title:"Approve",children:[Te.check,"Approve"]})]})]})}):null})(),b.kind==="result"&&(()=>{const J=F0(b.metadata_json);if(!J||!J.files_created||J.files_created.length===0)return null;const Ce=N.has(b.id),Be=()=>{k(Ht=>{const Vt=new Set(Ht);return Vt.has(b.id)?Vt.delete(b.id):Vt.add(b.id),Vt})};return u.jsxs("div",{className:"files-created-section",children:[u.jsxs("button",{className:`files-toggle-btn ${Ce?"expanded":""}`,onClick:Be,children:[Te.file,u.jsxs("span",{children:["Files Created (",J.files_created.length,")"]}),J.workspace&&u.jsxs("span",{className:"workspace-badge",title:J.workspace,children:[Te.folder,J.workspace.split("/").pop()]}),u.jsx("span",{className:"toggle-chevron",children:Ce?"▼":"▶"})]}),Ce&&u.jsx("ul",{className:"files-list",children:J.files_created.map((Ht,Vt)=>u.jsx("li",{className:"file-item",children:u.jsx("a",{href:`file://${J.workspace?J.workspace+"/":""}${Ht}`,target:"_blank",rel:"noopener noreferrer",title:Ht,children:Ht})},Vt))})]})})()]}),u.jsxs("div",{className:"message-footer",children:[u.jsxs("span",{className:"message-seq",children:["#",b.message_seq]}),b.delivery_state!=="acked"&&u.jsx("span",{className:`delivery-status ${b.delivery_state}`,children:b.delivery_state==="pending"?"sending...":"delivered"})]})]})]},b.id)}),u.jsx("div",{ref:o})]}),u.jsxs("div",{className:"input-area",children:[p&&u.jsxs("div",{className:"workspace-input-row",children:[u.jsx("input",{type:"text",value:f,onChange:b=>R(b.target.value),onBlur:()=>{r&&r(f)},placeholder:"/path/to/working/directory (leave empty for fresh workspace)",className:"workspace-input"}),u.jsx("button",{onClick:async()=>{try{const E=await(await fetch("/api/select-folder")).json();!E.cancelled&&E.path&&R(E.path)}catch(b){console.error("Failed to open folder picker:",b)}},className:"workspace-browse",title:"Browse for folder",children:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"}),u.jsx("line",{x1:"12",y1:"11",x2:"12",y2:"17"}),u.jsx("line",{x1:"9",y1:"14",x2:"15",y2:"14"})]})}),f&&u.jsx("button",{onClick:()=>{R(""),w(!1)},className:"workspace-clear",children:"Clear"})]}),u.jsxs("div",{className:"input-wrapper",children:[u.jsx("button",{onClick:()=>w(!p),className:`workspace-toggle ${f?"has-workspace":""}`,title:f||"Set working directory",children:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})})}),u.jsxs("select",{value:c,onChange:b=>d(b.target.value),className:"kind-selector",children:[u.jsx("option",{value:"directive",children:"Directive"}),u.jsx("option",{value:"question",children:"Question"})]}),u.jsx("textarea",{value:a,onChange:b=>s(b.target.value),onKeyPress:z,placeholder:f?`Message (workspace: ${f.split("/").pop()})`:"Type a message...",rows:1}),u.jsx("button",{onClick:P,className:"send-btn",disabled:!a.trim(),children:Te.send})]}),u.jsxs("div",{className:"input-hint",children:["Press ",u.jsx("kbd",{children:"Enter"})," to send, ",u.jsx("kbd",{children:"Shift + Enter"})," for new line"]})]}),u.jsx("style",{children:`
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
      `})]}):null};let Vp="disconnected",Wp=0;const Pa=new Set;function Mi(e,t){Vp=e,Wp=t,Pa.forEach(n=>n(e,t))}function U0(e){return Pa.add(e),e(Vp,Wp),()=>Pa.delete(e)}function H0(e,t=1e3,n=3e4){const r=Math.min(t*Math.pow(2,e),n),i=r*Math.random()*.3;return Math.round(r+i)}const V0=({url:e,instanceId:t,onMessage:n,onBatch:r,onError:i,maxReconnectAttempts:l=10})=>{const o=O.useRef(null),[a,s]=O.useState(!1),[c,d]=O.useState(null),[f,m]=O.useState(0),p=O.useRef(null),w=O.useRef(new Map),S=O.useCallback(()=>{try{const k=`${e}?instance_id=${t}`;o.current=new WebSocket(k),Mi(f>0?"reconnecting":"connecting",f),o.current.onopen=()=>{console.log("WebSocket connected"),s(!0),d(null),m(0),Mi("connected",0),w.current.forEach((j,_)=>{v(_,j)})},o.current.onmessage=j=>{try{const _=JSON.parse(j.data);I(_)}catch(_){console.error("Failed to parse WebSocket message:",_)}},o.current.onerror=j=>{console.error("WebSocket error:",j),d("Connection error")},o.current.onclose=()=>{if(console.log("WebSocket disconnected"),s(!1),Mi("disconnected",f),f<l){const j=H0(f);console.log(`WebSocket reconnecting in ${j}ms (attempt ${f+1}/${l})`),p.current=setTimeout(()=>{m(_=>_+1),S()},j)}else console.error("Max reconnection attempts reached"),d("Connection lost. Please refresh the page.")}}catch(k){console.error("Failed to connect to WebSocket:",k),d("Failed to connect"),Mi("disconnected",f)}},[e,t,f,l]),I=O.useCallback(k=>{switch(k.type){case"message":n&&k.data&&n(k.data);break;case"batch":if(r&&k.data){const j=k.data;r(j),n&&j.messages.forEach(_=>n(_))}break;case"error":i&&k.data&&i(k.data),console.error("WebSocket error event:",k.data);break;case"pong":break;default:console.log("Unknown event type:",k.type)}},[n,r,i]),h=O.useCallback(k=>{o.current&&o.current.readyState===WebSocket.OPEN?o.current.send(JSON.stringify(k)):console.warn("WebSocket not connected, cannot send event")},[]),v=O.useCallback((k,j=0)=>{w.current.set(k,j);const _={type:"subscribe",timestamp:Date.now(),data:{thread_id:k,from_seq:j}};h(_)},[h]),y=O.useCallback((k,j)=>{const _=w.current.get(k)||0;j>_&&w.current.set(k,j);const R={type:"ack",timestamp:Date.now(),data:{thread_id:k,ack_seq:j}};h(R)},[h]),C=O.useCallback(()=>{const k={type:"ping",timestamp:Date.now()};h(k)},[h]),N=O.useCallback(k=>{w.current.delete(k)},[]);return O.useEffect(()=>(S(),()=>{p.current&&clearTimeout(p.current),o.current&&o.current.close()}),[S]),O.useEffect(()=>{if(!a)return;const k=setInterval(()=>{C()},3e4);return()=>clearInterval(k)},[a,C]),{isConnected:a,connectionError:c,subscribe:v,unsubscribe:N,acknowledge:y,ping:C}},W0=({connected:e})=>u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",children:e?u.jsxs(u.Fragment,{children:[u.jsx("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"}),u.jsx("polyline",{points:"22 4 12 14.01 9 11.01"})]}):u.jsxs(u.Fragment,{children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("line",{x1:"15",y1:"9",x2:"9",y2:"15"}),u.jsx("line",{x1:"9",y1:"9",x2:"15",y2:"15"})]})}),Q0=({websocketUrl:e,instanceId:t,initialThreadId:n,onThreadNavigated:r})=>{const[i,l]=O.useState([]),[o,a]=O.useState(null),[s,c]=O.useState(new Map),[d,f]=O.useState(new Map),[m,p]=O.useState([]),[w,S]=O.useState(!1),[I,h]=O.useState(""),{isConnected:v,subscribe:y,acknowledge:C}=V0({url:e,instanceId:t,onMessage:N,onBatch:k});function N(E){const g={id:E.id,thread_id:E.thread_id,message_seq:E.message_seq,created_at:E.created_at,from_type:E.from_type,from_id:E.from_id,to_type:E.to_type,to_id:E.to_id,kind:E.kind,subject:E.subject,content:E.content,metadata_json:E.metadata_json,delivery_state:"visible",business_state:"open"};c(L=>{const $=L.get(g.thread_id)||[];return $.find(x=>x.id===g.id)?L:new Map(L).set(g.thread_id,[...$,g].sort((x,ne)=>x.message_seq-ne.message_seq))}),g.thread_id!==o&&f(L=>{const $=L.get(g.thread_id)||0;return new Map(L).set(g.thread_id,$+1)}),C(g.thread_id,g.message_seq)}function k(E){E.messages.forEach(g=>{N(g)})}const j=O.useCallback(E=>{if(a(E),f(g=>{const L=new Map(g);return L.delete(E),L}),v){const g=s.get(E)||[],L=g.length>0?Math.max(...g.map($=>$.message_seq)):0;y(E,L)}},[v,y,s]),_=O.useCallback(async(E,g,L)=>{if(!o)return;const $=L?JSON.stringify({workspace:L}):void 0;try{const x=await fetch("/api/messages",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({thread_id:o,from_type:"human",from_id:"user",to_type:"ailang_instance",to_id:t,kind:g,content:E,metadata_json:$})});if(!x.ok){console.error("Failed to send message:",await x.text());return}const ne=await x.json();c(be=>{const te=be.get(o)||[];return te.find(Me=>Me.id===ne.id)?be:new Map(be).set(o,[...te,ne])})}catch(x){console.error("Error sending message:",x)}},[o,t]);O.useEffect(()=>{(async()=>{try{const g=await fetch("/api/threads");if(!g.ok){console.error("Failed to fetch threads:",await g.text());return}const L=await g.json();l(L),L.length>0&&!o&&a(L[0].id)}catch(g){console.error("Error fetching threads:",g)}})()},[]),O.useEffect(()=>{n&&i.length>0&&(i.some(g=>g.id===n)&&(a(n),f(g=>{const L=new Map(g);return L.delete(n),L})),r&&r())},[n,i,r]);const R=O.useCallback(async E=>{try{const g=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:E,created_by_type:"human",created_by_id:"user",target_agent:t})});if(!g.ok){console.error("Failed to create thread:",await g.text());return}const L=await g.json();l($=>[L,...$]),a(L.id)}catch(g){console.error("Error creating thread:",g)}},[t]),P=O.useCallback(async()=>{try{const E=await fetch("/api/agents");if(!E.ok){console.error("Failed to fetch agents:",await E.text());return}const g=await E.json();p(g.running||[])}catch(E){console.error("Error fetching agents:",E)}},[]);O.useEffect(()=>{P();const E=setInterval(P,5e3);return()=>clearInterval(E)},[P]);const z=O.useCallback(async()=>{if(I.trim())try{const E=await fetch("/api/agents",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({instance_id:I.trim()})});if(!E.ok){const L=await E.text();console.error("Failed to launch agent:",L),alert(`Failed to launch agent: ${L}`);return}const g=await E.json();p(L=>[...L,g]),h(""),S(!1)}catch(E){console.error("Error launching agent:",E)}},[I]),D=O.useCallback(async E=>{try{const g=await fetch(`/api/agents/${E}`,{method:"DELETE"});if(!g.ok){console.error("Failed to stop agent:",await g.text());return}p(L=>L.filter($=>$.instance_id!==E))}catch(g){console.error("Error stopping agent:",g)}},[]),W=O.useCallback(async E=>{if(o)try{const g=await fetch(`/api/threads/${o}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({workspace:E})});if(!g.ok){console.error("Failed to update workspace:",await g.text());return}const L=await g.json();l($=>$.map(x=>x.id===o?L:x))}catch(g){console.error("Error updating workspace:",g)}},[o]),X=O.useCallback(async E=>{try{const g=await fetch(`/api/threads/${E}`,{method:"DELETE"});if(!g.ok){console.error("Failed to delete thread:",await g.text());return}l(L=>L.filter($=>$.id!==E)),c(L=>{const $=new Map(L);return $.delete(E),$}),f(L=>{const $=new Map(L);return $.delete(E),$}),o===E&&a(null)}catch(g){console.error("Error deleting thread:",g)}},[o]),U=O.useCallback(async(E,g)=>{try{const L=await fetch(`/api/threads/${E}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:g})});if(!L.ok){console.error("Failed to rename thread:",await L.text());return}const $=await L.json();l(x=>x.map(ne=>ne.id===E?$:ne))}catch(L){console.error("Error renaming thread:",L)}},[]),Q=O.useCallback(async(E,g)=>{try{const L=await fetch(`/api/approvals/${E}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!L.ok){const $=await L.text();console.error("Failed to approve request:",$),alert(`Failed to approve: ${$}`);return}console.log("Approval approved successfully")}catch(L){console.error("Error approving request:",L)}},[]),ie=O.useCallback(async(E,g)=>{try{const L=await fetch(`/api/approvals/${E}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({reviewed_by:"user",review_notes:g})});if(!L.ok){const $=await L.text();console.error("Failed to reject request:",$),alert(`Failed to reject: ${$}`);return}console.log("Approval rejected successfully")}catch(L){console.error("Error rejecting request:",L)}},[]),b=o?s.get(o)||[]:[];return u.jsxs("div",{className:"message-center",children:[u.jsxs("div",{className:"status-bar",children:[u.jsxs("div",{className:`status-indicator ${v?"connected":"disconnected"}`,children:[u.jsx(W0,{connected:v}),u.jsx("span",{children:v?"Connected":"Disconnected"})]}),u.jsxs("div",{className:"status-meta",children:[u.jsxs("span",{className:"thread-count",children:[i.length," threads"]}),u.jsxs("span",{className:"agent-count",children:[m.length," agents"]}),u.jsx("button",{className:"launch-agent-btn",onClick:()=>S(!0),children:"+ Agent"})]})]}),m.length>0&&u.jsx("div",{className:"agents-bar",children:m.map(E=>u.jsxs("div",{className:"agent-chip",children:[u.jsx("span",{className:"agent-pulse"}),u.jsx("span",{className:"agent-name",children:E.instance_id}),u.jsxs("span",{className:"agent-pid",children:["PID ",E.pid]}),u.jsx("button",{className:"agent-stop-btn",onClick:()=>D(E.instance_id),title:"Stop agent",children:"×"})]},E.instance_id))}),w&&u.jsx("div",{className:"modal-overlay",onClick:()=>S(!1),children:u.jsxs("div",{className:"modal-content",onClick:E=>E.stopPropagation(),children:[u.jsx("h3",{children:"Launch New Agent"}),u.jsx("input",{type:"text",value:I,onChange:E=>h(E.target.value),placeholder:"Enter instance ID (e.g., agent-2)",autoFocus:!0,onKeyDown:E=>{E.key==="Enter"&&z(),E.key==="Escape"&&S(!1)}}),u.jsxs("div",{className:"modal-actions",children:[u.jsx("button",{className:"cancel-btn",onClick:()=>S(!1),children:"Cancel"}),u.jsx("button",{className:"launch-btn",onClick:z,children:"Launch"})]})]})}),u.jsxs("div",{className:"center-layout",children:[u.jsx("aside",{className:"threads-panel",children:u.jsx(Xg,{threads:i,selectedThreadId:o,onSelectThread:j,onCreateThread:R,onDeleteThread:X,onRenameThread:U,unreadCounts:d})}),u.jsx("main",{className:"conversation-panel",children:o?u.jsx($0,{thread:i.find(E=>E.id===o),messages:b,onSendMessage:_,onWorkspaceChange:W,onApproveRequest:Q,onRejectRequest:ie}):u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:u.jsx("svg",{width:"48",height:"48",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})})}),u.jsx("h3",{children:"Select a conversation"}),u.jsx("p",{children:"Choose a thread from the sidebar or create a new one to get started"})]})})]}),u.jsx("style",{children:`
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
      `})]})},Ae={check:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"20 6 9 17 4 12"})}),x:u.jsxs("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"18",y1:"6",x2:"6",y2:"18"}),u.jsx("line",{x1:"6",y1:"6",x2:"18",y2:"18"})]}),chevronDown:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"6 9 12 15 18 9"})}),chevronUp:u.jsx("svg",{width:"16",height:"16",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("polyline",{points:"18 15 12 9 6 15"})}),bot:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("rect",{x:"3",y:"11",width:"18",height:"10",rx:"2"}),u.jsx("circle",{cx:"12",cy:"5",r:"2"}),u.jsx("path",{d:"M12 7v4"})]}),dollar:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("line",{x1:"12",y1:"1",x2:"12",y2:"23"}),u.jsx("path",{d:"M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"})]}),folder:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"})}),clock:u.jsxs("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("polyline",{points:"12 6 12 12 16 14"})]}),message:u.jsx("svg",{width:"14",height:"14",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"2",strokeLinecap:"round",strokeLinejoin:"round",children:u.jsx("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"})}),sparkles:u.jsxs("svg",{width:"40",height:"40",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("path",{d:"M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"}),u.jsx("path",{d:"M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z"}),u.jsx("path",{d:"M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z"})]})},q0=({approvals:e,history:t=[],onApprove:n,onReject:r,onNavigateToThread:i})=>{const[l,o]=O.useState(!0),[a,s]=O.useState(null),[c,d]=O.useState(new Map),f=h=>{try{return JSON.parse(h)}catch{return null}},m=h=>new Date(h).toLocaleString(void 0,{month:"short",day:"numeric",hour:"2-digit",minute:"2-digit"}),p=h=>{const v=c.get(h)||"";n(h,v),d(new Map(c.set(h,"")))},w=h=>{const v=c.get(h)||"";if(!v.trim()){alert("Please provide a reason for rejection");return}r(h,v),d(new Map(c.set(h,"")))},S=(h,v)=>{d(new Map(c.set(h,v)))},I=e.filter(h=>h.status==="pending");return u.jsxs("div",{className:"approval-queue",children:[u.jsx("div",{className:"queue-header",children:u.jsxs("div",{className:"header-title",children:[u.jsx("h2",{children:"Approval Queue"}),u.jsxs("span",{className:"pending-count",children:[I.length," pending"]})]})}),u.jsxs("div",{className:"approvals-container",children:[I.length===0?u.jsxs("div",{className:"empty-state",children:[u.jsx("div",{className:"empty-icon",children:Ae.sparkles}),u.jsx("h3",{children:"All caught up!"}),u.jsx("p",{children:"No pending approvals to review"})]}):u.jsx("div",{className:"approvals-list",children:I.map(h=>{const v=f(h.effect_delta_json),y=a===h.id;return u.jsxs("div",{className:`approval-card impact-${h.impact}`,children:[u.jsxs("div",{className:"card-header",onClick:()=>s(y?null:h.id),children:[u.jsxs("div",{className:"header-left",children:[u.jsx("div",{className:`impact-indicator ${h.impact}`}),u.jsxs("div",{className:"proposal-info",children:[u.jsx("span",{className:"proposal-text",children:h.proposal}),u.jsxs("div",{className:"proposal-meta",children:[h.thread_title&&u.jsxs("span",{className:"meta-item thread-link",onClick:C=>{C.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Ae.message,h.thread_title]}),u.jsxs("span",{className:"meta-item",children:[Ae.bot,h.instance_id]}),u.jsxs("span",{className:"meta-item",children:[Ae.clock,m(h.created_at)]})]})]})]}),u.jsxs("div",{className:"header-right",children:[u.jsxs("span",{className:"cost-badge",children:[Ae.dollar,"$",h.estimated_cost.toFixed(2)]}),u.jsx("span",{className:`impact-badge ${h.impact}`,children:h.impact}),u.jsx("button",{className:"expand-btn",children:y?Ae.chevronUp:Ae.chevronDown})]})]}),y&&u.jsxs("div",{className:"card-details",children:[v&&u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Effect Details"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Capability"}),u.jsx("span",{className:"detail-value code",children:v.cap_type})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Budget Delta"}),u.jsxs("span",{className:"detail-value",children:["$",v.budget_delta.toFixed(2)]})]}),v.paths.length>0&&u.jsxs("div",{className:"detail-item full-width",children:[u.jsx("span",{className:"detail-label",children:"Paths"}),u.jsx("div",{className:"paths-list",children:v.paths.map((C,N)=>u.jsxs("span",{className:"path-tag",children:[Ae.folder,C]},N))})]})]})]}),u.jsxs("div",{className:"detail-section",children:[u.jsx("h4",{children:"Request Info"}),u.jsxs("div",{className:"detail-grid",children:[u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Thread"}),u.jsx("span",{className:"detail-value code",children:h.thread_id})]}),u.jsxs("div",{className:"detail-item",children:[u.jsx("span",{className:"detail-label",children:"Impact Level"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]})]})]}),u.jsxs("div",{className:"review-section",children:[u.jsx("h4",{children:"Review Notes"}),u.jsx("textarea",{value:c.get(h.id)||"",onChange:C=>S(h.id,C.target.value),placeholder:"Add notes about your decision (required for rejection)...",rows:3}),u.jsxs("div",{className:"action-buttons",children:[u.jsxs("button",{className:"reject-btn",onClick:()=>w(h.id),children:[Ae.x,"Reject"]}),u.jsxs("button",{className:"approve-btn",onClick:()=>p(h.id),children:[Ae.check,"Approve"]})]})]})]})]},h.id)})}),t.length>0&&u.jsxs("div",{className:"history-section",children:[u.jsxs("div",{className:"history-header",onClick:()=>o(!l),children:[u.jsxs("h3",{children:[l?Ae.chevronDown:Ae.chevronUp,"Review History"]}),u.jsxs("span",{className:"history-count",children:[t.length," decisions"]})]}),l&&u.jsx("div",{className:"history-list",children:t.map(h=>{const v=a===`history-${h.id}`;return u.jsxs("div",{className:`history-card ${h.status}`,onClick:()=>s(v?null:`history-${h.id}`),children:[u.jsxs("div",{className:"history-card-header",children:[u.jsxs("div",{className:"history-status",children:[u.jsx("span",{className:`status-icon ${h.status}`,children:h.status==="approved"?Ae.check:Ae.x}),u.jsxs("div",{className:"history-info",children:[u.jsx("span",{className:"history-proposal",children:h.proposal}),h.thread_title&&u.jsxs("span",{className:"history-thread",onClick:y=>{y.stopPropagation(),i==null||i(h.thread_id)},title:"Go to thread",children:[Ae.message,h.thread_title]})]})]}),u.jsxs("div",{className:"history-meta",children:[u.jsx("span",{className:"history-agent",children:h.instance_id}),u.jsx("span",{className:`history-badge ${h.status}`,children:h.status}),u.jsx("span",{className:"history-time",children:h.reviewed_at?m(h.reviewed_at):m(h.created_at)})]})]}),v&&u.jsxs("div",{className:"history-details",children:[u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Reviewed by"}),u.jsx("span",{className:"detail-value",children:h.reviewed_by||"Unknown"})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Cost"}),u.jsxs("span",{className:"detail-value",children:["$",h.estimated_cost.toFixed(2)]})]}),u.jsxs("div",{className:"detail-row",children:[u.jsx("span",{className:"detail-label",children:"Impact"}),u.jsx("span",{className:`detail-value impact-text ${h.impact}`,children:h.impact.toUpperCase()})]}),h.review_notes&&u.jsxs("div",{className:"detail-row full-width",children:[u.jsx("span",{className:"detail-label",children:"Notes"}),u.jsx("span",{className:"detail-value notes",children:h.review_notes})]})]})]},`history-${h.id}`)})})]})]}),u.jsx("style",{children:`
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
      `})]})},K0="_indicator_1ctaf_1",Y0="_dot_1ctaf_12",X0="_connected_1ctaf_19",G0="_connecting_1ctaf_28",J0="_disconnected_1ctaf_37",Z0="_pulsing_1ctaf_46",ek="_text_1ctaf_61",Pt={indicator:K0,dot:Y0,connected:X0,connecting:G0,disconnected:J0,pulsing:Z0,text:ek};function tk(){const[e,t]=O.useState("disconnected"),[n,r]=O.useState(0);if(O.useEffect(()=>U0((o,a)=>{t(o),r(a)}),[]),e==="connected")return u.jsx("div",{className:`${Pt.indicator} ${Pt.connected}`,title:"Connected",children:u.jsx("span",{className:Pt.dot})});const i=()=>{switch(e){case"connecting":return"Connecting...";case"reconnecting":return`Reconnecting... (${n})`;case"disconnected":return n>0?"Disconnected":"Offline";default:return"Unknown"}},l=()=>{switch(e){case"connecting":case"reconnecting":return Pt.connecting;case"disconnected":return Pt.disconnected;default:return""}};return u.jsxs("div",{className:`${Pt.indicator} ${l()}`,title:i(),children:[u.jsx("span",{className:`${Pt.dot} ${e==="connecting"||e==="reconnecting"?Pt.pulsing:""}`}),u.jsx("span",{className:Pt.text,children:i()})]})}const nk=u.jsxs("svg",{width:"28",height:"28",viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:"1.5",strokeLinecap:"round",strokeLinejoin:"round",children:[u.jsx("circle",{cx:"12",cy:"12",r:"10"}),u.jsx("path",{d:"M12 6v12M6 12h12"}),u.jsx("circle",{cx:"12",cy:"12",r:"3",fill:"currentColor"})]}),rk=()=>{const[e,t]=O.useState({type:"overview"}),[n,r]=O.useState(null),[i,l]=O.useState([]),[o,a]=O.useState([]),[s,c]=O.useState(!1),[d,f]=O.useState(""),p=`${window.location.protocol==="https:"?"wss:":"ws:"}//${window.location.host}/ws`;O.useEffect(()=>{const N=async()=>{try{const j=await fetch("/api/hierarchy");if(j.ok){const _=await j.json();r(_)}}catch(j){console.error("Error fetching hierarchy:",j)}};N();const k=setInterval(N,5e3);return()=>clearInterval(k)},[]),O.useEffect(()=>{const N=async()=>{try{const j=await fetch("/api/approvals?status=pending");if(j.ok){const z=await j.json();l(z)}const[_,R]=await Promise.all([fetch("/api/approvals?status=approved"),fetch("/api/approvals?status=rejected")]),P=[];if(_.ok){const z=await _.json();P.push(...z)}if(R.ok){const z=await R.json();P.push(...z)}P.sort((z,D)=>{const W=z.reviewed_at?new Date(z.reviewed_at).getTime():0;return(D.reviewed_at?new Date(D.reviewed_at).getTime():0)-W}),a(P)}catch(j){console.error("Error fetching approvals:",j)}};N();const k=setInterval(N,5e3);return()=>clearInterval(k)},[]);const w=async(N,k)=>{try{const j=await fetch(`/api/approvals/${N}/approve`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:k})});if(!j.ok){console.error("Failed to approve:",await j.text());return}const _=i.find(R=>R.id===N);if(_){const R={..._,status:"approved",reviewed_by:"user",review_notes:k,reviewed_at:Date.now()};a(P=>[R,...P])}l(R=>R.filter(P=>P.id!==N))}catch(j){console.error("Error approving:",j)}},S=async(N,k)=>{try{const j=await fetch(`/api/approvals/${N}/reject`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({notes:k})});if(!j.ok){console.error("Failed to reject:",await j.text());return}const _=i.find(R=>R.id===N);if(_){const R={..._,status:"rejected",reviewed_by:"user",review_notes:k,reviewed_at:Date.now()};a(P=>[R,...P])}l(R=>R.filter(P=>P.id!==N))}catch(j){console.error("Error rejecting:",j)}},I=()=>{var k,j;const N=[{label:"All Agents",onClick:()=>t({type:"overview"})}];if(e.type==="agent"&&e.agentId&&N.push({label:e.agentId}),e.type==="thread"&&e.threadId){e.agentId&&N.push({label:e.agentId,onClick:()=>t({type:"agent",agentId:e.agentId})});const _=(k=n==null?void 0:n.root.children)==null?void 0:k.find(P=>P.id===e.agentId),R=(j=_==null?void 0:_.children)==null?void 0:j.find(P=>P.id===e.threadId);N.push({label:(R==null?void 0:R.label)||"Thread"})}return N},h=N=>{var j;const k=(j=n==null?void 0:n.root.children)==null?void 0:j.find(_=>{var R;return(R=_.children)==null?void 0:R.some(P=>P.id===N)});t({type:"thread",agentId:k==null?void 0:k.id,threadId:N})},v=async N=>{if(d.trim())try{const k=await fetch("/api/threads",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:d.trim(),created_by_type:"human",created_by_id:"user",target_agent:N})});if(!k.ok){console.error("Failed to create thread:",await k.text());return}const j=await k.json();f(""),c(!1),t({type:"thread",agentId:N,threadId:j.id})}catch(k){console.error("Error creating thread:",k)}},y=()=>{var N,k,j;if(e.type==="overview"&&n)return u.jsx(Kg,{aggregate:n.aggregate,agents:n.root.children||[],onSelectAgent:_=>t({type:"agent",agentId:_})});if(e.type==="agent"&&e.agentId){const _=(N=n==null?void 0:n.root.children)==null?void 0:N.find(P=>P.id===e.agentId),R=i.filter(P=>{var z;return(z=_==null?void 0:_.children)==null?void 0:z.some(D=>D.id===P.thread_id)});return u.jsxs("div",{className:"agent-view",children:[u.jsxs("div",{className:"agent-view-header",children:[u.jsx("h2",{children:e.agentId}),u.jsxs("span",{className:"agent-thread-count",children:[((k=_==null?void 0:_.children)==null?void 0:k.length)||0," threads"]})]}),u.jsxs("div",{className:"agent-view-content",children:[u.jsxs("div",{className:"agent-threads",children:[u.jsxs("div",{className:"threads-header",children:[u.jsx("h3",{children:"Threads"}),u.jsx("button",{className:"new-thread-btn",onClick:()=>c(!0),title:"New thread",children:"+ New Thread"})]}),s&&u.jsxs("div",{className:"new-thread-form",children:[u.jsx("input",{type:"text",value:d,onChange:P=>f(P.target.value),onKeyDown:P=>{P.key==="Enter"&&v(e.agentId),P.key==="Escape"&&(c(!1),f(""))},placeholder:"Thread title...",autoFocus:!0}),u.jsxs("div",{className:"form-actions",children:[u.jsx("button",{onClick:()=>{c(!1),f("")},children:"Cancel"}),u.jsx("button",{className:"create-btn",onClick:()=>v(e.agentId),children:"Create"})]})]}),(j=_==null?void 0:_.children)==null?void 0:j.map(P=>u.jsxs("div",{className:"thread-card",onClick:()=>t({type:"thread",agentId:e.agentId,threadId:P.id}),children:[u.jsx("span",{className:"thread-title",children:P.label}),P.badges&&P.badges.length>0&&u.jsx("span",{className:"thread-badges",children:P.badges.map((z,D)=>u.jsx("span",{className:`badge badge-${z.type}`,children:z.count},D))})]},P.id)),(!(_!=null&&_.children)||_.children.length===0)&&!s&&u.jsxs("div",{className:"no-threads",children:["No threads yet",u.jsx("button",{className:"start-thread-btn",onClick:()=>c(!0),children:"Start a conversation"})]})]}),R.length>0&&u.jsxs("div",{className:"agent-approvals",children:[u.jsx("h3",{children:"Pending Approvals"}),u.jsx(q0,{approvals:R,history:[],onApprove:w,onReject:S,onNavigateToThread:h})]})]})]})}return e.type==="thread"&&e.threadId?u.jsx(Q0,{websocketUrl:p,instanceId:e.agentId||"default",initialThreadId:e.threadId,onThreadNavigated:()=>{}}):u.jsx("div",{className:"empty-state",children:u.jsx("p",{children:"Select an agent or thread from the sidebar"})})},C=(i==null?void 0:i.filter(N=>N.status==="pending").length)||0;return u.jsxs("div",{className:"app",children:[u.jsxs("header",{className:"app-header",children:[u.jsxs("div",{className:"header-brand",children:[u.jsx("div",{className:"brand-logo",children:nk}),u.jsxs("div",{className:"brand-text",children:[u.jsx("h1",{children:"AILANG"}),u.jsx("span",{className:"brand-subtitle",children:"Collaboration Hub"})]})]}),u.jsxs("div",{className:"header-meta",children:[u.jsx(tk,{}),C>0&&u.jsxs("span",{className:"pending-badge",title:`${C} pending approvals`,children:[C," pending"]}),u.jsx("span",{className:"version-tag",children:"v0.5.0"})]})]}),u.jsxs("div",{className:"app-body",children:[u.jsx("aside",{className:"app-sidebar",children:u.jsx(Eg,{selection:e,onSelect:t})}),u.jsxs("main",{className:"app-main",children:[e.type!=="overview"&&u.jsx(Yg,{items:I()}),u.jsx("div",{className:"main-content",children:y()})]})]}),u.jsx("style",{children:`
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
      `})]})};_o.createRoot(document.getElementById("root")).render(u.jsx(qt.StrictMode,{children:u.jsx(rk,{})}));
