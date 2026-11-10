// Fixture: AILANG embedded in a page component as a template literal.
export default function Home() {
  return (
    <pre className="hero-code-content">
{`-- hero
module fixture/hero

export func main() -> () ! {IO} = print("hero")`}
    </pre>
  );
}

export const css = <style>{`
  .hero { color: red; }
`}</style>;
