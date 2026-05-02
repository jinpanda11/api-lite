const allowedTags = new Set([
  'b', 'i', 'u', 'strong', 'em', 'a', 'p', 'br', 'ul', 'ol', 'li',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'span', 'div', 'blockquote', 'code', 'pre', 'hr',
])
const allowedAttrs = new Set(['href', 'target', 'rel', 'class', 'id'])
const uriAttrs = new Set(['href'])

export function sanitizeHTML(html: string): string {
  const div = document.createElement('div')
  div.innerHTML = html

  const clean = (node: Node): void => {
    if (node.nodeType === 3) return
    if (node.nodeType !== 1) {
      node.parentNode?.removeChild(node)
      return
    }
    const el = node as HTMLElement
    const tag = el.tagName.toLowerCase()
    if (!allowedTags.has(tag)) {
      while (el.firstChild) {
        el.parentNode!.insertBefore(el.firstChild, el)
      }
      el.parentNode!.removeChild(el)
      return
    }
    for (let i = el.attributes.length - 1; i >= 0; i--) {
      const name = el.attributes[i].name.toLowerCase()
      if (!allowedAttrs.has(name)) {
        el.removeAttribute(name)
      } else if (uriAttrs.has(name)) {
        const val = el.getAttribute(name) || ''
        if (/^javascript:/i.test(val) || /^data:/i.test(val)) {
          el.removeAttribute(name)
        }
      }
    }
    if (tag === 'a') {
      el.setAttribute('rel', 'noopener noreferrer')
      el.setAttribute('target', '_blank')
    }
    const children = Array.from(el.childNodes)
    children.forEach(clean)
  }

  const children = Array.from(div.childNodes)
  children.forEach(clean)
  return div.innerHTML
}
