package schema

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/tags"
)

const (
	BOOLEAN = "boolean"
	NUMBER  = "number"
	STRING  = "string"
	OBJECT  = "object"
)

func normalizeTagName(tagName string) string {
	tagNameLower := strings.ToLower(tagName)
	ns, name, _ := tags.SplitNsName(tagNameLower, false)

	if ns != nil && (*ns == core.SVG_NAMESPACE || *ns == core.MATH_ML_NAMESPACE) {
		return fmt.Sprintf(":%s:%s", *ns, name)
	}
	return name
}

var SCHEMA = []string{
	"[Element]|textContent,%ariaActiveDescendantElement,%ariaAtomic,%ariaAutoComplete,%ariaBusy,%ariaChecked,%ariaColCount,%ariaColIndex,%ariaColIndexText,%ariaColSpan,%ariaControlsElements,%ariaCurrent,%ariaDescribedByElements,%ariaDescription,%ariaDetailsElements,%ariaDisabled,%ariaErrorMessageElements,%ariaExpanded,%ariaFlowToElements,%ariaHasPopup,%ariaHidden,%ariaInvalid,%ariaKeyShortcuts,%ariaLabel,%ariaLabelledByElements,%ariaLevel,%ariaLive,%ariaModal,%ariaMultiLine,%ariaMultiSelectable,%ariaOrientation,%ariaOwnsElements,%ariaPlaceholder,%ariaPosInSet,%ariaPressed,%ariaReadOnly,%ariaRelevant,%ariaRequired,%ariaRoleDescription,%ariaRowCount,%ariaRowIndex,%ariaRowIndexText,%ariaRowSpan,%ariaSelected,%ariaSetSize,%ariaSort,%ariaValueMax,%ariaValueMin,%ariaValueNow,%ariaValueText,%classList,className,elementTiming,id,innerHTML,*beforecopy,*beforecut,*beforepaste,*fullscreenchange,*fullscreenerror,*search,*webkitfullscreenchange,*webkitfullscreenerror,outerHTML,%part,#scrollLeft,#scrollTop,slot,*message,*mozfullscreenchange,*mozfullscreenerror,*mozpointerlockchange,*mozpointerlockerror,*webglcontextcreationerror,*webglcontextlost,*webglcontextrestored",
	"[HTMLElement]^[Element]|accessKey,autocapitalize,!autofocus,contentEditable,dir,!draggable,enterKeyHint,!hidden,!inert,innerText,inputMode,lang,nonce,*abort,*animationend,*animationiteration,*animationstart,*auxclick,*beforexrselect,*blur,*cancel,*canplay,*canplaythrough,*change,*click,*close,*contextmenu,*copy,*cuechange,*cut,*dblclick,*drag,*dragend,*dragenter,*dragleave,*dragover,*dragstart,*drop,*durationchange,*emptied,*ended,*error,*focus,*formdata,*gotpointercapture,*input,*invalid,*keydown,*keypress,*keyup,*load,*loadeddata,*loadedmetadata,*loadstart,*lostpointercapture,*mousedown,*mouseenter,*mouseleave,*mousemove,*mouseout,*mouseover,*mouseup,*mousewheel,*paste,*pause,*play,*playing,*pointercancel,*pointerdown,*pointerenter,*pointerleave,*pointermove,*pointerout,*pointerover,*pointerrawupdate,*pointerup,*progress,*ratechange,*reset,*resize,*scroll,*securitypolicyviolation,*seeked,*seeking,*select,*selectionchange,*selectstart,*slotchange,*stalled,*submit,*suspend,*timeupdate,*toggle,*transitioncancel,*transitionend,*transitionrun,*transitionstart,*volumechange,*waiting,*webkitanimationend,*webkitanimationiteration,*webkitanimationstart,*webkittransitionend,*wheel,outerText,!spellcheck,%style,#tabIndex,title,!translate,virtualKeyboardPolicy",
	"abbr,address,article,aside,b,bdi,bdo,cite,content,code,dd,dfn,dt,em,figcaption,figure,footer,header,hgroup,i,kbd,main,mark,nav,noscript,rb,rp,rt,rtc,ruby,s,samp,search,section,small,strong,sub,sup,u,var,wbr^[HTMLElement]|accessKey,autocapitalize,!autofocus,contentEditable,dir,!draggable,enterKeyHint,!hidden,innerText,inputMode,lang,nonce,*abort,*animationend,*animationiteration,*animationstart,*auxclick,*beforexrselect,*blur,*cancel,*canplay,*canplaythrough,*change,*click,*close,*contextmenu,*copy,*cuechange,*cut,*dblclick,*drag,*dragend,*dragenter,*dragleave,*dragover,*dragstart,*drop,*durationchange,*emptied,*ended,*error,*focus,*formdata,*gotpointercapture,*input,*invalid,*keydown,*keypress,*keyup,*load,*loadeddata,*loadedmetadata,*loadstart,*lostpointercapture,*mousedown,*mouseenter,*mouseleave,*mousemove,*mouseout,*mouseover,*mouseup,*mousewheel,*paste,*pause,*play,*playing,*pointercancel,*pointerdown,*pointerenter,*pointerleave,*pointermove,*pointerout,*pointerover,*pointerrawupdate,*pointerup,*progress,*ratechange,*reset,*resize,*scroll,*securitypolicyviolation,*seeked,*seeking,*select,*selectionchange,*selectstart,*slotchange,*stalled,*submit,*suspend,*timeupdate,*toggle,*transitioncancel,*transitionend,*transitionrun,*transitionstart,*volumechange,*waiting,*webkitanimationend,*webkitanimationiteration,*webkitanimationstart,*webkittransitionend,*wheel,outerText,!spellcheck,%style,#tabIndex,title,!translate,virtualKeyboardPolicy",
	"media^[HTMLElement]|!autoplay,!controls,%controlsList,%crossOrigin,#currentTime,!defaultMuted,#defaultPlaybackRate,!disableRemotePlayback,!loop,!muted,*encrypted,*waitingforkey,#playbackRate,preload,!preservesPitch,src,%srcObject,#volume",
	":svg:^[HTMLElement]|!autofocus,nonce,*abort,*animationend,*animationiteration,*animationstart,*auxclick,*beforexrselect,*blur,*cancel,*canplay,*canplaythrough,*change,*click,*close,*contextmenu,*copy,*cuechange,*cut,*dblclick,*drag,*dragend,*dragenter,*dragleave,*dragover,*dragstart,*drop,*durationchange,*emptied,*ended,*error,*focus,*formdata,*gotpointercapture,*input,*invalid,*keydown,*keypress,*keyup,*load,*loadeddata,*loadedmetadata,*loadstart,*lostpointercapture,*mousedown,*mouseenter,*mouseleave,*mousemove,*mouseout,*mouseover,*mouseup,*mousewheel,*paste,*pause,*play,*playing,*pointercancel,*pointerdown,*pointerenter,*pointerleave,*pointermove,*pointerout,*pointerover,*pointerrawupdate,*pointerup,*progress,*ratechange,*reset,*resize,*scroll,*securitypolicyviolation,*seeked,*seeking,*select,*selectionchange,*selectstart,*slotchange,*stalled,*submit,*suspend,*timeupdate,*toggle,*transitioncancel,*transitionend,*transitionrun,*transitionstart,*volumechange,*waiting,*webkitanimationend,*webkitanimationiteration,*webkitanimationstart,*webkittransitionend,*wheel,%style,#tabIndex",
	":svg:graphics^:svg:|",
	":svg:animation^:svg:|*begin,*end,*repeat",
	":svg:geometry^:svg:|",
	":svg:componentTransferFunction^:svg:|",
	":svg:gradient^:svg:|",
	":svg:textContent^:svg:graphics|",
	":svg:textPositioning^:svg:textContent|",
	"a^[HTMLElement]|charset,coords,download,hash,host,hostname,href,hreflang,name,password,pathname,ping,port,protocol,referrerPolicy,rel,%relList,rev,search,shape,target,text,type,username",
	"area^[HTMLElement]|alt,coords,download,hash,host,hostname,href,!noHref,password,pathname,ping,port,protocol,referrerPolicy,rel,%relList,search,shape,target,username",
	"audio^media|",
	"br^[HTMLElement]|clear",
	"base^[HTMLElement]|href,target",
	"body^[HTMLElement]|aLink,background,bgColor,link,*afterprint,*beforeprint,*beforeunload,*blur,*error,*focus,*hashchange,*languagechange,*load,*message,*messageerror,*offline,*online,*pagehide,*pageshow,*popstate,*rejectionhandled,*resize,*scroll,*storage,*unhandledrejection,*unload,text,vLink",
	"button^[HTMLElement]|!disabled,formAction,formEnctype,formMethod,!formNoValidate,formTarget,name,type,value",
	"canvas^[HTMLElement]|#height,#width",
	"content^[HTMLElement]|select",
	"dl^[HTMLElement]|!compact",
	"data^[HTMLElement]|value",
	"datalist^[HTMLElement]|",
	"details^[HTMLElement]|!open",
	"dialog^[HTMLElement]|!open,returnValue",
	"dir^[HTMLElement]|!compact",
	"div^[HTMLElement]|align",
	"embed^[HTMLElement]|align,height,name,src,type,width",
	"fieldset^[HTMLElement]|!disabled,name",
	"font^[HTMLElement]|color,face,size",
	"form^[HTMLElement]|acceptCharset,action,autocomplete,encoding,enctype,method,name,!noValidate,target",
	"frame^[HTMLElement]|frameBorder,longDesc,marginHeight,marginWidth,name,!noResize,scrolling,src",
	"frameset^[HTMLElement]|cols,*afterprint,*beforeprint,*beforeunload,*blur,*error,*focus,*hashchange,*languagechange,*load,*message,*messageerror,*offline,*online,*pagehide,*pageshow,*popstate,*rejectionhandled,*resize,*scroll,*storage,*unhandledrejection,*unload,rows",
	"geolocation^[HTMLElement]|accuracymode,!autolocate,*location,*promptaction,*promptdismiss,*validationstatuschange,!watch",
	"hr^[HTMLElement]|align,color,!noShade,size,width",
	"head^[HTMLElement]|",
	"h1,h2,h3,h4,h5,h6^[HTMLElement]|align",
	"html^[HTMLElement]|version",
	"iframe^[HTMLElement]|align,allow,!allowFullscreen,!allowPaymentRequest,csp,frameBorder,height,loading,longDesc,marginHeight,marginWidth,name,referrerPolicy,%sandbox,scrolling,src,srcdoc,width",
	"img^[HTMLElement]|align,alt,border,%crossOrigin,decoding,#height,#hspace,!isMap,loading,longDesc,lowsrc,name,referrerPolicy,sizes,src,srcset,useMap,#vspace,#width",
	"input^[HTMLElement]|accept,align,alt,autocomplete,!checked,!defaultChecked,defaultValue,dirName,!disabled,%files,formAction,formEnctype,formMethod,!formNoValidate,formTarget,#height,!incremental,!indeterminate,max,#maxLength,min,#minLength,!multiple,name,pattern,placeholder,!readOnly,!required,selectionDirection,#selectionEnd,#selectionStart,#size,src,step,type,useMap,value,%valueAsDate,#valueAsNumber,#width",
	"li^[HTMLElement]|type,#value",
	"label^[HTMLElement]|htmlFor",
	"legend^[HTMLElement]|align",
	"link^[HTMLElement]|as,charset,%crossOrigin,!disabled,href,hreflang,imageSizes,imageSrcset,integrity,media,referrerPolicy,rel,%relList,rev,%sizes,target,type",
	"map^[HTMLElement]|name",
	"marquee^[HTMLElement]|behavior,bgColor,direction,height,#hspace,#loop,#scrollAmount,#scrollDelay,!trueSpeed,#vspace,width",
	"menu^[HTMLElement]|!compact",
	"meta^[HTMLElement]|content,httpEquiv,media,name,scheme",
	"meter^[HTMLElement]|#high,#low,#max,#min,#optimum,#value",
	"ins,del^[HTMLElement]|cite,dateTime",
	"ol^[HTMLElement]|!compact,!reversed,#start,type",
	"object^[HTMLElement]|align,archive,border,code,codeBase,codeType,data,!declare,height,#hspace,name,standby,type,useMap,#vspace,width",
	"optgroup^[HTMLElement]|!disabled,label",
	"option^[HTMLElement]|!defaultSelected,!disabled,label,!selected,text,value",
	"output^[HTMLElement]|defaultValue,%htmlFor,name,value",
	"p^[HTMLElement]|align",
	"param^[HTMLElement]|name,type,value,valueType",
	"picture^[HTMLElement]|",
	"pre^[HTMLElement]|#width",
	"progress^[HTMLElement]|#max,#value",
	"q,blockquote,cite^[HTMLElement]|",
	"script^[HTMLElement]|!async,charset,%crossOrigin,!defer,event,htmlFor,integrity,!noModule,%referrerPolicy,src,text,type",
	"select^[HTMLElement]|autocomplete,!disabled,#length,!multiple,name,!required,#selectedIndex,#size,value",
	"selectedcontent^[HTMLElement]|",
	"slot^[HTMLElement]|name",
	"source^[HTMLElement]|#height,media,sizes,src,srcset,type,#width",
	"span^[HTMLElement]|",
	"style^[HTMLElement]|!disabled,media,type",
	"search^[HTMLELement]|",
	"caption^[HTMLElement]|align",
	"th,td^[HTMLElement]|abbr,align,axis,bgColor,ch,chOff,#colSpan,headers,height,!noWrap,#rowSpan,scope,vAlign,width",
	"col,colgroup^[HTMLElement]|align,ch,chOff,#span,vAlign,width",
	"table^[HTMLElement]|align,bgColor,border,%caption,cellPadding,cellSpacing,frame,rules,summary,%tFoot,%tHead,width",
	"tr^[HTMLElement]|align,bgColor,ch,chOff,vAlign",
	"tfoot,thead,tbody^[HTMLElement]|align,ch,chOff,vAlign",
	"template^[HTMLElement]|",
	"textarea^[HTMLElement]|autocomplete,#cols,defaultValue,dirName,!disabled,#maxLength,#minLength,name,placeholder,!readOnly,!required,#rows,selectionDirection,#selectionEnd,#selectionStart,value,wrap",
	"time^[HTMLElement]|dateTime",
	"title^[HTMLElement]|text",
	"track^[HTMLElement]|!default,kind,label,src,srclang",
	"ul^[HTMLElement]|!compact,type",
	"unknown^[HTMLElement]|",
	"video^media|!disablePictureInPicture,#height,*enterpictureinpicture,*leavepictureinpicture,!playsInline,poster,#width",
	":svg:a^:svg:graphics|",
	":svg:animate^:svg:animation|",
	":svg:animateMotion^:svg:animation|",
	":svg:animateTransform^:svg:animation|",
	":svg:circle^:svg:geometry|",
	":svg:clipPath^:svg:graphics|",
	":svg:defs^:svg:graphics|",
	":svg:desc^:svg:|",
	":svg:discard^:svg:|",
	":svg:ellipse^:svg:geometry|",
	":svg:feBlend^:svg:|",
	":svg:feColorMatrix^:svg:|",
	":svg:feComponentTransfer^:svg:|",
	":svg:feComposite^:svg:|",
	":svg:feConvolveMatrix^:svg:|",
	":svg:feDiffuseLighting^:svg:|",
	":svg:feDisplacementMap^:svg:|",
	":svg:feDistantLight^:svg:|",
	":svg:feDropShadow^:svg:|",
	":svg:feFlood^:svg:|",
	":svg:feFuncA^:svg:componentTransferFunction|",
	":svg:feFuncB^:svg:componentTransferFunction|",
	":svg:feFuncG^:svg:componentTransferFunction|",
	":svg:feFuncR^:svg:componentTransferFunction|",
	":svg:feGaussianBlur^:svg:|",
	":svg:feImage^:svg:|",
	":svg:feMerge^:svg:|",
	":svg:feMergeNode^:svg:|",
	":svg:feMorphology^:svg:|",
	":svg:feOffset^:svg:|",
	":svg:fePointLight^:svg:|",
	":svg:feSpecularLighting^:svg:|",
	":svg:feSpotLight^:svg:|",
	":svg:feTile^:svg:|",
	":svg:feTurbulence^:svg:|",
	":svg:filter^:svg:|",
	":svg:foreignObject^:svg:graphics|",
	":svg:g^:svg:graphics|",
	":svg:image^:svg:graphics|decoding",
	":svg:line^:svg:geometry|",
	":svg:linearGradient^:svg:gradient|",
	":svg:mpath^:svg:|",
	":svg:marker^:svg:|",
	":svg:mask^:svg:|",
	":svg:metadata^:svg:|",
	":svg:path^:svg:geometry|",
	":svg:pattern^:svg:|",
	":svg:polygon^:svg:geometry|",
	":svg:polyline^:svg:geometry|",
	":svg:radialGradient^:svg:gradient|",
	":svg:rect^:svg:geometry|",
	":svg:svg^:svg:graphics|#currentScale,#zoomAndPan",
	":svg:script^:svg:|type",
	":svg:set^:svg:animation|",
	":svg:stop^:svg:|",
	":svg:style^:svg:|!disabled,media,title,type",
	":svg:switch^:svg:graphics|",
	":svg:symbol^:svg:|",
	":svg:tspan^:svg:textPositioning|",
	":svg:text^:svg:textPositioning|",
	":svg:textPath^:svg:textContent|",
	":svg:title^:svg:|",
	":svg:use^:svg:graphics|",
	":svg:view^:svg:|#zoomAndPan",
	"data^[HTMLElement]|value",
	"keygen^[HTMLElement]|!autofocus,challenge,!disabled,form,keytype,name",
	"menuitem^[HTMLElement]|type,label,icon,!disabled,!checked,radiogroup,!default",
	"summary^[HTMLElement]|",
	"time^[HTMLElement]|dateTime",
	":svg:cursor^:svg:|",
	":math:^[HTMLElement]|!autofocus,nonce,*abort,*animationend,*animationiteration,*animationstart,*auxclick,*beforeinput,*beforematch,*beforetoggle,*beforexrselect,*blur,*cancel,*canplay,*canplaythrough,*change,*click,*close,*contentvisibilityautostatechange,*contextlost,*contextmenu,*contextrestored,*copy,*cuechange,*cut,*dblclick,*drag,*dragend,*dragenter,*dragleave,*dragover,*dragstart,*drop,*durationchange,*emptied,*ended,*error,*focus,*formdata,*gotpointercapture,*input,*invalid,*keydown,*keypress,*keyup,*load,*loadeddata,*loadedmetadata,*loadstart,*lostpointercapture,*mousedown,*mouseenter,*mouseleave,*mousemove,*mouseout,*mouseover,*mouseup,*mousewheel,*paste,*pause,*play,*playing,*pointercancel,*pointerdown,*pointerenter,*pointerleave,*pointermove,*pointerout,*pointerover,*pointerrawupdate,*pointerup,*progress,*ratechange,*reset,*resize,*scroll,*scrollend,*securitypolicyviolation,*seeked,*seeking,*select,*selectionchange,*selectstart,*slotchange,*stalled,*submit,*suspend,*timeupdate,*toggle,*transitioncancel,*transitionend,*transitionrun,*transitionstart,*volumechange,*waiting,*webkitanimationend,*webkitanimationiteration,*webkitanimationstart,*webkittransitionend,*wheel,%style,#tabIndex",
	":math:math^:math:|",
	":math:maction^:math:|",
	":math:menclose^:math:|",
	":math:merror^:math:|",
	":math:mfenced^:math:|",
	":math:mfrac^:math:|",
	":math:mi^:math:|",
	":math:mmultiscripts^:math:|",
	":math:mn^:math:|",
	":math:mo^:math:|",
	":math:mover^:math:|",
	":math:mpadded^:math:|",
	":math:mphantom^:math:|",
	":math:mroot^:math:|",
	":math:mrow^:math:|",
	":math:ms^:math:|",
	":math:mspace^:math:|",
	":math:msqrt^:math:|",
	":math:mstyle^:math:|",
	":math:msub^:math:|",
	":math:msubsup^:math:|",
	":math:msup^:math:|",
	":math:mtable^:math:|",
	":math:mtd^:math:|",
	":math:mtext^:math:|",
	":math:mtr^:math:|",
	":math:munder^:math:|",
	":math:munderover^:math:|",
	":math:semantics^:math:|",
}

