import utils from './utils.js';

export default {
	data: function() {
		return {
			path: '',
			file: null,
			percent: null,
		};
	},
	props: ['dataset'],
	mounted: function() {
		$(this.$refs.modal).on('shown.bs.modal', () => {
			$(this.$refs.modal).focus();
		});
	},
	methods: {
		click: function() {
			this.path = '';
			this.file = null;
			this.percent = null;
			$(this.$refs.modal).modal('show');
		},
		onFileChange: function(event) {
			this.file = event.target.files[0];
		},
		submitLocal: function() {
			utils.request(this, 'POST', '/datasets/'+this.dataset.ID+'/import-local', {path: this.path});
			$(this.$refs.modal).modal('hide');
		},
		submitUpload: function() {
			var data = new FormData();
			data.append('file', this.file);
			this.percent = null;
			$.ajax({
				type: 'POST',
				url: '/datasets/'+this.dataset.ID+'/import-upload',
				error: function(req, status, errorMsg) {
					app.setError(errorMsg);
				},
				data: data,
				processData: false,
				contentType: false,
				xhr: () => {
					var xhr = new window.XMLHttpRequest();
					xhr.upload.addEventListener('progress', (e) => {
						if(!e.lengthComputable) {
							return;
						}
						this.percent = parseInt(e.loaded * 100 / e.total);
					});
					return xhr;
				},
				success: () => {
					$(this.$refs.modal).modal('hide');
					this.$emit('imported');
				},
			});
		},
	},
	template: `
<span>
	<button type="button" class="btn btn-primary" v-on:click="click">Import Data</button>
	<div class="modal" tabindex="-1" role="dialog" ref="modal">
		<div class="modal-dialog" role="document">
			<div class="modal-content">
				<div class="modal-body">
					<ul class="nav nav-tabs">
						<li class="nav-item">
							<a class="nav-link active" data-toggle="tab" href="#import-local-tab" role="tab">Local</a>
						</li>
						<li class="nav-item">
							<a class="nav-link" data-toggle="tab" href="#import-upload-tab" role="tab">Upload</a>
						</li>
					</ul>
					<div class="tab-content">
						<div class="tab-pane show active" id="import-local-tab">
							<form v-on:submit.prevent="submitLocal">
								<div class="form-group row">
									<label class="col-sm-2 col-form-label">Path</label>
									<div class="col-sm-10">
										<input class="form-control" type="text" v-model="path" />
										<small class="form-text text-muted">
											The path to an export zip file on the local disk where SkyhookML is running.
										</small>
									</div>
								</div>
								<div class="form-group row">
									<div class="col-sm-10">
										<button type="submit" class="btn btn-primary">Import</button>
									</div>
								</div>
							</form>
						</div>
						<div class="tab-pane" id="import-upload-tab">
							<form v-on:submit.prevent="submitUpload">
								<div class="form-group row">
									<label class="col-sm-2 col-form-label">File</label>
									<div class="col-sm-10">
										<input class="form-control" type="file" @change="onFileChange" />
										<small class="form-text text-muted">
											Upload a ... file.
										</small>
									</div>
								</div>
								<div class="form-group row">
									<div class="col-sm-10">
										<button type="submit" class="btn btn-primary">Import</button>
									</div>
								</div>
							</form>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</span>
	`,
};
