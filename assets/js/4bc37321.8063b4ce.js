"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[2236],{3905:(e,n,t)=>{t.d(n,{Zo:()=>u,kt:()=>p});var o=t(7294);function a(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function r(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);n&&(o=o.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,o)}return t}function l(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?r(Object(t),!0).forEach((function(n){a(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):r(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function i(e,n){if(null==e)return{};var t,o,a=function(e,n){if(null==e)return{};var t,o,a={},r=Object.keys(e);for(o=0;o<r.length;o++)t=r[o],n.indexOf(t)>=0||(a[t]=e[t]);return a}(e,n);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);for(o=0;o<r.length;o++)t=r[o],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(a[t]=e[t])}return a}var s=o.createContext({}),c=function(e){var n=o.useContext(s),t=n;return e&&(t="function"==typeof e?e(n):l(l({},n),e)),t},u=function(e){var n=c(e.components);return o.createElement(s.Provider,{value:n},e.children)},d={inlineCode:"code",wrapper:function(e){var n=e.children;return o.createElement(o.Fragment,{},n)}},m=o.forwardRef((function(e,n){var t=e.components,a=e.mdxType,r=e.originalType,s=e.parentName,u=i(e,["components","mdxType","originalType","parentName"]),m=c(t),p=a,f=m["".concat(s,".").concat(p)]||m[p]||d[p]||r;return t?o.createElement(f,l(l({ref:n},u),{},{components:t})):o.createElement(f,l({ref:n},u))}));function p(e,n){var t=arguments,a=n&&n.mdxType;if("string"==typeof e||a){var r=t.length,l=new Array(r);l[0]=m;var i={};for(var s in n)hasOwnProperty.call(n,s)&&(i[s]=n[s]);i.originalType=e,i.mdxType="string"==typeof e?e:a,l[1]=i;for(var c=2;c<r;c++)l[c]=t[c];return o.createElement.apply(null,l)}return o.createElement.apply(null,t)}m.displayName="MDXCreateElement"},5162:(e,n,t)=>{t.d(n,{Z:()=>l});var o=t(7294),a=t(6010);const r="tabItem_Ymn6";function l(e){let{children:n,hidden:t,className:l}=e;return o.createElement("div",{role:"tabpanel",className:(0,a.Z)(r,l),hidden:t},n)}},5488:(e,n,t)=>{t.d(n,{Z:()=>p});var o=t(7462),a=t(7294),r=t(6010),l=t(2389),i=t(7392),s=t(7094),c=t(2466);const u="tabList__CuJ",d="tabItem_LNqP";function m(e){var n,t;const{lazy:l,block:m,defaultValue:p,values:f,groupId:y,className:v}=e,h=a.Children.map(e.children,(e=>{if((0,a.isValidElement)(e)&&"value"in e.props)return e;throw new Error("Docusaurus error: Bad <Tabs> child <"+("string"==typeof e.type?e.type:e.type.name)+'>: all children of the <Tabs> component should be <TabItem>, and every <TabItem> should have a unique "value" prop.')})),b=null!=f?f:h.map((e=>{let{props:{value:n,label:t,attributes:o}}=e;return{value:n,label:t,attributes:o}})),g=(0,i.l)(b,((e,n)=>e.value===n.value));if(g.length>0)throw new Error('Docusaurus error: Duplicate values "'+g.map((e=>e.value)).join(", ")+'" found in <Tabs>. Every value needs to be unique.');const w=null===p?p:null!=(n=null!=p?p:null==(t=h.find((e=>e.props.default)))?void 0:t.props.value)?n:h[0].props.value;if(null!==w&&!b.some((e=>e.value===w)))throw new Error('Docusaurus error: The <Tabs> has a defaultValue "'+w+'" but none of its children has the corresponding value. Available values are: '+b.map((e=>e.value)).join(", ")+". If you intend to show no default tab, use defaultValue={null} instead.");const{tabGroupChoices:k,setTabGroupChoices:N}=(0,s.U)(),[O,T]=(0,a.useState)(w),x=[],{blockElementScrollPositionUntilNextRender:j}=(0,c.o5)();if(null!=y){const e=k[y];null!=e&&e!==O&&b.some((n=>n.value===e))&&T(e)}const E=e=>{const n=e.currentTarget,t=x.indexOf(n),o=b[t].value;o!==O&&(j(n),T(o),null!=y&&N(y,String(o)))},C=e=>{var n;let t=null;switch(e.key){case"ArrowRight":{var o;const n=x.indexOf(e.currentTarget)+1;t=null!=(o=x[n])?o:x[0];break}case"ArrowLeft":{var a;const n=x.indexOf(e.currentTarget)-1;t=null!=(a=x[n])?a:x[x.length-1];break}}null==(n=t)||n.focus()};return a.createElement("div",{className:(0,r.Z)("tabs-container",u)},a.createElement("ul",{role:"tablist","aria-orientation":"horizontal",className:(0,r.Z)("tabs",{"tabs--block":m},v)},b.map((e=>{let{value:n,label:t,attributes:l}=e;return a.createElement("li",(0,o.Z)({role:"tab",tabIndex:O===n?0:-1,"aria-selected":O===n,key:n,ref:e=>x.push(e),onKeyDown:C,onFocus:E,onClick:E},l,{className:(0,r.Z)("tabs__item",d,null==l?void 0:l.className,{"tabs__item--active":O===n})}),null!=t?t:n)}))),l?(0,a.cloneElement)(h.filter((e=>e.props.value===O))[0],{className:"margin-top--md"}):a.createElement("div",{className:"margin-top--md"},h.map(((e,n)=>(0,a.cloneElement)(e,{key:n,hidden:e.props.value!==O})))))}function p(e){const n=(0,l.Z)();return a.createElement(m,(0,o.Z)({key:String(n)},e))}},3034:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>u,contentTitle:()=>s,default:()=>p,frontMatter:()=>i,metadata:()=>c,toc:()=>d});var o=t(7462),a=(t(7294),t(3905)),r=t(5488),l=t(5162);const i={sidebar_position:2,title:"Install Monaco"},s=void 0,c={unversionedId:"Get-started/installation",id:"version-1.7.0/Get-started/installation",title:"Install Monaco",description:"This guide shows you how to download Monaco and install it on your operating system (Linux/macOS or Windows).",source:"@site/versioned_docs/version-1.7.0/Get-started/installation.md",sourceDirName:"Get-started",slug:"/Get-started/installation",permalink:"/dynatrace-monitoring-as-code/1.7.0/Get-started/installation",draft:!1,editUrl:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/versioned_docs/version-1.7.0/Get-started/installation.md",tags:[],version:"1.7.0",sidebarPosition:2,frontMatter:{sidebar_position:2,title:"Install Monaco"},sidebar:"version-1.7.0/tutorialSidebar",previous:{title:"What is Monaco ?",permalink:"/dynatrace-monitoring-as-code/1.7.0/Get-started/intro"},next:{title:"Validate configuration",permalink:"/dynatrace-monitoring-as-code/1.7.0/commands/validating-configuration"}},u={},d=[],m={toc:d};function p(e){let{components:n,...t}=e;return(0,a.kt)("wrapper",(0,o.Z)({},m,t,{components:n,mdxType:"MDXLayout"}),(0,a.kt)("p",null,"This guide shows you how to download Monaco and install it on your operating system (Linux/macOS or Windows)."),(0,a.kt)("ol",null,(0,a.kt)("li",{parentName:"ol"},"Go to the Monaco ",(0,a.kt)("a",{parentName:"li",href:"https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases"},"release page"),"."),(0,a.kt)("li",{parentName:"ol"},"Download the appropriate version."),(0,a.kt)("li",{parentName:"ol"},"Check that the Monaco binary is available on your PATH. This process will differ depending on your operating system (see steps below). ")),(0,a.kt)(r.Z,{defaultValue:"operating system",values:[{label:"Operating System",value:"operating system"}],mdxType:"Tabs"},(0,a.kt)(l.Z,{value:"operating system",mdxType:"TabItem"},(0,a.kt)(r.Z,{defaultValue:"linux-macos",values:[{label:"Linux / macOS",value:"linux-macos"},{label:"Windows",value:"windows"}],mdxType:"Tabs"},(0,a.kt)(l.Z,{value:"linux-macos",mdxType:"TabItem"},(0,a.kt)("p",null,"For Linux/macOS, we recommend using ",(0,a.kt)("inlineCode",{parentName:"p"},"curl"),". You can download it from ",(0,a.kt)("a",{parentName:"p",href:"https://curl.se/"},"here")," or use ",(0,a.kt)("inlineCode",{parentName:"p"},"wget"),"."),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-shell"},"# Linux\n# x64\n curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.7.0/monaco-linux-amd64 -o monaco\n\n# x86\n curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.7.0/monaco-linux-386 -o monaco\n\n# macOS\n curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.7.0/monaco-darwin-10.16-amd64 -o monaco\n")),(0,a.kt)("p",null,"Make the binary executable:"),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-shell"}," chmod +x monaco\n")),(0,a.kt)("p",null,"Optionally, install Monaco to a central location in your ",(0,a.kt)("inlineCode",{parentName:"p"},"PATH"),".\nThis command assumes that the binary is currently in your downloads folder and that your $PATH includes ",(0,a.kt)("inlineCode",{parentName:"p"},"/usr/local/bin"),":"),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-shell"},"# use any path that suits you; this is just a standard example. Install sudo if needed.\n sudo mv ~/Downloads/monaco /usr/local/bin/\n")),(0,a.kt)("p",null,"Now you can execute the ",(0,a.kt)("inlineCode",{parentName:"p"},"monaco")," command to verify the download. "),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-shell"}," monaco\nYou are currently using the old CLI structure which will be used by\ndefault until monaco version 2.0.0\nCheck out the beta of the new CLI by adding the environment variable\n  \"NEW_CLI\".\nWe can't wait for your feedback.\nNAME:\n   monaco-linux-amd64 - Automates the deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.\nUSAGE:\n   monaco-linux-amd64 [global options] command [command options] [working directory]\nVERSION:\n   1.7.0\nDESCRIPTION:\n   Tool used to deploy dynatrace configurations via the cli\n   Examples:\n     Deploy a specific project inside a root config folder:\n       monaco -p='project-folder' -e='environments.yaml' projects-root-folder\n     Deploy a specific project to a specific tenant:\n       monaco --environments environments.yaml --specific-environment dev --project myProject\nCOMMANDS:\n   help, h  Shows a list of commands or help for one command\nGLOBAL OPTIONS:\n   --verbose, -v                             (default: false)\n   --environments value, -e value            Yaml file containing environments to deploy to\n   --specific-environment value, --se value  Specific environment (from list) to deploy to (default: none)\n   --project value, -p value                 Project configuration to deploy (also deploys any dependent configurations) (default: none)\n   --dry-run, -d                             Switches to just validation instead of actual deployment (default: false)\n   --continue-on-error, -c                   Proceed deployment even if config upload fails (default: false)\n   --help, -h                                show help (default: false)\n   --version                                 print the version (default: false)\n"))),(0,a.kt)(l.Z,{value:"windows",mdxType:"TabItem"},(0,a.kt)("p",null,"Before you start, you need to set the PATH on Windows: "),(0,a.kt)("ol",null,(0,a.kt)("li",{parentName:"ol"},"Go to Control Panel -> System -> System settings -> Environment Variables."),(0,a.kt)("li",{parentName:"ol"},"Scroll down in system variables until you find PATH."),(0,a.kt)("li",{parentName:"ol"},"Click edit and change accordingly."),(0,a.kt)("li",{parentName:"ol"},"Include a semicolon at the end of the previous as that is the delimiter, i.e., c:\\path;c:\\path2"),(0,a.kt)("li",{parentName:"ol"},"Launch a new console for the settings to take effect.")),(0,a.kt)("p",null,"Once your PATH is set, verify the installation by running ",(0,a.kt)("inlineCode",{parentName:"p"},"monaco")," from your terminal. "),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-shell"}," monaco\nYYou are currently using the old CLI structure which will be used by\ndefault until monaco version 2.0.0\n\nCheck out the beta of the new CLI by adding the environment variable\n  \"NEW_CLI\".\n\nWe can't wait for your feedback.\n\nNAME:\n   monaco.exe - Automates the deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.\n\nUSAGE:\n   monaco.exe [global options] command [command options] [working directory]\n\nVERSION:\n   1.7.0\n\nDESCRIPTION:\n   Tool used to deploy dynatrace configurations via the cli\n\n   Examples:\n     Deploy a specific project inside a root config folder:\n       monaco -p='project-folder' -e='environments.yaml' projects-root-folder\n\n     Deploy a specific project to a specific tenant:\n       monaco --environments environments.yaml --specific-environment dev --project myProject\n\nCOMMANDS:\n   help, h  Shows a list of commands or help for one command\n\nGLOBAL OPTIONS:\n   --verbose, -v                             (default: false)\n   --environments value, -e value            Yaml file containing environments to deploy to\n   --specific-environment value, --se value  Specific environment (from list) to deploy to (default: none)\n   --project value, -p value                 Project configuration to deploy (also deploys any dependent configurations) (default: none)\n   --dry-run, -d                             Switches to just validation instead of actual deployment (default: false)\n   --continue-on-error, -c                   Proceed deployment even if config upload fails (default: false)\n   --help, -h                                show help (default: false)\n   --version                                 print the version (default: false)\n")))))),(0,a.kt)("p",null,"Now that Monaco is installed, follow our introductory guide on ",(0,a.kt)("a",{parentName:"p",href:"../configuration/deploy_configuration"},"how to deploy a configuration to Dynatrace.")))}p.isMDXComponent=!0}}]);