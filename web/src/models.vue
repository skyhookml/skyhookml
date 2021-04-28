<template>
<div>
	<ul class="nav nav-tabs mb-3" id="m-nav" role="tablist">
		<li class="nav-item">
			<button class="nav-link active" id="m-components-tab" data-bs-toggle="tab" data-bs-target="#m-components-panel" role="tab">Components</button>
		</li>
		<li class="nav-item">
			<button class="nav-link" id="m-architectures-tab" data-bs-toggle="tab" data-bs-target="#m-architectures-panel" role="tab">Architectures</button>
		</li>
	</ul>
	<div class="tab-content">
		<div class="tab-pane fade show active" id="m-components-panel" role="tabpanel">
			<m-components :mtab="mtab"></m-components>
		</div>
		<div class="tab-pane fade" id="m-architectures-panel" role="tabpanel">
			<m-architectures :mtab="mtab"></m-architectures>
		</div>
	</div>
</div>
</template>

<script>
import MComponents from './m-components.vue';
import MArchitectures from './m-architectures.vue';

const Models = {
	components: {
		'm-components': MComponents,
		'm-architectures': MArchitectures,
	},
	data: function() {
		return {
			mtab: '',
		};
	},
	mounted: function() {
		this.mtab = $('#m-nav a[data-toggle="tab"].active').attr('href');
		$('#m-nav a[data-toggle="tab"]').on('shown.bs.tab', (e) => {
			var target = $(e.target).attr('href');
			this.mtab = target;
		});
	},
};
export default Models;
</script>