var ATTR_TO_PROP = map[string]string{
	"class":                 "className",
	"for":                   "htmlFor",
	"formaction":            "formAction",
	"innerHtml":             "innerHTML",
	"readonly":              "readOnly",
	"tabindex":              "tabIndex",
	"aria-activedescendant": "ariaActiveDescendantElement",
	"aria-atomic":           "ariaAtomic",
	"aria-autocomplete":     "ariaAutoComplete",
	"aria-busy":             "ariaBusy",
	"aria-checked":          "ariaChecked",
	"aria-colcount":         "ariaColCount",
	"aria-colindex":         "ariaColIndex",
	"aria-colindextext":     "ariaColIndexText",
	"aria-colspan":          "ariaColSpan",
	"aria-controls":         "ariaControlsElements",
	"aria-current":          "ariaCurrent",
	"aria-describedby":      "ariaDescribedByElements",
	"aria-description":      "ariaDescription",
	"aria-details":          "ariaDetailsElements",
	"aria-disabled":         "ariaDisabled",
	"aria-errormessage":     "ariaErrorMessageElements",
	"aria-expanded":         "ariaExpanded",
	"aria-flowto":           "ariaFlowToElements",
	"aria-haspopup":         "ariaHasPopup",
	"aria-hidden":           "ariaHidden",
	"aria-invalid":          "ariaInvalid",
	"aria-keyshortcuts":     "ariaKeyShortcuts",
	"aria-label":            "ariaLabel",
	"aria-labelledby":       "ariaLabelledByElements",
	"aria-level":            "ariaLevel",
	"aria-live":             "ariaLive",
	"aria-modal":            "ariaModal",
	"aria-multiline":        "ariaMultiLine",
	"aria-multiselectable":  "ariaMultiSelectable",
	"aria-orientation":      "ariaOrientation",
	"aria-owns":             "ariaOwnsElements",
	"aria-placeholder":      "ariaPlaceholder",
	"aria-posinset":         "ariaPosInSet",
	"aria-pressed":          "ariaPressed",
	"aria-readonly":         "ariaReadOnly",
	"aria-required":         "ariaRequired",
	"aria-roledescription":  "ariaRoleDescription",
	"aria-rowcount":         "ariaRowCount",
	"aria-rowindex":         "ariaRowIndex",
	"aria-rowindextext":     "ariaRowIndexText",
	"aria-rowspan":          "ariaRowSpan",
	"aria-selected":         "ariaSelected",
	"aria-setsize":          "ariaSetSize",
	"aria-sort":             "ariaSort",
	"aria-valuemax":         "ariaValueMax",
	"aria-valuemin":         "ariaValueMin",
	"aria-valuenow":         "ariaValueNow",
	"aria-valuetext":        "ariaValueText",
}

