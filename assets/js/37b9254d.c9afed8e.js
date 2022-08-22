"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[9586],{3905:(e,n,t)=>{t.d(n,{Zo:()=>d,kt:()=>m});var r=t(7294);function o(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function a(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);n&&(r=r.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,r)}return t}function i(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?a(Object(t),!0).forEach((function(n){o(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):a(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function c(e,n){if(null==e)return{};var t,r,o=function(e,n){if(null==e)return{};var t,r,o={},a=Object.keys(e);for(r=0;r<a.length;r++)t=a[r],n.indexOf(t)>=0||(o[t]=e[t]);return o}(e,n);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(r=0;r<a.length;r++)t=a[r],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var l=r.createContext({}),s=function(e){var n=r.useContext(l),t=n;return e&&(t="function"==typeof e?e(n):i(i({},n),e)),t},d=function(e){var n=s(e.components);return r.createElement(l.Provider,{value:n},e.children)},u={inlineCode:"code",wrapper:function(e){var n=e.children;return r.createElement(r.Fragment,{},n)}},p=r.forwardRef((function(e,n){var t=e.components,o=e.mdxType,a=e.originalType,l=e.parentName,d=c(e,["components","mdxType","originalType","parentName"]),p=s(t),m=o,f=p["".concat(l,".").concat(m)]||p[m]||u[m]||a;return t?r.createElement(f,i(i({ref:n},d),{},{components:t})):r.createElement(f,i({ref:n},d))}));function m(e,n){var t=arguments,o=n&&n.mdxType;if("string"==typeof e||o){var a=t.length,i=new Array(a);i[0]=p;var c={};for(var l in n)hasOwnProperty.call(n,l)&&(c[l]=n[l]);c.originalType=e,c.mdxType="string"==typeof e?e:o,i[1]=c;for(var s=2;s<a;s++)i[s]=t[s];return r.createElement.apply(null,i)}return r.createElement.apply(null,t)}p.displayName="MDXCreateElement"},2138:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>l,contentTitle:()=>i,default:()=>u,frontMatter:()=>a,metadata:()=>c,toc:()=>s});var r=t(7462),o=(t(7294),t(3905));const a={sidebar_position:1},i="Validate configuration",c={unversionedId:"commands/validating-configuration",id:"version-1.7.0/commands/validating-configuration",title:"Validate configuration",description:"Monaco validates configuration files in a directory by performing a dry run. It will check whether your Dynatrace config files are in a valid JSON format, and whether your tool configuration YAML files can be parsed and used.",source:"@site/versioned_docs/version-1.7.0/commands/validating-configuration.md",sourceDirName:"commands",slug:"/commands/validating-configuration",permalink:"/dynatrace-monitoring-as-code/1.7.0/commands/validating-configuration",draft:!1,editUrl:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/versioned_docs/version-1.7.0/commands/validating-configuration.md",tags:[],version:"1.7.0",sidebarPosition:1,frontMatter:{sidebar_position:1},sidebar:"version-1.7.0/tutorialSidebar",previous:{title:"Install Monaco",permalink:"/dynatrace-monitoring-as-code/1.7.0/Get-started/installation"},next:{title:"Deploy projects",permalink:"/dynatrace-monitoring-as-code/1.7.0/commands/deploying-projects"}},l={},s=[],d={toc:s};function u(e){let{components:n,...t}=e;return(0,o.kt)("wrapper",(0,r.Z)({},d,t,{components:n,mdxType:"MDXLayout"}),(0,o.kt)("h1",{id:"validate-configuration"},"Validate configuration"),(0,o.kt)("p",null,"Monaco validates configuration files in a directory by performing a dry run. It will check whether your Dynatrace config files are in a valid JSON format, and whether your tool configuration YAML files can be parsed and used."),(0,o.kt)("p",null,"To validate the configuration, execute a ",(0,o.kt)("inlineCode",{parentName:"p"},"monaco dry run")," on a YAML file as shown below."),(0,o.kt)("p",null,"Create the file at ",(0,o.kt)("inlineCode",{parentName:"p"},"project/sub-project/my-environments.yaml"),":"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-jsx",metastring:'title="run monaco in dry mode"',title:'"run',monaco:!0,in:!0,dry:!0,'mode"':!0}," ./monaco -dry-run --environments=project/sub-project/my-environments.yaml\n2020/06/16 16:22:30 monaco v1.0.0\n2020/06/16 16:22:30 Reading projects...\n2020/06/16 16:22:30 Sorting projects...\n...\n2020/06/16 16:22:30 Config validation SUCCESSFUL\n")))}u.isMDXComponent=!0}}]);