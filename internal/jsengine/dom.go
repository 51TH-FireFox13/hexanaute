package jsengine

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"

	"github.com/51TH-FireFox13/hexanaute/internal/engine"
)

// setupDocument injecte l'objet document dans la VM.
func setupDocument(vm *goja.Runtime, root *engine.Element, baseURL string, result *ExecResult) {
	doc := vm.NewObject()

	// document.title getter/setter
	currentTitle := engine.CollectText(engine.FindByTag(root, "title"))
	doc.DefineAccessorProperty("title",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(currentTitle)
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			currentTitle = call.Argument(0).String()
			result.Title = currentTitle
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	// document.getElementById
	doc.Set("getElementById", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		el := engine.FindByID(root, id)
		if el == nil {
			return goja.Null()
		}
		return makeElement(vm, el, "#"+id, result)
	})

	// document.querySelector
	doc.Set("querySelector", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		el := engine.QuerySelector(root, sel)
		if el == nil {
			return goja.Null()
		}
		return makeElement(vm, el, selectorForElement(el, sel), result)
	})

	// document.querySelectorAll
	doc.Set("querySelectorAll", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		els := engine.QuerySelectorAll(root, sel)
		arr := vm.NewArray(len(els))
		for i, el := range els {
			arr.Set(fmt.Sprintf("%d", i), makeElement(vm, el, selectorForElement(el, sel), result))
		}
		arr.Set("length", vm.ToValue(len(els)))
		arr.Set("item", vm.ToValue(func(call goja.FunctionCall) goja.Value {
			idx := int(call.Argument(0).ToInteger())
			if idx >= 0 && idx < len(els) {
				return makeElement(vm, els[idx], selectorForElement(els[idx], sel), result)
			}
			return goja.Null()
		}))
		arr.Set("forEach", vm.ToValue(func(call goja.FunctionCall) goja.Value {
			fn, ok := goja.AssertFunction(call.Argument(0))
			if !ok {
				return goja.Undefined()
			}
			for i, el := range els {
				_, err := fn(goja.Undefined(),
					makeElement(vm, el, selectorForElement(el, sel), result),
					vm.ToValue(i),
				)
				if err != nil {
					break
				}
			}
			return goja.Undefined()
		}))
		return arr
	})

	// document.getElementsByTagName
	doc.Set("getElementsByTagName", func(call goja.FunctionCall) goja.Value {
		tag := strings.ToLower(call.Argument(0).String())
		els := engine.FindAllByTag(root, tag)
		return makeNodeList(vm, els, tag, result)
	})

	// document.getElementsByClassName
	doc.Set("getElementsByClassName", func(call goja.FunctionCall) goja.Value {
		class := call.Argument(0).String()
		els := engine.FindAllByClass(root, class)
		return makeNodeList(vm, els, "."+class, result)
	})

	// document.getElementsByName (formulaires)
	doc.Set("getElementsByName", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		var found []*engine.Element
		engine.QuerySelectorAll(root, "[name="+name+"]")
		_ = name
		return makeNodeList(vm, found, "", result)
	})

	// document.createElement — retourne un élément détaché
	doc.Set("createElement", func(call goja.FunctionCall) goja.Value {
		tag := strings.ToLower(call.Argument(0).String())
		el := &engine.Element{Tag: tag, Attrs: make(map[string]string)}
		return makeElement(vm, el, "", result)
	})

	// document.createTextNode
	doc.Set("createTextNode", func(call goja.FunctionCall) goja.Value {
		text := call.Argument(0).String()
		el := &engine.Element{Tag: "#text", Text: text}
		return makeElement(vm, el, "", result)
	})

	// document.body
	body := engine.FindByTag(root, "body")
	if body != nil {
		doc.Set("body", makeElement(vm, body, "body", result))
	} else {
		doc.Set("body", goja.Null())
	}

	// document.head
	head := engine.FindByTag(root, "head")
	if head != nil {
		doc.Set("head", makeElement(vm, head, "head", result))
	} else {
		doc.Set("head", goja.Null())
	}

	// document.documentElement
	doc.Set("documentElement", makeElement(vm, root, "html", result))

	// document.location (alias de window.location)
	doc.Set("location", makeLocation(vm, result))

	// document.cookie (stub simple en mémoire)
	cookieStore := ""
	doc.DefineAccessorProperty("cookie",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(cookieStore)
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			cookieStore += call.Argument(0).String() + ";"
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	// document.readyState
	doc.Set("readyState", vm.ToValue("complete"))

	// document.addEventListener (stub — exécute immédiatement si "DOMContentLoaded")
	doc.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		eventName := call.Argument(0).String()
		fn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			return goja.Undefined()
		}
		// Exécuter immédiatement les handlers DOMContentLoaded et load
		if eventName == "DOMContentLoaded" || eventName == "load" || eventName == "readystatechange" {
			fn(goja.Undefined(), vm.NewObject()) // nolint
		}
		return goja.Undefined()
	})
	doc.Set("removeEventListener", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})
	doc.Set("dispatchEvent", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(true)
	})

	// document.write / writeln (stubs)
	doc.Set("write", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	doc.Set("writeln", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	vm.Set("document", doc)
}

// makeElement crée un objet JS représentant un élément DOM Fox.
func makeElement(vm *goja.Runtime, el *engine.Element, selector string, result *ExecResult) goja.Value {
	if el == nil {
		return goja.Null()
	}

	obj := vm.NewObject()

	// ── Propriétés de base ──
	obj.Set("tagName", vm.ToValue(strings.ToUpper(el.Tag)))
	obj.Set("nodeName", vm.ToValue(strings.ToUpper(el.Tag)))
	obj.Set("nodeType", vm.ToValue(1))

	// id
	if el.Attrs != nil {
		obj.Set("id", vm.ToValue(el.Attrs["id"]))
	} else {
		obj.Set("id", vm.ToValue(""))
	}

	// ── textContent getter/setter ──
	obj.DefineAccessorProperty("textContent",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(engine.CollectText(el))
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			val := call.Argument(0).String()
			sel := selectorFor(el, selector)
			if sel != "" {
				result.Changes = append(result.Changes, DOMChange{
					Selector: sel, Property: "textContent", Value: val,
				})
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	// ── innerText (alias textContent) ──
	obj.DefineAccessorProperty("innerText",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(engine.CollectText(el))
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			val := call.Argument(0).String()
			sel := selectorFor(el, selector)
			if sel != "" {
				result.Changes = append(result.Changes, DOMChange{
					Selector: sel, Property: "textContent", Value: val,
				})
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	// ── innerHTML getter/setter ──
	obj.DefineAccessorProperty("innerHTML",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(el.Text) // approximation
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			val := call.Argument(0).String()
			sel := selectorFor(el, selector)
			if sel != "" {
				result.Changes = append(result.Changes, DOMChange{
					Selector: sel, Property: "innerHTML", Value: val,
				})
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	// ── hidden getter/setter ──
	obj.DefineAccessorProperty("hidden",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if el.Attrs != nil && el.Attrs["data-fox-hidden"] == "true" {
				return vm.ToValue(true)
			}
			return vm.ToValue(false)
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			hidden := call.Argument(0).ToBoolean()
			sel := selectorFor(el, selector)
			if sel != "" {
				val := "false"
				if hidden {
					val = "true"
				}
				result.Changes = append(result.Changes, DOMChange{
					Selector: sel, Property: "hidden", Value: val,
				})
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	// ── className getter/setter ──
	obj.DefineAccessorProperty("className",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if el.Attrs != nil {
				return vm.ToValue(el.Attrs["class"])
			}
			return vm.ToValue("")
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if el.Attrs == nil {
				el.Attrs = make(map[string]string)
			}
			el.Attrs["class"] = call.Argument(0).String()
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	// ── style object ──
	style := vm.NewObject()
	style.DefineAccessorProperty("display",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if el.Attrs != nil && el.Attrs["data-fox-hidden"] == "true" {
				return vm.ToValue("none")
			}
			return vm.ToValue("block")
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			val := call.Argument(0).String()
			sel := selectorFor(el, selector)
			if sel != "" {
				result.Changes = append(result.Changes, DOMChange{
					Selector: sel, Property: "style.display", Value: val,
				})
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)
	style.DefineAccessorProperty("visibility",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue("visible")
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			val := call.Argument(0).String()
			sel := selectorFor(el, selector)
			if sel != "" {
				result.Changes = append(result.Changes, DOMChange{
					Selector: sel, Property: "style.visibility", Value: val,
				})
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)
	style.Set("setProperty", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	style.Set("removeProperty", func(call goja.FunctionCall) goja.Value { return vm.ToValue("") })
	style.Set("getPropertyValue", func(call goja.FunctionCall) goja.Value { return vm.ToValue("") })
	obj.Set("style", style)

	// ── classList object ──
	classList := vm.NewObject()
	classList.Set("add", func(call goja.FunctionCall) goja.Value {
		class := call.Argument(0).String()
		sel := selectorFor(el, selector)
		if sel != "" {
			result.Changes = append(result.Changes, DOMChange{
				Selector: sel, Property: "class.add", Value: class,
			})
		}
		return goja.Undefined()
	})
	classList.Set("remove", func(call goja.FunctionCall) goja.Value {
		class := call.Argument(0).String()
		sel := selectorFor(el, selector)
		if sel != "" {
			result.Changes = append(result.Changes, DOMChange{
				Selector: sel, Property: "class.remove", Value: class,
			})
		}
		return goja.Undefined()
	})
	classList.Set("contains", func(call goja.FunctionCall) goja.Value {
		class := call.Argument(0).String()
		return vm.ToValue(engine.HasClass(el, class))
	})
	classList.Set("toggle", func(call goja.FunctionCall) goja.Value {
		class := call.Argument(0).String()
		has := engine.HasClass(el, class)
		sel := selectorFor(el, selector)
		if sel != "" {
			prop := "class.add"
			if has {
				prop = "class.remove"
			}
			result.Changes = append(result.Changes, DOMChange{
				Selector: sel, Property: prop, Value: class,
			})
		}
		return vm.ToValue(!has)
	})
	obj.Set("classList", classList)

	// ── Attributs ──
	obj.Set("getAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if el.Attrs != nil {
			if v, ok := el.Attrs[name]; ok {
				return vm.ToValue(v)
			}
		}
		return goja.Null()
	})
	obj.Set("setAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		val := call.Argument(1).String()
		if el.Attrs == nil {
			el.Attrs = make(map[string]string)
		}
		el.Attrs[name] = val
		sel := selectorFor(el, selector)
		if sel != "" {
			result.Changes = append(result.Changes, DOMChange{
				Selector: sel, Property: "attr", AttrName: name, Value: val,
			})
		}
		return goja.Undefined()
	})
	obj.Set("hasAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if el.Attrs != nil {
			_, ok := el.Attrs[name]
			return vm.ToValue(ok)
		}
		return vm.ToValue(false)
	})
	obj.Set("removeAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if el.Attrs != nil {
			delete(el.Attrs, name)
		}
		return goja.Undefined()
	})

	// ── Navigation ──
	obj.Set("children", makeChildrenArray(vm, el, result))
	obj.Set("childNodes", makeChildrenArray(vm, el, result))
	obj.Set("firstChild", firstChildVal(vm, el, result))
	obj.Set("lastChild", lastChildVal(vm, el, result))
	obj.Set("nextSibling", goja.Null())
	obj.Set("previousSibling", goja.Null())
	obj.Set("parentNode", goja.Null())
	obj.Set("parentElement", goja.Null())
	obj.Set("childElementCount", vm.ToValue(len(el.Children)))

	// ── Méthodes DOM ──
	obj.Set("appendChild", func(call goja.FunctionCall) goja.Value {
		return call.Argument(0) // stub
	})
	obj.Set("insertBefore", func(call goja.FunctionCall) goja.Value {
		return call.Argument(0)
	})
	obj.Set("removeChild", func(call goja.FunctionCall) goja.Value {
		return call.Argument(0)
	})
	obj.Set("replaceChild", func(call goja.FunctionCall) goja.Value {
		return call.Argument(0)
	})
	obj.Set("cloneNode", func(call goja.FunctionCall) goja.Value {
		return makeElement(vm, el, "", result)
	})

	// ── querySelector sur cet élément ──
	obj.Set("querySelector", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		found := engine.QuerySelector(el, sel)
		if found == nil {
			return goja.Null()
		}
		return makeElement(vm, found, selectorForElement(found, sel), result)
	})
	obj.Set("querySelectorAll", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		els := engine.QuerySelectorAll(el, sel)
		return makeNodeList(vm, els, sel, result)
	})
	obj.Set("getElementsByTagName", func(call goja.FunctionCall) goja.Value {
		tag := strings.ToLower(call.Argument(0).String())
		els := engine.FindAllByTag(el, tag)
		return makeNodeList(vm, els, tag, result)
	})

	// ── Événements ──
	obj.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})
	obj.Set("removeEventListener", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})
	obj.Set("dispatchEvent", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(true)
	})
	obj.Set("click", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})
	obj.Set("focus", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	obj.Set("blur", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	// ── Dimensions (valeurs fictives mais cohérentes) ──
	obj.Set("offsetWidth", vm.ToValue(800))
	obj.Set("offsetHeight", vm.ToValue(600))
	obj.Set("offsetTop", vm.ToValue(0))
	obj.Set("offsetLeft", vm.ToValue(0))
	obj.Set("clientWidth", vm.ToValue(800))
	obj.Set("clientHeight", vm.ToValue(600))
	obj.Set("scrollWidth", vm.ToValue(800))
	obj.Set("scrollHeight", vm.ToValue(600))
	obj.Set("getBoundingClientRect", func(call goja.FunctionCall) goja.Value {
		rect := vm.NewObject()
		rect.Set("top", vm.ToValue(0))
		rect.Set("left", vm.ToValue(0))
		rect.Set("bottom", vm.ToValue(600))
		rect.Set("right", vm.ToValue(800))
		rect.Set("width", vm.ToValue(800))
		rect.Set("height", vm.ToValue(600))
		return rect
	})
	obj.Set("scrollIntoView", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	obj.Set("scrollTo", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	// ── Form/input spécifiques ──
	obj.DefineAccessorProperty("value",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if el.Attrs != nil {
				return vm.ToValue(el.Attrs["value"])
			}
			return vm.ToValue("")
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if el.Attrs == nil {
				el.Attrs = make(map[string]string)
			}
			el.Attrs["value"] = call.Argument(0).String()
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)
	obj.DefineAccessorProperty("checked",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if el.Attrs != nil {
				_, ok := el.Attrs["checked"]
				return vm.ToValue(ok)
			}
			return vm.ToValue(false)
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if el.Attrs == nil {
				el.Attrs = make(map[string]string)
			}
			if call.Argument(0).ToBoolean() {
				el.Attrs["checked"] = "checked"
			} else {
				delete(el.Attrs, "checked")
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	return obj
}

// makeNodeList crée un objet HTMLCollection/NodeList pour Goja.
func makeNodeList(vm *goja.Runtime, els []*engine.Element, sel string, result *ExecResult) goja.Value {
	arr := vm.NewArray(len(els))
	for i, el := range els {
		arr.Set(fmt.Sprintf("%d", i), makeElement(vm, el, selectorForElement(el, sel), result))
	}
	arr.Set("length", vm.ToValue(len(els)))
	arr.Set("item", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		idx := int(call.Argument(0).ToInteger())
		if idx >= 0 && idx < len(els) {
			return makeElement(vm, els[idx], selectorForElement(els[idx], sel), result)
		}
		return goja.Null()
	}))
	arr.Set("forEach", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			return goja.Undefined()
		}
		for i, el := range els {
			_, err := fn(goja.Undefined(),
				makeElement(vm, el, selectorForElement(el, sel), result),
				vm.ToValue(i),
			)
			if err != nil {
				break
			}
		}
		return goja.Undefined()
	}))
	arr.Set("entries", vm.ToValue(func(call goja.FunctionCall) goja.Value { return goja.Undefined() }))
	arr.Set("keys", vm.ToValue(func(call goja.FunctionCall) goja.Value { return goja.Undefined() }))
	arr.Set("values", vm.ToValue(func(call goja.FunctionCall) goja.Value { return goja.Undefined() }))
	return arr
}

