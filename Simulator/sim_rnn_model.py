import tensorflow as tf
from Simulator.sim_rnn_cell import SimulatorRNNCell

def SimulatorRNNModel(sim_config):
    num_steps = sim_config.num_steps
    input_size = sim_config.input_size
    output_size = sim_config.output_size
    decay_steps = sim_config.decay_steps
    learning_rate = sim_config.learning_rate
    learning_rate_decay = sim_config.learning_rate_decay

    model = tf.keras.Sequential([
        # tf.keras.layers.RNN(SimulatorRNNCell(units=256), input_shape=[num_steps, input_size]),
        tf.keras.layers.Input(shape=[num_steps, input_size], name="inputs"),
        tf.keras.layers.LSTM(256, return_sequences=True),
        tf.keras.layers.LSTM(128),
        tf.keras.layers.Dense(output_size, name="targets")
    ])

    model.compile(
        loss=tf.keras.losses.MeanSquaredError(), 
        optimizer=tf.keras.optimizers.Adam(
            learning_rate=tf.keras.optimizers.schedules.ExponentialDecay(
                initial_learning_rate=learning_rate, 
                decay_steps=decay_steps,
                decay_rate=learning_rate_decay
            )
        ), 
        metrics=['mean_absolute_error']
    )

    return model