var PROP_TO_ATTR = make(map[string]string)

func init() {
	for k, v := range ATTR_TO_PROP {
		PROP_TO_ATTR[v] = k
	}
}

type DomElementSchemaRegistry struct {
	schema      map[string]map[string]string
	eventSchema map[string]map[string]bool
}

func NewDomElementSchemaRegistry() *DomElementSchemaRegistry {
	r := &DomElementSchemaRegistry{
		schema:      make(map[string]map[string]string),
		eventSchema: make(map[string]map[string]bool),
	}

	for _, encodedType := range SCHEMA {
		typeMap := make(map[string]string)
		events := make(map[string]bool)

		parts := strings.Split(encodedType, "|")
		strType := parts[0]
		strProperties := parts[1]
		properties := strings.Split(strProperties, ",")

		typeParts := strings.Split(strType, "^")
		typeNames := typeParts[0]
		var superName string
		if len(typeParts) > 1 {
			superName = typeParts[1]
		}

		for _, tag := range strings.Split(typeNames, ",") {
			r.schema[strings.ToLower(tag)] = typeMap
			r.eventSchema[strings.ToLower(tag)] = events
		}

		if superName != "" {
			superType, ok := r.schema[strings.ToLower(superName)]
			if ok {
				for prop, value := range superType {
					typeMap[prop] = value
				}
				for superEvent := range r.eventSchema[strings.ToLower(superName)] {
					events[superEvent] = true
				}
			}
		}

		for _, property := range properties {
			if len(property) > 0 {
				switch property[0] {
				case '*':
					events[property[1:]] = true
				case '!':
					typeMap[property[1:]] = BOOLEAN
				case '#':
					typeMap[property[1:]] = NUMBER
				case '%':
					typeMap[property[1:]] = OBJECT
				default:
					typeMap[property] = STRING
				}
			}
		}
	}
	return r
}