func makeChildrenArray(vm *goja.Runtime, el *engine.Element, result *ExecResult) goja.Value {
	if len(el.Children) == 0 {
		arr := vm.NewArray(0)
		arr.Set("length", vm.ToValue(0))
		return arr
	}
	arr := vm.NewArray(len(el.Children))
	for i, child := range el.Children {
		arr.Set(fmt.Sprintf("%d", i), makeElement(vm, child, selectorForElement(child, ""), result))
	}
	arr.Set("length", vm.ToValue(len(el.Children)))
	return arr
}

func firstChildVal(vm *goja.Runtime, el *engine.Element, result *ExecResult) goja.Value {
	if len(el.Children) == 0 {
		return goja.Null()
	}
	return makeElement(vm, el.Children[0], "", result)
}

func lastChildVal(vm *goja.Runtime, el *engine.Element, result *ExecResult) goja.Value {
	if len(el.Children) == 0 {
		return goja.Null()
	}
	return makeElement(vm, el.Children[len(el.Children)-1], "", result)
}

// selectorFor retourne le meilleur sélecteur pour un élément.
func selectorFor(el *engine.Element, hint string) string {
	if hint != "" {
		return hint
	}
	return selectorForElement(el, "")
}

func selectorForElement(el *engine.Element, hint string) string {
	if hint != "" && hint != "" {
		return hint
	}
	if el == nil {
		return ""
	}
	if el.Attrs != nil && el.Attrs["id"] != "" {
		return "#" + el.Attrs["id"]
	}
	if el.Tag != "" && el.Tag != "div" && el.Tag != "span" {
		return el.Tag
	}
	return ""
}

// makeLocation crée l'objet location pour window et document.
func makeLocation(vm *goja.Runtime, result *ExecResult) *goja.Object {
	return vm.NewObject() // sera surchargé par setupWindow
}
