"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[2955],{3905:function(e,n,t){t.d(n,{Zo:function(){return u},kt:function(){return f}});var r=t(7294);function o(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function i(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);n&&(r=r.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,r)}return t}function a(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?i(Object(t),!0).forEach((function(n){o(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):i(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function c(e,n){if(null==e)return{};var t,r,o=function(e,n){if(null==e)return{};var t,r,o={},i=Object.keys(e);for(r=0;r<i.length;r++)t=i[r],n.indexOf(t)>=0||(o[t]=e[t]);return o}(e,n);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(r=0;r<i.length;r++)t=i[r],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var p=r.createContext({}),s=function(e){var n=r.useContext(p),t=n;return e&&(t="function"==typeof e?e(n):a(a({},n),e)),t},u=function(e){var n=s(e.components);return r.createElement(p.Provider,{value:n},e.children)},l={inlineCode:"code",wrapper:function(e){var n=e.children;return r.createElement(r.Fragment,{},n)}},m=r.forwardRef((function(e,n){var t=e.components,o=e.mdxType,i=e.originalType,p=e.parentName,u=c(e,["components","mdxType","originalType","parentName"]),m=s(t),f=o,d=m["".concat(p,".").concat(f)]||m[f]||l[f]||i;return t?r.createElement(d,a(a({ref:n},u),{},{components:t})):r.createElement(d,a({ref:n},u))}));function f(e,n){var t=arguments,o=n&&n.mdxType;if("string"==typeof e||o){var i=t.length,a=new Array(i);a[0]=m;var c={};for(var p in n)hasOwnProperty.call(n,p)&&(c[p]=n[p]);c.originalType=e,c.mdxType="string"==typeof e?e:o,a[1]=c;for(var s=2;s<i;s++)a[s]=t[s];return r.createElement.apply(null,a)}return r.createElement.apply(null,t)}m.displayName="MDXCreateElement"},408:function(e,n,t){t.r(n),t.d(n,{frontMatter:function(){return c},metadata:function(){return p},toc:function(){return s},default:function(){return l}});var r=t(7462),o=t(3366),i=(t(7294),t(3905)),a=["components"],c={sidebar_position:5},p={unversionedId:"configuration/plugin_config",id:"version-1.6.0/configuration/plugin_config",isDocsHomePage:!1,title:"Plugin Configuration",description:"Important",source:"@site/versioned_docs/version-1.6.0/configuration/plugin_config.md",sourceDirName:"configuration",slug:"/configuration/plugin_config",permalink:"/dynatrace-monitoring-as-code/1.6.0/configuration/plugin_config",editUrl:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/versioned_docs/version-1.6.0/configuration/plugin_config.md",version:"1.6.0",sidebarPosition:5,frontMatter:{sidebar_position:5},sidebar:"version-1.6.0/tutorialSidebar",previous:{title:"Configuration YAML Structure",permalink:"/dynatrace-monitoring-as-code/1.6.0/configuration/yaml_confg"},next:{title:"Delete Configuration",permalink:"/dynatrace-monitoring-as-code/1.6.0/configuration/delete_config"}},s=[{value:"Custom Extensions",id:"custom-extensions",children:[]}],u={toc:s};function l(e){var n=e.components,t=(0,o.Z)(e,a);return(0,i.kt)("wrapper",(0,r.Z)({},u,t,{components:n,mdxType:"MDXLayout"}),(0,i.kt)("blockquote",null,(0,i.kt)("p",{parentName:"blockquote"},(0,i.kt)("strong",{parentName:"p"},"Important")),(0,i.kt)("p",{parentName:"blockquote"},"If you define something that depends on a metric created by a plugin, make sure to reference the plugin by name, so that the configurations will be applied in the correct order (after the plugin was created)"),(0,i.kt)("p",{parentName:"blockquote"},"Plugins can not be referenced by ",(0,i.kt)("inlineCode",{parentName:"p"},"id")," as the Dynatrace plugin endpoint does not return this!"),(0,i.kt)("p",{parentName:"blockquote"},"Use only the plugin ",(0,i.kt)("inlineCode",{parentName:"p"},"name"))),(0,i.kt)("p",null,"e.g. ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/example-project/anomaly-detection-metrics/numberOfDistributionInProgressAlert.json")," depends on the plugin defined by ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.json")),(0,i.kt)("p",null,"So ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/example-project/anomaly-detection-metrics/example-anomaly.yaml")," references the plugin by name in a variable:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'- metricPrefix: "projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.name"\n')),(0,i.kt)("p",null,"to then construct the ",(0,i.kt)("inlineCode",{parentName:"p"},"metric-id")," in the ",(0,i.kt)("inlineCode",{parentName:"p"},"json")," as:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-json"},'"metricId": "ext:{{.metricPrefix}}.metric_NumberOfDistributionInProgressRequests"\n')),(0,i.kt)("h3",{id:"custom-extensions"},"Custom Extensions"),(0,i.kt)("p",null,"Monaco is able to deploy custom extensions and handles the zipping of extensions, as such the JSON file that defines an extension can just be checked in.\nAn example of a custom extension can be found ",(0,i.kt)("a",{parentName:"p",href:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/tree/main/cmd/monaco/test-resources/integration-all-configs/project/extension"},"here"),"."))}l.isMDXComponent=!0}}]);