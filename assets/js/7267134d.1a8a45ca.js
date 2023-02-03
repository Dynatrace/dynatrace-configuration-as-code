"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[8634],{3905:(e,t,n)=>{n.d(t,{Zo:()=>s,kt:()=>p});var o=n(7294);function r(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function i(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);t&&(o=o.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,o)}return n}function a(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?i(Object(n),!0).forEach((function(t){r(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):i(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function c(e,t){if(null==e)return{};var n,o,r=function(e,t){if(null==e)return{};var n,o,r={},i=Object.keys(e);for(o=0;o<i.length;o++)n=i[o],t.indexOf(n)>=0||(r[n]=e[n]);return r}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(o=0;o<i.length;o++)n=i[o],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(r[n]=e[n])}return r}var l=o.createContext({}),u=function(e){var t=o.useContext(l),n=t;return e&&(n="function"==typeof e?e(t):a(a({},t),e)),n},s=function(e){var t=u(e.components);return o.createElement(l.Provider,{value:t},e.children)},f={inlineCode:"code",wrapper:function(e){var t=e.children;return o.createElement(o.Fragment,{},t)}},d=o.forwardRef((function(e,t){var n=e.components,r=e.mdxType,i=e.originalType,l=e.parentName,s=c(e,["components","mdxType","originalType","parentName"]),d=u(n),p=r,g=d["".concat(l,".").concat(p)]||d[p]||f[p]||i;return n?o.createElement(g,a(a({ref:t},s),{},{components:n})):o.createElement(g,a({ref:t},s))}));function p(e,t){var n=arguments,r=t&&t.mdxType;if("string"==typeof e||r){var i=n.length,a=new Array(i);a[0]=d;var c={};for(var l in t)hasOwnProperty.call(t,l)&&(c[l]=t[l]);c.originalType=e,c.mdxType="string"==typeof e?e:r,a[1]=c;for(var u=2;u<i;u++)a[u]=n[u];return o.createElement.apply(null,a)}return o.createElement.apply(null,n)}d.displayName="MDXCreateElement"},4672:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>a,default:()=>f,frontMatter:()=>i,metadata:()=>c,toc:()=>u});var o=n(7462),r=(n(7294),n(3905));const i={sidebar_position:6},a="Delete configuration",c={unversionedId:"configuration/delete_config",id:"version-1.7.0/configuration/delete_config",title:"Delete configuration",description:"This guide shows you how to use the delete configuration tool to delete a configuration that is not needed.",source:"@site/versioned_docs/version-1.7.0/configuration/delete_config.md",sourceDirName:"configuration",slug:"/configuration/delete_config",permalink:"/dynatrace-configuration-as-code/1.7.0/configuration/delete_config",draft:!1,editUrl:"https://github.com/dynatrace/dynatrace-configuration-as-code/edit/main/documentation/versioned_docs/version-1.7.0/configuration/delete_config.md",tags:[],version:"1.7.0",sidebarPosition:6,frontMatter:{sidebar_position:6},sidebar:"version-1.7.0/tutorialSidebar",previous:{title:"Plugin configuration",permalink:"/dynatrace-configuration-as-code/1.7.0/configuration/plugin_config"},next:{title:"Configuration types and token permissions",permalink:"/dynatrace-configuration-as-code/1.7.0/configuration/configTypes_tokenPermissions"}},l={},u=[{value:"File structure",id:"file-structure",level:2}],s={toc:u};function f(e){let{components:t,...n}=e;return(0,r.kt)("wrapper",(0,o.Z)({},s,n,{components:t,mdxType:"MDXLayout"}),(0,r.kt)("h1",{id:"delete-configuration"},"Delete configuration"),(0,r.kt)("p",null,"This guide shows you how to use the delete configuration tool to delete a configuration that is not needed."),(0,r.kt)("p",null,"The delete configuration tool looks for a ",(0,r.kt)("inlineCode",{parentName:"p"},"delete.yaml")," file located in the project's root folder and deletes all configurations defined in this file after finishing deployment."),(0,r.kt)("h2",{id:"file-structure"},"File structure"),(0,r.kt)("p",null,"The ",(0,r.kt)("inlineCode",{parentName:"p"},"delete.yaml")," file structure should be as follows.  "),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},'delete:\n  - "auto-tag/my-tag"\n  - "custom-service-java/my custom service"\n...\n')),(0,r.kt)("p",null,"You must specify the API and the ",(0,r.kt)("inlineCode",{parentName:"p"},"name")," (not id) of the configuration to be deleted."),(0,r.kt)("blockquote",null,(0,r.kt)("p",{parentName:"blockquote"},"\u26a0\ufe0f if the same name is used for the new config and the config defined in delete.yaml, then the config will be deleted right after deployment.")))}f.isMDXComponent=!0}}]);