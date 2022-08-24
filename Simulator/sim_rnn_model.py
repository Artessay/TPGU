import tensorflow as tf
from Simulator.sim_rnn_cell import SimulatorRNNCell

def SimulatorRNNModel(sim_config):
    num_steps = sim_config.num_steps
    input_size = sim_config.input_size
    output_size = sim_config.output_size
    learning_rate = sim_config.learning_rate

    model = tf.keras.Sequential([
        tf.keras.layers.Input(shape=[num_steps, input_size], name="inputs"),
        tf.keras.layers.LSTM(256, return_sequences=True),
        tf.keras.layers.LSTM(128),
        tf.keras.layers.Dense(output_size, name="targets")
    ])

    model.compile(
        loss=tf.keras.losses.MeanSquaredError(), 
        optimizer=tf.keras.optimizers.Nadam(learning_rate=learning_rate), 
        metrics=['mean_absolute_error']
    )

    return model