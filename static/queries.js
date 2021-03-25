import utils from './utils.js';
import QueriesTable from './queries-table.js';
import QueriesGraph from './queries-graph.js';

export default {
	components: {
		'queries-table': QueriesTable,
		'queries-graph': QueriesGraph,
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
		<queries-table v-if="mode == 'table'"></queries-table>
		<queries-graph v-if="mode == 'graph'"></queries-graph>
	</div>
</div>
	`,
};
