/**
 * Prism grammar for AILANG (```ailang fences).
 *
 * Before this existed, AILANG blocks were fenced as `typescript` to get any
 * highlighting at all, which hid them from the doc-sample gate
 * (tools/docsamples, `make check-doc-samples`). The gate's info-string tags
 * (```ailang parse | sketch | repl | error=CODE | file=path.ail) are metastring
 * words after the language; Docusaurus ignores unknown metastring words, so
 * they do not affect rendering.
 *
 * Keywords: the reserved list taught by `ailang prompt` ("Reserved Keywords").
 */
export default function registerAilang(Prism) {
  Prism.languages.ailang = {
    comment: [
      {pattern: /\/\*[\s\S]*?\*\//, greedy: true},
      {pattern: /--.*/, greedy: true},
    ],
    string: {
      pattern: /"(?:\\.|\$\{[^}]*\}|[^"\\])*"/,
      greedy: true,
      inside: {
        interpolation: {
          pattern: /\$\{[^}]*\}/,
          inside: {
            punctuation: /^\$\{|\}$/,
          },
        },
      },
    },
    'repl-prompt': {
      pattern: /^\s*(?:λ[^>\n]*|ailang)>/m,
      alias: 'comment',
    },
    effect: {
      // the effect row of a signature: `! {IO, FS}`
      pattern: /!\s*\{[^}]*\}/,
      alias: 'important',
    },
    keyword:
      /\b(?:if|then|else|match|with|select|timeout|func|pure|let|letrec|in|type|class|instance|forall|exists|deriving|module|import|export|extern|test|tests|property|properties|assert|requires|ensures|invariant|spawn|parallel|channel|send|recv|and|or|not)\b/,
    boolean: /\b(?:true|false)\b/,
    'class-name': /\b[A-Z][A-Za-z0-9_]*\b/,
    function: /\b[a-z_][A-Za-z0-9_]*(?=\s*\()/,
    number: /\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b/,
    operator: /->|=>|::|\+\+|==|!=|<=|>=|&&|\|\||[-+*\/%<>=!|\\]/,
    punctuation: /[{}[\](),.;:#]/,
  };
  Prism.languages.ail = Prism.languages.ailang;
}
