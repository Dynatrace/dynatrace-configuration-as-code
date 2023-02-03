"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[671],{3905:(e,n,t)=>{t.d(n,{Zo:()=>c,kt:()=>m});var o=t(7294);function r(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function i(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);n&&(o=o.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,o)}return t}function a(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?i(Object(t),!0).forEach((function(n){r(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):i(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function l(e,n){if(null==e)return{};var t,o,r=function(e,n){if(null==e)return{};var t,o,r={},i=Object.keys(e);for(o=0;o<i.length;o++)t=i[o],n.indexOf(t)>=0||(r[t]=e[t]);return r}(e,n);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(o=0;o<i.length;o++)t=i[o],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(r[t]=e[t])}return r}var p=o.createContext({}),d=function(e){var n=o.useContext(p),t=n;return e&&(t="function"==typeof e?e(n):a(a({},n),e)),t},c=function(e){var n=d(e.components);return o.createElement(p.Provider,{value:n},e.children)},s={inlineCode:"code",wrapper:function(e){var n=e.children;return o.createElement(o.Fragment,{},n)}},u=o.forwardRef((function(e,n){var t=e.components,r=e.mdxType,i=e.originalType,p=e.parentName,c=l(e,["components","mdxType","originalType","parentName"]),u=d(t),m=r,f=u["".concat(p,".").concat(m)]||u[m]||s[m]||i;return t?o.createElement(f,a(a({ref:n},c),{},{components:t})):o.createElement(f,a({ref:n},c))}));function m(e,n){var t=arguments,r=n&&n.mdxType;if("string"==typeof e||r){var i=t.length,a=new Array(i);a[0]=u;var l={};for(var p in n)hasOwnProperty.call(n,p)&&(l[p]=n[p]);l.originalType=e,l.mdxType="string"==typeof e?e:r,a[1]=l;for(var d=2;d<i;d++)a[d]=t[d];return o.createElement.apply(null,a)}return o.createElement.apply(null,t)}u.displayName="MDXCreateElement"},2989:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>p,contentTitle:()=>a,default:()=>s,frontMatter:()=>i,metadata:()=>l,toc:()=>d});var o=t(7462),r=(t(7294),t(3905));const i={sidebar_position:4,title:"How to ensure ordering of application detection rules via monaco"},a=void 0,l={unversionedId:"Guides/ordering_app_detection_rules",id:"Guides/ordering_app_detection_rules",title:"How to ensure ordering of application detection rules via monaco",description:"While the UI allows you to define the ordering of Application Detection Rules, it is not possible to define an order via monaco or the API.",source:"@site/docs/Guides/ordering_app_detection_rules.md",sourceDirName:"Guides",slug:"/Guides/ordering_app_detection_rules",permalink:"/dynatrace-configuration-as-code/next/Guides/ordering_app_detection_rules",draft:!1,editUrl:"https://github.com/dynatrace/dynatrace-configuration-as-code/edit/main/documentation/docs/Guides/ordering_app_detection_rules.md",tags:[],version:"current",sidebarPosition:4,frontMatter:{sidebar_position:4,title:"How to ensure ordering of application detection rules via monaco"},sidebar:"tutorialSidebar",previous:{title:"Migration of application detection rules",permalink:"/dynatrace-configuration-as-code/next/Guides/app_detection_rules_migration"},next:{title:"License and Bill of Materials",permalink:"/dynatrace-configuration-as-code/next/Useful-links/bill-of-materials"}},p={},d=[],c={toc:d};function s(e){let{components:n,...t}=e;return(0,r.kt)("wrapper",(0,o.Z)({},c,t,{components:n,mdxType:"MDXLayout"}),(0,r.kt)("p",null,"While the UI allows you to define the ordering of Application Detection Rules, it is not possible to define an order via monaco or the API.\n(Using the API directly, you ",(0,r.kt)("em",{parentName:"p"},"can")," control if rules are appended or pre-pended but not in the exact order. By using monaco, this control is not possible)"),(0,r.kt)("p",null,"However, you can use monaco's handling of dependencies between configurations to enforce the ordering of rules."),(0,r.kt)("h1",{id:"defining-rule-ordering-using-dependencies"},"Defining rule ordering using dependencies"),(0,r.kt)("p",null,'By creating "fake" dependencies between rules, monaco will ensure that a rule will be created before another one that depends on it.'),(0,r.kt)("p",null,"This workaround only works if all the rules are created from monaco and don't already exist."),(0,r.kt)("blockquote",null,(0,r.kt)("p",{parentName:"blockquote"},"If rules already exist, you can manually re-order them, with updates from future configuration deployments not impacting the order.")),(0,r.kt)("p",null,"As newly added rules will be pre-pended to existing ones, you will likely need to define dependencies in the opposite order you expect."),(0,r.kt)("p",null,"The following sample details how to achieve an order using dependencies:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},'config:\n- det3: "det.json"\n- det2: "det.json"\n- det1: "det.json"\n\ndet3:\n- name: "App-Detection Rule 3"\n- application_id: "project/application-web/application.id"\n- order: "project/app-detection-rule/det2.name"\n\ndet2:\n- name: "App-Detection Rule 2"\n- application_id: "project/application-web/application.id"\n- order: "project/app-detection-rule/det1.name"\n\ndet1:\n- name: "App-Detection Rule 1"\n- application_id: "project/application-web/application.id"\n')),(0,r.kt)("p",null,"As you can see, ",(0,r.kt)("inlineCode",{parentName:"p"},"Rule 3")," depends on ",(0,r.kt)("inlineCode",{parentName:"p"},"Rule 2"),", and ",(0,r.kt)("inlineCode",{parentName:"p"},"Rule 2")," depends on ",(0,r.kt)("inlineCode",{parentName:"p"},"Rule 1"),"."),(0,r.kt)("p",null,"With this, monaco will ensure that the rules are created in the order 1, 2, 3."),(0,r.kt)("p",null,"As mentioned above, the API pre-pends new rules, so the rules will be applied as 3,2,1."),(0,r.kt)("p",null,"Flipping the dependencies will result in a 1,2,3 ordering of the created rules:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},'config:\n- det3: "det.json"\n- det2: "det.json"\n- det1: "det.json"\n\ndet3:\n- name: "App-Detection Rule 3"\n- application_id: "project/application-web/application.id"\n\ndet2:\n- name: "App-Detection Rule 2"\n- application_id: "project/application-web/application.id"\n- order: "project/app-detection-rule/det3.name"\n\ndet1:\n- name: "App-Detection Rule 1"\n- application_id: "project/application-web/application.id"\n- order: "project/app-detection-rule/det2.name"\n')))}s.isMDXComponent=!0}}]);