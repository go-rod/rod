// The reason to use an extra js file to hold the functions is the lint and IDE support.
// To debug just add "debugger" keyword to the line you want to pause, then run something like:
//
//     go run ./lib/js/generate/main.go
//     go test -run ^TestClick$ -- -rod=show,devtools

const functions = {
  element(selector) {
    const s = functions.selectable(this)
    return s.querySelector(selector)
  },

  triggerFavicon() {
    return new Promise((resolve, reject) => {
      const faviconElement = document.querySelector('link[rel~=icon]')
      const href = (faviconElement && faviconElement.href) || '/favicon.ico'
      const faviconUrl = new URL(href, window.location).toString()
      const xhr = new XMLHttpRequest()
      xhr.open('GET', faviconUrl)

      xhr.ontimeout = function () {
        reject({
          errorType: 'timeout_error',
          xhr: xhr
        })
      }

      xhr.onreadystatechange = function () {
        if (xhr.readyState === 4) {
          if ((xhr.status >= 200 && xhr.status < 300) || xhr.status === 304) {
            resolve({
              status: xhr.status,
              statusText: xhr.statusText,
              responseText: xhr.responseText
            })
          } else {
            reject({
              errorType: 'status_error',
              xhr: xhr,
              status: xhr.status,
              statusText: xhr.statusText,
              responseText: xhr.responseText
            })
          }
        }
      }

      xhr.onerror = function () {
        reject({
          errorType: 'onerror',
          xhr: xhr,
          status: xhr.status,
          statusText: xhr.statusText,
          responseText: xhr.responseText
        })
      }
      xhr.send()
    })
  },

  elements(selector) {
    return functions.selectable(this).querySelectorAll(selector)
  },

  elementX(xPath) {
    const s = functions.selectable(this)
    return document.evaluate(
      xPath,
      s,
      null,
      XPathResult.FIRST_ORDERED_NODE_TYPE
    ).singleNodeValue
  },

  elementsX(xpath) {
    const s = functions.selectable(this)
    const iter = document.evaluate(
      xpath,
      s,
      null,
      XPathResult.ORDERED_NODE_ITERATOR_TYPE
    )
    const list = []
    let el
    while ((el = iter.iterateNext())) list.push(el)
    return list
  },

  elementR(selector, regex) {
    var reg
    var m = regex.match(/(\/?)(.+)\1([a-z]*)/i)
    // cSpell:ignore gmix
    if (m[3] && !/^(?!.*?(.).*?\1)[gmixXsuUAJ]+$/.test(m[3]))
      reg = new RegExp(regex)
    else reg = new RegExp(m[2], m[3])

    const s = functions.selectable(this)
    const el = Array.from(s.querySelectorAll(selector)).find((e) =>
      reg.test(functions.text.call(e))
    )
    return el ? el : null
  },

  parents(selector) {
    let p = this.parentElement
    const list = []
    while (p) {
      if (p.matches(selector)) {
        list.push(p)
      }
      p = p.parentElement
    }
    return list
  },

  containsElement(target) {
    var node = target
    while (node != null) {
      if (node === this) {
        return true
      }
      node = node.parentElement
    }
    return false
  },

  async initMouseTracer(iconId, icon) {
    await functions.waitLoad()

    if (document.getElementById(iconId)) {
      return
    }

    const tmp = document.createElement('div')
    tmp.innerHTML = icon
    const svg = tmp.lastChild
    svg.id = iconId
    svg.style =
      'position: absolute; z-index: 2147483647; width: 17px; pointer-events: none;'
    svg.removeAttribute('width')
    svg.removeAttribute('height')
    document.body.parentElement.appendChild(svg)
  },

  updateMouseTracer(iconId, x, y) {
    const svg = document.getElementById(iconId)
    if (!svg) {
      return false
    }
    svg.style.left = x - 2 + 'px'
    svg.style.top = y - 3 + 'px'
    return true
  },

  rect() {
    const b = functions.tag(this).getBoundingClientRect()
    return { x: b.x, y: b.y, width: b.width, height: b.height }
  },

  async overlay(id, left, top, width, height, msg) {
    await functions.waitLoad()

    const div = document.createElement('div')
    div.id = id
    div.style = `position: fixed; z-index:2147483647; border: 2px dashed red;
        border-radius: 3px; box-shadow: #5f3232 0 0 3px; pointer-events: none;
        box-sizing: border-box;
        left: ${left}px;
        top: ${top}px;
        height: ${height}px;
        width: ${width}px;`

    if (width * height === 0) {
      div.style.border = 'none'
    }

    if (!msg) {
      document.body.parentElement.appendChild(div)
      return
    }

    const msgDiv = document.createElement('div')
    msgDiv.style = `position: absolute; color: #cc26d6; font-size: 12px; background: #ffffffeb;
        box-shadow: #333 0 0 3px; padding: 2px 5px; border-radius: 3px; white-space: nowrap;
        top: ${height}px;`

    msgDiv.innerHTML = msg
    div.appendChild(msgDiv)
    document.body.parentElement.appendChild(div)

    if (window.innerHeight < msgDiv.offsetHeight + top + height) {
      msgDiv.style.top = -msgDiv.offsetHeight - 2 + 'px'
    }

    if (window.innerWidth < msgDiv.offsetWidth + left) {
      msgDiv.style.left = window.innerWidth - msgDiv.offsetWidth - left + 'px'
    }
  },

  async elementOverlay(id, msg) {
    const interval = 100
    const el = functions.tag(this)

    let pre = el.getBoundingClientRect()
    await functions.overlay(id, pre.left, pre.top, pre.width, pre.height, msg)

    const update = () => {
      const overlay = document.getElementById(id)
      if (overlay === null) return

      const box = el.getBoundingClientRect()
      if (
        pre.left === box.left &&
        pre.top === box.top &&
        pre.width === box.width &&
        pre.height === box.height
      ) {
        setTimeout(update, interval)
        return
      }

      overlay.style.left = box.left + 'px'
      overlay.style.top = box.top + 'px'
      overlay.style.width = box.width + 'px'
      overlay.style.height = box.height + 'px'
      pre = box

      setTimeout(update, interval)
    }

    setTimeout(update, interval)
  },

  removeOverlay(id) {
    const el = document.getElementById(id)
    // prevent override like prototype.js
    el && Element.prototype.remove.call(el)
  },

  waitIdle(timeout) {
    return new Promise((resolve) => {
      window.requestIdleCallback(resolve, { timeout })
    })
  },

  waitLoad() {
    const isWin = this === window
    return new Promise((resolve, reject) => {
      if (isWin) {
        if (document.readyState === 'complete') return resolve()
        window.addEventListener('load', resolve)
      } else {
        if (this.complete === undefined || this.complete) {
          resolve()
        } else {
          this.addEventListener('load', resolve)
          this.addEventListener('error', reject)
        }
      }
    })
  },

  inputEvent() {
    this.dispatchEvent(new Event('input', { bubbles: true }))
    this.dispatchEvent(new Event('change', { bubbles: true }))
  },

  inputTime(stamp) {
    const time = new Date(stamp)

    const pad = (n) => n.toString().padStart(2, '0')

    const y = time.getFullYear()
    const mon = pad(time.getMonth() + 1)
    const d = pad(time.getDate())
    const h = pad(time.getHours())
    const min = pad(time.getMinutes())

    switch (this.type) {
      case 'date':
        this.value = `${y}-${mon}-${d}`
        break
      case 'datetime-local':
        this.value = `${y}-${mon}-${d}T${h}:${min}`
        break
      case 'month':
        this.value = mon
        break
      case 'time':
        this.value = `${h}:${min}`
        break
    }

    functions.inputEvent.call(this)
  },

  selectText(pattern) {
    const m = this.value.match(new RegExp(pattern))
    if (m) {
      this.setSelectionRange(m.index, m.index + m[0].length)
    }
  },

  selectAllText() {
    this.select()
  },

  select(selectors, selected, type) {
    let matchers
    switch (type) {
      case 'regex':
        matchers = selectors.map((s) => {
          const reg = new RegExp(s)
          return (el) => reg.test(el.innerText)
        })
        break
      case 'css-selector':
        matchers = selectors.map((s) => (el) => el.matches(s))
        break
      default:
        matchers = selectors.map((s) => (el) => el.innerText.includes(s))
        break
    }

    const opts = Array.from(this.options)
    let has = false
    matchers.forEach((s) => {
      const el = opts.find(s)
      if (el) {
        el.selected = selected
        has = true
        return
      }
    })

    this.dispatchEvent(new Event('input', { bubbles: true }))
    this.dispatchEvent(new Event('change', { bubbles: true }))

    return has
  },

  visible() {
    const el = functions.tag(this)
    const box = el.getBoundingClientRect()
    const style = window.getComputedStyle(el)
    return (
      style.display !== 'none' &&
      style.visibility !== 'hidden' &&
      !!(box.top || box.bottom || box.width || box.height)
    )
  },

  invisible() {
    return !functions.visible.apply(this)
  },

  text() {
    switch (this.tagName) {
      case 'INPUT':
      case 'TEXTAREA':
        return this.value || this.placeholder
      case 'SELECT':
        return Array.from(this.selectedOptions)
          .map((el) => el.innerText)
          .join()
      case undefined:
        return this.textContent
      default:
        return this.innerText
    }
  },

  resource() {
    return new Promise((resolve, reject) => {
      if (this.complete) {
        return resolve(this.currentSrc)
      }
      this.addEventListener('load', () => resolve(this.currentSrc))
      this.addEventListener('error', (e) => reject(e))
    })
  },

  addScriptTag(id, url, content) {
    if (document.getElementById(id)) return

    return new Promise((resolve, reject) => {
      var s = document.createElement('script')

      if (url) {
        s.src = url
        s.onload = resolve
      } else {
        s.type = 'text/javascript'
        s.text = content
        resolve()
      }

      s.id = id
      s.onerror = reject
      document.head.appendChild(s)
    })
  },

  addStyleTag(id, url, content) {
    if (document.getElementById(id)) return

    return new Promise((resolve, reject) => {
      var el

      if (url) {
        el = document.createElement('link')
        el.rel = 'stylesheet'
        el.href = url
      } else {
        el = document.createElement('style')
        el.type = 'text/css'
        el.appendChild(document.createTextNode(content))
        resolve()
      }

      el.id = id
      el.onload = resolve
      el.onerror = reject
      document.head.appendChild(el)
    })
  },

  selectable(s) {
    return s.querySelector ? s : document
  },

  tag(el) {
    return el.tagName ? el : el.parentElement
  },

  exposeFunc(name, bind) {
    let callbackCount = 0
    window[name] = (req) =>
      new Promise((resolve, reject) => {
        const cb = bind + '_cb' + callbackCount++
        window[cb] = (res, err) => {
          delete window[cb]
          err ? reject(err) : resolve(res)
        }
        window[bind](JSON.stringify({ req, cb }))
      })
  },

  getXPath(optimized) {
    class Step {
      constructor(value, optimized) {
        this.value = value
        this.optimized = optimized || false
      }
      toString() {
        return this.value
      }
    }
    const xPathValue = function xPathValue(node, optimized) {
      let ownValue
      const ownIndex = xPathIndex(node)
      if (ownIndex === -1) {
        return null
      }
      switch (node.nodeType) {
        case Node.ELEMENT_NODE:
          if (optimized && node.id) {
            return new Step(`//*[@id='${node.id}']`, true)
          }
          ownValue = node.localName
          break
        case Node.ATTRIBUTE_NODE:
          ownValue = `@${node.nodeName}`
          break
        case Node.TEXT_NODE:
        case Node.CDATA_SECTION_NODE:
          ownValue = 'text()'
          break
        case Node.PROCESSING_INSTRUCTION_NODE:
          ownValue = 'processing-instruction()'
          break
        case Node.COMMENT_NODE:
          ownValue = 'comment()'
          break
        case Node.DOCUMENT_NODE:
          ownValue = ''
          break
        default:
          ownValue = ''
          break
      }
      if (ownIndex > 0) {
        ownValue += `[${ownIndex}]`
      }
      return new Step(ownValue, node.nodeType === Node.DOCUMENT_NODE)
    }
    const xPathIndex = function xPathIndex(node) {
      function areNodesSimilar(left, right) {
        if (left === right) {
          return true
        }
        if (
          left.nodeType === Node.ELEMENT_NODE &&
          right.nodeType === Node.ELEMENT_NODE
        ) {
          return left.localName === right.localName
        }
        if (left.nodeType === right.nodeType) {
          return true
        }
        const leftType =
          left.nodeType === Node.CDATA_SECTION_NODE
            ? Node.TEXT_NODE
            : left.nodeType
        const rightType =
          right.nodeType === Node.CDATA_SECTION_NODE
            ? Node.TEXT_NODE
            : right.nodeType
        return leftType === rightType
      }
      const parentNode = node.parentNode
      const siblings = parentNode ? parentNode.children : null
      if (!siblings) {
        return 0
      }
      let hasSameNamedElements
      for (let i = 0; i < siblings.length; ++i) {
        if (areNodesSimilar(node, siblings[i]) && !(siblings[i] === node)) {
          hasSameNamedElements = true
          break
        }
      }
      if (!hasSameNamedElements) {
        return 0
      }
      let ownIndex = 1
      for (let i = 0; i < siblings.length; ++i) {
        if (areNodesSimilar(node, siblings[i])) {
          if (siblings[i] === node) {
            return ownIndex
          }
          ++ownIndex
        }
      }
      return -1
    }
    const node = this
    if (node.nodeType === Node.DOCUMENT_NODE) {
      return '/'
    }
    const steps = []
    let contextNode = node
    while (contextNode) {
      const step = xPathValue(contextNode, optimized)
      if (!step) {
        break
      }
      steps.push(step)
      if (step.optimized) {
        break
      }
      contextNode = contextNode.parentNode
    }
    steps.reverse()
    return (steps.length && steps[0].optimized ? '' : '/') + steps.join('/')
  }
}
