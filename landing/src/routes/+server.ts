import bootstrapScript from '../../static/bootstrap.sh?raw';

// curl nowbox.lol | sh  →  returns the bootstrap shell script
// browser visit         →  SvelteKit renders +page.svelte automatically
//
// SvelteKit routes: if both +server.ts and +page.svelte exist,
// GET requests with accept: text/html go to the page,
// other GET requests go to the server handler.
// This is exactly the content negotiation we need.

export function GET() {
	return new Response(bootstrapScript, {
		headers: { 'content-type': 'text/plain; charset=utf-8' }
	});
}
