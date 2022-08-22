"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[7614],{3905:(e,t,n)=>{n.d(t,{Zo:()=>p,kt:()=>m});var r=n(7294);function o(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function i(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,r)}return n}function a(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?i(Object(n),!0).forEach((function(t){o(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):i(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function c(e,t){if(null==e)return{};var n,r,o=function(e,t){if(null==e)return{};var n,r,o={},i=Object.keys(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||(o[n]=e[n]);return o}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(o[n]=e[n])}return o}var l=r.createContext({}),s=function(e){var t=r.useContext(l),n=t;return e&&(n="function"==typeof e?e(t):a(a({},t),e)),n},p=function(e){var t=s(e.components);return r.createElement(l.Provider,{value:t},e.children)},u={inlineCode:"code",wrapper:function(e){var t=e.children;return r.createElement(r.Fragment,{},t)}},g=r.forwardRef((function(e,t){var n=e.components,o=e.mdxType,i=e.originalType,l=e.parentName,p=c(e,["components","mdxType","originalType","parentName"]),g=s(n),m=o,d=g["".concat(l,".").concat(m)]||g[m]||u[m]||i;return n?r.createElement(d,a(a({ref:t},p),{},{components:n})):r.createElement(d,a({ref:t},p))}));function m(e,t){var n=arguments,o=t&&t.mdxType;if("string"==typeof e||o){var i=n.length,a=new Array(i);a[0]=g;var c={};for(var l in t)hasOwnProperty.call(t,l)&&(c[l]=t[l]);c.originalType=e,c.mdxType="string"==typeof e?e:o,a[1]=c;for(var s=2;s<i;s++)a[s]=n[s];return r.createElement.apply(null,a)}return r.createElement.apply(null,n)}g.displayName="MDXCreateElement"},746:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>a,default:()=>u,frontMatter:()=>i,metadata:()=>c,toc:()=>s});var r=n(7462),o=(n(7294),n(3905));const i={sidebar_position:5},a="Logging",c={unversionedId:"commands/logging",id:"commands/logging",title:"Logging",description:"Use the MONACOREQUESTLOG and MONACORESPONSELOG environment variables to specify a file",source:"@site/docs/commands/logging.md",sourceDirName:"commands",slug:"/commands/logging",permalink:"/dynatrace-monitoring-as-code/next/commands/logging",draft:!1,editUrl:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/docs/commands/logging.md",tags:[],version:"current",sidebarPosition:5,frontMatter:{sidebar_position:5},sidebar:"tutorialSidebar",previous:{title:"Download configuration",permalink:"/dynatrace-monitoring-as-code/next/commands/downloading-configuration"},next:{title:"Deploy configuration",permalink:"/dynatrace-monitoring-as-code/next/configuration/deploy_configuration"}},l={},s=[],p={toc:s};function u(e){let{components:t,...n}=e;return(0,o.kt)("wrapper",(0,r.Z)({},p,n,{components:t,mdxType:"MDXLayout"}),(0,o.kt)("h1",{id:"logging"},"Logging"),(0,o.kt)("p",null,"Use the ",(0,o.kt)("inlineCode",{parentName:"p"},"MONACO_REQUEST_LOG")," and ",(0,o.kt)("inlineCode",{parentName:"p"},"MONACO_RESPONSE_LOG")," environment variables to specify a file\nthat logs the HTTP traffic between Monaco and the Dynatrace API.\nThis is useful while working on Monaco's source code."),(0,o.kt)("p",null,"The path for the specified file can be absolute or relative to the current working directory."),(0,o.kt)("blockquote",null,(0,o.kt)("p",{parentName:"blockquote"},"\u26a0\ufe0f The specified file(s) ",(0,o.kt)("strong",{parentName:"p"},"will be truncated!"),".")),(0,o.kt)("p",null,"To specify the log file, set the environment variables:"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-shell",metastring:'title="Logging monaco requests and responses"',title:'"Logging',monaco:!0,requests:!0,and:!0,'responses"':!0},"MONACO_REQUEST_LOG=request.log MONACO_RESPONSE_LOG=response.log monaco -e environment project\n")),(0,o.kt)("p",null,"Monaco immediately starts writing all send requests to the specified file(s)."),(0,o.kt)("p",null,"The content of multipart post requests is currently not logged. This is a known limitation."))}u.isMDXComponent=!0}}]);