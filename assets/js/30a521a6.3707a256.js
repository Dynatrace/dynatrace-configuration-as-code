"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[1522],{3905:(e,n,t)=>{t.d(n,{Zo:()=>s,kt:()=>f});var r=t(7294);function o(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function i(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);n&&(r=r.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,r)}return t}function a(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?i(Object(t),!0).forEach((function(n){o(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):i(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function c(e,n){if(null==e)return{};var t,r,o=function(e,n){if(null==e)return{};var t,r,o={},i=Object.keys(e);for(r=0;r<i.length;r++)t=i[r],n.indexOf(t)>=0||(o[t]=e[t]);return o}(e,n);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(r=0;r<i.length;r++)t=i[r],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var p=r.createContext({}),l=function(e){var n=r.useContext(p),t=n;return e&&(t="function"==typeof e?e(n):a(a({},n),e)),t},s=function(e){var n=l(e.components);return r.createElement(p.Provider,{value:n},e.children)},u={inlineCode:"code",wrapper:function(e){var n=e.children;return r.createElement(r.Fragment,{},n)}},m=r.forwardRef((function(e,n){var t=e.components,o=e.mdxType,i=e.originalType,p=e.parentName,s=c(e,["components","mdxType","originalType","parentName"]),m=l(t),f=o,d=m["".concat(p,".").concat(f)]||m[f]||u[f]||i;return t?r.createElement(d,a(a({ref:n},s),{},{components:t})):r.createElement(d,a({ref:n},s))}));function f(e,n){var t=arguments,o=n&&n.mdxType;if("string"==typeof e||o){var i=t.length,a=new Array(i);a[0]=m;var c={};for(var p in n)hasOwnProperty.call(n,p)&&(c[p]=n[p]);c.originalType=e,c.mdxType="string"==typeof e?e:o,a[1]=c;for(var l=2;l<i;l++)a[l]=t[l];return r.createElement.apply(null,a)}return r.createElement.apply(null,t)}m.displayName="MDXCreateElement"},9026:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>p,contentTitle:()=>a,default:()=>u,frontMatter:()=>i,metadata:()=>c,toc:()=>l});var r=t(7462),o=(t(7294),t(3905));const i={sidebar_position:5},a="Plugin Configuration",c={unversionedId:"configuration/plugin_config",id:"version-1.5.3/configuration/plugin_config",title:"Plugin Configuration",description:"Important",source:"@site/versioned_docs/version-1.5.3/configuration/plugin_config.md",sourceDirName:"configuration",slug:"/configuration/plugin_config",permalink:"/dynatrace-configuration-as-code/1.5.3/configuration/plugin_config",draft:!1,editUrl:"https://github.com/dynatrace/dynatrace-configuration-as-code/edit/main/documentation/versioned_docs/version-1.5.3/configuration/plugin_config.md",tags:[],version:"1.5.3",sidebarPosition:5,frontMatter:{sidebar_position:5},sidebar:"version-1.5.3/tutorialSidebar",previous:{title:"Configuration YAML Structure",permalink:"/dynatrace-configuration-as-code/1.5.3/configuration/yaml_confg"},next:{title:"Delete Configuration",permalink:"/dynatrace-configuration-as-code/1.5.3/configuration/delete_config"}},p={},l=[{value:"Custom Extensions",id:"custom-extensions",level:3}],s={toc:l};function u(e){let{components:n,...t}=e;return(0,o.kt)("wrapper",(0,r.Z)({},s,t,{components:n,mdxType:"MDXLayout"}),(0,o.kt)("h1",{id:"plugin-configuration"},"Plugin Configuration"),(0,o.kt)("blockquote",null,(0,o.kt)("p",{parentName:"blockquote"},(0,o.kt)("strong",{parentName:"p"},"Important")),(0,o.kt)("p",{parentName:"blockquote"},"If you define something that depends on a metric created by a plugin, make sure to reference the plugin by name, so that the configurations will be applied in the correct order (after the plugin was created)"),(0,o.kt)("p",{parentName:"blockquote"},"Plugins can not be referenced by ",(0,o.kt)("inlineCode",{parentName:"p"},"id")," as the Dynatrace plugin endpoint does not return this!"),(0,o.kt)("p",{parentName:"blockquote"},"Use only the plugin ",(0,o.kt)("inlineCode",{parentName:"p"},"name"))),(0,o.kt)("p",null,"e.g. ",(0,o.kt)("inlineCode",{parentName:"p"},"projects/example-project/anomaly-detection-metrics/numberOfDistributionInProgressAlert.json")," depends on the plugin defined by ",(0,o.kt)("inlineCode",{parentName:"p"},"projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.json")),(0,o.kt)("p",null,"So ",(0,o.kt)("inlineCode",{parentName:"p"},"projects/example-project/anomaly-detection-metrics/example-anomaly.yaml")," references the plugin by name in a variable:"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-yaml"},'- metricPrefix: "projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.name"\n')),(0,o.kt)("p",null,"to then construct the ",(0,o.kt)("inlineCode",{parentName:"p"},"metric-id")," in the ",(0,o.kt)("inlineCode",{parentName:"p"},"json")," as:"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-json"},'"metricId": "ext:{{.metricPrefix}}.metric_NumberOfDistributionInProgressRequests"\n')),(0,o.kt)("h3",{id:"custom-extensions"},"Custom Extensions"),(0,o.kt)("p",null,"Monaco is able to deploy custom extensions and handles the zipping of extensions, as such the JSON file that defines an extension can just be checked in.\nAn example of a custom extension can be found ",(0,o.kt)("a",{parentName:"p",href:"https://github.com/dynatrace/dynatrace-configuration-as-code/tree/main/cmd/monaco/test-resources/integration-all-configs/project/extension"},"here"),"."))}u.isMDXComponent=!0}}]);