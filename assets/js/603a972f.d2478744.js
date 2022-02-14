"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[66],{3905:function(e,n,o){o.d(n,{Zo:function(){return p},kt:function(){return f}});var t=o(7294);function r(e,n,o){return n in e?Object.defineProperty(e,n,{value:o,enumerable:!0,configurable:!0,writable:!0}):e[n]=o,e}function i(e,n){var o=Object.keys(e);if(Object.getOwnPropertySymbols){var t=Object.getOwnPropertySymbols(e);n&&(t=t.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),o.push.apply(o,t)}return o}function a(e){for(var n=1;n<arguments.length;n++){var o=null!=arguments[n]?arguments[n]:{};n%2?i(Object(o),!0).forEach((function(n){r(e,n,o[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(o)):i(Object(o)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(o,n))}))}return e}function l(e,n){if(null==e)return{};var o,t,r=function(e,n){if(null==e)return{};var o,t,r={},i=Object.keys(e);for(t=0;t<i.length;t++)o=i[t],n.indexOf(o)>=0||(r[o]=e[o]);return r}(e,n);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(t=0;t<i.length;t++)o=i[t],n.indexOf(o)>=0||Object.prototype.propertyIsEnumerable.call(e,o)&&(r[o]=e[o])}return r}var c=t.createContext({}),s=function(e){var n=t.useContext(c),o=n;return e&&(o="function"==typeof e?e(n):a(a({},n),e)),o},p=function(e){var n=s(e.components);return t.createElement(c.Provider,{value:n},e.children)},u={inlineCode:"code",wrapper:function(e){var n=e.children;return t.createElement(t.Fragment,{},n)}},d=t.forwardRef((function(e,n){var o=e.components,r=e.mdxType,i=e.originalType,c=e.parentName,p=l(e,["components","mdxType","originalType","parentName"]),d=s(o),f=r,m=d["".concat(c,".").concat(f)]||d[f]||u[f]||i;return o?t.createElement(m,a(a({ref:n},p),{},{components:o})):t.createElement(m,a({ref:n},p))}));function f(e,n){var o=arguments,r=n&&n.mdxType;if("string"==typeof e||r){var i=o.length,a=new Array(i);a[0]=d;var l={};for(var c in n)hasOwnProperty.call(n,c)&&(l[c]=n[c]);l.originalType=e,l.mdxType="string"==typeof e?e:r,a[1]=l;for(var s=2;s<i;s++)a[s]=o[s];return t.createElement.apply(null,a)}return t.createElement.apply(null,o)}d.displayName="MDXCreateElement"},720:function(e,n,o){o.r(n),o.d(n,{frontMatter:function(){return l},metadata:function(){return c},toc:function(){return s},default:function(){return u}});var t=o(7462),r=o(3366),i=(o(7294),o(3905)),a=["components"],l={sidebar_position:1},c={unversionedId:"configuration/deploy_configuration",id:"version-1.6.0/configuration/deploy_configuration",isDocsHomePage:!1,title:"Deploying Configuration to Dynatrace",description:"Monaco allows for deploying a configuration or a set of configurations in the form of project(s). A project is a folder containing files that define configurations to be deployed to a environment or a group of environments. This is done by passing the --project flag (or -p for short).",source:"@site/versioned_docs/version-1.6.0/configuration/deploy_configuration.md",sourceDirName:"configuration",slug:"/configuration/deploy_configuration",permalink:"/dynatrace-monitoring-as-code/1.6.0/configuration/deploy_configuration",editUrl:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/versioned_docs/version-1.6.0/configuration/deploy_configuration.md",version:"1.6.0",sidebarPosition:1,frontMatter:{sidebar_position:1},sidebar:"version-1.6.0/tutorialSidebar",previous:{title:"Logging",permalink:"/dynatrace-monitoring-as-code/1.6.0/commands/Logging"},next:{title:"Environments file",permalink:"/dynatrace-monitoring-as-code/1.6.0/configuration/environments_file"}},s=[{value:"Running The Tool",id:"running-the-tool",children:[]},{value:"Running The Tool With A Proxy",id:"running-the-tool-with-a-proxy",children:[]}],p={toc:s};function u(e){var n=e.components,o=(0,r.Z)(e,a);return(0,i.kt)("wrapper",(0,t.Z)({},p,o,{components:n,mdxType:"MDXLayout"}),(0,i.kt)("p",null,"Monaco allows for deploying a configuration or a set of configurations in the form of project(s). A project is a folder containing files that define configurations to be deployed to a environment or a group of environments. This is done by passing the --project flag (or -p for short)."),(0,i.kt)("h3",{id:"running-the-tool"},"Running The Tool"),(0,i.kt)("p",null,"Below you find a few samples on how to run the tool to deploy your configurations:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-shell",metastring:'title="shell"',title:'"shell"'},'monaco -e=environments.yaml (deploy all projects in the current folder to all environments)\n\nmonaco -e=environments.yaml -p="project" projects-root-folder (deploy projects-root-folder/project and any projects in projects-root-folder it depends on to all environments)\n\nmonaco -e=environments.yaml -p="projectA, projectB" projects-root-folder (deploy projects-root-folder/projectA, projectB and dependencies to all environments)\n\nmonaco -e=environments.yaml -se dev (deploy all projects in the current folder to the "dev" environment defined in environments.yaml)\n')),(0,i.kt)("p",null,"If ",(0,i.kt)("inlineCode",{parentName:"p"},"project")," contains additional sub-projects, then all projects are deployed recursively."),(0,i.kt)("p",null,"If ",(0,i.kt)("inlineCode",{parentName:"p"},"project")," depends on different projects under the same root, those are also deployed."),(0,i.kt)("p",null,"Multiple projects could be specified by ",(0,i.kt)("inlineCode",{parentName:"p"},'-p="projectA, projectB, projectC/subproject"')),(0,i.kt)("p",null,"To deploy configuration the tool will need a valid API Token(s) for the given environments defined as ",(0,i.kt)("inlineCode",{parentName:"p"},"environment variables")," - you can define the name of that env var in the environments file."),(0,i.kt)("p",null,"To deploy to 1 specific environment within a ",(0,i.kt)("inlineCode",{parentName:"p"},"environments.yaml")," file, the ",(0,i.kt)("inlineCode",{parentName:"p"},"-specific-environment")," or -se flag can be passed:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-shell",metastring:'title="shell"',title:'"shell"'},'monaco -e=environments.yaml -se=my-environment -p="my-environment" cluster\n')),(0,i.kt)("h3",{id:"running-the-tool-with-a-proxy"},"Running The Tool With A Proxy"),(0,i.kt)("p",null,"In environments where access to Dynatrace API endpoints is only possible or allowed via a proxy server, monaco provides the options to specify the address of your proxy server when running a command:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-shell",metastring:'title="shell"',title:'"shell"'},'HTTPS_PROXY=localhost:5000 monaco -e=environments.yaml -se=my-environment -p="my-environment" cluster \n')),(0,i.kt)("p",null,"With the new CLI:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-shell",metastring:'title="shell"',title:'"shell"'},"HTTPS_PROXY=localhost:5000 NEW_CLI=1 monaco deploy -e environments.yaml \n")))}u.isMDXComponent=!0}}]);