import utils from './utils.js';
import PipelineTable from './pipeline-table.js';
import PipelineGraph from './pipeline-graph.js';

export default {
	components: {
		'pipeline-table': PipelineTable,
		'pipeline-graph': PipelineGraph,
	},
	data: function() {
		return {
			mode: 'table',
		};
	},
	template: `
<div class="flex-container">
	<div class="my-2">
		<div class="btn-group">
			<button
				class="btn btn-outline-secondary shadow-none"
				:class="{active: mode == 'table'}"
				v-on:click="mode = 'table'"
				>
				Table View
			</button>
			<button
				class="btn btn-outline-secondary shadow-none"
				:class="{active: this.mode == 'graph'}"
				v-on:click="mode = 'graph'"
				>
				Graph View
			</button>
		</div>
	</div>
	<div class="flex-content">
		<pipeline-table v-if="mode == 'table'"></pipeline-table>
		<pipeline-graph v-if="mode == 'graph'"></pipeline-graph>
	</div>
</div>
	`,
};
