// Package js generated by "lib/js/generate"
package js

// Element ...
var Element = &Function{
	Name:         "element",
	Definition:   `function(e){const t=functions.selectable(this);return t.querySelector(e)}`,
	Dependencies: []*Function{Selectable},
}

// Elements ...
var Elements = &Function{
	Name:         "elements",
	Definition:   `function(e){return functions.selectable(this).querySelectorAll(e)}`,
	Dependencies: []*Function{Selectable},
}

// ElementX ...
var ElementX = &Function{
	Name:         "elementX",
	Definition:   `function(e){var t=functions.selectable(this);return document.evaluate(e,t,null,XPathResult.FIRST_ORDERED_NODE_TYPE).singleNodeValue}`,
	Dependencies: []*Function{Selectable},
}

// ElementsX ...
var ElementsX = &Function{
	Name:         "elementsX",
	Definition:   `function(e){var t,n=functions.selectable(this);const i=document.evaluate(e,n,null,XPathResult.ORDERED_NODE_ITERATOR_TYPE),r=[];for(;t=i.iterateNext();)r.push(t);return r}`,
	Dependencies: []*Function{Selectable},
}

// ElementR ...
var ElementR = &Function{
	Name:         "elementR",
	Definition:   `function(e,t){var n=t.match(/(\/?)(.+)\1([a-z]*)/i),i=n[3]&&!/^(?!.*?(.).*?\1)[gmixXsuUAJ]+$/.test(n[3])?new RegExp(t):new RegExp(n[2],n[3]);const r=functions.selectable(this);e=Array.from(r.querySelectorAll(e)).find(e=>i.test(functions.text.call(e)));return e||null}`,
	Dependencies: []*Function{Selectable, Text},
}

// Parents ...
var Parents = &Function{
	Name:         "parents",
	Definition:   `function(e){let t=this.parentElement;const n=[];for(;t;)t.matches(e)&&n.push(t),t=t.parentElement;return n}`,
	Dependencies: []*Function{},
}

// ContainsElement ...
var ContainsElement = &Function{
	Name:         "containsElement",
	Definition:   `function(e){for(var t=e;null!=t;){if(t===this)return!0;t=t.parentElement}return!1}`,
	Dependencies: []*Function{},
}

// InitMouseTracer ...
var InitMouseTracer = &Function{
	Name:         "initMouseTracer",
	Definition:   `async function(e,t){if(await functions.waitLoad(),!document.getElementById(e)){const n=document.createElement("div");n.innerHTML=t;const i=n.lastChild;i.id=e,i.style="position: absolute; z-index: 2147483647; width: 17px; pointer-events: none;",i.removeAttribute("width"),i.removeAttribute("height"),document.body.parentElement.appendChild(i)}}`,
	Dependencies: []*Function{WaitLoad},
}

// UpdateMouseTracer ...
var UpdateMouseTracer = &Function{
	Name:         "updateMouseTracer",
	Definition:   `function(e,t,n){const i=document.getElementById(e);return!!i&&(i.style.left=t-2+"px",i.style.top=n-3+"px",!0)}`,
	Dependencies: []*Function{},
}

// Rect ...
var Rect = &Function{
	Name:         "rect",
	Definition:   `function(){var e=functions.tag(this).getBoundingClientRect();return{x:e.x,y:e.y,width:e.width,height:e.height}}`,
	Dependencies: []*Function{Tag},
}

// Overlay ...
var Overlay = &Function{
	Name: "overlay",
	Definition: `async function(e,t,n,i,r,o){await functions.waitLoad();const s=document.createElement("div");if(s.id=e,s.style=` + "`" + `position: fixed; z-index:2147483647; border: 2px dashed red;
        border-radius: 3px; box-shadow: #5f3232 0 0 3px; pointer-events: none;
        box-sizing: border-box;
        left: ${t}px;
        top: ${n}px;
        height: ${r}px;
        width: ${i}px;` + "`" + `,i*r==0&&(s.style.border="none"),o){const a=document.createElement("div");a.style=` + "`" + `position: absolute; color: #cc26d6; font-size: 12px; background: #ffffffeb;
        box-shadow: #333 0 0 3px; padding: 2px 5px; border-radius: 3px; white-space: nowrap;
        top: ${r}px;` + "`" + `,a.innerHTML=o,s.appendChild(a),document.body.parentElement.appendChild(s),window.innerHeight<a.offsetHeight+n+r&&(a.style.top=-a.offsetHeight-2+"px"),window.innerWidth<a.offsetWidth+t&&(a.style.left=window.innerWidth-a.offsetWidth-t+"px")}else document.body.parentElement.appendChild(s)}`,
	Dependencies: []*Function{WaitLoad},
}

