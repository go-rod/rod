// eslint-disable-next-line no-unused-expressions
;() => {
  const rod = {
    _() {},

    element(...selectors) {
      const scope = ensureScope(this)
      for (const selector of selectors) {
        const el = scope.querySelector(selector)
        if (el) {
          return el
        }
      }
      return null
    },

    elements(selector) {
      return ensureScope(this).querySelectorAll(selector)
    },

    elementX(...xPaths) {
      const scope = ensureScope(this)
      for (const xPath of xPaths) {
        const el = document.evaluate(
          xPath,
          scope,
          null,
          XPathResult.FIRST_ORDERED_NODE_TYPE
        ).singleNodeValue
        if (el) {
          return el
        }
      }
      return null
    },

    elementsX(xpath) {
      const scope = ensureScope(this)
      const iter = document.evaluate(
        xpath,
        scope,
        null,
        XPathResult.ORDERED_NODE_ITERATOR_TYPE
      )
      const list = []
      let el
      while ((el = iter.iterateNext())) list.push(el)
      return list
    },

    elementR(...pairs) {
      for (let i = 0; i < pairs.length - 1; i += 2) {
        const selector = pairs[i]
        const pattern = pairs[i + 1]
        const reg = new RegExp(pattern)
        const el = Array.from(
          (this.document || this).querySelectorAll(selector)
        ).find((e) => reg.test(rod.text.call(e)))
        if (el) {
          return el
        }
      }
      return null
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
      await rod.waitLoad()

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
      document.body.appendChild(svg)
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

    async overlay(id, left, top, width, height, msg) {
      await rod.waitLoad()

      const div = document.createElement('div')
      const msgDiv = document.createElement('div')
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

      msgDiv.style = `position: absolute; color: #cc26d6; font-size: 12px; background: #ffffffeb;
        box-shadow: #333 0 0 3px; padding: 2px 5px; border-radius: 3px; white-space: nowrap;
        top: ${height}px;`

      msgDiv.innerHTML = msg

      div.appendChild(msgDiv)
      document.body.appendChild(div)

      if (window.innerHeight < msgDiv.offsetHeight + top + height) {
        msgDiv.style.top = -msgDiv.offsetHeight - 2 + 'px'
      }

      if (window.innerWidth < msgDiv.offsetWidth + left) {
        msgDiv.style.left = window.innerWidth - msgDiv.offsetWidth - left + 'px'
      }
    },

    async elementOverlay(id, msg) {
      const interval = 100
      const el = ensureElement(this)

      let pre = el.getBoundingClientRect()
      await rod.overlay(id, pre.left, pre.top, pre.width, pre.height, msg)

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
      el && el.remove()
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

    selectText(pattern) {
      const m = this.value.match(new RegExp(pattern))
      if (m) {
        this.setSelectionRange(m.index, m.index + m[0].length)
      }
    },

    selectAllText() {
      this.select()
    },

    select(selectors) {
      selectors.forEach((s) => {
        Array.from(this.options).find((el) => {
          try {
            if (el.innerText.includes(s) || el.matches(s)) {
              el.selected = true
              return true
            }
          } catch (e) {}
        })
      })
      this.dispatchEvent(new Event('input', { bubbles: true }))
      this.dispatchEvent(new Event('change', { bubbles: true }))
    },

    visible() {
      const el = ensureElement(this)
      const box = el.getBoundingClientRect()
      const style = window.getComputedStyle(el)
      return (
        style.display !== 'none' &&
        style.visibility !== 'hidden' &&
        !!(box.top || box.bottom || box.width || box.height)
      )
    },

    invisible() {
      return !rod.visible.apply(this)
    },

    text() {
      switch (this.tagName) {
        case 'INPUT':
        case 'TEXTAREA':
          return this.value
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

    fetchAsDataURL(url) {
      return fetch(url)
        .then((res) => res.blob())
        .then(
          (data) =>
            new Promise((resolve, reject) => {
              var reader = new FileReader()
              reader.onload = () => resolve(reader.result)
              reader.onerror = () => reject(reader.error)
              reader.readAsDataURL(data)
            })
        )
    }
  }

  function ensureScope(s) {
    return s === window ? s.document : s
  }

  function ensureElement(el) {
    if (!el.tagName) {
      return el.parentElement
    }
    return el
  }

  return rod
}

// # sourceURL=__rod_helper__
