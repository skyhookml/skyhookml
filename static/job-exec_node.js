import utils from './utils.js';
import JobConsole from './job-console.js';

export default {
	components: {
		'job-console': JobConsole,
	},
	data: function() {
		return {
			progress: 0,
			lines: [],
		};
	},
	props: ['jobID'],
	created: function() {
		this.fetch();
		this.interval = setInterval(this.fetch, 1000);
	},
	destroyed: function() {
		clearInterval(this.interval);
	},
	methods: {
		fetch: function() {
			utils.request(this, 'POST', '/jobs/'+this.jobID+'/state', null, (state) => {
				if(!state) {
					return;
				}
				this.progress = parseInt(state.Progress);
				this.lines = state.Lines;
			});
		},
	},
	template: `
<div class="flex-container">
	<div class="mb-2">
		<div class="progress">
			<div
				class="progress-bar"
				role="progressbar"
				v-bind:style="{width: progress+'%'}"
				:aria-valuenow="progress"
				aria-valuemin="0"
				aria-valuemax="100"
				>
				{{ progress }}%
			</div>
		</div>
	</div>
	<div class="flex-content">
		<job-console :jobID="jobID" :lines="lines"></job-console>
	</div>
</div>
	`,
};