func (r *DomElementSchemaRegistry) HasProperty(tagName string, propName string, schemaMetas []core.SchemaMetadata) bool {
	for _, schema := range schemaMetas {
		if schema.Name == core.NoErrorsSchema.Name {
			return true
		}
	}

	normalizedTag := normalizeTagName(tagName)
	if strings.Contains(normalizedTag, "-") {
		if tags.IsNgContainer(normalizedTag) || tags.IsNgContent(normalizedTag) {
			return false
		}
		for _, schema := range schemaMetas {
			if schema.Name == core.CustomElementsSchema.Name {
				return true
			}
		}
	}

	elementProperties, ok := r.schema[normalizedTag]
	if !ok {
		elementProperties = r.schema["unknown"]
	}
	_, has := elementProperties[propName]
	return has
}

func (r *DomElementSchemaRegistry) HasElement(tagName string, schemaMetas []core.SchemaMetadata) bool {
	for _, schema := range schemaMetas {
		if schema.Name == core.NoErrorsSchema.Name {
			return true
		}
	}

	normalizedTag := normalizeTagName(tagName)
	if strings.Contains(normalizedTag, "-") {
		if tags.IsNgContainer(normalizedTag) || tags.IsNgContent(normalizedTag) {
			return true
		}
		for _, schema := range schemaMetas {
			if schema.Name == core.CustomElementsSchema.Name {
				return true
			}
		}
	}

	_, has := r.schema[normalizedTag]
	return has
}

