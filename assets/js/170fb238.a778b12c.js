(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[2998],{3905:function(e,n,t){"use strict";t.d(n,{Zo:function(){return u},kt:function(){return g}});var a=t(7294);function o(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function i(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);n&&(a=a.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,a)}return t}function r(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?i(Object(t),!0).forEach((function(n){o(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):i(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function l(e,n){if(null==e)return{};var t,a,o=function(e,n){if(null==e)return{};var t,a,o={},i=Object.keys(e);for(a=0;a<i.length;a++)t=i[a],n.indexOf(t)>=0||(o[t]=e[t]);return o}(e,n);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(a=0;a<i.length;a++)t=i[a],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var p=a.createContext({}),c=function(e){var n=a.useContext(p),t=n;return e&&(t="function"==typeof e?e(n):r(r({},n),e)),t},u=function(e){var n=c(e.components);return a.createElement(p.Provider,{value:n},e.children)},s={inlineCode:"code",wrapper:function(e){var n=e.children;return a.createElement(a.Fragment,{},n)}},m=a.forwardRef((function(e,n){var t=e.components,o=e.mdxType,i=e.originalType,p=e.parentName,u=l(e,["components","mdxType","originalType","parentName"]),m=c(t),g=o,f=m["".concat(p,".").concat(g)]||m[g]||s[g]||i;return t?a.createElement(f,r(r({ref:n},u),{},{components:t})):a.createElement(f,r({ref:n},u))}));function g(e,n){var t=arguments,o=n&&n.mdxType;if("string"==typeof e||o){var i=t.length,r=new Array(i);r[0]=m;var l={};for(var p in n)hasOwnProperty.call(n,p)&&(l[p]=n[p]);l.originalType=e,l.mdxType="string"==typeof e?e:o,r[1]=l;for(var c=2;c<i;c++)r[c]=t[c];return a.createElement.apply(null,r)}return a.createElement.apply(null,t)}m.displayName="MDXCreateElement"},8158:function(e,n,t){"use strict";t.r(n),t.d(n,{frontMatter:function(){return l},metadata:function(){return p},toc:function(){return c},default:function(){return s}});var a=t(2122),o=t(9756),i=(t(7294),t(3905)),r=["components"],l={sidebar_position:4},p={unversionedId:"configuration/yaml_confg",id:"version-1.5.2/configuration/yaml_confg",isDocsHomePage:!1,title:"Configuration YAML Structure",description:"Every configuration needs a YAML containing required and optional content.",source:"@site/versioned_docs/version-1.5.2/configuration/yaml_confg.md",sourceDirName:"configuration",slug:"/configuration/yaml_confg",permalink:"/dynatrace-monitoring-as-code/1.5.2/configuration/yaml_confg",editUrl:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/versioned_docs/version-1.5.2/configuration/yaml_confg.md",version:"1.5.2",sidebarPosition:4,frontMatter:{sidebar_position:4},sidebar:"version-1.5.2/tutorialSidebar",previous:{title:"Configuration Structure",permalink:"/dynatrace-monitoring-as-code/1.5.2/configuration/configuration_structure"},next:{title:"Plugin Configuration",permalink:"/dynatrace-monitoring-as-code/1.5.2/configuration/plugin_config"}},c=[{value:"Skip configuration deployment",id:"skip-configuration-deployment",children:[]},{value:"Specific Configuration per Environment or group",id:"specific-configuration-per-environment-or-group",children:[]},{value:"Referencing other Configurations",id:"referencing-other-configurations",children:[]},{value:"Referencing other json templates",id:"referencing-other-json-templates",children:[]},{value:"Templating of Environment Variables",id:"templating-of-environment-variables",children:[]}],u={toc:c};function s(e){var n=e.components,t=(0,o.Z)(e,r);return(0,i.kt)("wrapper",(0,a.Z)({},u,t,{components:n,mdxType:"MDXLayout"}),(0,i.kt)("p",null,"Every configuration needs a YAML containing required and optional content."),(0,i.kt)("p",null,"A minimal viable config needs to look like this:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'config:\n    - {config name} : "{path of config json template}"\n\n{config name}:\n    - name: "{a unique name}"\n')),(0,i.kt)("p",null,"e.g. in ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/infrastructure/alerting-profile/profiles.yaml")),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'config:\n  - profile: "projects/infrastructure/alerting-profile/profile.json"\n\nprofile:\n  - name: "profile-name"\n[...]\n')),(0,i.kt)("p",null,"Every config needs to provide a name for unique identification, omitting the name variable or using a duplicate name will result in a validation / deployment error."),(0,i.kt)("p",null,"Any defined ",(0,i.kt)("inlineCode",{parentName:"p"},"{config name}")," represents a variable that can then be used in a ",(0,i.kt)("a",{parentName:"p",href:"#config-json-templates"},"JSON template"),", and will be resolved and inserted into the config before deployment to Dynatrace."),(0,i.kt)("p",null,"e.g. ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/infrastructure/alerting-profile/profiles.yaml")," defines a ",(0,i.kt)("inlineCode",{parentName:"p"},"name"),":"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'[...]\nprofile:\n  - name: "EXAMPLE Infrastructure"\n[...]\n')),(0,i.kt)("p",null,"Which is then used in ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/infrastructure/alerting-profile/profile.json")," as ",(0,i.kt)("inlineCode",{parentName:"p"},"{{.name}}"),"."),(0,i.kt)("h3",{id:"skip-configuration-deployment"},"Skip configuration deployment"),(0,i.kt)("p",null,"To skip configuration from deploying you can use predefined ",(0,i.kt)("inlineCode",{parentName:"p"},"skipDeployment")," parameter. You can skip deployment of the whole configuration:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'my-config:\n  - name: "My config"\n  - skipDeployment: "true"\n')),(0,i.kt)("p",null,"enable it by default, but skip for environment or group:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'my-config:\n  - name: "My config"\n  - skipDeployment: "true"\n\nmy-config.development:\n  - skipDeployment: "false"\n')),(0,i.kt)("p",null,"or disable it by default and enable only for environment or group:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'my-config:\n  - name: "My config"\n  - skipDeployment: "false"\n\nmy-config.environment:\n  - skipDeployment: "true"\n')),(0,i.kt)("h3",{id:"specific-configuration-per-environment-or-group"},"Specific Configuration per Environment or group"),(0,i.kt)("p",null,"Configuration can be overwritten or extended:"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},"per environment by adding ",(0,i.kt)("inlineCode",{parentName:"li"},".{Environment}")," configurations"),(0,i.kt)("li",{parentName:"ul"},"per group by adding ",(0,i.kt)("inlineCode",{parentName:"li"},".{GROUP}")," configurations")),(0,i.kt)("p",null,"e.g. ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/infrastructure/notification/notifications.yaml")," defines different recipients for email notifications for each environment via"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},"email:\n    [...]\n\nemail.group:\n    [...]\n\nemail.environment1:\n    [...]\n\nemail.environment2:\n    [...]\n\nemail.environment3:\n    [...]\n")),(0,i.kt)("p",null,"Anything in the base ",(0,i.kt)("inlineCode",{parentName:"p"},"email")," configuration is still applied, unless it's re-defined in the ",(0,i.kt)("inlineCode",{parentName:"p"},".{GROUP}")," or ",(0,i.kt)("inlineCode",{parentName:"p"},".{Environment}")," config."),(0,i.kt)("p",null,(0,i.kt)("strong",{parentName:"p"},"If both environment and group configurations are defined, then environment\nis preferred over the group configuration.")),(0,i.kt)("h3",{id:"referencing-other-configurations"},"Referencing other Configurations"),(0,i.kt)("p",null,"In many cases one auto-deployed Dynatrace configuration will depend on another one."),(0,i.kt)("p",null,"E.g. Where most configurations depend on the management-zone defined in ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/infrastructure/management-zone")),(0,i.kt)("p",null,"The tool allows your configuration to reference either the ",(0,i.kt)("inlineCode",{parentName:"p"},"name")," or ",(0,i.kt)("inlineCode",{parentName:"p"},"id")," of the Dynatrace object of another configuration created on the cluster."),(0,i.kt)("p",null,"To reference these, the dependent ",(0,i.kt)("inlineCode",{parentName:"p"},"config yaml")," can configure a variable of the format"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},'{var} : "{name of the referenced configuration}.[id|name]"\n')),(0,i.kt)("p",null,"e.g. ",(0,i.kt)("inlineCode",{parentName:"p"},"projects/project-name/dashboard/dashboard.yaml")," references the management-zone defined by ",(0,i.kt)("inlineCode",{parentName:"p"},"/projects/infrastructure/management-zone/zone.json")," via"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'  - managementZoneId: "projects/infrastructure/management-zone/zone.id"\n')),(0,i.kt)("h3",{id:"referencing-other-json-templates"},"Referencing other json templates"),(0,i.kt)("p",null,"Json templates are usually defined inside of project configuration and then references in same project:"),(0,i.kt)("p",null,(0,i.kt)("strong",{parentName:"p"},"testproject/auto-tag/auto-tag.yaml:")),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'config:\n  - application-tagging-multiproject: "application-tagging.json"\n\napplication-tagging-multiproject:\n  - name: "Test Application Multiproject"\n')),(0,i.kt)("p",null,"In this example, ",(0,i.kt)("inlineCode",{parentName:"p"},"application-tagging.json")," is located in ",(0,i.kt)("inlineCode",{parentName:"p"},"auto-tag")," folder of same project and the path to it\ncan be defined relative to ",(0,i.kt)("inlineCode",{parentName:"p"},"auto-tag.yaml")," file. But, what if you would like to reuse one template defined outside of this project?\ncan be defined relative to ",(0,i.kt)("inlineCode",{parentName:"p"},"auto-tag.yaml")," file. But, what if you would like to reuse one template defined outside of this project?\ncan be defined relative to ",(0,i.kt)("inlineCode",{parentName:"p"},"auto-tag.yaml")," file. But, what if you would like to reuse one template defined outside of this project?\nIn this case, you need to define a full path of json template:"),(0,i.kt)("p",null,(0,i.kt)("strong",{parentName:"p"},"testproject/auto-tag/auto-tag.yaml:")),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'config:\n  - application-tagging-multiproject: "/path/to/project/auto-tag/application-tagging.json"\n\napplication-tagging-multiproject:\n  - name: "Test Application Multiproject"\n')),(0,i.kt)("p",null,"This would save us of content duplication and redefining same templates over and over again."),(0,i.kt)("p",null,"Of course, it is also possible to reuse one template multiple times within one or different yaml file(s):\n",(0,i.kt)("strong",{parentName:"p"},"testproject/auto-tag/auto-tag.yaml:")),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'config:\n  - application-tagging-multiproject: "/path/to/project/auto-tag/application-tagging.json"\n  - application-tagging-tesproject: "/path/to/project/auto-tag/application-tagging.json"\n  - application-tagging-otherproject: "/path/to/project/auto-tag/application-tagging.json"\n\napplication-tagging-multiproject:\n  - name: "Test Application Multiproject"\n  - param: "Multiproject parameter"\n\napplication-tagging-tesproject:\n  - name: "Test Application Tesproject"\n  - param: "Tesproject parameter"\n\napplication-tagging-otherproject:\n  - name: "Test Application Otherproject"\n  - param: "Otherproject parameter"\n')),(0,i.kt)("h3",{id:"templating-of-environment-variables"},"Templating of Environment Variables"),(0,i.kt)("p",null,"In addition to the templating of ",(0,i.kt)("inlineCode",{parentName:"p"},"json")," files, where you need to specify the values in the corresponding ",(0,i.kt)("inlineCode",{parentName:"p"},"yaml")," files, its also possible to resolve\nenvironment variables. This can be done in any ",(0,i.kt)("inlineCode",{parentName:"p"},"json")," or ",(0,i.kt)("inlineCode",{parentName:"p"},"yaml")," file using this syntax: ",(0,i.kt)("inlineCode",{parentName:"p"},"{{.Env.ENV_VAR}}"),"."),(0,i.kt)("p",null,"E.g. to resolve the URL of an environment, use the following snippet:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-yaml"},'development:\n    - name: "Dev"\n    - env-url: "{{ .Env.DEV_URL }}"\n    - env-token-name: "DEV_TOKEN_ENV_VAR"\n')),(0,i.kt)("p",null,"To resolve an environment variable directly in the ",(0,i.kt)("inlineCode",{parentName:"p"},"json")," is also possible. See the following example which sets the value\nof an alerting profile from the env var ",(0,i.kt)("inlineCode",{parentName:"p"},"ALERTING_PROFILE_VALUE"),"."),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-json"},'{\n  "name": "{{ .name }}",\n  "rules": [\n    {\n      "type": "APPLICATION",\n      "enabled": true,\n      "valueFormat": null,\n      "propagationTypes": [],\n      "conditions": [\n        {\n          "key": {\n            "attribute": "WEB_APPLICATION_NAME"\n          },\n          "comparisonInfo": {\n            "type": "STRING",\n            "operator": "CONTAINS",\n            "value": "{{ .Env.ALERTING_PROFILE_VALUE }}",\n            "negate": false,\n            "caseSensitive": true\n          }\n        }\n      ]\n    }\n  ]\n}\n')),(0,i.kt)("p",null,(0,i.kt)("strong",{parentName:"p"},"Attention"),": Values you pass into configuration via environment variables must not contain ",(0,i.kt)("inlineCode",{parentName:"p"},"="),"."))}s.isMDXComponent=!0}}]);