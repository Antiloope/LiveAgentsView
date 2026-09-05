import './QuestIndicator.css'

/**
 * Live tail of the transcript while a character is working: says what it is
 * doing right now, so the chat is not silent between the message that starts
 * a quest and the answer that ends it.
 */
export default function QuestIndicator({ text }: { text: string }) {
  return (
    <div className="ui-quest" role="status" aria-live="polite">
      <span className="ui-quest__embers" aria-hidden="true">
        <i />
        <i />
        <i />
      </span>
      <span className="ui-quest__text">{text}</span>
    </div>
  )
}