// ElementOverlay ...
var ElementOverlay = &Function{
	Name:         "elementOverlay",
	Definition:   `async function(n,e){const i=100,r=functions.tag(this);let o=r.getBoundingClientRect();await functions.overlay(n,o.left,o.top,o.width,o.height,e);const s=()=>{const e=document.getElementById(n);var t;null!==e&&(t=r.getBoundingClientRect(),o.left===t.left&&o.top===t.top&&o.width===t.width&&o.height===t.height||(e.style.left=t.left+"px",e.style.top=t.top+"px",e.style.width=t.width+"px",e.style.height=t.height+"px",o=t),setTimeout(s,i))};setTimeout(s,i)}`,
	Dependencies: []*Function{Tag, Overlay},
}

// RemoveOverlay ...
var RemoveOverlay = &Function{
	Name:         "removeOverlay",
	Definition:   `function(e){e=document.getElementById(e);e&&Element.prototype.remove.call(e)}`,
	Dependencies: []*Function{},
}

// WaitIdle ...
var WaitIdle = &Function{
	Name:         "waitIdle",
	Definition:   `function(t){return new Promise(e=>{window.requestIdleCallback(e,{timeout:t})})}`,
	Dependencies: []*Function{},
}

// WaitLoad ...
var WaitLoad = &Function{
	Name:         "waitLoad",
	Definition:   `function(){const n=this===window;return new Promise((e,t)=>{if(n){if("complete"===document.readyState)return e();window.addEventListener("load",e)}else void 0===this.complete||this.complete?e():(this.addEventListener("load",e),this.addEventListener("error",t))})}`,
	Dependencies: []*Function{},
}

// InputEvent ...
var InputEvent = &Function{
	Name:         "inputEvent",
	Definition:   `function(){this.dispatchEvent(new Event("input",{bubbles:!0})),this.dispatchEvent(new Event("change",{bubbles:!0}))}`,
	Dependencies: []*Function{},
}

// InputTime ...
var InputTime = &Function{
	Name:         "inputTime",
	Definition:   `function(e){const t=new Date(e);var e=e=>e.toString().padStart(2,"0"),n=t.getFullYear(),i=e(t.getMonth()+1),r=e(t.getDate()),o=e(t.getHours()),s=e(t.getMinutes());switch(this.type){case"date":this.value=n+` + "`" + `-${i}-` + "`" + `+r;break;case"datetime-local":this.value=n+` + "`" + `-${i}-${r}T${o}:` + "`" + `+s;break;case"month":this.value=i;break;case"time":this.value=o+":"+s}functions.inputEvent.call(this)}`,
	Dependencies: []*Function{InputEvent},
}

// SelectText ...
var SelectText = &Function{
	Name:         "selectText",
	Definition:   `function(e){e=this.value.match(new RegExp(e));e&&this.setSelectionRange(e.index,e.index+e[0].length)}`,
	Dependencies: []*Function{},
}

// TriggerFavicon ...
var TriggerFavicon = &Function{
	Name:         "triggerFavicon",
	Definition:   `function(){return new Promise((e,t)=>{var n=document.querySelector("link[rel~=icon]"),n=n&&n.href||"/favicon.ico",n=new URL(n,window.location).toString();const r=new XMLHttpRequest;r.open("GET",n),r.ontimeout=function(){t({errorType:"timeout_error",xhr:r})},r.onreadystatechange=function(){4===r.readyState&&(200<=r.status&&r.status<300||304===r.status?e(r.responseText):t({errorType:"status_error",xhr:r,status:r.status,statusText:r.statusText,responseText:r.responseText}))},r.onerror=function(){t({errorType:"onerror",xhr:r,status:r.status,statusText:r.statusText,responseText:r.responseText})},r.send()})}`,
	Dependencies: []*Function{},
}

// SelectAllText ...
var SelectAllText = &Function{
	Name:         "selectAllText",
	Definition:   `function(){this.select()}`,
	Dependencies: []*Function{},
}

// Select ...
var Select = &Function{
	Name:         "select",
	Definition:   `function(e,n,t){let i;switch(t){case"regex":i=e.map(e=>{const t=new RegExp(e);return e=>t.test(e.innerText)});break;case"css-selector":i=e.map(t=>e=>e.matches(t));break;default:i=e.map(t=>e=>e.innerText.includes(t))}const r=Array.from(this.options);let o=!1;return i.forEach(e=>{const t=r.find(e);t&&(t.selected=n,o=!0)}),this.dispatchEvent(new Event("input",{bubbles:!0})),this.dispatchEvent(new Event("change",{bubbles:!0})),o}`,
	Dependencies: []*Function{},
}

// Visible ...
var Visible = &Function{
	Name:         "visible",
	Definition:   `function(){const e=functions.tag(this);var t=e.getBoundingClientRect(),n=window.getComputedStyle(e);return"none"!==n.display&&"hidden"!==n.visibility&&!!(t.top||t.bottom||t.width||t.height)}`,
	Dependencies: []*Function{Tag},
}

