"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[197],{3905:(t,e,n)=>{n.d(e,{Zo:()=>m,kt:()=>k});var a=n(7294);function i(t,e,n){return e in t?Object.defineProperty(t,e,{value:n,enumerable:!0,configurable:!0,writable:!0}):t[e]=n,t}function r(t,e){var n=Object.keys(t);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(t);e&&(a=a.filter((function(e){return Object.getOwnPropertyDescriptor(t,e).enumerable}))),n.push.apply(n,a)}return n}function l(t){for(var e=1;e<arguments.length;e++){var n=null!=arguments[e]?arguments[e]:{};e%2?r(Object(n),!0).forEach((function(e){i(t,e,n[e])})):Object.getOwnPropertyDescriptors?Object.defineProperties(t,Object.getOwnPropertyDescriptors(n)):r(Object(n)).forEach((function(e){Object.defineProperty(t,e,Object.getOwnPropertyDescriptor(n,e))}))}return t}function o(t,e){if(null==t)return{};var n,a,i=function(t,e){if(null==t)return{};var n,a,i={},r=Object.keys(t);for(a=0;a<r.length;a++)n=r[a],e.indexOf(n)>=0||(i[n]=t[n]);return i}(t,e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(t);for(a=0;a<r.length;a++)n=r[a],e.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(t,n)&&(i[n]=t[n])}return i}var d=a.createContext({}),p=function(t){var e=a.useContext(d),n=e;return t&&(n="function"==typeof t?t(e):l(l({},e),t)),n},m=function(t){var e=p(t.components);return a.createElement(d.Provider,{value:e},t.children)},g={inlineCode:"code",wrapper:function(t){var e=t.children;return a.createElement(a.Fragment,{},e)}},u=a.forwardRef((function(t,e){var n=t.components,i=t.mdxType,r=t.originalType,d=t.parentName,m=o(t,["components","mdxType","originalType","parentName"]),u=p(n),k=i,N=u["".concat(d,".").concat(k)]||u[k]||g[k]||r;return n?a.createElement(N,l(l({ref:e},m),{},{components:n})):a.createElement(N,l({ref:e},m))}));function k(t,e){var n=arguments,i=e&&e.mdxType;if("string"==typeof t||i){var r=n.length,l=new Array(r);l[0]=u;var o={};for(var d in e)hasOwnProperty.call(e,d)&&(o[d]=e[d]);o.originalType=t,o.mdxType="string"==typeof t?t:i,l[1]=o;for(var p=2;p<r;p++)l[p]=n[p];return a.createElement.apply(null,l)}return a.createElement.apply(null,n)}u.displayName="MDXCreateElement"},6288:(t,e,n)=>{n.r(e),n.d(e,{assets:()=>d,contentTitle:()=>l,default:()=>g,frontMatter:()=>r,metadata:()=>o,toc:()=>p});var a=n(7462),i=(n(7294),n(3905));const r={sidebar_position:7},l="Configuration types and token permissions",o={unversionedId:"configuration/configTypes_tokenPermissions",id:"configuration/configTypes_tokenPermissions",title:"Configuration types and token permissions",description:"These are the supported configuration types, their API endpoints and the token permissions required for interacting with any of endpoint.",source:"@site/docs/configuration/configTypes_tokenPermissions.md",sourceDirName:"configuration",slug:"/configuration/configTypes_tokenPermissions",permalink:"/dynatrace-configuration-as-code/next/configuration/configTypes_tokenPermissions",draft:!1,editUrl:"https://github.com/dynatrace/dynatrace-configuration-as-code/edit/main/documentation/docs/configuration/configTypes_tokenPermissions.md",tags:[],version:"current",sidebarPosition:7,frontMatter:{sidebar_position:7},sidebar:"tutorialSidebar",previous:{title:"Delete configuration",permalink:"/dynatrace-configuration-as-code/next/configuration/delete_config"},next:{title:"Special Types of Configuration",permalink:"/dynatrace-configuration-as-code/next/configuration/special_config_types"}},d={},p=[],m={toc:p};function g(t){let{components:e,...n}=t;return(0,i.kt)("wrapper",(0,a.Z)({},m,n,{components:e,mdxType:"MDXLayout"}),(0,i.kt)("h1",{id:"configuration-types-and-token-permissions"},"Configuration types and token permissions"),(0,i.kt)("p",null,"These are the supported configuration types, their API endpoints and the token permissions required for interacting with any of endpoint."),(0,i.kt)("table",null,(0,i.kt)("thead",{parentName:"table"},(0,i.kt)("tr",{parentName:"thead"},(0,i.kt)("th",{parentName:"tr",align:null},"Configuration"),(0,i.kt)("th",{parentName:"tr",align:null},"Comment"),(0,i.kt)("th",{parentName:"tr",align:null},"Endpoint"),(0,i.kt)("th",{parentName:"tr",align:null},"Token Permission(s)"))),(0,i.kt)("tbody",{parentName:"table"},(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"alerting-profile"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/alertingProfiles")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-metrics"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/metricEvents")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-disks"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/diskEvents")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"allowed-beacon-origins"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/allowedBeaconOriginsForCors")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-applications"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/applications")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-aws"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/aws")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-database-services"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/databaseServices")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-hosts"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/hosts")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-metrics"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/metricEvents")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-services"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/services")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"anomaly-detection-vmware"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/anomalyDetection/applications")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"app-detection-rule"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("strong",{parentName:"td"},"deprecated in 2.0")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/applicationDetectionRules")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"app-detection-rule-v2"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#non-unique-name"},"Non-Unique Name Configuration"),(0,i.kt)("br",null),(0,i.kt)("strong",{parentName:"td"},"replaces app-detection-rule")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/applicationDetectionRules")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"app-detection-rule-host"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/applicationDetectionRules/hostDetection")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"application"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("strong",{parentName:"td"},"deprecated in 2.0")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/applications/web")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"application-web"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("strong",{parentName:"td"},"replaces application")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/applications/web")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"application-mobile"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/applications/mobile")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"auto-tag"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/autoTags")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"aws-credentials"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/aws/credentials")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"azure-credentials"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/azure/credentials")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"calculated-metrics-application-mobile"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/calculatedMetrics/mobile")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"calculated-metrics-application-web"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/calculatedMetrics/rum")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"calculated-metrics-log"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/calculatedMetrics/log")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"calculated-metrics-service"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/calculatedMetrics/service")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"calculated-metrics-synthetic"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},"_/api/config/v1/calculatedMetrics/synthetic"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"conditional-naming-host"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/conditionalNaming/host")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"conditional-naming-processgroup"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/conditionalNaming/processGroup")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"conditional-naming-service"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/conditionalNaming/service")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"content-resources"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/contentResources")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"credential-vault"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/credentials")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Credential Vault Entries")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Credential Vault Entries"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"custom-service-java"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/customServices/java")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"custom-service-dotnet"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/customServices/dotnet")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"custom-service-go"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/customServices/go")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"custom-service-nodejs"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/customServices/nodejs")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"custom-service-php"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/customServices/php")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"dashboard"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("strong",{parentName:"td"},"deprecated in 2.0")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/dashboards")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"dashboard-v2"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#non-unique-name"},"Non-Unique Name Configuration"),(0,i.kt)("br",null),(0,i.kt)("strong",{parentName:"td"},"replaces dashboard")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/dashboards")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"data-privacy"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/dataPrivacy")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration"),", ",(0,i.kt)("inlineCode",{parentName:"td"},"DataPrivacy"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"extension"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/extensions")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"extension-elasticsearch"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/extensions/dynatrace.python.elasticsearch/global")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"failure-detection-parametersets"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/failureDetection/parameterSelection/parameterSets")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"failure-detection-rules"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/failureDetection/parameterSelection/rules")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"frequent-issue-detection"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/frequentIssueDetection")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"geo-ip-address-mappings"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/geographicRegions/ipAddressMappings")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"geo-ip-detection-headers"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/geographicRegions/ipDetectionHeaders")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"hosts-auto-update"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/hosts/autoupdate")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"kubernetes-credentials"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/kubernetes/credentials")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"maintenance-window"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/maintenanceWindows")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuratio"),"  & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"management-zone"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/managementZones")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"notification"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/notifications")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"reports"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/reports")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"request-attributes"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/requestAttributes")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Capture request data"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"request-naming-service"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("strong",{parentName:"td"},"deprecated in 2.0")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/requestNaming")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"request-naming-service-v2"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#non-unique-name"},"Non-Unique Name Configuration"),(0,i.kt)("br",null),(0,i.kt)("strong",{parentName:"td"},"replaces request-naming-service ")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/requestNaming")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"slo"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/v2/slo")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read SLO")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write SLOs"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"service-detection-full-web-request"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/detectionRules/FULL_WEB_REQUEST")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"service-detection-full-web-service"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/detectionRules/FULL_WEB_SERVICE")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"service-detection-opaque-web-request"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_REQUEST")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"service-detection-opaque-web-service"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_SERVICE")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"service-resource-naming"),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("a",{parentName:"td",href:"/dynatrace-configuration-as-code/next/configuration/special_config_types#single-configuration-endpoint"},"Single Configuration Endpoint")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/config/v1/service/resourceNaming")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Read Configuration")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Write Configuration"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"synthetic-location"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/v1/synthetic/locations")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Access problem and event feed, metrics, and topology")," & ",(0,i.kt)("inlineCode",{parentName:"td"},"Create and read synthetic monitors, locations, and nodes"))),(0,i.kt)("tr",{parentName:"tbody"},(0,i.kt)("td",{parentName:"tr",align:null},"synthetic-monitor"),(0,i.kt)("td",{parentName:"tr",align:null}),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("em",{parentName:"td"},"/api/v1/synthetic/monitors")),(0,i.kt)("td",{parentName:"tr",align:null},(0,i.kt)("inlineCode",{parentName:"td"},"Create and read synthetic monitors, locations, and nodes"))))),(0,i.kt)("p",null,"For reference, refer to ",(0,i.kt)("a",{parentName:"p",href:"https://www.dynatrace.com/support/help/dynatrace-api/basics/dynatrace-api-authentication"},"this")," page for a detailed\ndescription to each token permission."),(0,i.kt)("p",null,"If your desired API is not in the table above, please consider adding it by following the instructions in ",(0,i.kt)("a",{parentName:"p",href:"/dynatrace-configuration-as-code/next/Guides/add_new_api"},"How to add new APIs"),"."))}g.isMDXComponent=!0}}]);