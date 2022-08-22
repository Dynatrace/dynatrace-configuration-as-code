"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[6677],{3905:(e,t,n)=>{n.d(t,{Zo:()=>l,kt:()=>d});var o=n(7294);function r(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function i(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);t&&(o=o.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,o)}return n}function a(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?i(Object(n),!0).forEach((function(t){r(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):i(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function c(e,t){if(null==e)return{};var n,o,r=function(e,t){if(null==e)return{};var n,o,r={},i=Object.keys(e);for(o=0;o<i.length;o++)n=i[o],t.indexOf(n)>=0||(r[n]=e[n]);return r}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(o=0;o<i.length;o++)n=i[o],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(r[n]=e[n])}return r}var p=o.createContext({}),s=function(e){var t=o.useContext(p),n=t;return e&&(n="function"==typeof e?e(t):a(a({},t),e)),n},l=function(e){var t=s(e.components);return o.createElement(p.Provider,{value:t},e.children)},u={inlineCode:"code",wrapper:function(e){var t=e.children;return o.createElement(o.Fragment,{},t)}},m=o.forwardRef((function(e,t){var n=e.components,r=e.mdxType,i=e.originalType,p=e.parentName,l=c(e,["components","mdxType","originalType","parentName"]),m=s(n),d=r,f=m["".concat(p,".").concat(d)]||m[d]||u[d]||i;return n?o.createElement(f,a(a({ref:t},l),{},{components:n})):o.createElement(f,a({ref:t},l))}));function d(e,t){var n=arguments,r=t&&t.mdxType;if("string"==typeof e||r){var i=n.length,a=new Array(i);a[0]=m;var c={};for(var p in t)hasOwnProperty.call(t,p)&&(c[p]=t[p]);c.originalType=e,c.mdxType="string"==typeof e?e:r,a[1]=c;for(var s=2;s<i;s++)a[s]=n[s];return o.createElement.apply(null,a)}return o.createElement.apply(null,n)}m.displayName="MDXCreateElement"},7503:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>p,contentTitle:()=>a,default:()=>u,frontMatter:()=>i,metadata:()=>c,toc:()=>s});var o=n(7462),r=(n(7294),n(3905));const i={sidebar_position:5},a="Plugin configuration",c={unversionedId:"configuration/plugin_config",id:"configuration/plugin_config",title:"Plugin configuration",description:"Important",source:"@site/docs/configuration/plugin_config.md",sourceDirName:"configuration",slug:"/configuration/plugin_config",permalink:"/dynatrace-monitoring-as-code/next/configuration/plugin_config",draft:!1,editUrl:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/docs/configuration/plugin_config.md",tags:[],version:"current",sidebarPosition:5,frontMatter:{sidebar_position:5},sidebar:"tutorialSidebar",previous:{title:"Configuration YAML structure",permalink:"/dynatrace-monitoring-as-code/next/configuration/yaml_config"},next:{title:"Delete configuration",permalink:"/dynatrace-monitoring-as-code/next/configuration/delete_config"}},p={},s=[{value:"Custom Extensions",id:"custom-extensions",level:3}],l={toc:s};function u(e){let{components:t,...n}=e;return(0,r.kt)("wrapper",(0,o.Z)({},l,n,{components:t,mdxType:"MDXLayout"}),(0,r.kt)("h1",{id:"plugin-configuration"},"Plugin configuration"),(0,r.kt)("blockquote",null,(0,r.kt)("p",{parentName:"blockquote"},"\u26a0\ufe0f ",(0,r.kt)("strong",{parentName:"p"},"Important")),(0,r.kt)("p",{parentName:"blockquote"},"If you define something that depends on a metric created by a plugin, make sure to reference the plugin by name, so that the configurations will be applied in the correct order (after the plugin is created)"),(0,r.kt)("p",{parentName:"blockquote"},"Plugins can not be referenced by ",(0,r.kt)("inlineCode",{parentName:"p"},"id")," as the Dynatrace plugin endpoint does not return this!"),(0,r.kt)("p",{parentName:"blockquote"},"Use only the plugin ",(0,r.kt)("inlineCode",{parentName:"p"},"name"))),(0,r.kt)("p",null,"e.g. ",(0,r.kt)("inlineCode",{parentName:"p"},"projects/example-project/anomaly-detection-metrics/numberOfDistributionInProgressAlert.json")," depends on the plugin defined by ",(0,r.kt)("inlineCode",{parentName:"p"},"projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.json")),(0,r.kt)("p",null,"So ",(0,r.kt)("inlineCode",{parentName:"p"},"projects/example-project/anomaly-detection-metrics/example-anomaly.yaml")," references the plugin by name in a variable:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},'- metricPrefix: "projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.name"\n')),(0,r.kt)("p",null,"to then construct the ",(0,r.kt)("inlineCode",{parentName:"p"},"metric-id")," in the ",(0,r.kt)("inlineCode",{parentName:"p"},"json")," as:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-json"},'"metricId": "ext:{{.metricPrefix}}.metric_NumberOfDistributionInProgressRequests"\n')),(0,r.kt)("h3",{id:"custom-extensions"},"Custom Extensions"),(0,r.kt)("p",null,"Monaco can deploy custom extensions and handles the zipping of extensions; as such, the JSON file that defines an extension can just be checked in. The version of the extension is checked before it is uploaded. If the version of the extension to be uploaded is the same or equal to what is already deployed, then the upload is skipped.\nAn example of a custom extension can be found ",(0,r.kt)("a",{parentName:"p",href:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/tree/main/cmd/monaco/test-resources/integration-all-configs/project/extension"},"here"),"."))}u.isMDXComponent=!0}}]);