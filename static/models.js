Vue.component('models-tab', {
	data: function() {
		return {
			mtab: '',
		};
	},
	props: ['tab'],
	mounted: function() {
		this.mtab = $('#m-nav a[data-toggle="tab"].active').attr('href');
		$('#m-nav a[data-toggle="tab"]').on('shown.bs.tab', (e) => {
			var target = $(e.target).attr('href');
			this.mtab = target;
		});
	},
	template: `
<div class="flex-container">
	<ul class="nav nav-tabs" id="m-nav" role="tablist">
		<li class="nav-item">
			<a class="nav-link active" id="m-training-tab" data-toggle="tab" href="#m-training-panel" role="tab">Training</a>
		</li>
		<li class="nav-item">
			<a class="nav-link" id="m-architectures-tab" data-toggle="tab" href="#m-architectures-panel" role="tab">Architectures</a>
		</li>
	</ul>
	<div class="tab-content mx-1 flex-content">
		<div class="tab-pane fade show active" id="m-training-panel" role="tabpanel">
			<m-training :mtab="mtab"></m-training>
		</div>
		<div class="tab-pane fade" id="m-architectures-panel" role="tabpanel">
			<m-architectures :mtab="mtab"></m-architectures>
		</div>
	</div>
</div>
	`,
});