// Invisible ...
var Invisible = &Function{
	Name:         "invisible",
	Definition:   `function(){return!functions.visible.apply(this)}`,
	Dependencies: []*Function{Visible},
}

// Text ...
var Text = &Function{
	Name:         "text",
	Definition:   `function(){switch(this.tagName){case"INPUT":case"TEXTAREA":return this.value||this.placeholder;case"SELECT":return Array.from(this.selectedOptions).map(e=>e.innerText).join();case void 0:return this.textContent;default:return this.innerText}}`,
	Dependencies: []*Function{},
}

// Resource ...
var Resource = &Function{
	Name:         "resource",
	Definition:   `function(){return new Promise((e,t)=>{if(this.complete)return e(this.currentSrc);this.addEventListener("load",()=>e(this.currentSrc)),this.addEventListener("error",e=>t(e))})}`,
	Dependencies: []*Function{},
}

// AddScriptTag ...
var AddScriptTag = &Function{
	Name:         "addScriptTag",
	Definition:   `function(i,r,o){if(!document.getElementById(i))return new Promise((e,t)=>{var n=document.createElement("script");r?(n.src=r,n.onload=e):(n.type="text/javascript",n.text=o,e()),n.id=i,n.onerror=t,document.head.appendChild(n)})}`,
	Dependencies: []*Function{},
}

// AddStyleTag ...
var AddStyleTag = &Function{
	Name:         "addStyleTag",
	Definition:   `function(i,r,o){if(!document.getElementById(i))return new Promise((e,t)=>{var n;r?((n=document.createElement("link")).rel="stylesheet",n.href=r):((n=document.createElement("style")).type="text/css",n.appendChild(document.createTextNode(o)),e()),n.id=i,n.onload=e,n.onerror=t,document.head.appendChild(n)})}`,
	Dependencies: []*Function{},
}

// Selectable ...
var Selectable = &Function{
	Name:         "selectable",
	Definition:   `function(e){return e.querySelector?e:document}`,
	Dependencies: []*Function{},
}

// Tag ...
var Tag = &Function{
	Name:         "tag",
	Definition:   `function(e){return e.tagName?e:e.parentElement}`,
	Dependencies: []*Function{},
}

// ExposeFunc ...
var ExposeFunc = &Function{
	Name:         "exposeFunc",
	Definition:   `function(e,t){let o=0;window[e]=e=>new Promise((n,i)=>{const r=t+"_cb"+o++;window[r]=(e,t)=>{delete window[r],t?i(t):n(e)},window[t](JSON.stringify({req:e,cb:r}))})}`,
	Dependencies: []*Function{},
}

// GetXPath ...
var GetXPath = &Function{
	Name:         "getXPath",
	Definition:   `function(e){class r{constructor(e,t){this.value=e,this.optimized=t||!1}toString(){return this.value}}function o(t){function n(e,t){return e===t||(e.nodeType===Node.ELEMENT_NODE&&t.nodeType===Node.ELEMENT_NODE?e.localName===t.localName:e.nodeType===t.nodeType||(e.nodeType===Node.CDATA_SECTION_NODE?Node.TEXT_NODE:e.nodeType)===(t.nodeType===Node.CDATA_SECTION_NODE?Node.TEXT_NODE:t.nodeType))}var e=t.parentNode,i=e?e.children:null;if(!i)return 0;let r;for(let e=0;e<i.length;++e)if(n(t,i[e])&&i[e]!==t){r=!0;break}if(!r)return 0;let o=1;for(let e=0;e<i.length;++e)if(n(t,i[e])){if(i[e]===t)return o;++o}return-1}if(this.nodeType===Node.DOCUMENT_NODE)return"/";const t=[];let n=this;for(;n;){var i=function(e,t){let n;var i=o(e);if(-1===i)return null;switch(e.nodeType){case Node.ELEMENT_NODE:if(t&&e.id)return new r(` + "`" + `//*[@id='${e.id}']` + "`" + `,!0);n=e.localName;break;case Node.ATTRIBUTE_NODE:n="@"+e.nodeName;break;case Node.TEXT_NODE:case Node.CDATA_SECTION_NODE:n="text()";break;case Node.PROCESSING_INSTRUCTION_NODE:n="processing-instruction()";break;case Node.COMMENT_NODE:n="comment()";break;default:Node.DOCUMENT_NODE;n=""}return 0<i&&(n+=` + "`" + `[${i}]` + "`" + `),new r(n,e.nodeType===Node.DOCUMENT_NODE)}(n,e);if(!i)break;if(t.push(i),i.optimized)break;n=n.parentNode}return t.reverse(),(t.length&&t[0].optimized?"":"/")+t.join("/")}`,
	Dependencies: []*Function{},
}