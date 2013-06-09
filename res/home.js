function getStatus(){
	$.ajax({
		type: "GET",
		url: "/status",
		success: function(data){
			$("#status_table").html(data);
			setTimeout(getStatus,2000);
		},
		error: function(XMLHttpRequest, textStatus, errorThrown){
			var errMsg = "Failed to contact the server<br/>Reload the page to restart polling or visit any other node's web interface";
			$("#output").append("<span style='color: red;'>"+errMsg+"</span>");
			$("#output").scrollTop($("#output")[0].scrollHeight);
		}
	});
};

$(document).ready(function(){
	getStatus();
	$("#commandSubmit").click(function() {
		$.ajax({
			type: "POST",
			url: "/exec",
			data: $("#commandForm").serialize(),
			success: function(data)	{
				$("#output").append(data);
				$("#output").scrollTop($("#output")[0].scrollHeight);
			}
		});

		return false;
	});
});
