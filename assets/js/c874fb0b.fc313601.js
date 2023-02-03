"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[1159],{3905:(e,n,t)=>{t.d(n,{Zo:()=>u,kt:()=>m});var o=t(7294);function a(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function r(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);n&&(o=o.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,o)}return t}function i(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?r(Object(t),!0).forEach((function(n){a(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):r(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function l(e,n){if(null==e)return{};var t,o,a=function(e,n){if(null==e)return{};var t,o,a={},r=Object.keys(e);for(o=0;o<r.length;o++)t=r[o],n.indexOf(t)>=0||(a[t]=e[t]);return a}(e,n);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);for(o=0;o<r.length;o++)t=r[o],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(a[t]=e[t])}return a}var c=o.createContext({}),s=function(e){var n=o.useContext(c),t=n;return e&&(t="function"==typeof e?e(n):i(i({},n),e)),t},u=function(e){var n=s(e.components);return o.createElement(c.Provider,{value:n},e.children)},d={inlineCode:"code",wrapper:function(e){var n=e.children;return o.createElement(o.Fragment,{},n)}},p=o.forwardRef((function(e,n){var t=e.components,a=e.mdxType,r=e.originalType,c=e.parentName,u=l(e,["components","mdxType","originalType","parentName"]),p=s(t),m=a,f=p["".concat(c,".").concat(m)]||p[m]||d[m]||r;return t?o.createElement(f,i(i({ref:n},u),{},{components:t})):o.createElement(f,i({ref:n},u))}));function m(e,n){var t=arguments,a=n&&n.mdxType;if("string"==typeof e||a){var r=t.length,i=new Array(r);i[0]=p;var l={};for(var c in n)hasOwnProperty.call(n,c)&&(l[c]=n[c]);l.originalType=e,l.mdxType="string"==typeof e?e:a,i[1]=l;for(var s=2;s<r;s++)i[s]=t[s];return o.createElement.apply(null,i)}return o.createElement.apply(null,t)}p.displayName="MDXCreateElement"},1392:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>c,contentTitle:()=>i,default:()=>d,frontMatter:()=>r,metadata:()=>l,toc:()=>s});var o=t(7462),a=(t(7294),t(3905));const r={sidebar_position:4},i="Downloading Configuration",l={unversionedId:"commands/downloading-configuration",id:"version-1.6.0/commands/downloading-configuration",title:"Downloading Configuration",description:"This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco. For this feature you will have to enable CLI version 2.0.",source:"@site/versioned_docs/version-1.6.0/commands/downloading-configuration.md",sourceDirName:"commands",slug:"/commands/downloading-configuration",permalink:"/dynatrace-configuration-as-code/1.6.0/commands/downloading-configuration",draft:!1,editUrl:"https://github.com/dynatrace/dynatrace-configuration-as-code/edit/main/documentation/versioned_docs/version-1.6.0/commands/downloading-configuration.md",tags:[],version:"1.6.0",sidebarPosition:4,frontMatter:{sidebar_position:4},sidebar:"version-1.6.0/tutorialSidebar",previous:{title:"Experimental New CLI",permalink:"/dynatrace-configuration-as-code/1.6.0/commands/experimental-new-cli"},next:{title:"Logging",permalink:"/dynatrace-configuration-as-code/1.6.0/commands/Logging"}},c={},s=[{value:"Steps",id:"steps",level:3},{value:"Options",id:"options",level:3},{value:"Notes",id:"notes",level:3},{value:"Application Detection Rules:",id:"application-detection-rules",level:4}],u={toc:s};function d(e){let{components:n,...t}=e;return(0,a.kt)("wrapper",(0,o.Z)({},u,t,{components:n,mdxType:"MDXLayout"}),(0,a.kt)("h1",{id:"downloading-configuration"},"Downloading Configuration"),(0,a.kt)("p",null,"This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco. For this feature you will have to enable CLI version 2.0."),(0,a.kt)("h3",{id:"steps"},"Steps"),(0,a.kt)("ol",null,(0,a.kt)("li",{parentName:"ol"},"Enable CLI version 2.0 by adding an environment variable called NEW_CLI with a non-empty value other than 0. ")),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-shell",metastring:'title="shell"',title:'"shell"'},"\n$ export NEW_CLI=1\n\n")),(0,a.kt)("ol",{start:2},(0,a.kt)("li",{parentName:"ol"},"Create an environment file."),(0,a.kt)("li",{parentName:"ol"},"Run monaco using the download command i.e.")),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-shell",metastring:'title="shell"',title:'"shell"'},"\n$ monaco download --environments=my-environment.yaml\n\n")),(0,a.kt)("h3",{id:"options"},"Options"),(0,a.kt)("p",null,"Instead of downloading all the configurations for all the APIs you can pass a list of API values separated by comma using the following flag ",(0,a.kt)("inlineCode",{parentName:"p"},"--downloadSpecificAPI"),"."),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-shell",metastring:'title="shell"',title:'"shell"'},"\n$ monaco download --downloadSpecificAPI alerting-profile,dashboard --environments=my-environment.yaml\n\n")),(0,a.kt)("h3",{id:"notes"},"Notes"),(0,a.kt)("p",null,"You should take into consideration the following limitations of the current process."),(0,a.kt)("h4",{id:"application-detection-rules"},"Application Detection Rules:"),(0,a.kt)("p",null,"When using download functionality you will only be able to update existing application detection rules. If you want to create a new app detection rule you can only do so if there are no other app detection rules for that application."))}d.isMDXComponent=!0}}]);