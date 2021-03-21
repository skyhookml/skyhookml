import utils from './utils.js';

// Shared component for showing the console and Stop button for a job.

export default {
	props: ['jobID', 'lines'],
	methods: {
		stopJob: function() {
			utils.request(this, 'POST', '/jobs/'+this.jobID+'/stop');
			this.$refs.pre.scrollTop = this.$refs.pre.scrollHeight;
		},
	},
	watch: {
		lines: function() {
			if(this.$refs.pre.scrollTop + this.$refs.pre.offsetHeight >= this.$refs.pre.scrollHeight) {
				Vue.nextTick(() => {
					this.$refs.pre.scrollTop = this.$refs.pre.scrollHeight;
				});
			}
		},
	},
	template: `
<div class="flex-container">
	<pre class="mx-2 flex-content mb-2" ref="pre"><template v-for="line in lines">{{ line }}
</template></pre>
	<div>
		<button class="btn btn-danger" v-on:click="stopJob">Stop</button>
	</div>
</div>
	`,
};
