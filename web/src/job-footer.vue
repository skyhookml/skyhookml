<template>
<div>
	<template v-if="job">
		<template v-if="!job.Done">
			<button class="btn btn-danger" v-on:click="stopJob">Terminate Job</button>
		</template>
		<template v-else-if="job.Error">
			<div class="alert alert-danger" role="alert">
				<strong>Job Failed:</strong>
				{{ job.Error }}
			</div>
		</template>
		<template v-else>
			<div class="alert alert-success" role="alert">
				<strong>Job completed successfully.</strong>
			</div>
		</template>
	</template>
</div>
</template>

<script>
import utils from './utils.js';

// Shared component for stop button if job is still running, or a message if it's done.

export default {
	props: ['job'],
	methods: {
		stopJob: function() {
			utils.request(this, 'POST', '/jobs/'+this.job.ID+'/stop');
		},
	},
};
</script>