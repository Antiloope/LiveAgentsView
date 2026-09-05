import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import './Markdown.css'

type MdNode = { type: string; value?: string; children?: MdNode[] }

// react-markdown drops raw-HTML nodes instead of rendering them, so a model
// that writes a tag in prose would have it silently disappear. Turning those
// nodes into text keeps the tag on screen as the characters the model wrote,
// still escaped by React and never markup the browser acts on.
function remarkHtmlAsText() {
  return (tree: unknown) => {
    const walk = (node: MdNode) => {
      if (!node.children) return
      node.children = node.children.map((child) =>
        child.type === 'html' ? { type: 'text', value: child.value } : child,
      )
      node.children.forEach(walk)
    }
    walk(tree as MdNode)
  }
}

/** A character's message rendered as GitHub-flavored markdown on parchment. */
export default function Markdown({ text }: { text: string }) {
  return (
    <div className="ui-md">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkHtmlAsText]}
        components={{
          // `node` is react-markdown's own mdast handle, dropped here so it
          // does not ride along into the DOM as an attribute.
          // Tables and code are the two things a model writes that can be
          // wider than the drawer; each scrolls in its own box so the
          // transcript column never does.
          table: ({ node, children, ...props }) => (
            <div className="ui-md__wide">
              <table {...props}>{children}</table>
            </div>
          ),
          pre: ({ node, children, ...props }) => (
            <pre className="ui-md__wide" {...props}>
              {children}
            </pre>
          ),
          a: ({ node, children, ...props }) => (
            <a {...props} target="_blank" rel="noreferrer noopener">
              {children}
            </a>
          ),
        }}
      >
        {text}
      </ReactMarkdown>
    </div>
  )
}
