"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[961],{3905:(e,t,n)=>{n.d(t,{Zo:()=>s,kt:()=>m});var a=n(7294);function o(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function r(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);t&&(a=a.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,a)}return n}function i(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?r(Object(n),!0).forEach((function(t){o(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):r(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function l(e,t){if(null==e)return{};var n,a,o=function(e,t){if(null==e)return{};var n,a,o={},r=Object.keys(e);for(a=0;a<r.length;a++)n=r[a],t.indexOf(n)>=0||(o[n]=e[n]);return o}(e,t);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);for(a=0;a<r.length;a++)n=r[a],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(o[n]=e[n])}return o}var d=a.createContext({}),c=function(e){var t=a.useContext(d),n=t;return e&&(n="function"==typeof e?e(t):i(i({},t),e)),n},s=function(e){var t=c(e.components);return a.createElement(d.Provider,{value:t},e.children)},p={inlineCode:"code",wrapper:function(e){var t=e.children;return a.createElement(a.Fragment,{},t)}},u=a.forwardRef((function(e,t){var n=e.components,o=e.mdxType,r=e.originalType,d=e.parentName,s=l(e,["components","mdxType","originalType","parentName"]),u=c(n),m=o,f=u["".concat(d,".").concat(m)]||u[m]||p[m]||r;return n?a.createElement(f,i(i({ref:t},s),{},{components:n})):a.createElement(f,i({ref:t},s))}));function m(e,t){var n=arguments,o=t&&t.mdxType;if("string"==typeof e||o){var r=n.length,i=new Array(r);i[0]=u;var l={};for(var d in t)hasOwnProperty.call(t,d)&&(l[d]=t[d]);l.originalType=e,l.mdxType="string"==typeof e?e:o,i[1]=l;for(var c=2;c<r;c++)i[c]=n[c];return a.createElement.apply(null,i)}return a.createElement.apply(null,n)}u.displayName="MDXCreateElement"},2123:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>d,contentTitle:()=>i,default:()=>p,frontMatter:()=>r,metadata:()=>l,toc:()=>c});var a=n(7462),o=(n(7294),n(3905));const r={sidebar_position:4},i="Download configuration",l={unversionedId:"commands/downloading-configuration",id:"commands/downloading-configuration",title:"Download configuration",description:"This feature lets you download the configuration from a Dynatrace tenant as Monaco files.",source:"@site/docs/commands/downloading-configuration.md",sourceDirName:"commands",slug:"/commands/downloading-configuration",permalink:"/dynatrace-configuration-as-code/next/commands/downloading-configuration",draft:!1,editUrl:"https://github.com/dynatrace/dynatrace-configuration-as-code/edit/main/documentation/docs/commands/downloading-configuration.md",tags:[],version:"current",sidebarPosition:4,frontMatter:{sidebar_position:4},sidebar:"tutorialSidebar",previous:{title:"Experimental New CLI",permalink:"/dynatrace-configuration-as-code/next/commands/experimental-new-cli"},next:{title:"Logging",permalink:"/dynatrace-configuration-as-code/next/commands/logging"}},d={},c=[{value:"Download configurations",id:"download-configurations",level:2},{value:"Options",id:"options",level:2},{value:"Notes",id:"notes",level:2}],s={toc:c};function p(e){let{components:t,...n}=e;return(0,o.kt)("wrapper",(0,a.Z)({},s,n,{components:t,mdxType:"MDXLayout"}),(0,o.kt)("h1",{id:"download-configuration"},"Download configuration"),(0,o.kt)("p",null,"This feature lets you download the configuration from a Dynatrace tenant as Monaco files.\nUse this feature to avoid starting from scratch when using Monaco. "),(0,o.kt)("blockquote",null,(0,o.kt)("p",{parentName:"blockquote"},"\u26a0\ufe0f This feature requires CLI version 2.0.")),(0,o.kt)("h2",{id:"download-configurations"},"Download configurations"),(0,o.kt)("ol",null,(0,o.kt)("li",{parentName:"ol"},"Enable CLI 2.0 by adding an environment variable called NEW_CLI with a non-empty value other than 0. ")),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-shell"},"export NEW_CLI=1\n")),(0,o.kt)("ol",{start:2},(0,o.kt)("li",{parentName:"ol"},"Create an environment file."),(0,o.kt)("li",{parentName:"ol"},"Run monaco using the download command")),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-shell"},"monaco download --environments=my-environment.yaml\n")),(0,o.kt)("h2",{id:"options"},"Options"),(0,o.kt)("p",null,"To download specific APIs only, use ",(0,o.kt)("inlineCode",{parentName:"p"},"--specific-api")," to pass a list of API values separated by a comma. "),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-shell"},"monaco download --specific-api alerting-profile,dashboard --environments=my-environment.yaml\n")),(0,o.kt)("h1",{id:"skipped-configurations"},"Skipped configurations"),(0,o.kt)("p",null,"Some configurations on Dynatrace are predefined and must not be downloaded and reuploaded on the same, or another environment."),(0,o.kt)("p",null,"Those skipped configurations are:"),(0,o.kt)("table",null,(0,o.kt)("thead",{parentName:"table"},(0,o.kt)("tr",{parentName:"thead"},(0,o.kt)("th",{parentName:"tr",align:null},"API"),(0,o.kt)("th",{parentName:"tr",align:null},"Details"))),(0,o.kt)("tbody",{parentName:"table"},(0,o.kt)("tr",{parentName:"tbody"},(0,o.kt)("td",{parentName:"tr",align:null},(0,o.kt)("inlineCode",{parentName:"td"},"dashboard"),"/",(0,o.kt)("inlineCode",{parentName:"td"},"dashboard-v2")),(0,o.kt)("td",{parentName:"tr",align:null},"Dashboards that have the owner ",(0,o.kt)("inlineCode",{parentName:"td"},"Dynatrace")," set, or dashboards, which are a preset")),(0,o.kt)("tr",{parentName:"tbody"},(0,o.kt)("td",{parentName:"tr",align:null},(0,o.kt)("inlineCode",{parentName:"td"},"anomaly-detection-metrics")),(0,o.kt)("td",{parentName:"tr",align:null},"Configurations that start with ",(0,o.kt)("inlineCode",{parentName:"td"},"dynatrace.")," or ",(0,o.kt)("inlineCode",{parentName:"td"},"ruxit."))),(0,o.kt)("tr",{parentName:"tbody"},(0,o.kt)("td",{parentName:"tr",align:null},(0,o.kt)("inlineCode",{parentName:"td"},"synthetic-locations")),(0,o.kt)("td",{parentName:"tr",align:null},(0,o.kt)("inlineCode",{parentName:"td"},"public")," (Dynatrace provided) locations")),(0,o.kt)("tr",{parentName:"tbody"},(0,o.kt)("td",{parentName:"tr",align:null},(0,o.kt)("inlineCode",{parentName:"td"},"extension")),(0,o.kt)("td",{parentName:"tr",align:null},"Extensions with an id that starts with ",(0,o.kt)("inlineCode",{parentName:"td"},"custom."))))),(0,o.kt)("h2",{id:"notes"},"Notes"),(0,o.kt)("blockquote",null,(0,o.kt)("p",{parentName:"blockquote"},"\u26a0\ufe0f ",(0,o.kt)("strong",{parentName:"p"},"Application Detection Rules.")," When using download functionality, you can only update existing application detection rules. You can only create a new app detection rule if no other app detection rules exist for that application.")))}p.isMDXComponent=!0}}]);