func (r *DomElementSchemaRegistry) SecurityContext(tagName string, propName string, isAttribute bool) core.SecurityContext {
	if isAttribute {
		propName = r.GetMappedPropName(propName)
	}

	normalizedTag := normalizeTagName(tagName)
	propName = strings.ToLower(propName)

	securitySchema := SECURITY_SCHEMA()
	if ctx, ok := securitySchema[normalizedTag+"|"+propName]; ok {
		return ctx
	}
	if ctx, ok := securitySchema["*|"+propName]; ok {
		return ctx
	}
	return core.SecurityContextNone
}

func (r *DomElementSchemaRegistry) GetMappedPropName(propName string) string {
	if mapped, ok := ATTR_TO_PROP[propName]; ok {
		return mapped
	}
	return propName
}

func (r *DomElementSchemaRegistry) GetDefaultComponentElementName() string {
	return "ng-component"
}

func (r *DomElementSchemaRegistry) ValidateProperty(name string) ValidationResult {
	if strings.HasPrefix(strings.ToLower(name), "on") {
		msg := fmt.Sprintf("Binding to event property '%s' is disallowed for security reasons, please use (%s)=...\nIf '%s' is a directive input, make sure the directive is imported by the current module.", name, name[2:], name)
		return ValidationResult{Error: true, Msg: &msg}
	}
	return ValidationResult{Error: false}
}

