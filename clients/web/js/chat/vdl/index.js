// This file was auto-generated by the vanadium vdl tool.
var vdl = require('vanadium').vdl;






module.exports = {};



// Types:




// Consts:



// Errors:



// Services:

  
    
function Chat(){}
module.exports.Chat = Chat

    
      
Chat.prototype.sendMessage = function(ctx, text) {
  throw new Error('Method SendMessage not implemented');
};
     

    
Chat.prototype._serviceDescription = {
  name: 'Chat',
  pkgPath: 'chat/vdl',
  doc: "",
  embeds: [],
  methods: [
    
      
    {
    name: 'SendMessage',
    doc: "// SendMessage sends a message to a user.",
    inArgs: [{
      name: 'text',
      doc: "",
      type: vdl.Types.STRING
    },
    ],
    outArgs: [],
    inStream: null,
    outStream: null,
    tags: []
  },
     
  ]
};

   
 


