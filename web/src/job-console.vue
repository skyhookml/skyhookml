<template>
<pre class="mx-2 flex-content mb-2" ref="pre"><template v-for="line in lines">{{ line }}
</template></pre>
</template>

<script>
import utils from './utils.js';

// Shared component for showing the console for a job.

export default {
	props: ['lines'],
	watch: {
		lines: function() {
			if(this.$refs.pre.scrollTop + this.$refs.pre.offsetHeight >= this.$refs.pre.scrollHeight) {
				Vue.nextTick(() => {
					this.$refs.pre.scrollTop = this.$refs.pre.scrollHeight;
				});
			}
		},
	},
};
</script>