func (r *DomElementSchemaRegistry) ValidateAttribute(name string) ValidationResult {
	if strings.HasPrefix(strings.ToLower(name), "on") {
		msg := fmt.Sprintf("Binding to event attribute '%s' is disallowed for security reasons, please use (%s)=...", name, name[2:])
		return ValidationResult{Error: true, Msg: &msg}
	}
	return ValidationResult{Error: false}
}

func (r *DomElementSchemaRegistry) AllKnownElementNames() []string {
	var keys []string
	for k := range r.schema {
		keys = append(keys, k)
	}
	return keys
}

func (r *DomElementSchemaRegistry) AllKnownAttributesOfElement(tagName string) []string {
	normalizedTag := normalizeTagName(tagName)
	elementProperties, ok := r.schema[normalizedTag]
	if !ok {
		elementProperties = r.schema["unknown"]
	}
	var attrs []string
	for prop := range elementProperties {
		if attr, ok := PROP_TO_ATTR[prop]; ok {
			attrs = append(attrs, attr)
		} else {
			attrs = append(attrs, prop)
		}
	}
	return attrs
}

func (r *DomElementSchemaRegistry) AllKnownEventsOfElement(tagName string) []string {
	normalizedTag := normalizeTagName(tagName)
	events, ok := r.eventSchema[normalizedTag]
	if !ok {
		return []string{}
	}
	var evtList []string
	for evt := range events {
		evtList = append(evtList, evt)
	}
	return evtList
}

