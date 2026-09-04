/** Decorative gold corner flourishes for Craft Pixel HUD frames (SVG, not sprites). */
export default function OrnateCorners({ large = false }: { large?: boolean }) {
  const cls = large ? 'corner lg' : 'corner'
  return (
    <>
      <span className={`${cls} tl`} aria-hidden="true">
        <CornerSvg />
      </span>
      <span className={`${cls} tr`} aria-hidden="true">
        <CornerSvg />
      </span>
      <span className={`${cls} bl`} aria-hidden="true">
        <CornerSvg />
      </span>
      <span className={`${cls} br`} aria-hidden="true">
        <CornerSvg />
      </span>
    </>
  )
}

function CornerSvg() {
  return (
    <svg viewBox="0 0 44 44" aria-hidden="true">
      <path fill="#e8c878" d="M2 2h20v4H6v16H2V2z" />
      <path fill="#c5a059" d="M6 6h12v3H9v9H6V6z" />
      <path fill="#8a6828" d="M2 2h4v4H2zm16 0h4v4h-4zM2 18h4v4H2z" />
      <path fill="#f0d078" d="M22 2h6l4 4v6h-4V8h-6V2zm4 8h8v3h-3v8h-3V13h-2V10z" />
      <path fill="#c5a059" d="M28 4h4v2h-4zM34 10h2v4h-2z" />
      <circle cx="14" cy="14" r="3" fill="#8a6828" />
      <circle cx="14" cy="14" r="2" fill="#e8c878" />
      <circle cx="14" cy="14" r="0.9" fill="#fff2c0" />
      <path fill="#e8c878" d="M2 24h4v10H2zm4 6h8v4H6z" />
      <path fill="#c5a059" d="M4 26h2v6H4zm6 8h4v2h-4z" />
      <path fill="#f0d078" opacity=".9" d="M10 4h2v2h-2zM4 10h2v2H4zM30 6h2v2h-2z" />
    </svg>
  )
}
