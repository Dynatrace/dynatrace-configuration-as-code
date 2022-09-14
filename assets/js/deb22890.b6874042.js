"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[4],{3905:(e,n,t)=>{t.d(n,{Zo:()=>u,kt:()=>f});var o=t(7294);function i(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function r(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);n&&(o=o.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,o)}return t}function a(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?r(Object(t),!0).forEach((function(n){i(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):r(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function s(e,n){if(null==e)return{};var t,o,i=function(e,n){if(null==e)return{};var t,o,i={},r=Object.keys(e);for(o=0;o<r.length;o++)t=r[o],n.indexOf(t)>=0||(i[t]=e[t]);return i}(e,n);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);for(o=0;o<r.length;o++)t=r[o],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(i[t]=e[t])}return i}var c=o.createContext({}),l=function(e){var n=o.useContext(c),t=n;return e&&(t="function"==typeof e?e(n):a(a({},n),e)),t},u=function(e){var n=l(e.components);return o.createElement(c.Provider,{value:n},e.children)},p={inlineCode:"code",wrapper:function(e){var n=e.children;return o.createElement(o.Fragment,{},n)}},d=o.forwardRef((function(e,n){var t=e.components,i=e.mdxType,r=e.originalType,c=e.parentName,u=s(e,["components","mdxType","originalType","parentName"]),d=l(t),f=i,g=d["".concat(c,".").concat(f)]||d[f]||p[f]||r;return t?o.createElement(g,a(a({ref:n},u),{},{components:t})):o.createElement(g,a({ref:n},u))}));function f(e,n){var t=arguments,i=n&&n.mdxType;if("string"==typeof e||i){var r=t.length,a=new Array(r);a[0]=d;var s={};for(var c in n)hasOwnProperty.call(n,c)&&(s[c]=n[c]);s.originalType=e,s.mdxType="string"==typeof e?e:i,a[1]=s;for(var l=2;l<r;l++)a[l]=t[l];return o.createElement.apply(null,a)}return o.createElement.apply(null,t)}d.displayName="MDXCreateElement"},7202:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>c,contentTitle:()=>a,default:()=>p,frontMatter:()=>r,metadata:()=>s,toc:()=>l});var o=t(7462),i=(t(7294),t(3905));const r={sidebar_position:8},a="Special Types of Configuration",s={unversionedId:"configuration/special_config_types",id:"version-1.8.3/configuration/special_config_types",title:"Special Types of Configuration",description:"Most types of configurations behave the same and entirely as described in other sections of this documentation.",source:"@site/versioned_docs/version-1.8.3/configuration/special_config_types.md",sourceDirName:"configuration",slug:"/configuration/special_config_types",permalink:"/dynatrace-monitoring-as-code/configuration/special_config_types",draft:!1,editUrl:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/versioned_docs/version-1.8.3/configuration/special_config_types.md",tags:[],version:"1.8.3",sidebarPosition:8,frontMatter:{sidebar_position:8},sidebar:"version-1.8.2/tutorialSidebar",previous:{title:"Configuration types and token permissions",permalink:"/dynatrace-monitoring-as-code/configuration/configTypes_tokenPermissions"},next:{title:"Add a new API",permalink:"/dynatrace-monitoring-as-code/Guides/add_new_api"}},c={},l=[{value:"Single Configuration Endpoint",id:"single-configuration-endpoint",level:2},{value:"Non-Unique Name",id:"non-unique-name",level:2}],u={toc:l};function p(e){let{components:n,...t}=e;return(0,i.kt)("wrapper",(0,o.Z)({},u,t,{components:n,mdxType:"MDXLayout"}),(0,i.kt)("h1",{id:"special-types-of-configuration"},"Special Types of Configuration"),(0,i.kt)("p",null,"Most ",(0,i.kt)("a",{parentName:"p",href:"/dynatrace-monitoring-as-code/configuration/configTypes_tokenPermissions"},"types of configurations")," behave the same and entirely as described in other sections of this documentation."),(0,i.kt)("p",null,"However, some have special behavior and constraints as they deal with special Dynatrace APIs - these are described in the following sections."),(0,i.kt)("h2",{id:"single-configuration-endpoint"},"Single Configuration Endpoint"),(0,i.kt)("p",null,"These configurations are global to a Dynatrace environment and only exist once."),(0,i.kt)("p",null,"Unlike other configurations, there is usually some default configuration that the API - or Monaco configuration - allows to update."),(0,i.kt)("p",null,"Be aware that only one such configuration should be present in your Monaco configuration."),(0,i.kt)("p",null,"Having several configurations - for example, in several projects deployed in one run - will result in the last applied one being active on the Dynatrace environment."),(0,i.kt)("h2",{id:"non-unique-name"},"Non-Unique Name"),(0,i.kt)("p",null,'Monaco assumes that the "Name" of a configuration is unique and will use it as the identifier when deciding to create or update a configuration.'),(0,i.kt)("p",null,"This is also the case for most configurations when created in the Dynatrace UI or via API calls."),(0,i.kt)("p",null,'However, some configurations can have overlapping names, which causes issues for Monaco - for example, there can be several Dashboards named "My Dashboard".'),(0,i.kt)("p",null,"If more than one configuration of a name is present, Monaco can not ensure the correct one is updated when searching by name.\nSimilar problems are present when downloading."),(0,i.kt)("p",null,"To work around this, special handling is present for these configuration APIS, ensuring:"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},"they receive a known identifier when originating from Monaco"),(0,i.kt)("li",{parentName:"ul"},"they are stored with their Dynatrace identifier rather than their name when downloading")),(0,i.kt)("p",null,"As this switch from names to IDs results in re-creating configurations if they were previously created by name,\nalready existing configuration types like ",(0,i.kt)("inlineCode",{parentName:"p"},"dashboard")," are retained with the previous flawed handling, while new ",(0,i.kt)("inlineCode",{parentName:"p"},"-v2")," configuration types were added with the non-unique-name constraint/handling."),(0,i.kt)("p",null,"To ensure configurations are correctly updated, please see the manual steps in the ",(0,i.kt)("a",{parentName:"p",href:"/dynatrace-monitoring-as-code/Guides/deprecated_migration"},"Migration Guide")," for how to deal with this."),(0,i.kt)("blockquote",null,(0,i.kt)("p",{parentName:"blockquote"},"NOTE: As the ",(0,i.kt)("inlineCode",{parentName:"p"},"-v2")," naming implies, the previous handling is deprecated and will be dropped in version 2.0.")))}p.isMDXComponent=!0}}]);