func dashCaseToCamelCase(input string) string {
	parts := strings.Split(input, "-")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func (r *DomElementSchemaRegistry) NormalizeAnimationStyleProperty(propName string) string {
	return dashCaseToCamelCase(propName)
}

func isPixelDimensionStyle(prop string) bool {
	switch prop {
	case "width", "height", "minWidth", "minHeight", "maxWidth", "maxHeight",
		"left", "top", "bottom", "right", "fontSize", "outlineWidth",
		"outlineOffset", "paddingTop", "paddingLeft", "paddingBottom", "paddingRight",
		"marginTop", "marginLeft", "marginBottom", "marginRight",
		"borderRadius", "borderWidth", "borderTopWidth", "borderLeftWidth",
		"borderRightWidth", "borderBottomWidth", "textIndent":
		return true
	default:
		return false
	}
}

func (r *DomElementSchemaRegistry) NormalizeAnimationStyleValue(camelCaseProp string, userProvidedProp string, val any) AnimationStyleNormalizationResult {
	unit := ""
	strVal := strings.TrimSpace(fmt.Sprintf("%v", val))
	errorMsg := ""

	if isPixelDimensionStyle(camelCaseProp) && val != 0 && val != "0" {
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			unit = "px"
		default:
			re := regexp.MustCompile(`^[+-]?[\d\.]+([a-z]*)$`)
			matches := re.FindStringSubmatch(strVal)
			if matches != nil && len(matches[1]) == 0 {
				errorMsg = fmt.Sprintf("Please provide a CSS unit value for %s:%v", userProvidedProp, val)
			}
		}
	}

	return AnimationStyleNormalizationResult{Error: errorMsg, Value: strVal + unit}
}

var _ ElementSchemaRegistry = &DomElementSchemaRegistry{}
