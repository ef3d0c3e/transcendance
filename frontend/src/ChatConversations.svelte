<script>
	import { relativeTime, fullDate } from './relative-time.js';

	let { conversations, ignored, username, locale, now, labels, onOpen, onUnignore } = $props();

	let start = $state('');

	// `current` is only there so Svelte re-renders the time when `now` changes
	function ago(date, current) {
		return current && relativeTime(date, locale);
	}

	function submit(event) {
		event.preventDefault();
		const name = start.trim();
		if (name) {
			start = '';
			onOpen(name);
		}
	}
</script>

<div class="chat-conversations" data-chat-scroll>
	<form class="chat-start" onsubmit={submit}>
		<input
			bind:value={start}
			type="text"
			class="chat-start-input"
			placeholder={labels.labelStartPlaceholder}
			aria-label={labels.labelStart}
			autocomplete="off"
			maxlength="16"
		/>
		<button type="submit" class="chat-start-button" disabled={!start.trim()}>{labels.labelStartButton}</button>
	</form>

	<ul class="chat-conversation-list" aria-label={labels.labelTabMessages}>
		{#each conversations as conversation (conversation.username)}
			<li>
				<button
					type="button"
					class="chat-conversation"
					class:is-unread={conversation.unread > 0}
					onclick={() => onOpen(conversation.username)}
				>
					<span class="chat-conversation-head">
						<span class="chat-conversation-name">{conversation.username}</span>
						<time datetime={conversation.last.createdAt} title={fullDate(conversation.last.createdAt, locale)}>
							{ago(conversation.last.createdAt, now)}
						</time>
					</span>
					<span class="chat-conversation-preview">
						{#if conversation.last.from === username}
							<span class="chat-conversation-you">{labels.labelYou}:</span>
						{/if}
						{conversation.last.body}
					</span>
					{#if conversation.unread > 0}
						<span class="chat-conversation-badge">
							{conversation.unread > 9 ? '9+' : conversation.unread}
							<span class="chat-sr">{labels.labelUnread}</span>
						</span>
					{/if}
				</button>
			</li>
		{:else}
			<li class="chat-conversations-empty">{labels.labelConversationsEmpty}</li>
		{/each}
	</ul>

	{#if ignored.length > 0}
		<details class="chat-ignored">
			<summary>{labels.labelIgnoredTitle} ({ignored.length})</summary>
			<ul>
				{#each ignored as user (user.username)}
					<li>
						<span class="chat-ignored-user">
							<a href="/profile/{encodeURIComponent(user.username)}">{user.username}</a>
							<span class="chat-ignored-until">{labels.labelIgnoredUntil} {fullDate(user.until, locale)}</span>
						</span>
						<button type="button" class="chat-ignored-lift" onclick={() => onUnignore(user.username)}>
							{labels.labelUnignore}
						</button>
					</li>
				{/each}
			</ul>
		</details>
	{/if}
</div>

<style>
	.chat-conversations {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;

		min-height: 0;
		padding: 0.5rem;
		overflow-y: auto;
		overscroll-behavior: contain;

		scrollbar-width: thin;
		scrollbar-color: var(--color-border-2) transparent;
	}

	.chat-start {
		display: flex;
		gap: 0.4rem;
		margin: 0;
	}

	.chat-start-input {
		flex: 1;
		min-width: 0;
		padding-inline: 0.75rem;
	}

	.chat-start-button {
		flex: none;
	}

	.chat-conversation-list,
	.chat-ignored ul {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;

		margin: 0;
		padding: 0;
		list-style: none;
	}

	.chat-conversation {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 0.1rem;

		width: 100%;
		padding: 0.45rem 0.5rem;

		color: var(--color-text);
		background: none;
		border-radius: 0.5rem;

		font-weight: 400;
		text-align: start;

		&:hover {
			background: var(--color-bg-3);
		}
	}

	.chat-conversation-head {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		padding-inline-end: 2rem;

		time {
			color: var(--color-text-muted);
			font-size: 0.75rem;
			font-variant-numeric: tabular-nums;
			white-space: nowrap;
		}
	}

	.chat-conversation-name {
		overflow: hidden;

		color: var(--color-accent-primary);
		font-size: 0.85rem;
		font-weight: 700;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.chat-conversation-preview {
		overflow: hidden;
		padding-inline-end: 2rem;

		color: var(--color-text-muted);
		font-size: 0.85rem;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.is-unread .chat-conversation-preview {
		color: var(--color-text);
		font-weight: 600;
	}

	.chat-conversation-you {
		font-weight: 600;
	}

	.chat-conversation-badge {
		position: absolute;
		top: 50%;
		right: 0.5rem;
		transform: translateY(-50%);

		min-width: 1.25rem;
		height: 1.25rem;
		padding: 0 0.3rem;

		color: var(--color-text-alt);
		background: var(--color-accent-secondary);
		border-radius: 999px;

		font-size: 0.7rem;
		font-weight: 700;
		line-height: 1.25rem;
		font-variant-numeric: tabular-nums;
		text-align: center;
	}

	/* Read by screen readers only */
	.chat-sr {
		position: absolute;
		overflow: hidden;
		width: 1px;
		height: 1px;
		clip-path: inset(50%);
		white-space: nowrap;
	}

	.chat-conversations-empty {
		margin: auto;
		padding: 1rem;

		color: var(--color-text-muted);
		text-align: center;
	}

	.chat-ignored {
		margin-top: auto;
		padding-top: 0.5rem;
		border-top: 1px solid var(--color-border-1);

		summary {
			padding: 0.25rem 0.5rem;

			color: var(--color-text-muted);
			font-size: 0.85rem;
			font-weight: 600;

			cursor: pointer;
		}

		li {
			display: flex;
			align-items: center;
			gap: 0.5rem;
			padding: 0.35rem 0.5rem;
		}
	}

	.chat-ignored-user {
		display: flex;
		flex: 1;
		flex-direction: column;
		min-width: 0;

		a {
			overflow: hidden;

			color: var(--color-text);
			font-size: 0.85rem;
			font-weight: 700;
			text-overflow: ellipsis;
		}
	}

	.chat-ignored-until {
		color: var(--color-text-muted);
		font-size: 0.75rem;
	}

	.chat-ignored-lift {
		flex: none;
		padding: 0.3rem 0.6rem;
		font-size: 0.8rem;
	}
</style>
